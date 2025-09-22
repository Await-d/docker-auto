<template>
  <el-dialog
    :model-value="modelValue"
    title="小部件配置"
    width="800px"
    :modal="true"
    class="widget-config-dialog"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="widget" class="config-container">
      <!-- 小部件信息 -->
      <div class="widget-info-section">
        <div class="widget-header">
          <div class="widget-icon">
            <el-icon :size="24">
              <component :is="widgetIcon" />
            </el-icon>
          </div>
          <div class="widget-details">
            <h3>{{ widget.title }}</h3>
            <p>{{ widgetDescription }}</p>
          </div>
        </div>
      </div>

      <!-- 配置选项卡 -->
      <el-tabs v-model="activeTab" class="config-tabs">
        <!-- 通用设置 -->
        <el-tab-pane label="通用" name="general">
          <div class="config-section">
            <el-form
              ref="generalFormRef"
              :model="generalConfig"
              :rules="generalRules"
              label-width="140px"
            >
              <el-form-item label="小部件标题" prop="title">
                <el-input
                  v-model="generalConfig.title"
                  placeholder="输入小部件标题"
                  clearable
                />
              </el-form-item>

              <el-form-item label="刷新间隔" prop="refreshInterval">
                <el-select
                  v-model="generalConfig.refreshInterval"
                  placeholder="选择刷新间隔"
                >
                  <el-option label="从不" :value="0" />
                  <el-option label="5秒" :value="5000" />
                  <el-option label="10秒" :value="10000" />
                  <el-option label="30秒" :value="30000" />
                  <el-option label="1分钟" :value="60000" />
                  <el-option label="5分钟" :value="300000" />
                  <el-option label="15分钟" :value="900000" />
                  <el-option label="30分钟" :value="1800000" />
                  <el-option label="1小时" :value="3600000" />
                </el-select>
              </el-form-item>

              <el-form-item label="启用小部件">
                <el-switch
                  v-model="generalConfig.enabled"
                  active-text="已启用"
                  inactive-text="已禁用"
                />
              </el-form-item>

              <el-form-item label="可拖动">
                <el-switch
                  v-model="generalConfig.draggable"
                  active-text="是"
                  inactive-text="否"
                />
              </el-form-item>

              <el-form-item label="可调整大小">
                <el-switch
                  v-model="generalConfig.resizable"
                  active-text="是"
                  inactive-text="否"
                />
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- 外观设置 -->
        <el-tab-pane label="外观" name="appearance">
          <div class="config-section">
            <el-form
              ref="appearanceFormRef"
              :model="appearanceConfig"
              label-width="140px"
            >
              <el-form-item label="主题">
                <el-radio-group v-model="appearanceConfig.theme">
                  <el-radio label="auto"> 自动 </el-radio>
                  <el-radio label="light"> 浅色 </el-radio>
                  <el-radio label="dark"> 深色 </el-radio>
                </el-radio-group>
              </el-form-item>

              <el-form-item label="显示模式">
                <el-select
                  v-model="appearanceConfig.displayMode"
                  placeholder="选择显示模式"
                >
                  <el-option label="默认" value="default" />
                  <el-option label="紧凑" value="compact" />
                  <el-option label="详细" value="detailed" />
                  <el-option label="精简" value="minimal" />
                </el-select>
              </el-form-item>

              <el-form-item label="显示头部">
                <el-switch
                  v-model="appearanceConfig.showHeader"
                  active-text="是"
                  inactive-text="否"
                />
              </el-form-item>

              <el-form-item label="显示底部">
                <el-switch
                  v-model="appearanceConfig.showFooter"
                  active-text="是"
                  inactive-text="否"
                />
              </el-form-item>

              <el-form-item label="动画效果">
                <el-switch
                  v-model="appearanceConfig.animations"
                  active-text="已启用"
                  inactive-text="已禁用"
                />
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- 小部件专用设置 -->
        <el-tab-pane
          v-if="hasSpecificSettings"
          :label="specificTabLabel"
          name="specific"
        >
          <div class="config-section">
            <component
              :is="specificConfigComponent"
              v-model="specificConfig"
              :widget="widget"
            />
          </div>
        </el-tab-pane>

        <!-- Data & Filters -->
        <el-tab-pane label="Data & Filters" name="data">
          <div class="config-section">
            <el-form ref="dataFormRef" :model="dataConfig" label-width="140px">
              <el-form-item label="数据源">
                <el-select
                  v-model="dataConfig.dataSource"
                  placeholder="选择数据源"
                >
                  <el-option label="API" value="api" />
                  <el-option label="WebSocket" value="websocket" />
                  <el-option label="本地存储" value="localStorage" />
                  <el-option label="模拟数据" value="mock" />
                </el-select>
              </el-form-item>

              <el-form-item label="缓存时间">
                <el-input-number
                  v-model="dataConfig.cacheDuration"
                  :min="0"
                  :max="3600000"
                  :step="1000"
                  controls-position="right"
                />
                <span class="input-suffix">ms</span>
              </el-form-item>

              <el-form-item label="最大数据点数">
                <el-input-number
                  v-model="dataConfig.maxDataPoints"
                  :min="10"
                  :max="1000"
                  :step="10"
                  controls-position="right"
                />
              </el-form-item>

              <el-form-item label="日期范围">
                <el-select
                  v-model="dataConfig.dateRange"
                  placeholder="选择日期范围"
                >
                  <el-option label="最近一小时" value="1h" />
                  <el-option label="最近6小时" value="6h" />
                  <el-option label="最近24小时" value="24h" />
                  <el-option label="最近7天" value="7d" />
                  <el-option label="最近30天" value="30d" />
                  <el-option label="自定义" value="custom" />
                </el-select>
              </el-form-item>

              <el-form-item label="过滤器">
                <key-value-editor
                  v-model="dataConfig.filters"
                  placeholder-key="Filter Key"
                  placeholder-value="Filter Value"
                />
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- Advanced Settings -->
        <el-tab-pane label="高级" name="advanced">
          <div class="config-section">
            <el-form
              ref="advancedFormRef"
              :model="advancedConfig"
              label-width="140px"
            >
              <el-form-item label="错误处理">
                <el-radio-group v-model="advancedConfig.errorHandling">
                  <el-radio label="retry"> Retry on Error </el-radio>
                  <el-radio label="fallback"> Show Fallback </el-radio>
                  <el-radio label="hide"> Hide Widget </el-radio>
                </el-radio-group>
              </el-form-item>

              <el-form-item label="重试次数">
                <el-input-number
                  v-model="advancedConfig.retryAttempts"
                  :min="0"
                  :max="10"
                  controls-position="right"
                />
              </el-form-item>

              <el-form-item label="重试延迟">
                <el-input-number
                  v-model="advancedConfig.retryDelay"
                  :min="1000"
                  :max="60000"
                  :step="1000"
                  controls-position="right"
                />
                <span class="input-suffix">ms</span>
              </el-form-item>

              <el-form-item label="调试模式">
                <el-switch
                  v-model="advancedConfig.debugMode"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>

              <el-form-item label="性能监控">
                <el-switch
                  v-model="advancedConfig.performanceMonitoring"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>

              <el-form-item label="自定义CSS">
                <el-input
                  v-model="advancedConfig.customCSS"
                  type="textarea"
                  :rows="4"
                  placeholder="输入自定义CSS规则"
                />
              </el-form-item>

              <el-form-item label="自定义属性">
                <key-value-editor
                  v-model="advancedConfig.customProperties"
                  placeholder-key="Property Name"
                  placeholder-value="Property Value"
                />
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- Permissions -->
        <el-tab-pane label="权限" name="permissions">
          <div class="config-section">
            <el-form
              ref="permissionsFormRef"
              :model="permissionsConfig"
              label-width="140px"
            >
              <el-form-item label="所需权限">
                <el-checkbox-group v-model="permissionsConfig.permissions">
                  <el-checkbox label="dashboard:read">
                    Dashboard Read
                  </el-checkbox>
                  <el-checkbox label="container:read">
                    Container Read
                  </el-checkbox>
                  <el-checkbox label="container:write">
                    Container Write
                  </el-checkbox>
                  <el-checkbox label="image:read"> Image Read </el-checkbox>
                  <el-checkbox label="update:read"> Update Read </el-checkbox>
                  <el-checkbox label="update:write"> Update Write </el-checkbox>
                  <el-checkbox label="monitor:read"> Monitor Read </el-checkbox>
                  <el-checkbox label="log:read"> Log Read </el-checkbox>
                  <el-checkbox label="security:read">
                    Security Read
                  </el-checkbox>
                  <el-checkbox label="admin"> Admin Access </el-checkbox>
                </el-checkbox-group>
              </el-form-item>

              <el-form-item label="最低角色">
                <el-select
                  v-model="permissionsConfig.minimumRole"
                  placeholder="选择最低角色"
                >
                  <el-option label="用户" value="user" />
                  <el-option label="操作员" value="operator" />
                  <el-option label="管理员" value="admin" />
                </el-select>
              </el-form-item>

              <el-form-item label="无权限时隐藏">
                <el-switch
                  v-model="permissionsConfig.hideIfNoAccess"
                  active-text="Yes"
                  inactive-text="No"
                />
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- Preview Section -->
      <div
v-if="showPreview" class="preview-section"
>
        <el-divider>Preview</el-divider>
        <div class="preview-container">
          <div class="preview-widget">
            <!-- Preview implementation would go here -->
            <div class="preview-placeholder">
              <el-icon :size="32">
                <View />
              </el-icon>
              <p>Widget preview will be shown here</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <div class="footer-left">
          <el-button
type="text" @click="resetToDefaults"
>
            恢复默认设置
          </el-button>
          <el-button type="text"
@click="loadPreset"
>
Load Preset
</el-button>
        </div>
        <div class="footer-right">
          <el-button @click="closeDialog"> 取消 </el-button>
          <el-button
type="primary" @click="saveAndClose"
>
            保存更改
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { View } from "@element-plus/icons-vue";

// Component imports
import KeyValueEditor from "@/components/settings/forms/KeyValueEditor.vue";

// Types
import type { DashboardWidget } from "@/store/dashboard";

// Props
interface Props {
  modelValue: boolean;
  widget: DashboardWidget | null;
}

const props = defineProps<Props>();

// Emits
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  save: [widgetId: string, config: any];
}>();

// Reactive state
const activeTab = ref("general");
const showPreview = ref(false);

// Form refs
const generalFormRef = ref();
const appearanceFormRef = ref();
const dataFormRef = ref();
const advancedFormRef = ref();
const permissionsFormRef = ref();

// Configuration objects
const generalConfig = ref({
  title: "",
  refreshInterval: 30000,
  enabled: true,
  draggable: true,
  resizable: true,
});

const appearanceConfig = ref({
  theme: "auto",
  displayMode: "default",
  showHeader: true,
  showFooter: false,
  animations: true,
});

const specificConfig = ref({});

const dataConfig = ref({
  dataSource: "api",
  cacheDuration: 30000,
  maxDataPoints: 100,
  dateRange: "24h",
  filters: {},
});

const advancedConfig = ref({
  errorHandling: "retry",
  retryAttempts: 3,
  retryDelay: 5000,
  debugMode: false,
  performanceMonitoring: true,
  customCSS: "",
  customProperties: {},
});

const permissionsConfig = ref({
  permissions: [] as string[],
  minimumRole: "user",
  hideIfNoAccess: false,
});

// Validation rules
const generalRules = {
  title: [
    { required: true, message: "Widget title is required", trigger: "blur" },
    {
      min: 1,
      max: 50,
      message: "Title must be between 1 and 50 characters",
      trigger: "blur",
    },
  ],
  refreshInterval: [
    {
      required: true,
      message: "Refresh interval is required",
      trigger: "change",
    },
  ],
};

// Computed properties
const widgetIcon = computed(() => {
  if (!props.widget) return "Box";

  const iconMap: Record<string, string> = {
    "system-overview": "Monitor",
    "container-stats": "Box",
    "update-activity": "Refresh",
    "realtime-monitor": "DataLine",
    "health-monitor": "CircleCheckFilled",
    "recent-activities": "Document",
    "quick-actions": "Lightning",
    "notification-center": "Bell",
    "resource-charts": "DataAnalysis",
    "security-dashboard": "Lock",
  };
  return iconMap[props.widget.type] || "Box";
});

const widgetDescription = computed(() => {
  if (!props.widget) return "";

  const descriptionMap: Record<string, string> = {
    "system-overview": "Displays overall system health and key metrics",
    "container-stats": "Shows container statistics and status distribution",
    "update-activity": "Tracks recent updates and update statistics",
    "realtime-monitor": "Provides live system activity and performance metrics",
    "health-monitor": "Monitors service health and availability",
    "recent-activities": "Shows timeline of recent system activities",
    "quick-actions": "Provides quick access to common operations",
    "notification-center": "Displays live notifications and alerts",
    "resource-charts": "Shows historical resource usage charts",
    "security-dashboard": "Monitors security status and vulnerabilities",
  };
  return descriptionMap[props.widget.type] || "Widget configuration";
});

const hasSpecificSettings = computed(() => {
  return props.widget?.type && specificConfigComponents[props.widget.type];
});

const specificTabLabel = computed(() => {
  if (!props.widget) return "Specific";
  return `${props.widget.type
    .split("-")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ")} Settings`;
});

const specificConfigComponent = computed(() => {
  if (!props.widget?.type) return null;
  return specificConfigComponents[props.widget.type] || null;
});

// Widget-specific configuration components
const specificConfigComponents: Record<string, any> = {
  "system-overview": () => import("./config/SystemOverviewConfig.vue"),
  "container-stats": () => import("./config/ContainerStatsConfig.vue"),
  "update-activity": () => import("./config/UpdateActivityConfig.vue"),
  "realtime-monitor": () => import("./config/RealtimeMonitorConfig.vue"),
  "health-monitor": () => import("./config/HealthMonitorConfig.vue"),
  "recent-activities": () => import("./config/RecentActivitiesConfig.vue"),
  "quick-actions": () => import("./config/QuickActionsConfig.vue"),
  "notification-center": () => import("./config/NotificationCenterConfig.vue"),
  "resource-charts": () => import("./config/ResourceChartsConfig.vue"),
  "security-dashboard": () => import("./config/SecurityDashboardConfig.vue"),
};

// Methods
const loadWidgetConfig = () => {
  if (!props.widget) return;

  // Load general config
  generalConfig.value = {
    title: props.widget.title,
    refreshInterval: props.widget.refreshInterval,
    enabled: props.widget.enabled,
    draggable: props.widget.draggable,
    resizable: props.widget.resizable,
  };

  // Load appearance config
  appearanceConfig.value = {
    theme: props.widget.settings.theme || "auto",
    displayMode: props.widget.settings.displayMode || "default",
    showHeader: props.widget.settings.showHeader !== false,
    showFooter: props.widget.settings.showFooter || false,
    animations: props.widget.settings.animations !== false,
  };

  // Load data config
  dataConfig.value = {
    dataSource: props.widget.settings.dataSource || "api",
    cacheDuration: props.widget.settings.cacheDuration || 30000,
    maxDataPoints: props.widget.settings.maxDataPoints || 100,
    dateRange: props.widget.settings.dateRange || "24h",
    filters: props.widget.settings.filters || {},
  };

  // Load advanced config
  advancedConfig.value = {
    errorHandling: props.widget.settings.errorHandling || "retry",
    retryAttempts: props.widget.settings.retryAttempts || 3,
    retryDelay: props.widget.settings.retryDelay || 5000,
    debugMode: props.widget.settings.debugMode || false,
    performanceMonitoring:
      props.widget.settings.performanceMonitoring !== false,
    customCSS: props.widget.settings.customCSS || "",
    customProperties: props.widget.settings.customProperties || {},
  };

  // Load permissions config
  permissionsConfig.value = {
    permissions: [...(props.widget.permissions || [])],
    minimumRole: props.widget.settings.minimumRole || "user",
    hideIfNoAccess: props.widget.settings.hideIfNoAccess || false,
  };

  // Load widget-specific config
  specificConfig.value = { ...props.widget.settings };
};

const validateAllForms = async (): Promise<boolean> => {
  const forms = [
    generalFormRef.value,
    appearanceFormRef.value,
    dataFormRef.value,
    advancedFormRef.value,
    permissionsFormRef.value,
  ].filter(Boolean);

  try {
    await Promise.all(forms.map((form) => form.validate()));
    return true;
  } catch (error) {
    console.error("Form validation failed:", error);
    return false;
  }
};

const saveAndClose = async () => {
  if (!props.widget) return;

  const isValid = await validateAllForms();
  if (!isValid) {
    ElMessage.error("Please fix validation errors before saving");
    return;
  }

  try {
    const updatedConfig = {
      title: generalConfig.value.title,
      refreshInterval: generalConfig.value.refreshInterval,
      enabled: generalConfig.value.enabled,
      draggable: generalConfig.value.draggable,
      resizable: generalConfig.value.resizable,
      permissions: permissionsConfig.value.permissions,
      settings: {
        ...appearanceConfig.value,
        ...dataConfig.value,
        ...advancedConfig.value,
        ...specificConfig.value,
        minimumRole: permissionsConfig.value.minimumRole,
        hideIfNoAccess: permissionsConfig.value.hideIfNoAccess,
      },
    };

    emit("save", props.widget.id, updatedConfig);
    ElMessage.success("Widget configuration saved");
  } catch (error) {
    console.error("Failed to save widget config:", error);
    ElMessage.error("Failed to save configuration");
  }
};

const closeDialog = () => {
  emit("update:modelValue", false);
};

const resetToDefaults = async () => {
  try {
    await ElMessageBox.confirm(
      "This will reset all settings to their default values. Continue?",
      "恢复默认设置",
      {
        type: "warning",
        confirmButtonText: "重置",
        cancelButtonText: "取消",
      },
    );

    loadWidgetConfig();
    ElMessage.success("Settings reset to defaults");
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to reset settings:", error);
    }
  }
};

const loadPreset = async () => {
  try {
    const presets = [
      "Performance Optimized",
      "Detailed View",
      "Minimal View",
      "Admin View",
    ];

    const { value: preset } = await ElMessageBox.prompt(
      "Enter preset name:\n" +
        presets.map((p, i) => `${i + 1}. ${p}`).join("\n"),
      "Load Preset",
      {
        inputValue: presets[0],
        confirmButtonText: "加载",
        cancelButtonText: "取消",
      },
    );

    // Implementation for loading presets would go here
    ElMessage.success(`${preset} preset loaded`);
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to load preset:", error);
    }
  }
};

// Watch for widget changes
watch(
  () => props.widget,
  (newWidget) => {
    if (newWidget) {
      loadWidgetConfig();
    }
  },
  { immediate: true, deep: true },
);

// Watch for dialog open/close
watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen && props.widget) {
      activeTab.value = "general";
      loadWidgetConfig();
    }
  },
);
</script>

<style scoped lang="scss">
.widget-config-dialog {
  .config-container {
    max-height: 70vh;
    overflow-y: auto;
  }

  .widget-info-section {
    margin-bottom: 24px;
    padding: 16px;
    background: var(--el-fill-color-extra-light);
    border-radius: 8px;

    .widget-header {
      display: flex;
      align-items: center;
      gap: 16px;

      .widget-icon {
        flex-shrink: 0;
        color: var(--el-color-primary);
      }

      .widget-details {
        h3 {
          margin: 0 0 4px 0;
          font-size: 18px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        p {
          margin: 0;
          font-size: 14px;
          color: var(--el-text-color-secondary);
          line-height: 1.4;
        }
      }
    }
  }

  .config-tabs {
    .config-section {
      padding: 16px 0;

      .input-suffix {
        margin-left: 8px;
        font-size: 12px;
        color: var(--el-text-color-placeholder);
      }
    }
  }

  .preview-section {
    margin-top: 24px;

    .preview-container {
      padding: 16px;
      background: var(--el-fill-color-extra-light);
      border-radius: 8px;
      border: 1px dashed var(--el-border-color);

      .preview-widget {
        min-height: 200px;
        background: var(--el-bg-color);
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;

        .preview-placeholder {
          text-align: center;
          color: var(--el-text-color-placeholder);

          p {
            margin: 8px 0 0 0;
            font-size: 14px;
          }
        }
      }
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .footer-left {
      display: flex;
      gap: 8px;
    }

    .footer-right {
      display: flex;
      gap: 12px;
    }
  }
}

// Form styling
:deep(.el-form-item) {
  margin-bottom: 18px;

  .el-form-item__label {
    font-weight: 500;
    color: var(--el-text-color-primary);
  }

  .el-form-item__content {
    .el-input,
    .el-select,
    .el-input-number {
      width: 100%;
    }

    .el-checkbox-group {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 8px;
    }
  }
}

:deep(.el-tabs__content) {
  overflow: visible;
}

// Responsive design
@media (max-width: 768px) {
  .widget-config-dialog {
    width: 95vw !important;

    .widget-info-section {
      .widget-header {
        flex-direction: column;
        align-items: flex-start;
        text-align: left;
      }
    }

    .dialog-footer {
      flex-direction: column-reverse;
      gap: 12px;

      .footer-left,
      .footer-right {
        width: 100%;
        justify-content: center;
      }
    }
  }

  :deep(.el-form-item__content) {
    .el-checkbox-group {
      grid-template-columns: 1fr;
    }
  }
}
</style>
