/**
 * 系统日志管理 API
 */
import { get, post } from "@/utils/request";

export interface LogEntry {
  id: string;
  timestamp: string;
  level: "error" | "warn" | "info" | "debug";
  source: string;
  message: string;
  details?: Record<string, any>;
  userId?: number;
  username?: string;
  ipAddress?: string;
  userAgent?: string;
}

export interface LogListResponse {
  data: LogEntry[];
  total: number;
  page: number;
  limit: number;
}

export interface LogFilterParams {
  page?: number;
  limit?: number;
  level?: string;
  source?: string;
  startTime?: string;
  endTime?: string;
  search?: string;
  userId?: number;
}

export interface LogExportParams {
  level?: string;
  source?: string;
  startTime?: string;
  endTime?: string;
  format?: "json" | "csv" | "txt";
}

/**
 * 获取系统日志列表
 */
export const getLogs = async (params?: LogFilterParams): Promise<LogListResponse> => {
  return await get("/api/logs", { params });
};

/**
 * 获取容器日志
 */
export const getContainerLogs = async (
  containerId: string,
  params?: {
    since?: string;
    until?: string;
    tail?: number;
    timestamps?: boolean;
    follow?: boolean;
  }
): Promise<string[]> => {
  return await get(`/api/containers/${containerId}/logs`, { params });
};

/**
 * 获取应用日志
 */
export const getApplicationLogs = async (params?: LogFilterParams): Promise<LogListResponse> => {
  return await get("/api/logs/application", { params });
};

/**
 * 获取审计日志
 */
export const getAuditLogs = async (params?: LogFilterParams): Promise<LogListResponse> => {
  return await get("/api/logs/audit", { params });
};

/**
 * 获取错误日志
 */
export const getErrorLogs = async (params?: LogFilterParams): Promise<LogListResponse> => {
  return await get("/api/logs/errors", { params });
};

/**
 * 导出日志
 */
export const exportLogs = async (params: LogExportParams): Promise<Blob> => {
  return await get("/api/logs/export", {
    params,
    responseType: "blob",
  });
};

/**
 * 清理旧日志
 */
export const cleanupLogs = async (params: {
  before?: string;
  level?: string;
  source?: string;
}): Promise<{ deletedCount: number }> => {
  return await post("/api/logs/cleanup", params);
};

/**
 * 获取日志统计信息
 */
export const getLogStats = async (params?: {
  startTime?: string;
  endTime?: string;
  groupBy?: "level" | "source" | "hour" | "day";
}): Promise<{
  total: number;
  byLevel: Record<string, number>;
  bySource: Record<string, number>;
  trend: Array<{ time: string; count: number }>;
}> => {
  return await get("/api/logs/stats", { params });
};

/**
 * 搜索日志
 */
export const searchLogs = async (params: {
  query: string;
  level?: string;
  source?: string;
  startTime?: string;
  endTime?: string;
  limit?: number;
}): Promise<LogListResponse> => {
  return await get("/api/logs/search", { params });
};