/**
 * 容器专用WebSocket服务，用于实时更新
 */
import { ref } from "vue";
import { ElNotification } from "element-plus";
import { WebSocketClient, type EventData } from "@/utils/websocket";
import { useContainerStore } from "@/store/containers";
import { useAuthStore } from "@/store/auth";
import type {
  ContainerStatusUpdate,
  ContainerStatsUpdate,
  ContainerLogUpdate,
} from "@/types/container";

export class ContainerWebSocketService {
  private client: WebSocketClient | null = null;
  private containerStore = useContainerStore();
  private authStore = useAuthStore();
  private isInitialized = false;

  // 连接状态
  public state = ref<
    "disconnected" | "connecting" | "connected" | "reconnecting" | "error"
  >("disconnected");
  public lastError = ref<string | null>(null);

  // 订阅跟踪
  private subscriptions = new Set<string>();

  constructor() {
    // 延迟初始化，等待认证完成
    this.watchAuthState();
  }

  private watchAuthState() {
    // 监听认证状态变化
    this.authStore.$subscribe((mutation, state) => {
      if (state.isAuthenticated && state.token && !this.isInitialized) {
        this.initialize();
      } else if (!state.isAuthenticated && this.isInitialized) {
        this.disconnect();
      }
    });

    // 如果已经认证，立即初始化
    if (this.authStore.isAuthenticated && this.authStore.token) {
      this.initialize();
    }
  }

  private async initialize() {
    if (this.isInitialized) return;

    try {
      const baseUrl =
        import.meta.env.VITE_API_BASE_URL || window.location.origin;
      const token = this.authStore.token;

      console.log("正在初始化WebSocket连接，认证令牌:", token ? "存在" : "缺失");

      this.client = new WebSocketClient(baseUrl, token, {
        autoReconnect: true,
        reconnectInterval: 1000,
        maxReconnectAttempts: 5,
        heartbeatInterval: 30000,
      });

      // Set up state tracking
      this.client.onStateChange((newState) => {
        this.state.value = newState;
        this.containerStore.setWebSocketConnected(newState === "connected");
      });

      this.client.onError((error) => {
        this.lastError.value = error;
        console.error("Container WebSocket error:", error);
      });

      await this.client.connect();
      this.isInitialized = true;

      console.log("容器WebSocket服务已初始化");
    } catch (error) {
      console.error("容器WebSocket服务初始化失败:", error);
      this.lastError.value =
        error instanceof Error ? error.message : "连接失败";
    }
  }

  /**
   * 订阅容器状态更新
   */
  subscribeToContainerStatus() {
    if (!this.client) return;

    const topic = "containers.status";
    if (this.subscriptions.has(topic)) return;

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const update = event.data as ContainerStatusUpdate;
        this.containerStore.handleWebSocketMessage({
          type: "container_status",
          data: update,
          timestamp: new Date(event.timestamp),
        });

        // 为重要状态变化显示通知
        if (update.status === "running" || update.status === "exited") {
          ElNotification({
            title: "容器状态更新",
            message: `容器 ${update.container} 现在状态为 ${update.status}`,
            type: update.status === "running" ? "success" : "warning",
            duration: 3000,
          });
        }
      } catch (error) {
        console.error("处理容器状态更新时出错:", error);
      }
    });

    this.subscriptions.add(topic);
    console.log("已订阅容器状态更新");
  }

  /**
   * 订阅容器统计更新
   */
  subscribeToContainerStats() {
    if (!this.client) return;

    const topic = "containers.stats";
    if (this.subscriptions.has(topic)) return;

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const update = event.data as ContainerStatsUpdate;
        this.containerStore.handleWebSocketMessage({
          type: "container_stats",
          data: update,
          timestamp: new Date(event.timestamp),
        });
      } catch (error) {
        console.error("处理容器统计更新时出错:", error);
      }
    });

    this.subscriptions.add(topic);
    console.log("已订阅容器统计更新");
  }

  /**
   * 订阅容器日志更新
   */
  subscribeToContainerLogs(containerId?: string) {
    if (!this.client) return;

    const topic = containerId
      ? `containers.logs.${containerId}`
      : "containers.logs";
    if (this.subscriptions.has(topic)) return;

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const update = event.data as ContainerLogUpdate;
        this.containerStore.handleWebSocketMessage({
          type: "container_logs",
          data: update,
          timestamp: new Date(event.timestamp),
        });
      } catch (error) {
        console.error("处理容器日志更新时出错:", error);
      }
    });

    this.subscriptions.add(topic);
    console.log(`已订阅容器日志: ${topic}`);
  }

  /**
   * 订阅容器事件（创建、删除等）
   */
  subscribeToContainerEvents() {
    if (!this.client) return;

    const topic = "containers.events";
    if (this.subscriptions.has(topic)) return;

    this.client.subscribe(topic, (event: EventData) => {
      try {
        this.containerStore.handleWebSocketMessage({
          type: "container_event",
          data: event.data,
          timestamp: new Date(event.timestamp),
        });

        // 为重要事件显示通知
        if (event.data) {
          const eventType = event.data.action;
          if (["create", "start", "stop", "remove"].includes(eventType)) {
            const severity = eventType === "remove" ? "warning" : "info";
            ElNotification({
            title: "容器事件",
            message: `容器 ${event.data.container} 已${eventType === "create" ? "创建" : eventType === "start" ? "启动" : eventType === "stop" ? "停止" : "移除"}`,
              type: severity,
              duration: 3000,
            });
          }
        }
      } catch (error) {
        console.error("处理容器事件时出错:", error);
      }
    });

    this.subscriptions.add(topic);
    console.log("已订阅容器事件");
  }

  /**
   * 订阅更新通知
   */
  subscribeToUpdateNotifications() {
    if (!this.client) return;

    const topic = "containers.updates";
    if (this.subscriptions.has(topic)) return;

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const updateData = event.data;
        if (!updateData) return;

        // 在存储中添加可用更新
        this.containerStore.availableUpdates.push({
          container: updateData.container,
          currentVersion: updateData.currentVersion,
          availableVersion: updateData.availableVersion,
          releaseNotes: updateData.releaseNotes,
          publishedAt: new Date(updateData.publishedAt),
          size: updateData.size,
          critical: updateData.critical || false,
        });

        // 显示通知
        ElNotification({
          title: "有可用更新",
          message: `${updateData.container}: ${updateData.currentVersion} → ${updateData.availableVersion}`,
          type: updateData.critical ? "warning" : "info",
          duration: 5000,
        });
      } catch (error) {
        console.error("处理更新通知时出错:", error);
      }
    });

    this.subscriptions.add(topic);
    console.log("已订阅更新通知");
  }

  /**
   * 订阅系统警报
   */
  subscribeToSystemAlerts() {
    if (!this.client) return;

    const topic = "system.alerts";
    if (this.subscriptions.has(topic)) return;

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const alert = event.data;
        if (!alert) return;

        // 显示系统全局通知
        ElNotification({
          title: alert.title || "系统警报",
          message: alert.message,
          type:
            alert.severity === "error"
              ? "error"
              : alert.severity === "warning"
                ? "warning"
                : "info",
          duration: alert.severity === "error" ? 0 : 5000, // 错误保持直到被关闭
        });
      } catch (error) {
        console.error("处理系统警报时出错:", error);
      }
    });

    this.subscriptions.add(topic);
    console.log("已订阅系统警报");
  }

  /**
   * 订阅所有容器相关更新
   */
  subscribeToAll() {
    this.subscribeToContainerStatus();
    this.subscribeToContainerStats();
    this.subscribeToContainerLogs();
    this.subscribeToContainerEvents();
    this.subscribeToUpdateNotifications();
    this.subscribeToSystemAlerts();
  }

  /**
   * 取消订阅特定主题
   */
  unsubscribe(topic: string) {
    if (!this.client || !this.subscriptions.has(topic)) return;

    this.client.unsubscribe(topic);
    this.subscriptions.delete(topic);
    console.log(`已取消订阅 ${topic}`);
  }

  /**
   * 取消订阅所有主题
   */
  unsubscribeAll() {
    if (!this.client) return;

    this.subscriptions.forEach((topic) => {
      this.client!.unsubscribe(topic);
    });
    this.subscriptions.clear();
    console.log("已取消订阅所有主题");
  }

  /**
   * 断开WebSocket连接
   */
  disconnect() {
    if (!this.client) return;

    this.unsubscribeAll();
    this.client.disconnect();
    this.client = null;
    this.isInitialized = false;
    this.state.value = "disconnected";
    console.log("WebSocket连接已断开");
  }

  /**
   * 重新连接WebSocket连接
   */
  async reconnect() {
    if (!this.client) {
      await this.initialize();
      return;
    }

    try {
      this.client.disconnect();
      await this.client.connect();

      // 重新订阅所有主题
      const currentSubscriptions = Array.from(this.subscriptions);
      this.subscriptions.clear();

      currentSubscriptions.forEach((topic) => {
        if (topic === "containers.status") this.subscribeToContainerStatus();
        else if (topic === "containers.stats") this.subscribeToContainerStats();
        else if (topic.startsWith("containers.logs"))
          this.subscribeToContainerLogs();
        else if (topic === "containers.events")
          this.subscribeToContainerEvents();
        else if (topic === "containers.updates")
          this.subscribeToUpdateNotifications();
        else if (topic === "system.alerts") this.subscribeToSystemAlerts();
      });
    } catch (error) {
      console.error("WebSocket重新连接失败:", error);
      this.lastError.value =
        error instanceof Error ? error.message : "重新连接失败";
    }
  }

  /**
   * 更新认证令牌
   */
  updateToken(token: string) {
    if (!this.client) return;

    this.client.updateToken(token);
  }

  /**
   * 获取连接统计
   */
  getStats() {
    if (!this.client) {
      return {
        subscriptions: 0,
        queuedMessages: 0,
        reconnectAttempts: 0,
        activeTopics: [],
      };
    }

    const stats = this.client.getStats();
    return {
      ...stats,
      activeTopics: Array.from(this.subscriptions),
    };
  }

  /**
   * 断开连接并清理
   */
  disconnect() {
    this.unsubscribeAll();

    if (this.client) {
      this.client.disconnect();
      this.client = null;
    }

    this.state.value = "disconnected";
    this.isInitialized = false;
    console.log("容器WebSocket服务已断开连接");
  }

  /**
   * 检查是否已连接
   */
  get isConnected() {
    return this.state.value === "connected";
  }

  /**
   * 获取活跃订阅
   */
  get activeSubscriptions() {
    return Array.from(this.subscriptions);
  }
}

// 创建单例实例
export const containerWebSocketService = new ContainerWebSocketService();

// Vue组合式函数，方便在组件中使用
export function useContainerWebSocket() {
  return {
    service: containerWebSocketService,
    state: containerWebSocketService.state,
    lastError: containerWebSocketService.lastError,
    isConnected: containerWebSocketService.isConnected,
    subscribeToAll: () => containerWebSocketService.subscribeToAll(),
    subscribeToContainerLogs: (containerId?: string) =>
      containerWebSocketService.subscribeToContainerLogs(containerId),
    unsubscribe: (topic: string) =>
      containerWebSocketService.unsubscribe(topic),
    reconnect: () => containerWebSocketService.reconnect(),
    getStats: () => containerWebSocketService.getStats(),
  };
}

export default containerWebSocketService;
