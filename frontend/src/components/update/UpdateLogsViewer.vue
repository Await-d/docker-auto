<template>
  <div class="logs-viewer">
    <div class="logs-header">
      <div class="logs-info">
        <span class="logs-count">{{ logs.length }} log entries</span>
        <el-tag size="small" effect="plain">{{ updateId }}</el-tag>
      </div>
      <div class="logs-controls">
        <el-checkbox v-model="autoScroll">Auto-scroll</el-checkbox>
        <el-button size="small" :icon="Refresh" @click="refreshLogs">Refresh</el-button>
      </div>
    </div>

    <div
      ref="logsContainer"
      class="logs-container"
      :class="{ 'auto-scroll': autoScroll }"
    >
      <div
        v-for="(log, index) in logs"
        :key="index"
        class="log-line"
        :class="`level-${log.level}`"
      >
        <span class="log-timestamp">{{ formatTimestamp(log.timestamp) }}</span>
        <span class="log-level">[{{ log.level.toUpperCase() }}]</span>
        <span class="log-message">{{ log.message }}</span>
      </div>

      <div v-if="logs.length === 0" class="empty-logs">
        <el-empty description="No logs available" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import type { UpdateLog } from '@/types/updates'

// Props
interface Props {
  updateId: string
  logs: UpdateLog[]
  autoScroll?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  autoScroll: true
})

// Local state
const autoScroll = ref(props.autoScroll)
const logsContainer = ref<HTMLElement>()

// Methods
const formatTimestamp = (timestamp: string) => {
  return new Date(timestamp).toLocaleTimeString()
}

const refreshLogs = () => {
  // Emit event to parent to refresh logs
}

const scrollToBottom = () => {
  if (logsContainer.value && autoScroll.value) {
    logsContainer.value.scrollTop = logsContainer.value.scrollHeight
  }
}

// Watch for new logs and auto-scroll
watch(() => props.logs.length, () => {
  nextTick(() => {
    scrollToBottom()
  })
})
</script>

<style scoped lang="scss">
.logs-viewer {
  display: flex;
  flex-direction: column;
  height: 400px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  overflow: hidden;
}

.logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: var(--el-bg-color-page);
  border-bottom: 1px solid var(--el-border-color);

  .logs-info {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 13px;
    color: var(--el-text-color-regular);
  }

  .logs-controls {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

.logs-container {
  flex: 1;
  overflow-y: auto;
  background: var(--el-color-black);
  font-family: monospace;
  font-size: 12px;
  line-height: 1.4;
  padding: 8px;

  .log-line {
    display: flex;
    gap: 8px;
    margin-bottom: 2px;
    color: var(--el-color-white);

    .log-timestamp {
      color: var(--el-color-info-light-3);
      width: 80px;
      flex-shrink: 0;
    }

    .log-level {
      width: 60px;
      flex-shrink: 0;
      font-weight: 600;
    }

    .log-message {
      flex: 1;
      word-break: break-all;
    }

    &.level-debug .log-level {
      color: var(--el-color-info-light-3);
    }

    &.level-info .log-level {
      color: var(--el-color-primary-light-3);
    }

    &.level-warn .log-level {
      color: var(--el-color-warning-light-3);
    }

    &.level-error .log-level {
      color: var(--el-color-danger-light-3);
    }
  }

  .empty-logs {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--el-color-white);
  }
}
</style>