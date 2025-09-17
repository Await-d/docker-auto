<template>
  <aside class="app-sidebar" :class="sidebarClass">
    <div class="sidebar-header">
      <div class="logo-container">
        <el-icon class="logo-icon" :size="32">
          <Box />
        </el-icon>
        <transition name="fade">
          <h1 v-show="!sidebarCollapsed"
class="logo-text"
>
Docker Auto
</h1>
        </transition>
      </div>
    </div>

    <div class="sidebar-content">
      <el-scrollbar class="sidebar-scrollbar">
        <el-menu
          :default-active="activeMenu"
          :collapse="sidebarCollapsed"
          :unique-opened="true"
          :router="true"
          class="sidebar-menu"
        >
          <template v-for="item in menuItems" :key="item.path">
            <!-- Single menu item -->
            <el-menu-item
              v-if="!item.children"
              :index="item.path"
              :disabled="item.disabled"
              @click="handleMenuClick(item)"
            >
              <el-icon>
                <component :is="item.icon" />
              </el-icon>
              <template #title>
                <span class="menu-title">{{ item.title }}</span>
                <el-badge
                  v-if="item.badge"
                  :value="item.badge"
                  :type="item.badgeType || 'primary'"
                  class="menu-badge"
                />
              </template>
            </el-menu-item>

            <!-- Submenu -->
            <el-sub-menu
v-else :index="item.path"
:disabled="item.disabled"
>
              <template #title>
                <el-icon>
                  <component :is="item.icon" />
                </el-icon>
                <span class="menu-title">{{ item.title }}</span>
                <el-badge
                  v-if="item.badge"
                  :value="item.badge"
                  :type="item.badgeType || 'primary'"
                  class="menu-badge"
                />
              </template>

              <el-menu-item
                v-for="child in item.children"
                :key="child.path"
                :index="child.path"
                :disabled="child.disabled"
                @click="handleMenuClick(child)"
              >
                <el-icon v-if="child.icon">
                  <component :is="child.icon" />
                </el-icon>
                <template #title>
                  <span class="menu-title">{{ child.title }}</span>
                  <el-badge
                    v-if="child.badge"
                    :value="child.badge"
                    :type="child.badgeType || 'primary'"
                    class="menu-badge"
                  />
                </template>
              </el-menu-item>
            </el-sub-menu>
          </template>
        </el-menu>
      </el-scrollbar>
    </div>

    <div class="sidebar-footer">
      <div
v-if="user" class="user-card"
>
        <el-avatar :size="sidebarCollapsed ? 32 : 40" :src="userAvatar">
          {{ userInitials }}
        </el-avatar>
        <transition name="fade">
          <div v-show="!sidebarCollapsed" class="user-info">
            <div class="user-name">
              {{ userDisplayName }}
            </div>
            <div class="user-role">
              {{ user.role }}
            </div>
          </div>
        </transition>
      </div>

      <el-tooltip
        :content="sidebarCollapsed ? 'Expand Sidebar' : 'Collapse Sidebar'"
        placement="right"
      >
        <el-button
          type="text"
          :icon="sidebarCollapsed ? Expand : Fold"
          class="toggle-button"
          @click="toggleSidebar"
        />
      </el-tooltip>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { Box, Expand, Fold } from "@element-plus/icons-vue";
import { useAuth } from "@/store/auth";
import { useApp } from "@/store/app";
import { storeToRefs } from "pinia";

// Composables
const route = useRoute();
const authStore = useAuth();
const appStore = useApp();

// Reactive refs from stores
const { user } = storeToRefs(authStore);
const { userDisplayName, userAvatar } = authStore;
const { sidebarCollapsed } = storeToRefs(appStore);

// Store methods
const { hasPermission, hasRole } = authStore;
const { toggleSidebar } = appStore;

interface MenuItem {
  path: string;
  title: string;
  icon: string;
  permission?: string;
  role?: string;
  disabled?: boolean;
  badge?: string | number;
  badgeType?: "primary" | "success" | "warning" | "danger" | "info";
  children?: MenuItem[];
}

// Computed properties
const sidebarClass = computed(() => ({
  "sidebar-collapsed": sidebarCollapsed.value,
}));

const activeMenu = computed(() => {
  return route.path;
});

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

// Get menu items based on user role and permissions
const menuItems = computed(() => {
  if (!user.value) return [];

  const items: MenuItem[] = [
    {
      path: "/dashboard",
      title: "Dashboard",
      icon: "Dashboard",
      permission: "read",
    },
    {
      path: "/containers",
      title: "Containers",
      icon: "Box",
      permission: "container:read",
      children: [
        {
          path: "/containers/running",
          title: "Running",
          icon: "SuccessFilled",
          permission: "container:read",
        },
        {
          path: "/containers/stopped",
          title: "Stopped",
          icon: "Warning",
          permission: "container:read",
        },
        {
          path: "/containers/all",
          title: "All Containers",
          icon: "Box",
          permission: "container:read",
        },
      ],
    },
    {
      path: "/images",
      title: "Images",
      icon: "Picture",
      permission: "image:read",
    },
    {
      path: "/updates",
      title: "Updates",
      icon: "Refresh",
      permission: "update:read",
    },
    {
      path: "/logs",
      title: "Logs",
      icon: "Document",
      permission: "log:read",
    },
    {
      path: "/monitoring",
      title: "Monitoring",
      icon: "Monitor",
      permission: "monitor:read",
      children: [
        {
          path: "/monitoring/metrics",
          title: "Metrics",
          icon: "DataAnalysis",
          permission: "monitor:read",
        },
        {
          path: "/monitoring/alerts",
          title: "Alerts",
          icon: "Bell",
          permission: "monitor:read",
        },
      ],
    },
  ];

  // Add admin-only items
  if (hasRole("admin")) {
    items.push(
      {
        path: "/users",
        title: "User Management",
        icon: "User",
        role: "admin",
      },
      {
        path: "/settings",
        title: "System Settings",
        icon: "Setting",
        role: "admin",
        children: [
          {
            path: "/settings/general",
            title: "General",
            icon: "Setting",
            role: "admin",
          },
          {
            path: "/settings/security",
            title: "Security",
            icon: "Shield",
            role: "admin",
          },
          {
            path: "/settings/api-keys",
            title: "API Keys",
            icon: "Key",
            role: "admin",
          },
        ],
      },
    );
  }

  // Filter items based on permissions
  return filterMenuItems(items);
});

// Methods
const filterMenuItems = (items: MenuItem[]): MenuItem[] => {
  return items.filter((item) => {
    // Check role requirement
    if (item.role && !hasRole(item.role)) {
      return false;
    }

    // Check permission requirement
    if (item.permission && !hasPermission(item.permission)) {
      return false;
    }

    // Filter children if they exist
    if (item.children) {
      item.children = filterMenuItems(item.children);
      // Hide parent if no children are visible
      return item.children.length > 0;
    }

    return true;
  });
};

const handleMenuClick = (item: MenuItem) => {
  if (item.disabled) {
    return;
  }

  // Add any custom click handling here
  // The router navigation is handled automatically by the el-menu
};

// Simulate dynamic badges (replace with actual data)
const updateBadges = () => {
  // This would typically come from your store or API
  const runningContainers = ref(12);
  const pendingUpdates = ref(3);
  const unreadAlerts = ref(5);

  // You can watch for changes and update badges accordingly
  watch([runningContainers, pendingUpdates, unreadAlerts], () => {
    // Update menu items with new badge values
    // This is just an example - implement based on your data structure
  });
};

// Initialize badges
updateBadges();
</script>

<style scoped lang="scss">
.app-sidebar {
  width: 260px;
  height: 100vh;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-light);
  display: flex;
  flex-direction: column;
  transition: width 0.3s ease;
  position: relative;
  z-index: 999;

  &.sidebar-collapsed {
    width: 64px;

    .sidebar-header .logo-text {
      opacity: 0;
    }

    .sidebar-footer .user-info {
      opacity: 0;
    }
  }
}

.sidebar-header {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  .logo-container {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;

    .logo-icon {
      color: var(--el-color-primary);
      flex-shrink: 0;
    }

    .logo-text {
      font-size: 18px;
      font-weight: 700;
      color: var(--el-text-color-primary);
      margin: 0;
      white-space: nowrap;
      overflow: hidden;
    }
  }
}

.sidebar-content {
  flex: 1;
  overflow: hidden;

  .sidebar-scrollbar {
    height: 100%;

    :deep(.el-scrollbar__view) {
      padding: 8px 0;
    }
  }

  .sidebar-menu {
    border: none;
    background: transparent;

    .el-menu-item,
    .el-sub-menu__title {
      height: 48px;
      line-height: 48px;
      margin: 2px 8px;
      border-radius: 6px;
      color: var(--el-text-color-primary);

      &:hover {
        background: var(--el-fill-color-light);
      }

      &.is-active {
        background: var(--el-color-primary-light-9);
        color: var(--el-color-primary);

        &::before {
          content: "";
          position: absolute;
          right: 8px;
          top: 50%;
          transform: translateY(-50%);
          width: 3px;
          height: 20px;
          background: var(--el-color-primary);
          border-radius: 2px;
        }
      }

      .el-icon {
        width: 20px;
        height: 20px;
        font-size: 18px;
        margin-right: 8px;
      }

      .menu-title {
        font-weight: 500;
      }

      .menu-badge {
        margin-left: auto;
      }
    }

    .el-sub-menu {
      .el-menu-item {
        height: 40px;
        line-height: 40px;
        margin: 1px 16px 1px 24px;
        padding-left: 32px !important;

        .el-icon {
          font-size: 16px;
        }
      }
    }
  }
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid var(--el-border-color-lighter);

  .user-card {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
    padding: 8px;
    border-radius: 6px;
    background: var(--el-fill-color-lighter);

    .user-info {
      flex: 1;
      min-width: 0;

      .user-name {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .user-role {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        text-transform: capitalize;
      }
    }
  }

  .toggle-button {
    width: 100%;
    justify-content: center;
    padding: 8px;
    border-radius: 6px;

    &:hover {
      background: var(--el-fill-color-light);
    }
  }
}

// Transitions
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

// Responsive design
@media (max-width: 1024px) {
  .app-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    z-index: 1001;
    transform: translateX(-100%);
    transition: transform 0.3s ease;

    &:not(.sidebar-collapsed) {
      transform: translateX(0);
      box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
    }
  }
}

// Dark mode adjustments
.dark {
  .sidebar-menu {
    .el-menu-item,
    .el-sub-menu__title {
      &.is-active {
        background: var(--el-color-primary-dark-2);
      }
    }
  }
}
</style>
