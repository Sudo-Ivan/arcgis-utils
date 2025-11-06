# arcgis-utils

[![Socket Badge](https://socket.dev/api/badge/go/package/github.com/Sudo-Ivan/arcgis-utils?version=v0.6.0)](https://socket.dev/go/package/github.com/Sudo-Ivan/arcgis-utils?version=v0.6.0)

Command-line tool and Go library for downloading data from public ArcGIS sources and converting to common geospatial formats.

## Installation

```bash
go install github.com/Sudo-Ivan/arcgis-utils/cmd/arcgis-utils@latest
```

Or download binaries from [releases](https://github.com/Sudo-Ivan/arcgis-utils/releases).

## Usage

```bash
arcgis-utils -url <ARCGIS_URL> [OPTIONS]
```

**Required:**
- `-url`: ArcGIS resource URL (Feature Layer, Feature Server, Map Server, or ArcGIS Online Item)

**Options:**
- `-format`: Output format (`geojson`, `kml`, `gpx`, `csv`, `json`, `txt`) - default: `geojson`
- `-output`: Output directory - default: current directory
- `-select-all`: Process all layers without prompting
- `-overwrite`: Overwrite existing files
- `-skip-existing`: Skip processing if output file exists
- `-prefix`: Prefix for output filenames
- `-timeout`: HTTP request timeout in seconds - default: 30
- `-exclude-symbols`: Exclude symbol information from output
- `-save-symbols`: Save symbology/images to separate folder
- `-no-color`: Disable colored output

## Examples

Download a Feature Layer as GeoJSON:
```bash
arcgis-utils -url https://services.arcgis.com/P3ePLMYs2RVChkJx/arcgis/rest/services/World_Time_Zones/FeatureServer/0
```

Download all layers from a Feature Server as KML:
```bash
arcgis-utils -url https://sampleserver6.arcgisonline.com/arcgis/rest/services/EmergencyFacilities/FeatureServer -format kml -select-all
```

Process an ArcGIS Online Web Map:
```bash
arcgis-utils -url https://www.arcgis.com/home/item.html?id=a12b34c56d78e90f1234567890abcdef -skip-existing
```

## Docker

```bash
docker run --rm -v $(pwd)/results:/results ghcr.io/sudo-ivan/arcgis-utils:latest -url <ARCGIS_URL> -output /results
```

## Building

```bash
make
```

Build with debug symbols:
```bash
make debug
```

Cross-compile for specific platform:
```bash
make linux-amd64
make windows-amd64
make darwin-arm64
```

See `Makefile` for all supported platforms.

## Library Usage

```go
import "github.com/Sudo-Ivan/arcgis-utils/pkg/arcgis"

client := arcgis.NewClient(30 * time.Second)
layers, err := client.FetchServiceLayers(url, "FeatureServer")
```

## License

MIT License
