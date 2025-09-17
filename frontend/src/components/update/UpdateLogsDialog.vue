<template>
  <el-dialog
    v-model="visible"
    title="Update Logs"
    width="80%"
    :before-close="handleClose"
  >
    <div class="logs-content">
      <div class="logs-header">
        <el-row :gutter="16" align="middle">
          <el-col :span="12">
            <h4 v-if="update">{{ update.containerName }} - Update Logs</h4>
          </el-col>
          <el-col :span="12" class="text-right">
            <el-button-group>
              <el-button
:icon="Download" size="small"
@click="downloadLogs"
>
                Download
              </el-button>
              <el-button
                :icon="Refresh"
                size="small"
                :loading="loading"
                @click="refreshLogs"
              >
                Refresh
              </el-button>
            </el-button-group>
          </el-col>
        </el-row>
      </div>

      <div class="logs-viewer">
        <el-scrollbar height="400px">
          <div class="log-lines">
            <div
              v-for="(line, index) in logLines"
              :key="index"
              class="log-line"
              :class="getLogLineClass(line)"
            >
              <span class="timestamp">{{ line.timestamp }}</span>
              <span class="level">{{ line.level }}</span>
              <span class="message">{{ line.message }}</span>
            </div>
          </div>
        </el-scrollbar>
      </div>

      <div v-if="!logLines.length && !loading" class="empty-logs">
        <el-empty description="No logs available" />
      </div>

      <div v-if="loading" class="loading-logs">
        <el-skeleton :rows="10" animated />
      </div>
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">Close</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { Download, Refresh } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import type { UpdateHistoryItem } from "@/types/updates";

interface LogLine {
  timestamp: string;
  level: "info" | "warn" | "error" | "debug";
  message: string;
}

interface Props {
  modelValue: boolean;
  update?: UpdateHistoryItem;
}

interface Emits {
  (e: "update:modelValue", value: boolean): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const visible = ref(false);
const loading = ref(false);
const logLines = ref<LogLine[]>([]);

const getLogLineClass = (line: LogLine) => {
  return {
    "log-info": line.level === "info",
    "log-warn": line.level === "warn",
    "log-error": line.level === "error",
    "log-debug": line.level === "debug",
  };
};

const fetchLogs = async () => {
  if (!props.update) return;

  loading.value = true;
  try {
    // Simulate API call to fetch logs
    await new Promise((resolve) => setTimeout(resolve, 1000));

    // Mock log data
    logLines.value = [
      {
        timestamp: "2024-01-15 10:30:15",
        level: "info",
        message:
          "Starting update process for container: " +
          props.update.containerName,
      },
      {
        timestamp: "2024-01-15 10:30:16",
        level: "info",
        message: "Pulling new image: " + props.update.toVersion,
      },
      {
        timestamp: "2024-01-15 10:30:45",
        level: "info",
        message: "Image pull completed successfully",
      },
      {
        timestamp: "2024-01-15 10:30:46",
        level: "info",
        message: "Stopping current container",
      },
      {
        timestamp: "2024-01-15 10:30:47",
        level: "info",
        message: "Container stopped successfully",
      },
      {
        timestamp: "2024-01-15 10:30:48",
        level: "info",
        message: "Creating new container with updated image",
      },
      {
        timestamp: "2024-01-15 10:30:50",
        level: "info",
        message: "Starting new container",
      },
      {
        timestamp: "2024-01-15 10:30:52",
        level: "info",
        message: "Container started successfully",
      },
      {
        timestamp: "2024-01-15 10:30:53",
        level: "info",
        message: "Update completed successfully",
      },
    ];
  } catch (error) {
    ElMessage.error("Failed to fetch logs");
    console.error("Failed to fetch logs:", error);
  } finally {
    loading.value = false;
  }
};

const refreshLogs = () => {
  fetchLogs();
};

const downloadLogs = () => {
  if (!logLines.value.length) {
    ElMessage.warning("No logs to download");
    return;
  }

  const logContent = logLines.value
    .map(
      (line) =>
        `[${line.timestamp}] ${line.level.toUpperCase()}: ${line.message}`,
    )
    .join("\n");

  const blob = new Blob([logContent], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `update-logs-${props.update?.containerName || "unknown"}-${Date.now()}.txt`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);

  ElMessage.success("Logs downloaded successfully");
};

const handleClose = () => {
  emit("update:modelValue", false);
};

watch(
  () => props.modelValue,
  (newValue) => {
    visible.value = newValue;
    if (newValue && props.update) {
      fetchLogs();
    }
  },
);

watch(visible, (newValue) => {
  emit("update:modelValue", newValue);
});
</script>

<style scoped lang="scss">
.logs-content {
  .logs-header {
    margin-bottom: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--el-border-color-light);

    h4 {
      margin: 0;
      color: var(--el-text-color-primary);
    }

    .text-right {
      text-align: right;
    }
  }

  .logs-viewer {
    background: var(--el-fill-color-darker);
    border-radius: 6px;
    padding: 12px;
    font-family: "Consolas", "Monaco", "Courier New", monospace;

    .log-lines {
      .log-line {
        display: flex;
        gap: 12px;
        padding: 2px 0;
        font-size: 13px;
        line-height: 1.4;
        border-bottom: 1px solid transparent;

        &:hover {
          background: var(--el-fill-color-light);
        }

        .timestamp {
          color: var(--el-text-color-regular);
          min-width: 150px;
          flex-shrink: 0;
        }

        .level {
          min-width: 50px;
          flex-shrink: 0;
          font-weight: 600;
          text-transform: uppercase;
        }

        .message {
          flex: 1;
          color: var(--el-text-color-primary);
        }

        &.log-info .level {
          color: var(--el-color-primary);
        }

        &.log-warn .level {
          color: var(--el-color-warning);
        }

        &.log-error .level {
          color: var(--el-color-danger);
        }

        &.log-debug .level {
          color: var(--el-color-info);
        }
      }
    }
  }

  .empty-logs,
  .loading-logs {
    padding: 40px 0;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style>
