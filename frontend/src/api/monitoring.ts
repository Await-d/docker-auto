/**
 * Monitoring and metrics API service
 */
import { get, post } from "@/utils/request";
import type { ResourceMetrics, ContainerStats } from "@/types/container";

export interface MonitoringSystemMetrics {
  cpu: {
    usage: number;
    cores: number;
    loadAvg: [number, number, number];
  };
  memory: {
    total: number;
    used: number;
    free: number;
    available: number;
    percentage: number;
  };
  disk: {
    total: number;
    used: number;
    free: number;
    percentage: number;
    devices: Array<{
      device: string;
      mountpoint: string;
      total: number;
      used: number;
      free: number;
      percentage: number;
    }>;
  };
  network: {
    interfaces: Array<{
      name: string;
      rxBytes: number;
      txBytes: number;
      rxPackets: number;
      txPackets: number;
      rxErrors: number;
      txErrors: number;
    }>;
  };
  docker: {
    containers: {
      total: number;
      running: number;
      stopped: number;
      paused: number;
    };
    images: {
      total: number;
      size: number;
    };
    volumes: {
      total: number;
      size: number;
    };
    networks: number;
  };
  timestamp: Date;
}

export class MonitoringAPI {
  private readonly baseUrl = "/api/monitoring";

  /**
   * Get real-time system metrics
   */
  async getSystemMetrics(): Promise<MonitoringSystemMetrics> {
    return get<MonitoringSystemMetrics>(`${this.baseUrl}/system`);
  }

  /**
   * Get container metrics with query options
   */
  async getContainerMetrics(
    containerId: string,
    timeRange?: { start: Date; end: Date },
    interval?: string
  ): Promise<ResourceMetrics[]> {
    const params = new URLSearchParams();

    if (timeRange) {
      params.append("start", timeRange.start.toISOString());
      params.append("end", timeRange.end.toISOString());
    }

    if (interval) {
      params.append("interval", interval);
    }

    return get<ResourceMetrics[]>(
      `${this.baseUrl}/containers/${containerId}/metrics?${params.toString()}`
    );
  }

  /**
   * Get historical statistics for multiple containers
   */
  async getMultiContainerStats(
    containerIds: string[],
    period: string = "1h",
    interval: string = "5m"
  ): Promise<Record<string, ContainerStats[]>> {
    return post<Record<string, ContainerStats[]>>(
      `${this.baseUrl}/containers/stats/bulk`,
      {
        containerIds,
        period,
        interval
      }
    );
  }

  /**
   * Get aggregated metrics across containers
   */
  async getAggregatedMetrics(
    containerIds: string[],
    timeRange: { start: Date; end: Date }
  ): Promise<{
    cpu: { avg: number; max: number; min: number };
    memory: { avg: number; max: number; min: number };
    network: { rxTotal: number; txTotal: number };
    disk: { readTotal: number; writeTotal: number };
  }> {
    return post<any>(`${this.baseUrl}/containers/aggregate`, {
      containerIds,
      timeRange
    });
  }

  /**
   * Export metrics data
   */
  async exportMetrics(
    containerId: string,
    timeRange: { start: Date; end: Date },
    format: 'csv' | 'json' = 'json'
  ): Promise<Blob> {
    const params = new URLSearchParams({
      start: timeRange.start.toISOString(),
      end: timeRange.end.toISOString(),
      format
    });

    const response = await fetch(`${this.baseUrl}/containers/${containerId}/export?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error(`Export failed: ${response.statusText}`);
    }

    return response.blob();
  }

  /**
   * Get resource usage summary
   */
  async getResourceSummary(
    timeRange: { start: Date; end: Date }
  ): Promise<{
    containers: Array<{
      id: string;
      name: string;
      avgCpu: number;
      maxCpu: number;
      avgMemory: number;
      maxMemory: number;
      networkTraffic: number;
      diskIO: number;
    }>;
    system: {
      avgCpu: number;
      avgMemory: number;
      totalNetworkTraffic: number;
      totalDiskIO: number;
    };
  }> {
    return post<any>(`${this.baseUrl}/analytics/summary`, { timeRange });
  }

  /**
   * Get top resource consumers
   */
  async getTopConsumers(
    metric: 'cpu' | 'memory' | 'network' | 'disk',
    limit: number = 10,
    timeRange?: { start: Date; end: Date }
  ): Promise<Array<{
    containerId: string;
    containerName: string;
    value: number;
    percentage: number;
  }>> {
    const params = new URLSearchParams({
      metric,
      limit: limit.toString()
    });

    if (timeRange) {
      params.append("start", timeRange.start.toISOString());
      params.append("end", timeRange.end.toISOString());
    }

    return get<any>(`${this.baseUrl}/analytics/top-consumers?${params.toString()}`);
  }

  /**
   * Get health services
   */
  async getHealthServices(): Promise<any[]> {
    return get<any[]>(`${this.baseUrl}/health/services`);
  }

  /**
   * Get health alerts
   */
  async getHealthAlerts(): Promise<any[]> {
    return get<any[]>(`${this.baseUrl}/health/alerts`);
  }

  /**
   * Get health events
   */
  async getHealthEvents(): Promise<any[]> {
    return get<any[]>(`${this.baseUrl}/health/events`);
  }

  /**
   * Run health check
   */
  async runHealthCheck(serviceId: string): Promise<any> {
    return post<any>(`${this.baseUrl}/health/check/${serviceId}`, {});
  }

  /**
   * Acknowledge alert
   */
  async acknowledgeAlert(alertId: string): Promise<any> {
    return post<any>(`${this.baseUrl}/health/alerts/${alertId}/acknowledge`, {});
  }

  /**
   * Resolve alert
   */
  async resolveAlert(alertId: string): Promise<any> {
    return post<any>(`${this.baseUrl}/health/alerts/${alertId}/resolve`, {});
  }

  /**
   * Get active alerts
   */
  async getActiveAlerts(): Promise<any[]> {
    return get<any[]>(`${this.baseUrl}/health/alerts/active`);
  }
}

// Export singleton instance
export const monitoringAPI = new MonitoringAPI();