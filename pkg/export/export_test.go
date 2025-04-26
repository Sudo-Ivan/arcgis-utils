package export

import (
	"strings"
	"testing"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
)

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
		feature  convert.GeoJSONFeature
		expected string
	}{
		{"Name Field Lowercase", convert.GeoJSONFeature{Properties: map[string]interface{}{"name": "Feature A", "id": 1}}, "Feature A"},
		{"Name Field Uppercase", convert.GeoJSONFeature{Properties: map[string]interface{}{"NAME": "Feature B", "id": 2}}, "Feature B"},
		{"Name Field Mixed Case", convert.GeoJSONFeature{Properties: map[string]interface{}{"Name": "Feature C", "id": 3}}, "Feature C"},
		{"Title Field Lowercase", convert.GeoJSONFeature{Properties: map[string]interface{}{"title": "Feature D", "id": 4}}, "Feature D"},
		{"Title Field Uppercase", convert.GeoJSONFeature{Properties: map[string]interface{}{"TITLE": "Feature E", "id": 5}}, "Feature E"},
		{"Title Field Mixed Case", convert.GeoJSONFeature{Properties: map[string]interface{}{"Title": "Feature F", "id": 6}}, "Feature F"},
		{"OBJECTID Field", convert.GeoJSONFeature{Properties: map[string]interface{}{"OBJECTID": 101, "other": "data"}}, "101"},
		{"FID Field", convert.GeoJSONFeature{Properties: map[string]interface{}{"FID": 202, "other": "data"}}, "202"},
		{"Multiple Name Fields Priority", convert.GeoJSONFeature{Properties: map[string]interface{}{"name": "Primary", "Name": "Secondary", "title": "Tertiary", "OBJECTID": 1}}, "Primary"},
		{"Only Title Field", convert.GeoJSONFeature{Properties: map[string]interface{}{"title": "Only Title", "OBJECTID": 2}}, "Only Title"},
		{"Only OBJECTID Field", convert.GeoJSONFeature{Properties: map[string]interface{}{"OBJECTID": 3}}, "3"},
		{"Only FID Field", convert.GeoJSONFeature{Properties: map[string]interface{}{"FID": 4}}, "4"},
		{"No Name/Title/ID Fields", convert.GeoJSONFeature{Properties: map[string]interface{}{"attribute1": "value1"}}, "Feature"},
		{"Empty Properties", convert.GeoJSONFeature{Properties: map[string]interface{}{}}, "Feature"},
		{"Nil Properties", convert.GeoJSONFeature{Properties: nil}, "Feature"},
		{"Name Field is Null", convert.GeoJSONFeature{Properties: map[string]interface{}{"name": nil, "title": "Title Here"}}, "Title Here"},
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
