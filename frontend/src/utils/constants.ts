/**
 * Application constants and configuration
 */

// API Base URL
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ||
  (import.meta.env.DEV ? "http://localhost:8080" : "");

// Authentication endpoints
export const AUTH_ENDPOINTS = {
  LOGIN: "/api/auth/login",
  LOGOUT: "/api/auth/logout",
  REFRESH: "/api/auth/refresh",
  PROFILE: "/api/auth/me",
} as const;

// System endpoints
export const SYSTEM_ENDPOINTS = {
  HEALTH: "/api/health",
  CONFIG: "/api/system/config",
  STATS: "/api/system/stats",
} as const;

// Docker endpoints
export const DOCKER_ENDPOINTS = {
  CONTAINERS: "/containers",
  IMAGES: "/images",
  UPDATES: "/updates",
  LOGS: "/logs",
} as const;

// Storage keys
export const STORAGE_KEYS = {
  ACCESS_TOKEN: "access_token",
  REFRESH_TOKEN: "refresh_token",
  USER_INFO: "user_info",
  THEME: "theme",
  LANGUAGE: "language",
} as const;

// User roles
export const USER_ROLES = {
  ADMIN: "admin",
  OPERATOR: "operator",
  VIEWER: "viewer",
} as const;

// Permission levels
export const PERMISSIONS = {
  READ: "read",
  WRITE: "write",
  ADMIN: "admin",
} as const;

// HTTP status codes
export const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  NO_CONTENT: 204,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  INTERNAL_SERVER_ERROR: 500,
} as const;

// Request timeout (in milliseconds)
export const REQUEST_TIMEOUT = 30000;

// Token refresh threshold (in minutes)
export const TOKEN_REFRESH_THRESHOLD = 5;

// Navigation menu items based on roles
export const MENU_ITEMS = {
  [USER_ROLES.ADMIN]: [
    {
      path: "/dashboard",
      title: "仪表盘",
      icon: "Dashboard",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/containers",
      title: "容器管理",
      icon: "Box",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/images",
      title: "镜像管理",
      icon: "Picture",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/updates",
      title: "更新管理",
      icon: "Refresh",
      permission: PERMISSIONS.WRITE,
    },
    {
      path: "/logs",
      title: "日志查看",
      icon: "Document",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/settings",
      title: "系统设置",
      icon: "Setting",
      permission: PERMISSIONS.ADMIN,
    },
    {
      path: "/users",
      title: "用户管理",
      icon: "User",
      permission: PERMISSIONS.ADMIN,
    },
  ],
  [USER_ROLES.OPERATOR]: [
    {
      path: "/dashboard",
      title: "仪表盘",
      icon: "Dashboard",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/containers",
      title: "容器管理",
      icon: "Box",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/images",
      title: "镜像管理",
      icon: "Picture",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/updates",
      title: "更新管理",
      icon: "Refresh",
      permission: PERMISSIONS.WRITE,
    },
    {
      path: "/logs",
      title: "日志查看",
      icon: "Document",
      permission: PERMISSIONS.READ,
    },
  ],
  [USER_ROLES.VIEWER]: [
    {
      path: "/dashboard",
      title: "仪表盘",
      icon: "Dashboard",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/containers",
      title: "容器管理",
      icon: "Box",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/images",
      title: "镜像管理",
      icon: "Picture",
      permission: PERMISSIONS.READ,
    },
    {
      path: "/logs",
      title: "日志查看",
      icon: "Document",
      permission: PERMISSIONS.READ,
    },
  ],
} as const;

// Notification types
export const NOTIFICATION_TYPES = {
  SUCCESS: "success",
  WARNING: "warning",
  ERROR: "error",
  INFO: "info",
} as const;

// WebSocket event types
export const WS_EVENTS = {
  CONNECT: "connect",
  DISCONNECT: "disconnect",
  UPDATE_AVAILABLE: "update_available",
  UPDATE_STARTED: "update_started",
  UPDATE_COMPLETED: "update_completed",
  UPDATE_FAILED: "update_failed",
  CONTAINER_STATUS: "container_status",
  SYSTEM_ALERT: "system_alert",
} as const;

// Update status
export const UPDATE_STATUS = {
  PENDING: "pending",
  RUNNING: "running",
  COMPLETED: "completed",
  FAILED: "failed",
  CANCELLED: "cancelled",
} as const;

// Container status
export const CONTAINER_STATUS = {
  RUNNING: "running",
  STOPPED: "stopped",
  PAUSED: "paused",
  RESTARTING: "restarting",
  DEAD: "dead",
  CREATED: "created",
  EXITED: "exited",
} as const;

// Date format patterns
export const DATE_FORMATS = {
  DATETIME: "YYYY-MM-DD HH:mm:ss",
  DATE: "YYYY-MM-DD",
  TIME: "HH:mm:ss",
  ISO: "YYYY-MM-DDTHH:mm:ssZ",
} as const;

// Pagination defaults
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 20,
  PAGE_SIZES: [10, 20, 50, 100],
} as const;

// Theme configuration
export const THEMES = {
  LIGHT: "light",
  DARK: "dark",
  AUTO: "auto",
} as const;

// Language options
export const LANGUAGES = {
  EN: "en",
  ZH: "zh",
} as const;
