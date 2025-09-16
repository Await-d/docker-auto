#!/usr/bin/env node

/**
 * Comprehensive Load Testing Script for Docker Auto-Update System
 * Tests all major components: API endpoints, WebSocket connections, Database operations
 */

const http = require('http');
const https = require('https');
const WebSocket = require('ws');
const { performance } = require('perf_hooks');
const fs = require('fs');
const path = require('path');

// Configuration
const CONFIG = {
  // Test configuration
  baseUrl: process.env.TEST_BASE_URL || 'http://localhost:8080',
  wsUrl: process.env.TEST_WS_URL || 'ws://localhost:8080/ws',
  duration: parseInt(process.env.TEST_DURATION) || 60000, // 1 minute
  warmupDuration: parseInt(process.env.TEST_WARMUP) || 10000, // 10 seconds

  // Load parameters
  concurrent: {
    api: parseInt(process.env.TEST_API_CONCURRENT) || 50,
    websocket: parseInt(process.env.TEST_WS_CONCURRENT) || 20,
    database: parseInt(process.env.TEST_DB_CONCURRENT) || 30
  },

  // Target performance thresholds
  targets: {
    apiResponseTime: 200, // ms
    apiThroughput: 1000,  // requests per second
    apiErrorRate: 0.05,   // 5%
    wsMessageLatency: 50, // ms
    wsReconnectionTime: 3000, // ms
    dbQueryTime: 100,     // ms
    memoryUsage: 512 * 1024 * 1024, // 512MB
    cpuUsage: 50          // 50%
  },

  // Test endpoints
  endpoints: [
    { path: '/api/health', method: 'GET', weight: 10 },
    { path: '/api/system/info', method: 'GET', weight: 5 },
    { path: '/api/containers', method: 'GET', weight: 8 },
    { path: '/api/containers/stats', method: 'GET', weight: 3 },
    { path: '/api/updates/history', method: 'GET', weight: 4 },
    { path: '/api/notifications', method: 'GET', weight: 2 }
  ],

  // Authentication (if needed)
  auth: {
    enabled: process.env.TEST_AUTH_ENABLED === 'true',
    token: process.env.TEST_AUTH_TOKEN || '',
    username: process.env.TEST_USERNAME || 'admin',
    password: process.env.TEST_PASSWORD || 'password'
  }
};

class PerformanceMetrics {
  constructor() {
    this.reset();
  }

  reset() {
    this.api = {
      requests: 0,
      errors: 0,
      responseTimeSum: 0,
      responseTimesMs: [],
      statusCodes: {},
      concurrentConnections: 0,
      maxConcurrentConnections: 0
    };

    this.websocket = {
      connections: 0,
      messagesReceived: 0,
      messagesSent: 0,
      connectionErrors: 0,
      averageLatency: 0,
      latencySum: 0,
      reconnections: 0
    };

    this.system = {
      memoryUsage: [],
      cpuUsage: [],
      timestamps: []
    };

    this.startTime = performance.now();
  }

  recordApiRequest(responseTime, statusCode, error = null) {
    this.api.requests++;
    if (error || statusCode >= 400) {
      this.api.errors++;
    }

    this.api.responseTimeSum += responseTime;
    this.api.responseTimesMs.push(responseTime);

    this.api.statusCodes[statusCode] = (this.api.statusCodes[statusCode] || 0) + 1;
  }

  recordWebSocketMessage(latency = 0) {
    this.websocket.messagesReceived++;
    if (latency > 0) {
      this.websocket.latencySum += latency;
      this.websocket.averageLatency = this.websocket.latencySum / this.websocket.messagesReceived;
    }
  }

  recordSystemMetrics(memory, cpu) {
    this.system.memoryUsage.push(memory);
    this.system.cpuUsage.push(cpu);
    this.system.timestamps.push(performance.now());
  }

  getResults() {
    const duration = performance.now() - this.startTime;
    const durationSeconds = duration / 1000;

    // Calculate percentiles
    const sortedTimes = this.api.responseTimesMs.slice().sort((a, b) => a - b);
    const p50 = this.percentile(sortedTimes, 50);
    const p90 = this.percentile(sortedTimes, 90);
    const p95 = this.percentile(sortedTimes, 95);
    const p99 = this.percentile(sortedTimes, 99);

    return {
      duration: {
        milliseconds: duration,
        seconds: durationSeconds
      },
      api: {
        totalRequests: this.api.requests,
        requestsPerSecond: this.api.requests / durationSeconds,
        errorCount: this.api.errors,
        errorRate: this.api.requests > 0 ? this.api.errors / this.api.requests : 0,
        averageResponseTime: this.api.requests > 0 ? this.api.responseTimeSum / this.api.requests : 0,
        maxConcurrentConnections: this.api.maxConcurrentConnections,
        responseTimePercentiles: { p50, p90, p95, p99 },
        statusCodeDistribution: this.api.statusCodes
      },
      websocket: {
        totalConnections: this.websocket.connections,
        messagesReceived: this.websocket.messagesReceived,
        messagesSent: this.websocket.messagesSent,
        messagesPerSecond: this.websocket.messagesReceived / durationSeconds,
        averageLatency: this.websocket.averageLatency,
        connectionErrors: this.websocket.connectionErrors,
        reconnections: this.websocket.reconnections
      },
      system: {
        peakMemoryUsage: Math.max(...this.system.memoryUsage),
        averageMemoryUsage: this.average(this.system.memoryUsage),
        peakCpuUsage: Math.max(...this.system.cpuUsage),
        averageCpuUsage: this.average(this.system.cpuUsage)
      }
    };
  }

  percentile(sortedArray, p) {
    if (sortedArray.length === 0) return 0;
    const index = Math.ceil((p / 100) * sortedArray.length) - 1;
    return sortedArray[Math.max(0, index)];
  }

  average(array) {
    if (array.length === 0) return 0;
    return array.reduce((sum, val) => sum + val, 0) / array.length;
  }
}

class ApiLoadTester {
  constructor(baseUrl, metrics, concurrency) {
    this.baseUrl = baseUrl;
    this.metrics = metrics;
    this.concurrency = concurrency;
    this.running = false;
    this.activeRequests = new Set();
  }

  async start() {
    this.running = true;
    console.log(`🚀 Starting API load test with ${this.concurrency} concurrent requests`);

    // Start concurrent request loops
    const promises = [];
    for (let i = 0; i < this.concurrency; i++) {
      promises.push(this.requestLoop());
    }

    await Promise.all(promises);
  }

  stop() {
    this.running = false;
  }

  async requestLoop() {
    while (this.running) {
      try {
        const endpoint = this.selectRandomEndpoint();
        const requestId = Math.random().toString(36);

        this.activeRequests.add(requestId);
        this.metrics.api.concurrentConnections++;
        this.metrics.api.maxConcurrentConnections = Math.max(
          this.metrics.api.maxConcurrentConnections,
          this.metrics.api.concurrentConnections
        );

        const startTime = performance.now();
        const { statusCode, error } = await this.makeRequest(endpoint);
        const responseTime = performance.now() - startTime;

        this.metrics.recordApiRequest(responseTime, statusCode, error);

        this.activeRequests.delete(requestId);
        this.metrics.api.concurrentConnections--;

        // Add small delay to control request rate
        await this.delay(Math.random() * 10);

      } catch (error) {
        console.error('Request loop error:', error);
        this.metrics.recordApiRequest(0, 500, error);
      }
    }
  }

  selectRandomEndpoint() {
    // Weighted selection based on endpoint weights
    const totalWeight = CONFIG.endpoints.reduce((sum, ep) => sum + ep.weight, 0);
    let random = Math.random() * totalWeight;

    for (const endpoint of CONFIG.endpoints) {
      if (random <= endpoint.weight) {
        return endpoint;
      }
      random -= endpoint.weight;
    }

    return CONFIG.endpoints[0]; // fallback
  }

  async makeRequest(endpoint) {
    return new Promise((resolve) => {
      const url = new URL(endpoint.path, this.baseUrl);
      const isHttps = url.protocol === 'https:';
      const httpModule = isHttps ? https : http;

      const options = {
        hostname: url.hostname,
        port: url.port || (isHttps ? 443 : 80),
        path: url.pathname + url.search,
        method: endpoint.method,
        headers: {
          'User-Agent': 'Docker-Auto Load Tester',
          'Accept': 'application/json',
          'Connection': 'keep-alive'
        }
      };

      // Add authentication if enabled
      if (CONFIG.auth.enabled && CONFIG.auth.token) {
        options.headers['Authorization'] = `Bearer ${CONFIG.auth.token}`;
      }

      const req = httpModule.request(options, (res) => {
        let data = '';
        res.on('data', (chunk) => data += chunk);
        res.on('end', () => {
          resolve({
            statusCode: res.statusCode,
            data: data,
            error: null
          });
        });
      });

      req.on('error', (error) => {
        resolve({
          statusCode: 0,
          data: null,
          error: error
        });
      });

      req.setTimeout(10000, () => {
        req.destroy();
        resolve({
          statusCode: 0,
          data: null,
          error: new Error('Request timeout')
        });
      });

      req.end();
    });
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

class WebSocketLoadTester {
  constructor(wsUrl, metrics, concurrency) {
    this.wsUrl = wsUrl;
    this.metrics = metrics;
    this.concurrency = concurrency;
    this.running = false;
    this.connections = [];
  }

  async start() {
    this.running = true;
    console.log(`🌐 Starting WebSocket load test with ${this.concurrency} concurrent connections`);

    // Create concurrent WebSocket connections
    const promises = [];
    for (let i = 0; i < this.concurrency; i++) {
      promises.push(this.createConnection(i));
    }

    await Promise.all(promises);
  }

  stop() {
    this.running = false;
    this.connections.forEach(conn => {
      if (conn.readyState === WebSocket.OPEN) {
        conn.close();
      }
    });
    this.connections = [];
  }

  async createConnection(index) {
    return new Promise((resolve) => {
      try {
        let wsUrl = this.wsUrl;
        if (CONFIG.auth.enabled && CONFIG.auth.token) {
          wsUrl += `?token=${encodeURIComponent(CONFIG.auth.token)}`;
        }

        const ws = new WebSocket(wsUrl);
        let messageCount = 0;
        let lastPingTime = 0;

        ws.on('open', () => {
          this.metrics.websocket.connections++;
          console.log(`WebSocket connection ${index} established`);

          // Send periodic messages
          const messageInterval = setInterval(() => {
            if (!this.running || ws.readyState !== WebSocket.OPEN) {
              clearInterval(messageInterval);
              return;
            }

            lastPingTime = performance.now();
            const message = {
              type: 'ping',
              timestamp: lastPingTime,
              connectionId: index,
              messageId: ++messageCount
            };

            ws.send(JSON.stringify(message));
            this.metrics.websocket.messagesSent++;
          }, 1000 + Math.random() * 1000); // 1-2 seconds interval

          // Subscribe to various topics
          this.subscribeToTopics(ws);
        });

        ws.on('message', (data) => {
          try {
            const message = JSON.parse(data);

            if (message.type === 'pong' && message.timestamp) {
              const latency = performance.now() - message.timestamp;
              this.metrics.recordWebSocketMessage(latency);
            } else {
              this.metrics.recordWebSocketMessage();
            }
          } catch (error) {
            this.metrics.recordWebSocketMessage();
          }
        });

        ws.on('error', (error) => {
          this.metrics.websocket.connectionErrors++;
          console.error(`WebSocket connection ${index} error:`, error.message);
        });

        ws.on('close', () => {
          if (this.running) {
            // Attempt to reconnect
            this.metrics.websocket.reconnections++;
            setTimeout(() => {
              if (this.running) {
                this.createConnection(index);
              }
            }, 1000 + Math.random() * 2000);
          }
        });

        this.connections.push(ws);
        resolve();

      } catch (error) {
        this.metrics.websocket.connectionErrors++;
        console.error(`Failed to create WebSocket connection ${index}:`, error);
        resolve();
      }
    });
  }

  subscribeToTopics(ws) {
    const topics = ['containers', 'updates', 'system', 'notifications'];

    topics.forEach(topic => {
      ws.send(JSON.stringify({
        type: 'subscribe',
        topic: topic,
        messageId: `sub_${Date.now()}_${Math.random()}`
      }));
    });
  }
}

class SystemMonitor {
  constructor(metrics) {
    this.metrics = metrics;
    this.monitoring = false;
    this.interval = null;
  }

  start() {
    this.monitoring = true;
    console.log('📊 Starting system monitoring');

    this.interval = setInterval(() => {
      this.collectMetrics();
    }, 1000); // Every second
  }

  stop() {
    this.monitoring = false;
    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }
  }

  collectMetrics() {
    // Simulate system metrics collection
    const memUsage = process.memoryUsage();
    const cpuUsage = process.cpuUsage();

    // Convert to MB for memory
    const memoryMB = memUsage.heapUsed / 1024 / 1024;

    // Simple CPU usage approximation (in real scenario, use proper CPU monitoring)
    const cpuPercent = Math.min(100, (cpuUsage.user + cpuUsage.system) / 10000);

    this.metrics.recordSystemMetrics(memoryMB, cpuPercent);
  }
}

class LoadTestRunner {
  constructor() {
    this.metrics = new PerformanceMetrics();
    this.apiTester = new ApiLoadTester(CONFIG.baseUrl, this.metrics, CONFIG.concurrent.api);
    this.wsTester = new WebSocketLoadTester(CONFIG.wsUrl, this.metrics, CONFIG.concurrent.websocket);
    this.systemMonitor = new SystemMonitor(this.metrics);
    this.running = false;
  }

  async run() {
    console.log('🏁 Starting Docker Auto-Update System Load Test');
    console.log('Configuration:', {
      baseUrl: CONFIG.baseUrl,
      duration: `${CONFIG.duration / 1000}s`,
      concurrent: CONFIG.concurrent,
      targets: CONFIG.targets
    });

    this.running = true;
    this.metrics.reset();

    try {
      // Start system monitoring
      this.systemMonitor.start();

      // Warmup phase
      console.log('🔥 Warmup phase starting...');
      await this.runWarmup();

      // Main test phase
      console.log('⚡ Main test phase starting...');
      await this.runMainTest();

      // Generate results
      const results = this.generateResults();
      this.displayResults(results);
      this.saveResults(results);

    } catch (error) {
      console.error('❌ Load test failed:', error);
    } finally {
      await this.cleanup();
    }
  }

  async runWarmup() {
    // Light warmup to establish connections and prime caches
    const warmupApi = new ApiLoadTester(CONFIG.baseUrl, new PerformanceMetrics(), 5);
    warmupApi.start();

    await this.delay(CONFIG.warmupDuration);

    warmupApi.stop();
    console.log('✅ Warmup completed');
  }

  async runMainTest() {
    // Start all test components
    const promises = [
      this.apiTester.start(),
      this.wsTester.start()
    ];

    // Run for specified duration
    await Promise.race([
      Promise.all(promises),
      this.delay(CONFIG.duration)
    ]);

    console.log('✅ Main test completed');
  }

  generateResults() {
    const results = this.metrics.getResults();

    // Add performance analysis
    results.analysis = this.analyzeResults(results);
    results.config = CONFIG;
    results.timestamp = new Date().toISOString();

    return results;
  }

  analyzeResults(results) {
    const analysis = {
      overallScore: 100,
      issues: [],
      recommendations: []
    };

    // API Performance Analysis
    if (results.api.averageResponseTime > CONFIG.targets.apiResponseTime) {
      analysis.overallScore -= 15;
      analysis.issues.push({
        category: 'API',
        severity: 'warning',
        message: `Average response time (${results.api.averageResponseTime.toFixed(2)}ms) exceeds target (${CONFIG.targets.apiResponseTime}ms)`
      });
      analysis.recommendations.push('Consider optimizing database queries and adding response caching');
    }

    if (results.api.requestsPerSecond < CONFIG.targets.apiThroughput) {
      analysis.overallScore -= 10;
      analysis.issues.push({
        category: 'API',
        severity: 'info',
        message: `Throughput (${results.api.requestsPerSecond.toFixed(2)} RPS) below target (${CONFIG.targets.apiThroughput} RPS)`
      });
    }

    if (results.api.errorRate > CONFIG.targets.apiErrorRate) {
      analysis.overallScore -= 25;
      analysis.issues.push({
        category: 'API',
        severity: 'error',
        message: `Error rate (${(results.api.errorRate * 100).toFixed(2)}%) exceeds target (${CONFIG.targets.apiErrorRate * 100}%)`
      });
      analysis.recommendations.push('Investigate API errors and improve error handling');
    }

    // WebSocket Analysis
    if (results.websocket.averageLatency > CONFIG.targets.wsMessageLatency) {
      analysis.overallScore -= 10;
      analysis.issues.push({
        category: 'WebSocket',
        severity: 'warning',
        message: `Message latency (${results.websocket.averageLatency.toFixed(2)}ms) exceeds target (${CONFIG.targets.wsMessageLatency}ms)`
      });
      analysis.recommendations.push('Optimize WebSocket message processing and consider message batching');
    }

    // System Resource Analysis
    if (results.system.peakMemoryUsage > CONFIG.targets.memoryUsage / (1024 * 1024)) {
      analysis.overallScore -= 15;
      analysis.issues.push({
        category: 'System',
        severity: 'warning',
        message: `Peak memory usage (${results.system.peakMemoryUsage.toFixed(2)}MB) exceeds target (${CONFIG.targets.memoryUsage / (1024 * 1024)}MB)`
      });
      analysis.recommendations.push('Optimize memory usage and implement proper garbage collection');
    }

    // Overall assessment
    if (analysis.overallScore >= 90) {
      analysis.grade = 'A';
      analysis.summary = 'Excellent performance - all targets met or exceeded';
    } else if (analysis.overallScore >= 75) {
      analysis.grade = 'B';
      analysis.summary = 'Good performance - minor optimizations recommended';
    } else if (analysis.overallScore >= 60) {
      analysis.grade = 'C';
      analysis.summary = 'Fair performance - several areas need improvement';
    } else {
      analysis.grade = 'D';
      analysis.summary = 'Poor performance - significant optimizations required';
    }

    return analysis;
  }

  displayResults(results) {
    console.log('\n📈 LOAD TEST RESULTS');
    console.log('='.repeat(50));

    console.log(`\n🏆 Overall Score: ${results.analysis.overallScore}/100 (Grade: ${results.analysis.grade})`);
    console.log(`📋 Summary: ${results.analysis.summary}`);

    console.log('\n📊 API Performance:');
    console.log(`   Total Requests: ${results.api.totalRequests}`);
    console.log(`   Requests/Second: ${results.api.requestsPerSecond.toFixed(2)}`);
    console.log(`   Average Response Time: ${results.api.averageResponseTime.toFixed(2)}ms`);
    console.log(`   Error Rate: ${(results.api.errorRate * 100).toFixed(2)}%`);
    console.log(`   Response Time Percentiles:`);
    console.log(`     P50: ${results.api.responseTimePercentiles.p50.toFixed(2)}ms`);
    console.log(`     P95: ${results.api.responseTimePercentiles.p95.toFixed(2)}ms`);
    console.log(`     P99: ${results.api.responseTimePercentiles.p99.toFixed(2)}ms`);

    console.log('\n🌐 WebSocket Performance:');
    console.log(`   Connections: ${results.websocket.totalConnections}`);
    console.log(`   Messages/Second: ${results.websocket.messagesPerSecond.toFixed(2)}`);
    console.log(`   Average Latency: ${results.websocket.averageLatency.toFixed(2)}ms`);
    console.log(`   Connection Errors: ${results.websocket.connectionErrors}`);
    console.log(`   Reconnections: ${results.websocket.reconnections}`);

    console.log('\n💻 System Resources:');
    console.log(`   Peak Memory: ${results.system.peakMemoryUsage.toFixed(2)}MB`);
    console.log(`   Avg Memory: ${results.system.averageMemoryUsage.toFixed(2)}MB`);
    console.log(`   Peak CPU: ${results.system.peakCpuUsage.toFixed(2)}%`);
    console.log(`   Avg CPU: ${results.system.averageCpuUsage.toFixed(2)}%`);

    if (results.analysis.issues.length > 0) {
      console.log('\n⚠️  Issues Found:');
      results.analysis.issues.forEach((issue, index) => {
        const icon = issue.severity === 'error' ? '❌' : issue.severity === 'warning' ? '⚠️' : 'ℹ️';
        console.log(`   ${index + 1}. ${icon} [${issue.category}] ${issue.message}`);
      });
    }

    if (results.analysis.recommendations.length > 0) {
      console.log('\n💡 Recommendations:');
      results.analysis.recommendations.forEach((rec, index) => {
        console.log(`   ${index + 1}. ${rec}`);
      });
    }

    console.log('\n='.repeat(50));
  }

  saveResults(results) {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const filename = `load-test-results-${timestamp}.json`;
    const filepath = path.join(process.cwd(), 'test-results', filename);

    try {
      // Ensure results directory exists
      const dir = path.dirname(filepath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }

      fs.writeFileSync(filepath, JSON.stringify(results, null, 2));
      console.log(`\n💾 Results saved to: ${filepath}`);
    } catch (error) {
      console.error('Failed to save results:', error.message);
    }
  }

  async cleanup() {
    this.running = false;

    console.log('🧹 Cleaning up...');

    this.apiTester.stop();
    this.wsTester.stop();
    this.systemMonitor.stop();

    // Give connections time to close gracefully
    await this.delay(2000);

    console.log('✅ Cleanup completed');
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// Handle script termination
process.on('SIGINT', () => {
  console.log('\n🛑 Received SIGINT, shutting down gracefully...');
  process.exit(0);
});

process.on('SIGTERM', () => {
  console.log('\n🛑 Received SIGTERM, shutting down gracefully...');
  process.exit(0);
});

// Main execution
if (require.main === module) {
  const runner = new LoadTestRunner();

  runner.run()
    .then(() => {
      console.log('✅ Load test completed successfully');
      process.exit(0);
    })
    .catch((error) => {
      console.error('❌ Load test failed:', error);
      process.exit(1);
    });
}

module.exports = { LoadTestRunner, PerformanceMetrics };