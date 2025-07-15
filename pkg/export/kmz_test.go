// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

package export

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
)

func TestConvertGeoJSONToKMZ(t *testing.T) {
	// Create test data with base64 image
	testImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

	geoJSON := &convert.GeoJSON{
		Type: "FeatureCollection",
		CRS: convert.CRS{
			Type: "name",
			Properties: convert.CRSProps{
				Name: "EPSG:4326",
			},
		},
		Features: []convert.GeoJSONFeature{
			{
				Type: "Feature",
				Properties: map[string]interface{}{
					"name": "Test Point",
					"id":   1,
				},
				Geometry: map[string]interface{}{
					"type":        "Point",
					"coordinates": []float64{-122.4194, 37.7749},
				},
				Symbol: &convert.Symbol{
					Type:        "esriPMS",
					ImageData:   testImageData,
					ContentType: "image/png",
					Width:       16,
					Height:      16,
					XOffset:     0,
					YOffset:     0,
					Angle:       0,
				},
			},
			{
				Type: "Feature",
				Properties: map[string]interface{}{
					"name": "Test Point 2",
					"id":   2,
				},
				Geometry: map[string]interface{}{
					"type":        "Point",
					"coordinates": []float64{-122.4094, 37.7849},
				},
				// No symbol for this feature
			},
		},
	}

	// Convert to KMZ
	kmzData, err := ConvertGeoJSONToKMZ(geoJSON, "Test Layer")
	if err != nil {
		t.Fatalf("Failed to convert GeoJSON to KMZ: %v", err)
	}

	// Verify KMZ structure
	reader, err := zip.NewReader(bytes.NewReader(kmzData), int64(len(kmzData)))
	if err != nil {
		t.Fatalf("Failed to read KMZ archive: %v", err)
	}

	// Check that we have the expected files
	expectedFiles := map[string]bool{
		"doc.kml":             false,
		"images/symbol_1.png": false,
	}

	for _, file := range reader.File {
		if _, expected := expectedFiles[file.Name]; expected {
			expectedFiles[file.Name] = true
		}
	}

	// Verify all expected files are present
	for filename, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not found in KMZ archive", filename)
		}
	}

	// Read and verify KML content
	var kmlContent string
	for _, file := range reader.File {
		if file.Name == "doc.kml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open KML file: %v", err)
			}
			defer rc.Close()

			buf := new(bytes.Buffer)
			buf.ReadFrom(rc)
			kmlContent = buf.String()
			break
		}
	}

	// Verify KML content contains expected elements
	if !strings.Contains(kmlContent, "<name>Test Layer</name>") {
		t.Error("KML content should contain layer name")
	}
	if !strings.Contains(kmlContent, "<name>Test Point</name>") {
		t.Error("KML content should contain feature name")
	}
	if !strings.Contains(kmlContent, "images/symbol_1.png") {
		t.Error("KML content should reference the embedded image")
	}
	if !strings.Contains(kmlContent, "<Point>") {
		t.Error("KML content should contain Point geometry")
	}

	// Verify image file content
	var imageData []byte
	for _, file := range reader.File {
		if file.Name == "images/symbol_1.png" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open image file: %v", err)
			}
			defer rc.Close()

			buf := new(bytes.Buffer)
			buf.ReadFrom(rc)
			imageData = buf.Bytes()
			break
		}
	}

	// Verify the image data matches what we put in
	expectedImageData, err := base64.StdEncoding.DecodeString(testImageData)
	if err != nil {
		t.Fatalf("Failed to decode test image data: %v", err)
	}

	if !bytes.Equal(imageData, expectedImageData) {
		t.Error("Image data in KMZ does not match original base64 data")
	}
}

func TestConvertGeoJSONToKMZNoSymbols(t *testing.T) {
	// Test with no symbols
	geoJSON := &convert.GeoJSON{
		Type: "FeatureCollection",
		CRS: convert.CRS{
			Type: "name",
			Properties: convert.CRSProps{
				Name: "EPSG:4326",
			},
		},
		Features: []convert.GeoJSONFeature{
			{
				Type: "Feature",
				Properties: map[string]interface{}{
					"name": "Test Point",
					"id":   1,
				},
				Geometry: map[string]interface{}{
					"type":        "Point",
					"coordinates": []float64{-122.4194, 37.7749},
				},
			},
		},
	}

	// Convert to KMZ
	kmzData, err := ConvertGeoJSONToKMZ(geoJSON, "Test Layer No Symbols")
	if err != nil {
		t.Fatalf("Failed to convert GeoJSON to KMZ: %v", err)
	}

	// Verify KMZ structure
	reader, err := zip.NewReader(bytes.NewReader(kmzData), int64(len(kmzData)))
	if err != nil {
		t.Fatalf("Failed to read KMZ archive: %v", err)
	}

	// Should only have KML file, no images
	if len(reader.File) != 1 {
		t.Errorf("Expected 1 file in KMZ archive, got %d", len(reader.File))
	}

	if reader.File[0].Name != "doc.kml" {
		t.Errorf("Expected doc.kml, got %s", reader.File[0].Name)
	}
}

func TestGetImageExtension(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/svg+xml", ".svg"},
		{"", ".png"},        // default
		{"unknown", ".png"}, // default
	}

	for _, test := range tests {
		result := getImageExtension(test.contentType)
		if result != test.expected {
			t.Errorf("getImageExtension(%s) = %s, expected %s", test.contentType, result, test.expected)
		}
	}
}
