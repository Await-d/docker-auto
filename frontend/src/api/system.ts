/**
 * 系统管理 API
 */
import { get, post, put } from "@/utils/request";

export interface SystemInfo {
  appName: string;
  version: string;
  buildTime: string;
  goVersion: string;
  platform: string;
  uptime: string;
  startTime: string;
}

export interface SystemConfig {
  appName: string;
  version: string;
  environment: string;
  logLevel: string;
  port: number;
  databaseType: string;
  cacheEnabled: boolean;
  jwtExpireHours: number;
  autoUpdate: boolean;
  maxConcurrentUpdates: number;
  updateSchedule: string;
}

export interface SystemMetrics {
  cpuUsage: number;
  memoryUsage: number;
  diskUsage: number;
  networkIn: number;
  networkOut: number;
  activeContainers: number;
  totalContainers: number;
  runningUpdates: number;
  pendingUpdates: number;
}

export interface SystemHealth {
  status: "healthy" | "warning" | "error";
  checks: Array<{
    name: string;
    status: "ok" | "warning" | "error";
    message?: string;
    lastCheck: string;
  }>;
  uptime: number;
  version: string;
}

/**
 * 获取系统信息
 */
export const getSystemInfo = async (): Promise<SystemInfo> => {
  return await get("/api/system/info");
};

/**
 * 获取系统配置
 */
export const getSystemConfig = async (): Promise<SystemConfig> => {
  return await get("/api/system/config");
};

/**
 * 更新系统配置
 */
export const updateSystemConfig = async (config: Partial<SystemConfig>): Promise<void> => {
  return await put("/api/system/config", config);
};

/**
 * 获取系统指标
 */
export const getSystemMetrics = async (): Promise<SystemMetrics> => {
  return await get("/api/system/metrics");
};

/**
 * 获取系统健康状态
 */
export const getSystemHealth = async (): Promise<SystemHealth> => {
  return await get("/api/system/health");
};

/**
 * 重启系统服务
 */
export const restartSystem = async (): Promise<void> => {
  return await post("/api/system/restart");
};

/**
 * 获取系统日志配置
 */
export const getLogConfig = async (): Promise<{
  level: string;
  output: string[];
  format: string;
  maxSize: string;
  maxAge: number;
  maxBackups: number;
  compress: boolean;
}> => {
  return await get("/api/system/logs/config");
};

/**
 * 更新系统日志配置
 */
export const updateLogConfig = async (config: {
  level?: string;
  output?: string[];
  format?: string;
  maxSize?: string;
  maxAge?: number;
  maxBackups?: number;
  compress?: boolean;
}): Promise<void> => {
  return await put("/api/system/logs/config", config);
};

/**
 * 获取环境变量
 */
export const getEnvironmentVars = async (): Promise<Record<string, string>> => {
  return await get("/api/system/environment");
};

/**
 * 更新环境变量
 */
export const updateEnvironmentVars = async (vars: Record<string, string>): Promise<void> => {
  return await put("/api/system/environment", vars);
};

/**
 * 获取系统资源使用历史
 */
export const getResourceHistory = async (params?: {
  hours?: number;
  metrics?: string[];
}): Promise<{
  timestamps: string[];
  cpu: number[];
  memory: number[];
  disk: number[];
  network: number[];
}> => {
  return await get("/api/system/resources/history", { params });
};

/**
 * 执行系统备份
 */
export const createSystemBackup = async (): Promise<{ backupId: string; filename: string }> => {
  return await post("/api/system/backup");
};

/**
 * 获取备份列表
 */
export const getBackupList = async (): Promise<Array<{
  id: string;
  filename: string;
  size: number;
  createdAt: string;
  type: string;
}>> => {
  return await get("/api/system/backups");
};

/**
 * 恢复系统备份
 */
export const restoreSystemBackup = async (backupId: string): Promise<void> => {
  return await post(`/api/system/backup/${backupId}/restore`);
};