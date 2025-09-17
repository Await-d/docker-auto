<template>
  <div
    class="system-overview-widget"
    :class="{ 'compact-mode': displayMode === 'compact' }"
  >
    <!-- System Status Header -->
    <div class="status-header">
      <div class="system-status" :class="systemStatusClass">
        <div class="status-indicator">
          <div class="status-dot" />
        </div>
        <div class="status-info">
          <h3 class="status-title">
            {{ systemStatusText }}
          </h3>
          <p class="status-subtitle">
            {{ systemStatusSubtitle }}
          </p>
        </div>
      </div>
      <div class="system-uptime">
        <el-icon><Timer /></el-icon>
        <span>{{ formatUptime(systemData.uptime) }}</span>
      </div>
    </div>

    <!-- Key Metrics Grid -->
    <div class="metrics-grid">
      <!-- Container Metrics -->
      <div class="metric-card containers">
        <div class="metric-header">
          <el-icon class="metric-icon">
            <Box />
          </el-icon>
          <span class="metric-label">Containers</span>
        </div>
        <div class="metric-content">
          <div class="metric-main">
            <span class="metric-value">{{ systemData.containers.total }}</span>
            <span class="metric-unit">total</span>
          </div>
          <div class="metric-breakdown">
            <div class="breakdown-item running">
              <span class="breakdown-label">Running</span>
              <span class="breakdown-value">{{
                systemData.containers.running
              }}</span>
            </div>
            <div class="breakdown-item stopped">
              <span class="breakdown-label">Stopped</span>
              <span class="breakdown-value">{{
                systemData.containers.stopped
              }}</span>
            </div>
            <div
              v-if="systemData.containers.updating > 0"
              class="breakdown-item updating"
            >
              <span class="breakdown-label">Updating</span>
              <span class="breakdown-value">{{
                systemData.containers.updating
              }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Update Status -->
      <div class="metric-card updates">
        <div class="metric-header">
          <el-icon class="metric-icon">
            <Refresh />
          </el-icon>
          <span class="metric-label">Updates</span>
        </div>
        <div class="metric-content">
          <div class="metric-main">
            <span class="metric-value">{{ systemData.updates.available }}</span>
            <span class="metric-unit">available</span>
          </div>
          <div class="metric-breakdown">
            <div
              v-if="systemData.updates.security > 0"
              class="breakdown-item security"
            >
              <span class="breakdown-label">Security</span>
              <span class="breakdown-value">{{
                systemData.updates.security
              }}</span>
            </div>
            <div class="breakdown-item recent">
              <span class="breakdown-label">Recent</span>
              <span class="breakdown-value">{{
                systemData.updates.recent
              }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- System Health -->
      <div class="metric-card health">
        <div class="metric-header">
          <el-icon class="metric-icon">
            <CircleCheckFilled />
          </el-icon>
          <span class="metric-label">Health</span>
        </div>
        <div class="metric-content">
          <div class="health-score" :class="healthScoreClass">
            <div class="score-circle">
              <span class="score-value">{{ systemData.health.score }}%</span>
            </div>
          </div>
          <div class="health-indicators">
            <div
              v-for="indicator in healthIndicators"
              :key="indicator.name"
              class="health-indicator"
              :class="indicator.status"
            >
              <el-icon class="indicator-icon">
                <component :is="indicator.icon" />
              </el-icon>
              <span class="indicator-name">{{ indicator.name }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Activity Summary -->
      <div class="metric-card activity">
        <div class="metric-header">
          <el-icon class="metric-icon">
            <DataLine />
          </el-icon>
          <span class="metric-label">Activity</span>
        </div>
        <div class="metric-content">
          <div class="activity-stats">
            <div class="activity-stat">
              <span class="stat-value">{{
                systemData.activity.events24h
              }}</span>
              <span class="stat-label">Events (24h)</span>
            </div>
            <div class="activity-stat">
              <span class="stat-value">{{ systemData.activity.errors }}</span>
              <span class="stat-label">Errors</span>
            </div>
            <div class="activity-stat">
              <span class="stat-value">{{ systemData.activity.warnings }}</span>
              <span class="stat-label">Warnings</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Resource Usage Bars -->
    <div
v-if="displayMode !== 'minimal'" class="resource-usage"
>
      <div class="resource-item cpu">
        <div class="resource-header">
          <span class="resource-label">CPU Usage</span>
          <span class="resource-value">{{ systemData.resources.cpu.toFixed(1) }}%</span>
        </div>
        <el-progress
          :percentage="systemData.resources.cpu"
          :color="getResourceColor(systemData.resources.cpu)"
          :show-text="false"
          :stroke-width="6"
        />
      </div>

      <div class="resource-item memory">
        <div class="resource-header">
          <span class="resource-label">Memory Usage</span>
          <span class="resource-value">{{ formatBytes(systemData.resources.memory.used) }} /
            {{ formatBytes(systemData.resources.memory.total) }}</span>
        </div>
        <el-progress
          :percentage="memoryPercentage"
          :color="getResourceColor(memoryPercentage)"
          :show-text="false"
          :stroke-width="6"
        />
      </div>

      <div class="resource-item disk">
        <div class="resource-header">
          <span class="resource-label">Disk Usage</span>
          <span class="resource-value">{{ formatBytes(systemData.resources.disk.used) }} /
            {{ formatBytes(systemData.resources.disk.total) }}</span>
        </div>
        <el-progress
          :percentage="diskPercentage"
          :color="getResourceColor(diskPercentage)"
          :show-text="false"
          :stroke-width="6"
        />
      </div>
    </div>

    <!-- Quick Actions -->
    <div
v-if="displayMode === 'detailed'" class="quick-actions"
>
      <el-button-group>
        <el-button size="small" @click="triggerUpdateScan">
          <el-icon><Refresh /></el-icon>
          Scan Updates
        </el-button>
        <el-button size="small" @click="viewSystemLogs">
          <el-icon><Document /></el-icon>
          View Logs
        </el-button>
        <el-button size="small" @click="openSystemSettings">
          <el-icon><Setting /></el-icon>
          Settings
        </el-button>
      </el-button-group>
    </div>

    <!-- Alerts Section -->
    <div
      v-if="systemData.alerts && systemData.alerts.length > 0"
      class="alerts-section"
    >
      <div class="alerts-header">
        <el-icon><Warning /></el-icon>
        <span>Active Alerts ({{ systemData.alerts.length }})</span>
      </div>
      <div class="alerts-list">
        <div
          v-for="alert in systemData.alerts.slice(0, 3)"
          :key="alert.id"
          class="alert-item"
          :class="alert.severity"
        >
          <div class="alert-icon">
            <el-icon>
              <component :is="getAlertIcon(alert.severity)" />
            </el-icon>
          </div>
          <div class="alert-content">
            <span class="alert-message">{{ alert.message }}</span>
            <span class="alert-time">{{
              formatRelativeTime(alert.timestamp)
            }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- System Information -->
    <div
v-if="displayMode === 'detailed'" class="system-info"
>
      <div class="info-grid">
        <div class="info-item">
          <span class="info-label">Version</span>
          <span class="info-value">{{ systemData.version }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Docker Version</span>
          <span class="info-value">{{ systemData.dockerVersion }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Last Backup</span>
          <span class="info-value">{{
            formatRelativeTime(systemData.lastBackup)
          }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Maintenance Mode</span>
          <el-tag
            :type="systemData.maintenanceMode ? 'warning' : 'success'"
            size="small"
          >
            {{ systemData.maintenanceMode ? "Active" : "Inactive" }}
          </el-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { ElMessage } from "element-plus";
import {
  Box,
  Refresh,
  CircleCheckFilled,
  DataLine,
  Timer,
  Warning,
  Document,
  Setting,
  InfoFilled,
  WarningFilled,
  CircleCloseFilled,
} from "@element-plus/icons-vue";

// Used in dynamic components
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = { InfoFilled, WarningFilled, CircleCloseFilled };

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
  "metrics-updated": [metrics: any];
}>();

// Reactive state
const isLoading = ref(false);
const systemData = ref({
  status: "healthy",
  uptime: 86400000, // 1 day in ms
  containers: {
    total: 12,
    running: 10,
    stopped: 2,
    updating: 0,
  },
  updates: {
    available: 3,
    security: 1,
    recent: 7,
  },
  health: {
    score: 92,
  },
  activity: {
    events24h: 156,
    errors: 2,
    warnings: 5,
  },
  resources: {
    cpu: 45.2,
    memory: {
      used: 3221225472, // 3GB
      total: 8589934592, // 8GB
    },
    disk: {
      used: 21474836480, // 20GB
      total: 107374182400, // 100GB
    },
  },
  alerts: [
    {
      id: "1",
      severity: "warning",
      message: "High memory usage detected on container web-server",
      timestamp: new Date(Date.now() - 300000), // 5 minutes ago
    },
    {
      id: "2",
      severity: "info",
      message: "System backup completed successfully",
      timestamp: new Date(Date.now() - 3600000), // 1 hour ago
    },
  ],
  version: "1.2.3",
  dockerVersion: "24.0.7",
  lastBackup: new Date(Date.now() - 3600000), // 1 hour ago
  maintenanceMode: false,
});

// Computed properties
const systemStatusClass = computed(() => {
  switch (systemData.value.status) {
    case "healthy":
      return "status-healthy";
    case "warning":
      return "status-warning";
    case "critical":
      return "status-critical";
    case "maintenance":
      return "status-maintenance";
    default:
      return "status-unknown";
  }
});

const systemStatusText = computed(() => {
  switch (systemData.value.status) {
    case "healthy":
      return "System Healthy";
    case "warning":
      return "Attention Required";
    case "critical":
      return "Critical Issues";
    case "maintenance":
      return "Maintenance Mode";
    default:
      return "Unknown Status";
  }
});

const systemStatusSubtitle = computed(() => {
  const { containers, updates } = systemData.value;
  return `${containers.running}/${containers.total} containers running • ${updates.available} updates available`;
});

const healthScoreClass = computed(() => {
  const score = systemData.value.health.score;
  if (score >= 90) return "score-excellent";
  if (score >= 75) return "score-good";
  if (score >= 60) return "score-fair";
  return "score-poor";
});

const healthIndicators = computed(() => [
  {
    name: "Services",
    status: systemData.value.health.score >= 90 ? "healthy" : "warning",
    icon: "CircleCheckFilled",
  },
  {
    name: "Network",
    status: systemData.value.health.score >= 80 ? "healthy" : "warning",
    icon: "CircleCheckFilled",
  },
  {
    name: "Storage",
    status: systemData.value.health.score >= 70 ? "healthy" : "warning",
    icon: "CircleCheckFilled",
  },
]);

const memoryPercentage = computed(() => {
  const { used, total } = systemData.value.resources.memory;
  return Math.round((used / total) * 100);
});

const diskPercentage = computed(() => {
  const { used, total } = systemData.value.resources.disk;
  return Math.round((used / total) * 100);
});

// Methods
const getResourceColor = (percentage: number) => {
  if (percentage < 70) return "#67c23a";
  if (percentage < 90) return "#e6a23c";
  return "#f56c6c";
};

const getAlertIcon = (severity: string) => {
  switch (severity) {
    case "critical":
      return "CircleCloseFilled";
    case "warning":
      return "WarningFilled";
    case "info":
      return "InfoFilled";
    default:
      return "InfoFilled";
  }
};

const formatUptime = (uptimeMs: number): string => {
  const seconds = Math.floor(uptimeMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}d ${hours % 24}h`;
  if (hours > 0) return `${hours}h ${minutes % 60}m`;
  if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
  return `${seconds}s`;
};

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return date.toLocaleDateString();
};

const fetchSystemData = async () => {
  try {
    emit("loading", true);
    isLoading.value = true;

    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1000));

    // In a real implementation, this would fetch from the API
    const response = await fetch("/api/v1/dashboard/system-overview");
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data = await response.json();
    systemData.value = { ...systemData.value, ...data };

    emit("data-updated", systemData.value);
    emit("metrics-updated", {
      loadTime: Date.now() - performance.now(),
      dataSize: JSON.stringify(systemData.value).length,
    });
  } catch (error) {
    console.error("Failed to fetch system data:", error);
    emit("error", error);
    ElMessage.error("Failed to load system overview data");
  } finally {
    emit("loading", false);
    isLoading.value = false;
  }
};

const triggerUpdateScan = () => {
  ElMessage.info("Starting update scan...");
  // Implementation for triggering update scan
};

const viewSystemLogs = () => {
  // Navigate to system logs
  ElMessage.info("Opening system logs...");
};

const openSystemSettings = () => {
  // Navigate to system settings
  ElMessage.info("Opening system settings...");
};

// Lifecycle hooks
onMounted(() => {
  fetchSystemData();

  // Set up auto-refresh if configured
  const refreshInterval = props.widgetConfig?.refreshInterval || 30000;
  if (refreshInterval > 0) {
    const interval = setInterval(fetchSystemData, refreshInterval);
    onUnmounted(() => clearInterval(interval));
  }
});

// Watch for widget data changes from WebSocket
watch(
  () => props.widgetData,
  (newData) => {
    if (newData) {
      systemData.value = { ...systemData.value, ...newData };
    }
  },
  { deep: true },
);
</script>

<style scoped lang="scss">
.system-overview-widget {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;

  &.compact-mode {
    padding: 12px;
    gap: 12px;

    .metrics-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
}

.status-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;

  .system-status {
    display: flex;
    align-items: center;
    gap: 12px;

    .status-indicator {
      .status-dot {
        width: 12px;
        height: 12px;
        border-radius: 50%;
        animation: pulse 2s infinite;
      }
    }

    &.status-healthy .status-dot {
      background: var(--el-color-success);
    }

    &.status-warning .status-dot {
      background: var(--el-color-warning);
    }

    &.status-critical .status-dot {
      background: var(--el-color-danger);
    }

    &.status-maintenance .status-dot {
      background: var(--el-color-info);
    }

    .status-info {
      .status-title {
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .status-subtitle {
        margin: 2px 0 0 0;
        font-size: 12px;
        color: var(--el-text-color-secondary);
      }
    }
  }

  .system-uptime {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    font-weight: 500;
  }
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;

  .metric-card {
    background: var(--el-fill-color-extra-light);
    border-radius: 8px;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);
    transition: all 0.3s ease;

    &:hover {
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    .metric-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;

      .metric-icon {
        color: var(--el-color-primary);
        font-size: 16px;
      }

      .metric-label {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    .metric-content {
      .metric-main {
        display: flex;
        align-items: baseline;
        gap: 6px;
        margin-bottom: 8px;

        .metric-value {
          font-size: 24px;
          font-weight: 700;
          color: var(--el-color-primary);
        }

        .metric-unit {
          font-size: 12px;
          color: var(--el-text-color-secondary);
        }
      }

      .metric-breakdown {
        display: flex;
        flex-direction: column;
        gap: 4px;

        .breakdown-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          font-size: 12px;

          .breakdown-label {
            color: var(--el-text-color-secondary);
          }

          .breakdown-value {
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          &.running .breakdown-value {
            color: var(--el-color-success);
          }

          &.stopped .breakdown-value {
            color: var(--el-color-warning);
          }

          &.updating .breakdown-value {
            color: var(--el-color-info);
          }

          &.security .breakdown-value {
            color: var(--el-color-danger);
          }
        }
      }

      .health-score {
        display: flex;
        justify-content: center;
        margin-bottom: 12px;

        .score-circle {
          width: 60px;
          height: 60px;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          border: 3px solid;
          transition: all 0.3s ease;

          .score-value {
            font-size: 16px;
            font-weight: 700;
          }
        }

        &.score-excellent .score-circle {
          border-color: var(--el-color-success);
          color: var(--el-color-success);
        }

        &.score-good .score-circle {
          border-color: var(--el-color-primary);
          color: var(--el-color-primary);
        }

        &.score-fair .score-circle {
          border-color: var(--el-color-warning);
          color: var(--el-color-warning);
        }

        &.score-poor .score-circle {
          border-color: var(--el-color-danger);
          color: var(--el-color-danger);
        }
      }

      .health-indicators {
        display: flex;
        justify-content: space-around;

        .health-indicator {
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 4px;
          font-size: 10px;

          .indicator-icon {
            font-size: 14px;
          }

          &.healthy .indicator-icon {
            color: var(--el-color-success);
          }

          &.warning .indicator-icon {
            color: var(--el-color-warning);
          }

          &.critical .indicator-icon {
            color: var(--el-color-danger);
          }

          .indicator-name {
            color: var(--el-text-color-secondary);
          }
        }
      }

      .activity-stats {
        display: flex;
        justify-content: space-between;

        .activity-stat {
          text-align: center;

          .stat-value {
            display: block;
            font-size: 18px;
            font-weight: 700;
            color: var(--el-color-primary);
          }

          .stat-label {
            font-size: 10px;
            color: var(--el-text-color-secondary);
          }
        }
      }
    }
  }
}

.resource-usage {
  .resource-item {
    margin-bottom: 12px;

    &:last-child {
      margin-bottom: 0;
    }

    .resource-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 6px;

      .resource-label {
        font-size: 13px;
        font-weight: 500;
        color: var(--el-text-color-primary);
      }

      .resource-value {
        font-size: 12px;
        color: var(--el-text-color-secondary);
      }
    }
  }
}

.quick-actions {
  display: flex;
  justify-content: center;
  margin-top: 8px;
}

.alerts-section {
  background: var(--el-fill-color-extra-light);
  border-radius: 8px;
  padding: 12px;
  border-left: 4px solid var(--el-color-warning);

  .alerts-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-color-warning);
  }

  .alerts-list {
    .alert-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: none;
      }

      .alert-icon {
        flex-shrink: 0;
        font-size: 14px;

        &.critical {
          color: var(--el-color-danger);
        }

        &.warning {
          color: var(--el-color-warning);
        }

        &.info {
          color: var(--el-color-info);
        }
      }

      .alert-content {
        flex: 1;
        min-width: 0;

        .alert-message {
          display: block;
          font-size: 12px;
          color: var(--el-text-color-primary);
          line-height: 1.4;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .alert-time {
          font-size: 11px;
          color: var(--el-text-color-placeholder);
        }
      }
    }
  }
}

.system-info {
  .info-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;

    .info-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 12px;
      background: var(--el-fill-color-light);
      border-radius: 6px;
      font-size: 12px;

      .info-label {
        color: var(--el-text-color-secondary);
        font-weight: 500;
      }

      .info-value {
        color: var(--el-text-color-primary);
        font-weight: 600;
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
    opacity: 0.7;
  }
}

// Responsive design
@media (max-width: 480px) {
  .system-overview-widget {
    &.compact-mode {
      .metrics-grid {
        grid-template-columns: 1fr;
      }
    }

    .status-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 8px;
    }

    .metrics-grid {
      grid-template-columns: 1fr;
    }

    .system-info .info-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>
