// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

// Package export provides functions for converting GeoJSON data to various export formats.
package export

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
)

// ConvertGeoJSONToKMZ converts a GeoJSON FeatureCollection to a KMZ (compressed KML) byte array.
// The function handles:
//   - Point, LineString, and Polygon geometries
//   - Feature properties and attributes
//   - Symbol styling and icons with embedded images
//   - Creates a compressed archive with KML and image files
//
// Parameters:
//   - geoJSON: Pointer to a GeoJSON FeatureCollection
//   - layerName: Name of the layer to be used in the KML document
//
// Returns:
//   - []byte: KMZ file as a byte array
//   - error: Any error that occurred during conversion
func ConvertGeoJSONToKMZ(geoJSON *convert.GeoJSON, layerName string) ([]byte, error) {
	var styles strings.Builder
	var placemarks strings.Builder
	styleMap := make(map[string]string)     // Map to track unique styles
	imageFiles := make(map[string][]byte)   // Map to store image files for KMZ
	imageCounter := 0

	// First pass: collect all unique styles and extract images
	for _, feature := range geoJSON.Features {
		// Try to get symbol from feature's Symbol field first
		if feature.Symbol != nil {
			styleID := generateStyleID(feature.Symbol)
			if _, exists := styleMap[styleID]; !exists {
				// Handle embedded image if present
				if feature.Symbol.ImageData != "" {
					imageCounter++
					imageName := fmt.Sprintf("images/symbol_%d%s", imageCounter, getImageExtension(feature.Symbol.ContentType))
					
					// Decode base64 image data
					imageData, err := base64.StdEncoding.DecodeString(feature.Symbol.ImageData)
					if err != nil {
						return nil, fmt.Errorf("failed to decode base64 image data: %v", err)
					}
					
					// Store image data for KMZ archive
					imageFiles[imageName] = imageData
					
					// Update symbol URL to reference the file in the KMZ
					feature.Symbol.URL = imageName
				}
				styleMap[styleID] = generateKMLStyleForKMZ(feature.Symbol)
			}
		} else if symbolData, ok := feature.Properties["symbol"]; ok {
			if symbolMap, ok := symbolData.(map[string]interface{}); ok {
				symbol := &convert.Symbol{
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
				feature.Symbol = symbol
				styleID := generateStyleID(symbol)
				if _, exists := styleMap[styleID]; !exists {
					if symbol.ImageData != "" {
						imageCounter++
						imageName := fmt.Sprintf("images/symbol_%d%s", imageCounter, getImageExtension(symbol.ContentType))
						
						// Decode base64 image data
						imageData, err := base64.StdEncoding.DecodeString(symbol.ImageData)
						if err != nil {
							return nil, fmt.Errorf("failed to decode base64 image data: %v", err)
						}
						
						// Store image data for KMZ archive
						imageFiles[imageName] = imageData
						
						// Update symbol URL to reference the file in the KMZ
						symbol.URL = imageName
					}
					styleMap[styleID] = generateKMLStyleForKMZ(symbol)
				}
			}
		} else if rendererData, ok := feature.Properties["renderer"]; ok {
			if rendererMap, ok := rendererData.(map[string]interface{}); ok {
				if rendererType, ok := rendererMap["type"].(string); ok && rendererType == "uniqueValue" {
					if field1, ok := rendererMap["field1"].(string); ok {
						if value, ok := feature.Properties[field1]; ok {
							if groups, ok := rendererMap["uniqueValueGroups"].([]interface{}); ok {
								for _, group := range groups {
									if groupMap, ok := group.(map[string]interface{}); ok {
										if classes, ok := groupMap["classes"].([]interface{}); ok {
											for _, class := range classes {
												if classMap, ok := class.(map[string]interface{}); ok {
													if values, ok := classMap["values"].([]interface{}); ok {
														for _, val := range values {
															if valArray, ok := val.([]interface{}); ok && len(valArray) > 0 {
																if valArray[0] == value {
																	if symbolMap, ok := classMap["symbol"].(map[string]interface{}); ok {
																		symbol := &convert.Symbol{
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
																		feature.Symbol = symbol
																		styleID := generateStyleID(symbol)
																		if _, exists := styleMap[styleID]; !exists {
																			if symbol.ImageData != "" {
																				imageCounter++
																				imageName := fmt.Sprintf("images/symbol_%d%s", imageCounter, getImageExtension(symbol.ContentType))
																				
																				// Decode base64 image data
																				imageData, err := base64.StdEncoding.DecodeString(symbol.ImageData)
																				if err != nil {
																					return nil, fmt.Errorf("failed to decode base64 image data: %v", err)
																				}
																				
																				// Store image data for KMZ archive
																				imageFiles[imageName] = imageData
																				
																				// Update symbol URL to reference the file in the KMZ
																				symbol.URL = imageName
																			}
																			styleMap[styleID] = generateKMLStyleForKMZ(symbol)
																		}
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Write all styles
	for styleID, styleXML := range styleMap {
		styles.WriteString(fmt.Sprintf(`
        <Style id="%s">
            %s
        </Style>`, styleID, styleXML))
	}

	// Second pass: write placemarks with style references
	for _, feature := range geoJSON.Features {
		if feature.Geometry == nil {
			continue
		}

		name := getFeatureName(feature)
		description := formatProperties(feature.Properties, "<br>")

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
			fmt.Printf("  Warning: Unsupported geometry type for KMZ conversion: %s\n", geometryType)
		}

		if geometryString != "" {
			styleRef := ""
			if feature.Symbol != nil {
				styleID := generateStyleID(feature.Symbol)
				styleRef = fmt.Sprintf(`<styleUrl>#%s</styleUrl>`, styleID)
			}

			placemarks.WriteString(fmt.Sprintf(`
        <Placemark>
            <name>%s</name>
            <description><![CDATA[%s]]></description>
            %s
            %s
        </Placemark>`, escapeXML(name), description, styleRef, geometryString))
		}
	}

	// Generate the complete KML content
	kmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
    <Document>
        <name>%s</name>
        <description>Exported from ArcGIS Utils</description>
        %s
        %s
    </Document>
</kml>`, escapeXML(layerName), styles.String(), placemarks.String())

	// Create KMZ archive
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add KML file to archive
	kmlFile, err := zipWriter.Create("doc.kml")
	if err != nil {
		return nil, fmt.Errorf("failed to create KML file in KMZ archive: %v", err)
	}
	_, err = kmlFile.Write([]byte(kmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to write KML content to KMZ archive: %v", err)
	}

	// Add image files to archive
	for imagePath, imageData := range imageFiles {
		imageFile, err := zipWriter.Create(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create image file %s in KMZ archive: %v", imagePath, err)
		}
		_, err = imageFile.Write(imageData)
		if err != nil {
			return nil, fmt.Errorf("failed to write image data for %s to KMZ archive: %v", imagePath, err)
		}
	}

	// Close the zip writer
	err = zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close KMZ archive: %v", err)
	}

	return buf.Bytes(), nil
}

// generateKMLStyleForKMZ generates KML style XML for KMZ format (with file references instead of data URLs)
func generateKMLStyleForKMZ(symbol *convert.Symbol) string {
	if symbol == nil {
		return generateDefaultStyle()
	}

	switch symbol.Type {
	case "esriPMS": // Picture Marker Symbol
		return generatePictureMarkerStyleForKMZ(symbol)
	case "esriSMS": // Simple Marker Symbol
		return generateSimpleMarkerStyle(symbol)
	case "esriSLS": // Simple Line Symbol
		return generateSimpleLineStyle(symbol)
	case "esriSFS": // Simple Fill Symbol
		return generateSimpleFillStyle(symbol)
	default:
		return generateDefaultStyle()
	}
}

// generatePictureMarkerStyleForKMZ generates a picture marker style for KMZ format
func generatePictureMarkerStyleForKMZ(symbol *convert.Symbol) string {
	iconURL := symbol.URL
	if iconURL == "" {
		iconURL = "http://maps.google.com/mapfiles/kml/pushpin/ylw-pushpin.png"
	}

	scale := 1.0
	if symbol.Width > 0 {
		scale = float64(symbol.Width) / 32.0 // Normalize to reasonable size
	}

	return fmt.Sprintf(`
            <IconStyle>
                <Icon>
                    <href>%s</href>
                </Icon>
                <scale>%.2f</scale>
                <hotSpot x="0.5" y="0" xunits="fraction" yunits="fraction"/>
            </IconStyle>`, iconURL, scale)
}

// generateSimpleMarkerStyle generates a simple marker style
func generateSimpleMarkerStyle(symbol *convert.Symbol) string {
	color := "ff0000ff" // Default red
	scale := 1.0
	if symbol.Width > 0 {
		scale = float64(symbol.Width) / 16.0
	}

	return fmt.Sprintf(`
            <IconStyle>
                <color>%s</color>
                <scale>%.2f</scale>
                <Icon>
                    <href>http://maps.google.com/mapfiles/kml/pushpin/ylw-pushpin.png</href>
                </Icon>
            </IconStyle>`, color, scale)
}

// getImageExtension returns the file extension based on content type
func getImageExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/png":
		return ".png"
	default:
		return ".png" // Default to PNG
	}
} 