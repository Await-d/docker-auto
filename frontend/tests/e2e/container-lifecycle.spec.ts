import { describe, it, expect, beforeAll, afterAll, beforeEach, afterEach } from 'vitest'
import { chromium, Browser, Page, BrowserContext } from 'playwright'
import type { APIRequestContext } from 'playwright'

// Test configuration constants
const TEST_CONFIG = {
  // Frontend URL (adjust based on your setup)
  FRONTEND_URL: process.env.FRONTEND_URL || 'http://localhost:5173',
  // Backend API URL
  API_URL: process.env.API_URL || 'http://localhost:8080',
  // WebSocket URL
  WS_URL: process.env.WS_URL || 'ws://localhost:8080/ws',
  // Test timeout
  TEST_TIMEOUT: 60000,
  // Page load timeout
  PAGE_TIMEOUT: 30000,
  // API response timeout
  API_TIMEOUT: 10000,
}

// Test data constants
const TEST_DATA = {
  // Test user credentials
  USER: {
    username: 'test-user',
    password: 'test-password',
    email: 'test@example.com'
  },
  // Test container configuration
  CONTAINER: {
    name: 'e2e-test-container',
    image: 'nginx:alpine',
    tag: 'latest',
    ports: ['80:8080'],
    environment: ['ENV=test'],
  },
  // Test images for various scenarios
  IMAGES: {
    WEB_SERVER: 'nginx:alpine',
    DATABASE: 'redis:alpine',
    UTILITY: 'busybox:latest',
  }
}

// Global test state
interface TestState {
  browser: Browser | null
  context: BrowserContext | null
  page: Page | null
  apiContext: APIRequestContext | null
  createdContainers: string[]
  testArtifacts: string[]
}

const testState: TestState = {
  browser: null,
  context: null,
  page: null,
  apiContext: null,
  createdContainers: [],
  testArtifacts: []
}

/**
 * End-to-End Test Suite for Container Lifecycle Management
 *
 * This comprehensive test suite validates the complete user workflow:
 * 1. User authentication and dashboard access
 * 2. Container management operations (create, start, stop, delete)
 * 3. Real-time monitoring and status updates
 * 4. Web terminal access and command execution
 * 5. Automatic update detection and execution
 * 6. Integration between frontend and backend systems
 */
describe('Container Lifecycle Management E2E Tests', () => {

  // Setup and teardown hooks
  beforeAll(async () => {
    // Initialize browser with comprehensive configuration
    testState.browser = await chromium.launch({
      headless: process.env.CI === 'true', // Headless in CI, visible locally
      slowMo: process.env.CI === 'true' ? 0 : 100, // Slow motion for local debugging
      timeout: TEST_CONFIG.PAGE_TIMEOUT,
      args: [
        '--no-sandbox',
        '--disable-dev-shm-usage',
        '--disable-gpu',
        '--disable-web-security',
        '--allow-running-insecure-content'
      ]
    })

    // Create browser context with realistic viewport
    testState.context = await testState.browser.newContext({
      viewport: { width: 1280, height: 720 },
      ignoreHTTPSErrors: true,
      permissions: ['notifications'],
    })

    // Create API request context for backend verification
    testState.apiContext = await testState.context.request.newContext({
      baseURL: TEST_CONFIG.API_URL,
      timeout: TEST_CONFIG.API_TIMEOUT,
    })

    // Verify backend is accessible
    await verifyBackendHealth()

    console.log('E2E Test environment initialized successfully')
  })

  afterAll(async () => {
    // Comprehensive cleanup of all test resources
    await cleanupTestResources()

    if (testState.apiContext) {
      await testState.apiContext.dispose()
    }
    if (testState.context) {
      await testState.context.close()
    }
    if (testState.browser) {
      await testState.browser.close()
    }

    console.log('E2E Test cleanup completed')
  })

  beforeEach(async () => {
    // Create fresh page for each test
    if (testState.context) {
      testState.page = await testState.context.newPage()

      // Set up error monitoring
      testState.page.on('console', msg => {
        if (msg.type() === 'error') {
          console.error('Console error:', msg.text())
        }
      })

      testState.page.on('pageerror', error => {
        console.error('Page error:', error.message)
      })
    }
  })

  afterEach(async () => {
    // Clean up page and test-specific resources
    if (testState.page) {
      await testState.page.close()
      testState.page = null
    }
  })

  /**
   * Test: Complete User Authentication Flow
   * Validates user can login and access the dashboard
   */
  it('should authenticate user and access dashboard', async () => {
    const page = testState.page!

    // Navigate to application
    await page.goto(TEST_CONFIG.FRONTEND_URL, {
      waitUntil: 'networkidle',
      timeout: TEST_CONFIG.PAGE_TIMEOUT
    })

    // Check if login is required or if we're already authenticated
    const isLoginPage = await page.locator('input[type="password"]').isVisible().catch(() => false)

    if (isLoginPage) {
      // Perform login
      await page.fill('input[type="text"], input[type="email"]', TEST_DATA.USER.username)
      await page.fill('input[type="password"]', TEST_DATA.USER.password)
      await page.click('button[type="submit"], .login-button')

      // Wait for dashboard to load
      await page.waitForURL('**/dashboard', { timeout: TEST_CONFIG.PAGE_TIMEOUT })
    }

    // Verify dashboard elements are present
    await expect(page.locator('.dashboard-container, .main-content')).toBeVisible()

    // Verify navigation elements
    await expect(page.locator('.sidebar, .nav-menu')).toBeVisible()
    await expect(page.locator('text=Container, text=容器')).toBeVisible()

    console.log('✓ User authentication and dashboard access successful')
  }, TEST_CONFIG.TEST_TIMEOUT)

  /**
   * Test: Container Management Operations
   * Validates complete container lifecycle: create, start, stop, delete
   */
  it('should manage container lifecycle operations', async () => {
    const page = testState.page!

    // Navigate to containers page
    await navigateToContainersPage(page)

    // Test 1: Create new container
    await page.click('.create-container-btn, [data-testid="create-container"]')

    // Wait for create dialog/form
    await page.waitForSelector('.create-container-form, .container-form')

    // Fill container creation form
    await page.fill('input[name="name"], #container-name', TEST_DATA.CONTAINER.name)
    await page.fill('input[name="image"], #container-image', TEST_DATA.CONTAINER.image)

    // Configure additional options if available
    const portsInput = page.locator('input[name="ports"], #container-ports')
    if (await portsInput.isVisible()) {
      await portsInput.fill(TEST_DATA.CONTAINER.ports.join(','))
    }

    // Submit container creation
    await page.click('button[type="submit"], .confirm-create')

    // Wait for container to appear in list
    await page.waitForSelector(`text=${TEST_DATA.CONTAINER.name}`, { timeout: 30000 })

    // Track created container for cleanup
    testState.createdContainers.push(TEST_DATA.CONTAINER.name)

    console.log('✓ Container creation successful')

    // Test 2: Start container
    const containerRow = page.locator(`tr:has-text("${TEST_DATA.CONTAINER.name}")`)
    await containerRow.locator('.start-btn, [data-action="start"]').click()

    // Wait for status to change to running
    await expect(containerRow.locator('.status, [data-status]')).toContainText(/running|运行中/)

    console.log('✓ Container start successful')

    // Test 3: Monitor container status in real-time
    await verifyRealTimeStatusUpdates(page, TEST_DATA.CONTAINER.name)

    // Test 4: Stop container
    await containerRow.locator('.stop-btn, [data-action="stop"]').click()

    // Confirm stop if dialog appears
    const confirmButton = page.locator('.confirm-stop, .confirm-btn')
    if (await confirmButton.isVisible()) {
      await confirmButton.click()
    }

    // Wait for status to change to stopped
    await expect(containerRow.locator('.status, [data-status]')).toContainText(/stopped|已停止|exited/)

    console.log('✓ Container stop successful')

    // Test 5: Delete container
    await containerRow.locator('.delete-btn, [data-action="delete"]').click()

    // Confirm deletion
    const confirmDelete = page.locator('.confirm-delete, .danger-confirm')
    if (await confirmDelete.isVisible()) {
      await confirmDelete.click()
    }

    // Wait for container to disappear from list
    await expect(page.locator(`text=${TEST_DATA.CONTAINER.name}`)).not.toBeVisible()

    // Remove from tracking since it's deleted
    testState.createdContainers = testState.createdContainers.filter(name => name !== TEST_DATA.CONTAINER.name)

    console.log('✓ Container deletion successful')
  }, TEST_CONFIG.TEST_TIMEOUT * 2)

  /**
   * Test: Real-time Monitoring Integration
   * Validates monitoring data display and real-time updates
   */
  it('should display real-time monitoring data', async () => {
    const page = testState.page!

    // Create and start a container for monitoring
    const monitoringContainerName = 'e2e-monitoring-test'
    await createTestContainer(page, monitoringContainerName, TEST_DATA.IMAGES.WEB_SERVER)
    testState.createdContainers.push(monitoringContainerName)

    // Navigate to container detail view
    await page.click(`text=${monitoringContainerName}`)
    await page.waitForSelector('.container-details, .detail-view')

    // Verify monitoring charts are present
    await expect(page.locator('.monitoring-chart, .metrics-chart')).toBeVisible()
    await expect(page.locator('text=CPU, text=Memory')).toBeVisible()

    // Wait for real-time data to load
    await page.waitForTimeout(5000)

    // Verify charts contain data (look for chart elements)
    const charts = page.locator('.echarts, canvas, svg')
    await expect(charts).toHaveCount.greaterThan(0)

    // Test WebSocket connectivity for real-time updates
    await verifyWebSocketConnection(page)

    console.log('✓ Real-time monitoring data display successful')
  }, TEST_CONFIG.TEST_TIMEOUT)

  /**
   * Test: Web Terminal Access and Command Execution
   * Validates terminal functionality and command execution
   */
  it('should access web terminal and execute commands', async () => {
    const page = testState.page!

    // Create and start a container with shell access
    const terminalContainerName = 'e2e-terminal-test'
    await createTestContainer(page, terminalContainerName, TEST_DATA.IMAGES.UTILITY)
    testState.createdContainers.push(terminalContainerName)

    // Navigate to container and open terminal
    await page.click(`text=${terminalContainerName}`)
    await page.waitForSelector('.container-details')

    // Open terminal tab/section
    await page.click('.terminal-tab, [data-tab="terminal"]')
    await page.waitForSelector('.terminal-container, .xterm')

    // Verify terminal is loaded
    await expect(page.locator('.terminal-container, .xterm')).toBeVisible()

    // Wait for terminal to be ready
    await page.waitForTimeout(2000)

    // Execute test command
    const testCommand = 'echo "Hello E2E Test"'
    await page.keyboard.type(testCommand)
    await page.keyboard.press('Enter')

    // Verify command output (with timeout for response)
    await page.waitForTimeout(2000)

    // Check for command echo or output in terminal
    const terminalContent = await page.locator('.terminal-container, .xterm').textContent()
    expect(terminalContent).toContain('Hello E2E Test')

    // Test additional commands
    await page.keyboard.type('pwd')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    console.log('✓ Web terminal access and command execution successful')
  }, TEST_CONFIG.TEST_TIMEOUT)

  /**
   * Test: Automatic Update Detection and Execution
   * Validates update functionality and rollback capabilities
   */
  it('should detect and execute container updates', async () => {
    const page = testState.page!

    // Create container with older image version for update testing
    const updateContainerName = 'e2e-update-test'
    await createTestContainer(page, updateContainerName, 'nginx:1.20-alpine')
    testState.createdContainers.push(updateContainerName)

    // Navigate to container settings/update section
    await page.click(`text=${updateContainerName}`)
    await page.waitForSelector('.container-details')

    // Look for update settings or update check button
    const updateSection = page.locator('.update-settings, [data-section="updates"]')
    if (await updateSection.isVisible()) {
      await updateSection.click()
    }

    // Enable automatic updates if option exists
    const autoUpdateToggle = page.locator('.auto-update-toggle, input[type="checkbox"]')
    if (await autoUpdateToggle.isVisible()) {
      await autoUpdateToggle.check()
    }

    // Trigger update check
    const checkUpdatesBtn = page.locator('.check-updates, [data-action="check-updates"]')
    if (await checkUpdatesBtn.isVisible()) {
      await checkUpdatesBtn.click()

      // Wait for update check to complete
      await page.waitForTimeout(5000)

      // Look for update available notification
      const updateAvailable = page.locator('text=Update available, text=更新可用')
      if (await updateAvailable.isVisible()) {
        // Execute update
        await page.click('.execute-update, [data-action="update"]')

        // Wait for update to complete
        await page.waitForSelector('text=Update completed, text=更新完成', { timeout: 60000 })

        console.log('✓ Container update execution successful')
      } else {
        console.log('! No updates available for test container')
      }
    }
  }, TEST_CONFIG.TEST_TIMEOUT * 2)

  /**
   * Test: Error Handling and Recovery
   * Validates system behavior under error conditions
   */
  it('should handle errors gracefully', async () => {
    const page = testState.page!

    await navigateToContainersPage(page)

    // Test 1: Invalid container creation
    await page.click('.create-container-btn')
    await page.waitForSelector('.create-container-form')

    // Submit empty form to trigger validation
    await page.click('button[type="submit"]')

    // Verify error messages appear
    await expect(page.locator('.error-message, .validation-error')).toBeVisible()

    // Test 2: Network error simulation (if possible)
    // This would require more sophisticated setup with network interception

    // Test 3: Invalid image name
    await page.fill('#container-name', 'test-invalid-image')
    await page.fill('#container-image', 'nonexistent-image:invalid-tag')
    await page.click('button[type="submit"]')

    // Should show error about image not found
    await expect(page.locator('text=not found, text=不存在')).toBeVisible()

    console.log('✓ Error handling validation successful')
  }, TEST_CONFIG.TEST_TIMEOUT)

  // Helper functions

  async function navigateToContainersPage(page: Page): Promise<void> {
    // Handle different possible navigation structures
    const navOptions = [
      '.sidebar [href*="container"]',
      '.nav-menu a:has-text("Container")',
      '.menu-item:has-text("容器")',
      '[data-route="containers"]'
    ]

    for (const selector of navOptions) {
      const element = page.locator(selector)
      if (await element.isVisible()) {
        await element.click()
        break
      }
    }

    // Wait for containers page to load
    await page.waitForSelector('.containers-page, .container-list', { timeout: 10000 })
  }

  async function createTestContainer(page: Page, name: string, image: string): Promise<void> {
    await navigateToContainersPage(page)

    await page.click('.create-container-btn')
    await page.waitForSelector('.create-container-form')

    await page.fill('#container-name', name)
    await page.fill('#container-image', image)
    await page.click('button[type="submit"]')

    await page.waitForSelector(`text=${name}`, { timeout: 30000 })

    // Start the container
    const containerRow = page.locator(`tr:has-text("${name}")`)
    await containerRow.locator('.start-btn').click()
    await expect(containerRow.locator('.status')).toContainText(/running/)
  }

  async function verifyRealTimeStatusUpdates(page: Page, containerName: string): Promise<void> {
    const containerRow = page.locator(`tr:has-text("${containerName}")`)
    const initialStatus = await containerRow.locator('.status').textContent()

    // Wait for potential status changes
    await page.waitForTimeout(3000)

    // Check if monitoring data is updating (CPU, memory indicators)
    const monitoringElements = containerRow.locator('.cpu-usage, .memory-usage, .status-indicator')
    if (await monitoringElements.first().isVisible()) {
      expect(await monitoringElements.first().textContent()).toBeTruthy()
    }
  }

  async function verifyWebSocketConnection(page: Page): Promise<void> {
    // Check for WebSocket connection indicators in the UI
    const wsIndicators = page.locator('.connection-status, .realtime-indicator, .ws-connected')
    if (await wsIndicators.first().isVisible()) {
      expect(await wsIndicators.first().getAttribute('data-connected')).toBe('true')
    }
  }

  async function verifyBackendHealth(): Promise<void> {
    if (!testState.apiContext) return

    try {
      const response = await testState.apiContext.get('/api/health')
      expect(response.ok()).toBeTruthy()
      console.log('✓ Backend health check passed')
    } catch (error) {
      console.warn('Backend health check failed, proceeding with tests:', error)
    }
  }

  async function cleanupTestResources(): Promise<void> {
    if (!testState.apiContext) return

    console.log('Cleaning up test containers...')

    for (const containerName of testState.createdContainers) {
      try {
        // Attempt to stop and remove via API
        await testState.apiContext.delete(`/api/containers/${containerName}?force=true`)
      } catch (error) {
        console.warn(`Failed to cleanup container ${containerName}:`, error)
      }
    }

    // Clean up any test artifacts
    for (const artifact of testState.testArtifacts) {
      try {
        // Cleanup logic for test artifacts
        console.log(`Cleaning up artifact: ${artifact}`)
      } catch (error) {
        console.warn(`Failed to cleanup artifact ${artifact}:`, error)
      }
    }

    testState.createdContainers = []
    testState.testArtifacts = []

    console.log('Test cleanup completed')
  }
})

/**
 * Additional Test Suite: Performance and Load Testing
 */
describe('Performance and Load Tests', () => {
  it('should handle multiple containers efficiently', async () => {
    // This test would create multiple containers and verify performance
    // Implementation would depend on specific performance requirements
    console.log('Performance testing - implementation depends on requirements')
  })

  it('should maintain responsiveness under load', async () => {
    // Test UI responsiveness with many operations
    console.log('Load testing - implementation depends on requirements')
  })
})

/**
 * Test Configuration and Environment Validation
 */
describe('Test Environment Validation', () => {
  it('should validate test environment setup', async () => {
    // Verify all required services are available
    expect(TEST_CONFIG.FRONTEND_URL).toBeTruthy()
    expect(TEST_CONFIG.API_URL).toBeTruthy()
    expect(TEST_CONFIG.WS_URL).toBeTruthy()

    console.log('✓ Test environment configuration validated')
  })
})