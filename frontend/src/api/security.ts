/**
 * Security API service for container vulnerability scanning and compliance
 */
import { get, post, put, del } from "@/utils/request";

export interface SecurityOverview {
  overallScore: number;
  totalVulnerabilities: number;
  vulnerabilities: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
  compliance: {
    passed: number;
    failed: number;
    total: number;
  };
  lastScan?: string;
  lastScanDuration?: number;
  scannedContainers?: number;
}

export interface ContainerSecurity {
  id: string;
  name: string;
  image: string;
  securityLevel: 'secure' | 'warning' | 'danger';
  vulnerabilities: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
  isCompliant: boolean;
  lastScan: string;
  scanDuration?: number;
}

export interface SecurityEvent {
  id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  timestamp: string;
  containerName?: string;
  resolved: boolean;
  type: string;
}

export interface SecurityRecommendation {
  id: string;
  title: string;
  description: string;
  type: 'security' | 'compliance' | 'performance' | 'configuration';
  priority: 'high' | 'medium' | 'low';
  actionable: boolean;
  actionText?: string;
  affectedContainers?: string[];
}

export interface Vulnerability {
  id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  cve?: string;
  cvssScore?: number;
  containerName: string;
  containerImage: string;
  fixAvailable: boolean;
  fixVersion?: string;
  publishedDate: string;
  modifiedDate?: string;
}

export interface ScanResult {
  scanId: string;
  status: 'queued' | 'running' | 'completed' | 'failed';
  progress?: number;
  startTime: string;
  endTime?: string;
  containersScanned: number;
  vulnerabilitiesFound: number;
  complianceIssues: number;
}

export interface ComplianceCheck {
  id: string;
  name: string;
  description: string;
  category: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  status: 'passed' | 'failed' | 'warning' | 'not_applicable';
  containerId?: string;
  containerName?: string;
  remediation?: string;
}

export interface SecurityPolicy {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  rules: Array<{
    type: string;
    condition: string;
    action: string;
    parameters?: any;
  }>;
  appliesTo: {
    containers?: string[];
    images?: string[];
    labels?: Record<string, string>;
  };
}

export const securityAPI = {
  /**
   * Get overall security overview and metrics
   */
  async getSecurityOverview(): Promise<SecurityOverview> {
    return get<SecurityOverview>("/api/security/overview");
  },

  /**
   * Get security status for all containers
   */
  async getContainerSecurity(): Promise<ContainerSecurity[]> {
    return get<ContainerSecurity[]>("/api/security/containers");
  },

  /**
   * Get security status for a specific container
   */
  async getContainerSecurityDetails(containerId: string): Promise<ContainerSecurity & {
    vulnerabilities: Vulnerability[];
    complianceChecks: ComplianceCheck[];
    scanHistory: Array<{
      scanId: string;
      timestamp: string;
      vulnerabilitiesFound: number;
      complianceScore: number;
    }>;
  }> {
    return get<ContainerSecurity & {
      vulnerabilities: Vulnerability[];
      complianceChecks: ComplianceCheck[];
      scanHistory: Array<{
        scanId: string;
        timestamp: string;
        vulnerabilitiesFound: number;
        complianceScore: number;
      }>;
    }>(`/api/security/containers/${containerId}`);
  },

  /**
   * Get recent security events
   */
  async getSecurityEvents(limit = 20): Promise<SecurityEvent[]> {
    return get<SecurityEvent[]>(`/api/security/events?limit=${limit}`);
  },

  /**
   * Get security recommendations
   */
  async getSecurityRecommendations(): Promise<SecurityRecommendation[]> {
    return get<SecurityRecommendation[]>("/api/security/recommendations");
  },

  /**
   * Get critical vulnerabilities that need immediate attention
   */
  async getCriticalVulnerabilities(): Promise<Vulnerability[]> {
    return get<Vulnerability[]>("/api/security/vulnerabilities/critical");
  },

  /**
   * Get all vulnerabilities with filtering and pagination
   */
  async getVulnerabilities(params?: {
    severity?: 'critical' | 'high' | 'medium' | 'low';
    containerId?: string;
    status?: 'open' | 'fixed' | 'ignored';
    page?: number;
    limit?: number;
  }): Promise<{
    vulnerabilities: Vulnerability[];
    total: number;
    page: number;
    limit: number;
  }> {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.set(key, value.toString());
        }
      });
    }

    return get<{
      vulnerabilities: Vulnerability[];
      total: number;
      page: number;
      limit: number;
    }>(`/api/security/vulnerabilities?${searchParams}`);
  },

  /**
   * Start a security scan for all containers or specific containers
   */
  async startSecurityScan(containerIds?: string[]): Promise<{
    scanId: string;
    status: string;
    containersQueued: number;
  }> {
    return post<{
      scanId: string;
      status: string;
      containersQueued: number;
    }>("/api/security/scan", { containerIds });
  },

  /**
   * Get security scan status and results
   */
  async getScanStatus(scanId: string): Promise<ScanResult> {
    return get<ScanResult>(`/api/security/scan/${scanId}`);
  },

  /**
   * Get scan history
   */
  async getScanHistory(page = 1, limit = 20): Promise<{
    scans: ScanResult[];
    total: number;
    page: number;
    limit: number;
  }> {
    return get<{
      scans: ScanResult[];
      total: number;
      page: number;
      limit: number;
    }>(`/api/security/scans?page=${page}&limit=${limit}`);
  },

  /**
   * Cancel a running security scan
   */
  async cancelScan(scanId: string): Promise<void> {
    return post<void>(`/api/security/scan/${scanId}/cancel`);
  },

  /**
   * Fix vulnerabilities automatically
   */
  async autoFixVulnerabilities(vulnerabilityIds: string[]): Promise<{
    operationId: string;
    fixableCount: number;
    unfixableCount: number;
  }> {
    return post<{
      operationId: string;
      fixableCount: number;
      unfixableCount: number;
    }>("/api/security/vulnerabilities/fix", { vulnerabilityIds });
  },

  /**
   * Fix specific vulnerabilities
   */
  async fixVulnerabilities(vulnerabilityIds: string[]): Promise<{
    operationId: string;
    fixableCount: number;
    unfixableCount: number;
  }> {
    return post<{
      operationId: string;
      fixableCount: number;
      unfixableCount: number;
    }>("/api/security/vulnerabilities/fix", { vulnerabilityIds });
  },

  /**
   * Ignore/suppress vulnerabilities
   */
  async ignoreVulnerabilities(
    vulnerabilityIds: string[],
    reason?: string,
    expiresAt?: string
  ): Promise<void> {
    return post<void>("/api/security/vulnerabilities/ignore", {
      vulnerabilityIds,
      reason,
      expiresAt,
    });
  },

  /**
   * Unignore vulnerabilities
   */
  async unignoreVulnerabilities(vulnerabilityIds: string[]): Promise<void> {
    return post<void>("/api/security/vulnerabilities/unignore", {
      vulnerabilityIds,
    });
  },

  /**
   * Execute a security recommendation
   */
  async executeRecommendation(recommendation: SecurityRecommendation): Promise<{
    operationId: string;
    status: string;
  }> {
    return post<{
      operationId: string;
      status: string;
    }>(`/api/security/recommendations/${recommendation.id}/execute`);
  },

  /**
   * Dismiss a security recommendation
   */
  async dismissRecommendation(recommendationId: string, reason?: string): Promise<void> {
    return post<void>(`/api/security/recommendations/${recommendationId}/dismiss`, {
      reason,
    });
  },

  /**
   * Get compliance checks and results
   */
  async getComplianceChecks(containerId?: string): Promise<ComplianceCheck[]> {
    const params = containerId ? `?containerId=${containerId}` : '';
    return get<ComplianceCheck[]>(`/api/security/compliance${params}`);
  },

  /**
   * Run compliance checks
   */
  async runComplianceChecks(containerIds?: string[]): Promise<{
    operationId: string;
    containersQueued: number;
  }> {
    return post<{
      operationId: string;
      containersQueued: number;
    }>("/api/security/compliance/run", { containerIds });
  },

  /**
   * Get security policies
   */
  async getSecurityPolicies(): Promise<SecurityPolicy[]> {
    return get<SecurityPolicy[]>("/api/security/policies");
  },

  /**
   * Create a new security policy
   */
  async createSecurityPolicy(policy: Omit<SecurityPolicy, 'id'>): Promise<SecurityPolicy> {
    return post<SecurityPolicy>("/api/security/policies", policy);
  },

  /**
   * Update an existing security policy
   */
  async updateSecurityPolicy(
    policyId: string,
    policy: Partial<SecurityPolicy>
  ): Promise<SecurityPolicy> {
    return put<SecurityPolicy>(`/api/security/policies/${policyId}`, policy);
  },

  /**
   * Delete a security policy
   */
  async deleteSecurityPolicy(policyId: string): Promise<void> {
    return del<void>(`/api/security/policies/${policyId}`);
  },

  /**
   * Apply security policies to containers
   */
  async applySecurityPolicies(
    policyIds: string[],
    containerIds?: string[]
  ): Promise<{
    operationId: string;
    appliedPolicies: number;
    affectedContainers: number;
  }> {
    return post<{
      operationId: string;
      appliedPolicies: number;
      affectedContainers: number;
    }>("/api/security/policies/apply", { policyIds, containerIds });
  },

  /**
   * Get security benchmarks (CIS, NIST, etc.)
   */
  async getSecurityBenchmarks(): Promise<Array<{
    id: string;
    name: string;
    description: string;
    version: string;
    framework: string;
    checks: Array<{
      id: string;
      title: string;
      description: string;
      severity: string;
      status: 'passed' | 'failed' | 'not_applicable';
      remediation?: string;
    }>;
  }>> {
    return get<Array<{
      id: string;
      name: string;
      description: string;
      version: string;
      framework: string;
      checks: Array<{
        id: string;
        title: string;
        description: string;
        severity: string;
        status: 'passed' | 'failed' | 'not_applicable';
        remediation?: string;
      }>;
    }>>("/api/security/benchmarks");
  },

  /**
   * Run security benchmarks
   */
  async runSecurityBenchmarks(
    benchmarkIds: string[],
    containerIds?: string[]
  ): Promise<{
    operationId: string;
    benchmarksQueued: number;
    containersQueued: number;
  }> {
    return post<{
      operationId: string;
      benchmarksQueued: number;
      containersQueued: number;
    }>("/api/security/benchmarks/run", { benchmarkIds, containerIds });
  },

  /**
   * Get security audit logs
   */
  async getSecurityAuditLogs(params?: {
    startDate?: string;
    endDate?: string;
    eventType?: string;
    severity?: string;
    containerId?: string;
    page?: number;
    limit?: number;
  }): Promise<{
    logs: Array<{
      id: string;
      timestamp: string;
      eventType: string;
      severity: string;
      message: string;
      containerId?: string;
      containerName?: string;
      userId?: string;
      userAgent?: string;
      ipAddress?: string;
      details?: any;
    }>;
    total: number;
    page: number;
    limit: number;
  }> {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.set(key, value.toString());
        }
      });
    }

    return get<{
      logs: Array<{
        id: string;
        timestamp: string;
        eventType: string;
        severity: string;
        message: string;
        containerId?: string;
        containerName?: string;
        userId?: string;
        userAgent?: string;
        ipAddress?: string;
        details?: any;
      }>;
      total: number;
      page: number;
      limit: number;
    }>(`/api/security/audit-logs?${searchParams}`);
  },

  /**
   * Export security report
   */
  async exportSecurityReport(
    format: 'pdf' | 'csv' | 'json',
    params?: {
      includeVulnerabilities?: boolean;
      includeCompliance?: boolean;
      includeBenchmarks?: boolean;
      containerIds?: string[];
      dateRange?: { start: string; end: string };
    }
  ): Promise<Blob> {
    const { request } = await import('@/utils/request');

    const searchParams = new URLSearchParams({ format });
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          if (Array.isArray(value)) {
            value.forEach(v => searchParams.append(key, v.toString()));
          } else if (typeof value === 'object') {
            searchParams.set(key, JSON.stringify(value));
          } else {
            searchParams.set(key, value.toString());
          }
        }
      });
    }

    const response = await request({
      url: `/api/security/report/export?${searchParams}`,
      method: 'GET',
      responseType: 'blob',
    });

    return response.data as Blob;
  },

  /**
   * Get security settings
   */
  async getSecuritySettings(): Promise<{
    scanSchedule: {
      enabled: boolean;
      frequency: string; // cron expression
      scanOptions: {
        includeVulnerabilities: boolean;
        includeCompliance: boolean;
        includeBenchmarks: boolean;
      };
    };
    notifications: {
      enabled: boolean;
      channels: string[];
      severity: string[];
    };
    autoRemediation: {
      enabled: boolean;
      allowedActions: string[];
      maxRiskLevel: string;
    };
    retention: {
      scanResults: number; // days
      auditLogs: number; // days
      reports: number; // days
    };
  }> {
    return get<{
      scanSchedule: {
        enabled: boolean;
        frequency: string;
        scanOptions: {
          includeVulnerabilities: boolean;
          includeCompliance: boolean;
          includeBenchmarks: boolean;
        };
      };
      notifications: {
        enabled: boolean;
        channels: string[];
        severity: string[];
      };
      autoRemediation: {
        enabled: boolean;
        allowedActions: string[];
        maxRiskLevel: string;
      };
      retention: {
        scanResults: number;
        auditLogs: number;
        reports: number;
      };
    }>("/api/security/settings");
  },

  /**
   * Update security settings
   */
  async updateSecuritySettings(settings: {
    scanSchedule?: {
      enabled?: boolean;
      frequency?: string;
      scanOptions?: {
        includeVulnerabilities?: boolean;
        includeCompliance?: boolean;
        includeBenchmarks?: boolean;
      };
    };
    notifications?: {
      enabled?: boolean;
      channels?: string[];
      severity?: string[];
    };
    autoRemediation?: {
      enabled?: boolean;
      allowedActions?: string[];
      maxRiskLevel?: string;
    };
    retention?: {
      scanResults?: number;
      auditLogs?: number;
      reports?: number;
    };
  }): Promise<void> {
    return put<void>("/api/security/settings", settings);
  },

  /**
   * Test security scanner connectivity
   */
  async testScannerConnectivity(): Promise<{
    status: 'connected' | 'disconnected' | 'error';
    scannerType: string;
    version?: string;
    lastUpdate?: string;
    databases?: Array<{
      name: string;
      version: string;
      lastUpdate: string;
    }>;
  }> {
    return get<{
      status: 'connected' | 'disconnected' | 'error';
      scannerType: string;
      version?: string;
      lastUpdate?: string;
      databases?: Array<{
        name: string;
        version: string;
        lastUpdate: string;
      }>;
    }>("/api/security/scanner/status");
  },

  /**
   * Update vulnerability databases
   */
  async updateVulnerabilityDatabases(): Promise<{
    operationId: string;
    status: string;
    databases: string[];
  }> {
    return post<{
      operationId: string;
      status: string;
      databases: string[];
    }>("/api/security/scanner/update-databases");
  },
};