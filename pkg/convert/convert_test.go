package convert

import (
	"fmt"
	"strings"
	"testing"
)

// Sample features for testing conversions
var testFeatures = []Feature{
	{
		Attributes: map[string]interface{}{
			"OBJECTID": 1,
			"Name":     "Point Feature",
			"Value":    10.5,
			"symbol": &Symbol{
				Type:        "esriPMS",
				URL:         "test.png",
				ImageData:   "base64data",
				ContentType: "image/png",
				Width:       20,
				Height:      20,
				XOffset:     0,
				YOffset:     0,
				Angle:       0,
			},
		},
		Geometry: map[string]interface{}{"x": -122.0, "y": 37.0},
	},
	{
		Attributes: map[string]interface{}{
			"OBJECTID": 2,
			"Name":     "Line Feature",
			"Status":   "Active",
			"symbol": &Symbol{
				Type:        "esriSLS",
				URL:         "line.png",
				ImageData:   "base64data2",
				ContentType: "image/png",
				Width:       2,
				Height:      0,
				XOffset:     0,
				YOffset:     0,
				Angle:       45,
			},
		},
		Geometry: map[string]interface{}{"paths": []interface{}{ // Array of paths
			[]interface{}{ // First path: array of points
				[]interface{}{-122.0, 37.0}, // Point
				[]interface{}{-122.1, 37.1}, // Point
			},
		}},
	},
	{
		Attributes: map[string]interface{}{
			"OBJECTID": 3,
			"Name":     "Polygon Feature",
			"Area":     1234.5,
			"symbol": &Symbol{
				Type:        "esriSFS",
				URL:         "poly.png",
				ImageData:   "base64data3",
				ContentType: "image/png",
				Width:       0,
				Height:      0,
				XOffset:     0,
				YOffset:     0,
				Angle:       0,
			},
		},
		Geometry: map[string]interface{}{"rings": []interface{}{ // Array of rings
			[]interface{}{ // First ring: array of points
				[]interface{}{-1.0, 1.0},
				[]interface{}{-2.0, 1.0},
				[]interface{}{-2.0, 2.0},
				[]interface{}{-1.0, 1.0}, // Closed explicitly for simplicity here
			},
		}},
	},
	{
		Attributes: map[string]interface{}{"OBJECTID": 4, "Name": "Attribute Only"},
		Geometry:   nil, // No geometry
	},
}

func TestConvertToGeoJSON(t *testing.T) {
	// Test with symbols included
	geoJSON, err := ToGeoJSON(testFeatures)
	if err != nil {
		t.Fatalf("ToGeoJSON failed: %v", err)
	}

	if geoJSON == nil {
		t.Fatal("ToGeoJSON returned nil GeoJSON object")
	}

	if geoJSON.Type != "FeatureCollection" {
		t.Errorf("Expected GeoJSON Type 'FeatureCollection', got %q", geoJSON.Type)
	}

	expectedFeatureCount := 3 // Feature 4 has no geometry, should be skipped
	if len(geoJSON.Features) != expectedFeatureCount {
		t.Errorf("Expected %d GeoJSON features, got %d", expectedFeatureCount, len(geoJSON.Features))
	}

	// Basic checks on the first feature (Point)
	if len(geoJSON.Features) > 0 {
		f1 := geoJSON.Features[0]
		if f1.Type != "Feature" {
			t.Errorf("Feature 1: Expected Type 'Feature', got %q", f1.Type)
		}
		if f1.Properties["Name"] != "Point Feature" {
			t.Errorf("Feature 1: Expected Name property 'Point Feature', got %v", f1.Properties["Name"])
		}
		geom, ok := f1.Geometry.(map[string]interface{})
		if !ok || geom["type"] != "Point" {
			t.Errorf("Feature 1: Expected Point geometry, got %v", f1.Geometry)
		}

		// Check symbol information
		symbol, ok := f1.Properties["symbol"].(*Symbol)
		if !ok {
			t.Errorf("Feature 1: Expected Symbol property, got %v", f1.Properties["symbol"])
		} else {
			if symbol.Type != "esriPMS" {
				t.Errorf("Feature 1: Expected symbol type 'esriPMS', got %q", symbol.Type)
			}
			if symbol.URL != "test.png" {
				t.Errorf("Feature 1: Expected symbol URL 'test.png', got %q", symbol.URL)
			}
			if symbol.ImageData != "base64data" {
				t.Errorf("Feature 1: Expected symbol imageData 'base64data', got %q", symbol.ImageData)
			}
			if symbol.ContentType != "image/png" {
				t.Errorf("Feature 1: Expected symbol contentType 'image/png', got %q", symbol.ContentType)
			}
			if symbol.Width != 20 {
				t.Errorf("Feature 1: Expected symbol width 20, got %d", symbol.Width)
			}
			if symbol.Height != 20 {
				t.Errorf("Feature 1: Expected symbol height 20, got %d", symbol.Height)
			}
		}
	}

	// Test with symbols excluded
	featuresWithoutSymbols := make([]Feature, len(testFeatures))
	for i, f := range testFeatures {
		featuresWithoutSymbols[i] = Feature{
			Attributes: make(map[string]interface{}),
			Geometry:   f.Geometry,
		}
		// Copy all attributes except symbol
		for k, v := range f.Attributes {
			if k != "symbol" {
				featuresWithoutSymbols[i].Attributes[k] = v
			}
		}
	}

	geoJSONNoSymbols, err := ToGeoJSON(featuresWithoutSymbols)
	if err != nil {
		t.Fatalf("ToGeoJSON failed with excluded symbols: %v", err)
	}

	if len(geoJSONNoSymbols.Features) > 0 {
		f1 := geoJSONNoSymbols.Features[0]
		if _, ok := f1.Properties["symbol"]; ok {
			t.Errorf("Feature 1: Expected no symbol property when symbols are excluded")
		}
	}
}

func TestConvertFeaturesToCSV(t *testing.T) {
	// Test with symbols included
	csvString, err := FeaturesToCSV(testFeatures)
	if err != nil {
		t.Fatalf("FeaturesToCSV failed: %v", err)
	}

	// Basic structural checks - more robust parsing could be added
	expectedHeader := "Area,Name,OBJECTID,Status,Value,symbol,WKT_Geometry"
	if !strings.HasPrefix(csvString, expectedHeader+"\n") {
		t.Errorf("CSV Header mismatch. Got: %q", strings.SplitN(csvString, "\n", 2)[0])
	}

	// Test with symbols excluded
	featuresWithoutSymbols := make([]Feature, len(testFeatures))
	for i, f := range testFeatures {
		featuresWithoutSymbols[i] = Feature{
			Attributes: make(map[string]interface{}),
			Geometry:   f.Geometry,
		}
		// Copy all attributes except symbol
		for k, v := range f.Attributes {
			if k != "symbol" {
				featuresWithoutSymbols[i].Attributes[k] = v
			}
		}
	}

	csvStringNoSymbols, err := FeaturesToCSV(featuresWithoutSymbols)
	if err != nil {
		t.Fatalf("FeaturesToCSV failed with excluded symbols: %v", err)
	}

	expectedHeaderNoSymbols := "Area,Name,OBJECTID,Status,Value,WKT_Geometry"
	if !strings.HasPrefix(csvStringNoSymbols, expectedHeaderNoSymbols+"\n") {
		t.Errorf("CSV Header mismatch when symbols excluded. Got: %q", strings.SplitN(csvStringNoSymbols, "\n", 2)[0])
	}
}

func TestConvertFeaturesToText(t *testing.T) {
	layerName := "Test Layer"
	textString, err := FeaturesToText(testFeatures, layerName)
	if err != nil {
		t.Fatalf("FeaturesToText failed: %v", err)
	}

	// Basic structural checks
	if !strings.Contains(textString, fmt.Sprintf("Layer: %s\n", layerName)) {
		t.Errorf("Text output missing Layer name header.")
	}
	if !strings.Contains(textString, fmt.Sprintf("Total Features: %d\n", len(testFeatures))) {
		t.Errorf("Text output missing Total Features header.")
	}
	if !strings.Contains(textString, "--- Feature 1 ---") {
		t.Errorf("Text output missing marker for Feature 1.")
	}
	if !strings.Contains(textString, "--- Feature 4 ---") {
		t.Errorf("Text output missing marker for Feature 4.")
	}
	if !strings.Contains(textString, "Name: Point Feature") {
		t.Errorf("Text output missing attribute 'Name: Point Feature'.")
	}
	if !strings.Contains(textString, "Geometry (WKT):\n  POINT (-122.0000000000 37.0000000000)") {
		t.Errorf("Text output missing WKT for Point Feature.")
	}
	if !strings.Contains(textString, "Geometry (WKT):\n  <No Geometry>") {
		t.Errorf("Text output missing '<No Geometry>' marker for nil geometry feature.")
	}
}
