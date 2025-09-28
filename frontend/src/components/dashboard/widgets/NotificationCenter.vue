<template>
  <div class="notification-center-widget">
    <!-- Loading State -->
    <div v-if="isLoading && !hasNotifications" class="loading-container">
      <el-skeleton :rows="3" animated />
      <div class="loading-text">正在加载通知...</div>
    </div>

    <!-- Error State -->
    <div v-else-if="error && !hasNotifications" class="error-container">
      <el-icon :size="48" class="error-icon">
        <Bell />
      </el-icon>
      <h3 class="error-title">无法加载通知</h3>
      <p class="error-message">{{ error }}</p>
      <el-button @click="retryLoad" type="primary" size="small">
        <el-icon><Refresh /></el-icon>
        重试
      </el-button>
    </div>

    <!-- Main Content -->
    <div v-else class="notification-content">
      <!-- Notification Header -->
      <div class="notification-header">
        <div class="header-info">
          <h3 class="header-title">通知中心</h3>
          <p class="header-subtitle">{{ notificationSummary }}</p>
        </div>

        <div class="header-actions">
          <el-button-group size="small">
            <el-button @click="refreshNotifications" :loading="isRefreshing" type="primary">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-button @click="markAllAsRead" :disabled="unreadCount === 0" type="default">
              <el-icon><Check /></el-icon>
              全部已读
            </el-button>
          </el-button-group>
        </div>
      </div>

      <!-- Notification Filters -->
      <div class="notification-filters">
        <div class="filter-buttons">
          <el-button
            v-for="filter in notificationFilters"
            :key="filter.key"
            :type="activeFilter === filter.key ? 'primary' : 'default'"
            size="small"
            @click="setActiveFilter(filter.key)"
            :class="{ active: activeFilter === filter.key }"
          >
            <el-icon><component :is="filter.icon" /></el-icon>
            {{ filter.label }}
            <el-badge v-if="filter.count > 0" :value="filter.count" class="filter-badge" />
          </el-button>
        </div>

        <div class="filter-actions">
          <el-dropdown @command="handleBulkAction">
            <el-button size="small" type="text">
              批量操作 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="markRead">标记已读</el-dropdown-item>
                <el-dropdown-item command="delete">删除</el-dropdown-item>
                <el-dropdown-item command="archive">归档</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <!-- Notification List -->
      <div class="notification-list" ref="notificationListRef">
        <div
          v-for="notification in filteredNotifications"
          :key="notification.id"
          class="notification-item"
          :class="{
            unread: !notification.read,
            selected: selectedNotifications.has(notification.id),
            [notification.type]: true,
            [notification.priority]: true
          }"
          @click="handleNotificationClick(notification)"
        >
          <div class="notification-checkbox">
            <el-checkbox
              v-model="selectedNotifications"
              :label="notification.id"
              @click.stop
            />
          </div>

          <div class="notification-icon">
            <el-icon :size="20">
              <component :is="getNotificationIcon(notification.type)" />
            </el-icon>
          </div>

          <div class="notification-content">
            <div class="notification-header">
              <span class="notification-title">{{ notification.title }}</span>
              <div class="notification-meta">
                <el-tag
                  v-if="notification.priority !== 'normal'"
                  :type="getPriorityTagType(notification.priority)"
                  size="small"
                >
                  {{ getPriorityText(notification.priority) }}
                </el-tag>
                <span class="notification-time">{{ formatRelativeTime(notification.timestamp) }}</span>
              </div>
            </div>

            <div class="notification-message">{{ notification.message }}</div>

            <div v-if="notification.source" class="notification-source">
              <el-icon :size="12"><Monitor /></el-icon>
              <span>{{ notification.source }}</span>
            </div>

            <div v-if="notification.actions && notification.actions.length > 0" class="notification-actions">
              <el-button
                v-for="action in notification.actions"
                :key="action.key"
                :type="action.type || 'primary'"
                size="small"
                @click.stop="executeNotificationAction(notification.id, action)"
              >
                <el-icon v-if="action.icon"><component :is="action.icon" /></el-icon>
                {{ action.label }}
              </el-button>
            </div>
          </div>

          <div class="notification-status">
            <el-icon
              v-if="!notification.read"
              class="unread-indicator"
              :size="8"
            >
              <CircleCheckFilled />
            </el-icon>

            <el-dropdown @command="(cmd) => handleNotificationAction(cmd, notification)">
              <el-button size="small" type="text" class="more-btn">
                <el-icon><More /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item :command="`${notification.read ? 'unread' : 'read'}`">
                    {{ notification.read ? '标记未读' : '标记已读' }}
                  </el-dropdown-item>
                  <el-dropdown-item command="archive">归档</el-dropdown-item>
                  <el-dropdown-item command="delete" class="danger-item">删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <div v-if="filteredNotifications.length === 0" class="empty-state">
          <el-icon :size="48"><Bell /></el-icon>
          <h3>{{ getEmptyStateTitle() }}</h3>
          <p>{{ getEmptyStateMessage() }}</p>
        </div>

        <!-- Load More Button -->
        <div v-if="hasMoreNotifications" class="load-more">
          <el-button @click="loadMoreNotifications" :loading="isLoadingMore" type="text">
            加载更多通知
          </el-button>
        </div>
      </div>

      <!-- Recent Alerts Summary -->
      <div v-if="recentAlerts.length > 0" class="recent-alerts">
        <div class="section-header">
          <el-icon class="section-icon"><Warning /></el-icon>
          <span class="section-title">最近告警</span>
          <el-badge :value="recentAlerts.length" type="warning" />
        </div>

        <div class="alerts-summary">
          <div
            v-for="alert in recentAlerts"
            :key="alert.id"
            class="alert-item"
            :class="alert.severity"
          >
            <el-icon class="alert-icon" :size="14">
              <component :is="getAlertIcon(alert.severity)" />
            </el-icon>
            <div class="alert-content">
              <span class="alert-message">{{ alert.message }}</span>
              <span class="alert-time">{{ formatRelativeTime(alert.timestamp) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Notification Detail Dialog -->
    <el-dialog
      v-model="showNotificationDialog"
      :title="selectedNotification?.title"
      width="600px"
    >
      <div v-if="selectedNotification" class="notification-dialog">
        <div class="dialog-header">
          <div class="notification-type">
            <el-icon :size="24">
              <component :is="getNotificationIcon(selectedNotification.type)" />
            </el-icon>
            <div class="type-info">
              <span class="type-label">{{ getTypeText(selectedNotification.type) }}</span>
              <span class="notification-time">{{ formatFullTime(selectedNotification.timestamp) }}</span>
            </div>
          </div>
        </div>

        <div class="dialog-content">
          <div class="notification-message">{{ selectedNotification.message }}</div>

          <div v-if="selectedNotification.details" class="notification-details">
            <h4>详细信息</h4>
            <pre>{{ JSON.stringify(selectedNotification.details, null, 2) }}</pre>
          </div>

          <div v-if="selectedNotification.source" class="notification-source">
            <strong>来源:</strong> {{ selectedNotification.source }}
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="showNotificationDialog = false">关闭</el-button>
        <el-button
          v-if="selectedNotification && selectedNotification.actions"
          type="primary"
          @click="executeNotificationActions"
        >
          执行操作
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import {
  Bell,
  Refresh,
  Check,
  ArrowDown,
  More,
  CircleCheckFilled,
  Warning,
  InfoFilled,
  SuccessFilled,
  CircleCloseFilled,
  Monitor,
  Setting,
  DataBoard,
  Upload,
} from '@element-plus/icons-vue';

// Icons for dynamic components
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = {
  Warning,
  InfoFilled,
  SuccessFilled,
  CircleCloseFilled,
  Monitor,
  Setting,
  DataBoard,
  Upload,
  Bell,
  Check,
  Refresh,
};

// Props
interface Props {
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
  displayMode?: 'default' | 'compact' | 'detailed' | 'minimal';
}

const props = withDefaults(defineProps<Props>(), {
  displayMode: 'default',
});

// Emits
const emit = defineEmits<{
  'data-updated': [data: any];
  error: [error: any];
  loading: [loading: boolean];
}>();

// Types
interface NotificationAction {
  key: string;
  label: string;
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info';
  icon?: string;
  url?: string;
}

interface Notification {
  id: string;
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error' | 'system' | 'security' | 'update';
  priority: 'low' | 'normal' | 'high' | 'critical';
  timestamp: Date;
  read: boolean;
  source?: string;
  details?: any;
  actions?: NotificationAction[];
  archived?: boolean;
}

interface AlertSummary {
  id: string;
  message: string;
  severity: 'critical' | 'warning' | 'info';
  timestamp: Date;
}

// Reactive state
const isLoading = ref(false);
const isRefreshing = ref(false);
const isLoadingMore = ref(false);
const error = ref<string | null>(null);
const activeFilter = ref('all');
const showNotificationDialog = ref(false);
const selectedNotification = ref<Notification | null>(null);
const selectedNotifications = ref(new Set<string>());
const notificationListRef = ref<HTMLElement>();

// Data state
const notifications = ref<Notification[]>([]);
const recentAlerts = ref<AlertSummary[]>([]);
const currentPage = ref(1);
const totalNotifications = ref(0);
const pageSize = ref(20);

// Update intervals
const refreshInterval = ref<NodeJS.Timeout>();

// Computed properties
const hasNotifications = computed(() => notifications.value.length > 0);

const hasMoreNotifications = computed(() => {
  return notifications.value.length < totalNotifications.value;
});

const unreadCount = computed(() => {
  return notifications.value.filter(n => !n.read && !n.archived).length;
});

const notificationSummary = computed(() => {
  const total = notifications.value.length;
  const unread = unreadCount.value;

  if (total === 0) return '暂无通知';
  if (unread === 0) return `${total} 条通知，全部已读`;
  return `${total} 条通知，${unread} 条未读`;
});

const notificationFilters = computed(() => [
  { key: 'all', label: '全部', icon: 'Bell', count: notifications.value.length },
  { key: 'unread', label: '未读', icon: 'CircleCheckFilled', count: unreadCount.value },
  { key: 'error', label: '错误', icon: 'CircleCloseFilled', count: notifications.value.filter(n => n.type === 'error').length },
  { key: 'warning', label: '警告', icon: 'Warning', count: notifications.value.filter(n => n.type === 'warning').length },
  { key: 'system', label: '系统', icon: 'Setting', count: notifications.value.filter(n => n.type === 'system').length },
  { key: 'security', label: '安全', icon: 'DataBoard', count: notifications.value.filter(n => n.type === 'security').length },
]);

const filteredNotifications = computed(() => {
  let filtered = notifications.value.filter(n => !n.archived);

  switch (activeFilter.value) {
    case 'unread':
      filtered = filtered.filter(n => !n.read);
      break;
    case 'error':
      filtered = filtered.filter(n => n.type === 'error');
      break;
    case 'warning':
      filtered = filtered.filter(n => n.type === 'warning');
      break;
    case 'system':
      filtered = filtered.filter(n => n.type === 'system');
      break;
    case 'security':
      filtered = filtered.filter(n => n.type === 'security');
      break;
    // 'all' shows all non-archived notifications
  }

  return filtered.sort((a, b) => {
    // Sort by read status (unread first), then by timestamp (newest first)
    if (a.read !== b.read) return a.read ? 1 : -1;
    return b.timestamp.getTime() - a.timestamp.getTime();
  });
});

// Methods
const loadNotifications = async (page = 1, append = false) => {
  try {
    if (!append) {
      isLoading.value = true;
      emit('loading', true);
    } else {
      isLoadingMore.value = true;
    }

    error.value = null;

    // Import API modules dynamically
    const { updatesAPI } = await import('@/api/updates');

    const response = await updatesAPI.getNotifications(page, pageSize.value);

    const newNotifications = response.notifications.map((notif: any) => ({
      id: notif.id,
      title: notif.title,
      message: notif.message,
      type: notif.type,
      priority: notif.priority || 'normal',
      timestamp: new Date(notif.timestamp),
      read: notif.read || false,
      source: notif.source,
      details: notif.data,
      actions: notif.actions,
      archived: notif.archived || false,
    }));

    if (append) {
      notifications.value = [...notifications.value, ...newNotifications];
    } else {
      notifications.value = newNotifications;
    }

    totalNotifications.value = response.total;
    currentPage.value = page;

    // Also load recent alerts
    const { monitoringAPI } = await import('@/api/monitoring');
    const alertsData = await monitoringAPI.getActiveAlerts();

    recentAlerts.value = alertsData.slice(0, 3).map((alert: any) => ({
      id: alert.id,
      message: alert.message,
      severity: alert.severity,
      timestamp: new Date(alert.timestamp),
    }));

    emit('data-updated', {
      totalNotifications: totalNotifications.value,
      unreadCount: unreadCount.value,
      recentAlerts: recentAlerts.value.length,
    });

  } catch (err: any) {
    console.error('Failed to load notifications:', err);
    if (!append) {
      error.value = err.message || '加载通知失败';
      emit('error', err);
    }
  } finally {
    isLoading.value = false;
    isLoadingMore.value = false;
    emit('loading', false);
  }
};

const refreshNotifications = async () => {
  isRefreshing.value = true;
  try {
    await loadNotifications(1, false);
    ElMessage.success('通知已刷新');
  } finally {
    isRefreshing.value = false;
  }
};

const loadMoreNotifications = async () => {
  if (hasMoreNotifications.value) {
    await loadNotifications(currentPage.value + 1, true);
  }
};

const setActiveFilter = (filterKey: string) => {
  activeFilter.value = filterKey;
  selectedNotifications.value.clear();
};

const handleNotificationClick = (notification: Notification) => {
  if (!notification.read) {
    markAsRead(notification.id);
  }
  selectedNotification.value = notification;
  showNotificationDialog.value = true;
};

const markAsRead = async (notificationId: string) => {
  try {
    const { updatesAPI } = await import('@/api/updates');
    await updatesAPI.markNotificationRead(notificationId);

    // Update local state
    const notification = notifications.value.find(n => n.id === notificationId);
    if (notification) {
      notification.read = true;
    }
  } catch (err: any) {
    console.error('Failed to mark notification as read:', err);
  }
};

const markAllAsRead = async () => {
  try {
    const unreadIds = notifications.value
      .filter(n => !n.read && !n.archived)
      .map(n => n.id);

    if (unreadIds.length === 0) return;

    const { updatesAPI } = await import('@/api/updates');

    // Mark all as read via API
    await Promise.all(unreadIds.map(id => updatesAPI.markNotificationRead(id)));

    // Update local state
    notifications.value.forEach(notification => {
      if (unreadIds.includes(notification.id)) {
        notification.read = true;
      }
    });

    ElMessage.success(`已标记 ${unreadIds.length} 条通知为已读`);
  } catch (err: any) {
    console.error('Failed to mark all as read:', err);
    ElMessage.error('批量标记失败: ' + (err.message || '未知错误'));
  }
};

const handleBulkAction = async (command: string) => {
  const selectedIds = Array.from(selectedNotifications.value);
  if (selectedIds.length === 0) {
    ElMessage.warning('请先选择通知');
    return;
  }

  try {
    switch (command) {
      case 'markRead':
        await Promise.all(selectedIds.map(id => markAsRead(id)));
        ElMessage.success(`已标记 ${selectedIds.length} 条通知为已读`);
        break;
      case 'delete':
        await deleteNotifications(selectedIds);
        break;
      case 'archive':
        await archiveNotifications(selectedIds);
        break;
    }

    selectedNotifications.value.clear();
  } catch (err: any) {
    console.error('Bulk action failed:', err);
    ElMessage.error('批量操作失败: ' + (err.message || '未知错误'));
  }
};

const handleNotificationAction = async (command: string, notification: Notification) => {
  try {
    switch (command) {
      case 'read':
        await markAsRead(notification.id);
        break;
      case 'unread':
        await markAsUnread(notification.id);
        break;
      case 'archive':
        await archiveNotifications([notification.id]);
        break;
      case 'delete':
        await deleteNotifications([notification.id]);
        break;
    }
  } catch (err: any) {
    console.error('Notification action failed:', err);
    ElMessage.error('操作失败: ' + (err.message || '未知错误'));
  }
};

const markAsUnread = async (notificationId: string) => {
  // This would need to be implemented in the API
  const notification = notifications.value.find(n => n.id === notificationId);
  if (notification) {
    notification.read = false;
  }
  ElMessage.success('已标记为未读');
};

const archiveNotifications = async (notificationIds: string[]) => {
  notifications.value = notifications.value.filter(n => !notificationIds.includes(n.id));
  ElMessage.success(`已归档 ${notificationIds.length} 条通知`);
};

const deleteNotifications = async (notificationIds: string[]) => {
  await ElMessageBox.confirm(
    `确定要删除 ${notificationIds.length} 条通知吗？此操作不可撤销。`,
    '确认删除',
    { type: 'warning' }
  );

  notifications.value = notifications.value.filter(n => !notificationIds.includes(n.id));
  ElMessage.success(`已删除 ${notificationIds.length} 条通知`);
};

const executeNotificationAction = async (notificationId: string, action: NotificationAction) => {
  try {
    if (action.url) {
      window.open(action.url, '_blank');
    } else {
      // Execute custom action via API
      ElMessage.info(`执行操作: ${action.label}`);
    }

    // Mark as read after action
    await markAsRead(notificationId);
  } catch (err: any) {
    console.error('Failed to execute notification action:', err);
    ElMessage.error('执行操作失败: ' + (err.message || '未知错误'));
  }
};

const executeNotificationActions = () => {
  if (selectedNotification.value && selectedNotification.value.actions) {
    selectedNotification.value.actions.forEach(action => {
      executeNotificationAction(selectedNotification.value!.id, action);
    });
  }
  showNotificationDialog.value = false;
};

const retryLoad = () => {
  loadNotifications(1, false);
};

// Helper functions
const getNotificationIcon = (type: string): string => {
  switch (type) {
    case 'error':
      return 'CircleCloseFilled';
    case 'warning':
      return 'Warning';
    case 'success':
      return 'SuccessFilled';
    case 'info':
      return 'InfoFilled';
    case 'system':
      return 'Setting';
    case 'security':
      return 'DataBoard';
    case 'update':
      return 'Upload';
    default:
      return 'Bell';
  }
};

const getPriorityTagType = (priority: string) => {
  switch (priority) {
    case 'critical':
      return 'danger';
    case 'high':
      return 'warning';
    case 'low':
      return 'info';
    default:
      return 'primary';
  }
};

const getPriorityText = (priority: string): string => {
  switch (priority) {
    case 'critical':
      return '紧急';
    case 'high':
      return '高';
    case 'low':
      return '低';
    default:
      return '普通';
  }
};

const getTypeText = (type: string): string => {
  switch (type) {
    case 'error':
      return '错误';
    case 'warning':
      return '警告';
    case 'success':
      return '成功';
    case 'info':
      return '信息';
    case 'system':
      return '系统';
    case 'security':
      return '安全';
    case 'update':
      return '更新';
    default:
      return '通知';
  }
};

const getAlertIcon = (severity: string): string => {
  switch (severity) {
    case 'critical':
      return 'CircleCloseFilled';
    case 'warning':
      return 'Warning';
    case 'info':
      return 'InfoFilled';
    default:
      return 'InfoFilled';
  }
};

const getEmptyStateTitle = (): string => {
  switch (activeFilter.value) {
    case 'unread':
      return '没有未读通知';
    case 'error':
      return '没有错误通知';
    case 'warning':
      return '没有警告通知';
    default:
      return '暂无通知';
  }
};

const getEmptyStateMessage = (): string => {
  switch (activeFilter.value) {
    case 'unread':
      return '所有通知都已阅读完毕';
    case 'error':
      return '系统运行正常，没有错误';
    case 'warning':
      return '一切正常，没有警告';
    default:
      return '暂时没有收到任何通知';
  }
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  if (hours < 24) return `${hours}小时前`;
  if (days < 7) return `${days}天前`;
  return date.toLocaleDateString();
};

const formatFullTime = (date: Date): string => {
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
};

// Lifecycle hooks
onMounted(async () => {
  await loadNotifications(1, false);

  // Set up periodic refresh
  const refreshIntervalMs = props.widgetConfig?.refreshInterval || 60000; // 1 minute
  if (refreshIntervalMs > 0) {
    refreshInterval.value = setInterval(() => loadNotifications(1, false), refreshIntervalMs);
  }
});

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
  }
});
</script>

<style scoped lang="scss">
.notification-center-widget {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.loading-container,
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  gap: 16px;

  .loading-text {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .error-icon {
    color: var(--el-color-warning);
  }

  .error-title {
    color: var(--el-text-color-primary);
    margin: 0;
  }

  .error-message {
    color: var(--el-text-color-secondary);
    margin: 0;
    font-size: 14px;
  }
}

.notification-content {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 16px;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  .header-info {
    .header-title {
      margin: 0 0 4px 0;
      font-size: 18px;
      color: var(--el-text-color-primary);
    }

    .header-subtitle {
      margin: 0;
      font-size: 14px;
      color: var(--el-text-color-secondary);
    }
  }
}

.notification-filters {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;

  .filter-buttons {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;

    .el-button {
      position: relative;

      .filter-badge {
        position: absolute;
        top: -5px;
        right: -5px;
      }
    }
  }
}

.notification-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;

  .notification-item {
    display: flex;
    gap: 12px;
    padding: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.3s ease;

    &:hover {
      border-color: var(--el-color-primary);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    &.unread {
      background: var(--el-color-primary-light-9);
      border-left: 4px solid var(--el-color-primary);
    }

    &.selected {
      background: var(--el-color-primary-light-8);
    }

    &.error {
      border-left: 4px solid var(--el-color-danger);
    }

    &.warning {
      border-left: 4px solid var(--el-color-warning);
    }

    &.success {
      border-left: 4px solid var(--el-color-success);
    }

    &.critical {
      background: rgba(var(--el-color-danger-rgb), 0.05);
    }

    &.high {
      background: rgba(var(--el-color-warning-rgb), 0.05);
    }

    .notification-checkbox {
      flex-shrink: 0;
      margin-top: 2px;
    }

    .notification-icon {
      flex-shrink: 0;
      width: 40px;
      height: 40px;
      background: var(--el-fill-color-light);
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      color: var(--el-text-color-secondary);

      &.error {
        background: rgba(var(--el-color-danger-rgb), 0.1);
        color: var(--el-color-danger);
      }

      &.warning {
        background: rgba(var(--el-color-warning-rgb), 0.1);
        color: var(--el-color-warning);
      }

      &.success {
        background: rgba(var(--el-color-success-rgb), 0.1);
        color: var(--el-color-success);
      }
    }

    .notification-content {
      flex: 1;
      min-width: 0;

      .notification-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 8px;

        .notification-title {
          font-size: 14px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          line-height: 1.4;
        }

        .notification-meta {
          display: flex;
          align-items: center;
          gap: 8px;
          flex-shrink: 0;

          .notification-time {
            font-size: 12px;
            color: var(--el-text-color-placeholder);
          }
        }
      }

      .notification-message {
        font-size: 13px;
        color: var(--el-text-color-regular);
        line-height: 1.4;
        margin-bottom: 8px;
      }

      .notification-source {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: 11px;
        color: var(--el-text-color-secondary);
        margin-bottom: 8px;
      }

      .notification-actions {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;
      }
    }

    .notification-status {
      display: flex;
      align-items: flex-start;
      gap: 8px;
      flex-shrink: 0;

      .unread-indicator {
        color: var(--el-color-primary);
        margin-top: 8px;
      }

      .more-btn {
        opacity: 0;
        transition: opacity 0.3s ease;
      }
    }

    &:hover .more-btn {
      opacity: 1;
    }
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    text-align: center;
    color: var(--el-text-color-secondary);

    h3 {
      margin: 12px 0 8px 0;
      color: var(--el-text-color-primary);
    }

    p {
      margin: 0;
      font-size: 14px;
    }
  }

  .load-more {
    text-align: center;
    padding: 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;

  .section-icon {
    color: var(--el-color-primary);
    font-size: 16px;
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.recent-alerts {
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 16px;

  .alerts-summary {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .alert-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      background: var(--el-fill-color-extra-light);
      border-radius: 6px;

      &.critical .alert-icon {
        color: var(--el-color-danger);
      }

      &.warning .alert-icon {
        color: var(--el-color-warning);
      }

      &.info .alert-icon {
        color: var(--el-color-info);
      }

      .alert-content {
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 100%;

        .alert-message {
          font-size: 12px;
          color: var(--el-text-color-regular);
        }

        .alert-time {
          font-size: 11px;
          color: var(--el-text-color-placeholder);
        }
      }
    }
  }
}

.notification-dialog {
  .dialog-header {
    margin-bottom: 16px;

    .notification-type {
      display: flex;
      align-items: center;
      gap: 12px;

      .type-info {
        display: flex;
        flex-direction: column;
        gap: 4px;

        .type-label {
          font-size: 14px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        .notification-time {
          font-size: 12px;
          color: var(--el-text-color-secondary);
        }
      }
    }
  }

  .dialog-content {
    .notification-message {
      font-size: 14px;
      color: var(--el-text-color-regular);
      line-height: 1.6;
      margin-bottom: 16px;
    }

    .notification-details {
      margin-bottom: 16px;

      h4 {
        margin: 0 0 8px 0;
        font-size: 14px;
        color: var(--el-text-color-primary);
      }

      pre {
        background: var(--el-fill-color-light);
        padding: 12px;
        border-radius: 4px;
        font-size: 12px;
        line-height: 1.4;
        overflow-x: auto;
      }
    }

    .notification-source {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }
}

.danger-item {
  color: var(--el-color-danger);
}

// Responsive design
@media (max-width: 768px) {
  .notification-center-widget {
    padding: 12px;
  }

  .notification-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .notification-filters {
    flex-direction: column;
    align-items: flex-start;

    .filter-buttons {
      width: 100%;
      justify-content: flex-start;
    }
  }

  .notification-item {
    .notification-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 4px;
    }
  }
}
</style>