package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"docker-auto/internal/model"
)

// DockerHub API endpoints
const (
	DockerHubBaseURL     = "https://hub.docker.com"
	DockerHubAPIV2       = "https://hub.docker.com/v2"
	DockerHubRegistryV2  = "https://registry-1.docker.io/v2"
	DockerHubSearchURL   = "https://hub.docker.com/v2/search/repositories"
	DockerHubRepoURL     = "https://hub.docker.com/v2/repositories"
)

// dockerHubClient implements the Client interface for Docker Hub
type dockerHubClient struct {
	baseURL    string
	httpClient *http.Client
	auth       *AuthConfig
	timeout    time.Duration
}

// NewDockerHubClient creates a new Docker Hub client
func NewDockerHubClient(auth *AuthConfig) Client {
	return &dockerHubClient{
		baseURL: DockerHubAPIV2,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth:    auth,
		timeout: 30 * time.Second,
	}
}

// NewDockerHubClientWithConfig creates a Docker Hub client with custom configuration
func NewDockerHubClientWithConfig(config *ClientConfig) Client {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DockerHubAPIV2
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	return &dockerHubClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		auth:       config.Auth,
		timeout:    timeout,
	}
}

// TestConnection tests the connection to Docker Hub
func (c *dockerHubClient) TestConnection(ctx context.Context) error {
	// Try to fetch repository information for a known public repository
	_, err := c.GetRepositoryInfo(ctx, "library/hello-world")
	if err != nil {
		return fmt.Errorf("failed to connect to Docker Hub: %w", err)
	}
	return nil
}

// GetRegistryInfo returns information about Docker Hub
func (c *dockerHubClient) GetRegistryInfo(ctx context.Context) (*RegistryInfo, error) {
	return &RegistryInfo{
		Name:        "Docker Hub",
		URL:         DockerHubBaseURL,
		Type:        "dockerhub",
		Version:     "v2",
		Available:   true,
		Features:    []string{"search", "public_repos", "official_images", "automated_builds"},
		LastChecked: time.Now(),
	}, nil
}

// CheckImageUpdate checks if there's an update available for an image
func (c *dockerHubClient) CheckImageUpdate(ctx context.Context, image, currentDigest string) (*UpdateCheckResult, error) {
	// Parse image reference
	registry, namespace, repository, tag := ParseImageRef(image)
	if registry != "docker.io" && registry != "" {
		return nil, fmt.Errorf("image is not from Docker Hub: %s", image)
	}

	// Build repository name
	var repoName string
	if namespace != "" && namespace != "library" {
		repoName = fmt.Sprintf("%s/%s", namespace, repository)
	} else {
		repoName = repository
	}

	// Get latest image information
	latestInfo, err := c.GetLatestImageInfo(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest image info: %w", err)
	}

	// Compare with current digest
	updateAvailable := latestInfo.Digest != currentDigest

	result := &UpdateCheckResult{
		Repository:      repoName,
		CurrentTag:      tag,
		CurrentDigest:   currentDigest,
		LatestTag:       latestInfo.Tag,
		LatestDigest:    latestInfo.Digest,
		UpdateAvailable: updateAvailable,
		LastChecked:     time.Now(),
	}

	// Determine update type based on tag comparison
	if updateAvailable {
		result.UpdateType = c.determineUpdateType(tag, latestInfo.Tag)
	}

	return result, nil
}

// GetLatestImageInfo gets the latest version information for an image
func (c *dockerHubClient) GetLatestImageInfo(ctx context.Context, image string) (*model.ImageVersion, error) {
	// Parse image reference
	registry, namespace, repository, tag := ParseImageRef(image)
	if registry != "docker.io" && registry != "" {
		return nil, fmt.Errorf("image is not from Docker Hub: %s", image)
	}

	// Build repository name
	var repoName string
	if namespace != "" && namespace != "library" {
		repoName = fmt.Sprintf("%s/%s", namespace, repository)
	} else {
		repoName = repository
	}

	// If tag is empty or "latest", get the latest tag
	if tag == "" || tag == "latest" {
		tags, err := c.GetImageTags(ctx, repoName, &TagListOptions{
			Repository: repoName,
			Limit:      1,
			Sort:       "last_updated",
			Order:      "desc",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get image tags: %w", err)
		}
		if len(tags) == 0 {
			return nil, fmt.Errorf("no tags found for repository: %s", repoName)
		}
		tag = tags[0].Name
	}

	// Get manifest for the specific tag
	manifest, err := c.GetImageManifest(ctx, repoName, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get image manifest: %w", err)
	}

	// Create ImageVersion from manifest
	imageVersion := &model.ImageVersion{
		ImageName:    image,
		Tag:          tag,
		Digest:       manifest.Digest,
		SizeBytes:    manifest.Size,
		Architecture: manifest.Architecture,
		OS:           manifest.OS,
		PublishedAt:  &manifest.Created,
		CheckedAt:    time.Now(),
	}

	// Add metadata if available
	if manifest.Labels != nil {
		labelsJSON, err := json.Marshal(manifest.Labels)
		if err == nil {
			imageVersion.Metadata = string(labelsJSON)
		}
	}

	return imageVersion, nil
}

// GetImageTags gets available tags for a repository
func (c *dockerHubClient) GetImageTags(ctx context.Context, repository string, options *TagListOptions) ([]*ImageTag, error) {
	// Handle library repositories
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	// Build URL
	tagsURL := fmt.Sprintf("%s/repositories/%s/tags", c.baseURL, repository)

	// Add query parameters
	params := url.Values{}
	if options != nil {
		if options.Limit > 0 {
			params.Set("page_size", strconv.Itoa(options.Limit))
		}
		if options.Sort != "" {
			params.Set("ordering", options.Sort)
			if options.Order == "desc" {
				params.Set("ordering", "-"+options.Sort)
			}
		}
	}

	if len(params) > 0 {
		tagsURL += "?" + params.Encode()
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", tagsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available
	if c.auth != nil {
		c.addAuthHeader(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub API returned status %d", resp.StatusCode)
	}

	// Parse response
	var tagsResponse struct {
		Count    int `json:"count"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
		Results  []struct {
			Name        string    `json:"name"`
			FullSize    int64     `json:"full_size"`
			ID          int64     `json:"id"`
			Repository  int64     `json:"repository"`
			Creator     int64     `json:"creator"`
			LastUpdater int64     `json:"last_updater"`
			LastUpdated time.Time `json:"last_updated"`
			ImageID     string    `json:"image_id"`
			V2          bool      `json:"v2"`
			Platforms   []struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
				OSVersion    string `json:"os_version,omitempty"`
				OSFeatures   string `json:"os_features,omitempty"`
				Variant      string `json:"variant,omitempty"`
				Features     string `json:"features,omitempty"`
			} `json:"platforms,omitempty"`
			Images []struct {
				Architecture string    `json:"architecture"`
				Features     string    `json:"features,omitempty"`
				Variant      string    `json:"variant,omitempty"`
				Digest       string    `json:"digest"`
				OS           string    `json:"os"`
				OSFeatures   string    `json:"os_features,omitempty"`
				OSVersion    string    `json:"os_version,omitempty"`
				Size         int64     `json:"size"`
				Status       string    `json:"status"`
				LastPulled   time.Time `json:"last_pulled"`
				LastPushed   time.Time `json:"last_pushed"`
			} `json:"images,omitempty"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to ImageTag objects
	tags := make([]*ImageTag, len(tagsResponse.Results))
	for i, result := range tagsResponse.Results {
		tag := &ImageTag{
			Name:    result.Name,
			Created: result.LastUpdated,
			Size:    result.FullSize,
		}

		// Get digest from images if available
		if len(result.Images) > 0 {
			tag.Digest = result.Images[0].Digest
			tag.Architecture = result.Images[0].Architecture
			tag.OS = result.Images[0].OS
		}

		tags[i] = tag
	}

	return tags, nil
}

// GetImageManifest gets manifest information for a specific image tag
func (c *dockerHubClient) GetImageManifest(ctx context.Context, repository, tag string) (*ImageManifest, error) {
	// Handle library repositories
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	// Use Docker Registry API v2 for manifest
	manifestURL := fmt.Sprintf("%s/%s/manifests/%s", DockerHubRegistryV2, repository, tag)

	req, err := http.NewRequestWithContext(ctx, "GET", manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate Accept headers for Docker Registry API
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	// Add authentication if available
	if c.auth != nil {
		c.addAuthHeader(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Registry API returned status %d for %s:%s", resp.StatusCode, repository, tag)
	}

	// Get Docker-Content-Digest header
	digest := resp.Header.Get("Docker-Content-Digest")

	// Parse manifest
	var manifestV2 struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
		Config        struct {
			MediaType string `json:"mediaType"`
			Size      int64  `json:"size"`
			Digest    string `json:"digest"`
		} `json:"config"`
		Layers []struct {
			MediaType string `json:"mediaType"`
			Size      int64  `json:"size"`
			Digest    string `json:"digest"`
		} `json:"layers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&manifestV2); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	// Calculate total size
	totalSize := manifestV2.Config.Size
	for _, layer := range manifestV2.Layers {
		totalSize += layer.Size
	}

	manifest := &ImageManifest{
		Digest:    digest,
		MediaType: manifestV2.MediaType,
		Size:      totalSize,
		Created:   time.Now(), // Would need to get from config blob for accurate time
	}

	// Convert layers
	manifest.Layers = make([]LayerInfo, len(manifestV2.Layers))
	for i, layer := range manifestV2.Layers {
		manifest.Layers[i] = LayerInfo{
			Digest:    layer.Digest,
			Size:      layer.Size,
			MediaType: layer.MediaType,
		}
	}

	// Set config info
	manifest.Config = &ConfigInfo{
		Digest:    manifestV2.Config.Digest,
		Size:      manifestV2.Config.Size,
		MediaType: manifestV2.Config.MediaType,
	}

	return manifest, nil
}

// SearchRepositories searches for repositories on Docker Hub
func (c *dockerHubClient) SearchRepositories(ctx context.Context, options *SearchOptions) ([]*RepositorySearchResult, error) {
	if options == nil || options.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Build search URL
	searchURL := DockerHubSearchURL
	params := url.Values{}
	params.Set("q", options.Query)

	if options.Limit > 0 {
		params.Set("page_size", strconv.Itoa(options.Limit))
	}

	if options.IsOfficial != nil && *options.IsOfficial {
		params.Set("is_official", "true")
	}

	if options.IsAutomated != nil && *options.IsAutomated {
		params.Set("is_automated", "true")
	}

	searchURL += "?" + params.Encode()

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub search API returned status %d", resp.StatusCode)
	}

	// Parse response
	var searchResponse struct {
		Count    int `json:"count"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
		Results  []struct {
			RepoName      string `json:"repo_name"`
			ShortDescription string `json:"short_description"`
			StarCount     int    `json:"star_count"`
			PullCount     string `json:"pull_count"`
			RepoOwner     string `json:"repo_owner"`
			IsOfficial    bool   `json:"is_official"`
			IsAutomated   bool   `json:"is_automated"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to RepositorySearchResult
	results := make([]*RepositorySearchResult, len(searchResponse.Results))
	for i, result := range searchResponse.Results {
		results[i] = &RepositorySearchResult{
			Name:        result.RepoName,
			Description: result.ShortDescription,
			Stars:       result.StarCount,
			IsOfficial:  result.IsOfficial,
			IsAutomated: result.IsAutomated,
		}
	}

	return results, nil
}

// GetRepositoryInfo gets detailed information about a repository
func (c *dockerHubClient) GetRepositoryInfo(ctx context.Context, repository string) (*RepositoryInfo, error) {
	// Handle library repositories
	if !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}

	// Build URL
	repoURL := fmt.Sprintf("%s/repositories/%s", c.baseURL, repository)

	req, err := http.NewRequestWithContext(ctx, "GET", repoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if available
	if c.auth != nil {
		c.addAuthHeader(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, NewRegistryError(ErrorCodeImageNotFound, fmt.Sprintf("repository not found: %s", repository))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub API returned status %d", resp.StatusCode)
	}

	// Parse response
	var repoResponse struct {
		User        string    `json:"user"`
		Name        string    `json:"name"`
		Namespace   string    `json:"namespace"`
		Status      int       `json:"status"`
		Description string    `json:"description"`
		IsPrivate   bool      `json:"is_private"`
		IsAutomated bool      `json:"is_automated"`
		CanEdit     bool      `json:"can_edit"`
		StarCount   int       `json:"star_count"`
		PullCount   int64     `json:"pull_count"`
		LastUpdated time.Time `json:"last_updated"`
		DateCreated time.Time `json:"date_created"`
		Affiliation string    `json:"affiliation"`
		Architecture []string  `json:"architectures,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	info := &RepositoryInfo{
		Name:         repository,
		Description:  repoResponse.Description,
		Stars:        repoResponse.StarCount,
		Pulls:        repoResponse.PullCount,
		IsOfficial:   repoResponse.Namespace == "library",
		IsAutomated:  repoResponse.IsAutomated,
		LastUpdated:  repoResponse.LastUpdated,
		Architecture: repoResponse.Architecture,
	}

	return info, nil
}

// GetSecurityScanResult gets security scan results (not available for Docker Hub free tier)
func (c *dockerHubClient) GetSecurityScanResult(ctx context.Context, repository, tag string) (*ScanResult, error) {
	return nil, fmt.Errorf("security scanning not available for Docker Hub free tier")
}

// Close closes the client connection
func (c *dockerHubClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// Helper methods

// addAuthHeader adds authentication header to request
func (c *dockerHubClient) addAuthHeader(req *http.Request) {
	if c.auth == nil {
		return
	}

	switch c.auth.AuthType {
	case "basic":
		if c.auth.Username != "" && c.auth.Password != "" {
			req.SetBasicAuth(c.auth.Username, c.auth.Password)
		}
	case "token":
		if c.auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.auth.Token)
		}
	}
}

// determineUpdateType determines the type of update based on tag comparison
func (c *dockerHubClient) determineUpdateType(currentTag, latestTag string) string {
	// Simple version comparison logic
	// In a real implementation, you might want to use semantic versioning

	if currentTag == latestTag {
		return "none"
	}

	// Check if tags follow semantic versioning pattern
	if c.isSemanticVersion(currentTag) && c.isSemanticVersion(latestTag) {
		return c.compareSemanticVersions(currentTag, latestTag)
	}

	// Check if tags are date-based
	if c.isDateVersion(currentTag) && c.isDateVersion(latestTag) {
		if latestTag > currentTag {
			return "minor"
		}
	}

	// Default to unknown for non-standard versioning
	return "unknown"
}

// isSemanticVersion checks if a tag follows semantic versioning
func (c *dockerHubClient) isSemanticVersion(tag string) bool {
	// Remove 'v' prefix if present
	if strings.HasPrefix(tag, "v") {
		tag = tag[1:]
	}

	parts := strings.Split(tag, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

// compareSemanticVersions compares two semantic version tags
func (c *dockerHubClient) compareSemanticVersions(current, latest string) string {
	// Remove 'v' prefix if present
	if strings.HasPrefix(current, "v") {
		current = current[1:]
	}
	if strings.HasPrefix(latest, "v") {
		latest = latest[1:]
	}

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	if len(currentParts) != 3 || len(latestParts) != 3 {
		return "unknown"
	}

	// Compare major version
	currentMajor, _ := strconv.Atoi(currentParts[0])
	latestMajor, _ := strconv.Atoi(latestParts[0])
	if latestMajor > currentMajor {
		return "major"
	}

	// Compare minor version
	currentMinor, _ := strconv.Atoi(currentParts[1])
	latestMinor, _ := strconv.Atoi(latestParts[1])
	if latestMinor > currentMinor {
		return "minor"
	}

	// Compare patch version
	currentPatch, _ := strconv.Atoi(currentParts[2])
	latestPatch, _ := strconv.Atoi(latestParts[2])
	if latestPatch > currentPatch {
		return "patch"
	}

	return "none"
}

// isDateVersion checks if a tag is date-based (YYYY-MM-DD or YYYYMMDD format)
func (c *dockerHubClient) isDateVersion(tag string) bool {
	// Check YYYY-MM-DD format
	if len(tag) == 10 && tag[4] == '-' && tag[7] == '-' {
		return true
	}

	// Check YYYYMMDD format
	if len(tag) == 8 {
		if _, err := strconv.Atoi(tag); err == nil {
			return true
		}
	}

	return false
}