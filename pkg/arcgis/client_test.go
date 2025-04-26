package arcgis

import (
	"testing"
	"time"
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
			actual := NormalizeArcGISURL(tt.input)
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
			if got := IsValidHTTPURL(tt.input); got != tt.want {
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
			if got := IsArcGISOnlineItemURL(tt.input); got != tt.want {
				t.Errorf("IsArcGISOnlineItemURL(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	timeout := 30 * time.Second
	client := NewClient(timeout)

	if client.Timeout != timeout {
		t.Errorf("NewClient timeout = %v; want %v", client.Timeout, timeout)
	}

	if client.HTTPClient == nil {
		t.Error("NewClient HTTPClient is nil")
	}

	if client.HTTPClient.Timeout != timeout {
		t.Errorf("NewClient HTTPClient timeout = %v; want %v", client.HTTPClient.Timeout, timeout)
	}
}
