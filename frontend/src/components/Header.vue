<template>
  <header class="app-header">
    <div class="header-content">
      <!-- Left section -->
      <div class="header-left">
        <el-button
          type="text"
          :icon="Expand"
          class="sidebar-toggle"
          :title="sidebarCollapsed ? 'Expand Sidebar' : 'Collapse Sidebar'"
          @click="toggleSidebar"
        />

        <div class="breadcrumb-container">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item
              v-for="item in breadcrumbs"
              :key="item.path"
              :to="item.path === currentPath ? undefined : item.path"
            >
              <el-icon v-if="item.icon" class="breadcrumb-icon">
                <component :is="item.icon" />
              </el-icon>
              {{ item.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>
      </div>

      <!-- Center section -->
      <div class="header-center">
        <div class="system-status" :class="systemStatusClass">
          <el-icon class="status-icon">
            <component :is="systemStatusIcon" />
          </el-icon>
          <span class="status-text">{{ systemStatusText }}</span>
        </div>
      </div>

      <!-- Right section -->
      <div class="header-right">
        <!-- Notifications -->
        <el-badge
          :value="unreadNotifications"
          :max="99"
          :hidden="unreadNotifications === 0"
        >
          <el-button type="text" :icon="Bell" @click="showNotifications" />
        </el-badge>

        <!-- Theme toggle -->
        <el-tooltip content="Toggle Theme" placement="bottom">
          <el-button type="text" :icon="themeIcon" @click="toggleTheme" />
        </el-tooltip>

        <!-- Full screen toggle -->
        <el-tooltip
          :content="isFullscreen ? 'Exit Fullscreen' : 'Enter Fullscreen'"
          placement="bottom"
        >
          <el-button
            type="text"
            :icon="fullscreenIcon"
            @click="toggleFullscreen"
          />
        </el-tooltip>

        <!-- User dropdown -->
        <el-dropdown
placement="bottom-end" @command="handleUserCommand"
>
          <div class="user-info">
            <el-avatar :size="32" :src="userAvatar">
              {{ userInitials }}
            </el-avatar>
            <span class="username">{{ userDisplayName }}</span>
            <el-icon class="dropdown-arrow">
              <ArrowDown />
            </el-icon>
          </div>

          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile" :icon="User">
                Profile
              </el-dropdown-item>
              <el-dropdown-item command="settings" :icon="Setting">
                Settings
              </el-dropdown-item>
              <el-dropdown-item command="help" :icon="QuestionFilled">
                Help
              </el-dropdown-item>
              <el-dropdown-item divided command="logout" :icon="SwitchButton">
                Logout
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- Notifications drawer -->
    <el-drawer
      v-model="notificationDrawer"
      title="Notifications"
      direction="rtl"
      size="400px"
    >
      <div class="notifications-content">
        <div class="notifications-header">
          <el-button size="small" type="primary" @click="markAllAsRead">
            Mark All as Read
          </el-button>
          <el-button size="small" @click="clearAllNotifications">
            Clear All
          </el-button>
        </div>

        <div class="notifications-list">
          <div
            v-for="notification in notifications"
            :key="notification.id"
            class="notification-item"
            :class="{ unread: !notification.read }"
          >
            <el-icon
              class="notification-icon"
              :class="`notification-${notification.type}`"
            >
              <component :is="getNotificationIcon(notification.type)" />
            </el-icon>
            <div class="notification-content">
              <h4 class="notification-title">
                {{ notification.title }}
              </h4>
              <p class="notification-message">
                {{ notification.message }}
              </p>
              <span class="notification-time">{{
                formatTime(notification.timestamp)
              }}</span>
            </div>
            <el-button
              size="small"
              type="text"
              :icon="Close"
              @click="removeNotification(notification.id)"
            />
          </div>

          <el-empty
            v-if="notifications.length === 0"
            description="No notifications"
          />
        </div>
      </div>
    </el-drawer>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { dayjs } from "element-plus";
import {
  Expand,
  Bell,
  User,
  Setting,
  SwitchButton,
  ArrowDown,
  QuestionFilled,
  Close,
  Sunny,
  Moon,
  Monitor,
  FullScreen,
  Aim,
  SuccessFilled,
  WarningFilled,
  CircleCloseFilled,
  InfoFilled,
  CircleCheckFilled,
  CircleClose,
} from "@element-plus/icons-vue";
import { useAuth } from "@/store/auth";
import { useApp, type Notification } from "@/store/app";
import { storeToRefs } from "pinia";

// Composables
const route = useRoute();
const router = useRouter();
const authStore = useAuth();
const appStore = useApp();

// Reactive refs from stores
const { user } = storeToRefs(authStore);
const { userDisplayName, userAvatar } = authStore;
const { sidebarCollapsed, theme, notifications } = storeToRefs(appStore);

// Store methods
const { logout } = authStore;
const { toggleSidebar, toggleTheme, removeNotification, clearNotifications } =
  appStore;

// Reactive state
const notificationDrawer = ref(false);
const isFullscreen = ref(false);
const systemStatus = ref<"healthy" | "warning" | "error">("healthy");

// Computed properties
const breadcrumbs = computed(() => {
  const matched = route.matched.filter((item) => item.meta?.title);
  const crumbs = matched.map((item) => ({
    path: item.path,
    title: item.meta?.title as string,
    icon: item.meta?.icon as string,
  }));

  // Add home breadcrumb if not already present
  if (crumbs.length > 0 && crumbs[0].path !== "/") {
    crumbs.unshift({
      path: "/",
      title: "Dashboard",
      icon: "House",
    });
  }

  return crumbs;
});

const currentPath = computed(() => route.path);

const userInitials = computed(() => {
  if (!user.value) return "U";
  const name = userDisplayName.value;
  return name
    .split(" ")
    .map((word: string) => word.charAt(0))
    .join("")
    .toUpperCase()
    .slice(0, 2);
});

const unreadNotifications = computed(
  () => notifications.value.filter((n: Notification) => !n.read).length,
);

const themeIcon = computed(() => {
  switch (theme.value) {
    case "light":
      return Sunny;
    case "dark":
      return Moon;
    default:
      return Monitor;
  }
});

const fullscreenIcon = computed(() => (isFullscreen.value ? Aim : FullScreen));

const systemStatusClass = computed(() => `status-${systemStatus.value}`);

const systemStatusIcon = computed(() => {
  switch (systemStatus.value) {
    case "healthy":
      return CircleCheckFilled;
    case "warning":
      return WarningFilled;
    case "error":
      return CircleClose;
    default:
      return CircleCheckFilled;
  }
});

const systemStatusText = computed(() => {
  switch (systemStatus.value) {
    case "healthy":
      return "System Healthy";
    case "warning":
      return "System Warning";
    case "error":
      return "System Error";
    default:
      return "System Status";
  }
});

// Methods
const showNotifications = () => {
  notificationDrawer.value = true;
};

const handleUserCommand = async (command: string) => {
  switch (command) {
    case "profile":
      router.push("/profile");
      break;
    case "settings":
      router.push("/settings");
      break;
    case "help":
      router.push("/help");
      break;
    case "logout":
      await logout();
      break;
  }
};

const toggleFullscreen = () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen();
    isFullscreen.value = true;
  } else {
    document.exitFullscreen();
    isFullscreen.value = false;
  }
};

const markAllAsRead = () => {
  notifications.value.forEach((notification: Notification) => {
    notification.read = true;
  });
};

const clearAllNotifications = () => {
  clearNotifications();
  notificationDrawer.value = false;
};

const getNotificationIcon = (type: string) => {
  switch (type) {
    case "success":
      return SuccessFilled;
    case "warning":
      return WarningFilled;
    case "error":
      return CircleCloseFilled;
    case "info":
      return InfoFilled;
    default:
      return InfoFilled;
  }
};

const formatTime = (timestamp: number) => {
  return dayjs(timestamp).fromNow();
};

// Fullscreen event listeners
const handleFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement;
};

// System status simulation (replace with actual API call)
const checkSystemStatus = async () => {
  try {
    // Simulate API call
    const response = await fetch("/api/system/health");
    const data = await response.json();
    systemStatus.value = data.status || "healthy";
  } catch (error) {
    systemStatus.value = "error";
  }
};

// Lifecycle
onMounted(() => {
  document.addEventListener("fullscreenchange", handleFullscreenChange);
  checkSystemStatus();

  // Check system status periodically
  const statusInterval = setInterval(checkSystemStatus, 30000); // Every 30 seconds

  onUnmounted(() => {
    document.removeEventListener("fullscreenchange", handleFullscreenChange);
    clearInterval(statusInterval);
  });
});
</script>

<style scoped lang="scss">
.app-header {
  height: 64px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  position: relative;
  z-index: 1000;
}

.header-content {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  max-width: 100%;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
  min-width: 0;

  .sidebar-toggle {
    padding: 8px;

    &:hover {
      background: var(--el-fill-color-light);
    }
  }

  .breadcrumb-container {
    flex: 1;
    min-width: 0;

    .el-breadcrumb {
      font-weight: 500;

      .breadcrumb-icon {
        margin-right: 4px;
        vertical-align: -2px;
      }
    }
  }
}

.header-center {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;

  .system-status {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 500;

    .status-icon {
      font-size: 16px;
    }

    &.status-healthy {
      color: var(--el-color-success);
      background: var(--el-color-success-light-9);
    }

    &.status-warning {
      color: var(--el-color-warning);
      background: var(--el-color-warning-light-9);
    }

    &.status-error {
      color: var(--el-color-danger);
      background: var(--el-color-danger-light-9);
    }
  }
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;

  .el-button {
    padding: 8px;

    &:hover {
      background: var(--el-fill-color-light);
    }
  }

  .user-info {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 8px;
    border-radius: 6px;
    cursor: pointer;
    transition: background-color 0.2s;

    &:hover {
      background: var(--el-fill-color-light);
    }

    .username {
      font-size: 14px;
      font-weight: 500;
      color: var(--el-text-color-primary);
    }

    .dropdown-arrow {
      font-size: 12px;
      color: var(--el-text-color-regular);
      transition: transform 0.2s;
    }

    &:hover .dropdown-arrow {
      transform: rotate(180deg);
    }
  }
}

// Notifications
.notifications-content {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.notifications-header {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.notifications-list {
  flex: 1;
  overflow-y: auto;
}

.notification-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 8px;
  transition: background-color 0.2s;

  &:hover {
    background: var(--el-fill-color-lighter);
  }

  &.unread {
    background: var(--el-color-primary-light-9);
    border-left: 3px solid var(--el-color-primary);
  }

  .notification-icon {
    font-size: 18px;
    margin-top: 2px;

    &.notification-success {
      color: var(--el-color-success);
    }

    &.notification-warning {
      color: var(--el-color-warning);
    }

    &.notification-error {
      color: var(--el-color-danger);
    }

    &.notification-info {
      color: var(--el-color-info);
    }
  }

  .notification-content {
    flex: 1;
    min-width: 0;

    .notification-title {
      font-size: 14px;
      font-weight: 600;
      margin: 0 0 4px;
      color: var(--el-text-color-primary);
    }

    .notification-message {
      font-size: 13px;
      margin: 0 0 6px;
      color: var(--el-text-color-regular);
      line-height: 1.4;
    }

    .notification-time {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }
}

// Responsive design
@media (max-width: 768px) {
  .header-content {
    padding: 0 12px;
  }

  .header-left {
    gap: 8px;

    .breadcrumb-container {
      .el-breadcrumb {
        font-size: 13px;
      }
    }
  }

  .header-center {
    display: none; // Hide system status on mobile
  }

  .header-right {
    gap: 4px;

    .user-info .username {
      display: none; // Hide username on mobile
    }
  }
}

@media (max-width: 480px) {
  .header-left .breadcrumb-container {
    display: none; // Hide breadcrumbs on very small screens
  }
}
</style>
