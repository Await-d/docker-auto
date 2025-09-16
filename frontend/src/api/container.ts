/**
 * Container API service
 */
import { http } from '@/utils/request'
import type { ApiResponse } from '@/utils/request'
import type {
  Container,
  ContainerFormData,
  ContainerFilter,
  ContainerSort,
  ContainerListResponse,
  ContainerLog,
  ContainerStats,
  BulkOperation,
  BulkOperationResult,
  UpdateAvailable,
  ContainerTemplate,
  ContainerImage,
  ImageUpdateCheck,
  ResourceMetrics
} from '@/types/container'

export class ContainerAPI {
  private readonly baseUrl = '/api/containers'

  /**
   * Get all containers with optional filtering and sorting
   */
  async getContainers(
    page = 1,
    limit = 20,
    filters?: ContainerFilter,
    sort?: ContainerSort
  ): Promise<ContainerListResponse> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString()
    })

    if (filters) {
      if (filters.status?.length) {
        params.append('status', filters.status.join(','))
      }
      if (filters.image) {
        params.append('image', filters.image)
      }
      if (filters.registry) {
        params.append('registry', filters.registry)
      }
      if (filters.search) {
        params.append('search', filters.search)
      }
      if (filters.labels) {
        Object.entries(filters.labels).forEach(([key, value]) => {
          params.append(`label.${key}`, value)
        })
      }
    }

    if (sort) {
      params.append('sort', `${sort.field}:${sort.direction}`)
    }

    const response = await http.get<ContainerListResponse>(
      `${this.baseUrl}?${params.toString()}`
    )

    return response.data!
  }

  /**
   * Get container by ID
   */
  async getContainer(id: string): Promise<Container> {
    const response = await http.get<Container>(`${this.baseUrl}/${id}`)
    return response.data!
  }

  /**
   * Create new container
   */
  async createContainer(data: ContainerFormData): Promise<Container> {
    const response = await http.post<Container>(this.baseUrl, data, {
      showSuccess: true
    })
    return response.data!
  }

  /**
   * Update container configuration
   */
  async updateContainer(id: string, data: Partial<ContainerFormData>): Promise<Container> {
    const response = await http.put<Container>(`${this.baseUrl}/${id}`, data, {
      showSuccess: true
    })
    return response.data!
  }

  /**
   * Delete container
   */
  async deleteContainer(id: string, force = false): Promise<void> {
    await http.delete(`${this.baseUrl}/${id}`, {
      params: { force },
      showSuccess: true
    })
  }

  /**
   * Start container
   */
  async startContainer(id: string): Promise<void> {
    await http.post(`${this.baseUrl}/${id}/start`, null, {
      showSuccess: true
    })
  }

  /**
   * Stop container
   */
  async stopContainer(id: string, timeout = 10): Promise<void> {
    await http.post(`${this.baseUrl}/${id}/stop`, { timeout }, {
      showSuccess: true
    })
  }

  /**
   * Restart container
   */
  async restartContainer(id: string, timeout = 10): Promise<void> {
    await http.post(`${this.baseUrl}/${id}/restart`, { timeout }, {
      showSuccess: true
    })
  }

  /**
   * Pause container
   */
  async pauseContainer(id: string): Promise<void> {
    await http.post(`${this.baseUrl}/${id}/pause`, null, {
      showSuccess: true
    })
  }

  /**
   * Unpause container
   */
  async unpauseContainer(id: string): Promise<void> {
    await http.post(`${this.baseUrl}/${id}/unpause`, null, {
      showSuccess: true
    })
  }

  /**
   * Get container logs
   */
  async getLogs(
    id: string,
    options: {
      follow?: boolean
      tail?: number
      since?: Date
      until?: Date
      timestamps?: boolean
    } = {}
  ): Promise<ContainerLog[]> {
    const params = new URLSearchParams()

    if (options.follow) params.append('follow', 'true')
    if (options.tail) params.append('tail', options.tail.toString())
    if (options.since) params.append('since', options.since.toISOString())
    if (options.until) params.append('until', options.until.toISOString())
    if (options.timestamps) params.append('timestamps', 'true')

    const response = await http.get<ContainerLog[]>(
      `${this.baseUrl}/${id}/logs?${params.toString()}`
    )
    return response.data!
  }

  /**
   * Get container statistics
   */
  async getStats(id: string): Promise<ResourceMetrics> {
    const response = await http.get<ResourceMetrics>(`${this.baseUrl}/${id}/stats`)
    return response.data!
  }

  /**
   * Get historical stats
   */
  async getHistoricalStats(
    id: string,
    period: string = '1h',
    interval: string = '1m'
  ): Promise<ContainerStats[]> {
    const response = await http.get<ContainerStats[]>(
      `${this.baseUrl}/${id}/stats/history`,
      {
        params: { period, interval }
      }
    )
    return response.data!
  }

  /**
   * Execute command in container
   */
  async execCommand(
    id: string,
    command: string[],
    options: {
      workingDir?: string
      user?: string
      env?: Record<string, string>
      tty?: boolean
      interactive?: boolean
    } = {}
  ): Promise<{ output: string; exitCode: number }> {
    const response = await http.post<{ output: string; exitCode: number }>(
      `${this.baseUrl}/${id}/exec`,
      {
        command,
        ...options
      }
    )
    return response.data!
  }

  /**
   * Get container inspect information
   */
  async inspectContainer(id: string): Promise<any> {
    const response = await http.get<any>(`${this.baseUrl}/${id}/inspect`)
    return response.data!
  }

  /**
   * Update container image
   */
  async updateContainer(id: string, options: {
    pullPolicy?: 'always' | 'missing' | 'never'
    recreate?: boolean
    preserveVolumes?: boolean
  } = {}): Promise<void> {
    await http.post(`${this.baseUrl}/${id}/update`, options, {
      showSuccess: true
    })
  }

  /**
   * Check for available updates
   */
  async checkUpdates(id?: string): Promise<UpdateAvailable[]> {
    const url = id ? `${this.baseUrl}/${id}/updates` : `${this.baseUrl}/updates`
    const response = await http.get<UpdateAvailable[]>(url)
    return response.data!
  }

  /**
   * Perform bulk operations
   */
  async bulkOperation(operation: BulkOperation): Promise<BulkOperationResult> {
    const response = await http.post<BulkOperationResult>(
      `${this.baseUrl}/bulk`,
      operation,
      {
        showSuccess: true
      }
    )
    return response.data!
  }

  /**
   * Export container configuration
   */
  async exportConfig(id: string, format: 'json' | 'yaml' | 'compose' = 'json'): Promise<string> {
    const response = await http.get<string>(`${this.baseUrl}/${id}/export`, {
      params: { format }
    })
    return response.data!
  }

  /**
   * Import container from configuration
   */
  async importConfig(
    config: string,
    format: 'json' | 'yaml' | 'compose' = 'json'
  ): Promise<Container> {
    const response = await http.post<Container>(`${this.baseUrl}/import`, {
      config,
      format
    }, {
      showSuccess: true
    })
    return response.data!
  }

  /**
   * Get container templates
   */
  async getTemplates(): Promise<ContainerTemplate[]> {
    const response = await http.get<ContainerTemplate[]>(`${this.baseUrl}/templates`)
    return response.data!
  }

  /**
   * Create template from container
   */
  async createTemplate(
    id: string,
    template: Omit<ContainerTemplate, 'id' | 'createdAt' | 'createdBy'>
  ): Promise<ContainerTemplate> {
    const response = await http.post<ContainerTemplate>(
      `${this.baseUrl}/${id}/template`,
      template,
      {
        showSuccess: true
      }
    )
    return response.data!
  }

  /**
   * Create container from template
   */
  async createFromTemplate(
    templateId: string,
    overrides?: Partial<ContainerFormData>
  ): Promise<Container> {
    const response = await http.post<Container>(
      `${this.baseUrl}/templates/${templateId}/create`,
      overrides,
      {
        showSuccess: true
      }
    )
    return response.data!
  }

  /**
   * Get container images
   */
  async getImages(filters?: {
    repository?: string
    tag?: string
    registry?: string
  }): Promise<ContainerImage[]> {
    const params = new URLSearchParams()
    if (filters?.repository) params.append('repository', filters.repository)
    if (filters?.tag) params.append('tag', filters.tag)
    if (filters?.registry) params.append('registry', filters.registry)

    const response = await http.get<ContainerImage[]>(
      `/api/images?${params.toString()}`
    )
    return response.data!
  }

  /**
   * Pull image
   */
  async pullImage(
    image: string,
    tag: string = 'latest',
    registry?: string
  ): Promise<void> {
    await http.post('/api/images/pull', {
      image,
      tag,
      registry
    }, {
      showSuccess: true
    })
  }

  /**
   * Check for image updates
   */
  async checkImageUpdates(images?: string[]): Promise<ImageUpdateCheck[]> {
    const response = await http.post<ImageUpdateCheck[]>('/api/images/check-updates', {
      images
    })
    return response.data!
  }

  /**
   * Get container health check history
   */
  async getHealthHistory(id: string, limit = 50): Promise<any[]> {
    const response = await http.get<any[]>(`${this.baseUrl}/${id}/health`, {
      params: { limit }
    })
    return response.data!
  }

  /**
   * Set custom health check
   */
  async setHealthCheck(id: string, healthCheck: {
    command: string[]
    interval: number
    timeout: number
    retries: number
    startPeriod: number
  }): Promise<void> {
    await http.put(`${this.baseUrl}/${id}/health`, healthCheck, {
      showSuccess: true
    })
  }

  /**
   * Get container events
   */
  async getEvents(
    id?: string,
    since?: Date,
    until?: Date
  ): Promise<any[]> {
    const params = new URLSearchParams()
    if (since) params.append('since', since.toISOString())
    if (until) params.append('until', until.toISOString())

    const url = id
      ? `${this.baseUrl}/${id}/events?${params.toString()}`
      : `${this.baseUrl}/events?${params.toString()}`

    const response = await http.get<any[]>(url)
    return response.data!
  }

  /**
   * Create backup of container
   */
  async createBackup(id: string, options: {
    includeVolumes?: boolean
    compression?: 'none' | 'gzip' | 'bzip2'
    name?: string
  } = {}): Promise<{ backupId: string; size: number }> {
    const response = await http.post<{ backupId: string; size: number }>(
      `${this.baseUrl}/${id}/backup`,
      options,
      {
        showSuccess: true
      }
    )
    return response.data!
  }

  /**
   * Restore container from backup
   */
  async restoreFromBackup(
    backupId: string,
    options: {
      name?: string
      restoreVolumes?: boolean
    } = {}
  ): Promise<Container> {
    const response = await http.post<Container>(
      `${this.baseUrl}/restore/${backupId}`,
      options,
      {
        showSuccess: true
      }
    )
    return response.data!
  }
}

// Export singleton instance
export const containerAPI = new ContainerAPI()