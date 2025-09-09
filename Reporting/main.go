package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Config struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

type APIEndpoint struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description"`
}

type ReportFile struct {
	Name         string `json:"name"`
	CreatedAt    string `json:"created_at"`
	Type         string `json:"type"`
	DownloadPath string `json:"download_path"`
}

var apiEndpoints = []APIEndpoint{
	{
		Name:        "System Info",
		Endpoint:    "/api/?type=op&cmd=<show><system><info></info></system></show>",
		Description: "Get system information",
	},
	{
		Name:        "Interface Info",
		Endpoint:    "/api/?type=op&cmd=<show><interface>all</interface></show>",
		Description: "Get interface information",
	},
	{
		Name:        "Security Rules",
		Endpoint:    "/api/?type=config&action=get&xpath=/config/devices/entry/vsys/entry/rulebase/security/rules",
		Description: "Get security rules configuration",
	},
	{
		Name:        "System Resources",
		Endpoint:    "/api/?type=op&cmd=<show><system><resources></resources></system></show>",
		Description: "Get system resource utilization",
	},
	{
		Name:        "Traffic Logs",
		Endpoint:    "/api/?type=log&log-type=traffic",
		Description: "Get traffic logs",
	},
	{
		Name:        "Threat Logs",
		Endpoint:    "/api/?type=log&log-type=threat",
		Description: "Get threat logs",
	},
	{
		Name:        "URL Filtering Logs",
		Endpoint:    "/api/?type=log&log-type=url",
		Description: "Get URL filtering logs",
	},
	{
		Name:        "GlobalProtect Users",
		Endpoint:    "/api/?type=op&cmd=<show><global-protect-gateway><current-user></current-user></global-protect-gateway></show>",
		Description: "Get current GlobalProtect users",
	},
	{
		Name:        "Active Sessions",
		Endpoint:    "/api/?type=op&cmd=<show><session><all></all></session></show>",
		Description: "Get all active sessions",
	},
	{
		Name:        "System Software Version",
		Endpoint:    "/api/?type=op&cmd=<show><system><software></software></system></show>",
		Description: "Get system software version information",
	},
}

func main() {
	// Ensure config directory exists
	configDir := "config"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Create reports directory if it doesn't exist
	if err := os.MkdirAll("reports", 0755); err != nil {
		log.Fatal(err)
	}

	// Set up routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/save-config", handleSaveConfig)
	http.HandleFunc("/generate-report", handleGenerateReport)
	http.HandleFunc("/list-reports", handleListReports)
	http.HandleFunc("/reports/", handleDownloadReport)

	fmt.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "API server running",
	})
}

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := Config{
		APIURL: r.FormValue("api_url"),
		APIKey: r.FormValue("api_key"),
	}

	configFile := filepath.Join("config", "config.json")
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func loadConfig() (Config, error) {
	var config Config
	configFile := filepath.Join("config", "config.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return config, err
	}

	err = json.Unmarshal(data, &config)
	return config, err
}

func handleGenerateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpoint := r.FormValue("endpoint")
	if endpoint == "" {
		http.Error(w, "No endpoint selected", http.StatusBadRequest)
		return
	}

	config, err := loadConfig()
	if err != nil {
		http.Error(w, "Failed to load configuration", http.StatusInternalServerError)
		return
	}

	if config.APIURL == "" || config.APIKey == "" {
		http.Error(w, "API configuration not set", http.StatusBadRequest)
		return
	}

	err = generateReport(config.APIURL, config.APIKey, endpoint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleListReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reports, err := listReportFiles()
	if err != nil {
		http.Error(w, "Failed to list reports", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func handleDownloadReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/reports/")
	if filename == "" {
		http.Error(w, "No file specified", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("reports", filename)
	if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean("reports")) {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

	if strings.HasSuffix(filename, ".pdf") {
		w.Header().Set("Content-Type", "application/pdf")
	} else if strings.HasSuffix(filename, ".csv") {
		w.Header().Set("Content-Type", "text/csv")
	}

	http.ServeFile(w, r, filePath)
}

func listReportFiles() ([]ReportFile, error) {
	var reports []ReportFile
	err := filepath.Walk("reports", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".pdf" || ext == ".csv" {
				reports = append(reports, ReportFile{
					Name:         info.Name(),
					CreatedAt:    info.ModTime().Format(time.RFC3339),
					Type:         strings.TrimPrefix(ext, "."),
					DownloadPath: "/reports/" + info.Name(),
				})
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort reports by creation time (newest first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].CreatedAt > reports[j].CreatedAt
	})

	return reports, nil
}
