<template>
  <div class="container-detail-page">
    <!-- Loading State -->
    <div v-if="loadingDetails" class="loading-container">
      <el-skeleton :rows="6" animated />
    </div>

    <!-- Container Not Found -->
    <div v-else-if="!currentContainer" class="not-found">
      <div class="not-found-content">
        <el-icon class="not-found-icon"><Warning /></el-icon>
        <h2>Container Not Found</h2>
        <p>The container you're looking for doesn't exist or has been removed.</p>
        <el-button type="primary" @click="$router.push('/containers')">
          Back to Containers
        </el-button>
      </div>
    </div>

    <!-- Container Details -->
    <div v-else class="container-details">
      <!-- Header -->
      <div class="detail-header">
        <div class="header-content">
          <div class="breadcrumb">
            <el-breadcrumb>
              <el-breadcrumb-item :to="{ path: '/containers' }">
                Containers
              </el-breadcrumb-item>
              <el-breadcrumb-item>
                {{ currentContainer.name }}
              </el-breadcrumb-item>
            </el-breadcrumb>
          </div>

          <div class="container-header">
            <div class="container-title">
              <h1>{{ currentContainer.name }}</h1>
              <el-tag
                :type="getStatusType(currentContainer.status)"
                size="large"
                effect="dark"
                class="status-tag"
              >
                <el-icon>
                  <component :is="getStatusIcon(currentContainer.status)" />
                </el-icon>
                {{ currentContainer.status }}
              </el-tag>
            </div>

            <div class="header-actions">
              <!-- Quick Actions -->
              <el-button-group>
                <el-button
                  v-if="currentContainer.status === 'exited'"
                  type="success"
                  :loading="isOperationLoading(currentContainer.id)"
                  @click="performOperation('start')"
                  :disabled="!canPerformAction('start')"
                >
                  <el-icon><VideoPlay /></el-icon>
                  Start
                </el-button>

                <el-button
                  v-else-if="currentContainer.status === 'running'"
                  type="warning"
                  :loading="isOperationLoading(currentContainer.id)"
                  @click="performOperation('stop')"
                  :disabled="!canPerformAction('stop')"
                >
                  <el-icon><VideoPause /></el-icon>
                  Stop
                </el-button>

                <el-button
                  :loading="isOperationLoading(currentContainer.id)"
                  @click="performOperation('restart')"
                  :disabled="!canPerformAction('restart')"
                >
                  <el-icon><Refresh /></el-icon>
                  Restart
                </el-button>
              </el-button-group>

              <!-- More Actions -->
              <el-dropdown @command="handleAction">
                <el-button>
                  <el-icon><MoreFilled /></el-icon>
                  More Actions
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="update" :disabled="!canPerformAction('update')">
                      <el-icon><Download /></el-icon>
                      Update Container
                    </el-dropdown-item>
                    <el-dropdown-item command="edit" :disabled="!canPerformAction('edit')">
                      <el-icon><Edit /></el-icon>
                      Edit Configuration
                    </el-dropdown-item>
                    <el-dropdown-item command="clone" :disabled="!canPerformAction('clone')">
                      <el-icon><CopyDocument /></el-icon>
                      Clone Container
                    </el-dropdown-item>
                    <el-dropdown-item command="backup" :disabled="!canPerformAction('backup')">
                      <el-icon><Upload /></el-icon>
                      Create Backup
                    </el-dropdown-item>
                    <el-dropdown-item command="export" :disabled="!canPerformAction('export')">
                      <el-icon><Download /></el-icon>
                      Export Configuration
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" :disabled="!canPerformAction('delete')" divided>
                      <el-icon><Delete /></el-icon>
                      Delete Container
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>

              <!-- Refresh Button -->
              <el-button @click="refreshContainer">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- Content Tabs -->
      <div class="detail-content">
        <el-tabs v-model="activeTab" type="border-card" @tab-change="handleTabChange">
          <!-- Overview Tab -->
          <el-tab-pane label="Overview" name="overview">
            <div class="overview-content">
              <!-- Basic Information -->
              <div class="info-section">
                <h3 class="section-title">
                  <el-icon><InfoFilled /></el-icon>
                  Basic Information
                </h3>
                <div class="info-grid">
                  <div class="info-item">
                    <span class="info-label">Container ID:</span>
                    <span class="info-value">{{ currentContainer.id }}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Image:</span>
                    <span class="info-value">{{ currentContainer.image }}:{{ currentContainer.tag }}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Status:</span>
                    <el-tag :type="getStatusType(currentContainer.status)">
                      {{ currentContainer.status }}
                    </el-tag>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Health:</span>
                    <el-tag :type="getHealthType(currentContainer.health.status)">
                      {{ formatHealthStatus(currentContainer.health.status) }}
                    </el-tag>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Created:</span>
                    <span class="info-value">{{ formatFullDate(currentContainer.createdAt) }}</span>
                  </div>
                  <div class="info-item" v-if="currentContainer.startedAt">
                    <span class="info-label">Started:</span>
                    <span class="info-value">{{ formatFullDate(currentContainer.startedAt) }}</span>
                  </div>
                  <div class="info-item" v-if="currentContainer.workingDir">
                    <span class="info-label">Working Directory:</span>
                    <span class="info-value">{{ currentContainer.workingDir }}</span>
                  </div>
                  <div class="info-item" v-if="currentContainer.user">
                    <span class="info-label">User:</span>
                    <span class="info-value">{{ currentContainer.user }}</span>
                  </div>
                </div>
              </div>

              <!-- Resource Usage -->
              <div class="info-section">
                <h3 class="section-title">
                  <el-icon><Monitor /></el-icon>
                  Resource Usage
                </h3>
                <ResourceMonitor
                  :container-id="currentContainer.id"
                  :container-name="currentContainer.name"
                  :show-historical="true"
                />
              </div>

              <!-- Health Status -->
              <div class="info-section" v-if="currentContainer.health.status !== 'none'">
                <h3 class="section-title">
                  <el-icon><CircleCheckFilled /></el-icon>
                  Health Status
                </h3>
                <div class="health-details">
                  <div class="health-summary">
                    <div class="health-item">
                      <span class="health-label">Status:</span>
                      <el-tag :type="getHealthType(currentContainer.health.status)">
                        {{ formatHealthStatus(currentContainer.health.status) }}
                      </el-tag>
                    </div>
                    <div class="health-item" v-if="currentContainer.health.failingStreak > 0">
                      <span class="health-label">Failing Streak:</span>
                      <span class="health-value">{{ currentContainer.health.failingStreak }}</span>
                    </div>
                  </div>

                  <!-- Health Check History -->
                  <div v-if="currentContainer.health.log.length > 0" class="health-history">
                    <h4>Recent Health Checks</h4>
                    <div class="health-log">
                      <div
                        v-for="(entry, index) in currentContainer.health.log.slice(0, 5)"
                        :key="index"
                        class="health-entry"
                        :class="{
                          'health-success': entry.exitCode === 0,
                          'health-failure': entry.exitCode !== 0
                        }"
                      >
                        <div class="health-time">{{ formatFullDate(entry.start) }}</div>
                        <div class="health-status">
                          <el-icon>
                            <component :is="entry.exitCode === 0 ? 'SuccessFilled' : 'CircleCloseFilled'" />
                          </el-icon>
                          {{ entry.exitCode === 0 ? 'Passed' : 'Failed' }}
                        </div>
                        <div class="health-output" v-if="entry.output">
                          {{ entry.output }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Update Status -->
              <div class="info-section" v-if="hasAvailableUpdate">
                <h3 class="section-title">
                  <el-icon><Download /></el-icon>
                  Update Available
                </h3>
                <UpdateManager
                  :container-id="currentContainer.id"
                  :current-version="currentContainer.tag"
                />
              </div>
            </div>
          </el-tab-pane>

          <!-- Configuration Tab -->
          <el-tab-pane label="Configuration" name="configuration">
            <div class="configuration-content">
              <!-- Environment Variables -->
              <div class="config-section" v-if="Object.keys(currentContainer.environment).length > 0">
                <h3 class="section-title">
                  <el-icon><Setting /></el-icon>
                  Environment Variables
                </h3>
                <div class="env-variables">
                  <div
                    v-for="(value, key) in currentContainer.environment"
                    :key="key"
                    class="env-item"
                  >
                    <span class="env-key">{{ key }}</span>
                    <span class="env-value">{{ formatEnvValue(value) }}</span>
                  </div>
                </div>
              </div>

              <!-- Port Mappings -->
              <div class="config-section" v-if="currentContainer.ports.length > 0">
                <h3 class="section-title">
                  <el-icon><Connection /></el-icon>
                  Port Mappings
                </h3>
                <el-table :data="currentContainer.ports" stripe>
                  <el-table-column prop="hostPort" label="Host Port" width="120" />
                  <el-table-column prop="containerPort" label="Container Port" width="140" />
                  <el-table-column prop="protocol" label="Protocol" width="100">
                    <template #default="{ row }">
                      <el-tag size="small">{{ row.protocol.toUpperCase() }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="hostIp" label="Host IP" />
                </el-table>
              </div>

              <!-- Volume Mounts -->
              <div class="config-section" v-if="currentContainer.volumes.length > 0">
                <h3 class="section-title">
                  <el-icon><FolderOpened /></el-icon>
                  Volume Mounts
                </h3>
                <el-table :data="currentContainer.volumes" stripe>
                  <el-table-column prop="source" label="Source" min-width="200" />
                  <el-table-column prop="target" label="Target" min-width="200" />
                  <el-table-column prop="type" label="Type" width="100">
                    <template #default="{ row }">
                      <el-tag size="small" :type="getVolumeTypeColor(row.type)">
                        {{ row.type }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="readOnly" label="Read Only" width="100">
                    <template #default="{ row }">
                      <el-icon v-if="row.readOnly" color="#67c23a"><Check /></el-icon>
                      <el-icon v-else color="#f56c6c"><Close /></el-icon>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <!-- Networks -->
              <div class="config-section" v-if="currentContainer.networks.length > 0">
                <h3 class="section-title">
                  <el-icon><Share /></el-icon>
                  Networks
                </h3>
                <el-table :data="currentContainer.networks" stripe>
                  <el-table-column prop="name" label="Network Name" />
                  <el-table-column prop="ipAddress" label="IP Address" />
                  <el-table-column prop="gateway" label="Gateway" />
                  <el-table-column prop="macAddress" label="MAC Address" />
                </el-table>
              </div>

              <!-- Labels -->
              <div class="config-section" v-if="Object.keys(currentContainer.labels).length > 0">
                <h3 class="section-title">
                  <el-icon><CollectionTag /></el-icon>
                  Labels
                </h3>
                <div class="labels-list">
                  <div
                    v-for="(value, key) in currentContainer.labels"
                    :key="key"
                    class="label-item"
                  >
                    <span class="label-key">{{ key }}</span>
                    <span class="label-value">{{ value }}</span>
                  </div>
                </div>
              </div>

              <!-- Update Policy -->
              <div class="config-section">
                <h3 class="section-title">
                  <el-icon><Refresh /></el-icon>
                  Update Policy
                </h3>
                <div class="update-policy">
                  <div class="policy-item">
                    <span class="policy-label">Auto Update:</span>
                    <el-tag :type="currentContainer.updatePolicy.autoUpdate ? 'success' : 'info'">
                      {{ currentContainer.updatePolicy.autoUpdate ? 'Enabled' : 'Disabled' }}
                    </el-tag>
                  </div>
                  <div class="policy-item">
                    <span class="policy-label">Strategy:</span>
                    <el-tag>{{ currentContainer.updatePolicy.strategy }}</el-tag>
                  </div>
                  <div class="policy-item" v-if="currentContainer.updatePolicy.schedule">
                    <span class="policy-label">Schedule:</span>
                    <span class="policy-value">{{ currentContainer.updatePolicy.schedule }}</span>
                  </div>
                  <div class="policy-item">
                    <span class="policy-label">Rollback on Failure:</span>
                    <el-tag :type="currentContainer.updatePolicy.rollbackOnFailure ? 'success' : 'info'">
                      {{ currentContainer.updatePolicy.rollbackOnFailure ? 'Enabled' : 'Disabled' }}
                    </el-tag>
                  </div>
                </div>
              </div>
            </div>
          </el-tab-pane>

          <!-- Logs Tab -->
          <el-tab-pane label="Logs" name="logs">
            <LogViewer
              :container-id="currentContainer.id"
              :container-name="currentContainer.name"
              :full-height="true"
            />
          </el-tab-pane>

          <!-- Monitoring Tab -->
          <el-tab-pane label="Monitoring" name="monitoring">
            <ResourceMonitor
              :container-id="currentContainer.id"
              :container-name="currentContainer.name"
              :show-historical="true"
              :detailed-view="true"
            />
          </el-tab-pane>

          <!-- Terminal Tab -->
          <el-tab-pane
            label="Terminal"
            name="terminal"
            :disabled="currentContainer.status !== 'running'"
          >
            <div v-if="currentContainer.status !== 'running'" class="terminal-disabled">
              <el-alert
                title="Terminal Unavailable"
                description="Terminal access is only available for running containers."
                type="warning"
                :closable="false"
              />
            </div>
            <div v-else class="terminal-container">
              <!-- Terminal component would go here -->
              <div class="terminal-placeholder">
                <p>Terminal functionality will be implemented here.</p>
                <p>This would provide web-based terminal access to the container.</p>
              </div>
            </div>
          </el-tab-pane>

          <!-- Events Tab -->
          <el-tab-pane label="Events" name="events">
            <div class="events-content">
              <div class="events-header">
                <h3>Container Events</h3>
                <el-button @click="refreshEvents" size="small">
                  <el-icon><Refresh /></el-icon>
                  Refresh
                </el-button>
              </div>

              <div class="events-list">
                <!-- Events timeline would go here -->
                <div class="events-placeholder">
                  <p>Container events and activity timeline will be displayed here.</p>
                  <p>This includes start/stop events, configuration changes, and errors.</p>
                </div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- Edit Dialog -->
    <el-dialog
      v-model="showEditDialog"
      title="Edit Container Configuration"
      width="80%"
      :before-close="handleEditDialogClose"
    >
      <ContainerForm
        v-if="showEditDialog"
        :container="currentContainer"
        @submit="handleEditContainer"
        @cancel="showEditDialog = false"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Warning,
  VideoPlay,
  VideoPause,
  Refresh,
  MoreFilled,
  Download,
  Edit,
  CopyDocument,
  Upload,
  Delete,
  InfoFilled,
  Monitor,
  CircleCheckFilled,
  Setting,
  Connection,
  FolderOpened,
  Share,
  CollectionTag,
  Check,
  Close,
  SuccessFilled,
  CircleCloseFilled
} from '@element-plus/icons-vue'

import { useContainerStore } from '@/store/containers'
import { useAuthStore } from '@/store/auth'
import ResourceMonitor from '@/components/container/ResourceMonitor.vue'
import LogViewer from '@/components/container/LogViewer.vue'
import UpdateManager from '@/components/container/UpdateManager.vue'
import ContainerForm from '@/components/container/ContainerForm.vue'

import type { ContainerFormData } from '@/types/container'

const route = useRoute()
const router = useRouter()
const containerStore = useContainerStore()
const authStore = useAuthStore()

// Store refs
const {
  currentContainer,
  loadingDetails,
  availableUpdates
} = storeToRefs(containerStore)

// Local state
const activeTab = ref('overview')
const showEditDialog = ref(false)

// Computed
const containerId = computed(() => route.params.id as string)

const hasAvailableUpdate = computed(() => {
  return availableUpdates.value.some(update => update.container === containerId.value)
})

// Methods
function getStatusType(status: string): string {
  const types: Record<string, string> = {
    running: 'success',
    exited: 'info',
    paused: 'warning',
    restarting: 'warning',
    removing: 'danger',
    dead: 'danger'
  }
  return types[status] || 'info'
}

function getStatusIcon(status: string) {
  const icons: Record<string, any> = {
    running: SuccessFilled,
    exited: CircleCloseFilled,
    paused: Warning,
    restarting: Refresh,
    removing: Delete,
    dead: CircleCloseFilled
  }
  return icons[status] || Warning
}

function getHealthType(health: string): string {
  const types: Record<string, string> = {
    healthy: 'success',
    unhealthy: 'danger',
    starting: 'warning',
    none: 'info'
  }
  return types[health] || 'info'
}

function formatHealthStatus(status: string): string {
  const statuses: Record<string, string> = {
    healthy: 'Healthy',
    unhealthy: 'Unhealthy',
    starting: 'Starting',
    none: 'No Health Check'
  }
  return statuses[status] || status
}

function formatFullDate(date: Date | string): string {
  return new Date(date).toLocaleString()
}

function formatEnvValue(value: string): string {
  // Hide sensitive values
  if (value.toLowerCase().includes('password') ||
      value.toLowerCase().includes('secret') ||
      value.toLowerCase().includes('key')) {
    return '••••••••'
  }
  return value
}

function getVolumeTypeColor(type: string): string {
  const colors: Record<string, string> = {
    bind: 'primary',
    volume: 'success',
    tmpfs: 'warning'
  }
  return colors[type] || ''
}

function canPerformAction(action: string): boolean {
  const permissions: Record<string, string> = {
    start: 'container:start',
    stop: 'container:stop',
    restart: 'container:restart',
    update: 'container:update',
    edit: 'container:update',
    clone: 'container:create',
    backup: 'container:backup',
    export: 'container:export',
    delete: 'container:delete'
  }

  const permission = permissions[action]
  return permission ? authStore.hasPermission(permission) : false
}

function isOperationLoading(id: string): boolean {
  return containerStore.isOperationLoading(id)
}

async function performOperation(operation: 'start' | 'stop' | 'restart') {
  if (!currentContainer.value) return

  try {
    await containerStore.performOperation(currentContainer.value.id, operation)
  } catch (error) {
    console.error('Operation failed:', error)
  }
}

function handleAction(command: string) {
  if (!currentContainer.value) return

  switch (command) {
    case 'update':
      containerStore.updateContainerImage(currentContainer.value.id)
      break
    case 'edit':
      showEditDialog.value = true
      break
    case 'clone':
      // Handle clone
      break
    case 'backup':
      // Handle backup
      break
    case 'export':
      // Handle export
      break
    case 'delete':
      ElMessageBox.confirm(
        `Are you sure you want to delete container "${currentContainer.value.name}"?`,
        'Confirm Deletion',
        {
          type: 'warning',
          confirmButtonText: 'Delete',
          cancelButtonText: 'Cancel'
        }
      ).then(() => {
        if (currentContainer.value) {
          containerStore.deleteContainer(currentContainer.value.id).then(() => {
            router.push('/containers')
          })
        }
      })
      break
  }
}

function handleTabChange(tabName: string) {
  // Load data specific to the tab
  if (tabName === 'monitoring' && currentContainer.value) {
    containerStore.fetchStats(currentContainer.value.id)
    containerStore.fetchHistoricalStats(currentContainer.value.id)
  }
}

function refreshContainer() {
  if (containerId.value) {
    containerStore.fetchContainer(containerId.value)
  }
}

function refreshEvents() {
  // Refresh events data
  ElMessage.success('Events refreshed')
}

function handleEditContainer(data: Partial<ContainerFormData>) {
  if (!currentContainer.value) return

  containerStore.updateContainer(currentContainer.value.id, data).then(() => {
    showEditDialog.value = false
  })
}

function handleEditDialogClose(done: () => void) {
  ElMessageBox.confirm('Discard changes and close?')
    .then(() => done())
    .catch(() => {})
}

// Watch for route changes
watch(
  () => route.params.id,
  (newId) => {
    if (newId) {
      containerStore.fetchContainer(newId as string)
    }
  },
  { immediate: true }
)

// Lifecycle
onMounted(() => {
  if (containerId.value) {
    containerStore.fetchContainer(containerId.value)
    containerStore.checkUpdates(containerId.value)
  }
})

onUnmounted(() => {
  // Cleanup if needed
})
</script>

<style scoped>
.container-detail-page {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.loading-container {
  padding: 24px;
}

.not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}

.not-found-content {
  text-align: center;
  max-width: 400px;
}

.not-found-icon {
  font-size: 64px;
  color: #f56c6c;
  margin-bottom: 16px;
}

.not-found-content h2 {
  margin: 0 0 8px 0;
  color: #303133;
}

.not-found-content p {
  margin: 0 0 24px 0;
  color: #606266;
}

.container-details {
  min-height: 100vh;
}

.detail-header {
  background: white;
  border-bottom: 1px solid #e4e7ed;
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-content {
  padding: 24px;
}

.breadcrumb {
  margin-bottom: 16px;
}

.container-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.container-title {
  display: flex;
  align-items: center;
  gap: 16px;
}

.container-title h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
  color: #303133;
}

.status-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.detail-content {
  background: white;
  min-height: calc(100vh - 120px);
}

.overview-content {
  padding: 24px;
}

.info-section {
  margin-bottom: 32px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.info-label {
  font-weight: 500;
  color: #606266;
  min-width: 120px;
}

.info-value {
  color: #303133;
  word-break: break-all;
}

.health-details {
  margin-top: 16px;
}

.health-summary {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
}

.health-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.health-label {
  font-weight: 500;
  color: #606266;
}

.health-value {
  color: #303133;
}

.health-history h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #606266;
}

.health-log {
  background: #f8f9fa;
  border-radius: 6px;
  padding: 12px;
  max-height: 200px;
  overflow-y: auto;
}

.health-entry {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid #e4e7ed;
}

.health-entry:last-child {
  border-bottom: none;
}

.health-time {
  font-size: 12px;
  color: #909399;
  min-width: 140px;
}

.health-status {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  min-width: 80px;
}

.health-success {
  color: #67c23a;
}

.health-failure {
  color: #f56c6c;
}

.health-output {
  font-size: 11px;
  color: #606266;
  flex: 1;
  word-break: break-all;
}

.configuration-content {
  padding: 24px;
}

.config-section {
  margin-bottom: 32px;
}

.env-variables {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 8px;
}

.env-item {
  display: flex;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 4px;
  border: 1px solid #e4e7ed;
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.env-key {
  font-weight: 600;
  color: #409eff;
  margin-right: 8px;
}

.env-value {
  color: #303133;
  word-break: break-all;
}

.labels-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 8px;
}

.label-item {
  display: flex;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 4px;
  border: 1px solid #e4e7ed;
  font-size: 12px;
}

.label-key {
  font-weight: 600;
  color: #606266;
  margin-right: 8px;
}

.label-value {
  color: #303133;
  word-break: break-all;
}

.update-policy {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.policy-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
}

.policy-label {
  font-weight: 500;
  color: #606266;
  min-width: 140px;
}

.policy-value {
  color: #303133;
}

.terminal-disabled {
  padding: 24px;
}

.terminal-container {
  height: 500px;
  background: #000;
  color: #fff;
  font-family: 'Courier New', monospace;
}

.terminal-placeholder {
  padding: 24px;
  text-align: center;
  color: #606266;
}

.events-content {
  padding: 24px;
}

.events-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.events-header h3 {
  margin: 0;
  color: #303133;
}

.events-placeholder {
  text-align: center;
  padding: 40px;
  color: #606266;
}

/* Responsive Design */
@media (max-width: 768px) {
  .header-content {
    padding: 16px;
  }

  .container-header {
    flex-direction: column;
    gap: 16px;
  }

  .container-title {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .header-actions {
    align-self: stretch;
    justify-content: center;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .health-summary {
    flex-direction: column;
    gap: 12px;
  }

  .env-variables,
  .labels-list {
    grid-template-columns: 1fr;
  }

  .update-policy {
    grid-template-columns: 1fr;
  }
}
</style>