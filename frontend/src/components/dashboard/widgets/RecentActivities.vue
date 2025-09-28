<template>
  <div class="recent-activities-widget">
    <div class="widget-header">
      <div class="header-info">
        <el-icon class="header-icon">
          <Clock />
        </el-icon>
        <h3>最近活动</h3>
        <el-badge :value="unreadCount" :hidden="unreadCount === 0" class="activity-badge" />
      </div>
      <div class="header-actions">
        <el-tooltip content="刷新活动">
          <el-button
            :icon="RefreshLeft"
            :loading="isRefreshing"
            size="small"
            text
            @click="refreshActivities"
          />
        </el-tooltip>
        <el-tooltip content="标记全部已读">
          <el-button
            :icon="Check"
            size="small"
            text
            :disabled="unreadCount === 0"
            @click="markAllAsRead"
          />
        </el-tooltip>
        <el-dropdown @command="handleFilterCommand">
          <el-button :icon="Filter" size="small" text />
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="all" :class="{ active: currentFilter === 'all' }">
                全部活动
              </el-dropdown-item>
              <el-dropdown-item command="container" :class="{ active: currentFilter === 'container' }">
                容器活动
              </el-dropdown-item>
              <el-dropdown-item command="update" :class="{ active: currentFilter === 'update' }">
                更新活动
              </el-dropdown-item>
              <el-dropdown-item command="system" :class="{ active: currentFilter === 'system' }">
                系统活动
              </el-dropdown-item>
              <el-dropdown-item command="security" :class="{ active: currentFilter === 'security' }">
                安全活动
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <div class="widget-content">
      <el-scrollbar v-if="!loading && activities.length > 0" class="activities-scroll">
        <div class="activities-timeline">
          <div
            v-for="activity in filteredActivities"
            :key="activity.id"
            :class="[
              'activity-item',
              `activity-${activity.type}`,
              { 'unread': !activity.read }
            ]"
            @click="handleActivityClick(activity)"
          >
            <div class="activity-marker">
              <el-icon :class="getActivityIcon(activity.type)">
                <component :is="getActivityIcon(activity.type)" />
              </el-icon>
            </div>
            <div class="activity-content">
              <div class="activity-header">
                <span class="activity-title">{{ activity.title }}</span>
                <span class="activity-time">{{ formatTime(activity.timestamp) }}</span>
              </div>
              <div class="activity-description">{{ activity.description }}</div>
              <div v-if="activity.containerName" class="activity-meta">
                <el-tag size="small" type="info">{{ activity.containerName }}</el-tag>
                <el-tag v-if="activity.status" size="small" :type="getStatusType(activity.status)">
                  {{ activity.status }}
                </el-tag>
              </div>
            </div>
            <div class="activity-actions">
              <el-button
                v-if="!activity.read"
                :icon="Check"
                size="small"
                text
                @click.stop="markAsRead(activity.id)"
              />
              <el-dropdown @command="(cmd: string) => handleActivityAction(cmd, activity)" trigger="click">
                <el-button :icon="MoreFilled" size="small" text @click.stop />
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="details">查看详情</el-dropdown-item>
                    <el-dropdown-item v-if="activity.containerName" command="container">
                      查看容器
                    </el-dropdown-item>
                    <el-dropdown-item command="copy">复制信息</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </div>

        <div v-if="hasMore" class="load-more">
          <el-button
            :loading="loadingMore"
            @click="loadMoreActivities"
            text
          >
            加载更多
          </el-button>
        </div>
      </el-scrollbar>

      <div v-else-if="loading" class="loading-state">
        <div v-for="i in 5" :key="i" class="activity-skeleton">
          <el-skeleton animated>
            <template #template>
              <div class="skeleton-item">
                <el-skeleton-item variant="circle" style="width: 24px; height: 24px; margin-right: 12px;" />
                <div class="skeleton-content">
                  <el-skeleton-item variant="text" style="width: 60%; margin-bottom: 4px;" />
                  <el-skeleton-item variant="text" style="width: 80%;" />
                  <el-skeleton-item variant="text" style="width: 30%; margin-top: 4px;" />
                </div>
              </div>
            </template>
          </el-skeleton>
        </div>
      </div>

      <div v-else-if="error" class="error-state">
        <el-result icon="warning" title="加载失败" :sub-title="error">
          <template #extra>
            <el-button type="primary" @click="loadActivities">重试</el-button>
          </template>
        </el-result>
      </div>

      <div v-else class="empty-state">
        <el-empty description="暂无活动记录">
          <el-button type="primary" @click="refreshActivities">刷新</el-button>
        </el-empty>
      </div>
    </div>

    <!-- 活动详情对话框 -->
    <el-dialog
      v-model="showActivityDialog"
      title="活动详情"
      width="500px"
      :before-close="handleDialogClose"
    >
      <div v-if="selectedActivity" class="activity-details">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="标题">
            {{ selectedActivity.title }}
          </el-descriptions-item>
          <el-descriptions-item label="描述">
            {{ selectedActivity.description }}
          </el-descriptions-item>
          <el-descriptions-item label="类型">
            <el-tag :type="getActivityTypeColor(selectedActivity.type)">
              {{ getActivityTypeLabel(selectedActivity.type) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="时间">
            {{ formatFullTime(selectedActivity.timestamp) }}
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedActivity.containerName" label="容器">
            <el-tag type="info">{{ selectedActivity.containerName }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedActivity.status" label="状态">
            <el-tag :type="getStatusType(selectedActivity.status)">
              {{ selectedActivity.status }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedActivity.details" label="详细信息">
            <pre class="activity-details-text">{{ JSON.stringify(selectedActivity.details, null, 2) }}</pre>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      <template #footer>
        <el-button @click="showActivityDialog = false">关闭</el-button>
        <el-button v-if="selectedActivity?.containerName" type="primary" @click="viewContainer">
          查看容器
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import {
  Clock,
  RefreshLeft,
  Check,
  Filter,
  MoreFilled,
  Monitor,
  Upload,
  Setting,
  Warning,
  InfoFilled,
  SuccessFilled
} from '@element-plus/icons-vue';

// 定义活动类型接口
export interface Activity {
  id: string;
  title: string;
  description: string;
  type: 'container' | 'update' | 'system' | 'security' | 'backup' | 'network';
  timestamp: string;
  containerName?: string;
  containerId?: string;
  status?: 'success' | 'warning' | 'error' | 'info' | 'running';
  read: boolean;
  details?: any;
  userId?: string;
  ipAddress?: string;
}

defineProps<{
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
}>();

const router = useRouter();

// 响应式状态
const loading = ref(true);
const loadingMore = ref(false);
const isRefreshing = ref(false);
const error = ref('');
const activities = ref<Activity[]>([]);
const currentFilter = ref<string>('all');
const hasMore = ref(false);
const currentPage = ref(1);
const pageSize = 20;

// 对话框状态
const showActivityDialog = ref(false);
const selectedActivity = ref<Activity | null>(null);

// 实时更新
let refreshInterval: NodeJS.Timeout | null = null;
let wsConnection: WebSocket | null = null;

// 计算属性
const unreadCount = computed(() => {
  return activities.value.filter(activity => !activity.read).length;
});

const filteredActivities = computed(() => {
  if (currentFilter.value === 'all') {
    return activities.value;
  }
  return activities.value.filter(activity => activity.type === currentFilter.value);
});

// 加载活动数据
const loadActivities = async (reset = false) => {
  try {
    if (reset) {
      loading.value = true;
      currentPage.value = 1;
      error.value = '';
    }

    // 动态导入API
    const { updatesAPI } = await import('@/api/updates');

    const response = await updatesAPI.getActivityLog({
      page: currentPage.value,
      limit: pageSize,
      type: currentFilter.value === 'all' ? undefined : currentFilter.value
    });

    if (reset) {
      activities.value = response.activities;
    } else {
      activities.value.push(...response.activities);
    }

    hasMore.value = currentPage.value * pageSize < response.total;

  } catch (err: any) {
    console.error('Failed to load activities:', err);
    error.value = err.message || '加载活动记录失败';
    if (reset) {
      activities.value = [];
    }
  } finally {
    loading.value = false;
    loadingMore.value = false;
  }
};

// 加载更多活动
const loadMoreActivities = async () => {
  if (loadingMore.value || !hasMore.value) return;

  loadingMore.value = true;
  currentPage.value++;
  await loadActivities(false);
};

// 刷新活动
const refreshActivities = async () => {
  isRefreshing.value = true;
  try {
    await loadActivities(true);
  } finally {
    isRefreshing.value = false;
  }
};

// 标记活动为已读
const markAsRead = async (activityId: string) => {
  try {
    const { updatesAPI } = await import('@/api/updates');
    await updatesAPI.markActivityAsRead(activityId);

    const activity = activities.value.find(a => a.id === activityId);
    if (activity) {
      activity.read = true;
    }

    ElMessage.success('已标记为已读');
  } catch (err: any) {
    console.error('Failed to mark activity as read:', err);
    ElMessage.error('标记失败');
  }
};

// 标记全部为已读
const markAllAsRead = async () => {
  if (unreadCount.value === 0) return;

  try {
    const { updatesAPI } = await import('@/api/updates');
    await updatesAPI.markAllActivitiesAsRead();

    activities.value.forEach(activity => {
      activity.read = true;
    });

    ElMessage.success(`已标记 ${unreadCount.value} 条活动为已读`);
  } catch (err: any) {
    console.error('Failed to mark all activities as read:', err);
    ElMessage.error('批量标记失败');
  }
};

// 处理筛选命令
const handleFilterCommand = (command: string) => {
  currentFilter.value = command;
  loadActivities(true);
};

// 处理活动点击
const handleActivityClick = (activity: Activity) => {
  selectedActivity.value = activity;
  showActivityDialog.value = true;

  // 如果未读，自动标记为已读
  if (!activity.read) {
    markAsRead(activity.id);
  }
};

// 处理活动操作
const handleActivityAction = (command: string, activity: Activity) => {
  switch (command) {
    case 'details':
      handleActivityClick(activity);
      break;
    case 'container':
      if (activity.containerId) {
        router.push(`/containers/${activity.containerId}`);
      }
      break;
    case 'copy':
      copyActivityInfo(activity);
      break;
  }
};

// 复制活动信息
const copyActivityInfo = (activity: Activity) => {
  const info = `${activity.title}\n${activity.description}\n时间: ${formatFullTime(activity.timestamp)}`;
  navigator.clipboard.writeText(info).then(() => {
    ElMessage.success('已复制到剪贴板');
  }).catch(() => {
    ElMessage.error('复制失败');
  });
};

// 查看容器
const viewContainer = () => {
  if (selectedActivity.value?.containerId) {
    router.push(`/containers/${selectedActivity.value.containerId}`);
    showActivityDialog.value = false;
  }
};

// 关闭对话框
const handleDialogClose = (done: () => void) => {
  selectedActivity.value = null;
  done();
};

// 获取活动图标
const getActivityIcon = (type: string) => {
  const iconMap: Record<string, any> = {
    container: Monitor,
    update: Upload,
    system: Setting,
    security: Warning,
    backup: InfoFilled,
    network: InfoFilled
  };
  return iconMap[type] || InfoFilled;
};

// 获取活动类型颜色
const getActivityTypeColor = (type: string) => {
  const colorMap: Record<string, string> = {
    container: 'primary',
    update: 'success',
    system: 'info',
    security: 'danger',
    backup: 'warning',
    network: 'info'
  };
  return colorMap[type] || 'info';
};

// 获取活动类型标签
const getActivityTypeLabel = (type: string) => {
  const labelMap: Record<string, string> = {
    container: '容器活动',
    update: '更新活动',
    system: '系统活动',
    security: '安全活动',
    backup: '备份活动',
    network: '网络活动'
  };
  return labelMap[type] || '未知类型';
};

// 获取状态类型
const getStatusType = (status: string) => {
  const statusMap: Record<string, string> = {
    success: 'success',
    warning: 'warning',
    error: 'danger',
    info: 'info',
    running: 'primary'
  };
  return statusMap[status] || 'info';
};

// 格式化时间
const formatTime = (timestamp: string) => {
  const now = new Date();
  const time = new Date(timestamp);
  const diff = now.getTime() - time.getTime();

  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  if (hours < 24) return `${hours}小时前`;
  if (days < 7) return `${days}天前`;

  return time.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
};

// 格式化完整时间
const formatFullTime = (timestamp: string) => {
  return new Date(timestamp).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
};

// WebSocket实时更新
const setupWebSocket = () => {
  try {
    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
    wsConnection = new WebSocket(wsUrl);

    wsConnection.onopen = () => {
      console.log('Activity WebSocket connected');
    };

    wsConnection.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'activity' || data.type === 'container_event') {
          // 新活动或容器事件，刷新活动列表
          refreshActivities();
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    wsConnection.onerror = (error) => {
      console.error('Activity WebSocket error:', error);
    };

    wsConnection.onclose = () => {
      console.log('Activity WebSocket disconnected');
      // 5秒后重连
      setTimeout(setupWebSocket, 5000);
    };
  } catch (err) {
    console.error('Failed to setup WebSocket:', err);
  }
};

// 监听筛选变化
watch(currentFilter, () => {
  loadActivities(true);
});

// 生命周期
onMounted(async () => {
  await loadActivities(true);

  // 设置定时刷新（30秒）
  refreshInterval = setInterval(() => {
    if (!loading.value && !isRefreshing.value) {
      refreshActivities();
    }
  }, 30000);

  // 建立WebSocket连接
  setupWebSocket();
});

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }

  if (wsConnection) {
    wsConnection.close();
    wsConnection = null;
  }
});
</script>

<style scoped lang="scss">
.recent-activities-widget {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--el-bg-color);
  border-radius: 8px;
  overflow: hidden;
}

.widget-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color-page);

  .header-info {
    display: flex;
    align-items: center;
    gap: 8px;

    .header-icon {
      color: var(--el-color-primary);
    }

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .activity-badge {
      margin-left: 4px;
    }
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }
}

.widget-content {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.activities-scroll {
  height: 100%;

  :deep(.el-scrollbar__view) {
    padding: 0;
  }
}

.activities-timeline {
  padding: 8px 0;
}

.activity-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 20px;
  cursor: pointer;
  transition: all 0.2s ease;
  border-left: 3px solid transparent;
  position: relative;

  &:hover {
    background: var(--el-fill-color-lighter);
  }

  &.unread {
    background: var(--el-color-primary-light-9);
    border-left-color: var(--el-color-primary);

    &::before {
      content: '';
      position: absolute;
      left: 8px;
      top: 18px;
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: var(--el-color-primary);
    }
  }

  &.activity-container {
    border-left-color: var(--el-color-primary);
  }

  &.activity-update {
    border-left-color: var(--el-color-success);
  }

  &.activity-system {
    border-left-color: var(--el-color-info);
  }

  &.activity-security {
    border-left-color: var(--el-color-danger);
  }

  &.activity-backup {
    border-left-color: var(--el-color-warning);
  }

  .activity-marker {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: var(--el-fill-color);
    flex-shrink: 0;

    .el-icon {
      font-size: 14px;
      color: var(--el-text-color-regular);
    }
  }

  .activity-content {
    flex: 1;
    min-width: 0;

    .activity-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 4px;
      gap: 12px;

      .activity-title {
        font-weight: 500;
        color: var(--el-text-color-primary);
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .activity-time {
        font-size: 12px;
        color: var(--el-text-color-placeholder);
        flex-shrink: 0;
      }
    }

    .activity-description {
      font-size: 13px;
      color: var(--el-text-color-regular);
      line-height: 1.4;
      margin-bottom: 6px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .activity-meta {
      display: flex;
      align-items: center;
      gap: 6px;
      flex-wrap: wrap;

      .el-tag {
        height: 20px;
        line-height: 18px;
        font-size: 11px;
      }
    }
  }

  .activity-actions {
    display: flex;
    align-items: center;
    gap: 4px;
    opacity: 0;
    transition: opacity 0.2s ease;
    flex-shrink: 0;
  }

  &:hover .activity-actions {
    opacity: 1;
  }
}

.load-more {
  text-align: center;
  padding: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.loading-state {
  padding: 16px 20px;

  .activity-skeleton {
    margin-bottom: 16px;

    &:last-child {
      margin-bottom: 0;
    }

    .skeleton-item {
      display: flex;
      align-items: flex-start;
      gap: 12px;

      .skeleton-content {
        flex: 1;
      }
    }
  }
}

.error-state,
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 20px;
}

.activity-details {
  .activity-details-text {
    background: var(--el-fill-color-light);
    padding: 12px;
    border-radius: 4px;
    font-size: 12px;
    max-height: 200px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }
}

// Dropdown样式
:deep(.el-dropdown-menu__item.active) {
  color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
}

// 响应式设计
@media (max-width: 768px) {
  .widget-header {
    padding: 12px 16px 8px;

    .header-info h3 {
      font-size: 14px;
    }

    .header-actions {
      gap: 2px;
    }
  }

  .activity-item {
    padding: 10px 16px;
    gap: 10px;

    .activity-content {
      .activity-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 2px;

        .activity-time {
          align-self: flex-end;
        }
      }

      .activity-description {
        font-size: 12px;
      }
    }
  }
}</style>
