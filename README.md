# arcgis-utils-go

[![Socket Badge](https://socket.dev/api/badge/go/package/github.com/Sudo-Ivan/arcgis-utils?version=v0.6.0)](https://socket.dev/go/package/github.com/Sudo-Ivan/arcgis-utils?version=v0.6.0)

A command-line tool written in Go to download data from various **public** ArcGIS sources (Feature Layers, Feature Servers, Map Servers, ArcGIS Online Items including Web Maps) and convert it to common geospatial formats.

## Features

| Feature                                      | Description                                                                                                                               |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Supported ArcGIS REST API Endpoints          | Single Feature Layer URLs (`.../FeatureServer/0`), Feature Server URLs (`.../FeatureServer`), Map Server URLs (`.../MapServer`), ArcGIS Online Item URLs (`arcgis.com/home/item.html?id=...`) (supports Feature Service, Map Service, and Web Map items) |
| Layer Discovery                              | Handles discovery of layers within services and web maps.                                                                                 |
| Interactive Layer Selection                  | Provides interactive selection of layers to download.                                                                                    |
| Export Formats                               | GeoJSON (`.geojson`), KML (`.kml`), GPX (`.gpx`), CSV (`.csv`), JSON (`.json`), Text (`.txt`)                                             |
| Configuration Options                        | Configurable options for file handling, output naming, and request timeouts.                                                              |

### To-Do

- [ ] Updating data only if the last edit date has changed
- [ ] Versioning
- [ ] ArcGIS API Support
- [ ] Symbology and Image Support from ArcGIS Layers
- [ ] QGIS Support

## Installation

Download the binary for your platform from the [releases page](https://github.com/Sudo-Ivan/arcgis-utils/releases). 

### Usage

Linux/MacOS Example:

```bash
arcgis-utils-go-linux-amd64 -url <ARCGIS_URL> [OPTIONS]
```

Windows Example:

```powershell
.\arcgis-utils-go-windows-amd64.exe -url <ARCGIS_URL> [OPTIONS]
```

With Go:

```bash
go install github.com/Sudo-Ivan/arcgis-utils@latest
```

```bash
arcgis-utils -url <ARCGIS_URL> [OPTIONS]
```

### Docker

```bash
mkdir -p results
chown -R 1000:1000 results
docker run --rm -v ./results:/results -u 1000:1000 ghcr.io/sudo-ivan/arcgis-utils:latest -url <ARCGIS_URL> [OPTIONS]
```

## Building from Source

Ensure you have Go installed (version 1.24 or later recommended).

```bash
git clone https://github.com/Sudo-Ivan/arcgis-utils-go.git
cd arcgis-utils-go
go build -o arcgis-utils main.go
```

This will create an executable named `arcgis-utils` (or `arcgis-utils.exe` on Windows) in the current directory.

## Usage

```bash
arcgis-utils -url <ARCGIS_URL> [OPTIONS]
```

Docker Build:

```bash
docker build -t arcgis-utils .
docker run --rm -v ./results:/results -u 1000:1000 arcgis-utils -url <ARCGIS_URL> [OPTIONS]
```

**Required:**

*   `-url <ARCGIS_URL>`: The URL of the ArcGIS resource to process. This can be a Feature Layer, Feature Server, Map Server, or ArcGIS Online Item page URL.

**Options:**

*   `-format <format>`: Output format. Options are `geojson` (default), `kml`, `gpx`, `csv`, `json`, `text`.
*   `-output <directory>`: Directory to save the output files. Defaults to the current working directory.
*   `-select-all`: Automatically select and process all discoverable Feature Layers without prompting.
*   `-no-color`: Disable colored terminal output.
*   `-overwrite`: Allow overwriting existing output files in the target directory. Without this flag, the tool will error if an output file already exists.
*   `-skip-existing`: If an output file for a layer already exists, skip processing that layer instead of erroring or overwriting.
*   `-prefix <prefix>`: Add a specified prefix to all output filenames.
*   `-timeout <seconds>`: Set the timeout in seconds for HTTP requests (default: 30).
*   `-exclude-symbols`: Exclude symbol information (images, styling) from the output. Useful for reducing file size or when symbols aren't needed.

## Examples

**1. Download a specific Feature Layer as GeoJSON:**

```bash
arcgis-utils -url https://services.arcgis.com/P3ePLMYs2RVChkJx/arcgis/rest/services/World_Time_Zones/FeatureServer/0
```

**2. Download all layers from a Feature Server as KML to a specific directory, overwriting existing files:**

```bash
arcgis-utils -url https://sampleserver6.arcgisonline.com/arcgis/rest/services/EmergencyFacilities/FeatureServer -format kml -output ./kml_output -select-all -overwrite
```

**3. Download selected layers from a Map Server as GPX with a filename prefix:**

```bash
arcgis-utils -url https://sampleserver6.arcgisonline.com/arcgis/rest/services/USA/MapServer -format gpx -prefix USA_Data_
# (Follow interactive prompts to select layers)
```

**4. Process an ArcGIS Online Web Map item, skipping layers if their output files already exist:**

```bash
arcgis-utils -url https://www.arcgis.com/home/item.html?id=a12b34c56d78e90f1234567890abcdef -skip-existing
```

**5. Download a Feature Layer as GeoJSON without symbol information:**

```bash
arcgis-utils -url https://services.arcgis.com/P3ePLMYs2RVChkJx/arcgis/rest/services/World_Time_Zones/FeatureServer/0 -exclude-symbols
```

**6. Download a Feature Layer as CSV with a custom timeout:**

```bash
arcgis-utils -url https://services.arcgis.com/P3ePLMYs2RVChkJx/arcgis/rest/services/World_Time_Zones/FeatureServer/0 -format csv -timeout 60
```
## License

MIT License
