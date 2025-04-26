package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ANSI color codes for console output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// useColor controls whether colored output is enabled.
var useColor = true

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

// layersToProcess stores the layers selected for processing.
var layersToProcess = make(map[string]AvailableLayerInfo)

func main() {
	urlPtr := flag.String("url", "", "ArcGIS Feature Layer, Feature Server, Map Server, or ArcGIS Online Item URL")
	formatPtr := flag.String("format", "geojson", "Output format (geojson, kml, gpx, csv, json, text)")
	outputPtr := flag.String("output", "", "Output directory (default: current directory)")
	selectAllPtr := flag.Bool("select-all", false, "Select all found Feature Layers automatically (no prompt)")
	noColorPtr := flag.Bool("no-color", false, "Disable colored output")
	overwritePtr := flag.Bool("overwrite", false, "Overwrite existing output files")
	skipExistingPtr := flag.Bool("skip-existing", false, "Skip processing if output file already exists")
	prefixPtr := flag.String("prefix", "", "Prefix for output filenames")
	timeoutPtr := flag.Int("timeout", 30, "HTTP request timeout in seconds")
	excludeSymbolsPtr := flag.Bool("exclude-symbols", false, "Exclude symbol information from output")

	flag.Parse()

	useColor = !*noColorPtr

	if *urlPtr == "" {
		printError("URL is required")
		flag.Usage()
		os.Exit(1)
	}

	inputURL := *urlPtr
	if !isValidHTTPURL(inputURL) {
		normalizedURL := normalizeArcGISURL(inputURL)
		if !isValidHTTPURL(normalizedURL) {
			fmt.Println("Error: Invalid URL provided.")
			os.Exit(1)
		}
		inputURL = normalizedURL
	} else {
		inputURL = normalizeArcGISURL(inputURL)
	}

	outputDir := *outputPtr
	if outputDir == "" {
		outputDir, _ = os.Getwd()
	}

	httpClient = &http.Client{
		Timeout: time.Duration(*timeoutPtr) * time.Second,
	}

	var err error
	if isArcGISOnlineItemURL(inputURL) {
		fmt.Println("Detected ArcGIS Online Item URL...")
		err = handleArcGISOnlineItem(inputURL, *selectAllPtr)
	} else if strings.Contains(strings.ToLower(inputURL), "/mapserver") {
		fmt.Println("Detected Map Server URL...")
		err = handleMapServerURL(inputURL, *selectAllPtr)
	} else if strings.Contains(strings.ToLower(inputURL), "/featureserver") {
		fmt.Println("Detected Feature Server URL...")
		err = handleFeatureServerURL(inputURL, *selectAllPtr)
	} else {
		fmt.Println("Assuming single Feature Layer URL...")
		parts := strings.Split(inputURL, "/")
		if len(parts) < 2 {
			err = fmt.Errorf("invalid single layer URL format")
		} else {
			layerID := parts[len(parts)-1]
			baseURL := strings.Join(parts[:len(parts)-1], "/")
			layersToProcess[inputURL] = AvailableLayerInfo{
				ID:             layerID,
				Name:           fmt.Sprintf("Layer_%s", layerID),
				ServiceURL:     baseURL,
				IsFeatureLayer: true,
			}
		}
	}

	if err != nil {
		printError(fmt.Sprintf("Error identifying or fetching initial data: %v", err))
		os.Exit(1)
	}

	if len(layersToProcess) == 0 {
		printInfo("No Feature Layers were selected or found to process.")
		os.Exit(0)
	}

	printInfo(fmt.Sprintf("\nProcessing %d selected layer(s) concurrently...", len(layersToProcess)))
	var successCount, skippedCount, errorCount atomic.Int32
	processedKeys := make(map[string]bool)
	var wg sync.WaitGroup
	mu := sync.Mutex{} // Mutex to protect processedKeys map

	for key, layerInfo := range layersToProcess {
		mu.Lock()
		if processedKeys[key] {
			mu.Unlock()
			continue
		}
		processedKeys[key] = true
		mu.Unlock()

		wg.Add(1)
		// Capture range variables for goroutine
		layerInfoCopy := layerInfo
		formatCopy := *formatPtr
		outputDirCopy := outputDir
		overwriteCopy := *overwritePtr
		skipExistingCopy := *skipExistingPtr
		prefixCopy := *prefixPtr
		excludeSymbolsCopy := *excludeSymbolsPtr

		go func() {
			defer wg.Done()
			printInfo(fmt.Sprintf("Processing Layer: %s (ID: %s)", layerInfoCopy.Name, layerInfoCopy.ID))
			err := processSelectedLayer(layerInfoCopy, formatCopy, outputDirCopy, overwriteCopy, skipExistingCopy, prefixCopy, excludeSymbolsCopy)
			if err != nil {
				if err.Error() == "skipped existing file" {
					printWarning(fmt.Sprintf("  Skipped layer %s (output file exists).", layerInfoCopy.Name))
					skippedCount.Add(1)
				} else if err.Error() == "no features found" {
					printWarning(fmt.Sprintf("  Skipped layer %s (no features found).", layerInfoCopy.Name))
					skippedCount.Add(1)
				} else {
					printError(fmt.Sprintf("  Error processing layer %s: %v", layerInfoCopy.Name, err))
					errorCount.Add(1)
				}
			} else {
				printSuccess(fmt.Sprintf("  Successfully processed layer %s.", layerInfoCopy.Name))
				successCount.Add(1)
			}
		}()
	}

	wg.Wait() // Wait for all processing goroutines to finish

	finalSuccessCount := successCount.Load()
	finalSkippedCount := skippedCount.Load()
	finalErrorCount := errorCount.Load()

	summary := fmt.Sprintf("\nProcessing Complete. %d layers succeeded, %d skipped, %d failed.", finalSuccessCount, finalSkippedCount, finalErrorCount)
	if finalErrorCount > 0 {
		printError(summary)
		os.Exit(1)
	} else if finalSkippedCount > 0 {
		printWarning(summary)
	} else {
		printSuccess(summary)
	}
}

// printColor prints a message to the console with the specified color.
func printColor(colorCode string, message string) {
	if useColor {
		fmt.Printf("%s%s%s\n", colorCode, message, colorReset)
	} else {
		fmt.Println(message)
	}
}

// printInfo prints an informational message to the console.
func printInfo(message string) {
	printColor(colorCyan, message)
}

// printSuccess prints a success message to the console.
func printSuccess(message string) {
	printColor(colorGreen, message)
}

// printWarning prints a warning message to the console.
func printWarning(message string) {
	printColor(colorYellow, message)
}

// printError prints an error message to the console.
func printError(message string) {
	printColor(colorRed, message)
}

// httpClient is a reusable HTTP client with a timeout.
var httpClient *http.Client

// isArcGISOnlineItemURL checks if a URL points to an ArcGIS Online item page.
func isArcGISOnlineItemURL(rawURL string) bool {
	return strings.Contains(strings.ToLower(rawURL), "arcgis.com/home/item.html")
}

// fetchAndDecode fetches data from a URL and decodes it into the target interface.
func fetchAndDecode(urlStr string, target interface{}) error {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %v", urlStr, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return fmt.Errorf("request timed out fetching data from %s: %v", urlStr, err)
		}
		return fmt.Errorf("failed to fetch data from %s: %v", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK HTTP status %d from %s", resp.StatusCode, urlStr)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %v", urlStr, err)
	}

	return nil
}

// handleArcGISOnlineItem handles processing for an ArcGIS Online item URL.
func handleArcGISOnlineItem(itemPageURL string, selectAll bool) error {
	re := regexp.MustCompile(`id=([a-f0-9]+)`)
	matches := re.FindStringSubmatch(itemPageURL)
	if len(matches) < 2 {
		return fmt.Errorf("could not extract item ID from URL: %s", itemPageURL)
	}
	itemID := matches[1]
	printInfo(fmt.Sprintf("  Item ID: %s", itemID))

	itemAPIURL := fmt.Sprintf("https://www.arcgis.com/sharing/rest/content/items/%s?f=json", itemID)

	var itemData ItemData
	err := fetchAndDecode(itemAPIURL, &itemData)
	if err != nil {
		return fmt.Errorf("failed to fetch item metadata: %v", err)
	}
	if itemData.Error != nil {
		return fmt.Errorf("item API error: %s", itemData.Error.Message)
	}

	printInfo(fmt.Sprintf("  Item Type: %s", itemData.Type))

	switch itemData.Type {
	case "Feature Service":
		if itemData.URL == "" {
			return fmt.Errorf("feature Service item has no URL")
		}
		return handleFeatureServerURL(itemData.URL, selectAll)
	case "Map Service":
		if itemData.URL == "" {
			return fmt.Errorf("map Service item has no URL")
		}
		return handleMapServerURL(itemData.URL, selectAll)
	case "Web Map":
		return handleWebMap(itemID, selectAll)
	default:
		return fmt.Errorf("unsupported item type: %s. Currently supports Feature Service, Map Service, Web Map", itemData.Type)
	}
}

// handleWebMap handles processing for an ArcGIS Online Web Map item.
func handleWebMap(itemID string, selectAll bool) error {
	webMapDataURL := fmt.Sprintf("https://www.arcgis.com/sharing/rest/content/items/%s/data?f=json", itemID)
	printInfo(fmt.Sprintf("  Fetching Web Map data: %s", webMapDataURL))

	var webMapData WebMapData
	err := fetchAndDecode(webMapDataURL, &webMapData)
	if err != nil {
		return fmt.Errorf("failed to fetch web map data: %v", err)
	}
	if webMapData.Error != nil {
		return fmt.Errorf("web map data API error: %s", webMapData.Error.Message)
	}

	availableLayers := []AvailableLayerInfo{}
	for _, opLayer := range webMapData.OperationalLayers {
		processOperationalLayer(opLayer, []string{}, &availableLayers)
	}

	if len(availableLayers) == 0 {
		fmt.Println("  No processable Feature Layers found in this Web Map.")
		return nil
	}

	fmt.Printf("  Found %d potential Feature Layers in Web Map.\n", len(availableLayers))
	return selectAndAddLayers(availableLayers, selectAll)
}

// processOperationalLayer recursively processes operational layers in a Web Map.
func processOperationalLayer(opLayer OperationalLayer, parentPath []string, availableLayers *[]AvailableLayerInfo) {
	currentPath := append(parentPath, opLayer.Title)

	if opLayer.LayerType == "GroupLayer" || len(opLayer.Layers) > 0 {
		fmt.Printf("    Processing Group: %s\n", strings.Join(currentPath, " > "))
		for _, subLayer := range opLayer.Layers {
			processOperationalLayer(subLayer, currentPath, availableLayers)
		}
	} else if opLayer.FeatureCollection != nil && len(opLayer.FeatureCollection.Layers) > 0 {
		fmt.Printf("    Skipping Inline Feature Collection: %s (Direct processing not yet implemented)\n", strings.Join(currentPath, " > "))
	} else if opLayer.URL != "" && (strings.Contains(strings.ToLower(opLayer.URL), "/featureserver") || strings.Contains(strings.ToLower(opLayer.URL), "/mapserver")) {
		serviceURL := normalizeArcGISURL(opLayer.URL)
		layerIDStr := ""
		parts := strings.Split(serviceURL, "/")
		lastPart := parts[len(parts)-1]
		if _, err := strconv.Atoi(lastPart); err == nil {
			layerIDStr = lastPart
			serviceURL = strings.Join(parts[:len(parts)-1], "/")
		} else {
			fmt.Printf("    Service URL found: %s for layer %s. Fetching service layers...\n", serviceURL, opLayer.Title)
			var subLayers []AvailableLayerInfo
			var err error
			if strings.Contains(strings.ToLower(serviceURL), "/featureserver") {
				subLayers, err = fetchServiceLayers(serviceURL, "FeatureServer")
			} else {
				subLayers, err = fetchServiceLayers(serviceURL, "MapServer")
			}
			if err != nil {
				fmt.Printf("      Warning: Failed to fetch layers for service %s: %v\n", serviceURL, err)
			} else {
				for _, sl := range subLayers {
					sl.ParentPath = currentPath
					*availableLayers = append(*availableLayers, sl)
				}
			}
			return
		}

		if layerIDStr != "" {
			fmt.Printf("    Adding Layer Reference: %s (ID: %s) from Service: %s\n", strings.Join(currentPath, " > "), layerIDStr, serviceURL)
			layerInfo := AvailableLayerInfo{
				ID:             layerIDStr,
				Name:           opLayer.Title,
				ServiceURL:     serviceURL,
				ParentPath:     currentPath,
				IsFeatureLayer: true,
				Type:           opLayer.LayerType,
			}
			*availableLayers = append(*availableLayers, layerInfo)
		}
	} else if opLayer.ItemID != "" {
		fmt.Printf("    Processing Item Reference: %s (ID: %s)\n", strings.Join(currentPath, " > "), opLayer.ItemID)
		itemAPIURL := fmt.Sprintf("https://www.arcgis.com/sharing/rest/content/items/%s?f=json", opLayer.ItemID)
		var itemData ItemData
		err := fetchAndDecode(itemAPIURL, &itemData)
		if err != nil || itemData.Error != nil || itemData.URL == "" {
			fmt.Printf("      Warning: Failed to fetch or use referenced item %s: %v\n", opLayer.ItemID, err)
			if itemData.Error != nil {
				fmt.Printf("      Item API Error: %s\n", itemData.Error.Message)
			}
		} else {
			fmt.Printf("      Referenced Item Type: %s, URL: %s\n", itemData.Type, itemData.URL)
			var subLayers []AvailableLayerInfo
			var fetchErr error
			if strings.Contains(strings.ToLower(itemData.URL), "/featureserver") {
				subLayers, fetchErr = fetchServiceLayers(itemData.URL, "FeatureServer")
			} else if strings.Contains(strings.ToLower(itemData.URL), "/mapserver") {
				subLayers, fetchErr = fetchServiceLayers(itemData.URL, "MapServer")
			} else {
				fmt.Printf("      Warning: Referenced item %s has unsupported service URL type: %s\n", opLayer.ItemID, itemData.URL)
			}

			if fetchErr != nil {
				fmt.Printf("      Warning: Failed to fetch layers for referenced item service %s: %v\n", itemData.URL, fetchErr)
			} else {
				for _, sl := range subLayers {
					sl.ParentPath = append(currentPath, sl.ParentPath...)
					*availableLayers = append(*availableLayers, sl)
				}
			}
		}
	} else {
		fmt.Printf("    Skipping layer '%s': No URL, ItemID, or FeatureCollection found.\n", opLayer.Title)
	}
}

// handleMapServerURL handles processing for an ArcGIS Map Server URL.
func handleMapServerURL(mapServerURL string, selectAll bool) error {
	layers, err := fetchServiceLayers(mapServerURL, "MapServer")
	if err != nil {
		return err
	}
	if len(layers) == 0 {
		fmt.Println("  No processable Feature Layers found in this Map Service.")
		return nil
	}
	fmt.Printf("  Found %d potential Feature Layers in Map Service.\n", len(layers))
	return selectAndAddLayers(layers, selectAll)
}

// handleFeatureServerURL handles processing for an ArcGIS Feature Server URL.
func handleFeatureServerURL(featureServerURL string, selectAll bool) error {
	layerID := ""
	parts := strings.Split(featureServerURL, "/")
	lastPart := parts[len(parts)-1]
	if _, err := strconv.Atoi(lastPart); err == nil {
		layerID = lastPart
		featureServerURL = strings.Join(parts[:len(parts)-1], "/")
	}

	if layerID != "" {
		fmt.Println("  Processing as single Feature Layer URL...")
		layersToProcess[featureServerURL+"/"+layerID] = AvailableLayerInfo{
			ID:             layerID,
			Name:           fmt.Sprintf("Layer_%s", layerID),
			ServiceURL:     featureServerURL,
			IsFeatureLayer: true,
		}
		return nil
	} else {
		layers, err := fetchServiceLayers(featureServerURL, "FeatureServer")
		if err != nil {
			return err
		}
		if len(layers) == 0 {
			fmt.Println("  No processable Feature Layers found in this Feature Service.")
			return nil
		}
		fmt.Printf("  Found %d potential Feature Layers in Feature Service.\n", len(layers))
		return selectAndAddLayers(layers, selectAll)
	}
}

// fetchServiceLayers fetches the layers from an ArcGIS Feature Server or Map Server.
func fetchServiceLayers(serviceURL string, serviceType string) ([]AvailableLayerInfo, error) {
	fetchURL := fmt.Sprintf("%s?f=json", serviceURL)
	printInfo(fmt.Sprintf("    Fetching service metadata: %s", fetchURL))

	availableLayers := []AvailableLayerInfo{}

	if serviceType == "FeatureServer" {
		var metadata FeatureServerMetadata
		err := fetchAndDecode(fetchURL, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Feature Server metadata from %s: %v", fetchURL, err)
		}
		if metadata.Error != nil {
			return nil, fmt.Errorf("feature Server API error: %s", metadata.Error.Message)
		}

		if len(metadata.Layers) == 0 && len(metadata.Tables) == 0 {
			printWarning(fmt.Sprintf("      No layers or tables found in Feature Server metadata at %s", fetchURL))
		}

		for _, layer := range metadata.Layers {
			layerIDStr, ok := layer.ID.(json.Number)
			if !ok {
				fmt.Printf("      Warning: Could not parse layer ID for %s\n", layer.Name)
				continue
			}
			availableLayers = append(availableLayers, AvailableLayerInfo{
				ID:             layerIDStr.String(),
				Name:           layer.Name,
				Type:           layer.Type,
				GeometryType:   layer.GeometryType,
				ServiceURL:     serviceURL,
				IsFeatureLayer: true,
			})
		}
	} else if serviceType == "MapServer" {
		var metadata MapServiceMetadata
		err := fetchAndDecode(fetchURL, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Map Server metadata from %s: %v", fetchURL, err)
		}
		if metadata.Error != nil {
			return nil, fmt.Errorf("map Server API error: %s", metadata.Error.Message)
		}

		if len(metadata.Layers) == 0 {
			printWarning(fmt.Sprintf("      No layers found in Map Server metadata at %s", fetchURL))
		}

		layerMap := make(map[int]MapServiceLayer)
		for _, layer := range metadata.Layers {
			layerMap[layer.ID] = layer
		}

		layerHierarchy := make(map[int][]string)
		var buildPath func(layerID int) []string
		buildPath = func(layerID int) []string {
			if path, exists := layerHierarchy[layerID]; exists {
				return path
			}
			layer, ok := layerMap[layerID]
			if !ok {
				return []string{}
			}

			layerHierarchy[layerID] = []string{}

			var path []string
			if layer.ParentLayerId != -1 {
				parentPath := buildPath(layer.ParentLayerId)
				path = append(parentPath, layer.Name)
			} else {
				path = []string{layer.Name}
			}
			layerHierarchy[layerID] = path
			return path
		}

		for _, layer := range metadata.Layers {
			if layer.Type == "Feature Layer" {
				parentPath := buildPath(layer.ID)
				if len(parentPath) > 0 {
					parentPath = parentPath[:len(parentPath)-1]
				}

				availableLayers = append(availableLayers, AvailableLayerInfo{
					ID:             strconv.Itoa(layer.ID),
					Name:           layer.Name,
					Type:           layer.Type,
					GeometryType:   layer.GeometryType,
					ServiceURL:     serviceURL,
					ParentPath:     parentPath,
					IsFeatureLayer: true,
				})
			} else {
				fmt.Printf("    Skipping layer '%s' (ID: %d) - Type is '%s', not 'Feature Layer'.\n", layer.Name, layer.ID, layer.Type)
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported service type for fetching layers: %s", serviceType)
	}

	return availableLayers, nil
}

// selectAndAddLayers prompts the user to select layers from a list and adds them to the processing queue.
func selectAndAddLayers(availableLayers []AvailableLayerInfo, selectAll bool) error {
	if selectAll {
		printInfo("  --select-all flag detected, selecting all found Feature Layers.")
		if len(availableLayers) == 0 {
			printWarning("  No layers available to select.")
			return nil
		}
		for _, layer := range availableLayers {
			uniqueKey := fmt.Sprintf("%s/%s", layer.ServiceURL, layer.ID)
			layersToProcess[uniqueKey] = layer
		}
		return nil
	}

	fmt.Println("  Please select the Feature Layers to process:")
	for i, layer := range availableLayers {
		pathStr := ""
		if len(layer.ParentPath) > 0 {
			pathStr = fmt.Sprintf(" (Path: %s)", strings.Join(layer.ParentPath, " > "))
		}
		fmt.Printf("    %d: %s (ID: %s, Type: %s, Geometry: %s)%s\n", i+1, layer.Name, layer.ID, layer.Type, layer.GeometryType, pathStr)
	}
	fmt.Print("  Enter comma-separated numbers (e.g., 1,3,4) or 'all': ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if strings.ToLower(input) == "all" {
		for _, layer := range availableLayers {
			uniqueKey := fmt.Sprintf("%s/%s", layer.ServiceURL, layer.ID)
			layersToProcess[uniqueKey] = layer
		}
	} else {
		parts := strings.Split(input, ",")
		selectedIndices := make(map[int]bool)
		for _, part := range parts {
			numStr := strings.TrimSpace(part)
			if num, err := strconv.Atoi(numStr); err == nil {
				if num >= 1 && num <= len(availableLayers) {
					selectedIndices[num-1] = true
				} else {
					printWarning(fmt.Sprintf("  Warning: Invalid selection '%d', out of range.", num))
				}
			} else {
				printWarning(fmt.Sprintf("  Warning: Invalid input '%s', skipping.", numStr))
			}
		}

		if len(selectedIndices) == 0 {
			printWarning("  No valid layers selected.")
			return nil
		}

		for index := range selectedIndices {
			layer := availableLayers[index]
			uniqueKey := fmt.Sprintf("%s/%s", layer.ServiceURL, layer.ID)
			layersToProcess[uniqueKey] = layer
		}
	}
	return nil
}

// processSelectedLayer processes a single selected layer and exports it to the specified format.
func processSelectedLayer(layerInfo AvailableLayerInfo, format, outputDir string, overwrite, skipExisting bool, prefix string, excludeSymbols bool) error {
	metadataURL := fmt.Sprintf("%s/%s?f=json", layerInfo.ServiceURL, layerInfo.ID)
	var layerMetadata Layer
	err := fetchAndDecode(metadataURL, &layerMetadata)
	if err != nil {
		return fmt.Errorf("failed to fetch layer metadata from %s: %v", metadataURL, err)
	}
	if layerMetadata.Error != nil {
		return fmt.Errorf("layer metadata API error for %s: %s", metadataURL, layerMetadata.Error.Message)
	}

	actualLayerName := layerMetadata.Name
	if actualLayerName == "" {
		actualLayerName = layerInfo.Name
	}
	if actualLayerName == "" {
		actualLayerName = fmt.Sprintf("Layer_%s", layerInfo.ID)
	}

	features, err := fetchFeatures(layerInfo.ServiceURL, layerInfo.ID)
	if err != nil {
		if strings.Contains(err.Error(), "no features found") {
			return fmt.Errorf("no features found")
		}
		return fmt.Errorf("failed to fetch features: %v", err)
	}

	// Add symbol information to features if available in layer metadata and not excluded
	if !excludeSymbols && layerMetadata.DrawingInfo != nil && layerMetadata.DrawingInfo.Renderer != nil {
		renderer := layerMetadata.DrawingInfo.Renderer

		// Handle default symbol
		if renderer.DefaultSymbol != nil {
			for i := range features {
				if features[i].Attributes == nil {
					features[i].Attributes = make(map[string]interface{})
				}
				features[i].Attributes["symbol"] = renderer.DefaultSymbol
			}
		}

		// Handle unique value renderer
		if renderer.Type == "uniqueValue" && len(renderer.UniqueValueGroups) > 0 {
			for _, group := range renderer.UniqueValueGroups {
				for _, class := range group.Classes {
					if class.Symbol != nil {
						for i := range features {
							if features[i].Attributes == nil {
								features[i].Attributes = make(map[string]interface{})
							}
							// Check if feature matches this class's values
							for _, valueSet := range class.Values {
								if len(valueSet) > 0 {
									fieldValue := features[i].Attributes[renderer.Field1]
									if fieldValue != nil && fmt.Sprintf("%v", fieldValue) == valueSet[0] {
										features[i].Attributes["symbol"] = class.Symbol
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}

	var data string
	var fileExt string
	switch strings.ToLower(format) {
	case "geojson":
		geojsonData, err := convertToGeoJSON(features)
		if err != nil {
			return fmt.Errorf("failed to convert features to GeoJSON objects: %v", err)
		}
		data, err = marshalGeoJSON(geojsonData, actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to marshal GeoJSON: %v", err)
		}
		fileExt = "geojson"
	case "kml":
		geojsonData, err := convertToGeoJSON(features)
		if err != nil {
			return fmt.Errorf("failed to convert features to GeoJSON for KML: %v", err)
		}
		data, err = convertGeoJSONToKML(geojsonData, actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to convert to KML: %v", err)
		}
		fileExt = "kml"
	case "gpx":
		geojsonData, err := convertToGeoJSON(features)
		if err != nil {
			return fmt.Errorf("failed to convert features to GeoJSON for GPX: %v", err)
		}
		data, err = convertGeoJSONToGPX(geojsonData, actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to convert to GPX: %v", err)
		}
		fileExt = "gpx"
	case "json":
		jsonDataBytes, err := json.MarshalIndent(features, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal features to JSON: %v", err)
		}
		data = string(jsonDataBytes)
		fileExt = "json"
	case "csv":
		data, err = convertFeaturesToCSV(features)
		if err != nil {
			return fmt.Errorf("failed to convert features to CSV: %v", err)
		}
		fileExt = "csv"
	case "text":
		txtData, err := convertFeaturesToText(features, actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to convert features to text: %v", err)
		}
		data = txtData
		fileExt = "txt"
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	safeFilenameBase := strings.ReplaceAll(actualLayerName, " ", "_")
	safeFilenameBase = regexp.MustCompile(`[<>:"/\|?* - ]`).ReplaceAllString(safeFilenameBase, "")
	if safeFilenameBase == "" {
		safeFilenameBase = fmt.Sprintf("Layer_%s", layerInfo.ID)
	}
	if prefix != "" {
		safeFilenameBase = prefix + safeFilenameBase
	}

	filename := fmt.Sprintf("%s.%s", safeFilenameBase, fileExt)
	outputPath := filepath.Join(outputDir, filename)

	if _, err := os.Stat(outputPath); err == nil {
		if skipExisting {
			return fmt.Errorf("skipped existing file")
		}
		if !overwrite {
			return fmt.Errorf("output file %s already exists. Use --overwrite or --skip-existing", outputPath)
		}
		printWarning(fmt.Sprintf("  Overwriting existing file: %s", outputPath))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check output file status %s: %v", outputPath, err)
	}

	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory %s: %v", outputDir, err)
	}

	if err := os.WriteFile(outputPath, []byte(data), 0600); err != nil {
		return fmt.Errorf("failed to write output file %s: %v", outputPath, err)
	}

	return nil
}

// normalizeArcGISURL normalizes an ArcGIS URL.
func normalizeArcGISURL(rawURL string) string {
	lowerURL := strings.ToLower(rawURL)
	isArcGISService := strings.Contains(lowerURL, "/rest/services") || strings.Contains(lowerURL, "/arcgis/rest")
	isAGOLItem := strings.Contains(lowerURL, "arcgis.com/home/item.html")

	if !isArcGISService && !isAGOLItem {
		// If it doesn't look like an ArcGIS service or item URL, only ensure it has a scheme
		u, err := url.Parse(rawURL)
		if err == nil && u.Scheme == "" {
			// Check if it looks like a domain/path without scheme
			if strings.Contains(rawURL, ".") && !strings.Contains(rawURL, " ") && !strings.HasPrefix(rawURL, "/") {
				return "https://" + rawURL
			}
		}
		// Otherwise, return as is or handle parsing errors if needed
		return rawURL
	}

	// If it IS an ArcGIS service URL (or AGOL item URL, though AGOL items are less likely to need scheme added)
	u, err := url.Parse(rawURL)
	if err != nil {
		fmt.Printf("Warning: Failed to parse URL for normalization: %v\n", err)
		return rawURL // Return original on parse error
	}

	// Ensure scheme (only if it doesn't exist)
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	if isArcGISService {
		// Normalize path casing only for service URLs
		pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i, part := range pathParts {
			lowerPart := strings.ToLower(part)
			if lowerPart == "arcgis" {
				pathParts[i] = "ArcGIS"
			} else if lowerPart == "rest" {
				pathParts[i] = "rest"
			} else if lowerPart == "services" {
				pathParts[i] = "services"
			} else if lowerPart == "featureserver" {
				pathParts[i] = "FeatureServer"
			} else if lowerPart == "mapserver" {
				pathParts[i] = "MapServer"
			}
		}
		// Reconstruct path, respecting if original had leading slash
		if strings.HasPrefix(u.Path, "/") {
			u.Path = "/" + strings.Join(pathParts, "/")
		} else {
			u.Path = strings.Join(pathParts, "/")
		}

		// Handle trailing slashes more carefully
		lowerPathEnd := ""
		if len(pathParts) > 0 {
			lowerPathEnd = strings.ToLower(pathParts[len(pathParts)-1])
		}

		// Check if it's a base service URL
		isBaseServiceURL := lowerPathEnd == "mapserver" || lowerPathEnd == "featureserver"

		if isBaseServiceURL {
			// Ensure base service URLs end with a slash
			if !strings.HasSuffix(u.Path, "/") {
				u.Path += "/"
			}
		} else {
			// Remove trailing slash if it's not a base service URL
			if len(u.Path) > 1 && strings.HasSuffix(u.Path, "/") {
				u.Path = u.Path[:len(u.Path)-1]
			}
		}

		// Remove ?f= query parameter
		q := u.Query()
		q.Del("f")
		u.RawQuery = q.Encode()
	}

	return u.String()
}

// isValidHTTPURL checks if a URL is a valid HTTP or HTTPS URL.
func isValidHTTPURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// fetchFeatures fetches features from an ArcGIS FeatureServer layer.
func fetchFeatures(baseURL, layerID string) ([]Feature, error) {
	queryURL := fmt.Sprintf("%s/%s/query", baseURL, layerID)
	u, _ := url.Parse(queryURL)
	q := u.Query()
	q.Set("f", "json")
	q.Set("where", "1=1")
	q.Set("outFields", "*")
	q.Set("returnGeometry", "true")
	q.Set("outSR", "4326")
	u.RawQuery = q.Encode()

	printInfo(fmt.Sprintf("    Fetching features: %s", u.String()))

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create feature query request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return nil, fmt.Errorf("feature fetch request timed out: %v", err)
		}
		return nil, fmt.Errorf("feature fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feature fetch request failed with status %d", resp.StatusCode)
	}

	var featureResp FeatureResponse
	if err := json.NewDecoder(resp.Body).Decode(&featureResp); err != nil {
		return nil, fmt.Errorf("failed to parse feature response: %v", err)
	}

	if featureResp.Error != nil {
		return nil, fmt.Errorf("feature query API error: %s", featureResp.Error.Message)
	}

	if len(featureResp.Features) == 0 && !featureResp.ExceededTransferLimit {
		return nil, fmt.Errorf("no features found for layer %s at %s", layerID, baseURL)
	}
	if featureResp.ExceededTransferLimit {
		printWarning(fmt.Sprintf("  Warning: Feature transfer limit exceeded for layer %s. Results may be incomplete.", layerID))
	}

	return featureResp.Features, nil
}

// convertToGeoJSON converts a slice of Feature structs to a GeoJSON FeatureCollection.
func convertToGeoJSON(features []Feature) (*GeoJSON, error) {
	geoJSON := GeoJSON{
		Type: "FeatureCollection",
		CRS: CRS{
			Type: "name",
			Properties: CRSProps{
				Name: "urn:ogc:def:crs:OGC:1.3:CRS84",
			},
		},
		Features: []GeoJSONFeature{},
	}

	for _, feature := range features {
		var geometry map[string]interface{}
		geometryMap, geomOk := feature.Geometry.(map[string]interface{})
		if geomOk {
			geometry = geometryMap
		}

		var geoJSONFeature GeoJSONFeature
		if geometry != nil {
			if xVal, xOk := geometry["x"]; xOk {
				if yVal, yOk := geometry["y"]; yOk {
					x, xFloatOk := xVal.(float64)
					y, yFloatOk := yVal.(float64)
					if xFloatOk && yFloatOk {
						geoJSONFeature.Geometry = map[string]interface{}{
							"type":        "Point",
							"coordinates": []float64{x, y},
						}
					}
				}
			} else if paths, ok := geometry["paths"]; ok {
				pathArray, pathArrayOk := paths.([]interface{})
				if pathArrayOk && len(pathArray) > 0 {
					firstPath, firstPathOk := pathArray[0].([]interface{})
					if firstPathOk {
						coords := [][]float64{}
						for _, p := range firstPath {
							point, pointOk := p.([]interface{})
							if pointOk && len(point) >= 2 {
								x, xOk := point[0].(float64)
								y, yOk := point[1].(float64)
								if xOk && yOk {
									coords = append(coords, []float64{x, y})
								}
							}
						}
						geoJSONFeature.Geometry = map[string]interface{}{
							"type":        "LineString",
							"coordinates": coords,
						}
					}
				}
			} else if rings, ok := geometry["rings"]; ok {
				ringArray, ringArrayOk := rings.([]interface{})
				if ringArrayOk && len(ringArray) > 0 {
					allRings := [][][]float64{}
					for _, r := range ringArray {
						ringCoords, ringCoordsOk := r.([]interface{})
						if ringCoordsOk {
							singleRing := [][]float64{}
							for _, p := range ringCoords {
								point, pointOk := p.([]interface{})
								if pointOk && len(point) >= 2 {
									x, xOk := point[0].(float64)
									y, yOk := point[1].(float64)
									if xOk && yOk {
										singleRing = append(singleRing, []float64{x, y})
									}
								}
							}
							if len(singleRing) > 0 && (singleRing[0][0] != singleRing[len(singleRing)-1][0] || singleRing[0][1] != singleRing[len(singleRing)-1][1]) {
								singleRing = append(singleRing, singleRing[0])
							}
							allRings = append(allRings, singleRing)
						}
					}
					geoJSONFeature.Geometry = map[string]interface{}{
						"type":        "Polygon",
						"coordinates": allRings,
					}
				}
			}
		}

		if geoJSONFeature.Geometry != nil {
			geoJSONFeature.Type = "Feature"
			geoJSONFeature.Properties = feature.Attributes

			// Add symbol information if available in attributes
			if symbolData, ok := feature.Attributes["symbol"]; ok {
				if symbolMap, ok := symbolData.(map[string]interface{}); ok {
					symbol := &Symbol{
						Type:        getString(symbolMap, "type"),
						URL:         getString(symbolMap, "url"),
						ImageData:   getString(symbolMap, "imageData"),
						ContentType: getString(symbolMap, "contentType"),
						Width:       getInt(symbolMap, "width"),
						Height:      getInt(symbolMap, "height"),
						XOffset:     getInt(symbolMap, "xoffset"),
						YOffset:     getInt(symbolMap, "yoffset"),
						Angle:       getFloat(symbolMap, "angle"),
					}
					geoJSONFeature.Symbol = symbol
				}
			}

			geoJSON.Features = append(geoJSON.Features, geoJSONFeature)
		} else if feature.Attributes != nil && len(feature.Attributes) > 0 {
			printWarning(fmt.Sprintf("  Warning: Feature found with attributes but no convertible geometry. Skipping feature."))
		}
	}

	if len(geoJSON.Features) == 0 && len(features) > 0 {
		printWarning("  Warning: Processed features resulted in an empty GeoJSON FeatureCollection (likely due to geometry issues).")
	}

	return &geoJSON, nil
}

// Helper functions to safely extract values from map[string]interface{}
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return int(num)
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0
}

// marshalGeoJSON marshals a GeoJSON struct into a JSON string.
func marshalGeoJSON(geoJSON *GeoJSON, layerName string) (string, error) {
	data, err := json.MarshalIndent(geoJSON, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// convertGeoJSONToKML converts a GeoJSON FeatureCollection to a KML string.
func convertGeoJSONToKML(geoJSON *GeoJSON, layerName string) (string, error) {
	var placemarks strings.Builder
	for _, feature := range geoJSON.Features {
		if feature.Geometry == nil {
			continue
		}

		name := getFeatureName(feature)
		description := formatProperties(feature.Properties)

		geometryMap := feature.Geometry.(map[string]interface{})
		geometryType := geometryMap["type"].(string)
		coordinates := geometryMap["coordinates"]

		var geometryString string
		switch geometryType {
		case "Point":
			coords, ok := coordinates.([]float64)
			if ok && len(coords) >= 2 {
				geometryString = fmt.Sprintf("<Point><coordinates>%.10f,%.10f,0</coordinates></Point>", coords[0], coords[1])
			}
		case "LineString":
			coords, ok := coordinates.([][]float64)
			if ok && len(coords) > 0 {
				coordStr := make([]string, len(coords))
				for i, c := range coords {
					coordStr[i] = fmt.Sprintf("%.10f,%.10f,0", c[0], c[1])
				}
				geometryString = fmt.Sprintf("<LineString><coordinates>%s</coordinates></LineString>", strings.Join(coordStr, " "))
			}
		case "Polygon":
			coords, ok := coordinates.([][][]float64)
			if ok && len(coords) > 0 {
				var outerBoundary, innerBoundaries strings.Builder
				outerRing := coords[0]
				outerCoordStr := make([]string, len(outerRing))
				for i, c := range outerRing {
					outerCoordStr[i] = fmt.Sprintf("%.10f,%.10f,0", c[0], c[1])
				}
				outerBoundary.WriteString(fmt.Sprintf("<outerBoundaryIs><LinearRing><coordinates>%s</coordinates></LinearRing></outerBoundaryIs>", strings.Join(outerCoordStr, " ")))

				if len(coords) > 1 {
					for _, innerRing := range coords[1:] {
						innerCoordStr := make([]string, len(innerRing))
						for i, c := range innerRing {
							innerCoordStr[i] = fmt.Sprintf("%.10f,%.10f,0", c[0], c[1])
						}
						innerBoundaries.WriteString(fmt.Sprintf("<innerBoundaryIs><LinearRing><coordinates>%s</coordinates></LinearRing></innerBoundaryIs>", strings.Join(innerCoordStr, " ")))
					}
				}
				geometryString = fmt.Sprintf("<Polygon>%s%s</Polygon>", outerBoundary.String(), innerBoundaries.String())
			}
		default:
			printWarning(fmt.Sprintf("  Warning: Unsupported geometry type for KML conversion: %s", geometryType))
		}

		if geometryString != "" {
			placemarks.WriteString(fmt.Sprintf(`
        <Placemark>
            <name>%s</name>
            <description><![CDATA[%s]]></description>
            %s
        </Placemark>`, escapeXML(name), description, geometryString))
		}
	}

	kml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
    <Document>
        <name>%s</name>%s
    </Document>
</kml>`, escapeXML(layerName), placemarks.String())

	return kml, nil
}

// convertGeoJSONToGPX converts a GeoJSON FeatureCollection to a GPX string.
func convertGeoJSONToGPX(geoJSON *GeoJSON, layerName string) (string, error) {
	var waypoints strings.Builder
	var tracks strings.Builder

	for _, feature := range geoJSON.Features {
		if feature.Geometry == nil {
			continue
		}

		name := getFeatureName(feature)
		desc := formatProperties(feature.Properties, ", ")

		geometryMap := feature.Geometry.(map[string]interface{})
		geometryType := geometryMap["type"].(string)
		coordinates := geometryMap["coordinates"]

		switch geometryType {
		case "Point":
			coords, ok := coordinates.([]float64)
			if ok && len(coords) >= 2 {
				waypoints.WriteString(fmt.Sprintf(`
    <wpt lat="%.10f" lon="%.10f">
        <name>%s</name>
        <desc>%s</desc>
    </wpt>`, coords[1], coords[0], escapeXML(name), escapeXML(desc)))
			}
		case "LineString":
			coords, ok := coordinates.([][]float64)
			if ok && len(coords) > 0 {
				tracks.WriteString(fmt.Sprintf(`
    <trk>
        <name>%s</name>
        <desc>%s</desc>
        <trkseg>`, escapeXML(name), escapeXML(desc)))
				for _, c := range coords {
					tracks.WriteString(fmt.Sprintf(`<trkpt lat="%.10f" lon="%.10f"></trkpt>`, c[1], c[0]))
				}
				tracks.WriteString(`
        </trkseg>
    </trk>`)
			}
		case "Polygon":
			coords, ok := coordinates.([][][]float64)
			if ok && len(coords) > 0 {
				outerRing := coords[0]
				tracks.WriteString(fmt.Sprintf(`
    <trk>
        <name>%s (Boundary)</name>
        <desc>%s</desc>
        <trkseg>`, escapeXML(name), escapeXML(desc)))
				for _, c := range outerRing {
					tracks.WriteString(fmt.Sprintf(`<trkpt lat="%.10f" lon="%.10f"></trkpt>`, c[1], c[0]))
				}
				tracks.WriteString(`
        </trkseg>
    </trk>`)
			}
		default:
			printWarning(fmt.Sprintf("  Warning: Unsupported geometry type for GPX conversion: %s", geometryType))
		}
	}

	gpxContent := waypoints.String() + tracks.String()

	gpx := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="arcgis-utils-go"
    xmlns="http://www.topografix.com/GPX/1/1"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd">
    <metadata>
        <name>%s</name>
    </metadata>%s
</gpx>`, escapeXML(layerName), gpxContent)

	return gpx, nil
}

// getFeatureName extracts a suitable name from a GeoJSON feature's properties.
func getFeatureName(feature GeoJSONFeature) string {
	props := feature.Properties
	for _, key := range []string{"name", "Name", "NAME", "title", "Title", "TITLE", "OBJECTID", "FID"} {
		if val, ok := props[key]; ok && val != nil {
			return fmt.Sprintf("%v", val)
		}
	}
	return "Feature"
}

// formatProperties formats a map of properties into a string.
func formatProperties(props map[string]interface{}, separator ...string) string {
	sep := "<br>"
	if len(separator) > 0 {
		sep = separator[0]
	}
	var parts []string
	for k, v := range props {
		if k == "geometry" {
			continue
		}
		parts = append(parts, fmt.Sprintf("<strong>%s</strong>: %v", escapeXML(k), escapeXML(fmt.Sprintf("%v", v))))
	}
	return strings.Join(parts, sep)
}

// escapeXML escapes XML special characters in a string.
func escapeXML(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
		"/", "&#x2F;",
	).Replace(s)
}

// geometryToWKT converts a geometry interface to a WKT string.
func geometryToWKT(geometry interface{}) string {
	if geometry == nil {
		return ""
	}

	geomMap, ok := geometry.(map[string]interface{})
	if !ok {
		return ""
	}

	if xVal, xOk := geomMap["x"]; xOk {
		if yVal, yOk := geomMap["y"]; yOk {
			x, xFloatOk := xVal.(float64)
			y, yFloatOk := yVal.(float64)
			if xFloatOk && yFloatOk {
				return fmt.Sprintf("POINT (%.10f %.10f)", x, y)
			}
		}
	} else if paths, pOk := geomMap["paths"]; pOk {
		pathArray, pathArrayOk := paths.([]interface{})
		if pathArrayOk && len(pathArray) > 0 {
			firstPath, firstPathOk := pathArray[0].([]interface{})
			if firstPathOk {
				var points []string
				for _, p := range firstPath {
					point, pointOk := p.([]interface{})
					if pointOk && len(point) >= 2 {
						x, xOk := point[0].(float64)
						y, yOk := point[1].(float64)
						if xOk && yOk {
							points = append(points, fmt.Sprintf("%.10f %.10f", x, y))
						}
					}
				}
				if len(points) == 0 { // Check if points were actually added
					return ""
				}
				return fmt.Sprintf("LINESTRING (%s)", strings.Join(points, ", "))
			}
		}
	} else if rings, rOk := geomMap["rings"]; rOk {
		ringArray, ringArrayOk := rings.([]interface{})
		if ringArrayOk && len(ringArray) > 0 {
			var polygonRings []string
			for _, r := range ringArray {
				ringCoords, ringCoordsOk := r.([]interface{})
				if ringCoordsOk {
					var points []string
					for _, p := range ringCoords {
						point, pointOk := p.([]interface{})
						if pointOk && len(point) >= 2 {
							x, xOk := point[0].(float64)
							y, yOk := point[1].(float64)
							if xOk && yOk {
								points = append(points, fmt.Sprintf("%.10f %.10f", x, y))
							}
						}
					}
					// Ensure ring is closed for WKT
					if len(points) > 0 {
						if points[0] != points[len(points)-1] {
							points = append(points, points[0])
						}
						polygonRings = append(polygonRings, fmt.Sprintf("(%s)", strings.Join(points, ", ")))
					}
				}
			}
			if len(polygonRings) == 0 { // Check if any valid rings were processed
				return ""
			}
			return fmt.Sprintf("POLYGON (%s)", strings.Join(polygonRings, ", "))
		}
	}

	return ""
}

// convertFeaturesToCSV converts a slice of Feature structs to a CSV string.
func convertFeaturesToCSV(features []Feature) (string, error) {
	if len(features) == 0 {
		// Return empty string or just header if no features? Decide based on desired behavior.
		// For now, let's return an error like GeoJSON conversion does for empty results.
		// Or perhaps just return the header? Returning empty string for now.
		// If a header is desired even for no features, create it here.
		return "", nil // Return empty string if no features
	}

	// Determine all unique headers from all features' attributes
	headerMap := make(map[string]bool)
	for _, feature := range features {
		for k := range feature.Attributes {
			headerMap[k] = true
		}
	}

	var headers []string
	for k := range headerMap {
		headers = append(headers, k)
	}
	sort.Strings(headers)                     // Sort for consistent column order
	headers = append(headers, "WKT_Geometry") // Add geometry column header

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Write header row
	if err := w.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %v", err)
	}

	// Write data rows
	for _, feature := range features {
		row := make([]string, len(headers))
		for i, header := range headers {
			if header == "WKT_Geometry" {
				row[i] = geometryToWKT(feature.Geometry)
			} else {
				if val, ok := feature.Attributes[header]; ok && val != nil {
					row[i] = fmt.Sprintf("%v", val)
				} else {
					row[i] = "" // Handle nil or missing attributes
				}
			}
		}
		if err := w.Write(row); err != nil {
			// Log warning but continue processing other rows
			printWarning(fmt.Sprintf("Warning: Failed to write row to CSV: %v", err))
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return "", fmt.Errorf("error during CSV writing: %v", err)
	}

	return buf.String(), nil
}

// convertFeaturesToText converts a slice of Feature structs to a formatted text string.
func convertFeaturesToText(features []Feature, layerName string) (string, error) {
	if len(features) == 0 {
		return "", fmt.Errorf("no features to convert to text")
	}

	var output strings.Builder

	output.WriteString(fmt.Sprintf("Layer: %s\n", layerName))
	output.WriteString(fmt.Sprintf("Total Features: %d\n", len(features)))
	output.WriteString("========================================\n\n")

	for i, feature := range features {
		output.WriteString(fmt.Sprintf("--- Feature %d ---\n", i+1))

		// Sort attribute keys for consistent order
		var keys []string
		for k := range feature.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		output.WriteString("Attributes:\n")
		for _, k := range keys {
			output.WriteString(fmt.Sprintf("  %s: %v\n", k, feature.Attributes[k]))
		}

		output.WriteString("Geometry (WKT):\n")
		wkt := geometryToWKT(feature.Geometry)
		if wkt == "" {
			output.WriteString("  <No Geometry>\n")
		} else {
			output.WriteString(fmt.Sprintf("  %s\n", wkt))
		}
		output.WriteString("\n") // Add a blank line between features
	}

	return output.String(), nil
}
