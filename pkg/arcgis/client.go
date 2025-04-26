package arcgis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Client represents an ArcGIS client with configuration.
type Client struct {
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewClient creates a new ArcGIS client with the specified timeout.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Timeout: timeout,
	}
}

// IsArcGISOnlineItemURL checks if a URL points to an ArcGIS Online item page.
func IsArcGISOnlineItemURL(rawURL string) bool {
	return strings.Contains(strings.ToLower(rawURL), "arcgis.com/home/item.html")
}

// NormalizeArcGISURL normalizes an ArcGIS URL.
func NormalizeArcGISURL(rawURL string) string {
	lowerURL := strings.ToLower(rawURL)
	isArcGISService := strings.Contains(lowerURL, "/rest/services") || strings.Contains(lowerURL, "/arcgis/rest")
	isAGOLItem := strings.Contains(lowerURL, "arcgis.com/home/item.html")

	if !isArcGISService && !isAGOLItem {
		// If it doesn't look like an ArcGIS service or item URL, only ensure it has a scheme
		u, err := url.Parse(rawURL)
		if err == nil && u.Scheme == "" {
			// Check if it looks like a domain/path without scheme
			if strings.Contains(rawURL, ".") && !strings.Contains(rawURL, " ") && !strings.HasPrefix(rawURL, "/") {
				return "https://" + rawURL
			}
		}
		// Otherwise, return as is or handle parsing errors if needed
		return rawURL
	}

	// If it IS an ArcGIS service URL (or AGOL item URL, though AGOL items are less likely to need scheme added)
	u, err := url.Parse(rawURL)
	if err != nil {
		fmt.Printf("Warning: Failed to parse URL for normalization: %v\n", err)
		return rawURL // Return original on parse error
	}

	// Ensure scheme (only if it doesn't exist)
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	if isArcGISService {
		// Normalize path casing only for service URLs
		pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i, part := range pathParts {
			lowerPart := strings.ToLower(part)
			if lowerPart == "arcgis" {
				pathParts[i] = "ArcGIS"
			} else if lowerPart == "rest" {
				pathParts[i] = "rest"
			} else if lowerPart == "services" {
				pathParts[i] = "services"
			} else if lowerPart == "featureserver" {
				pathParts[i] = "FeatureServer"
			} else if lowerPart == "mapserver" {
				pathParts[i] = "MapServer"
			}
		}
		// Reconstruct path, respecting if original had leading slash
		if strings.HasPrefix(u.Path, "/") {
			u.Path = "/" + strings.Join(pathParts, "/")
		} else {
			u.Path = strings.Join(pathParts, "/")
		}

		// Handle trailing slashes more carefully
		lowerPathEnd := ""
		if len(pathParts) > 0 {
			lowerPathEnd = strings.ToLower(pathParts[len(pathParts)-1])
		}

		// Check if it's a base service URL
		isBaseServiceURL := lowerPathEnd == "mapserver" || lowerPathEnd == "featureserver"

		if isBaseServiceURL {
			// Ensure base service URLs end with a slash
			if !strings.HasSuffix(u.Path, "/") {
				u.Path += "/"
			}
		} else {
			// Remove trailing slash if it's not a base service URL
			if len(u.Path) > 1 && strings.HasSuffix(u.Path, "/") {
				u.Path = u.Path[:len(u.Path)-1]
			}
		}

		// Remove ?f= query parameter
		q := u.Query()
		q.Del("f")
		u.RawQuery = q.Encode()
	}

	return u.String()
}

// IsValidHTTPURL checks if a URL is a valid HTTP or HTTPS URL.
func IsValidHTTPURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// FetchAndDecode fetches data from a URL and decodes it into the target interface.
func (c *Client) FetchAndDecode(urlStr string, target interface{}) error {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %v", urlStr, err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return fmt.Errorf("request timed out fetching data from %s: %v", urlStr, err)
		}
		return fmt.Errorf("failed to fetch data from %s: %v", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK HTTP status %d from %s", resp.StatusCode, urlStr)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %v", urlStr, err)
	}

	return nil
}

// FetchFeatures fetches features from an ArcGIS FeatureServer layer.
func (c *Client) FetchFeatures(baseURL, layerID string) ([]Feature, error) {
	queryURL := fmt.Sprintf("%s/%s/query", baseURL, layerID)
	u, _ := url.Parse(queryURL)
	q := u.Query()
	q.Set("f", "json")
	q.Set("where", "1=1")
	q.Set("outFields", "*")
	q.Set("returnGeometry", "true")
	q.Set("outSR", "4326")
	u.RawQuery = q.Encode()

	fmt.Printf("    Fetching features: %s\n", u.String())

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create feature query request: %v", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return nil, fmt.Errorf("feature fetch request timed out: %v", err)
		}
		return nil, fmt.Errorf("feature fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feature fetch request failed with status %d", resp.StatusCode)
	}

	var featureResp FeatureResponse
	if err := json.NewDecoder(resp.Body).Decode(&featureResp); err != nil {
		return nil, fmt.Errorf("failed to parse feature response: %v", err)
	}

	if featureResp.Error != nil {
		return nil, fmt.Errorf("feature query API error: %s", featureResp.Error.Message)
	}

	if len(featureResp.Features) == 0 && !featureResp.ExceededTransferLimit {
		return nil, fmt.Errorf("no features found for layer %s at %s", layerID, baseURL)
	}
	if featureResp.ExceededTransferLimit {
		fmt.Printf("  Warning: Feature transfer limit exceeded for layer %s. Results may be incomplete.\n", layerID)
	}

	return featureResp.Features, nil
}

// FetchServiceLayers fetches the layers from an ArcGIS Feature Server or Map Server.
func (c *Client) FetchServiceLayers(serviceURL string, serviceType string) ([]AvailableLayerInfo, error) {
	fetchURL := fmt.Sprintf("%s?f=json", serviceURL)
	fmt.Printf("    Fetching service metadata: %s\n", fetchURL)

	availableLayers := []AvailableLayerInfo{}

	if serviceType == "FeatureServer" {
		var metadata FeatureServerMetadata
		err := c.FetchAndDecode(fetchURL, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Feature Server metadata from %s: %v", fetchURL, err)
		}
		if metadata.Error != nil {
			return nil, fmt.Errorf("feature Server API error: %s", metadata.Error.Message)
		}

		if len(metadata.Layers) == 0 && len(metadata.Tables) == 0 {
			fmt.Printf("      No layers or tables found in Feature Server metadata at %s\n", fetchURL)
		}

		for _, layer := range metadata.Layers {
			layerIDStr, ok := layer.ID.(json.Number)
			if !ok {
				fmt.Printf("      Warning: Could not parse layer ID for %s\n", layer.Name)
				continue
			}
			availableLayers = append(availableLayers, AvailableLayerInfo{
				ID:             layerIDStr.String(),
				Name:           layer.Name,
				Type:           layer.Type,
				GeometryType:   layer.GeometryType,
				ServiceURL:     serviceURL,
				IsFeatureLayer: true,
			})
		}
	} else if serviceType == "MapServer" {
		var metadata MapServiceMetadata
		err := c.FetchAndDecode(fetchURL, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Map Server metadata from %s: %v", fetchURL, err)
		}
		if metadata.Error != nil {
			return nil, fmt.Errorf("map Server API error: %s", metadata.Error.Message)
		}

		if len(metadata.Layers) == 0 {
			fmt.Printf("      No layers found in Map Server metadata at %s\n", fetchURL)
		}

		layerMap := make(map[int]MapServiceLayer)
		for _, layer := range metadata.Layers {
			layerMap[layer.ID] = layer
		}

		layerHierarchy := make(map[int][]string)
		var buildPath func(layerID int) []string
		buildPath = func(layerID int) []string {
			if path, exists := layerHierarchy[layerID]; exists {
				return path
			}
			layer, ok := layerMap[layerID]
			if !ok {
				return []string{}
			}

			layerHierarchy[layerID] = []string{}

			var path []string
			if layer.ParentLayerId != -1 {
				parentPath := buildPath(layer.ParentLayerId)
				path = append(parentPath, layer.Name)
			} else {
				path = []string{layer.Name}
			}
			layerHierarchy[layerID] = path
			return path
		}

		for _, layer := range metadata.Layers {
			if layer.Type == "Feature Layer" {
				parentPath := buildPath(layer.ID)
				if len(parentPath) > 0 {
					parentPath = parentPath[:len(parentPath)-1]
				}

				availableLayers = append(availableLayers, AvailableLayerInfo{
					ID:             strconv.Itoa(layer.ID),
					Name:           layer.Name,
					Type:           layer.Type,
					GeometryType:   layer.GeometryType,
					ServiceURL:     serviceURL,
					ParentPath:     parentPath,
					IsFeatureLayer: true,
				})
			} else {
				fmt.Printf("    Skipping layer '%s' (ID: %d) - Type is '%s', not 'Feature Layer'.\n", layer.Name, layer.ID, layer.Type)
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported service type for fetching layers: %s", serviceType)
	}

	return availableLayers, nil
}

// HandleArcGISOnlineItem handles processing for an ArcGIS Online item URL.
func (c *Client) HandleArcGISOnlineItem(itemPageURL string) (*ItemData, error) {
	re := regexp.MustCompile(`id=([a-f0-9]+)`)
	matches := re.FindStringSubmatch(itemPageURL)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract item ID from URL: %s", itemPageURL)
	}
	itemID := matches[1]
	fmt.Printf("  Item ID: %s\n", itemID)

	itemAPIURL := fmt.Sprintf("https://www.arcgis.com/sharing/rest/content/items/%s?f=json", itemID)

	var itemData ItemData
	err := c.FetchAndDecode(itemAPIURL, &itemData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch item metadata: %v", err)
	}
	if itemData.Error != nil {
		return nil, fmt.Errorf("item API error: %s", itemData.Error.Message)
	}

	fmt.Printf("  Item Type: %s\n", itemData.Type)
	return &itemData, nil
}

// HandleWebMap handles processing for an ArcGIS Online Web Map item.
func (c *Client) HandleWebMap(itemID string) (*WebMapData, error) {
	webMapDataURL := fmt.Sprintf("https://www.arcgis.com/sharing/rest/content/items/%s/data?f=json", itemID)
	fmt.Printf("  Fetching Web Map data: %s\n", webMapDataURL)

	var webMapData WebMapData
	err := c.FetchAndDecode(webMapDataURL, &webMapData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch web map data: %v", err)
	}
	if webMapData.Error != nil {
		return nil, fmt.Errorf("web map data API error: %s", webMapData.Error.Message)
	}

	return &webMapData, nil
}
