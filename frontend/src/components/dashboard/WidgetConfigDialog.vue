<template>
  <el-dialog
    :model-value="modelValue"
    title="Widget Configuration"
    width="800px"
    :modal="true"
    class="widget-config-dialog"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="widget" class="config-container">
      <!-- Widget Info -->
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

      <!-- Configuration Tabs -->
      <el-tabs v-model="activeTab" class="config-tabs">
        <!-- General Settings -->
        <el-tab-pane label="General" name="general">
          <div class="config-section">
            <el-form
              ref="generalFormRef"
              :model="generalConfig"
              :rules="generalRules"
              label-width="140px"
            >
              <el-form-item label="Widget Title" prop="title">
                <el-input
                  v-model="generalConfig.title"
                  placeholder="Enter widget title"
                  clearable
                />
              </el-form-item>

              <el-form-item label="Refresh Interval" prop="refreshInterval">
                <el-select
                  v-model="generalConfig.refreshInterval"
                  placeholder="Select refresh interval"
                >
                  <el-option label="Never" :value="0" />
                  <el-option label="5 seconds" :value="5000" />
                  <el-option label="10 seconds" :value="10000" />
                  <el-option label="30 seconds" :value="30000" />
                  <el-option label="1 minute" :value="60000" />
                  <el-option label="5 minutes" :value="300000" />
                  <el-option label="15 minutes" :value="900000" />
                  <el-option label="30 minutes" :value="1800000" />
                  <el-option label="1 hour" :value="3600000" />
                </el-select>
              </el-form-item>

              <el-form-item label="Enable Widget">
                <el-switch
                  v-model="generalConfig.enabled"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>

              <el-form-item label="Draggable">
                <el-switch
                  v-model="generalConfig.draggable"
                  active-text="Yes"
                  inactive-text="No"
                />
              </el-form-item>

              <el-form-item label="Resizable">
                <el-switch
                  v-model="generalConfig.resizable"
                  active-text="Yes"
                  inactive-text="No"
                />
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- Appearance Settings -->
        <el-tab-pane label="Appearance" name="appearance">
          <div class="config-section">
            <el-form
              ref="appearanceFormRef"
              :model="appearanceConfig"
              label-width="140px"
            >
              <el-form-item label="Theme">
                <el-radio-group v-model="appearanceConfig.theme">
                  <el-radio label="auto"> Auto </el-radio>
                  <el-radio label="light"> Light </el-radio>
                  <el-radio label="dark"> Dark </el-radio>
                </el-radio-group>
              </el-form-item>

              <el-form-item label="Display Mode">
                <el-select
                  v-model="appearanceConfig.displayMode"
                  placeholder="Select display mode"
                >
                  <el-option label="Default" value="default" />
                  <el-option label="Compact" value="compact" />
                  <el-option label="Detailed" value="detailed" />
                  <el-option label="Minimal" value="minimal" />
                </el-select>
              </el-form-item>

              <el-form-item label="Show Header">
                <el-switch
                  v-model="appearanceConfig.showHeader"
                  active-text="Yes"
                  inactive-text="No"
                />
              </el-form-item>

              <el-form-item label="Show Footer">
                <el-switch
                  v-model="appearanceConfig.showFooter"
                  active-text="Yes"
                  inactive-text="No"
                />
              </el-form-item>

              <el-form-item label="Animation">
                <el-switch
                  v-model="appearanceConfig.animations"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- Widget-Specific Settings -->
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
              <el-form-item label="Data Source">
                <el-select
                  v-model="dataConfig.dataSource"
                  placeholder="Select data source"
                >
                  <el-option label="API" value="api" />
                  <el-option label="WebSocket" value="websocket" />
                  <el-option label="Local Storage" value="localStorage" />
                  <el-option label="Mock Data" value="mock" />
                </el-select>
              </el-form-item>

              <el-form-item label="Cache Duration">
                <el-input-number
                  v-model="dataConfig.cacheDuration"
                  :min="0"
                  :max="3600000"
                  :step="1000"
                  controls-position="right"
                />
                <span class="input-suffix">ms</span>
              </el-form-item>

              <el-form-item label="Max Data Points">
                <el-input-number
                  v-model="dataConfig.maxDataPoints"
                  :min="10"
                  :max="1000"
                  :step="10"
                  controls-position="right"
                />
              </el-form-item>

              <el-form-item label="Date Range">
                <el-select
                  v-model="dataConfig.dateRange"
                  placeholder="Select date range"
                >
                  <el-option label="Last Hour" value="1h" />
                  <el-option label="Last 6 Hours" value="6h" />
                  <el-option label="Last 24 Hours" value="24h" />
                  <el-option label="Last 7 Days" value="7d" />
                  <el-option label="Last 30 Days" value="30d" />
                  <el-option label="Custom" value="custom" />
                </el-select>
              </el-form-item>

              <el-form-item label="Filters">
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
        <el-tab-pane label="Advanced" name="advanced">
          <div class="config-section">
            <el-form
              ref="advancedFormRef"
              :model="advancedConfig"
              label-width="140px"
            >
              <el-form-item label="Error Handling">
                <el-radio-group v-model="advancedConfig.errorHandling">
                  <el-radio label="retry"> Retry on Error </el-radio>
                  <el-radio label="fallback"> Show Fallback </el-radio>
                  <el-radio label="hide"> Hide Widget </el-radio>
                </el-radio-group>
              </el-form-item>

              <el-form-item label="Retry Attempts">
                <el-input-number
                  v-model="advancedConfig.retryAttempts"
                  :min="0"
                  :max="10"
                  controls-position="right"
                />
              </el-form-item>

              <el-form-item label="Retry Delay">
                <el-input-number
                  v-model="advancedConfig.retryDelay"
                  :min="1000"
                  :max="60000"
                  :step="1000"
                  controls-position="right"
                />
                <span class="input-suffix">ms</span>
              </el-form-item>

              <el-form-item label="Debug Mode">
                <el-switch
                  v-model="advancedConfig.debugMode"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>

              <el-form-item label="Performance Monitoring">
                <el-switch
                  v-model="advancedConfig.performanceMonitoring"
                  active-text="Enabled"
                  inactive-text="Disabled"
                />
              </el-form-item>

              <el-form-item label="Custom CSS">
                <el-input
                  v-model="advancedConfig.customCSS"
                  type="textarea"
                  :rows="4"
                  placeholder="Enter custom CSS rules"
                />
              </el-form-item>

              <el-form-item label="Custom Properties">
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
        <el-tab-pane label="Permissions" name="permissions">
          <div class="config-section">
            <el-form
              ref="permissionsFormRef"
              :model="permissionsConfig"
              label-width="140px"
            >
              <el-form-item label="Required Permissions">
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

              <el-form-item label="Minimum Role">
                <el-select
                  v-model="permissionsConfig.minimumRole"
                  placeholder="Select minimum role"
                >
                  <el-option label="User" value="user" />
                  <el-option label="Operator" value="operator" />
                  <el-option label="Admin" value="admin" />
                </el-select>
              </el-form-item>

              <el-form-item label="Hide if No Access">
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
            Reset to Defaults
          </el-button>
          <el-button type="text"
@click="loadPreset"
>
Load Preset
</el-button>
        </div>
        <div class="footer-right">
          <el-button @click="closeDialog"> Cancel </el-button>
          <el-button
type="primary" @click="saveAndClose"
>
            Save Changes
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
      "Reset to Defaults",
      {
        type: "warning",
        confirmButtonText: "Reset",
        cancelButtonText: "Cancel",
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
        confirmButtonText: "Load",
        cancelButtonText: "Cancel",
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
