import { createRouter, createWebHistory } from "vue-router";
import type { RouteRecordRaw } from "vue-router";
import { useAuthStore } from "@/store/auth";
import { useAppStore } from "@/store/app";
import { TokenManager, AuthUtils } from "@/utils/auth";
import NProgress from "nprogress";

// Configure NProgress
NProgress.configure({
  showSpinner: false,
  speed: 500,
  minimum: 0.2,
});

const routes: RouteRecordRaw[] = [
  // Authentication routes
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: {
      title: "Login",
      requiresAuth: false,
      hideInMenu: true,
    },
  },
  {
    path: "/register",
    name: "Register",
    component: () => import("@/views/Register.vue"),
    meta: {
      title: "Register",
      requiresAuth: false,
      hideInMenu: true,
    },
  },
  {
    path: "/forgot-password",
    name: "ForgotPassword",
    component: () => import("@/views/ForgotPassword.vue"),
    meta: {
      title: "Forgot Password",
      requiresAuth: false,
      hideInMenu: true,
    },
  },

  // Main application routes
  {
    path: "/",
    redirect: "/dashboard",
    component: () => import("@/views/Layout.vue"),
    meta: {
      requiresAuth: true,
    },
    children: [
      // Dashboard
      {
        path: "/dashboard",
        name: "Dashboard",
        component: () => import("@/views/Dashboard.vue"),
        meta: {
          title: "Dashboard",
          icon: "Dashboard",
          permission: "read",
        },
      },

      // Container management
      {
        path: "/containers",
        name: "Containers",
        component: () => import("@/views/Containers.vue"),
        meta: {
          title: "Containers",
          icon: "Box",
          permission: "container:read",
        },
      },
      {
        path: "/containers/running",
        name: "RunningContainers",
        component: () => import("@/views/containers/Running.vue"),
        meta: {
          title: "Running Containers",
          icon: "SuccessFilled",
          permission: "container:read",
          hideInMenu: true,
        },
      },
      {
        path: "/containers/stopped",
        name: "StoppedContainers",
        component: () => import("@/views/containers/Stopped.vue"),
        meta: {
          title: "Stopped Containers",
          icon: "Warning",
          permission: "container:read",
          hideInMenu: true,
        },
      },
      {
        path: "/containers/:id",
        name: "ContainerDetail",
        component: () => import("@/views/ContainerDetail.vue"),
        meta: {
          title: "Container Details",
          permission: "container:read",
          hideInMenu: true,
        },
      },

      // Update management
      {
        path: "/updates",
        name: "Updates",
        component: () => import("@/views/Updates.vue"),
        meta: {
          title: "Updates Center",
          icon: "UpdateFilled",
          permission: "update:read",
        },
      },
      {
        path: "/updates/history",
        name: "UpdateHistory",
        component: () => import("@/views/UpdateHistory.vue"),
        meta: {
          title: "Update History",
          icon: "Clock",
          permission: "update:read",
        },
      },

      // Settings (Admin only)
      {
        path: "/settings",
        name: "Settings",
        component: () => import("@/views/Settings.vue"),
        meta: {
          title: "System Settings",
          icon: "Setting",
          role: "admin",
        },
      },
    ],
  },

  // Error pages
  {
    path: "/403",
    name: "Forbidden",
    component: () => import("@/views/errors/403.vue"),
    meta: {
      title: "Access Denied",
      hideInMenu: true,
    },
  },
  {
    path: "/404",
    name: "NotFound",
    component: () => import("@/views/errors/404.vue"),
    meta: {
      title: "Page Not Found",
      hideInMenu: true,
    },
  },
  {
    path: "/500",
    name: "ServerError",
    component: () => import("@/views/errors/500.vue"),
    meta: {
      title: "Server Error",
      hideInMenu: true,
    },
  },

  // Catch all route
  {
    path: "/:pathMatch(.*)*",
    redirect: "/404",
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 };
  },
});

// Global navigation guards
router.beforeEach(async (to, from, next) => {
  NProgress.start();

  try {
    // Set page title
    const title = to.meta?.title
      ? `${to.meta.title} - Docker Auto Update System`
      : "Docker Auto Update System";
    document.title = title;

    // Get stores
    const authStore = useAuthStore();
    const appStore = useAppStore();

    // Set page loading state
    appStore.setPageLoading(true);

    // Check if route requires authentication
    const requiresAuth = to.meta?.requiresAuth !== false;

    if (requiresAuth) {
      // Check if user is authenticated
      if (!authStore.isAuthenticated) {
        // Check if there's a valid token that can be used
        const token = TokenManager.getAccessToken();
        if (token && TokenManager.isTokenValid(token)) {
          try {
            // Try to get user info with the token
            await authStore.getCurrentUser();
          } catch (error) {
            console.error("Failed to authenticate with existing token:", error);
            // Clear invalid tokens and redirect to login
            TokenManager.clearTokens();
            const redirectUrl = to.fullPath !== "/" ? to.fullPath : undefined;
            AuthUtils.redirectToLogin(redirectUrl);
            return;
          }
        } else {
          // No valid token, redirect to login
          const redirectUrl = to.fullPath !== "/" ? to.fullPath : undefined;
          AuthUtils.redirectToLogin(redirectUrl);
          return;
        }
      }

      // Check role-based access
      if (to.meta?.role) {
        if (!authStore.hasRole(to.meta.role as string)) {
          console.warn(
            `Access denied: Required role '${to.meta.role}' not found`,
          );
          next("/403");
          return;
        }
      }

      // Check permission-based access
      if (to.meta?.permission) {
        if (!authStore.hasPermission(to.meta.permission as string)) {
          console.warn(
            `Access denied: Required permission '${to.meta.permission}' not found`,
          );
          next("/403");
          return;
        }
      }

      // Check if token needs refresh
      const token = TokenManager.getAccessToken();
      if (token && TokenManager.needsRefresh(token)) {
        try {
          await authStore.refreshToken();
        } catch (error) {
          console.warn("Token refresh failed:", error);
          // Continue anyway, the request interceptor will handle it
        }
      }
    } else {
      // Route doesn't require auth
      // If user is already authenticated and tries to access auth pages, redirect to dashboard
      if (authStore.isAuthenticated) {
        const authPages = ["/login", "/register", "/forgot-password"];
        if (authPages.includes(to.path)) {
          next("/dashboard");
          return;
        }
      }
    }

    // Check for any custom route validation
    if (to.meta?.validate && typeof to.meta.validate === "function") {
      const validationResult = await to.meta.validate(to, from);
      if (validationResult !== true) {
        if (typeof validationResult === "string") {
          next(validationResult);
        } else {
          next(false);
        }
        return;
      }
    }

    // All checks passed, proceed to route
    next();
  } catch (error) {
    console.error("Navigation guard error:", error);

    // Handle navigation errors gracefully
    if (to.path !== "/500") {
      next("/500");
    } else {
      next(false);
    }
  }
});

router.afterEach((to, from) => {
  NProgress.done();

  // Clear page loading state
  const appStore = useAppStore();
  appStore.setPageLoading(false);

  // Log route changes in development
  if (import.meta.env.DEV) {
    console.log(`Route changed: ${from.path} -> ${to.path}`);
  }

  // Analytics tracking (add your analytics service here)
  // trackPageView(to.path, to.meta?.title)
});

// Navigation error handler
router.onError((error) => {
  console.error("Router error:", error);
  NProgress.done();

  // Clear page loading state
  const appStore = useAppStore();
  appStore.setPageLoading(false);

  // Show error notification
  appStore.addNotification({
    type: "error",
    title: "Navigation Error",
    message: "Failed to load the requested page. Please try again.",
    duration: 5000,
  });
});

// Add route change detection for auth token validation
const tokenCheckInterval: NodeJS.Timeout | null = null;

router.beforeEach(() => {
  // Start periodic token validation when user is authenticated
  const authStore = useAuthStore();
  if (authStore.isAuthenticated && !tokenCheckInterval) {
    authStore.startTokenCheck();
  } else if (!authStore.isAuthenticated && tokenCheckInterval) {
    authStore.stopTokenCheck();
  }
});

export default router;
