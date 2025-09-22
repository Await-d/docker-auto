<template>
  <div class="error-boundary">
    <div v-if="hasError" class="error-container">
      <div class="error-content">
        <div class="error-icon">
          <el-icon :size="64" color="#f56c6c">
            <WarningFilled />
          </el-icon>
        </div>

        <h2 class="error-title">{{ errorTitle }}</h2>
        <p class="error-message">{{ errorMessage }}</p>

        <div v-if="showDetails && errorDetails" class="error-details">
          <el-collapse>
            <el-collapse-item title="错误详情">
              <pre class="error-stack">{{ errorDetails }}</pre>
            </el-collapse-item>
          </el-collapse>
        </div>

        <div class="error-actions">
          <el-button type="primary" @click="handleRetry" :loading="retrying">
            <el-icon><Refresh /></el-icon>
            重试
          </el-button>
          <el-button @click="handleReload">
            <el-icon><RefreshRight /></el-icon>
            刷新页面
          </el-button>
          <el-button
            v-if="canGoBack"
            @click="handleGoBack"
            type="text"
          >
            <el-icon><ArrowLeft /></el-icon>
            返回上级
          </el-button>
        </div>

        <div class="error-meta">
          <p class="error-time">错误时间: {{ errorTime }}</p>
          <p class="error-id">错误ID: {{ errorId }}</p>
        </div>
      </div>
    </div>

    <slot v-else />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onErrorCaptured } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import {
  WarningFilled,
  Refresh,
  RefreshRight,
  ArrowLeft,
} from "@element-plus/icons-vue";

interface Props {
  fallbackTitle?: string;
  fallbackMessage?: string;
  showDetails?: boolean;
  canGoBack?: boolean;
  onRetry?: () => Promise<void> | void;
}

interface Emits {
  (e: "error", error: Error): void;
  (e: "retry"): void;
}

const props = withDefaults(defineProps<Props>(), {
  fallbackTitle: "出现了一些问题",
  fallbackMessage: "页面加载时发生错误，请稍后重试。",
  showDetails: false,
  canGoBack: true,
});

const emit = defineEmits<Emits>();

const router = useRouter();

// State
const hasError = ref(false);
const error = ref<Error | null>(null);
const retrying = ref(false);
const errorId = ref("");
const errorTime = ref("");

// Computed
const errorTitle = computed(() => {
  if (error.value?.name === "NetworkError") {
    return "网络连接错误";
  }
  if (error.value?.name === "TypeError") {
    return "数据处理错误";
  }
  if (error.value?.name === "AuthenticationError") {
    return "身份验证失败";
  }
  return props.fallbackTitle;
});

const errorMessage = computed(() => {
  if (error.value?.message) {
    // Provide user-friendly messages for common errors
    if (error.value.message.includes("Network Error")) {
      return "无法连接到服务器，请检查网络连接。";
    }
    if (error.value.message.includes("Unauthorized")) {
      return "您的登录状态已过期，请重新登录。";
    }
    if (error.value.message.includes("Forbidden")) {
      return "您没有权限执行此操作。";
    }
    if (error.value.message.includes("Not Found")) {
      return "请求的资源不存在。";
    }
    return error.value.message;
  }
  return props.fallbackMessage;
});

const errorDetails = computed(() => {
  if (!error.value) return "";

  return JSON.stringify({
    name: error.value.name,
    message: error.value.message,
    stack: error.value.stack,
    timestamp: errorTime.value,
    userAgent: navigator.userAgent,
    url: window.location.href,
  }, null, 2);
});

// Error capture
onErrorCaptured((err: Error) => {
  captureError(err);
  return false; // Prevent the error from propagating further
});

// Methods
const captureError = (err: Error) => {
  console.error("ErrorBoundary caught error:", err);

  hasError.value = true;
  error.value = err;
  errorId.value = generateErrorId();
  errorTime.value = new Date().toLocaleString();

  emit("error", err);

  // Report error to monitoring service (if available)
  reportError(err);
};

const generateErrorId = (): string => {
  return `ERR_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

const reportError = async (err: Error) => {
  try {
    // This would typically send the error to a monitoring service
    // For now, we'll just log it
    console.log("Error reported:", {
      id: errorId.value,
      name: err.name,
      message: err.message,
      stack: err.stack,
      timestamp: errorTime.value,
      url: window.location.href,
      userAgent: navigator.userAgent,
    });
  } catch (reportError) {
    console.warn("Failed to report error:", reportError);
  }
};

const handleRetry = async () => {
  retrying.value = true;

  try {
    if (props.onRetry) {
      await props.onRetry();
    }

    // Reset error state
    hasError.value = false;
    error.value = null;
    errorId.value = "";
    errorTime.value = "";

    emit("retry");
    ElMessage.success("重试成功");
  } catch (retryError) {
    console.error("Retry failed:", retryError);
    ElMessage.error("重试失败，请稍后再试");

    // Update error with retry failure
    if (retryError instanceof Error) {
      captureError(retryError);
    }
  } finally {
    retrying.value = false;
  }
};

const handleReload = () => {
  window.location.reload();
};

const handleGoBack = () => {
  if (window.history.length > 1) {
    router.go(-1);
  } else {
    router.push("/");
  }
};

// Global error handler
window.addEventListener("error", (event) => {
  captureError(new Error(event.message));
});

window.addEventListener("unhandledrejection", (event) => {
  captureError(new Error(event.reason));
});

// Expose methods for manual error handling
defineExpose({
  captureError,
  reset: () => {
    hasError.value = false;
    error.value = null;
    errorId.value = "";
    errorTime.value = "";
  },
});
</script>

<style scoped lang="scss">
.error-boundary {
  height: 100%;
  width: 100%;
}

.error-container {
  min-height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
}

.error-content {
  text-align: center;
  max-width: 600px;
  width: 100%;
}

.error-icon {
  margin-bottom: 24px;
}

.error-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 16px;
}

.error-message {
  font-size: 16px;
  color: var(--el-text-color-regular);
  margin-bottom: 32px;
  line-height: 1.6;
}

.error-details {
  margin-bottom: 32px;
  text-align: left;
}

.error-stack {
  background: var(--el-fill-color-light);
  padding: 16px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.4;
  max-height: 300px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
  margin-bottom: 32px;
}

.error-meta {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  text-align: center;
}

.error-time,
.error-id {
  margin: 4px 0;
}

@media (max-width: 768px) {
  .error-container {
    padding: 20px 16px;
  }

  .error-title {
    font-size: 20px;
  }

  .error-message {
    font-size: 14px;
  }

  .error-actions {
    flex-direction: column;
    align-items: center;
  }

  .error-actions .el-button {
    width: 200px;
  }
}
</style>