<template>
  <div class="password-field">
    <el-input
      v-model="inputValue"
      :type="showPassword ? 'text' : 'password'"
      :placeholder="placeholder"
      :disabled="disabled"
      :size="size"
      :clearable="clearable"
      show-password
      class="password-input"
      @input="handleInput"
      @blur="handleBlur"
      @focus="handleFocus"
    >
      <template v-if="showStrengthMeter" #suffix>
        <div class="password-actions">
          <el-tooltip
            v-if="showGenerator"
            content="Generate password"
            placement="top"
          >
            <el-button
              type="text"
              size="small"
              :disabled="disabled"
              @click="generatePassword"
            >
              <el-icon><Key /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </template>
    </el-input>

    <!-- Password Strength Meter -->
    <div v-if="showStrengthMeter && inputValue" class="password-strength">
      <div class="strength-meter">
        <div
          :class="['strength-bar', `strength-${strengthLevel}`]"
          :style="{ width: `${strengthPercentage}%` }"
        />
      </div>
      <div class="strength-info">
        <span :class="['strength-text', `strength-${strengthLevel}`]">
          {{ strengthText }}
        </span>
        <span class="strength-score">{{ strengthScore }}/4</span>
      </div>
    </div>

    <!-- Password Requirements -->
    <div v-if="showRequirements && focused" class="password-requirements">
      <div class="requirement-title">Password Requirements:</div>
      <div class="requirements-list">
        <div
          v-for="requirement in requirements"
          :key="requirement.key"
          :class="['requirement-item', { met: requirement.met }]"
        >
          <el-icon>
            <Check v-if="requirement.met" />
            <Close v-else />
          </el-icon>
          <span>{{ requirement.text }}</span>
        </div>
      </div>
    </div>

    <!-- Password Confirmation -->
    <el-input
      v-if="requireConfirmation"
      v-model="confirmValue"
      type="password"
      :placeholder="confirmPlaceholder"
      :disabled="disabled"
      :size="size"
      show-password
      class="password-confirm"
      :class="{ 'is-error': confirmValue && !passwordsMatch }"
      @input="handleConfirmInput"
    />

    <div
      v-if="requireConfirmation && confirmValue && !passwordsMatch"
      class="confirm-error"
    >
      Passwords do not match
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { Key, Check, Close } from "@element-plus/icons-vue";

interface PasswordPolicy {
  minLength: number;
  requireUppercase: boolean;
  requireLowercase: boolean;
  requireNumbers: boolean;
  requireSpecialChars: boolean;
}

interface Props {
  modelValue: string;
  placeholder?: string;
  confirmPlaceholder?: string;
  disabled?: boolean;
  size?: "large" | "default" | "small";
  clearable?: boolean;
  showStrengthMeter?: boolean;
  showRequirements?: boolean;
  showGenerator?: boolean;
  requireConfirmation?: boolean;
  policy?: PasswordPolicy;
}

interface Emits {
  (e: "update:modelValue", value: string): void;
  (e: "strength-change", score: number, level: string): void;
  (e: "confirmation-change", matches: boolean): void;
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: "Enter password",
  confirmPlaceholder: "Confirm password",
  disabled: false,
  size: "default",
  clearable: true,
  showStrengthMeter: true,
  showRequirements: true,
  showGenerator: true,
  requireConfirmation: false,
  policy: () => ({
    minLength: 8,
    requireUppercase: true,
    requireLowercase: true,
    requireNumbers: true,
    requireSpecialChars: true,
  }),
});

const emit = defineEmits<Emits>();

const inputValue = ref(props.modelValue);
const confirmValue = ref("");
const showPassword = ref(false);
const focused = ref(false);

const strengthScore = computed(() => {
  let score = 0;
  const password = inputValue.value;

  if (!password) return 0;

  // Length check
  if (password.length >= props.policy.minLength) score++;

  // Character type checks
  if (props.policy.requireUppercase && /[A-Z]/.test(password)) score++;
  if (props.policy.requireLowercase && /[a-z]/.test(password)) score++;
  if (props.policy.requireNumbers && /\d/.test(password)) score++;
  if (
    props.policy.requireSpecialChars &&
    /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(password)
  )
    score++;

  return Math.min(score, 4);
});

const strengthLevel = computed(() => {
  const score = strengthScore.value;
  if (score === 0) return "none";
  if (score === 1) return "weak";
  if (score === 2) return "fair";
  if (score === 3) return "good";
  return "strong";
});

const strengthText = computed(() => {
  const level = strengthLevel.value;
  const texts = {
    none: "No password",
    weak: "Weak",
    fair: "Fair",
    good: "Good",
    strong: "Strong",
  };
  return texts[level] || "";
});

const strengthPercentage = computed(() => {
  return (strengthScore.value / 4) * 100;
});

const requirements = computed(() =>
  [
    {
      key: "length",
      text: `At least ${props.policy.minLength} characters`,
      met: inputValue.value.length >= props.policy.minLength,
    },
    {
      key: "uppercase",
      text: "At least one uppercase letter",
      met: !props.policy.requireUppercase || /[A-Z]/.test(inputValue.value),
    },
    {
      key: "lowercase",
      text: "At least one lowercase letter",
      met: !props.policy.requireLowercase || /[a-z]/.test(inputValue.value),
    },
    {
      key: "numbers",
      text: "At least one number",
      met: !props.policy.requireNumbers || /\d/.test(inputValue.value),
    },
    {
      key: "special",
      text: "At least one special character",
      met:
        !props.policy.requireSpecialChars ||
        /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(inputValue.value),
    },
  ].filter(
    (req) =>
      req.key === "length" ||
      props.policy[
        `require${req.key.charAt(0).toUpperCase() + req.key.slice(1)}` as keyof PasswordPolicy
      ],
  ),
);

const passwordsMatch = computed(() => {
  if (!props.requireConfirmation) return true;
  return inputValue.value === confirmValue.value;
});

const handleInput = (value: string) => {
  inputValue.value = value;
  emit("update:modelValue", value);
};

const handleConfirmInput = (value: string) => {
  confirmValue.value = value;
  emit("confirmation-change", inputValue.value === value);
};

const handleFocus = () => {
  focused.value = true;
};

const handleBlur = () => {
  setTimeout(() => {
    focused.value = false;
  }, 150);
};

const generatePassword = () => {
  const charset = {
    lowercase: "abcdefghijklmnopqrstuvwxyz",
    uppercase: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
    numbers: "0123456789",
    special: "!@#$%^&*()_+-=[]{}|;:,.<>?",
  };

  let password = "";
  let availableChars = "";

  // Ensure required character types are included
  if (props.policy.requireLowercase) {
    password += charset.lowercase.charAt(
      Math.floor(Math.random() * charset.lowercase.length),
    );
    availableChars += charset.lowercase;
  }
  if (props.policy.requireUppercase) {
    password += charset.uppercase.charAt(
      Math.floor(Math.random() * charset.uppercase.length),
    );
    availableChars += charset.uppercase;
  }
  if (props.policy.requireNumbers) {
    password += charset.numbers.charAt(
      Math.floor(Math.random() * charset.numbers.length),
    );
    availableChars += charset.numbers;
  }
  if (props.policy.requireSpecialChars) {
    password += charset.special.charAt(
      Math.floor(Math.random() * charset.special.length),
    );
    availableChars += charset.special;
  }

  // If no specific requirements, use all character types
  if (!availableChars) {
    availableChars = Object.values(charset).join("");
  }

  // Fill remaining length with random characters
  const remainingLength = Math.max(props.policy.minLength - password.length, 0);
  for (let i = 0; i < remainingLength; i++) {
    password += availableChars.charAt(
      Math.floor(Math.random() * availableChars.length),
    );
  }

  // Shuffle the password
  password = password
    .split("")
    .sort(() => Math.random() - 0.5)
    .join("");

  inputValue.value = password;
  emit("update:modelValue", password);
};

watch(
  () => props.modelValue,
  (newValue) => {
    inputValue.value = newValue;
  },
);

watch([strengthScore, strengthLevel], () => {
  emit("strength-change", strengthScore.value, strengthLevel.value);
});

watch(passwordsMatch, (matches) => {
  if (props.requireConfirmation) {
    emit("confirmation-change", matches);
  }
});
</script>

<style scoped lang="scss">
.password-field {
  .password-input {
    margin-bottom: 8px;
  }

  .password-actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .password-strength {
    margin-bottom: 12px;

    .strength-meter {
      height: 4px;
      background: var(--el-border-color-lighter);
      border-radius: 2px;
      overflow: hidden;
      margin-bottom: 4px;

      .strength-bar {
        height: 100%;
        transition: all 0.3s;
        border-radius: 2px;

        &.strength-weak {
          background: var(--el-color-danger);
        }

        &.strength-fair {
          background: var(--el-color-warning);
        }

        &.strength-good {
          background: var(--el-color-primary);
        }

        &.strength-strong {
          background: var(--el-color-success);
        }
      }
    }

    .strength-info {
      display: flex;
      justify-content: space-between;
      align-items: center;

      .strength-text {
        font-size: 12px;
        font-weight: 500;

        &.strength-weak {
          color: var(--el-color-danger);
        }

        &.strength-fair {
          color: var(--el-color-warning);
        }

        &.strength-good {
          color: var(--el-color-primary);
        }

        &.strength-strong {
          color: var(--el-color-success);
        }
      }

      .strength-score {
        font-size: 12px;
        color: var(--el-text-color-regular);
      }
    }
  }

  .password-requirements {
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    padding: 12px;
    margin-bottom: 12px;

    .requirement-title {
      font-size: 12px;
      font-weight: 500;
      color: var(--el-text-color-primary);
      margin-bottom: 8px;
    }

    .requirements-list {
      display: flex;
      flex-direction: column;
      gap: 4px;
    }

    .requirement-item {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 12px;
      color: var(--el-text-color-regular);

      &.met {
        color: var(--el-color-success);
      }

      .el-icon {
        font-size: 12px;
      }
    }
  }

  .password-confirm {
    margin-top: 12px;

    &.is-error {
      :deep(.el-input__wrapper) {
        border-color: var(--el-color-danger);
      }
    }
  }

  .confirm-error {
    color: var(--el-color-danger);
    font-size: 12px;
    margin-top: 4px;
    line-height: 1.4;
  }
}
</style>
