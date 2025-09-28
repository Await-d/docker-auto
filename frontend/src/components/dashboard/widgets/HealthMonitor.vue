<template>
  <div class="health-monitor-widget">
    <!-- Loading State -->
    <div v-if="isLoading && !hasHealthData" class="loading-container">
      <el-skeleton :rows="3" animated />
      <div class="loading-text">正在加载健康数据...</div>
    </div>

    <!-- Error State -->
    <div v-else-if="error && !hasHealthData" class="error-container">
      <el-icon :size="48" class="error-icon">
        <CircleCloseFilled />
      </el-icon>
      <h3 class="error-title">无法加载健康数据</h3>
      <p class="error-message">{{ error }}</p>
      <el-button @click="retryLoad" type="primary" size="small">
        <el-icon><Refresh /></el-icon>
        重试
      </el-button>
    </div>

    <!-- Main Content -->
    <div v-else class="health-content">
      <!-- Health Overview -->
      <div class="health-overview">
        <div class="overall-health" :class="overallHealthStatus">
          <div class="health-indicator">
            <el-icon class="health-icon" :size="32">
              <component :is="getOverallHealthIcon()" />
            </el-icon>
          </div>
          <div class="health-info">
            <h3 class="health-title">{{ getOverallHealthText() }}</h3>
            <p class="health-subtitle">{{ healthSummary }}</p>
          </div>
        </div>

        <div class="health-actions">
          <el-button-group size="small">
            <el-button @click="refreshHealthData" :loading="isRefreshing" type="primary">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-button @click="viewHealthReport" type="default">
              <el-icon><Document /></el-icon>
              报告
            </el-button>
          </el-button-group>
        </div>
      </div>

      <!-- Health Metrics Grid -->
      <div class="health-metrics">
        <div
          v-for="service in healthServices"
          :key="service.id"
          class="health-service"
          :class="service.status"
          @click="viewServiceDetails(service.id)"
        >
          <div class="service-header">
            <div class="service-icon">
              <el-icon :size="20">
                <component :is="getServiceIcon(service.type)" />
              </el-icon>
            </div>
            <div class="service-info">
              <span class="service-name">{{ service.name }}</span>
              <span class="service-type">{{ getServiceTypeText(service.type) }}</span>
            </div>
            <div class="service-status">
              <el-tag :type="getStatusTagType(service.status)" size="small">
                {{ getStatusText(service.status) }}
              </el-tag>
            </div>
          </div>

          <div class="service-details">
            <div class="health-checks">
              <div
                v-for="check in service.healthChecks"
                :key="check.name"
                class="health-check"
                :class="check.status"
              >
                <span class="check-name">{{ check.name }}</span>
                <div class="check-result">
                  <el-icon class="check-icon" :size="14">
                    <component :is="getCheckIcon(check.status)" />
                  </el-icon>
                  <span class="check-duration">{{ check.duration }}ms</span>
                </div>
              </div>
            </div>

            <div class="service-metrics" v-if="service.metrics">
              <div class="metric-item">
                <span class="metric-label">响应时间</span>
                <span class="metric-value">{{ service.metrics.responseTime }}ms</span>
              </div>
              <div class="metric-item">
                <span class="metric-label">可用性</span>
                <span class="metric-value">{{ service.metrics.uptime }}%</span>
              </div>
              <div class="metric-item" v-if="service.metrics.errorRate !== undefined">
                <span class="metric-label">错误率</span>
                <span class="metric-value error-rate" :class="{ high: service.metrics.errorRate > 5 }">
                  {{ service.metrics.errorRate }}%
                </span>
              </div>
            </div>
          </div>

          <div class="service-footer">
            <span class="last-check">
              最后检查: {{ formatLastCheck(service.lastCheck) }}
            </span>
            <el-button size="small" type="text" @click.stop="runHealthCheck(service.id)">
              检查
            </el-button>
          </div>
        </div>

        <div v-if="healthServices.length === 0" class="empty-state">
          <el-icon :size="32"><Monitor /></el-icon>
          <p>暂无健康监控数据</p>
        </div>
      </div>

      <!-- Health Alerts -->
      <div v-if="healthAlerts.length > 0" class="health-alerts">
        <div class="section-header">
          <el-icon class="section-icon"><Warning /></el-icon>
          <span class="section-title">健康告警</span>
          <el-badge :value="healthAlerts.length" type="danger" />
        </div>

        <div class="alerts-list">
          <div
            v-for="alert in healthAlerts"
            :key="alert.id"
            class="alert-item"
            :class="alert.severity"
          >
            <div class="alert-icon">
              <el-icon :size="16">
                <component :is="getAlertIcon(alert.severity)" />
              </el-icon>
            </div>
            <div class="alert-content">
              <div class="alert-title">{{ alert.title }}</div>
              <div class="alert-message">{{ alert.message }}</div>
              <div class="alert-meta">
                <span class="alert-service">{{ alert.serviceName }}</span>
                <span class="alert-time">{{ formatRelativeTime(alert.timestamp) }}</span>
              </div>
            </div>
            <div class="alert-actions">
              <el-button size="small" type="text" @click="acknowledgeAlert(alert.id)">
                确认
              </el-button>
              <el-button size="small" type="text" @click="resolveAlert(alert.id)" class="resolve-btn">
                解决
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Health Events -->
      <div class="health-events">
        <div class="section-header">
          <el-icon class="section-icon"><List /></el-icon>
          <span class="section-title">健康事件</span>
        </div>

        <div class="events-timeline">
          <div
            v-for="event in recentHealthEvents"
            :key="event.id"
            class="event-item"
            :class="event.type"
          >
            <div class="event-indicator">
              <div class="indicator-dot" />
            </div>

            <div class="event-content">
              <div class="event-header">
                <span class="event-title">{{ event.title }}</span>
                <span class="event-time">{{ formatRelativeTime(event.timestamp) }}</span>
              </div>
              <div class="event-details">
                <span class="event-service">{{ event.serviceName }}</span>
                <span class="event-description">{{ event.description }}</span>
              </div>
            </div>

            <div class="event-status">
              <el-icon class="status-icon" :class="`status-${event.type}`" :size="16">
                <component :is="getEventIcon(event.type)" />
              </el-icon>
            </div>
          </div>

          <div v-if="recentHealthEvents.length === 0" class="empty-timeline">
            <p>暂无健康事件</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { ElMessage } from 'element-plus';
import {
  CircleCloseFilled,
  Refresh,
  Document,
  Monitor,
  Warning,
  List,
  CircleCheckFilled,
  Loading,
  InfoFilled,
  SuccessFilled,
  Service,
  DataBoard,
  Connection,
  Setting,
} from '@element-plus/icons-vue';

// Icons for dynamic components
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = {
  CircleCheckFilled,
  Warning,
  CircleCloseFilled,
  Loading,
  InfoFilled,
  SuccessFilled,
  Service,
  DataBoard,
  Connection,
  Setting,
};

// Props
interface Props {
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
  displayMode?: 'default' | 'compact' | 'detailed' | 'minimal';
}

const props = withDefaults(defineProps<Props>(), {
  displayMode: 'default',
});

// Emits
const emit = defineEmits<{
  'data-updated': [data: any];
  error: [error: any];
  loading: [loading: boolean];
}>();

// Types
interface HealthCheck {
  name: string;
  status: 'healthy' | 'unhealthy' | 'degraded' | 'unknown';
  duration: number;
  message?: string;
}

interface HealthService {
  id: string;
  name: string;
  type: 'web' | 'database' | 'cache' | 'api' | 'service';
  status: 'healthy' | 'unhealthy' | 'degraded' | 'unknown';
  healthChecks: HealthCheck[];
  metrics?: {
    responseTime: number;
    uptime: number;
    errorRate?: number;
  };
  lastCheck: Date;
  endpoint?: string;
  dependencies?: string[];
}

interface HealthAlert {
  id: string;
  title: string;
  message: string;
  severity: 'critical' | 'warning' | 'info';
  serviceName: string;
  serviceId: string;
  timestamp: Date;
  acknowledged: boolean;
  resolved: boolean;
}

interface HealthEvent {
  id: string;
  title: string;
  description: string;
  type: 'recovery' | 'degraded' | 'failure' | 'maintenance';
  serviceName: string;
  serviceId: string;
  timestamp: Date;
}

// Reactive state
const isLoading = ref(false);
const isRefreshing = ref(false);
const error = ref<string | null>(null);

// Data state
const healthServices = ref<HealthService[]>([]);
const healthAlerts = ref<HealthAlert[]>([]);
const recentHealthEvents = ref<HealthEvent[]>([]);

// Update intervals
const refreshInterval = ref<NodeJS.Timeout>();

// Computed properties
const hasHealthData = computed(() => {
  return healthServices.value.length > 0 ||
         healthAlerts.value.length > 0 ||
         recentHealthEvents.value.length > 0;
});

const overallHealthStatus = computed(() => {
  if (healthServices.value.length === 0) return 'unknown';

  const hasUnhealthy = healthServices.value.some(s => s.status === 'unhealthy');
  const hasDegraded = healthServices.value.some(s => s.status === 'degraded');

  if (hasUnhealthy) return 'unhealthy';
  if (hasDegraded) return 'degraded';
  return 'healthy';
});

const healthSummary = computed(() => {
  const total = healthServices.value.length;
  const healthy = healthServices.value.filter(s => s.status === 'healthy').length;
  const degraded = healthServices.value.filter(s => s.status === 'degraded').length;
  const unhealthy = healthServices.value.filter(s => s.status === 'unhealthy').length;

  if (total === 0) return '暂无服务监控';
  if (unhealthy > 0) return `${unhealthy} 个服务异常，需要立即处理`;
  if (degraded > 0) return `${degraded} 个服务性能下降，建议检查`;
  return `所有 ${healthy} 个服务运行正常`;
});

// Methods
const loadHealthData = async () => {
  try {
    isLoading.value = true;
    emit('loading', true);
    error.value = null;

    // Import API modules dynamically to avoid dependency issues
    const { monitoringAPI } = await import('@/api/monitoring');

    // Load all health data in parallel
    const [servicesData, alertsData, eventsData] = await Promise.all([
      monitoringAPI.getHealthServices(),
      monitoringAPI.getHealthAlerts(),
      monitoringAPI.getHealthEvents(10),
    ]);

    // Update state with real API data
    healthServices.value = (servicesData || []).map((service: any) => ({
      id: service.id,
      name: service.name,
      type: service.type,
      status: service.status,
      healthChecks: service.healthChecks || [],
      metrics: service.metrics,
      lastCheck: new Date(service.lastCheck),
      endpoint: service.endpoint,
      dependencies: service.dependencies,
    }));

    healthAlerts.value = (alertsData || []).map((alert: any) => ({
      id: alert.id,
      title: alert.title,
      message: alert.message,
      severity: alert.severity,
      serviceName: alert.serviceName,
      serviceId: alert.serviceId,
      timestamp: new Date(alert.timestamp),
      acknowledged: alert.acknowledged || false,
      resolved: alert.resolved || false,
    }));

    recentHealthEvents.value = (eventsData || []).map((event: any) => ({
      id: event.id,
      title: event.title,
      description: event.description,
      type: event.type,
      serviceName: event.serviceName,
      serviceId: event.serviceId,
      timestamp: new Date(event.timestamp),
    }));

    emit('data-updated', {
      totalServices: healthServices.value.length,
      healthyServices: healthServices.value.filter(s => s.status === 'healthy').length,
      alertsCount: healthAlerts.value.length,
      overallStatus: overallHealthStatus.value,
    });

  } catch (err: any) {
    console.error('Failed to load health data:', err);
    error.value = err.message || '加载健康数据失败';
    emit('error', err);
  } finally {
    isLoading.value = false;
    emit('loading', false);
  }
};

const refreshHealthData = async () => {
  isRefreshing.value = true;
  try {
    await loadHealthData();
    ElMessage.success('健康数据已刷新');
  } finally {
    isRefreshing.value = false;
  }
};

const viewHealthReport = () => {
  ElMessage.info('跳转到健康报告页面...');
};

const viewServiceDetails = (serviceId: string) => {
  ElMessage.info(`查看服务 ${serviceId} 的详细信息...`);
};

const runHealthCheck = async (serviceId: string) => {
  try {
    const { monitoringAPI } = await import('@/api/monitoring');
    await monitoringAPI.runHealthCheck(serviceId);
    ElMessage.success('健康检查已开始');

    // Refresh data after a short delay
    setTimeout(() => {
      loadHealthData();
    }, 2000);
  } catch (err: any) {
    console.error('Failed to run health check:', err);
    ElMessage.error('启动健康检查失败: ' + (err.message || '未知错误'));
  }
};

const acknowledgeAlert = async (alertId: string) => {
  try {
    const { monitoringAPI } = await import('@/api/monitoring');
    await monitoringAPI.acknowledgeAlert(alertId);

    // Update local state
    const alert = healthAlerts.value.find(a => a.id === alertId);
    if (alert) {
      alert.acknowledged = true;
    }

    ElMessage.success('告警已确认');
  } catch (err: any) {
    console.error('Failed to acknowledge alert:', err);
    ElMessage.error('确认告警失败: ' + (err.message || '未知错误'));
  }
};

const resolveAlert = async (alertId: string) => {
  try {
    const { monitoringAPI } = await import('@/api/monitoring');
    await monitoringAPI.resolveAlert(alertId);

    // Remove from local state
    healthAlerts.value = healthAlerts.value.filter(a => a.id !== alertId);

    ElMessage.success('告警已解决');
  } catch (err: any) {
    console.error('Failed to resolve alert:', err);
    ElMessage.error('解决告警失败: ' + (err.message || '未知错误'));
  }
};

const retryLoad = () => {
  loadHealthData();
};

// Helper functions
const getOverallHealthIcon = (): string => {
  switch (overallHealthStatus.value) {
    case 'healthy':
      return 'CircleCheckFilled';
    case 'degraded':
      return 'Warning';
    case 'unhealthy':
      return 'CircleCloseFilled';
    default:
      return 'InfoFilled';
  }
};

const getOverallHealthText = (): string => {
  switch (overallHealthStatus.value) {
    case 'healthy':
      return '系统健康';
    case 'degraded':
      return '性能下降';
    case 'unhealthy':
      return '系统异常';
    default:
      return '状态未知';
  }
};

const getServiceIcon = (type: string): string => {
  switch (type) {
    case 'web':
      return 'Monitor';
    case 'database':
      return 'DataBoard';
    case 'cache':
      return 'Connection';
    case 'api':
      return 'Service';
    default:
      return 'Setting';
  }
};

const getServiceTypeText = (type: string): string => {
  switch (type) {
    case 'web':
      return 'Web服务';
    case 'database':
      return '数据库';
    case 'cache':
      return '缓存';
    case 'api':
      return 'API服务';
    case 'service':
      return '服务';
    default:
      return '未知';
  }
};

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'healthy':
      return 'success';
    case 'degraded':
      return 'warning';
    case 'unhealthy':
      return 'danger';
    default:
      return 'info';
  }
};

const getStatusText = (status: string): string => {
  switch (status) {
    case 'healthy':
      return '健康';
    case 'degraded':
      return '降级';
    case 'unhealthy':
      return '异常';
    default:
      return '未知';
  }
};

const getCheckIcon = (status: string): string => {
  switch (status) {
    case 'healthy':
      return 'SuccessFilled';
    case 'degraded':
      return 'Warning';
    case 'unhealthy':
      return 'CircleCloseFilled';
    default:
      return 'InfoFilled';
  }
};

const getAlertIcon = (severity: string): string => {
  switch (severity) {
    case 'critical':
      return 'CircleCloseFilled';
    case 'warning':
      return 'Warning';
    case 'info':
      return 'InfoFilled';
    default:
      return 'InfoFilled';
  }
};

const getEventIcon = (type: string): string => {
  switch (type) {
    case 'recovery':
      return 'SuccessFilled';
    case 'degraded':
      return 'Warning';
    case 'failure':
      return 'CircleCloseFilled';
    case 'maintenance':
      return 'Setting';
    default:
      return 'InfoFilled';
  }
};

const formatLastCheck = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  return date.toLocaleDateString();
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return date.toLocaleDateString();
};

// Lifecycle hooks
onMounted(async () => {
  await loadHealthData();

  // Set up periodic refresh
  const refreshIntervalMs = props.widgetConfig?.refreshInterval || 30000; // 30 seconds
  if (refreshIntervalMs > 0) {
    refreshInterval.value = setInterval(loadHealthData, refreshIntervalMs);
  }
});

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
  }
});
</script>

<style scoped lang="scss">
.health-monitor-widget {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.loading-container,
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  gap: 16px;

  .loading-text {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .error-icon {
    color: var(--el-color-danger);
  }

  .error-title {
    color: var(--el-text-color-primary);
    margin: 0;
  }

  .error-message {
    color: var(--el-text-color-secondary);
    margin: 0;
    font-size: 14px;
  }
}

.health-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  height: 100%;
}

.health-overview {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-fill-color-extra-light);
  border-radius: 8px;

  .overall-health {
    display: flex;
    align-items: center;
    gap: 12px;

    &.healthy .health-icon {
      color: var(--el-color-success);
    }

    &.degraded .health-icon {
      color: var(--el-color-warning);
    }

    &.unhealthy .health-icon {
      color: var(--el-color-danger);
    }

    &.unknown .health-icon {
      color: var(--el-color-info);
    }

    .health-info {
      .health-title {
        margin: 0 0 4px 0;
        font-size: 18px;
        color: var(--el-text-color-primary);
      }

      .health-subtitle {
        margin: 0;
        font-size: 14px;
        color: var(--el-text-color-secondary);
      }
    }
  }
}

.health-metrics {
  display: flex;
  flex-direction: column;
  gap: 12px;

  .health-service {
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    padding: 16px;
    cursor: pointer;
    transition: all 0.3s ease;

    &:hover {
      border-color: var(--el-color-primary);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    &.healthy {
      border-left: 4px solid var(--el-color-success);
    }

    &.degraded {
      border-left: 4px solid var(--el-color-warning);
    }

    &.unhealthy {
      border-left: 4px solid var(--el-color-danger);
    }

    &.unknown {
      border-left: 4px solid var(--el-color-info);
    }

    .service-header {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 12px;

      .service-icon {
        flex-shrink: 0;
        width: 40px;
        height: 40px;
        background: var(--el-color-primary-light-9);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--el-color-primary);
      }

      .service-info {
        flex: 1;
        min-width: 0;

        .service-name {
          display: block;
          font-size: 16px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 2px;
        }

        .service-type {
          font-size: 12px;
          color: var(--el-text-color-secondary);
        }
      }

      .service-status {
        flex-shrink: 0;
      }
    }

    .service-details {
      margin-bottom: 12px;

      .health-checks {
        margin-bottom: 8px;

        .health-check {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 4px 0;
          font-size: 12px;

          .check-name {
            color: var(--el-text-color-regular);
          }

          .check-result {
            display: flex;
            align-items: center;
            gap: 4px;

            .check-icon {
              &.healthy {
                color: var(--el-color-success);
              }

              &.degraded {
                color: var(--el-color-warning);
              }

              &.unhealthy {
                color: var(--el-color-danger);
              }
            }

            .check-duration {
              color: var(--el-text-color-secondary);
              font-size: 11px;
            }
          }
        }
      }

      .service-metrics {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(80px, 1fr));
        gap: 8px;

        .metric-item {
          display: flex;
          flex-direction: column;
          align-items: center;
          text-align: center;

          .metric-label {
            font-size: 11px;
            color: var(--el-text-color-secondary);
            margin-bottom: 2px;
          }

          .metric-value {
            font-size: 13px;
            font-weight: 600;
            color: var(--el-text-color-primary);

            &.error-rate.high {
              color: var(--el-color-danger);
            }
          }
        }
      }
    }

    .service-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-top: 8px;
      border-top: 1px solid var(--el-border-color-lighter);

      .last-check {
        font-size: 11px;
        color: var(--el-text-color-placeholder);
      }
    }
  }

  .empty-state {
    text-align: center;
    padding: 32px;
    color: var(--el-text-color-secondary);

    p {
      margin: 8px 0 0 0;
      font-size: 14px;
    }
  }
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;

  .section-icon {
    color: var(--el-color-primary);
    font-size: 16px;
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.health-alerts {
  .alerts-list {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .alert-item {
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 6px;
      padding: 12px;
      display: flex;
      gap: 12px;

      &.critical {
        border-left: 4px solid var(--el-color-danger);
      }

      &.warning {
        border-left: 4px solid var(--el-color-warning);
      }

      &.info {
        border-left: 4px solid var(--el-color-info);
      }

      .alert-icon {
        flex-shrink: 0;
        margin-top: 2px;

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

        .alert-title {
          font-size: 13px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 2px;
        }

        .alert-message {
          font-size: 12px;
          color: var(--el-text-color-regular);
          margin-bottom: 4px;
          line-height: 1.4;
        }

        .alert-meta {
          display: flex;
          gap: 12px;
          font-size: 11px;
          color: var(--el-text-color-secondary);

          .alert-service {
            background: var(--el-fill-color-light);
            padding: 2px 6px;
            border-radius: 4px;
          }
        }
      }

      .alert-actions {
        display: flex;
        gap: 4px;
        flex-shrink: 0;

        .resolve-btn {
          color: var(--el-color-success);
        }
      }
    }
  }
}

.health-events {
  .events-timeline {
    position: relative;

    .event-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 8px 0;
      position: relative;

      &:not(:last-child)::after {
        content: '';
        position: absolute;
        left: 6px;
        top: 32px;
        width: 1px;
        height: 20px;
        background: var(--el-border-color-light);
      }

      .event-indicator {
        flex-shrink: 0;

        .indicator-dot {
          width: 12px;
          height: 12px;
          border-radius: 50%;
          background: var(--el-color-info);
        }
      }

      &.recovery .indicator-dot {
        background: var(--el-color-success);
      }

      &.degraded .indicator-dot {
        background: var(--el-color-warning);
      }

      &.failure .indicator-dot {
        background: var(--el-color-danger);
      }

      &.maintenance .indicator-dot {
        background: var(--el-color-primary);
      }

      .event-content {
        flex: 1;
        min-width: 0;

        .event-header {
          display: flex;
          justify-content: space-between;
          align-items: baseline;
          margin-bottom: 4px;

          .event-title {
            font-size: 13px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          .event-time {
            font-size: 11px;
            color: var(--el-text-color-placeholder);
          }
        }

        .event-details {
          display: flex;
          gap: 8px;
          font-size: 12px;
          color: var(--el-text-color-secondary);

          .event-service {
            font-family: monospace;
            background: var(--el-fill-color-light);
            padding: 2px 6px;
            border-radius: 4px;
          }
        }
      }

      .event-status {
        flex-shrink: 0;

        .status-icon {
          &.status-recovery {
            color: var(--el-color-success);
          }

          &.status-degraded {
            color: var(--el-color-warning);
          }

          &.status-failure {
            color: var(--el-color-danger);
          }

          &.status-maintenance {
            color: var(--el-color-primary);
          }
        }
      }
    }

    .empty-timeline {
      text-align: center;
      padding: 24px;
      color: var(--el-text-color-secondary);
      font-size: 14px;
    }
  }
}

// Responsive design
@media (max-width: 768px) {
  .health-monitor-widget {
    padding: 12px;
  }

  .health-overview {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .service-metrics {
    grid-template-columns: repeat(2, 1fr);
  }

  .alert-item {
    flex-direction: column;
    gap: 8px;

    .alert-actions {
      align-self: flex-end;
    }
  }
}
</style>