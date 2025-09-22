<template>
  <el-dialog
    v-model="visible"
    title="计划更新"
    width="800px"
    :before-close="handleClose"
  >
    <div class="scheduler-container">
      <!-- Selected Updates Summary -->
      <div v-if="selectedUpdates.length > 0" class="selected-summary">
        <h4>已选择的更新 ({{ selectedUpdates.length }})</h4>
        <div class="updates-list">
          <div
            v-for="update in getSelectedUpdateDetails()"
            :key="update.id"
            class="update-item"
          >
            <div class="update-info">
              <span class="container-name">{{ update.containerName }}</span>
              <span class="version-change">
                {{ update.currentVersion }} → {{ update.availableVersion }}
              </span>
            </div>
            <el-tag
:type="getRiskLevelType(update.riskLevel)" size="small"
>
              {{ update.riskLevel }}
            </el-tag>
          </div>
        </div>
      </div>

      <!-- Schedule Configuration -->
      <div class="schedule-config">
        <el-form
          ref="formRef"
          :model="scheduleForm"
          :rules="formRules"
          label-width="140px"
        >
          <!-- Schedule Type -->
          <el-form-item label="计划类型" prop="scheduleType">
            <el-radio-group v-model="scheduleForm.scheduleType">
              <el-radio label="once"> 一次性 </el-radio>
              <el-radio label="recurring"> 重复 </el-radio>
            </el-radio-group>
          </el-form-item>

          <!-- Date and Time -->
          <el-form-item
            v-if="scheduleForm.scheduleType === 'once'"
            label="日期和时间"
            prop="scheduledAt"
          >
            <el-date-picker
              v-model="scheduleForm.scheduledAt"
              type="datetime"
              placeholder="选择日期和时间"
              format="YYYY-MM-DD HH:mm"
              value-format="YYYY-MM-DD HH:mm:ss"
              :disabled-date="disabledDate"
              style="width: 100%"
            />
          </el-form-item>

          <!-- Recurring Pattern -->
          <el-form-item
            v-if="scheduleForm.scheduleType === 'recurring'"
            label="重复模式"
            prop="recurringPattern"
          >
            <div class="recurring-options">
              <el-select
                v-model="recurringType"
                placeholder="选择模式"
                style="width: 150px"
                @change="updateRecurringPattern"
              >
                <el-option label="每日" value="daily" />
                <el-option label="每周" value="weekly" />
                <el-option label="每月" value="monthly" />
                <el-option label="自定义" value="custom" />
              </el-select>

              <!-- Daily Options -->
              <div v-if="recurringType === 'daily'" class="pattern-details">
                <el-time-picker
                  v-model="dailyTime"
                  placeholder="选择时间"
                  format="HH:mm"
                  @change="updateRecurringPattern"
                />
              </div>

              <!-- Weekly Options -->
              <div v-if="recurringType === 'weekly'" class="pattern-details">
                <el-select
                  v-model="weeklyDays"
                  multiple
                  placeholder="选择天数"
                  style="width: 200px"
                  @change="updateRecurringPattern"
                >
                  <el-option label="周一" value="1" />
                  <el-option label="周二" value="2" />
                  <el-option label="周三" value="3" />
                  <el-option label="周四" value="4" />
                  <el-option label="周五" value="5" />
                  <el-option label="周六" value="6" />
                  <el-option label="周日" value="0" />
                </el-select>
                <el-time-picker
                  v-model="weeklyTime"
                  placeholder="选择时间"
                  format="HH:mm"
                  @change="updateRecurringPattern"
                />
              </div>

              <!-- Custom Cron -->
              <div v-if="recurringType === 'custom'" class="pattern-details">
                <el-input
                  v-model="scheduleForm.recurringPattern"
                  placeholder="0 2 * * 0 (Every Sunday at 2:00 AM)"
                  @input="validateCronExpression"
                />
                <div class="cron-help">
                  <el-tooltip
                    content="Cron expression format: minute hour day month day-of-week"
                  >
                    <el-button text type="primary" size="small">
                      <el-icon><QuestionFilled /></el-icon>
                      Cron Help
                    </el-button>
                  </el-tooltip>
                  <span v-if="cronDescription" class="cron-description">
                    {{ cronDescription }}
                  </span>
                </div>
              </div>
            </div>
          </el-form-item>

          <!-- Timezone -->
          <el-form-item label="时区" prop="timezone">
            <el-select
              v-model="scheduleForm.timezone"
              filterable
              placeholder="选择时区"
              style="width: 100%"
            >
              <el-option
                v-for="tz in timezones"
                :key="tz.value"
                :label="tz.label"
                :value="tz.value"
              />
            </el-select>
          </el-form-item>

          <!-- Update Strategy -->
          <el-form-item label="更新策略" prop="strategy">
            <el-select v-model="scheduleForm.strategy" style="width: 100%">
              <el-option label="重新创建" value="recreate">
                <div class="strategy-option">
                  <span>重新创建</span>
                  <small>停止容器，拉取镜像，创建新容器</small>
                </div>
              </el-option>
              <el-option label="滚动更新" value="rolling">
                <div class="strategy-option">
                  <span>滚动更新</span>
                  <small>使用负载均衡器进行零停机更新</small>
                </div>
              </el-option>
              <el-option label="蓝绿部署" value="blue-green">
                <div class="strategy-option">
                  <span>蓝绿部署</span>
                  <small>与现有环境并行部署，准备就绪后切换</small>
                </div>
              </el-option>
              <el-option label="金丝雀部署" value="canary">
                <div class="strategy-option">
                  <span>金丝雀部署</span>
                  <small>带监控的渐进式发布</small>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <!-- Advanced Options -->
          <el-form-item>
            <el-checkbox v-model="scheduleForm.rollbackOnFailure">
              失败时自动回滚
            </el-checkbox>
          </el-form-item>

          <el-form-item>
            <el-checkbox v-model="scheduleForm.runTests">
              更新前运行测试
            </el-checkbox>
          </el-form-item>

          <!-- Notification Settings -->
          <el-form-item label="通知设置">
            <div class="notification-settings">
              <el-checkbox v-model="scheduleForm.notifications.enabled">
                启用通知
              </el-checkbox>

              <div
                v-if="scheduleForm.notifications.enabled"
                class="notification-options"
              >
                <el-form-item label="提前通知">
                  <el-select
                    v-model="scheduleForm.notifyBefore"
                    style="width: 150px"
                  >
                    <el-option label="5分钟" :value="300000" />
                    <el-option label="15分钟" :value="900000" />
                    <el-option label="30分钟" :value="1800000" />
                    <el-option label="1小时" :value="3600000" />
                    <el-option label="2小时" :value="7200000" />
                  </el-select>
                </el-form-item>

                <el-checkbox-group v-model="scheduleForm.notifications.events">
                  <el-checkbox label="update_started">
                    Update started
                  </el-checkbox>
                  <el-checkbox label="update_completed">
                    Update completed
                  </el-checkbox>
                  <el-checkbox label="update_failed">
                    Update failed
                  </el-checkbox>
                  <el-checkbox label="rollback_started">
                    Rollback started
                  </el-checkbox>
                </el-checkbox-group>
              </div>
            </div>
          </el-form-item>

          <!-- Dependencies -->
          <el-form-item v-if="hasDependencies" label="依赖关系">
            <div class="dependencies-info">
              <el-alert
                title="某些容器存在依赖关系"
                type="warning"
                :closable="false"
                show-icon
              />
              <el-checkbox v-model="scheduleForm.respectDependencies">
                按依赖顺序更新容器
              </el-checkbox>
              <el-select
                v-model="scheduleForm.dependencyStrategy"
                style="width: 200px"
                :disabled="!scheduleForm.respectDependencies"
              >
                <el-option label="严格顺序" value="strict" />
                <el-option label="宽松顺序" value="loose" />
                <el-option label="忽略依赖" value="ignore" />
              </el-select>
            </div>
          </el-form-item>

          <!-- Preview -->
          <el-form-item label="预览">
            <div class="schedule-preview">
              <div class="preview-item">
                <strong>下次执行：</strong>
                <span>{{ getNextExecutionTime() }}</span>
              </div>
              <div
                v-if="scheduleForm.scheduleType === 'recurring'"
                class="preview-item"
              >
                <strong>模式：</strong>
                <span>{{ getPatternDescription() }}</span>
              </div>
              <div class="preview-item">
                <strong>预估持续时间：</strong>
                <span>{{ getEstimatedDuration() }}</span>
              </div>
            </div>
          </el-form-item>
        </el-form>
      </div>

      <!-- Calendar View -->
      <div v-if="scheduleForm.scheduleType === 'once'" class="calendar-view">
        <h4>日历</h4>
        <el-calendar v-model="calendarValue">
          <template #date-cell="{ data }">
            <div class="calendar-day">
              <span>{{ data.day.split("-").slice(-1)[0] }}</span>
              <div
                v-if="hasScheduledUpdates(data.day)"
                class="scheduled-indicator"
              >
                <el-badge
                  :value="getScheduledCount(data.day)"
                  class="scheduled-badge"
                />
              </div>
            </div>
          </template>
        </el-calendar>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose"> 取消 </el-button>
        <el-button
          type="primary"
          :loading="scheduling"
          :disabled="!isFormValid"
          @click="handleSchedule"
        >
          计划更新{{ selectedUpdates.length > 1 ? "" : "" }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { ElMessage } from "element-plus";
import { QuestionFilled } from "@element-plus/icons-vue";
import type { FormInstance, FormRules } from "element-plus";

// Store
import { useUpdatesStore } from "@/store/updates";

// Types
import type {
  UpdateStrategy,
  NotificationSettings,
  ContainerUpdate,
} from "@/types/updates";

// Props
interface Props {
  modelValue: boolean;
  selectedUpdates: string[];
}

const props = defineProps<Props>();

// Emits
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  scheduled: [];
}>();

// Store
const updatesStore = useUpdatesStore();

// Local state
const formRef = ref<FormInstance>();
const scheduling = ref(false);
const recurringType = ref("daily");
const dailyTime = ref(new Date());
const weeklyDays = ref(["1"]);
const weeklyTime = ref(new Date());
const calendarValue = ref(new Date());
const cronDescription = ref("");

const scheduleForm = ref({
  scheduleType: "once" as "once" | "recurring",
  scheduledAt: "",
  recurringPattern: "",
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  strategy: "recreate" as UpdateStrategy,
  rollbackOnFailure: true,
  runTests: false,
  notifyBefore: 300000, // 5 minutes
  respectDependencies: true,
  dependencyStrategy: "strict" as "strict" | "loose" | "ignore",
  notifications: {
    enabled: true,
    events: ["update_started", "update_completed", "update_failed"],
  } as NotificationSettings,
});

// Computed
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

const timezones = computed(() => [
  { label: "UTC", value: "UTC" },
  { label: "America/New_York (EST/EDT)", value: "America/New_York" },
  { label: "America/Chicago (CST/CDT)", value: "America/Chicago" },
  { label: "America/Denver (MST/MDT)", value: "America/Denver" },
  { label: "America/Los_Angeles (PST/PDT)", value: "America/Los_Angeles" },
  { label: "Europe/London (GMT/BST)", value: "Europe/London" },
  { label: "Europe/Berlin (CET/CEST)", value: "Europe/Berlin" },
  { label: "Asia/Tokyo (JST)", value: "Asia/Tokyo" },
  { label: "Asia/Shanghai (CST)", value: "Asia/Shanghai" },
  { label: "Australia/Sydney (AEDT/AEST)", value: "Australia/Sydney" },
]);

const hasDependencies = computed(() => {
  const selectedUpdateDetails = getSelectedUpdateDetails();
  return selectedUpdateDetails.some(
    (update) => update.dependencies.length > 0 || update.conflicts.length > 0,
  );
});

const isFormValid = computed(() => {
  if (scheduleForm.value.scheduleType === "once") {
    return !!scheduleForm.value.scheduledAt;
  } else {
    return !!scheduleForm.value.recurringPattern;
  }
});

// Form validation rules
const formRules: FormRules = {
  scheduledAt: [
    {
      required: true,
      message: "Please select a date and time",
      trigger: "blur",
    },
  ],
  recurringPattern: [
    {
      required: true,
      message: "Please set a recurring pattern",
      trigger: "blur",
    },
  ],
  strategy: [
    {
      required: true,
      message: "Please select an update strategy",
      trigger: "change",
    },
  ],
};

// Methods
const getSelectedUpdateDetails = (): ContainerUpdate[] => {
  return props.selectedUpdates
    .map((id) => updatesStore.availableUpdates.find((u) => u.id === id))
    .filter(Boolean) as ContainerUpdate[];
};

const getRiskLevelType = (riskLevel: string) => {
  switch (riskLevel) {
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

const disabledDate = (time: Date) => {
  return time.getTime() < Date.now() - 24 * 60 * 60 * 1000;
};

const updateRecurringPattern = () => {
  if (recurringType.value === "daily") {
    const time = dailyTime.value;
    const hour = time.getHours();
    const minute = time.getMinutes();
    scheduleForm.value.recurringPattern = `${minute} ${hour} * * *`;
  } else if (recurringType.value === "weekly") {
    const time = weeklyTime.value;
    const hour = time.getHours();
    const minute = time.getMinutes();
    const days = weeklyDays.value.join(",");
    scheduleForm.value.recurringPattern = `${minute} ${hour} * * ${days}`;
  }
  updateCronDescription();
};

const validateCronExpression = () => {
  // Basic cron validation
  const pattern = scheduleForm.value.recurringPattern;
  const parts = pattern.split(" ");

  if (parts.length !== 5) {
    cronDescription.value = "Invalid cron expression";
    return false;
  }

  updateCronDescription();
  return true;
};

const updateCronDescription = () => {
  const pattern = scheduleForm.value.recurringPattern;
  if (!pattern) return;

  try {
    // This is a simplified description generator
    // In a real app, you'd use a cron parser library
    cronDescription.value = parseCronExpression(pattern);
  } catch (error) {
    cronDescription.value = "Invalid cron expression";
  }
};

const parseCronExpression = (pattern: string): string => {
  const parts = pattern.split(" ");
  const [minute, hour, day, _month, dayOfWeek] = parts;

  let description = "Runs ";

  if (dayOfWeek !== "*") {
    const days = [
      "Sunday",
      "Monday",
      "Tuesday",
      "Wednesday",
      "Thursday",
      "Friday",
      "Saturday",
    ];
    const selectedDays = dayOfWeek.split(",").map((d) => days[parseInt(d)]);
    description += `every ${selectedDays.join(", ")} `;
  } else if (day !== "*") {
    description += `on day ${day} of each month `;
  } else {
    description += "daily ";
  }

  if (hour !== "*" && minute !== "*") {
    description += `at ${hour.padStart(2, "0")}:${minute.padStart(2, "0")}`;
  }

  return description;
};

const getNextExecutionTime = () => {
  if (
    scheduleForm.value.scheduleType === "once" &&
    scheduleForm.value.scheduledAt
  ) {
    return new Date(scheduleForm.value.scheduledAt).toLocaleString();
  } else if (
    scheduleForm.value.scheduleType === "recurring" &&
    scheduleForm.value.recurringPattern
  ) {
    // Calculate next execution based on cron pattern
    // This is simplified - in reality you'd use a cron library
    return calculateNextCronExecution(scheduleForm.value.recurringPattern);
  }
  return "未配置";
};

const calculateNextCronExecution = (_pattern: string): string => {
  // Simplified next execution calculation
  // In a real app, use a proper cron library
  const now = new Date();
  const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
  return tomorrow.toLocaleString();
};

const getPatternDescription = () => {
  if (scheduleForm.value.recurringPattern) {
    return cronDescription.value || scheduleForm.value.recurringPattern;
  }
  return "未配置";
};

const getEstimatedDuration = () => {
  const selectedUpdateDetails = getSelectedUpdateDetails();
  const totalTime = selectedUpdateDetails.reduce(
    (sum, update) => sum + update.estimatedDowntime,
    0,
  );

  if (totalTime < 60) return `~${totalTime}s`;
  if (totalTime < 3600) return `~${Math.floor(totalTime / 60)}m`;
  return `~${Math.floor(totalTime / 3600)}h ${Math.floor((totalTime % 3600) / 60)}m`;
};

const hasScheduledUpdates = (_date: string): boolean => {
  // Check if there are scheduled updates on this date
  // This would come from the store in a real app
  return false;
};

const getScheduledCount = (_date: string): number => {
  // Return count of scheduled updates on this date
  return 0;
};

const handleSchedule = async () => {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();

    scheduling.value = true;

    // Schedule each selected update
    const promises = props.selectedUpdates.map((updateId) => {
      const scheduledAt =
        scheduleForm.value.scheduleType === "once"
          ? new Date(scheduleForm.value.scheduledAt)
          : new Date(); // For recurring, calculate next execution

      return updatesStore.scheduleUpdate(updateId, scheduledAt, {
        recurring: scheduleForm.value.scheduleType === "recurring",
        recurringPattern: scheduleForm.value.recurringPattern,
        notifyBefore: scheduleForm.value.notifyBefore,
      });
    });

    await Promise.all(promises);

    ElMessage.success(
      `成功计划了 ${props.selectedUpdates.length} 个更新`,
    );
    emit("scheduled");
    handleClose();
  } catch (error) {
    console.error("计划更新失败:", error);
    ElMessage.error("计划更新失败");
  } finally {
    scheduling.value = false;
  }
};

const handleClose = () => {
  visible.value = false;
};

// Watch for changes to update cron description
watch(() => scheduleForm.value.recurringPattern, updateCronDescription);

// Initialize daily time to 2 AM
dailyTime.value.setHours(2, 0, 0, 0);
weeklyTime.value.setHours(2, 0, 0, 0);

// Set default recurring pattern
updateRecurringPattern();
</script>

<style scoped lang="scss">
.scheduler-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

.selected-summary {
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

  .updates-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 150px;
    overflow-y: auto;
  }

  .update-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;

    .update-info {
      display: flex;
      flex-direction: column;
      gap: 2px;

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
  }
}

.schedule-config {
  .recurring-options {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;

    .pattern-details {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .cron-help {
      display: flex;
      align-items: center;
      gap: 8px;

      .cron-description {
        font-size: 12px;
        color: var(--el-text-color-regular);
        font-style: italic;
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

  .notification-settings {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .notification-options {
      margin-left: 24px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
  }

  .dependencies-info {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .schedule-preview {
    padding: 12px;
    background: var(--el-bg-color-page);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;

    .preview-item {
      display: flex;
      justify-content: space-between;
      margin-bottom: 8px;

      &:last-child {
        margin-bottom: 0;
      }

      strong {
        color: var(--el-text-color-primary);
      }

      span {
        color: var(--el-text-color-regular);
      }
    }
  }
}

.calendar-view {
  h4 {
    margin: 0 0 16px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .calendar-day {
    position: relative;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;

    .scheduled-indicator {
      position: absolute;
      top: 4px;
      right: 4px;

      .scheduled-badge {
        .el-badge__content {
          font-size: 10px;
          min-width: 16px;
          height: 16px;
          line-height: 16px;
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

@media (max-width: 768px) {
  .scheduler-container {
    max-height: 60vh;
  }

  .recurring-options {
    flex-direction: column;
    align-items: stretch;

    .pattern-details {
      justify-content: center;
    }
  }

  .schedule-preview {
    .preview-item {
      flex-direction: column;
      gap: 4px;
    }
  }
}
</style>
