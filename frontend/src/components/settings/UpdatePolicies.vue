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
            <span>Global Update Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Default Strategy" prop="defaultStrategy" required>
              <el-select
                v-model="formData.defaultStrategy"
                placeholder="Select update strategy"
                @change="handleFieldChange('defaultStrategy', $event)"
              >
                <el-option label="Automatic" value="auto" />
                <el-option label="Manual" value="manual" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Max Concurrent Updates" prop="maxConcurrentUpdates" required>
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
            <el-form-item label="Retry Attempts" prop="retryAttempts" required>
              <el-input-number
                v-model="formData.retryAttempts"
                :min="0"
                :max="10"
                @change="handleFieldChange('retryAttempts', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Rollback Timeout" prop="rollbackTimeout" required>
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.rollbackTimeout"
                  :min="30"
                  :max="3600"
                  :step="30"
                  @change="handleFieldChange('rollbackTimeout', $event)"
                />
                <span class="timeout-unit">seconds</span>
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
            <span>Maintenance Windows</span>
          </div>
        </template>

        <div class="maintenance-windows">
          <div v-if="formData.maintenanceWindows.length === 0" class="empty-state">
            <el-empty description="No maintenance windows configured">
              <el-button type="primary" @click="addMaintenanceWindow">
                Add Maintenance Window
              </el-button>
            </el-empty>
          </div>

          <div v-else class="windows-list">
            <div
              v-for="(window, index) in formData.maintenanceWindows"
              :key="window.id"
              class="window-item"
            >
              <div class="window-header">
                <el-input
                  v-model="window.name"
                  placeholder="Window name"
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
                  @click="removeMaintenanceWindow(index)"
                  class="danger-button"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>

              <el-row :gutter="16">
                <el-col :span="8">
                  <el-form-item label="Start Time">
                    <el-time-picker
                      v-model="window.startTime"
                      format="HH:mm"
                      value-format="HH:mm"
                      @change="updateMaintenanceWindow(index)"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="End Time">
                    <el-time-picker
                      v-model="window.endTime"
                      format="HH:mm"
                      value-format="HH:mm"
                      @change="updateMaintenanceWindow(index)"
                    />
                  </el-form-item>
                </el-col>
                <el-col :span="8">
                  <el-form-item label="Days">
                    <el-checkbox-group
                      v-model="window.dayOfWeek"
                      @change="updateMaintenanceWindow(index)"
                    >
                      <el-checkbox :label="0">Sun</el-checkbox>
                      <el-checkbox :label="1">Mon</el-checkbox>
                      <el-checkbox :label="2">Tue</el-checkbox>
                      <el-checkbox :label="3">Wed</el-checkbox>
                      <el-checkbox :label="4">Thu</el-checkbox>
                      <el-checkbox :label="5">Fri</el-checkbox>
                      <el-checkbox :label="6">Sat</el-checkbox>
                    </el-checkbox-group>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>

            <el-button type="primary" @click="addMaintenanceWindow">
              <el-icon><Plus /></el-icon>
              Add Window
            </el-button>
          </div>
        </div>
      </el-card>

      <!-- Version Comparison -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><DocumentCopy /></el-icon>
            <span>Version Comparison Rules</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Semantic Versioning">
              <el-switch
                v-model="formData.semanticVersioning"
                @change="handleFieldChange('semanticVersioning', $event)"
              />
              <div class="field-help">
                Use semantic versioning for comparison
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Allow Pre-release">
              <el-switch
                v-model="formData.allowPrerelease"
                @change="handleFieldChange('allowPrerelease', $event)"
              />
              <div class="field-help">
                Include pre-release versions
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Security Update Priority">
              <el-switch
                v-model="formData.securityUpdatePriority"
                @change="handleFieldChange('securityUpdatePriority', $event)"
              />
              <div class="field-help">
                Prioritize security updates
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
            <span>Notification Preferences</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="Update Available">
              <el-switch
                v-model="formData.notifyOnAvailable"
                @change="handleFieldChange('notifyOnAvailable', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Update Complete">
              <el-switch
                v-model="formData.notifyOnComplete"
                @change="handleFieldChange('notifyOnComplete', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Update Failure">
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
import { ref, computed, watch } from 'vue'
import {
  Refresh,
  Clock,
  DocumentCopy,
  Bell,
  Plus,
  Delete
} from '@element-plus/icons-vue'
import ConfigForm from './forms/ConfigForm.vue'
import type { UpdateSettings, MaintenanceWindow } from '@/store/settings'

interface Props {
  modelValue: UpdateSettings
  loading?: boolean
  validationErrors?: Record<string, string[]>
}

interface Emits {
  (e: 'update:modelValue', value: UpdateSettings): void
  (e: 'field-change', field: string, value: any): void
  (e: 'field-validate', field: string, value: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const formData = ref<UpdateSettings>({
  defaultStrategy: 'manual',
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
  notifyOnFailure: true
} as any)

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue)
})

const formRules = computed(() => ({
  defaultStrategy: [
    { required: true, message: 'Default strategy is required', trigger: 'change' }
  ],
  maxConcurrentUpdates: [
    { required: true, message: 'Max concurrent updates is required', trigger: 'blur' },
    { type: 'number', min: 1, max: 20, message: 'Must be between 1 and 20', trigger: 'blur' }
  ],
  retryAttempts: [
    { required: true, message: 'Retry attempts is required', trigger: 'blur' },
    { type: 'number', min: 0, max: 10, message: 'Must be between 0 and 10', trigger: 'blur' }
  ],
  rollbackTimeout: [
    { required: true, message: 'Rollback timeout is required', trigger: 'blur' },
    { type: 'number', min: 30, max: 3600, message: 'Must be between 30 and 3600 seconds', trigger: 'blur' }
  ]
}))

const generateId = (): string => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9)
}

const addMaintenanceWindow = () => {
  const newWindow: MaintenanceWindow = {
    id: generateId(),
    name: `Window ${formData.value.maintenanceWindows.length + 1}`,
    dayOfWeek: [1, 2, 3, 4, 5], // Monday to Friday
    startTime: '02:00',
    endTime: '04:00',
    timezone: 'UTC',
    enabled: true
  }

  formData.value.maintenanceWindows.push(newWindow)
  updateMaintenanceWindows()
}

const removeMaintenanceWindow = (index: number) => {
  formData.value.maintenanceWindows.splice(index, 1)
  updateMaintenanceWindows()
}

const updateMaintenanceWindow = (index: number) => {
  updateMaintenanceWindows()
}

const updateMaintenanceWindows = () => {
  handleFieldChange('maintenanceWindows', formData.value.maintenanceWindows)
}

const handleSave = () => {
  emit('update:modelValue', formData.value)
}

const handleReset = () => {
  formData.value = { ...props.modelValue }
}

const handleFieldChange = (field: string, value: any) => {
  emit('field-change', field, value)
}

// Initialize form data from props
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    formData.value = { ...newValue }
  }
}, { immediate: true, deep: true })
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