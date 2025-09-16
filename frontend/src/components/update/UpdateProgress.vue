<template>
  <div
    class="update-progress"
    :class="{
      'is-paused': update.status === 'paused',
      'is-error': update.status === 'failed',
      'is-stopping': update.status === 'stopping'
    }"
  >
    <!-- Header -->
    <div class="progress-header">
      <div class="container-info">
        <div class="container-name">
          <el-icon class="container-icon"><Box /></el-icon>
          <span class="name">{{ update.containerName }}</span>
          <el-tag
            :type="getStatusTagType(update.status)"
            size="small"
            effect="dark"
          >
            <el-icon>
              <component :is="getStatusIcon(update.status)" />
            </el-icon>
            {{ update.status.toUpperCase() }}
          </el-tag>
        </div>

        <div class="version-info">
          <span class="version-change">
            {{ update.fromVersion }} → {{ update.toVersion }}
          </span>
        </div>
      </div>

      <div class="progress-actions">
        <el-tooltip
          v-if="update.canPause && update.status === 'running'"
          content="Pause Update"
        >
          <el-button
            size="small"
            :icon="VideoPause"
            circle
            @click="$emit('pause', update.id)"
          />
        </el-tooltip>

        <el-tooltip
          v-if="update.status === 'paused'"
          content="Resume Update"
        >
          <el-button
            size="small"
            :icon="VideoPlay"
            type="primary"
            circle
            @click="$emit('resume', update.id)"
          />
        </el-tooltip>

        <el-tooltip
          v-if="update.canCancel"
          content="Cancel Update"
        >
          <el-button
            size="small"
            :icon="Close"
            type="danger"
            circle
            @click="handleCancel"
          />
        </el-tooltip>

        <el-dropdown trigger="click">
          <el-button size="small" :icon="MoreFilled" circle />
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="showLogs = true">
                <el-icon><Document /></el-icon>
                View Logs
              </el-dropdown-item>
              <el-dropdown-item @click="showDetails = true">
                <el-icon><View /></el-icon>
                View Details
              </el-dropdown-item>
              <el-dropdown-item
                v-if="update.status === 'failed' && update.canRetry"
                divided
                @click="$emit('retry', update.id)"
              >
                <el-icon><Refresh /></el-icon>
                Retry Update
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- Overall Progress -->
    <div class="overall-progress">
      <div class="progress-info">
        <div class="progress-text">
          <span class="current-step">
            Step {{ update.currentStep + 1 }} of {{ update.totalSteps }}
          </span>
          <span class="step-name">
            {{ getCurrentStepName() }}
          </span>
        </div>

        <div class="progress-stats">
          <span class="elapsed-time">
            {{ formatDuration(update.elapsedTime) }}
          </span>
          <span v-if="update.remainingTime" class="remaining-time">
            / ~{{ formatDuration(update.remainingTime) }}
          </span>
        </div>
      </div>

      <el-progress
        :percentage="update.progress"
        :status="getProgressStatus()"
        :stroke-width="8"
        :show-text="false"
      />

      <div class="progress-percentage">
        {{ Math.round(update.progress) }}%
      </div>
    </div>

    <!-- Step Progress (Expanded View) -->
    <el-collapse-transition>
      <div v-show="expanded" class="step-progress">
        <div class="steps-list">
          <div
            v-for="(step, index) in update.steps"
            :key="step.id"
            class="step-item"
            :class="{
              'is-current': index === update.currentStep,
              'is-completed': step.status === 'completed',
              'is-failed': step.status === 'failed',
              'is-running': step.status === 'running',
              'is-skipped': step.status === 'skipped'
            }"
          >
            <div class="step-indicator">
              <div class="step-number">{{ index + 1 }}</div>
              <div class="step-status-icon">
                <el-icon>
                  <component :is="getStepStatusIcon(step.status)" />
                </el-icon>
              </div>
            </div>

            <div class="step-content">
              <div class="step-header">
                <h4 class="step-title">{{ step.name }}</h4>
                <div class="step-meta">
                  <span v-if="step.duration" class="step-duration">
                    {{ formatDuration(step.duration) }}
                  </span>
                  <el-tag
                    v-if="step.status === 'failed'"
                    type="danger"
                    size="small"
                    effect="dark"
                  >
                    FAILED
                  </el-tag>
                  <el-tag
                    v-else-if="step.status === 'skipped'"
                    type="warning"
                    size="small"
                    effect="plain"
                  >
                    SKIPPED
                  </el-tag>
                </div>
              </div>

              <p class="step-description">{{ step.description }}</p>

              <!-- Step Progress Bar -->
              <div v-if="step.status === 'running'" class="step-progress-bar">
                <el-progress
                  :percentage="step.progress"
                  :stroke-width="4"
                  :show-text="false"
                />
              </div>

              <!-- Step Error -->
              <div v-if="step.error" class="step-error">
                <el-alert
                  :title="step.error"
                  type="error"
                  :closable="false"
                  show-icon
                />
                <div v-if="step.retryable" class="step-retry">
                  <el-button
                    size="small"
                    type="primary"
                    :loading="retryingStep === step.id"
                    @click="retryStep(step)"
                  >
                    Retry Step
                  </el-button>
                  <span class="retry-count">
                    Attempt {{ (step.retryCount || 0) + 1 }} of {{ step.maxRetries || 3 }}
                  </span>
                </div>
              </div>

              <!-- Step Logs -->
              <div v-if="step.logs.length > 0 && showStepLogs === step.id" class="step-logs">
                <div class="logs-header">
                  <span>Step Logs</span>
                  <el-button
                    text
                    size="small"
                    @click="showStepLogs = null"
                  >
                    Hide
                  </el-button>
                </div>
                <div class="logs-content">
                  <div
                    v-for="(log, logIndex) in step.logs.slice(-10)"
                    :key="logIndex"
                    class="log-line"
                  >
                    {{ log }}
                  </div>
                </div>
              </div>

              <div v-else-if="step.logs.length > 0" class="step-logs-toggle">
                <el-button
                  text
                  size="small"
                  @click="showStepLogs = step.id"
                >
                  View Step Logs ({{ step.logs.length }})
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-collapse-transition>

    <!-- Metrics (if available) -->
    <div v-if="update.metrics && expanded" class="update-metrics">
      <h4>Performance Metrics</h4>
      <div class="metrics-grid">
        <div class="metric-item">
          <span class="metric-label">Download Speed</span>
          <span class="metric-value">
            {{ formatSpeed(update.metrics.downloadSpeed) }}
          </span>
        </div>

        <div class="metric-item">
          <span class="metric-label">CPU Usage</span>
          <span class="metric-value">
            {{ Math.round(update.metrics.cpuUsage) }}%
          </span>
        </div>

        <div class="metric-item">
          <span class="metric-label">Memory Usage</span>
          <span class="metric-value">
            {{ formatBytes(update.metrics.memoryUsage) }}
          </span>
        </div>

        <div class="metric-item">
          <span class="metric-label">Disk I/O</span>
          <span class="metric-value">
            {{ formatSpeed(update.metrics.diskIo) }}
          </span>
        </div>
      </div>
    </div>

    <!-- Footer -->
    <div class="progress-footer">
      <div class="expand-toggle">
        <el-button
          text
          size="small"
          :icon="expanded ? 'ArrowUp' : 'ArrowDown'"
          @click="toggleExpanded"
        >
          {{ expanded ? 'Less Details' : 'More Details' }}
        </el-button>
      </div>

      <div class="strategy-info">
        <el-tag size="small" effect="plain">
          <el-icon><Setting /></el-icon>
          {{ update.strategy }}
        </el-tag>
      </div>
    </div>

    <!-- Logs Dialog -->
    <el-dialog
      v-model="showLogs"
      title="Update Logs"
      width="80%"
      :before-close="handleLogsClose"
    >
      <UpdateLogsViewer
        :update-id="update.id"
        :logs="update.logs"
        :auto-scroll="true"
      />
    </el-dialog>

    <!-- Details Dialog -->
    <el-dialog
      v-model="showDetails"
      title="Update Details"
      width="60%"
    >
      <UpdateDetailsViewer :update="update" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessageBox } from 'element-plus'
import {
  Box,
  VideoPause,
  VideoPlay,
  Close,
  MoreFilled,
  Document,
  View,
  Refresh,
  Setting
} from '@element-plus/icons-vue'

// Components
import UpdateLogsViewer from './UpdateLogsViewer.vue'
import UpdateDetailsViewer from './UpdateDetailsViewer.vue'

// Types
import type { RunningUpdate, UpdateStep } from '@/types/updates'

// Props
interface Props {
  update: RunningUpdate
}

const props = defineProps<Props>()

// Emits
defineEmits<{
  cancel: [updateId: string]
  pause: [updateId: string]
  resume: [updateId: string]
  retry: [updateId: string]
}>()

// Local state
const expanded = ref(false)
const showLogs = ref(false)
const showDetails = ref(false)
const showStepLogs = ref<number | null>(null)
const retryingStep = ref<number | null>(null)

// Methods
const toggleExpanded = () => {
  expanded.value = !expanded.value
}

const handleCancel = async () => {
  try {
    await ElMessageBox.confirm(
      'Are you sure you want to cancel this update? This may leave the container in an inconsistent state.',
      'Cancel Update',
      {
        confirmButtonText: 'Yes, Cancel',
        cancelButtonText: 'No',
        type: 'warning'
      }
    )
    $emit('cancel', props.update.id)
  } catch {
    // User cancelled
  }
}

const handleLogsClose = () => {
  showLogs.value = false
}

const retryStep = async (step: UpdateStep) => {
  retryingStep.value = step.id
  try {
    // This would call an API to retry the specific step
    // await updatesAPI.retryStep(props.update.id, step.id)
    console.log('Retrying step:', step.id)
  } catch (error) {
    console.error('Failed to retry step:', error)
  } finally {
    retryingStep.value = null
  }
}

const getCurrentStepName = () => {
  if (props.update.currentStep < props.update.steps.length) {
    return props.update.steps[props.update.currentStep]?.name || 'Unknown'
  }
  return 'Completed'
}

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'running': return 'primary'
    case 'paused': return 'warning'
    case 'stopping': return 'danger'
    case 'failed': return 'danger'
    default: return 'info'
  }
}

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'running': return 'Loading'
    case 'paused': return 'VideoPause'
    case 'stopping': return 'Close'
    case 'failed': return 'CircleClose'
    case 'queued': return 'Clock'
    default: return 'InfoFilled'
  }
}

const getProgressStatus = () => {
  if (props.update.status === 'failed') return 'exception'
  if (props.update.status === 'paused') return 'warning'
  return undefined
}

const getStepStatusIcon = (status: string) => {
  switch (status) {
    case 'completed': return 'Check'
    case 'failed': return 'Close'
    case 'running': return 'Loading'
    case 'skipped': return 'Minus'
    case 'pending': return 'Clock'
    default: return 'InfoFilled'
  }
}

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

const formatSpeed = (bytesPerSecond: number) => {
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  if (bytesPerSecond === 0) return '0 B/s'
  const i = Math.floor(Math.log(bytesPerSecond) / Math.log(1024))
  return `${(bytesPerSecond / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`
}

const formatBytes = (bytes: number) => {
  const sizes = ['B', 'KB', 'MB', 'GB']
  if (bytes === 0) return '0 B'
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`
}
</script>

<style scoped lang="scss">
.update-progress {
  padding: 20px;
  background: var(--el-bg-color);
  border: 2px solid var(--el-color-primary-light-7);
  border-radius: 12px;
  position: relative;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: linear-gradient(90deg, var(--el-color-primary), var(--el-color-success));
    animation: progress-pulse 2s infinite;
  }

  &.is-paused {
    border-color: var(--el-color-warning-light-5);

    &::before {
      background: var(--el-color-warning);
      animation: none;
    }
  }

  &.is-error {
    border-color: var(--el-color-danger-light-5);

    &::before {
      background: var(--el-color-danger);
      animation: none;
    }
  }

  &.is-stopping {
    border-color: var(--el-color-danger-light-7);

    &::before {
      background: var(--el-color-danger);
      animation: stopping-pulse 1s infinite;
    }
  }
}

@keyframes progress-pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

@keyframes stopping-pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.container-info {
  flex: 1;

  .container-name {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;

    .container-icon {
      color: var(--el-color-primary);
    }

    .name {
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .version-info {
    .version-change {
      font-size: 14px;
      color: var(--el-text-color-regular);
      font-family: monospace;
    }
  }
}

.progress-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.overall-progress {
  margin-bottom: 16px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;

  .progress-text {
    display: flex;
    flex-direction: column;
    gap: 2px;

    .current-step {
      font-size: 12px;
      color: var(--el-text-color-regular);
      font-weight: 500;
    }

    .step-name {
      font-size: 14px;
      color: var(--el-text-color-primary);
      font-weight: 600;
    }
  }

  .progress-stats {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--el-text-color-regular);

    .elapsed-time {
      font-weight: 600;
      color: var(--el-color-primary);
    }
  }
}

.progress-percentage {
  text-align: center;
  margin-top: 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-color-primary);
}

.step-progress {
  margin-bottom: 16px;
  padding: 16px;
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.steps-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-item {
  display: flex;
  gap: 16px;
  position: relative;

  &:not(:last-child)::after {
    content: '';
    position: absolute;
    left: 20px;
    top: 40px;
    bottom: -16px;
    width: 2px;
    background: var(--el-border-color);
  }

  &.is-completed::after {
    background: var(--el-color-success);
  }

  &.is-failed::after {
    background: var(--el-color-danger);
  }

  &.is-running::after {
    background: var(--el-color-primary);
  }
}

.step-indicator {
  position: relative;
  z-index: 1;

  .step-number {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--el-bg-color);
    border: 2px solid var(--el-border-color);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-regular);
  }

  .step-status-icon {
    position: absolute;
    bottom: -4px;
    right: -4px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--el-bg-color);
    border: 2px solid var(--el-bg-color);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
  }

  .step-item.is-completed & {
    .step-number {
      background: var(--el-color-success);
      border-color: var(--el-color-success);
      color: white;
    }

    .step-status-icon {
      color: var(--el-color-success);
    }
  }

  .step-item.is-failed & {
    .step-number {
      background: var(--el-color-danger);
      border-color: var(--el-color-danger);
      color: white;
    }

    .step-status-icon {
      color: var(--el-color-danger);
    }
  }

  .step-item.is-running & {
    .step-number {
      background: var(--el-color-primary);
      border-color: var(--el-color-primary);
      color: white;
      animation: running-pulse 1.5s infinite;
    }

    .step-status-icon {
      color: var(--el-color-primary);
    }
  }

  .step-item.is-current & {
    .step-number {
      border-color: var(--el-color-primary);
      color: var(--el-color-primary);
      font-weight: 700;
    }
  }

  .step-item.is-skipped & {
    .step-number {
      background: var(--el-color-warning-light-9);
      border-color: var(--el-color-warning-light-5);
      color: var(--el-color-warning);
    }

    .step-status-icon {
      color: var(--el-color-warning);
    }
  }
}

@keyframes running-pulse {
  0%, 100% {
    transform: scale(1);
    box-shadow: 0 0 0 0 rgba(64, 158, 255, 0.7);
  }
  50% {
    transform: scale(1.05);
    box-shadow: 0 0 0 8px rgba(64, 158, 255, 0);
  }
}

.step-content {
  flex: 1;

  .step-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    .step-title {
      margin: 0;
      font-size: 15px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .step-meta {
      display: flex;
      align-items: center;
      gap: 8px;

      .step-duration {
        font-size: 12px;
        color: var(--el-text-color-regular);
        font-weight: 500;
      }
    }
  }

  .step-description {
    margin: 0 0 12px 0;
    font-size: 13px;
    color: var(--el-text-color-regular);
    line-height: 1.4;
  }

  .step-progress-bar {
    margin-bottom: 12px;
  }

  .step-error {
    margin-bottom: 12px;

    .step-retry {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-top: 8px;

      .retry-count {
        font-size: 12px;
        color: var(--el-text-color-regular);
      }
    }
  }

  .step-logs,
  .step-logs-toggle {
    margin-top: 12px;
  }

  .step-logs {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
    overflow: hidden;

    .logs-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 12px;
      background: var(--el-bg-color-page);
      border-bottom: 1px solid var(--el-border-color-lighter);
      font-size: 12px;
      font-weight: 600;
      color: var(--el-text-color-regular);
    }

    .logs-content {
      max-height: 200px;
      overflow-y: auto;
      padding: 8px 12px;
      background: var(--el-color-black);
      font-family: monospace;
      font-size: 11px;
      line-height: 1.4;

      .log-line {
        color: var(--el-color-white);
        margin-bottom: 2px;
        word-break: break-all;

        &:last-child {
          margin-bottom: 0;
        }
      }
    }
  }
}

.update-metrics {
  margin-bottom: 16px;
  padding: 16px;
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;

  h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 16px;
  }

  .metric-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;

    .metric-label {
      font-size: 12px;
      color: var(--el-text-color-regular);
      margin-bottom: 4px;
    }

    .metric-value {
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }
}

.progress-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.strategy-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

@media (max-width: 768px) {
  .update-progress {
    padding: 16px;
  }

  .progress-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .progress-actions {
    justify-content: center;
  }

  .progress-info {
    flex-direction: column;
    gap: 8px;
    align-items: stretch;

    .progress-stats {
      justify-content: center;
    }
  }

  .step-item {
    flex-direction: column;
    gap: 8px;

    &:not(:last-child)::after {
      display: none;
    }

    .step-indicator {
      align-self: flex-start;
    }
  }

  .progress-footer {
    flex-direction: column;
    gap: 12px;
    align-items: center;
  }

  .update-metrics .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>