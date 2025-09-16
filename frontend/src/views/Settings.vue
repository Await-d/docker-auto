<template>
  <div class="settings-view">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-main">
          <h1 class="page-title">
            <el-icon><Setting /></el-icon>
            System Settings
          </h1>
          <p class="page-description">
            Configure and manage your Docker Auto-Update System
          </p>
        </div>

        <div class="header-actions">
          <!-- Search -->
          <el-input
            v-model="searchQuery"
            placeholder="Search settings..."
            class="search-input"
            :prefix-icon="Search"
            clearable
            @input="handleSearch"
          />

          <!-- Global Actions -->
          <el-dropdown trigger="click" @command="handleGlobalAction">
            <el-button type="primary">
              Actions
              <el-icon><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="export">
                  <el-icon><Download /></el-icon>
                  Export Settings
                </el-dropdown-item>
                <el-dropdown-item command="import">
                  <el-icon><Upload /></el-icon>
                  Import Settings
                </el-dropdown-item>
                <el-dropdown-item divided command="reset-all">
                  <el-icon><RefreshLeft /></el-icon>
                  Reset All to Defaults
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
            Save All
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
          <span>You have unsaved changes</span>
          <el-button
            type="text"
            size="small"
            @click="discardAllChanges"
            class="discard-btn"
          >
            Discard All
          </el-button>
        </template>
      </el-alert>
    </div>

    <!-- Settings Content -->
    <div class="settings-content">
      <!-- Sidebar Navigation -->
      <div class="settings-sidebar">
        <div class="sidebar-header">
          <h3>Settings Categories</h3>
        </div>

        <div class="sidebar-content">
          <div class="settings-sections">
            <div
              v-for="section in visibleSections"
              :key="section.key"
              :class="[
                'section-item',
                { 'active': currentSection === section.key },
                { 'has-changes': section.hasChanges },
                { 'has-errors': !section.isValid }
              ]"
              @click="selectSection(section.key)"
            >
              <div class="section-icon">
                <component :is="section.icon" />
              </div>

              <div class="section-info">
                <h4 class="section-title">{{ section.title }}</h4>
                <p class="section-description">{{ section.description }}</p>
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
            <p class="section-description">{{ currentSectionInfo?.description }}</p>
          </div>

          <div class="section-actions">
            <el-button
              v-if="currentSectionHasChanges"
              @click="resetCurrentSection"
              :disabled="saving"
            >
              <el-icon><RefreshLeft /></el-icon>
              Reset
            </el-button>

            <el-button
              type="primary"
              :disabled="!canSaveCurrentSection"
              :loading="saving"
              @click="saveCurrentSection"
            >
              <el-icon><Check /></el-icon>
              Save Section
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
            v-else-if="settings && currentSectionComponent"
            :is="currentSectionComponent"
            v-model="currentSectionData"
            :loading="saving"
            :validation-errors="currentSectionValidationErrors"
            @field-change="handleFieldChange"
            @field-validate="handleFieldValidate"
            @test-configuration="handleTestConfiguration"
          />

          <el-empty
            v-else
            description="Select a settings section to configure"
            class="empty-state"
          />
        </div>
      </div>
    </div>

    <!-- Import Dialog -->
    <el-dialog
      v-model="importDialogVisible"
      title="Import Settings"
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
            Importing settings will overwrite current configuration
          </template>
          This action cannot be undone. Make sure to export your current settings as a backup.
        </el-alert>

        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :show-file-list="true"
          :limit="1"
          accept=".json"
          @change="handleFileSelect"
          @remove="handleFileRemove"
          class="import-upload"
        >
          <el-button type="primary">
            <el-icon><FolderOpened /></el-icon>
            Select Settings File
          </el-button>
          <template #tip>
            <div class="upload-tip">
              Only JSON files exported from this system are supported
            </div>
          </template>
        </el-upload>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="importDialogVisible = false">Cancel</el-button>
          <el-button
            type="primary"
            :disabled="!selectedFile"
            :loading="importing"
            @click="confirmImport"
          >
            Import Settings
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Setting,
  Search,
  ArrowDown,
  Download,
  Upload,
  RefreshLeft,
  Check,
  Warning,
  FolderOpened
} from '@element-plus/icons-vue'

// Import settings components
import SystemConfig from '@/components/settings/SystemConfig.vue'
import DockerConfig from '@/components/settings/DockerConfig.vue'
import UpdatePolicies from '@/components/settings/UpdatePolicies.vue'
import RegistryConfig from '@/components/settings/RegistryConfig.vue'
import UserManagement from '@/components/settings/UserManagement.vue'
import NotificationConfig from '@/components/settings/NotificationConfig.vue'
import SchedulerConfig from '@/components/settings/SchedulerConfig.vue'
import SecurityConfig from '@/components/settings/SecurityConfig.vue'
import MonitoringConfig from '@/components/settings/MonitoringConfig.vue'

import { useSettings } from '@/store/settings'
import { useAuthStore } from '@/store/auth'
import { useApp } from '@/store/app'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const app = useApp()
const {
  settings,
  loading,
  saving,
  isDirty,
  canSave,
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
  setValidationErrors
} = useSettings()

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
  monitoring: MonitoringConfig
}

// Local state
const importDialogVisible = ref(false)
const importing = ref(false)
const selectedFile = ref<File | null>(null)
const uploadRef = ref()

// Computed properties
const visibleSections = computed(() => {
  return filteredSections.value.filter(section => {
    // Check permissions for each section
    return section.permissions.every(permission => {
      if (permission === 'admin') {
        return auth.hasRole('admin')
      }
      return auth.hasPermission(permission)
    })
  })
})

const currentSectionInfo = computed(() => {
  return settingsSections.value.find(section => section.key === currentSection.value)
})

const currentSectionComponent = computed(() => {
  return sectionComponents[currentSection.value as keyof typeof sectionComponents]
})

const currentSectionData = computed({
  get() {
    if (!settings.value || !currentSection.value) return null
    return settings.value[currentSection.value as keyof typeof settings.value]
  },
  set(value) {
    if (value && currentSection.value) {
      updateField(currentSection.value, value)
    }
  }
})

const currentSectionHasChanges = computed(() => {
  return currentSectionInfo.value?.hasChanges || false
})

const currentSectionValidationErrors = computed(() => {
  const prefix = `${currentSection.value}.`
  const errors: Record<string, string[]> = {}

  Object.entries(validationErrors.value).forEach(([field, fieldErrors]) => {
    if (field.startsWith(prefix)) {
      const localField = field.replace(prefix, '')
      errors[localField] = fieldErrors
    }
  })

  return errors
})

const canSaveCurrentSection = computed(() => {
  return currentSectionHasChanges.value &&
         currentSectionInfo.value?.isValid &&
         !saving.value
})

const canSaveGlobal = computed(() => {
  return isDirty.value && !saving.value
})

// Methods
const selectSection = (sectionKey: string) => {
  if (isDirty.value) {
    ElMessageBox.confirm(
      'You have unsaved changes. Do you want to continue?',
      'Unsaved Changes',
      {
        confirmButtonText: 'Continue',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    ).then(() => {
      setCurrentSection(sectionKey)
      updateRouteSection(sectionKey)
    }).catch(() => {
      // User cancelled
    })
  } else {
    setCurrentSection(sectionKey)
    updateRouteSection(sectionKey)
  }
}

const updateRouteSection = (sectionKey: string) => {
  router.replace({
    name: 'Settings',
    query: { section: sectionKey }
  })
}

const handleSearch = (query: string) => {
  setSearchQuery(query)
}

const handleFieldChange = (field: string, value: any) => {
  const fullField = `${currentSection.value}.${field}`
  updateField(fullField, value)
}

const handleFieldValidate = (field: string, value: any) => {
  const fullField = `${currentSection.value}.${field}`
  const errors = validateField(fullField, value)
  setValidationErrors(fullField, errors)
}

const handleTestConfiguration = async (config: any) => {
  try {
    app.showInfo('Testing configuration...')
    // Implementation would depend on the specific section
    const result = await testConfiguration(currentSection.value, config)
    app.showSuccess('Configuration test successful')
    return result
  } catch (error) {
    app.showError('Configuration test failed')
    throw error
  }
}

const saveCurrentSection = async () => {
  try {
    await saveSettings(currentSection.value)
  } catch (error) {
    console.error('Failed to save section:', error)
  }
}

const saveAllSettings = async () => {
  try {
    await saveSettings()
  } catch (error) {
    console.error('Failed to save all settings:', error)
  }
}

const resetCurrentSection = async () => {
  try {
    await ElMessageBox.confirm(
      `Reset ${currentSectionInfo.value?.title} to saved values?`,
      'Reset Section',
      {
        confirmButtonText: 'Reset',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    await resetSettings(currentSection.value)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to reset section:', error)
    }
  }
}

const discardAllChanges = async () => {
  try {
    await ElMessageBox.confirm(
      'Discard all unsaved changes?',
      'Discard Changes',
      {
        confirmButtonText: 'Discard',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    await resetSettings()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to discard changes:', error)
    }
  }
}

const handleGlobalAction = async (command: string) => {
  switch (command) {
    case 'export':
      await handleExport()
      break
    case 'import':
      importDialogVisible.value = true
      break
    case 'reset-all':
      await handleResetAll()
      break
  }
}

const handleExport = async () => {
  try {
    await exportSettings()
  } catch (error) {
    console.error('Failed to export settings:', error)
  }
}

const handleResetAll = async () => {
  try {
    await ElMessageBox.confirm(
      'Reset ALL settings to factory defaults? This cannot be undone.',
      'Reset All Settings',
      {
        confirmButtonText: 'Reset All',
        cancelButtonText: 'Cancel',
        type: 'error'
      }
    )

    // This would call a special API endpoint to reset to defaults
    // await resetToDefaults()
    app.showWarning('Reset to defaults functionality not yet implemented')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to reset all settings:', error)
    }
  }
}

const handleFileSelect = (file: any) => {
  selectedFile.value = file.raw
}

const handleFileRemove = () => {
  selectedFile.value = null
}

const resetImportDialog = () => {
  selectedFile.value = null
  uploadRef.value?.clearFiles()
}

const confirmImport = async () => {
  if (!selectedFile.value) return

  try {
    importing.value = true
    await importSettings(selectedFile.value)
    importDialogVisible.value = false
    resetImportDialog()

    // Reload settings after import
    await loadSettings()
  } catch (error) {
    console.error('Failed to import settings:', error)
  } finally {
    importing.value = false
  }
}

// Initialize settings when component mounts
onMounted(async () => {
  try {
    await loadSettings()

    // Set initial section from route query
    const sectionFromRoute = route.query.section as string
    if (sectionFromRoute && settingsSections.value.some(s => s.key === sectionFromRoute)) {
      setCurrentSection(sectionFromRoute)
    } else {
      setCurrentSection('general')
    }
  } catch (error) {
    console.error('Failed to load settings:', error)
  }
})

// Watch for route changes
watch(() => route.query.section, (newSection) => {
  if (newSection && typeof newSection === 'string') {
    setCurrentSection(newSection)
  }
})
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
  width: 320px;
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
@media (max-width: 1024px) {
  .settings-content {
    flex-direction: column;
  }

  .settings-sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--el-border-color-light);
    max-height: 200px;
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
}
</style>