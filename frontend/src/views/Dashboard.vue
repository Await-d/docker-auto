<template>
  <div class="dashboard-container">
    <!-- Dashboard Header -->
    <div class="dashboard-header">
      <div class="header-left">
        <h1 class="dashboard-title">
          <el-icon><Monitor /></el-icon>
          Dashboard
        </h1>
        <div class="layout-selector">
          <el-select
            v-model="currentLayoutId"
            @change="switchLayout"
            placeholder="Select Layout"
            size="small"
          >
            <el-option
              v-for="layout in layouts"
              :key="layout.id"
              :label="layout.name"
              :value="layout.id"
            >
              <span class="layout-option">
                <span>{{ layout.name }}</span>
                <el-tag v-if="layout.isDefault" size="mini" type="primary">Default</el-tag>
                <el-tag v-if="layout.isShared" size="mini" type="success">Shared</el-tag>
              </span>
            </el-option>
          </el-select>
        </div>
      </div>

      <div class="header-actions">
        <!-- Global Settings -->
        <el-dropdown @command="handleGlobalAction">
          <el-button size="small" type="text">
            <el-icon><Setting /></el-icon>
            Settings
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="auto-refresh">
                <el-icon><Refresh /></el-icon>
                Auto Refresh: {{ globalSettings.autoRefresh ? 'On' : 'Off' }}
              </el-dropdown-item>
              <el-dropdown-item command="theme">
                <el-icon><Sunny /></el-icon>
                Theme: {{ globalSettings.theme }}
              </el-dropdown-item>
              <el-dropdown-item command="animations">
                <el-icon><Magic /></el-icon>
                Animations: {{ globalSettings.animations ? 'On' : 'Off' }}
              </el-dropdown-item>
              <el-dropdown-item command="compact-mode">
                <el-icon><Compress /></el-icon>
                Compact Mode: {{ globalSettings.compactMode ? 'On' : 'Off' }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <!-- Layout Management -->
        <el-dropdown @command="handleLayoutAction">
          <el-button size="small" type="text">
            <el-icon><Grid /></el-icon>
            Layout
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="edit-mode">
                <el-icon><Edit /></el-icon>
                {{ isEditMode ? 'Exit Edit Mode' : 'Edit Mode' }}
              </el-dropdown-item>
              <el-dropdown-item command="add-widget">
                <el-icon><Plus /></el-icon>
                Add Widget
              </el-dropdown-item>
              <el-dropdown-item command="create-layout">
                <el-icon><DocumentAdd /></el-icon>
                Create Layout
              </el-dropdown-item>
              <el-dropdown-item command="manage-layouts">
                <el-icon><FolderOpened /></el-icon>
                Manage Layouts
              </el-dropdown-item>
              <el-dropdown-item divided command="reset-layout">
                <el-icon><RefreshLeft /></el-icon>
                Reset Layout
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <!-- Refresh All -->
        <el-button
          size="small"
          @click="refreshAllWidgets"
          :loading="isRefreshing"
          type="primary"
        >
          <el-icon><Refresh /></el-icon>
          Refresh All
        </el-button>
      </div>
    </div>

    <!-- Dashboard Content -->
    <div class="dashboard-content" :class="{ 'edit-mode': isEditMode, 'compact-mode': globalSettings.compactMode }">
      <!-- Loading State -->
      <div v-if="isLoading" class="dashboard-loading">
        <el-skeleton animated>
          <template #template>
            <div class="skeleton-grid">
              <el-skeleton-item
                v-for="i in 6"
                :key="i"
                variant="rect"
                class="skeleton-widget"
              />
            </div>
          </template>
        </el-skeleton>
      </div>

      <!-- Widget Grid -->
      <grid-layout
        v-else
        v-model:layout="currentLayout.widgets"
        :col-num="gridConfig.cols"
        :row-height="gridConfig.rowHeight"
        :is-draggable="isEditMode"
        :is-resizable="isEditMode"
        :is-mirrored="false"
        :vertical-compact="true"
        :margin="gridConfig.margin"
        :use-css-transforms="true"
        @layout-updated="onLayoutUpdated"
        class="widget-grid"
      >
        <grid-item
          v-for="widget in currentLayout.widgets"
          :key="widget.id"
          :x="widget.position.x"
          :y="widget.position.y"
          :w="widget.position.w"
          :h="widget.position.h"
          :i="widget.id"
          :min-w="widget.size.minW"
          :min-h="widget.size.minH"
          :max-w="widget.size.maxW"
          :max-h="widget.size.maxH"
          :is-draggable="isEditMode && widget.draggable"
          :is-resizable="isEditMode && widget.resizable"
          class="widget-container"
          :data-widget-id="widget.id"
        >
          <!-- Widget Component -->
          <widget-wrapper
            :widget="widget"
            :is-edit-mode="isEditMode"
            @remove="removeWidget"
            @configure="configureWidget"
            @refresh="refreshWidget"
          />
        </grid-item>
      </grid-layout>

      <!-- Empty State -->
      <div v-if="!isLoading && currentLayout.widgets.length === 0" class="empty-state">
        <el-empty description="No widgets configured">
          <el-button type="primary" @click="showAddWidgetDialog">
            <el-icon><Plus /></el-icon>
            Add Your First Widget
          </el-button>
        </el-empty>
      </div>
    </div>

    <!-- Add Widget Dialog -->
    <el-dialog
      v-model="addWidgetDialogVisible"
      title="Add Widget"
      width="800px"
      :modal="true"
      class="add-widget-dialog"
    >
      <div class="widget-gallery">
        <div v-for="(widgets, category) in widgetsByCategory" :key="category" class="widget-category">
          <h3 class="category-title">{{ formatCategoryName(category) }}</h3>
          <div class="widget-grid">
            <div
              v-for="widget in widgets"
              :key="widget.type"
              class="widget-card"
              @click="selectWidget(widget)"
              :class="{ active: selectedWidgetType === widget.type }"
            >
              <div class="widget-icon">
                <el-icon :size="24"><component :is="widget.icon" /></el-icon>
              </div>
              <div class="widget-info">
                <h4>{{ widget.name }}</h4>
                <p>{{ widget.description }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="addWidgetDialogVisible = false">Cancel</el-button>
        <el-button
          type="primary"
          @click="addSelectedWidget"
          :disabled="!selectedWidgetType"
        >
          Add Widget
        </el-button>
      </template>
    </el-dialog>

    <!-- Layout Management Dialog -->
    <el-dialog
      v-model="layoutDialogVisible"
      title="Manage Layouts"
      width="600px"
      :modal="true"
    >
      <div class="layout-management">
        <div class="layout-list">
          <div
            v-for="layout in layouts"
            :key="layout.id"
            class="layout-item"
            :class="{ active: layout.id === currentLayoutId }"
          >
            <div class="layout-info">
              <h4>{{ layout.name }}</h4>
              <p>{{ layout.description || 'No description' }}</p>
              <div class="layout-meta">
                <el-tag v-if="layout.isDefault" size="mini" type="primary">Default</el-tag>
                <el-tag v-if="layout.isShared" size="mini" type="success">Shared</el-tag>
                <span class="layout-date">{{ formatDate(layout.updatedAt) }}</span>
              </div>
            </div>
            <div class="layout-actions">
              <el-button
                v-if="layout.id !== currentLayoutId"
                size="mini"
                @click="switchLayout(layout.id)"
              >
                Switch
              </el-button>
              <el-dropdown @command="(cmd) => handleLayoutItemAction(cmd, layout)">
                <el-button size="mini" type="text">
                  <el-icon><More /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="duplicate">Duplicate</el-dropdown-item>
                    <el-dropdown-item command="rename">Rename</el-dropdown-item>
                    <el-dropdown-item command="export">Export</el-dropdown-item>
                    <el-dropdown-item
                      v-if="!layout.isDefault && layouts.length > 1"
                      command="delete"
                      divided
                    >
                      Delete
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="layoutDialogVisible = false">Close</el-button>
        <el-button type="primary" @click="showCreateLayoutDialog">
          Create New Layout
        </el-button>
      </template>
    </el-dialog>

    <!-- Widget Configuration Dialog -->
    <widget-config-dialog
      v-model="configDialogVisible"
      :widget="configWidget"
      @save="saveWidgetConfig"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { GridLayout, GridItem } from 'vue-grid-layout'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Monitor, Setting, Refresh, Grid, Edit, Plus, DocumentAdd,
  FolderOpened, RefreshLeft, Sunny, Magic, Compress, More
} from '@element-plus/icons-vue'

// Store imports
import { useDashboardStore } from '@/store/dashboard'
import type { DashboardWidget, DashboardLayout } from '@/store/dashboard'
import { useWidgetManager } from '@/services/widgetManager'

// Component imports
import WidgetWrapper from '@/components/dashboard/WidgetWrapper.vue'
import WidgetConfigDialog from '@/components/dashboard/WidgetConfigDialog.vue'

// Initialize stores and services
const dashboardStore = useDashboardStore()
const widgetManager = useWidgetManager()

// Reactive state
const isRefreshing = ref(false)
const addWidgetDialogVisible = ref(false)
const layoutDialogVisible = ref(false)
const configDialogVisible = ref(false)
const selectedWidgetType = ref<string>('')
const configWidget = ref<DashboardWidget | null>(null)

// Computed properties
const isLoading = computed(() => dashboardStore.state.isLoading)
const isEditMode = computed(() => dashboardStore.state.isEditMode)
const layouts = computed(() => dashboardStore.state.layouts)
const currentLayout = computed(() => dashboardStore.currentLayout!)
const currentLayoutId = computed({
  get: () => dashboardStore.state.currentLayoutId,
  set: (value: string) => switchLayout(value)
})
const globalSettings = computed(() => dashboardStore.state.globalSettings)
const widgetsByCategory = computed(() => dashboardStore.widgetsByCategory)

// Grid configuration
const gridConfig = computed(() => ({
  cols: 12,
  rowHeight: 80,
  margin: globalSettings.value.compactMode ? [5, 5] : [10, 10]
}))

// Methods
const switchLayout = async (layoutId: string) => {
  try {
    await dashboardStore.switchLayout(layoutId)
    ElMessage.success('Layout switched successfully')
  } catch (error) {
    console.error('Failed to switch layout:', error)
    ElMessage.error('Failed to switch layout')
  }
}

const refreshAllWidgets = async () => {
  try {
    isRefreshing.value = true
    const widgetIds = currentLayout.value.widgets.map(w => w.id)
    await widgetManager.refreshMultipleWidgets(widgetIds, true)
    ElMessage.success('All widgets refreshed')
  } catch (error) {
    console.error('Failed to refresh widgets:', error)
    ElMessage.error('Failed to refresh some widgets')
  } finally {
    isRefreshing.value = false
  }
}

const refreshWidget = async (widgetId: string) => {
  try {
    await widgetManager.refreshWidget(widgetId, true)
    ElMessage.success('Widget refreshed')
  } catch (error) {
    console.error('Failed to refresh widget:', error)
    ElMessage.error('Failed to refresh widget')
  }
}

const showAddWidgetDialog = () => {
  selectedWidgetType.value = ''
  addWidgetDialogVisible.value = true
}

const selectWidget = (widget: any) => {
  selectedWidgetType.value = widget.type
}

const addSelectedWidget = async () => {
  if (!selectedWidgetType.value) return

  try {
    await dashboardStore.addWidget(selectedWidgetType.value)
    addWidgetDialogVisible.value = false
    ElMessage.success('Widget added successfully')
  } catch (error) {
    console.error('Failed to add widget:', error)
    ElMessage.error('Failed to add widget')
  }
}

const removeWidget = async (widgetId: string) => {
  try {
    await ElMessageBox.confirm(
      'Are you sure you want to remove this widget?',
      'Confirm Removal',
      {
        type: 'warning',
        confirmButtonText: 'Remove',
        cancelButtonText: 'Cancel'
      }
    )

    await dashboardStore.removeWidget(widgetId)
    ElMessage.success('Widget removed successfully')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to remove widget:', error)
      ElMessage.error('Failed to remove widget')
    }
  }
}

const configureWidget = (widget: DashboardWidget) => {
  configWidget.value = widget
  configDialogVisible.value = true
}

const saveWidgetConfig = async (widgetId: string, config: any) => {
  try {
    await dashboardStore.updateWidget(widgetId, config)
    configDialogVisible.value = false
    ElMessage.success('Widget configuration saved')
  } catch (error) {
    console.error('Failed to save widget config:', error)
    ElMessage.error('Failed to save configuration')
  }
}

const onLayoutUpdated = async (layout: any[]) => {
  if (!currentLayout.value) return

  try {
    // Update widget positions
    const updatedWidgets = currentLayout.value.widgets.map(widget => {
      const layoutItem = layout.find(item => item.i === widget.id)
      if (layoutItem) {
        return {
          ...widget,
          position: {
            x: layoutItem.x,
            y: layoutItem.y,
            w: layoutItem.w,
            h: layoutItem.h
          }
        }
      }
      return widget
    })

    await dashboardStore.updateLayout(currentLayout.value.id, {
      widgets: updatedWidgets
    })
  } catch (error) {
    console.error('Failed to update layout:', error)
  }
}

const handleGlobalAction = async (command: string) => {
  switch (command) {
    case 'auto-refresh':
      await dashboardStore.updateGlobalSettings({
        autoRefresh: !globalSettings.value.autoRefresh
      })
      break
    case 'theme':
      const themes = ['auto', 'light', 'dark']
      const currentIndex = themes.indexOf(globalSettings.value.theme)
      const nextTheme = themes[(currentIndex + 1) % themes.length]
      await dashboardStore.updateGlobalSettings({ theme: nextTheme })
      break
    case 'animations':
      await dashboardStore.updateGlobalSettings({
        animations: !globalSettings.value.animations
      })
      break
    case 'compact-mode':
      await dashboardStore.updateGlobalSettings({
        compactMode: !globalSettings.value.compactMode
      })
      break
  }
}

const handleLayoutAction = async (command: string) => {
  switch (command) {
    case 'edit-mode':
      dashboardStore.setEditMode(!isEditMode.value)
      break
    case 'add-widget':
      showAddWidgetDialog()
      break
    case 'create-layout':
      showCreateLayoutDialog()
      break
    case 'manage-layouts':
      layoutDialogVisible.value = true
      break
    case 'reset-layout':
      await resetLayout()
      break
  }
}

const handleLayoutItemAction = async (command: string, layout: DashboardLayout) => {
  switch (command) {
    case 'duplicate':
      await duplicateLayout(layout)
      break
    case 'rename':
      await renameLayout(layout)
      break
    case 'export':
      await exportLayout(layout)
      break
    case 'delete':
      await deleteLayout(layout)
      break
  }
}

const showCreateLayoutDialog = async () => {
  try {
    const { value: name } = await ElMessageBox.prompt(
      'Enter layout name',
      'Create New Layout',
      {
        confirmButtonText: 'Create',
        cancelButtonText: 'Cancel',
        inputPattern: /^.+$/,
        inputErrorMessage: 'Layout name is required'
      }
    )

    await dashboardStore.createLayout(name)
    ElMessage.success('Layout created successfully')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to create layout:', error)
      ElMessage.error('Failed to create layout')
    }
  }
}

const duplicateLayout = async (layout: DashboardLayout) => {
  try {
    const { value: name } = await ElMessageBox.prompt(
      'Enter name for the duplicated layout',
      'Duplicate Layout',
      {
        confirmButtonText: 'Duplicate',
        cancelButtonText: 'Cancel',
        inputValue: `${layout.name} (Copy)`,
        inputPattern: /^.+$/,
        inputErrorMessage: 'Layout name is required'
      }
    )

    await dashboardStore.createLayout(name, `Copy of ${layout.name}`, layout.id)
    ElMessage.success('Layout duplicated successfully')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to duplicate layout:', error)
      ElMessage.error('Failed to duplicate layout')
    }
  }
}

const renameLayout = async (layout: DashboardLayout) => {
  try {
    const { value: name } = await ElMessageBox.prompt(
      'Enter new layout name',
      'Rename Layout',
      {
        confirmButtonText: 'Rename',
        cancelButtonText: 'Cancel',
        inputValue: layout.name,
        inputPattern: /^.+$/,
        inputErrorMessage: 'Layout name is required'
      }
    )

    await dashboardStore.updateLayout(layout.id, { name })
    ElMessage.success('Layout renamed successfully')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to rename layout:', error)
      ElMessage.error('Failed to rename layout')
    }
  }
}

const exportLayout = async (layout: DashboardLayout) => {
  try {
    const data = JSON.stringify(layout, null, 2)
    const blob = new Blob([data], { type: 'application/json' })
    const url = URL.createObjectURL(blob)

    const a = document.createElement('a')
    a.href = url
    a.download = `dashboard-layout-${layout.name.replace(/\s+/g, '-').toLowerCase()}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    ElMessage.success('Layout exported successfully')
  } catch (error) {
    console.error('Failed to export layout:', error)
    ElMessage.error('Failed to export layout')
  }
}

const deleteLayout = async (layout: DashboardLayout) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to delete the layout "${layout.name}"?`,
      'Confirm Deletion',
      {
        type: 'warning',
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel'
      }
    )

    await dashboardStore.deleteLayout(layout.id)
    ElMessage.success('Layout deleted successfully')
    layoutDialogVisible.value = false
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete layout:', error)
      ElMessage.error('Failed to delete layout')
    }
  }
}

const resetLayout = async () => {
  try {
    await ElMessageBox.confirm(
      'Are you sure you want to reset the current layout to its default state?',
      'Confirm Reset',
      {
        type: 'warning',
        confirmButtonText: 'Reset',
        cancelButtonText: 'Cancel'
      }
    )

    // Implementation for resetting layout
    ElMessage.success('Layout reset successfully')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to reset layout:', error)
      ElMessage.error('Failed to reset layout')
    }
  }
}

const formatCategoryName = (category: string): string => {
  return category.charAt(0).toUpperCase() + category.slice(1).replace(/[-_]/g, ' ')
}

const formatDate = (date: Date): string => {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

// Lifecycle hooks
onMounted(async () => {
  await dashboardStore.initialize()
})

onUnmounted(() => {
  // Cleanup if needed
})

// Watch for edit mode changes
watch(isEditMode, (editMode) => {
  if (editMode) {
    ElMessage.info('Edit mode enabled. Drag and resize widgets as needed.')
  }
})
</script>

<style scoped lang="scss">
.dashboard-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

.dashboard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.04);

  .header-left {
    display: flex;
    align-items: center;
    gap: 24px;

    .dashboard-title {
      margin: 0;
      font-size: 20px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .layout-selector {
      min-width: 200px;

      .layout-option {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
      }
    }
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

.dashboard-content {
  flex: 1;
  padding: 24px;
  overflow: auto;

  &.compact-mode {
    padding: 12px;
  }

  &.edit-mode {
    background: repeating-linear-gradient(
      45deg,
      transparent,
      transparent 10px,
      rgba(var(--el-color-primary-rgb), 0.03) 10px,
      rgba(var(--el-color-primary-rgb), 0.03) 20px
    );
  }
}

.dashboard-loading {
  .skeleton-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 20px;

    .skeleton-widget {
      height: 200px;
      border-radius: 8px;
    }
  }
}

.widget-grid {
  min-height: 600px;
}

.widget-container {
  background: var(--el-bg-color);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid var(--el-border-color-lighter);
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  }

  .dashboard-content.edit-mode & {
    border: 2px dashed var(--el-color-primary);

    &:hover {
      border-color: var(--el-color-primary-light-3);
      background: rgba(var(--el-color-primary-rgb), 0.02);
    }
  }
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 400px;
}

.add-widget-dialog {
  .widget-gallery {
    .widget-category {
      margin-bottom: 32px;

      .category-title {
        margin: 0 0 16px 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        border-bottom: 1px solid var(--el-border-color-light);
        padding-bottom: 8px;
      }

      .widget-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
        gap: 16px;

        .widget-card {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 16px;
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 8px;
          cursor: pointer;
          transition: all 0.3s ease;

          &:hover {
            border-color: var(--el-color-primary);
            box-shadow: 0 2px 8px rgba(var(--el-color-primary-rgb), 0.2);
          }

          &.active {
            border-color: var(--el-color-primary);
            background: rgba(var(--el-color-primary-rgb), 0.05);
          }

          .widget-icon {
            flex-shrink: 0;
            color: var(--el-color-primary);
          }

          .widget-info {
            h4 {
              margin: 0 0 4px 0;
              font-size: 14px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }

            p {
              margin: 0;
              font-size: 12px;
              color: var(--el-text-color-secondary);
              line-height: 1.4;
            }
          }
        }
      }
    }
  }
}

.layout-management {
  .layout-list {
    .layout-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 16px;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
      margin-bottom: 12px;
      transition: all 0.3s ease;

      &:hover {
        border-color: var(--el-color-primary);
      }

      &.active {
        border-color: var(--el-color-primary);
        background: rgba(var(--el-color-primary-rgb), 0.05);
      }

      .layout-info {
        flex: 1;

        h4 {
          margin: 0 0 4px 0;
          font-size: 16px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        p {
          margin: 0 0 8px 0;
          font-size: 14px;
          color: var(--el-text-color-secondary);
        }

        .layout-meta {
          display: flex;
          align-items: center;
          gap: 8px;

          .layout-date {
            font-size: 12px;
            color: var(--el-text-color-placeholder);
          }
        }
      }

      .layout-actions {
        display: flex;
        align-items: center;
        gap: 8px;
      }
    }
  }
}

// Responsive design
@media (max-width: 768px) {
  .dashboard-header {
    flex-direction: column;
    gap: 16px;

    .header-left {
      width: 100%;
      justify-content: space-between;
    }

    .header-actions {
      width: 100%;
      justify-content: center;
    }
  }

  .dashboard-content {
    padding: 12px;
  }

  .widget-grid {
    min-height: 400px;
  }
}

@media (max-width: 480px) {
  .dashboard-header {
    padding: 12px 16px;

    .header-left {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;
    }
  }

  .add-widget-dialog {
    .widget-gallery {
      .widget-category {
        .widget-grid {
          grid-template-columns: 1fr;
        }
      }
    }
  }
}
</style>