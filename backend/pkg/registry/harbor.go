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

// harborClient implements the HarborClient interface for Harbor registries
type harborClient struct {
	baseURL    string
	httpClient *http.Client
	auth       *AuthConfig
	timeout    time.Duration
	apiVersion string // v1.0 or v2.0
}

// NewHarborClient creates a new Harbor client
func NewHarborClient(baseURL string, auth *AuthConfig) HarborClient {
	return &harborClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth:       auth,
		timeout:    30 * time.Second,
		apiVersion: "v2.0", // Default to v2.0
	}
}

// NewHarborClientWithConfig creates a Harbor client with custom configuration
func NewHarborClientWithConfig(config *ClientConfig) HarborClient {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Set up TLS config if provided
	if config.TLSConfig != nil {
		// TODO: Implement TLS configuration
	}

	return &harborClient{
		baseURL:    strings.TrimSuffix(config.BaseURL, "/"),
		httpClient: httpClient,
		auth:       config.Auth,
		timeout:    timeout,
		apiVersion: "v2.0",
	}
}

// TestConnection tests the connection to Harbor
func (c *harborClient) TestConnection(ctx context.Context) error {
	// Try to get system info
	_, err := c.GetRegistryInfo(ctx)
	return err
}

// GetRegistryInfo returns information about Harbor registry
func (c *harborClient) GetRegistryInfo(ctx context.Context) (*RegistryInfo, error) {
	// Get system info
	systemInfoURL := fmt.Sprintf("%s/api/%s/systeminfo", c.baseURL, c.apiVersion)

	req, err := http.NewRequestWithContext(ctx, "GET", systemInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	var systemInfo struct {
		HarborVersion        string `json:"harbor_version"`
		RegistryURL          string `json:"registry_url"`
		ExternalURL          string `json:"external_url"`
		AuthMode             string `json:"auth_mode"`
		ProjectCreationRestriction string `json:"project_creation_restriction"`
		SelfRegistration     bool   `json:"self_registration"`
		HasCARoot            bool   `json:"has_ca_root"`
		HarborVersionDetails string `json:"harbor_version_details"`
		RegistryStorageProviderName string `json:"registry_storage_provider_name"`
		ReadOnly             bool   `json:"read_only"`
		WithNotary           bool   `json:"with_notary"`
		WithClair            bool   `json:"with_clair"`
		WithChartmuseum      bool   `json:"with_chartmuseum"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&systemInfo); err != nil {
		return nil, fmt.Errorf("failed to decode system info: %w", err)
	}

	features := []string{"projects", "repositories", "artifacts"}
	if systemInfo.WithClair {
		features = append(features, "vulnerability_scanning")
	}
	if systemInfo.WithNotary {
		features = append(features, "content_trust")
	}
	if systemInfo.WithChartmuseum {
		features = append(features, "helm_charts")
	}

	return &RegistryInfo{
		Name:        "Harbor",
		URL:         c.baseURL,
		Type:        "harbor",
		Version:     systemInfo.HarborVersion,
		Available:   true,
		Features:    features,
		Metadata: map[string]string{
			"auth_mode":          systemInfo.AuthMode,
			"external_url":       systemInfo.ExternalURL,
			"registry_url":       systemInfo.RegistryURL,
			"read_only":          strconv.FormatBool(systemInfo.ReadOnly),
			"with_clair":         strconv.FormatBool(systemInfo.WithClair),
			"with_notary":        strconv.FormatBool(systemInfo.WithNotary),
			"with_chartmuseum":   strconv.FormatBool(systemInfo.WithChartmuseum),
		},
		LastChecked: time.Now(),
	}, nil
}

// CheckImageUpdate checks if there's an update available for an image in Harbor
func (c *harborClient) CheckImageUpdate(ctx context.Context, image, currentDigest string) (*UpdateCheckResult, error) {
	// Parse Harbor image reference (harbor.example.com/project/repository:tag)
	parts := strings.Split(image, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid Harbor image reference: %s", image)
	}

	projectName := parts[1]
	repositoryPath := strings.Join(parts[2:], "/")

	// Extract tag from repository if present
	repository := repositoryPath
	tag := "latest"
	if strings.Contains(repositoryPath, ":") {
		repoParts := strings.SplitN(repositoryPath, ":", 2)
		repository = repoParts[0]
		tag = repoParts[1]
	}

	// Get latest artifact information
	latestInfo, err := c.GetLatestImageInfo(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest image info: %w", err)
	}

	// Compare with current digest
	updateAvailable := latestInfo.Digest != currentDigest

	result := &UpdateCheckResult{
		Repository:      fmt.Sprintf("%s/%s", projectName, repository),
		CurrentTag:      tag,
		CurrentDigest:   currentDigest,
		LatestTag:       latestInfo.Tag,
		LatestDigest:    latestInfo.Digest,
		UpdateAvailable: updateAvailable,
		LastChecked:     time.Now(),
	}

	// Get security scan results if available
	if updateAvailable {
		if scanResult, err := c.GetImageScanResult(ctx, projectName, repository, tag); err == nil && scanResult != nil {
			result.SecurityIssues = scanResult.Vulnerabilities
		}
	}

	return result, nil
}

// GetLatestImageInfo gets the latest version information for an image in Harbor
func (c *harborClient) GetLatestImageInfo(ctx context.Context, image string) (*model.ImageVersion, error) {
	// Parse Harbor image reference
	parts := strings.Split(image, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid Harbor image reference: %s", image)
	}

	projectName := parts[1]
	repositoryPath := strings.Join(parts[2:], "/")

	// Extract tag from repository if present
	repository := repositoryPath
	tag := "latest"
	if strings.Contains(repositoryPath, ":") {
		repoParts := strings.SplitN(repositoryPath, ":", 2)
		repository = repoParts[0]
		tag = repoParts[1]
	}

	// Get artifacts for the repository
	artifacts, err := c.GetArtifacts(ctx, projectName, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}

	if len(artifacts) == 0 {
		return nil, fmt.Errorf("no artifacts found for %s/%s", projectName, repository)
	}

	// Find the artifact with the specified tag or latest
	var targetArtifact *Artifact
	for _, artifact := range artifacts {
		for _, artifactTag := range artifact.Tags {
			if artifactTag.Name == tag {
				targetArtifact = artifact
				break
			}
		}
		if targetArtifact != nil {
			break
		}
	}

	// If tag not found, use the most recent artifact
	if targetArtifact == nil {
		targetArtifact = artifacts[0]
		if len(targetArtifact.Tags) > 0 {
			tag = targetArtifact.Tags[0].Name
		}
	}

	// Create ImageVersion from artifact
	imageVersion := &model.ImageVersion{
		ImageName:   image,
		Tag:         tag,
		Digest:      targetArtifact.Digest,
		SizeBytes:   targetArtifact.Size,
		PublishedAt: &targetArtifact.PushTime,
		CheckedAt:   time.Now(),
	}

	// Add metadata if available
	if targetArtifact.ExtraAttrs != nil {
		metadataJSON, err := json.Marshal(targetArtifact.ExtraAttrs)
		if err == nil {
			imageVersion.Metadata = string(metadataJSON)
		}
	}

	return imageVersion, nil
}

// GetImageTags gets available tags for a repository in Harbor
func (c *harborClient) GetImageTags(ctx context.Context, repository string, options *TagListOptions) ([]*ImageTag, error) {
	// Parse repository path (project/repository)
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format, expected project/repository: %s", repository)
	}

	projectName := parts[0]
	repoName := parts[1]

	// Get repository tags using Harbor API
	tags, err := c.GetRepositoryTags(ctx, projectName, repoName)
	if err != nil {
		return nil, err
	}

	// Convert to ImageTag format
	imageTags := make([]*ImageTag, len(tags))
	for i, tag := range tags {
		imageTags[i] = &ImageTag{
			Name:    tag.Name,
			Created: tag.PushTime,
		}
	}

	// Apply sorting and limits if specified
	if options != nil {
		// TODO: Implement sorting and pagination
	}

	return imageTags, nil
}

// GetImageManifest gets manifest information for a specific image tag in Harbor
func (c *harborClient) GetImageManifest(ctx context.Context, repository, tag string) (*ImageManifest, error) {
	// Parse repository path
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", repository)
	}

	projectName := parts[0]
	repoName := parts[1]

	// Get artifacts to find the one with the specified tag
	artifacts, err := c.GetArtifacts(ctx, projectName, repoName)
	if err != nil {
		return nil, err
	}

	// Find artifact with the specified tag
	var targetArtifact *Artifact
	for _, artifact := range artifacts {
		for _, artifactTag := range artifact.Tags {
			if artifactTag.Name == tag {
				targetArtifact = artifact
				break
			}
		}
		if targetArtifact != nil {
			break
		}
	}

	if targetArtifact == nil {
		return nil, fmt.Errorf("tag %s not found in %s", tag, repository)
	}

	// Build manifest from artifact information
	manifest := &ImageManifest{
		Digest:    targetArtifact.Digest,
		MediaType: targetArtifact.ManifestMediaType,
		Size:      targetArtifact.Size,
		Created:   targetArtifact.PushTime,
	}

	// Add labels from annotations
	if targetArtifact.Annotations != nil {
		manifest.Labels = targetArtifact.Annotations
	}

	return manifest, nil
}

// SearchRepositories searches for repositories in Harbor
func (c *harborClient) SearchRepositories(ctx context.Context, options *SearchOptions) ([]*RepositorySearchResult, error) {
	if options == nil || options.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Harbor search endpoint
	searchURL := fmt.Sprintf("%s/api/%s/search", c.baseURL, c.apiVersion)
	params := url.Values{}
	params.Set("q", options.Query)

	searchURL += "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Harbor search API returned status %d", resp.StatusCode)
	}

	var searchResponse struct {
		Repository []struct {
			ProjectID    int    `json:"project_id"`
			ProjectName  string `json:"project_name"`
			ProjectPublic bool  `json:"project_public"`
			RepositoryName string `json:"repository_name"`
			PullCount    int64  `json:"pull_count"`
			TagsCount    int    `json:"tags_count"`
		} `json:"repository"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to RepositorySearchResult
	results := make([]*RepositorySearchResult, len(searchResponse.Repository))
	for i, repo := range searchResponse.Repository {
		results[i] = &RepositorySearchResult{
			Name:        fmt.Sprintf("%s/%s", repo.ProjectName, repo.RepositoryName),
			Description: "", // Harbor doesn't provide description in search
		}
	}

	return results, nil
}

// GetRepositoryInfo gets detailed information about a repository in Harbor
func (c *harborClient) GetRepositoryInfo(ctx context.Context, repository string) (*RepositoryInfo, error) {
	// Parse repository path
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", repository)
	}

	projectName := parts[0]
	repoName := parts[1]

	// Get repository information
	repositories, err := c.GetRepositories(ctx, projectName)
	if err != nil {
		return nil, err
	}

	// Find the specific repository
	var targetRepo *Repository
	for _, repo := range repositories {
		if strings.HasSuffix(repo.Name, repoName) {
			targetRepo = repo
			break
		}
	}

	if targetRepo == nil {
		return nil, fmt.Errorf("repository not found: %s", repository)
	}

	info := &RepositoryInfo{
		Name:        repository,
		Description: targetRepo.Description,
		Pulls:       targetRepo.PullCount,
		LastUpdated: targetRepo.UpdateTime,
	}

	return info, nil
}

// GetSecurityScanResult gets security scan results for an image in Harbor
func (c *harborClient) GetSecurityScanResult(ctx context.Context, repository, tag string) (*ScanResult, error) {
	// Parse repository path
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", repository)
	}

	projectName := parts[0]
	repoName := parts[1]

	return c.GetImageScanResult(ctx, projectName, repoName, tag)
}

// Close closes the Harbor client
func (c *harborClient) Close() error {
	return nil
}

// Harbor-specific methods

// GetProjects gets all projects from Harbor
func (c *harborClient) GetProjects(ctx context.Context) ([]*Project, error) {
	projectsURL := fmt.Sprintf("%s/api/%s/projects", c.baseURL, c.apiVersion)

	req, err := http.NewRequestWithContext(ctx, "GET", projectsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	var projects []*Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %w", err)
	}

	return projects, nil
}

// GetRepositories gets repositories for a specific project
func (c *harborClient) GetRepositories(ctx context.Context, projectName string) ([]*Repository, error) {
	reposURL := fmt.Sprintf("%s/api/%s/projects/%s/repositories", c.baseURL, c.apiVersion, projectName)

	req, err := http.NewRequestWithContext(ctx, "GET", reposURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	var repositories []*Repository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, fmt.Errorf("failed to decode repositories: %w", err)
	}

	return repositories, nil
}

// GetArtifacts gets artifacts for a specific repository
func (c *harborClient) GetArtifacts(ctx context.Context, projectName, repoName string) ([]*Artifact, error) {
	artifactsURL := fmt.Sprintf("%s/api/%s/projects/%s/repositories/%s/artifacts",
		c.baseURL, c.apiVersion, projectName, url.PathEscape(repoName))

	req, err := http.NewRequestWithContext(ctx, "GET", artifactsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	var artifacts []*Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifacts); err != nil {
		return nil, fmt.Errorf("failed to decode artifacts: %w", err)
	}

	return artifacts, nil
}

// GetImageScanResult gets vulnerability scan results for a specific image
func (c *harborClient) GetImageScanResult(ctx context.Context, projectName, repoName, reference string) (*ScanResult, error) {
	scanURL := fmt.Sprintf("%s/api/%s/projects/%s/repositories/%s/artifacts/%s/scan/%s",
		c.baseURL, c.apiVersion, projectName, url.PathEscape(repoName), reference, "vulnerability")

	req, err := http.NewRequestWithContext(ctx, "GET", scanURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No scan results available
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Harbor scan API returned status %d", resp.StatusCode)
	}

	var scanResult *ScanResult
	if err := json.NewDecoder(resp.Body).Decode(&scanResult); err != nil {
		return nil, fmt.Errorf("failed to decode scan result: %w", err)
	}

	return scanResult, nil
}

// GetRepositoryTags gets tags for a specific repository
func (c *harborClient) GetRepositoryTags(ctx context.Context, projectName, repoName string) ([]*ArtifactTag, error) {
	// Get artifacts first, then extract tags
	artifacts, err := c.GetArtifacts(ctx, projectName, repoName)
	if err != nil {
		return nil, err
	}

	var allTags []*ArtifactTag
	for _, artifact := range artifacts {
		for _, tag := range artifact.Tags {
			allTags = append(allTags, &tag)
		}
	}

	return allTags, nil
}

// Project management methods

// CreateProject creates a new project in Harbor
func (c *harborClient) CreateProject(ctx context.Context, project *Project) error {
	projectsURL := fmt.Sprintf("%s/api/%s/projects", c.baseURL, c.apiVersion)

	projectData := map[string]interface{}{
		"project_name": project.Name,
		"public":       project.Public,
		"metadata":     project.Metadata,
	}

	jsonData, err := json.Marshal(projectData)
	if err != nil {
		return fmt.Errorf("failed to marshal project data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", projectsURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	return nil
}

// DeleteProject deletes a project from Harbor
func (c *harborClient) DeleteProject(ctx context.Context, projectID int) error {
	deleteURL := fmt.Sprintf("%s/api/%s/projects/%d", c.baseURL, c.apiVersion, projectID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	return nil
}

// UpdateProject updates a project in Harbor
func (c *harborClient) UpdateProject(ctx context.Context, project *Project) error {
	updateURL := fmt.Sprintf("%s/api/%s/projects/%d", c.baseURL, c.apiVersion, project.ID)

	projectData := map[string]interface{}{
		"metadata": project.Metadata,
	}

	jsonData, err := json.Marshal(projectData)
	if err != nil {
		return fmt.Errorf("failed to marshal project data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", updateURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	return nil
}

// DeleteRepository deletes a repository from Harbor
func (c *harborClient) DeleteRepository(ctx context.Context, projectName, repoName string) error {
	deleteURL := fmt.Sprintf("%s/api/%s/projects/%s/repositories/%s",
		c.baseURL, c.apiVersion, projectName, url.PathEscape(repoName))

	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Harbor API returned status %d", resp.StatusCode)
	}

	return nil
}

// Helper methods

// addAuthHeader adds authentication header to request
func (c *harborClient) addAuthHeader(req *http.Request) {
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