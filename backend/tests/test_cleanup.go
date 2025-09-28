package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// TestCleanupManager provides comprehensive cleanup functionality for tests
type TestCleanupManager struct {
	dockerClient *client.Client
	logger       *log.Logger
}

// NewTestCleanupManager creates a new test cleanup manager
func NewTestCleanupManager() (*TestCleanupManager, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	logger := log.New(os.Stdout, "[TEST-CLEANUP] ", log.LstdFlags)

	return &TestCleanupManager{
		dockerClient: dockerClient,
		logger:       logger,
	}, nil
}

// CleanupAllTestResources performs comprehensive cleanup of all test-related resources
func (tcm *TestCleanupManager) CleanupAllTestResources(ctx context.Context) error {
	tcm.logger.Println("Starting comprehensive test cleanup...")

	var errors []string

	// Cleanup test containers
	if err := tcm.cleanupTestContainers(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("container cleanup: %v", err))
	}

	// Cleanup test images (except base images)
	if err := tcm.cleanupTestImages(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("image cleanup: %v", err))
	}

	// Cleanup test networks
	if err := tcm.cleanupTestNetworks(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("network cleanup: %v", err))
	}

	// Cleanup test volumes
	if err := tcm.cleanupTestVolumes(ctx); err != nil {
		errors = append(errors, fmt.Sprintf("volume cleanup: %v", err))
	}

	// Cleanup test files
	if err := tcm.cleanupTestFiles(); err != nil {
		errors = append(errors, fmt.Sprintf("file cleanup: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors occurred: %s", strings.Join(errors, "; "))
	}

	tcm.logger.Println("Test cleanup completed successfully")
	return nil
}

// cleanupTestContainers removes all containers with test labels
func (tcm *TestCleanupManager) cleanupTestContainers(ctx context.Context) error {
	tcm.logger.Println("Cleaning up test containers...")

	// List all containers with test labels
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "docker-auto.test=true")

	containers, err := tcm.dockerClient.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list test containers: %w", err)
	}

	if len(containers) == 0 {
		tcm.logger.Println("No test containers found")
		return nil
	}

	tcm.logger.Printf("Found %d test containers to cleanup", len(containers))

	for _, container := range containers {
		containerName := strings.TrimPrefix(container.Names[0], "/")

		tcm.logger.Printf("Removing container: %s (%s)", containerName, container.ID[:12])

		// Force remove container
		err := tcm.dockerClient.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
			RemoveLinks:   true,
		})
		if err != nil {
			tcm.logger.Printf("Warning: Failed to remove container %s: %v", containerName, err)
		} else {
			tcm.logger.Printf("Successfully removed container: %s", containerName)
		}
	}

	return nil
}

// cleanupTestImages removes test-created images (preserves base images)
func (tcm *TestCleanupManager) cleanupTestImages(ctx context.Context) error {
	tcm.logger.Println("Cleaning up test images...")

	// List all images
	images, err := tcm.dockerClient.ImageList(ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	baseImages := map[string]bool{
		"alpine":      true,
		"nginx":       true,
		"busybox":     true,
		"redis":       true,
		"postgres":    true,
		"mysql":       true,
		"ubuntu":      true,
		"debian":      true,
	}

	var testImages []types.ImageSummary
	for _, image := range images {
		// Check if image has test labels or is a test-created image
		if hasTestLabel(image.Labels) || isTestCreatedImage(image.RepoTags, image.RepoDigests) {
			// Skip base images even if they have test labels
			if !isBaseImage(image.RepoTags, baseImages) {
				testImages = append(testImages, image)
			}
		}
	}

	if len(testImages) == 0 {
		tcm.logger.Println("No test images found")
		return nil
	}

	tcm.logger.Printf("Found %d test images to cleanup", len(testImages))

	for _, image := range testImages {
		imageName := getImageName(image.RepoTags, image.ID)

		tcm.logger.Printf("Removing image: %s", imageName)

		_, err := tcm.dockerClient.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{
			Force: true,
		})
		if err != nil {
			tcm.logger.Printf("Warning: Failed to remove image %s: %v", imageName, err)
		} else {
			tcm.logger.Printf("Successfully removed image: %s", imageName)
		}
	}

	return nil
}

// cleanupTestNetworks removes test-created networks
func (tcm *TestCleanupManager) cleanupTestNetworks(ctx context.Context) error {
	tcm.logger.Println("Cleaning up test networks...")

	// List all networks
	networks, err := tcm.dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	var testNetworks []types.NetworkResource
	for _, network := range networks {
		// Skip default networks
		if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
			continue
		}

		// Check for test networks
		if hasTestLabel(network.Labels) || strings.Contains(network.Name, "test") {
			testNetworks = append(testNetworks, network)
		}
	}

	if len(testNetworks) == 0 {
		tcm.logger.Println("No test networks found")
		return nil
	}

	tcm.logger.Printf("Found %d test networks to cleanup", len(testNetworks))

	for _, network := range testNetworks {
		tcm.logger.Printf("Removing network: %s (%s)", network.Name, network.ID[:12])

		err := tcm.dockerClient.NetworkRemove(ctx, network.ID)
		if err != nil {
			tcm.logger.Printf("Warning: Failed to remove network %s: %v", network.Name, err)
		} else {
			tcm.logger.Printf("Successfully removed network: %s", network.Name)
		}
	}

	return nil
}

// cleanupTestVolumes removes test-created volumes
func (tcm *TestCleanupManager) cleanupTestVolumes(ctx context.Context) error {
	tcm.logger.Println("Cleaning up test volumes...")

	// List all volumes
	volumeListResponse, err := tcm.dockerClient.VolumeList(ctx, filters.Args{})
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	var testVolumes []*types.Volume
	for _, volume := range volumeListResponse.Volumes {
		// Check for test volumes
		if hasTestLabel(volume.Labels) || strings.Contains(volume.Name, "test") {
			testVolumes = append(testVolumes, volume)
		}
	}

	if len(testVolumes) == 0 {
		tcm.logger.Println("No test volumes found")
		return nil
	}

	tcm.logger.Printf("Found %d test volumes to cleanup", len(testVolumes))

	for _, volume := range testVolumes {
		tcm.logger.Printf("Removing volume: %s", volume.Name)

		err := tcm.dockerClient.VolumeRemove(ctx, volume.Name, true)
		if err != nil {
			tcm.logger.Printf("Warning: Failed to remove volume %s: %v", volume.Name, err)
		} else {
			tcm.logger.Printf("Successfully removed volume: %s", volume.Name)
		}
	}

	return nil
}

// cleanupTestFiles removes test-generated files and directories
func (tcm *TestCleanupManager) cleanupTestFiles() error {
	tcm.logger.Println("Cleaning up test files...")

	testFiles := []string{
		"docker-auto-server-test",
		"docker-auto-server-test1",
		"docker-auto-server-test2",
		".env.test",
		"test.db",
		"test.log",
		"integration-test.db",
	}

	var removedCount int
	for _, filename := range testFiles {
		if _, err := os.Stat(filename); err == nil {
			if err := os.Remove(filename); err != nil {
				tcm.logger.Printf("Warning: Failed to remove file %s: %v", filename, err)
			} else {
				tcm.logger.Printf("Removed test file: %s", filename)
				removedCount++
			}
		}
	}

	if removedCount == 0 {
		tcm.logger.Println("No test files found")
	} else {
		tcm.logger.Printf("Removed %d test files", removedCount)
	}

	return nil
}

// Helper functions

func hasTestLabel(labels map[string]string) bool {
	if labels == nil {
		return false
	}

	testLabelKeys := []string{
		"docker-auto.test",
		"test",
		"testing",
		"integration-test",
		"e2e-test",
	}

	for _, key := range testLabelKeys {
		if value, exists := labels[key]; exists && (value == "true" || value == "1") {
			return true
		}
	}

	return false
}

func isTestCreatedImage(repoTags, repoDigests []string) bool {
	testPrefixes := []string{
		"docker-auto-test",
		"test-",
		"e2e-test",
		"integration-test",
	}

	allTags := append(repoTags, repoDigests...)
	for _, tag := range allTags {
		for _, prefix := range testPrefixes {
			if strings.Contains(strings.ToLower(tag), prefix) {
				return true
			}
		}
	}

	return false
}

func isBaseImage(repoTags []string, baseImages map[string]bool) bool {
	for _, tag := range repoTags {
		if tag == "<none>:<none>" {
			continue
		}

		// Extract base name (before :)
		parts := strings.Split(tag, ":")
		if len(parts) > 0 {
			baseName := parts[0]
			// Remove registry prefix if present
			if slashIndex := strings.LastIndex(baseName, "/"); slashIndex != -1 {
				baseName = baseName[slashIndex+1:]
			}

			if baseImages[baseName] {
				return true
			}
		}
	}

	return false
}

func getImageName(repoTags []string, imageID string) string {
	if len(repoTags) > 0 && repoTags[0] != "<none>:<none>" {
		return repoTags[0]
	}
	return imageID[:12]
}

// Close closes the Docker client connection
func (tcm *TestCleanupManager) Close() error {
	if tcm.dockerClient != nil {
		return tcm.dockerClient.Close()
	}
	return nil
}

// CleanupAndExit provides a convenience function for test cleanup with exit
func CleanupAndExit() {
	tcm, err := NewTestCleanupManager()
	if err != nil {
		log.Printf("Failed to create cleanup manager: %v", err)
		os.Exit(1)
	}
	defer tcm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := tcm.CleanupAllTestResources(ctx); err != nil {
		log.Printf("Cleanup failed: %v", err)
		os.Exit(1)
	}

	log.Println("All test resources cleaned up successfully")
}