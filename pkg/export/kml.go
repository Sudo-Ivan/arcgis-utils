package export

import (
	"fmt"
	"strings"

	"github.com/Sudo-Ivan/arcgis-utils/pkg/convert"
)

// ConvertGeoJSONToKML converts a GeoJSON FeatureCollection to a KML string.
func ConvertGeoJSONToKML(geoJSON *convert.GeoJSON, layerName string) (string, error) {
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
			fmt.Printf("  Warning: Unsupported geometry type for KML conversion: %s\n", geometryType)
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

// getFeatureName extracts a suitable name from a GeoJSON feature's properties.
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
