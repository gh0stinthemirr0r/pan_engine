package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-pdf/fpdf"
)

type ReportData struct {
	Data interface{} `json:"data"`
}

func generateReport(apiURL, apiKey, endpoint string) error {
	// Create unique filename based on timestamp
	timestamp := time.Now().Format("20060102_150405")
	baseFilename := fmt.Sprintf("pan_engine_%s", timestamp)

	// Construct full API URL
	fullURL := fmt.Sprintf("%s%s", apiURL, endpoint)

	// Create HTTP request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Add API key to header
	req.Header.Add("X-PAN-KEY", apiKey)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Parse JSON response
	var reportData ReportData
	if err := json.Unmarshal(body, &reportData); err != nil {
		return fmt.Errorf("error parsing JSON: %v", err)
	}

	// Generate CSV report
	if err := generateCSVReport(reportData.Data, baseFilename); err != nil {
		return fmt.Errorf("error generating CSV: %v", err)
	}

	// Generate PDF report
	if err := generatePDFReport(reportData.Data, baseFilename); err != nil {
		return fmt.Errorf("error generating PDF: %v", err)
	}

	return nil
}

func generateCSVReport(data interface{}, baseFilename string) error {
	filename := filepath.Join("reports", baseFilename+".csv")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Convert data to map for CSV writing
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data format")
	}

	// Write headers
	var headers []string
	for key := range dataMap {
		headers = append(headers, key)
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write values
	var values []string
	for _, header := range headers {
		values = append(values, fmt.Sprintf("%v", dataMap[header]))
	}
	if err := writer.Write(values); err != nil {
		return err
	}

	return nil
}

func generatePDFReport(data interface{}, baseFilename string) error {
	filename := filepath.Join("reports", baseFilename+".pdf")

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Set font
	pdf.SetFont("Arial", "B", 16)

	// Add title
	pdf.Cell(40, 10, "Palo Alto Report")
	pdf.Ln(10)

	// Set font for content
	pdf.SetFont("Arial", "", 12)

	// Convert data to map for PDF writing
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data format")
	}

	// Write data
	for key, value := range dataMap {
		pdf.Cell(40, 10, fmt.Sprintf("%s: %v", key, value))
		pdf.Ln(10)
	}

	return pdf.OutputFileAndClose(filename)
}
