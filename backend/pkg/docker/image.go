package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
)

// Image pulling and management

// PullImage pulls a Docker image from a registry
func (d *DockerClient) PullImage(ctx context.Context, imageName string, options types.ImagePullOptions) (io.ReadCloser, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	reader, err := d.client.ImagePull(ctx, imageName, options)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}

	return reader, nil
}

// PullImageWithAuth pulls a Docker image with authentication
func (d *DockerClient) PullImageWithAuth(ctx context.Context, imageName string, authConfig *registry.AuthConfig) (io.ReadCloser, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	var options types.ImagePullOptions
	if authConfig != nil {
		authConfigBytes, err := json.Marshal(authConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal auth config: %w", err)
		}
		options.RegistryAuth = base64.URLEncoding.EncodeToString(authConfigBytes)
	}

	return d.PullImage(ctx, imageName, options)
}

// PullImageAndWait pulls an image and waits for completion
func (d *DockerClient) PullImageAndWait(ctx context.Context, imageName string, options types.ImagePullOptions) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	reader, err := d.PullImage(ctx, imageName, options)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Read all data to ensure pull completes
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}

	return nil
}

// BuildImage builds a Docker image from a Dockerfile
func (d *DockerClient) BuildImage(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if buildContext == nil {
		return types.ImageBuildResponse{}, fmt.Errorf("build context cannot be nil")
	}

	response, err := d.client.ImageBuild(ctx, buildContext, options)
	if err != nil {
		return types.ImageBuildResponse{}, fmt.Errorf("failed to build image: %w", err)
	}

	return response, nil
}

// Image information and inspection

// InspectImage gets detailed information about a Docker image
func (d *DockerClient) InspectImage(ctx context.Context, imageID string) (*types.ImageInspect, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if imageID == "" {
		return nil, fmt.Errorf("image ID cannot be empty")
	}

	imageInspect, _, err := d.client.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image %s: %w", imageID, err)
	}

	return &imageInspect, nil
}

// ListImages lists Docker images with optional filters
func (d *DockerClient) ListImages(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	images, err := d.client.ImageList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return images, nil
}

// ListAllImages lists all Docker images
func (d *DockerClient) ListAllImages(ctx context.Context) ([]types.ImageSummary, error) {
	return d.ListImages(ctx, types.ImageListOptions{All: true})
}

// FindImageByName finds images by name or tag
func (d *DockerClient) FindImageByName(ctx context.Context, imageName string) ([]types.ImageSummary, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	images, err := d.ListAllImages(ctx)
	if err != nil {
		return nil, err
	}

	var foundImages []types.ImageSummary
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if strings.Contains(tag, imageName) {
				foundImages = append(foundImages, img)
				break
			}
		}
	}

	return foundImages, nil
}

// GetImageHistory gets the history of an image
func (d *DockerClient) GetImageHistory(ctx context.Context, imageID string) ([]image.HistoryResponseItem, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if imageID == "" {
		return nil, fmt.Errorf("image ID cannot be empty")
	}

	history, err := d.client.ImageHistory(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image history for %s: %w", imageID, err)
	}

	return history, nil
}

// Image operations

// RemoveImage removes a Docker image
func (d *DockerClient) RemoveImage(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if imageID == "" {
		return nil, fmt.Errorf("image ID cannot be empty")
	}

	deleteResponse, err := d.client.ImageRemove(ctx, imageID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to remove image %s: %w", imageID, err)
	}

	return deleteResponse, nil
}

// TagImage tags an image with a new name and tag
func (d *DockerClient) TagImage(ctx context.Context, source, target string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if source == "" {
		return fmt.Errorf("source image cannot be empty")
	}

	if target == "" {
		return fmt.Errorf("target image cannot be empty")
	}

	err := d.client.ImageTag(ctx, source, target)
	if err != nil {
		return fmt.Errorf("failed to tag image %s as %s: %w", source, target, err)
	}

	return nil
}

// PushImage pushes an image to a registry
func (d *DockerClient) PushImage(ctx context.Context, imageName string, options types.ImagePushOptions) (io.ReadCloser, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	reader, err := d.client.ImagePush(ctx, imageName, options)
	if err != nil {
		return nil, fmt.Errorf("failed to push image %s: %w", imageName, err)
	}

	return reader, nil
}

// PushImageWithAuth pushes an image with authentication
func (d *DockerClient) PushImageWithAuth(ctx context.Context, imageName string, authConfig *registry.AuthConfig) (io.ReadCloser, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	var options types.ImagePushOptions
	if authConfig != nil {
		authConfigBytes, err := json.Marshal(authConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal auth config: %w", err)
		}
		options.RegistryAuth = base64.URLEncoding.EncodeToString(authConfigBytes)
	}

	return d.PushImage(ctx, imageName, options)
}

// SaveImage saves an image to a tar archive
func (d *DockerClient) SaveImage(ctx context.Context, imageNames []string) (io.ReadCloser, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if len(imageNames) == 0 {
		return nil, fmt.Errorf("image names cannot be empty")
	}

	reader, err := d.client.ImageSave(ctx, imageNames)
	if err != nil {
		return nil, fmt.Errorf("failed to save images: %w", err)
	}

	return reader, nil
}

// LoadImage loads an image from a tar archive
func (d *DockerClient) LoadImage(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if input == nil {
		return types.ImageLoadResponse{}, fmt.Errorf("input cannot be nil")
	}

	response, err := d.client.ImageLoad(ctx, input, quiet)
	if err != nil {
		return types.ImageLoadResponse{}, fmt.Errorf("failed to load image: %w", err)
	}

	return response, nil
}

// Image searching and registry operations

// SearchImages searches for images in Docker Hub
func (d *DockerClient) SearchImages(ctx context.Context, term string, options types.ImageSearchOptions) ([]registry.SearchResult, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if term == "" {
		return nil, fmt.Errorf("search term cannot be empty")
	}

	results, err := d.client.ImageSearch(ctx, term, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search images for term %s: %w", term, err)
	}

	return results, nil
}

// Image comparison and digest operations

// GetImageDigest gets the digest of an image
func (d *DockerClient) GetImageDigest(ctx context.Context, imageName string) (string, error) {
	imageInspect, err := d.InspectImage(ctx, imageName)
	if err != nil {
		return "", err
	}

	if len(imageInspect.RepoDigests) == 0 {
		return "", fmt.Errorf("no digest found for image %s", imageName)
	}

	// Extract digest from repo digest (format: repo@sha256:digest)
	for _, repoDigest := range imageInspect.RepoDigests {
		parts := strings.Split(repoDigest, "@")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("invalid digest format for image %s", imageName)
}

// CompareImageDigests compares digests of two images
func (d *DockerClient) CompareImageDigests(ctx context.Context, image1, image2 string) (bool, error) {
	digest1, err := d.GetImageDigest(ctx, image1)
	if err != nil {
		return false, fmt.Errorf("failed to get digest for image %s: %w", image1, err)
	}

	digest2, err := d.GetImageDigest(ctx, image2)
	if err != nil {
		return false, fmt.Errorf("failed to get digest for image %s: %w", image2, err)
	}

	return digest1 == digest2, nil
}

// Image utility functions

// ImageExists checks if an image exists
func (d *DockerClient) ImageExists(ctx context.Context, imageName string) (bool, error) {
	_, err := d.InspectImage(ctx, imageName)
	if err != nil {
		if IsImageNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsImageNotFoundError checks if an error is an image not found error
func IsImageNotFoundError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "No such image") ||
		strings.Contains(err.Error(), "not found"))
}

// GetImageSize gets the size of an image
func (d *DockerClient) GetImageSize(ctx context.Context, imageName string) (int64, error) {
	imageInspect, err := d.InspectImage(ctx, imageName)
	if err != nil {
		return 0, err
	}

	return imageInspect.Size, nil
}

// GetImageCreatedTime gets the creation time of an image
func (d *DockerClient) GetImageCreatedTime(ctx context.Context, imageName string) (time.Time, error) {
	imageInspect, err := d.InspectImage(ctx, imageName)
	if err != nil {
		return time.Time{}, err
	}

	createdTime, err := time.Parse(time.RFC3339Nano, imageInspect.Created)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse image created time: %w", err)
	}

	return createdTime, nil
}

// GetImageLabels gets the labels of an image
func (d *DockerClient) GetImageLabels(ctx context.Context, imageName string) (map[string]string, error) {
	imageInspect, err := d.InspectImage(ctx, imageName)
	if err != nil {
		return nil, err
	}

	return imageInspect.Config.Labels, nil
}

// GetImageArchitecture gets the architecture of an image
func (d *DockerClient) GetImageArchitecture(ctx context.Context, imageName string) (string, error) {
	imageInspect, err := d.InspectImage(ctx, imageName)
	if err != nil {
		return "", err
	}

	return imageInspect.Architecture, nil
}

// GetImageOS gets the operating system of an image
func (d *DockerClient) GetImageOS(ctx context.Context, imageName string) (string, error) {
	imageInspect, err := d.InspectImage(ctx, imageName)
	if err != nil {
		return "", err
	}

	return imageInspect.Os, nil
}

// Image filtering and advanced operations

// FilterImagesByLabel filters images by label
func (d *DockerClient) FilterImagesByLabel(ctx context.Context, labelKey, labelValue string) ([]types.ImageSummary, error) {
	filterArgs := filters.NewArgs()
	if labelValue != "" {
		filterArgs.Add("label", fmt.Sprintf("%s=%s", labelKey, labelValue))
	} else {
		filterArgs.Add("label", labelKey)
	}

	options := types.ImageListOptions{
		All:     true,
		Filters: filterArgs,
	}

	return d.ListImages(ctx, options)
}

// FilterImagesByReference filters images by reference
func (d *DockerClient) FilterImagesByReference(ctx context.Context, reference string) ([]types.ImageSummary, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("reference", reference)

	options := types.ImageListOptions{
		All:     true,
		Filters: filterArgs,
	}

	return d.ListImages(ctx, options)
}

// GetDanglingImages gets all dangling images
func (d *DockerClient) GetDanglingImages(ctx context.Context) ([]types.ImageSummary, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("dangling", "true")

	options := types.ImageListOptions{
		All:     true,
		Filters: filterArgs,
	}

	return d.ListImages(ctx, options)
}

// PruneImages removes unused images
func (d *DockerClient) PruneImages(ctx context.Context, options filters.Args) (types.ImagesPruneReport, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	report, err := d.client.ImagesPrune(ctx, options)
	if err != nil {
		return types.ImagesPruneReport{}, fmt.Errorf("failed to prune images: %w", err)
	}

	return report, nil
}

// Image update checking

// CheckForImageUpdate checks if there's a newer version of an image available
func (d *DockerClient) CheckForImageUpdate(ctx context.Context, imageName string) (*ImageUpdateInfo, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	// Get current image info
	currentImage, err := d.InspectImage(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect current image: %w", err)
	}

	// For update checking, we'll use the registry API approach
	// This is a simplified version - in production, you'd want to use registry API
	updateInfo := &ImageUpdateInfo{
		ImageName:     imageName,
		CurrentDigest: "",
		LatestDigest:  "",
		UpdateAvailable: false,
		CurrentCreated: currentImage.Created,
	}

	// Get current digest
	if len(currentImage.RepoDigests) > 0 {
		parts := strings.Split(currentImage.RepoDigests[0], "@")
		if len(parts) == 2 {
			updateInfo.CurrentDigest = parts[1]
		}
	}

	// In a real implementation, you would:
	// 1. Query the registry API for the latest manifest
	// 2. Compare digests
	// 3. Check timestamps

	return updateInfo, nil
}

// ImageUpdateInfo contains information about image updates
type ImageUpdateInfo struct {
	ImageName       string    `json:"image_name"`
	CurrentDigest   string    `json:"current_digest"`
	LatestDigest    string    `json:"latest_digest"`
	UpdateAvailable bool      `json:"update_available"`
	CurrentCreated  string    `json:"current_created"`
	LatestCreated   string    `json:"latest_created,omitempty"`
	SizeChange      int64     `json:"size_change,omitempty"`
}

// HasUpdate returns true if an update is available
func (i *ImageUpdateInfo) HasUpdate() bool {
	return i.UpdateAvailable
}

// GetSizeChangeFormatted returns formatted size change string
func (i *ImageUpdateInfo) GetSizeChangeFormatted() string {
	if i.SizeChange == 0 {
		return "No change"
	}
	if i.SizeChange > 0 {
		return fmt.Sprintf("+%d bytes", i.SizeChange)
	}
	return fmt.Sprintf("%d bytes", i.SizeChange)
}

// Image import and export

// ImportImage imports an image from a tarball
func (d *DockerClient) ImportImage(ctx context.Context, source types.ImageImportSource, ref string, options types.ImageImportOptions) (io.ReadCloser, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	reader, err := d.client.ImageImport(ctx, source, ref, options)
	if err != nil {
		return nil, fmt.Errorf("failed to import image: %w", err)
	}

	return reader, nil
}

// CommitContainer creates a new image from a container
func (d *DockerClient) CommitContainer(ctx context.Context, containerID string, options types.ContainerCommitOptions) (types.IDResponse, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return types.IDResponse{}, fmt.Errorf("container ID cannot be empty")
	}

	response, err := d.client.ContainerCommit(ctx, containerID, options)
	if err != nil {
		return types.IDResponse{}, fmt.Errorf("failed to commit container %s: %w", containerID, err)
	}

	return response, nil
}