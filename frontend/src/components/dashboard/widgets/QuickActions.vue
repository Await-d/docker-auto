<template>
  <div
    class="quick-actions-widget"
    :class="{ 'compact-mode': displayMode === 'compact' }"
  >
    <!-- Primary Actions -->
    <div
v-if="displayMode !== 'minimal'" class="primary-actions"
>
      <div class="section-title">Quick Actions</div>
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
      <div class="section-title">Container Management</div>
      <div class="actions-row">
        <el-button-group size="small">
          <el-button
            :loading="loadingActions.has('start-all')"
            @click="executeContainerAction('start-all')"
          >
            <el-icon><CaretRight /></el-icon>
            Start All
          </el-button>
          <el-button
            :loading="loadingActions.has('stop-all')"
            @click="executeContainerAction('stop-all')"
          >
            <el-icon><VideoPlay /></el-icon>
            Stop All
          </el-button>
          <el-button
            :loading="loadingActions.has('restart-all')"
            @click="executeContainerAction('restart-all')"
          >
            <el-icon><Refresh /></el-icon>
            Restart All
          </el-button>
        </el-button-group>
      </div>
    </div>

    <!-- System Actions -->
    <div class="system-actions">
      <div
v-if="displayMode !== 'minimal'" class="section-title">System</div>
      <div class="actions-row">
        <el-button-group size="small">
          <el-button
            :loading="loadingActions.has('scan-updates')"
            type="primary"
            @click="executeSystemAction('scan-updates')"
          >
            <el-icon><Search /></el-icon>
            <span v-if="displayMode !== 'compact'">Scan Updates</span>
          </el-button>
          <el-button
            :loading="loadingActions.has('cleanup')"
            @click="executeSystemAction('cleanup')"
          >
            <el-icon><Delete /></el-icon>
            <span v-if="displayMode !== 'compact'">Cleanup</span>
          </el-button>
          <el-button
            :loading="loadingActions.has('backup')"
            @click="executeSystemAction('backup')"
          >
            <el-icon><Download /></el-icon>
            <span v-if="displayMode !== 'compact'">Backup</span>
          </el-button>
        </el-button-group>
      </div>
    </div>

    <!-- Navigation Actions -->
    <div
v-if="displayMode === 'detailed'" class="navigation-actions"
>
      <div class="section-title">Quick Navigation</div>
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
        Custom Actions
        <el-button size="small" type="text" @click="showCustomActionDialog">
          <el-icon><Plus /></el-icon>
          Add
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
                <el-dropdown-item command="edit"> Edit </el-dropdown-item>
                <el-dropdown-item command="duplicate">
                  Duplicate
                </el-dropdown-item>
                <el-dropdown-item command="delete" divided>
                  Delete
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
      <div class="section-title">Recent Actions</div>
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
      <el-divider>Emergency</el-divider>
      <div class="emergency-buttons">
        <el-button
          :loading="loadingActions.has('maintenance-mode')"
          type="warning"
          size="small"
          @click="executeEmergencyAction('maintenance-mode')"
        >
          <el-icon><Warning /></el-icon>
          {{ maintenanceMode ? "Exit" : "Enter" }} Maintenance
        </el-button>
        <el-button
          :loading="loadingActions.has('emergency-stop')"
          type="danger"
          size="small"
          @click="executeEmergencyAction('emergency-stop')"
        >
          <el-icon><SwitchButton /></el-icon>
          Emergency Stop
        </el-button>
      </div>
    </div>

    <!-- Custom Action Dialog -->
    <el-dialog
      v-model="customActionDialogVisible"
      title="Add Custom Action"
      width="500px"
    >
      <el-form
        ref="customActionFormRef"
        :model="customActionForm"
        :rules="customActionRules"
        label-width="100px"
      >
        <el-form-item label="Name" prop="name">
          <el-input v-model="customActionForm.name" placeholder="Action name" />
        </el-form-item>
        <el-form-item label="Command" prop="command">
          <el-input
            v-model="customActionForm.command"
            placeholder="Command to execute"
          />
        </el-form-item>
        <el-form-item label="Icon" prop="icon">
          <el-select v-model="customActionForm.icon" placeholder="Select icon">
            <el-option label="Setting" value="Setting" />
            <el-option label="Tools" value="Tools" />
            <el-option label="Operation" value="Operation" />
            <el-option label="Monitor" value="Monitor" />
          </el-select>
        </el-form-item>
        <el-form-item label="Confirmation">
          <el-switch v-model="customActionForm.requiresConfirmation" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="customActionDialogVisible = false">
          Cancel
        </el-button>
        <el-button
type="primary" @click="saveCustomAction"> Save </el-button>
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
    { required: true, message: "Action name is required", trigger: "blur" },
  ],
  command: [
    { required: true, message: "Command is required", trigger: "blur" },
  ],
};

const customActions = ref([
  {
    id: "custom1",
    name: "Restart Nginx",
    command: "docker restart nginx",
    icon: "Refresh",
    requiresConfirmation: true,
  },
  {
    id: "custom2",
    name: "Clear Logs",
    command: "docker system prune --volumes",
    icon: "Delete",
    requiresConfirmation: true,
  },
]);

const recentActions = ref([
  {
    id: "recent1",
    action: "Container restart: web-server",
    status: "success",
    timestamp: new Date(Date.now() - 300000),
  },
  {
    id: "recent2",
    action: "System cleanup",
    status: "success",
    timestamp: new Date(Date.now() - 600000),
  },
  {
    id: "recent3",
    action: "Update scan",
    status: "failed",
    timestamp: new Date(Date.now() - 900000),
  },
]);

// Computed properties
const primaryActions = computed(() => [
  {
    id: "update-scan",
    title: "Scan Updates",
    description: "Check for new updates",
    icon: "Search",
    type: "primary",
    badge: 3,
    disabled: false,
  },
  {
    id: "system-health",
    title: "Health Check",
    description: "Run system diagnostics",
    icon: "Monitor",
    type: "info",
    badge: null,
    disabled: false,
  },
  {
    id: "container-prune",
    title: "Cleanup System",
    description: "Remove unused resources",
    icon: "Delete",
    type: "warning",
    badge: null,
    disabled: false,
  },
  {
    id: "backup-create",
    title: "Create Backup",
    description: "Backup system state",
    icon: "Download",
    type: "success",
    badge: null,
    disabled: false,
  },
]);

const navigationItems = computed(() => [
  {
    path: "/containers",
    label: "Containers",
    icon: "Box",
    badge: 12,
    badgeType: "primary",
  },
  {
    path: "/images",
    label: "Images",
    icon: "Picture",
    badge: null,
  },
  {
    path: "/monitoring",
    label: "Monitoring",
    icon: "DataAnalysis",
    badge: null,
  },
  {
    path: "/logs",
    label: "Logs",
    icon: "Document",
    badge: 5,
    badgeType: "warning",
  },
  {
    path: "/settings",
    label: "Settings",
    icon: "Setting",
    badge: null,
  },
  {
    path: "/users",
    label: "Users",
    icon: "User",
    badge: null,
  },
]);

// Methods
const executeAction = async (action: any) => {
  if (action.disabled || loadingActions.value.has(action.id)) return;

  try {
    loadingActions.value.add(action.id);

    // Simulate action execution
    await new Promise((resolve) => setTimeout(resolve, 1500));

    // Add to recent actions
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: `${action.title}: ${action.description}`,
      status: "success",
      timestamp: new Date(),
    });

    ElMessage.success(`${action.title} completed successfully`);
    emit("data-updated", { recentActions: recentActions.value });
  } catch (error) {
    recentActions.value.unshift({
      id: Date.now().toString(),
      action: `${action.title}: ${action.description}`,
      status: "failed",
      timestamp: new Date(),
    });

    ElMessage.error(`${action.title} failed`);
    emit("error", error);
  } finally {
    loadingActions.value.delete(action.id);
  }
};

const executeContainerAction = async (actionType: string) => {
  if (loadingActions.value.has(actionType)) return;

  try {
    loadingActions.value.add(actionType);

    const actionMap = {
      "start-all": "Starting all containers...",
      "stop-all": "Stopping all containers...",
      "restart-all": "Restarting all containers...",
    };

    ElMessage.info(actionMap[actionType as keyof typeof actionMap]);

    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 2000));

    ElMessage.success("Container action completed");
  } catch (error) {
    ElMessage.error("Container action failed");
    emit("error", error);
  } finally {
    loadingActions.value.delete(actionType);
  }
};

const executeSystemAction = async (actionType: string) => {
  if (loadingActions.value.has(actionType)) return;

  try {
    loadingActions.value.add(actionType);

    const actionMap = {
      "scan-updates": "Scanning for updates...",
      cleanup: "Cleaning up system...",
      backup: "Creating backup...",
    };

    ElMessage.info(actionMap[actionType as keyof typeof actionMap]);

    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 2000));

    ElMessage.success("System action completed");
  } catch (error) {
    ElMessage.error("System action failed");
    emit("error", error);
  } finally {
    loadingActions.value.delete(actionType);
  }
};

const executeCustomAction = async (action: any) => {
  try {
    if (action.requiresConfirmation) {
      await ElMessageBox.confirm(
        `Are you sure you want to execute: ${action.command}?`,
        "Confirm Action",
        {
          type: "warning",
          confirmButtonText: "Execute",
          cancelButtonText: "Cancel",
        },
      );
    }

    ElMessage.info(`Executing: ${action.name}`);

    // Simulate command execution
    await new Promise((resolve) => setTimeout(resolve, 1500));

    ElMessage.success(`${action.name} executed successfully`);
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(`Failed to execute ${action.name}`);
    }
  }
};

const executeEmergencyAction = async (actionType: string) => {
  if (loadingActions.value.has(actionType)) return;

  try {
    const confirmMessage =
      actionType === "maintenance-mode"
        ? `Are you sure you want to ${maintenanceMode.value ? "exit" : "enter"} maintenance mode?`
        : "Are you sure you want to perform an emergency stop? This will stop all containers immediately.";

    await ElMessageBox.confirm(confirmMessage, "Confirm Emergency Action", {
      type: "warning",
      confirmButtonText: "Confirm",
      cancelButtonText: "Cancel",
    });

    loadingActions.value.add(actionType);

    if (actionType === "maintenance-mode") {
      maintenanceMode.value = !maintenanceMode.value;
      ElMessage.success(
        `Maintenance mode ${maintenanceMode.value ? "activated" : "deactivated"}`,
      );
    } else {
      ElMessage.warning("Emergency stop initiated");
    }

    // Simulate action
    await new Promise((resolve) => setTimeout(resolve, 1000));
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("Emergency action failed");
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
    customActionDialogVisible.value = false;
    ElMessage.success("Custom action added");
  } catch (error) {
    console.error("Form validation failed:", error);
  }
};

const handleCustomActionMenu = (command: string, action: any) => {
  switch (command) {
    case "edit":
      ElMessage.info(`Editing ${action.name}`);
      break;
    case "duplicate": {
      const duplicated = {
        ...action,
        id: Date.now().toString(),
        name: `${action.name} (Copy)`,
      };
      customActions.value.push(duplicated);
      ElMessage.success("Action duplicated");
      break;
    }
    case "delete": {
      const index = customActions.value.findIndex((a) => a.id === action.id);
      if (index !== -1) {
        customActions.value.splice(index, 1);
        ElMessage.success("Action deleted");
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
onMounted(() => {
  emit("data-updated", {
    customActions: customActions.value,
    recentActions: recentActions.value,
    maintenanceMode: maintenanceMode.value,
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
