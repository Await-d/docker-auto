package service

import (
	"context"
	"time"
)

// DockerServiceInterface defines Docker service operations
type DockerServiceInterface interface {
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	GetContainer(ctx context.Context, containerID string) (*ContainerInfo, error)
	GetLatestImageTag(ctx context.Context, imageName string) (string, error)
	ExecCommand(ctx context.Context, containerID, command string) error
	PullImage(ctx context.Context, imageName string) error
	StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
	UpdateContainer(ctx context.Context, containerID, newImage string) error
	GetImageLayers(ctx context.Context, imageID string) ([]string, error)
	GetImagePackages(ctx context.Context, imageID string) ([]PackageInfo, error)
}

// VulnerabilityDBInterface defines vulnerability database operations
type VulnerabilityDBInterface interface {
	QueryVulnerabilities(ctx context.Context, packageName, version string) ([]VulnerabilityInfo, error)
	UpdateVulnerabilityDB(ctx context.Context) error
	GetLastUpdateTime() time.Time
}

// SecurityScannerInterface defines security scanner operations
type SecurityScannerInterface interface {
	ScanImage(ctx context.Context, imageID, imageName string) (*ScanResult, error)
	GetScannerInfo() ScannerInfo
}

// ContainerInfo represents container information
type ContainerInfo struct {
	ID      string            `json:"id"`
	Names   []string          `json:"names"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	ImageID string            `json:"imageId"`
	Status  string            `json:"status"`
	State   string            `json:"state"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels"`
	Ports   []PortInfo        `json:"ports"`
}

// PortInfo represents container port information
type PortInfo struct {
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort,omitempty"`
	Type        string `json:"type"`
	IP          string `json:"ip,omitempty"`
}

// VulnerabilityInfo represents vulnerability information
type VulnerabilityInfo struct {
	CVE          string    `json:"cve"`
	Severity     string    `json:"severity"`
	Score        float64   `json:"score"`
	Vector       string    `json:"vector"`
	Description  string    `json:"description"`
	FixedVersion string    `json:"fixedVersion"`
	PublishedAt  time.Time `json:"publishedAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}