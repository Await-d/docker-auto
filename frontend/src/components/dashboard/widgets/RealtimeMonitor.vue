<template>
  <div
    class="realtime-monitor-widget"
    :class="{ 'compact-mode': displayMode === 'compact' }"
  >
    <!-- Header Controls -->
    <div class="monitor-header">
      <div
        class="status-indicator"
        :class="{ connected: isConnected, disconnected: !isConnected }"
      >
        <div class="status-dot" />
        <span class="status-text">{{
          isConnected ? "Live" : "Disconnected"
        }}</span>
      </div>
      <div class="controls">
        <el-dropdown @command="handleTimeRangeChange">
          <el-button size="small" type="text">
            {{ timeRangeLabel }}
            <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="1m"> 1 Minute </el-dropdown-item>
              <el-dropdown-item command="5m"> 5 Minutes </el-dropdown-item>
              <el-dropdown-item command="15m"> 15 Minutes </el-dropdown-item>
              <el-dropdown-item command="1h"> 1 Hour </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button size="small" @click="togglePause" type="text">
          <el-icon>
            <component :is="isPaused ? 'VideoPlay' : 'VideoPause'" />
          </el-icon>
        </el-button>
      </div>
    </div>

    <!-- Metrics Overview -->
    <div class="metrics-overview">
      <div class="metric-item cpu">
        <div class="metric-header">
          <span class="metric-label">CPU</span>
          <span class="metric-value">{{ currentMetrics.cpu.toFixed(1) }}%</span>
        </div>
        <div class="metric-trend" :class="getTrendClass(cpuTrend)">
          <el-icon><component :is="getTrendIcon(cpuTrend)" /></el-icon>
          <span>{{ Math.abs(cpuTrend).toFixed(1) }}%</span>
        </div>
      </div>

      <div class="metric-item memory">
        <div class="metric-header">
          <span class="metric-label">Memory</span>
          <span class="metric-value">{{ memoryPercentage.toFixed(1) }}%</span>
        </div>
        <div class="metric-trend" :class="getTrendClass(memoryTrend)">
          <el-icon><component :is="getTrendIcon(memoryTrend)" /></el-icon>
          <span>{{ Math.abs(memoryTrend).toFixed(1) }}%</span>
        </div>
      </div>

      <div class="metric-item network">
        <div class="metric-header">
          <span class="metric-label">Network</span>
          <span class="metric-value">{{ formatBytes(currentMetrics.network.total) }}/s</span>
        </div>
        <div class="network-breakdown">
          <span class="network-in">↓{{ formatBytes(currentMetrics.network.in) }}</span>
          <span class="network-out">↑{{ formatBytes(currentMetrics.network.out) }}</span>
        </div>
      </div>

      <div class="metric-item disk">
        <div class="metric-header">
          <span class="metric-label">Disk I/O</span>
          <span class="metric-value">{{ formatBytes(currentMetrics.disk.total) }}/s</span>
        </div>
        <div class="disk-breakdown">
          <span class="disk-read">R:{{ formatBytes(currentMetrics.disk.read) }}</span>
          <span class="disk-write">W:{{ formatBytes(currentMetrics.disk.write) }}</span>
        </div>
      </div>
    </div>

    <!-- Live Charts -->
    <div
v-if="displayMode !== 'minimal'" class="charts-section"
>
      <div class="chart-container cpu-chart">
        <div class="chart-title">CPU Usage</div>
        <canvas
ref="cpuChartRef" width="300" height="100" />
      </div>

      <div class="chart-container memory-chart">
        <div class="chart-title">Memory Usage</div>
        <canvas
ref="memoryChartRef" width="300" height="100" />
      </div>
    </div>

    <!-- Activity Feed -->
    <div
v-if="displayMode !== 'compact'" class="activity-feed"
>
      <div class="feed-header">
        <span class="feed-title">Live Activity</span>
        <el-button size="small" type="text" @click="clearFeed">
          <el-icon><Delete /></el-icon>
          Clear
        </el-button>
      </div>
      <div
ref="feedContentRef" class="feed-content"
>
        <div
          v-for="event in activityFeed.slice(0, 50)"
          :key="event.id"
          class="feed-item"
          :class="event.type"
        >
          <div class="event-time">
            {{ formatTime(event.timestamp) }}
          </div>
          <div class="event-type">
            <el-icon>
              <component :is="getEventIcon(event.type)" />
            </el-icon>
          </div>
          <div class="event-message">
            {{ event.message }}
          </div>
          <div
v-if="event.source" class="event-source"
>
            {{ event.source }}
          </div>
        </div>
      </div>
    </div>

    <!-- Container Activity -->
    <div
v-if="displayMode === 'detailed'" class="container-activity"
>
      <div class="activity-header">
        <span class="activity-title">Container Events</span>
        <span class="activity-count">{{ containerEvents.length }} events</span>
      </div>
      <div class="activity-list">
        <div
          v-for="event in containerEvents.slice(0, 5)"
          :key="event.id"
          class="activity-item"
          :class="event.action"
        >
          <div class="activity-icon">
            <el-icon>
              <component :is="getContainerEventIcon(event.action)" />
            </el-icon>
          </div>
          <div class="activity-content">
            <span class="activity-container">{{ event.container }}</span>
            <span class="activity-action">{{ event.action }}</span>
            <span class="activity-time">{{
              formatRelativeTime(event.timestamp)
            }}</span>
          </div>
          <div class="activity-status" :class="event.status">
            <div class="status-dot" />
          </div>
        </div>
      </div>
    </div>

    <!-- System Alerts -->
    <div
v-if="activeAlerts.length > 0" class="system-alerts"
>
      <div class="alerts-header">
        <el-icon class="alert-icon">
          <Warning />
        </el-icon>
        <span class="alerts-title">Active Alerts</span>
        <el-badge :value="activeAlerts.length" type="danger" />
      </div>
      <div class="alerts-list">
        <div
          v-for="alert in activeAlerts.slice(0, 3)"
          :key="alert.id"
          class="alert-item"
          :class="alert.severity"
        >
          <div class="alert-content">
            <span class="alert-message">{{ alert.message }}</span>
            <span class="alert-time">{{
              formatRelativeTime(alert.timestamp)
            }}</span>
          </div>
          <el-button size="small" type="text" @click="dismissAlert(alert.id)">
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <!-- Performance Stats -->
    <div
v-if="displayMode === 'detailed'" class="performance-stats"
>
      <div class="stats-grid">
        <div class="stat-item">
          <span class="stat-label">Containers</span>
          <span class="stat-value">{{ performanceStats.containers }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">Load Avg</span>
          <span class="stat-value">{{
            performanceStats.loadAverage.toFixed(2)
          }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">Processes</span>
          <span class="stat-value">{{ performanceStats.processes }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">Uptime</span>
          <span class="stat-value">{{
            formatUptime(performanceStats.uptime)
          }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from "vue";
import {
  ArrowDown,
  VideoPlay,
  VideoPause,
  Delete,
  Warning,
  Close,
  TrendCharts,
  CaretTop,
  CaretBottom,
  CaretRight,
  Refresh,
  SuccessFilled,
  CircleCloseFilled,
  InfoFilled,
} from "@element-plus/icons-vue";

// Used in dynamic components and conditionally
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = {
  VideoPlay,
  VideoPause,
  TrendCharts,
  CaretTop,
  CaretBottom,
  CaretRight,
  Refresh,
  SuccessFilled,
  CircleCloseFilled,
  InfoFilled,
};

// Props
interface Props {
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
  displayMode?: "default" | "compact" | "detailed" | "minimal";
}

const props = withDefaults(defineProps<Props>(), {
  displayMode: "default",
});

// Emits
const emit = defineEmits<{
  "data-updated": [data: any];
  error: [error: any];
  loading: [loading: boolean];
}>();

// Refs
const cpuChartRef = ref<HTMLCanvasElement>();
const memoryChartRef = ref<HTMLCanvasElement>();
const feedContentRef = ref<HTMLDivElement>();

// Reactive state
const isConnected = ref(true);
const isPaused = ref(false);
const timeRange = ref("5m");
const updateInterval = ref<NodeJS.Timeout>();
const chartUpdateInterval = ref<NodeJS.Timeout>();

const currentMetrics = ref({
  cpu: 45.2,
  memory: {
    used: 3221225472, // 3GB
    total: 8589934592, // 8GB
  },
  network: {
    in: 1048576, // 1MB/s
    out: 524288, // 512KB/s
    total: 1572864, // 1.5MB/s
  },
  disk: {
    read: 2097152, // 2MB/s
    write: 1048576, // 1MB/s
    total: 3145728, // 3MB/s
  },
});

const previousMetrics = ref({ ...currentMetrics.value });

const activityFeed = ref([
  {
    id: "1",
    type: "container",
    message: "Container web-server started",
    source: "docker",
    timestamp: new Date(),
  },
  {
    id: "2",
    type: "network",
    message: "High network traffic detected",
    source: "monitoring",
    timestamp: new Date(Date.now() - 30000),
  },
  {
    id: "3",
    type: "system",
    message: "System backup completed",
    source: "system",
    timestamp: new Date(Date.now() - 60000),
  },
]);

const containerEvents = ref([
  {
    id: "1",
    container: "web-server-1",
    action: "started",
    status: "success",
    timestamp: new Date(),
  },
  {
    id: "2",
    container: "database",
    action: "restarted",
    status: "success",
    timestamp: new Date(Date.now() - 120000),
  },
  {
    id: "3",
    container: "cache",
    action: "stopped",
    status: "warning",
    timestamp: new Date(Date.now() - 180000),
  },
]);

const activeAlerts = ref([
  {
    id: "1",
    severity: "warning",
    message: "High CPU usage on container web-server",
    timestamp: new Date(Date.now() - 300000),
  },
  {
    id: "2",
    severity: "critical",
    message: "Container database unhealthy",
    timestamp: new Date(Date.now() - 600000),
  },
]);

const performanceStats = ref({
  containers: 12,
  loadAverage: 1.45,
  processes: 156,
  uptime: 86400000, // 1 day in ms
});

// Chart data
const cpuHistory = ref<number[]>([]);
const memoryHistory = ref<number[]>([]);
const maxDataPoints = computed(() => {
  switch (timeRange.value) {
    case "1m":
      return 60;
    case "5m":
      return 300;
    case "15m":
      return 900;
    case "1h":
      return 3600;
    default:
      return 300;
  }
});

// Computed properties
const timeRangeLabel = computed(() => {
  switch (timeRange.value) {
    case "1m":
      return "1 Minute";
    case "5m":
      return "5 Minutes";
    case "15m":
      return "15 Minutes";
    case "1h":
      return "1 Hour";
    default:
      return "5 Minutes";
  }
});

const memoryPercentage = computed(() => {
  const { used, total } = currentMetrics.value.memory;
  return (used / total) * 100;
});

const cpuTrend = computed(() => {
  return currentMetrics.value.cpu - previousMetrics.value.cpu;
});

const memoryTrend = computed(() => {
  const current = memoryPercentage.value;
  const previous =
    (previousMetrics.value.memory.used / previousMetrics.value.memory.total) *
    100;
  return current - previous;
});

// Methods
const generateMockData = () => {
  // Simulate realistic metrics with some variation
  const cpuVariation = (Math.random() - 0.5) * 10;
  const memoryVariation = (Math.random() - 0.5) * 0.1;
  const networkVariation = (Math.random() - 0.5) * 0.5;
  const diskVariation = (Math.random() - 0.5) * 0.3;

  previousMetrics.value = { ...currentMetrics.value };

  currentMetrics.value = {
    cpu: Math.max(0, Math.min(100, currentMetrics.value.cpu + cpuVariation)),
    memory: {
      used: Math.max(
        0,
        currentMetrics.value.memory.used +
          currentMetrics.value.memory.used * memoryVariation,
      ),
      total: currentMetrics.value.memory.total,
    },
    network: {
      in: Math.max(0, currentMetrics.value.network.in * (1 + networkVariation)),
      out: Math.max(
        0,
        currentMetrics.value.network.out * (1 + networkVariation),
      ),
      total: 0,
    },
    disk: {
      read: Math.max(0, currentMetrics.value.disk.read * (1 + diskVariation)),
      write: Math.max(0, currentMetrics.value.disk.write * (1 + diskVariation)),
      total: 0,
    },
  };

  // Update totals
  currentMetrics.value.network.total =
    currentMetrics.value.network.in + currentMetrics.value.network.out;
  currentMetrics.value.disk.total =
    currentMetrics.value.disk.read + currentMetrics.value.disk.write;

  // Add to chart data
  cpuHistory.value.push(currentMetrics.value.cpu);
  memoryHistory.value.push(memoryPercentage.value);

  // Trim data to max points
  if (cpuHistory.value.length > maxDataPoints.value) {
    cpuHistory.value = cpuHistory.value.slice(-maxDataPoints.value);
  }
  if (memoryHistory.value.length > maxDataPoints.value) {
    memoryHistory.value = memoryHistory.value.slice(-maxDataPoints.value);
  }

  // Occasionally add activity events
  if (Math.random() < 0.1) {
    addRandomActivityEvent();
  }
};

const addRandomActivityEvent = () => {
  const events = [
    {
      type: "container",
      message: "Container health check passed",
      source: "docker",
    },
    {
      type: "network",
      message: "Network connection established",
      source: "networking",
    },
    { type: "system", message: "Disk cleanup completed", source: "system" },
    {
      type: "security",
      message: "Security scan completed",
      source: "security",
    },
  ];

  const randomEvent = events[Math.floor(Math.random() * events.length)];
  activityFeed.value.unshift({
    id: Date.now().toString(),
    ...randomEvent,
    timestamp: new Date(),
  });

  // Limit feed size
  if (activityFeed.value.length > 100) {
    activityFeed.value = activityFeed.value.slice(0, 100);
  }

  // Auto-scroll feed
  nextTick(() => {
    if (feedContentRef.value) {
      feedContentRef.value.scrollTop = 0;
    }
  });
};

const drawChart = (
  canvas: HTMLCanvasElement,
  data: number[],
  color: string,
  _label: string,
) => {
  if (!canvas) return;

  const ctx = canvas.getContext("2d");
  if (!ctx) return;

  const width = canvas.width;
  const height = canvas.height;
  const padding = 10;

  ctx.clearRect(0, 0, width, height);

  if (data.length < 2) return;

  // Draw background grid
  ctx.strokeStyle = "#f0f0f0";
  ctx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const y = padding + (i * (height - 2 * padding)) / 4;
    ctx.beginPath();
    ctx.moveTo(padding, y);
    ctx.lineTo(width - padding, y);
    ctx.stroke();
  }

  // Draw data line
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  ctx.beginPath();

  const stepX = (width - 2 * padding) / (data.length - 1);
  const maxValue = Math.max(...data, 100); // Ensure min scale of 100
  const minValue = Math.min(...data, 0);
  const range = maxValue - minValue || 1;

  data.forEach((value, index) => {
    const x = padding + index * stepX;
    const y =
      height - padding - ((value - minValue) / range) * (height - 2 * padding);

    if (index === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  });

  ctx.stroke();

  // Draw fill area
  ctx.fillStyle = color + "20"; // Add transparency
  ctx.lineTo(width - padding, height - padding);
  ctx.lineTo(padding, height - padding);
  ctx.closePath();
  ctx.fill();
};

const updateCharts = () => {
  if (cpuChartRef.value) {
    drawChart(cpuChartRef.value, cpuHistory.value, "#409eff", "CPU");
  }
  if (memoryChartRef.value) {
    drawChart(memoryChartRef.value, memoryHistory.value, "#67c23a", "Memory");
  }
};

const getTrendClass = (trend: number) => {
  if (trend > 1) return "trend-up";
  if (trend < -1) return "trend-down";
  return "trend-stable";
};

const getTrendIcon = (trend: number) => {
  if (trend > 1) return "CaretTop";
  if (trend < -1) return "CaretBottom";
  return "CaretRight";
};

const getEventIcon = (type: string) => {
  switch (type) {
    case "container":
      return "Box";
    case "network":
      return "Connection";
    case "system":
      return "Monitor";
    case "security":
      return "Lock";
    default:
      return "InfoFilled";
  }
};

const getContainerEventIcon = (action: string) => {
  switch (action) {
    case "started":
      return "CaretRight";
    case "stopped":
      return "VideoPause";
    case "restarted":
      return "Refresh";
    default:
      return "InfoFilled";
  }
};

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
};

const formatTime = (date: Date): string => {
  return date.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);

  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  return date.toLocaleDateString();
};

const formatUptime = (uptimeMs: number): string => {
  const days = Math.floor(uptimeMs / (24 * 60 * 60 * 1000));
  const hours = Math.floor(
    (uptimeMs % (24 * 60 * 60 * 1000)) / (60 * 60 * 1000),
  );

  if (days > 0) return `${days}d ${hours}h`;
  return `${hours}h`;
};

const handleTimeRangeChange = (command: string) => {
  timeRange.value = command;
  // Clear existing data and restart with new range
  cpuHistory.value = [];
  memoryHistory.value = [];
};

const togglePause = () => {
  isPaused.value = !isPaused.value;
  if (isPaused.value) {
    clearInterval(updateInterval.value);
    clearInterval(chartUpdateInterval.value);
  } else {
    startUpdates();
  }
};

const clearFeed = () => {
  activityFeed.value = [];
};

const dismissAlert = (alertId: string) => {
  const index = activeAlerts.value.findIndex((alert) => alert.id === alertId);
  if (index !== -1) {
    activeAlerts.value.splice(index, 1);
  }
};

const startUpdates = () => {
  updateInterval.value = setInterval(() => {
    if (!isPaused.value) {
      generateMockData();
      emit("data-updated", {
        metrics: currentMetrics.value,
        activityFeed: activityFeed.value,
        alerts: activeAlerts.value,
      });
    }
  }, 1000);

  chartUpdateInterval.value = setInterval(() => {
    if (!isPaused.value) {
      updateCharts();
    }
  }, 1000);
};

// Lifecycle hooks
onMounted(() => {
  startUpdates();

  // Initialize with some data
  for (let i = 0; i < 30; i++) {
    generateMockData();
  }

  nextTick(() => {
    updateCharts();
  });
});

onUnmounted(() => {
  clearInterval(updateInterval.value);
  clearInterval(chartUpdateInterval.value);
});

// Watch for widget data changes
watch(
  () => props.widgetData,
  (newData) => {
    if (newData) {
      if (newData.metrics) {
        currentMetrics.value = { ...currentMetrics.value, ...newData.metrics };
      }
      if (newData.activityFeed) {
        activityFeed.value = newData.activityFeed;
      }
    }
  },
  { deep: true },
);
</script>

<style scoped lang="scss">
.realtime-monitor-widget {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;

  &.compact-mode {
    padding: 12px;
    gap: 12px;
  }
}

.monitor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 8px;

    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      animation: pulse 2s infinite;
    }

    &.connected .status-dot {
      background: var(--el-color-success);
    }

    &.disconnected .status-dot {
      background: var(--el-color-danger);
    }

    .status-text {
      font-size: 12px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

.metrics-overview {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;

  .metric-item {
    background: var(--el-fill-color-extra-light);
    border-radius: 8px;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);

    .metric-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 8px;

      .metric-label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
      }

      .metric-value {
        font-size: 16px;
        font-weight: 700;
        color: var(--el-color-primary);
      }
    }

    .metric-trend {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 11px;

      &.trend-up {
        color: var(--el-color-danger);
      }

      &.trend-down {
        color: var(--el-color-success);
      }

      &.trend-stable {
        color: var(--el-text-color-placeholder);
      }
    }

    .network-breakdown,
    .disk-breakdown {
      display: flex;
      justify-content: space-between;
      font-size: 10px;
      color: var(--el-text-color-secondary);

      .network-in,
      .disk-read {
        color: var(--el-color-success);
      }

      .network-out,
      .disk-write {
        color: var(--el-color-warning);
      }
    }
  }
}

.charts-section {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;

  .chart-container {
    background: var(--el-fill-color-extra-light);
    border-radius: 8px;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);

    .chart-title {
      font-size: 12px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin-bottom: 8px;
    }

    canvas {
      width: 100%;
      height: 100px;
    }
  }
}

.activity-feed {
  .feed-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    .feed-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .feed-content {
    max-height: 200px;
    overflow-y: auto;
    background: var(--el-fill-color-extra-light);
    border-radius: 6px;
    padding: 8px;

    .feed-item {
      display: grid;
      grid-template-columns: auto auto 1fr auto;
      gap: 8px;
      align-items: center;
      padding: 4px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);
      font-size: 11px;

      &:last-child {
        border-bottom: none;
      }

      .event-time {
        color: var(--el-text-color-placeholder);
        font-family: monospace;
      }

      .event-type {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 16px;
        height: 16px;
        border-radius: 50%;
        font-size: 10px;

        &.container {
          background: rgba(var(--el-color-primary-rgb), 0.2);
          color: var(--el-color-primary);
        }

        &.network {
          background: rgba(var(--el-color-success-rgb), 0.2);
          color: var(--el-color-success);
        }

        &.system {
          background: rgba(var(--el-color-warning-rgb), 0.2);
          color: var(--el-color-warning);
        }

        &.security {
          background: rgba(var(--el-color-danger-rgb), 0.2);
          color: var(--el-color-danger);
        }
      }

      .event-message {
        color: var(--el-text-color-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .event-source {
        color: var(--el-text-color-secondary);
        font-size: 10px;
      }
    }
  }
}

.container-activity {
  .activity-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    .activity-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .activity-count {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .activity-list {
    .activity-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: none;
      }

      .activity-icon {
        flex-shrink: 0;
        width: 20px;
        height: 20px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 12px;

        &.started {
          background: rgba(var(--el-color-success-rgb), 0.2);
          color: var(--el-color-success);
        }

        &.stopped {
          background: rgba(var(--el-color-warning-rgb), 0.2);
          color: var(--el-color-warning);
        }

        &.restarted {
          background: rgba(var(--el-color-info-rgb), 0.2);
          color: var(--el-color-info);
        }
      }

      .activity-content {
        flex: 1;

        .activity-container {
          display: block;
          font-size: 12px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        .activity-action {
          display: block;
          font-size: 11px;
          color: var(--el-text-color-secondary);
          text-transform: capitalize;
        }

        .activity-time {
          display: block;
          font-size: 10px;
          color: var(--el-text-color-placeholder);
        }
      }

      .activity-status {
        flex-shrink: 0;

        .status-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
        }

        &.success .status-dot {
          background: var(--el-color-success);
        }

        &.warning .status-dot {
          background: var(--el-color-warning);
        }

        &.error .status-dot {
          background: var(--el-color-danger);
        }
      }
    }
  }
}

.system-alerts {
  background: rgba(var(--el-color-danger-rgb), 0.05);
  border: 1px solid rgba(var(--el-color-danger-rgb), 0.2);
  border-radius: 8px;
  padding: 12px;

  .alerts-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .alert-icon {
      color: var(--el-color-danger);
    }

    .alerts-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--el-color-danger);
    }
  }

  .alerts-list {
    .alert-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
      padding: 6px 0;
      border-bottom: 1px solid rgba(var(--el-color-danger-rgb), 0.1);

      &:last-child {
        border-bottom: none;
      }

      .alert-content {
        flex: 1;

        .alert-message {
          display: block;
          font-size: 12px;
          color: var(--el-text-color-primary);
          font-weight: 500;
        }

        .alert-time {
          font-size: 10px;
          color: var(--el-text-color-secondary);
        }
      }
    }
  }
}

.performance-stats {
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;

    .stat-item {
      text-align: center;
      padding: 8px;
      background: var(--el-fill-color-extra-light);
      border-radius: 6px;

      .stat-label {
        display: block;
        font-size: 10px;
        color: var(--el-text-color-secondary);
        margin-bottom: 4px;
      }

      .stat-value {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-color-primary);
      }
    }
  }
}

// Animations
@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

// Responsive design
@media (max-width: 480px) {
  .realtime-monitor-widget {
    .metrics-overview {
      grid-template-columns: 1fr;
    }

    .performance-stats .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
}
</style>
