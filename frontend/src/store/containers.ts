/**
 * Container management store
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { containerAPI } from '@/api/container'
import type {
  Container,
  ContainerFilter,
  ContainerSort,
  ContainerFormData,
  BulkOperation,
  UpdateAvailable,
  ContainerTemplate,
  ResourceMetrics,
  ContainerLog,
  ContainerStats
} from '@/types/container'

export const useContainerStore = defineStore('containers', () => {
  // State
  const containers = ref<Container[]>([])
  const currentContainer = ref<Container | null>(null)
  const templates = ref<ContainerTemplate[]>([])
  const availableUpdates = ref<UpdateAvailable[]>([])
  const logs = ref<Map<string, ContainerLog[]>>(new Map())
  const stats = ref<Map<string, ResourceMetrics>>(new Map())
  const historicalStats = ref<Map<string, ContainerStats[]>>(new Map())

  // Loading states
  const loading = ref(false)
  const loadingDetails = ref(false)
  const loadingOperation = ref<Record<string, boolean>>({})
  const loadingBulk = ref(false)

  // Pagination and filtering
  const currentPage = ref(1)
  const pageSize = ref(20)
  const totalContainers = ref(0)
  const filters = ref<ContainerFilter>({})
  const sortConfig = ref<ContainerSort>({ field: 'name', direction: 'asc' })

  // UI state
  const selectedContainers = ref<Set<string>>(new Set())
  const viewMode = ref<'grid' | 'list'>('grid')
  const showFilters = ref(false)
  const autoRefresh = ref(true)
  const refreshInterval = ref(30000) // 30 seconds

  // Real-time updates
  const wsConnected = ref(false)
  const lastUpdate = ref<Date>(new Date())

  // Computed
  const runningContainers = computed(() =>
    containers.value.filter(c => c.status === 'running')
  )

  const stoppedContainers = computed(() =>
    containers.value.filter(c => c.status === 'exited')
  )

  const unhealthyContainers = computed(() =>
    containers.value.filter(c => c.health.status === 'unhealthy')
  )

  const containersWithUpdates = computed(() =>
    containers.value.filter(c =>
      availableUpdates.value.some(u => u.container === c.id)
    )
  )

  const totalPages = computed(() =>
    Math.ceil(totalContainers.value / pageSize.value)
  )

  const isAllSelected = computed(() =>
    containers.value.length > 0 &&
    containers.value.every(c => selectedContainers.value.has(c.id))
  )

  const hasSelection = computed(() =>
    selectedContainers.value.size > 0
  )

  const filteredContainers = computed(() => {
    let result = containers.value

    if (filters.value.status?.length) {
      result = result.filter(c => filters.value.status!.includes(c.status))
    }

    if (filters.value.image) {
      result = result.filter(c =>
        c.image.toLowerCase().includes(filters.value.image!.toLowerCase())
      )
    }

    if (filters.value.search) {
      const search = filters.value.search.toLowerCase()
      result = result.filter(c =>
        c.name.toLowerCase().includes(search) ||
        c.image.toLowerCase().includes(search) ||
        Object.keys(c.labels).some(label =>
          label.toLowerCase().includes(search) ||
          c.labels[label].toLowerCase().includes(search)
        )
      )
    }

    return result
  })

  // Actions
  async function fetchContainers(page?: number) {
    if (page) currentPage.value = page

    loading.value = true
    try {
      const response = await containerAPI.getContainers(
        currentPage.value,
        pageSize.value,
        filters.value,
        sortConfig.value
      )

      containers.value = response.containers
      totalContainers.value = response.total
      lastUpdate.value = new Date()

      // Clear selection if containers changed
      selectedContainers.value.clear()

    } catch (error) {
      console.error('Failed to fetch containers:', error)
      ElMessage.error('Failed to load containers')
    } finally {
      loading.value = false
    }
  }

  async function fetchContainer(id: string) {
    loadingDetails.value = true
    try {
      const container = await containerAPI.getContainer(id)
      currentContainer.value = container

      // Update in the list if it exists
      const index = containers.value.findIndex(c => c.id === id)
      if (index !== -1) {
        containers.value[index] = container
      }

      return container
    } catch (error) {
      console.error('Failed to fetch container:', error)
      ElMessage.error('Failed to load container details')
      throw error
    } finally {
      loadingDetails.value = false
    }
  }

  async function createContainer(data: ContainerFormData) {
    try {
      const container = await containerAPI.createContainer(data)
      containers.value.unshift(container)
      totalContainers.value++

      ElNotification({
        title: 'Container Created',
        message: `Container "${container.name}" has been created successfully`,
        type: 'success'
      })

      return container
    } catch (error) {
      console.error('Failed to create container:', error)
      throw error
    }
  }

  async function updateContainer(id: string, data: Partial<ContainerFormData>) {
    try {
      const container = await containerAPI.updateContainer(id, data)

      // Update in list
      const index = containers.value.findIndex(c => c.id === id)
      if (index !== -1) {
        containers.value[index] = container
      }

      // Update current container if it's the same
      if (currentContainer.value?.id === id) {
        currentContainer.value = container
      }

      ElNotification({
        title: 'Container Updated',
        message: `Container "${container.name}" has been updated successfully`,
        type: 'success'
      })

      return container
    } catch (error) {
      console.error('Failed to update container:', error)
      throw error
    }
  }

  async function deleteContainer(id: string, force = false) {
    try {
      await containerAPI.deleteContainer(id, force)

      // Remove from list
      const index = containers.value.findIndex(c => c.id === id)
      if (index !== -1) {
        const container = containers.value[index]
        containers.value.splice(index, 1)
        totalContainers.value--

        ElNotification({
          title: 'Container Deleted',
          message: `Container "${container.name}" has been deleted`,
          type: 'success'
        })
      }

      // Clear current container if it's the same
      if (currentContainer.value?.id === id) {
        currentContainer.value = null
      }

      // Remove from selection
      selectedContainers.value.delete(id)

    } catch (error) {
      console.error('Failed to delete container:', error)
      throw error
    }
  }

  async function performOperation(
    id: string,
    operation: 'start' | 'stop' | 'restart' | 'pause' | 'unpause',
    options?: any
  ) {
    const container = containers.value.find(c => c.id === id)
    if (!container) return

    setOperationLoading(id, true)
    try {
      switch (operation) {
        case 'start':
          await containerAPI.startContainer(id)
          break
        case 'stop':
          await containerAPI.stopContainer(id, options?.timeout)
          break
        case 'restart':
          await containerAPI.restartContainer(id, options?.timeout)
          break
        case 'pause':
          await containerAPI.pauseContainer(id)
          break
        case 'unpause':
          await containerAPI.unpauseContainer(id)
          break
      }

      // Refresh container status
      await fetchContainer(id)

      ElMessage.success(`Container ${operation} operation completed`)

    } catch (error) {
      console.error(`Failed to ${operation} container:`, error)
      ElMessage.error(`Failed to ${operation} container`)
      throw error
    } finally {
      setOperationLoading(id, false)
    }
  }

  async function performBulkOperation(operation: BulkOperation) {
    loadingBulk.value = true
    try {
      const result = await containerAPI.bulkOperation(operation)

      // Refresh containers to get updated states
      await fetchContainers()

      const { successful, failed, total } = result.summary

      if (failed === 0) {
        ElNotification({
          title: 'Bulk Operation Completed',
          message: `Successfully ${operation.action}ed ${successful} containers`,
          type: 'success'
        })
      } else {
        ElNotification({
          title: 'Bulk Operation Completed with Errors',
          message: `${successful}/${total} containers processed successfully`,
          type: 'warning'
        })
      }

      // Clear selection
      selectedContainers.value.clear()

      return result
    } catch (error) {
      console.error('Bulk operation failed:', error)
      ElMessage.error('Bulk operation failed')
      throw error
    } finally {
      loadingBulk.value = false
    }
  }

  async function checkUpdates(containerId?: string) {
    try {
      const updates = await containerAPI.checkUpdates(containerId)
      availableUpdates.value = updates

      if (updates.length > 0) {
        ElNotification({
          title: 'Updates Available',
          message: `${updates.length} container(s) have available updates`,
          type: 'info'
        })
      }

      return updates
    } catch (error) {
      console.error('Failed to check updates:', error)
      ElMessage.error('Failed to check for updates')
      throw error
    }
  }

  async function updateContainerImage(id: string, options?: any) {
    const container = containers.value.find(c => c.id === id)
    if (!container) return

    setOperationLoading(id, true)
    try {
      await containerAPI.updateContainer(id, options)
      await fetchContainer(id)

      // Remove from available updates
      const updateIndex = availableUpdates.value.findIndex(u => u.container === id)
      if (updateIndex !== -1) {
        availableUpdates.value.splice(updateIndex, 1)
      }

      ElNotification({
        title: 'Container Updated',
        message: `Container "${container.name}" has been updated to the latest version`,
        type: 'success'
      })

    } catch (error) {
      console.error('Failed to update container:', error)
      ElMessage.error('Failed to update container')
      throw error
    } finally {
      setOperationLoading(id, false)
    }
  }

  async function fetchLogs(id: string, options?: any) {
    try {
      const containerLogs = await containerAPI.getLogs(id, options)
      logs.value.set(id, containerLogs)
      return containerLogs
    } catch (error) {
      console.error('Failed to fetch logs:', error)
      ElMessage.error('Failed to load container logs')
      throw error
    }
  }

  async function fetchStats(id: string) {
    try {
      const containerStats = await containerAPI.getStats(id)
      stats.value.set(id, containerStats)
      return containerStats
    } catch (error) {
      console.error('Failed to fetch stats:', error)
      throw error
    }
  }

  async function fetchHistoricalStats(id: string, period = '1h', interval = '1m') {
    try {
      const historical = await containerAPI.getHistoricalStats(id, period, interval)
      historicalStats.value.set(id, historical)
      return historical
    } catch (error) {
      console.error('Failed to fetch historical stats:', error)
      throw error
    }
  }

  async function fetchTemplates() {
    try {
      const containerTemplates = await containerAPI.getTemplates()
      templates.value = containerTemplates
      return containerTemplates
    } catch (error) {
      console.error('Failed to fetch templates:', error)
      ElMessage.error('Failed to load container templates')
      throw error
    }
  }

  async function createFromTemplate(templateId: string, overrides?: Partial<ContainerFormData>) {
    try {
      const container = await containerAPI.createFromTemplate(templateId, overrides)
      containers.value.unshift(container)
      totalContainers.value++

      ElNotification({
        title: 'Container Created from Template',
        message: `Container "${container.name}" has been created successfully`,
        type: 'success'
      })

      return container
    } catch (error) {
      console.error('Failed to create container from template:', error)
      throw error
    }
  }

  // Selection management
  function toggleSelection(id: string) {
    if (selectedContainers.value.has(id)) {
      selectedContainers.value.delete(id)
    } else {
      selectedContainers.value.add(id)
    }
  }

  function selectAll() {
    if (isAllSelected.value) {
      selectedContainers.value.clear()
    } else {
      containers.value.forEach(c => selectedContainers.value.add(c.id))
    }
  }

  function clearSelection() {
    selectedContainers.value.clear()
  }

  // Utility functions
  function setOperationLoading(id: string, loading: boolean) {
    if (loading) {
      loadingOperation.value[id] = true
    } else {
      delete loadingOperation.value[id]
    }
  }

  function isOperationLoading(id: string): boolean {
    return loadingOperation.value[id] === true
  }

  function setFilters(newFilters: ContainerFilter) {
    filters.value = { ...newFilters }
    currentPage.value = 1
    fetchContainers()
  }

  function setSorting(field: ContainerSort['field'], direction?: ContainerSort['direction']) {
    if (sortConfig.value.field === field && !direction) {
      sortConfig.value.direction = sortConfig.value.direction === 'asc' ? 'desc' : 'asc'
    } else {
      sortConfig.value = { field, direction: direction || 'asc' }
    }
    currentPage.value = 1
    fetchContainers()
  }

  function clearFilters() {
    filters.value = {}
    currentPage.value = 1
    fetchContainers()
  }

  function refreshData() {
    fetchContainers()
    if (currentContainer.value) {
      fetchContainer(currentContainer.value.id)
    }
  }

  // WebSocket integration
  function handleWebSocketMessage(message: any) {
    const { type, data } = message

    switch (type) {
      case 'container_status':
        handleStatusUpdate(data)
        break
      case 'container_stats':
        handleStatsUpdate(data)
        break
      case 'container_logs':
        handleLogsUpdate(data)
        break
      case 'container_event':
        handleEventUpdate(data)
        break
    }
  }

  function handleStatusUpdate(data: any) {
    const container = containers.value.find(c => c.id === data.container)
    if (container) {
      container.status = data.status
      container.state = data.state
    }

    if (currentContainer.value?.id === data.container) {
      currentContainer.value.status = data.status
      currentContainer.value.state = data.state
    }

    lastUpdate.value = new Date()
  }

  function handleStatsUpdate(data: any) {
    stats.value.set(data.container, data.stats)

    // Update container resource usage in the list
    const container = containers.value.find(c => c.id === data.container)
    if (container) {
      container.resourceUsage = data.stats
    }

    if (currentContainer.value?.id === data.container) {
      currentContainer.value.resourceUsage = data.stats
    }
  }

  function handleLogsUpdate(data: any) {
    const existingLogs = logs.value.get(data.container) || []
    logs.value.set(data.container, [...existingLogs, ...data.logs])
  }

  function handleEventUpdate(data: any) {
    // Handle container events (creation, deletion, etc.)
    if (data.action === 'create' || data.action === 'start' || data.action === 'stop') {
      // Refresh the specific container or the entire list
      if (data.container) {
        fetchContainer(data.container).catch(() => {
          // Container might be deleted, refresh list
          fetchContainers()
        })
      } else {
        fetchContainers()
      }
    }
  }

  function setWebSocketConnected(connected: boolean) {
    wsConnected.value = connected
  }

  // Reset store
  function $reset() {
    containers.value = []
    currentContainer.value = null
    templates.value = []
    availableUpdates.value = []
    logs.value.clear()
    stats.value.clear()
    historicalStats.value.clear()
    selectedContainers.value.clear()
    loading.value = false
    loadingDetails.value = false
    loadingOperation.value = {}
    loadingBulk.value = false
    currentPage.value = 1
    pageSize.value = 20
    totalContainers.value = 0
    filters.value = {}
    sortConfig.value = { field: 'name', direction: 'asc' }
    viewMode.value = 'grid'
    showFilters.value = false
    autoRefresh.value = true
    refreshInterval.value = 30000
    wsConnected.value = false
    lastUpdate.value = new Date()
  }

  return {
    // State
    containers,
    currentContainer,
    templates,
    availableUpdates,
    logs,
    stats,
    historicalStats,
    loading,
    loadingDetails,
    loadingOperation,
    loadingBulk,
    currentPage,
    pageSize,
    totalContainers,
    filters,
    sortConfig,
    selectedContainers,
    viewMode,
    showFilters,
    autoRefresh,
    refreshInterval,
    wsConnected,
    lastUpdate,

    // Computed
    runningContainers,
    stoppedContainers,
    unhealthyContainers,
    containersWithUpdates,
    totalPages,
    isAllSelected,
    hasSelection,
    filteredContainers,

    // Actions
    fetchContainers,
    fetchContainer,
    createContainer,
    updateContainer,
    deleteContainer,
    performOperation,
    performBulkOperation,
    checkUpdates,
    updateContainerImage,
    fetchLogs,
    fetchStats,
    fetchHistoricalStats,
    fetchTemplates,
    createFromTemplate,
    toggleSelection,
    selectAll,
    clearSelection,
    setOperationLoading,
    isOperationLoading,
    setFilters,
    setSorting,
    clearFilters,
    refreshData,
    handleWebSocketMessage,
    setWebSocketConnected,
    $reset
  }
})