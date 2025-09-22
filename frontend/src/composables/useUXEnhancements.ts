/**
 * UX Enhancement composables for better user experience
 */
import { ref, onMounted, onUnmounted } from "vue";
import { ElMessage, ElMessageBox, ElLoading } from "element-plus";
import { retryWithBackoff, debounce, throttle } from "@/utils/network";
import type { LoadingInstance } from "element-plus";

export interface LoadingState {
  isLoading: boolean;
  message: string;
  progress: number;
}

export interface RetryOptions {
  maxRetries?: number;
  baseDelay?: number;
  showProgress?: boolean;
  progressMessage?: string;
}

/**
 * Enhanced loading state management
 */
export const useLoadingState = () => {
  const loadingStates = ref<Map<string, LoadingState>>(new Map());
  const globalLoading = ref(false);

  const startLoading = (
    key: string,
    message: string = "加载中...",
    progress: number = -1
  ) => {
    loadingStates.value.set(key, {
      isLoading: true,
      message,
      progress,
    });
    updateGlobalLoading();
  };

  const updateLoading = (
    key: string,
    message?: string,
    progress?: number
  ) => {
    const state = loadingStates.value.get(key);
    if (state) {
      if (message !== undefined) state.message = message;
      if (progress !== undefined) state.progress = progress;
      loadingStates.value.set(key, state);
    }
  };

  const stopLoading = (key: string) => {
    loadingStates.value.delete(key);
    updateGlobalLoading();
  };

  const updateGlobalLoading = () => {
    globalLoading.value = loadingStates.value.size > 0;
  };

  const isLoading = (key?: string) => {
    if (key) {
      return loadingStates.value.get(key)?.isLoading || false;
    }
    return globalLoading.value;
  };

  const getLoadingMessage = (key: string) => {
    return loadingStates.value.get(key)?.message || "";
  };

  const getLoadingProgress = (key: string) => {
    return loadingStates.value.get(key)?.progress || -1;
  };

  return {
    loadingStates,
    globalLoading,
    startLoading,
    updateLoading,
    stopLoading,
    isLoading,
    getLoadingMessage,
    getLoadingProgress,
  };
};

/**
 * Enhanced error handling with retry capabilities
 */
export const useErrorHandling = () => {
  const errors = ref<Map<string, Error>>(new Map());

  const handleError = (
    error: Error,
    key?: string,
    showMessage: boolean = true
  ) => {
    console.error("Error handled:", error);

    if (key) {
      errors.value.set(key, error);
    }

    if (showMessage) {
      const userFriendlyMessage = getUserFriendlyErrorMessage(error);
      ElMessage.error(userFriendlyMessage);
    }
  };

  const clearError = (key: string) => {
    errors.value.delete(key);
  };

  const hasError = (key: string) => {
    return errors.value.has(key);
  };

  const getError = (key: string) => {
    return errors.value.get(key);
  };

  const getUserFriendlyErrorMessage = (error: Error): string => {
    if (error.message.includes("Network Error")) {
      return "网络连接失败，请检查网络设置";
    }
    if (error.message.includes("timeout")) {
      return "请求超时，请稍后重试";
    }
    if (error.message.includes("401")) {
      return "身份验证失败，请重新登录";
    }
    if (error.message.includes("403")) {
      return "权限不足，无法执行此操作";
    }
    if (error.message.includes("404")) {
      return "请求的资源不存在";
    }
    if (error.message.includes("500")) {
      return "服务器内部错误，请稍后重试";
    }
    return error.message || "操作失败，请重试";
  };

  const retryOperation = async <T>(
    operation: () => Promise<T>,
    options: RetryOptions = {}
  ): Promise<T> => {
    const {
      maxRetries = 3,
      baseDelay = 1000,
      showProgress = true,
      progressMessage = "重试中...",
    } = options;

    let loadingInstance: LoadingInstance | null = null;

    if (showProgress) {
      loadingInstance = ElLoading.service({
        lock: true,
        text: progressMessage,
        spinner: "el-icon-loading",
      });
    }

    try {
      const result = await retryWithBackoff(operation, {
        maxRetries,
        baseDelay,
        maxDelay: 10000,
        backoffFactor: 2,
      });

      if (loadingInstance) {
        loadingInstance.close();
      }

      return result;
    } catch (error) {
      if (loadingInstance) {
        loadingInstance.close();
      }

      handleError(error as Error);
      throw error;
    }
  };

  return {
    errors,
    handleError,
    clearError,
    hasError,
    getError,
    getUserFriendlyErrorMessage,
    retryOperation,
  };
};

/**
 * User feedback and confirmation utilities
 */
export const useUserFeedback = () => {
  const showConfirmDialog = async (
    message: string,
    title: string = "确认操作",
    type: "warning" | "info" | "error" = "warning"
  ): Promise<boolean> => {
    try {
      await ElMessageBox.confirm(message, title, {
        confirmButtonText: "确认",
        cancelButtonText: "取消",
        type,
      });
      return true;
    } catch {
      return false;
    }
  };

  const showSuccessMessage = (message: string, duration: number = 3000) => {
    ElMessage.success({
      message,
      duration,
      showClose: true,
    });
  };

  const showErrorMessage = (message: string, duration: number = 5000) => {
    ElMessage.error({
      message,
      duration,
      showClose: true,
    });
  };

  const showWarningMessage = (message: string, duration: number = 4000) => {
    ElMessage.warning({
      message,
      duration,
      showClose: true,
    });
  };

  const showInfoMessage = (message: string, duration: number = 3000) => {
    ElMessage.info({
      message,
      duration,
      showClose: true,
    });
  };

  const showPromptDialog = async (
    message: string,
    title: string = "输入",
    inputPattern?: RegExp,
    inputErrorMessage?: string
  ): Promise<string | null> => {
    try {
      const result = await ElMessageBox.prompt(message, title, {
        confirmButtonText: "确认",
        cancelButtonText: "取消",
        inputPattern,
        inputErrorMessage: inputErrorMessage || "输入格式不正确",
      });
      return result.value;
    } catch {
      return null;
    }
  };

  return {
    showConfirmDialog,
    showSuccessMessage,
    showErrorMessage,
    showWarningMessage,
    showInfoMessage,
    showPromptDialog,
  };
};

/**
 * Performance optimization utilities
 */
export const usePerformanceOptimization = () => {
  const debouncedCallbacks = new Map<string, (...args: any[]) => void>();
  const throttledCallbacks = new Map<string, (...args: any[]) => void>();

  const createDebouncedCallback = <T extends (...args: any[]) => any>(
    key: string,
    callback: T,
    delay: number = 300
  ): T => {
    if (!debouncedCallbacks.has(key)) {
      debouncedCallbacks.set(key, debounce(callback, delay));
    }
    return debouncedCallbacks.get(key) as T;
  };

  const createThrottledCallback = <T extends (...args: any[]) => any>(
    key: string,
    callback: T,
    delay: number = 300
  ): T => {
    if (!throttledCallbacks.has(key)) {
      throttledCallbacks.set(key, throttle(callback, delay));
    }
    return throttledCallbacks.get(key) as T;
  };

  const measurePerformance = async <T>(
    operation: () => Promise<T>,
    label: string = "Operation"
  ): Promise<T> => {
    const start = performance.now();

    try {
      const result = await operation();
      const end = performance.now();
      console.log(`${label} took ${(end - start).toFixed(2)}ms`);
      return result;
    } catch (error) {
      const end = performance.now();
      console.error(`${label} failed after ${(end - start).toFixed(2)}ms:`, error);
      throw error;
    }
  };

  const cleanup = () => {
    debouncedCallbacks.clear();
    throttledCallbacks.clear();
  };

  return {
    createDebouncedCallback,
    createThrottledCallback,
    measurePerformance,
    cleanup,
  };
};

/**
 * Auto-save functionality
 */
export const useAutoSave = <T>(
  data: () => T,
  saveCallback: (data: T) => Promise<void>,
  interval: number = 30000 // 30 seconds
) => {
  const lastSaved = ref<Date | null>(null);
  const saving = ref(false);
  const hasUnsavedChanges = ref(false);
  const autoSaveEnabled = ref(true);

  let autoSaveTimer: NodeJS.Timeout | null = null;
  let lastDataSnapshot = JSON.stringify(data());

  const triggerAutoSave = async () => {
    if (!autoSaveEnabled.value || saving.value) return;

    const currentSnapshot = JSON.stringify(data());
    if (currentSnapshot === lastDataSnapshot) return;

    saving.value = true;
    hasUnsavedChanges.value = true;

    try {
      await saveCallback(data());
      lastSaved.value = new Date();
      hasUnsavedChanges.value = false;
      lastDataSnapshot = currentSnapshot;
    } catch (error) {
      console.error("Auto-save failed:", error);
    } finally {
      saving.value = false;
    }
  };

  const startAutoSave = () => {
    if (autoSaveTimer) clearInterval(autoSaveTimer);

    autoSaveTimer = setInterval(triggerAutoSave, interval);
  };

  const stopAutoSave = () => {
    if (autoSaveTimer) {
      clearInterval(autoSaveTimer);
      autoSaveTimer = null;
    }
  };

  const manualSave = async () => {
    await triggerAutoSave();
  };

  onMounted(() => {
    if (autoSaveEnabled.value) {
      startAutoSave();
    }
  });

  onUnmounted(() => {
    stopAutoSave();
  });

  return {
    lastSaved,
    saving,
    hasUnsavedChanges,
    autoSaveEnabled,
    manualSave,
    startAutoSave,
    stopAutoSave,
  };
};

/**
 * Accessibility enhancements
 */
export const useAccessibility = () => {
  const announceToScreenReader = (message: string) => {
    const announcement = document.createElement("div");
    announcement.setAttribute("aria-live", "polite");
    announcement.setAttribute("aria-atomic", "true");
    announcement.style.position = "absolute";
    announcement.style.left = "-10000px";
    announcement.style.width = "1px";
    announcement.style.height = "1px";
    announcement.style.overflow = "hidden";

    document.body.appendChild(announcement);
    announcement.textContent = message;

    setTimeout(() => {
      document.body.removeChild(announcement);
    }, 1000);
  };

  const trapFocus = (element: HTMLElement) => {
    const focusableElements = element.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );

    const firstElement = focusableElements[0] as HTMLElement;
    const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

    const handleTabKey = (e: KeyboardEvent) => {
      if (e.key === "Tab") {
        if (e.shiftKey) {
          if (document.activeElement === firstElement) {
            lastElement.focus();
            e.preventDefault();
          }
        } else {
          if (document.activeElement === lastElement) {
            firstElement.focus();
            e.preventDefault();
          }
        }
      }
    };

    element.addEventListener("keydown", handleTabKey);

    return () => {
      element.removeEventListener("keydown", handleTabKey);
    };
  };

  return {
    announceToScreenReader,
    trapFocus,
  };
};