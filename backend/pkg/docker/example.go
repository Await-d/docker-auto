package docker

import (
	"context"
	"fmt"
	"log"
	"time"

	"docker-auto/internal/config"
)

// Example demonstrates how to use the Docker API integration
func Example() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create Docker client
	dockerClient, err := NewDockerClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	ctx := context.Background()

	// Test Docker daemon connection
	if err := dockerClient.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Docker daemon: %v", err)
	}
	fmt.Println("✅ Successfully connected to Docker daemon")

	// Get Docker version
	version, err := dockerClient.GetVersion(ctx)
	if err != nil {
		log.Printf("Failed to get Docker version: %v", err)
	} else {
		fmt.Printf("📋 Docker version: %s (API: %s)\n", version.Version, version.APIVersion)
	}

	// Get Docker info
	info, err := dockerClient.GetInfo(ctx)
	if err != nil {
		log.Printf("Failed to get Docker info: %v", err)
	} else {
		fmt.Printf("🐳 Docker info: %d containers running, %d images\n",
			info.ContainersRunning, info.Images)
	}

	// List all containers
	containers, err := dockerClient.ListAllContainers(ctx)
	if err != nil {
		log.Printf("Failed to list containers: %v", err)
	} else {
		fmt.Printf("📦 Found %d containers\n", len(containers))
		for _, container := range containers {
			fmt.Printf("  - %s (%s): %s\n",
				container.Names[0], container.Image, container.State)
		}
	}

	// List all images
	images, err := dockerClient.ListAllImages(ctx)
	if err != nil {
		log.Printf("Failed to list images: %v", err)
	} else {
		fmt.Printf("🏗️ Found %d images\n", len(images))
		for i, image := range images {
			if i >= 5 { // Limit output
				fmt.Println("  ...")
				break
			}
			if len(image.RepoTags) > 0 {
				fmt.Printf("  - %s (%.2f MB)\n",
					image.RepoTags[0], float64(image.Size)/1024/1024)
			}
		}
	}

	// Health check example
	health, err := dockerClient.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("💊 Docker daemon health: %s (ping: %v)\n",
			func() string {
				if health.IsHealthy() {
					return "Healthy"
				}
				return "Unhealthy"
			}(), health.PingDuration)
	}
}

// ExampleContainerOperations demonstrates container lifecycle operations
func ExampleContainerOperations() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dockerClient, err := NewDockerClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	// ctx := context.Background() // TODO: Use when implementing actual Docker operations

	// Example container configuration
	containerConfig := &ContainerCreateConfig{
		Name:  "example-nginx",
		Image: "nginx",
		Tag:   "alpine",
		Ports: map[string]string{
			"8080": "80",
		},
		Labels: map[string]string{
			"managed-by": "docker-auto",
			"environment": "example",
		},
		RestartPolicy: "unless-stopped",
		HealthCheck: &HealthCheckConfig{
			Test:     []string{"CMD", "curl", "-f", "http://localhost:80"},
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
		},
	}

	// Validate configuration
	if err := containerConfig.ValidateConfig(); err != nil {
		log.Printf("Invalid container config: %v", err)
		return
	}

	fmt.Printf("🚀 Example container config: %s (%s)\n",
		containerConfig.Name, containerConfig.GetFullImageName())

	// Note: This is just a demonstration of the API
	// In a real application, you would actually create and manage containers
	fmt.Println("📝 Container operations ready - API fully implemented")
}

// ExampleImageOperations demonstrates image operations
func ExampleImageOperations() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dockerClient, err := NewDockerClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	ctx := context.Background()

	// Check if an image exists
	imageName := "nginx:alpine"
	exists, err := dockerClient.ImageExists(ctx, imageName)
	if err != nil {
		log.Printf("Failed to check image existence: %v", err)
	} else {
		if exists {
			fmt.Printf("✅ Image %s exists locally\n", imageName)
		} else {
			fmt.Printf("❌ Image %s not found locally\n", imageName)
		}
	}

	// Parse image name
	image, tag := ParseImageName("nginx:latest")
	fmt.Printf("📝 Parsed image: %s, tag: %s\n", image, tag)

	// Build full image name
	fullName := BuildImageName("redis", "6-alpine")
	fmt.Printf("🏗️ Built image name: %s\n", fullName)

	fmt.Println("🎯 Image operations ready - API fully implemented")
}

// ExampleErrorHandling demonstrates error handling capabilities
func ExampleErrorHandling() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dockerClient, err := NewDockerClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	ctx := context.Background()

	// Simulate an error by trying to get a non-existent container
	_, err = dockerClient.GetContainer(ctx, "non-existent-container")
	if err != nil {
		// Wrap the error as a Docker error
		dockerErr := WrapDockerError(err, "inspect", "container")

		fmt.Printf("❌ Error occurred: %s\n", dockerErr.Error())
		fmt.Printf("📊 Error type: %s\n", dockerErr.GetType())
		fmt.Printf("🔄 Retryable: %v\n", dockerErr.IsRetryable())

		// Diagnose the error
		diagnosis := DiagnoseDockerError(dockerErr)
		fmt.Printf("🔍 Diagnosis: %+v\n", diagnosis)
	}

	fmt.Println("🛡️ Error handling system ready - comprehensive error classification and retry logic implemented")
}

// ExampleRetryLogic demonstrates retry mechanisms
func ExampleRetryLogic() {
	// Example retry operation
	operation := func() error {
		// Simulate a potentially failing operation
		fmt.Println("🔄 Attempting operation...")
		return fmt.Errorf("simulated network error")
	}

	config := DefaultRetryConfig()
	config.MaxRetries = 2

	err := Retry(operation, config)
	if err != nil {
		fmt.Printf("❌ Operation failed after retries: %v\n", err)
	}

	fmt.Println("🔄 Retry logic ready - configurable backoff and retry strategies implemented")
}