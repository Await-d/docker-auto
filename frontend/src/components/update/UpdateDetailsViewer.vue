<template>
  <div class="update-details">
    <div class="details-section">
      <h4>Update Information</h4>
      <div class="detail-grid">
        <div class="detail-item">
          <span class="label">Container:</span>
          <span class="value">{{ update.containerName }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Strategy:</span>
          <span class="value">{{ update.strategy }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Status:</span>
          <el-tag :type="getStatusType(update.status)">
            {{ update.status }}
          </el-tag>
        </div>
        <div class="detail-item">
          <span class="label">Progress:</span>
          <span class="value">{{ update.progress }}%</span>
        </div>
      </div>
    </div>

    <div class="details-section">
      <h4>Timing</h4>
      <div class="detail-grid">
        <div class="detail-item">
          <span class="label">Started:</span>
          <span class="value">{{ formatDateTime(update.startedAt) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Elapsed:</span>
          <span class="value">{{ formatDuration(update.elapsedTime) }}</span>
        </div>
        <div v-if="update.remainingTime" class="detail-item">
          <span class="label">Remaining:</span>
          <span class="value">{{ formatDuration(update.remainingTime) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Estimated:</span>
          <span class="value">{{ formatDuration(update.estimatedDuration) }}</span>
        </div>
      </div>
    </div>

    <div v-if="update.steps.length > 0" class="details-section">
      <h4>Steps ({{ update.currentStep + 1 }} / {{ update.totalSteps }})</h4>
      <div class="steps-list">
        <div
          v-for="(step, index) in update.steps"
          :key="step.id"
          class="step-summary"
          :class="{
            current: index === update.currentStep,
            completed: step.status === 'completed',
            failed: step.status === 'failed'
          }"
        >
          <div class="step-info">
            <span class="step-name">{{ step.name }}</span>
            <span class="step-status">{{ step.status }}</span>
          </div>
          <div v-if="step.status === 'running'" class="step-progress">
            <el-progress :percentage="step.progress" :stroke-width="4" />
          </div>
        </div>
      </div>
    </div>

    <div v-if="update.metrics" class="details-section">
      <h4>Performance Metrics</h4>
      <div class="metrics-grid">
        <div class="metric-item">
          <span class="metric-label">Download Speed:</span>
          <span class="metric-value">{{ formatSpeed(update.metrics.downloadSpeed) }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">CPU Usage:</span>
          <span class="metric-value">{{ Math.round(update.metrics.cpuUsage) }}%</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">Memory Usage:</span>
          <span class="metric-value">{{ formatBytes(update.metrics.memoryUsage) }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">Network I/O:</span>
          <span class="metric-value">{{ formatSpeed(update.metrics.networkIo) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RunningUpdate } from '@/types/updates'

// Props
interface Props {
  update: RunningUpdate
}

defineProps<Props>()

// Methods
const getStatusType = (status: string) => {
  switch (status) {
    case 'running': return 'primary'
    case 'paused': return 'warning'
    case 'failed': return 'danger'
    case 'completed': return 'success'
    default: return 'info'
  }
}

const formatDateTime = (dateString: string) => {
  return new Date(dateString).toLocaleString()
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
.update-details {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-height: 500px;
  overflow-y: auto;
}

.details-section {
  h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding-bottom: 8px;
  }
}

.detail-grid,
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.detail-item,
.metric-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px;
  background: var(--el-bg-color-page);
  border-radius: 4px;

  .label,
  .metric-label {
    font-size: 13px;
    color: var(--el-text-color-regular);
  }

  .value,
  .metric-value {
    font-size: 13px;
    font-weight: 500;
    color: var(--el-text-color-primary);
  }
}

.steps-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.step-summary {
  padding: 12px;
  background: var(--el-bg-color-page);
  border-radius: 6px;
  border: 1px solid var(--el-border-color-lighter);

  &.current {
    border-color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &.completed {
    border-color: var(--el-color-success);
    background: var(--el-color-success-light-9);
  }

  &.failed {
    border-color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
  }

  .step-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    .step-name {
      font-weight: 500;
      color: var(--el-text-color-primary);
    }

    .step-status {
      font-size: 12px;
      color: var(--el-text-color-regular);
      text-transform: uppercase;
    }
  }

  .step-progress {
    margin-top: 8px;
  }
}

@media (max-width: 768px) {
  .detail-grid,
  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .detail-item,
  .metric-item {
    flex-direction: column;
    align-items: stretch;
    gap: 4px;
  }
}
</style>