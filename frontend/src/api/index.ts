/**
 * API 接口统一导出
 */

// 用户和认证相关
export * from "./user";

// 容器管理
export * from "./container";

// 镜像管理
export * from "./images";

// 更新管理
export * from "./updates";

// 日志管理
export * from "./logs";

// 系统管理
export * from "./system";

// 监控相关
export * from "./monitoring";

// 重新导出类型
export type { ApiResponse } from "@/utils/request";

// 统一的API基础URL
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";