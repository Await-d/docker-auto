<template>
  <div class="update-policies">
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
      <!-- Global Update Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Refresh /></el-icon>
            <span>全局更新设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="默认策略"
              prop="defaultStrategy"
              required
            >
              <el-select
                v-model="formData.defaultStrategy"
                placeholder="选择更新策略"
                @change="handleFieldChange('defaultStrategy', $event)"
              >
                <el-option label="自动" value="auto" />
                <el-option label="手动" value="manual" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="最大并发更新数"
              prop="maxConcurrentUpdates"
              required
            >
              <el-input-number
                v-model="formData.maxConcurrentUpdates"
                :min="1"
                :max="20"
                @change="handleFieldChange('maxConcurrentUpdates', $event)"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="重试次数" prop="retryAttempts" required>
              <el-input-number
                v-model="formData.retryAttempts"
                :min="0"
                :max="10"
                @change="handleFieldChange('retryAttempts', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="回滚超时"
              prop="rollbackTimeout"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.rollbackTimeout"
                  :min="30"
                  :max="3600"
                  :step="30"
                  @change="handleFieldChange('rollbackTimeout', $event)"
                />
                <span class="timeout-unit">秒</span>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Maintenance Windows -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Clock /></el-icon>
            <span>维护窗口</span>
          </div>
        </template>

        <div class="maintenance-windows">
          <div
            v-if="!formData.maintenanceWindows || formData.maintenanceWindows.length === 0"
            class="empty-state"
          >
            <el-empty description="未配置维护窗口">
              <el-button type="primary" @click="addMaintenanceWindow">
                添加维护窗口
              </el-button>
            </el-empty>
          </div>

          <div v-else class="windows-list">
            <div
              v-for="(window, index) in (formData.maintenanceWindows || [])"
              :key="window.id"
              class="window-item"
            >
              <div class="window-header">
                <el-input
                  v-model="window.name"
                  placeholder="窗口名称"
                  class="window-name"
                  @input="updateMaintenanceWindow(index)"
                />
                <el-switch
                  v-model="window.enabled"
                  @change="updateMaintenanceWindow(index)"
                />
                <el-button
                  type="text"
                  size="small"
                  class="danger-button"
                  @click="removeMaintenanceWindow(index)"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>

              <el-row :gutter="16">
                <el-col :span="8">
                  <el-form-item label="开始时间">
                    <el-time-picker
                      v-model="window.startTime"
                      format="HH:mm"
                      value-format="HH:mm"
                      @change="updateMaintenanceWindow(index)"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="结束时间">
                    <el-time-picker
                      v-model="window.endTime"
                      format="HH:mm"
                      value-format="HH:mm"
                      @change="updateMaintenanceWindow(index)"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="星期">
                    <el-checkbox-group
                      v-model="window.dayOfWeek"
                      @change="updateMaintenanceWindow(index)"
                    >
                      <el-checkbox :label="0"> 日 </el-checkbox>
                      <el-checkbox :label="1"> 一 </el-checkbox>
                      <el-checkbox :label="2"> 二 </el-checkbox>
                      <el-checkbox :label="3"> 三 </el-checkbox>
                      <el-checkbox :label="4"> 四 </el-checkbox>
                      <el-checkbox :label="5"> 五 </el-checkbox>
                      <el-checkbox :label="6"> 六 </el-checkbox>
                    </el-checkbox-group>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>

            <el-button type="primary" @click="addMaintenanceWindow">
              <el-icon><Plus /></el-icon>
              添加窗口
            </el-button>
          </div>
        </div>
      </el-card>

      <!-- Version Comparison -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><DocumentCopy /></el-icon>
            <span>版本比较规则</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="语义化版本控制">
              <el-switch
                v-model="formData.semanticVersioning"
                @change="handleFieldChange('semanticVersioning', $event)"
              />
              <div class="field-help">
                使用语义化版本进行比较
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="允许预发布版本">
              <el-switch
                v-model="formData.allowPrerelease"
                @change="handleFieldChange('allowPrerelease', $event)"
              />
              <div class="field-help">
包含预发布版本
</div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="安全更新优先级">
              <el-switch
                v-model="formData.securityUpdatePriority"
                @change="handleFieldChange('securityUpdatePriority', $event)"
              />
              <div class="field-help">
优先处理安全更新
</div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Notification Preferences -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Bell /></el-icon>
            <span>通知偏好设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="更新可用">
              <el-switch
                v-model="formData.notifyOnAvailable"
                @change="handleFieldChange('notifyOnAvailable', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="更新完成">
              <el-switch
                v-model="formData.notifyOnComplete"
                @change="handleFieldChange('notifyOnComplete', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="更新失败">
              <el-switch
                v-model="formData.notifyOnFailure"
                @change="handleFieldChange('notifyOnFailure', $event)"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import {
  Refresh,
  Clock,
  DocumentCopy,
  Bell,
  Plus,
  Delete,
} from "@element-plus/icons-vue";
import ConfigForm from "./forms/ConfigForm.vue";
import type { UpdateSettings, MaintenanceWindow } from "@/store/settings";

interface Props {
  modelValue: UpdateSettings;
  loading?: boolean;
  validationErrors?: Record<string, string[]>;
}

interface Emits {
  (e: "update:modelValue", value: UpdateSettings): void;
  (e: "field-change", field: string, value: any): void;
  (e: "field-validate", field: string, value: any): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const formData = ref<UpdateSettings>({
  defaultStrategy: "manual",
  maintenanceWindows: [],
  maxConcurrentUpdates: 3,
  retryAttempts: 3,
  retryDelay: 300,
  rollbackTimeout: 600,
  semanticVersioning: true,
  allowPrerelease: false,
  securityUpdatePriority: true,
  notifyOnAvailable: true,
  notifyOnComplete: true,
  notifyOnFailure: true,
} as any);

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

const formRules = computed(() => ({
  defaultStrategy: [
    {
      required: true,
      message: "默认策略为必填项",
      trigger: "change",
    },
  ],
  maxConcurrentUpdates: [
    {
      required: true,
      message: "最大并发更新数为必填项",
      trigger: "blur",
    },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 1 || value > 20) {
          callback(new Error("必须在1咁20之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  retryAttempts: [
    { required: true, message: "重试次数为必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 0 || value > 10) {
          callback(new Error("必须在0咁10之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  rollbackTimeout: [
    {
      required: true,
      message: "回滚超时为必填项",
      trigger: "blur",
    },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 30 || value > 3600) {
          callback(new Error("必须在30到3600秒之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
}));

const generateId = (): string => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9);
};

const addMaintenanceWindow = () => {
  // Ensure maintenanceWindows array exists
  if (!formData.value.maintenanceWindows) {
    formData.value.maintenanceWindows = [];
  }
  
  const newWindow: MaintenanceWindow = {
    id: generateId(),
    name: `窗口 ${formData.value.maintenanceWindows.length + 1}`,
    dayOfWeek: [1, 2, 3, 4, 5], // Monday to Friday
    startTime: "02:00",
    endTime: "04:00",
    timezone: "UTC",
    enabled: true,
  };

  formData.value.maintenanceWindows.push(newWindow);
  updateMaintenanceWindows();
};

const removeMaintenanceWindow = (index: number) => {
  if (formData.value.maintenanceWindows && formData.value.maintenanceWindows.length > index) {
    formData.value.maintenanceWindows.splice(index, 1);
    updateMaintenanceWindows();
  }
};

const updateMaintenanceWindow = (_index: number) => {
  updateMaintenanceWindows();
};

const updateMaintenanceWindows = () => {
  handleFieldChange("maintenanceWindows", formData.value.maintenanceWindows);
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

// Initialize form data from props
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      formData.value = {
        ...formData.value, // Keep existing defaults
        ...newValue, // Override with new values
        // Ensure required arrays are always present
        maintenanceWindows: newValue.maintenanceWindows || [],
      };
    }
  },
  { immediate: true, deep: true },
);
</script>

<style scoped lang="scss">
.update-policies {
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

  .maintenance-windows {
    .windows-list {
      .window-item {
        background: var(--el-fill-color-extra-light);
        border: 1px solid var(--el-border-color-lighter);
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 16px;

        .window-header {
          display: flex;
          align-items: center;
          gap: 12px;
          margin-bottom: 16px;

          .window-name {
            flex: 1;
          }

          .danger-button {
            color: var(--el-color-danger);
          }
        }

        :deep(.el-checkbox-group) {
          display: flex;
          gap: 8px;
          flex-wrap: wrap;

          .el-checkbox {
            margin-right: 0;
          }
        }
      }
    }
  }
}

@media (max-width: 768px) {
  .update-policies {
    .config-section {
      :deep(.el-card__body) {
        padding: 16px;
      }
    }

    .maintenance-windows {
      .windows-list {
        .window-item {
          .window-header {
            flex-direction: column;
            align-items: stretch;
            gap: 8px;
          }

          :deep(.el-checkbox-group) {
            .el-checkbox {
              flex: 1;
              min-width: 60px;
              justify-content: center;
            }
          }
        }
      }
    }
  }
}
</style>
