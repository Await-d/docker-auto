<template>
  <div class="user-management">
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
      <!-- Password Policy -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Lock /></el-icon>
            <span>Password Policy</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Minimum Length" prop="passwordPolicy.minLength" required>
              <el-input-number
                v-model="formData.passwordPolicy.minLength"
                :min="6"
                :max="50"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Maximum Age (days)" prop="passwordPolicy.maxAge">
              <el-input-number
                v-model="formData.passwordPolicy.maxAge"
                :min="0"
                :max="365"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="6">
            <el-form-item label="Require Uppercase">
              <el-switch
                v-model="formData.passwordPolicy.requireUppercase"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="Require Lowercase">
              <el-switch
                v-model="formData.passwordPolicy.requireLowercase"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="Require Numbers">
              <el-switch
                v-model="formData.passwordPolicy.requireNumbers"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="Require Special Characters">
              <el-switch
                v-model="formData.passwordPolicy.requireSpecialChars"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Session Policy -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Clock /></el-icon>
            <span>Session Policy</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="JWT Expiration" prop="jwtExpiration" required>
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.jwtExpiration"
                  :min="15"
                  :max="1440"
                  :step="15"
                  @change="handleFieldChange('jwtExpiration', $event)"
                />
                <span class="timeout-unit">minutes</span>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max Concurrent Sessions" prop="sessionPolicy.maxConcurrentSessions">
              <el-input-number
                v-model="formData.sessionPolicy.maxConcurrentSessions"
                :min="1"
                :max="10"
                @change="updateSessionPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Two-Factor Authentication">
              <el-switch
                v-model="formData.twoFactorEnabled"
                @change="handleFieldChange('twoFactorEnabled', $event)"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Account Lockout -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Shield /></el-icon>
            <span>Account Lockout</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="Enable Account Lockout">
              <el-switch
                v-model="formData.accountLockoutEnabled"
                @change="handleFieldChange('accountLockoutEnabled', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Max Login Attempts" prop="maxLoginAttempts">
              <el-input-number
                v-model="formData.maxLoginAttempts"
                :min="3"
                :max="10"
                :disabled="!formData.accountLockoutEnabled"
                @change="handleFieldChange('maxLoginAttempts', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Lockout Duration" prop="lockoutDuration">
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.lockoutDuration"
                  :min="5"
                  :max="1440"
                  :step="5"
                  :disabled="!formData.accountLockoutEnabled"
                  @change="handleFieldChange('lockoutDuration', $event)"
                />
                <span class="timeout-unit">minutes</span>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Lock, Clock, Shield } from '@element-plus/icons-vue'
import ConfigForm from './forms/ConfigForm.vue'
import type { UserSettings } from '@/store/settings'

interface Props {
  modelValue: UserSettings
  loading?: boolean
  validationErrors?: Record<string, string[]>
}

interface Emits {
  (e: 'update:modelValue', value: UserSettings): void
  (e: 'field-change', field: string, value: any): void
  (e: 'field-validate', field: string, value: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const formData = ref<UserSettings>({
  passwordPolicy: {
    minLength: 8,
    requireUppercase: true,
    requireLowercase: true,
    requireNumbers: true,
    requireSpecialChars: true,
    maxAge: 90,
    preventReuse: 5
  },
  sessionPolicy: {
    maxConcurrentSessions: 3,
    idleTimeout: 30,
    absoluteTimeout: 480,
    requireReauth: false
  },
  roles: [],
  defaultRole: 'user',
  jwtExpiration: 60,
  refreshTokenExpiration: 10080,
  twoFactorEnabled: false,
  accountLockoutEnabled: true,
  maxLoginAttempts: 5,
  lockoutDuration: 30
} as any)

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue)
})

const formRules = computed(() => ({
  'passwordPolicy.minLength': [
    { required: true, message: 'Minimum length is required', trigger: 'blur' },
    { type: 'number', min: 6, max: 50, message: 'Must be between 6 and 50', trigger: 'blur' }
  ],
  jwtExpiration: [
    { required: true, message: 'JWT expiration is required', trigger: 'blur' },
    { type: 'number', min: 15, max: 1440, message: 'Must be between 15 and 1440 minutes', trigger: 'blur' }
  ]
}))

const updatePasswordPolicy = () => {
  handleFieldChange('passwordPolicy', formData.value.passwordPolicy)
}

const updateSessionPolicy = () => {
  handleFieldChange('sessionPolicy', formData.value.sessionPolicy)
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
.user-management {
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
}
</style>