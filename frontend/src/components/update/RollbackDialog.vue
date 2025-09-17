<template>
  <el-dialog
    v-model="visible"
    title="Rollback Update"
    width="60%"
    :before-close="handleClose"
  >
    <div
v-if="update" class="rollback-content"
>
      <el-alert
title="Warning" type="warning"
:closable="false" show-icon
>
        <template #default>
          This will rollback the container
          <strong>{{ update.containerName }}</strong> from version
          <strong>{{ update.toVersion }}</strong> back to
          <strong>{{ update.fromVersion }}</strong>.
        </template>
      </el-alert>

      <el-card shadow="never" class="update-info">
        <template #header>
          <h4>Update Information</h4>
        </template>

        <el-descriptions :column="2" border>
          <el-descriptions-item label="Container">
            {{ update.containerName }}
          </el-descriptions-item>
          <el-descriptions-item label="Current Version">
            {{ update.toVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="Rollback to Version">
            {{ update.fromVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="Update Date">
            {{ formatDate(update.startedAt) }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-form :model="rollbackFormData" label-width="140px">
        <el-form-item label="Rollback Strategy">
          <el-radio-group v-model="rollbackFormData.strategy">
            <el-radio value="immediate">
              <div class="strategy-option">
                <strong>Immediate</strong>
                <small>Stop current container and start with previous version</small>
              </div>
            </el-radio>
            <el-radio value="graceful">
              <div class="strategy-option">
                <strong>Graceful</strong>
                <small>Allow current container to finish processing and then
                  rollback</small>
              </div>
            </el-radio>
            <el-radio value="blue-green">
              <div class="strategy-option">
                <strong>Blue-Green</strong>
                <small>Start previous version alongside current, then switch
                  traffic</small>
              </div>
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="Backup Current">
          <el-checkbox v-model="rollbackFormData.createBackup">
            Create backup of current container state before rollback
          </el-checkbox>
        </el-form-item>

        <el-form-item label="Health Check">
          <el-checkbox v-model="rollbackFormData.performHealthCheck">
            Perform health check after rollback
          </el-checkbox>
          <div
v-if="rollbackFormData.performHealthCheck" class="field-help"
>
            Health check timeout:
            <el-input-number
              v-model="rollbackFormData.healthCheckTimeout"
              :min="30"
              :max="600"
              size="small"
              style="width: 100px; margin: 0 8px"
            />
            seconds
          </div>
        </el-form-item>

        <el-form-item label="Notification">
          <el-checkbox v-model="rollbackFormData.notifyOnCompletion">
            Send notification when rollback is completed
          </el-checkbox>
        </el-form-item>

        <el-form-item label="Reason">
          <el-input
            v-model="rollbackFormData.reason"
            type="textarea"
            :rows="3"
            placeholder="Optional: Provide a reason for the rollback..."
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>

      <el-card shadow="never" class="risk-assessment">
        <template #header>
          <h4>Risk Assessment</h4>
        </template>

        <div class="risk-items">
          <div class="risk-item" :class="assessRiskLevel()">
            <el-icon><Warning /></el-icon>
            <div class="risk-content">
              <span class="risk-title">{{ getRiskTitle() }}</span>
              <p class="risk-description">
                {{ getRiskDescription() }}
              </p>
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">Cancel</el-button>
        <el-button
type="danger" @click="handleRollback"
:loading="rolling"
>
          <el-icon><RefreshLeft /></el-icon>
          Confirm Rollback
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { Warning, RefreshLeft } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { UpdateHistoryItem } from "@/types/updates";

interface RollbackFormData {
  strategy: "immediate" | "graceful" | "blue-green";
  createBackup: boolean;
  performHealthCheck: boolean;
  healthCheckTimeout: number;
  notifyOnCompletion: boolean;
  reason: string;
}

interface Props {
  modelValue: boolean;
  update?: UpdateHistoryItem;
}

interface Emits {
  (e: "update:modelValue", value: boolean): void;
  (
    e: "rollback",
    data: { update: UpdateHistoryItem; options: RollbackFormData },
  ): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const visible = ref(false);
const rolling = ref(false);

const rollbackFormData = ref<RollbackFormData>({
  strategy: "graceful",
  createBackup: true,
  performHealthCheck: true,
  healthCheckTimeout: 120,
  notifyOnCompletion: true,
  reason: "",
});

const assessRiskLevel = () => {
  if (!props.update) return "medium";

  // Simple risk assessment based on strategy and update recency
  const daysSinceUpdate = Math.floor(
    (Date.now() - new Date(props.update.startedAt).getTime()) /
      (1000 * 60 * 60 * 24),
  );

  if (rollbackFormData.value.strategy === "immediate" || daysSinceUpdate > 7) {
    return "high";
  } else if (daysSinceUpdate > 3) {
    return "medium";
  } else {
    return "low";
  }
};

const getRiskTitle = () => {
  const level = assessRiskLevel();
  return {
    low: "Low Risk Rollback",
    medium: "Medium Risk Rollback",
    high: "High Risk Rollback",
  }[level];
};

const getRiskDescription = () => {
  const level = assessRiskLevel();
  return {
    low: "Recent update with graceful rollback strategy. Minimal risk of data loss or service disruption.",
    medium:
      "Moderate time since update or some risk factors present. Consider backup and health checks.",
    high: "Older update or immediate rollback strategy. Higher risk of issues. Backup and monitoring recommended.",
  }[level];
};

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString();
};

const handleRollback = async () => {
  if (!props.update) return;

  try {
    await ElMessageBox.confirm(
      "Are you sure you want to rollback this update? This action cannot be undone.",
      "Confirm Rollback",
      {
        confirmButtonText: "Yes, Rollback",
        cancelButtonText: "Cancel",
        type: "warning",
      },
    );

    rolling.value = true;

    // Emit rollback event
    emit("rollback", {
      update: props.update,
      options: { ...rollbackFormData.value },
    });

    // Simulate rollback process
    await new Promise((resolve) => setTimeout(resolve, 3000));

    ElMessage.success("Rollback initiated successfully");
    handleClose();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("Failed to initiate rollback");
      console.error("Rollback error:", error);
    }
  } finally {
    rolling.value = false;
  }
};

const handleClose = () => {
  emit("update:modelValue", false);
};

watch(
  () => props.modelValue,
  (newValue) => {
    visible.value = newValue;
  },
);

watch(visible, (newValue) => {
  emit("update:modelValue", newValue);
});
</script>

<style scoped lang="scss">
.rollback-content {
  .update-info {
    margin: 16px 0;

    h4 {
      margin: 0;
      color: var(--el-text-color-primary);
    }
  }

  .strategy-option {
    display: flex;
    flex-direction: column;
    width: 100%;

    small {
      color: var(--el-text-color-regular);
      font-size: 11px;
    }
  }

  .field-help {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-top: 4px;
    display: flex;
    align-items: center;
  }

  .risk-assessment {
    margin-top: 16px;

    h4 {
      margin: 0;
      color: var(--el-text-color-primary);
    }

    .risk-items {
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
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.el-radio-group) {
  display: flex;
  flex-direction: column;
  gap: 12px;

  .el-radio {
    margin-right: 0;
    align-items: flex-start;

    .el-radio__label {
      width: 100%;
    }
  }
}
</style>
