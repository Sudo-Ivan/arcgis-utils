// Copyright (c) 2024 Sudo-Ivan
// Licensed under the MIT License

// Package convert provides types and functions for converting between different geospatial data formats.
package convert

// GeoJSON represents a GeoJSON FeatureCollection.
// It contains a collection of features with their geometries and properties,
// along with coordinate reference system information.
type GeoJSON struct {
	Type     string           `json:"type"`
	CRS      CRS              `json:"crs"`
	Features []GeoJSONFeature `json:"features"`
}

// GeoJSONFeature represents a GeoJSON Feature.
// It contains a single feature's geometry, properties, and optional symbol information.
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   interface{}            `json:"geometry"`
	Symbol     *Symbol                `json:"symbol,omitempty"`
}

// CRS represents a Coordinate Reference System.
// It defines the spatial reference system used for the geometry coordinates.
type CRS struct {
	Type       string   `json:"type"`
	Properties CRSProps `json:"properties"`
}

// CRSProps represents Coordinate Reference System properties.
// It contains the name of the coordinate reference system.
type CRSProps struct {
	Name string `json:"name"`
}

// Symbol represents a symbol used for rendering features.
// It contains information about the symbol's appearance, including:
//   - Type: The type of symbol (e.g., esriPMS, esriSMS)
//   - URL: The URL of the symbol image
//   - ImageData: Base64-encoded image data
//   - ContentType: The MIME type of the image
//   - Dimensions and positioning information
type Symbol struct {
	Type        string  `json:"type"`
	URL         string  `json:"url"`
	ImageData   string  `json:"imageData"`
	ContentType string  `json:"contentType"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	XOffset     int     `json:"xoffset"`
	YOffset     int     `json:"yoffset"`
	Angle       float64 `json:"angle"`
}

// Feature represents a geographic feature with attributes and geometry.
// It is used as an intermediate representation for converting between different formats.
type Feature struct {
	Attributes map[string]interface{} `json:"attributes"`
	Geometry   interface{}            `json:"geometry"`
}
