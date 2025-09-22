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
            <span>密码策略</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="最小长度"
              prop="passwordPolicy.minLength"
              required
            >
              <el-input-number
                v-model="passwordMinLength"
                :min="6"
                :max="50"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="最大使用天数"
              prop="passwordPolicy.maxAge"
            >
              <el-input-number
                v-model="passwordMaxAge"
                :min="0"
                :max="365"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="6">
            <el-form-item label="要求大写字母">
              <el-switch
                v-model="passwordRequireUppercase"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="要求小写字母">
              <el-switch
                v-model="passwordRequireLowercase"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="要求数字">
              <el-switch
                v-model="passwordRequireNumbers"
                @change="updatePasswordPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="要求特殊字符">
              <el-switch
                v-model="passwordRequireSpecialChars"
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
            <span>会话策略</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="JWT过期时间" prop="jwtExpiration" required>
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.jwtExpiration"
                  :min="15"
                  :max="1440"
                  :step="15"
                  @change="handleFieldChange('jwtExpiration', $event)"
                />
                <span class="timeout-unit">分钟</span>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item
              label="最大并发会话数"
              prop="sessionPolicy.maxConcurrentSessions"
            >
              <el-input-number
                v-model="sessionMaxConcurrent"
                :min="1"
                :max="10"
                @change="updateSessionPolicy"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="双因素认证">
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
            <el-icon><Lock /></el-icon>
            <span>账户锁定</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="启用账户锁定">
              <el-switch
                v-model="formData.accountLockoutEnabled"
                @change="handleFieldChange('accountLockoutEnabled', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="最大登录尝试次数" prop="maxLoginAttempts">
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
            <el-form-item label="锁定持续时间" prop="lockoutDuration">
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.lockoutDuration"
                  :min="5"
                  :max="1440"
                  :step="5"
                  :disabled="!formData.accountLockoutEnabled"
                  @change="handleFieldChange('lockoutDuration', $event)"
                />
                <span class="timeout-unit">分钟</span>
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
import { Lock, Clock } from "@element-plus/icons-vue";
import ConfigForm from "./forms/ConfigForm.vue";
import type { UserSettings } from "@/store/settings";

interface Props {
  modelValue: UserSettings;
  loading?: boolean;
  validationErrors?: Record<string, string[]>;
}

interface Emits {
  (e: "update:modelValue", value: UserSettings): void;
  (e: "field-change", field: string, value: any): void;
  (e: "field-validate", field: string, value: any): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const formData = ref<UserSettings>({
  passwordPolicy: {
    minLength: 8,
    requireUppercase: true,
    requireLowercase: true,
    requireNumbers: true,
    requireSpecialChars: true,
    maxAge: 90,
    preventReuse: 5,
  },
  sessionPolicy: {
    maxConcurrentSessions: 3,
    idleTimeout: 30,
    absoluteTimeout: 480,
    requireReauth: false,
  },
  roles: [],
  defaultRole: "user",
  jwtExpiration: 60,
  refreshTokenExpiration: 10080,
  twoFactorEnabled: false,
  accountLockoutEnabled: true,
  maxLoginAttempts: 5,
  lockoutDuration: 30,
} as any);

// Initialize form data from props
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      formData.value = {
        ...formData.value, // Keep existing defaults
        ...newValue, // Override with new values
        // Ensure required objects are always present
        passwordPolicy: {
          ...formData.value.passwordPolicy,
          ...(newValue.passwordPolicy || {}),
        },
        sessionPolicy: {
          ...formData.value.sessionPolicy,
          ...(newValue.sessionPolicy || {}),
        },
        roles: newValue.roles || [],
      };
    }
  },
  { immediate: true, deep: true },
);

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

// Computed properties for safe two-way binding
const passwordMinLength = computed({
  get: () => formData.value.passwordPolicy?.minLength || 8,
  set: (value) => {
    if (!formData.value.passwordPolicy) {
      formData.value.passwordPolicy = {
        minLength: 8,
        requireUppercase: true,
        requireLowercase: true,
        requireNumbers: true,
        requireSpecialChars: true,
        maxAge: 90,
        preventReuse: 5,
      };
    }
    formData.value.passwordPolicy.minLength = value;
  }
});

const passwordMaxAge = computed({
  get: () => formData.value.passwordPolicy?.maxAge || 90,
  set: (value) => {
    if (!formData.value.passwordPolicy) {
      formData.value.passwordPolicy = {
        minLength: 8,
        requireUppercase: true,
        requireLowercase: true,
        requireNumbers: true,
        requireSpecialChars: true,
        maxAge: 90,
        preventReuse: 5,
      };
    }
    formData.value.passwordPolicy.maxAge = value;
  }
});

const passwordRequireUppercase = computed({
  get: () => formData.value.passwordPolicy?.requireUppercase || false,
  set: (value) => {
    if (!formData.value.passwordPolicy) {
      formData.value.passwordPolicy = {
        minLength: 8,
        requireUppercase: true,
        requireLowercase: true,
        requireNumbers: true,
        requireSpecialChars: true,
        maxAge: 90,
        preventReuse: 5,
      };
    }
    formData.value.passwordPolicy.requireUppercase = value;
  }
});

const passwordRequireLowercase = computed({
  get: () => formData.value.passwordPolicy?.requireLowercase || false,
  set: (value) => {
    if (!formData.value.passwordPolicy) {
      formData.value.passwordPolicy = {
        minLength: 8,
        requireUppercase: true,
        requireLowercase: true,
        requireNumbers: true,
        requireSpecialChars: true,
        maxAge: 90,
        preventReuse: 5,
      };
    }
    formData.value.passwordPolicy.requireLowercase = value;
  }
});

const passwordRequireNumbers = computed({
  get: () => formData.value.passwordPolicy?.requireNumbers || false,
  set: (value) => {
    if (!formData.value.passwordPolicy) {
      formData.value.passwordPolicy = {
        minLength: 8,
        requireUppercase: true,
        requireLowercase: true,
        requireNumbers: true,
        requireSpecialChars: true,
        maxAge: 90,
        preventReuse: 5,
      };
    }
    formData.value.passwordPolicy.requireNumbers = value;
  }
});

const passwordRequireSpecialChars = computed({
  get: () => formData.value.passwordPolicy?.requireSpecialChars || false,
  set: (value) => {
    if (!formData.value.passwordPolicy) {
      formData.value.passwordPolicy = {
        minLength: 8,
        requireUppercase: true,
        requireLowercase: true,
        requireNumbers: true,
        requireSpecialChars: true,
        maxAge: 90,
        preventReuse: 5,
      };
    }
    formData.value.passwordPolicy.requireSpecialChars = value;
  }
});

const sessionMaxConcurrent = computed({
  get: () => formData.value.sessionPolicy?.maxConcurrentSessions || 3,
  set: (value) => {
    if (!formData.value.sessionPolicy) {
      formData.value.sessionPolicy = {
        maxConcurrentSessions: 3,
        idleTimeout: 30,
        absoluteTimeout: 480,
        requireReauth: false,
      };
    }
    formData.value.sessionPolicy.maxConcurrentSessions = value;
  }
});

const formRules = computed(() => ({
  "passwordPolicy.minLength": [
    { required: true, message: "最小长度为必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 6 || value > 50) {
          callback(new Error("必须在6咁50之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  jwtExpiration: [
    { required: true, message: "JWT过期时间为必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 15 || value > 1440) {
          callback(new Error("必须在15到1440分钟之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
}));

const updatePasswordPolicy = () => {
  handleFieldChange("passwordPolicy", formData.value.passwordPolicy);
};

const updateSessionPolicy = () => {
  handleFieldChange("sessionPolicy", formData.value.sessionPolicy);
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
