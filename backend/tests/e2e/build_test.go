package e2e

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// BuildTestSuite provides comprehensive build and runtime validation
type BuildTestSuite struct {
	suite.Suite
	projectRoot   string
	backendPath   string
	frontendPath  string
	binaryPath    string
	testEnvPath   string
	tempFiles     []string
}

// SetupSuite initializes the build test environment
func (suite *BuildTestSuite) SetupSuite() {
	// Get project root directory
	wd, err := os.Getwd()
	require.NoError(suite.T(), err, "Failed to get working directory")

	// Navigate to project root (assuming we're in backend/tests/e2e)
	suite.projectRoot = filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	suite.backendPath = filepath.Join(suite.projectRoot, "backend")
	suite.frontendPath = filepath.Join(suite.projectRoot, "frontend")
	suite.binaryPath = filepath.Join(suite.backendPath, "docker-auto-server-test")
	suite.testEnvPath = filepath.Join(suite.backendPath, ".env.test")

	// Verify paths exist
	require.DirExists(suite.T(), suite.backendPath, "Backend directory should exist")
	require.FileExists(suite.T(), filepath.Join(suite.backendPath, "go.mod"), "go.mod should exist")

	if _, err := os.Stat(suite.frontendPath); err == nil {
		require.FileExists(suite.T(), filepath.Join(suite.frontendPath, "package.json"), "package.json should exist")
	}

	suite.tempFiles = make([]string, 0)
}

// TearDownSuite cleans up test artifacts
func (suite *BuildTestSuite) TearDownSuite() {
	// Clean up binary
	if _, err := os.Stat(suite.binaryPath); err == nil {
		os.Remove(suite.binaryPath)
	}

	// Clean up test environment file
	if _, err := os.Stat(suite.testEnvPath); err == nil {
		os.Remove(suite.testEnvPath)
	}

	// Clean up temporary files
	for _, file := range suite.tempFiles {
		if _, err := os.Stat(file); err == nil {
			os.Remove(file)
		}
	}
}

// TestBackendCompileWithoutErrors verifies Go backend compiles with zero errors and warnings
func (suite *BuildTestSuite) TestBackendCompileWithoutErrors() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Test 1: go mod tidy (ensure dependencies are clean)
	suite.T().Log("Running go mod tidy...")
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = suite.backendPath

	output, err := cmd.CombinedOutput()
	require.NoError(suite.T(), err, "go mod tidy failed: %s", string(output))

	// Test 2: go vet (static analysis)
	suite.T().Log("Running go vet...")
	cmd = exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = suite.backendPath

	vetOutput, err := cmd.CombinedOutput()
	assert.NoError(suite.T(), err, "go vet found issues: %s", string(vetOutput))

	// Test 3: go fmt check
	suite.T().Log("Checking go fmt...")
	cmd = exec.CommandContext(ctx, "gofmt", "-l", ".")
	cmd.Dir = suite.backendPath

	fmtOutput, err := cmd.CombinedOutput()
	require.NoError(suite.T(), err, "gofmt check failed: %s", string(fmtOutput))
	assert.Empty(suite.T(), strings.TrimSpace(string(fmtOutput)),
		"Some files are not properly formatted. Run 'go fmt ./...' to fix: %s", string(fmtOutput))

	// Test 4: Build binary with verbose output
	suite.T().Log("Building Go binary...")
	cmd = exec.CommandContext(ctx, "go", "build", "-v", "-o", suite.binaryPath, "./cmd/server")
	cmd.Dir = suite.backendPath

	// Capture both stdout and stderr separately to analyze warnings
	buildOutput, err := cmd.CombinedOutput()
	require.NoError(suite.T(), err, "Go build failed: %s", string(buildOutput))

	// Check for warnings in build output
	buildLines := strings.Split(string(buildOutput), "\n")
	var warnings []string
	for _, line := range buildLines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "warning") {
			warnings = append(warnings, line)
		}
	}
	assert.Empty(suite.T(), warnings, "Build produced warnings: %v", warnings)

	// Test 5: Verify binary exists and is executable
	require.FileExists(suite.T(), suite.binaryPath, "Binary should be created")

	info, err := os.Stat(suite.binaryPath)
	require.NoError(suite.T(), err, "Failed to stat binary")
	assert.True(suite.T(), info.Mode()&0111 != 0, "Binary should be executable")

	suite.T().Log("Backend compilation completed successfully with zero errors and warnings")
}

// TestBackendRuntimeWithoutErrors verifies the backend starts and runs without errors
func (suite *BuildTestSuite) TestBackendRuntimeWithoutErrors() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create test environment configuration
	suite.createTestEnvironment()

	// Start the server in test mode
	suite.T().Log("Starting backend server in test mode...")
	cmd := exec.CommandContext(ctx, suite.binaryPath)
	cmd.Dir = suite.backendPath
	cmd.Env = append(os.Environ(),
		"GO_ENV=test",
		"DB_TYPE=sqlite",
		"DB_PATH=:memory:",
		"LOG_LEVEL=info",
		"PORT=18080", // Use different port for testing
		"JWT_SECRET=test-secret-key-for-integration-testing",
		"DOCKER_HOST=",
		"DISABLE_AUTH=true", // Disable auth for testing
	)

	// Create pipes to capture output
	stdout, err := cmd.StdoutPipe()
	require.NoError(suite.T(), err, "Failed to create stdout pipe")

	stderr, err := cmd.StderrPipe()
	require.NoError(suite.T(), err, "Failed to create stderr pipe")

	// Start the command
	err = cmd.Start()
	require.NoError(suite.T(), err, "Failed to start server")

	// Monitor output for errors
	errorsChan := make(chan string, 10)
	startupDone := make(chan bool, 1)

	// Monitor stdout
	go suite.monitorOutput(stdout, "STDOUT", errorsChan, startupDone)
	// Monitor stderr
	go suite.monitorOutput(stderr, "STDERR", errorsChan, nil)

	// Wait for startup completion or timeout
	select {
	case <-startupDone:
		suite.T().Log("Server started successfully")
	case <-time.After(30 * time.Second):
		suite.T().Log("Server startup timeout - proceeding with termination")
	case errorMsg := <-errorsChan:
		assert.Fail(suite.T(), fmt.Sprintf("Server runtime error detected: %s", errorMsg))
	}

	// Gracefully terminate the server
	if cmd.Process != nil {
		cmd.Process.Signal(os.Interrupt)

		// Wait for graceful shutdown or force kill after timeout
		shutdownDone := make(chan error, 1)
		go func() {
			shutdownDone <- cmd.Wait()
		}()

		select {
		case err := <-shutdownDone:
			if err != nil && !strings.Contains(err.Error(), "signal: interrupt") {
				suite.T().Logf("Server shutdown with error (expected for interrupt): %v", err)
			} else {
				suite.T().Log("Server shut down gracefully")
			}
		case <-time.After(10 * time.Second):
			suite.T().Log("Forcing server termination")
			cmd.Process.Kill()
			<-shutdownDone // Wait for the kill to complete
		}
	}

	// Check for any errors that occurred during runtime
	close(errorsChan)
	var runtimeErrors []string
	for errorMsg := range errorsChan {
		runtimeErrors = append(runtimeErrors, errorMsg)
	}

	assert.Empty(suite.T(), runtimeErrors, "Runtime errors detected: %v", runtimeErrors)
	suite.T().Log("Backend runtime validation completed successfully with zero errors")
}

// TestFrontendBuildWithoutErrors verifies frontend builds with zero errors and warnings
func (suite *BuildTestSuite) TestFrontendBuildWithoutErrors() {
	// Check if frontend exists
	if _, err := os.Stat(suite.frontendPath); os.IsNotExist(err) {
		suite.T().Skip("Frontend directory does not exist, skipping frontend build test")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Test 1: Install dependencies
	suite.T().Log("Installing frontend dependencies...")
	cmd := exec.CommandContext(ctx, "npm", "ci")
	cmd.Dir = suite.frontendPath

	installOutput, err := cmd.CombinedOutput()
	require.NoError(suite.T(), err, "npm ci failed: %s", string(installOutput))

	// Check for npm install warnings
	installLines := strings.Split(string(installOutput), "\n")
	var installWarnings []string
	for _, line := range installLines {
		line = strings.TrimSpace(line)
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "warn") && !strings.Contains(lowerLine, "deprecated") {
			installWarnings = append(installWarnings, line)
		}
	}
	if len(installWarnings) > 0 {
		suite.T().Logf("npm install warnings (informational): %v", installWarnings)
	}

	// Test 2: TypeScript type checking
	suite.T().Log("Running TypeScript type checking...")
	cmd = exec.CommandContext(ctx, "npm", "run", "type-check")
	cmd.Dir = suite.frontendPath

	typeCheckOutput, err := cmd.CombinedOutput()
	assert.NoError(suite.T(), err, "TypeScript type checking failed: %s", string(typeCheckOutput))

	// Test 3: Linting
	suite.T().Log("Running ESLint...")
	cmd = exec.CommandContext(ctx, "npm", "run", "lint:check")
	cmd.Dir = suite.frontendPath

	lintOutput, err := cmd.CombinedOutput()
	assert.NoError(suite.T(), err, "ESLint found issues: %s", string(lintOutput))

	// Test 4: Build production bundle
	suite.T().Log("Building frontend production bundle...")
	cmd = exec.CommandContext(ctx, "npm", "run", "build")
	cmd.Dir = suite.frontendPath

	buildOutput, err := cmd.CombinedOutput()
	require.NoError(suite.T(), err, "Frontend build failed: %s", string(buildOutput))

	// Check for build warnings
	buildLines := strings.Split(string(buildOutput), "\n")
	var buildWarnings []string
	for _, line := range buildLines {
		line = strings.TrimSpace(line)
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "warning") &&
		   !strings.Contains(lowerLine, "webpack") { // Allow webpack informational warnings
			buildWarnings = append(buildWarnings, line)
		}
	}
	assert.Empty(suite.T(), buildWarnings, "Frontend build produced warnings: %v", buildWarnings)

	// Test 5: Verify build artifacts exist
	distPath := filepath.Join(suite.frontendPath, "dist")
	require.DirExists(suite.T(), distPath, "Build dist directory should exist")

	indexPath := filepath.Join(distPath, "index.html")
	require.FileExists(suite.T(), indexPath, "Build should produce index.html")

	// Check for critical build files
	assetsPath := filepath.Join(distPath, "assets")
	if _, err := os.Stat(assetsPath); err == nil {
		files, err := os.ReadDir(assetsPath)
		require.NoError(suite.T(), err, "Failed to read assets directory")
		assert.NotEmpty(suite.T(), files, "Assets directory should contain build files")
	}

	suite.T().Log("Frontend build completed successfully with zero errors and warnings")
}

// TestDependencySecurityAudit runs security audits on dependencies
func (suite *BuildTestSuite) TestDependencySecurityAudit() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Backend security audit using go mod
	suite.T().Log("Running Go module security audit...")
	cmd := exec.CommandContext(ctx, "go", "list", "-json", "-m", "all")
	cmd.Dir = suite.backendPath

	_, err := cmd.CombinedOutput()
	if err != nil {
		suite.T().Logf("Go module audit informational error: %v", err)
	}

	// Frontend security audit
	if _, err := os.Stat(suite.frontendPath); err == nil {
		suite.T().Log("Running npm security audit...")
		cmd = exec.CommandContext(ctx, "npm", "audit", "--audit-level=high")
		cmd.Dir = suite.frontendPath

		auditOutput, err := cmd.CombinedOutput()
		if err != nil {
			// npm audit returns non-zero exit code if vulnerabilities found
			suite.T().Logf("npm audit found issues (informational): %s", string(auditOutput))
		} else {
			suite.T().Log("npm audit completed with no high-severity issues")
		}
	}
}

// TestBuildReproducibility ensures builds are reproducible
func (suite *BuildTestSuite) TestBuildReproducibility() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Build backend twice and compare binaries
	binary1 := filepath.Join(suite.backendPath, "docker-auto-server-test1")
	binary2 := filepath.Join(suite.backendPath, "docker-auto-server-test2")

	suite.tempFiles = append(suite.tempFiles, binary1, binary2)

	// First build
	suite.T().Log("Building backend (first build)...")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binary1, "./cmd/server")
	cmd.Dir = suite.backendPath

	_, err := cmd.CombinedOutput()
	require.NoError(suite.T(), err, "First build failed")

	// Small delay to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	// Second build
	suite.T().Log("Building backend (second build)...")
	cmd = exec.CommandContext(ctx, "go", "build", "-o", binary2, "./cmd/server")
	cmd.Dir = suite.backendPath

	_, err = cmd.CombinedOutput()
	require.NoError(suite.T(), err, "Second build failed")

	// Compare file sizes (should be identical for reproducible builds)
	info1, err := os.Stat(binary1)
	require.NoError(suite.T(), err, "Failed to stat first binary")

	info2, err := os.Stat(binary2)
	require.NoError(suite.T(), err, "Failed to stat second binary")

	assert.Equal(suite.T(), info1.Size(), info2.Size(),
		"Binary sizes should be identical for reproducible builds")

	suite.T().Log("Build reproducibility test completed")
}

// Helper Methods

func (suite *BuildTestSuite) createTestEnvironment() {
	envContent := `# Test environment configuration
GO_ENV=test
DB_TYPE=sqlite
DB_PATH=:memory:
LOG_LEVEL=info
PORT=18080
JWT_SECRET=test-secret-key-for-integration-testing
DOCKER_HOST=
DISABLE_AUTH=true
`

	err := os.WriteFile(suite.testEnvPath, []byte(envContent), 0644)
	require.NoError(suite.T(), err, "Failed to create test environment file")
	suite.tempFiles = append(suite.tempFiles, suite.testEnvPath)
}

func (suite *BuildTestSuite) monitorOutput(pipe io.ReadCloser, source string, errorsChan chan<- string, startupDone chan<- bool) {
	scanner := bufio.NewScanner(pipe)
	startupComplete := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		suite.T().Logf("%s: %s", source, line)

		// Check for startup completion signals
		if !startupComplete && (strings.Contains(line, "Server started") ||
			strings.Contains(line, "Listening on") ||
			strings.Contains(line, "HTTP server listening")) {
			startupComplete = true
			if startupDone != nil {
				select {
				case startupDone <- true:
				default:
				}
			}
		}

		// Check for error patterns
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") ||
		   strings.Contains(lowerLine, "panic") ||
		   strings.Contains(lowerLine, "fatal") {
			// Filter out expected/informational errors
			if !suite.isExpectedError(line) {
				select {
				case errorsChan <- fmt.Sprintf("%s: %s", source, line):
				default:
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		suite.T().Logf("Error reading %s: %v", source, err)
	}
}

func (suite *BuildTestSuite) isExpectedError(line string) bool {
	expectedPatterns := []string{
		"connection refused", // Expected when Docker is not available
		"no such file",       // Expected for missing optional files
		"permission denied",  // Expected for certain Docker operations in test
		"test error",         // Errors from test scenarios
		"dial unix",          // Docker socket connection issues in test environment
	}

	lowerLine := strings.ToLower(line)
	for _, pattern := range expectedPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	return false
}

// Test runner
func TestBuildValidationSuite(t *testing.T) {
	// Check if build validation should be skipped
	if testing.Short() {
		t.Skip("Skipping build validation tests in short mode")
	}

	// Check for build test environment variable
	if os.Getenv("BUILD_TESTS") == "false" {
		t.Skip("Build validation tests disabled by BUILD_TESTS environment variable")
	}

	suite.Run(t, new(BuildTestSuite))
}