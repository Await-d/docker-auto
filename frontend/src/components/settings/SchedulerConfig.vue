<template>
  <div class="scheduler-config">
    <ConfigForm
      v-model="formData"
      :rules="formRules"
      :saving="loading"
      :has-changes="hasChanges"
      :testable="false"
      @save="handleSave"
      @reset="handleReset"
      @field-change="handleFieldChange"
    >
      <!-- Global Scheduler Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Timer /></el-icon>
            <span>全局调度器设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="最大并发任务数"
              prop="maxConcurrentTasks"
              required
            >
              <el-input-number
                v-model="formData.maxConcurrentTasks"
                :min="1"
                :max="50"
                @change="handleFieldChange('maxConcurrentTasks', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="默认超时时间"
              prop="defaultTimeout"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.defaultTimeout"
                  :min="30"
                  :max="7200"
                  :step="30"
                  @change="handleFieldChange('defaultTimeout', $event)"
                />
                <span class="timeout-unit">秒</span>
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="死信队列">
              <el-switch
                v-model="formData.deadLetterQueueEnabled"
                @change="handleFieldChange('deadLetterQueueEnabled', $event)"
              />
              <div class="field-help">
存储失败任务以供手动审查
</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="默认时区" prop="timezone" required>
              <el-select
                v-model="formData.timezone"
                placeholder="选择时区"
                @change="handleFieldChange('timezone', $event)"
              >
                <el-option label="UTC" value="UTC" />
                <el-option label="America/New_York" value="America/New_York" />
                <el-option label="Europe/London" value="Europe/London" />
                <el-option label="Asia/Tokyo" value="Asia/Tokyo" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Retry Policy -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Refresh /></el-icon>
            <span>重试策略</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="启用重试">
              <el-switch
                v-model="formData.retryPolicy.enabled"
                @change="updateRetryPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="最大尝试次数">
              <el-input-number
                v-model="formData.retryPolicy.maxAttempts"
                :min="1"
                :max="10"
                :disabled="!formData.retryPolicy.enabled"
                @change="updateRetryPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="退避策略">
              <el-select
                v-model="formData.retryPolicy.backoffStrategy"
                :disabled="!formData.retryPolicy.enabled"
                @change="updateRetryPolicy"
              >
                <el-option label="线性" value="linear" />
                <el-option label="指数" value="exponential" />
                <el-option label="固定" value="fixed" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Performance Monitoring -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Monitor /></el-icon>
            <span>性能监控</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="指标保留期"
              prop="performanceMonitoring.metricsRetention"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.performanceMonitoring.metricsRetention"
                  :min="1"
                  :max="365"
                  @change="updatePerformanceMonitoring"
                />
                <span class="timeout-unit">天</span>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="健康检查间隔"
              prop="performanceMonitoring.healthCheckInterval"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.performanceMonitoring.healthCheckInterval"
                  :min="30"
                  :max="3600"
                  :step="30"
                  @change="updatePerformanceMonitoring"
                />
                <span class="timeout-unit">秒</span>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { Timer, Refresh, Monitor } from "@element-plus/icons-vue";
import ConfigForm from "./forms/ConfigForm.vue";
import type { SchedulerSettings } from "@/store/settings";

interface Props {
  modelValue: SchedulerSettings;
  loading?: boolean;
  validationErrors?: Record<string, string[]>;
}

interface Emits {
  (e: "update:modelValue", value: SchedulerSettings): void;
  (e: "field-change", field: string, value: any): void;
  (e: "field-validate", field: string, value: any): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const formData = ref<SchedulerSettings>({
  maxConcurrentTasks: 10,
  defaultTimeout: 300,
  retryPolicy: {
    enabled: true,
    maxAttempts: 3,
    backoffStrategy: "exponential",
    initialDelay: 30,
    maxDelay: 600,
    multiplier: 2,
  },
  deadLetterQueueEnabled: true,
  timezone: "UTC",
  taskTemplates: [],
  performanceMonitoring: {
    metricsRetention: 30,
    alertThresholds: {
      taskFailureRate: 10,
      avgExecutionTime: 300,
      queueSize: 100,
      resourceUsage: 80,
    },
    healthCheckInterval: 60,
  },
} as any);

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

const formRules = computed(() => ({
  maxConcurrentTasks: [
    {
      required: true,
      message: "最大并发任务数为必填项",
      trigger: "blur",
    },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 1 || value > 50) {
          callback(new Error("必须在1咁50之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  defaultTimeout: [
    { required: true, message: "默认超时时间为必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 30 || value > 7200) {
          callback(new Error("必须在30到7200秒之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  timezone: [
    { required: true, message: "时区为必填项", trigger: "change" },
  ],
}));

const updateRetryPolicy = () => {
  handleFieldChange("retryPolicy", formData.value.retryPolicy);
};

const updatePerformanceMonitoring = () => {
  handleFieldChange(
    "performanceMonitoring",
    formData.value.performanceMonitoring,
  );
};

const handleSave = () => {
  emit("update:modelValue", formData.value);
};

const handleReset = () => {
  formData.value = { ...props.modelValue };
};

const handleFieldChange = (field: string, value: any) => {
  emit("field-change", field, value);
};

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      formData.value = { ...newValue };
    }
  },
  { immediate: true, deep: true },
);
</script>

<style scoped lang="scss">
.scheduler-config {
  .config-section {
    margin-bottom: 24px;
    border: 1px solid var(--el-border-color-lighter);

    :deep(.el-card__header) {
      background: var(--el-fill-color-extra-light);
      border-bottom: 1px solid var(--el-border-color-lighter);
      padding: 16px 20px;

      .section-header {
        display: flex;
        align-items: center;
        gap: 8px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    :deep(.el-card__body) {
      padding: 24px;
    }
  }

  .timeout-input {
    display: flex;
    align-items: center;
    gap: 8px;

    .timeout-unit {
      color: var(--el-text-color-regular);
      font-size: 14px;
      white-space: nowrap;
    }
  }

  .field-help {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-top: 4px;
    line-height: 1.4;
  }
}
</style>
