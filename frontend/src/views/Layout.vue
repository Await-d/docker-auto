<template>
  <div class="app-layout" :class="layoutClass">
    <!-- Sidebar -->
    <Sidebar />

    <!-- Mobile overlay -->
    <div
      v-if="isMobile && !sidebarCollapsed"
      class="sidebar-overlay"
      @click="setSidebarCollapsed(true)"
    />

    <!-- Main content area -->
    <div class="main-container">
      <!-- Header -->
      <Header />

      <!-- Page content -->
      <main class="page-content">
        <div class="content-wrapper">
          <!-- Page loading -->
          <Loading
            v-if="pageLoading"
            type="spinner"
            text="Loading..."
            overlay
          />

          <!-- Router view with transitions -->
          <router-view v-slot="{ Component, route }">
            <transition :name="getTransitionName(route)" mode="out-in">
              <keep-alive :include="keepAlivePages">
                <component :is="Component" :key="route.fullPath" />
              </keep-alive>
            </transition>
          </router-view>
        </div>
      </main>

      <!-- Footer -->
      <footer class="app-footer" v-if="showFooter">
        <div class="footer-content">
          <div class="footer-left">
            <span class="copyright">
              © {{ currentYear }} Docker Auto-Update System
            </span>
            <span class="version">v{{ appVersion }}</span>
          </div>

          <div class="footer-right">
            <el-link href="/docs" target="_blank" type="primary">
              Documentation
            </el-link>
            <el-divider direction="vertical" />
            <el-link href="/api" target="_blank" type="primary">
              API
            </el-link>
            <el-divider direction="vertical" />
            <el-link href="/support" target="_blank" type="primary">
              Support
            </el-link>
          </div>
        </div>
      </footer>
    </div>

    <!-- Global notifications -->
    <div class="notification-container">
      <transition-group name="notification" tag="div">
        <div
          v-for="notification in notifications"
          :key="notification.id"
          class="notification-item"
          :class="`notification-${notification.type}`"
        >
          <el-alert
            :title="notification.title"
            :description="notification.message"
            :type="notification.type"
            :closable="true"
            @close="removeNotification(notification.id)"
          />
        </div>
      </transition-group>
    </div>

    <!-- Back to top button -->
    <el-backtop
      :right="30"
      :bottom="30"
      :visibility-height="300"
    >
      <el-icon :size="20">
        <CaretTop />
      </el-icon>
    </el-backtop>

    <!-- Settings drawer -->
    <SettingsDrawer v-if="showSettingsDrawer" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { CaretTop } from '@element-plus/icons-vue'
import Header from '@/components/Header.vue'
import Sidebar from '@/components/Sidebar.vue'
import Loading from '@/components/Loading.vue'
import SettingsDrawer from '@/components/SettingsDrawer.vue'
import { useApp } from '@/store/app'
import { useAuth } from '@/store/auth'
import { useContainerWebSocket } from '@/services/containerWebSocket'

// Composables
const route = useRoute()
const {
  sidebarCollapsed,
  setSidebarCollapsed,
  pageLoading,
  isMobile,
  isDarkMode,
  notifications,
  removeNotification
} = useApp()
const { isAuthenticated } = useAuth()
const { subscribeToAll, isConnected, state } = useContainerWebSocket()

// Reactive state
const showSettingsDrawer = ref(false)
const appVersion = ref('1.0.0')
const currentYear = ref(new Date().getFullYear())

// Computed properties
const layoutClass = computed(() => ({
  'layout-mobile': isMobile.value,
  'layout-collapsed': sidebarCollapsed.value,
  'layout-dark': isDarkMode.value
}))

const showFooter = computed(() => {
  // Hide footer on certain pages
  const hideFooterPages = ['/login', '/register', '/forgot-password']
  return !hideFooterPages.includes(route.path) && isAuthenticated.value
})

const keepAlivePages = computed(() => {
  // Pages to keep alive for better performance
  return ['Dashboard', 'Containers', 'Images', 'Logs']
})

// Methods
const getTransitionName = (route: any) => {
  // Different transitions based on route depth or type
  if (route.meta?.transition) {
    return route.meta.transition
  }

  // Default transitions
  if (route.path === '/login') {
    return 'slide-up'
  }

  return 'fade'
}

const handleResize = () => {
  // Auto-collapse sidebar on mobile
  if (isMobile.value && !sidebarCollapsed.value) {
    setSidebarCollapsed(true)
  }
}

const handleKeyboardShortcuts = (event: KeyboardEvent) => {
  // Global keyboard shortcuts
  if (event.ctrlKey || event.metaKey) {
    switch (event.key) {
      case 'b':
        // Toggle sidebar
        event.preventDefault()
        setSidebarCollapsed(!sidebarCollapsed.value)
        break
      case 'k':
        // Global search (implement if needed)
        event.preventDefault()
        // Open search modal
        break
      case ',':
        // Open settings
        event.preventDefault()
        showSettingsDrawer.value = true
        break
    }
  }

  // ESC key handling
  if (event.key === 'Escape') {
    if (showSettingsDrawer.value) {
      showSettingsDrawer.value = false
    } else if (isMobile.value && !sidebarCollapsed.value) {
      setSidebarCollapsed(true)
    }
  }
}

// Watchers
watch(isMobile, (isMobile) => {
  if (isMobile && !sidebarCollapsed.value) {
    setSidebarCollapsed(true)
  }
})

// Lifecycle
onMounted(() => {
  window.addEventListener('resize', handleResize)
  window.addEventListener('keydown', handleKeyboardShortcuts)

  // Set initial sidebar state based on screen size
  handleResize()

  // Initialize WebSocket connections when authenticated
  if (isAuthenticated.value) {
    subscribeToAll()
  }
})

// Watch for authentication changes
watch(isAuthenticated, (authenticated) => {
  if (authenticated) {
    subscribeToAll()
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('keydown', handleKeyboardShortcuts)
})
</script>

<style scoped lang="scss">
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--el-bg-color-page);

  &.layout-mobile {
    .main-container {
      margin-left: 0;
    }
  }

  &.layout-collapsed {
    .main-container {
      margin-left: 64px;
    }
  }
}

.sidebar-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-left: 260px;
  transition: margin-left 0.3s ease;
  min-width: 0;

  @media (max-width: 1024px) {
    margin-left: 0;
  }
}

.page-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;

  .content-wrapper {
    flex: 1;
    overflow: auto;
    padding: 20px;
    position: relative;

    @media (max-width: 768px) {
      padding: 12px;
    }
  }
}

.app-footer {
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color-lighter);
  padding: 16px 20px;
  flex-shrink: 0;

  .footer-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    max-width: 100%;

    @media (max-width: 768px) {
      flex-direction: column;
      gap: 8px;
      text-align: center;
    }
  }

  .footer-left {
    display: flex;
    align-items: center;
    gap: 16px;

    .copyright {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }

    .version {
      font-size: 12px;
      color: var(--el-text-color-placeholder);
      background: var(--el-fill-color-light);
      padding: 2px 6px;
      border-radius: 3px;
    }
  }

  .footer-right {
    display: flex;
    align-items: center;
    gap: 8px;

    .el-link {
      font-size: 13px;
    }
  }
}

// Notifications
.notification-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 2000;
  max-width: 400px;

  @media (max-width: 768px) {
    top: 10px;
    right: 10px;
    left: 10px;
    max-width: none;
  }
}

.notification-item {
  margin-bottom: 12px;

  .el-alert {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    border-radius: 6px;

    .dark & {
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
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

.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.4s ease;
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
  transition: all 0.3s ease;
}

.slide-left-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

// Notification transitions
.notification-enter-active {
  transition: all 0.4s ease;
}

.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.notification-leave-to {
  opacity: 0;
  transform: translateX(100%) scale(0.9);
}

// Back to top button customization
:deep(.el-backtop) {
  background: var(--el-color-primary);
  color: white;
  border-radius: 50%;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);

  &:hover {
    background: var(--el-color-primary-light-3);
  }
}

// Scrollbar customization
:deep(.el-scrollbar__bar) {
  right: 2px;
  bottom: 2px;

  .el-scrollbar__thumb {
    background: var(--el-border-color-darker);
    border-radius: 4px;

    &:hover {
      background: var(--el-border-color-dark);
    }
  }
}

// Dark mode adjustments
.layout-dark {
  .sidebar-overlay {
    background: rgba(0, 0, 0, 0.7);
  }
}

// Loading overlay
:deep(.loading-overlay) {
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(2px);

  .dark & {
    background: rgba(0, 0, 0, 0.9);
  }
}

// Responsive design
@media (max-width: 1024px) {
  .app-layout.layout-mobile {
    .main-container {
      margin-left: 0;
    }
  }
}

@media (max-width: 768px) {
  .page-content .content-wrapper {
    padding: 12px;
  }

  .notification-container {
    top: 10px;
    right: 10px;
    left: 10px;
    max-width: none;
  }
}

// Print styles
@media print {
  .app-layout {
    .app-sidebar,
    .app-header,
    .app-footer,
    .notification-container,
    .el-backtop {
      display: none !important;
    }

    .main-container {
      margin-left: 0 !important;
    }

    .page-content .content-wrapper {
      padding: 0 !important;
      overflow: visible !important;
    }
  }
}
</style>