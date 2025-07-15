
Create `save-layers.yml` workflow

```yaml
name: Save ArcGIS Layers to Repo

on:
  workflow_dispatch: # Allows manual triggering
  # schedule:
  #   - cron: '0 0 * * *' # Daily at midnight UTC

permissions:
  contents: write

jobs:
  save-layers:
    runs-on: ubuntu-latest

    steps:
    - name: Checkout repository
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.24'

    - name: Install arcgis-utils
      run: go install github.com/Sudo-Ivan/arcgis-utils/cmd/arcgis-utils@latest

    - name: Run arcgis-utils and save layers
      env:
        OUTPUT_DIR: "arcgis_data"
      run: |
        mkdir -p ${{ env.OUTPUT_DIR }}
        arcgis-utils -layers-csv layers.csv -select-all -versioned-output -output "${{ env.OUTPUT_DIR }}" -format "geojson"

    - name: Commit and push changes
      run: |
        git config user.name "github-actions[bot]"
        git config user.email "github-actions[bot]@users.noreply.github.com"
        git add ${{ env.OUTPUT_DIR }}
        git commit -m "Automated: Save ArcGIS layers from layers.csv" || echo "No changes to commit"
        git push
```

Create a layers.csv

```csv
URL
https://services.arcgis.com/P3ePLMYs2RVChIv1/arcgis/rest/services/USA_Counties_Generalized/FeatureServer/0
https://services.arcgis.com/P3ePLMYs2RVChIv1/arcgis/rest/services/USA_Counties_Generalized/FeatureServer/1
```
