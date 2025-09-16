<template>
  <div class="notification-config">
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
      <!-- Notification Channels -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Bell /></el-icon>
            <span>Notification Channels</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Email Notifications">
              <el-switch v-model="emailEnabled" @change="toggleEmailChannel" />
              <div class="field-help">Send notifications via email</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Webhook Notifications">
              <el-switch v-model="webhookEnabled" @change="toggleWebhookChannel" />
              <div class="field-help">Send notifications via webhooks</div>
            </el-form-item>
          </el-col>
        </el-row>

        <div v-if="emailEnabled" class="channel-config">
          <h4>Email Configuration</h4>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="SMTP Server">
                <el-input v-model="emailConfig.smtpServer" placeholder="smtp.example.com" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="Port">
                <el-input-number v-model="emailConfig.port" :min="25" :max="65535" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="Use TLS">
                <el-switch v-model="emailConfig.useTLS" />
              </el-form-item>
            </el-col>
          </el-row>
        </div>

        <div v-if="webhookEnabled" class="channel-config">
          <h4>Webhook Configuration</h4>
          <el-row :gutter="16">
            <el-col :span="24">
              <el-form-item label="Webhook URL">
                <el-input v-model="webhookConfig.url" placeholder="https://api.example.com/webhook" />
              </el-form-item>
            </el-col>
          </el-row>
        </div>
      </el-card>

      <!-- Rate Limiting -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Timer /></el-icon>
            <span>Rate Limiting</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="Enable Rate Limiting">
              <el-switch
                v-model="formData.rateLimiting.enabled"
                @change="updateRateLimiting"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max per Minute">
              <el-input-number
                v-model="formData.rateLimiting.maxPerMinute"
                :min="1"
                :max="100"
                :disabled="!formData.rateLimiting.enabled"
                @change="updateRateLimiting"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max per Hour">
              <el-input-number
                v-model="formData.rateLimiting.maxPerHour"
                :min="10"
                :max="1000"
                :disabled="!formData.rateLimiting.enabled"
                @change="updateRateLimiting"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Quiet Hours -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><MoonNight /></el-icon>
            <span>Quiet Hours</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="Enable Quiet Hours">
              <el-switch
                v-model="formData.quietHours.enabled"
                @change="updateQuietHours"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Start Time">
              <el-time-picker
                v-model="formData.quietHours.startTime"
                format="HH:mm"
                value-format="HH:mm"
                :disabled="!formData.quietHours.enabled"
                @change="updateQuietHours"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="End Time">
              <el-time-picker
                v-model="formData.quietHours.endTime"
                format="HH:mm"
                value-format="HH:mm"
                :disabled="!formData.quietHours.enabled"
                @change="updateQuietHours"
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
import { Bell, Timer, MoonNight } from '@element-plus/icons-vue'
import ConfigForm from './forms/ConfigForm.vue'
import type { NotificationSettings } from '@/store/settings'

interface Props {
  modelValue: NotificationSettings
  loading?: boolean
  validationErrors?: Record<string, string[]>
}

interface Emits {
  (e: 'update:modelValue', value: NotificationSettings): void
  (e: 'field-change', field: string, value: any): void
  (e: 'field-validate', field: string, value: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const formData = ref<NotificationSettings>({
  channels: [],
  rules: [],
  templates: [],
  rateLimiting: {
    enabled: true,
    maxPerMinute: 10,
    maxPerHour: 100,
    maxPerDay: 1000
  },
  quietHours: {
    enabled: false,
    startTime: '22:00',
    endTime: '08:00',
    timezone: 'UTC',
    days: [0, 1, 2, 3, 4, 5, 6]
  }
} as any)

const emailEnabled = ref(false)
const webhookEnabled = ref(false)
const emailConfig = ref({
  smtpServer: '',
  port: 587,
  useTLS: true
})
const webhookConfig = ref({
  url: ''
})

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue)
})

const formRules = computed(() => ({}))

const toggleEmailChannel = (enabled: boolean) => {
  // Implementation for toggling email channel
}

const toggleWebhookChannel = (enabled: boolean) => {
  // Implementation for toggling webhook channel
}

const updateRateLimiting = () => {
  handleFieldChange('rateLimiting', formData.value.rateLimiting)
}

const updateQuietHours = () => {
  handleFieldChange('quietHours', formData.value.quietHours)
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
.notification-config {
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

  .field-help {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-top: 4px;
    line-height: 1.4;
  }

  .channel-config {
    margin-top: 16px;
    padding: 16px;
    background: var(--el-fill-color-extra-light);
    border-radius: 8px;

    h4 {
      margin: 0 0 12px 0;
      font-size: 14px;
      color: var(--el-text-color-primary);
    }
  }
}
</style>