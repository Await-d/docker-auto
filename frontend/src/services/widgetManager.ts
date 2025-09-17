/**
 * Widget Manager Service - Handles widget lifecycle, data management, and communication
 */
import { reactive } from "vue";
import type { DashboardWidget } from "@/store/dashboard";
import { useContainerWebSocket } from "@/services/containerWebSocket";
import { useAuthStore } from "@/store/auth";

// Widget data refresh strategies
export enum RefreshStrategy {
  INTERVAL = "interval",
  WEBSOCKET = "websocket",
  MANUAL = "manual",
  ON_FOCUS = "on_focus",
}

// Widget status states
export enum WidgetStatus {
  LOADING = "loading",
  LOADED = "loaded",
  ERROR = "error",
  OFFLINE = "offline",
}

// Widget communication events
export interface WidgetEvent {
  type: string;
  source: string;
  target?: string;
  data: any;
  timestamp: Date;
}

// Widget performance metrics
export interface WidgetMetrics {
  widgetId: string;
  loadTime: number;
  renderTime: number;
  dataSize: number;
  errorCount: number;
  lastUpdate: Date;
  refreshCount: number;
}

// Widget data cache entry
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

  // Performance monitoring
  private performanceObserver?: PerformanceObserver;

  constructor() {
    this.initializePerformanceMonitoring();
    this.setupGlobalErrorHandler();
  }

  /**
   * Initialize performance monitoring for widgets
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
   * Setup global error handler for widgets
   */
  private setupGlobalErrorHandler() {
    window.addEventListener("error", (event) => {
      // Check if error is from a widget
      const target = event.target as any;
      if (target?.closest?.("[data-widget-id]")) {
        const widgetId = target.closest("[data-widget-id]").dataset.widgetId;
        this.handleWidgetError(widgetId, event.error);
      }
    });
  }

  /**
   * Register a widget instance
   */
  registerWidget(widget: DashboardWidget): void {
    this.widgets.set(widget.id, widget);
    this.widgetStatus.set(widget.id, WidgetStatus.LOADING);

    // Initialize metrics
    this.widgetMetrics.set(widget.id, {
      widgetId: widget.id,
      loadTime: 0,
      renderTime: 0,
      dataSize: 0,
      errorCount: 0,
      lastUpdate: new Date(),
      refreshCount: 0,
    });

    // Setup refresh strategy
    this.setupWidgetRefresh(widget);

    this.emitEvent({
      type: "widget:registered",
      source: widget.id,
      data: { widget },
      timestamp: new Date(),
    });
  }

  /**
   * Unregister a widget instance
   */
  unregisterWidget(widgetId: string): void {
    // Clear refresh interval
    this.clearWidgetRefresh(widgetId);

    // Clean up data
    this.widgets.delete(widgetId);
    this.widgetStatus.delete(widgetId);
    this.widgetData.delete(widgetId);
    this.widgetMetrics.delete(widgetId);

    // Clear cache entries
    this.clearWidgetCache(widgetId);

    this.emitEvent({
      type: "widget:unregistered",
      source: widgetId,
      data: { widgetId },
      timestamp: new Date(),
    });
  }

  /**
   * Setup widget refresh strategy
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
   * Determine refresh strategy for widget
   */
  private getRefreshStrategy(widget: DashboardWidget): RefreshStrategy {
    // High-frequency widgets use WebSocket
    if (widget.refreshInterval <= 5000) {
      return RefreshStrategy.WEBSOCKET;
    }

    // Interactive widgets use focus-based refresh
    if (["quick-actions", "notification-center"].includes(widget.type)) {
      return RefreshStrategy.ON_FOCUS;
    }

    // Default to interval
    return RefreshStrategy.INTERVAL;
  }

  /**
   * Setup WebSocket-based refresh
   */
  private setupWebSocketRefresh(widget: DashboardWidget): void {
    const containerWS = useContainerWebSocket();

    // Subscribe to relevant WebSocket events based on widget type
    const events = this.getWebSocketEvents(widget.type);

    events.forEach((event) => {
      // Note: Widget-specific WebSocket subscription would go here
      // For now, use the general subscription methods based on event type
      if (event.includes("container")) {
        containerWS.subscribeToAll();
      }
    });
  }

  /**
   * Get WebSocket events for widget type
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
   * Setup focus-based refresh
   */
  private setupFocusRefresh(widget: DashboardWidget): void {
    const handleFocus = () => {
      this.refreshWidget(widget.id);
    };

    window.addEventListener("focus", handleFocus);

    // Store cleanup function for later use
    this.cleanupFunctions.set(widget.id, () => {
      window.removeEventListener("focus", handleFocus);
    });
  }

  /**
   * Clear widget refresh mechanisms
   */
  private clearWidgetRefresh(widgetId: string): void {
    const interval = this.refreshIntervals.get(widgetId);
    if (interval) {
      clearInterval(interval);
      this.refreshIntervals.delete(widgetId);
    }
  }

  /**
   * Refresh widget data
   */
  async refreshWidget(widgetId: string, force = false): Promise<void> {
    const widget = this.widgets.get(widgetId);
    if (!widget) return;

    try {
      this.widgetStatus.set(widgetId, WidgetStatus.LOADING);

      const startTime = performance.now();

      // Check cache first (unless forced)
      if (!force) {
        const cachedData = this.getCachedData(widgetId);
        if (cachedData) {
          this.updateWidgetData(widgetId, cachedData);
          this.widgetStatus.set(widgetId, WidgetStatus.LOADED);
          return;
        }
      }

      // Fetch fresh data
      const data = await this.fetchWidgetData(widget);

      const loadTime = performance.now() - startTime;

      // Update data and cache
      this.updateWidgetData(widgetId, data);
      this.setCachedData(widgetId, data);

      // Update metrics
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
   * Fetch widget data from appropriate source
   */
  private async fetchWidgetData(widget: DashboardWidget): Promise<any> {
    const authStore = useAuthStore();

    // Build API endpoint based on widget type
    const endpoint = this.getWidgetEndpoint(widget.type);

    const response = await fetch(endpoint, {
      headers: {
        Authorization: `Bearer ${authStore.token}`,
        "Content-Type": "application/json",
      },
    });

    if (!response.ok) {
      throw new Error(
        `Failed to fetch data for widget ${widget.type}: ${response.statusText}`,
      );
    }

    return await response.json();
  }

  /**
   * Get API endpoint for widget type
   */
  private getWidgetEndpoint(widgetType: string): string {
    const endpointMap: Record<string, string> = {
      "system-overview": "/api/v1/dashboard/system-overview",
      "container-stats": "/api/v1/dashboard/container-stats",
      "update-activity": "/api/v1/dashboard/update-activity",
      "realtime-monitor": "/api/v1/dashboard/realtime-metrics",
      "health-monitor": "/api/v1/dashboard/health-status",
      "recent-activities": "/api/v1/dashboard/recent-activities",
      "quick-actions": "/api/v1/dashboard/quick-actions",
      "notification-center": "/api/v1/dashboard/notifications",
      "resource-charts": "/api/v1/dashboard/resource-metrics",
      "security-dashboard": "/api/v1/dashboard/security-status",
    };

    return endpointMap[widgetType] || "/api/v1/dashboard/generic";
  }

  /**
   * Update widget data
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
   * Get widget data
   */
  getWidgetData(widgetId: string): any {
    return this.widgetData.get(widgetId);
  }

  /**
   * Get widget status
   */
  getWidgetStatus(widgetId: string): WidgetStatus {
    return this.widgetStatus.get(widgetId) || WidgetStatus.LOADING;
  }

  /**
   * Handle widget errors
   */
  private handleWidgetError(widgetId: string, error: any): void {
    console.error(`Widget ${widgetId} error:`, error);

    this.widgetStatus.set(widgetId, WidgetStatus.ERROR);

    // Update error metrics
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
   * Cache management
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
    // Implement LRU cache eviction
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
   * Update widget metrics
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
   * Event bus for widget communication
   */
  private emitEvent(event: WidgetEvent): void {
    this.eventBus.push(event);

    // Limit event history
    if (this.eventBus.length > this.maxEventHistory) {
      this.eventBus.splice(0, this.eventBus.length - this.maxEventHistory);
    }
  }

  /**
   * Subscribe to widget events
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

    // Store the handler for event subscription (simplified implementation)
    const handlerKey = `${eventType}-${Date.now()}`;
    this.eventHandlers.set(handlerKey, handler);

    const stopWatching = () => {
      this.eventHandlers.delete(handlerKey);
    };

    return stopWatching;
  }

  /**
   * Batch operations for performance
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
   * Widget analytics and metrics
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
   * Performance optimization
   */
  optimizePerformance(): void {
    // Clear old cache entries
    const now = Date.now();
    for (const [key, entry] of this.widgetCache.entries()) {
      if (now - entry.timestamp.getTime() > entry.ttl) {
        this.widgetCache.delete(key);
      }
    }

    // Cleanup old events
    if (this.eventBus.length > this.maxEventHistory) {
      this.eventBus.splice(0, this.eventBus.length - this.maxEventHistory);
    }
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    // Clear all intervals
    for (const interval of this.refreshIntervals.values()) {
      clearInterval(interval);
    }
    this.refreshIntervals.clear();

    // Clear performance observer
    if (this.performanceObserver) {
      this.performanceObserver.disconnect();
    }

    // Clear all data
    this.widgets.clear();
    this.widgetStatus.clear();
    this.widgetData.clear();
    this.widgetCache.clear();
    this.widgetMetrics.clear();
    this.eventBus.length = 0;
  }
}

// Singleton instance
export const widgetManager = new WidgetManagerService();

// Vue composable for widget management
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
