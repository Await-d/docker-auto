/**
 * Updates API service
 */
import { http } from './http'
import type {
  ContainerUpdate,
  UpdateHistoryItem,
  RunningUpdate,
  UpdatePolicy,
  ScheduledUpdate,
  UpdateTemplate,
  BulkUpdateOperation,
  UpdateFilter,
  UpdateSort,
  UpdateAnalytics,
  RollbackOperation,
  VersionComparison,
  SecurityPatch,
  UpdateStrategy
} from '@/types/updates'

export interface CheckUpdatesResponse {
  updates: ContainerUpdate[]
  total: number
  lastChecked: string
}

export interface UpdateHistoryResponse {
  items: UpdateHistoryItem[]
  total: number
  page: number
  pageSize: number
}

export interface StartUpdateRequest {
  strategy?: UpdateStrategy
  rollbackOnFailure?: boolean
  notifyOnCompletion?: boolean
  runTests?: boolean
  maintenanceWindow?: {
    start?: string
    end?: string
  }
}

export interface StartUpdateResponse {
  id: string
  status: 'queued' | 'running'
  estimatedDuration: number
  steps: Array<{
    name: string
    description: string
    estimatedDuration: number
  }>
}

export interface ScheduleUpdateRequest {
  scheduledAt: string
  recurring?: boolean
  recurringPattern?: string // cron pattern
  notifyBefore?: number // milliseconds
  strategy?: UpdateStrategy
  rollbackOnFailure?: boolean
}

export const updatesAPI = {
  /**
   * Check for available updates
   */
  async checkUpdates(containerId?: string, force = false): Promise<CheckUpdatesResponse> {
    const params = new URLSearchParams()
    if (containerId) params.set('containerId', containerId)
    if (force) params.set('force', 'true')

    return http.get(`/api/updates/check?${params}`)
  },

  /**
   * Get update history with pagination and filtering
   */
  async getUpdateHistory(
    page = 1,
    pageSize = 50,
    filters?: UpdateFilter,
    sort?: UpdateSort
  ): Promise<UpdateHistoryResponse> {
    const params = new URLSearchParams({
      page: page.toString(),
      pageSize: pageSize.toString()
    })

    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, v.toString()))
          } else {
            params.set(key, value.toString())
          }
        }
      })
    }

    if (sort) {
      params.set('sortBy', sort.field)
      params.set('sortOrder', sort.direction)
    }

    return http.get(`/api/updates/history?${params}`)
  },

  /**
   * Get running updates
   */
  async getRunningUpdates(): Promise<RunningUpdate[]> {
    return http.get('/api/updates/running')
  },

  /**
   * Get scheduled updates
   */
  async getScheduledUpdates(): Promise<ScheduledUpdate[]> {
    return http.get('/api/updates/scheduled')
  },

  /**
   * Start a single container update
   */
  async startUpdate(updateId: string, options?: StartUpdateRequest): Promise<StartUpdateResponse> {
    return http.post(`/api/updates/${updateId}/start`, options)
  },

  /**
   * Start bulk update operation
   */
  async startBulkUpdate(operation: BulkUpdateOperation): Promise<{ operationId: string; queuedUpdates: number }> {
    return http.post('/api/updates/bulk', operation)
  },

  /**
   * Schedule an update
   */
  async scheduleUpdate(updateId: string, request: ScheduleUpdateRequest): Promise<ScheduledUpdate> {
    return http.post(`/api/updates/${updateId}/schedule`, request)
  },

  /**
   * Cancel a running update
   */
  async cancelUpdate(runningUpdateId: string): Promise<void> {
    return http.post(`/api/updates/running/${runningUpdateId}/cancel`)
  },

  /**
   * Cancel a scheduled update
   */
  async cancelScheduledUpdate(scheduledUpdateId: string): Promise<void> {
    return http.delete(`/api/updates/scheduled/${scheduledUpdateId}`)
  },

  /**
   * Rollback an update
   */
  async rollbackUpdate(operation: RollbackOperation): Promise<{ id: string; status: string }> {
    return http.post('/api/updates/rollback', operation)
  },

  /**
   * Ignore an update
   */
  async ignoreUpdate(updateId: string, reason?: string): Promise<void> {
    return http.post(`/api/updates/${updateId}/ignore`, { reason })
  },

  /**
   * Unignore an update
   */
  async unignoreUpdate(updateId: string): Promise<void> {
    return http.post(`/api/updates/${updateId}/unignore`)
  },

  /**
   * Get update policies
   */
  async getUpdatePolicies(): Promise<UpdatePolicy[]> {
    return http.get('/api/updates/policies')
  },

  /**
   * Create a new update policy
   */
  async createPolicy(policy: Partial<UpdatePolicy>): Promise<UpdatePolicy> {
    return http.post('/api/updates/policies', policy)
  },

  /**
   * Update an existing policy
   */
  async updatePolicy(policyId: string, policy: Partial<UpdatePolicy>): Promise<UpdatePolicy> {
    return http.put(`/api/updates/policies/${policyId}`, policy)
  },

  /**
   * Delete an update policy
   */
  async deletePolicy(policyId: string): Promise<void> {
    return http.delete(`/api/updates/policies/${policyId}`)
  },

  /**
   * Apply a policy to containers
   */
  async applyPolicy(policyId: string, containerIds: string[]): Promise<{ applied: number; skipped: number }> {
    return http.post(`/api/updates/policies/${policyId}/apply`, { containerIds })
  },

  /**
   * Get update templates
   */
  async getUpdateTemplates(): Promise<UpdateTemplate[]> {
    return http.get('/api/updates/templates')
  },

  /**
   * Create update template
   */
  async createTemplate(template: Partial<UpdateTemplate>): Promise<UpdateTemplate> {
    return http.post('/api/updates/templates', template)
  },

  /**
   * Update template
   */
  async updateTemplate(templateId: string, template: Partial<UpdateTemplate>): Promise<UpdateTemplate> {
    return http.put(`/api/updates/templates/${templateId}`, template)
  },

  /**
   * Delete template
   */
  async deleteTemplate(templateId: string): Promise<void> {
    return http.delete(`/api/updates/templates/${templateId}`)
  },

  /**
   * Compare versions
   */
  async compareVersions(
    containerId: string,
    fromVersion: string,
    toVersion: string
  ): Promise<VersionComparison> {
    return http.post(`/api/updates/compare`, {
      containerId,
      fromVersion,
      toVersion
    })
  },

  /**
   * Get security patches for a version
   */
  async getSecurityPatches(
    imageName: string,
    fromVersion: string,
    toVersion: string
  ): Promise<SecurityPatch[]> {
    const params = new URLSearchParams({
      imageName,
      fromVersion,
      toVersion
    })

    return http.get(`/api/updates/security-patches?${params}`)
  },

  /**
   * Test update before applying
   */
  async testUpdate(updateId: string): Promise<{
    passed: boolean
    results: Array<{
      test: string
      status: 'passed' | 'failed' | 'warning'
      message: string
      details?: any
    }>
  }> {
    return http.post(`/api/updates/${updateId}/test`)
  },

  /**
   * Get update analytics
   */
  async getAnalytics(period = '30d'): Promise<UpdateAnalytics> {
    return http.get(`/api/updates/analytics?period=${period}`)
  },

  /**
   * Export update history
   */
  async exportHistory(
    format: 'csv' | 'json' | 'pdf',
    filters?: UpdateFilter,
    dateRange?: { start: string; end: string }
  ): Promise<Blob> {
    const params = new URLSearchParams({ format })

    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, v.toString()))
          } else {
            params.set(key, value.toString())
          }
        }
      })
    }

    if (dateRange) {
      params.set('startDate', dateRange.start)
      params.set('endDate', dateRange.end)
    }

    const response = await fetch(`/api/updates/export?${params}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
      },
    })

    if (!response.ok) {
      throw new Error(`Export failed: ${response.statusText}`)
    }

    return response.blob()
  },

  /**
   * Get update settings
   */
  async getSettings(): Promise<any> {
    return http.get('/api/updates/settings')
  },

  /**
   * Update settings
   */
  async updateSettings(settings: any): Promise<any> {
    return http.put('/api/updates/settings', settings)
  },

  /**
   * Get update notifications
   */
  async getNotifications(page = 1, pageSize = 20): Promise<{
    notifications: Array<{
      id: string
      type: 'info' | 'warning' | 'error' | 'success'
      title: string
      message: string
      timestamp: string
      read: boolean
      data?: any
    }>
    total: number
  }> {
    return http.get(`/api/updates/notifications?page=${page}&pageSize=${pageSize}`)
  },

  /**
   * Mark notification as read
   */
  async markNotificationRead(notificationId: string): Promise<void> {
    return http.post(`/api/updates/notifications/${notificationId}/read`)
  },

  /**
   * Clear all notifications
   */
  async clearNotifications(): Promise<void> {
    return http.delete('/api/updates/notifications')
  },

  /**
   * Get update logs
   */
  async getUpdateLogs(
    updateId: string,
    options?: {
      lines?: number
      follow?: boolean
      since?: string
    }
  ): Promise<{
    logs: Array<{
      timestamp: string
      level: 'debug' | 'info' | 'warn' | 'error'
      message: string
      source?: string
    }>
  }> {
    const params = new URLSearchParams()
    if (options?.lines) params.set('lines', options.lines.toString())
    if (options?.follow) params.set('follow', 'true')
    if (options?.since) params.set('since', options.since)

    return http.get(`/api/updates/${updateId}/logs?${params}`)
  },

  /**
   * Get health check results after update
   */
  async getHealthCheck(containerId: string): Promise<{
    status: 'healthy' | 'unhealthy' | 'starting'
    checks: Array<{
      name: string
      status: 'passed' | 'failed' | 'warning'
      message: string
      duration: number
      timestamp: string
    }>
  }> {
    return http.get(`/api/containers/${containerId}/health`)
  },

  /**
   * Validate update configuration
   */
  async validateUpdateConfig(config: {
    updateIds: string[]
    strategy: UpdateStrategy
    options: any
  }): Promise<{
    valid: boolean
    errors: string[]
    warnings: string[]
    recommendations: string[]
  }> {
    return http.post('/api/updates/validate', config)
  },

  /**
   * Get update dependencies
   */
  async getUpdateDependencies(updateId: string): Promise<{
    dependencies: Array<{
      containerId: string
      containerName: string
      relationship: 'depends_on' | 'linked' | 'network' | 'volume'
      required: boolean
    }>
    conflicts: Array<{
      containerId: string
      containerName: string
      reason: string
    }>
  }> {
    return http.get(`/api/updates/${updateId}/dependencies`)
  },

  /**
   * Simulate update
   */
  async simulateUpdate(updateId: string): Promise<{
    simulation: {
      steps: Array<{
        name: string
        description: string
        estimatedDuration: number
        riskLevel: 'low' | 'medium' | 'high'
        reversible: boolean
      }>
      totalEstimatedTime: number
      overallRiskLevel: 'low' | 'medium' | 'high'
      requiredDowntime: number
      potentialIssues: Array<{
        issue: string
        probability: number
        impact: 'low' | 'medium' | 'high'
        mitigation: string
      }>
    }
  }> {
    return http.post(`/api/updates/${updateId}/simulate`)
  }
}