<template>
  <div
    class="quick-actions-widget"
    :class="{ 'compact-mode': displayMode === 'compact' }"
  >
    <!-- Primary Actions -->
    <div
v-if="displayMode !== 'minimal'" class="primary-actions"
>
      <div class="section-title">快速操作</div>
      <div class="actions-grid">
        <div
          v-for="action in primaryActions"
          :key="action.id"
          class="action-button"
          :class="[
            action.type,
            {
              disabled: action.disabled,
              loading: loadingActions.has(action.id),
            },
          ]"
          @click="executeAction(action)"
        >
          <div class="action-icon">
            <el-icon v-if="!loadingActions.has(action.id)" :size="20">
              <component :is="action.icon" />
            </el-icon>
            <el-icon v-else class="is-loading" :size="20">
              <Loading />
            </el-icon>
          </div>
          <div class="action-content">
            <span class="action-title">{{ action.title }}</span>
            <span class="action-description">{{ action.description }}</span>
          </div>
          <div
v-if="action.badge" class="action-indicator"
>
            <el-badge :value="action.badge" :type="getBadgeType(action.type)" />
          </div>
        </div>
      </div>
    </div>

    <!-- Container Actions -->
    <div
v-if="displayMode === 'detailed'" class="container-actions"
>
      <div class="section-title">容器管理</div>
      <div class="actions-row">
        <el-button-group size="small">
          <el-button
            :loading="loadingActions.has('start-all')"
            @click="executeContainerAction('start-all')"
          >
            <el-icon><CaretRight /></el-icon>
            全部启动
          </el-button>
          <el-button
            :loading="loadingActions.has('stop-all')"
            @click="executeContainerAction('stop-all')"
          >
            <el-icon><VideoPlay /></el-icon>
            全部停止
          </el-button>
          <el-button
            :loading="loadingActions.has('restart-all')"
            @click="executeContainerAction('restart-all')"
          >
            <el-icon><Refresh /></el-icon>
            全部重启
          </el-button>
        </el-button-group>
      </div>
    </div>

    <!-- System Actions -->
    <div class="system-actions">
      <div
v-if="displayMode !== 'minimal'" class="section-title">系统</div>
      <div class="actions-row">
        <el-button-group size="small">
          <el-button
            :loading="loadingActions.has('scan-updates')"
            type="primary"
            @click="executeSystemAction('scan-updates')"
          >
            <el-icon><Search /></el-icon>
            <span v-if="displayMode !== 'compact'">扫描更新</span>
          </el-button>
          <el-button
            :loading="loadingActions.has('cleanup')"
            @click="executeSystemAction('cleanup')"
          >
            <el-icon><Delete /></el-icon>
            <span v-if="displayMode !== 'compact'">清理</span>
          </el-button>
          <el-button
            :loading="loadingActions.has('backup')"
            @click="executeSystemAction('backup')"
          >
            <el-icon><Download /></el-icon>
            <span v-if="displayMode !== 'compact'">备份</span>
          </el-button>
        </el-button-group>
      </div>
    </div>

    <!-- Navigation Actions -->
    <div
v-if="displayMode === 'detailed'" class="navigation-actions"
>
      <div class="section-title">快速导航</div>
      <div class="nav-grid">
        <div
          v-for="nav in navigationItems"
          :key="nav.path"
          class="nav-item"
          @click="navigateTo(nav.path)"
        >
          <el-icon class="nav-icon">
            <component :is="nav.icon" />
          </el-icon>
          <span class="nav-label">{{ nav.label }}</span>
          <div
v-if="nav.badge" class="nav-badge"
>
            <el-badge
              :value="nav.badge"
              :type="
                (nav.badgeType as
                  | 'success'
                  | 'warning'
                  | 'info'
                  | 'primary'
                  | 'danger') || 'primary'
              "
            />
          </div>
        </div>
      </div>
    </div>

    <!-- Custom Actions -->
    <div
      v-if="customActions.length > 0 && displayMode !== 'minimal'"
      class="custom-actions"
    >
      <div class="section-title">
        自定义操作
        <el-button size="small" type="text" @click="showCustomActionDialog">
          <el-icon><Plus /></el-icon>
          添加
        </el-button>
      </div>
      <div class="custom-list">
        <div
          v-for="action in customActions"
          :key="action.id"
          class="custom-item"
          @click="executeCustomAction(action)"
        >
          <div class="custom-icon">
            <el-icon>
              <component :is="action.icon || 'Setting'" />
            </el-icon>
          </div>
          <span class="custom-label">{{ action.name }}</span>
          <el-dropdown
            @command="(cmd: string) => handleCustomActionMenu(cmd, action)"
          >
            <el-button size="small" type="text">
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit"> 编辑 </el-dropdown-item>
                <el-dropdown-item command="duplicate">
                  复制
                </el-dropdown-item>
                <el-dropdown-item command="delete" divided>
                  删除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <!-- Recent Actions -->
    <div
      v-if="recentActions.length > 0 && displayMode === 'detailed'"
      class="recent-actions"
    >
      <div class="section-title">最近操作</div>
      <div class="recent-list">
        <div
          v-for="action in recentActions.slice(0, 3)"
          :key="action.id"
          class="recent-item"
          :class="action.status"
        >
          <div class="recent-icon">
            <el-icon>
              <component :is="getStatusIcon(action.status)" />
            </el-icon>
          </div>
          <div class="recent-content">
            <span class="recent-action">{{ action.action }}</span>
            <span class="recent-time">{{
              formatRelativeTime(action.timestamp)
            }}</span>
          </div>
          <div class="recent-status">
            <el-tag :type="getStatusType(action.status)" size="small">
              {{ action.status }}
            </el-tag>
          </div>
        </div>
      </div>
    </div>

    <!-- Emergency Actions -->
    <div
v-if="displayMode === 'detailed'" class="emergency-actions"
>
      <el-divider>紧急操作</el-divider>
      <div class="emergency-buttons">
        <el-button
          :loading="loadingActions.has('maintenance-mode')"
          type="warning"
          size="small"
          @click="executeEmergencyAction('maintenance-mode')"
        >
          <el-icon><Warning /></el-icon>
          {{ maintenanceMode ? "退出" : "进入" }}维护模式
        </el-button>
        <el-button
          :loading="loadingActions.has('emergency-stop')"
          type="danger"
          size="small"
          @click="executeEmergencyAction('emergency-stop')"
        >
          <el-icon><SwitchButton /></el-icon>
          紧急停止
        </el-button>
      </div>
    </div>

    <!-- Custom Action Dialog -->
    <el-dialog
      v-model="customActionDialogVisible"
      title="添加自定义操作"
      width="500px"
    >
      <el-form
        ref="customActionFormRef"
        :model="customActionForm"
        :rules="customActionRules"
        label-width="100px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="customActionForm.name" placeholder="操作名称" />
        </el-form-item>
        <el-form-item label="命令" prop="command">
          <el-input
            v-model="customActionForm.command"
            placeholder="要执行的命令"
          />
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <el-select v-model="customActionForm.icon" placeholder="选择图标">
            <el-option label="设置" value="Setting" />
            <el-option label="工具" value="Tools" />
            <el-option label="操作" value="Operation" />
            <el-option label="监控" value="Monitor" />
          </el-select>
        </el-form-item>
        <el-form-item label="需要确认">
          <el-switch v-model="customActionForm.requiresConfirmation" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="customActionDialogVisible = false">
          取消
        </el-button>
        <el-button
type="primary" @click="saveCustomAction"> 保存 </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { useRouter } from "vue-router";
import {
  CaretRight,
  VideoPlay,
  Refresh,
  Search,
  Delete,
  Download,
  Plus,
  MoreFilled,
  Warning,
  SwitchButton,
  Loading,
  SuccessFilled,
  CircleCloseFilled,
  Clock,
  Box,
  Monitor,
  Document,
  DataAnalysis,
  Bell,
  User,
  Setting,
} from "@element-plus/icons-vue";

// Icons used in dynamic template components - create reference object for TypeScript
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = {
  SuccessFilled,
  CircleCloseFilled,
  Clock,
  Box,
  Monitor,
  Document,
  DataAnalysis,
  Bell,
  User,
  Setting,
};

// Props
interface Props {
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
  displayMode?: "default" | "compact" | "detailed" | "minimal";
}

withDefaults(defineProps<Props>(), {
  displayMode: "default",
});

// Emits
const emit = defineEmits<{
  "data-updated": [data: any];
  error: [error: any];
  loading: [loading: boolean];
}>();

// Router
const router = useRouter();

// Reactive state
const loadingActions = ref(new Set<string>());
const customActionDialogVisible = ref(false);
const customActionFormRef = ref();
const maintenanceMode = ref(false);

const customActionForm = ref({
  name: "",
  command: "",
  icon: "Setting",
  requiresConfirmation: false,
});

const customActionRules = {
  name: [
    { required: true, message: "操作名称是必需的", trigger: "blur" },
  ],
  command: [
    { required: true, message: "命令是必需的", trigger: "blur" },
  ],
};

const customActions = ref<any[]>([]);
const recentActions = ref<any[]>([]);
const systemStats = ref({
  containerCount: 0,
  availableUpdates: 0,
  warningLogs: 0,
  errorLogs: 0,
});
const isLoading = ref(false);
const error = ref('');

// Computed properties
const primaryActions = computed(() => [
  {
    id: "update-scan",
    title: "扫描更新",
    description: "检查新的更新",
    icon: "Search",
    type: "primary",
    badge: systemStats.value.availableUpdates || null,
    disabled: false,
  },
  {
    id: "system-health",
    title: "健康检查",
    description: "运行系统诊断",
    icon: "Monitor",
    type: "info",
    badge: null,
    disabled: false,
  },
  {
    id: "container-prune",
    title: "清理系统",
    description: "移除未使用的资源",
    icon: "Delete",
    type: "warning",
    badge: null,
    disabled: false,
  },
  {
    id: "backup-create",
    title: "创建备份",
    description: "备份系统状态",
    icon: "Download",
    type: "success",
    badge: null,
    disabled: false,
  },
]);

const navigationItems = computed(() => [
  {
    path: "/containers",
    label: "容器",
    icon: "Box",
    badge: systemStats.value.containerCount || null,
    badgeType: "primary",
  },
  {
    path: "/images",
    label: "镜像",
    icon: "Picture",
    badge: null,
  },
  {
    path: "/monitoring",
    label: "监控",
    icon: "DataAnalysis",
    badge: null,
  },
  {
    path: "/logs",
    label: "日志",
    icon: "Document",
    badge: systemStats.value.warningLogs + systemStats.value.errorLogs || null,
    badgeType: systemStats.value.errorLogs > 0 ? "danger" : "warning",
  },
  {
    path: "/settings",
    label: "设置",
    icon: "Setting",
    badge: null,
  },
  {
    path: "/users",
    label: "用户",
    icon: "User",
    badge: null,
  },
]);

// 加载初始数据
const loadWidgetData = async () => {
  try {
    isLoading.value = true;
    error.value = '';

    // 并行加载各种数据
    const [containerSummary, updateSummary, activityLogs, savedActions] = await Promise.all([
      loadContainerSummary(),
      loadUpdateSummary(),
      loadRecentActions(),
      loadCustomActions()
    ]);

    systemStats.value = {
      containerCount: containerSummary.total || 0,
      availableUpdates: updateSummary.available || 0,
      warningLogs: activityLogs.warnings || 0,
      errorLogs: activityLogs.errors || 0,
    };

    recentActions.value = activityLogs.recentActions || [];
    customActions.value = savedActions || [];

  } catch (err: any) {
    console.error('Failed to load widget data:', err);
    error.value = err.message || '加载数据失败';
  } finally {
    isLoading.value = false;
  }
};

// 加载容器摘要
const loadContainerSummary = async () => {
  try {
    const { containerAPI } = await import('@/api/container');
    return await containerAPI.getContainersSummary();
  } catch (err) {
    console.error('Failed to load container summary:', err);
    return { total: 0, running: 0, stopped: 0 };
  }
};

// 加载更新摘要
const loadUpdateSummary = async () => {
  try {
    const { updatesAPI } = await import('@/api/updates');
    return await updatesAPI.getUpdatesSummary();
  } catch (err) {
    console.error('Failed to load update summary:', err);
    return { available: 0, security: 0 };
  }
};

// 加载最近操作
const loadRecentActions = async () => {
  try {
    const { updatesAPI } = await import('@/api/updates');
    const activities = await updatesAPI.getActivityLog({ limit: 5, timeRange: '24h' });
    return {
      recentActions: activities.activities?.map((a: any) => ({
        id: a.id,
        action: a.title || a.description,
        status: a.status || (a.type === 'error' ? 'failed' : 'success'),
        timestamp: new Date(a.timestamp)
      })) || [],
      warnings: activities.activities?.filter((a: any) => a.type === 'warning').length || 0,
      errors: activities.activities?.filter((a: any) => a.type === 'error').length || 0,
    };
  } catch (err) {
    console.error('Failed to load recent actions:', err);
    return { recentActions: [], warnings: 0, errors: 0 };
  }
};

// 加载自定义操作
const loadCustomActions = async () => {
  try {
    // 从本地存储或API加载自定义操作
    const saved = localStorage.getItem('customActions');
    return saved ? JSON.parse(saved) : [];
  } catch (err) {
    console.error('Failed to load custom actions:', err);
    return [];
  }
};

// Methods
const executeAction = async (action: any) => {
  if (action.disabled || loadingActions.value.has(action.id)) return;

  try {
    loadingActions.value.add(action.id);
    let result;

    // 执行真实的API调用
    switch (action.id) {
      case 'update-scan':
        result = await executeUpdateScan();
        break;
      case 'system-health':
        result = await executeHealthCheck();
        break;
      case 'container-prune':
        result = await executeSystemCleanup();
        break;
      case 'backup-create':
        result = await executeSystemBackup();
        break;
      default:
        throw new Error(`Unknown action: ${action.id}`);
    }

    // 添加到最近操作
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: `${action.title}: ${result.message || action.description}`,
      status: "success",
      timestamp: new Date(),
    });

    ElMessage.success(`${action.title}执行成功`);
    emit("data-updated", { recentActions: recentActions.value });

    // 刷新相关数据
    await loadWidgetData();
  } catch (error: any) {
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: `${action.title}: ${error.message || action.description}`,
      status: "failed",
      timestamp: new Date(),
    });

    ElMessage.error(`${action.title}执行失败: ${error.message}`);
    emit("error", error);
  } finally {
    loadingActions.value.delete(action.id);
  }
};

// 执行更新扫描
const executeUpdateScan = async () => {
  const { updatesAPI } = await import('@/api/updates');
  const result = await updatesAPI.checkUpdates();
  return { message: `发现${result.availableUpdates || 0}个可用更新` };
};

// 执行健康检查
const executeHealthCheck = async () => {
  const { monitoringAPI } = await import('@/api/monitoring');
  const systemMetrics = await monitoringAPI.getSystemMetrics();
  const healthScore = calculateHealthScore(systemMetrics);
  return { message: `系统健康评分: ${healthScore}/100` };
};

// 计算健康评分
const calculateHealthScore = (metrics: any) => {
  let score = 100;
  if (metrics.cpu?.usage > 80) score -= 20;
  if (metrics.memory && (metrics.memory.used / metrics.memory.total) > 0.85) score -= 25;
  if (metrics.disk && (metrics.disk.used / metrics.disk.total) > 0.8) score -= 15;
  return Math.max(0, score);
};

// 执行系统清理
const executeSystemCleanup = async () => {
  const { containerAPI } = await import('@/api/container');
  await containerAPI.pruneSystem();
  return { message: '系统清理完成' };
};

// 执行系统备份
const executeSystemBackup = async () => {
  const { updatesAPI } = await import('@/api/updates');
  const backupId = await updatesAPI.createBackup();
  return { message: `备份已创建 (ID: ${backupId})` };
};

const executeContainerAction = async (actionType: string) => {
  if (loadingActions.value.has(actionType)) return;

  try {
    loadingActions.value.add(actionType);
    const { containerAPI } = await import('@/api/container');

    let result;
    let message = '';

    switch (actionType) {
      case 'start-all':
        ElMessage.info('正在启动所有容器...');
        result = await containerAPI.startAllContainers();
        message = `已启动 ${result.started || 0} 个容器`;
        break;
      case 'stop-all':
        ElMessage.info('正在停止所有容器...');
        result = await containerAPI.stopAllContainers();
        message = `已停止 ${result.stopped || 0} 个容器`;
        break;
      case 'restart-all':
        ElMessage.info('正在重启所有容器...');
        result = await containerAPI.restartAllContainers();
        message = `已重启 ${result.restarted || 0} 个容器`;
        break;
      default:
        throw new Error(`Unknown container action: ${actionType}`);
    }

    // 添加到最近操作
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: message,
      status: "success",
      timestamp: new Date(),
    });

    ElMessage.success('容器操作完成');
    await loadWidgetData(); // 刷新数据
  } catch (error: any) {
    ElMessage.error(`容器操作失败: ${error.message}`);
    emit("error", error);
  } finally {
    loadingActions.value.delete(actionType);
  }
};

const executeSystemAction = async (actionType: string) => {
  if (loadingActions.value.has(actionType)) return;

  try {
    loadingActions.value.add(actionType);
    let result;
    let message = '';

    switch (actionType) {
      case 'scan-updates':
        ElMessage.info('正在扫描更新...');
        result = await executeUpdateScan();
        message = result.message;
        break;
      case 'cleanup':
        ElMessage.info('正在清理系统...');
        result = await executeSystemCleanup();
        message = result.message;
        break;
      case 'backup':
        ElMessage.info('正在创建备份...');
        result = await executeSystemBackup();
        message = result.message;
        break;
      default:
        throw new Error(`Unknown system action: ${actionType}`);
    }

    // 添加到最近操作
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: message,
      status: "success",
      timestamp: new Date(),
    });

    ElMessage.success('系统操作完成');
    await loadWidgetData(); // 刷新数据
  } catch (error: any) {
    ElMessage.error(`系统操作失败: ${error.message}`);
    emit("error", error);
  } finally {
    loadingActions.value.delete(actionType);
  }
};

const executeCustomAction = async (action: any) => {
  try {
    if (action.requiresConfirmation) {
      await ElMessageBox.confirm(
        `确定要执行自定义操作: ${action.command}?`,
        "确认操作",
        {
          type: "warning",
          confirmButtonText: "执行",
          cancelButtonText: "取消",
        },
      );
    }

    ElMessage.info(`正在执行: ${action.name}`);

    // 执行真实的命令
    const { containerAPI } = await import('@/api/container');
    const result = await containerAPI.executeCommand(action.command);

    // 添加到最近操作
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: `自定义操作: ${action.name}`,
      status: "success",
      timestamp: new Date(),
    });

    ElMessage.success(`${action.name} 执行成功`);
    await loadWidgetData(); // 刷新数据
  } catch (error: any) {
    if (error !== "cancel") {
      // 添加失败记录
      recentActions.value.unshift({
        id: Date.now().toString(),
        action: `自定义操作: ${action.name} (失败)`,
        status: "failed",
        timestamp: new Date(),
      });

      ElMessage.error(`执行失败 ${action.name}: ${error.message}`);
    }
  }
};

const executeEmergencyAction = async (actionType: string) => {
  if (loadingActions.value.has(actionType)) return;

  try {
    const confirmMessage =
      actionType === "maintenance-mode"
        ? `确定要${maintenanceMode.value ? "退出" : "进入"}维护模式吗?`
        : "确定要执行紧急停止吗？这将立即停止所有容器。";

    await ElMessageBox.confirm(confirmMessage, "确认紧急操作", {
      type: "warning",
      confirmButtonText: "确认",
      cancelButtonText: "取消",
    });

    loadingActions.value.add(actionType);
    const { containerAPI } = await import('@/api/container');

    let message = '';
    if (actionType === "maintenance-mode") {
      const newMode = !maintenanceMode.value;
      await containerAPI.setMaintenanceMode(newMode);
      maintenanceMode.value = newMode;
      message = `维护模式已${newMode ? "激活" : "停用"}`;
      ElMessage.success(message);
    } else {
      await containerAPI.emergencyStop();
      message = "紧急停止已执行";
      ElMessage.warning(message);
    }

    // 添加到最近操作
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: message,
      status: "success",
      timestamp: new Date(),
    });

    await loadWidgetData(); // 刷新数据
  } catch (error: any) {
    if (error !== "cancel") {
      ElMessage.error(`紧急操作失败: ${error.message}`);
    }
  } finally {
    loadingActions.value.delete(actionType);
  }
};

const navigateTo = (path: string) => {
  router.push(path);
  ElMessage.info(`Navigating to ${path}`);
};

const showCustomActionDialog = () => {
  customActionForm.value = {
    name: "",
    command: "",
    icon: "Setting",
    requiresConfirmation: false,
  };
  customActionDialogVisible.value = true;
};

const saveCustomAction = async () => {
  try {
    await customActionFormRef.value.validate();

    const newAction = {
      id: Date.now().toString(),
      ...customActionForm.value,
    };

    customActions.value.push(newAction);

    // 保存到本地存储或API
    localStorage.setItem('customActions', JSON.stringify(customActions.value));

    customActionDialogVisible.value = false;
    ElMessage.success("自定义操作已添加");
  } catch (error) {
    console.error("表单验证失败:", error);
  }
};

const handleCustomActionMenu = (command: string, action: any) => {
  switch (command) {
    case "edit":
      // 编辑自定义操作
      customActionForm.value = {
        name: action.name,
        command: action.command,
        icon: action.icon,
        requiresConfirmation: action.requiresConfirmation,
      };
      customActionDialogVisible.value = true;
      ElMessage.info(`正在编辑 ${action.name}`);
      break;
    case "duplicate": {
      const duplicated = {
        ...action,
        id: Date.now().toString(),
        name: `${action.name} (副本)`,
      };
      customActions.value.push(duplicated);
      localStorage.setItem('customActions', JSON.stringify(customActions.value));
      ElMessage.success("操作已复制");
      break;
    }
    case "delete": {
      const index = customActions.value.findIndex((a) => a.id === action.id);
      if (index !== -1) {
        customActions.value.splice(index, 1);
        localStorage.setItem('customActions', JSON.stringify(customActions.value));
        ElMessage.success("操作已删除");
      }
      break;
    }
  }
};

const getBadgeType = (actionType: string) => {
  switch (actionType) {
    case "primary":
      return "primary";
    case "warning":
      return "warning";
    case "danger":
      return "danger";
    case "success":
      return "success";
    default:
      return "info";
  }
};

const getStatusIcon = (status: string) => {
  switch (status) {
    case "success":
      return "SuccessFilled";
    case "failed":
      return "CircleCloseFilled";
    case "pending":
      return "Clock";
    default:
      return "Clock";
  }
};

const getStatusType = (status: string) => {
  switch (status) {
    case "success":
      return "success";
    case "failed":
      return "danger";
    case "pending":
      return "warning";
    default:
      return "info";
  }
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);

  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  return date.toLocaleDateString();
};

// Lifecycle hooks
onMounted(async () => {
  await loadWidgetData();

  emit("data-updated", {
    customActions: customActions.value,
    recentActions: recentActions.value,
    maintenanceMode: maintenanceMode.value,
    systemStats: systemStats.value,
  });
});
</script>

<style scoped lang="scss">
.quick-actions-widget {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;

  &.compact-mode {
    padding: 12px;
    gap: 12px;
  }
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.primary-actions {
  .actions-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 8px;

    .action-button {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px;
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
      cursor: pointer;
      transition: all 0.3s ease;
      position: relative;

      &:hover:not(.disabled):not(.loading) {
        border-color: var(--el-color-primary);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      }

      &.disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      &.loading {
        opacity: 0.8;
        cursor: not-allowed;
      }

      .action-icon {
        flex-shrink: 0;
        width: 40px;
        height: 40px;
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.3s ease;
      }

      &.primary .action-icon {
        background: rgba(var(--el-color-primary-rgb), 0.1);
        color: var(--el-color-primary);
      }

      &.success .action-icon {
        background: rgba(var(--el-color-success-rgb), 0.1);
        color: var(--el-color-success);
      }

      &.warning .action-icon {
        background: rgba(var(--el-color-warning-rgb), 0.1);
        color: var(--el-color-warning);
      }

      &.danger .action-icon {
        background: rgba(var(--el-color-danger-rgb), 0.1);
        color: var(--el-color-danger);
      }

      &.info .action-icon {
        background: rgba(var(--el-color-info-rgb), 0.1);
        color: var(--el-color-info);
      }

      .action-content {
        flex: 1;
        min-width: 0;

        .action-title {
          display: block;
          font-size: 14px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 2px;
        }

        .action-description {
          font-size: 12px;
          color: var(--el-text-color-secondary);
          line-height: 1.4;
        }
      }

      .action-indicator {
        flex-shrink: 0;
      }
    }
  }
}

.container-actions,
.system-actions {
  .actions-row {
    display: flex;
    justify-content: center;
  }
}

.navigation-actions {
  .nav-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;

    .nav-item {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 4px;
      padding: 8px;
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 6px;
      cursor: pointer;
      transition: all 0.3s ease;
      position: relative;

      &:hover {
        border-color: var(--el-color-primary);
        background: rgba(var(--el-color-primary-rgb), 0.05);
      }

      .nav-icon {
        font-size: 18px;
        color: var(--el-color-primary);
      }

      .nav-label {
        font-size: 11px;
        color: var(--el-text-color-secondary);
        text-align: center;
        line-height: 1.2;
      }

      .nav-badge {
        position: absolute;
        top: -2px;
        right: -2px;
      }
    }
  }
}

.custom-actions {
  .custom-list {
    .custom-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 6px;
      margin-bottom: 6px;
      cursor: pointer;
      transition: all 0.3s ease;

      &:hover {
        border-color: var(--el-color-primary);
      }

      &:last-child {
        margin-bottom: 0;
      }

      .custom-icon {
        color: var(--el-color-primary);
        font-size: 16px;
      }

      .custom-label {
        flex: 1;
        font-size: 13px;
        color: var(--el-text-color-primary);
        font-weight: 500;
      }
    }
  }
}

.recent-actions {
  .recent-list {
    .recent-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: none;
      }

      .recent-icon {
        flex-shrink: 0;
        font-size: 14px;

        &.success {
          color: var(--el-color-success);
        }

        &.failed {
          color: var(--el-color-danger);
        }

        &.pending {
          color: var(--el-color-warning);
        }
      }

      .recent-content {
        flex: 1;
        min-width: 0;

        .recent-action {
          display: block;
          font-size: 12px;
          color: var(--el-text-color-primary);
          font-weight: 500;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .recent-time {
          font-size: 11px;
          color: var(--el-text-color-placeholder);
        }
      }

      .recent-status {
        flex-shrink: 0;
      }
    }
  }
}

.emergency-actions {
  .emergency-buttons {
    display: flex;
    gap: 8px;
    justify-content: center;
  }
}

// Responsive design
@media (max-width: 480px) {
  .quick-actions-widget {
    .navigation-actions .nav-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .emergency-actions .emergency-buttons {
      flex-direction: column;
    }
  }
}

// Animations
.is-loading {
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
