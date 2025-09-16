<template>
  <div class="monitoring-config">
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
      <!-- System Monitoring -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Monitor /></el-icon>
            <span>System Monitoring</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Health Check Interval" prop="systemMonitoring.healthCheckInterval" required>
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.systemMonitoring.healthCheckInterval"
                  :min="30"
                  :max="3600"
                  :step="30"
                  @change="updateSystemMonitoring"
                />
                <span class="timeout-unit">seconds</span>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Alerting Enabled">
              <el-switch
                v-model="formData.systemMonitoring.alertingEnabled"
                @change="updateSystemMonitoring"
              />
              <div class="field-help">
                Send alerts when thresholds are exceeded
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="CPU Warning Threshold" prop="systemMonitoring.resourceThresholds.cpuWarning">
              <div class="percentage-input">
                <el-input-number
                  v-model="formData.systemMonitoring.resourceThresholds.cpuWarning"
                  :min="10"
                  :max="95"
                  :step="5"
                  @change="updateSystemMonitoring"
                />
                <span class="percentage-unit">%</span>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Memory Warning Threshold" prop="systemMonitoring.resourceThresholds.memoryWarning">
              <div class="percentage-input">
                <el-input-number
                  v-model="formData.systemMonitoring.resourceThresholds.memoryWarning"
                  :min="10"
                  :max="95"
                  :step="5"
                  @change="updateSystemMonitoring"
                />
                <span class="percentage-unit">%</span>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Logging Configuration -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Document /></el-icon>
            <span>Logging Configuration</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Log Level" prop="logging.level" required>
              <el-select
                v-model="formData.logging.level"
                placeholder="Select log level"
                @change="updateLogging"
              >
                <el-option label="Debug" value="debug" />
                <el-option label="Info" value="info" />
                <el-option label="Warning" value="warn" />
                <el-option label="Error" value="error" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Structured Logging">
              <el-switch
                v-model="formData.logging.structured"
                @change="updateLogging"
              />
              <div class="field-help">
                Use structured JSON logging format
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="Max File Size" prop="logging.rotation.maxSize">
              <div class="size-input">
                <el-input-number
                  v-model="formData.logging.rotation.maxSize"
                  :min="1"
                  :max="1000"
                  @change="updateLogging"
                />
                <span class="size-unit">MB</span>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max Files" prop="logging.rotation.maxFiles">
              <el-input-number
                v-model="formData.logging.rotation.maxFiles"
                :min="1"
                :max="50"
                @change="updateLogging"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max Age" prop="logging.rotation.maxAge">
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.logging.rotation.maxAge"
                  :min="1"
                  :max="365"
                  @change="updateLogging"
                />
                <span class="timeout-unit">days</span>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- External Integrations -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Connection /></el-icon>
            <span>External Integrations</span>
            <el-button type="primary" size="small" @click="addIntegration">
              <el-icon><Plus /></el-icon>
              Add Integration
            </el-button>
          </div>
        </template>

        <div v-if="formData.externalIntegrations.length === 0" class="empty-state">
          <el-empty description="No integrations configured">
            <el-button type="primary" @click="addIntegration">
              Add First Integration
            </el-button>
          </el-empty>
        </div>

        <div v-else class="integrations-list">
          <div
            v-for="(integration, index) in formData.externalIntegrations"
            :key="integration.id"
            class="integration-item"
          >
            <div class="integration-header">
              <el-input
                v-model="integration.name"
                placeholder="Integration name"
                class="integration-name"
                @input="updateIntegrations"
              />
              <el-select
                v-model="integration.type"
                placeholder="Select type"
                @change="updateIntegrations"
              >
                <el-option label="Prometheus" value="prometheus" />
                <el-option label="Grafana" value="grafana" />
                <el-option label="Elasticsearch" value="elasticsearch" />
                <el-option label="SNMP" value="snmp" />
              </el-select>
              <el-switch
                v-model="integration.enabled"
                @change="updateIntegrations"
              />
              <el-button
                type="text"
                size="small"
                @click="removeIntegration(index)"
                class="danger-button"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>
        </div>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Monitor,
  Document,
  Connection,
  Plus,
  Delete
} from '@element-plus/icons-vue'
import ConfigForm from './forms/ConfigForm.vue'
import type { MonitoringSettings, ExternalIntegration } from '@/store/settings'

interface Props {
  modelValue: MonitoringSettings
  loading?: boolean
  validationErrors?: Record<string, string[]>
}

interface Emits {
  (e: 'update:modelValue', value: MonitoringSettings): void
  (e: 'field-change', field: string, value: any): void
  (e: 'field-validate', field: string, value: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const formData = ref<MonitoringSettings>({
  systemMonitoring: {
    resourceThresholds: {
      cpuWarning: 80,
      cpuCritical: 90,
      memoryWarning: 80,
      memoryCritical: 90,
      diskWarning: 80,
      diskCritical: 90,
      networkWarning: 80,
      networkCritical: 90
    },
    healthCheckInterval: 60,
    alertingEnabled: true,
    metricsCollection: {
      enabled: true,
      interval: 30,
      retention: 30,
      aggregation: ['avg', 'max', 'min']
    }
  },
  logging: {
    level: 'info',
    components: {},
    rotation: {
      maxSize: 100,
      maxFiles: 10,
      maxAge: 30,
      compress: true
    },
    export: {
      enabled: false,
      destination: '',
      format: 'json',
      filters: []
    },
    structured: true
  },
  externalIntegrations: []
} as any)

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue)
})

const formRules = computed(() => ({
  'systemMonitoring.healthCheckInterval': [
    { required: true, message: 'Health check interval is required', trigger: 'blur' },
    { type: 'number', min: 30, max: 3600, message: 'Must be between 30 and 3600 seconds', trigger: 'blur' }
  ],
  'logging.level': [
    { required: true, message: 'Log level is required', trigger: 'change' }
  ]
}))

const generateId = (): string => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9)
}

const addIntegration = () => {
  const newIntegration: ExternalIntegration = {
    id: generateId(),
    type: 'prometheus',
    name: `Integration ${formData.value.externalIntegrations.length + 1}`,
    config: {},
    enabled: true
  }

  formData.value.externalIntegrations.push(newIntegration)
  updateIntegrations()
}

const removeIntegration = (index: number) => {
  formData.value.externalIntegrations.splice(index, 1)
  updateIntegrations()
}

const updateSystemMonitoring = () => {
  handleFieldChange('systemMonitoring', formData.value.systemMonitoring)
}

const updateLogging = () => {
  handleFieldChange('logging', formData.value.logging)
}

const updateIntegrations = () => {
  handleFieldChange('externalIntegrations', formData.value.externalIntegrations)
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

watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    formData.value = { ...newValue }
  }
}, { immediate: true, deep: true })
</script>

<style scoped lang="scss">
.monitoring-config {
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
        justify-content: space-between;
        gap: 8px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    :deep(.el-card__body) {
      padding: 24px;
    }
  }

  .timeout-input,
  .percentage-input,
  .size-input {
    display: flex;
    align-items: center;
    gap: 8px;

    .timeout-unit,
    .percentage-unit,
    .size-unit {
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

  .integrations-list {
    .integration-item {
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
      padding: 16px;
      margin-bottom: 16px;

      .integration-header {
        display: flex;
        align-items: center;
        gap: 12px;

        .integration-name {
          flex: 1;
        }

        .danger-button {
          color: var(--el-color-danger);
        }
      }
    }
  }
}

@media (max-width: 768px) {
  .monitoring-config {
    .config-section {
      :deep(.el-card__body) {
        padding: 16px;
      }

      :deep(.section-header) {
        flex-direction: column;
        gap: 12px;
      }
    }

    .integrations-list {
      .integration-item {
        .integration-header {
          flex-direction: column;
          align-items: stretch;
          gap: 8px;
        }
      }
    }
  }
}
</style>