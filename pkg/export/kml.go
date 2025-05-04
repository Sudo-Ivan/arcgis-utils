// Copyright (c) 2024 Sudo-Ivan
// Licensed under the MIT License

// Package export provides functions for converting GeoJSON data to various export formats.
package export

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
)

// ConvertGeoJSONToKML converts a GeoJSON FeatureCollection to a KML string.
// The function handles:
//   - Point, LineString, and Polygon geometries
//   - Feature properties and attributes
//   - Symbol styling and icons
//   - Embedded images and base64 data
//
// Parameters:
//   - geoJSON: Pointer to a GeoJSON FeatureCollection
//   - layerName: Name of the layer to be used in the KML document
//
// Returns:
//   - string: KML document as a string
//   - error: Any error that occurred during conversion
func ConvertGeoJSONToKML(geoJSON *convert.GeoJSON, layerName string) (string, error) {
	var styles strings.Builder
	var placemarks strings.Builder
	styleMap := make(map[string]string) // Map to track unique styles
	imageMap := make(map[string]string) // Map to track embedded images

	// First pass: collect all unique styles and images
	for _, feature := range geoJSON.Features {
		// Try to get symbol from feature's Symbol field first
		if feature.Symbol != nil {
			styleID := generateStyleID(feature.Symbol)
			if _, exists := styleMap[styleID]; !exists {
				// Handle embedded image if present
				if feature.Symbol.ImageData != "" {
					feature.Symbol.URL = fmt.Sprintf("data:%s;base64,%s",
						feature.Symbol.ContentType,
						feature.Symbol.ImageData)
				}
				styleMap[styleID] = generateKMLStyle(feature.Symbol)
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
						symbol.URL = fmt.Sprintf("data:%s;base64,%s",
							symbol.ContentType,
							symbol.ImageData)
					}
					styleMap[styleID] = generateKMLStyle(symbol)
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
																				symbol.URL = fmt.Sprintf("data:%s;base64,%s",
																					symbol.ContentType,
																					symbol.ImageData)
																			}
																			styleMap[styleID] = generateKMLStyle(symbol)
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

	// Write all embedded images
	for imageID, imageData := range imageMap {
		styles.WriteString(fmt.Sprintf(`
        <GroundOverlay id="%s">
            <Icon>
                <href>data:%s;base64,%s</href>
            </Icon>
        </GroundOverlay>`, imageID, getContentType(imageData), imageData))
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
			fmt.Printf("  Warning: Unsupported geometry type for KML conversion: %s\n", geometryType)
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

	kml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
    <Document>
        <name>%s</name>%s%s
    </Document>
</kml>`, escapeXML(layerName), styles.String(), placemarks.String())

	return kml, nil
}

// getContentType determines the content type from base64 data.
// It examines the first few bytes of the decoded data to identify common image formats:
//   - JPEG: Starts with 0xFF 0xD8
//   - PNG: Starts with 0x89 0x50
//   - GIF: Starts with 0x47 0x49
//   - SVG: Starts with 0x3C 0x3F
//
// Returns "image/png" as default if format cannot be determined.
func getContentType(base64Data string) string {
	if len(base64Data) < 20 {
		return "image/png" // Default to PNG if data is too short
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Data[:20])
	if err != nil {
		return "image/png" // Default to PNG on error
	}

	if len(decoded) >= 2 {
		switch {
		case decoded[0] == 0xFF && decoded[1] == 0xD8:
			return "image/jpeg"
		case decoded[0] == 0x89 && decoded[1] == 0x50:
			return "image/png"
		case decoded[0] == 0x47 && decoded[1] == 0x49:
			return "image/gif"
		case decoded[0] == 0x3C && decoded[1] == 0x3F:
			return "image/svg+xml"
		}
	}

	return "image/png" // Default to PNG if no match
}

// generateStyleID creates a unique style ID for a symbol.
// The ID is based on the symbol's type, dimensions, offset, and angle.
func generateStyleID(symbol *convert.Symbol) string {
	return fmt.Sprintf("style_%s_%d_%d_%d_%d_%.2f",
		symbol.Type,
		symbol.Width,
		symbol.Height,
		symbol.XOffset,
		symbol.YOffset,
		symbol.Angle)
}

// generateKMLStyle creates a KML style based on the symbol type.
// Supports the following symbol types:
//   - esriPMS: Picture marker symbol
//   - esriSMS: Simple marker symbol
//   - esriSLS: Simple line symbol
//   - esriSFS: Simple fill symbol
func generateKMLStyle(symbol *convert.Symbol) string {
	switch symbol.Type {
	case "esriPMS", "esriSMS":
		return generatePictureMarkerStyle(symbol)
	case "esriSLS":
		return generateSimpleLineStyle(symbol)
	case "esriSFS":
		return generateSimpleFillStyle(symbol)
	default:
		return generateDefaultStyle()
	}
}

// generatePictureMarkerStyle creates a KML style for picture markers.
// Handles icon scaling, rotation, and hotspot positioning.
func generatePictureMarkerStyle(symbol *convert.Symbol) string {
	scale := 1.0
	if symbol.Width > 0 && symbol.Height > 0 {
		scale = float64(symbol.Width) / 32.0 // Normalize to a reasonable size
	}

	return fmt.Sprintf(`
            <IconStyle>
                <scale>%.2f</scale>
                <heading>%.2f</heading>
                <Icon>
                    <href>%s</href>
                </Icon>
                <hotSpot x="%.2f" y="%.2f" xunits="fraction" yunits="fraction"/>
            </IconStyle>
            <LabelStyle>
                <scale>1.0</scale>
            </LabelStyle>`,
		scale,
		symbol.Angle,
		symbol.URL,
		float64(symbol.XOffset)/float64(symbol.Width),
		float64(symbol.YOffset)/float64(symbol.Height))
}

// generateSimpleLineStyle creates a KML style for simple lines.
// Sets line width and color.
func generateSimpleLineStyle(symbol *convert.Symbol) string {
	width := 2
	if symbol.Width > 0 {
		width = symbol.Width
	}

	return fmt.Sprintf(`
            <LineStyle>
                <width>%d</width>
                <color>ff0000ff</color>
            </LineStyle>
            <LabelStyle>
                <scale>1.0</scale>
            </LabelStyle>`, width)
}

// generateSimpleFillStyle creates a KML style for simple fills.
// Sets polygon fill color, outline, and label style.
func generateSimpleFillStyle(symbol *convert.Symbol) string {
	return `
            <PolyStyle>
                <color>7f0000ff</color>
                <fill>1</fill>
                <outline>1</outline>
            </PolyStyle>
            <LineStyle>
                <width>2</width>
                <color>ff0000ff</color>
            </LineStyle>
            <LabelStyle>
                <scale>1.0</scale>
            </LabelStyle>`
}

// generateDefaultStyle creates a default KML style.
// Uses a standard placemark circle icon.
func generateDefaultStyle() string {
	return `
            <IconStyle>
                <scale>1.0</scale>
                <Icon>
                    <href>http://maps.google.com/mapfiles/kml/shapes/placemark_circle.png</href>
                </Icon>
            </IconStyle>
            <LabelStyle>
                <scale>1.0</scale>
            </LabelStyle>`
}

// getFeatureName extracts a suitable name from a GeoJSON feature's properties.
// Checks common property names in order: name, Name, NAME, title, Title, TITLE, OBJECTID, FID.
// Returns "Feature" if no suitable name is found.
func getFeatureName(feature convert.GeoJSONFeature) string {
	props := feature.Properties
	for _, key := range []string{"name", "Name", "NAME", "title", "Title", "TITLE", "OBJECTID", "FID"} {
		if val, ok := props[key]; ok && val != nil {
			return fmt.Sprintf("%v", val)
		}
	}
	return "Feature"
}

// formatProperties formats a map of properties into a string.
// Excludes geometry and symbol properties.
// Uses HTML formatting for better readability in KML.
func formatProperties(props map[string]interface{}, separator ...string) string {
	sep := "<br>"
	if len(separator) > 0 {
		sep = separator[0]
	}
	var parts []string
	for k, v := range props {
		if k == "geometry" || k == "symbol" {
			continue
		}
		parts = append(parts, fmt.Sprintf("<strong>%s</strong>: %v", escapeXML(k), escapeXML(fmt.Sprintf("%v", v))))
	}
	return strings.Join(parts, sep)
}

// escapeXML escapes XML special characters in a string.
// Handles: &, <, >, ", ', and / characters.
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

// getString extracts a string value from a map.
// Returns empty string if key doesn't exist or value is not a string.
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getInt extracts an integer value from a map.
// Returns 0 if key doesn't exist or value is not a number.
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return int(num)
		}
	}
	return 0
}

// getFloat extracts a float64 value from a map.
// Returns 0 if key doesn't exist or value is not a number.
func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0
}
