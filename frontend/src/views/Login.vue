<template>
  <div class="login-page">
    <div class="login-container">
      <!-- Logo and title -->
      <div class="login-header">
        <div class="logo">
          <el-icon :size="48" color="var(--el-color-primary)">
            <Box />
          </el-icon>
        </div>
        <h1 class="title">Docker 自动更新</h1>
        <p class="subtitle">容器自动更新管理系统</p>
      </div>

      <!-- Login form -->
      <el-card class="login-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <h2>登录系统</h2>
            <p>请输入您的凭据以访问系统</p>
          </div>
        </template>

        <!-- Error display area -->
        <el-alert
          v-if="loginError"
          :title="loginError"
          type="error"
          :closable="true"
          show-icon
          @close="clearError"
          style="margin-bottom: 20px;"
        />

        <el-form
          ref="loginFormRef"
          :model="loginForm"
          :rules="loginRules"
          label-position="top"
          size="large"
          @submit.prevent="handleLogin"
        >
          <el-form-item label="用户名" prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              clearable
              autocomplete="username"
              @keyup.enter="focusPassword"
              @input="clearError"
            />
          </el-form-item>

          <el-form-item label="密码" prop="password">
            <el-input
              ref="passwordInputRef"
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              :show-password="true"
              clearable
              autocomplete="current-password"
              @keyup.enter="handleLogin"
              @input="clearError"
            />
          </el-form-item>

          <el-form-item>
            <div class="form-options">
              <el-checkbox v-model="loginForm.remember">
                记住我
              </el-checkbox>
              <el-link type="primary" @click="showForgotPassword">
                忘记密码？
              </el-link>
            </div>
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              class="login-button"
              :loading="isLoading"
              :disabled="!isFormValid"
              @click="handleLogin"
            >
              <span v-if="!isLoading">登录</span>
              <span v-else>登录中...</span>
            </el-button>
          </el-form-item>
        </el-form>

        <!-- Alternative login methods -->
        <div class="login-footer">
          <p class="register-link">
            没有账户？
            <el-link type="primary" @click="showRegister">
              点此注册
            </el-link>
          </p>
        </div>
      </el-card>

    </div>

    <!-- Background decoration -->
    <div class="login-background">
      <div class="bg-pattern" />
      <div class="bg-gradient" />
    </div>

    <!-- Forgot password dialog -->
    <el-dialog
      v-model="forgotPasswordVisible"
      title="重置密码"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="forgotFormRef"
        :model="forgotForm"
        :rules="forgotRules"
        label-position="top"
      >
        <el-form-item label="邮箱地址" prop="email">
          <el-input
            v-model="forgotForm.email"
            placeholder="请输入您的邮箱地址"
            :prefix-icon="Message"
            clearable
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="forgotPasswordVisible = false">
          取消
        </el-button>
        <el-button
          type="primary"
          :loading="isForgotLoading"
          @click="handleForgotPassword"
        >
          发送重置链接
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from "vue";
import { useRouter } from "vue-router";
import { Box, User, Lock, Message } from "@element-plus/icons-vue";
import {
  ElMessage,
  type FormInstance,
  type FormRules,
} from "element-plus";
import { useAuth } from "@/store/auth";
import { useApp } from "@/store/app";
import type { LoginForm } from "@/types/auth";

// Composables
const router = useRouter();
const { login, isLoading } = useAuth();
const { showError } = useApp();

// Form refs
const loginFormRef = ref<FormInstance>();
const passwordInputRef = ref();
const forgotFormRef = ref<FormInstance>();

// Form data
const loginForm = ref<LoginForm>({
  username: "",
  password: "",
  remember: false,
});

const forgotForm = ref({
  email: "",
});

// UI state
const forgotPasswordVisible = ref(false);
const isForgotLoading = ref(false);
const loginError = ref<string>("");

// Form validation rules
const loginRules = computed<FormRules>(() => ({
  username: [
    {
      required: true,
      message: "用户名不能为空",
      trigger: "blur",
    },
    {
      min: 2,
      max: 50,
      message: "用户名长度在 2 到 50 个字符之间",
      trigger: "blur",
    },
    {
      pattern: /^[a-zA-Z0-9_.-]+$/,
      message: "用户名只能包含字母、数字、点号、连字符和下划线",
      trigger: "blur",
    },
  ],
  password: [
    {
      required: true,
      message: "密码不能为空",
      trigger: "blur",
    },
    {
      min: 3,
      message: "密码至少需要 3 个字符",
      trigger: "blur",
    },
  ],
}));

const forgotRules = computed<FormRules>(() => ({
  email: [
    {
      required: true,
      message: "邮箱地址不能为空",
      trigger: "blur",
    },
    {
      type: "email",
      message: "请输入有效的邮箱地址",
      trigger: "blur",
    },
  ],
}));

// Computed properties
const isFormValid = computed(() => {
  return (
    loginForm.value.username.trim() !== "" &&
    loginForm.value.password.trim() !== "" &&
    !isLoading
  );
});

// Methods
const handleLogin = async () => {
  if (!loginFormRef.value) return;

  loginError.value = "";

  try {
    await loginFormRef.value.validate();
    await login(loginForm.value);
  } catch (error: any) {
    if (error.validation) {
      return;
    }
    const errorMessage = getErrorMessage(error);
    loginError.value = errorMessage;
    if (!loginError.value) {
      loginError.value = "登录失败：" + (error?.message || "请检查网络连接并重试");
    }
    loginForm.value.password = "";
    nextTick(() => {
      passwordInputRef.value?.focus();
    });
  }
};

const clearError = () => {
  loginError.value = "";
};

const getErrorMessage = (error: any): string => {
  // 直接返回错误消息，如果存在的话
  if (error?.message) {
    return error.message;
  }
  
  // Handle errors from request utils (ApiError format)
  if (error?.code !== undefined) {
    switch (error.code) {
      case 401:
        return "用户名或密码错误，请重试";
      case 403:
        return "账户已被禁用，请联系管理员";
      case 404:
        return "用户不存在，请检查用户名";
      case 429:
        return "登录尝试过于频繁，请稍后再试";
      case 500:
        return "服务器内部错误，请稍后再试";
      case 0:
        return "网络连接失败，请检查网络连接";
      default:
        return `登录失败 (错误代码: ${error.code})`;
    }
  }
  
  // Handle direct Axios response errors (fallback)
  if (error?.response) {
    const status = error.response.status;
    const message = error.response.data?.message || error.response.data?.error;
    
    if (message) {
      return message;
    }
    
    switch (status) {
      case 401:
        return "用户名或密码错误，请重试";
      case 403:
        return "账户已被禁用，请联系管理员";
      case 404:
        return "用户不存在，请检查用户名";
      case 429:
        return "登录尝试过于频繁，请稍后再试";
      case 500:
        return "服务器内部错误，请稍后再试";
      default:
        return `登录失败 (错误代码: ${status})`;
    }
  }
  
  // 最终回退
  return typeof error === 'string' ? error : "登录失败，请检查您的凭据";
};

const focusPassword = () => {
  passwordInputRef.value?.focus();
};


const showForgotPassword = () => {
  forgotPasswordVisible.value = true;
  forgotForm.value.email = "";
};

const showRegister = () => {
  router.push("/register");
};

const handleForgotPassword = async () => {
  if (!forgotFormRef.value) return;

  try {
    await forgotFormRef.value.validate();
    isForgotLoading.value = true;

    // Simulate API call for password reset
    await new Promise((resolve) => setTimeout(resolve, 2000));

    ElMessage.success("密码重置链接已发送到您的邮箱！");
    forgotPasswordVisible.value = false;
  } catch (error: any) {
    if (error.validation) {
      return;
    }

    showError("发送重置链接失败，请重试。");
  } finally {
    isForgotLoading.value = false;
  }
};


// Lifecycle
onMounted(() => {
  // Auto-focus username field
  nextTick(() => {
    const usernameInput = document.querySelector(
      'input[autocomplete="username"]',
    ) as HTMLInputElement;
    usernameInput?.focus();
  });
});
</script>

<style scoped lang="scss">
.login-page {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  overflow: hidden;
}

.login-container {
  position: relative;
  z-index: 2;
  width: 100%;
  max-width: 400px;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;

  .logo {
    margin-bottom: 16px;
  }

  .title {
    font-size: 28px;
    font-weight: 700;
    color: var(--el-text-color-primary);
    margin: 0 0 8px;
  }

  .subtitle {
    font-size: 14px;
    color: var(--el-text-color-secondary);
    margin: 0;
  }
}

.login-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);

  .dark & {
    background: rgba(0, 0, 0, 0.8);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .card-header {
    text-align: center;
    margin-bottom: 24px;

    h2 {
      font-size: 24px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin: 0 0 8px;
    }

    p {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      margin: 0;
    }
  }
}

.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  margin-bottom: 8px;
}

.login-button {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
}

.login-footer {
  margin-top: 24px;

  .divider-text {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    padding: 0 16px;
  }


  .register-link {
    text-align: center;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    margin: 16px 0 0;
  }
}


.login-background {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1;

  .bg-pattern {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-image:
      radial-gradient(
        circle at 25% 25%,
        var(--el-color-primary-light-7) 0%,
        transparent 50%
      ),
      radial-gradient(
        circle at 75% 75%,
        var(--el-color-success-light-7) 0%,
        transparent 50%
      );
    opacity: 0.3;
  }

  .bg-gradient {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: linear-gradient(
      135deg,
      var(--el-bg-color-page) 0%,
      var(--el-color-primary-light-9) 50%,
      var(--el-bg-color-page) 100%
    );
  }
}

// Responsive design
@media (max-width: 768px) {
  .login-page {
    padding: 12px;
  }

  .login-container {
    max-width: 100%;
  }

  .login-header {
    margin-bottom: 24px;

    .title {
      font-size: 24px;
    }
  }

}

@media (max-width: 480px) {
  .login-page {
    padding: 8px;
  }

  .login-header .title {
    font-size: 20px;
  }

  .card-header h2 {
    font-size: 20px;
  }
}

// Form animations
.el-form-item {
  margin-bottom: 20px;
}

.el-input {
  transition: all 0.3s ease;

  &:hover,
  &:focus-within {
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }
}

.login-button {
  transition: all 0.3s ease;

  &:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(var(--el-color-primary-rgb), 0.3);
  }

  &:active {
    transform: translateY(0);
  }
}

// Loading animation
.login-button.is-loading {
  .el-icon {
    animation: rotate 1s linear infinite;
  }
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

// Dark mode adjustments
.dark {
  .login-page {
    background: var(--el-bg-color-page);
  }

  .bg-pattern {
    opacity: 0.2;
  }
}

// Print styles
@media print {
  .login-background {
    display: none !important;
  }
}
</style>
