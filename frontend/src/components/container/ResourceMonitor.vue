<template>
  <div class="resource-monitor" :class="{ 'detailed-view': detailedView }">
    <!-- 实时统计 -->
    <div class="current-stats">
      <div class="stats-grid">
        <!-- CPU使用情况 -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Monitor /></el-icon>
              CPU使用率
            </div>
            <div class="stat-value">
              {{ formatPercentage(currentStats?.cpu.usage || 0) }}
            </div>
          </div>
          <div class="stat-content">
            <el-progress
              :percentage="currentStats?.cpu.usage || 0"
              :show-text="false"
              :stroke-width="8"
              :color="getResourceColor(currentStats?.cpu.usage || 0)"
            />
            <div class="stat-details">
              <span v-if="currentStats?.cpu.cores">
                {{ currentStats.cpu.cores }} cores
              </span>
              <span v-if="currentStats?.cpu.limit">
                Limit: {{ formatPercentage(currentStats.cpu.limit) }}
              </span>
            </div>
          </div>
        </div>

        <!-- 内存使用情况 -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Cpu /></el-icon>
              内存使用率
            </div>
            <div class="stat-value">
              {{ formatBytes(currentStats?.memory.usage || 0) }}
            </div>
          </div>
          <div class="stat-content">
            <el-progress
              :percentage="currentStats?.memory.percentage || 0"
              :show-text="false"
              :stroke-width="8"
              :color="getResourceColor(currentStats?.memory.percentage || 0)"
            />
            <div class="stat-details">
              <span>{{
                formatPercentage(currentStats?.memory.percentage || 0)
              }}</span>
              <span v-if="currentStats?.memory.limit">
                / {{ formatBytes(currentStats.memory.limit) }}
              </span>
            </div>
          </div>
        </div>

        <!-- 网络 I/O -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Connection /></el-icon>
              网络 I/O
            </div>
            <div class="stat-value">{{ formatBytes(networkTotal) }}/s</div>
          </div>
          <div class="stat-content">
            <div class="network-details">
              <div class="network-item">
                <el-icon class="network-icon tx">
                  <ArrowUp />
                </el-icon>
                <span class="network-label">TX:</span>
                <span class="network-value">{{
                  formatBytes(currentStats?.network.txBytes || 0)
                }}</span>
              </div>
              <div class="network-item">
                <el-icon class="network-icon rx">
                  <ArrowDown />
                </el-icon>
                <span class="network-label">RX:</span>
                <span class="network-value">{{
                  formatBytes(currentStats?.network.rxBytes || 0)
                }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 磁盘 I/O -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Folder /></el-icon>
              磁盘 I/O
            </div>
            <div class="stat-value">{{ formatBytes(diskTotal) }}/s</div>
          </div>
          <div class="stat-content">
            <div class="disk-details">
              <div class="disk-item">
                <el-icon class="disk-icon write">
                  <Edit />
                </el-icon>
                <span class="disk-label">Write:</span>
                <span class="disk-value">{{
                  formatBytes(currentStats?.disk.writeBytes || 0)
                }}</span>
              </div>
              <div class="disk-item">
                <el-icon class="disk-icon read">
                  <View />
                </el-icon>
                <span class="disk-label">Read:</span>
                <span class="disk-value">{{
                  formatBytes(currentStats?.disk.readBytes || 0)
                }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Last Updated -->
      <div class="stats-footer">
        <div class="last-updated">
          <el-icon><Clock /></el-icon>
          最后更新： {{ formatTime(currentStats?.timestamp) }}
        </div>
        <div class="update-controls">
          <el-button
size="small" @click="refreshStats"
:loading="loading"
>
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
          <el-button
            size="small"
            :type="autoRefresh ? 'primary' : 'default'"
            @click="toggleAutoRefresh"
          >
            <el-icon><Timer /></el-icon>
            自动刷新
          </el-button>
        </div>
      </div>
    </div>

    <!-- Historical Charts -->
    <div v-if="showHistorical" class="historical-charts">
      <div class="charts-header">
        <h3>历史数据</h3>
        <div class="time-range-selector">
          <el-select
            v-model="timeRange"
            size="small"
            @change="fetchHistoricalData"
          >
            <el-option label="Last Hour" value="1h" />
            <el-option label="Last 6 Hours" value="6h" />
            <el-option label="Last 24 Hours" value="24h" />
            <el-option label="Last 7 Days" value="7d" />
          </el-select>
        </div>
      </div>

      <div class="charts-grid">
        <!-- CPU Chart -->
        <div class="chart-container">
          <h4>CPU使用率趋势</h4>
          <div
ref="cpuChartRef" class="chart" />
        </div>

        <!-- Memory Chart -->
        <div class="chart-container">
          <h4>内存使用率趋势</h4>
          <div
ref="memoryChartRef" class="chart" />
        </div>

        <!-- Network Chart -->
        <div class="chart-container">
          <h4>网络 I/O 趋势</h4>
          <div
ref="networkChartRef" class="chart" />
        </div>

        <!-- Disk Chart -->
        <div class="chart-container">
          <h4>磁盘 I/O 趋势</h4>
          <div
ref="diskChartRef" class="chart" />
        </div>
      </div>
    </div>

    <!-- Detailed Metrics -->
    <div v-if="detailedView" class="detailed-metrics">
      <div class="metrics-header">
        <h3>详细指标</h3>
        <el-button size="small" @click="exportMetrics">
          <el-icon><Download /></el-icon>
          导出数据
        </el-button>
      </div>

      <el-tabs v-model="activeMetricTab">
        <!-- System Metrics -->
        <el-tab-pane label="系统" name="system">
          <div class="metrics-table">
            <el-table :data="systemMetrics" stripe>
              <el-table-column prop="metric" label="指标" />
              <el-table-column prop="value" label="当前值" />
              <el-table-column prop="unit" label="单位" />
              <el-table-column prop="description" label="描述" />
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Process Metrics -->
        <el-tab-pane label="进程" name="processes">
          <div class="process-metrics">
            <p>进程级别的指标将在此处显示</p>
          </div>
        </el-tab-pane>

        <!-- Network Metrics -->
        <el-tab-pane label="网络" name="network">
          <div class="network-metrics">
            <el-table :data="networkMetrics" stripe>
              <el-table-column prop="interface" label="网卡" />
              <el-table-column prop="rxBytes" label="接收字节" />
              <el-table-column prop="txBytes" label="发送字节" />
              <el-table-column prop="rxPackets" label="接收包" />
              <el-table-column prop="txPackets" label="发送包" />
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- Alerts -->
    <div v-if="alerts.length > 0" class="resource-alerts">
      <h4>资源告警</h4>
      <div class="alerts-list">
        <el-alert
          v-for="alert in alerts"
          :key="alert.id"
          :type="alert.type"
          :title="alert.title"
          :description="alert.message"
          :closable="false"
          class="alert-item"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from "vue";
import { storeToRefs } from "pinia";
import { ElMessage } from "element-plus";
import {
  Monitor,
  Cpu,
  Connection,
  Folder,
  ArrowUp,
  ArrowDown,
  Edit,
  View,
  Clock,
  Refresh,
  Timer,
  Download,
} from "@element-plus/icons-vue";

import { useContainerStore } from "@/store/containers";
import { useMonitoringStore } from "@/store/monitoring";
import * as echarts from 'echarts';
import { useDockerWebSocket } from "@/api/websocket";
import type { ECBasicOption } from "@/types/echarts";

interface Props {
  containerId: string;
  containerName: string;
  showHistorical?: boolean;
  detailedView?: boolean;
  autoStart?: boolean;
}

interface Alert {
  id: string;
  type: "success" | "warning" | "error" | "info";
  title: string;
  message: string;
}

const props = withDefaults(defineProps<Props>(), {
  showHistorical: false,
  detailedView: false,
  autoStart: true,
});

const containerStore = useContainerStore();
const monitoringStore = useMonitoringStore();
const { stats, historicalStats } = storeToRefs(containerStore);
const { realTimeMetrics, historicalData } = storeToRefs(monitoringStore);

// Local state
const loading = ref(false);
const autoRefresh = ref(true);
const timeRange = ref("1h");
const activeMetricTab = ref("system");
const alerts = ref<Alert[]>([]);

// Chart refs
const cpuChartRef = ref<HTMLElement>();
const memoryChartRef = ref<HTMLElement>();
const networkChartRef = ref<HTMLElement>();
const diskChartRef = ref<HTMLElement>();

// Chart instances
let cpuChart: echarts.ECharts | null = null;
let memoryChart: echarts.ECharts | null = null;
let networkChart: echarts.ECharts | null = null;
let diskChart: echarts.ECharts | null = null;

// Auto-refresh interval
let refreshInterval: NodeJS.Timeout | null = null;

// WebSocket connection
const { api: wsAPI } = useDockerWebSocket();
let unsubscribeStats: (() => void) | null = null;

// Computed
const currentStats = computed(() => {
  // Try to get from monitoring store first (real-time), fallback to container store
  return realTimeMetrics.value.get(props.containerId) || stats.value.get(props.containerId);
});

const containerHistoricalStats = computed(() => {
  // Try monitoring store first, fallback to container store
  const monitoringHistorical = historicalData.value.get(props.containerId);
  if (monitoringHistorical) {
    return monitoringHistorical.data;
  }
  return historicalStats.value.get(props.containerId) || [];
});

const networkTotal = computed(() => {
  if (!currentStats.value) return 0;
  return (
    currentStats.value.network.txBytes + currentStats.value.network.rxBytes
  );
});

const diskTotal = computed(() => {
  if (!currentStats.value) return 0;
  return currentStats.value.disk.readBytes + currentStats.value.disk.writeBytes;
});

const systemMetrics = computed(() => {
  if (!currentStats.value) return [];

  return [
    {
      metric: "CPU使用率",
      value: formatPercentage(currentStats.value.cpu.usage),
      unit: "%",
      description: "当前CPU利用率",
    },
    {
      metric: "内存使用率",
      value: formatBytes(currentStats.value.memory.usage),
      unit: "Bytes",
      description: "当前内存消耗",
    },
    {
      metric: "内存百分比",
      value: formatPercentage(currentStats.value.memory.percentage),
      unit: "%",
      description: "内存使用率占限制的百分比",
    },
    {
      metric: "网络接收",
      value: formatBytes(currentStats.value.network.rxBytes),
      unit: "Bytes",
      description: "接收的总字节数",
    },
    {
      metric: "网络发送",
      value: formatBytes(currentStats.value.network.txBytes),
      unit: "Bytes",
      description: "发送的总字节数",
    },
    {
      metric: "磁盘读取",
      value: formatBytes(currentStats.value.disk.readBytes),
      unit: "Bytes",
      description: "从磁盘读取的总字节数",
    },
    {
      metric: "磁盘写入",
      value: formatBytes(currentStats.value.disk.writeBytes),
      unit: "Bytes",
      description: "写入磁盘的总字节数",
    },
  ];
});

const networkMetrics = computed(() => {
  if (!currentStats.value) return [];

  // This would typically come from more detailed network stats
  return [
    {
      interface: "eth0",
      rxBytes: formatBytes(currentStats.value.network.rxBytes),
      txBytes: formatBytes(currentStats.value.network.txBytes),
      rxPackets: currentStats.value.network.rxPackets.toLocaleString(),
      txPackets: currentStats.value.network.txPackets.toLocaleString(),
    },
  ];
});

// Methods
function formatPercentage(value: number): string {
  return `${Math.round(value * 100) / 100}%`;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";

  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));

  return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`;
}

function formatTime(timestamp?: Date): string {
  if (!timestamp) return "Never";
  return new Date(timestamp).toLocaleTimeString();
}

function getResourceColor(percentage: number): string {
  if (percentage < 50) return "#67c23a";
  if (percentage < 80) return "#e6a23c";
  return "#f56c6c";
}

async function refreshStats() {
  loading.value = true;
  try {
    // Use monitoring store for real-time stats if available
    if (monitoringStore.isMonitored(props.containerId)) {
      await monitoringStore.fetchRealTimeMetrics(props.containerId);
    } else {
      await containerStore.fetchStats(props.containerId);
    }
    checkAlerts();
  } catch (error) {
    console.error("Failed to refresh stats:", error);
    ElMessage.error("刷新统计信息失败");
  } finally {
    loading.value = false;
  }
}

async function fetchHistoricalData() {
  if (!props.showHistorical) return;

  try {
    // Use monitoring store for historical data
    await monitoringStore.fetchHistoricalData(
      props.containerId,
      timeRange.value,
      getIntervalForRange(timeRange.value),
    );
    await nextTick();
    renderCharts();
  } catch (error) {
    console.error("Failed to fetch historical data:", error);
    ElMessage.error("加载历史数据失败");
  }
}

function getIntervalForRange(range: string): string {
  const intervals: Record<string, string> = {
    "1h": "1m",
    "6h": "5m",
    "24h": "15m",
    "7d": "1h",
  };
  return intervals[range] || "1m";
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value;

  if (autoRefresh.value) {
    startAutoRefresh();
  } else {
    stopAutoRefresh();
  }
}

function startAutoRefresh() {
  if (refreshInterval) return;

  // Start monitoring with the monitoring store
  monitoringStore.startMonitoring(props.containerId, {
    enableRealTime: true,
    enableHistorical: props.showHistorical,
    historicalPeriod: timeRange.value,
    historicalInterval: getIntervalForRange(timeRange.value),
  }).then(() => {
    console.log(`Started monitoring for container ${props.containerId}`);
    // Subscribe to WebSocket updates
    setupWebSocketSubscription();
  }).catch(error => {
    console.warn('Monitoring start failed, falling back to polling:', error);
    // Fallback to polling
    refreshInterval = setInterval(() => {
      refreshStats();
    }, 10000);
  });
}

function stopAutoRefresh() {
  // Stop monitoring
  if (monitoringStore.isMonitored(props.containerId)) {
    monitoringStore.stopMonitoring(props.containerId);
  }

  // Clean up WebSocket subscription
  if (unsubscribeStats) {
    unsubscribeStats();
    unsubscribeStats = null;
  }

  // Stop polling fallback
  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }
}

function checkAlerts() {
  const newAlerts: Alert[] = [];

  if (currentStats.value) {
    // CPU alert
    if (currentStats.value.cpu.usage > 80) {
      newAlerts.push({
        id: "cpu-high",
        type: "warning",
        title: "高CPU使用率",
        message: `CPU usage is at ${formatPercentage(currentStats.value.cpu.usage)}`,
      });
    }

    // Memory alert
    if (currentStats.value.memory.percentage > 85) {
      newAlerts.push({
        id: "memory-high",
        type: "error",
        title: "高内存使用率",
        message: `Memory usage is at ${formatPercentage(currentStats.value.memory.percentage)}`,
      });
    }
  }

  alerts.value = newAlerts;
}

async function exportMetrics() {
  try {
    const data = {
      container: props.containerName,
      timestamp: new Date().toISOString(),
      currentStats: currentStats.value,
      historicalStats: containerHistoricalStats.value,
      systemMetrics: systemMetrics.value,
      networkMetrics: networkMetrics.value,
    };

    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: "application/json",
    });

    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${props.containerName}-metrics-${Date.now()}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);

    ElMessage.success("指标导出成功");
  } catch (error) {
    console.error("Failed to export metrics:", error);
    ElMessage.error("导出指标失败");
  }
}

function renderCharts() {
  const historicalStats = containerHistoricalStats.value;
  if (!historicalStats.length) {
    console.log('No historical data available for charts');
    return;
  }

  console.log("Rendering charts with data:", historicalStats.length, "data points");

  // Prepare data
  const timestamps = historicalStats.map(stat =>
    new Date(stat.timestamp).toLocaleTimeString()
  );

  // Render CPU Chart
  if (cpuChartRef.value) {
    if (!cpuChart) {
      cpuChart = echarts.init(cpuChartRef.value);
    }

    const cpuOption: ECBasicOption = {
      title: {
        text: 'CPU 使用率',
        textStyle: { fontSize: 14, color: '#606266' },
      },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          const data = params[0];
          return `${data.name}<br/>CPU: ${data.value.toFixed(2)}%`;
        },
      },
      xAxis: {
        type: 'category',
        data: timestamps,
        axisLabel: { fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { formatter: '{value}%', fontSize: 10 },
      },
      series: [{
        name: 'CPU Usage',
        type: 'line',
        data: historicalStats.map(stat => stat.metrics.cpu.usage),
        smooth: true,
        lineStyle: { color: '#409eff', width: 2 },
        areaStyle: { color: 'rgba(64, 158, 255, 0.1)' },
        symbol: 'circle',
        symbolSize: 4,
      }],
      grid: {
        left: '10%',
        right: '10%',
        bottom: '20%',
        top: '20%',
      },
    };

    cpuChart.setOption(cpuOption);
  }

  // Render Memory Chart
  if (memoryChartRef.value) {
    if (!memoryChart) {
      memoryChart = echarts.init(memoryChartRef.value);
    }

    const memoryOption: ECBasicOption = {
      title: {
        text: '内存使用率',
        textStyle: { fontSize: 14, color: '#606266' },
      },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          const data = params[0];
          return `${data.name}<br/>内存: ${data.value.toFixed(2)}%`;
        },
      },
      xAxis: {
        type: 'category',
        data: timestamps,
        axisLabel: { fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { formatter: '{value}%', fontSize: 10 },
      },
      series: [{
        name: 'Memory Usage',
        type: 'line',
        data: historicalStats.map(stat => stat.metrics.memory.percentage),
        smooth: true,
        lineStyle: { color: '#67c23a', width: 2 },
        areaStyle: { color: 'rgba(103, 194, 58, 0.1)' },
        symbol: 'circle',
        symbolSize: 4,
      }],
      grid: {
        left: '10%',
        right: '10%',
        bottom: '20%',
        top: '20%',
      },
    };

    memoryChart.setOption(memoryOption);
  }

  // Render Network Chart
  if (networkChartRef.value) {
    if (!networkChart) {
      networkChart = echarts.init(networkChartRef.value);
    }

    const networkOption: ECBasicOption = {
      title: {
        text: '网络 I/O',
        textStyle: { fontSize: 14, color: '#606266' },
      },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          let result = params[0].name + '<br/>';
          params.forEach((param: any) => {
            result += `${param.seriesName}: ${formatBytes(param.value)}<br/>`;
          });
          return result;
        },
      },
      legend: {
        data: ['接收', '发送'],
        bottom: 0,
        textStyle: { fontSize: 10 },
      },
      xAxis: {
        type: 'category',
        data: timestamps,
        axisLabel: { fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        axisLabel: {
          formatter: (value: number) => formatBytes(value),
          fontSize: 10,
        },
      },
      series: [{
        name: '接收',
        type: 'line',
        data: historicalStats.map(stat => stat.metrics.network.rxBytes),
        smooth: true,
        lineStyle: { color: '#e6a23c', width: 2 },
        symbol: 'circle',
        symbolSize: 4,
      }, {
        name: '发送',
        type: 'line',
        data: historicalStats.map(stat => stat.metrics.network.txBytes),
        smooth: true,
        lineStyle: { color: '#f56c6c', width: 2 },
        symbol: 'circle',
        symbolSize: 4,
      }],
      grid: {
        left: '10%',
        right: '10%',
        bottom: '25%',
        top: '20%',
      },
    };

    networkChart.setOption(networkOption);
  }

  // Render Disk Chart
  if (diskChartRef.value) {
    if (!diskChart) {
      diskChart = echarts.init(diskChartRef.value);
    }

    const diskOption: ECBasicOption = {
      title: {
        text: '磁盘 I/O',
        textStyle: { fontSize: 14, color: '#606266' },
      },
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          let result = params[0].name + '<br/>';
          params.forEach((param: any) => {
            result += `${param.seriesName}: ${formatBytes(param.value)}<br/>`;
          });
          return result;
        },
      },
      legend: {
        data: ['读取', '写入'],
        bottom: 0,
        textStyle: { fontSize: 10 },
      },
      xAxis: {
        type: 'category',
        data: timestamps,
        axisLabel: { fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        axisLabel: {
          formatter: (value: number) => formatBytes(value),
          fontSize: 10,
        },
      },
      series: [{
        name: '读取',
        type: 'line',
        data: historicalStats.map(stat => stat.metrics.disk.readBytes),
        smooth: true,
        lineStyle: { color: '#909399', width: 2 },
        symbol: 'circle',
        symbolSize: 4,
      }, {
        name: '写入',
        type: 'line',
        data: historicalStats.map(stat => stat.metrics.disk.writeBytes),
        smooth: true,
        lineStyle: { color: '#409eff', width: 2 },
        symbol: 'circle',
        symbolSize: 4,
      }],
      grid: {
        left: '10%',
        right: '10%',
        bottom: '25%',
        top: '20%',
      },
    };

    diskChart.setOption(diskOption);
  }
}

// Setup WebSocket subscription
function setupWebSocketSubscription() {
  if (wsAPI?.isConnected()) {
    console.log('Setting up WebSocket subscription for container stats:', props.containerId);
    unsubscribeStats = wsAPI.subscribeToContainerStats(
      props.containerId,
      (data) => {
        console.log('Received real-time stats via WebSocket:', data);
        if (data.stats) {
          monitoringStore.updateRealTimeMetrics(props.containerId, data.stats);
          checkAlerts();
          // Update charts in real-time if they exist
          updateChartsRealTime(data.stats);
        }
      }
    );
  }
}

// Update charts with real-time data
function updateChartsRealTime(metrics: any) {
  if (!props.showHistorical) return;

  const now = new Date().toLocaleTimeString();

  // Update CPU chart
  if (cpuChart) {
    cpuChart.setOption({
      xAxis: {
        data: [...(cpuChart.getOption().xAxis[0].data || []), now].slice(-20) // Keep last 20 points
      },
      series: [{
        data: [...(cpuChart.getOption().series[0].data || []), metrics.cpu.usage].slice(-20)
      }]
    });
  }

  // Update Memory chart
  if (memoryChart) {
    memoryChart.setOption({
      xAxis: {
        data: [...(memoryChart.getOption().xAxis[0].data || []), now].slice(-20)
      },
      series: [{
        data: [...(memoryChart.getOption().series[0].data || []), metrics.memory.percentage].slice(-20)
      }]
    });
  }

  // Update Network chart
  if (networkChart) {
    const currentOption = networkChart.getOption();
    networkChart.setOption({
      xAxis: {
        data: [...(currentOption.xAxis[0].data || []), now].slice(-20)
      },
      series: [{
        data: [...(currentOption.series[0].data || []), metrics.network.rxBytes].slice(-20)
      }, {
        data: [...(currentOption.series[1].data || []), metrics.network.txBytes].slice(-20)
      }]
    });
  }

  // Update Disk chart
  if (diskChart) {
    const currentOption = diskChart.getOption();
    diskChart.setOption({
      xAxis: {
        data: [...(currentOption.xAxis[0].data || []), now].slice(-20)
      },
      series: [{
        data: [...(currentOption.series[0].data || []), metrics.disk.readBytes].slice(-20)
      }, {
        data: [...(currentOption.series[1].data || []), metrics.disk.writeBytes].slice(-20)
      }]
    });
  }
}

// Lifecycle
onMounted(() => {
  if (props.autoStart) {
    refreshStats();

    if (props.showHistorical) {
      fetchHistoricalData();
    }

    if (autoRefresh.value) {
      startAutoRefresh();
    }
  }

  // Handle window resize for charts
  const handleResize = () => {
    cpuChart?.resize();
    memoryChart?.resize();
    networkChart?.resize();
    diskChart?.resize();
  };

  window.addEventListener('resize', handleResize);

  // Cleanup on unmount
  onUnmounted(() => {
    window.removeEventListener('resize', handleResize);
    cpuChart?.dispose();
    memoryChart?.dispose();
    networkChart?.dispose();
    diskChart?.dispose();
  });
});

onUnmounted(() => {
  stopAutoRefresh();

  // Dispose chart instances
  cpuChart?.dispose();
  memoryChart?.dispose();
  networkChart?.dispose();
  diskChart?.dispose();
});

// Watch for container changes
watch(
  () => props.containerId,
  (newId, oldId) => {
    if (oldId && oldId !== newId) {
      // Stop monitoring old container
      if (monitoringStore.isMonitored(oldId)) {
        monitoringStore.stopMonitoring(oldId);
      }
      // Clean up WebSocket subscription
      if (unsubscribeStats) {
        unsubscribeStats();
        unsubscribeStats = null;
      }
    }

    if (newId) {
      refreshStats();
      if (props.showHistorical) {
        fetchHistoricalData();
      }
      if (autoRefresh.value) {
        startAutoRefresh();
      }
    }
  },
);

// Watch for time range changes
watch(timeRange, () => {
  if (props.showHistorical) {
    fetchHistoricalData();
  }
});
</script>

<style scoped>
.resource-monitor {
  background: white;
  border-radius: 8px;
  overflow: hidden;
}

.resource-monitor.detailed-view {
  border: 1px solid #e4e7ed;
}

.current-stats {
  padding: 20px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.stat-card {
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 16px;
  transition: box-shadow 0.3s ease;
}

.stat-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.stat-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: #606266;
  font-size: 14px;
}

.stat-value {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.stat-content {
  margin-top: 8px;
}

.stat-details {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
}

.network-details,
.disk-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.network-item,
.disk-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.network-icon,
.disk-icon {
  font-size: 14px;
}

.network-icon.tx,
.disk-icon.write {
  color: #e6a23c;
}

.network-icon.rx,
.disk-icon.read {
  color: #67c23a;
}

.network-label,
.disk-label {
  font-weight: 500;
  color: #606266;
  min-width: 40px;
}

.network-value,
.disk-value {
  color: #303133;
  font-weight: 500;
}

.stats-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid #e4e7ed;
}

.last-updated {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #909399;
}

.update-controls {
  display: flex;
  gap: 8px;
}

.historical-charts {
  padding: 20px;
  border-top: 1px solid #e4e7ed;
  background: #fafafa;
}

.charts-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.charts-header h3 {
  margin: 0;
  color: #303133;
}

.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 20px;
}

.chart-container {
  background: white;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 16px;
}

.chart-container h4 {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: #606266;
}

.chart {
  height: 200px;
  border-radius: 4px;
  min-height: 200px;
}

.detailed-metrics {
  padding: 20px;
  border-top: 1px solid #e4e7ed;
}

.metrics-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.metrics-header h3 {
  margin: 0;
  color: #303133;
}

.metrics-table {
  margin-top: 16px;
}

.process-metrics,
.network-metrics {
  margin-top: 16px;
}

.resource-alerts {
  padding: 20px;
  border-top: 1px solid #e4e7ed;
  background: #fef9e7;
}

.resource-alerts h4 {
  margin: 0 0 16px 0;
  color: #e6a23c;
  display: flex;
  align-items: center;
  gap: 8px;
}

.alerts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.alert-item {
  border-radius: 6px;
}

/* Responsive Design */
@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .charts-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .stats-footer {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .update-controls {
    justify-content: center;
  }

  .charts-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .metrics-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
}
</style>
