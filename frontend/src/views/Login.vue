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
        <h1 class="title">Docker Auto</h1>
        <p class="subtitle">Container Update Management System</p>
      </div>

      <!-- Login form -->
      <el-card class="login-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <h2>Sign In</h2>
            <p>Enter your credentials to access the system</p>
          </div>
        </template>

        <el-form
          ref="loginFormRef"
          :model="loginForm"
          :rules="loginRules"
          label-position="top"
          size="large"
          @submit.prevent="handleLogin"
        >
          <el-form-item label="Username" prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="Enter your username"
              :prefix-icon="User"
              clearable
              autocomplete="username"
              @keyup.enter="focusPassword"
            />
          </el-form-item>

          <el-form-item label="Password" prop="password">
            <el-input
              ref="passwordInputRef"
              v-model="loginForm.password"
              type="password"
              placeholder="Enter your password"
              :prefix-icon="Lock"
              :show-password="true"
              clearable
              autocomplete="current-password"
              @keyup.enter="handleLogin"
            />
          </el-form-item>

          <el-form-item>
            <div class="form-options">
              <el-checkbox v-model="loginForm.remember">
                Remember me
              </el-checkbox>
              <el-link type="primary" @click="showForgotPassword">
                Forgot password?
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
              <span v-if="!isLoading">Sign In</span>
              <span v-else>Signing In...</span>
            </el-button>
          </el-form-item>
        </el-form>

        <!-- Alternative login methods -->
        <div class="login-footer">
          <el-divider>
            <span class="divider-text">Or</span>
          </el-divider>

          <div class="alternative-login">
            <el-tooltip content="Demo Admin Account" placement="top">
              <el-button size="small" @click="fillDemoCredentials('admin')">
                Demo Admin
              </el-button>
            </el-tooltip>
            <el-tooltip content="Demo Operator Account" placement="top">
              <el-button size="small" @click="fillDemoCredentials('operator')">
                Demo Operator
              </el-button>
            </el-tooltip>
            <el-tooltip content="Demo Viewer Account" placement="top">
              <el-button size="small" @click="fillDemoCredentials('viewer')">
                Demo Viewer
              </el-button>
            </el-tooltip>
          </div>

          <p class="register-link">
            Don't have an account?
            <el-link type="primary" @click="showRegister">
              Sign up here
            </el-link>
          </p>
        </div>
      </el-card>

      <!-- System status -->
      <div class="system-status">
        <el-alert
          v-if="systemStatus"
          :title="systemStatus.title"
          :description="systemStatus.description"
          :type="systemStatus.type"
          :closable="false"
          show-icon
        />
      </div>
    </div>

    <!-- Background decoration -->
    <div class="login-background">
      <div class="bg-pattern"></div>
      <div class="bg-gradient"></div>
    </div>

    <!-- Forgot password dialog -->
    <el-dialog
      v-model="forgotPasswordVisible"
      title="Reset Password"
      width="400px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="forgotFormRef"
        :model="forgotForm"
        :rules="forgotRules"
        label-position="top"
      >
        <el-form-item label="Email Address" prop="email">
          <el-input
            v-model="forgotForm.email"
            placeholder="Enter your email address"
            :prefix-icon="Message"
            clearable
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="forgotPasswordVisible = false">
          Cancel
        </el-button>
        <el-button
          type="primary"
          :loading="isForgotLoading"
          @click="handleForgotPassword"
        >
          Send Reset Link
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import {
  Box,
  User,
  Lock,
  Message
} from '@element-plus/icons-vue'
import { ElMessage, ElNotification, type FormInstance, type FormRules } from 'element-plus'
import { useAuth } from '@/store/auth'
import { useApp } from '@/store/app'
import type { LoginForm } from '@/types/auth'

// Composables
const router = useRouter()
const { login, isLoading } = useAuth()
const { showError } = useApp()

// Form refs
const loginFormRef = ref<FormInstance>()
const passwordInputRef = ref()
const forgotFormRef = ref<FormInstance>()

// Form data
const loginForm = ref<LoginForm>({
  username: '',
  password: '',
  remember: false
})

const forgotForm = ref({
  email: ''
})

// UI state
const forgotPasswordVisible = ref(false)
const isForgotLoading = ref(false)
const systemStatus = ref<{
  title: string
  description: string
  type: 'success' | 'warning' | 'error' | 'info'
} | null>(null)

// Form validation rules
const loginRules = computed<FormRules>(() => ({
  username: [
    {
      required: true,
      message: 'Username is required',
      trigger: 'blur'
    },
    {
      min: 3,
      max: 50,
      message: 'Username must be between 3 and 50 characters',
      trigger: 'blur'
    },
    {
      pattern: /^[a-zA-Z0-9_.-]+$/,
      message: 'Username can only contain letters, numbers, dots, hyphens, and underscores',
      trigger: 'blur'
    }
  ],
  password: [
    {
      required: true,
      message: 'Password is required',
      trigger: 'blur'
    },
    {
      min: 6,
      message: 'Password must be at least 6 characters long',
      trigger: 'blur'
    }
  ]
}))

const forgotRules = computed<FormRules>(() => ({
  email: [
    {
      required: true,
      message: 'Email is required',
      trigger: 'blur'
    },
    {
      type: 'email',
      message: 'Please enter a valid email address',
      trigger: 'blur'
    }
  ]
}))

// Computed properties
const isFormValid = computed(() => {
  return loginForm.value.username.trim() !== '' &&
         loginForm.value.password.trim() !== '' &&
         !isLoading.value
})

// Methods
const handleLogin = async () => {
  if (!loginFormRef.value) return

  try {
    await loginFormRef.value.validate()
    await login(loginForm.value)

    ElMessage.success('Login successful!')

    // Navigation is handled by the auth store
  } catch (error: any) {
    console.error('Login error:', error)

    if (error.validation) {
      // Form validation errors are handled by Element Plus
      return
    }

    // Handle API errors
    const errorMessage = error.message || 'Login failed. Please check your credentials.'
    showError(errorMessage)

    // Clear password on error
    loginForm.value.password = ''

    // Focus password field
    nextTick(() => {
      passwordInputRef.value?.focus()
    })
  }
}

const focusPassword = () => {
  passwordInputRef.value?.focus()
}

const fillDemoCredentials = (role: 'admin' | 'operator' | 'viewer') => {
  const demoAccounts = {
    admin: { username: 'admin', password: 'admin123' },
    operator: { username: 'operator', password: 'operator123' },
    viewer: { username: 'viewer', password: 'viewer123' }
  }

  const account = demoAccounts[role]
  loginForm.value.username = account.username
  loginForm.value.password = account.password
  loginForm.value.remember = false

  ElNotification({
    title: 'Demo Account',
    message: `Demo ${role} credentials have been filled in. Click "Sign In" to continue.`,
    type: 'info',
    duration: 3000
  })
}

const showForgotPassword = () => {
  forgotPasswordVisible.value = true
  forgotForm.value.email = ''
}

const showRegister = () => {
  router.push('/register')
}

const handleForgotPassword = async () => {
  if (!forgotFormRef.value) return

  try {
    await forgotFormRef.value.validate()
    isForgotLoading.value = true

    // Simulate API call for password reset
    await new Promise(resolve => setTimeout(resolve, 2000))

    ElMessage.success('Password reset link has been sent to your email!')
    forgotPasswordVisible.value = false
  } catch (error: any) {
    if (error.validation) {
      return
    }

    showError('Failed to send reset link. Please try again.')
  } finally {
    isForgotLoading.value = false
  }
}

const checkSystemStatus = async () => {
  try {
    // Simulate system health check
    await new Promise(resolve => setTimeout(resolve, 1000))

    // You would normally call your health check API here
    // const response = await http.get('/system/health')

    systemStatus.value = {
      title: 'System Online',
      description: 'All services are running normally.',
      type: 'success'
    }
  } catch (error) {
    systemStatus.value = {
      title: 'System Unavailable',
      description: 'Some services may be temporarily unavailable.',
      type: 'warning'
    }
  }
}

// Lifecycle
onMounted(() => {
  // Auto-focus username field
  nextTick(() => {
    const usernameInput = document.querySelector('input[autocomplete="username"]') as HTMLInputElement
    usernameInput?.focus()
  })

  // Check system status
  checkSystemStatus()
})
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

  .alternative-login {
    display: flex;
    justify-content: center;
    gap: 8px;
    margin: 16px 0;
  }

  .register-link {
    text-align: center;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    margin: 16px 0 0;
  }
}

.system-status {
  margin-top: 24px;

  .el-alert {
    background: rgba(255, 255, 255, 0.9);
    border: 1px solid rgba(255, 255, 255, 0.2);
    backdrop-filter: blur(10px);

    .dark & {
      background: rgba(0, 0, 0, 0.7);
      border: 1px solid rgba(255, 255, 255, 0.1);
    }
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
      radial-gradient(circle at 25% 25%, var(--el-color-primary-light-7) 0%, transparent 50%),
      radial-gradient(circle at 75% 75%, var(--el-color-success-light-7) 0%, transparent 50%);
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

  .alternative-login {
    flex-direction: column;
    gap: 8px;

    .el-button {
      width: 100%;
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
  .login-background,
  .system-status,
  .alternative-login {
    display: none !important;
  }
}
</style>