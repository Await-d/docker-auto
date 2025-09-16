/**
 * Update management store
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { updatesAPI } from '@/api/updates'
import type {
  ContainerUpdate,
  UpdateHistoryItem,
  RunningUpdate,
  UpdatePolicy,
  ScheduledUpdate,
  UpdateSettings,
  UpdateFilter,
  UpdateSort,
  BulkUpdateOperation,
  UpdateTemplate,
  SecurityPatch,
  UpdateAnalytics,
  RollbackOperation,
  UpdateNotification,
  VersionComparison,
  UpdateStrategy
} from '@/types/updates'

export const useUpdatesStore = defineStore('updates', () => {
  // State
  const availableUpdates = ref<ContainerUpdate[]>([])
  const updateHistory = ref<UpdateHistoryItem[]>([])
  const runningUpdates = ref<RunningUpdate[]>([])
  const updatePolicies = ref<UpdatePolicy[]>([])
  const scheduledUpdates = ref<ScheduledUpdate[]>([])
  const updateTemplates = ref<UpdateTemplate[]>([])
  const updateSettings = ref<UpdateSettings>({
    autoCheckInterval: 3600000, // 1 hour
    enableNotifications: true,
    requireApproval: true,
    maxConcurrentUpdates: 3,
    updateStrategy: 'rolling',
    rollbackOnFailure: true,
    testBeforeUpdate: false,
    maintenanceWindow: {
      enabled: false,
      start: '02:00',
      end: '04:00',
      timezone: 'UTC',
      days: ['sunday']
    }
  })

  // Loading states
  const loading = ref(false)
  const loadingHistory = ref(false)
  const loadingPolicies = ref(false)
  const loadingScheduled = ref(false)
  const checkingUpdates = ref(false)
  const updatingContainer = ref<Record<string, boolean>>({})

  // Pagination and filtering
  const currentPage = ref(1)
  const pageSize = ref(20)
  const totalUpdates = ref(0)
  const historyPage = ref(1)
  const historyPageSize = ref(50)
  const totalHistoryItems = ref(0)
  const filters = ref<UpdateFilter>({})
  const historyFilters = ref<UpdateFilter>({})
  const sortConfig = ref<UpdateSort>({ field: 'available_date', direction: 'desc' })

  // UI state
  const selectedUpdates = ref<Set<string>>(new Set())
  const viewMode = ref<'grid' | 'list'>('list')
  const showFilters = ref(false)
  const autoRefresh = ref(true)
  const refreshInterval = ref(300000) // 5 minutes
  const expandedUpdates = ref<Set<string>>(new Set())

  // Real-time updates
  const wsConnected = ref(false)
  const lastUpdateCheck = ref<Date>(new Date())
  const updateNotifications = ref<UpdateNotification[]>([])

  // Analytics
  const updateAnalytics = ref<UpdateAnalytics>({
    totalUpdatesAvailable: 0,
    securityUpdates: 0,
    criticalUpdates: 0,
    successRate: 0,
    averageUpdateTime: 0,
    failedUpdatesLast30Days: 0,
    updatesThisMonth: 0
  })

  // Computed
  const securityUpdates = computed(() =>
    availableUpdates.value.filter(u => u.updateType === 'security' || u.securityPatches.length > 0)
  )

  const criticalUpdates = computed(() =>
    availableUpdates.value.filter(u => u.riskLevel === 'high' || u.priority === 'critical')
  )

  const pendingUpdates = computed(() =>
    availableUpdates.value.filter(u => !u.ignored && !u.scheduled)
  )

  const ignoredUpdates = computed(() =>
    availableUpdates.value.filter(u => u.ignored)
  )

  const scheduledUpdatesCount = computed(() =>
    scheduledUpdates.value.length
  )

  const runningUpdatesCount = computed(() =>
    runningUpdates.value.length
  )

  const totalPages = computed(() =>
    Math.ceil(totalUpdates.value / pageSize.value)
  )

  const totalHistoryPages = computed(() =>
    Math.ceil(totalHistoryItems.value / historyPageSize.value)
  )

  const isAllSelected = computed(() =>
    availableUpdates.value.length > 0 &&
    availableUpdates.value.every(u => selectedUpdates.value.has(u.id))
  )

  const hasSelection = computed(() =>
    selectedUpdates.value.size > 0
  )

  const filteredUpdates = computed(() => {
    let result = availableUpdates.value

    if (filters.value.updateType?.length) {
      result = result.filter(u => filters.value.updateType!.includes(u.updateType))
    }

    if (filters.value.riskLevel?.length) {
      result = result.filter(u => filters.value.riskLevel!.includes(u.riskLevel))
    }

    if (filters.value.containerName) {
      const search = filters.value.containerName.toLowerCase()
      result = result.filter(u =>
        u.containerName.toLowerCase().includes(search)
      )
    }

    if (filters.value.imageName) {
      const search = filters.value.imageName.toLowerCase()
      result = result.filter(u =>
        u.currentVersion.toLowerCase().includes(search) ||
        u.availableVersion.toLowerCase().includes(search)
      )
    }

    if (filters.value.securityOnly) {
      result = result.filter(u => u.updateType === 'security' || u.securityPatches.length > 0)
    }

    if (filters.value.ignored !== undefined) {
      result = result.filter(u => u.ignored === filters.value.ignored)
    }

    return result
  })

  const recentHistory = computed(() =>
    updateHistory.value.slice(0, 10)
  )

  // Actions
  async function checkForUpdates(containerId?: string, force = false) {
    checkingUpdates.value = true
    try {
      const updates = await updatesAPI.checkUpdates(containerId, force)
      availableUpdates.value = updates.updates
      totalUpdates.value = updates.total
      lastUpdateCheck.value = new Date()

      // Update analytics
      updateAnalytics.value.totalUpdatesAvailable = updates.total
      updateAnalytics.value.securityUpdates = updates.updates.filter(u =>
        u.updateType === 'security' || u.securityPatches.length > 0
      ).length
      updateAnalytics.value.criticalUpdates = updates.updates.filter(u =>
        u.riskLevel === 'high'
      ).length

      if (updates.total > 0 && updateSettings.value.enableNotifications) {
        ElNotification({
          title: 'Updates Available',
          message: `${updates.total} container update(s) available`,
          type: updates.updates.some(u => u.updateType === 'security') ? 'warning' : 'info',
          duration: 5000
        })
      }

      return updates
    } catch (error) {
      console.error('Failed to check updates:', error)
      ElMessage.error('Failed to check for updates')
      throw error
    } finally {
      checkingUpdates.value = false
    }
  }

  async function fetchUpdateHistory(page?: number, filters?: UpdateFilter) {
    if (page) historyPage.value = page
    if (filters) historyFilters.value = filters

    loadingHistory.value = true
    try {
      const response = await updatesAPI.getUpdateHistory(
        historyPage.value,
        historyPageSize.value,
        historyFilters.value,
        sortConfig.value
      )

      updateHistory.value = response.items
      totalHistoryItems.value = response.total

      // Update analytics from history
      const last30Days = new Date()
      last30Days.setDate(last30Days.getDate() - 30)

      const recentHistory = response.items.filter(item =>
        new Date(item.startedAt) > last30Days
      )

      updateAnalytics.value.failedUpdatesLast30Days = recentHistory.filter(
        item => item.status === 'failed'
      ).length

      updateAnalytics.value.updatesThisMonth = recentHistory.filter(
        item => item.status === 'completed'
      ).length

      if (recentHistory.length > 0) {
        updateAnalytics.value.successRate =
          (recentHistory.filter(item => item.status === 'completed').length / recentHistory.length) * 100

        const completedUpdates = recentHistory.filter(item =>
          item.status === 'completed' && item.duration
        )
        if (completedUpdates.length > 0) {
          updateAnalytics.value.averageUpdateTime =
            completedUpdates.reduce((sum, item) => sum + (item.duration || 0), 0) / completedUpdates.length
        }
      }

      return response
    } catch (error) {
      console.error('Failed to fetch update history:', error)
      ElMessage.error('Failed to load update history')
      throw error
    } finally {
      loadingHistory.value = false
    }
  }

  async function startUpdate(updateId: string, options?: {
    strategy?: UpdateStrategy
    rollbackOnFailure?: boolean
    notifyOnCompletion?: boolean
    runTests?: boolean
  }) {
    const update = availableUpdates.value.find(u => u.id === updateId)
    if (!update) return

    setUpdateLoading(update.containerId, true)
    try {
      const result = await updatesAPI.startUpdate(updateId, {
        strategy: options?.strategy || updateSettings.value.updateStrategy,
        rollbackOnFailure: options?.rollbackOnFailure ?? updateSettings.value.rollbackOnFailure,
        notifyOnCompletion: options?.notifyOnCompletion ?? updateSettings.value.enableNotifications,
        runTests: options?.runTests ?? updateSettings.value.testBeforeUpdate
      })

      // Add to running updates
      runningUpdates.value.push({
        id: result.id,
        updateId: updateId,
        containerId: update.containerId,
        containerName: update.containerName,
        fromVersion: update.currentVersion,
        toVersion: update.availableVersion,
        status: 'running',
        progress: 0,
        startedAt: new Date().toISOString(),
        steps: result.steps || [],
        currentStep: 0,
        logs: [],
        estimatedDuration: update.estimatedDowntime
      })

      // Remove from available updates
      const index = availableUpdates.value.findIndex(u => u.id === updateId)
      if (index !== -1) {
        availableUpdates.value.splice(index, 1)
        totalUpdates.value--
      }

      ElNotification({
        title: 'Update Started',
        message: `Update for "${update.containerName}" has been initiated`,
        type: 'info'
      })

      return result
    } catch (error) {
      console.error('Failed to start update:', error)
      ElMessage.error(`Failed to start update for ${update.containerName}`)
      throw error
    } finally {
      setUpdateLoading(update.containerId, false)
    }
  }

  async function startBulkUpdate(updateIds: string[], options?: {
    strategy?: 'sequential' | 'parallel' | 'rolling'
    maxConcurrent?: number
    continueOnError?: boolean
  }) {
    const operation: BulkUpdateOperation = {
      updateIds,
      strategy: options?.strategy || 'sequential',
      maxConcurrent: options?.maxConcurrent || updateSettings.value.maxConcurrentUpdates,
      continueOnError: options?.continueOnError ?? true,
      rollbackOnFailure: updateSettings.value.rollbackOnFailure,
      runTests: updateSettings.value.testBeforeUpdate
    }

    try {
      const result = await updatesAPI.startBulkUpdate(operation)

      ElNotification({
        title: 'Bulk Update Started',
        message: `Bulk update operation started for ${updateIds.length} container(s)`,
        type: 'info'
      })

      // Clear selection
      selectedUpdates.value.clear()

      // Refresh data
      await Promise.all([
        checkForUpdates(),
        fetchRunningUpdates()
      ])

      return result
    } catch (error) {
      console.error('Failed to start bulk update:', error)
      ElMessage.error('Failed to start bulk update operation')
      throw error
    }
  }

  async function scheduleUpdate(
    updateId: string,
    scheduledAt: Date,
    options?: {
      recurring?: boolean
      recurringPattern?: string
      notifyBefore?: number
    }
  ) {
    try {
      const result = await updatesAPI.scheduleUpdate(updateId, {
        scheduledAt: scheduledAt.toISOString(),
        recurring: options?.recurring || false,
        recurringPattern: options?.recurringPattern,
        notifyBefore: options?.notifyBefore || 300000, // 5 minutes
        strategy: updateSettings.value.updateStrategy,
        rollbackOnFailure: updateSettings.value.rollbackOnFailure
      })

      // Add to scheduled updates
      scheduledUpdates.value.push(result)

      const update = availableUpdates.value.find(u => u.id === updateId)
      if (update) {
        update.scheduled = true
        update.scheduledAt = scheduledAt.toISOString()
      }

      ElMessage.success('Update scheduled successfully')
      return result
    } catch (error) {
      console.error('Failed to schedule update:', error)
      ElMessage.error('Failed to schedule update')
      throw error
    }
  }

  async function cancelUpdate(runningUpdateId: string) {
    const runningUpdate = runningUpdates.value.find(u => u.id === runningUpdateId)
    if (!runningUpdate) return

    try {
      await updatesAPI.cancelUpdate(runningUpdateId)

      // Remove from running updates
      const index = runningUpdates.value.findIndex(u => u.id === runningUpdateId)
      if (index !== -1) {
        runningUpdates.value.splice(index, 1)
      }

      ElMessage.success('Update cancelled successfully')
    } catch (error) {
      console.error('Failed to cancel update:', error)
      ElMessage.error('Failed to cancel update')
      throw error
    }
  }

  async function rollbackUpdate(historyItemId: string, targetVersion?: string) {
    const historyItem = updateHistory.value.find(h => h.id === historyItemId)
    if (!historyItem) return

    try {
      const rollback: RollbackOperation = {
        historyItemId,
        containerId: historyItem.containerId,
        targetVersion: targetVersion || historyItem.fromVersion,
        reason: 'Manual rollback requested'
      }

      const result = await updatesAPI.rollbackUpdate(rollback)

      // Add rollback to history
      const rollbackHistoryItem: UpdateHistoryItem = {
        id: result.id,
        containerId: historyItem.containerId,
        containerName: historyItem.containerName,
        fromVersion: historyItem.toVersion,
        toVersion: rollback.targetVersion,
        updateType: 'rollback',
        status: 'running',
        startedAt: new Date().toISOString(),
        triggeredBy: 'manual',
        reason: rollback.reason
      }

      updateHistory.value.unshift(rollbackHistoryItem)
      totalHistoryItems.value++

      ElNotification({
        title: 'Rollback Started',
        message: `Rolling back "${historyItem.containerName}" to version ${rollback.targetVersion}`,
        type: 'warning'
      })

      return result
    } catch (error) {
      console.error('Failed to rollback update:', error)
      ElMessage.error('Failed to rollback update')
      throw error
    }
  }

  async function ignoreUpdate(updateId: string, reason?: string) {
    const update = availableUpdates.value.find(u => u.id === updateId)
    if (!update) return

    try {
      await updatesAPI.ignoreUpdate(updateId, reason)
      update.ignored = true
      update.ignoredReason = reason
      update.ignoredAt = new Date().toISOString()

      ElMessage.success(`Update for "${update.containerName}" has been ignored`)
    } catch (error) {
      console.error('Failed to ignore update:', error)
      ElMessage.error('Failed to ignore update')
      throw error
    }
  }

  async function unignoreUpdate(updateId: string) {
    const update = availableUpdates.value.find(u => u.id === updateId)
    if (!update) return

    try {
      await updatesAPI.unignoreUpdate(updateId)
      update.ignored = false
      update.ignoredReason = undefined
      update.ignoredAt = undefined

      ElMessage.success(`Update for "${update.containerName}" is no longer ignored`)
    } catch (error) {
      console.error('Failed to unignore update:', error)
      ElMessage.error('Failed to unignore update')
      throw error
    }
  }

  async function fetchRunningUpdates() {
    try {
      const running = await updatesAPI.getRunningUpdates()
      runningUpdates.value = running
      return running
    } catch (error) {
      console.error('Failed to fetch running updates:', error)
      throw error
    }
  }

  async function fetchScheduledUpdates() {
    loadingScheduled.value = true
    try {
      const scheduled = await updatesAPI.getScheduledUpdates()
      scheduledUpdates.value = scheduled
      return scheduled
    } catch (error) {
      console.error('Failed to fetch scheduled updates:', error)
      ElMessage.error('Failed to load scheduled updates')
      throw error
    } finally {
      loadingScheduled.value = false
    }
  }

  async function fetchUpdatePolicies() {
    loadingPolicies.value = true
    try {
      const policies = await updatesAPI.getUpdatePolicies()
      updatePolicies.value = policies
      return policies
    } catch (error) {
      console.error('Failed to fetch update policies:', error)
      ElMessage.error('Failed to load update policies')
      throw error
    } finally {
      loadingPolicies.value = false
    }
  }

  async function saveUpdatePolicy(policy: Partial<UpdatePolicy>) {
    try {
      const savedPolicy = policy.id
        ? await updatesAPI.updatePolicy(policy.id, policy)
        : await updatesAPI.createPolicy(policy)

      if (policy.id) {
        const index = updatePolicies.value.findIndex(p => p.id === policy.id)
        if (index !== -1) {
          updatePolicies.value[index] = savedPolicy
        }
      } else {
        updatePolicies.value.push(savedPolicy)
      }

      ElMessage.success('Update policy saved successfully')
      return savedPolicy
    } catch (error) {
      console.error('Failed to save update policy:', error)
      ElMessage.error('Failed to save update policy')
      throw error
    }
  }

  async function deleteUpdatePolicy(policyId: string) {
    try {
      await updatesAPI.deletePolicy(policyId)

      const index = updatePolicies.value.findIndex(p => p.id === policyId)
      if (index !== -1) {
        updatePolicies.value.splice(index, 1)
      }

      ElMessage.success('Update policy deleted successfully')
    } catch (error) {
      console.error('Failed to delete update policy:', error)
      ElMessage.error('Failed to delete update policy')
      throw error
    }
  }

  async function compareVersions(updateId: string) {
    const update = availableUpdates.value.find(u => u.id === updateId)
    if (!update) return null

    try {
      const comparison = await updatesAPI.compareVersions(
        update.containerId,
        update.currentVersion,
        update.availableVersion
      )
      return comparison
    } catch (error) {
      console.error('Failed to compare versions:', error)
      ElMessage.error('Failed to compare versions')
      throw error
    }
  }

  async function getUpdateAnalytics(period = '30d') {
    try {
      const analytics = await updatesAPI.getAnalytics(period)
      updateAnalytics.value = { ...updateAnalytics.value, ...analytics }
      return analytics
    } catch (error) {
      console.error('Failed to fetch update analytics:', error)
      throw error
    }
  }

  async function exportUpdateHistory(format: 'csv' | 'json' | 'pdf', filters?: UpdateFilter) {
    try {
      const blob = await updatesAPI.exportHistory(format, filters)

      // Create download link
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `update-history-${new Date().toISOString().split('T')[0]}.${format}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)

      ElMessage.success(`Update history exported as ${format.toUpperCase()}`)
    } catch (error) {
      console.error('Failed to export update history:', error)
      ElMessage.error('Failed to export update history')
      throw error
    }
  }

  // Selection management
  function toggleSelection(updateId: string) {
    if (selectedUpdates.value.has(updateId)) {
      selectedUpdates.value.delete(updateId)
    } else {
      selectedUpdates.value.add(updateId)
    }
  }

  function selectAll() {
    if (isAllSelected.value) {
      selectedUpdates.value.clear()
    } else {
      availableUpdates.value.forEach(u => selectedUpdates.value.add(u.id))
    }
  }

  function clearSelection() {
    selectedUpdates.value.clear()
  }

  function selectByType(type: 'security' | 'critical' | 'all') {
    selectedUpdates.value.clear()

    let updates: ContainerUpdate[] = []
    switch (type) {
      case 'security':
        updates = securityUpdates.value
        break
      case 'critical':
        updates = criticalUpdates.value
        break
      case 'all':
        updates = availableUpdates.value
        break
    }

    updates.forEach(u => selectedUpdates.value.add(u.id))
  }

  // Utility functions
  function setUpdateLoading(containerId: string, loading: boolean) {
    if (loading) {
      updatingContainer.value[containerId] = true
    } else {
      delete updatingContainer.value[containerId]
    }
  }

  function isUpdateLoading(containerId: string): boolean {
    return updatingContainer.value[containerId] === true
  }

  function setFilters(newFilters: UpdateFilter) {
    filters.value = { ...newFilters }
    currentPage.value = 1
  }

  function setSorting(field: UpdateSort['field'], direction?: UpdateSort['direction']) {
    if (sortConfig.value.field === field && !direction) {
      sortConfig.value.direction = sortConfig.value.direction === 'asc' ? 'desc' : 'asc'
    } else {
      sortConfig.value = { field, direction: direction || 'desc' }
    }
    currentPage.value = 1
  }

  function clearFilters() {
    filters.value = {}
    currentPage.value = 1
  }

  function toggleExpanded(updateId: string) {
    if (expandedUpdates.value.has(updateId)) {
      expandedUpdates.value.delete(updateId)
    } else {
      expandedUpdates.value.add(updateId)
    }
  }

  function isExpanded(updateId: string): boolean {
    return expandedUpdates.value.has(updateId)
  }

  function addNotification(notification: UpdateNotification) {
    updateNotifications.value.unshift(notification)

    // Keep only last 50 notifications
    if (updateNotifications.value.length > 50) {
      updateNotifications.value = updateNotifications.value.slice(0, 50)
    }
  }

  function clearNotifications() {
    updateNotifications.value = []
  }

  // WebSocket integration
  function handleWebSocketMessage(message: any) {
    const { type, data } = message

    switch (type) {
      case 'update_progress':
        handleUpdateProgress(data)
        break
      case 'update_completed':
        handleUpdateCompleted(data)
        break
      case 'update_failed':
        handleUpdateFailed(data)
        break
      case 'update_available':
        handleUpdateAvailable(data)
        break
      case 'update_notification':
        handleUpdateNotification(data)
        break
    }
  }

  function handleUpdateProgress(data: any) {
    const runningUpdate = runningUpdates.value.find(u => u.id === data.updateId)
    if (runningUpdate) {
      runningUpdate.progress = data.progress
      runningUpdate.currentStep = data.currentStep
      runningUpdate.status = data.status
      if (data.logs) {
        runningUpdate.logs.push(...data.logs)
      }
    }
  }

  function handleUpdateCompleted(data: any) {
    const runningUpdate = runningUpdates.value.find(u => u.id === data.updateId)
    if (runningUpdate) {
      // Remove from running updates
      const index = runningUpdates.value.findIndex(u => u.id === data.updateId)
      if (index !== -1) {
        runningUpdates.value.splice(index, 1)
      }

      // Add to history
      const historyItem: UpdateHistoryItem = {
        id: data.historyId || `${data.updateId}-${Date.now()}`,
        containerId: runningUpdate.containerId,
        containerName: runningUpdate.containerName,
        fromVersion: runningUpdate.fromVersion,
        toVersion: runningUpdate.toVersion,
        updateType: data.updateType || 'patch',
        status: 'completed',
        startedAt: runningUpdate.startedAt,
        completedAt: data.completedAt,
        duration: data.duration,
        triggeredBy: data.triggeredBy || 'manual',
        logs: runningUpdate.logs
      }

      updateHistory.value.unshift(historyItem)
      totalHistoryItems.value++

      if (updateSettings.value.enableNotifications) {
        ElNotification({
          title: 'Update Completed',
          message: `Successfully updated "${runningUpdate.containerName}" to ${runningUpdate.toVersion}`,
          type: 'success'
        })
      }
    }
  }

  function handleUpdateFailed(data: any) {
    const runningUpdate = runningUpdates.value.find(u => u.id === data.updateId)
    if (runningUpdate) {
      // Remove from running updates
      const index = runningUpdates.value.findIndex(u => u.id === data.updateId)
      if (index !== -1) {
        runningUpdates.value.splice(index, 1)
      }

      // Add to history
      const historyItem: UpdateHistoryItem = {
        id: data.historyId || `${data.updateId}-${Date.now()}`,
        containerId: runningUpdate.containerId,
        containerName: runningUpdate.containerName,
        fromVersion: runningUpdate.fromVersion,
        toVersion: runningUpdate.toVersion,
        updateType: data.updateType || 'patch',
        status: 'failed',
        startedAt: runningUpdate.startedAt,
        completedAt: data.completedAt,
        duration: data.duration,
        triggeredBy: data.triggeredBy || 'manual',
        error: data.error,
        logs: runningUpdate.logs
      }

      updateHistory.value.unshift(historyItem)
      totalHistoryItems.value++

      ElNotification({
        title: 'Update Failed',
        message: `Failed to update "${runningUpdate.containerName}": ${data.error}`,
        type: 'error',
        duration: 10000
      })
    }
  }

  function handleUpdateAvailable(data: any) {
    // Check if update already exists
    const existing = availableUpdates.value.find(u => u.containerId === data.containerId)
    if (existing) {
      // Update existing
      Object.assign(existing, data)
    } else {
      // Add new update
      availableUpdates.value.push(data)
      totalUpdates.value++
    }

    if (data.updateType === 'security' && updateSettings.value.enableNotifications) {
      ElNotification({
        title: 'Security Update Available',
        message: `Security update available for "${data.containerName}"`,
        type: 'warning'
      })
    }
  }

  function handleUpdateNotification(data: UpdateNotification) {
    addNotification(data)

    if (updateSettings.value.enableNotifications) {
      ElNotification({
        title: data.title,
        message: data.message,
        type: data.type as any,
        duration: data.duration || 5000
      })
    }
  }

  function setWebSocketConnected(connected: boolean) {
    wsConnected.value = connected
  }

  // Auto-refresh functionality
  let refreshTimer: NodeJS.Timeout | null = null

  function startAutoRefresh() {
    if (!autoRefresh.value || refreshTimer) return

    refreshTimer = setInterval(() => {
      if (document.visibilityState === 'visible') {
        checkForUpdates().catch(console.error)
        fetchRunningUpdates().catch(console.error)
      }
    }, refreshInterval.value)
  }

  function stopAutoRefresh() {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }

  function setAutoRefresh(enabled: boolean) {
    autoRefresh.value = enabled
    if (enabled) {
      startAutoRefresh()
    } else {
      stopAutoRefresh()
    }
  }

  // Reset store
  function $reset() {
    availableUpdates.value = []
    updateHistory.value = []
    runningUpdates.value = []
    updatePolicies.value = []
    scheduledUpdates.value = []
    updateTemplates.value = []
    selectedUpdates.value.clear()
    expandedUpdates.value.clear()
    updateNotifications.value = []
    loading.value = false
    loadingHistory.value = false
    loadingPolicies.value = false
    loadingScheduled.value = false
    checkingUpdates.value = false
    updatingContainer.value = {}
    currentPage.value = 1
    pageSize.value = 20
    totalUpdates.value = 0
    historyPage.value = 1
    historyPageSize.value = 50
    totalHistoryItems.value = 0
    filters.value = {}
    historyFilters.value = {}
    sortConfig.value = { field: 'available_date', direction: 'desc' }
    viewMode.value = 'list'
    showFilters.value = false
    autoRefresh.value = true
    refreshInterval.value = 300000
    wsConnected.value = false
    lastUpdateCheck.value = new Date()
    stopAutoRefresh()
  }

  return {
    // State
    availableUpdates,
    updateHistory,
    runningUpdates,
    updatePolicies,
    scheduledUpdates,
    updateTemplates,
    updateSettings,
    loading,
    loadingHistory,
    loadingPolicies,
    loadingScheduled,
    checkingUpdates,
    updatingContainer,
    currentPage,
    pageSize,
    totalUpdates,
    historyPage,
    historyPageSize,
    totalHistoryItems,
    filters,
    historyFilters,
    sortConfig,
    selectedUpdates,
    viewMode,
    showFilters,
    autoRefresh,
    refreshInterval,
    expandedUpdates,
    wsConnected,
    lastUpdateCheck,
    updateNotifications,
    updateAnalytics,

    // Computed
    securityUpdates,
    criticalUpdates,
    pendingUpdates,
    ignoredUpdates,
    scheduledUpdatesCount,
    runningUpdatesCount,
    totalPages,
    totalHistoryPages,
    isAllSelected,
    hasSelection,
    filteredUpdates,
    recentHistory,

    // Actions
    checkForUpdates,
    fetchUpdateHistory,
    startUpdate,
    startBulkUpdate,
    scheduleUpdate,
    cancelUpdate,
    rollbackUpdate,
    ignoreUpdate,
    unignoreUpdate,
    fetchRunningUpdates,
    fetchScheduledUpdates,
    fetchUpdatePolicies,
    saveUpdatePolicy,
    deleteUpdatePolicy,
    compareVersions,
    getUpdateAnalytics,
    exportUpdateHistory,
    toggleSelection,
    selectAll,
    clearSelection,
    selectByType,
    setUpdateLoading,
    isUpdateLoading,
    setFilters,
    setSorting,
    clearFilters,
    toggleExpanded,
    isExpanded,
    addNotification,
    clearNotifications,
    handleWebSocketMessage,
    setWebSocketConnected,
    startAutoRefresh,
    stopAutoRefresh,
    setAutoRefresh,
    $reset
  }
})