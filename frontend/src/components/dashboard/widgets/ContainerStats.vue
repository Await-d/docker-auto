<template>
  <div
    class="container-stats-widget"
    :class="{ 'compact-mode': displayMode === 'compact' }"
  >
    <!-- Header Section -->
    <div class="stats-header">
      <div class="total-containers">
        <span class="total-count">{{ containerData.total }}</span>
        <span class="total-label">Total Containers</span>
      </div>
      <div class="status-summary">
        <div class="status-item running">
          <span class="status-count">{{ containerData.running }}</span>
          <span class="status-label">Running</span>
        </div>
        <div class="status-item stopped">
          <span class="status-count">{{ containerData.stopped }}</span>
          <span class="status-label">Stopped</span>
        </div>
        <div
v-if="containerData.error > 0" class="status-item error"
>
          <span class="status-count">{{ containerData.error }}</span>
          <span class="status-label">Error</span>
        </div>
      </div>
    </div>

    <!-- Status Distribution Chart -->
    <div
v-if="displayMode !== 'minimal'" class="chart-section"
>
      <div class="chart-container">
        <div class="chart-title">Container Status Distribution</div>
        <!-- Donut Chart would be implemented with a chart library like Chart.js or ECharts -->
        <div
ref="chartRef" class="donut-chart"
>
          <canvas
ref="canvasRef" width="200" height="200" />
        </div>
        <div class="chart-legend">
          <div
            v-for="item in chartData"
            :key="item.label"
            class="legend-item"
            :style="{ color: item.color }"
          >
            <div class="legend-dot" :style="{ backgroundColor: item.color }" />
            <span class="legend-label">{{ item.label }}</span>
            <span class="legend-value">{{ item.value }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Container Categories -->
    <div class="categories-section">
      <div class="category-header">
        <span class="category-title">By Image</span>
        <el-button size="small" type="text" @click="showAllImages">
          View All
        </el-button>
      </div>
      <div class="category-list">
        <div
v-for="image in topImages" :key="image.name"
class="category-item"
>
          <div class="category-info">
            <span class="category-name">{{ image.name }}</span>
            <span class="category-tag">{{ image.tag }}</span>
          </div>
          <div class="category-stats">
            <span class="category-count">{{ image.containers }}</span>
            <div class="category-status">
              <span
                class="running-dots"
                :style="{
                  width: `${(image.running / image.containers) * 100}%`,
                }"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Registry Distribution -->
    <div
v-if="displayMode === 'detailed'" class="registry-section"
>
      <div class="registry-header">
        <span class="registry-title">By Registry</span>
      </div>
      <div class="registry-list">
        <div
          v-for="registry in registryStats"
          :key="registry.name"
          class="registry-item"
        >
          <div class="registry-icon">
            <el-icon><Box /></el-icon>
          </div>
          <div class="registry-info">
            <span class="registry-name">{{ registry.name }}</span>
            <span class="registry-url">{{ registry.url }}</span>
          </div>
          <div class="registry-count">
            <span class="count-value">{{ registry.containers }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Resource Usage Top Containers -->
    <div
v-if="displayMode !== 'minimal'" class="resource-section"
>
      <div class="resource-header">
        <span class="resource-title">Resource Usage (Top 5)</span>
        <el-dropdown @command="handleSortChange">
          <el-button size="small" type="text">
            Sort by {{ sortBy }}
            <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="cpu"> CPU Usage </el-dropdown-item>
              <el-dropdown-item command="memory">
                Memory Usage
              </el-dropdown-item>
              <el-dropdown-item command="network">
                Network I/O
              </el-dropdown-item>
              <el-dropdown-item command="disk"> Disk I/O </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <div class="resource-list">
        <div
          v-for="container in topResourceContainers"
          :key="container.id"
          class="resource-item"
          @click="viewContainerDetails(container.id)"
        >
          <div class="container-info">
            <div class="container-name">
              {{ container.name }}
            </div>
            <div class="container-image">
              {{ container.image }}
            </div>
          </div>
          <div class="container-metrics">
            <div class="metric cpu">
              <span class="metric-label">CPU</span>
              <span class="metric-value">{{ container.cpu.toFixed(1) }}%</span>
              <div class="metric-bar">
                <div
                  class="metric-fill"
                  :style="{
                    width: `${Math.min(container.cpu, 100)}%`,
                    backgroundColor: getResourceColor(container.cpu),
                  }"
                />
              </div>
            </div>
            <div class="metric memory">
              <span class="metric-label">RAM</span>
              <span class="metric-value">{{
                formatBytes(container.memory)
              }}</span>
              <div class="metric-bar">
                <div
                  class="metric-fill"
                  :style="{
                    width: `${Math.min(container.memoryPercent, 100)}%`,
                    backgroundColor: getResourceColor(container.memoryPercent),
                  }"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div
v-if="displayMode === 'detailed'" class="actions-section"
>
      <el-button-group size="small">
        <el-button @click="startAllStopped">
          <el-icon><CaretRight /></el-icon>
          Start All Stopped
        </el-button>
        <el-button @click="pruneContainers">
          <el-icon><Delete /></el-icon>
          Prune Unused
        </el-button>
        <el-button @click="viewAllContainers">
          <el-icon><View /></el-icon>
          View All
        </el-button>
      </el-button-group>
    </div>

    <!-- Health Check Status -->
    <div
      v-if="containerData.healthChecks && displayMode !== 'minimal'"
      class="health-section"
    >
      <div class="health-header">
        <span class="health-title">Health Checks</span>
        <span class="health-summary">{{ healthySummary }}</span>
      </div>
      <div class="health-grid">
        <div
          v-for="health in containerData.healthChecks"
          :key="health.container"
          class="health-item"
          :class="health.status"
        >
          <el-icon class="health-icon">
            <component :is="getHealthIcon(health.status)" />
          </el-icon>
          <span class="health-container">{{ health.container }}</span>
          <span class="health-status">{{ health.status }}</span>
        </div>
      </div>
    </div>

    <!-- Recent Events -->
    <div
      v-if="displayMode === 'detailed' && containerData.recentEvents"
      class="events-section"
    >
      <div class="events-header">
        <span class="events-title">Recent Events</span>
        <span class="events-count">{{
          containerData.recentEvents.length
        }}</span>
      </div>
      <div class="events-list">
        <div
          v-for="event in containerData.recentEvents.slice(0, 3)"
          :key="event.id"
          class="event-item"
        >
          <div class="event-type" :class="event.type">
            <el-icon>
              <component :is="getEventIcon(event.type)" />
            </el-icon>
          </div>
          <div class="event-content">
            <span class="event-container">{{ event.container }}</span>
            <span class="event-action">{{ event.action }}</span>
            <span class="event-time">{{
              formatRelativeTime(event.timestamp)
            }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from "vue";
import { ElMessage } from "element-plus";
import {
  Box,
  ArrowDown,
  CaretRight,
  Delete,
  View,
  Warning,
  SuccessFilled,
  CircleCloseFilled,
  InfoFilled,
} from "@element-plus/icons-vue";

// Icons used in dynamic template components - create reference object for TypeScript
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = {
  Warning,
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
const chartRef = ref<HTMLDivElement>();
const canvasRef = ref<HTMLCanvasElement>();

// Reactive state
const sortBy = ref("cpu");
const containerData = ref({
  total: 18,
  running: 14,
  stopped: 3,
  error: 1,
  healthChecks: [
    { container: "web-server", status: "healthy" },
    { container: "database", status: "healthy" },
    { container: "cache", status: "unhealthy" },
    { container: "worker", status: "healthy" },
  ],
  recentEvents: [
    {
      id: "1",
      container: "web-server",
      action: "Container started",
      type: "start",
      timestamp: new Date(Date.now() - 300000),
    },
    {
      id: "2",
      container: "cache",
      action: "Health check failed",
      type: "health",
      timestamp: new Date(Date.now() - 600000),
    },
    {
      id: "3",
      container: "worker",
      action: "Container restarted",
      type: "restart",
      timestamp: new Date(Date.now() - 900000),
    },
  ],
});

const topImages = ref([
  { name: "nginx", tag: "latest", containers: 4, running: 4 },
  { name: "postgres", tag: "14", containers: 2, running: 2 },
  { name: "redis", tag: "7-alpine", containers: 2, running: 1 },
  { name: "node", tag: "18-alpine", containers: 3, running: 3 },
  { name: "python", tag: "3.11", containers: 2, running: 2 },
]);

const registryStats = ref([
  { name: "Docker Hub", url: "docker.io", containers: 12 },
  { name: "Private Registry", url: "registry.local", containers: 4 },
  { name: "GitHub Packages", url: "ghcr.io", containers: 2 },
]);

const resourceContainers = ref([
  {
    id: "cont1",
    name: "web-server-1",
    image: "nginx:latest",
    cpu: 45.2,
    memory: 536870912, // 512MB
    memoryPercent: 25,
    network: { rx: 1024000, tx: 2048000 },
    disk: { read: 1024000, write: 512000 },
  },
  {
    id: "cont2",
    name: "database-primary",
    image: "postgres:14",
    cpu: 32.8,
    memory: 1073741824, // 1GB
    memoryPercent: 50,
    network: { rx: 512000, tx: 1024000 },
    disk: { read: 2048000, write: 1024000 },
  },
  {
    id: "cont3",
    name: "cache-redis",
    image: "redis:7-alpine",
    cpu: 15.6,
    memory: 268435456, // 256MB
    memoryPercent: 12,
    network: { rx: 256000, tx: 512000 },
    disk: { read: 128000, write: 64000 },
  },
  {
    id: "cont4",
    name: "worker-queue",
    image: "node:18-alpine",
    cpu: 28.4,
    memory: 805306368, // 768MB
    memoryPercent: 38,
    network: { rx: 128000, tx: 256000 },
    disk: { read: 512000, write: 256000 },
  },
  {
    id: "cont5",
    name: "api-service",
    image: "python:3.11",
    cpu: 22.1,
    memory: 671088640, // 640MB
    memoryPercent: 32,
    network: { rx: 384000, tx: 768000 },
    disk: { read: 256000, write: 128000 },
  },
]);

// Computed properties
const chartData = computed(() => [
  { label: "Running", value: containerData.value.running, color: "#67c23a" },
  { label: "Stopped", value: containerData.value.stopped, color: "#e6a23c" },
  { label: "Error", value: containerData.value.error, color: "#f56c6c" },
]);

const topResourceContainers = computed(() => {
  const containers = [...resourceContainers.value];
  containers.sort((a, b) => {
    switch (sortBy.value) {
      case "cpu":
        return b.cpu - a.cpu;
      case "memory":
        return b.memory - a.memory;
      case "network":
        return b.network.rx + b.network.tx - (a.network.rx + a.network.tx);
      case "disk":
        return b.disk.read + b.disk.write - (a.disk.read + a.disk.write);
      default:
        return 0;
    }
  });
  return containers.slice(0, 5);
});

const healthySummary = computed(() => {
  const healthy =
    containerData.value.healthChecks?.filter((h) => h.status === "healthy")
      .length || 0;
  const total = containerData.value.healthChecks?.length || 0;
  return `${healthy}/${total} healthy`;
});

// Methods
const drawDonutChart = () => {
  if (!canvasRef.value) return;

  const canvas = canvasRef.value;
  const ctx = canvas.getContext("2d");
  if (!ctx) return;

  const centerX = canvas.width / 2;
  const centerY = canvas.height / 2;
  const radius = 70;
  const innerRadius = 40;

  let startAngle = 0;
  const total = chartData.value.reduce((sum, item) => sum + item.value, 0);

  ctx.clearRect(0, 0, canvas.width, canvas.height);

  chartData.value.forEach((item) => {
    const sliceAngle = (item.value / total) * 2 * Math.PI;

    // Draw outer arc
    ctx.beginPath();
    ctx.arc(centerX, centerY, radius, startAngle, startAngle + sliceAngle);
    ctx.arc(
      centerX,
      centerY,
      innerRadius,
      startAngle + sliceAngle,
      startAngle,
      true,
    );
    ctx.closePath();
    ctx.fillStyle = item.color;
    ctx.fill();

    startAngle += sliceAngle;
  });

  // Draw center text
  ctx.fillStyle = "#333";
  ctx.font = "bold 16px Arial";
  ctx.textAlign = "center";
  ctx.fillText(total.toString(), centerX, centerY - 5);
  ctx.font = "12px Arial";
  ctx.fillText("Total", centerX, centerY + 15);
};

const getResourceColor = (percentage: number): string => {
  if (percentage < 50) return "#67c23a";
  if (percentage < 80) return "#e6a23c";
  return "#f56c6c";
};

const getHealthIcon = (status: string) => {
  switch (status) {
    case "healthy":
      return "SuccessFilled";
    case "unhealthy":
      return "CircleCloseFilled";
    case "starting":
      return "InfoFilled";
    default:
      return "Warning";
  }
};

const getEventIcon = (type: string) => {
  switch (type) {
    case "start":
      return "CaretRight";
    case "stop":
      return "CircleCloseFilled";
    case "restart":
      return "Refresh";
    case "health":
      return "Warning";
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

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);

  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  return date.toLocaleDateString();
};

const handleSortChange = (command: string) => {
  sortBy.value = command;
};

const showAllImages = () => {
  ElMessage.info("Opening images view...");
};

const viewContainerDetails = (containerId: string) => {
  ElMessage.info(`Opening details for container: ${containerId}`);
};

const startAllStopped = async () => {
  try {
    ElMessage.info("Starting all stopped containers...");
    // Implementation for starting containers
  } catch (error) {
    ElMessage.error("Failed to start containers");
  }
};

const pruneContainers = async () => {
  try {
    ElMessage.info("Pruning unused containers...");
    // Implementation for pruning containers
  } catch (error) {
    ElMessage.error("Failed to prune containers");
  }
};

const viewAllContainers = () => {
  ElMessage.info("Opening containers view...");
};

const fetchContainerStats = async () => {
  try {
    emit("loading", true);

    // Simulate API call
    const response = await fetch("/api/v1/dashboard/container-stats");
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data = await response.json();
    containerData.value = { ...containerData.value, ...data };

    emit("data-updated", containerData.value);

    // Redraw chart after data update
    await nextTick();
    drawDonutChart();
  } catch (error) {
    console.error("Failed to fetch container stats:", error);
    emit("error", error);
  } finally {
    emit("loading", false);
  }
};

// Lifecycle hooks
onMounted(async () => {
  await fetchContainerStats();

  // Draw initial chart
  await nextTick();
  drawDonutChart();

  // Set up auto-refresh
  const refreshInterval = props.widgetConfig?.refreshInterval || 15000;
  if (refreshInterval > 0) {
    const interval = setInterval(fetchContainerStats, refreshInterval);
    onUnmounted(() => clearInterval(interval));
  }
});
</script>

<style scoped lang="scss">
.container-stats-widget {
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

.stats-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .total-containers {
    text-align: center;

    .total-count {
      display: block;
      font-size: 28px;
      font-weight: 700;
      color: var(--el-color-primary);
    }

    .total-label {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .status-summary {
    display: flex;
    gap: 16px;

    .status-item {
      text-align: center;

      .status-count {
        display: block;
        font-size: 18px;
        font-weight: 600;
      }

      .status-label {
        font-size: 11px;
        color: var(--el-text-color-secondary);
      }

      &.running .status-count {
        color: var(--el-color-success);
      }

      &.stopped .status-count {
        color: var(--el-color-warning);
      }

      &.error .status-count {
        color: var(--el-color-danger);
      }
    }
  }
}

.chart-section {
  .chart-container {
    text-align: center;

    .chart-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin-bottom: 12px;
    }

    .donut-chart {
      display: inline-block;
      margin-bottom: 12px;
    }

    .chart-legend {
      display: flex;
      justify-content: center;
      gap: 16px;
      flex-wrap: wrap;

      .legend-item {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;

        .legend-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }

        .legend-label {
          color: var(--el-text-color-secondary);
        }

        .legend-value {
          font-weight: 600;
          color: var(--el-text-color-primary);
        }
      }
    }
  }
}

.categories-section,
.registry-section,
.resource-section,
.health-section,
.events-section {
  .category-header,
  .registry-header,
  .resource-header,
  .health-header,
  .events-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;

    .category-title,
    .registry-title,
    .resource-title,
    .health-title,
    .events-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .health-summary,
    .events-count {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .category-list,
  .registry-list,
  .resource-list {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .category-item,
    .registry-item,
    .resource-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 8px 12px;
      background: var(--el-fill-color-extra-light);
      border-radius: 6px;
      border: 1px solid var(--el-border-color-lighter);
      transition: all 0.3s ease;

      &:hover {
        border-color: var(--el-color-primary);
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
      }
    }

    .resource-item {
      cursor: pointer;
    }

    .category-info,
    .registry-info,
    .container-info {
      flex: 1;
      min-width: 0;

      .category-name,
      .registry-name,
      .container-name {
        display: block;
        font-size: 13px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .category-tag,
      .registry-url,
      .container-image {
        font-size: 11px;
        color: var(--el-text-color-secondary);
      }
    }

    .category-stats {
      display: flex;
      align-items: center;
      gap: 8px;

      .category-count {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-color-primary);
      }

      .category-status {
        width: 30px;
        height: 4px;
        background: var(--el-border-color-lighter);
        border-radius: 2px;
        overflow: hidden;

        .running-dots {
          height: 100%;
          background: var(--el-color-success);
          border-radius: 2px;
          transition: width 0.3s ease;
        }
      }
    }

    .registry-count {
      .count-value {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-color-primary);
      }
    }

    .container-metrics {
      display: flex;
      flex-direction: column;
      gap: 4px;
      min-width: 120px;

      .metric {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 11px;

        .metric-label {
          width: 25px;
          color: var(--el-text-color-secondary);
        }

        .metric-value {
          width: 40px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        .metric-bar {
          flex: 1;
          height: 3px;
          background: var(--el-border-color-lighter);
          border-radius: 2px;
          overflow: hidden;

          .metric-fill {
            height: 100%;
            border-radius: 2px;
            transition: width 0.3s ease;
          }
        }
      }
    }
  }
}

.registry-section {
  .registry-item {
    .registry-icon {
      color: var(--el-color-primary);
      margin-right: 8px;
    }
  }
}

.health-section {
  .health-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 8px;

    .health-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 8px;
      background: var(--el-fill-color-extra-light);
      border-radius: 4px;
      font-size: 11px;

      .health-icon {
        font-size: 14px;
      }

      &.healthy .health-icon {
        color: var(--el-color-success);
      }

      &.unhealthy .health-icon {
        color: var(--el-color-danger);
      }

      &.starting .health-icon {
        color: var(--el-color-warning);
      }

      .health-container {
        flex: 1;
        font-weight: 500;
        color: var(--el-text-color-primary);
      }

      .health-status {
        color: var(--el-text-color-secondary);
        text-transform: capitalize;
      }
    }
  }
}

.events-section {
  .events-list {
    .event-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: none;
      }

      .event-type {
        flex-shrink: 0;
        width: 24px;
        height: 24px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 12px;

        &.start {
          background: rgba(var(--el-color-success-rgb), 0.2);
          color: var(--el-color-success);
        }

        &.stop {
          background: rgba(var(--el-color-danger-rgb), 0.2);
          color: var(--el-color-danger);
        }

        &.restart {
          background: rgba(var(--el-color-warning-rgb), 0.2);
          color: var(--el-color-warning);
        }

        &.health {
          background: rgba(var(--el-color-info-rgb), 0.2);
          color: var(--el-color-info);
        }
      }

      .event-content {
        flex: 1;

        .event-container {
          display: block;
          font-size: 12px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        .event-action {
          display: block;
          font-size: 11px;
          color: var(--el-text-color-secondary);
        }

        .event-time {
          display: block;
          font-size: 10px;
          color: var(--el-text-color-placeholder);
        }
      }
    }
  }
}

.actions-section {
  display: flex;
  justify-content: center;
  margin-top: 8px;
}

// Responsive design
@media (max-width: 480px) {
  .container-stats-widget {
    .stats-header {
      flex-direction: column;
      gap: 12px;
    }

    .chart-section .chart-legend {
      flex-direction: column;
      gap: 8px;
    }

    .health-section .health-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>
