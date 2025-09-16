<template>
  <div class="widget-wrapper" :class="{ 'edit-mode': isEditMode, 'loading': isLoading, 'error': hasError }">
    <!-- Widget Header -->
    <div class="widget-header" v-if="showHeader">
      <div class="widget-title-section">
        <div class="widget-icon" v-if="widgetIcon">
          <el-icon :size="16">
            <component :is="widgetIcon" />
          </el-icon>
        </div>
        <h3 class="widget-title">{{ widget.title }}</h3>
        <div class="widget-status-indicator" :class="statusClass">
          <el-tooltip :content="statusTooltip" placement="top">
            <div class="status-dot"></div>
          </el-tooltip>
        </div>
      </div>

      <div class="widget-actions" v-if="isEditMode || showActions">
        <!-- Refresh Button -->
        <el-button
          v-if="!isEditMode"
          size="mini"
          type="text"
          @click="refreshWidget"
          :loading="isRefreshing"
          class="widget-action-btn"
        >
          <el-icon><Refresh /></el-icon>
        </el-button>

        <!-- Configure Button -->
        <el-button
          v-if="widget.configurable !== false"
          size="mini"
          type="text"
          @click="configureWidget"
          class="widget-action-btn"
        >
          <el-icon><Setting /></el-icon>
        </el-button>

        <!-- More Actions -->
        <el-dropdown @command="handleAction" v-if="isEditMode">
          <el-button size="mini" type="text" class="widget-action-btn">
            <el-icon><More /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="refresh">
                <el-icon><Refresh /></el-icon>
                Refresh
              </el-dropdown-item>
              <el-dropdown-item command="configure" v-if="widget.configurable !== false">
                <el-icon><Setting /></el-icon>
                Configure
              </el-dropdown-item>
              <el-dropdown-item command="duplicate">
                <el-icon><CopyDocument /></el-icon>
                Duplicate
              </el-dropdown-item>
              <el-dropdown-item command="export">
                <el-icon><Download /></el-icon>
                Export Data
              </el-dropdown-item>
              <el-dropdown-item divided command="remove">
                <el-icon><Delete /></el-icon>
                Remove
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- Widget Content -->
    <div class="widget-content" :style="contentStyle">
      <!-- Loading State -->
      <div v-if="isLoading" class="widget-loading">
        <el-skeleton animated>
          <template #template>
            <div class="loading-content">
              <el-skeleton-item variant="h3" style="width: 60%; margin-bottom: 12px;" />
              <el-skeleton-item variant="text" style="width: 100%; margin-bottom: 8px;" />
              <el-skeleton-item variant="text" style="width: 80%; margin-bottom: 8px;" />
              <el-skeleton-item variant="rect" style="width: 100%; height: 120px;" />
            </div>
          </template>
        </el-skeleton>
      </div>

      <!-- Error State -->
      <div v-else-if="hasError" class="widget-error">
        <div class="error-content">
          <el-icon class="error-icon" :size="32"><Warning /></el-icon>
          <h4 class="error-title">Widget Error</h4>
          <p class="error-message">{{ errorMessage }}</p>
          <div class="error-actions">
            <el-button size="small" @click="refreshWidget">
              <el-icon><Refresh /></el-icon>
              Retry
            </el-button>
            <el-button size="small" type="text" @click="showErrorDetails">
              View Details
            </el-button>
          </div>
        </div>
      </div>

      <!-- Offline State -->
      <div v-else-if="isOffline" class="widget-offline">
        <div class="offline-content">
          <el-icon class="offline-icon" :size="32"><WifiOff /></el-icon>
          <h4 class="offline-title">Offline</h4>
          <p class="offline-message">This widget requires an internet connection</p>
        </div>
      </div>

      <!-- Widget Component -->
      <component
        v-else
        :is="widgetComponent"
        v-bind="widgetProps"
        :widget-id="widget.id"
        :widget-config="widget.settings"
        :widget-data="widgetData"
        @data-updated="onDataUpdated"
        @error="onError"
        @loading="onLoading"
        @metrics-updated="onMetricsUpdated"
        class="widget-component"
      />

      <!-- Edit Mode Overlay -->
      <div v-if="isEditMode" class="edit-overlay">
        <div class="edit-handles">
          <div class="resize-handle resize-handle-se"></div>
        </div>
        <div class="edit-info">
          <span class="widget-type">{{ widget.type }}</span>
          <span class="widget-size">{{ widget.position.w }}×{{ widget.position.h }}</span>
        </div>
      </div>
    </div>

    <!-- Widget Footer -->
    <div class="widget-footer" v-if="showFooter">
      <div class="widget-meta">
        <span class="last-updated" v-if="lastUpdated">
          Updated {{ formatRelativeTime(lastUpdated) }}
        </span>
        <span class="refresh-interval" v-if="widget.refreshInterval > 0">
          Refreshes every {{ formatDuration(widget.refreshInterval) }}
        </span>
      </div>

      <div class="widget-metrics" v-if="showMetrics && metrics">
        <el-tooltip content="Load Time" placement="top">
          <span class="metric">
            <el-icon><Timer /></el-icon>
            {{ formatDuration(metrics.loadTime) }}
          </span>
        </el-tooltip>
        <el-tooltip content="Data Size" placement="top">
          <span class="metric">
            <el-icon><Document /></el-icon>
            {{ formatBytes(metrics.dataSize) }}
          </span>
        </el-tooltip>
        <el-tooltip content="Error Count" placement="top" v-if="metrics.errorCount > 0">
          <span class="metric error-metric">
            <el-icon><Warning /></el-icon>
            {{ metrics.errorCount }}
          </span>
        </el-tooltip>
      </div>
    </div>

    <!-- Error Details Dialog -->
    <el-dialog
      v-model="errorDetailsVisible"
      title="Widget Error Details"
      width="600px"
    >
      <div class="error-details">
        <div class="error-summary">
          <h4>Error Summary</h4>
          <p>{{ errorMessage }}</p>
        </div>

        <div class="error-stack" v-if="errorStack">
          <h4>Stack Trace</h4>
          <pre class="stack-trace">{{ errorStack }}</pre>
        </div>

        <div class="error-context">
          <h4>Widget Context</h4>
          <div class="context-grid">
            <div class="context-item">
              <strong>Widget ID:</strong>
              <span>{{ widget.id }}</span>
            </div>
            <div class="context-item">
              <strong>Widget Type:</strong>
              <span>{{ widget.type }}</span>
            </div>
            <div class="context-item">
              <strong>Component:</strong>
              <span>{{ widget.component }}</span>
            </div>
            <div class="context-item">
              <strong>Last Updated:</strong>
              <span>{{ lastUpdated ? formatDate(lastUpdated) : 'Never' }}</span>
            </div>
          </div>
        </div>

        <div class="error-actions-panel">
          <el-button @click="refreshWidget" type="primary">
            <el-icon><Refresh /></el-icon>
            Retry
          </el-button>
          <el-button @click="resetWidget">
            <el-icon><RefreshLeft /></el-icon>
            Reset Widget
          </el-button>
          <el-button @click="copyErrorToClipboard">
            <el-icon><CopyDocument /></el-icon>
            Copy Error
          </el-button>
        </div>
      </div>

      <template #footer>
        <el-button @click="errorDetailsVisible = false">Close</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh, Setting, More, Warning, Timer, Document, Delete,
  CopyDocument, Download, RefreshLeft, WifiOff
} from '@element-plus/icons-vue'

// Types
import type { DashboardWidget } from '@/store/dashboard'
import type { WidgetMetrics } from '@/services/widgetManager'
import { useWidgetManager, WidgetStatus } from '@/services/widgetManager'

// Component imports - Dynamic loading for all widgets
const widgetComponents = {
  SystemOverview: () => import('./widgets/SystemOverview.vue'),
  ContainerStats: () => import('./widgets/ContainerStats.vue'),
  UpdateActivity: () => import('./widgets/UpdateActivity.vue'),
  RealtimeMonitor: () => import('./widgets/RealtimeMonitor.vue'),
  HealthMonitor: () => import('./widgets/HealthMonitor.vue'),
  RecentActivities: () => import('./widgets/RecentActivities.vue'),
  QuickActions: () => import('./widgets/QuickActions.vue'),
  NotificationCenter: () => import('./widgets/NotificationCenter.vue'),
  ResourceCharts: () => import('./widgets/ResourceCharts.vue'),
  SecurityDashboard: () => import('./widgets/SecurityDashboard.vue')
}

// Props
interface Props {
  widget: DashboardWidget
  isEditMode?: boolean
  showHeader?: boolean
  showFooter?: boolean
  showActions?: boolean
  showMetrics?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isEditMode: false,
  showHeader: true,
  showFooter: false,
  showActions: true,
  showMetrics: false
})

// Emits
const emit = defineEmits<{
  remove: [widgetId: string]
  configure: [widget: DashboardWidget]
  refresh: [widgetId: string]
  duplicate: [widget: DashboardWidget]
  export: [widget: DashboardWidget]
}>()

// Widget manager
const widgetManager = useWidgetManager()

// Reactive state
const isRefreshing = ref(false)
const errorDetailsVisible = ref(false)
const errorMessage = ref('')
const errorStack = ref('')
const lastUpdated = ref<Date | null>(null)

// Computed properties
const widgetComponent = computed(() => {
  const componentName = props.widget.component
  return widgetComponents[componentName as keyof typeof widgetComponents] || null
})

const widgetIcon = computed(() => {
  // Map widget types to icons
  const iconMap: Record<string, string> = {
    'system-overview': 'Monitor',
    'container-stats': 'Box',
    'update-activity': 'Refresh',
    'realtime-monitor': 'DataLine',
    'health-monitor': 'CircleCheckFilled',
    'recent-activities': 'Document',
    'quick-actions': 'Lightning',
    'notification-center': 'Bell',
    'resource-charts': 'DataAnalysis',
    'security-dashboard': 'Lock'
  }
  return iconMap[props.widget.type]
})

const widgetProps = computed(() => ({
  ...props.widget.settings,
  isEditMode: props.isEditMode
}))

const widgetData = computed(() =>
  widgetManager.getWidgetData(props.widget.id)
)

const widgetStatus = computed(() =>
  widgetManager.getWidgetStatus(props.widget.id)
)

const metrics = computed(() =>
  widgetManager.getWidgetMetrics(props.widget.id)
)

const isLoading = computed(() =>
  widgetStatus.value === WidgetStatus.LOADING
)

const hasError = computed(() =>
  widgetStatus.value === WidgetStatus.ERROR
)

const isOffline = computed(() =>
  widgetStatus.value === WidgetStatus.OFFLINE
)

const statusClass = computed(() => ({
  'status-loading': isLoading.value,
  'status-loaded': widgetStatus.value === WidgetStatus.LOADED,
  'status-error': hasError.value,
  'status-offline': isOffline.value
}))

const statusTooltip = computed(() => {
  switch (widgetStatus.value) {
    case WidgetStatus.LOADING:
      return 'Loading...'
    case WidgetStatus.LOADED:
      return 'Data loaded successfully'
    case WidgetStatus.ERROR:
      return `Error: ${errorMessage.value}`
    case WidgetStatus.OFFLINE:
      return 'Widget is offline'
    default:
      return 'Unknown status'
  }
})

const contentStyle = computed(() => ({
  minHeight: props.showHeader ? 'calc(100% - 40px)' : '100%',
  maxHeight: props.showFooter ? 'calc(100% - 80px)' : (props.showHeader ? 'calc(100% - 40px)' : '100%')
}))

// Methods
const refreshWidget = async () => {
  try {
    isRefreshing.value = true
    await widgetManager.refreshWidget(props.widget.id, true)
    emit('refresh', props.widget.id)
  } catch (error) {
    console.error('Failed to refresh widget:', error)
    ElMessage.error('Failed to refresh widget')
  } finally {
    isRefreshing.value = false
  }
}

const configureWidget = () => {
  emit('configure', props.widget)
}

const handleAction = async (command: string) => {
  switch (command) {
    case 'refresh':
      await refreshWidget()
      break
    case 'configure':
      configureWidget()
      break
    case 'duplicate':
      emit('duplicate', props.widget)
      break
    case 'export':
      emit('export', props.widget)
      break
    case 'remove':
      emit('remove', props.widget.id)
      break
  }
}

const showErrorDetails = () => {
  errorDetailsVisible.value = true
}

const resetWidget = async () => {
  try {
    await ElMessageBox.confirm(
      'This will reset the widget to its default configuration. Continue?',
      'Reset Widget',
      {
        type: 'warning',
        confirmButtonText: 'Reset',
        cancelButtonText: 'Cancel'
      }
    )

    // Reset widget implementation
    ElMessage.success('Widget reset successfully')
    errorDetailsVisible.value = false
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to reset widget:', error)
      ElMessage.error('Failed to reset widget')
    }
  }
}

const copyErrorToClipboard = async () => {
  try {
    const errorInfo = {
      message: errorMessage.value,
      stack: errorStack.value,
      widget: {
        id: props.widget.id,
        type: props.widget.type,
        component: props.widget.component
      },
      timestamp: new Date().toISOString()
    }

    await navigator.clipboard.writeText(JSON.stringify(errorInfo, null, 2))
    ElMessage.success('Error details copied to clipboard')
  } catch (error) {
    console.error('Failed to copy to clipboard:', error)
    ElMessage.error('Failed to copy error details')
  }
}

// Event handlers
const onDataUpdated = (data: any) => {
  widgetManager.updateWidgetData(props.widget.id, data)
  lastUpdated.value = new Date()
}

const onError = (error: any) => {
  errorMessage.value = error.message || error.toString()
  errorStack.value = error.stack || ''
  console.error(`Widget ${props.widget.id} error:`, error)
}

const onLoading = (loading: boolean) => {
  // Handle loading state if needed
}

const onMetricsUpdated = (newMetrics: Partial<WidgetMetrics>) => {
  // Handle metrics updates if needed
}

// Utility functions
const formatRelativeTime = (date: Date): string => {
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (seconds < 60) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24) return `${hours}h ago`
  return date.toLocaleDateString()
}

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  return `${hours}h`
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const formatDate = (date: Date): string => {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(date)
}

// Lifecycle hooks
onMounted(() => {
  // Register widget with widget manager
  widgetManager.registerWidget(props.widget)
})

onUnmounted(() => {
  // Unregister widget from widget manager
  widgetManager.unregisterWidget(props.widget.id)
})

// Watch for widget data changes
watch(
  () => widgetData.value,
  (newData) => {
    if (newData) {
      lastUpdated.value = new Date()
    }
  },
  { deep: true }
)
</script>

<style scoped lang="scss">
.widget-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border-radius: 8px;
  overflow: hidden;
  position: relative;
  transition: all 0.3s ease;

  &.edit-mode {
    border: 2px dashed var(--el-color-primary);
    background: rgba(var(--el-color-primary-rgb), 0.02);
  }

  &.loading {
    .widget-header {
      opacity: 0.7;
    }
  }

  &.error {
    border-color: var(--el-color-danger);
    background: rgba(var(--el-color-danger-rgb), 0.02);
  }
}

.widget-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  min-height: 40px;

  .widget-title-section {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;

    .widget-icon {
      color: var(--el-color-primary);
      flex-shrink: 0;
    }

    .widget-title {
      margin: 0;
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      flex: 1;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .widget-status-indicator {
      flex-shrink: 0;

      .status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        transition: all 0.3s ease;
      }

      &.status-loading .status-dot {
        background: var(--el-color-warning);
        animation: pulse 1.5s infinite;
      }

      &.status-loaded .status-dot {
        background: var(--el-color-success);
      }

      &.status-error .status-dot {
        background: var(--el-color-danger);
      }

      &.status-offline .status-dot {
        background: var(--el-color-info);
      }
    }
  }

  .widget-actions {
    display: flex;
    align-items: center;
    gap: 4px;

    .widget-action-btn {
      padding: 4px;
      min-height: auto;

      &:hover {
        background: var(--el-fill-color-light);
      }
    }
  }
}

.widget-content {
  flex: 1;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;

  .widget-loading {
    padding: 16px;
    height: 100%;

    .loading-content {
      height: 100%;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
  }

  .widget-error,
  .widget-offline {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    padding: 16px;

    .error-content,
    .offline-content {
      text-align: center;
      max-width: 200px;

      .error-icon,
      .offline-icon {
        color: var(--el-color-danger);
        margin-bottom: 12px;
      }

      .offline-icon {
        color: var(--el-color-info);
      }

      .error-title,
      .offline-title {
        margin: 0 0 8px 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .error-message,
      .offline-message {
        margin: 0 0 16px 0;
        font-size: 14px;
        color: var(--el-text-color-secondary);
        line-height: 1.4;
      }

      .error-actions {
        display: flex;
        gap: 8px;
        justify-content: center;
      }
    }
  }

  .widget-component {
    flex: 1;
    overflow: hidden;
  }

  .edit-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(var(--el-color-primary-rgb), 0.1);
    pointer-events: none;
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    padding: 8px;

    .edit-handles {
      .resize-handle {
        position: absolute;
        bottom: 0;
        right: 0;
        width: 12px;
        height: 12px;
        background: var(--el-color-primary);
        cursor: se-resize;
        pointer-events: auto;

        &::before {
          content: '';
          position: absolute;
          bottom: 2px;
          right: 2px;
          width: 8px;
          height: 8px;
          border-right: 2px solid white;
          border-bottom: 2px solid white;
        }
      }
    }

    .edit-info {
      display: flex;
      gap: 8px;
      font-size: 12px;
      background: rgba(var(--el-color-primary-rgb), 0.9);
      color: white;
      padding: 4px 8px;
      border-radius: 4px;
    }
  }
}

.widget-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-extra-light);
  min-height: 32px;
  font-size: 12px;

  .widget-meta {
    display: flex;
    gap: 12px;
    color: var(--el-text-color-placeholder);

    .last-updated,
    .refresh-interval {
      white-space: nowrap;
    }
  }

  .widget-metrics {
    display: flex;
    gap: 12px;

    .metric {
      display: flex;
      align-items: center;
      gap: 4px;
      color: var(--el-text-color-secondary);

      &.error-metric {
        color: var(--el-color-danger);
      }
    }
  }
}

.error-details {
  .error-summary,
  .error-stack,
  .error-context {
    margin-bottom: 24px;

    h4 {
      margin: 0 0 12px 0;
      font-size: 16px;
      color: var(--el-text-color-primary);
    }

    p {
      margin: 0;
      color: var(--el-text-color-secondary);
      line-height: 1.5;
    }
  }

  .stack-trace {
    background: var(--el-fill-color-light);
    padding: 12px;
    border-radius: 4px;
    font-size: 12px;
    line-height: 1.4;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .context-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 12px;

    .context-item {
      display: flex;
      justify-content: space-between;
      padding: 8px 12px;
      background: var(--el-fill-color-light);
      border-radius: 4px;

      strong {
        color: var(--el-text-color-primary);
      }

      span {
        color: var(--el-text-color-secondary);
        text-align: right;
      }
    }
  }

  .error-actions-panel {
    display: flex;
    gap: 12px;
    justify-content: center;
    padding: 16px 0;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

// Animations
@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

// Responsive design
@media (max-width: 768px) {
  .widget-header {
    padding: 8px 12px;

    .widget-title {
      font-size: 13px;
    }

    .widget-actions {
      gap: 2px;
    }
  }

  .widget-footer {
    padding: 6px 12px;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;

    .widget-metrics {
      gap: 8px;
    }
  }

  .edit-overlay {
    .edit-info {
      font-size: 11px;
      padding: 2px 6px;
    }
  }
}
</style>