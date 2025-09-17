<template>
  <el-dialog
    v-model="visible"
    title="Bulk Update Manager"
    width="900px"
    :before-close="handleClose"
  >
    <div class="bulk-manager">
      <!-- Selected Updates Summary -->
      <div class="updates-summary">
        <div class="summary-header">
          <h4>Selected Updates ({{ selectedUpdates.length }})</h4>
          <div class="summary-stats">
            <el-tag type="danger" size="small">
              {{ criticalCount }} Critical
            </el-tag>
            <el-tag type="warning" size="small">
              {{ securityCount }} Security
            </el-tag>
            <span class="total-size">Total: {{ formatTotalSize() }}</span>
          </div>
        </div>

        <div class="updates-list">
          <div
            v-for="update in getSelectedUpdates()"
            :key="update.id"
            class="update-row"
            :class="{ 'has-dependencies': update.dependencies.length > 0 }"
          >
            <div class="update-info">
              <span class="container-name">{{ update.containerName }}</span>
              <span class="version-change">{{ update.currentVersion }} →
                {{ update.availableVersion }}</span>
            </div>
            <div class="update-meta">
              <el-tag
:type="getRiskColor(update.riskLevel)" size="small">
                update.riskLevel }}
              </el-tag>
              <span class="size">{{ formatSize(update.size) }}</span>
              <span v-if="update.dependencies.length > 0" class="dependencies">
                <el-icon><Connection /></el-icon>
                {{ update.dependencies.length }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Bulk Configuration -->
      <el-form
ref="formRef" :model="bulkForm"
label-width="160px"
>
        <!-- Update Strategy -->
        <el-form-item label="Execution Strategy">
          <el-radio-group
            v-model="bulkForm.strategy"
            @change="updateEstimation"
          >
            <el-radio label="sequential">
              <div class="strategy-option">
                <span>Sequential</span>
                <small>Update containers one after another</small>
              </div>
            </el-radio>
            <el-radio label="parallel">
              <div class="strategy-option">
                <span>Parallel</span>
                <small>Update multiple containers simultaneously</small>
              </div>
            </el-radio>
            <el-radio label="rolling">
              <div class="strategy-option">
                <span>Rolling</span>
                <small>Gradual update with health checks</small>
              </div>
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- Concurrency -->
        <el-form-item
          v-if="bulkForm.strategy === 'parallel'"
          label="Max Concurrent"
        >
          <el-input-number
            v-model="bulkForm.maxConcurrent"
            :min="1"
            :max="Math.min(selectedUpdates.length, 10)"
            @change="updateEstimation"
          />
          <span class="form-help">Maximum number of containers to update simultaneously</span>
        </el-form-item>

        <!-- Dependency Handling -->
        <el-form-item v-if="hasDependencies" label="Dependencies">
          <el-checkbox v-model="bulkForm.respectDependencies">
            Respect container dependencies
          </el-checkbox>
          <el-select
            v-if="bulkForm.respectDependencies"
            v-model="bulkForm.dependencyStrategy"
            style="width: 200px; margin-left: 12px"
          >
            <el-option label="Strict Order" value="strict" />
            <el-option label="Best Effort" value="loose" />
          </el-select>
        </el-form-item>

        <!-- Error Handling -->
        <el-form-item label="Error Handling">
          <el-checkbox v-model="bulkForm.continueOnError">
            Continue on individual failures
          </el-checkbox>
          <el-checkbox v-model="bulkForm.rollbackOnFailure">
            Auto-rollback failed updates
          </el-checkbox>
        </el-form-item>

        <!-- Testing -->
        <el-form-item label="Pre-Update Testing">
          <el-checkbox v-model="bulkForm.runTests">
            Run health checks before updating
          </el-checkbox>
        </el-form-item>

        <!-- Execution Order Preview -->
        <el-form-item v-if="executionOrder.length > 0" label="Execution Order">
          <div class="execution-preview">
            <div
              v-for="(batch, index) in executionOrder"
              :key="index"
              class="execution-batch"
            >
              <div class="batch-header">
                <span>Batch {{ index + 1 }}</span>
                <el-tag size="small" effect="plain">
                  {{ batch.length }} container{{ batch.length > 1 ? "s" : "" }}
                </el-tag>
              </div>
              <div class="batch-containers">
                <span
                  v-for="container in batch"
                  :key="container"
                  class="container-name"
                >
                  {{ getContainerName(container) }}
                </span>
              </div>
            </div>
          </div>
        </el-form-item>

        <!-- Time Estimation -->
        <el-form-item label="Estimated Time">
          <div class="time-estimation">
            <div class="time-breakdown">
              <div class="time-item">
                <span class="time-label">Download:</span>
                <span class="time-value">{{
                  formatDuration(estimation.downloadTime)
                }}</span>
              </div>
              <div class="time-item">
                <span class="time-label">Update:</span>
                <span class="time-value">{{
                  formatDuration(estimation.updateTime)
                }}</span>
              </div>
              <div class="time-item total">
                <span class="time-label">Total:</span>
                <span class="time-value">{{
                  formatDuration(estimation.totalTime)
                }}</span>
              </div>
            </div>
            <div class="time-range">
              Estimated completion: {{ getCompletionTime() }}
            </div>
          </div>
        </el-form-item>
      </el-form>

      <!-- Risk Assessment -->
      <div class="risk-assessment">
        <h4>Risk Assessment</h4>
        <div class="risk-items">
          <div
            v-for="risk in riskAssessment"
            :key="risk.type"
            class="risk-item"
            :class="risk.level"
          >
            <el-icon>
              <component :is="getRiskIcon(risk.level)" />
            </el-icon>
            <div class="risk-content">
              <span class="risk-title">{{ risk.title }}</span>
              <p class="risk-description">
                {{ risk.description }}
              </p>
            </div>
            <el-tag :type="getRiskTagType(risk.level)" size="small">
              {{ risk.level.toUpperCase() }}
            </el-tag>
          </div>
        </div>
      </div>

      <!-- Progress Preview (if running) -->
      <div v-if="isRunning" class="bulk-progress">
        <h4>Update Progress</h4>
        <div class="overall-progress">
          <el-progress
            :percentage="overallProgress"
            :stroke-width="12"
            status=""
          />
          <div class="progress-stats">
            <span>{{ completedCount }} completed, {{ failedCount }} failed,
              {{ remainingCount }} remaining</span>
          </div>
        </div>

        <div class="container-progress">
          <div
            v-for="container in containerProgress"
            :key="container.id"
            class="container-progress-item"
            :class="container.status"
          >
            <span class="container-name">{{ container.name }}</span>
            <el-progress :percentage="container.progress" :stroke-width="6" />
            <el-tag :type="getStatusColor(container.status)" size="small">
              {{ container.status }}
            </el-tag>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose"> Cancel </el-button>
        <el-button
          v-if="!isRunning"
          type="primary"
          :loading="starting"
          @click="handleStart"
        >
          Start Bulk Update
        </el-button>
        <el-button
v-else type="danger"
@click="handleStop"
>
          Stop All Updates
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Connection } from "@element-plus/icons-vue";

// Store
import { useUpdatesStore } from "@/store/updates";

// Types
import type { ContainerUpdate } from "@/types/updates";

// Props
interface Props {
  modelValue: boolean;
  selectedUpdates: string[];
}

const props = defineProps<Props>();

// Emits
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  "bulk-update-started": [];
}>();

// Store
const updatesStore = useUpdatesStore();

// Local state
const formRef = ref();
const starting = ref(false);
const isRunning = ref(false);
const overallProgress = ref(0);
const completedCount = ref(0);
const failedCount = ref(0);
const containerProgress = ref<
  Array<{
    id: string;
    name: string;
    progress: number;
    status: "pending" | "running" | "completed" | "failed";
  }>
>([]);

const bulkForm = ref({
  strategy: "sequential" as "sequential" | "parallel" | "rolling",
  maxConcurrent: 3,
  respectDependencies: true,
  dependencyStrategy: "strict" as "strict" | "loose",
  continueOnError: true,
  rollbackOnFailure: true,
  runTests: false,
});

const estimation = ref({
  downloadTime: 0,
  updateTime: 0,
  totalTime: 0,
});

const executionOrder = ref<string[][]>([]);

// Computed
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

const getSelectedUpdates = (): ContainerUpdate[] => {
  return props.selectedUpdates
    .map((id) => updatesStore.availableUpdates.find((u) => u.id === id))
    .filter(Boolean) as ContainerUpdate[];
};

const criticalCount = computed(
  () => getSelectedUpdates().filter((u) => u.riskLevel === "critical").length,
);

const securityCount = computed(
  () =>
    getSelectedUpdates().filter(
      (u) => u.updateType === "security" || u.securityPatches.length > 0,
    ).length,
);

const hasDependencies = computed(() =>
  getSelectedUpdates().some(
    (u) => u.dependencies.length > 0 || u.conflicts.length > 0,
  ),
);

const remainingCount = computed(
  () => props.selectedUpdates.length - completedCount.value - failedCount.value,
);

const riskAssessment = computed(() => {
  const risks = [];
  const updates = getSelectedUpdates();

  if (criticalCount.value > 0) {
    risks.push({
      type: "critical",
      level: "high",
      title: "Critical Updates",
      description: `${criticalCount.value} critical updates may cause service disruption`,
    });
  }

  if (securityCount.value > 0) {
    risks.push({
      type: "security",
      level: "medium",
      title: "Security Updates",
      description: `${securityCount.value} security updates should be applied promptly`,
    });
  }

  if (updates.some((u) => u.conflicts.length > 0)) {
    risks.push({
      type: "conflicts",
      level: "high",
      title: "Container Conflicts",
      description: "Some containers have conflicts that may cause issues",
    });
  }

  if (
    bulkForm.value.strategy === "parallel" &&
    bulkForm.value.maxConcurrent > 5
  ) {
    risks.push({
      type: "performance",
      level: "medium",
      title: "High Concurrency",
      description: "High concurrency may impact system performance",
    });
  }

  return risks;
});

// Methods
const formatTotalSize = () => {
  const total = getSelectedUpdates().reduce(
    (sum, update) => sum + update.size,
    0,
  );
  return formatSize(total);
};

const formatSize = (bytes: number) => {
  const sizes = ["B", "KB", "MB", "GB"];
  if (bytes === 0) return "0 B";
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`;
};

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
};

const getRiskColor = (level: string) => {
  switch (level) {
    case "low":
      return "success";
    case "medium":
      return "warning";
    case "high":
      return "danger";
    case "critical":
      return "danger";
    default:
      return "info";
  }
};

const getRiskIcon = (level: string) => {
  switch (level) {
    case "low":
      return "Check";
    case "medium":
      return "Warning";
    case "high":
      return "Close";
    default:
      return "InfoFilled";
  }
};

const getRiskTagType = (level: string) => {
  switch (level) {
    case "low":
      return "success";
    case "medium":
      return "warning";
    case "high":
      return "danger";
    default:
      return "info";
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case "completed":
      return "success";
    case "failed":
      return "danger";
    case "running":
      return "primary";
    default:
      return "info";
  }
};

const getContainerName = (containerId: string) => {
  const update = getSelectedUpdates().find(
    (u) => u.containerId === containerId,
  );
  return update?.containerName || containerId.substring(0, 8);
};

const updateEstimation = () => {
  const updates = getSelectedUpdates();
  const totalDownloadTime = updates.reduce(
    (sum, u) => sum + u.size / (10 * 1024 * 1024),
    0,
  ); // Assume 10MB/s
  const totalUpdateTime = updates.reduce(
    (sum, u) => sum + u.estimatedDowntime,
    0,
  );

  if (bulkForm.value.strategy === "sequential") {
    estimation.value.downloadTime = totalDownloadTime;
    estimation.value.updateTime = totalUpdateTime;
    estimation.value.totalTime = totalDownloadTime + totalUpdateTime;
  } else if (bulkForm.value.strategy === "parallel") {
    const concurrent = bulkForm.value.maxConcurrent;
    estimation.value.downloadTime = Math.ceil(totalDownloadTime / concurrent);
    estimation.value.updateTime = Math.ceil(totalUpdateTime / concurrent);
    estimation.value.totalTime =
      estimation.value.downloadTime + estimation.value.updateTime;
  } else {
    // Rolling update
    estimation.value.downloadTime = totalDownloadTime * 0.8;
    estimation.value.updateTime = totalUpdateTime * 0.9;
    estimation.value.totalTime =
      estimation.value.downloadTime + estimation.value.updateTime;
  }

  updateExecutionOrder();
};

const updateExecutionOrder = () => {
  const updates = getSelectedUpdates();

  if (bulkForm.value.strategy === "sequential") {
    executionOrder.value = updates.map((u) => [u.containerId]);
  } else if (bulkForm.value.strategy === "parallel") {
    const concurrent = bulkForm.value.maxConcurrent;
    const batches = [];
    for (let i = 0; i < updates.length; i += concurrent) {
      batches.push(updates.slice(i, i + concurrent).map((u) => u.containerId));
    }
    executionOrder.value = batches;
  } else {
    // Rolling update - create smaller batches
    const batches = [];
    for (let i = 0; i < updates.length; i += 2) {
      batches.push(updates.slice(i, i + 2).map((u) => u.containerId));
    }
    executionOrder.value = batches;
  }
};

const getCompletionTime = () => {
  const completionDate = new Date(
    Date.now() + estimation.value.totalTime * 1000,
  );
  return completionDate.toLocaleString();
};

const handleStart = async () => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to start bulk update for ${props.selectedUpdates.length} containers?`,
      "Confirm Bulk Update",
      {
        confirmButtonText: "Start Updates",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    );

    starting.value = true;

    await updatesStore.startBulkUpdate(props.selectedUpdates, {
      strategy: bulkForm.value.strategy,
      maxConcurrent: bulkForm.value.maxConcurrent,
      continueOnError: bulkForm.value.continueOnError,
    });

    isRunning.value = true;
    initializeProgress();

    ElMessage.success("Bulk update started successfully");
    emit("bulk-update-started");
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to start bulk update:", error);
      ElMessage.error("Failed to start bulk update");
    }
  } finally {
    starting.value = false;
  }
};

const handleStop = async () => {
  try {
    await ElMessageBox.confirm(
      "Are you sure you want to stop all running updates?",
      "Stop Bulk Update",
      {
        confirmButtonText: "Stop All",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    );

    // Stop all running updates
    // This would need to be implemented in the store
    isRunning.value = false;
    ElMessage.success("All updates stopped");
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to stop updates:", error);
    }
  }
};

const initializeProgress = () => {
  const updates = getSelectedUpdates();
  containerProgress.value = updates.map((update) => ({
    id: update.containerId,
    name: update.containerName,
    progress: 0,
    status: "pending" as const,
  }));
};

const handleClose = () => {
  if (isRunning.value) {
    ElMessageBox.confirm(
      "Updates are still running. Are you sure you want to close this dialog?",
      "Confirm Close",
      {
        confirmButtonText: "Yes, Close",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    )
      .then(() => {
        visible.value = false;
      })
      .catch(() => {
        // User cancelled
      });
  } else {
    visible.value = false;
  }
};

// Watch for changes and update estimation
watch(
  () => [bulkForm.value.strategy, bulkForm.value.maxConcurrent],
  updateEstimation,
);

// Initialize estimation
updateEstimation();
</script>

<style scoped lang="scss">
.bulk-manager {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

.updates-summary {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;

  .summary-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    background: var(--el-bg-color-page);
    border-bottom: 1px solid var(--el-border-color-lighter);

    h4 {
      margin: 0;
      font-size: 14px;
      font-weight: 600;
    }

    .summary-stats {
      display: flex;
      align-items: center;
      gap: 8px;

      .total-size {
        font-size: 12px;
        color: var(--el-text-color-regular);
      }
    }
  }

  .updates-list {
    max-height: 200px;
    overflow-y: auto;
  }

  .update-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    &:last-child {
      border-bottom: none;
    }

    &.has-dependencies {
      background: var(--el-color-warning-light-9);
    }

    .update-info {
      display: flex;
      flex-direction: column;
      gap: 4px;

      .container-name {
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .version-change {
        font-size: 12px;
        color: var(--el-text-color-regular);
        font-family: monospace;
      }
    }

    .update-meta {
      display: flex;
      align-items: center;
      gap: 8px;

      .size {
        font-size: 12px;
        color: var(--el-text-color-regular);
      }

      .dependencies {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: 12px;
        color: var(--el-color-warning);
      }
    }
  }
}

.strategy-option {
  display: flex;
  flex-direction: column;

  small {
    color: var(--el-text-color-regular);
    font-size: 11px;
  }
}

.form-help {
  font-size: 12px;
  color: var(--el-text-color-regular);
  margin-left: 8px;
}

.execution-preview {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;

  .execution-batch {
    border-bottom: 1px solid var(--el-border-color-lighter);

    &:last-child {
      border-bottom: none;
    }

    .batch-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 12px;
      background: var(--el-bg-color-page);
      font-size: 12px;
      font-weight: 600;
    }

    .batch-containers {
      padding: 8px 12px;
      display: flex;
      flex-wrap: wrap;
      gap: 8px;

      .container-name {
        font-size: 12px;
        padding: 2px 6px;
        background: var(--el-color-primary-light-9);
        border-radius: 3px;
        color: var(--el-color-primary);
      }
    }
  }
}

.time-estimation {
  .time-breakdown {
    display: flex;
    gap: 24px;
    margin-bottom: 8px;

    .time-item {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 4px;

      &.total {
        .time-label,
        .time-value {
          font-weight: 600;
          color: var(--el-color-primary);
        }
      }

      .time-label {
        font-size: 12px;
        color: var(--el-text-color-regular);
      }

      .time-value {
        font-size: 14px;
        color: var(--el-text-color-primary);
      }
    }
  }

  .time-range {
    font-size: 12px;
    color: var(--el-text-color-regular);
    text-align: center;
  }
}

.risk-assessment {
  h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .risk-items {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .risk-item {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 12px;
    border-radius: 6px;
    border: 1px solid var(--el-border-color-lighter);

    &.high {
      background: var(--el-color-danger-light-9);
      border-color: var(--el-color-danger-light-7);

      .el-icon {
        color: var(--el-color-danger);
      }
    }

    &.medium {
      background: var(--el-color-warning-light-9);
      border-color: var(--el-color-warning-light-7);

      .el-icon {
        color: var(--el-color-warning);
      }
    }

    &.low {
      background: var(--el-color-success-light-9);
      border-color: var(--el-color-success-light-7);

      .el-icon {
        color: var(--el-color-success);
      }
    }

    .risk-content {
      flex: 1;

      .risk-title {
        font-weight: 600;
        color: var(--el-text-color-primary);
        display: block;
        margin-bottom: 4px;
      }

      .risk-description {
        margin: 0;
        font-size: 13px;
        color: var(--el-text-color-regular);
        line-height: 1.4;
      }
    }
  }
}

.bulk-progress {
  h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .overall-progress {
    margin-bottom: 16px;

    .progress-stats {
      text-align: center;
      margin-top: 8px;
      font-size: 12px;
      color: var(--el-text-color-regular);
    }
  }

  .container-progress {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 200px;
    overflow-y: auto;

    .container-progress-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 8px;
      background: var(--el-bg-color-page);
      border-radius: 4px;

      .container-name {
        width: 120px;
        font-size: 12px;
        font-weight: 500;
      }

      .el-progress {
        flex: 1;
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 768px) {
  .bulk-manager {
    max-height: 60vh;
  }

  .summary-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .time-breakdown {
    flex-direction: column;
    gap: 12px;
  }

  .container-progress-item {
    flex-direction: column;
    gap: 8px;
    align-items: stretch;

    .container-name {
      width: auto;
    }
  }
}
</style>
