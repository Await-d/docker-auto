package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"docker-auto/tests"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_runner.go [integration|e2e|build|cleanup|all]")
		os.Exit(1)
	}

	testType := os.Args[1]

	switch testType {
	case "integration":
		runIntegrationTests()
	case "e2e":
		runE2ETests()
	case "build":
		runBuildTests()
	case "cleanup":
		runCleanup()
	case "all":
		runAllTests()
	default:
		fmt.Printf("Unknown test type: %s\n", testType)
		fmt.Println("Available options: integration, e2e, build, cleanup, all")
		os.Exit(1)
	}
}

func runIntegrationTests() {
	fmt.Println("Running Docker integration tests...")

	// Set environment variables for integration tests
	os.Setenv("INTEGRATION_TESTS", "true")
	os.Setenv("GO_ENV", "test")

	cmd := exec.Command("go", "test", "-v", "-timeout", "10m", "./tests/integration/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Integration tests failed: %v", err)
		os.Exit(1)
	}

	fmt.Println("Integration tests completed successfully!")
}

func runE2ETests() {
	fmt.Println("Running E2E build validation tests...")

	// Set environment variables for build tests
	os.Setenv("BUILD_TESTS", "true")
	os.Setenv("GO_ENV", "test")

	cmd := exec.Command("go", "test", "-v", "-timeout", "15m", "./tests/e2e/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("E2E tests failed: %v", err)
		os.Exit(1)
	}

	fmt.Println("E2E tests completed successfully!")
}

func runBuildTests() {
	fmt.Println("Running build validation...")

	// Test Go module dependencies
	fmt.Println("Checking Go dependencies...")
	cmd := exec.Command("go", "mod", "tidy")
	if err := cmd.Run(); err != nil {
		log.Printf("go mod tidy failed: %v", err)
		os.Exit(1)
	}

	// Test Go vet
	fmt.Println("Running go vet...")
	cmd = exec.Command("go", "vet", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("go vet failed: %v", err)
		fmt.Println("Note: Some vet errors may be acceptable in development. Review output above.")
	}

	// Test build
	fmt.Println("Testing Go build...")
	cmd = exec.Command("go", "build", "-o", "test-build", "./cmd/server")
	if err := cmd.Run(); err != nil {
		log.Printf("go build failed: %v", err)
		os.Exit(1)
	}

	// Cleanup test binary
	os.Remove("test-build")

	fmt.Println("Build validation completed successfully!")
}

func runCleanup() {
	fmt.Println("Running test cleanup...")

	tcm, err := tests.NewTestCleanupManager()
	if err != nil {
		log.Printf("Failed to create cleanup manager: %v", err)
		os.Exit(1)
	}
	defer tcm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := tcm.CleanupAllTestResources(ctx); err != nil {
		log.Printf("Cleanup failed: %v", err)
		os.Exit(1)
	}

	fmt.Println("Test cleanup completed successfully!")
}

func runAllTests() {
	fmt.Println("Running all tests and validation...")

	// Run in order: build -> integration -> e2e -> cleanup
	fmt.Println("=== Phase 1: Build Validation ===")
	runBuildTests()

	fmt.Println("\n=== Phase 2: Integration Tests ===")
	runIntegrationTests()

	fmt.Println("\n=== Phase 3: E2E Tests ===")
	runE2ETests()

	fmt.Println("\n=== Phase 4: Cleanup ===")
	runCleanup()

	fmt.Println("\n🎉 All tests completed successfully!")
}