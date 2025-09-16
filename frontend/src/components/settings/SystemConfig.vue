<template>
  <div class="system-config">
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
      <!-- System Information (Read-only) -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><InfoFilled /></el-icon>
            <span>System Information</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Version">
              <el-input
                :value="systemInfo.version"
                readonly
                class="readonly-input"
              >
                <template #suffix>
                  <el-tag type="success" size="small">Current</el-tag>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Build Date">
              <el-input
                :value="formatDate(systemInfo.buildDate)"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Runtime">
              <el-input
                :value="systemInfo.runtime"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Platform">
              <el-input
                :value="systemInfo.platform"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="Installation Path">
              <el-input
                :value="systemInfo.installPath"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- General Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Setting /></el-icon>
            <span>General Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="System Name" prop="systemName" required>
              <el-input
                v-model="formData.systemName"
                placeholder="Enter system name"
                maxlength="100"
                show-word-limit
                @input="handleFieldChange('systemName', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Default Timezone" prop="timezone" required>
              <el-select
                v-model="formData.timezone"
                placeholder="Select timezone"
                filterable
                @change="handleFieldChange('timezone', $event)"
              >
                <el-option
                  v-for="tz in timezones"
                  :key="tz.value"
                  :label="tz.label"
                  :value="tz.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="System Description" prop="systemDescription">
              <el-input
                v-model="formData.systemDescription"
                type="textarea"
                :rows="3"
                placeholder="Enter system description"
                maxlength="500"
                show-word-limit
                @input="handleFieldChange('systemDescription', $event)"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="Language" prop="language" required>
              <el-select
                v-model="formData.language"
                placeholder="Select language"
                @change="handleFieldChange('language', $event)"
              >
                <el-option
                  v-for="lang in languages"
                  :key="lang.value"
                  :label="lang.label"
                  :value="lang.value"
                >
                  <div class="language-option">
                    <span class="language-flag">{{ lang.flag }}</span>
                    <span class="language-name">{{ lang.label }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Date Format" prop="dateFormat" required>
              <el-select
                v-model="formData.dateFormat"
                placeholder="Select date format"
                @change="handleFieldChange('dateFormat', $event)"
              >
                <el-option
                  v-for="format in dateFormats"
                  :key="format.value"
                  :label="format.label"
                  :value="format.value"
                >
                  <div class="format-option">
                    <span class="format-pattern">{{ format.label }}</span>
                    <span class="format-example">{{ format.example }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Time Format" prop="timeFormat" required>
              <el-select
                v-model="formData.timeFormat"
                placeholder="Select time format"
                @change="handleFieldChange('timeFormat', $event)"
              >
                <el-option
                  v-for="format in timeFormats"
                  :key="format.value"
                  :label="format.label"
                  :value="format.value"
                >
                  <div class="format-option">
                    <span class="format-pattern">{{ format.label }}</span>
                    <span class="format-example">{{ format.example }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Session Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Clock /></el-icon>
            <span>Session Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="Session Timeout"
              prop="sessionTimeout"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.sessionTimeout"
                  :min="5"
                  :max="1440"
                  :step="5"
                  @change="handleFieldChange('sessionTimeout', $event)"
                />
                <span class="timeout-unit">minutes</span>
              </div>
              <div class="field-help">
                Session will expire after this period of inactivity
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Auto Logout Warning">
              <el-switch
                v-model="formData.autoLogoutWarning"
                @change="handleFieldChange('autoLogoutWarning', $event)"
              />
              <div class="field-help">
                Show warning before session expires
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Performance Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Monitor /></el-icon>
            <span>Performance Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item
              label="Max Concurrent Operations"
              prop="maxConcurrentOperations"
              required
            >
              <el-input-number
                v-model="formData.maxConcurrentOperations"
                :min="1"
                :max="100"
                @change="handleFieldChange('maxConcurrentOperations', $event)"
              />
              <div class="field-help">
                Maximum number of operations running simultaneously
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item
              label="Request Timeout"
              prop="requestTimeout"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.requestTimeout"
                  :min="5"
                  :max="300"
                  :step="5"
                  @change="handleFieldChange('requestTimeout', $event)"
                />
                <span class="timeout-unit">seconds</span>
              </div>
              <div class="field-help">
                Timeout for API requests
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item
              label="Cache TTL"
              prop="cacheTtl"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.cacheTtl"
                  :min="1"
                  :max="1440"
                  @change="handleFieldChange('cacheTtl', $event)"
                />
                <span class="timeout-unit">minutes</span>
              </div>
              <div class="field-help">
                Time to live for cached data
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="Resource Usage Limits"
              prop="resourceLimits"
            >
              <div class="resource-limits">
                <div class="limit-item">
                  <label>CPU Limit (%)</label>
                  <el-slider
                    v-model="formData.resourceLimits.cpu"
                    :min="10"
                    :max="100"
                    :step="5"
                    show-input
                    @change="handleResourceLimitChange"
                  />
                </div>
                <div class="limit-item">
                  <label>Memory Limit (%)</label>
                  <el-slider
                    v-model="formData.resourceLimits.memory"
                    :min="10"
                    :max="100"
                    :step="5"
                    show-input
                    @change="handleResourceLimitChange"
                  />
                </div>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="Log Retention"
              prop="logRetention"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.logRetention"
                  :min="1"
                  :max="365"
                  @change="handleFieldChange('logRetention', $event)"
                />
                <span class="timeout-unit">days</span>
              </div>
              <div class="field-help">
                How long to keep system logs
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Maintenance Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Tools /></el-icon>
            <span>Maintenance Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Auto Cleanup">
              <el-switch
                v-model="formData.autoCleanup"
                @change="handleFieldChange('autoCleanup', $event)"
              />
              <div class="field-help">
                Automatically clean up temporary files and logs
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Auto Updates">
              <el-switch
                v-model="formData.autoUpdates"
                @change="handleFieldChange('autoUpdates', $event)"
              />
              <div class="field-help">
                Automatically update system components
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item
              label="Maintenance Window"
              prop="maintenanceWindow"
            >
              <div class="maintenance-window">
                <div class="window-item">
                  <label>Start Time</label>
                  <el-time-picker
                    v-model="formData.maintenanceWindow.start"
                    format="HH:mm"
                    value-format="HH:mm"
                    @change="handleMaintenanceWindowChange"
                  />
                </div>
                <div class="window-item">
                  <label>End Time</label>
                  <el-time-picker
                    v-model="formData.maintenanceWindow.end"
                    format="HH:mm"
                    value-format="HH:mm"
                    @change="handleMaintenanceWindowChange"
                  />
                </div>
                <div class="window-item">
                  <label>Days</label>
                  <el-checkbox-group
                    v-model="formData.maintenanceWindow.days"
                    @change="handleMaintenanceWindowChange"
                  >
                    <el-checkbox :label="0">Sun</el-checkbox>
                    <el-checkbox :label="1">Mon</el-checkbox>
                    <el-checkbox :label="2">Tue</el-checkbox>
                    <el-checkbox :label="3">Wed</el-checkbox>
                    <el-checkbox :label="4">Thu</el-checkbox>
                    <el-checkbox :label="5">Fri</el-checkbox>
                    <el-checkbox :label="6">Sat</el-checkbox>
                  </el-checkbox-group>
                </div>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  InfoFilled,
  Setting,
  Clock,
  Monitor,
  Tools
} from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import ConfigForm from './forms/ConfigForm.vue'
import type { GeneralSettings } from '@/store/settings'

interface Props {
  modelValue: GeneralSettings
  loading?: boolean
  validationErrors?: Record<string, string[]>
}

interface Emits {
  (e: 'update:modelValue', value: GeneralSettings): void
  (e: 'field-change', field: string, value: any): void
  (e: 'field-validate', field: string, value: any): void
  (e: 'test-configuration', config: GeneralSettings): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const formData = ref<GeneralSettings>({
  systemName: '',
  systemDescription: '',
  timezone: 'UTC',
  language: 'en',
  dateFormat: 'YYYY-MM-DD',
  timeFormat: '24',
  sessionTimeout: 30,
  autoLogoutWarning: true,
  maxConcurrentOperations: 10,
  requestTimeout: 30,
  cacheTtl: 60,
  resourceLimits: {
    cpu: 80,
    memory: 80
  },
  logRetention: 30,
  autoCleanup: true,
  autoUpdates: false,
  maintenanceWindow: {
    start: '02:00',
    end: '04:00',
    days: [0] // Sunday
  }
} as any)

const systemInfo = ref({
  version: '1.0.0',
  buildDate: new Date().toISOString(),
  runtime: 'Node.js 18.17.0',
  platform: 'Linux x64',
  installPath: '/opt/docker-auto'
})

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue)
})

const timezones = ref([
  { label: 'UTC', value: 'UTC' },
  { label: 'America/New_York (EST/EDT)', value: 'America/New_York' },
  { label: 'America/Chicago (CST/CDT)', value: 'America/Chicago' },
  { label: 'America/Denver (MST/MDT)', value: 'America/Denver' },
  { label: 'America/Los_Angeles (PST/PDT)', value: 'America/Los_Angeles' },
  { label: 'Europe/London (GMT/BST)', value: 'Europe/London' },
  { label: 'Europe/Paris (CET/CEST)', value: 'Europe/Paris' },
  { label: 'Asia/Tokyo (JST)', value: 'Asia/Tokyo' },
  { label: 'Asia/Shanghai (CST)', value: 'Asia/Shanghai' },
  { label: 'Australia/Sydney (AEST/AEDT)', value: 'Australia/Sydney' }
])

const languages = ref([
  { label: 'English', value: 'en', flag: '🇺🇸' },
  { label: 'Español', value: 'es', flag: '🇪🇸' },
  { label: 'Français', value: 'fr', flag: '🇫🇷' },
  { label: 'Deutsch', value: 'de', flag: '🇩🇪' },
  { label: '中文', value: 'zh', flag: '🇨🇳' },
  { label: '日本語', value: 'ja', flag: '🇯🇵' }
])

const dateFormats = ref([
  { label: 'YYYY-MM-DD', value: 'YYYY-MM-DD', example: '2024-01-15' },
  { label: 'MM/DD/YYYY', value: 'MM/DD/YYYY', example: '01/15/2024' },
  { label: 'DD/MM/YYYY', value: 'DD/MM/YYYY', example: '15/01/2024' },
  { label: 'DD.MM.YYYY', value: 'DD.MM.YYYY', example: '15.01.2024' },
  { label: 'MMM DD, YYYY', value: 'MMM DD, YYYY', example: 'Jan 15, 2024' }
])

const timeFormats = ref([
  { label: '24-hour', value: '24', example: '14:30' },
  { label: '12-hour', value: '12', example: '2:30 PM' }
])

const formRules = computed(() => ({
  systemName: [
    { required: true, message: 'System name is required', trigger: 'blur' },
    { min: 3, max: 100, message: 'Length should be 3 to 100 characters', trigger: 'blur' }
  ],
  timezone: [
    { required: true, message: 'Timezone is required', trigger: 'change' }
  ],
  language: [
    { required: true, message: 'Language is required', trigger: 'change' }
  ],
  dateFormat: [
    { required: true, message: 'Date format is required', trigger: 'change' }
  ],
  timeFormat: [
    { required: true, message: 'Time format is required', trigger: 'change' }
  ],
  sessionTimeout: [
    { required: true, message: 'Session timeout is required', trigger: 'blur' },
    { type: 'number', min: 5, max: 1440, message: 'Must be between 5 and 1440 minutes', trigger: 'blur' }
  ],
  maxConcurrentOperations: [
    { required: true, message: 'Max concurrent operations is required', trigger: 'blur' },
    { type: 'number', min: 1, max: 100, message: 'Must be between 1 and 100', trigger: 'blur' }
  ],
  requestTimeout: [
    { required: true, message: 'Request timeout is required', trigger: 'blur' },
    { type: 'number', min: 5, max: 300, message: 'Must be between 5 and 300 seconds', trigger: 'blur' }
  ],
  cacheTtl: [
    { required: true, message: 'Cache TTL is required', trigger: 'blur' },
    { type: 'number', min: 1, max: 1440, message: 'Must be between 1 and 1440 minutes', trigger: 'blur' }
  ],
  logRetention: [
    { required: true, message: 'Log retention is required', trigger: 'blur' },
    { type: 'number', min: 1, max: 365, message: 'Must be between 1 and 365 days', trigger: 'blur' }
  ]
}))

const formatDate = (dateString: string): string => {
  return dayjs(dateString).format('YYYY-MM-DD HH:mm:ss')
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

const handleResourceLimitChange = () => {
  emit('field-change', 'resourceLimits', formData.value.resourceLimits)
}

const handleMaintenanceWindowChange = () => {
  emit('field-change', 'maintenanceWindow', formData.value.maintenanceWindow)
}

// Initialize form data from props
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    formData.value = { ...newValue }
  }
}, { immediate: true, deep: true })

onMounted(() => {
  // Load system info
  // This would typically come from an API call
  console.log('SystemConfig mounted')
})
</script>

<style scoped lang="scss">
.system-config {
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

  .readonly-input {
    :deep(.el-input__wrapper) {
      background: var(--el-fill-color-extra-light);
      cursor: not-allowed;
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

  .language-option {
    display: flex;
    align-items: center;
    gap: 8px;

    .language-flag {
      font-size: 16px;
    }
  }

  .format-option {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;

    .format-example {
      color: var(--el-text-color-regular);
      font-size: 12px;
    }
  }

  .resource-limits {
    .limit-item {
      margin-bottom: 16px;

      label {
        display: block;
        font-size: 12px;
        font-weight: 500;
        color: var(--el-text-color-primary);
        margin-bottom: 8px;
      }

      &:last-child {
        margin-bottom: 0;
      }
    }
  }

  .maintenance-window {
    display: flex;
    flex-direction: column;
    gap: 16px;

    .window-item {
      label {
        display: block;
        font-size: 12px;
        font-weight: 500;
        color: var(--el-text-color-primary);
        margin-bottom: 8px;
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

@media (max-width: 768px) {
  .system-config {
    .config-section {
      :deep(.el-card__body) {
        padding: 16px;
      }
    }

    .maintenance-window {
      .window-item {
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
</style>