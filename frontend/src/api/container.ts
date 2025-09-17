/**
 * Container API service
 */
import { get, post, put, del } from "@/utils/request";
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
  ResourceMetrics,
} from "@/types/container";

export class ContainerAPI {
  private readonly baseUrl = "/api/containers";

  /**
   * Get all containers with optional filtering and sorting
   */
  async getContainers(
    page = 1,
    limit = 20,
    filters?: ContainerFilter,
    sort?: ContainerSort,
  ): Promise<ContainerListResponse> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
    });

    if (filters) {
      if (filters.status?.length) {
        params.append("status", filters.status.join(","));
      }
      if (filters.image) {
        params.append("image", filters.image);
      }
      if (filters.registry) {
        params.append("registry", filters.registry);
      }
      if (filters.search) {
        params.append("search", filters.search);
      }
      if (filters.labels) {
        Object.entries(filters.labels).forEach(([key, value]) => {
          params.append(`label.${key}`, value);
        });
      }
    }

    if (sort) {
      params.append("sort", `${sort.field}:${sort.direction}`);
    }

    return get<ContainerListResponse>(`${this.baseUrl}?${params.toString()}`);
  }

  /**
   * Get container by ID
   */
  async getContainer(id: string): Promise<Container> {
    return get<Container>(`${this.baseUrl}/${id}`);
  }

  /**
   * Create new container
   */
  async createContainer(data: ContainerFormData): Promise<Container> {
    return post<Container>(this.baseUrl, data);
  }

  /**
   * Update container configuration
   */
  async updateContainer(
    id: string,
    data: Partial<ContainerFormData>,
  ): Promise<Container> {
    return put<Container>(`${this.baseUrl}/${id}`, data);
  }

  /**
   * Delete container
   */
  async deleteContainer(id: string, force = false): Promise<void> {
    return del<void>(`${this.baseUrl}/${id}?force=${force}`);
  }

  /**
   * Start container
   */
  async startContainer(id: string): Promise<void> {
    return post<void>(`${this.baseUrl}/${id}/start`);
  }

  /**
   * Stop container
   */
  async stopContainer(id: string, timeout = 10): Promise<void> {
    return post<void>(`${this.baseUrl}/${id}/stop`, { timeout });
  }

  /**
   * Restart container
   */
  async restartContainer(id: string, timeout = 10): Promise<void> {
    return post<void>(`${this.baseUrl}/${id}/restart`, { timeout });
  }

  /**
   * Pause container
   */
  async pauseContainer(id: string): Promise<void> {
    return post<void>(`${this.baseUrl}/${id}/pause`);
  }

  /**
   * Unpause container
   */
  async unpauseContainer(id: string): Promise<void> {
    return post<void>(`${this.baseUrl}/${id}/unpause`);
  }

  /**
   * Get container logs
   */
  async getLogs(
    id: string,
    options: {
      follow?: boolean;
      tail?: number;
      since?: Date;
      until?: Date;
      timestamps?: boolean;
    } = {},
  ): Promise<ContainerLog[]> {
    const params = new URLSearchParams();

    if (options.follow) params.append("follow", "true");
    if (options.tail) params.append("tail", options.tail.toString());
    if (options.since) params.append("since", options.since.toISOString());
    if (options.until) params.append("until", options.until.toISOString());
    if (options.timestamps) params.append("timestamps", "true");

    return get<ContainerLog[]>(
      `${this.baseUrl}/${id}/logs?${params.toString()}`,
    );
  }

  /**
   * Get container statistics
   */
  async getStats(id: string): Promise<ResourceMetrics> {
    return get<ResourceMetrics>(`${this.baseUrl}/${id}/stats`);
  }

  /**
   * Get historical stats
   */
  async getHistoricalStats(
    id: string,
    period: string = "1h",
    interval: string = "1m",
  ): Promise<ContainerStats[]> {
    return get<ContainerStats[]>(
      `${this.baseUrl}/${id}/stats/history?period=${period}&interval=${interval}`,
    );
  }

  /**
   * Execute command in container
   */
  async execCommand(
    id: string,
    command: string[],
    options: {
      workingDir?: string;
      user?: string;
      env?: Record<string, string>;
      tty?: boolean;
      interactive?: boolean;
    } = {},
  ): Promise<{ output: string; exitCode: number }> {
    return post<{ output: string; exitCode: number }>(
      `${this.baseUrl}/${id}/exec`,
      {
        command,
        ...options,
      },
    );
  }

  /**
   * Get container inspect information
   */
  async inspectContainer(id: string): Promise<any> {
    return get<any>(`${this.baseUrl}/${id}/inspect`);
  }

  /**
   * Update container image
   */
  async updateContainerImage(
    id: string,
    options: {
      pullPolicy?: "always" | "missing" | "never";
      recreate?: boolean;
      preserveVolumes?: boolean;
    } = {},
  ): Promise<void> {
    return post<void>(`${this.baseUrl}/${id}/update`, options);
  }

  /**
   * Check for available updates
   */
  async checkUpdates(id?: string): Promise<UpdateAvailable[]> {
    const url = id
      ? `${this.baseUrl}/${id}/updates`
      : `${this.baseUrl}/updates`;
    return get<UpdateAvailable[]>(url);
  }

  /**
   * Perform bulk operations
   */
  async bulkOperation(operation: BulkOperation): Promise<BulkOperationResult> {
    return post<BulkOperationResult>(`${this.baseUrl}/bulk`, operation);
  }

  /**
   * Export container configuration
   */
  async exportConfig(
    id: string,
    format: "json" | "yaml" | "compose" = "json",
  ): Promise<string> {
    return get<string>(`${this.baseUrl}/${id}/export?format=${format}`);
  }

  /**
   * Import container from configuration
   */
  async importConfig(
    config: string,
    format: "json" | "yaml" | "compose" = "json",
  ): Promise<Container> {
    return post<Container>(`${this.baseUrl}/import`, { config, format });
  }

  /**
   * Get container templates
   */
  async getTemplates(): Promise<ContainerTemplate[]> {
    return get<ContainerTemplate[]>(`${this.baseUrl}/templates`);
  }

  /**
   * Create template from container
   */
  async createTemplate(
    id: string,
    template: Omit<ContainerTemplate, "id" | "createdAt" | "createdBy">,
  ): Promise<ContainerTemplate> {
    return post<ContainerTemplate>(`${this.baseUrl}/${id}/template`, template);
  }

  /**
   * Create container from template
   */
  async createFromTemplate(
    templateId: string,
    overrides?: Partial<ContainerFormData>,
  ): Promise<Container> {
    return post<Container>(
      `${this.baseUrl}/templates/${templateId}/create`,
      overrides,
    );
  }

  /**
   * Get container images
   */
  async getImages(filters?: {
    repository?: string;
    tag?: string;
    registry?: string;
  }): Promise<ContainerImage[]> {
    const params = new URLSearchParams();
    if (filters?.repository) params.append("repository", filters.repository);
    if (filters?.tag) params.append("tag", filters.tag);
    if (filters?.registry) params.append("registry", filters.registry);

    return get<ContainerImage[]>(`/api/images?${params.toString()}`);
  }

  /**
   * Pull image
   */
  async pullImage(
    image: string,
    tag: string = "latest",
    registry?: string,
  ): Promise<void> {
    return post<void>("/api/images/pull", { image, tag, registry });
  }

  /**
   * Check for image updates
   */
  async checkImageUpdates(images?: string[]): Promise<ImageUpdateCheck[]> {
    return post<ImageUpdateCheck[]>("/api/images/check-updates", { images });
  }

  /**
   * Get container health check history
   */
  async getHealthHistory(id: string, limit = 50): Promise<any[]> {
    return get<any[]>(`${this.baseUrl}/${id}/health?limit=${limit}`);
  }

  /**
   * Set custom health check
   */
  async setHealthCheck(
    id: string,
    healthCheck: {
      command: string[];
      interval: number;
      timeout: number;
      retries: number;
      startPeriod: number;
    },
  ): Promise<void> {
    return put<void>(`${this.baseUrl}/${id}/health`, healthCheck);
  }

  /**
   * Get container events
   */
  async getEvents(id?: string, since?: Date, until?: Date): Promise<any[]> {
    const params = new URLSearchParams();
    if (since) params.append("since", since.toISOString());
    if (until) params.append("until", until.toISOString());

    const url = id
      ? `${this.baseUrl}/${id}/events?${params.toString()}`
      : `${this.baseUrl}/events?${params.toString()}`;

    return get<any[]>(url);
  }

  /**
   * Create backup of container
   */
  async createBackup(
    id: string,
    options: {
      includeVolumes?: boolean;
      compression?: "none" | "gzip" | "bzip2";
      name?: string;
    } = {},
  ): Promise<{ backupId: string; size: number }> {
    return post<{ backupId: string; size: number }>(
      `${this.baseUrl}/${id}/backup`,
      options,
    );
  }

  /**
   * Restore container from backup
   */
  async restoreFromBackup(
    backupId: string,
    options: {
      name?: string;
      restoreVolumes?: boolean;
    } = {},
  ): Promise<Container> {
    return post<Container>(`${this.baseUrl}/restore/${backupId}`, options);
  }
}

// Export singleton instance
export const containerAPI = new ContainerAPI();
