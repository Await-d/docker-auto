<template>
  <div class="resource-charts-widget">
    <div class="widget-header">
      <div class="header-content">
        <div class="header-info">
          <el-icon class="header-icon">
            <TrendCharts />
          </el-icon>
          <div class="header-text">
            <h3>资源使用图表</h3>
            <p class="subtitle">系统资源实时监控与历史趋势</p>
          </div>
        </div>
        <div class="header-controls">
          <el-select
            v-model="selectedTimeRange"
            class="time-range-selector"
            size="small"
            @change="handleTimeRangeChange"
          >
            <el-option
              v-for="option in timeRangeOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <el-button
            :icon="Refresh"
            :loading="loading"
            size="small"
            type="primary"
            circle
            @click="refreshData"
          />
          <el-dropdown @command="handleExport">
            <el-button :icon="Download" size="small" circle />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="png">导出PNG</el-dropdown-item>
                <el-dropdown-item command="svg">导出SVG</el-dropdown-item>
                <el-dropdown-item command="csv">导出CSV</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <div class="widget-content" v-loading="loading">
      <div v-if="error" class="error-state">
        <el-result icon="error" title="数据加载失败" :sub-title="error">
          <template #extra>
            <el-button type="primary" @click="refreshData">重新加载</el-button>
          </template>
        </el-result>
      </div>

      <div v-else-if="!hasData" class="empty-state">
        <el-empty description="暂无监控数据">
          <el-button type="primary" @click="initializeMonitoring">开始监控</el-button>
        </el-empty>
      </div>

      <div v-else class="charts-container">
        <!-- Metric Summary Cards -->
        <div class="metrics-summary">
          <div class="metric-card cpu">
            <div class="metric-icon">
              <el-icon><Cpu /></el-icon>
            </div>
            <div class="metric-content">
              <div class="metric-value">{{ formatPercentage(currentMetrics.cpu) }}</div>
              <div class="metric-label">CPU使用率</div>
              <div class="metric-trend" :class="cpuTrend">
                <el-icon v-if="cpuTrend === 'up'"><ArrowUp /></el-icon>
                <el-icon v-else-if="cpuTrend === 'down'"><ArrowDown /></el-icon>
                <el-icon v-else><Minus /></el-icon>
              </div>
            </div>
          </div>

          <div class="metric-card memory">
            <div class="metric-icon">
              <el-icon><Memory /></el-icon>
            </div>
            <div class="metric-content">
              <div class="metric-value">{{ formatPercentage(currentMetrics.memory) }}</div>
              <div class="metric-label">内存使用率</div>
              <div class="metric-trend" :class="memoryTrend">
                <el-icon v-if="memoryTrend === 'up'"><ArrowUp /></el-icon>
                <el-icon v-else-if="memoryTrend === 'down'"><ArrowDown /></el-icon>
                <el-icon v-else><Minus /></el-icon>
              </div>
            </div>
          </div>

          <div class="metric-card disk">
            <div class="metric-icon">
              <el-icon><HardDrive /></el-icon>
            </div>
            <div class="metric-content">
              <div class="metric-value">{{ formatBytes(currentMetrics.diskIO) }}</div>
              <div class="metric-label">磁盘I/O</div>
              <div class="metric-trend" :class="diskTrend">
                <el-icon v-if="diskTrend === 'up'"><ArrowUp /></el-icon>
                <el-icon v-else-if="diskTrend === 'down'"><ArrowDown /></el-icon>
                <el-icon v-else><Minus /></el-icon>
              </div>
            </div>
          </div>

          <div class="metric-card network">
            <div class="metric-icon">
              <el-icon><Network /></el-icon>
            </div>
            <div class="metric-content">
              <div class="metric-value">{{ formatBytes(currentMetrics.networkTotal) }}</div>
              <div class="metric-label">网络流量</div>
              <div class="metric-trend" :class="networkTrend">
                <el-icon v-if="networkTrend === 'up'"><ArrowUp /></el-icon>
                <el-icon v-else-if="networkTrend === 'down'"><ArrowDown /></el-icon>
                <el-icon v-else><Minus /></el-icon>
              </div>
            </div>
          </div>
        </div>

        <!-- Chart Tabs -->
        <el-tabs v-model="activeTab" class="chart-tabs" @tab-change="handleTabChange">
          <el-tab-pane label="系统概览" name="overview">
            <div
              ref="overviewChartRef"
              class="chart-container overview-chart"
              :style="{ height: chartHeight + 'px' }"
            />
          </el-tab-pane>

          <el-tab-pane label="CPU详情" name="cpu">
            <div
              ref="cpuChartRef"
              class="chart-container cpu-chart"
              :style="{ height: chartHeight + 'px' }"
            />
          </el-tab-pane>

          <el-tab-pane label="内存详情" name="memory">
            <div
              ref="memoryChartRef"
              class="chart-container memory-chart"
              :style="{ height: chartHeight + 'px' }"
            />
          </el-tab-pane>

          <el-tab-pane label="网络I/O" name="network">
            <div
              ref="networkChartRef"
              class="chart-container network-chart"
              :style="{ height: chartHeight + 'px' }"
            />
          </el-tab-pane>

          <el-tab-pane label="磁盘I/O" name="disk">
            <div
              ref="diskChartRef"
              class="chart-container disk-chart"
              :style="{ height: chartHeight + 'px' }"
            />
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue';
import * as echarts from 'echarts/core';
import {
  LineChart,
  BarChart,
  PieChart
} from 'echarts/charts';
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  ToolboxComponent
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import {
  TrendCharts,
  Refresh,
  Download,
  ArrowUp,
  ArrowDown,
  Minus,
  Cpu,
  Memo,
  Folder,
  Connection
} from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import { useMonitoringStore } from '@/store/monitoring';
import { monitoringAPI } from '@/api/monitoring';
import { useDockerWebSocket } from '@/api/websocket';
import type { SystemMetrics, ResourceMetrics } from '@/api/monitoring';

// Register ECharts components
echarts.use([
  LineChart,
  BarChart,
  PieChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  ToolboxComponent,
  CanvasRenderer
]);

interface Props {
  widgetId: string;
  widgetConfig?: {
    chartHeight?: number;
    refreshInterval?: number;
    maxDataPoints?: number;
    enableWebSocket?: boolean;
  };
  widgetData?: any;
}

const props = withDefaults(defineProps<Props>(), {
  widgetConfig: () => ({
    chartHeight: 300,
    refreshInterval: 5000,
    maxDataPoints: 100,
    enableWebSocket: true
  })
});

const monitoringStore = useMonitoringStore();
const { connect: connectWS, disconnect: disconnectWS, isConnected } = useDockerWebSocket();

// Reactive state
const loading = ref(true);
const error = ref<string | null>(null);
const activeTab = ref('overview');
const selectedTimeRange = ref('1h');
const historicalData = ref<SystemMetrics[]>([]);
const currentMetrics = ref({
  cpu: 0,
  memory: 0,
  diskIO: 0,
  networkTotal: 0
});

// Chart refs
const overviewChartRef = ref<HTMLDivElement>();
const cpuChartRef = ref<HTMLDivElement>();
const memoryChartRef = ref<HTMLDivElement>();
const networkChartRef = ref<HTMLDivElement>();
const diskChartRef = ref<HTMLDivElement>();

// Chart instances
let overviewChart: echarts.ECharts | null = null;
let cpuChart: echarts.ECharts | null = null;
let memoryChart: echarts.ECharts | null = null;
let networkChart: echarts.ECharts | null = null;
let diskChart: echarts.ECharts | null = null;

// Configuration
const chartHeight = computed(() => props.widgetConfig?.chartHeight || 300);
const refreshInterval = computed(() => props.widgetConfig?.refreshInterval || 5000);
const maxDataPoints = computed(() => props.widgetConfig?.maxDataPoints || 100);

const timeRangeOptions = [
  { label: '最近15分钟', value: '15m' },
  { label: '最近30分钟', value: '30m' },
  { label: '最近1小时', value: '1h' },
  { label: '最近6小时', value: '6h' },
  { label: '最近24小时', value: '24h' },
  { label: '最近7天', value: '7d' }
];

const hasData = computed(() => historicalData.value.length > 0);

// Trend calculations
const cpuTrend = computed(() => calculateTrend('cpu'));
const memoryTrend = computed(() => calculateTrend('memory'));
const diskTrend = computed(() => calculateTrend('disk'));
const networkTrend = computed(() => calculateTrend('network'));

// WebSocket connection
let wsUnsubscribe: (() => void) | null = null;
let refreshTimer: NodeJS.Timeout | null = null;

/**
 * Calculate trend for a specific metric
 */
function calculateTrend(metric: string): 'up' | 'down' | 'stable' {
  if (historicalData.value.length < 5) return 'stable';

  const recent = historicalData.value.slice(-5);
  const values: number[] = [];

  recent.forEach(data => {
    switch (metric) {
      case 'cpu':
        values.push(data.cpu.usage);
        break;
      case 'memory':
        values.push(data.memory.percentage);
        break;
      case 'disk':
        values.push(data.disk.used);
        break;
      case 'network':
        const totalNetwork = data.network.interfaces.reduce((sum, iface) =>
          sum + iface.rxBytes + iface.txBytes, 0);
        values.push(totalNetwork);
        break;
    }
  });

  const firstValue = values[0];
  const lastValue = values[values.length - 1];
  const change = ((lastValue - firstValue) / firstValue) * 100;

  if (Math.abs(change) < 5) return 'stable';
  return change > 0 ? 'up' : 'down';
}

/**
 * Format percentage values
 */
function formatPercentage(value: number): string {
  return `${Math.round(value * 100) / 100}%`;
}

/**
 * Format byte values
 */
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

/**
 * Get time range in milliseconds
 */
function getTimeRangeMs(range: string): number {
  const timeMap: Record<string, number> = {
    '15m': 15 * 60 * 1000,
    '30m': 30 * 60 * 1000,
    '1h': 60 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000
  };
  return timeMap[range] || timeMap['1h'];
}

/**
 * Fetch historical system metrics
 */
async function fetchHistoricalData() {
  try {
    loading.value = true;
    error.value = null;

    const endTime = new Date();
    const startTime = new Date(endTime.getTime() - getTimeRangeMs(selectedTimeRange.value));

    // For now, we'll simulate historical data by fetching current system metrics
    // In a real implementation, you would have an API endpoint that returns historical data
    const metrics = await monitoringAPI.getSystemMetrics();

    // Simulate historical data points
    const dataPoints: SystemMetrics[] = [];
    const intervalMs = getTimeRangeMs(selectedTimeRange.value) / 50; // 50 data points

    for (let i = 0; i < 50; i++) {
      const timestamp = new Date(startTime.getTime() + (i * intervalMs));
      // Add some random variation to simulate real data
      const cpuVariation = (Math.random() - 0.5) * 20;
      const memoryVariation = (Math.random() - 0.5) * 15;

      dataPoints.push({
        ...metrics,
        cpu: {
          ...metrics.cpu,
          usage: Math.max(0, Math.min(100, metrics.cpu.usage + cpuVariation))
        },
        memory: {
          ...metrics.memory,
          percentage: Math.max(0, Math.min(100, metrics.memory.percentage + memoryVariation))
        },
        timestamp
      });
    }

    historicalData.value = dataPoints;
    updateCurrentMetrics(metrics);

  } catch (err) {
    console.error('Failed to fetch historical data:', err);
    error.value = err instanceof Error ? err.message : '数据获取失败';
    ElMessage.error('获取监控数据失败');
  } finally {
    loading.value = false;
  }
}

/**
 * Update current metrics summary
 */
function updateCurrentMetrics(metrics: SystemMetrics) {
  const networkTotal = metrics.network.interfaces.reduce((sum, iface) =>
    sum + iface.rxBytes + iface.txBytes, 0);

  currentMetrics.value = {
    cpu: metrics.cpu.usage,
    memory: metrics.memory.percentage,
    diskIO: metrics.disk.used,
    networkTotal
  };
}

/**
 * Initialize charts
 */
async function initializeCharts() {
  await nextTick();

  // Initialize all chart instances
  if (overviewChartRef.value) {
    overviewChart = echarts.init(overviewChartRef.value);
  }
  if (cpuChartRef.value) {
    cpuChart = echarts.init(cpuChartRef.value);
  }
  if (memoryChartRef.value) {
    memoryChart = echarts.init(memoryChartRef.value);
  }
  if (networkChartRef.value) {
    networkChart = echarts.init(networkChartRef.value);
  }
  if (diskChartRef.value) {
    diskChart = echarts.init(diskChartRef.value);
  }

  // Update charts with data
  updateCharts();

  // Handle window resize
  window.addEventListener('resize', handleResize);
}

/**
 * Update all charts with current data
 */
function updateCharts() {
  if (!hasData.value) return;

  const timeData = historicalData.value.map(d => d.timestamp);
  const cpuData = historicalData.value.map(d => d.cpu.usage);
  const memoryData = historicalData.value.map(d => d.memory.percentage);

  updateOverviewChart(timeData, cpuData, memoryData);
  updateCpuChart(timeData, cpuData);
  updateMemoryChart(timeData, memoryData);
  updateNetworkChart();
  updateDiskChart();
}

/**
 * Update overview chart
 */
function updateOverviewChart(timeData: Date[], cpuData: number[], memoryData: number[]) {
  if (!overviewChart) return;

  const option = {
    title: {
      text: '系统资源概览',
      left: 'center',
      textStyle: { fontSize: 16, color: '#333' }
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
      formatter: (params: any) => {
        let result = `<div>${params[0].axisValue}</div>`;
        params.forEach((param: any) => {
          result += `<div>${param.marker} ${param.seriesName}: ${param.value.toFixed(2)}%</div>`;
        });
        return result;
      }
    },
    legend: {
      data: ['CPU使用率', '内存使用率'],
      bottom: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '10%',
      containLabel: true
    },
    toolbox: {
      feature: {
        dataZoom: { yAxisIndex: 'none' },
        restore: {},
        saveAsImage: {}
      }
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: timeData.map(d => d.toLocaleTimeString())
    },
    yAxis: {
      type: 'value',
      name: '使用率 (%)',
      min: 0,
      max: 100
    },
    dataZoom: [{
      type: 'inside',
      start: 0,
      end: 100
    }, {
      start: 0,
      end: 100
    }],
    series: [
      {
        name: 'CPU使用率',
        type: 'line',
        smooth: true,
        data: cpuData,
        itemStyle: { color: '#409EFF' },
        areaStyle: { opacity: 0.3 }
      },
      {
        name: '内存使用率',
        type: 'line',
        smooth: true,
        data: memoryData,
        itemStyle: { color: '#67C23A' },
        areaStyle: { opacity: 0.3 }
      }
    ]
  };

  overviewChart.setOption(option);
}

/**
 * Update CPU chart with detailed information
 */
function updateCpuChart(timeData: Date[], cpuData: number[]) {
  if (!cpuChart) return;

  const option = {
    title: {
      text: 'CPU使用率详情',
      left: 'center',
      textStyle: { fontSize: 16, color: '#333' }
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const data = params[0];
        return `时间: ${data.axisValue}<br/>CPU使用率: ${data.value.toFixed(2)}%`;
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: timeData.map(d => d.toLocaleTimeString())
    },
    yAxis: {
      type: 'value',
      name: 'CPU使用率 (%)',
      min: 0,
      max: 100
    },
    series: [{
      type: 'line',
      smooth: true,
      data: cpuData,
      itemStyle: { color: '#E6A23C' },
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(230, 162, 60, 0.3)' },
          { offset: 1, color: 'rgba(230, 162, 60, 0)' }
        ])
      }
    }]
  };

  cpuChart.setOption(option);
}

/**
 * Update memory chart
 */
function updateMemoryChart(timeData: Date[], memoryData: number[]) {
  if (!memoryChart) return;

  const option = {
    title: {
      text: '内存使用率详情',
      left: 'center',
      textStyle: { fontSize: 16, color: '#333' }
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const data = params[0];
        return `时间: ${data.axisValue}<br/>内存使用率: ${data.value.toFixed(2)}%`;
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: timeData.map(d => d.toLocaleTimeString())
    },
    yAxis: {
      type: 'value',
      name: '内存使用率 (%)',
      min: 0,
      max: 100
    },
    series: [{
      type: 'line',
      smooth: true,
      data: memoryData,
      itemStyle: { color: '#67C23A' },
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(103, 194, 58, 0.3)' },
          { offset: 1, color: 'rgba(103, 194, 58, 0)' }
        ])
      }
    }]
  };

  memoryChart.setOption(option);
}

/**
 * Update network chart
 */
function updateNetworkChart() {
  if (!networkChart) return;

  const networkData = historicalData.value.map(d => {
    const totalRx = d.network.interfaces.reduce((sum, iface) => sum + iface.rxBytes, 0);
    const totalTx = d.network.interfaces.reduce((sum, iface) => sum + iface.txBytes, 0);
    return { rx: totalRx, tx: totalTx, time: d.timestamp };
  });

  const option = {
    title: {
      text: '网络I/O流量',
      left: 'center',
      textStyle: { fontSize: 16, color: '#333' }
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        let result = `<div>${params[0].axisValue}</div>`;
        params.forEach((param: any) => {
          const value = formatBytes(param.value);
          result += `<div>${param.marker} ${param.seriesName}: ${value}</div>`;
        });
        return result;
      }
    },
    legend: {
      data: ['接收流量', '发送流量'],
      bottom: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: networkData.map(d => d.time.toLocaleTimeString())
    },
    yAxis: {
      type: 'value',
      name: '流量 (Bytes)',
      axisLabel: {
        formatter: (value: number) => formatBytes(value)
      }
    },
    series: [
      {
        name: '接收流量',
        type: 'line',
        smooth: true,
        data: networkData.map(d => d.rx),
        itemStyle: { color: '#F56C6C' },
        areaStyle: { opacity: 0.3 }
      },
      {
        name: '发送流量',
        type: 'line',
        smooth: true,
        data: networkData.map(d => d.tx),
        itemStyle: { color: '#909399' },
        areaStyle: { opacity: 0.3 }
      }
    ]
  };

  networkChart.setOption(option);
}

/**
 * Update disk chart
 */
function updateDiskChart() {
  if (!diskChart) return;

  const diskData = historicalData.value.map(d => ({
    used: d.disk.used,
    free: d.disk.free,
    time: d.timestamp
  }));

  const option = {
    title: {
      text: '磁盘使用情况',
      left: 'center',
      textStyle: { fontSize: 16, color: '#333' }
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        let result = `<div>${params[0].axisValue}</div>`;
        params.forEach((param: any) => {
          const value = formatBytes(param.value);
          result += `<div>${param.marker} ${param.seriesName}: ${value}</div>`;
        });
        return result;
      }
    },
    legend: {
      data: ['已用空间', '可用空间'],
      bottom: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: diskData.map(d => d.time.toLocaleTimeString())
    },
    yAxis: {
      type: 'value',
      name: '存储空间 (Bytes)',
      axisLabel: {
        formatter: (value: number) => formatBytes(value)
      }
    },
    series: [
      {
        name: '已用空间',
        type: 'line',
        smooth: true,
        data: diskData.map(d => d.used),
        itemStyle: { color: '#E6A23C' },
        areaStyle: { opacity: 0.3 }
      },
      {
        name: '可用空间',
        type: 'line',
        smooth: true,
        data: diskData.map(d => d.free),
        itemStyle: { color: '#909399' },
        areaStyle: { opacity: 0.3 }
      }
    ]
  };

  diskChart.setOption(option);
}

/**
 * Handle window resize
 */
function handleResize() {
  overviewChart?.resize();
  cpuChart?.resize();
  memoryChart?.resize();
  networkChart?.resize();
  diskChart?.resize();
}

/**
 * Handle time range change
 */
async function handleTimeRangeChange() {
  await fetchHistoricalData();
  updateCharts();
}

/**
 * Handle tab change
 */
function handleTabChange(tabName: string) {
  activeTab.value = tabName;
  // Trigger chart resize after tab switch
  nextTick(() => {
    setTimeout(handleResize, 100);
  });
}

/**
 * Refresh data manually
 */
async function refreshData() {
  await fetchHistoricalData();
  updateCharts();
}

/**
 * Initialize monitoring
 */
async function initializeMonitoring() {
  await fetchHistoricalData();

  if (props.widgetConfig?.enableWebSocket) {
    await setupWebSocketConnection();
  }

  await initializeCharts();
  setupAutoRefresh();
}

/**
 * Setup WebSocket connection for real-time updates
 */
async function setupWebSocketConnection() {
  try {
    const wsAPI = await connectWS('ws://localhost:8080/ws', localStorage.getItem('token') || '');

    wsUnsubscribe = wsAPI.subscribeToSystemMetrics((metrics: SystemMetrics) => {
      // Add new data point
      historicalData.value.push(metrics);

      // Limit data points
      if (historicalData.value.length > maxDataPoints.value) {
        historicalData.value = historicalData.value.slice(-maxDataPoints.value);
      }

      updateCurrentMetrics(metrics);
      updateCharts();
    }, refreshInterval.value);

  } catch (err) {
    console.error('WebSocket connection failed:', err);
    // Fall back to polling
    setupAutoRefresh();
  }
}

/**
 * Setup auto-refresh timer
 */
function setupAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }

  refreshTimer = setInterval(async () => {
    if (!isConnected()) {
      await fetchHistoricalData();
      updateCharts();
    }
  }, refreshInterval.value);
}

/**
 * Handle export functionality
 */
function handleExport(format: string) {
  const activeChart = getActiveChart();
  if (!activeChart) return;

  switch (format) {
    case 'png':
      const url = activeChart.getDataURL({
        pixelRatio: 2,
        backgroundColor: '#fff'
      });
      const link = document.createElement('a');
      link.href = url;
      link.download = `resource-chart-${activeTab.value}.png`;
      link.click();
      break;
    case 'svg':
      const svgUrl = activeChart.getDataURL({
        type: 'svg'
      });
      const svgLink = document.createElement('a');
      svgLink.href = svgUrl;
      svgLink.download = `resource-chart-${activeTab.value}.svg`;
      svgLink.click();
      break;
    case 'csv':
      exportToCSV();
      break;
  }
}

/**
 * Get currently active chart instance
 */
function getActiveChart(): echarts.ECharts | null {
  switch (activeTab.value) {
    case 'overview': return overviewChart;
    case 'cpu': return cpuChart;
    case 'memory': return memoryChart;
    case 'network': return networkChart;
    case 'disk': return diskChart;
    default: return overviewChart;
  }
}

/**
 * Export data to CSV
 */
function exportToCSV() {
  if (!hasData.value) return;

  let csvContent = 'Timestamp,CPU Usage (%),Memory Usage (%),Network RX (Bytes),Network TX (Bytes),Disk Used (Bytes)\n';

  historicalData.value.forEach(data => {
    const networkRx = data.network.interfaces.reduce((sum, iface) => sum + iface.rxBytes, 0);
    const networkTx = data.network.interfaces.reduce((sum, iface) => sum + iface.txBytes, 0);

    csvContent += `${data.timestamp.toISOString()},${data.cpu.usage},${data.memory.percentage},${networkRx},${networkTx},${data.disk.used}\n`;
  });

  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.download = `resource-metrics-${selectedTimeRange.value}.csv`;
  link.click();
}

/**
 * Cleanup function
 */
function cleanup() {
  // Dispose chart instances
  overviewChart?.dispose();
  cpuChart?.dispose();
  memoryChart?.dispose();
  networkChart?.dispose();
  diskChart?.dispose();

  // Clear timers and WebSocket
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }

  if (wsUnsubscribe) {
    wsUnsubscribe();
  }

  disconnectWS();

  // Remove event listeners
  window.removeEventListener('resize', handleResize);
}

// Lifecycle hooks
onMounted(async () => {
  await initializeMonitoring();
});

onUnmounted(() => {
  cleanup();
});

// Watch for tab changes to ensure charts are properly rendered
watch(activeTab, () => {
  nextTick(() => {
    setTimeout(handleResize, 100);
  });
});
</script>

<style scoped lang="scss">
.resource-charts-widget {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border-radius: 8px;
  overflow: hidden;

  .widget-header {
    padding: 16px 20px 12px;
    border-bottom: 1px solid var(--el-border-color-light);
    background: var(--el-bg-color-page);

    .header-content {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;

      .header-info {
        display: flex;
        align-items: flex-start;
        gap: 12px;

        .header-icon {
          font-size: 24px;
          color: var(--el-color-primary);
          margin-top: 2px;
        }

        .header-text {
          h3 {
            margin: 0 0 4px 0;
            font-size: 18px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          .subtitle {
            margin: 0;
            font-size: 13px;
            color: var(--el-text-color-regular);
            line-height: 1.4;
          }
        }
      }

      .header-controls {
        display: flex;
        align-items: center;
        gap: 8px;

        .time-range-selector {
          width: 120px;
        }
      }
    }
  }

  .widget-content {
    flex: 1;
    padding: 20px;
    overflow: hidden;

    .error-state {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
    }

    .empty-state {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
    }

    .charts-container {
      height: 100%;
      display: flex;
      flex-direction: column;

      .metrics-summary {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;
        margin-bottom: 24px;

        .metric-card {
          display: flex;
          align-items: center;
          padding: 16px;
          background: var(--el-bg-color-page);
          border-radius: 8px;
          border: 1px solid var(--el-border-color-lighter);
          transition: all 0.3s ease;

          &:hover {
            border-color: var(--el-color-primary);
            transform: translateY(-2px);
            box-shadow: 0 4px 12px var(--el-box-shadow-light);
          }

          .metric-icon {
            display: flex;
            align-items: center;
            justify-content: center;
            width: 48px;
            height: 48px;
            border-radius: 50%;
            margin-right: 12px;

            .el-icon {
              font-size: 24px;
            }
          }

          .metric-content {
            flex: 1;
            position: relative;

            .metric-value {
              font-size: 24px;
              font-weight: 700;
              line-height: 1;
              margin-bottom: 4px;
            }

            .metric-label {
              font-size: 13px;
              color: var(--el-text-color-regular);
            }

            .metric-trend {
              position: absolute;
              top: 0;
              right: 0;
              font-size: 16px;

              &.up { color: var(--el-color-danger); }
              &.down { color: var(--el-color-success); }
              &.stable { color: var(--el-text-color-secondary); }
            }
          }

          &.cpu {
            .metric-icon {
              background: rgba(230, 162, 60, 0.1);
              color: var(--el-color-warning);
            }
            .metric-value { color: var(--el-color-warning); }
          }

          &.memory {
            .metric-icon {
              background: rgba(103, 194, 58, 0.1);
              color: var(--el-color-success);
            }
            .metric-value { color: var(--el-color-success); }
          }

          &.disk {
            .metric-icon {
              background: rgba(245, 108, 108, 0.1);
              color: var(--el-color-danger);
            }
            .metric-value { color: var(--el-color-danger); }
          }

          &.network {
            .metric-icon {
              background: rgba(64, 158, 255, 0.1);
              color: var(--el-color-primary);
            }
            .metric-value { color: var(--el-color-primary); }
          }
        }
      }

      .chart-tabs {
        flex: 1;
        display: flex;
        flex-direction: column;

        :deep(.el-tabs__content) {
          flex: 1;
          overflow: hidden;
        }

        :deep(.el-tab-pane) {
          height: 100%;
        }

        .chart-container {
          width: 100%;
          background: var(--el-bg-color-page);
          border-radius: 6px;
          border: 1px solid var(--el-border-color-lighter);
        }
      }
    }
  }
}

@media (max-width: 768px) {
  .resource-charts-widget {
    .widget-header {
      padding: 12px 16px 8px;

      .header-content {
        flex-direction: column;
        gap: 12px;
        align-items: stretch;

        .header-controls {
          justify-content: flex-end;
        }
      }
    }

    .widget-content {
      padding: 16px 12px;

      .charts-container {
        .metrics-summary {
          grid-template-columns: 1fr;
          gap: 12px;
          margin-bottom: 16px;

          .metric-card {
            padding: 12px;

            .metric-icon {
              width: 40px;
              height: 40px;
              margin-right: 8px;

              .el-icon {
                font-size: 20px;
              }
            }

            .metric-content {
              .metric-value {
                font-size: 20px;
              }
            }
          }
        }
      }
    }
  }
}
</style>