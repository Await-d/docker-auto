<template>
  <div
    class="container-card"
    :class="{
      'is-selected': selected,
      'is-loading': loading,
      'is-running': container.status === 'running',
      'is-stopped': container.status === 'exited',
      'is-unhealthy': container.health.status === 'unhealthy'
    }"
    @click="$emit('click')"
  >
    <!-- Selection Checkbox -->
    <div class="card-header">
      <el-checkbox
        :model-value="selected"
        @change="$emit('select')"
        @click.stop
        class="selection-checkbox"
      />

      <!-- Status Badge -->
      <el-tag
        :type="getStatusType(container.status)"
        size="small"
        effect="dark"
        class="status-badge"
      >
        <el-icon class="status-icon">
          <component :is="getStatusIcon(container.status)" />
        </el-icon>
        {{ container.status }}
      </el-tag>

      <!-- Update Available Badge -->
      <el-badge
        v-if="hasUpdate"
        is-dot
        type="warning"
        class="update-badge"
      >
        <el-icon><Download /></el-icon>
      </el-badge>
    </div>

    <!-- Container Info -->
    <div class="card-body">
      <!-- Name and Image -->
      <div class="container-info">
        <h3 class="container-name" :title="container.name">
          {{ container.name }}
        </h3>
        <div class="image-info">
          <span class="image-name" :title="container.image">
            {{ formatImageName(container.image) }}
          </span>
          <el-tag size="small" class="tag-badge">
            {{ container.tag }}
          </el-tag>
        </div>
      </div>

      <!-- Health Status -->
      <div class="health-section">
        <div class="health-indicator">
          <el-tag
            :type="getHealthType(container.health.status)"
            size="small"
            class="health-tag"
          >
            <el-icon>
              <component :is="getHealthIcon(container.health.status)" />
            </el-icon>
            {{ formatHealthStatus(container.health.status) }}
          </el-tag>

          <!-- Health Failing Streak -->
          <span
            v-if="container.health.status === 'unhealthy' && container.health.failingStreak > 0"
            class="failing-streak"
          >
            Failed {{ container.health.failingStreak }} times
          </span>
        </div>
      </div>

      <!-- Resource Usage -->
      <div class="resource-section">
        <div class="resource-item">
          <div class="resource-header">
            <span class="resource-label">CPU</span>
            <span class="resource-value">{{ formatPercentage(container.resourceUsage.cpu.usage) }}</span>
          </div>
          <el-progress
            :percentage="container.resourceUsage.cpu.usage"
            :show-text="false"
            :stroke-width="4"
            :color="getResourceColor(container.resourceUsage.cpu.usage)"
          />
        </div>

        <div class="resource-item">
          <div class="resource-header">
            <span class="resource-label">Memory</span>
            <span class="resource-value">
              {{ formatBytes(container.resourceUsage.memory.usage) }}
              ({{ formatPercentage(container.resourceUsage.memory.percentage) }})
            </span>
          </div>
          <el-progress
            :percentage="container.resourceUsage.memory.percentage"
            :show-text="false"
            :stroke-width="4"
            :color="getResourceColor(container.resourceUsage.memory.percentage)"
          />
        </div>
      </div>

      <!-- Network Traffic -->
      <div class="network-section">
        <div class="network-stats">
          <div class="network-item">
            <el-icon class="network-icon"><ArrowUp /></el-icon>
            <span class="network-label">TX:</span>
            <span class="network-value">{{ formatBytes(container.resourceUsage.network.txBytes) }}</span>
          </div>
          <div class="network-item">
            <el-icon class="network-icon"><ArrowDown /></el-icon>
            <span class="network-label">RX:</span>
            <span class="network-value">{{ formatBytes(container.resourceUsage.network.rxBytes) }}</span>
          </div>
        </div>
      </div>

      <!-- Ports -->
      <div v-if="container.ports.length > 0" class="ports-section">
        <div class="ports-header">
          <el-icon><Connection /></el-icon>
          <span>Ports</span>
        </div>
        <div class="ports-list">
          <el-tag
            v-for="port in container.ports.slice(0, 3)"
            :key="`${port.hostPort}-${port.containerPort}`"
            size="small"
            type="info"
            class="port-tag"
          >
            {{ port.hostPort }}:{{ port.containerPort }}/{{ port.protocol }}
          </el-tag>
          <el-tag
            v-if="container.ports.length > 3"
            size="small"
            type="info"
            class="port-tag more-ports"
          >
            +{{ container.ports.length - 3 }} more
          </el-tag>
        </div>
      </div>

      <!-- Labels -->
      <div v-if="displayLabels.length > 0" class="labels-section">
        <div class="labels-header">
          <el-icon><Tag /></el-icon>
          <span>Labels</span>
        </div>
        <div class="labels-list">
          <el-tag
            v-for="label in displayLabels"
            :key="label.key"
            size="small"
            class="label-tag"
            :title="`${label.key}=${label.value}`"
          >
            {{ label.key }}={{ formatLabelValue(label.value) }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- Card Footer -->
    <div class="card-footer">
      <!-- Timestamps -->
      <div class="timestamps">
        <div class="timestamp-item">
          <span class="timestamp-label">Created:</span>
          <span class="timestamp-value" :title="formatFullDate(container.createdAt)">
            {{ formatRelativeTime(container.createdAt) }}
          </span>
        </div>
        <div v-if="container.startedAt" class="timestamp-item">
          <span class="timestamp-label">Started:</span>
          <span class="timestamp-value" :title="formatFullDate(container.startedAt)">
            {{ formatRelativeTime(container.startedAt) }}
          </span>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="quick-actions">
        <!-- Start/Stop Button -->
        <el-button
          v-if="container.status === 'exited'"
          size="small"
          type="success"
          :loading="loading"
          @click.stop="$emit('action', 'start', container.id)"
          :disabled="!canPerformAction('start')"
        >
          <el-icon><VideoPlay /></el-icon>
        </el-button>

        <el-button
          v-else-if="container.status === 'running'"
          size="small"
          type="warning"
          :loading="loading"
          @click.stop="$emit('action', 'stop', container.id)"
          :disabled="!canPerformAction('stop')"
        >
          <el-icon><VideoPause /></el-icon>
        </el-button>

        <!-- Restart Button -->
        <el-button
          size="small"
          :loading="loading"
          @click.stop="$emit('action', 'restart', container.id)"
          :disabled="!canPerformAction('restart')"
        >
          <el-icon><Refresh /></el-icon>
        </el-button>

        <!-- Update Button -->
        <el-button
          v-if="hasUpdate"
          size="small"
          type="primary"
          :loading="loading"
          @click.stop="$emit('action', 'update', container.id)"
          :disabled="!canPerformAction('update')"
        >
          <el-icon><Download /></el-icon>
        </el-button>

        <!-- More Actions -->
        <el-dropdown
          @command="handleAction"
          @click.stop
        >
          <el-button size="small">
            <el-icon><MoreFilled /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                command="logs"
                :disabled="!canPerformAction('logs')"
              >
                <el-icon><Document /></el-icon>
                View Logs
              </el-dropdown-item>

              <el-dropdown-item
                command="terminal"
                :disabled="!canPerformAction('terminal') || container.status !== 'running'"
              >
                <el-icon><Monitor /></el-icon>
                Terminal
              </el-dropdown-item>

              <el-dropdown-item
                command="inspect"
                :disabled="!canPerformAction('inspect')"
              >
                <el-icon><View /></el-icon>
                Inspect
              </el-dropdown-item>

              <el-dropdown-item
                command="edit"
                :disabled="!canPerformAction('edit')"
                divided
              >
                <el-icon><Edit /></el-icon>
                Edit Configuration
              </el-dropdown-item>

              <el-dropdown-item
                command="clone"
                :disabled="!canPerformAction('clone')"
              >
                <el-icon><CopyDocument /></el-icon>
                Clone Container
              </el-dropdown-item>

              <el-dropdown-item
                command="backup"
                :disabled="!canPerformAction('backup')"
              >
                <el-icon><Upload /></el-icon>
                Create Backup
              </el-dropdown-item>

              <el-dropdown-item
                command="export"
                :disabled="!canPerformAction('export')"
              >
                <el-icon><Download /></el-icon>
                Export Config
              </el-dropdown-item>

              <el-dropdown-item
                command="delete"
                :disabled="!canPerformAction('delete')"
                divided
              >
                <el-icon><Delete /></el-icon>
                Delete Container
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- Loading Overlay -->
    <div v-if="loading" class="loading-overlay">
      <el-icon class="loading-spinner">
        <Loading />
      </el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  Download,
  ArrowUp,
  ArrowDown,
  Connection,
  Tag,
  VideoPlay,
  VideoPause,
  Refresh,
  MoreFilled,
  Document,
  Monitor,
  View,
  Edit,
  CopyDocument,
  Upload,
  Delete,
  Loading,
  SuccessFilled,
  CircleCloseFilled,
  WarningFilled,
  Clock,
  Warning,
  CircleCheckFilled,
  QuestionFilled
} from '@element-plus/icons-vue'

import { useAuthStore } from '@/store/auth'
import type { Container } from '@/types/container'

interface Props {
  container: Container
  selected?: boolean
  loading?: boolean
  hasUpdate?: boolean
}

interface Emits {
  (e: 'click'): void
  (e: 'select'): void
  (e: 'action', action: string, containerId: string): void
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
  loading: false,
  hasUpdate: false
})

const emit = defineEmits<Emits>()

const authStore = useAuthStore()

// Computed
const displayLabels = computed(() => {
  const labels = Object.entries(props.container.labels)
    .filter(([key]) => !key.startsWith('com.docker.'))
    .slice(0, 3)
    .map(([key, value]) => ({ key, value }))
  return labels
})

// Methods
function getStatusType(status: string): string {
  const types: Record<string, string> = {
    running: 'success',
    exited: 'info',
    paused: 'warning',
    restarting: 'warning',
    removing: 'danger',
    dead: 'danger',
    created: 'info'
  }
  return types[status] || 'info'
}

function getStatusIcon(status: string) {
  const icons: Record<string, any> = {
    running: SuccessFilled,
    exited: CircleCloseFilled,
    paused: WarningFilled,
    restarting: Loading,
    removing: Delete,
    dead: CircleCloseFilled,
    created: Clock
  }
  return icons[status] || QuestionFilled
}

function getHealthType(health: string): string {
  const types: Record<string, string> = {
    healthy: 'success',
    unhealthy: 'danger',
    starting: 'warning',
    none: 'info'
  }
  return types[health] || 'info'
}

function getHealthIcon(health: string) {
  const icons: Record<string, any> = {
    healthy: CircleCheckFilled,
    unhealthy: Warning,
    starting: Loading,
    none: QuestionFilled
  }
  return icons[health] || QuestionFilled
}

function formatHealthStatus(status: string): string {
  const statuses: Record<string, string> = {
    healthy: 'Healthy',
    unhealthy: 'Unhealthy',
    starting: 'Starting',
    none: 'No Check'
  }
  return statuses[status] || status
}

function formatImageName(image: string): string {
  // Truncate long image names
  const maxLength = 30
  if (image.length <= maxLength) return image

  const parts = image.split('/')
  const name = parts[parts.length - 1]

  if (name.length <= maxLength) {
    return `.../${name}`
  }

  return `${image.substring(0, maxLength - 3)}...`
}

function formatPercentage(value: number): string {
  return `${Math.round(value)}%`
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'

  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))

  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`
}

function formatLabelValue(value: string): string {
  const maxLength = 20
  if (value.length <= maxLength) return value
  return `${value.substring(0, maxLength - 3)}...`
}

function formatRelativeTime(date: Date | string): string {
  const now = new Date()
  const target = new Date(date)
  const diffMs = now.getTime() - target.getTime()

  const diffMinutes = Math.floor(diffMs / (1000 * 60))
  const diffHours = Math.floor(diffMinutes / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMinutes < 1) return 'Just now'
  if (diffMinutes < 60) return `${diffMinutes}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`

  return target.toLocaleDateString()
}

function formatFullDate(date: Date | string): string {
  return new Date(date).toLocaleString()
}

function getResourceColor(percentage: number): string {
  if (percentage < 50) return '#67c23a' // Green
  if (percentage < 80) return '#e6a23c' // Orange
  return '#f56c6c' // Red
}

function canPerformAction(action: string): boolean {
  const permissions: Record<string, string> = {
    start: 'container:start',
    stop: 'container:stop',
    restart: 'container:restart',
    update: 'container:update',
    logs: 'container:logs',
    terminal: 'container:exec',
    inspect: 'container:read',
    edit: 'container:update',
    clone: 'container:create',
    backup: 'container:backup',
    export: 'container:export',
    delete: 'container:delete'
  }

  const permission = permissions[action]
  return permission ? authStore.hasPermission(permission) : false
}

function handleAction(command: string) {
  emit('action', command, props.container.id)
}
</script>

<style scoped>
.container-card {
  background: white;
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s ease;
  cursor: pointer;
  position: relative;
  min-height: 320px;
  display: flex;
  flex-direction: column;
}

.container-card:hover {
  border-color: #409eff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
  transform: translateY(-2px);
}

.container-card.is-selected {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
}

.container-card.is-running {
  border-left: 4px solid #67c23a;
}

.container-card.is-stopped {
  border-left: 4px solid #909399;
}

.container-card.is-unhealthy {
  border-left: 4px solid #f56c6c;
}

.container-card.is-loading {
  pointer-events: none;
  opacity: 0.7;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #f8f9fa;
  border-bottom: 1px solid #e4e7ed;
}

.selection-checkbox {
  margin-right: 8px;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 4px;
}

.status-icon {
  font-size: 12px;
}

.update-badge {
  color: #e6a23c;
}

.card-body {
  padding: 16px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.container-info {
  margin-bottom: 8px;
}

.container-name {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.image-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.image-name {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #606266;
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tag-badge {
  flex-shrink: 0;
}

.health-section {
  display: flex;
  align-items: center;
  gap: 8px;
}

.health-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.health-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.failing-streak {
  font-size: 11px;
  color: #f56c6c;
  font-weight: 500;
}

.resource-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.resource-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.resource-label {
  font-size: 12px;
  font-weight: 500;
  color: #606266;
}

.resource-value {
  font-size: 11px;
  font-weight: 500;
  color: #303133;
}

.network-section {
  margin-top: 8px;
}

.network-stats {
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.network-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #606266;
}

.network-icon {
  font-size: 12px;
}

.network-label {
  font-weight: 500;
}

.network-value {
  color: #303133;
  font-weight: 500;
}

.ports-section {
  margin-top: 12px;
}

.ports-header {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 500;
  color: #606266;
}

.ports-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.port-tag {
  font-size: 10px;
  font-family: monospace;
}

.more-ports {
  color: #909399;
}

.labels-section {
  margin-top: 12px;
}

.labels-header {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 500;
  color: #606266;
}

.labels-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.label-tag {
  font-size: 10px;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-footer {
  padding: 12px 16px;
  border-top: 1px solid #e4e7ed;
  background: #f8f9fa;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
}

.timestamps {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.timestamp-item {
  display: flex;
  gap: 4px;
  font-size: 10px;
}

.timestamp-label {
  color: #909399;
  font-weight: 500;
}

.timestamp-value {
  color: #606266;
}

.quick-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.loading-spinner {
  font-size: 24px;
  color: #409eff;
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Responsive Design */
@media (max-width: 768px) {
  .container-card {
    min-height: auto;
  }

  .card-body {
    padding: 12px;
    gap: 12px;
  }

  .network-stats {
    flex-direction: column;
    gap: 8px;
  }

  .quick-actions {
    flex-wrap: wrap;
  }

  .card-footer {
    padding: 8px 12px;
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .timestamps {
    align-self: flex-start;
  }

  .quick-actions {
    justify-content: center;
  }
}
</style>