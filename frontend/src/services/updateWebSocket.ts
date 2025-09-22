/**
 * 更新WebSocket服务，用于实时更新
 */
import type { UpdateWebSocketMessage } from "@/types/updates";

export interface UpdateWebSocketCallbacks {
  onUpdateProgress?: (data: any) => void;
  onUpdateCompleted?: (data: any) => void;
  onUpdateFailed?: (data: any) => void;
  onUpdateAvailable?: (data: any) => void;
  onUpdateNotification?: (data: any) => void;
  onConnected?: () => void;
  onDisconnected?: () => void;
  onError?: (error: any) => void;
}

export class UpdateWebSocketService {
  private ws: WebSocket | null = null;
  private url: string;
  private callbacks: UpdateWebSocketCallbacks = {};
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // Start with 1 second
  private reconnectTimer: NodeJS.Timeout | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private isIntentionallyClosed = false;
  private subscriptions = new Set<string>();

  constructor(url?: string) {
    // Use backend server address for WebSocket connection
    const wsHost = import.meta.env.DEV ? 'localhost:8080' : window.location.host;
    this.url =
      url ||
      `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${wsHost}/ws/updates`;
  }

  /**
   * 连接到WebSocket
   */
  connect(callbacks?: UpdateWebSocketCallbacks): Promise<void> {
    if (callbacks) {
      this.callbacks = { ...this.callbacks, ...callbacks };
    }

    return new Promise(async (resolve, reject) => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.isIntentionallyClosed = false;

      try {
        const { TokenManager } = await import('@/utils/auth');
        const token = TokenManager.getAccessToken();
        const wsUrl = token ? `${this.url}?token=${token}` : this.url;

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log("更新WebSocket已连接");
          this.reconnectAttempts = 0;
          this.reconnectDelay = 1000;
          this.startHeartbeat();
          this.resubscribe();
          this.callbacks.onConnected?.();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: UpdateWebSocketMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.error(
              "解析WebSocket消息失败:",
              error,
              event.data,
            );
          }
        };

        this.ws.onclose = (event) => {
          console.log(
            "更新WebSocket已断开连接:",
            event.code,
            event.reason,
          );
          this.stopHeartbeat();
          this.callbacks.onDisconnected?.();

          if (
            !this.isIntentionallyClosed &&
            this.reconnectAttempts < this.maxReconnectAttempts
          ) {
            this.scheduleReconnect();
          }
        };

        this.ws.onerror = (error) => {
          console.error("更新WebSocket错误:", error);
          this.callbacks.onError?.(error);
          reject(error);
        };
      } catch (error) {
        console.error("创建 WebSocket 连接失败:", error);
        reject(error);
      }
    });
  }

  /**
   * 从 WebSocket 断开连接
   */
  disconnect(): void {
    this.isIntentionallyClosed = true;
    this.stopHeartbeat();
    this.clearReconnectTimer();

    if (this.ws) {
      this.ws.close(1000, "客户端断开连接");
      this.ws = null;
    }
  }

  /**
   * 订阅特定更新事件
   */
  subscribe(subscription: string): void {
    this.subscriptions.add(subscription);

    if (this.isConnected()) {
      this.send({
        type: "subscribe",
        data: { subscription },
      });
    }
  }

  /**
   * 取消订阅特定更新事件
   */
  unsubscribe(subscription: string): void {
    this.subscriptions.delete(subscription);

    if (this.isConnected()) {
      this.send({
        type: "unsubscribe",
        data: { subscription },
      });
    }
  }

  /**
   * 订阅容器特定更新
   */
  subscribeToContainer(containerId: string): void {
    this.subscribe(`container:${containerId}`);
  }

  /**
   * 取消订阅容器特定更新
   */
  unsubscribeFromContainer(containerId: string): void {
    this.unsubscribe(`container:${containerId}`);
  }

  /**
   * 订阅更新操作
   */
  subscribeToUpdate(updateId: string): void {
    this.subscribe(`update:${updateId}`);
  }

  /**
   * 取消订阅更新操作
   */
  unsubscribeFromUpdate(updateId: string): void {
    this.unsubscribe(`update:${updateId}`);
  }

  /**
   * 订阅批量更新操作
   */
  subscribeToBulkUpdate(operationId: string): void {
    this.subscribe(`bulk:${operationId}`);
  }

  /**
   * 取消订阅批量更新操作
   */
  unsubscribeFromBulkUpdate(operationId: string): void {
    this.unsubscribe(`bulk:${operationId}`);
  }

  /**
   * 订阅所有更新
   */
  subscribeToAllUpdates(): void {
    this.subscribe("updates:all");
  }

  /**
   * 仅订阅安全更新
   */
  subscribeToSecurityUpdates(): void {
    this.subscribe("updates:security");
  }

  /**
   * 订阅更新通知
   */
  subscribeToNotifications(): void {
    this.subscribe("notifications");
  }

  /**
   * 检查 WebSocket 是否已连接
   */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * 获取连接状态
   */
  getState(): "connecting" | "connected" | "disconnected" | "error" {
    if (!this.ws) return "disconnected";

    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return "connecting";
      case WebSocket.OPEN:
        return "connected";
      case WebSocket.CLOSING:
      case WebSocket.CLOSED:
        return "disconnected";
      default:
        return "error";
    }
  }

  /**
   * 向服务器发送消息
   */
  private send(message: any): void {
    if (this.isConnected()) {
      this.ws!.send(JSON.stringify(message));
    }
  }

  /**
   * 处理传入的 WebSocket 消息
   */
  private handleMessage(message: UpdateWebSocketMessage): void {
    const { type, data } = message;

    switch (type) {
      case "update_progress":
        this.callbacks.onUpdateProgress?.(data);
        break;

      case "update_completed":
        this.callbacks.onUpdateCompleted?.(data);
        break;

      case "update_failed":
        this.callbacks.onUpdateFailed?.(data);
        break;

      case "update_available":
        this.callbacks.onUpdateAvailable?.(data);
        break;

      case "update_notification":
        this.callbacks.onUpdateNotification?.(data);
        break;

      default:
        console.warn("未知的 WebSocket 消息类型:", type);
    }
  }

  /**
   * 安排重连尝试
   */
  private scheduleReconnect(): void {
    this.clearReconnectTimer();

    const delay = Math.min(
      this.reconnectDelay * Math.pow(2, this.reconnectAttempts),
      30000,
    );

    console.log(
      `安排 WebSocket 重连，${delay}ms 后进行（第 ${this.reconnectAttempts + 1} 次尝试）`,
    );

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++;
      this.connect().catch(() => {
        // 重连失败，将自动重试
      });
    }, delay);
  }

  /**
   * 清除重连定时器
   */
  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /**
   * 启动心跳以保持连接活跃
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();

    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.send({ type: "ping" });
      }
    }, 30000); // 30秒
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  /**
   * 重连后重新订阅所有订阅
   */
  private resubscribe(): void {
    for (const subscription of this.subscriptions) {
      this.send({
        type: "subscribe",
        data: { subscription },
      });
    }
  }

  /**
   * 更新回调函数
   */
  setCallbacks(callbacks: UpdateWebSocketCallbacks): void {
    this.callbacks = { ...this.callbacks, ...callbacks };
  }

  /**
   * 获取当前订阅
   */
  getSubscriptions(): string[] {
    return Array.from(this.subscriptions);
  }

  /**
   * 清除所有订阅
   */
  clearSubscriptions(): void {
    for (const subscription of this.subscriptions) {
      this.unsubscribe(subscription);
    }
    this.subscriptions.clear();
  }
}

// 导出单例实例
export const updateWebSocket = new UpdateWebSocketService();

// 为 Vue 组合式 API 导出
export function useUpdateWebSocket() {
  return {
    updateWebSocket,
    connect: updateWebSocket.connect.bind(updateWebSocket),
    disconnect: updateWebSocket.disconnect.bind(updateWebSocket),
    subscribe: updateWebSocket.subscribe.bind(updateWebSocket),
    unsubscribe: updateWebSocket.unsubscribe.bind(updateWebSocket),
    subscribeToContainer:
      updateWebSocket.subscribeToContainer.bind(updateWebSocket),
    unsubscribeFromContainer:
      updateWebSocket.unsubscribeFromContainer.bind(updateWebSocket),
    subscribeToUpdate: updateWebSocket.subscribeToUpdate.bind(updateWebSocket),
    unsubscribeFromUpdate:
      updateWebSocket.unsubscribeFromUpdate.bind(updateWebSocket),
    subscribeToBulkUpdate:
      updateWebSocket.subscribeToBulkUpdate.bind(updateWebSocket),
    unsubscribeFromBulkUpdate:
      updateWebSocket.unsubscribeFromBulkUpdate.bind(updateWebSocket),
    subscribeToAllUpdates:
      updateWebSocket.subscribeToAllUpdates.bind(updateWebSocket),
    subscribeToSecurityUpdates:
      updateWebSocket.subscribeToSecurityUpdates.bind(updateWebSocket),
    subscribeToNotifications:
      updateWebSocket.subscribeToNotifications.bind(updateWebSocket),
    isConnected: updateWebSocket.isConnected.bind(updateWebSocket),
    getState: updateWebSocket.getState.bind(updateWebSocket),
  };
}
