package utils

import (
	"testing"
)

func TestGeometryToWKT(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"Nil Geometry", nil, ""},
		{"Invalid Type", "not a map", ""},
		{"Point Geometry", map[string]interface{}{"x": -122.5, "y": 37.8}, "POINT (-122.5000000000 37.8000000000)"},
		{"Point Geometry Integer Coords", map[string]interface{}{"x": -122.0, "y": 37.0}, "POINT (-122.0000000000 37.0000000000)"},
		{"LineString Geometry", map[string]interface{}{"paths": []interface{}{ // Array of paths
			[]interface{}{ // First path: array of points
				[]interface{}{-122.0, 37.0}, // Point
				[]interface{}{-122.1, 37.1}, // Point
			},
		}}, "LINESTRING (-122.0000000000 37.0000000000, -122.1000000000 37.1000000000)"},
		{"Polygon Geometry Single Ring", map[string]interface{}{"rings": []interface{}{ // Array of rings
			[]interface{}{ // First ring: array of points
				[]interface{}{-122.0, 37.0},
				[]interface{}{-122.1, 37.0},
				[]interface{}{-122.1, 37.1},
				[]interface{}{-122.0, 37.1},
				[]interface{}{-122.0, 37.0},
			},
		}}, "POLYGON ((-122.0000000000 37.0000000000, -122.1000000000 37.0000000000, -122.1000000000 37.1000000000, -122.0000000000 37.1000000000, -122.0000000000 37.0000000000))"},
		{"Polygon Geometry Unclosed Ring", map[string]interface{}{"rings": []interface{}{ // Array of rings
			[]interface{}{ // First ring: array of points
				[]interface{}{-1.0, 1.0},
				[]interface{}{-2.0, 1.0},
				[]interface{}{-2.0, 2.0},
			},
		}}, "POLYGON ((-1.0000000000 1.0000000000, -2.0000000000 1.0000000000, -2.0000000000 2.0000000000, -1.0000000000 1.0000000000))"}, // Expect auto-close
		{"Polygon With Hole", map[string]interface{}{"rings": []interface{}{ // Array of rings
			[]interface{}{ // Outer ring
				[]interface{}{0.0, 0.0}, []interface{}{10.0, 0.0}, []interface{}{10.0, 10.0}, []interface{}{0.0, 10.0}, []interface{}{0.0, 0.0},
			},
			[]interface{}{ // Inner ring (hole)
				[]interface{}{1.0, 1.0}, []interface{}{1.0, 2.0}, []interface{}{2.0, 2.0}, []interface{}{2.0, 1.0}, []interface{}{1.0, 1.0},
			},
		}}, "POLYGON ((0.0000000000 0.0000000000, 10.0000000000 0.0000000000, 10.0000000000 10.0000000000, 0.0000000000 10.0000000000, 0.0000000000 0.0000000000), (1.0000000000 1.0000000000, 1.0000000000 2.0000000000, 2.0000000000 2.0000000000, 2.0000000000 1.0000000000, 1.0000000000 1.0000000000))"},
		{"Missing Coordinates Point", map[string]interface{}{"x": -122.5}, ""},
		{"Missing Paths", map[string]interface{}{"paths": []interface{}{}}, ""},
		{"Empty Path", map[string]interface{}{"paths": []interface{}{[]interface{}{}}}, ""},
		{"Missing Rings", map[string]interface{}{"rings": []interface{}{}}, ""},
		{"Empty Ring", map[string]interface{}{"rings": []interface{}{[]interface{}{}}}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := GeometryToWKT(tt.input)
			if actual != tt.expected {
				t.Errorf("GeometryToWKT(): expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
