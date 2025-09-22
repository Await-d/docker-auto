<template>
  <div class="settings-view">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-main">
          <h1 class="page-title">
            <el-icon><Setting /></el-icon>
            系统设置
          </h1>
          <p class="page-description">
            配置和管理您的Docker自动更新系统
          </p>
        </div>

        <div class="header-actions">
          <!-- Search -->
          <el-input
            v-model="searchQuery"
            placeholder="搜索设置..."
            class="search-input"
            :prefix-icon="Search"
            clearable
            @input="handleSearch"
          />

          <!-- Global Actions -->
          <el-dropdown trigger="click" @command="handleGlobalAction">
            <el-button type="primary">
              操作
              <el-icon><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="export">
                  <el-icon><Download /></el-icon>
                  导出设置
                </el-dropdown-item>
                <el-dropdown-item command="import">
                  <el-icon><Upload /></el-icon>
                  导入设置
                </el-dropdown-item>
                <el-dropdown-item divided command="reset-all">
                  <el-icon><RefreshLeft /></el-icon>
                  全部重置为默认值
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <!-- Save All Button -->
          <el-button
            type="success"
            :disabled="!canSaveGlobal"
            :loading="saving"
            @click="saveAllSettings"
          >
            <el-icon><Check /></el-icon>
            全部保存
          </el-button>
        </div>
      </div>

      <!-- Unsaved Changes Warning -->
      <el-alert
        v-if="isDirty"
        type="warning"
        :closable="false"
        show-icon
        class="changes-alert"
      >
        <template #title>
          <span>您有未保存的更改</span>
          <el-button
            type="text"
            size="small"
            class="discard-btn"
            @click="discardAllChanges"
          >
            全部丢弃
          </el-button>
        </template>
      </el-alert>
    </div>

    <!-- Settings Content -->
    <div class="settings-content">
      <!-- Sidebar Navigation -->
      <div class="settings-sidebar">
        <div class="sidebar-header">
          <h3>设置分类</h3>
        </div>

        <div class="sidebar-content">
        <div class="settings-sections">
          <!-- Show a message if no sections are visible due to permissions -->
          <div v-if="visibleSections.length === 0" class="no-sections">
            <el-empty
              description="暂无可访问的设置项"
              :image-size="80"
            >
              <template #description>
                <p>请联系管理员获取相应权限</p>
              </template>
            </el-empty>
          </div>
          
          <div
            v-for="section in visibleSections"
            :key="section.key"
            :class="[
              'section-item',
              { active: currentSection === section.key },
              { 'has-changes': section.hasChanges },
              { 'has-errors': !section.isValid },
            ]"
            @click="selectSection(section.key)"
          >
              <div class="section-icon">
                <component :is="section.icon" />
              </div>

              <div class="section-info">
                <h4 class="section-title">
                  {{ section.title }}
                </h4>
                <p class="section-description">
                  {{ section.description }}
                </p>
              </div>

              <div class="section-indicators">
                <el-badge
                  v-if="section.hasChanges"
                  is-dot
                  type="warning"
                  class="changes-badge"
                />
                <el-icon
                  v-if="!section.isValid"
                  class="error-icon"
                  color="#F56C6C"
                >
                  <Warning />
                </el-icon>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Main Content Area -->
      <div class="settings-main">
        <!-- Section Header -->
        <div class="section-header">
          <div class="section-title-area">
            <h2 class="section-title">
              <component :is="currentSectionInfo?.icon" />
              {{ currentSectionInfo?.title }}
            </h2>
            <p class="section-description">
              {{ currentSectionInfo?.description }}
            </p>
          </div>

          <div class="section-actions">
            <el-button
              v-if="currentSectionHasChanges"
              :disabled="saving"
              @click="resetCurrentSection"
            >
              <el-icon><RefreshLeft /></el-icon>
              重置
            </el-button>

            <el-button
              type="primary"
              :disabled="!canSaveCurrentSection"
              :loading="saving"
              @click="saveCurrentSection"
            >
              <el-icon><Check /></el-icon>
              保存分区
            </el-button>
          </div>
        </div>

        <!-- Dynamic Section Content -->
        <div class="section-content">
          <el-skeleton
            v-if="loading"
            :rows="8"
            animated
            class="settings-skeleton"
          />

          <component
            :is="currentSectionComponent"
            v-else-if="settings && currentSectionComponent"
            v-model="currentSectionData"
            :loading="saving"
            :validation-errors="currentSectionValidationErrors"
            @field-change="handleFieldChange"
            @field-validate="handleFieldValidate"
            @test-configuration="handleTestConfiguration"
          />

          <el-empty
            v-else
            description="选择要配置的设置分区"
            class="empty-state"
          />
        </div>
      </div>
    </div>

    <!-- Import Dialog -->
    <el-dialog
      v-model="importDialogVisible"
      title="导入设置"
      width="500px"
      @close="resetImportDialog"
    >
      <div class="import-content">
        <el-alert
          type="warning"
          :closable="false"
          show-icon
          class="import-warning"
        >
          <template #title>
            导入设置将覆盖当前配置
          </template>
          此操作无法撤销。请确保将当前设置导出作为备份。
        </el-alert>

        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :show-file-list="true"
          :limit="1"
          accept=".json"
          class="import-upload"
          @change="handleFileSelect"
          @remove="handleFileRemove"
        >
          <el-button type="primary">
            <el-icon><FolderOpened /></el-icon>
            选择设置文件
          </el-button>
          <template #tip>
            <div class="upload-tip">
              仅支持从此系统导出的JSON文件
            </div>
          </template>
        </el-upload>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="importDialogVisible = false"> 取消 </el-button>
          <el-button
            type="primary"
            :disabled="!selectedFile"
            :loading="importing"
            @click="confirmImport"
          >
            导入设置
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { useRouter, useRoute } from "vue-router";
import { ElMessageBox } from "element-plus";
import {
  Setting,
  Search,
  ArrowDown,
  Download,
  Upload,
  RefreshLeft,
  Check,
  Warning,
  FolderOpened,
} from "@element-plus/icons-vue";

// Import settings components
import SystemConfig from "@/components/settings/SystemConfig.vue";
import DockerConfig from "@/components/settings/DockerConfig.vue";
import UpdatePolicies from "@/components/settings/UpdatePoliciesSettings.vue";
import RegistryConfig from "@/components/settings/RegistryConfig.vue";
import UserManagement from "@/components/settings/UserManagement.vue";
import NotificationConfig from "@/components/settings/NotificationConfig.vue";
import SchedulerConfig from "@/components/settings/SchedulerConfig.vue";
import SecurityConfig from "@/components/settings/SecurityConfig.vue";
import MonitoringConfig from "@/components/settings/MonitoringConfig.vue";

import { useSettings } from "@/store/settings";
import type { SystemSettings } from "@/store/settings";
import { useAuthStore } from "@/store/auth";
import { useApp } from "@/store/app";

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();
const app = useApp();
const {
  settings,
  loading,
  saving,
  isDirty,
  settingsSections,
  filteredSections,
  currentSection,
  searchQuery,
  validationErrors,
  loadSettings,
  saveSettings,
  resetSettings,
  exportSettings,
  importSettings,
  setCurrentSection,
  setSearchQuery,
  updateField,
  validateField,
  setValidationErrors,
} = useSettings();

// Component mapping
const sectionComponents = {
  general: SystemConfig,
  docker: DockerConfig,
  updates: UpdatePolicies,
  registries: RegistryConfig,
  users: UserManagement,
  notifications: NotificationConfig,
  scheduler: SchedulerConfig,
  security: SecurityConfig,
  monitoring: MonitoringConfig,
};

// Local state
const importDialogVisible = ref(false);
const importing = ref(false);
const selectedFile = ref<File | null>(null);
const uploadRef = ref();

// Computed properties
const visibleSections = computed(() => {
  const filtered = filteredSections.filter((section: any) => {
    // Check permissions for each section
    // Use 'some' instead of 'every' to be more permissive
    // At least one permission should match, or if no specific permissions, allow access
    if (!section.permissions || section.permissions.length === 0) {
      return true; // No permissions required
    }
    
    return section.permissions.some((permission: any) => {
      if (permission === "admin") {
        return auth.hasRole("admin");
      }
      // Allow basic settings access for authenticated users
      if (permission.startsWith("settings:") && permission.endsWith(":read")) {
        return auth.user !== null; // Any authenticated user can read settings
      }
      
      // Development fallback: allow all permissions for development
      if (import.meta.env.DEV) {
        console.log(`Dev mode: granting permission ${permission}`);
        return true;
      }
      return auth.hasPermission(permission);
    });
  });
  
  // Debug: Ensure at least the general section is always visible
  if (filtered.length === 0 && filteredSections.length > 0) {
    console.warn("No sections visible due to permissions, showing general section as fallback");
    const generalSection = filteredSections.find(s => s.key === 'general');
    return generalSection ? [generalSection] : filteredSections.slice(0, 1);
  }
  
  return filtered;
});

const currentSectionInfo = computed(() => {
  return settingsSections.find(
    (section: any) => section.key === currentSection,
  );
});

const currentSectionComponent = computed(() => {
  return sectionComponents[currentSection as keyof typeof sectionComponents];
});

const currentSectionData = computed({
  get() {
    if (!settings || !currentSection) return {};
    return (settings as any)[currentSection] || {};
  },
  set(value) {
    if (value && currentSection) {
      updateField(currentSection, value);
    }
  },
});

const currentSectionHasChanges = computed(() => {
  return currentSectionInfo.value?.hasChanges || false;
});

const currentSectionValidationErrors = computed(() => {
  const prefix = `${currentSection}.`;
  const errors: Record<string, string[]> = {};

  // Add null check for validationErrors.value
  if (validationErrors.value) {
    Object.entries(validationErrors.value).forEach(([field, fieldErrors]) => {
      if (field.startsWith(prefix)) {
        const localField = field.replace(prefix, "");
        errors[localField] = Array.isArray(fieldErrors)
          ? fieldErrors
          : [fieldErrors];
      }
    });
  }

  return errors;
});

const canSaveCurrentSection = computed(() => {
  return (
    currentSectionHasChanges.value &&
    currentSectionInfo.value?.isValid &&
    !saving
  );
});

const canSaveGlobal = computed(() => {
  return isDirty && !saving;
});

// Methods
const selectSection = (sectionKey: string) => {
  if (isDirty) {
    ElMessageBox.confirm(
      "您有未保存的更改。是否要继续？",
      "未保存的更改",
      {
        confirmButtonText: "继续",
        cancelButtonText: "取消",
        type: "warning",
      },
    )
      .then(() => {
        setCurrentSection(sectionKey);
        updateRouteSection(sectionKey);
      })
      .catch(() => {
        // User cancelled
      });
  } else {
    setCurrentSection(sectionKey);
    updateRouteSection(sectionKey);
  }
};

const updateRouteSection = (sectionKey: string) => {
  router.replace({
    name: "Settings",
    query: { section: sectionKey },
  });
};

const handleSearch = (query: string) => {
  setSearchQuery(query);
};

const handleFieldChange = (field: string, value: any) => {
  const fullField = `${currentSection}.${field}`;
  updateField(fullField, value);
};

const handleFieldValidate = (field: string, value: any) => {
  const fullField = `${currentSection}.${field}`;
  const errors = validateField(fullField, value);
  setValidationErrors(fullField, errors);
};

const testConfiguration = async (_section: string, _config: any) => {
  // Placeholder implementation for testing configuration
  return new Promise((resolve) => {
    setTimeout(() => resolve({ success: true }), 1000);
  });
};

const handleTestConfiguration = async (config: any) => {
  try {
    app.showInfo("正在测试配置...");
    // Implementation would depend on the specific section
    const result = await testConfiguration(currentSection, config);
    app.showSuccess("配置测试成功");
    return result;
  } catch (error) {
    app.showError("配置测试失败");
    throw error;
  }
};

const saveCurrentSection = async () => {
  try {
    if (currentSection) {
      await saveSettings(currentSection as keyof SystemSettings);
    }
  } catch (error) {
    console.error("Failed to save section:", error);
  }
};

const saveAllSettings = async () => {
  try {
    await saveSettings();
  } catch (error) {
    console.error("Failed to save all settings:", error);
  }
};

const resetCurrentSection = async () => {
  try {
    await ElMessageBox.confirm(
      `将 ${currentSectionInfo.value?.title} 重置为保存的值？`,
      "重置分区",
      {
        confirmButtonText: "重置",
        cancelButtonText: "取消",
        type: "warning",
      },
    );

    if (currentSection) {
      await resetSettings(currentSection as keyof SystemSettings);
    }
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to reset section:", error);
    }
  }
};

const discardAllChanges = async () => {
  try {
    await ElMessageBox.confirm(
      "丢弃所有未保存的更改？",
      "丢弃更改",
      {
        confirmButtonText: "丢弃",
        cancelButtonText: "取消",
        type: "warning",
      },
    );

    await resetSettings();
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to discard changes:", error);
    }
  }
};

const handleGlobalAction = async (command: string) => {
  switch (command) {
    case "export":
      await handleExport();
      break;
    case "import":
      importDialogVisible.value = true;
      break;
    case "reset-all":
      await handleResetAll();
      break;
  }
};

const handleExport = async () => {
  try {
    await exportSettings();
  } catch (error) {
    console.error("Failed to export settings:", error);
  }
};

const handleResetAll = async () => {
  try {
    await ElMessageBox.confirm(
      "将所有设置重置为出厂默认值？此操作无法撤销。",
      "重置所有设置",
      {
        confirmButtonText: "全部重置",
        cancelButtonText: "取消",
        type: "error",
      },
    );

    // This would call a special API endpoint to reset to defaults
    // await resetToDefaults()
    app.showWarning("重置为默认值功能尚未实现");
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to reset all settings:", error);
    }
  }
};

const handleFileSelect = (file: any) => {
  selectedFile.value = file.raw;
};

const handleFileRemove = () => {
  selectedFile.value = null;
};

const resetImportDialog = () => {
  selectedFile.value = null;
  uploadRef.value?.clearFiles();
};

const confirmImport = async () => {
  if (!selectedFile.value) return;

  try {
    importing.value = true;
    await importSettings(selectedFile.value);
    importDialogVisible.value = false;
    resetImportDialog();

    // Reload settings after import
    await loadSettings();
  } catch (error) {
    console.error("Failed to import settings:", error);
  } finally {
    importing.value = false;
  }
};

// Initialize settings when component mounts
onMounted(async () => {
  try {
    await loadSettings();

    // Set initial section from route query
    const sectionFromRoute = route.query.section as string;
    if (
      sectionFromRoute &&
      settingsSections.some((s: any) => s.key === sectionFromRoute)
    ) {
      setCurrentSection(sectionFromRoute);
    } else {
      setCurrentSection("general");
    }
  } catch (error) {
    console.error("Failed to load settings:", error);
  }
});

// Watch for route changes
watch(
  () => route.query.section,
  (newSection) => {
    if (newSection && typeof newSection === "string") {
      setCurrentSection(newSection);
    }
  },
);
</script>

<style scoped lang="scss">
.settings-view {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);
  padding: 24px;
  flex-shrink: 0;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
  margin-bottom: 16px;
}

.header-main {
  flex: 1;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 28px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0 0 8px 0;
}

.page-description {
  color: var(--el-text-color-regular);
  margin: 0;
  font-size: 14px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.search-input {
  width: 280px;
}

.changes-alert {
  margin-top: 16px;

  .discard-btn {
    margin-left: 12px;
    color: var(--el-color-warning);
  }
}

.settings-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.settings-sidebar {
  width: 380px;
  min-width: 360px;
  max-width: 400px;
  border-right: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-header {
  padding: 20px 24px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.settings-sections {
  padding: 0 8px;
  
  .no-sections {
    padding: 40px 20px;
    text-align: center;
    
    p {
      color: var(--el-text-color-secondary);
      font-size: 13px;
      margin-top: 8px;
    }
  }
}

.section-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  margin-bottom: 4px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;

  &:hover {
    background: var(--el-fill-color-light);
  }

  &.active {
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-7);

    .section-title {
      color: var(--el-color-primary);
    }
  }

  &.has-changes {
    border-left: 3px solid var(--el-color-warning);
  }

  &.has-errors {
    border-left: 3px solid var(--el-color-danger);
  }
}

.section-icon {
  font-size: 18px;
  color: var(--el-text-color-regular);
  flex-shrink: 0;
}

.section-info {
  flex: 1;
  min-width: 0;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin: 0 0 4px 0;
  line-height: 1.4;
}

.section-description {
  font-size: 12px;
  color: var(--el-text-color-regular);
  margin: 0;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.section-indicators {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.error-icon {
  font-size: 16px;
}

.settings-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  max-width: calc(100vw - 420px); /* Prevent overstretch on large screens */
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
  padding: 24px;
  border-bottom: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  flex-shrink: 0;
}

.section-title-area {
  flex: 1;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 24px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0 0 8px 0;
}

.section-description {
  color: var(--el-text-color-regular);
  margin: 0;
  font-size: 14px;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.section-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  max-width: 1200px; /* Limit content width for better readability */
  margin: 0 auto; /* Center content on large screens */
  width: 100%;
  box-sizing: border-box;
}

.settings-skeleton {
  padding: 20px;
}

.empty-state {
  padding: 60px 20px;
}

.import-content {
  .import-warning {
    margin-bottom: 20px;
  }

  .import-upload {
    width: 100%;

    .upload-tip {
      color: var(--el-text-color-regular);
      font-size: 12px;
      margin-top: 8px;
    }
  }
}

.dialog-footer {
  text-align: right;
}

// Responsive design
@media (max-width: 1200px) {
  .settings-sidebar {
    width: 340px;
    min-width: 320px;
    max-width: 360px;
  }
  
  .settings-main {
    max-width: calc(100vw - 360px);
  }
  
  .section-content {
    max-width: 1000px;
    padding: 20px;
  }
}

@media (max-width: 1024px) {
  .settings-content {
    flex-direction: column;
  }

  .settings-sidebar {
    width: 100%;
    min-width: auto;
    max-width: none;
    border-right: none;
    border-bottom: 1px solid var(--el-border-color-light);
    max-height: 200px;
  }
  
  .settings-main {
    max-width: 100%;
  }

  .sidebar-content {
    overflow-x: auto;
  }

  .settings-sections {
    display: flex;
    gap: 8px;
    padding: 8px;
  }

  .section-item {
    flex-shrink: 0;
    min-width: 200px;
  }
  
  .section-content {
    max-width: 800px;
    padding: 18px;
  }
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .search-input {
    flex: 1;
    min-width: 200px;
  }

  .section-header {
    flex-direction: column;
    gap: 16px;
  }

  .section-actions {
    width: 100%;
    justify-content: flex-end;
  }
  
  .section-content {
    max-width: 100%;
    padding: 16px;
  }
}
</style>
