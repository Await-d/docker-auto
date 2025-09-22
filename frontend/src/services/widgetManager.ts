/**
 * 小部件管理服务 - 处理小部件生命周期、数据管理和通信
 */
import { reactive } from "vue";
import type { DashboardWidget } from "@/store/dashboard";
import { useContainerWebSocket } from "@/services/containerWebSocket";
// import { useAuthStore } from "@/store/auth"; // Unused import

// 小部件数据刷新策略
export enum RefreshStrategy {
  INTERVAL = "interval",
  WEBSOCKET = "websocket",
  MANUAL = "manual",
  ON_FOCUS = "on_focus",
}

// 小部件状态
export enum WidgetStatus {
  LOADING = "loading",
  LOADED = "loaded",
  ERROR = "error",
  OFFLINE = "offline",
}

// 小部件通信事件
export interface WidgetEvent {
  type: string;
  source: string;
  target?: string;
  data: any;
  timestamp: Date;
}

// 小部件性能指标
export interface WidgetMetrics {
  widgetId: string;
  loadTime: number;
  renderTime: number;
  dataSize: number;
  errorCount: number;
  lastUpdate: Date;
  refreshCount: number;
}

// 小部件数据缓存条目
export interface CacheEntry {
  data: any;
  timestamp: Date;
  ttl: number;
  version: string;
}

class WidgetManagerService {
  private widgets = reactive<Map<string, DashboardWidget>>(new Map());
  private widgetStatus = reactive<Map<string, WidgetStatus>>(new Map());
  private widgetData = reactive<Map<string, any>>(new Map());
  private widgetCache = reactive<Map<string, CacheEntry>>(new Map());
  private widgetMetrics = reactive<Map<string, WidgetMetrics>>(new Map());
  private refreshIntervals = new Map<string, NodeJS.Timeout>();
  private eventBus = reactive<WidgetEvent[]>([]);
  private eventHandlers = new Map<string, (event: WidgetEvent) => void>();
  private cleanupFunctions = new Map<string, () => void>();
  private maxCacheSize = 100;
  private maxEventHistory = 1000;

  // 性能监控
  private performanceObserver?: PerformanceObserver;

  constructor() {
    this.initializePerformanceMonitoring();
    this.setupGlobalErrorHandler();
  }

  /**
   * 初始化小部件性能监控
   */
  private initializePerformanceMonitoring() {
    if (typeof PerformanceObserver !== "undefined") {
      this.performanceObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.name.startsWith("widget-")) {
            const widgetId = entry.name.replace("widget-", "");
            this.updateMetrics(widgetId, {
              renderTime: entry.duration,
            });
          }
        }
      });

      this.performanceObserver.observe({ entryTypes: ["measure"] });
    }
  }

  /**
   * 为小部件设置全局错误处理程序
   */
  private setupGlobalErrorHandler() {
    window.addEventListener("error", (event) => {
      // 检查错误是否来自小部件
      const target = event.target as any;
      if (target?.closest?.("[data-widget-id]")) {
        const widgetId = target.closest("[data-widget-id]").dataset.widgetId;
        this.handleWidgetError(widgetId, event.error);
      }
    });
  }

  /**
   * 注册小部件实例
   */
  registerWidget(widget: DashboardWidget): void {
    this.widgets.set(widget.id, widget);
    this.widgetStatus.set(widget.id, WidgetStatus.LOADING);

    // 初始化指标
    this.widgetMetrics.set(widget.id, {
      widgetId: widget.id,
      loadTime: 0,
      renderTime: 0,
      dataSize: 0,
      errorCount: 0,
      lastUpdate: new Date(),
      refreshCount: 0,
    });

    // 设置刷新策略
    this.setupWidgetRefresh(widget);

    this.emitEvent({
      type: "widget:registered",
      source: widget.id,
      data: { widget },
      timestamp: new Date(),
    });
  }

  /**
   * 取消注册小部件实例
   */
  unregisterWidget(widgetId: string): void {
    // 清除刷新间隔
    this.clearWidgetRefresh(widgetId);

    // 清理数据
    this.widgets.delete(widgetId);
    this.widgetStatus.delete(widgetId);
    this.widgetData.delete(widgetId);
    this.widgetMetrics.delete(widgetId);

    // 清除缓存条目
    this.clearWidgetCache(widgetId);

    this.emitEvent({
      type: "widget:unregistered",
      source: widgetId,
      data: { widgetId },
      timestamp: new Date(),
    });
  }

  /**
   * 设置小部件刷新策略
   */
  private setupWidgetRefresh(widget: DashboardWidget): void {
    this.clearWidgetRefresh(widget.id);

    const strategy = this.getRefreshStrategy(widget);

    switch (strategy) {
      case RefreshStrategy.INTERVAL:
        if (widget.refreshInterval > 0) {
          const interval = setInterval(() => {
            this.refreshWidget(widget.id);
          }, widget.refreshInterval);
          this.refreshIntervals.set(widget.id, interval);
        }
        break;

      case RefreshStrategy.WEBSOCKET:
        this.setupWebSocketRefresh(widget);
        break;

      case RefreshStrategy.ON_FOCUS:
        this.setupFocusRefresh(widget);
        break;
    }
  }

  /**
   * 确定小部件的刷新策略
   */
  private getRefreshStrategy(widget: DashboardWidget): RefreshStrategy {
    // 高频率小部件使用 WebSocket
    if (widget.refreshInterval <= 5000) {
      return RefreshStrategy.WEBSOCKET;
    }

    // 交互式小部件使用基于焦点的刷新
    if (["quick-actions", "notification-center"].includes(widget.type)) {
      return RefreshStrategy.ON_FOCUS;
    }

    // 默认使用间隔刷新
    return RefreshStrategy.INTERVAL;
  }

  /**
   * 设置基于 WebSocket 的刷新
   */
  private setupWebSocketRefresh(widget: DashboardWidget): void {
    const containerWS = useContainerWebSocket();

    // 根据小部件类型订阅相关的 WebSocket 事件
    const events = this.getWebSocketEvents(widget.type);

    events.forEach((event) => {
      // 注意：小部件特定的 WebSocket 订阅将放在这里
      // 目前，根据事件类型使用通用订阅方法
      if (event.includes("container")) {
        containerWS.subscribeToAll();
      }
    });
  }

  /**
   * 获取小部件类型的 WebSocket 事件
   */
  private getWebSocketEvents(widgetType: string): string[] {
    const eventMap: Record<string, string[]> = {
      "system-overview": ["system:status", "system:metrics"],
      "container-stats": ["container:stats", "container:status"],
      "update-activity": [
        "update:started",
        "update:completed",
        "update:failed",
      ],
      "realtime-monitor": ["system:metrics", "container:events"],
      "health-monitor": ["health:check", "service:status"],
      "recent-activities": ["activity:new"],
      "notification-center": ["notification:new", "alert:new"],
      "resource-charts": ["metrics:resource"],
      "security-dashboard": ["security:scan", "vulnerability:found"],
    };

    return eventMap[widgetType] || [];
  }

  /**
   * 设置基于焦点的刷新
   */
  private setupFocusRefresh(widget: DashboardWidget): void {
    const handleFocus = () => {
      this.refreshWidget(widget.id);
    };

    window.addEventListener("focus", handleFocus);

    // 存储清理函数以便后续使用
    this.cleanupFunctions.set(widget.id, () => {
      window.removeEventListener("focus", handleFocus);
    });
  }

  /**
   * 清除小部件刷新机制
   */
  private clearWidgetRefresh(widgetId: string): void {
    const interval = this.refreshIntervals.get(widgetId);
    if (interval) {
      clearInterval(interval);
      this.refreshIntervals.delete(widgetId);
    }
  }

  /**
   * 刷新小部件数据
   */
  async refreshWidget(widgetId: string, force = false): Promise<void> {
    const widget = this.widgets.get(widgetId);
    if (!widget) return;

    try {
      this.widgetStatus.set(widgetId, WidgetStatus.LOADING);

      const startTime = performance.now();

      // 首先检查缓存（除非强制）
      if (!force) {
        const cachedData = this.getCachedData(widgetId);
        if (cachedData) {
          this.updateWidgetData(widgetId, cachedData);
          this.widgetStatus.set(widgetId, WidgetStatus.LOADED);
          return;
        }
      }

      // 获取新数据
      const data = await this.fetchWidgetData(widget);

      const loadTime = performance.now() - startTime;

      // 更新数据和缓存
      this.updateWidgetData(widgetId, data);
      this.setCachedData(widgetId, data);

      // 更新指标
      this.updateMetrics(widgetId, {
        loadTime,
        dataSize: JSON.stringify(data).length,
        lastUpdate: new Date(),
        refreshCount: (this.widgetMetrics.get(widgetId)?.refreshCount || 0) + 1,
      });

      this.widgetStatus.set(widgetId, WidgetStatus.LOADED);

      this.emitEvent({
        type: "widget:refreshed",
        source: widgetId,
        data: { widget, loadTime },
        timestamp: new Date(),
      });
    } catch (error) {
      this.handleWidgetError(widgetId, error);
    }
  }

  /**
   * 从适当的源获取小部件数据
   */
  private async fetchWidgetData(widget: DashboardWidget): Promise<any> {
    // 根据小部件类型构建 API 端点
    const endpoint = this.getWidgetEndpoint(widget.type);

    const { get } = await import('@/utils/request');
    return await get(endpoint, {
      showLoading: false,
      showError: false,
    });
  }


  /**
   * 获取小部件类型的 API 端点
   */
  private getWidgetEndpoint(widgetType: string): string {
    const endpointMap: Record<string, string> = {
      "system-overview": "/api/dashboard/system-overview",
      "container-stats": "/api/dashboard/container-stats",
      "update-activity": "/api/dashboard/update-activity",
      "realtime-monitor": "/api/dashboard/realtime-metrics",
      "health-monitor": "/api/dashboard/health-status",
      "recent-activities": "/api/dashboard/recent-activities",
      "quick-actions": "/api/dashboard/quick-actions",
      "notification-center": "/api/dashboard/notifications",
      "resource-charts": "/api/dashboard/resource-metrics",
      "security-dashboard": "/api/dashboard/security-status",
    };

    return endpointMap[widgetType] || "/api/dashboard/generic";
  }

  /**
   * 更新小部件数据
   */
  updateWidgetData(widgetId: string, data: any): void {
    this.widgetData.set(widgetId, data);

    this.emitEvent({
      type: "widget:data-updated",
      source: widgetId,
      data,
      timestamp: new Date(),
    });
  }

  /**
   * 获取小部件数据
   */
  getWidgetData(widgetId: string): any {
    return this.widgetData.get(widgetId);
  }

  /**
   * 获取小部件状态
   */
  getWidgetStatus(widgetId: string): WidgetStatus {
    return this.widgetStatus.get(widgetId) || WidgetStatus.LOADING;
  }

  /**
   * 处理小部件错误
   */
  private handleWidgetError(widgetId: string, error: any): void {
    console.error(`小部件 ${widgetId} 错误:`, error);

    this.widgetStatus.set(widgetId, WidgetStatus.ERROR);

    // 更新错误指标
    const metrics = this.widgetMetrics.get(widgetId);
    if (metrics) {
      metrics.errorCount++;
      this.widgetMetrics.set(widgetId, metrics);
    }

    this.emitEvent({
      type: "widget:error",
      source: widgetId,
      data: { error: error.message || error.toString() },
      timestamp: new Date(),
    });
  }

  /**
   * 缓存管理
   */
  private getCachedData(widgetId: string): any | null {
    const entry = this.widgetCache.get(widgetId);
    if (!entry) return null;

    const now = Date.now();
    const isExpired = now - entry.timestamp.getTime() > entry.ttl;

    if (isExpired) {
      this.widgetCache.delete(widgetId);
      return null;
    }

    return entry.data;
  }

  private setCachedData(widgetId: string, data: any, ttl = 30000): void {
    // 实现 LRU 缓存清除
    if (this.widgetCache.size >= this.maxCacheSize) {
      const oldestKey = this.widgetCache.keys().next().value;
      if (oldestKey) {
        this.widgetCache.delete(oldestKey);
      }
    }

    this.widgetCache.set(widgetId, {
      data,
      timestamp: new Date(),
      ttl,
      version: "1.0",
    });
  }

  private clearWidgetCache(widgetId: string): void {
    this.widgetCache.delete(widgetId);
  }

  /**
   * 更新小部件指标
   */
  private updateMetrics(
    widgetId: string,
    updates: Partial<WidgetMetrics>,
  ): void {
    const current = this.widgetMetrics.get(widgetId);
    if (current) {
      this.widgetMetrics.set(widgetId, { ...current, ...updates });
    }
  }

  /**
   * 小部件通信的事件总线
   */
  private emitEvent(event: WidgetEvent): void {
    this.eventBus.push(event);

    // 限制事件历史
    if (this.eventBus.length > this.maxEventHistory) {
      this.eventBus.splice(0, this.eventBus.length - this.maxEventHistory);
    }
  }

  /**
   * 订阅小部件事件
   */
  subscribeToEvents(
    eventType: string,
    callback: (event: WidgetEvent) => void,
  ): () => void {
    const handler = (event: WidgetEvent) => {
      if (event.type === eventType) {
        callback(event);
      }
    };

    // 存储事件订阅的处理程序（简化实现）
    const handlerKey = `${eventType}-${Date.now()}`;
    this.eventHandlers.set(handlerKey, handler);

    const stopWatching = () => {
      this.eventHandlers.delete(handlerKey);
    };

    return stopWatching;
  }

  /**
   * 批量操作以提高性能
   */
  async refreshMultipleWidgets(
    widgetIds: string[],
    force = false,
  ): Promise<void> {
    const refreshPromises = widgetIds.map((id) =>
      this.refreshWidget(id, force),
    );
    await Promise.allSettled(refreshPromises);
  }

  /**
   * 小部件分析和指标
   */
  getWidgetMetrics(): Map<string, WidgetMetrics>;
  getWidgetMetrics(widgetId: string): WidgetMetrics | undefined;
  getWidgetMetrics(
    widgetId?: string,
  ): WidgetMetrics | undefined | Map<string, WidgetMetrics> {
    if (widgetId) {
      return this.widgetMetrics.get(widgetId);
    }
    return new Map(this.widgetMetrics);
  }

  /**
   * 性能优化
   */
  optimizePerformance(): void {
    // 清除旧的缓存条目
    const now = Date.now();
    for (const [key, entry] of this.widgetCache.entries()) {
      if (now - entry.timestamp.getTime() > entry.ttl) {
        this.widgetCache.delete(key);
      }
    }

    // 清理旧事件
    if (this.eventBus.length > this.maxEventHistory) {
      this.eventBus.splice(0, this.eventBus.length - this.maxEventHistory);
    }
  }

  /**
   * 清理资源
   */
  destroy(): void {
    // 清除所有间隔
    for (const interval of this.refreshIntervals.values()) {
      clearInterval(interval);
    }
    this.refreshIntervals.clear();

    // 清除性能观察器
    if (this.performanceObserver) {
      this.performanceObserver.disconnect();
    }

    // 清除所有数据
    this.widgets.clear();
    this.widgetStatus.clear();
    this.widgetData.clear();
    this.widgetCache.clear();
    this.widgetMetrics.clear();
    this.eventBus.length = 0;
  }
}

// 单例实例
export const widgetManager = new WidgetManagerService();

// 用于小部件管理的 Vue 组合式函数
export function useWidgetManager() {
  return {
    registerWidget: widgetManager.registerWidget.bind(widgetManager),
    unregisterWidget: widgetManager.unregisterWidget.bind(widgetManager),
    refreshWidget: widgetManager.refreshWidget.bind(widgetManager),
    getWidgetData: widgetManager.getWidgetData.bind(widgetManager),
    getWidgetStatus: widgetManager.getWidgetStatus.bind(widgetManager),
    updateWidgetData: widgetManager.updateWidgetData.bind(widgetManager),
    subscribeToEvents: widgetManager.subscribeToEvents.bind(widgetManager),
    getWidgetMetrics: widgetManager.getWidgetMetrics.bind(widgetManager),
    refreshMultipleWidgets:
      widgetManager.refreshMultipleWidgets.bind(widgetManager),
  };
}

export default widgetManager;
