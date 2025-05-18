// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

// Package main implements a command-line tool to download data from various public ArcGIS sources
// and convert it to common geospatial formats. It supports Feature Layers, Feature Servers,
// Map Servers, and ArcGIS Online Items including Web Maps.
//
// The tool provides interactive layer selection, concurrent processing, and various output formats
// including GeoJSON, KML, GPX, CSV, JSON, and Text. It also handles symbol information and
// supports custom output naming and request timeouts.
//
// Usage:
//
//	arcgis-utils [-format format] [-output dir] [-select-all] [-overwrite] [-skip-existing]
//	             [-prefix prefix] [-timeout seconds] [-exclude-symbols] [-save-symbols] -url <ARCGIS_URL>
//
// Flags:
//
//	-url string
//	      ArcGIS resource URL (required)
//	-format string
//	      Output format (geojson, kml, gpx, csv, json, text) (default "geojson")
//	-output string
//	      Output directory (default: current directory)
//	-select-all
//	      Process all layers without prompting
//	-overwrite
//	      Overwrite existing output files
//	-skip-existing
//	      Skip processing if output file exists
//	-prefix string
//	      Add prefix to output filenames
//	-timeout int
//	      HTTP request timeout in seconds (default 30)
//	-exclude-symbols
//	      Exclude symbol information from output
//	-save-symbols
//	      Save symbology to separate folder
//	-no-color
//	      Disable colored terminal output

package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/arcgis"
	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
	"github.com/Sudo-Ivan/arcgis-utils/pkg/export"
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
var layersToProcess = make(map[string]arcgis.AvailableLayerInfo)

func main() {
	urlPtr := flag.String("url", "", "ArcGIS Feature Layer, Feature Server, Map Server, or ArcGIS Online Item URL")
	formatPtr := flag.String("format", "geojson", "Output format (geojson, kml, gpx, csv, json, txt)")
	outputPtr := flag.String("output", "", "Output directory (default: current directory)")
	selectAllPtr := flag.Bool("select-all", false, "Select all found Feature Layers automatically (no prompt)")
	noColorPtr := flag.Bool("no-color", false, "Disable colored output")
	overwritePtr := flag.Bool("overwrite", false, "Overwrite existing output files")
	skipExistingPtr := flag.Bool("skip-existing", false, "Skip processing if output file already exists")
	prefixPtr := flag.String("prefix", "", "Prefix for output filenames")
	timeoutPtr := flag.Int("timeout", 30, "HTTP request timeout in seconds")
	excludeSymbolsPtr := flag.Bool("exclude-symbols", false, "Exclude symbol information from output")
	saveSymbolsPtr := flag.Bool("save-symbols", false, "Save symbology/images to a separate folder")

	flag.Parse()

	useColor = !*noColorPtr

	if *urlPtr == "" {
		printError("URL is required")
		flag.Usage()
		os.Exit(1)
	}

	inputURL := *urlPtr
	if !arcgis.IsValidHTTPURL(inputURL) {
		normalizedURL := arcgis.NormalizeArcGISURL(inputURL)
		if !arcgis.IsValidHTTPURL(normalizedURL) {
			fmt.Println("Error: Invalid URL provided.")
			os.Exit(1)
		}
		inputURL = normalizedURL
	} else {
		inputURL = arcgis.NormalizeArcGISURL(inputURL)
	}

	outputDir := *outputPtr
	if outputDir == "" {
		outputDir, _ = os.Getwd()
	}

	client := arcgis.NewClient(time.Duration(*timeoutPtr) * time.Second)

	var err error
	if arcgis.IsArcGISOnlineItemURL(inputURL) {
		fmt.Println("Detected ArcGIS Online Item URL...")
		err = handleArcGISOnlineItem(client, inputURL, *selectAllPtr)
	} else if strings.Contains(strings.ToLower(inputURL), "/mapserver") {
		fmt.Println("Detected Map Server URL...")
		err = handleMapServerURL(client, inputURL, *selectAllPtr)
	} else if strings.Contains(strings.ToLower(inputURL), "/featureserver") {
		fmt.Println("Detected Feature Server URL...")
		err = handleFeatureServerURL(client, inputURL, *selectAllPtr)
	} else {
		fmt.Println("Assuming single Feature Layer URL...")
		parts := strings.Split(inputURL, "/")
		if len(parts) < 2 {
			err = fmt.Errorf("invalid single layer URL format")
		} else {
			layerID := parts[len(parts)-1]
			baseURL := strings.Join(parts[:len(parts)-1], "/")
			layersToProcess[inputURL] = arcgis.AvailableLayerInfo{
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
		saveSymbolsCopy := *saveSymbolsPtr

		go func() {
			defer wg.Done()
			printInfo(fmt.Sprintf("Processing Layer: %s (ID: %s)", layerInfoCopy.Name, layerInfoCopy.ID))
			err := processSelectedLayer(client, layerInfoCopy, formatCopy, outputDirCopy, overwriteCopy, skipExistingCopy, prefixCopy, excludeSymbolsCopy, saveSymbolsCopy)
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

// handleArcGISOnlineItem handles processing for an ArcGIS Online item URL.
func handleArcGISOnlineItem(client *arcgis.Client, itemPageURL string, selectAll bool) error {
	itemData, err := client.HandleArcGISOnlineItem(itemPageURL)
	if err != nil {
		return err
	}

	switch itemData.Type {
	case "Feature Service":
		if itemData.URL == "" {
			return fmt.Errorf("feature Service item has no URL")
		}
		return handleFeatureServerURL(client, itemData.URL, selectAll)
	case "Map Service":
		if itemData.URL == "" {
			return fmt.Errorf("map Service item has no URL")
		}
		return handleMapServerURL(client, itemData.URL, selectAll)
	case "Web Map":
		return handleWebMap(client, itemData.ID, selectAll)
	default:
		return fmt.Errorf("unsupported item type: %s. Currently supports Feature Service, Map Service, Web Map", itemData.Type)
	}
}

// handleWebMap handles processing for an ArcGIS Online Web Map item.
func handleWebMap(client *arcgis.Client, itemID string, selectAll bool) error {
	webMapData, err := client.HandleWebMap(itemID)
	if err != nil {
		return err
	}

	availableLayers := []arcgis.AvailableLayerInfo{}
	for _, opLayer := range webMapData.OperationalLayers {
		processOperationalLayer(client, opLayer, []string{}, &availableLayers)
	}

	if len(availableLayers) == 0 {
		fmt.Println("  No processable Feature Layers found in this Web Map.")
		return nil
	}

	fmt.Printf("  Found %d potential Feature Layers in Web Map.\n", len(availableLayers))
	return selectAndAddLayers(availableLayers, selectAll)
}

// processOperationalLayer recursively processes operational layers in a Web Map.
func processOperationalLayer(client *arcgis.Client, opLayer arcgis.OperationalLayer, parentPath []string, availableLayers *[]arcgis.AvailableLayerInfo) {
	currentPath := append(parentPath, opLayer.Title)

	if opLayer.LayerType == "GroupLayer" || len(opLayer.Layers) > 0 {
		fmt.Printf("    Processing Group: %s\n", strings.Join(currentPath, " > "))
		for _, subLayer := range opLayer.Layers {
			processOperationalLayer(client, subLayer, currentPath, availableLayers)
		}
	} else if opLayer.FeatureCollection != nil && len(opLayer.FeatureCollection.Layers) > 0 {
		fmt.Printf("    Skipping Inline Feature Collection: %s (Direct processing not yet implemented)\n", strings.Join(currentPath, " > "))
	} else if opLayer.URL != "" && (strings.Contains(strings.ToLower(opLayer.URL), "/featureserver") || strings.Contains(strings.ToLower(opLayer.URL), "/mapserver")) {
		serviceURL := arcgis.NormalizeArcGISURL(opLayer.URL)
		layerIDStr := ""
		parts := strings.Split(serviceURL, "/")
		lastPart := parts[len(parts)-1]
		if _, err := strconv.Atoi(lastPart); err == nil {
			layerIDStr = lastPart
			serviceURL = strings.Join(parts[:len(parts)-1], "/")
		} else {
			fmt.Printf("    Service URL found: %s for layer %s. Fetching service layers...\n", serviceURL, opLayer.Title)
			var subLayers []arcgis.AvailableLayerInfo
			var err error
			if strings.Contains(strings.ToLower(serviceURL), "/featureserver") {
				subLayers, err = client.FetchServiceLayers(serviceURL, "FeatureServer")
			} else {
				subLayers, err = client.FetchServiceLayers(serviceURL, "MapServer")
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
			layerInfo := arcgis.AvailableLayerInfo{
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
		itemData, err := client.HandleArcGISOnlineItem(fmt.Sprintf("https://www.arcgis.com/home/item.html?id=%s", opLayer.ItemID))
		if err != nil || itemData.Error != nil || itemData.URL == "" {
			fmt.Printf("      Warning: Failed to fetch or use referenced item %s: %v\n", opLayer.ItemID, err)
			if itemData.Error != nil {
				fmt.Printf("      Item API Error: %s\n", itemData.Error.Message)
			}
		} else {
			fmt.Printf("      Referenced Item Type: %s, URL: %s\n", itemData.Type, itemData.URL)
			var subLayers []arcgis.AvailableLayerInfo
			var fetchErr error
			if strings.Contains(strings.ToLower(itemData.URL), "/featureserver") {
				subLayers, fetchErr = client.FetchServiceLayers(itemData.URL, "FeatureServer")
			} else if strings.Contains(strings.ToLower(itemData.URL), "/mapserver") {
				subLayers, fetchErr = client.FetchServiceLayers(itemData.URL, "MapServer")
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
func handleMapServerURL(client *arcgis.Client, mapServerURL string, selectAll bool) error {
	layers, err := client.FetchServiceLayers(mapServerURL, "MapServer")
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
func handleFeatureServerURL(client *arcgis.Client, featureServerURL string, selectAll bool) error {
	layerID := ""
	parts := strings.Split(featureServerURL, "/")
	lastPart := parts[len(parts)-1]
	if _, err := strconv.Atoi(lastPart); err == nil {
		layerID = lastPart
		featureServerURL = strings.Join(parts[:len(parts)-1], "/")
	}

	if layerID != "" {
		fmt.Println("  Processing as single Feature Layer URL...")
		layersToProcess[featureServerURL+"/"+layerID] = arcgis.AvailableLayerInfo{
			ID:             layerID,
			Name:           fmt.Sprintf("Layer_%s", layerID),
			ServiceURL:     featureServerURL,
			IsFeatureLayer: true,
		}
		return nil
	} else {
		layers, err := client.FetchServiceLayers(featureServerURL, "FeatureServer")
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

// selectAndAddLayers prompts the user to select layers from a list and adds them to the processing queue.
func selectAndAddLayers(availableLayers []arcgis.AvailableLayerInfo, selectAll bool) error {
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
func processSelectedLayer(client *arcgis.Client, layerInfo arcgis.AvailableLayerInfo, format, outputDir string, overwrite, skipExisting bool, prefix string, excludeSymbols, saveSymbols bool) error {
	metadataURL := fmt.Sprintf("%s/%s?f=json", layerInfo.ServiceURL, layerInfo.ID)
	var layerMetadata arcgis.Layer
	err := client.FetchAndDecode(metadataURL, &layerMetadata)
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

	features, err := client.FetchFeatures(layerInfo.ServiceURL, layerInfo.ID)
	if err != nil {
		if strings.Contains(err.Error(), "no features found") {
			return fmt.Errorf("no features found")
		}
		return fmt.Errorf("failed to fetch features: %v", err)
	}

	// Create symbols directory if needed
	symbolsDir := ""
	if saveSymbols {
		symbolsDir = filepath.Join(outputDir, "symbols", actualLayerName)
		if err := os.MkdirAll(symbolsDir, 0750); err != nil {
			return fmt.Errorf("failed to create symbols directory %s: %v", symbolsDir, err)
		}
	}

	// Add symbol information to features if available in layer metadata and not excluded
	if !excludeSymbols && layerMetadata.DrawingInfo != nil && layerMetadata.DrawingInfo.Renderer != nil {
		renderer := layerMetadata.DrawingInfo.Renderer

		// Determine relative path for symbols if saving
		relativeSymbolsDir := ""
		if saveSymbols {
			// Use only the layer name subdirectory for the relative path in KML
			relativeSymbolsDir = filepath.Join("symbols", actualLayerName)
		}

		// Handle default symbol
		if renderer.DefaultSymbol != nil {
			defaultSymbolCopy := *renderer.DefaultSymbol // Make a copy to modify URL if needed
			if saveSymbols {
				symbolFilenameBase := "default"
				// Save default symbol
				if err := saveSymbol(&defaultSymbolCopy, symbolsDir, symbolFilenameBase); err != nil {
					printWarning(fmt.Sprintf("  Warning: Failed to save default symbol: %v", err))
				} else {
					// Update URL to relative path
					ext := getSymbolFileExtension(&defaultSymbolCopy)
					defaultSymbolCopy.URL = filepath.ToSlash(filepath.Join(relativeSymbolsDir, symbolFilenameBase+ext)) // Use forward slashes for KML
				}
			}
			for i := range features {
				if features[i].Attributes == nil {
					features[i].Attributes = make(map[string]interface{})
				}
				features[i].Attributes["symbol"] = &defaultSymbolCopy // Use the (potentially modified) copy
			}
		}

		// Handle unique value renderer
		if renderer.Type == "uniqueValue" && renderer.Field1 != "" && len(renderer.UniqueValueGroups) > 0 {
			// Create a map for faster symbol lookup based on attribute value
			symbolMap := make(map[string]*arcgis.Symbol)
			for _, group := range renderer.UniqueValueGroups {
				for _, class := range group.Classes {
					if class.Symbol != nil {
						classSymbolCopy := *class.Symbol // Make a copy
						if saveSymbols {
							// Sanitize label for filename
							safLabel := regexp.MustCompile(`[<>:"/\|?*\s]`).ReplaceAllString(class.Label, "_")
							if safLabel == "" {
								safLabel = fmt.Sprintf("class_%d", len(symbolMap)) // Fallback name
							}
							symbolFilenameBase := fmt.Sprintf("class_%s", safLabel)
							// Save class symbol
							if err := saveSymbol(&classSymbolCopy, symbolsDir, symbolFilenameBase); err != nil {
								printWarning(fmt.Sprintf("  Warning: Failed to save class symbol %s: %v", symbolFilenameBase, err))
							} else {
								// Update URL to relative path
								text := getSymbolFileExtension(&classSymbolCopy)
								classSymbolCopy.URL = filepath.ToSlash(filepath.Join(relativeSymbolsDir, symbolFilenameBase+text))
							}
						}
						// Map values to the potentially modified symbol copy
						for _, valueSet := range class.Values {
							if len(valueSet) > 0 {
								symbolMap[valueSet[0]] = &classSymbolCopy
							}
						}
					}
				}
			}

			// Assign symbols to features based on the map
			for i := range features {
				if features[i].Attributes == nil {
					features[i].Attributes = make(map[string]interface{})
				}
				// Check if the feature already has a symbol (e.g., default)
				if _, hasSymbol := features[i].Attributes["symbol"]; hasSymbol {
					continue // Skip if default symbol was already assigned
				}
				// Get the attribute value
				if fieldValue, ok := features[i].Attributes[renderer.Field1]; ok && fieldValue != nil {
					fieldValueStr := fmt.Sprintf("%v", fieldValue)
					// Look up the symbol in the map
					if mappedSymbol, found := symbolMap[fieldValueStr]; found {
						features[i].Attributes["symbol"] = mappedSymbol
					}
				}
			}
		}
	}

	var data string
	var fileExt string
	switch strings.ToLower(format) {
	case "geojson":
		geojsonData, err := convert.ConvertToGeoJSON(convertFeatures(features))
		if err != nil {
			return fmt.Errorf("failed to convert features to GeoJSON objects: %v", err)
		}
		data, err = marshalGeoJSON(geojsonData, actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to marshal GeoJSON: %v", err)
		}
		fileExt = "geojson"
	case "kml":
		geojsonData, err := convert.ConvertToGeoJSON(convertFeatures(features))
		if err != nil {
			return fmt.Errorf("failed to convert features to GeoJSON for KML: %v", err)
		}
		data, err = export.ConvertGeoJSONToKML(geojsonData, actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to convert to KML: %v", err)
		}
		fileExt = "kml"
	case "gpx":
		geojsonData, err := convert.ConvertToGeoJSON(convertFeatures(features))
		if err != nil {
			return fmt.Errorf("failed to convert features to GeoJSON for GPX: %v", err)
		}
		data, err = export.ConvertGeoJSONToGPX(geojsonData, actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to convert to GPX: %v", err)
		}
		fileExt = "gpx"
	case "json":
		jsonDataBytes, err := json.MarshalIndent(convertFeatures(features), "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal features to JSON: %v", err)
		}
		data = string(jsonDataBytes)
		fileExt = "json"
	case "csv":
		data, err = convert.ConvertFeaturesToCSV(convertFeatures(features))
		if err != nil {
			return fmt.Errorf("failed to convert features to CSV: %v", err)
		}
		fileExt = "csv"
	case "txt":
		data, err = convert.ConvertFeaturesToText(convertFeatures(features), actualLayerName)
		if err != nil {
			return fmt.Errorf("failed to convert features to text: %v", err)
		}
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

// marshalGeoJSON marshals a GeoJSON struct into a JSON string.
func marshalGeoJSON(geoJSON *convert.GeoJSON, layerName string) (string, error) {
	data, err := json.MarshalIndent(geoJSON, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// convertToConvertFeature converts a local Feature to a convert.Feature
func convertToConvertFeature(f arcgis.Feature) convert.Feature {
	return convert.Feature{
		Attributes: f.Attributes,
		Geometry:   f.Geometry,
	}
}

// convertFeatures converts a slice of local Features to a slice of convert.Features
func convertFeatures(features []arcgis.Feature) []convert.Feature {
	convertFeatures := make([]convert.Feature, len(features))
	for i, f := range features {
		convertFeatures[i] = convertToConvertFeature(f)
	}
	return convertFeatures
}

// saveSymbol saves a symbol to the specified directory
func saveSymbol(symbol *arcgis.Symbol, dir, name string) error {
	if symbol == nil {
		return nil
	}

	// Save image data if available
	if symbol.ImageData != "" {
		// Decode base64 data
		imageData, err := base64.StdEncoding.DecodeString(symbol.ImageData)
		if err != nil {
			return fmt.Errorf("failed to decode image data: %v", err)
		}

		// Determine file extension from content type
		ext := ".png" // default
		if symbol.ContentType != "" {
			switch symbol.ContentType {
			case "image/jpeg":
				ext = ".jpg"
			case "image/gif":
				ext = ".gif"
			case "image/svg+xml":
				ext = ".svg"
			}
		}

		// Save image file
		imagePath := filepath.Join(dir, name+ext)
		if err := os.WriteFile(imagePath, imageData, 0600); err != nil {
			return fmt.Errorf("failed to write image file: %v", err)
		}
	}

	// Save symbol metadata
	metadata := map[string]interface{}{
		"type":        symbol.Type,
		"url":         symbol.URL,
		"contentType": symbol.ContentType,
		"width":       symbol.Width,
		"height":      symbol.Height,
		"xoffset":     symbol.XOffset,
		"yoffset":     symbol.YOffset,
		"angle":       symbol.Angle,
	}

	metadataPath := filepath.Join(dir, name+".json")
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal symbol metadata: %v", err)
	}

	if err := os.WriteFile(metadataPath, metadataBytes, 0600); err != nil {
		return fmt.Errorf("failed to write symbol metadata: %v", err)
	}

	return nil
}

// Helper function to get file extension based on symbol content type
func getSymbolFileExtension(symbol *arcgis.Symbol) string {
	ext := ".png" // default
	if symbol.ContentType != "" {
		switch symbol.ContentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/gif":
			ext = ".gif"
		case "image/svg+xml":
			ext = ".svg"
		}
	}
	return ext
}
