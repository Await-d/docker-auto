/**
 * Application store for global state management
 */
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { THEMES, LANGUAGES, STORAGE_KEYS } from "@/utils/constants";

export interface AppState {
  theme: "light" | "dark" | "auto";
  language: string;
  sidebarCollapsed: boolean;
  pageLoading: boolean;
  notifications: Notification[];
}

export interface Notification {
  id: string;
  type: "success" | "warning" | "error" | "info";
  title: string;
  message: string;
  duration?: number;
  timestamp: number;
  read: boolean;
}

export const useAppStore = defineStore("app", () => {
  // State
  const theme = ref<"light" | "dark" | "auto">(
    (localStorage.getItem(STORAGE_KEYS.THEME) as any) || THEMES.AUTO,
  );
  const language = ref<string>(
    localStorage.getItem(STORAGE_KEYS.LANGUAGE) || LANGUAGES.EN,
  );
  const sidebarCollapsed = ref(false);
  const pageLoading = ref(false);
  const notifications = ref<Notification[]>([]);

  // Computed
  const isDarkMode = computed(() => {
    if (theme.value === THEMES.DARK) {
      return true;
    }
    if (theme.value === THEMES.LIGHT) {
      return false;
    }
    // Auto mode - check system preference
    return window.matchMedia("(prefers-color-scheme: dark)").matches;
  });

  const currentLanguage = computed(() => language.value);

  // Actions
  const setTheme = (newTheme: "light" | "dark" | "auto") => {
    theme.value = newTheme;
    localStorage.setItem(STORAGE_KEYS.THEME, newTheme);

    // Apply theme to document
    const html = document.documentElement;
    if (
      newTheme === THEMES.DARK ||
      (newTheme === THEMES.AUTO && isDarkMode.value)
    ) {
      html.classList.add("dark");
    } else {
      html.classList.remove("dark");
    }
  };

  const setLanguage = (newLanguage: string) => {
    language.value = newLanguage;
    localStorage.setItem(STORAGE_KEYS.LANGUAGE, newLanguage);

    // Update document language
    document.documentElement.lang = newLanguage;
  };

  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value;
  };

  const setSidebarCollapsed = (collapsed: boolean) => {
    sidebarCollapsed.value = collapsed;
  };

  const setPageLoading = (loading: boolean) => {
    pageLoading.value = loading;
  };

  const addNotification = (
    notification: Omit<Notification, "id" | "timestamp" | "read">,
  ) => {
    const newNotification: Notification = {
      ...notification,
      id: Date.now().toString() + Math.random().toString(36).substr(2, 9),
      timestamp: Date.now(),
      read: false,
    };

    notifications.value.push(newNotification);

    // Auto remove notification if duration is set
    if (notification.duration && notification.duration > 0) {
      setTimeout(() => {
        removeNotification(newNotification.id);
      }, notification.duration);
    }

    return newNotification.id;
  };

  const removeNotification = (id: string) => {
    const index = notifications.value.findIndex((n) => n.id === id);
    if (index > -1) {
      notifications.value.splice(index, 1);
    }
  };

  const clearNotifications = () => {
    notifications.value = [];
  };

  const updateNotification = (id: string, updates: Partial<Notification>) => {
    const notification = notifications.value.find((n) => n.id === id);
    if (notification) {
      Object.assign(notification, updates);
    }
  };

  // Initialize theme on store creation
  const initializeTheme = () => {
    // Listen for system theme changes
    if (theme.value === THEMES.AUTO) {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      mediaQuery.addEventListener("change", () => {
        if (theme.value === THEMES.AUTO) {
          setTheme(THEMES.AUTO); // Re-apply auto theme
        }
      });
    }

    // Apply initial theme
    setTheme(theme.value);
  };

  // Responsive breakpoint utilities
  const getScreenSize = () => {
    const width = window.innerWidth;
    if (width < 640) return "xs";
    if (width < 768) return "sm";
    if (width < 1024) return "md";
    if (width < 1280) return "lg";
    return "xl";
  };

  const screenSize = ref(getScreenSize());

  const updateScreenSize = () => {
    screenSize.value = getScreenSize();
  };

  // Viewport utilities
  const isMobile = computed(
    () => screenSize.value === "xs" || screenSize.value === "sm",
  );
  const isTablet = computed(() => screenSize.value === "md");
  const isDesktop = computed(
    () => screenSize.value === "lg" || screenSize.value === "xl",
  );

  // Initialize screen size tracking
  const initializeScreenSize = () => {
    window.addEventListener("resize", updateScreenSize);
    updateScreenSize();
  };

  // Cleanup
  const cleanup = () => {
    window.removeEventListener("resize", updateScreenSize);
  };

  // Global error handling
  const handleError = (error: Error, context?: string) => {
    console.error(`[${context || "App"}] Error:`, error);

    addNotification({
      type: "error",
      title: "An error occurred",
      message: error.message || "Something went wrong",
      duration: 5000,
    });
  };

  // Initialize app
  const initialize = () => {
    initializeTheme();
    initializeScreenSize();
  };

  return {
    // State
    theme: readonly(theme),
    language: readonly(language),
    sidebarCollapsed: readonly(sidebarCollapsed),
    pageLoading: readonly(pageLoading),
    notifications: readonly(notifications),
    screenSize: readonly(screenSize),

    // Computed
    isDarkMode,
    currentLanguage,
    isMobile,
    isTablet,
    isDesktop,

    // Actions
    setTheme,
    setLanguage,
    toggleSidebar,
    setSidebarCollapsed,
    setPageLoading,
    addNotification,
    removeNotification,
    clearNotifications,
    updateNotification,
    handleError,
    initialize,
    cleanup,
  };
});

// Export convenience composable
export const useApp = () => {
  const appStore = useAppStore();

  return {
    ...appStore,

    // Additional convenience methods
    showSuccess: (message: string, title = "Success") =>
      appStore.addNotification({
        type: "success",
        title,
        message,
        duration: 3000,
      }),

    showError: (message: string, title = "Error") =>
      appStore.addNotification({
        type: "error",
        title,
        message,
        duration: 5000,
      }),

    showWarning: (message: string, title = "Warning") =>
      appStore.addNotification({
        type: "warning",
        title,
        message,
        duration: 4000,
      }),

    showInfo: (message: string, title = "Info") =>
      appStore.addNotification({
        type: "info",
        title,
        message,
        duration: 3000,
      }),

    // Theme helpers
    toggleTheme: () => {
      const current = appStore.theme;
      const next =
        current === THEMES.LIGHT
          ? THEMES.DARK
          : current === THEMES.DARK
            ? THEMES.AUTO
            : THEMES.LIGHT;
      appStore.setTheme(next);
    },

    // Responsive helpers
    isBreakpoint: (breakpoint: string) => appStore.screenSize === breakpoint,
    isMinBreakpoint: (breakpoint: string) => {
      const order = ["xs", "sm", "md", "lg", "xl"];
      const currentIndex = order.indexOf(appStore.screenSize);
      const targetIndex = order.indexOf(breakpoint);
      return currentIndex >= targetIndex;
    },
  };
};
