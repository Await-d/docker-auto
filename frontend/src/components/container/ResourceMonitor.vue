<template>
  <div class="resource-monitor" :class="{ 'detailed-view': detailedView }">
    <!-- Real-time Stats -->
    <div class="current-stats">
      <div class="stats-grid">
        <!-- CPU Usage -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Monitor /></el-icon>
              CPU Usage
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

        <!-- Memory Usage -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Cpu /></el-icon>
              Memory Usage
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

        <!-- Network I/O -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Connection /></el-icon>
              Network I/O
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

        <!-- Disk I/O -->
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-title">
              <el-icon><Folder /></el-icon>
              Disk I/O
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
          Last updated: {{ formatTime(currentStats?.timestamp) }}
        </div>
        <div class="update-controls">
          <el-button
size="small" @click="refreshStats"
:loading="loading"
>
            <el-icon><Refresh /></el-icon>
            Refresh
          </el-button>
          <el-button
            size="small"
            :type="autoRefresh ? 'primary' : 'default'"
            @click="toggleAutoRefresh"
          >
            <el-icon><Timer /></el-icon>
            Auto Refresh
          </el-button>
        </div>
      </div>
    </div>

    <!-- Historical Charts -->
    <div v-if="showHistorical" class="historical-charts">
      <div class="charts-header">
        <h3>Historical Data</h3>
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
          <h4>CPU Usage Over Time</h4>
          <div
ref="cpuChartRef" class="chart" />
        </div>

        <!-- Memory Chart -->
        <div class="chart-container">
          <h4>Memory Usage Over Time</h4>
          <div
ref="memoryChartRef" class="chart" />
        </div>

        <!-- Network Chart -->
        <div class="chart-container">
          <h4>Network I/O Over Time</h4>
          <div
ref="networkChartRef" class="chart" />
        </div>

        <!-- Disk Chart -->
        <div class="chart-container">
          <h4>Disk I/O Over Time</h4>
          <div
ref="diskChartRef" class="chart" />
        </div>
      </div>
    </div>

    <!-- Detailed Metrics -->
    <div v-if="detailedView" class="detailed-metrics">
      <div class="metrics-header">
        <h3>Detailed Metrics</h3>
        <el-button size="small" @click="exportMetrics">
          <el-icon><Download /></el-icon>
          Export Data
        </el-button>
      </div>

      <el-tabs v-model="activeMetricTab">
        <!-- System Metrics -->
        <el-tab-pane label="System" name="system">
          <div class="metrics-table">
            <el-table :data="systemMetrics" stripe>
              <el-table-column prop="metric" label="Metric" />
              <el-table-column prop="value" label="Current Value" />
              <el-table-column prop="unit" label="Unit" />
              <el-table-column prop="description" label="Description" />
            </el-table>
          </div>
        </el-tab-pane>

        <!-- Process Metrics -->
        <el-tab-pane label="Processes" name="processes">
          <div class="process-metrics">
            <p>Process-level metrics will be displayed here</p>
          </div>
        </el-tab-pane>

        <!-- Network Metrics -->
        <el-tab-pane label="Network" name="network">
          <div class="network-metrics">
            <el-table :data="networkMetrics" stripe>
              <el-table-column prop="interface" label="Interface" />
              <el-table-column prop="rxBytes" label="RX Bytes" />
              <el-table-column prop="txBytes" label="TX Bytes" />
              <el-table-column prop="rxPackets" label="RX Packets" />
              <el-table-column prop="txPackets" label="TX Packets" />
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- Alerts -->
    <div v-if="alerts.length > 0" class="resource-alerts">
      <h4>Resource Alerts</h4>
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
const { stats, historicalStats } = storeToRefs(containerStore);

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

// Auto-refresh interval
let refreshInterval: NodeJS.Timeout | null = null;

// Computed
const currentStats = computed(() => {
  return stats.value.get(props.containerId);
});

const containerHistoricalStats = computed(() => {
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
      metric: "CPU Usage",
      value: formatPercentage(currentStats.value.cpu.usage),
      unit: "%",
      description: "Current CPU utilization",
    },
    {
      metric: "Memory Usage",
      value: formatBytes(currentStats.value.memory.usage),
      unit: "Bytes",
      description: "Current memory consumption",
    },
    {
      metric: "Memory Percentage",
      value: formatPercentage(currentStats.value.memory.percentage),
      unit: "%",
      description: "Memory usage as percentage of limit",
    },
    {
      metric: "Network RX",
      value: formatBytes(currentStats.value.network.rxBytes),
      unit: "Bytes",
      description: "Total bytes received",
    },
    {
      metric: "Network TX",
      value: formatBytes(currentStats.value.network.txBytes),
      unit: "Bytes",
      description: "Total bytes transmitted",
    },
    {
      metric: "Disk Read",
      value: formatBytes(currentStats.value.disk.readBytes),
      unit: "Bytes",
      description: "Total bytes read from disk",
    },
    {
      metric: "Disk Write",
      value: formatBytes(currentStats.value.disk.writeBytes),
      unit: "Bytes",
      description: "Total bytes written to disk",
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
    await containerStore.fetchStats(props.containerId);
    checkAlerts();
  } catch (error) {
    console.error("Failed to refresh stats:", error);
    ElMessage.error("Failed to refresh statistics");
  } finally {
    loading.value = false;
  }
}

async function fetchHistoricalData() {
  if (!props.showHistorical) return;

  try {
    await containerStore.fetchHistoricalStats(
      props.containerId,
      timeRange.value,
      getIntervalForRange(timeRange.value),
    );
    await nextTick();
    renderCharts();
  } catch (error) {
    console.error("Failed to fetch historical data:", error);
    ElMessage.error("Failed to load historical data");
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

  refreshInterval = setInterval(() => {
    refreshStats();
  }, 10000); // Refresh every 10 seconds
}

function stopAutoRefresh() {
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
        title: "High CPU Usage",
        message: `CPU usage is at ${formatPercentage(currentStats.value.cpu.usage)}`,
      });
    }

    // Memory alert
    if (currentStats.value.memory.percentage > 85) {
      newAlerts.push({
        id: "memory-high",
        type: "error",
        title: "High Memory Usage",
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

    ElMessage.success("Metrics exported successfully");
  } catch (error) {
    console.error("Failed to export metrics:", error);
    ElMessage.error("Failed to export metrics");
  }
}

function renderCharts() {
  // This would integrate with a charting library like Chart.js or ECharts
  // For now, we'll just log that charts would be rendered
  console.log("Rendering charts with data:", containerHistoricalStats.value);

  // Example Chart.js integration:
  /*
  if (cpuChartRef.value) {
    new Chart(cpuChartRef.value, {
      type: 'line',
      data: {
        labels: containerHistoricalStats.value.map(stat =>
          new Date(stat.timestamp).toLocaleTimeString()
        ),
        datasets: [{
          label: 'CPU Usage (%)',
          data: containerHistoricalStats.value.map(stat => stat.metrics.cpu.usage),
          borderColor: '#409eff',
          backgroundColor: 'rgba(64, 158, 255, 0.1)',
          tension: 0.4
        }]
      },
      options: {
        responsive: true,
        scales: {
          y: {
            beginAtZero: true,
            max: 100
          }
        }
      }
    })
  }
  */
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
});

onUnmounted(() => {
  stopAutoRefresh();
});

// Watch for container changes
watch(
  () => props.containerId,
  (newId) => {
    if (newId) {
      refreshStats();
      if (props.showHistorical) {
        fetchHistoricalData();
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
  background: #f8f9fa;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  font-size: 12px;
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
