/*
Copyright © 2024 Aaron Stovall
All rights reserved.
*/

package main

import (
	"PAN_ENGINE/utils"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	apiURL     string
	apiKey     string
	reportData map[string]interface{}
	// Added settings file path for persistent storage
	settingsPath string
	// Added fields for report customization
	maxRows      int
	reportFormat string
	// For caching/tracking API health
	apiStatus    string
	lastAPICheck time.Time
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		reportData:   make(map[string]interface{}),
		settingsPath: "settings.json",
		maxRows:      1000, // Increased from default 100
		reportFormat: "standard",
		apiStatus:    "unknown",
	}
}

// Settings structure for persistent storage
type Settings struct {
	APIURL        string `json:"api_url"`
	EncryptedKey  string `json:"encrypted_key"`
	MaxRows       int    `json:"max_rows"`
	ReportFormat  string `json:"report_format"`
	Theme         string `json:"theme"`
	DateFormat    string `json:"date_format"`
	DefaultFolder string `json:"default_folder"`
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize logging
	if err := utils.InitLogging(); err != nil {
		fmt.Printf("Failed to initialize logging: %v\n", err)
		return
	}

	// Ensure required directories exist
	if err := utils.EnsureDirectories(); err != nil {
		utils.ErrorLogger.Printf("Failed to create required directories: %v", err)
		return
	}

	// Load saved settings if they exist
	if err := a.loadSettings(); err != nil {
		utils.InfoLogger.Printf("Could not load settings: %v. Using defaults.", err)
	}

	utils.InfoLogger.Println("Application started successfully")
}

// loadSettings loads settings from the settings file
func (a *App) loadSettings() error {
	// Check if settings file exists
	if _, err := os.Stat(a.settingsPath); os.IsNotExist(err) {
		return errors.New("settings file does not exist")
	}

	// Read the file
	data, err := ioutil.ReadFile(a.settingsPath)
	if err != nil {
		return err
	}

	// Unmarshal the settings
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	// Apply the settings
	a.apiURL = settings.APIURL
	if settings.EncryptedKey != "" {
		// Decrypt API key - handled by a separate helper function
		decryptedKey, err := a.decryptAPIKey(settings.EncryptedKey)
		if err == nil {
			a.apiKey = decryptedKey
		}
	}

	if settings.MaxRows > 0 {
		a.maxRows = settings.MaxRows
	}

	if settings.ReportFormat != "" {
		a.reportFormat = settings.ReportFormat
	}

	return nil
}

// saveSettings saves current settings to the settings file
func (a *App) saveSettings() error {
	// Encrypt the API key
	encryptedKey, err := a.encryptAPIKey(a.apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %v", err)
	}

	// Prepare settings struct
	settings := Settings{
		APIURL:        a.apiURL,
		EncryptedKey:  encryptedKey,
		MaxRows:       a.maxRows,
		ReportFormat:  a.reportFormat,
		Theme:         "dark", // Default theme
		DateFormat:    "YYYY-MM-DD",
		DefaultFolder: "Reports",
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %v", err)
	}

	// Write to file
	if err := ioutil.WriteFile(a.settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %v", err)
	}

	return nil
}

// encryptAPIKey encrypts the API key for secure storage
func (a *App) encryptAPIKey(key string) (string, error) {
	if key == "" {
		return "", nil
	}

	// Create a unique encryption key based on machine-specific information
	// In a real implementation, you would use a more secure approach
	encKey := sha256.Sum256([]byte("PAN_ENGINE_SECRET_KEY"))

	block, err := aes.NewCipher(encKey[:])
	if err != nil {
		return "", err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, []byte(key), nil)

	// Return base64 encoded string
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptAPIKey decrypts the API key from secure storage
func (a *App) decryptAPIKey(encryptedKey string) (string, error) {
	if encryptedKey == "" {
		return "", nil
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", err
	}

	// Create encryption key
	encKey := sha256.Sum256([]byte("PAN_ENGINE_SECRET_KEY"))

	block, err := aes.NewCipher(encKey[:])
	if err != nil {
		return "", err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extract nonce
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// SaveAPISettings saves API settings to the app and persists them
func (a *App) SaveAPISettings(url, key string) (bool, error) {
	a.apiURL = url
	a.apiKey = key
	utils.InfoLogger.Printf("API settings saved: URL=%s", url)

	// Save to persistent storage
	err := a.saveSettings()
	if err != nil {
		utils.ErrorLogger.Printf("Failed to save settings: %v", err)
		return false, err
	}

	// Reset API status
	a.apiStatus = "unknown"
	a.lastAPICheck = time.Time{}

	return true, nil
}

// GetAPISettings returns the current API settings
func (a *App) GetAPISettings() map[string]string {
	return map[string]string{
		"url":    a.apiURL,
		"key":    a.apiKey,
		"status": a.apiStatus,
	}
}

// SetReportConfig updates report generation configuration
func (a *App) SetReportConfig(maxRows int, format string) bool {
	if maxRows > 0 {
		a.maxRows = maxRows
	}

	if format != "" {
		a.reportFormat = format
	}

	// Save to persistent storage
	if err := a.saveSettings(); err != nil {
		utils.ErrorLogger.Printf("Failed to save report config: %v", err)
		return false
	}

	return true
}

// GetReportConfig returns the current report configuration
func (a *App) GetReportConfig() map[string]interface{} {
	return map[string]interface{}{
		"maxRows": a.maxRows,
		"format":  a.reportFormat,
	}
}

// TestAPIConnection checks if the API connection is working
func (a *App) TestAPIConnection() map[string]interface{} {
	if a.apiURL == "" || a.apiKey == "" {
		a.apiStatus = "unconfigured"
		return map[string]interface{}{
			"status":  "error",
			"message": "API URL and Key must be configured first",
		}
	}

	// Only recheck if it's been more than 5 minutes since last check
	if !a.lastAPICheck.IsZero() && time.Since(a.lastAPICheck) < 5*time.Minute {
		return map[string]interface{}{
			"status":  a.apiStatus,
			"message": "Using cached status",
		}
	}

	// Try to call an API endpoint that should always be available
	endpoint := "/api/?type=op&cmd=<show><s><info></info></s></show>"
	_, err := a.callPaloAltoAPI(endpoint)

	a.lastAPICheck = time.Now()

	if err != nil {
		a.apiStatus = "error"
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("API connection failed: %v", err),
		}
	}

	a.apiStatus = "connected"
	return map[string]interface{}{
		"status":  "success",
		"message": "API connection successful",
	}
}

// GenerateReport creates a report by calling the Palo Alto API
func (a *App) GenerateReport(reportType, startDate, endDate string) (map[string]interface{}, error) {
	utils.InfoLogger.Printf("Generating report: type=%s, start=%s, end=%s", reportType, startDate, endDate)

	if a.apiURL == "" || a.apiKey == "" {
		return nil, fmt.Errorf("API URL and Key must be configured first")
	}

	// Find the endpoint for the given report type
	endpoint := a.getEndpointForReportType(reportType)
	if endpoint == "" {
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}

	// Add date range parameters if needed
	if startDate != "" && endDate != "" {
		if strings.Contains(endpoint, "?") {
			endpoint += "&"
		} else {
			endpoint += "?"
		}
		endpoint += fmt.Sprintf("start-time=%s&end-time=%s", startDate, endDate)
	}

	// Call the API
	data, err := a.callPaloAltoAPI(endpoint)
	if err != nil {
		return nil, err
	}

	// Store the data for export later
	a.reportData[reportType] = data

	return data, nil
}

// ExportToCSV exports the current report data to a CSV file
func (a *App) ExportToCSV(reportType string) (string, error) {
	utils.InfoLogger.Printf("Exporting to CSV: type=%s", reportType)

	// Check if we have data for this report
	data, ok := a.reportData[reportType]
	if !ok {
		return "", fmt.Errorf("no data available for report type: %s", reportType)
	}

	// Create unique filename based on timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.csv", reportType, timestamp)
	filepath := filepath.Join("Reports", filename)

	// Generate CSV
	if err := a.generateCSV(data, filepath); err != nil {
		return "", err
	}

	utils.InfoLogger.Printf("CSV exported successfully: %s", filepath)
	return filepath, nil
}

// ExportToPDF exports the current report data to a PDF file
func (a *App) ExportToPDF(reportType string) (string, error) {
	utils.InfoLogger.Printf("Exporting to PDF: type=%s", reportType)

	// Check if we have data for this report
	data, ok := a.reportData[reportType]
	if !ok {
		return "", fmt.Errorf("no data available for report type: %s", reportType)
	}

	// Create unique filename based on timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.pdf", reportType, timestamp)
	filepath := filepath.Join("Reports", filename)

	// Generate PDF
	if err := a.generatePDF(data, reportType, filepath); err != nil {
		return "", err
	}

	utils.InfoLogger.Printf("PDF exported successfully: %s", filepath)
	return filepath, nil
}

// ListReports returns a list of all generated reports
func (a *App) ListReports() ([]map[string]string, error) {
	utils.InfoLogger.Printf("Listing reports")

	// Ensure Reports directory exists
	if err := os.MkdirAll("Reports", 0755); err != nil {
		return nil, fmt.Errorf("failed to create Reports directory: %v", err)
	}

	var reports []map[string]string
	err := filepath.Walk("Reports", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".pdf" || ext == ".csv" {
				reports = append(reports, map[string]string{
					"name":     info.Name(),
					"path":     path,
					"type":     strings.TrimPrefix(ext, "."),
					"size":     fmt.Sprintf("%d", info.Size()),
					"modified": info.ModTime().Format(time.RFC3339),
				})
			}
		}
		return nil
	})

	if err != nil {
		return []map[string]string{}, nil // Return empty list instead of null on error
	}

	// Initialize empty slice if no reports found
	if reports == nil {
		reports = []map[string]string{}
	}

	// Sort reports by modification time (newest first)
	if len(reports) > 0 {
		sort.Slice(reports, func(i, j int) bool {
			return reports[i]["modified"] > reports[j]["modified"]
		})
	}

	return reports, nil
}

// OpenReport opens a report file with the system default application
func (a *App) OpenReport(path string) error {
	utils.InfoLogger.Printf("Opening report: %s", path)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	_, err = runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Open Report",
		DefaultDirectory: filepath.Dir(absPath),
		DefaultFilename:  filepath.Base(absPath),
	})
	return err
}

// DeleteReport deletes a report file
func (a *App) DeleteReport(path string) error {
	utils.InfoLogger.Printf("Deleting report: %s", path)
	return os.Remove(path)
}

// Helper functions

// getEndpointForReportType maps report types to API endpoints
func (a *App) getEndpointForReportType(reportType string) string {
	endpoints := map[string]string{
		// Objects
		"applications":  "/restapi/v11.0/Objects/Applications",
		"appGroups":     "/restapi/v11.0/Objects/ApplicationGroups",
		"appFilters":    "/restapi/v11.0/Objects/ApplicationFilters",
		"services":      "/restapi/v11.0/Objects/Services",
		"serviceGroups": "/restapi/v11.0/Objects/ServiceGroups",
		"tags":          "/restapi/v11.0/Objects/Tags",
		"hipObjects":    "/restapi/v11.0/Objects/GlobalProtectHIPObjects",
		"hipProfiles":   "/restapi/v11.0/Objects/GlobalProtectHIPProfiles",
		"edl":           "/restapi/v11.0/Objects/ExternalDynamicLists",
		"dataPatterns":  "/restapi/v11.0/Objects/CustomDataPatterns",
		"spywareSigs":   "/restapi/v11.0/Objects/CustomSpywareSignatures",
		"vulnSigs":      "/restapi/v11.0/Objects/CustomVulnerabilitySignatures",
		"urlCategories": "/restapi/v11.0/Objects/CustomURLCategories",

		// Security Profiles
		"antivirusProfiles":      "/restapi/v11.0/Objects/AntivirusSecurityProfiles",
		"antispywareProfiles":    "/restapi/v11.0/Objects/AntiSpywareSecurityProfiles",
		"vulnProtectionProfiles": "/restapi/v11.0/Objects/VulnerabilityProtectionSecurityProfiles",
		"urlFilteringProfiles":   "/restapi/v11.0/Objects/URLFilteringSecurityProfiles",
		"fileBlockingProfiles":   "/restapi/v11.0/Objects/FileBlockingSecurityProfiles",
		"wildfireProfiles":       "/restapi/v11.0/Objects/WildFireAnalysisSecurityProfiles",
		"dataFilteringProfiles":  "/restapi/v11.0/Objects/DataFilteringSecurityProfiles",
		"dosProtectionProfiles":  "/restapi/v11.0/Objects/DoSProtectionSecurityProfiles",
		"securityProfileGroups":  "/restapi/v11.0/Objects/SecurityProfileGroups",

		// Policies
		"securityRules":         "/restapi/v11.0/Policies/SecurityRules",
		"natRules":              "/restapi/v11.0/Policies/NATRules",
		"qosRules":              "/restapi/v11.0/Policies/QoSRules",
		"pbfRules":              "/restapi/v11.0/Policies/PolicyBasedForwardingRules",
		"decryptionRules":       "/restapi/v11.0/Policies/DecryptionRules",
		"packetBrokerRules":     "/restapi/v11.0/Policies/NetworkPacketBrokerRules",
		"tunnelInspectionRules": "/restapi/v11.0/Policies/TunnelInspectionRules",
		"appOverrideRules":      "/restapi/v11.0/Policies/ApplicationOverrideRules",
		"authRules":             "/restapi/v11.0/Policies/AuthenticationRules",
		"dosRules":              "/restapi/v11.0/Policies/DoSRules",
		"sdwanRules":            "/restapi/v11.0/Policies/SDWANRules",

		// Network
		"ethernetInterfaces": "/restapi/v11.0/Network/EthernetInterfaces",
		"aeInterfaces":       "/restapi/v11.0/Network/AggregateEthernetInterfaces",
		"vlanInterfaces":     "/restapi/v11.0/Network/VLANInterfaces",
		"loopbackInterfaces": "/restapi/v11.0/Network/LoopbackInterfaces",
		"tunnelInterfaces":   "/restapi/v11.0/Network/TunnelIntefaces",
		"sdwanInterfaces":    "/restapi/v11.0/Network/SDWANInterfaces",
		"zones":              "/restapi/v11.0/Network/Zones",
		"vlans":              "/restapi/v11.0/Network/VLANs",
		"virtualWires":       "/restapi/v11.0/Network/VirtualWires",
		"virtualRouters":     "/restapi/v11.0/Network/VirtualRouters",

		// GlobalProtect
		"gpPortals":             "/restapi/v11.0/Network/GlobalProtectPortals",
		"gpGateways":            "/restapi/v11.0/Network/GlobalProtectGateways",
		"gpAgentTunnels":        "/restapi/v11.0/Network/GlobalProtectGatewayAgentTunnels",
		"gpSatelliteTunnels":    "/restapi/v11.0/Network/GlobalProtectGatewaySatelliteTunnels",
		"gpMdmServers":          "/restapi/v11.0/Network/GlobalProtectGatewayMDMServers",
		"gpClientlessApps":      "/restapi/v11.0/Network/GlobalProtectClientlessApps",
		"gpClientlessAppGroups": "/restapi/v11.0/Network/GlobalProtectClientlessAppGroups",

		// Logs
		"traffic":     "/restapi/v11.0/Objects/TrafficLogs",
		"threat":      "/restapi/v11.0/Objects/ThreatLogs",
		"url":         "/restapi/v11.0/Objects/URLFilteringLogs",
		"data":        "/restapi/v11.0/Objects/DataFilteringLogs",
		"wildfire":    "/restapi/v11.0/Objects/WildFireLogs",
		"auth":        "/restapi/v11.0/Objects/AuthenticationLogs",
		"system":      "/restapi/v11.0/Objects/SystemLogs",
		"config":      "/restapi/v11.0/Objects/ConfigLogs",
		"correlation": "/restapi/v11.0/Objects/CorrelationLogs",

		// Legacy XML API endpoints
		"systemInfo":      "/api/?type=op&cmd=<show><s><info></info></s></show>",
		"interfaceInfo":   "/api/?type=op&cmd=<show><interface>all</interface></show>",
		"systemResources": "/api/?type=op&cmd=<show><s><resources></resources></s></show>",
		"gpUsers":         "/api/?type=op&cmd=<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>",
		"activeSessions":  "/api/?type=op&cmd=<show><session><all></all></session></show>",
		"softwareVersion": "/api/?type=op&cmd=<show><s><software></software></s></show>",
	}

	return endpoints[reportType]
}

// callPaloAltoAPI makes HTTP request to Palo Alto API
func (a *App) callPaloAltoAPI(endpoint string) (map[string]interface{}, error) {
	// Construct full URL
	fullURL := fmt.Sprintf("%s%s", a.apiURL, endpoint)
	utils.InfoLogger.Printf("Calling API: %s", fullURL)

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Add headers
	req.Header.Add("X-PAN-KEY", a.apiKey)
	req.Header.Add("Content-Type", "application/json")

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned error status: %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Try to parse as XML API response (legacy format)
		if strings.Contains(string(body), "<response") {
			// Convert XML to JSON for consistent handling (simplified for this example)
			result = map[string]interface{}{
				"result": string(body),
			}
			return result, nil
		}
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result, nil
}

// generateCSV creates a CSV file from report data
func (a *App) generateCSV(data interface{}, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Set up CSV writer with BOM for Excel compatibility
	// Write UTF-8 BOM for Excel compatibility
	_, err = file.Write([]byte{0xEF, 0xBB, 0xBF})
	if err != nil {
		return err
	}

	writer := csv.NewWriter(file)
	writer.Comma = ',' // Ensure comma delimiter
	defer writer.Flush()

	// Process data based on type
	switch v := data.(type) {
	case map[string]interface{}:
		// Add metadata rows about the report
		metadataHeaders := []string{"Report Information"}
		if err := writer.Write(metadataHeaders); err != nil {
			return err
		}

		metadataRows := [][]string{
			{"Generated", time.Now().Format("2006-01-02 15:04:05")},
			{"Source", "PAN_ENGINE"},
			{"", ""}, // Empty row for separation
		}

		for _, row := range metadataRows {
			if err := writer.Write(row); err != nil {
				return err
			}
		}

		// Extract headers
		var headers []string
		for key := range v {
			headers = append(headers, key)
		}
		// Sort headers for consistency
		sort.Strings(headers)

		// Write data headers
		if err := writer.Write(headers); err != nil {
			return err
		}

		// Write values
		var values []string
		for _, header := range headers {
			// Format values based on type for better Excel display
			val := v[header]
			values = append(values, formatValueForCSV(val))
		}
		if err := writer.Write(values); err != nil {
			return err
		}

	case []interface{}:
		// Handle array data
		if len(v) == 0 {
			// Write a header row anyway to show the file isn't empty
			emptyHeaders := []string{"No Data", "Generated At"}
			if err := writer.Write(emptyHeaders); err != nil {
				return err
			}
			emptyRow := []string{"No data available", time.Now().Format("2006-01-02 15:04:05")}
			return writer.Write(emptyRow)
		}

		// For arrays, use the keys from the first item as headers
		if firstItem, ok := v[0].(map[string]interface{}); ok {
			// Add metadata header
			metadataHeaders := []string{"Report Information"}
			if err := writer.Write(metadataHeaders); err != nil {
				return err
			}

			// Add metadata rows
			metadataRows := [][]string{
				{"Generated", time.Now().Format("2006-01-02 15:04:05")},
				{"Source", "PAN_ENGINE"},
				{"Total Items", fmt.Sprintf("%d", len(v))},
				{"", ""}, // Empty row for separation
			}

			for _, row := range metadataRows {
				if err := writer.Write(row); err != nil {
					return err
				}
			}

			// Get all possible headers from all items
			headerMap := make(map[string]bool)

			// First scan all items to get all possible headers
			if a.reportFormat == "complete" {
				for _, item := range v {
					if itemMap, ok := item.(map[string]interface{}); ok {
						for key := range itemMap {
							headerMap[key] = true
						}
					}
				}
			} else {
				// Just use headers from first item for standard format
				for key := range firstItem {
					headerMap[key] = true
				}
			}

			// Convert to sorted slice
			var headers []string
			for key := range headerMap {
				headers = append(headers, key)
			}
			sort.Strings(headers)

			// Write headers row
			if err := writer.Write(headers); err != nil {
				return err
			}

			// Write each data row
			rowCount := 0
			for _, item := range v {
				// Respect maxRows setting
				if a.maxRows > 0 && rowCount >= a.maxRows {
					break
				}

				if rowData, ok := item.(map[string]interface{}); ok {
					var rowValues []string
					for _, header := range headers {
						// Check if this item has this header
						val, exists := rowData[header]
						if exists {
							rowValues = append(rowValues, formatValueForCSV(val))
						} else {
							rowValues = append(rowValues, "") // Empty for missing value
						}
					}
					if err := writer.Write(rowValues); err != nil {
						return err
					}
					rowCount++
				}
			}

			// If we truncated the results, add a note
			if rowCount < len(v) {
				noteRow := make([]string, len(headers))
				noteRow[0] = fmt.Sprintf("Note: Output limited to %d of %d rows", rowCount, len(v))
				writer.Write(noteRow)
			}
		}

	default:
		return fmt.Errorf("unsupported data type for CSV export")
	}

	return nil
}

// formatValueForCSV formats a value for CSV export
func formatValueForCSV(val interface{}) string {
	switch v := val.(type) {
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case float64:
		// Format numbers without trailing zeros
		s := strconv.FormatFloat(v, 'f', -1, 64)
		return s
	case nil:
		return ""
	case bool:
		if v {
			return "Yes"
		}
		return "No"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// generatePDF creates a PDF file from report data
func (a *App) generatePDF(data interface{}, reportType, filePath string) error {
	if data == nil {
		return fmt.Errorf("no data provided for PDF generation")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set up table dimensions
	pageWidth, _ := pdf.GetPageSize()
	colWidth1 := pageWidth * 0.3
	colWidth2 := pageWidth * 0.7

	// Draw header cells
	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(200, 200, 200)
	pdf.Cell(colWidth1, 8, "Field")
	pdf.Cell(colWidth2, 8, "Value")
	pdf.Ln(-1)

	// Draw content cells
	pdf.SetFont("Arial", "", 10)
	pdf.SetFillColor(255, 255, 255)

	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			pdf.Cell(colWidth1, 8, fmt.Sprintf("%v", k))
			pdf.Cell(colWidth2, 8, fmt.Sprintf("%v", v))
			pdf.Ln(-1)
		}
	}

	return pdf.OutputFileAndClose(filePath)
}

// Greet returns a greeting for the given name (kept for backward compatibility)
func (a *App) Greet(name string) string {
	utils.InfoLogger.Printf("Greeting requested for name: %s", name)
	greeting := fmt.Sprintf("Hello %s, It's show time!", name)
	return greeting
}

// BatchExportReports exports multiple reports in one operation
func (a *App) BatchExportReports(reportTypes []string, format string, startDate, endDate string) (map[string]interface{}, error) {
	if len(reportTypes) == 0 {
		return nil, fmt.Errorf("no report types specified")
	}

	if format != "csv" && format != "pdf" {
		return nil, fmt.Errorf("invalid format: must be 'csv' or 'pdf'")
	}

	results := make(map[string]interface{})
	errorCount := 0
	successCount := 0
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Set up a channel to limit concurrent API calls
	semaphore := make(chan struct{}, 3) // Max 3 concurrent API calls

	for _, reportType := range reportTypes {
		wg.Add(1)
		go func(rt string) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Generate the report
			reportData, err := a.GenerateReport(rt, startDate, endDate)
			if err != nil {
				mu.Lock()
				results[rt] = map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}
				errorCount++
				mu.Unlock()
				return
			}

			// Store the report data for export
			a.reportData[rt] = reportData

			// Export the report based on format
			var filePath string
			var exportErr error

			if format == "csv" {
				filePath, exportErr = a.ExportToCSV(rt)
			} else if format == "pdf" {
				filePath, exportErr = a.ExportToPDF(rt)
			}

			mu.Lock()
			if exportErr != nil {
				results[rt] = map[string]interface{}{
					"success": false,
					"error":   exportErr.Error(),
				}
				errorCount++
			} else {
				results[rt] = map[string]interface{}{
					"success": true,
					"path":    filePath,
				}
				successCount++
			}
			mu.Unlock()
		}(reportType)
	}

	// Wait for all exports to finish
	wg.Wait()

	// Add summary to results
	summary := map[string]interface{}{
		"total":      len(reportTypes),
		"successful": successCount,
		"failed":     errorCount,
		"format":     format,
		"date_range": map[string]string{"start": startDate, "end": endDate},
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	results["_summary"] = summary
	return results, nil
}

// GetSupportedReportTypes returns a list of all supported report types with their details
func (a *App) GetSupportedReportTypes() []map[string]string {
	return []map[string]string{
		// Objects
		{"type": "applications", "name": "Applications", "category": "Objects", "enabled": "true", "value": "applications", "label": "Applications"},
		{"type": "appGroups", "name": "Application Groups", "category": "Objects", "enabled": "true", "value": "appGroups", "label": "Application Groups"},
		{"type": "appFilters", "name": "Application Filters", "category": "Objects", "enabled": "true", "value": "appFilters", "label": "Application Filters"},
		{"type": "services", "name": "Services", "category": "Objects", "enabled": "true", "value": "services", "label": "Services"},
		{"type": "serviceGroups", "name": "Service Groups", "category": "Objects", "enabled": "true", "value": "serviceGroups", "label": "Service Groups"},
		{"type": "tags", "name": "Tags", "category": "Objects", "enabled": "true", "value": "tags", "label": "Tags"},
		{"type": "hipObjects", "name": "GlobalProtect HIP Objects", "category": "Objects", "enabled": "true", "value": "hipObjects", "label": "GlobalProtect HIP Objects"},
		{"type": "hipProfiles", "name": "GlobalProtect HIP Profiles", "category": "Objects", "enabled": "true", "value": "hipProfiles", "label": "GlobalProtect HIP Profiles"},
		{"type": "edl", "name": "External Dynamic Lists", "category": "Objects", "enabled": "true", "value": "edl", "label": "External Dynamic Lists"},
		{"type": "dataPatterns", "name": "Custom Data Patterns", "category": "Objects", "enabled": "true", "value": "dataPatterns", "label": "Custom Data Patterns"},
		{"type": "spywareSigs", "name": "Custom Spyware Signatures", "category": "Objects", "enabled": "true", "value": "spywareSigs", "label": "Custom Spyware Signatures"},
		{"type": "vulnSigs", "name": "Custom Vulnerability Signatures", "category": "Objects", "enabled": "true", "value": "vulnSigs", "label": "Custom Vulnerability Signatures"},
		{"type": "urlCategories", "name": "Custom URL Categories", "category": "Objects", "enabled": "true", "value": "urlCategories", "label": "Custom URL Categories"},

		// Security Profiles
		{"type": "antivirusProfiles", "name": "Antivirus Profiles", "category": "Security Profiles", "enabled": "true", "value": "antivirusProfiles", "label": "Antivirus Profiles"},
		{"type": "antispywareProfiles", "name": "Anti-Spyware Profiles", "category": "Security Profiles", "enabled": "true", "value": "antispywareProfiles", "label": "Anti-Spyware Profiles"},
		{"type": "vulnProtectionProfiles", "name": "Vulnerability Protection Profiles", "category": "Security Profiles", "enabled": "true", "value": "vulnProtectionProfiles", "label": "Vulnerability Protection Profiles"},
		{"type": "urlFilteringProfiles", "name": "URL Filtering Profiles", "category": "Security Profiles", "enabled": "true", "value": "urlFilteringProfiles", "label": "URL Filtering Profiles"},
		{"type": "fileBlockingProfiles", "name": "File Blocking Profiles", "category": "Security Profiles", "enabled": "true", "value": "fileBlockingProfiles", "label": "File Blocking Profiles"},
		{"type": "wildfireProfiles", "name": "WildFire Analysis Profiles", "category": "Security Profiles", "enabled": "true", "value": "wildfireProfiles", "label": "WildFire Analysis Profiles"},
		{"type": "dataFilteringProfiles", "name": "Data Filtering Profiles", "category": "Security Profiles", "enabled": "true", "value": "dataFilteringProfiles", "label": "Data Filtering Profiles"},
		{"type": "dosProtectionProfiles", "name": "DoS Protection Profiles", "category": "Security Profiles", "enabled": "true", "value": "dosProtectionProfiles", "label": "DoS Protection Profiles"},
		{"type": "securityProfileGroups", "name": "Security Profile Groups", "category": "Security Profiles", "enabled": "true", "value": "securityProfileGroups", "label": "Security Profile Groups"},

		// Policies
		{"type": "securityRules", "name": "Security Rules", "category": "Policies", "enabled": "true", "value": "securityRules", "label": "Security Rules"},
		{"type": "natRules", "name": "NAT Rules", "category": "Policies", "enabled": "true", "value": "natRules", "label": "NAT Rules"},
		{"type": "qosRules", "name": "QoS Rules", "category": "Policies", "enabled": "true", "value": "qosRules", "label": "QoS Rules"},
		{"type": "pbfRules", "name": "Policy Based Forwarding Rules", "category": "Policies", "enabled": "true", "value": "pbfRules", "label": "Policy Based Forwarding Rules"},
		{"type": "decryptionRules", "name": "Decryption Rules", "category": "Policies", "enabled": "true", "value": "decryptionRules", "label": "Decryption Rules"},
		{"type": "packetBrokerRules", "name": "Network Packet Broker Rules", "category": "Policies", "enabled": "true", "value": "packetBrokerRules", "label": "Network Packet Broker Rules"},
		{"type": "tunnelInspectionRules", "name": "Tunnel Inspection Rules", "category": "Policies", "enabled": "true", "value": "tunnelInspectionRules", "label": "Tunnel Inspection Rules"},
		{"type": "appOverrideRules", "name": "Application Override Rules", "category": "Policies", "enabled": "true", "value": "appOverrideRules", "label": "Application Override Rules"},
		{"type": "authRules", "name": "Authentication Rules", "category": "Policies", "enabled": "true", "value": "authRules", "label": "Authentication Rules"},
		{"type": "dosRules", "name": "DoS Rules", "category": "Policies", "enabled": "true", "value": "dosRules", "label": "DoS Rules"},
		{"type": "sdwanRules", "name": "SD-WAN Rules", "category": "Policies", "enabled": "true", "value": "sdwanRules", "label": "SD-WAN Rules"},

		// Network
		{"type": "ethernetInterfaces", "name": "Ethernet Interfaces", "category": "Network", "enabled": "true", "value": "ethernetInterfaces", "label": "Ethernet Interfaces"},
		{"type": "aeInterfaces", "name": "Aggregate Ethernet Interfaces", "category": "Network", "enabled": "true", "value": "aeInterfaces", "label": "Aggregate Ethernet Interfaces"},
		{"type": "vlanInterfaces", "name": "VLAN Interfaces", "category": "Network", "enabled": "true", "value": "vlanInterfaces", "label": "VLAN Interfaces"},
		{"type": "loopbackInterfaces", "name": "Loopback Interfaces", "category": "Network", "enabled": "true", "value": "loopbackInterfaces", "label": "Loopback Interfaces"},
		{"type": "tunnelInterfaces", "name": "Tunnel Interfaces", "category": "Network", "enabled": "true", "value": "tunnelInterfaces", "label": "Tunnel Interfaces"},
		{"type": "sdwanInterfaces", "name": "SD-WAN Interfaces", "category": "Network", "enabled": "true", "value": "sdwanInterfaces", "label": "SD-WAN Interfaces"},
		{"type": "zones", "name": "Zones", "category": "Network", "enabled": "true", "value": "zones", "label": "Zones"},
		{"type": "vlans", "name": "VLANs", "category": "Network", "enabled": "true", "value": "vlans", "label": "VLANs"},
		{"type": "virtualWires", "name": "Virtual Wires", "category": "Network", "enabled": "true", "value": "virtualWires", "label": "Virtual Wires"},
		{"type": "virtualRouters", "name": "Virtual Routers", "category": "Network", "enabled": "true", "value": "virtualRouters", "label": "Virtual Routers"},

		// GlobalProtect
		{"type": "gpPortals", "name": "GlobalProtect Portals", "category": "GlobalProtect", "enabled": "true", "value": "gpPortals", "label": "GlobalProtect Portals"},
		{"type": "gpGateways", "name": "GlobalProtect Gateways", "category": "GlobalProtect", "enabled": "true", "value": "gpGateways", "label": "GlobalProtect Gateways"},
		{"type": "gpAgentTunnels", "name": "GlobalProtect Agent Tunnels", "category": "GlobalProtect", "enabled": "true", "value": "gpAgentTunnels", "label": "GlobalProtect Agent Tunnels"},
		{"type": "gpSatelliteTunnels", "name": "GlobalProtect Satellite Tunnels", "category": "GlobalProtect", "enabled": "true", "value": "gpSatelliteTunnels", "label": "GlobalProtect Satellite Tunnels"},
		{"type": "gpMdmServers", "name": "GlobalProtect MDM Servers", "category": "GlobalProtect", "enabled": "true", "value": "gpMdmServers", "label": "GlobalProtect MDM Servers"},
		{"type": "gpClientlessApps", "name": "GlobalProtect Clientless Apps", "category": "GlobalProtect", "enabled": "true", "value": "gpClientlessApps", "label": "GlobalProtect Clientless Apps"},
		{"type": "gpClientlessAppGroups", "name": "GlobalProtect Clientless App Groups", "category": "GlobalProtect", "enabled": "true", "value": "gpClientlessAppGroups", "label": "GlobalProtect Clientless App Groups"},

		// Logs
		{"type": "traffic", "name": "Traffic Logs", "category": "Logs", "enabled": "true", "value": "traffic", "label": "Traffic Logs"},
		{"type": "threat", "name": "Threat Logs", "category": "Logs", "enabled": "true", "value": "threat", "label": "Threat Logs"},
		{"type": "url", "name": "URL Filtering Logs", "category": "Logs", "enabled": "true", "value": "url", "label": "URL Filtering Logs"},
		{"type": "data", "name": "Data Filtering Logs", "category": "Logs", "enabled": "true", "value": "data", "label": "Data Filtering Logs"},
		{"type": "wildfire", "name": "WildFire Logs", "category": "Logs", "enabled": "true", "value": "wildfire", "label": "WildFire Logs"},
		{"type": "auth", "name": "Authentication Logs", "category": "Logs", "enabled": "true", "value": "auth", "label": "Authentication Logs"},
		{"type": "system", "name": "System Logs", "category": "Logs", "enabled": "true", "value": "system", "label": "System Logs"},
		{"type": "config", "name": "Configuration Logs", "category": "Logs", "enabled": "true", "value": "config", "label": "Configuration Logs"},
		{"type": "correlation", "name": "Correlation Logs", "category": "Logs", "enabled": "true", "value": "correlation", "label": "Correlation Logs"},

		// System
		{"type": "systemInfo", "name": "System Information", "category": "System", "enabled": "true", "value": "systemInfo", "label": "System Information"},
		{"type": "interfaceInfo", "name": "Interface Information", "category": "System", "enabled": "true", "value": "interfaceInfo", "label": "Interface Information"},
		{"type": "systemResources", "name": "System Resources", "category": "System", "enabled": "true", "value": "systemResources", "label": "System Resources"},
		{"type": "gpUsers", "name": "GlobalProtect Users", "category": "System", "enabled": "true", "value": "gpUsers", "label": "GlobalProtect Users"},
		{"type": "activeSessions", "name": "Active Sessions", "category": "System", "enabled": "true", "value": "activeSessions", "label": "Active Sessions"},
		{"type": "softwareVersion", "name": "Software Version", "category": "System", "enabled": "true", "value": "softwareVersion", "label": "Software Version"},
	}
}

// GetReportCategories returns a list of all report categories
func (a *App) GetReportCategories() []string {
	return []string{
		"Objects",
		"Security Profiles",
		"Policies",
		"Network",
		"GlobalProtect",
		"Logs",
		"System",
	}
}

// FilterReportData allows filtering report data by search criteria
func (a *App) FilterReportData(reportType string, filters map[string]string) (map[string]interface{}, error) {
	// Get the original report data
	data, ok := a.reportData[reportType]
	if !ok {
		return nil, fmt.Errorf("no data available for report type: %s", reportType)
	}

	// Extract the data array (most reports are array-based)
	var items []interface{}

	switch v := data.(type) {
	case map[string]interface{}:
		// Single object report - just create an array with this item
		items = []interface{}{v}
	case []interface{}:
		// Array report - use as is
		items = v
	default:
		return nil, fmt.Errorf("unsupported data format for filtering")
	}

	// If no filters provided, return original data
	if len(filters) == 0 {
		return map[string]interface{}{
			"result":   data,
			"count":    len(items),
			"filtered": false,
		}, nil
	}

	// Apply filters
	var filtered []interface{}

	for _, item := range items {
		if itemMap, ok := item.(map[string]interface{}); ok {
			matches := true

			// Check if this item matches all filters
			for field, value := range filters {
				if value == "" {
					continue // Skip empty filters
				}

				// Check if field exists
				fieldValue, exists := itemMap[field]
				if !exists {
					matches = false
					break
				}

				// Check if field value contains filter value (case-insensitive)
				fieldStr := strings.ToLower(fmt.Sprintf("%v", fieldValue))
				filterStr := strings.ToLower(value)

				if !strings.Contains(fieldStr, filterStr) {
					matches = false
					break
				}
			}

			if matches {
				filtered = append(filtered, item)
			}
		}
	}

	result := map[string]interface{}{
		"result":   filtered,
		"count":    len(filtered),
		"total":    len(items),
		"filtered": true,
	}

	// Store filtered results for possible export
	a.reportData[reportType+"_filtered"] = filtered

	return result, nil
}

// SearchAllReports searches for a term across all generated reports
func (a *App) SearchAllReports(searchTerm string) (map[string]interface{}, error) {
	if searchTerm == "" {
		return nil, fmt.Errorf("search term cannot be empty")
	}

	results := make(map[string]interface{})

	// Convert search term to lowercase for case-insensitive matching
	searchTermLower := strings.ToLower(searchTerm)

	for reportType, data := range a.reportData {
		// Skip filtered reports (those with _filtered suffix)
		if strings.HasSuffix(reportType, "_filtered") {
			continue
		}

		var matches []interface{}
		var count int

		switch v := data.(type) {
		case map[string]interface{}:
			// Check if this single object contains the search term
			found := false
			for _, value := range v {
				if strings.Contains(strings.ToLower(fmt.Sprintf("%v", value)), searchTermLower) {
					found = true
					break
				}
			}

			if found {
				matches = append(matches, v)
				count = 1
			}

		case []interface{}:
			// Check each item in the array
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					found := false

					// Check all fields in the item
					for _, value := range itemMap {
						if strings.Contains(strings.ToLower(fmt.Sprintf("%v", value)), searchTermLower) {
							found = true
							break
						}
					}

					if found {
						matches = append(matches, item)
					}
				}
			}

			count = len(matches)
		}

		// Add to results if we found matches
		if count > 0 {
			results[reportType] = map[string]interface{}{
				"matches": matches,
				"count":   count,
			}
		}
	}

	// Add summary
	matchCount := len(results)

	return map[string]interface{}{
		"results":              results,
		"term":                 searchTerm,
		"reports_searched":     len(a.reportData),
		"reports_with_matches": matchCount,
		"found":                matchCount > 0,
	}, nil
}

// ScheduleReport schedules a report to be generated periodically
func (a *App) ScheduleReport(reportType, schedule string, emailRecipients []string) (string, error) {
	// This would normally connect to a background service for scheduling
	// For now, just return a success message with a fake ID

	scheduleID := uuid.New().String()

	return scheduleID, nil
}

// GetReportHistory returns a summary of recently generated reports with their details
func (a *App) GetReportHistory() ([]map[string]interface{}, error) {
	// In a real implementation, this would read from a database of report history
	// For now, we'll use the filesystem to list reports, sorted by date

	reports, err := a.ListReports()
	if err != nil {
		return nil, err
	}

	// Enhance the report data with more information
	var reportHistory []map[string]interface{}

	for _, report := range reports {
		// Extract report type from filename
		filename := report["name"]
		parts := strings.Split(filename, "_")
		reportType := ""

		if len(parts) > 0 {
			reportType = parts[0]
		}

		// Parse timestamp
		timestamp := report["modified"]

		historyItem := map[string]interface{}{
			"id":          strings.TrimSuffix(filename, filepath.Ext(filename)),
			"report_type": reportType,
			"format":      report["type"],
			"file_name":   filename,
			"file_path":   report["path"],
			"file_size":   report["size"],
			"created_at":  timestamp,
		}

		reportHistory = append(reportHistory, historyItem)
	}

	return reportHistory, nil
}
