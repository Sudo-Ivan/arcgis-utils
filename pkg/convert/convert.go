// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

// Package convert provides functions for converting between different geospatial data formats.
package convert

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/arcgis"
)

// ToGeoJSON converts a slice of Feature structs to a GeoJSON FeatureCollection.
// It handles:
//   - Point geometries (x,y coordinates)
//   - LineString geometries (paths)
//   - Polygon geometries (rings)
//   - Feature attributes and properties
//   - Symbol information
//
// Parameters:
//   - features: Slice of Feature structs to convert
//
// Returns:
//   - *GeoJSON: Pointer to the converted GeoJSON FeatureCollection
//   - error: Any error that occurred during conversion
func ToGeoJSON(features []Feature) (*GeoJSON, error) {
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
				// Attempt to cast the attribute value to *arcgis.Symbol
				if arcSymbol, castOk := symbolData.(*arcgis.Symbol); castOk && arcSymbol != nil {
					// Create a convert.Symbol and copy fields
					symbol := &Symbol{
						Type:        arcSymbol.Type,
						URL:         arcSymbol.URL,
						ImageData:   arcSymbol.ImageData,
						ContentType: arcSymbol.ContentType,
						Width:       arcSymbol.Width,
						Height:      arcSymbol.Height,
						XOffset:     arcSymbol.XOffset,
						YOffset:     arcSymbol.YOffset,
						Angle:       arcSymbol.Angle,
					}
					geoJSONFeature.Symbol = symbol
				}
			}

			geoJSON.Features = append(geoJSON.Features, geoJSONFeature)
		}
	}

	return &geoJSON, nil
}

// FeaturesToCSV converts a slice of Feature structs to a CSV string.
// The CSV includes:
//   - All unique attribute fields as columns
//   - WKT geometry representation in the last column
//   - Sorted column headers for consistency
//
// Parameters:
//   - features: Slice of Feature structs to convert
//
// Returns:
//   - string: CSV formatted string
//   - error: Any error that occurred during conversion
func FeaturesToCSV(features []Feature) (string, error) {
	if len(features) == 0 {
		return "", nil
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
			return "", fmt.Errorf("failed to write row to CSV: %v", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("error during CSV writing: %v", err)
	}

	return buf.String(), nil
}

// FeaturesToText converts a slice of Feature structs to a formatted text string.
// The output includes:
//   - Layer name and feature count
//   - Feature attributes in sorted order
//   - WKT geometry representation
//
// Parameters:
//   - features: Slice of Feature structs to convert
//   - layerName: Name of the layer for the header
//
// Returns:
//   - string: Formatted text output
//   - error: Any error that occurred during conversion
func FeaturesToText(features []Feature, layerName string) (string, error) {
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

// Helper functions

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

// geometryToWKT converts a geometry interface to a WKT string.
// Supports:
//   - Point geometries (x,y coordinates)
//   - LineString geometries (paths)
//   - Polygon geometries (rings)
//
// Returns empty string if geometry is nil or invalid.
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
				if len(points) == 0 {
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
					if len(points) > 0 {
						if points[0] != points[len(points)-1] {
							points = append(points, points[0])
						}
						polygonRings = append(polygonRings, fmt.Sprintf("(%s)", strings.Join(points, ", ")))
					}
				}
			}
			if len(polygonRings) == 0 {
				return ""
			}
			return fmt.Sprintf("POLYGON (%s)", strings.Join(polygonRings, ", "))
		}
	}

	return ""
}
