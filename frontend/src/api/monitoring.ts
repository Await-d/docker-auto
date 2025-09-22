/**
 * Real-time monitoring and metrics API service
 */
import { get } from "@/utils/request";

export interface MonitoringMetrics {
  cpu: number;
  memory: {
    used: number;
    total: number;
  };
  network: {
    in: number;
    out: number;
    total: number;
  };
  disk: {
    read: number;
    write: number;
    total: number;
  };
  timestamp: string;
}

export interface ActivityEvent {
  id: string;
  type: "container" | "update" | "system" | "security" | "network";
  severity: "info" | "warning" | "error" | "success";
  title: string;
  description: string;
  timestamp: string;
  source?: string;
  metadata?: Record<string, any>;
}

export interface SystemAlert {
  id: string;
  type: "cpu" | "memory" | "disk" | "network" | "security" | "other";
  severity: "low" | "medium" | "high" | "critical";
  title: string;
  message: string;
  timestamp: string;
  acknowledged: boolean;
  resolvedAt?: string;
  data?: Record<string, any>;
}

export interface PerformanceMetrics {
  responseTime: number;
  throughput: number;
  errorRate: number;
  uptime: number;
  activeConnections: number;
}

export interface HealthCheckResult {
  status: "healthy" | "unhealthy" | "degraded";
  checks: Array<{
    name: string;
    status: "pass" | "fail" | "warn";
    message: string;
    duration: number;
    timestamp: string;
  }>;
  overallHealth: number; // 0-100
}

export interface ResourceUsageHistory {
  timestamps: string[];
  cpu: number[];
  memory: number[];
  network: number[];
  disk: number[];
}

export const monitoringAPI = {
  /**
   * Get current system metrics
   */
  async getCurrentMetrics(): Promise<MonitoringMetrics> {
    return get<MonitoringMetrics>("/api/monitoring/metrics", {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Get metrics history for a time range
   */
  async getMetricsHistory(
    timeRange: "1m" | "5m" | "15m" | "1h" | "1d" = "1h",
    resolution?: number
  ): Promise<ResourceUsageHistory> {
    const params = new URLSearchParams({ timeRange });
    if (resolution) params.set("resolution", resolution.toString());

    return get<ResourceUsageHistory>(`/api/monitoring/metrics/history?${params}`, {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Get recent activity events
   */
  async getActivityEvents(
    limit = 50,
    types?: string[],
    severity?: string[]
  ): Promise<ActivityEvent[]> {
    const params = new URLSearchParams({ limit: limit.toString() });

    if (types?.length) {
      types.forEach(type => params.append("type", type));
    }
    if (severity?.length) {
      severity.forEach(sev => params.append("severity", sev));
    }

    return get<ActivityEvent[]>(`/api/monitoring/activity?${params}`, {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Get active system alerts
   */
  async getActiveAlerts(): Promise<SystemAlert[]> {
    return get<SystemAlert[]>("/api/monitoring/alerts", {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Acknowledge an alert
   */
  async acknowledgeAlert(alertId: string): Promise<void> {
    return get<void>(`/api/monitoring/alerts/${alertId}/acknowledge`, {
      showLoading: false,
      showSuccess: true,
    });
  },

  /**
   * Resolve an alert
   */
  async resolveAlert(alertId: string, resolution?: string): Promise<void> {
    const requestData = resolution ? { resolution } : {};
    return get<void>(`/api/monitoring/alerts/${alertId}/resolve`, {
      data: requestData,
      showLoading: false,
      showSuccess: true,
    });
  },

  /**
   * Get performance metrics
   */
  async getPerformanceMetrics(): Promise<PerformanceMetrics> {
    return get<PerformanceMetrics>("/api/monitoring/performance", {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Get system health check results
   */
  async getHealthCheck(): Promise<HealthCheckResult> {
    return get<HealthCheckResult>("/api/monitoring/health", {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Get container-specific metrics
   */
  async getContainerMetrics(containerId: string): Promise<{
    cpu: number;
    memory: number;
    network: {
      in: number;
      out: number;
    };
    disk: {
      read: number;
      write: number;
    };
    status: string;
    uptime: number;
  }> {
    return get<{
      cpu: number;
      memory: number;
      network: {
        in: number;
        out: number;
      };
      disk: {
        read: number;
        write: number;
      };
      status: string;
      uptime: number;
    }>(`/api/monitoring/containers/${containerId}/metrics`, {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Get monitoring configuration
   */
  async getMonitoringConfig(): Promise<{
    updateInterval: number;
    alertThresholds: {
      cpu: number;
      memory: number;
      disk: number;
      network: number;
    };
    retentionPeriod: number;
    enabledCollectors: string[];
  }> {
    return get<{
      updateInterval: number;
      alertThresholds: {
        cpu: number;
        memory: number;
        disk: number;
        network: number;
      };
      retentionPeriod: number;
      enabledCollectors: string[];
    }>("/api/monitoring/config", {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Update monitoring configuration
   */
  async updateMonitoringConfig(config: {
    updateInterval?: number;
    alertThresholds?: {
      cpu?: number;
      memory?: number;
      disk?: number;
      network?: number;
    };
    retentionPeriod?: number;
    enabledCollectors?: string[];
  }): Promise<void> {
    return get<void>("/api/monitoring/config", {
      data: config,
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Export monitoring data
   */
  async exportMetrics(
    format: "csv" | "json" | "excel",
    timeRange: "1h" | "1d" | "1w" | "1m" = "1d",
    metrics?: string[]
  ): Promise<Blob> {
    const params = new URLSearchParams({ format, timeRange });

    if (metrics?.length) {
      metrics.forEach(metric => params.append("metric", metric));
    }

    const { request } = await import('@/utils/request');
    const response = await request({
      url: `/api/monitoring/export?${params}`,
      method: "GET",
      responseType: 'blob',
    });

    return response.data as Blob;
  },

  /**
   * Test monitoring connectivity
   */
  async testConnection(): Promise<{
    status: "connected" | "disconnected" | "degraded";
    latency: number;
    lastUpdate: string;
    errors?: string[];
  }> {
    return get<{
      status: "connected" | "disconnected" | "degraded";
      latency: number;
      lastUpdate: string;
      errors?: string[];
    }>("/api/monitoring/test", {
      showLoading: false,
      showError: false,
    });
  },
};