package arcgis

// FeatureServerMetadata represents the metadata for an ArcGIS Feature Server.
type FeatureServerMetadata struct {
	CurrentVersion string  `json:"currentVersion"`
	Layers         []Layer `json:"layers"`
	Tables         []Layer `json:"tables"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	ServiceItemId  string  `json:"serviceItemId"`
	Error          *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Layer represents a layer in an ArcGIS Feature Server or Map Server.
type Layer struct {
	ID           interface{}  `json:"id"`
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	GeometryType string       `json:"geometryType"`
	Description  string       `json:"description"`
	DrawingInfo  *DrawingInfo `json:"drawingInfo"`
	Error        *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// DrawingInfo represents drawing information for a layer.
type DrawingInfo struct {
	Renderer *Renderer `json:"renderer"`
}

// Renderer represents the renderer for a layer.
type Renderer struct {
	Type              string             `json:"type"`
	Field1            string             `json:"field1"`
	DefaultSymbol     *Symbol            `json:"defaultSymbol"`
	DefaultLabel      string             `json:"defaultLabel"`
	UniqueValueGroups []UniqueValueGroup `json:"uniqueValueGroups"`
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

// UniqueValueGroup represents a group of unique values for rendering.
type UniqueValueGroup struct {
	Heading string             `json:"heading"`
	Classes []UniqueValueClass `json:"classes"`
}

// UniqueValueClass represents a class of unique values for rendering.
type UniqueValueClass struct {
	Label       string     `json:"label"`
	Description string     `json:"description"`
	Values      [][]string `json:"values"`
	Symbol      *Symbol    `json:"symbol"`
}

// FeatureResponse represents the response from a feature query.
type FeatureResponse struct {
	Features              []Feature `json:"features"`
	ExceededTransferLimit bool      `json:"exceededTransferLimit"`
	Error                 *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Feature represents a geographic feature with attributes and geometry.
type Feature struct {
	Attributes map[string]interface{} `json:"attributes"`
	Geometry   interface{}            `json:"geometry"`
}

// ItemData represents metadata for an ArcGIS Online item.
type ItemData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Type  string `json:"type"`
	URL   string `json:"url"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// WebMapData represents data for an ArcGIS Online Web Map.
type WebMapData struct {
	OperationalLayers []OperationalLayer `json:"operationalLayers"`
	Error             *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// OperationalLayer represents an operational layer in a Web Map.
type OperationalLayer struct {
	ID                string             `json:"id"`
	Title             string             `json:"title"`
	URL               string             `json:"url"`
	ItemID            string             `json:"itemId"`
	LayerType         string             `json:"layerType"`
	Layers            []OperationalLayer `json:"layers"`
	FeatureCollection *struct {
		Layers []FeatureCollectionLayer `json:"layers"`
	} `json:"featureCollection"`
}

// FeatureCollectionLayer represents a layer within a FeatureCollection.
type FeatureCollectionLayer struct {
	ID              int                    `json:"id"`
	LayerDefinition map[string]interface{} `json:"layerDefinition"`
	FeatureSet      *struct {
		Features []Feature `json:"features"`
	} `json:"featureSet"`
}

// MapServiceMetadata represents the metadata for an ArcGIS Map Service.
type MapServiceMetadata struct {
	Name        string            `json:"name"`
	Layers      []MapServiceLayer `json:"layers"`
	Description string            `json:"description"`
	Error       *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// MapServiceLayer represents a layer in an ArcGIS Map Service.
type MapServiceLayer struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	GeometryType  string `json:"geometryType"`
	ParentLayerId int    `json:"parentLayerId"`
	SubLayerIds   []int  `json:"subLayerIds"`
}

// AvailableLayerInfo stores information about a layer available for processing.
type AvailableLayerInfo struct {
	ID                    string
	Name                  string
	Type                  string
	GeometryType          string
	ServiceURL            string
	ParentPath            []string
	IsFeatureLayer        bool
	FeatureCollectionData *OperationalLayer
}
