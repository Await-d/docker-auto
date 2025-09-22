<template>
  <el-dialog
    v-model="visible"
    title="回滚更新"
    width="60%"
    :before-close="handleClose"
  >
    <div
v-if="update" class="rollback-content"
>
      <el-alert
title="警告" type="warning"
:closable="false" show-icon
>
        <template #default>
          这将把容器
          <strong>{{ update.containerName }}</strong> 从版本
          <strong>{{ update.toVersion }}</strong> 回滚到
          <strong>{{ update.fromVersion }}</strong>。
        </template>
      </el-alert>

      <el-card shadow="never" class="update-info">
        <template #header>
          <h4>更新信息</h4>
        </template>

        <el-descriptions :column="2" border>
          <el-descriptions-item label="容器">
            {{ update.containerName }}
          </el-descriptions-item>
          <el-descriptions-item label="当前版本">
            {{ update.toVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="回滚到版本">
            {{ update.fromVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="更新日期">
            {{ formatDate(update.startedAt) }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-form :model="rollbackFormData" label-width="140px">
        <el-form-item label="回滚策略">
          <el-radio-group v-model="rollbackFormData.strategy">
            <el-radio value="immediate">
              <div class="strategy-option">
                <strong>立即执行</strong>
                <small>停止当前容器，使用之前的版本重新启动</small>
              </div>
            </el-radio>
            <el-radio value="graceful">
              <div class="strategy-option">
                <strong>优雅关闭</strong>
                <small>允许当前容器完成处理，然后再回滚</small>
              </div>
            </el-radio>
            <el-radio value="blue-green">
              <div class="strategy-option">
                <strong>蓝绿部署</strong>
                <small>在当前版本旁边启动之前的版本，然后切换流量</small>
              </div>
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="备份当前版本">
          <el-checkbox v-model="rollbackFormData.createBackup">
            回滚前创建当前容器状态的备份
          </el-checkbox>
        </el-form-item>

        <el-form-item label="健康检查">
          <el-checkbox v-model="rollbackFormData.performHealthCheck">
            回滚后执行健康检查
          </el-checkbox>
          <div
v-if="rollbackFormData.performHealthCheck" class="field-help"
>
            健康检查超时：
            <el-input-number
              v-model="rollbackFormData.healthCheckTimeout"
              :min="30"
              :max="600"
              size="small"
              style="width: 100px; margin: 0 8px"
            />
            秒
          </div>
        </el-form-item>

        <el-form-item label="通知">
          <el-checkbox v-model="rollbackFormData.notifyOnCompletion">
            回滚完成后发送通知
          </el-checkbox>
        </el-form-item>

        <el-form-item label="原因">
          <el-input
            v-model="rollbackFormData.reason"
            type="textarea"
            :rows="3"
            placeholder="可选：提供回滚原因..."
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>

      <el-card shadow="never" class="risk-assessment">
        <template #header>
          <h4>风险评估</h4>
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
        <el-button @click="handleClose">取消</el-button>
        <el-button
type="danger" @click="handleRollback"
:loading="rolling"
>
          <el-icon><RefreshLeft /></el-icon>
          确认回滚
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
    low: "低风险回滚",
    medium: "中风险回滚",
    high: "高风险回滚",
  }[level];
};

const getRiskDescription = () => {
  const level = assessRiskLevel();
  return {
    low: "最近的更新，使用优雅回滚策略。数据丢失或服务中断的风险最小。",
    medium:
      "更新后过了适中的时间或存在一些风险因素。建议考虑备份和健康检查。",
    high: "较旧的更新或立即回滚策略。出现问题的风险较高。建议进行备份和监控。",
  }[level];
};

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString();
};

const handleRollback = async () => {
  if (!props.update) return;

  try {
    await ElMessageBox.confirm(
      "您确定要回滚此更新吗？此操作无法撤销。",
      "确认回滚",
      {
        confirmButtonText: "是，回滚",
        cancelButtonText: "取消",
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

    ElMessage.success("回滚成功启动");
    handleClose();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("启动回滚失败");
      console.error("回滚错误:", error);
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
