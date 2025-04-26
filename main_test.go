package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/arcgis"
	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
	"github.com/Sudo-Ivan/arcgis-utils/pkg/export"
)

func TestMain(m *testing.M) {
	// Set up test environment
	useColor = false // Disable color output for tests
	os.Exit(m.Run())
}

func TestNormalizeArcGISURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Basic HTTPS FeatureServer", "https://services.arcgis.com/abc/arcgis/rest/services/MyService/FeatureServer/0", "https://services.arcgis.com/abc/ArcGIS/rest/services/MyService/FeatureServer/0"},
		{"Basic HTTP MapServer with slash", "http://example.com/arcgis/rest/services/MyMap/MapServer/", "http://example.com/ArcGIS/rest/services/MyMap/MapServer/"},
		{"No scheme adds HTTPS", "myserver.com/arcgis/rest/services/Data/FeatureServer", "https://myserver.com/ArcGIS/rest/services/Data/FeatureServer/"},
		{"Lower case parts", "https://test.com/arcgis/rest/services/lower/featureserver/1", "https://test.com/ArcGIS/rest/services/lower/FeatureServer/1"},
		{"Mixed case parts", "https://mixed.org/ArcGIS/rest/SERVICES/MixedCase/MapServer", "https://mixed.org/ArcGIS/rest/services/MixedCase/MapServer/"},
		{"Query param f removed", "https://query.net/arcgis/rest/services/Query/FeatureServer/0?f=json", "https://query.net/ArcGIS/rest/services/Query/FeatureServer/0"},
		{"Other query params kept", "https://query.net/arcgis/rest/services/Query/FeatureServer/0?token=123&f=pjson", "https://query.net/ArcGIS/rest/services/Query/FeatureServer/0?token=123"},
		{"AGOL Item URL unchanged", "https://www.arcgis.com/home/item.html?id=abcdef123456", "https://www.arcgis.com/home/item.html?id=abcdef123456"},
		{"Trailing slash added to Server URL", "https://server.com/arcgis/rest/services/NeedsSlash/MapServer", "https://server.com/ArcGIS/rest/services/NeedsSlash/MapServer/"},
		{"Trailing slash kept on Server URL", "https://server.com/arcgis/rest/services/KeepSlash/FeatureServer/", "https://server.com/ArcGIS/rest/services/KeepSlash/FeatureServer/"},
		{"No trailing slash on Layer URL", "https://server.com/arcgis/rest/services/NoSlashLayer/FeatureServer/5/", "https://server.com/ArcGIS/rest/services/NoSlashLayer/FeatureServer/5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := arcgis.NormalizeArcGISURL(tt.input)
			if actual != tt.expected {
				t.Errorf("NormalizeArcGISURL(%q): expected %q, got %q", tt.input, tt.expected, actual)
			}
		})
	}
}

func TestIsValidHTTPURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"Valid HTTPS", "https://example.com", true},
		{"Valid HTTP", "http://example.com/path", true},
		{"No Scheme", "example.com", false},
		{"Invalid Scheme", "ftp://example.com", false},
		{"Just Scheme", "http://", true}, // url.Parse considers this valid
		{"Empty String", "", false},
		{"Garbage Input", "://?##", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := arcgis.IsValidHTTPURL(tt.input); got != tt.want {
				t.Errorf("IsValidHTTPURL(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsArcGISOnlineItemURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"Valid AGOL Item URL", "https://www.arcgis.com/home/item.html?id=abc123", true},
		{"Valid AGOL Item URL with other params", "https://www.arcgis.com/home/item.html?id=abc123&other=param", true},
		{"Invalid URL", "https://example.com", false},
		{"Empty String", "", false},
		{"Just domain", "arcgis.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := arcgis.IsArcGISOnlineItemURL(tt.input); got != tt.want {
				t.Errorf("IsArcGISOnlineItemURL(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestProcessSelectedLayer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock layer metadata response
		if r.URL.Path == "/0" && r.URL.Query().Get("f") == "json" {
			layer := arcgis.Layer{
				Name: "Test Layer",
				Type: "Feature Layer",
				DrawingInfo: &arcgis.DrawingInfo{
					Renderer: &arcgis.Renderer{
						Type: "simple",
						DefaultSymbol: &arcgis.Symbol{
							Type: "esriPMS",
							URL:  "test.png",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(layer)
			return
		}

		// Mock features response
		if r.URL.Path == "/0/query" {
			features := arcgis.FeatureResponse{
				Features: []arcgis.Feature{
					{
						Attributes: map[string]interface{}{
							"OBJECTID": 1,
							"Name":     "Test Feature",
						},
						Geometry: map[string]interface{}{
							"x": -122.0,
							"y": 37.0,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(features)
			return
		}

		http.Error(w, "Not found", http.StatusNotFound)
	}))
	defer server.Close()

	// Create a test client
	client := arcgis.NewClient(30 * time.Second)

	// Create a temporary directory for test output
	tempDir, err := os.MkdirTemp("", "arcgis-utils-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases for different formats
	formats := []string{"geojson", "kml", "gpx", "csv", "json", "txt"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			// Create a test layer info
			layerInfo := arcgis.AvailableLayerInfo{
				ID:             "0",
				Name:           "Test Layer",
				ServiceURL:     server.URL,
				IsFeatureLayer: true,
			}

			// Test processing the layer
			err := processSelectedLayer(client, layerInfo, format, tempDir, true, false, "test_", false)
			if err != nil {
				t.Errorf("processSelectedLayer failed for format %s: %v", format, err)
			}

			// Check if output file was created
			expectedFile := filepath.Join(tempDir, "test_Test_Layer."+format)
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Errorf("Output file %s was not created", expectedFile)
			}
		})
	}
}

func TestConvertFeatures(t *testing.T) {
	// Create test features
	features := []arcgis.Feature{
		{
			Attributes: map[string]interface{}{
				"OBJECTID": 1,
				"Name":     "Test Feature 1",
			},
			Geometry: map[string]interface{}{
				"x": -122.0,
				"y": 37.0,
			},
		},
		{
			Attributes: map[string]interface{}{
				"OBJECTID": 2,
				"Name":     "Test Feature 2",
			},
			Geometry: map[string]interface{}{
				"paths": []interface{}{
					[]interface{}{
						[]interface{}{-122.0, 37.0},
						[]interface{}{-122.1, 37.1},
					},
				},
			},
		},
	}

	// Test conversion to convert.Feature
	convertedFeatures := convertFeatures(features)
	if len(convertedFeatures) != len(features) {
		t.Errorf("Expected %d converted features, got %d", len(features), len(convertedFeatures))
	}

	// Test conversion to GeoJSON
	geojson, err := convert.ConvertToGeoJSON(convertedFeatures)
	if err != nil {
		t.Errorf("ConvertToGeoJSON failed: %v", err)
	}
	if geojson == nil {
		t.Error("ConvertToGeoJSON returned nil")
	}
	if len(geojson.Features) != len(features) {
		t.Errorf("Expected %d GeoJSON features, got %d", len(features), len(geojson.Features))
	}

	// Test conversion to KML
	kml, err := export.ConvertGeoJSONToKML(geojson, "Test Layer")
	if err != nil {
		t.Errorf("ConvertGeoJSONToKML failed: %v", err)
	}
	if kml == "" {
		t.Error("ConvertGeoJSONToKML returned empty string")
	}

	// Test conversion to GPX
	gpx, err := export.ConvertGeoJSONToGPX(geojson, "Test Layer")
	if err != nil {
		t.Errorf("ConvertGeoJSONToGPX failed: %v", err)
	}
	if gpx == "" {
		t.Error("ConvertGeoJSONToGPX returned empty string")
	}

	// Test conversion to CSV
	csv, err := convert.ConvertFeaturesToCSV(convertedFeatures)
	if err != nil {
		t.Errorf("ConvertFeaturesToCSV failed: %v", err)
	}
	if csv == "" {
		t.Error("ConvertFeaturesToCSV returned empty string")
	}

	// Test conversion to text
	text, err := convert.ConvertFeaturesToText(convertedFeatures, "Test Layer")
	if err != nil {
		t.Errorf("ConvertFeaturesToText failed: %v", err)
	}
	if text == "" {
		t.Error("ConvertFeaturesToText returned empty string")
	}
}

func TestPrintFunctions(t *testing.T) {
	// Test each print function
	tests := []struct {
		name     string
		function func(string)
		message  string
	}{
		{"printInfo", printInfo, "Info message"},
		{"printSuccess", printSuccess, "Success message"},
		{"printWarning", printWarning, "Warning message"},
		{"printError", printError, "Error message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we disabled color output in TestMain, these should just print the message
			tt.function(tt.message)
		})
	}
}
