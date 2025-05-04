// Copyright (c) 2024 Sudo-Ivan
// Licensed under the MIT License

// Package export provides functions for converting GeoJSON data to various export formats.
package export

import (
	"fmt"
	"strings"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
)

// ConvertGeoJSONToGPX converts a GeoJSON FeatureCollection to a GPX string.
// The function handles:
//   - Point geometries as waypoints
//   - LineString geometries as tracks
//   - Polygon geometries as track boundaries
//
// Parameters:
//   - geoJSON: Pointer to a GeoJSON FeatureCollection
//   - layerName: Name of the layer to be used in the GPX metadata
//
// Returns:
//   - string: GPX document as a string
//   - error: Any error that occurred during conversion
func ConvertGeoJSONToGPX(geoJSON *convert.GeoJSON, layerName string) (string, error) {
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
			fmt.Printf("  Warning: Unsupported geometry type for GPX conversion: %s\n", geometryType)
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
