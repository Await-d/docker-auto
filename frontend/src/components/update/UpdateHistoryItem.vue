<template>
  <div
    class="update-history-item"
    :class="[viewMode, `status-${item.status}`]"
  >
    <div v-if="viewMode === 'timeline'" class="timeline-indicator">
      <div class="timeline-dot" :class="getStatusClass(item.status)">
        <el-icon>
          <component :is="getStatusIcon(item.status)" />
        </el-icon>
      </div>
    </div>

    <div class="item-content">
      <div class="item-header">
        <div class="container-info">
          <h4 class="container-name">{{ item.containerName }}</h4>
          <div class="version-change">
            <el-tag size="small" type="info">{{ item.fromVersion }}</el-tag>
            <el-icon><Right /></el-icon>
            <el-tag size="small" type="primary">{{ item.toVersion }}</el-tag>
          </div>
        </div>

        <div class="item-status">
          <el-tag
            :type="getStatusTagType(item.status)"
            size="small"
          >
            <el-icon>
              <component :is="getStatusIcon(item.status)" />
            </el-icon>
            {{ item.status.toUpperCase() }}
          </el-tag>
          <span class="update-time">{{ formatTime(item.startedAt) }}</span>
        </div>
      </div>

      <div class="item-meta">
        <div class="meta-item">
          <el-icon><User /></el-icon>
          <span>{{ item.triggeredBy }}</span>
        </div>
        <div class="meta-item">
          <el-icon><Clock /></el-icon>
          <span v-if="item.duration">{{ formatDuration(item.duration) }}</span>
          <span v-else>-</span>
        </div>
        <div v-if="item.updateType" class="meta-item">
          <el-icon><Tag /></el-icon>
          <span>{{ item.updateType }}</span>
        </div>
      </div>

      <div v-if="item.error && item.status === 'failed'" class="error-info">
        <el-alert
          :title="item.error"
          type="error"
          :closable="false"
          show-icon
        />
      </div>

      <div class="item-actions">
        <el-button
          size="small"
          :icon="View"
          @click="$emit('view-details', item)"
        >
          Details
        </el-button>

        <el-button
          v-if="item.logs?.length"
          size="small"
          :icon="Document"
          @click="$emit('view-logs', item)"
        >
          Logs
        </el-button>

        <el-button
          v-if="item.canRollback && item.status === 'completed'"
          size="small"
          :icon="RefreshLeft"
          type="warning"
          @click="$emit('rollback', item)"
        >
          Rollback
        </el-button>

        <el-button
          v-if="item.status === 'failed'"
          size="small"
          :icon="Refresh"
          type="primary"
          @click="$emit('retry', item)"
        >
          Retry
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Right,
  User,
  Clock,
  Tag,
  View,
  Document,
  RefreshLeft,
  Refresh
} from '@element-plus/icons-vue'

import type { UpdateHistoryItem } from '@/types/updates'

// Props
interface Props {
  item: UpdateHistoryItem
  viewMode?: 'timeline' | 'card' | 'table'
}

const props = withDefaults(defineProps<Props>(), {
  viewMode: 'timeline'
})

// Emits
defineEmits<{
  rollback: [item: UpdateHistoryItem]
  retry: [item: UpdateHistoryItem]
  'view-logs': [item: UpdateHistoryItem]
  'view-details': [item: UpdateHistoryItem]
}>()

// Methods
const getStatusClass = (status: string) => {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'error'
    case 'cancelled': return 'warning'
    case 'running': return 'primary'
    default: return 'info'
  }
}

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'danger'
    case 'cancelled': return 'warning'
    case 'running': return 'primary'
    default: return 'info'
  }
}

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'completed': return 'Check'
    case 'failed': return 'Close'
    case 'cancelled': return 'Warning'
    case 'running': return 'Loading'
    default: return 'InfoFilled'
  }
}

const formatTime = (dateString: string) => {
  return new Date(dateString).toLocaleString()
}

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}
</script>

<style scoped lang="scss">
.update-history-item {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--el-color-primary-light-7);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }

  &.timeline {
    margin-left: 40px;
  }

  &.card {
    margin-bottom: 12px;
  }

  &.status-failed {
    border-left: 4px solid var(--el-color-danger);
  }

  &.status-completed {
    border-left: 4px solid var(--el-color-success);
  }

  &.status-running {
    border-left: 4px solid var(--el-color-primary);
  }
}

.timeline-indicator {
  position: absolute;
  left: 12px;
  top: 20px;

  .timeline-dot {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    color: white;
    z-index: 2;

    &.success {
      background: var(--el-color-success);
    }

    &.error {
      background: var(--el-color-danger);
    }

    &.warning {
      background: var(--el-color-warning);
    }

    &.primary {
      background: var(--el-color-primary);
    }

    &.info {
      background: var(--el-color-info);
    }
  }
}

.item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;

  .container-info {
    flex: 1;

    .container-name {
      margin: 0 0 8px 0;
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .version-change {
      display: flex;
      align-items: center;
      gap: 8px;
    }
  }

  .item-status {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 4px;

    .update-time {
      font-size: 12px;
      color: var(--el-text-color-regular);
    }
  }
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 13px;
  color: var(--el-text-color-regular);

  .meta-item {
    display: flex;
    align-items: center;
    gap: 4px;
  }
}

.error-info {
  margin: 4px 0;
}

.item-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

@media (max-width: 768px) {
  .update-history-item {
    padding: 12px;

    &.timeline {
      margin-left: 24px;
    }

    .item-header {
      flex-direction: column;
      gap: 12px;
      align-items: stretch;

      .item-status {
        align-items: flex-start;
      }
    }

    .item-meta {
      flex-direction: column;
      align-items: flex-start;
      gap: 8px;
    }

    .item-actions {
      flex-wrap: wrap;
      justify-content: center;
    }
  }

  .timeline-indicator {
    left: 8px;

    .timeline-dot {
      width: 20px;
      height: 20px;
      font-size: 10px;
    }
  }
}
</style>