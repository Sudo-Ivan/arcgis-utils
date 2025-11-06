// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

// Package utils provides utility functions for handling geometry conversions and transformations.
package utils

import (
	"fmt"
	"strings"
)

// GeometryToWKT converts a geometry interface to a Well-Known Text (WKT) string.
// It supports the following geometry types:
//   - Point: Converts x,y coordinates to "POINT (x y)"
//   - LineString: Converts array of points to "LINESTRING (x1 y1, x2 y2, ...)"
//   - Polygon: Converts array of rings to "POLYGON ((x1 y1, x2 y2, ...), (x1 y1, x2 y2, ...))"
//
// The function handles ArcGIS geometry format and ensures rings are closed for polygons.
// Returns an empty string if the geometry is nil or invalid.
func GeometryToWKT(geometry interface{}) string {
	if geometry == nil {
		return EmptyString
	}

	geomMap, ok := geometry.(map[string]interface{})
	if !ok {
		return EmptyString
	}

	if xVal, xOk := geomMap[KeyX]; xOk {
		if yVal, yOk := geomMap[KeyY]; yOk {
			x, xFloatOk := xVal.(float64)
			y, yFloatOk := yVal.(float64)
			if xFloatOk && yFloatOk {
				return fmt.Sprintf("POINT (%.10f %.10f)", x, y)
			}
		}
	} else if paths, pOk := geomMap[KeyPaths]; pOk {
		pathArray, pathArrayOk := paths.([]interface{})
		if pathArrayOk && len(pathArray) > 0 {
			firstPath, firstPathOk := pathArray[0].([]interface{})
			if firstPathOk {
				var points []string
				for _, p := range firstPath {
					point, pointOk := p.([]interface{})
					if pointOk && len(point) >= MinPointCoords {
						x, xOk := point[IndexX].(float64)
						y, yOk := point[IndexY].(float64)
						if xOk && yOk {
							points = append(points, fmt.Sprintf("%.10f %.10f", x, y))
						}
					}
				}
				if len(points) == 0 {
					return EmptyString
				}
				return fmt.Sprintf("LINESTRING (%s)", strings.Join(points, CommaSpace))
			}
		}
	} else if rings, rOk := geomMap[KeyRings]; rOk {
		ringArray, ringArrayOk := rings.([]interface{})
		if ringArrayOk && len(ringArray) > 0 {
			var polygonRings []string
			for _, r := range ringArray {
				ringCoords, ringCoordsOk := r.([]interface{})
				if ringCoordsOk {
					var points []string
					for _, p := range ringCoords {
						point, pointOk := p.([]interface{})
						if pointOk && len(point) >= MinPointCoords {
							x, xOk := point[IndexX].(float64)
							y, yOk := point[IndexY].(float64)
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
						polygonRings = append(polygonRings, fmt.Sprintf("(%s)", strings.Join(points, CommaSpace)))
					}
				}
			}
			if len(polygonRings) == 0 {
				return EmptyString
			}
			return fmt.Sprintf("POLYGON (%s)", strings.Join(polygonRings, CommaSpace))
		}
	}

	return EmptyString
}
