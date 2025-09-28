<template>
  <div class="update-activity-widget">
    <!-- Loading State -->
    <div v-if="isLoading && !hasData" class="loading-container">
      <el-skeleton :rows="4" animated />
      <div class="loading-text">正在加载更新活动...</div>
    </div>

    <!-- Error State -->
    <div v-else-if="error && !hasData" class="error-container">
      <el-icon :size="48" class="error-icon">
        <CircleCloseFilled />
      </el-icon>
      <h3 class="error-title">无法加载更新活动</h3>
      <p class="error-message">{{ error }}</p>
      <el-button @click="retryLoad" type="primary" size="small">
        <el-icon><Refresh /></el-icon>
        重试
      </el-button>
    </div>

    <!-- Main Content -->
    <div v-else class="update-content">
      <!-- Header Section -->
      <div class="update-header">
        <div class="header-main">
          <div class="update-summary">
            <div class="summary-item total">
              <span class="summary-value">{{ statistics.total }}</span>
              <span class="summary-label">总更新</span>
            </div>
            <div class="summary-item available">
              <span class="summary-value available">{{ statistics.available }}</span>
              <span class="summary-label">可用</span>
            </div>
            <div class="summary-item running" v-if="statistics.running > 0">
              <span class="summary-value running">{{ statistics.running }}</span>
              <span class="summary-label">进行中</span>
            </div>
          </div>

          <div class="header-actions">
            <el-button-group size="small">
              <el-button @click="checkForUpdates" :loading="isCheckingUpdates" type="primary">
                <el-icon><Refresh /></el-icon>
                检查更新
              </el-button>
              <el-button @click="viewAllUpdates" type="default">
                <el-icon><View /></el-icon>
                查看全部
              </el-button>
            </el-button-group>
          </div>
        </div>

        <div class="last-check">
          <el-icon><Clock /></el-icon>
          <span>最后检查: {{ formatLastChecked(lastChecked) }}</span>
        </div>
      </div>

      <!-- Running Updates Section -->
      <div v-if="runningUpdates.length > 0" class="running-updates">
        <div class="section-header">
          <el-icon class="section-icon"><Loading /></el-icon>
          <span class="section-title">正在进行的更新</span>
          <el-badge :value="runningUpdates.length" type="primary" />
        </div>

        <div class="running-list">
          <div
            v-for="update in runningUpdates"
            :key="update.id"
            class="running-item"
            :class="update.status"
          >
            <div class="update-container">
              <div class="container-info">
                <span class="container-name">{{ update.containerName }}</span>
                <span class="container-image">{{ update.fromVersion }} → {{ update.toVersion }}</span>
              </div>
              <div class="update-status">
                <el-progress
                  :percentage="update.progress"
                  :status="getProgressStatus(update.status)"
                  :show-text="false"
                  :stroke-width="4"
                />
                <span class="status-text">{{ getStatusText(update.status) }}</span>
              </div>
            </div>

            <div class="update-actions">
              <el-button size="small" type="text" @click="viewUpdateDetails(update.id)">
                详情
              </el-button>
              <el-button
                size="small"
                type="text"
                @click="cancelUpdate(update.id)"
                class="cancel-btn"
                v-if="canCancelUpdate(update.status)"
              >
                取消
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- Available Updates Section -->
      <div v-if="availableUpdates.length > 0" class="available-updates">
        <div class="section-header">
          <el-icon class="section-icon"><Download /></el-icon>
          <span class="section-title">可用更新</span>
          <el-badge :value="availableUpdates.length" type="warning" />
        </div>

        <div class="available-list">
          <div
            v-for="update in displayedAvailableUpdates"
            :key="update.id"
            class="available-item"
            :class="{
              'security-update': update.securityPatches && update.securityPatches.length > 0,
              'major-update': update.updateType === 'major'
            }"
          >
            <div class="update-info">
              <div class="update-header-row">
                <span class="container-name">{{ update.containerName }}</span>
                <div class="update-badges">
                  <el-tag v-if="update.securityPatches && update.securityPatches.length > 0" type="danger" size="small">
                    安全更新
                  </el-tag>
                  <el-tag v-if="update.updateType === 'major'" type="warning" size="small">
                    主版本
                  </el-tag>
                </div>
              </div>

              <div class="version-info">
                <span class="current-version">{{ update.currentVersion }}</span>
                <el-icon class="version-arrow"><ArrowRight /></el-icon>
                <span class="new-version">{{ update.availableVersion }}</span>
              </div>

              <div class="update-details">
                <span class="update-size">{{ formatSize(update.size) }}</span>
                <span class="update-date">{{ formatRelativeTime(update.releaseDate) }}</span>
              </div>
            </div>

            <div class="update-actions">
              <el-button size="small" @click="startUpdate(update.id)" type="primary">
                <el-icon><CaretRight /></el-icon>
                更新
              </el-button>
              <el-dropdown @command="(cmd) => handleUpdateAction(cmd, update)">
                <el-button size="small" type="text">
                  <el-icon><More /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="schedule">计划更新</el-dropdown-item>
                    <el-dropdown-item command="details">查看详情</el-dropdown-item>
                    <el-dropdown-item command="ignore">忽略此更新</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </div>

        <div v-if="availableUpdates.length > 3" class="show-more">
          <el-button size="small" type="text" @click="showAllAvailable = !showAllAvailable">
            {{ showAllAvailable ? '收起' : `查看全部 ${availableUpdates.length} 个更新` }}
            <el-icon><component :is="showAllAvailable ? 'ArrowUp' : 'ArrowDown'" /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- Recent Activity Section -->
      <div class="recent-activity">
        <div class="section-header">
          <el-icon class="section-icon"><List /></el-icon>
          <span class="section-title">最近活动</span>
        </div>

        <div class="activity-timeline">
          <div
            v-for="activity in recentActivity"
            :key="activity.id"
            class="activity-item"
            :class="activity.status"
          >
            <div class="activity-indicator">
              <div class="indicator-dot" />
            </div>

            <div class="activity-content">
              <div class="activity-header">
                <span class="activity-container">{{ activity.containerName }}</span>
                <span class="activity-action" :class="activity.status">
                  {{ getActivityActionText(activity.action, activity.status) }}
                </span>
              </div>

              <div class="activity-details">
                <span class="activity-version" v-if="activity.version">
                  {{ activity.version }}
                </span>
                <span class="activity-time">
                  {{ formatRelativeTime(activity.timestamp) }}
                </span>
              </div>
            </div>

            <div class="activity-status">
              <el-icon class="status-icon" :class="`status-${activity.status}`">
                <component :is="getActivityStatusIcon(activity.status)" />
              </el-icon>
            </div>
          </div>
        </div>
      </div>

      <!-- Quick Actions -->
      <div v-if="displayMode !== 'minimal'" class="quick-actions">
        <el-button-group size="small">
          <el-button @click="startBulkUpdate" :disabled="!hasAvailableUpdates">
            <el-icon><Upload /></el-icon>
            批量更新
          </el-button>
          <el-button @click="manageScheduledUpdates">
            <el-icon><Timer /></el-icon>
            计划任务
          </el-button>
          <el-button @click="openUpdateSettings">
            <el-icon><Setting /></el-icon>
            设置
          </el-button>
        </el-button-group>
      </div>
    </div>

    <!-- Update Progress Dialog -->
    <el-dialog
      v-model="showUpdateDialog"
      title="更新进度"
      width="600px"
      :close-on-click-modal="false"
    >
      <div v-if="selectedUpdate" class="update-dialog-content">
        <!-- Dialog content would be implemented here -->
        <div class="update-progress">
          <el-progress
            :percentage="selectedUpdate.progress"
            :status="getProgressStatus(selectedUpdate.status)"
          />
          <p class="progress-text">{{ selectedUpdate.currentStep }}</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import {
  CircleCloseFilled,
  Refresh,
  View,
  Clock,
  Loading,
  Download,
  ArrowRight,
  CaretRight,
  More,
  ArrowUp,
  ArrowDown,
  List,
  Upload,
  Timer,
  Setting,
  SuccessFilled,
  Warning,
  InfoFilled,
  QuestionFilled,
  WarningFilled,
} from '@element-plus/icons-vue';
import { updatesAPI } from '@/api/updates';
import type {
  ContainerUpdate,
  RunningUpdate,
  UpdateHistoryItem
} from '@/types/updates';

// Icons for dynamic components
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = {
  ArrowUp,
  ArrowDown,
  SuccessFilled,
  Warning,
  InfoFilled,
  QuestionFilled,
  WarningFilled,
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

// Reactive state
const isLoading = ref(false);
const isCheckingUpdates = ref(false);
const error = ref<string | null>(null);
const showAllAvailable = ref(false);
const showUpdateDialog = ref(false);
const selectedUpdate = ref<RunningUpdate | null>(null);

// Data state
const availableUpdates = ref<ContainerUpdate[]>([]);
const runningUpdates = ref<RunningUpdate[]>([]);
const recentActivity = ref<UpdateHistoryItem[]>([]);
const lastChecked = ref<Date | null>(null);

// Update intervals
const refreshInterval = ref<NodeJS.Timeout>();
const runningUpdatesInterval = ref<NodeJS.Timeout>();
const wsConnection = ref<WebSocket | null>(null);
const wsReconnectTimer = ref<NodeJS.Timeout>();

// Computed properties
const hasData = computed(() => {
  return availableUpdates.value.length > 0 ||
         runningUpdates.value.length > 0 ||
         recentActivity.value.length > 0;
});

const hasAvailableUpdates = computed(() => availableUpdates.value.length > 0);

const displayedAvailableUpdates = computed(() => {
  if (showAllAvailable.value || availableUpdates.value.length <= 3) {
    return availableUpdates.value;
  }
  return availableUpdates.value.slice(0, 3);
});

const statistics = computed(() => ({
  total: availableUpdates.value.length + recentActivity.value.length,
  available: availableUpdates.value.length,
  running: runningUpdates.value.length,
  security: availableUpdates.value.filter(u => u.hasSecurity).length,
}));

// Bulk update computed properties
const estimatedBulkDuration = computed(() => {
  const selectedUpdates = availableUpdates.value.filter(u =>
    selectedBulkUpdates.value.includes(u.id)
  );
  const totalMinutes = selectedUpdates.reduce((sum, update) =>
    sum + (update.estimatedDowntime || 300), 0
  ) / 60;

  if (bulkForm.value.strategy === 'sequential') {
    return `${Math.ceil(totalMinutes)}\u5206\u949f`;
  } else {
    const concurrent = Math.min(bulkForm.value.maxConcurrent, selectedUpdates.length);
    return `${Math.ceil(totalMinutes / concurrent)}\u5206\u949f`;
  }
});

const totalBulkSize = computed(() => {
  const selectedUpdates = availableUpdates.value.filter(u =>
    selectedBulkUpdates.value.includes(u.id)
  );
  const totalBytes = selectedUpdates.reduce((sum, update) =>
    sum + (update.size || 0), 0
  );
  return formatSize(totalBytes);
});

// Methods
const loadUpdateData = async () => {
  try {
    isLoading.value = true;
    emit('loading', true);
    error.value = null;

    // Load data in parallel
    const [updatesResponse, runningResponse, historyResponse] = await Promise.all([
      updatesAPI.checkUpdates(),
      updatesAPI.getRunningUpdates(),
      updatesAPI.getUpdateHistory(1, 10),
    ]);

    availableUpdates.value = updatesResponse.updates || [];
    runningUpdates.value = runningResponse || [];
    recentActivity.value = historyResponse.items || [];
    lastChecked.value = new Date(updatesResponse.lastChecked);

    emit('data-updated', {
      available: availableUpdates.value.length,
      running: runningUpdates.value.length,
      recent: recentActivity.value.length,
      statistics: statistics.value,
    });

  } catch (err: any) {
    console.error('Failed to load update data:', err);
    error.value = err.message || '加载更新数据失败';
    emit('error', err);
  } finally {
    isLoading.value = false;
    emit('loading', false);
  }
};

const checkForUpdates = async () => {
  try {
    isCheckingUpdates.value = true;

    const response = await updatesAPI.checkUpdates(undefined, true);

    availableUpdates.value = response.updates || [];
    lastChecked.value = new Date(response.lastChecked);

    ElMessage.success(`发现 ${response.updates.length} 个可用更新`);

    emit('data-updated', {
      available: availableUpdates.value.length,
      statistics: statistics.value,
    });

  } catch (err: any) {
    console.error('Failed to check for updates:', err);
    ElMessage.error('检查更新失败: ' + (err.message || '未知错误'));
  } finally {
    isCheckingUpdates.value = false;
  }
};

const startUpdate = async (updateId: string) => {
  try {
    const update = availableUpdates.value.find(u => u.id === updateId);
    if (!update) return;

    const confirmResult = await ElMessageBox.confirm(
      `确定要更新容器 "${update.containerName}" 吗？`,
      '确认更新',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info',
      }
    );

    if (confirmResult !== 'confirm') return;

    await updatesAPI.startUpdate(updateId, {
      strategy: 'rolling',
      rollbackOnFailure: true,
      notifyOnCompletion: true,
    });

    ElMessage.success('更新已开始');

    // Refresh running updates
    await loadRunningUpdates();

    // Remove from available updates
    availableUpdates.value = availableUpdates.value.filter(u => u.id !== updateId);

  } catch (err: any) {
    console.error('Failed to start update:', err);
    ElMessage.error('启动更新失败: ' + (err.message || '未知错误'));
  }
};

const cancelUpdate = async (updateId: string) => {
  try {
    await ElMessageBox.confirm('确定要取消此更新吗？', '确认取消', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    });

    await updatesAPI.cancelUpdate(updateId);
    ElMessage.success('更新已取消');

    await loadRunningUpdates();

  } catch (err: any) {
    if (err === 'cancel') return;
    console.error('Failed to cancel update:', err);
    ElMessage.error('取消更新失败: ' + (err.message || '未知错误'));
  }
};

const loadRunningUpdates = async () => {
  try {
    const response = await updatesAPI.getRunningUpdates();
    runningUpdates.value = response || [];
  } catch (err) {
    console.error('Failed to load running updates:', err);
  }
};

const handleUpdateAction = async (command: string, update: ContainerUpdate) => {
  switch (command) {
    case 'schedule':
      // Implement schedule update dialog
      ElMessage.info('计划更新功能开发中...');
      break;
    case 'details':
      viewUpdateDetails(update.id);
      break;
    case 'ignore':
      await ignoreUpdate(update.id);
      break;
  }
};

const ignoreUpdate = async (updateId: string) => {
  try {
    await updatesAPI.ignoreUpdate(updateId, '用户手动忽略');
    availableUpdates.value = availableUpdates.value.filter(u => u.id !== updateId);
    ElMessage.success('已忽略此更新');
  } catch (err: any) {
    ElMessage.error('忽略更新失败: ' + (err.message || '未知错误'));
  }
};

const viewUpdateDetails = (updateId: string) => {
  const update = runningUpdates.value.find(u => u.id === updateId) ||
                 availableUpdates.value.find(u => u.id === updateId);

  if (update) {
    selectedUpdate.value = update as RunningUpdate;
    showUpdateDialog.value = true;
  }
};

const viewAllUpdates = () => {
  // Navigate to updates page
  ElMessage.info('跳转到更新管理页面...');
};

const startBulkUpdate = () => {
  if (availableUpdates.value.length === 0) {
    ElMessage.warning('没有可用的更新');
    return;
  }
  resetBulkUpdateForm();
  showBulkUpdateDialog.value = true;
};

const manageScheduledUpdates = () => {
  // Navigate to scheduled updates
  ElMessage.info('跳转到计划任务页面...');
};

// Schedule update methods
const openScheduleUpdateDialog = (update: ContainerUpdate) => {
  scheduleForm.value = {
    updateId: update.id,
    updateInfo: update,
    scheduledAt: '',
    recurring: false,
    recurringPattern: '',
    customCron: '',
    notifyBeforeMinutes: 30,
    strategy: 'rolling',
    rollbackOnFailure: true,
  };
  showScheduleDialog.value = true;
};

const resetScheduleForm = () => {
  scheduleForm.value = {
    updateId: '',
    updateInfo: null,
    scheduledAt: '',
    recurring: false,
    recurringPattern: '',
    customCron: '',
    notifyBeforeMinutes: 30,
    strategy: 'rolling',
    rollbackOnFailure: true,
  };
  if (scheduleFormRef.value) {
    scheduleFormRef.value.resetFields();
  }
};

const onRecurringChange = (value: boolean) => {
  if (!value) {
    scheduleForm.value.recurringPattern = '';
    scheduleForm.value.customCron = '';
  }
};

const disabledDate = (date: Date) => {
  return date < new Date(Date.now() - 24 * 60 * 60 * 1000);
};

const isValidCron = (cron: string): boolean => {
  const cronRegex = /^(\*|[0-5]?\d)\s+(\*|[01]?\d|2[0-3])\s+(\*|[0-2]?\d|3[01])\s+(\*|[0-9]|1[0-2]|\w{3})\s+(\*|[0-6]|\w{3})$/;
  return cronRegex.test(cron.trim());
};

const confirmScheduleUpdate = async () => {
  try {
    if (!scheduleFormRef.value) return;

    const valid = await scheduleFormRef.value.validate();
    if (!valid) return;

    isScheduling.value = true;

    const scheduleRequest = {
      scheduledAt: scheduleForm.value.scheduledAt,
      recurring: scheduleForm.value.recurring,
      recurringPattern: scheduleForm.value.recurring
        ? (scheduleForm.value.recurringPattern === 'custom'
           ? scheduleForm.value.customCron
           : scheduleForm.value.recurringPattern)
        : undefined,
      notifyBefore: scheduleForm.value.notifyBeforeMinutes * 60 * 1000,
      strategy: scheduleForm.value.strategy,
      rollbackOnFailure: scheduleForm.value.rollbackOnFailure,
    };

    const scheduledUpdate = await updatesAPI.scheduleUpdate(
      scheduleForm.value.updateId,
      scheduleRequest
    );

    ElMessage.success(
      `成功计划更新 "${scheduleForm.value.updateInfo?.containerName}"，将于 ${new Date(scheduleRequest.scheduledAt).toLocaleString()} 执行`
    );

    // Remove from available updates if scheduled
    availableUpdates.value = availableUpdates.value.filter(u => u.id !== scheduleForm.value.updateId);

    showScheduleDialog.value = false;
    resetScheduleForm();

  } catch (err: any) {
    console.error('Failed to schedule update:', err);
    ElMessage.error('计划更新失败: ' + (err.message || '未知错误'));
  } finally {
    isScheduling.value = false;
  }
};

// Bulk update methods
const resetBulkUpdateForm = () => {
  bulkUpdateStep.value = 1;
  selectedBulkUpdates.value = [];
  bulkForm.value = {
    selectAll: false,
    indeterminate: false,
    strategy: 'sequential',
    maxConcurrent: 3,
    continueOnError: false,
    rollbackOnFailure: true,
    runTests: false,
    respectDependencies: true,
    dependencyStrategy: 'strict',
  };
  bulkProgress.value = {
    overall: 0,
    completed: 0,
    total: 0,
    status: 'idle',
    items: [],
  };
  currentBulkOperationId.value = null;
};

const handleSelectAll = (checked: boolean) => {
  bulkForm.value.selectAll = checked;
  bulkForm.value.indeterminate = false;

  if (checked) {
    selectedBulkUpdates.value = availableUpdates.value.map(u => u.id);
  } else {
    selectedBulkUpdates.value = [];
  }
};

const toggleBulkUpdate = (updateId: string, checked: boolean) => {
  if (checked) {
    if (!selectedBulkUpdates.value.includes(updateId)) {
      selectedBulkUpdates.value.push(updateId);
    }
  } else {
    selectedBulkUpdates.value = selectedBulkUpdates.value.filter(id => id !== updateId);
  }

  const total = availableUpdates.value.length;
  const selected = selectedBulkUpdates.value.length;

  bulkForm.value.selectAll = selected === total;
  bulkForm.value.indeterminate = selected > 0 && selected < total;
};

const nextBulkStep = () => {
  if (bulkUpdateStep.value < 3) {
    bulkUpdateStep.value++;
  }
};

const prevBulkStep = () => {
  if (bulkUpdateStep.value > 1) {
    bulkUpdateStep.value--;
  }
};

const startBulkUpdateOperation = async () => {
  try {
    if (!bulkFormRef.value) return;

    const valid = await bulkFormRef.value.validate();
    if (!valid) return;

    isBulkUpdating.value = true;

    const operation = {
      updateIds: selectedBulkUpdates.value,
      strategy: bulkForm.value.strategy,
      maxConcurrent: bulkForm.value.maxConcurrent,
      continueOnError: bulkForm.value.continueOnError,
      rollbackOnFailure: bulkForm.value.rollbackOnFailure,
      runTests: bulkForm.value.runTests,
      respectDependencies: bulkForm.value.respectDependencies,
      dependencyStrategy: bulkForm.value.dependencyStrategy,
    };

    const response = await updatesAPI.startBulkUpdate(operation);

    currentBulkOperationId.value = response.operationId;

    // Initialize progress tracking
    bulkProgress.value = {
      overall: 0,
      completed: 0,
      total: selectedBulkUpdates.value.length,
      status: 'running',
      items: selectedBulkUpdates.value.map(updateId => {
        const update = availableUpdates.value.find(u => u.id === updateId);
        return {
          updateId,
          containerName: update?.containerName || 'Unknown',
          status: 'queued',
          progress: 0,
        };
      }),
    };

    ElMessage.success(`已启动批量更新，正在处理 ${response.queuedUpdates} 个容器`);

    bulkUpdateStep.value = 3;

    // Start monitoring progress
    monitorBulkUpdateProgress();

    // Remove scheduled updates from available list
    availableUpdates.value = availableUpdates.value.filter(u =>
      !selectedBulkUpdates.value.includes(u.id)
    );

  } catch (err: any) {
    console.error('Failed to start bulk update:', err);
    ElMessage.error('批量更新启动失败: ' + (err.message || '未知错误'));
  } finally {
    isBulkUpdating.value = false;
  }
};

const monitorBulkUpdateProgress = () => {
  const progressInterval = setInterval(async () => {
    if (!currentBulkOperationId.value || bulkProgress.value.status !== 'running') {
      clearInterval(progressInterval);
      return;
    }

    try {
      // WebSocket will handle real-time progress updates. This is fallback polling.
      // Check if WebSocket is working, if not, use polling fallback\n      if (!wsConnection.value || wsConnection.value.readyState !== WebSocket.OPEN) {\n        console.log('WebSocket not available, using polling fallback for bulk updates');\n      }\n      \n      // Simulate progress for demo (remove in production)
      const running = bulkProgress.value.items.filter(item => item.status === 'running');
      const queued = bulkProgress.value.items.filter(item => item.status === 'queued');

      // Simulate some queued items starting
      if (queued.length > 0 && running.length < bulkForm.value.maxConcurrent) {
        const itemsToStart = Math.min(
          bulkForm.value.maxConcurrent - running.length,
          queued.length
        );

        for (let i = 0; i < itemsToStart; i++) {
          queued[i].status = 'running';
          queued[i].progress = 10;
        }
      }

      // Simulate progress for running items
      bulkProgress.value.items.forEach(item => {
        if (item.status === 'running') {
          item.progress += Math.random() * 20;
          if (item.progress >= 100) {
            item.status = Math.random() > 0.1 ? 'completed' : 'failed';
            item.progress = 100;
            if (item.status === 'failed') {
              item.error = '更新过程中发生错误';
            }
          }
        }
      });

      // Update overall progress
      const completed = bulkProgress.value.items.filter(item =>
        item.status === 'completed' || item.status === 'failed'
      ).length;

      bulkProgress.value.completed = completed;
      bulkProgress.value.overall = Math.round((completed / bulkProgress.value.total) * 100);

      // Check if all done
      if (completed === bulkProgress.value.total) {
        const failed = bulkProgress.value.items.filter(item => item.status === 'failed').length;
        bulkProgress.value.status = failed > 0 ? 'failed' : 'completed';

        if (failed === 0) {
          ElMessage.success('批量更新已完成');
        } else {
          ElMessage.warning(`批量更新完成，${failed} 个容器更新失败`);
        }

        clearInterval(progressInterval);
      }

    } catch (err) {
      console.error('Failed to monitor bulk update progress:', err);
      clearInterval(progressInterval);
    }
  }, 10000); // Longer interval since WebSocket handles real-time updates
};

const cancelBulkUpdate = async () => {
  try {
    if (!currentBulkOperationId.value) return;

    await ElMessageBox.confirm(
      '确定要取消正在进行的批量更新吗？已完成的更新不会回滚。',
      '确认取消',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    // In a real implementation, this would call an API to cancel the bulk operation
    bulkProgress.value.status = 'failed';
    bulkProgress.value.items.forEach(item => {
      if (item.status === 'running' || item.status === 'queued') {
        item.status = 'cancelled';
      }
    });

    ElMessage.info('批量更新已取消');

  } catch (err: any) {
    if (err !== 'cancel') {
      console.error('Failed to cancel bulk update:', err);
      ElMessage.error('取消批量更新失败');
    }
  }
};

const closeBulkUpdateDialog = () => {
  showBulkUpdateDialog.value = false;
  resetBulkUpdateForm();
  // Refresh the data to get latest updates
  loadUpdateData();
};

const getProgressTagType = (status: string): string => {
  switch (status) {
    case 'completed': return 'success';
    case 'failed': return 'danger';
    case 'running': return 'primary';
    case 'cancelled': return 'warning';
    default: return 'info';
  }
};

// WebSocket methods for real-time updates
const initializeWebSocket = () => {
  try {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;

    wsConnection.value = new WebSocket(wsUrl);

    wsConnection.value.onopen = () => {
      console.log('WebSocket connected for update monitoring');
      // Subscribe to update events
      wsConnection.value?.send(JSON.stringify({
        type: 'subscribe',
        channel: 'updates',
        events: ['update_progress', 'update_completed', 'update_failed', 'update_available']
      }));
    };

    wsConnection.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    wsConnection.value.onclose = (event) => {
      console.log('WebSocket connection closed:', event.code, event.reason);
      // Attempt to reconnect after a delay
      if (!event.wasClean) {
        wsReconnectTimer.value = setTimeout(() => {
          initializeWebSocket();
        }, 5000);
      }
    };

    wsConnection.value.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

  } catch (err) {
    console.error('Failed to initialize WebSocket:', err);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'update_progress':
      handleUpdateProgress(message.data);
      break;
    case 'update_completed':
      handleUpdateCompleted(message.data);
      break;
    case 'update_failed':
      handleUpdateFailed(message.data);
      break;
    case 'update_available':
      handleNewUpdateAvailable(message.data);
      break;
    case 'bulk_update_progress':
      handleBulkUpdateProgress(message.data);
      break;
    default:
      console.log('Unknown WebSocket message type:', message.type);
  }
};

const handleUpdateProgress = (data: any) => {
  // Update running update progress
  const runningUpdate = runningUpdates.value.find(u => u.id === data.updateId);
  if (runningUpdate) {
    runningUpdate.progress = data.progress;
    runningUpdate.currentStep = data.currentStep;
    runningUpdate.status = data.status;
    runningUpdate.remainingTime = data.remainingTime;
  }

  // Update bulk update progress if applicable
  if (currentBulkOperationId.value && bulkProgress.value.status === 'running') {
    const bulkItem = bulkProgress.value.items.find(item => item.updateId === data.updateId);
    if (bulkItem) {
      bulkItem.progress = data.progress;
      bulkItem.status = data.status;

      // Recalculate overall progress
      const completed = bulkProgress.value.items.filter(item =>
        item.status === 'completed' || item.status === 'failed'
      ).length;
      bulkProgress.value.completed = completed;
      bulkProgress.value.overall = Math.round((completed / bulkProgress.value.total) * 100);
    }
  }
};

const handleUpdateCompleted = (data: any) => {
  // Remove from running updates
  runningUpdates.value = runningUpdates.value.filter(u => u.id !== data.updateId);

  // Add to recent activity
  recentActivity.value.unshift({
    id: data.historyId,
    containerId: data.containerId,
    containerName: data.containerName,
    imageName: data.imageName,
    fromVersion: data.fromVersion,
    toVersion: data.toVersion,
    updateType: data.updateType,
    status: 'completed',
    action: 'update_completed',
    timestamp: new Date(data.timestamp),
    version: data.toVersion
  });

  // Limit recent activity to 10 items
  if (recentActivity.value.length > 10) {
    recentActivity.value = recentActivity.value.slice(0, 10);
  }

  // Update bulk progress if applicable
  if (currentBulkOperationId.value && bulkProgress.value.status === 'running') {
    const bulkItem = bulkProgress.value.items.find(item => item.updateId === data.updateId);
    if (bulkItem) {
      bulkItem.status = 'completed';
      bulkItem.progress = 100;

      const completed = bulkProgress.value.items.filter(item =>
        item.status === 'completed' || item.status === 'failed'
      ).length;
      bulkProgress.value.completed = completed;
      bulkProgress.value.overall = Math.round((completed / bulkProgress.value.total) * 100);

      // Check if all bulk updates are done
      if (completed === bulkProgress.value.total) {
        const failed = bulkProgress.value.items.filter(item => item.status === 'failed').length;
        bulkProgress.value.status = failed > 0 ? 'failed' : 'completed';

        if (failed === 0) {
          ElMessage.success('批量更新已完成');
        } else {
          ElMessage.warning(`批量更新完成，${failed} 个容器更新失败`);
        }
      }
    }
  }

  ElMessage.success(`容器 "${data.containerName}" 更新完成`);
};

const handleUpdateFailed = (data: any) => {
  // Update running updates
  const runningUpdate = runningUpdates.value.find(u => u.id === data.updateId);
  if (runningUpdate) {
    runningUpdate.status = 'failed';
  }

  // Add to recent activity
  recentActivity.value.unshift({
    id: data.historyId,
    containerId: data.containerId,
    containerName: data.containerName,
    imageName: data.imageName,
    fromVersion: data.fromVersion,
    toVersion: data.toVersion,
    updateType: data.updateType,
    status: 'failed',
    action: 'update_failed',
    timestamp: new Date(data.timestamp),
    version: data.toVersion
  });

  // Update bulk progress if applicable
  if (currentBulkOperationId.value && bulkProgress.value.status === 'running') {
    const bulkItem = bulkProgress.value.items.find(item => item.updateId === data.updateId);
    if (bulkItem) {
      bulkItem.status = 'failed';
      bulkItem.error = data.error;

      const completed = bulkProgress.value.items.filter(item =>
        item.status === 'completed' || item.status === 'failed'
      ).length;
      bulkProgress.value.completed = completed;
      bulkProgress.value.overall = Math.round((completed / bulkProgress.value.total) * 100);
    }
  }

  ElMessage.error(`容器 "${data.containerName}" 更新失败: ${data.error}`);
};

const handleNewUpdateAvailable = (data: any) => {
  // Add new available update if not already present
  const exists = availableUpdates.value.find(u => u.id === data.id);
  if (!exists) {
    availableUpdates.value.push(data);
    ElMessage.info(`发现新的可用更新: ${data.containerName}`);
  }
};

const handleBulkUpdateProgress = (data: any) => {
  if (data.operationId === currentBulkOperationId.value) {
    bulkProgress.value.overall = data.overallProgress;
    bulkProgress.value.completed = data.completedCount;
    bulkProgress.value.status = data.status;

    // Update individual items
    data.items?.forEach((item: any) => {
      const bulkItem = bulkProgress.value.items.find(bi => bi.updateId === item.updateId);
      if (bulkItem) {
        bulkItem.status = item.status;
        bulkItem.progress = item.progress;
        if (item.error) {
          bulkItem.error = item.error;
        }
      }
    });
  }
};

const closeWebSocket = () => {
  if (wsReconnectTimer.value) {
    clearTimeout(wsReconnectTimer.value);
  }

  if (wsConnection.value) {
    wsConnection.value.close();
    wsConnection.value = null;
  }
};

// Validation rules
const scheduleRules = {
  scheduledAt: [
    { required: true, message: '请选择执行时间', trigger: 'change' },
    {
      validator: (rule: any, value: any, callback: any) => {
        if (value && new Date(value) <= new Date()) {
          callback(new Error('执行时间不能早于当前时间'));
        } else {
          callback();
        }
      },
      trigger: 'change',
    },
  ],
  recurringPattern: [
    {
      validator: (rule: any, value: any, callback: any) => {
        if (scheduleForm.value.recurring && !value) {
          callback(new Error('启用重复时必须选择重复频率'));
        } else {
          callback();
        }
      },
      trigger: 'change',
    },
  ],
  customCron: [
    {
      validator: (rule: any, value: any, callback: any) => {
        if (scheduleForm.value.recurringPattern === 'custom') {
          if (!value) {
            callback(new Error('请输入Cron表达式'));
          } else if (!isValidCron(value)) {
            callback(new Error('Cron表达式格式不正确'));
          }
        }
        callback();
      },
      trigger: 'blur',
    },
  ],
};

const bulkFormRules = {
  strategy: [
    { required: true, message: '请选择执行策略', trigger: 'change' },
  ],
  maxConcurrent: [
    {
      validator: (rule: any, value: any, callback: any) => {
        if (bulkForm.value.strategy !== 'sequential') {
          if (!value || value < 1 || value > 10) {
            callback(new Error('最大并发数必须在1-10之间'));
          }
        }
        callback();
      },
      trigger: 'change',
    },
  ],
};

const openUpdateSettings = () => {
  // Navigate to settings
  ElMessage.info('跳转到更新设置页面...');
};

const retryLoad = () => {
  loadUpdateData();
};

// Helper functions
const canCancelUpdate = (status: string): boolean => {
  return ['queued', 'running', 'downloading'].includes(status);
};

const getProgressStatus = (status: string) => {
  switch (status) {
    case 'success':
    case 'completed':
      return 'success';
    case 'failed':
    case 'error':
      return 'exception';
    case 'cancelled':
      return 'warning';
    default:
      return undefined;
  }
};

const getStatusText = (status: string): string => {
  const statusMap: Record<string, string> = {
    queued: '排队中',
    downloading: '下载中',
    running: '更新中',
    testing: '测试中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
  };
  return statusMap[status] || status;
};

const getActivityActionText = (action: string, status: string): string => {
  const actionMap: Record<string, string> = {
    update_started: '开始更新',
    update_completed: '更新完成',
    update_failed: '更新失败',
    update_cancelled: '取消更新',
    rollback_completed: '回滚完成',
  };
  return actionMap[action] || action;
};

const getActivityStatusIcon = (status: string): string => {
  switch (status) {
    case 'success':
    case 'completed':
      return 'SuccessFilled';
    case 'failed':
    case 'error':
      return 'CircleCloseFilled';
    case 'warning':
    case 'cancelled':
      return 'Warning';
    default:
      return 'InfoFilled';
  }
};

const formatLastChecked = (date: Date | null): string => {
  if (!date) return '从未';
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  return date.toLocaleDateString();
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return date.toLocaleDateString();
};

const formatSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
};

// Lifecycle hooks
onMounted(async () => {
  await loadUpdateData();

  // Initialize WebSocket for real-time updates
  initializeWebSocket();

  // Set up periodic refresh for running updates
  runningUpdatesInterval.value = setInterval(loadRunningUpdates, 10000);

  // Set up periodic refresh for all data
  const refreshIntervalMs = props.widgetConfig?.refreshInterval || 60000;
  if (refreshIntervalMs > 0) {
    refreshInterval.value = setInterval(loadUpdateData, refreshIntervalMs);
  }
});

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
  }
  if (runningUpdatesInterval.value) {
    clearInterval(runningUpdatesInterval.value);
  }

  // Close WebSocket connection
  closeWebSocket();
});

// Watch for widget data changes
watch(
  () => props.widgetData,
  (newData) => {
    if (newData) {
      if (newData.availableUpdates) {
        availableUpdates.value = newData.availableUpdates;
      }
      if (newData.runningUpdates) {
        runningUpdates.value = newData.runningUpdates;
      }
    }
  },
  { deep: true }
);
</script>

<style scoped lang="scss">
.update-activity-widget {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
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
    color: var(--el-color-danger);
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

.update-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  height: 100%;
}

.update-header {
  .header-main {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 12px;
  }

  .update-summary {
    display: flex;
    gap: 24px;

    .summary-item {
      text-align: center;

      .summary-value {
        display: block;
        font-size: 24px;
        font-weight: 700;
        line-height: 1;

        &.available {
          color: var(--el-color-warning);
        }

        &.running {
          color: var(--el-color-primary);
        }
      }

      .summary-label {
        display: block;
        font-size: 12px;
        color: var(--el-text-color-secondary);
        margin-top: 4px;
      }
    }
  }

  .last-check {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
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

.running-updates {
  .running-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .running-item {
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      padding: 12px;
      display: flex;
      justify-content: space-between;
      align-items: center;

      .update-container {
        flex: 1;
        min-width: 0;

        .container-info {
          .container-name {
            display: block;
            font-size: 14px;
            font-weight: 600;
            color: var(--el-text-color-primary);
            margin-bottom: 4px;
          }

          .container-image {
            font-size: 12px;
            color: var(--el-text-color-secondary);
            font-family: monospace;
          }
        }

        .update-status {
          margin-top: 8px;
          display: flex;
          align-items: center;
          gap: 8px;

          .el-progress {
            flex: 1;
          }

          .status-text {
            font-size: 11px;
            color: var(--el-text-color-secondary);
            white-space: nowrap;
          }
        }
      }

      .update-actions {
        display: flex;
        gap: 8px;
        flex-shrink: 0;

        .cancel-btn {
          color: var(--el-color-danger);
        }
      }
    }
  }
}

.available-updates {
  .available-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .available-item {
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      padding: 12px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      transition: all 0.3s ease;

      &:hover {
        border-color: var(--el-color-primary);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      }

      &.security-update {
        border-color: var(--el-color-danger);
        background: rgba(var(--el-color-danger-rgb), 0.05);
      }

      &.major-update {
        border-color: var(--el-color-warning);
      }

      .update-info {
        flex: 1;
        min-width: 0;

        .update-header-row {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 8px;

          .container-name {
            font-size: 14px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          .update-badges {
            display: flex;
            gap: 4px;
          }
        }

        .version-info {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 6px;
          font-family: monospace;
          font-size: 12px;

          .current-version {
            color: var(--el-text-color-secondary);
          }

          .version-arrow {
            color: var(--el-text-color-placeholder);
          }

          .new-version {
            color: var(--el-color-primary);
            font-weight: 600;
          }
        }

        .update-details {
          display: flex;
          gap: 16px;
          font-size: 11px;
          color: var(--el-text-color-secondary);

          .update-size {
            display: flex;
            align-items: center;
            gap: 4px;
          }
        }
      }

      .update-actions {
        display: flex;
        gap: 8px;
        flex-shrink: 0;
      }
    }
  }

  .show-more {
    text-align: center;
    margin-top: 8px;
  }
}

.recent-activity {
  .activity-timeline {
    position: relative;

    .activity-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 8px 0;
      position: relative;

      &:not(:last-child)::after {
        content: '';
        position: absolute;
        left: 6px;
        top: 32px;
        width: 1px;
        height: 20px;
        background: var(--el-border-color-light);
      }

      .activity-indicator {
        flex-shrink: 0;

        .indicator-dot {
          width: 12px;
          height: 12px;
          border-radius: 50%;
          background: var(--el-color-info);
        }
      }

      &.success .indicator-dot {
        background: var(--el-color-success);
      }

      &.failed .indicator-dot {
        background: var(--el-color-danger);
      }

      &.warning .indicator-dot {
        background: var(--el-color-warning);
      }

      .activity-content {
        flex: 1;
        min-width: 0;

        .activity-header {
          display: flex;
          gap: 8px;
          align-items: baseline;
          margin-bottom: 2px;

          .activity-container {
            font-size: 13px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          .activity-action {
            font-size: 12px;
            color: var(--el-text-color-secondary);

            &.success {
              color: var(--el-color-success);
            }

            &.failed {
              color: var(--el-color-danger);
            }

            &.warning {
              color: var(--el-color-warning);
            }
          }
        }

        .activity-details {
          display: flex;
          gap: 12px;
          font-size: 11px;
          color: var(--el-text-color-secondary);

          .activity-version {
            font-family: monospace;
          }
        }
      }

      .activity-status {
        flex-shrink: 0;

        .status-icon {
          font-size: 16px;

          &.status-success {
            color: var(--el-color-success);
          }

          &.status-failed {
            color: var(--el-color-danger);
          }

          &.status-warning {
            color: var(--el-color-warning);
          }

          &.status-info {
            color: var(--el-color-info);
          }
        }
      }
    }
  }
}

.quick-actions {
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
  display: flex;
  justify-content: center;
}

.update-dialog-content {
  .update-progress {
    .progress-text {
      margin-top: 12px;
      text-align: center;
      color: var(--el-text-color-secondary);
      font-size: 14px;
    }
  }
}

// Schedule Update Dialog Styles
.schedule-update-info {
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;

  .container-info {
    .container-name {
      display: block;
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin-bottom: 6px;
    }

    .version-info {
      display: flex;
      align-items: center;
      gap: 8px;
      font-family: monospace;
      font-size: 12px;

      .current-version {
        color: var(--el-text-color-secondary);
      }

      .version-arrow {
        color: var(--el-text-color-placeholder);
      }

      .new-version {
        color: var(--el-color-primary);
        font-weight: 600;
      }
    }
  }
}

.cron-help {
  h4 {
    margin-top: 0;
    color: var(--el-text-color-primary);
  }

  p {
    margin-bottom: 16px;
    color: var(--el-text-color-regular);

    code {
      background: var(--el-fill-color-light);
      padding: 2px 6px;
      border-radius: 3px;
      font-family: monospace;
    }
  }

  .cron-tips {
    margin-top: 16px;

    h5 {
      margin: 8px 0;
      color: var(--el-text-color-primary);
    }

    ul {
      margin: 0;
      padding-left: 20px;

      li {
        margin: 4px 0;
        color: var(--el-text-color-regular);

        code {
          background: var(--el-fill-color-light);
          padding: 1px 4px;
          border-radius: 2px;
          font-family: monospace;
        }
      }
    }
  }
}

// Bulk Update Dialog Styles
.bulk-update-content {
  .bulk-step {
    h3 {
      margin: 0 0 16px 0;
      color: var(--el-text-color-primary);
      font-size: 16px;
      font-weight: 600;
    }
  }

  .bulk-updates-list {
    .update-items {
      max-height: 300px;
      overflow-y: auto;
      margin-top: 12px;

      .bulk-update-item {
        padding: 8px 0;
        border-bottom: 1px solid var(--el-border-color-lighter);

        &:last-child {
          border-bottom: none;
        }

        .update-item-content {
          width: 100%;
          margin-left: 8px;

          .container-info {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 4px;

            .container-name {
              font-size: 14px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }

            .update-badges {
              display: flex;
              gap: 4px;
            }
          }

          .version-info {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 4px;
            font-family: monospace;
            font-size: 12px;

            .version-from {
              color: var(--el-text-color-secondary);
            }

            .version-to {
              color: var(--el-color-primary);
              font-weight: 600;
            }
          }

          .update-details {
            font-size: 11px;
            color: var(--el-text-color-secondary);
          }
        }
      }
    }
  }

  .bulk-summary {
    margin-top: 20px;

    .summary-content {
      h4 {
        margin: 0 0 12px 0;
        color: var(--el-text-color-primary);
        font-size: 14px;
      }

      .summary-stats {
        display: flex;
        gap: 24px;

        .stat-item {
          display: flex;
          flex-direction: column;
          align-items: center;
          text-align: center;

          .stat-label {
            font-size: 12px;
            color: var(--el-text-color-secondary);
            margin-bottom: 4px;
          }

          .stat-value {
            font-size: 16px;
            font-weight: 600;
            color: var(--el-color-primary);
          }
        }
      }
    }
  }

  .bulk-progress {
    .overall-progress {
      text-align: center;
      margin-bottom: 20px;

      .progress-text {
        margin-top: 8px;
        color: var(--el-text-color-regular);
        font-size: 14px;
      }
    }

    .individual-progress {
      .progress-item {
        padding: 12px 0;
        border-bottom: 1px solid var(--el-border-color-lighter);

        &:last-child {
          border-bottom: none;
        }

        .item-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 8px;

          .container-name {
            font-size: 14px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }
        }

        .item-error {
          display: flex;
          align-items: center;
          gap: 6px;
          margin-top: 8px;
          padding: 6px 8px;
          background: rgba(var(--el-color-danger-rgb), 0.1);
          border-radius: 4px;

          .error-icon {
            color: var(--el-color-danger);
            font-size: 14px;
          }

          .error-message {
            font-size: 12px;
            color: var(--el-color-danger);
          }
        }
      }
    }
  }
}

// Responsive design
@media (max-width: 640px) {
  .update-activity-widget {
    padding: 12px;
  }

  .update-header .header-main {
    flex-direction: column;
    gap: 12px;

    .update-summary {
      gap: 16px;
    }
  }

  .running-item,
  .available-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;

    .update-actions {
      align-self: flex-end;
    }
  }

  .bulk-update-content {
    .bulk-updates-list {
      .update-items {
        max-height: 200px;
      }
    }

    .bulk-summary {
      .summary-stats {
        flex-direction: column;
        gap: 12px;
      }
    }
  }
}
</style>