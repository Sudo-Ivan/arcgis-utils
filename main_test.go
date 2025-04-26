package main

import (
	"fmt"
	"strings"
	"testing"
)

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
			actual := normalizeArcGISURL(tt.input)
			if actual != tt.expected {
				t.Errorf("normalizeArcGISURL(%q): expected %q, got %q", tt.input, tt.expected, actual)
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
			if got := isValidHTTPURL(tt.input); got != tt.want {
				t.Errorf("isValidHTTPURL(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGeometryToWKT(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"Nil Geometry", nil, ""},
		{"Invalid Type", "not a map", ""},
		{"Point Geometry", map[string]interface{}{"x": -122.5, "y": 37.8}, "POINT (-122.5000000000 37.8000000000)"},
		{"Point Geometry Integer Coords", map[string]interface{}{"x": -122.0, "y": 37.0}, "POINT (-122.0000000000 37.0000000000)"},
		{"LineString Geometry", map[string]interface{}{"paths": []interface{}{ // Array of paths
			[]interface{}{ // First path: array of points
				[]interface{}{-122.0, 37.0}, // Point
				[]interface{}{-122.1, 37.1}, // Point
			},
		}}, "LINESTRING (-122.0000000000 37.0000000000, -122.1000000000 37.1000000000)"},
		{"Polygon Geometry Single Ring", map[string]interface{}{"rings": []interface{}{ // Array of rings
			[]interface{}{ // First ring: array of points
				[]interface{}{-122.0, 37.0},
				[]interface{}{-122.1, 37.0},
				[]interface{}{-122.1, 37.1},
				[]interface{}{-122.0, 37.1},
				[]interface{}{-122.0, 37.0},
			},
		}}, "POLYGON ((-122.0000000000 37.0000000000, -122.1000000000 37.0000000000, -122.1000000000 37.1000000000, -122.0000000000 37.1000000000, -122.0000000000 37.0000000000))"},
		{"Polygon Geometry Unclosed Ring", map[string]interface{}{"rings": []interface{}{ // Array of rings
			[]interface{}{ // First ring: array of points
				[]interface{}{-1.0, 1.0},
				[]interface{}{-2.0, 1.0},
				[]interface{}{-2.0, 2.0},
			},
		}}, "POLYGON ((-1.0000000000 1.0000000000, -2.0000000000 1.0000000000, -2.0000000000 2.0000000000, -1.0000000000 1.0000000000))"}, // Expect auto-close
		{"Polygon With Hole", map[string]interface{}{"rings": []interface{}{ // Array of rings
			[]interface{}{ // Outer ring
				[]interface{}{0.0, 0.0}, []interface{}{10.0, 0.0}, []interface{}{10.0, 10.0}, []interface{}{0.0, 10.0}, []interface{}{0.0, 0.0},
			},
			[]interface{}{ // Inner ring (hole)
				[]interface{}{1.0, 1.0}, []interface{}{1.0, 2.0}, []interface{}{2.0, 2.0}, []interface{}{2.0, 1.0}, []interface{}{1.0, 1.0},
			},
		}}, "POLYGON ((0.0000000000 0.0000000000, 10.0000000000 0.0000000000, 10.0000000000 10.0000000000, 0.0000000000 10.0000000000, 0.0000000000 0.0000000000), (1.0000000000 1.0000000000, 1.0000000000 2.0000000000, 2.0000000000 2.0000000000, 2.0000000000 1.0000000000, 1.0000000000 1.0000000000))"},
		{"Missing Coordinates Point", map[string]interface{}{"x": -122.5}, ""},
		{"Missing Paths", map[string]interface{}{"paths": []interface{}{}}, ""},
		{"Empty Path", map[string]interface{}{"paths": []interface{}{[]interface{}{}}}, ""},
		{"Missing Rings", map[string]interface{}{"rings": []interface{}{}}, ""},
		{"Empty Ring", map[string]interface{}{"rings": []interface{}{[]interface{}{}}}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := geometryToWKT(tt.input)
			if actual != tt.expected {
				t.Errorf("geometryToWKT(): expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"No Escaping Needed", "Hello World", "Hello World"},
		{"Ampersand", "Me & You", "Me &amp; You"},
		{"Less Than", "1 < 2", "1 &lt; 2"},
		{"Greater Than", "2 > 1", "2 &gt; 1"},
		{"Double Quote", `He said "Hi"`, `He said &quot;Hi&quot;`},
		{"Single Quote", "It's mine", "It&apos;s mine"},
		{"Forward Slash", "path/to/file", "path&#x2F;to&#x2F;file"},
		{"All Characters", `<tag attr="val'ue">&/`, `&lt;tag attr=&quot;val&apos;ue&quot;&gt;&amp;&#x2F;`},
		{"Empty String", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeXML(tt.input); got != tt.want {
				t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetFeatureName(t *testing.T) {
	tests := []struct {
		name     string
		feature  GeoJSONFeature
		expected string
	}{
		{"Name Field Lowercase", GeoJSONFeature{Properties: map[string]interface{}{"name": "Feature A", "id": 1}}, "Feature A"},
		{"Name Field Uppercase", GeoJSONFeature{Properties: map[string]interface{}{"NAME": "Feature B", "id": 2}}, "Feature B"},
		{"Name Field Mixed Case", GeoJSONFeature{Properties: map[string]interface{}{"Name": "Feature C", "id": 3}}, "Feature C"},
		{"Title Field Lowercase", GeoJSONFeature{Properties: map[string]interface{}{"title": "Feature D", "id": 4}}, "Feature D"},
		{"Title Field Uppercase", GeoJSONFeature{Properties: map[string]interface{}{"TITLE": "Feature E", "id": 5}}, "Feature E"},
		{"Title Field Mixed Case", GeoJSONFeature{Properties: map[string]interface{}{"Title": "Feature F", "id": 6}}, "Feature F"},
		{"OBJECTID Field", GeoJSONFeature{Properties: map[string]interface{}{"OBJECTID": 101, "other": "data"}}, "101"},
		{"FID Field", GeoJSONFeature{Properties: map[string]interface{}{"FID": 202, "other": "data"}}, "202"},
		{"Multiple Name Fields Priority", GeoJSONFeature{Properties: map[string]interface{}{"name": "Primary", "Name": "Secondary", "title": "Tertiary", "OBJECTID": 1}}, "Primary"},
		{"Only Title Field", GeoJSONFeature{Properties: map[string]interface{}{"title": "Only Title", "OBJECTID": 2}}, "Only Title"},
		{"Only OBJECTID Field", GeoJSONFeature{Properties: map[string]interface{}{"OBJECTID": 3}}, "3"},
		{"Only FID Field", GeoJSONFeature{Properties: map[string]interface{}{"FID": 4}}, "4"},
		{"No Name/Title/ID Fields", GeoJSONFeature{Properties: map[string]interface{}{"attribute1": "value1"}}, "Feature"},
		{"Empty Properties", GeoJSONFeature{Properties: map[string]interface{}{}}, "Feature"},
		{"Nil Properties", GeoJSONFeature{Properties: nil}, "Feature"},
		{"Name Field is Null", GeoJSONFeature{Properties: map[string]interface{}{"name": nil, "title": "Title Here"}}, "Title Here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFeatureName(tt.feature); got != tt.expected {
				t.Errorf("getFeatureName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatProperties(t *testing.T) {
	tests := []struct {
		name      string
		props     map[string]interface{}
		separator []string
		expected  string
	}{
		{"Simple Properties HTML", map[string]interface{}{"Name": "Test", "Value": 123}, []string{}, "<strong>Name</strong>: Test<br><strong>Value</strong>: 123"},                                                                                                   // Order might vary
		{"Simple Properties Custom Separator", map[string]interface{}{"City": "Paris", "Country": "France"}, []string{", "}, "<strong>City</strong>: Paris, <strong>Country</strong>: France"},                                                                       // Order might vary
		{"Properties with Escaping HTML", map[string]interface{}{"Desc": "<script>alert('XSS')</script>", "Attr": "Me & You"}, []string{}, "<strong>Attr</strong>: Me &amp; You<br><strong>Desc</strong>: &lt;script&gt;alert(&apos;XSS&apos;)&lt;&#x2F;script&gt;"}, // Order might vary
		{"Properties with Escaping Custom Separator", map[string]interface{}{"Tag": "<tag>", "Value": `"Quote"`}, []string{" | "}, "<strong>Tag</strong>: &lt;tag&gt; | <strong>Value</strong>: &quot;Quote&quot;"},                                                  // Order might vary
		{"Empty Properties", map[string]interface{}{}, []string{}, ""},
		{"Nil Properties", nil, []string{}, ""},
		{"Geometry Property Ignored", map[string]interface{}{"Name": "Test", "geometry": map[string]interface{}{}}, []string{}, "<strong>Name</strong>: Test"},
		{"Numeric Property", map[string]interface{}{"Count": 10.5}, []string{}, "<strong>Count</strong>: 10.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatProperties(tt.props, tt.separator...)
			// Note: Since map iteration order is not guaranteed, we check for parts existence instead of exact match for multiple properties.
			if len(tt.props) <= 1 {
				if got != tt.expected {
					t.Errorf("formatProperties() = %q, want %q", got, tt.expected)
				}
			} else if len(tt.props) > 1 {
				sep := "<br>"
				if len(tt.separator) > 0 {
					sep = tt.separator[0]
				}
				expectedParts := strings.Split(tt.expected, sep)
				gotParts := strings.Split(got, sep)
				if len(gotParts) != len(expectedParts) {
					t.Errorf("formatProperties() produced wrong number of parts: got %d, want %d. Got: %q, Want: %q", len(gotParts), len(expectedParts), got, tt.expected)
				}
				// Simple check: Ensure all expected parts are present in the output parts
				for _, expPart := range expectedParts {
					found := false
					for _, gotPart := range gotParts {
						if gotPart == expPart {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("formatProperties() missing expected part: %q in output %q", expPart, got)
					}
				}
			}
		})
	}
}

// --- Feature Conversion Tests ---

// Sample features for testing conversions
var testFeatures = []Feature{
	{
		Attributes: map[string]interface{}{"OBJECTID": 1, "Name": "Point Feature", "Value": 10.5},
		Geometry:   map[string]interface{}{"x": -122.0, "y": 37.0},
	},
	{
		Attributes: map[string]interface{}{"OBJECTID": 2, "Name": "Line Feature", "Status": "Active"},
		Geometry: map[string]interface{}{"paths": []interface{}{ // Array of paths
			[]interface{}{ // First path: array of points
				[]interface{}{-122.0, 37.0}, // Point
				[]interface{}{-122.1, 37.1}, // Point
			},
		}},
	},
	{
		Attributes: map[string]interface{}{"OBJECTID": 3, "Name": "Polygon Feature", "Area": 1234.5},
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
	geoJSON, err := convertToGeoJSON(testFeatures)

	if err != nil {
		t.Fatalf("convertToGeoJSON failed: %v", err)
	}

	if geoJSON == nil {
		t.Fatal("convertToGeoJSON returned nil GeoJSON object")
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
	}
	// Add more detailed checks for other features/geometries if needed
}

func TestConvertFeaturesToCSV(t *testing.T) {
	csvString, err := convertFeaturesToCSV(testFeatures)
	if err != nil {
		t.Fatalf("convertFeaturesToCSV failed: %v", err)
	}

	// Basic structural checks - more robust parsing could be added
	if !strings.HasPrefix(csvString, "Area,Name,OBJECTID,Status,Value,WKT_Geometry\n") {
		t.Errorf("CSV Header mismatch. Got: %q", strings.SplitN(csvString, "\n", 2)[0])
	}

	expectedLines := 5 // Header + 4 features
	actualLines := len(strings.Split(strings.TrimSpace(csvString), "\n"))
	if actualLines != expectedLines {
		t.Errorf("Expected %d lines in CSV output, got %d", expectedLines, actualLines)
	}

	// Check if WKT for Point is present in the second line (first data row)
	lines := strings.Split(csvString, "\n")
	if len(lines) > 1 && !strings.Contains(lines[1], "POINT (-122.0000000000 37.0000000000)") {
		t.Errorf("CSV output missing expected WKT for Point Feature. Line 1: %q", lines[1])
	}
	// Check WKT for nil geometry feature
	if len(lines) > 4 && !strings.HasSuffix(strings.TrimSpace(lines[4]), ",,") { // Expect empty WKT and maybe other empty fields
		t.Errorf("CSV output for nil geometry feature doesn't end with empty WKT. Line 4: %q", lines[4])
	}
}

func TestConvertFeaturesToText(t *testing.T) {
	layerName := "Test Layer"
	textString, err := convertFeaturesToText(testFeatures, layerName)
	if err != nil {
		t.Fatalf("convertFeaturesToText failed: %v", err)
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

// Add more tests later...