package registry

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"docker-auto/internal/model"

	"github.com/sirupsen/logrus"
)

// imageChecker implements the ImageChecker interface
type imageChecker struct {
	clients           map[string]Client
	clientsMutex      sync.RWMutex
	cache             map[string]*cacheEntry
	cacheMutex        sync.RWMutex
	cacheConfig       *CacheConfig
	defaultRegistry   string
	cleanupTicker     *time.Ticker
	ctx               context.Context
	cancel            context.CancelFunc
}

// cacheEntry represents a cached image information entry
type cacheEntry struct {
	data      *model.ImageVersion
	expiresAt time.Time
}

// NewImageChecker creates a new image checker instance
func NewImageChecker() ImageChecker {
	ctx, cancel := context.WithCancel(context.Background())

	checker := &imageChecker{
		clients: make(map[string]Client),
		cache:   make(map[string]*cacheEntry),
		cacheConfig: &CacheConfig{
			TTL:             6 * time.Hour,
			MaxEntries:      1000,
			CleanupInterval: 30 * time.Minute,
		},
		defaultRegistry: "docker.io",
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start cache cleanup routine
	checker.startCacheCleanup()

	return checker
}

// NewImageCheckerWithConfig creates a new image checker with custom configuration
func NewImageCheckerWithConfig(config *CacheConfig) ImageChecker {
	checker := NewImageChecker().(*imageChecker)
	if config != nil {
		checker.cacheConfig = config
	}
	return checker
}

// CheckImageUpdate checks for updates for a specific image
func (c *imageChecker) CheckImageUpdate(ctx context.Context, image, currentDigest string, registryURL string) (*UpdateCheckResult, error) {
	// Get appropriate client
	client, err := c.getClientForImage(image, registryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry client: %w", err)
	}

	// Check for update using the client
	result, err := client.CheckImageUpdate(ctx, image, currentDigest)
	if err != nil {
		return nil, fmt.Errorf("failed to check image update: %w", err)
	}

	// Cache the latest image info if update is available
	if result.UpdateAvailable && result.LatestDigest != "" {
		latestInfo, err := client.GetLatestImageInfo(ctx, image)
		if err == nil {
			c.CacheImageInfo(image, latestInfo, c.cacheConfig.TTL)
		}
	}

	return result, nil
}

// CheckAllImages checks for updates for multiple containers
func (c *imageChecker) CheckAllImages(ctx context.Context, containers []*model.Container) ([]*UpdateCheckResult, error) {
	results := make([]*UpdateCheckResult, len(containers))

	// Use goroutines for concurrent checking
	var wg sync.WaitGroup
	resultChan := make(chan struct {
		index  int
		result *UpdateCheckResult
		err    error
	}, len(containers))

	for i, container := range containers {
		wg.Add(1)
		go func(index int, cont *model.Container) {
			defer wg.Done()

			image := cont.GetFullImageName()
			var currentDigest string

			// For now, we don't have access to current digest from container
			// In a real implementation, this would be stored in container metadata
			currentDigest = ""

			result, err := c.CheckImageUpdate(ctx, image, currentDigest, cont.RegistryURL)
			resultChan <- struct {
				index  int
				result *UpdateCheckResult
				err    error
			}{index, result, err}
		}(i, container)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for item := range resultChan {
		if item.err != nil {
			logrus.WithError(item.err).WithField("container_id", containers[item.index].ID).Warn("Failed to check image update")
			// Create a failed result
			results[item.index] = &UpdateCheckResult{
				Repository:      containers[item.index].GetFullImageName(),
				CurrentTag:      containers[item.index].Tag,
				UpdateAvailable: false,
				LastChecked:     time.Now(),
			}
		} else {
			results[item.index] = item.result
		}
	}

	return results, nil
}

// CompareVersions compares two image versions
func (c *imageChecker) CompareVersions(current, latest *model.ImageVersion) (*VersionComparisonResult, error) {
	if current == nil || latest == nil {
		return nil, fmt.Errorf("both current and latest versions must be provided")
	}

	result := &VersionComparisonResult{
		CurrentVersion: current.Tag,
		LatestVersion:  latest.Tag,
		Confidence:     0.8, // Default confidence
	}

	// Compare digests first
	if current.Digest == latest.Digest {
		result.CompareResult = 0
		result.Recommendation = "No update needed - versions are identical"
		result.Confidence = 1.0
		return result, nil
	}

	// Determine version type and compare
	versionType := c.detectVersionType(current.Tag, latest.Tag)
	result.VersionType = versionType

	switch versionType {
	case "semantic":
		result.CompareResult = c.compareSemanticVersions(current.Tag, latest.Tag)
		result.Confidence = 0.95
	case "date":
		result.CompareResult = c.compareDateVersions(current.Tag, latest.Tag)
		result.Confidence = 0.9
	case "hash":
		result.CompareResult = c.compareHashVersions(current.Tag, latest.Tag)
		result.Confidence = 0.7
	default:
		result.CompareResult = c.compareStringVersions(current.Tag, latest.Tag)
		result.Confidence = 0.5
		result.VersionType = "unknown"
	}

	// Generate recommendation
	result.Recommendation = c.generateRecommendation(result)

	// Add change information if available
	result.Changes = c.generateVersionChanges(current, latest, result)

	return result, nil
}

// ShouldUpdate determines if an update should be applied based on policy
func (c *imageChecker) ShouldUpdate(comparison *VersionComparisonResult, policy string) bool {
	if comparison.CompareResult >= 0 {
		return false // Current is same or newer
	}

	switch policy {
	case "auto":
		// Auto update for patch and minor versions with high confidence
		if comparison.Confidence >= 0.8 {
			switch comparison.VersionType {
			case "semantic":
				// Only patch and minor updates for semantic versions
				return !c.isMajorVersionChange(comparison.CurrentVersion, comparison.LatestVersion)
			case "date":
				return true // Date-based versions are usually safe to update
			}
		}
		return false
	case "manual":
		return false // Manual updates only
	case "security":
		// Update if there are security fixes
		for _, issue := range comparison.SecurityIssues {
			if issue.Severity == "critical" || issue.Severity == "high" {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// GetUpdateStrategy determines the best update strategy
func (c *imageChecker) GetUpdateStrategy(comparison *VersionComparisonResult) string {
	if comparison.CompareResult >= 0 {
		return "none"
	}

	// Check for breaking changes or major version updates
	if c.isMajorVersionChange(comparison.CurrentVersion, comparison.LatestVersion) {
		return "blue_green" // Safer for major changes
	}

	// Check for security issues
	hasHighSeverityIssues := false
	for _, issue := range comparison.SecurityIssues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			hasHighSeverityIssues = true
			break
		}
	}

	if hasHighSeverityIssues {
		return "rolling" // Quick rollout for security fixes
	}

	// Default strategy based on confidence
	if comparison.Confidence >= 0.9 {
		return "recreate" // Simple and fast for high confidence
	}

	return "rolling" // Conservative approach
}

// RegisterClient registers a client for a specific registry type
func (c *imageChecker) RegisterClient(registryType string, client Client) {
	c.clientsMutex.Lock()
	defer c.clientsMutex.Unlock()
	c.clients[registryType] = client
}

// GetClient gets a client for a specific registry URL
func (c *imageChecker) GetClient(registryURL string) (Client, error) {
	if registryURL == "" {
		registryURL = c.defaultRegistry
	}

	registryType := c.detectRegistryType(registryURL)

	c.clientsMutex.RLock()
	client, exists := c.clients[registryType]
	c.clientsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no client registered for registry type: %s", registryType)
	}

	return client, nil
}

// GetSupportedRegistries returns list of supported registry types
func (c *imageChecker) GetSupportedRegistries() []string {
	c.clientsMutex.RLock()
	defer c.clientsMutex.RUnlock()

	registries := make([]string, 0, len(c.clients))
	for registryType := range c.clients {
		registries = append(registries, registryType)
	}

	return registries
}

// Cache management methods

// CacheImageInfo caches image information
func (c *imageChecker) CacheImageInfo(image string, info *model.ImageVersion, ttl time.Duration) error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Check cache size limit
	if len(c.cache) >= c.cacheConfig.MaxEntries {
		c.evictOldestEntries(c.cacheConfig.MaxEntries / 4) // Evict 25% of entries
	}

	c.cache[image] = &cacheEntry{
		data:      info,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// GetCachedImageInfo retrieves cached image information
func (c *imageChecker) GetCachedImageInfo(image string) (*model.ImageVersion, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[image]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.expiresAt) {
		delete(c.cache, image)
		return nil, false
	}

	return entry.data, true
}

// InvalidateCache removes specific image from cache
func (c *imageChecker) InvalidateCache(image string) error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	delete(c.cache, image)
	return nil
}

// ClearCache clears all cached data
func (c *imageChecker) ClearCache() error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cache = make(map[string]*cacheEntry)
	return nil
}

// Batch operations

// CheckMultipleImages checks multiple images concurrently
func (c *imageChecker) CheckMultipleImages(ctx context.Context, images []string, registryURL string) ([]*UpdateCheckResult, error) {
	results := make([]*UpdateCheckResult, len(images))

	var wg sync.WaitGroup
	resultChan := make(chan struct {
		index  int
		result *UpdateCheckResult
		err    error
	}, len(images))

	for i, image := range images {
		wg.Add(1)
		go func(index int, img string) {
			defer wg.Done()

			// Check cache first
			if cachedInfo, found := c.GetCachedImageInfo(img); found {
				result := &UpdateCheckResult{
					Repository:      img,
					LatestTag:       cachedInfo.Tag,
					LatestDigest:    cachedInfo.Digest,
					UpdateAvailable: false, // Can't determine without current digest
					LastChecked:     time.Now(),
				}
				resultChan <- struct {
					index  int
					result *UpdateCheckResult
					err    error
				}{index, result, nil}
				return
			}

			// Check for update
			result, err := c.CheckImageUpdate(ctx, img, "", registryURL)
			resultChan <- struct {
				index  int
				result *UpdateCheckResult
				err    error
			}{index, result, err}
		}(i, image)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for item := range resultChan {
		if item.err != nil {
			logrus.WithError(item.err).WithField("image", images[item.index]).Warn("Failed to check image")
		}
		results[item.index] = item.result
	}

	return results, nil
}

// RefreshAllCache refreshes all cached image information
func (c *imageChecker) RefreshAllCache(ctx context.Context) error {
	c.cacheMutex.RLock()
	images := make([]string, 0, len(c.cache))
	for image := range c.cache {
		images = append(images, image)
	}
	c.cacheMutex.RUnlock()

	// Refresh each image in the cache
	for _, image := range images {
		go func(img string) {
			registry, _, _, _ := ParseImageRef(img)
			client, err := c.GetClient(registry)
			if err != nil {
				return
			}

			latestInfo, err := client.GetLatestImageInfo(ctx, img)
			if err != nil {
				logrus.WithError(err).WithField("image", img).Warn("Failed to refresh cached image info")
				return
			}

			c.CacheImageInfo(img, latestInfo, c.cacheConfig.TTL)
		}(image)
	}

	return nil
}

// Configuration methods

// SetDefaultRegistry sets the default registry URL
func (c *imageChecker) SetDefaultRegistry(registryURL string) {
	c.defaultRegistry = registryURL
}

// GetDefaultRegistry gets the default registry URL
func (c *imageChecker) GetDefaultRegistry() string {
	return c.defaultRegistry
}

// SetCacheConfig sets cache configuration
func (c *imageChecker) SetCacheConfig(config *CacheConfig) {
	if config != nil {
		c.cacheConfig = config
	}
}

// Helper methods

// getClientForImage gets the appropriate client for an image
func (c *imageChecker) getClientForImage(image, registryURL string) (Client, error) {
	// If registryURL is provided, use it
	if registryURL != "" {
		return c.GetClient(registryURL)
	}

	// Extract registry from image reference
	registry, _, _, _ := ParseImageRef(image)
	return c.GetClient(registry)
}

// detectRegistryType detects registry type from URL
func (c *imageChecker) detectRegistryType(registryURL string) string {
	if registryURL == "" || registryURL == "docker.io" {
		return "dockerhub"
	}

	// Check for Harbor indicators
	if strings.Contains(registryURL, "harbor") {
		return "harbor"
	}

	// Check for other registry types
	if strings.Contains(registryURL, "gcr.io") {
		return "gcr"
	}
	if strings.Contains(registryURL, "amazonaws.com") {
		return "ecr"
	}
	if strings.Contains(registryURL, "azurecr.io") {
		return "acr"
	}

	// Default to generic registry
	return "generic"
}

// detectVersionType detects the versioning scheme used
func (c *imageChecker) detectVersionType(version1, version2 string) string {
	// Check if both versions follow semantic versioning
	if c.isSemanticVersion(version1) && c.isSemanticVersion(version2) {
		return "semantic"
	}

	// Check if both versions are date-based
	if c.isDateVersion(version1) && c.isDateVersion(version2) {
		return "date"
	}

	// Check if both versions are hash-based
	if c.isHashVersion(version1) && c.isHashVersion(version2) {
		return "hash"
	}

	return "unknown"
}

// isSemanticVersion checks if version follows semantic versioning
func (c *imageChecker) isSemanticVersion(version string) bool {
	// Remove 'v' prefix if present
	v := strings.TrimPrefix(version, "v")

	// Basic semantic version pattern: major.minor.patch
	pattern := `^\d+\.\d+\.\d+(-[a-zA-Z0-9\-\.]+)?(\+[a-zA-Z0-9\-\.]+)?$`
	matched, _ := regexp.MatchString(pattern, v)
	return matched
}

// isDateVersion checks if version is date-based
func (c *imageChecker) isDateVersion(version string) bool {
	// Check various date formats
	datePatterns := []string{
		`^\d{4}-\d{2}-\d{2}$`,         // YYYY-MM-DD
		`^\d{4}\d{2}\d{2}$`,           // YYYYMMDD
		`^\d{4}-\d{2}-\d{2}T\d{2}$`,   // YYYY-MM-DDTHH
		`^\d{4}\.\d{2}\.\d{2}$`,       // YYYY.MM.DD
	}

	for _, pattern := range datePatterns {
		if matched, _ := regexp.MatchString(pattern, version); matched {
			return true
		}
	}

	return false
}

// isHashVersion checks if version is hash-based (like git SHA)
func (c *imageChecker) isHashVersion(version string) bool {
	// Check for hex strings of various lengths
	patterns := []string{
		`^[a-f0-9]{7}$`,   // Short SHA
		`^[a-f0-9]{8}$`,   // 8-char hash
		`^[a-f0-9]{40}$`,  // Full SHA-1
		`^[a-f0-9]{64}$`,  // SHA-256
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, version); matched {
			return true
		}
	}

	return false
}

// compareSemanticVersions compares semantic versions
func (c *imageChecker) compareSemanticVersions(v1, v2 string) int {
	// Remove 'v' prefix
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split by dots and compare each part
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Ensure we have at least 3 parts
	for len(parts1) < 3 {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < 3 {
		parts2 = append(parts2, "0")
	}

	// Compare major, minor, patch
	for i := 0; i < 3; i++ {
		// Handle pre-release versions (remove everything after -)
		part1 := strings.Split(parts1[i], "-")[0]
		part2 := strings.Split(parts2[i], "-")[0]

		num1, _ := strconv.Atoi(part1)
		num2, _ := strconv.Atoi(part2)

		if num1 < num2 {
			return -1
		} else if num1 > num2 {
			return 1
		}
	}

	return 0
}

// compareDateVersions compares date-based versions
func (c *imageChecker) compareDateVersions(v1, v2 string) int {
	// Normalize date formats for comparison
	date1 := c.normalizeDateVersion(v1)
	date2 := c.normalizeDateVersion(v2)

	if date1 < date2 {
		return -1
	} else if date1 > date2 {
		return 1
	}
	return 0
}

// compareHashVersions compares hash-based versions
func (c *imageChecker) compareHashVersions(v1, v2 string) int {
	// For hash versions, we can't determine chronological order
	// So we just compare lexicographically
	if v1 < v2 {
		return -1
	} else if v1 > v2 {
		return 1
	}
	return 0
}

// compareStringVersions compares versions as strings
func (c *imageChecker) compareStringVersions(v1, v2 string) int {
	if v1 < v2 {
		return -1
	} else if v1 > v2 {
		return 1
	}
	return 0
}

// normalizeDateVersion normalizes date version to YYYYMMDD format
func (c *imageChecker) normalizeDateVersion(version string) string {
	// Remove common separators
	normalized := strings.ReplaceAll(version, "-", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, "T", "")

	// Take first 8 digits for YYYYMMDD comparison
	if len(normalized) >= 8 {
		return normalized[:8]
	}

	return normalized
}

// isMajorVersionChange checks if the version change is a major version change
func (c *imageChecker) isMajorVersionChange(currentVersion, latestVersion string) bool {
	if !c.isSemanticVersion(currentVersion) || !c.isSemanticVersion(latestVersion) {
		return false
	}

	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(latestVersion, "v")

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	if len(currentParts) == 0 || len(latestParts) == 0 {
		return false
	}

	currentMajor, _ := strconv.Atoi(currentParts[0])
	latestMajor, _ := strconv.Atoi(latestParts[0])

	return latestMajor > currentMajor
}

// generateRecommendation generates update recommendation
func (c *imageChecker) generateRecommendation(result *VersionComparisonResult) string {
	if result.CompareResult == 0 {
		return "No update needed - versions are identical"
	}

	if result.CompareResult > 0 {
		return "Current version is newer than available version"
	}

	// Current version is older
	switch result.VersionType {
	case "semantic":
		if c.isMajorVersionChange(result.CurrentVersion, result.LatestVersion) {
			return "Major version update available - review breaking changes before updating"
		}
		return "Update recommended - minor/patch version available"
	case "date":
		return "Newer date-based version available - update recommended"
	case "hash":
		return "Different hash version available - review changes before updating"
	default:
		return "Newer version available - review changes before updating"
	}
}

// generateVersionChanges generates version change information
func (c *imageChecker) generateVersionChanges(current, latest *model.ImageVersion, result *VersionComparisonResult) []VersionChange {
	changes := []VersionChange{}

	// Add generic change based on version type
	switch result.VersionType {
	case "semantic":
		if c.isMajorVersionChange(current.Tag, latest.Tag) {
			changes = append(changes, VersionChange{
				Type:        "breaking",
				Description: fmt.Sprintf("Major version update from %s to %s", current.Tag, latest.Tag),
				Impact:      "high",
				Date:        latest.CheckedAt,
			})
		} else {
			changes = append(changes, VersionChange{
				Type:        "feature",
				Description: fmt.Sprintf("Minor/patch update from %s to %s", current.Tag, latest.Tag),
				Impact:      "low",
				Date:        latest.CheckedAt,
			})
		}
	case "date":
		changes = append(changes, VersionChange{
			Type:        "feature",
			Description: fmt.Sprintf("Date-based update from %s to %s", current.Tag, latest.Tag),
			Impact:      "medium",
			Date:        latest.CheckedAt,
		})
	default:
		changes = append(changes, VersionChange{
			Type:        "unknown",
			Description: fmt.Sprintf("Version update from %s to %s", current.Tag, latest.Tag),
			Impact:      "medium",
			Date:        latest.CheckedAt,
		})
	}

	return changes
}

// extractDigestFromMetadata extracts digest from container metadata
func (c *imageChecker) extractDigestFromMetadata(metadata string) string {
	// This would parse the metadata JSON and extract digest if available
	// For now, return empty string as placeholder
	return ""
}

// Cache cleanup and maintenance

// startCacheCleanup starts the cache cleanup routine
func (c *imageChecker) startCacheCleanup() {
	if c.cacheConfig.CleanupInterval <= 0 {
		return
	}

	c.cleanupTicker = time.NewTicker(c.cacheConfig.CleanupInterval)

	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.cleanupExpiredEntries()
			case <-c.ctx.Done():
				return
			}
		}
	}()
}

// cleanupExpiredEntries removes expired entries from cache
func (c *imageChecker) cleanupExpiredEntries() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	now := time.Now()
	for image, entry := range c.cache {
		if now.After(entry.expiresAt) {
			delete(c.cache, image)
		}
	}
}

// evictOldestEntries removes the oldest entries from cache
func (c *imageChecker) evictOldestEntries(count int) {
	if count <= 0 {
		return
	}

	// Create slice of entries with timestamps
	type entryWithTime struct {
		image     string
		expiresAt time.Time
	}

	entries := make([]entryWithTime, 0, len(c.cache))
	for image, entry := range c.cache {
		entries = append(entries, entryWithTime{
			image:     image,
			expiresAt: entry.expiresAt,
		})
	}

	// Sort by expiration time (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].expiresAt.Before(entries[j].expiresAt)
	})

	// Remove oldest entries
	for i := 0; i < count && i < len(entries); i++ {
		delete(c.cache, entries[i].image)
	}
}