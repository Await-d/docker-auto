import { createApp } from "vue";
import { createPinia } from "pinia";
import router from "./router";
import App from "./App.vue";

// Configure dayjs
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";

dayjs.extend(relativeTime);
dayjs.extend(utc);
dayjs.extend(timezone);

// Element Plus styles
import "element-plus/dist/index.css";
import "element-plus/theme-chalk/dark/css-vars.css";

// Custom styles
import "@/styles/index.scss";

// NProgress styles
import "nprogress/nprogress.css";

// Performance monitoring
console.log("🚀 Starting Docker Auto-Update System...");
console.log("Environment:", import.meta.env.MODE);
console.log("Version:", import.meta.env.VITE_APP_VERSION || "1.0.0");

// Create Vue app
const app = createApp(App);

// Create and install Pinia store
const pinia = createPinia();
app.use(pinia);

// Install router
app.use(router);

// Global properties for debugging
if (import.meta.env.DEV) {
  app.config.globalProperties.$log = console.log;
  app.config.globalProperties.$error = console.error;
  app.config.globalProperties.$warn = console.warn;
}

// Global error handler
app.config.errorHandler = (err: any, instance: any, info: string) => {
  console.error("Global Vue error:", err);
  console.error("Component instance:", instance);
  console.error("Error info:", info);

  // Report to monitoring service in production
  if (import.meta.env.PROD) {
    // Add your error reporting service here
    // reportError(err, { instance, info })
  }
};

// Global warning handler
app.config.warnHandler = (msg: string, _instance: any, trace: string) => {
  if (import.meta.env.DEV) {
    console.warn("Vue warning:", msg);
    console.warn("Component trace:", trace);
  }
};

// Performance mark
if (import.meta.env.DEV) {
  performance.mark("app-start");
}

// Mount the app
app.mount("#app");

// Performance measurement
if (import.meta.env.DEV) {
  performance.mark("app-mounted");
  performance.measure("app-startup", "app-start", "app-mounted");

  const startupTime = performance.getEntriesByName("app-startup")[0];
  console.log(`⚡ App mounted in ${startupTime.duration.toFixed(2)}ms`);
}

// Register service worker for PWA (if available)
if ("serviceWorker" in navigator && import.meta.env.PROD) {
  window.addEventListener("load", () => {
    navigator.serviceWorker
      .register("/sw.js")
      .then((registration) => {
        console.log("SW registered: ", registration);
      })
      .catch((registrationError) => {
        console.log("SW registration failed: ", registrationError);
      });
  });
}

// Global unhandled promise rejection handler
window.addEventListener("unhandledrejection", (event) => {
  console.error("Unhandled promise rejection:", event.reason);

  // Prevent the default browser behavior
  event.preventDefault();

  // Report to monitoring service
  if (import.meta.env.PROD) {
    // reportError(event.reason, { type: 'unhandledrejection' })
  }
});

// Global error handler for non-Vue errors
window.addEventListener("error", (event) => {
  console.error("Global error:", event.error);

  // Report to monitoring service
  if (import.meta.env.PROD) {
    // reportError(event.error, { type: 'global' })
  }
});

// Console welcome message
if (import.meta.env.DEV) {
  console.log(
    "%c🐳 Docker Auto-Update System %c🚀",
    "color: #409EFF; font-size: 20px; font-weight: bold;",
    "color: #67C23A; font-size: 20px;",
  );
  console.log(
    "%cDevelopment Mode - Debug features enabled",
    "color: #E6A23C; font-weight: bold;",
  );
  console.log(
    "%cKeyboard shortcuts:\n• Ctrl/Cmd + Shift + D: Toggle debug panel\n• Ctrl/Cmd + Shift + R: Reload app",
    "color: #909399;",
  );
}
