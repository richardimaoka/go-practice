package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// HTML template for the upload form
const uploadFormHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>CSV to JSON Converter</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            text-align: center;
            margin-bottom: 30px;
        }
        .upload-form {
            text-align: center;
        }
        input[type="file"] {
            margin: 20px 0;
            padding: 10px;
            border: 2px dashed #ccc;
            border-radius: 5px;
            background: #f9f9f9;
        }
        input[type="submit"] {
            background: #007bff;
            color: white;
            padding: 12px 30px;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 16px;
        }
        input[type="submit"]:hover {
            background: #0056b3;
        }
        .info {
            margin-top: 20px;
            padding: 15px;
            background: #e7f3ff;
            border-left: 4px solid #007bff;
            border-radius: 3px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>CSV to JSON Converter</h1>
        <form class="upload-form" action="/convert" method="post" enctype="multipart/form-data">
            <div>
                <input type="file" name="csvfile" accept=".csv" required>
            </div>
            <div>
                <input type="submit" value="Convert to JSON">
            </div>
        </form>
        <div class="info">
            <strong>Instructions:</strong>
            <ul style="text-align: left;">
                <li>Select a CSV file from your computer</li>
                <li>Click "Convert to JSON" to upload and convert</li>
                <li>The converted JSON file will automatically download</li>
                <li>The first row of your CSV will be treated as column headers</li>
            </ul>
        </div>
    </div>
</body>
</html>
`

// convertValue attempts to convert string values to appropriate JSON types
func convertValue(value string) interface{} {
	// Trim whitespace
	value = strings.TrimSpace(value)

	// Return null for empty values
	if value == "" {
		return nil
	}

	// Try to convert to integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try to convert to float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Try to convert to boolean
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal
	}

	// Return as string if no other conversion works
	return value
}

// csvToJSON converts CSV data to JSON
func csvToJSON(csvData io.Reader) ([]byte, error) {
	reader := csv.NewReader(csvData)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %v", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// First record contains headers
	headers := records[0]

	// Convert each record to a map
	var jsonData []map[string]interface{}

	for i := 1; i < len(records); i++ {
		record := records[i]
		rowData := make(map[string]interface{})

		// Handle cases where row has fewer columns than headers
		for j, header := range headers {
			header = strings.TrimSpace(header)
			if j < len(record) {
				rowData[header] = convertValue(record[j])
			} else {
				rowData[header] = nil
			}
		}

		jsonData = append(jsonData, rowData)
	}

	// Convert to JSON with proper indentation
	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error converting to JSON: %v", err)
	}

	return jsonBytes, nil
}

// uploadHandler serves the upload form
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("upload").Parse(uploadFormHTML)
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}

// convertHandler handles the CSV upload and conversion
func convertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max file size
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, fileHeader, err := r.FormFile("csvfile")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file extension
	if filepath.Ext(fileHeader.Filename) != ".csv" {
		http.Error(w, "Please upload a CSV file", http.StatusBadRequest)
		return
	}

	// Convert CSV to JSON
	jsonData, err := csvToJSON(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Conversion error: %v", err), http.StatusBadRequest)
		return
	}

	// Generate filename for JSON download
	baseFilename := strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
	jsonFilename := baseFilename + ".json"

	// Set headers for file download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", jsonFilename))
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))

	// Write JSON data to response
	w.Write(jsonData)
}

func main() {
	// Set up routes
	http.HandleFunc("/", uploadHandler)
	http.HandleFunc("/convert", convertHandler)

	// Start server
	port := ":8080"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	fmt.Println("Upload CSV files and convert them to JSON!")

	log.Fatal(http.ListenAndServe(port, nil))
}
