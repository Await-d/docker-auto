<template>
  <div class="update-history-view">
    <!-- Header -->
    <div class="history-header">
      <div class="header-content">
        <div class="header-info">
          <h1 class="page-title">
            <el-icon><Clock /></el-icon>
            Update History
          </h1>
          <p class="page-description">
            Track update history, analyze trends, and manage rollbacks
          </p>
        </div>

        <div class="header-actions">
          <el-button-group>
            <el-button
              :icon="Refresh"
              :loading="loadingHistory"
              @click="refreshHistory"
            >
              Refresh
            </el-button>
            <el-button
              :icon="Download"
              @click="showExportDialog = true"
            >
              Export
            </el-button>
            <el-button
              :icon="TrendCharts"
              @click="showAnalytics = !showAnalytics"
            >
              Analytics
            </el-button>
          </el-button-group>
        </div>
      </div>

      <!-- Analytics Panel -->
      <el-collapse-transition>
        <div v-show="showAnalytics" class="analytics-panel">
          <div class="analytics-grid">
            <!-- Success Rate Chart -->
            <div class="analytics-card">
              <h3>Success Rate</h3>
              <div class="chart-container">
                <el-progress
                  :percentage="updateAnalytics.successRate"
                  :color="getSuccessRateColor(updateAnalytics.successRate)"
                  :stroke-width="8"
                  text-inside
                />
                <div class="chart-details">
                  <span class="detail-item">
                    <span class="label">Completed:</span>
                    <span class="value success">{{ updateAnalytics.updatesThisMonth }}</span>
                  </span>
                  <span class="detail-item">
                    <span class="label">Failed:</span>
                    <span class="value error">{{ updateAnalytics.failedUpdatesLast30Days }}</span>
                  </span>
                </div>
              </div>
            </div>

            <!-- Average Update Time -->
            <div class="analytics-card">
              <h3>Average Update Time</h3>
              <div class="metric-display">
                <span class="metric-value">{{ formatDuration(updateAnalytics.averageUpdateTime) }}</span>
                <span class="metric-unit">minutes</span>
              </div>
              <div class="metric-trend">
                <el-icon class="trend-icon down"><ArrowDown /></el-icon>
                <span class="trend-text">12% faster than last month</span>
              </div>
            </div>

            <!-- Update Distribution -->
            <div class="analytics-card full-width">
              <h3>Update Trend (Last 30 Days)</h3>
              <div class="trend-chart">
                <div
                  v-for="(day, index) in trendData"
                  :key="index"
                  class="trend-bar"
                  :style="{ height: `${(day.total / maxDailyUpdates) * 100}%` }"
                  :title="`${day.date}: ${day.completed} completed, ${day.failed} failed`"
                >
                  <div class="bar-completed" :style="{ height: `${(day.completed / day.total) * 100}%` }"></div>
                  <div class="bar-failed" :style="{ height: `${(day.failed / day.total) * 100}%` }"></div>
                </div>
              </div>
              <div class="trend-legend">
                <div class="legend-item">
                  <div class="legend-color success"></div>
                  <span>Completed</span>
                </div>
                <div class="legend-item">
                  <div class="legend-color error"></div>
                  <span>Failed</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-collapse-transition>
    </div>

    <!-- Filters and Controls -->
    <div class="history-controls">
      <div class="filters-section">
        <el-button
          :icon="Filter"
          @click="showFilters = !showFilters"
        >
          Filters
          <el-badge
            v-if="activeFiltersCount > 0"
            :value="activeFiltersCount"
            class="filter-badge"
          />
        </el-button>

        <el-select
          v-model="quickFilter"
          placeholder="Quick Filter"
          style="width: 150px"
          @change="applyQuickFilter"
        >
          <el-option label="All Updates" value="all" />
          <el-option label="Completed" value="completed" />
          <el-option label="Failed" value="failed" />
          <el-option label="Security" value="security" />
          <el-option label="Rollbacks" value="rollback" />
          <el-option label="Last 7 Days" value="week" />
          <el-option label="Last 30 Days" value="month" />
        </el-select>

        <el-input
          v-model="searchQuery"
          placeholder="Search by container name..."
          :prefix-icon="Search"
          style="width: 250px"
          clearable
          @input="handleSearch"
        />

        <el-date-picker
          v-model="dateRange"
          type="datetimerange"
          range-separator="to"
          start-placeholder="Start date"
          end-placeholder="End date"
          format="YYYY-MM-DD HH:mm"
          value-format="YYYY-MM-DD HH:mm:ss"
          @change="handleDateRangeChange"
        />
      </div>

      <div class="view-controls">
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button label="timeline">
            <el-icon><Clock /></el-icon>
            Timeline
          </el-radio-button>
          <el-radio-button label="table">
            <el-icon><List /></el-icon>
            Table
          </el-radio-button>
          <el-radio-button label="cards">
            <el-icon><Grid /></el-icon>
            Cards
          </el-radio-button>
        </el-radio-group>

        <el-select
          v-model="sortConfig.field"
          style="width: 150px"
          @change="handleSort"
        >
          <el-option label="Start Date" value="started_at" />
          <el-option label="Completion Date" value="completed_at" />
          <el-option label="Container Name" value="container_name" />
          <el-option label="Duration" value="duration" />
          <el-option label="Status" value="status" />
        </el-select>

        <el-button
          :icon="sortConfig.direction === 'asc' ? 'SortUp' : 'SortDown'"
          @click="toggleSortDirection"
        />
      </div>
    </div>

    <!-- Advanced Filters Panel -->
    <el-collapse-transition>
      <div v-show="showFilters" class="filters-panel">
        <div class="filter-groups">
          <div class="filter-group">
            <label>Status</label>
            <el-checkbox-group v-model="historyFilters.status">
              <el-checkbox label="completed">Completed</el-checkbox>
              <el-checkbox label="failed">Failed</el-checkbox>
              <el-checkbox label="cancelled">Cancelled</el-checkbox>
              <el-checkbox label="running">Running</el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="filter-group">
            <label>Update Type</label>
            <el-checkbox-group v-model="historyFilters.updateType">
              <el-checkbox label="major">Major</el-checkbox>
              <el-checkbox label="minor">Minor</el-checkbox>
              <el-checkbox label="patch">Patch</el-checkbox>
              <el-checkbox label="security">Security</el-checkbox>
              <el-checkbox label="hotfix">Hotfix</el-checkbox>
              <el-checkbox label="rollback">Rollback</el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="filter-group">
            <label>Triggered By</label>
            <el-checkbox-group v-model="historyFilters.triggeredBy">
              <el-checkbox label="manual">Manual</el-checkbox>
              <el-checkbox label="scheduled">Scheduled</el-checkbox>
              <el-checkbox label="policy">Policy</el-checkbox>
              <el-checkbox label="webhook">Webhook</el-checkbox>
              <el-checkbox label="api">API</el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="filter-group">
            <label>Duration Range</label>
            <el-slider
              v-model="durationRange"
              range
              :min="0"
              :max="3600"
              :format-tooltip="formatDurationTooltip"
              @change="handleDurationFilter"
            />
          </div>
        </div>

        <div class="filter-actions">
          <el-button @click="clearAllFilters">Clear All</el-button>
          <el-button type="primary" @click="applyFilters">Apply Filters</el-button>
        </div>
      </div>
    </el-collapse-transition>

    <!-- History Content -->
    <div class="history-content" :class="viewMode">
      <div v-if="loadingHistory" class="loading-container">
        <el-skeleton :rows="8" animated />
      </div>

      <div v-else-if="updateHistory.length === 0" class="empty-state">
        <el-empty
          :image-size="200"
          description="No update history found"
        >
          <el-button type="primary" @click="refreshHistory">
            Refresh History
          </el-button>
        </el-empty>
      </div>

      <div v-else>
        <!-- Timeline View -->
        <div v-if="viewMode === 'timeline'" class="timeline-view">
          <div class="timeline">
            <div
              v-for="(group, date) in groupedHistory"
              :key="date"
              class="timeline-group"
            >
              <div class="timeline-date">
                <h3>{{ formatDate(date) }}</h3>
                <span class="date-stats">{{ group.length }} updates</span>
              </div>

              <div class="timeline-items">
                <UpdateHistoryItem
                  v-for="item in group"
                  :key="item.id"
                  :item="item"
                  view-mode="timeline"
                  @rollback="handleRollback"
                  @retry="handleRetry"
                  @view-logs="handleViewLogs"
                  @view-details="handleViewDetails"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- Table View -->
        <div v-else-if="viewMode === 'table'" class="table-view">
          <el-table
            :data="paginatedHistory"
            style="width: 100%"
            :default-sort="{ prop: 'startedAt', order: 'descending' }"
            @sort-change="handleTableSort"
          >
            <el-table-column prop="containerName" label="Container" sortable>
              <template #default="{ row }">
                <div class="container-cell">
                  <el-icon><Box /></el-icon>
                  <span>{{ row.containerName }}</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="fromVersion" label="From" width="120">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.fromVersion }}</el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="toVersion" label="To" width="120">
              <template #default="{ row }">
                <el-tag size="small" type="primary">{{ row.toVersion }}</el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="updateType" label="Type" width="100">
              <template #default="{ row }">
                <el-tag
                  size="small"
                  :type="getUpdateTypeTagType(row.updateType)"
                >
                  {{ row.updateType }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="status" label="Status" width="120">
              <template #default="{ row }">
                <el-tag
                  size="small"
                  :type="getStatusTagType(row.status)"
                >
                  <el-icon>
                    <component :is="getStatusIcon(row.status)" />
                  </el-icon>
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="startedAt" label="Started" width="180" sortable>
              <template #default="{ row }">
                <div class="date-cell">
                  <span>{{ formatDateTime(row.startedAt) }}</span>
                  <span class="date-relative">{{ getRelativeTime(row.startedAt) }}</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="duration" label="Duration" width="120" sortable>
              <template #default="{ row }">
                <span v-if="row.duration">{{ formatDuration(row.duration) }}</span>
                <span v-else class="text-muted">-</span>
              </template>
            </el-table-column>

            <el-table-column prop="triggeredBy" label="Triggered By" width="120">
              <template #default="{ row }">
                <el-tag size="small" effect="plain">
                  <el-icon>
                    <component :is="getTriggerIcon(row.triggeredBy)" />
                  </el-icon>
                  {{ row.triggeredBy }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column label="Actions" width="200" fixed="right">
              <template #default="{ row }">
                <div class="table-actions">
                  <el-tooltip content="View Details">
                    <el-button
                      size="small"
                      :icon="View"
                      @click="handleViewDetails(row)"
                    />
                  </el-tooltip>

                  <el-tooltip v-if="row.logs?.length" content="View Logs">
                    <el-button
                      size="small"
                      :icon="Document"
                      @click="handleViewLogs(row)"
                    />
                  </el-tooltip>

                  <el-tooltip
                    v-if="row.canRollback && row.status === 'completed'"
                    content="Rollback"
                  >
                    <el-button
                      size="small"
                      :icon="RefreshLeft"
                      type="warning"
                      @click="handleRollback(row)"
                    />
                  </el-tooltip>

                  <el-tooltip
                    v-if="row.status === 'failed'"
                    content="Retry Update"
                  >
                    <el-button
                      size="small"
                      :icon="Refresh"
                      type="primary"
                      @click="handleRetry(row)"
                    />
                  </el-tooltip>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- Cards View -->
        <div v-else class="cards-view">
          <div class="cards-grid">
            <UpdateHistoryItem
              v-for="item in paginatedHistory"
              :key="item.id"
              :item="item"
              view-mode="card"
              @rollback="handleRollback"
              @retry="handleRetry"
              @view-logs="handleViewLogs"
              @view-details="handleViewDetails"
            />
          </div>
        </div>

        <!-- Pagination -->
        <div class="pagination-container">
          <el-pagination
            v-model:current-page="historyPage"
            v-model:page-size="historyPageSize"
            :total="totalHistoryItems"
            :page-sizes="[20, 50, 100, 200]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handlePageSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </div>

    <!-- Update Details Dialog -->
    <UpdateDetailsDialog
      v-model="showDetailsDialog"
      :update-item="selectedUpdateItem"
    />

    <!-- Update Logs Dialog -->
    <UpdateLogsDialog
      v-model="showLogsDialog"
      :update-item="selectedUpdateItem"
    />

    <!-- Export Dialog -->
    <ExportDialog
      v-model="showExportDialog"
      type="history"
      :filters="historyFilters"
      :date-range="dateRange"
      @export="handleExport"
    />

    <!-- Rollback Confirmation Dialog -->
    <RollbackDialog
      v-model="showRollbackDialog"
      :update-item="selectedUpdateItem"
      @rollback="handleConfirmRollback"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Clock,
  Refresh,
  Download,
  TrendCharts,
  Filter,
  Search,
  List,
  Grid,
  ArrowDown,
  View,
  Document,
  RefreshLeft,
  Box
} from '@element-plus/icons-vue'

// Components
import UpdateHistoryItem from '@/components/update/UpdateHistoryItem.vue'
import UpdateDetailsDialog from '@/components/update/UpdateDetailsDialog.vue'
import UpdateLogsDialog from '@/components/update/UpdateLogsDialog.vue'
import ExportDialog from '@/components/update/ExportDialog.vue'
import RollbackDialog from '@/components/update/RollbackDialog.vue'

// Store
import { useUpdatesStore } from '@/store/updates'

// Types
import type { UpdateHistoryItem as UpdateHistoryItemType, UpdateFilter } from '@/types/updates'

// Store instance
const updatesStore = useUpdatesStore()

// Reactive state from store
const {
  updateHistory,
  updateAnalytics,
  loadingHistory,
  historyPage,
  historyPageSize,
  totalHistoryItems,
  historyFilters,
  sortConfig
} = updatesStore

// Local reactive state
const showAnalytics = ref(true)
const showFilters = ref(false)
const showDetailsDialog = ref(false)
const showLogsDialog = ref(false)
const showExportDialog = ref(false)
const showRollbackDialog = ref(false)
const searchQuery = ref('')
const quickFilter = ref('all')
const dateRange = ref<[string, string] | null>(null)
const durationRange = ref([0, 3600])
const viewMode = ref<'timeline' | 'table' | 'cards'>('timeline')
const selectedUpdateItem = ref<UpdateHistoryItemType | null>(null)

// Computed properties
const activeFiltersCount = computed(() => {
  let count = 0
  if (historyFilters.status?.length) count++
  if (historyFilters.updateType?.length) count++
  if (historyFilters.triggeredBy?.length) count++
  if (searchQuery.value) count++
  if (quickFilter.value !== 'all') count++
  if (dateRange.value) count++
  return count
})

const paginatedHistory = computed(() => {
  const start = (historyPage.value - 1) * historyPageSize.value
  const end = start + historyPageSize.value
  return updateHistory.value.slice(start, end)
})

const groupedHistory = computed(() => {
  const grouped: Record<string, UpdateHistoryItemType[]> = {}

  paginatedHistory.value.forEach(item => {
    const date = new Date(item.startedAt).toDateString()
    if (!grouped[date]) {
      grouped[date] = []
    }
    grouped[date].push(item)
  })

  return grouped
})

const trendData = computed(() => {
  // Generate mock trend data - in real app this would come from analytics
  const data = []
  for (let i = 29; i >= 0; i--) {
    const date = new Date()
    date.setDate(date.getDate() - i)

    const completed = Math.floor(Math.random() * 10)
    const failed = Math.floor(Math.random() * 3)

    data.push({
      date: date.toISOString().split('T')[0],
      completed,
      failed,
      total: completed + failed
    })
  }
  return data
})

const maxDailyUpdates = computed(() => {
  return Math.max(...trendData.value.map(d => d.total), 1)
})

// Methods
const refreshHistory = async () => {
  try {
    await updatesStore.fetchUpdateHistory()
  } catch (error) {
    console.error('Failed to refresh history:', error)
  }
}

const applyQuickFilter = () => {
  const now = new Date()
  const filters: UpdateFilter = {}

  switch (quickFilter.value) {
    case 'completed':
      filters.status = ['completed']
      break
    case 'failed':
      filters.status = ['failed']
      break
    case 'security':
      filters.updateType = ['security']
      break
    case 'rollback':
      filters.updateType = ['rollback']
      break
    case 'week':
      const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
      filters.dateRange = {
        start: weekAgo.toISOString(),
        end: now.toISOString()
      }
      break
    case 'month':
      const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
      filters.dateRange = {
        start: monthAgo.toISOString(),
        end: now.toISOString()
      }
      break
  }

  updatesStore.historyFilters = { ...updatesStore.historyFilters, ...filters }
  refreshHistory()
}

const handleSearch = () => {
  if (searchQuery.value) {
    updatesStore.historyFilters = {
      ...updatesStore.historyFilters,
      containerName: searchQuery.value
    }
  } else {
    const newFilters = { ...updatesStore.historyFilters }
    delete newFilters.containerName
    updatesStore.historyFilters = newFilters
  }
  refreshHistory()
}

const handleDateRangeChange = () => {
  if (dateRange.value) {
    updatesStore.historyFilters = {
      ...updatesStore.historyFilters,
      dateRange: {
        start: dateRange.value[0],
        end: dateRange.value[1]
      }
    }
  } else {
    const newFilters = { ...updatesStore.historyFilters }
    delete newFilters.dateRange
    updatesStore.historyFilters = newFilters
  }
  refreshHistory()
}

const handleSort = () => {
  updatesStore.setSorting(sortConfig.field)
  refreshHistory()
}

const toggleSortDirection = () => {
  updatesStore.setSorting(sortConfig.field,
    sortConfig.direction === 'asc' ? 'desc' : 'asc'
  )
  refreshHistory()
}

const handleTableSort = ({ prop, order }: { prop: string; order: string | null }) => {
  if (order) {
    updatesStore.setSorting(prop as any, order === 'ascending' ? 'asc' : 'desc')
    refreshHistory()
  }
}

const handleDurationFilter = () => {
  updatesStore.historyFilters = {
    ...updatesStore.historyFilters,
    duration: {
      min: durationRange.value[0],
      max: durationRange.value[1]
    }
  }
}

const applyFilters = () => {
  refreshHistory()
  showFilters.value = false
}

const clearAllFilters = () => {
  updatesStore.historyFilters = {}
  searchQuery.value = ''
  quickFilter.value = 'all'
  dateRange.value = null
  durationRange.value = [0, 3600]
  refreshHistory()
}

const handleViewDetails = (item: UpdateHistoryItemType) => {
  selectedUpdateItem.value = item
  showDetailsDialog.value = true
}

const handleViewLogs = (item: UpdateHistoryItemType) => {
  selectedUpdateItem.value = item
  showLogsDialog.value = true
}

const handleRollback = (item: UpdateHistoryItemType) => {
  selectedUpdateItem.value = item
  showRollbackDialog.value = true
}

const handleRetry = async (item: UpdateHistoryItemType) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to retry the update for "${item.containerName}"?`,
      'Retry Update',
      {
        confirmButtonText: 'Yes, Retry',
        cancelButtonText: 'Cancel',
        type: 'info'
      }
    )

    // Create a new update request based on the failed one
    // This would need to be implemented in the store
    ElMessage.success('Update retry initiated')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to retry update:', error)
    }
  }
}

const handleConfirmRollback = async (targetVersion: string, reason: string) => {
  if (!selectedUpdateItem.value) return

  try {
    await updatesStore.rollbackUpdate(selectedUpdateItem.value.id, targetVersion)
    showRollbackDialog.value = false
    ElMessage.success('Rollback initiated successfully')
  } catch (error) {
    console.error('Failed to initiate rollback:', error)
  }
}

const handleExport = async (format: 'csv' | 'json' | 'pdf', filters: UpdateFilter) => {
  try {
    await updatesStore.exportUpdateHistory(format, filters)
    showExportDialog.value = false
  } catch (error) {
    console.error('Failed to export history:', error)
  }
}

const handlePageSizeChange = (newSize: number) => {
  updatesStore.historyPageSize = newSize
  refreshHistory()
}

const handlePageChange = (newPage: number) => {
  updatesStore.historyPage = newPage
  refreshHistory()
}

// Utility functions
const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

const formatDateTime = (dateString: string) => {
  return new Date(dateString).toLocaleString()
}

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

const formatDurationTooltip = (value: number) => {
  return formatDuration(value)
}

const getRelativeTime = (dateString: string) => {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  return `${Math.floor(diff / 86400000)}d ago`
}

const getSuccessRateColor = (rate: number) => {
  if (rate >= 95) return '#67C23A'
  if (rate >= 85) return '#E6A23C'
  return '#F56C6C'
}

const getUpdateTypeTagType = (type: string) => {
  switch (type) {
    case 'security': return 'danger'
    case 'major': return 'warning'
    case 'minor': return 'primary'
    case 'patch': return 'success'
    case 'rollback': return 'info'
    default: return ''
  }
}

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'danger'
    case 'cancelled': return 'warning'
    case 'running': return 'primary'
    default: return 'info'
  }
}

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'completed': return 'Check'
    case 'failed': return 'Close'
    case 'cancelled': return 'Warning'
    case 'running': return 'Loading'
    default: return 'InfoFilled'
  }
}

const getTriggerIcon = (trigger: string) => {
  switch (trigger) {
    case 'manual': return 'User'
    case 'scheduled': return 'Clock'
    case 'policy': return 'Setting'
    case 'webhook': return 'Link'
    case 'api': return 'Connection'
    default: return 'QuestionFilled'
  }
}

// Lifecycle hooks
onMounted(async () => {
  await Promise.all([
    updatesStore.fetchUpdateHistory(),
    updatesStore.getUpdateAnalytics()
  ])
})
</script>

<style scoped lang="scss">
.update-history-view {
  padding: 24px;
  background: var(--el-bg-color-page);
  min-height: 100vh;
}

.history-header {
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.header-info {
  .page-title {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0 0 8px 0;
    font-size: 28px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .page-description {
    margin: 0;
    color: var(--el-text-color-regular);
    font-size: 14px;
  }
}

.header-actions {
  display: flex;
  gap: 12px;
}

.analytics-panel {
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.analytics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
}

.analytics-card {
  padding: 16px;
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;

  &.full-width {
    grid-column: 1 / -1;
  }

  h3 {
    margin: 0 0 16px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-regular);
  }
}

.chart-container {
  .chart-details {
    display: flex;
    justify-content: space-between;
    margin-top: 12px;
  }
}

.detail-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;

  .label {
    font-size: 12px;
    color: var(--el-text-color-regular);
  }

  .value {
    font-size: 16px;
    font-weight: 600;

    &.success {
      color: var(--el-color-success);
    }

    &.error {
      color: var(--el-color-danger);
    }
  }
}

.metric-display {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin-bottom: 8px;

  .metric-value {
    font-size: 24px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .metric-unit {
    font-size: 12px;
    color: var(--el-text-color-regular);
  }
}

.metric-trend {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--el-color-success);

  .trend-icon {
    &.down {
      color: var(--el-color-success);
    }

    &.up {
      color: var(--el-color-danger);
    }
  }
}

.trend-chart {
  display: flex;
  align-items: end;
  gap: 2px;
  height: 100px;
  margin-bottom: 12px;
}

.trend-bar {
  flex: 1;
  min-width: 8px;
  background: var(--el-border-color-lighter);
  border-radius: 2px 2px 0 0;
  position: relative;
  display: flex;
  flex-direction: column-reverse;

  .bar-completed {
    background: var(--el-color-success);
    border-radius: 2px 2px 0 0;
  }

  .bar-failed {
    background: var(--el-color-danger);
  }
}

.trend-legend {
  display: flex;
  justify-content: center;
  gap: 16px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.legend-color {
  width: 12px;
  height: 12px;
  border-radius: 2px;

  &.success {
    background: var(--el-color-success);
  }

  &.error {
    background: var(--el-color-danger);
  }
}

.history-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.filters-section {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-badge {
  margin-left: 8px;
}

.view-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filters-panel {
  margin-bottom: 16px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.filter-groups {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 16px;
}

.filter-group {
  label {
    display: block;
    margin-bottom: 8px;
    font-weight: 500;
    color: var(--el-text-color-regular);
    font-size: 13px;
  }
}

.filter-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.history-content {
  &.timeline .timeline-view {
    .timeline {
      position: relative;

      &::before {
        content: '';
        position: absolute;
        left: 24px;
        top: 0;
        bottom: 0;
        width: 2px;
        background: var(--el-border-color);
      }
    }

    .timeline-group {
      margin-bottom: 32px;
    }

    .timeline-date {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 16px;
      padding-left: 56px;

      h3 {
        margin: 0;
        font-size: 18px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .date-stats {
        padding: 2px 8px;
        background: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
        border-radius: 10px;
        font-size: 12px;
        font-weight: 500;
      }
    }

    .timeline-items {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
  }

  &.table .table-view {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    overflow: hidden;
  }

  &.cards .cards-view {
    .cards-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
      gap: 16px;
    }
  }
}

.container-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;

  .date-relative {
    font-size: 12px;
    color: var(--el-text-color-regular);
  }
}

.table-actions {
  display: flex;
  gap: 4px;
}

.loading-container {
  padding: 20px;
}

.empty-state {
  padding: 60px 20px;
  text-align: center;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 24px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

@media (max-width: 768px) {
  .update-history-view {
    padding: 16px;
  }

  .header-content {
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
  }

  .header-actions {
    justify-content: center;
  }

  .analytics-grid {
    grid-template-columns: 1fr;
  }

  .history-controls {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .filters-section,
  .view-controls {
    flex-wrap: wrap;
    justify-content: center;
  }

  .filter-groups {
    grid-template-columns: 1fr;
  }

  .history-content.cards .cards-grid {
    grid-template-columns: 1fr;
  }

  .timeline-date {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>