<template>
  <div id="app" :class="appClass">
    <!-- Global loading overlay -->
    <div v-if="isInitializing" class="app-initializing">
      <div class="initializing-content">
        <el-icon class="initializing-icon" :size="48">
          <Box />
        </el-icon>
        <h2 class="initializing-title">Docker Auto</h2>
        <p class="initializing-text">正在初始化应用程序...</p>
        <div class="initializing-spinner">
          <el-icon class="is-loading" :size="24">
            <Loading />
          </el-icon>
        </div>
      </div>
    </div>

    <!-- Main application -->
    <router-view v-else v-slot="{ Component, route }">
      <transition :name="getTransitionName(route)" mode="out-in">
        <component :is="Component" :key="route.fullPath" />
      </transition>
    </router-view>

    <!-- Global error boundary -->
    <div v-if="hasGlobalError" class="global-error-overlay">
      <div class="error-content">
        <el-icon class="error-icon" :size="64" color="var(--el-color-danger)">
          <WarningFilled />
        </el-icon>
        <h2 class="error-title">出现了错误</h2>
        <p class="error-message">
          {{ globalError }}
        </p>
        <div class="error-actions">
          <el-button type="primary" @click="reloadApp">
            重新加载应用
          </el-button>
          <el-button @click="clearError">
重试
</el-button>
        </div>
      </div>
    </div>

    <!-- Debug panel (development only) -->
    <div v-if="showDebugPanel" class="debug-panel">
      <div class="debug-header">
        <span>调试面板</span>
        <el-button size="small" text @click="toggleDebugPanel">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>
      <div class="debug-content">
        <div class="debug-item">
          <strong>环境:</strong> {{ env.MODE }}
        </div>
        <div class="debug-item">
<strong>路由:</strong> {{ currentRoute }}
</div>
        <div class="debug-item">
          <strong>用户:</strong> {{ user?.username || "未登录" }}
        </div>
        <div class="debug-item">
<strong>主题:</strong> {{ theme }}
</div>
        <div class="debug-item">
          <strong>屏幕尺寸:</strong> {{ screenSize }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ref,
  computed,
  onMounted,
  onUnmounted,
  onErrorCaptured,
  watch,
} from "vue";
import { useRoute } from "vue-router";
import { ElMessage } from "element-plus";
import { Box, Loading, WarningFilled, Close } from "@element-plus/icons-vue";
import { useAuthStore } from "@/store/auth";
import { useAppStore } from "@/store/app";

// Composables
const route = useRoute();
const authStore = useAuthStore();
const appStore = useAppStore();

// State
const isInitializing = ref(true);
const hasGlobalError = ref(false);
const globalError = ref("");
const showDebugPanel = ref(false);

// Environment info
const env = import.meta.env;

// Computed properties
const appClass = computed(() => ({
  "app-dark": appStore.isDarkMode,
  "app-mobile": appStore.isMobile,
  "app-sidebar-collapsed": appStore.sidebarCollapsed,
  "app-initializing": isInitializing.value,
  "app-error": hasGlobalError.value,
}));

const currentRoute = computed(() => route.fullPath);
const user = computed(() => authStore.user);
const theme = computed(() => appStore.theme);
const screenSize = computed(() => appStore.screenSize);

// Methods
const getTransitionName = (route: any) => {
  if (route.meta?.transition) {
    return route.meta.transition;
  }

  // Default transition based on route type
  if (route.path.startsWith("/login")) {
    return "slide-up";
  }

  return "fade";
};

const reloadApp = () => {
  window.location.reload();
};

const clearError = () => {
  hasGlobalError.value = false;
  globalError.value = "";
};

const toggleDebugPanel = () => {
  showDebugPanel.value = !showDebugPanel.value;
};

// Initialize application
const initializeApp = async () => {
  try {
    console.log("🚀 正在初始化Docker自动更新系统...");

    // Initialize app store
    appStore.initialize();

    // Initialize authentication
    await authStore.initialize();

    // Start token validation for authenticated users
    if (authStore.isAuthenticated) {
      authStore.startTokenCheck();
    }

    console.log("✅ 应用程序初始化成功");
  } catch (error: any) {
    console.error("❌ 应用程序初始化失败:", error);
    hasGlobalError.value = true;
    globalError.value = error.message || "应用程序初始化失败";
  } finally {
    // Add a small delay for better UX
    setTimeout(() => {
      isInitializing.value = false;
      // Mark app as fully initialized after UI transition
      setTimeout(() => {
        isAppInitialized.value = true;
      }, 500);
    }, 1000);
  }
};

// Global error handler
onErrorCaptured((error: any, _instance: any, info: string) => {
  console.error("捕获到全局错误:", error, info);

  hasGlobalError.value = true;
  globalError.value = error.message || "发生了意外错误";

  // Report error to monitoring service
  if (env.PROD) {
    // Add your error reporting service here
    // reportError(error, { instance, info })
  }

  return false; // Prevent the error from propagating further
});

// Keyboard shortcuts for development
const handleKeyboardShortcuts = (event: KeyboardEvent) => {
  if (env.DEV) {
    // Ctrl/Cmd + Shift + D to toggle debug panel
    if (
      (event.ctrlKey || event.metaKey) &&
      event.shiftKey &&
      event.key === "D"
    ) {
      event.preventDefault();
      toggleDebugPanel();
    }

    // Ctrl/Cmd + Shift + R to reload
    if (
      (event.ctrlKey || event.metaKey) &&
      event.shiftKey &&
      event.key === "R"
    ) {
      event.preventDefault();
      reloadApp();
    }
  }
};

// Handle app visibility changes
const isAppInitialized = ref(false);
const handleVisibilityChange = () => {
  // Skip visibility handling during initial app load
  if (!isAppInitialized.value) {
    return;
  }

  if (document.hidden) {
    // App became hidden - pause timers, etc.
    console.debug("应用程序已隐藏");
  } else {
    // App became visible - resume timers, check for updates, etc.
    console.debug("应用程序已显示");

    // Check authentication status when app becomes visible
    if (authStore.isAuthenticated) {
      authStore.checkTokenExpiration();
    }
  }
};

// Handle online/offline status
const handleOnline = () => {
  // Only show message if app is initialized and was previously offline
  if (isAppInitialized.value && !navigator.onLine) {
    ElMessage.success("连接已恢复");
    console.log("应用程序重新联机");
  }
};

const handleOffline = () => {
  // Only show message if app is initialized
  if (isAppInitialized.value) {
    ElMessage.warning("连接中断 - 离线工作");
    console.log("应用程序已离线");
  }
};

// Watch for auth state changes
watch(
  () => authStore.isAuthenticated,
  (isAuthenticated) => {
    if (isAuthenticated) {
      authStore.startTokenCheck();
    } else {
      authStore.stopTokenCheck();
    }
  },
);

// Lifecycle
onMounted(async () => {
  // Add event listeners
  document.addEventListener("keydown", handleKeyboardShortcuts);
  document.addEventListener("visibilitychange", handleVisibilityChange);
  window.addEventListener("online", handleOnline);
  window.addEventListener("offline", handleOffline);

  // Initialize application
  await initializeApp();

  // Show debug panel in development
  if (env.DEV && env.VITE_SHOW_DEBUG === "true") {
    showDebugPanel.value = true;
  }
});

onUnmounted(() => {
  // Cleanup
  document.removeEventListener("keydown", handleKeyboardShortcuts);
  document.removeEventListener("visibilitychange", handleVisibilityChange);
  window.removeEventListener("online", handleOnline);
  window.removeEventListener("offline", handleOffline);

  // Stop auth checks
  authStore.stopTokenCheck();

  // Cleanup app store
  appStore.cleanup();
});
</script>

<style lang="scss">
@use "@/styles/variables" as *;

#app {
  width: 100%;
  height: 100vh;
  position: relative;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
  font-family: $font-family-base;
  overflow: hidden;
  display: flex;
  flex-direction: column;

  &.app-initializing {
    overflow: hidden;
  }

  &.app-error {
    overflow: hidden;
  }
}

// Initialization screen
.app-initializing {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--el-bg-color-page);
  z-index: 9999;
  @include flex-center;

  .initializing-content {
    text-align: center;
    animation: fade-in 0.5s ease-out;

    .initializing-icon {
      color: var(--el-color-primary);
      margin-bottom: $spacing-lg;
    }

    .initializing-title {
      font-size: 2rem;
      font-weight: $font-weight-bold;
      color: var(--el-text-color-primary);
      margin-bottom: $spacing-sm;
    }

    .initializing-text {
      color: var(--el-text-color-secondary);
      margin-bottom: $spacing-xl;
    }

    .initializing-spinner {
      .el-icon {
        color: var(--el-color-primary);
        animation: rotate 1s linear infinite;
      }
    }
  }
}

// Global error overlay
.global-error-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  backdrop-filter: blur(4px);
  z-index: 9998;
  @include flex-center;

  .error-content {
    background: var(--el-bg-color);
    border-radius: $border-radius-large;
    padding: $spacing-xxxl;
    max-width: 500px;
    margin: $spacing-lg;
    text-align: center;
    box-shadow: $box-shadow-dark;

    .error-icon {
      margin-bottom: $spacing-lg;
    }

    .error-title {
      font-size: 1.5rem;
      font-weight: $font-weight-bold;
      color: var(--el-text-color-primary);
      margin-bottom: $spacing-base;
    }

    .error-message {
      color: var(--el-text-color-secondary);
      margin-bottom: $spacing-xl;
      line-height: 1.5;
    }

    .error-actions {
      display: flex;
      gap: $spacing-base;
      justify-content: center;
    }
  }
}

// Debug panel
.debug-panel {
  position: fixed;
  top: $spacing-lg;
  right: $spacing-lg;
  width: 300px;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: $border-radius-base;
  box-shadow: $box-shadow-medium;
  z-index: 9000;
  font-size: $font-size-small;

  .debug-header {
    @include flex-between;
    padding: $spacing-sm $spacing-base;
    background: var(--el-fill-color-lighter);
    border-bottom: 1px solid var(--el-border-color-light);
    font-weight: $font-weight-primary;
  }

  .debug-content {
    padding: $spacing-base;
    max-height: 300px;
    overflow-y: auto;
  }

  .debug-item {
    margin-bottom: $spacing-sm;
    line-height: 1.4;

    strong {
      color: var(--el-color-primary);
    }
  }
}

// Page transitions
.fade-enter-active,
.fade-leave-active {
  transition: opacity $transition-base ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: all $transition-base $ease-out-quart;
}

.slide-up-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.slide-up-leave-to {
  opacity: 0;
  transform: translateY(-20px);
}

.slide-left-enter-active,
.slide-left-leave-active {
  transition: all $transition-base ease;
}

.slide-left-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

// Responsive design
@include max-screen($breakpoint-sm) {
  .debug-panel {
    top: $spacing-sm;
    right: $spacing-sm;
    left: $spacing-sm;
    width: auto;
  }

  .global-error-overlay {
    .error-content {
      margin: $spacing-base;
      padding: $spacing-lg;

      .error-actions {
        flex-direction: column;
      }
    }
  }
}

// Dark mode adjustments
.app-dark {
  .app-initializing {
    background: var(--el-bg-color-page);
  }

  .debug-panel {
    background: var(--el-bg-color-overlay);
    border-color: var(--el-border-color-darker);
  }
}

// Print styles
@media print {
  .debug-panel,
  .global-error-overlay {
    display: none !important;
  }
}

// Animations
@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes fade-in {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
