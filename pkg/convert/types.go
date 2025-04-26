package convert

// GeoJSON represents a GeoJSON FeatureCollection.
type GeoJSON struct {
	Type     string           `json:"type"`
	CRS      CRS              `json:"crs"`
	Features []GeoJSONFeature `json:"features"`
}

// GeoJSONFeature represents a GeoJSON Feature.
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   interface{}            `json:"geometry"`
	Symbol     *Symbol                `json:"symbol,omitempty"`
}

// CRS represents a Coordinate Reference System.
type CRS struct {
	Type       string   `json:"type"`
	Properties CRSProps `json:"properties"`
}

// CRSProps represents Coordinate Reference System properties.
type CRSProps struct {
	Name string `json:"name"`
}

// Symbol represents a symbol used for rendering features.
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
type Feature struct {
	Attributes map[string]interface{} `json:"attributes"`
	Geometry   interface{}            `json:"geometry"`
}
