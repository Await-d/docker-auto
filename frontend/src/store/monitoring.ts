/**
 * Real-time monitoring store for Docker containers
 */
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { ElMessage } from "element-plus";
import { containerAPI } from "@/api/container";
import { useWebSocket } from "@/utils/websocket";
import type { ResourceMetrics, ContainerStats } from "@/types/container";

interface MonitoringData {
  containerId: string;
  metrics: ResourceMetrics;
  timestamp: Date;
}

interface HistoricalData {
  containerId: string;
  data: ContainerStats[];
  period: string;
  interval: string;
}

export const useMonitoringStore = defineStore("monitoring", () => {
  // State
  const realTimeMetrics = ref<Map<string, ResourceMetrics>>(new Map());
  const historicalData = ref<Map<string, HistoricalData>>(new Map());
  const monitoredContainers = ref<Set<string>>(new Set());
  const loading = ref<Record<string, boolean>>({});
  const refreshInterval = ref(5000); // 5 seconds
  const maxDataPoints = ref(100);

  // Auto-refresh timers
  const refreshTimers = ref<Map<string, NodeJS.Timeout>>(new Map());

  // WebSocket integration
  const wsConnected = ref(false);
  let wsClient: any = null;

  // Computed
  const activeMonitoring = computed(
    () => monitoredContainers.value.size > 0,
  );

  const averageCpuUsage = computed(() => {
    if (realTimeMetrics.value.size === 0) return 0;
    let total = 0;
    realTimeMetrics.value.forEach((metrics) => {
      total += metrics.cpu.usage;
    });
    return total / realTimeMetrics.value.size;
  });

  const averageMemoryUsage = computed(() => {
    if (realTimeMetrics.value.size === 0) return 0;
    let total = 0;
    realTimeMetrics.value.forEach((metrics) => {
      total += metrics.memory.percentage;
    });
    return total / realTimeMetrics.value.size;
  });

  const totalNetworkTraffic = computed(() => {
    let rxTotal = 0;
    let txTotal = 0;
    realTimeMetrics.value.forEach((metrics) => {
      rxTotal += metrics.network.rxBytes;
      txTotal += metrics.network.txBytes;
    });
    return { rx: rxTotal, tx: txTotal };
  });

  // Actions
  async function startMonitoring(
    containerId: string,
    options: {
      enableRealTime?: boolean;
      enableHistorical?: boolean;
      historicalPeriod?: string;
      historicalInterval?: string;
    } = {},
  ) {
    const {
      enableRealTime = true,
      enableHistorical = false,
      historicalPeriod = "1h",
      historicalInterval = "1m",
    } = options;

    if (monitoredContainers.value.has(containerId)) {
      console.warn(`Already monitoring container ${containerId}`);
      return;
    }

    monitoredContainers.value.add(containerId);
    loading.value[containerId] = true;

    try {
      // Initial data fetch
      if (enableRealTime) {
        await fetchRealTimeMetrics(containerId);
        startRealTimeRefresh(containerId);
      }

      if (enableHistorical) {
        await fetchHistoricalData(
          containerId,
          historicalPeriod,
          historicalInterval,
        );
      }

      // Subscribe to WebSocket updates
      if (wsClient && wsConnected.value) {
        subscribeToContainerMetrics(containerId);
      }

      console.log(`Started monitoring container ${containerId}`);
    } catch (error) {
      console.error(`Failed to start monitoring for ${containerId}:`, error);
      monitoredContainers.value.delete(containerId);
      ElMessage.error(`监控启动失败: ${containerId}`);
      throw error;
    } finally {
      loading.value[containerId] = false;
    }
  }

  async function stopMonitoring(containerId: string) {
    if (!monitoredContainers.value.has(containerId)) {
      return;
    }

    // Stop real-time refresh
    const timer = refreshTimers.value.get(containerId);
    if (timer) {
      clearInterval(timer);
      refreshTimers.value.delete(containerId);
    }

    // Unsubscribe from WebSocket updates
    if (wsClient && wsConnected.value) {
      unsubscribeFromContainerMetrics(containerId);
    }

    // Clean up data
    realTimeMetrics.value.delete(containerId);
    historicalData.value.delete(containerId);
    monitoredContainers.value.delete(containerId);
    delete loading.value[containerId];

    console.log(`Stopped monitoring container ${containerId}`);
  }

  async function fetchRealTimeMetrics(containerId: string) {
    try {
      const metrics = await containerAPI.getStats(containerId);
      realTimeMetrics.value.set(containerId, {
        ...metrics,
        timestamp: new Date(),
      });
      return metrics;
    } catch (error) {
      console.error(`Failed to fetch metrics for ${containerId}:`, error);
      throw error;
    }
  }

  async function fetchHistoricalData(
    containerId: string,
    period = "1h",
    interval = "1m",
  ) {
    try {
      const data = await containerAPI.getHistoricalStats(
        containerId,
        period,
        interval,
      );
      historicalData.value.set(containerId, {
        containerId,
        data,
        period,
        interval,
      });
      return data;
    } catch (error) {
      console.error(
        `Failed to fetch historical data for ${containerId}:`,
        error,
      );
      throw error;
    }
  }

  function startRealTimeRefresh(containerId: string) {
    // Clear any existing timer
    const existingTimer = refreshTimers.value.get(containerId);
    if (existingTimer) {
      clearInterval(existingTimer);
    }

    // Start new timer
    const timer = setInterval(async () => {
      if (monitoredContainers.value.has(containerId)) {
        try {
          await fetchRealTimeMetrics(containerId);
        } catch (error) {
          console.error(
            `Real-time refresh failed for ${containerId}:`,
            error,
          );
          // Continue refreshing - don't stop on error
        }
      }
    }, refreshInterval.value);

    refreshTimers.value.set(containerId, timer);
  }

  function updateRealTimeMetrics(containerId: string, metrics: ResourceMetrics) {
    realTimeMetrics.value.set(containerId, {
      ...metrics,
      timestamp: new Date(),
    });

    // Update historical data if available
    const historical = historicalData.value.get(containerId);
    if (historical) {
      historical.data.push({
        container: containerId,
        metrics,
        timestamp: new Date(),
      });

      // Limit data points to prevent memory issues
      if (historical.data.length > maxDataPoints.value) {
        historical.data = historical.data.slice(-maxDataPoints.value);
      }
    }
  }

  function getRealTimeMetrics(containerId: string): ResourceMetrics | null {
    return realTimeMetrics.value.get(containerId) || null;
  }

  function getHistoricalData(containerId: string): ContainerStats[] | null {
    const historical = historicalData.value.get(containerId);
    return historical?.data || null;
  }

  function getMetricsForTimeRange(
    containerId: string,
    startTime: Date,
    endTime: Date,
  ): ContainerStats[] {
    const historical = historicalData.value.get(containerId);
    if (!historical) return [];

    return historical.data.filter(
      (stat) =>
        stat.timestamp >= startTime && stat.timestamp <= endTime,
    );
  }

  function isMonitored(containerId: string): boolean {
    return monitoredContainers.value.has(containerId);
  }

  function isLoading(containerId: string): boolean {
    return loading.value[containerId] || false;
  }

  // WebSocket management
  function initializeWebSocket(baseUrl: string, token: string) {
    try {
      const { client, isConnected } = useWebSocket(baseUrl, token, {
        autoReconnect: true,
        reconnectInterval: 1000,
        maxReconnectAttempts: 10,
      });

      wsClient = client;

      // Watch connection state
      const stopWatching = client.onStateChange((state) => {
        wsConnected.value = state === "connected";

        if (state === "connected") {
          // Resubscribe to all monitored containers
          monitoredContainers.value.forEach((containerId) => {
            subscribeToContainerMetrics(containerId);
          });
        }
      });

      return () => {
        stopWatching();
        client.disconnect();
      };
    } catch (error) {
      console.error("Failed to initialize WebSocket:", error);
      return () => {};
    }
  }

  function subscribeToContainerMetrics(containerId: string) {
    if (!wsClient) return;

    // Subscribe to container stats updates
    wsClient.subscribe(
      `container.${containerId}.stats`,
      (data: any) => {
        if (data.metrics) {
          updateRealTimeMetrics(containerId, data.metrics);
        }
      },
    );

    // Subscribe to container events
    wsClient.subscribe(
      `container.${containerId}.events`,
      (data: any) => {
        console.log(`Container ${containerId} event:`, data);
      },
    );
  }

  function unsubscribeFromContainerMetrics(containerId: string) {
    if (!wsClient) return;

    wsClient.unsubscribe(`container.${containerId}.stats`);
    wsClient.unsubscribe(`container.${containerId}.events`);
  }

  // Bulk operations
  async function startMonitoringMultiple(
    containerIds: string[],
    options?: Parameters<typeof startMonitoring>[1],
  ) {
    const results = await Promise.allSettled(
      containerIds.map((id) => startMonitoring(id, options)),
    );

    const successful = results.filter(
      (result) => result.status === "fulfilled",
    ).length;
    const failed = results.length - successful;

    if (failed > 0) {
      ElMessage.warning(`${successful}/${results.length} 个容器监控启动成功`);
    } else {
      ElMessage.success(`成功启动 ${successful} 个容器的监控`);
    }

    return {
      total: results.length,
      successful,
      failed,
      results,
    };
  }

  async function stopMonitoringAll() {
    const containerIds = Array.from(monitoredContainers.value);
    await Promise.all(containerIds.map(stopMonitoring));

    ElMessage.success("已停止所有容器监控");
  }

  function setRefreshInterval(interval: number) {
    refreshInterval.value = interval;

    // Restart all active timers with new interval
    monitoredContainers.value.forEach((containerId) => {
      if (refreshTimers.value.has(containerId)) {
        startRealTimeRefresh(containerId);
      }
    });
  }

  function getMonitoringStats() {
    return {
      monitoredContainers: monitoredContainers.value.size,
      realTimeDataPoints: realTimeMetrics.value.size,
      historicalDataSets: historicalData.value.size,
      activeTimers: refreshTimers.value.size,
      averageCpu: averageCpuUsage.value,
      averageMemory: averageMemoryUsage.value,
      networkTraffic: totalNetworkTraffic.value,
      wsConnected: wsConnected.value,
    };
  }

  // Utility functions
  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  function formatPercentage(value: number): string {
    return `${Math.round(value * 100) / 100}%`;
  }

  function calculateTrend(data: ContainerStats[], field: string): 'up' | 'down' | 'stable' {
    if (data.length < 2) return 'stable';

    const recent = data.slice(-5);
    const first = recent[0];
    const last = recent[recent.length - 1];

    let firstValue: number, lastValue: number;

    switch (field) {
      case 'cpu':
        firstValue = first.metrics.cpu.usage;
        lastValue = last.metrics.cpu.usage;
        break;
      case 'memory':
        firstValue = first.metrics.memory.percentage;
        lastValue = last.metrics.memory.percentage;
        break;
      default:
        return 'stable';
    }

    const diff = lastValue - firstValue;
    const threshold = 5; // 5% change threshold

    if (Math.abs(diff) < threshold) return 'stable';
    return diff > 0 ? 'up' : 'down';
  }

  // Reset store
  function $reset() {
    // Stop all monitoring
    monitoredContainers.value.forEach((containerId) => {
      stopMonitoring(containerId);
    });

    // Clear all data
    realTimeMetrics.value.clear();
    historicalData.value.clear();
    monitoredContainers.value.clear();
    refreshTimers.value.clear();

    // Reset state
    loading.value = {};
    refreshInterval.value = 5000;
    maxDataPoints.value = 100;
    wsConnected.value = false;
    wsClient = null;
  }

  return {
    // State
    realTimeMetrics,
    historicalData,
    monitoredContainers,
    loading,
    refreshInterval,
    maxDataPoints,
    wsConnected,

    // Computed
    activeMonitoring,
    averageCpuUsage,
    averageMemoryUsage,
    totalNetworkTraffic,

    // Actions
    startMonitoring,
    stopMonitoring,
    fetchRealTimeMetrics,
    fetchHistoricalData,
    updateRealTimeMetrics,
    getRealTimeMetrics,
    getHistoricalData,
    getMetricsForTimeRange,
    isMonitored,
    isLoading,

    // WebSocket
    initializeWebSocket,
    subscribeToContainerMetrics,
    unsubscribeFromContainerMetrics,

    // Bulk operations
    startMonitoringMultiple,
    stopMonitoringAll,

    // Configuration
    setRefreshInterval,
    getMonitoringStats,

    // Utilities
    formatBytes,
    formatPercentage,
    calculateTrend,

    // Reset
    $reset,
  };
});