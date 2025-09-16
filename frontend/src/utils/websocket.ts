// WebSocket client for real-time communication
import { ref, readonly, computed, onMounted, onUnmounted } from 'vue';

export interface ClientMessage {
  type: 'subscribe' | 'unsubscribe' | 'ping' | 'ack';
  topic?: string;
  data?: any;
  messageId?: string;
}

export interface ServerMessage {
  type: 'notification' | 'event' | 'pong' | 'error' | 'subscription_confirmed' | 'unsubscription_confirmed';
  topic: string;
  data: any;
  timestamp: number;
  messageId?: string;
}

export interface EventData {
  id: string;
  type: string;
  severity: 'info' | 'warning' | 'error' | 'success' | 'debug';
  source: string;
  title: string;
  message: string;
  data?: Record<string, any>;
  timestamp: number;
  tags?: string[];
  resource_id?: string;
  resource_type?: string;
}

export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error' | 'reconnecting';

export type EventCallback = (data: EventData) => void;
export type ErrorCallback = (error: string) => void;
export type StateChangeCallback = (state: ConnectionState) => void;

export interface WebSocketOptions {
  autoReconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
  messageTimeout?: number;
  enableMessageBatching?: boolean;
  batchSize?: number;
  batchTimeout?: number;
  enableCompression?: boolean;
  compressionThreshold?: number;
}

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private token: string;
  private baseUrl: string;
  private state: ConnectionState = 'disconnected';
  private subscriptions = new Map<string, Set<EventCallback>>();
  private messageQueue: ClientMessage[] = [];
  private reconnectAttempts = 0;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private pendingMessages = new Map<string, { resolve: Function; reject: Function; timeout: NodeJS.Timeout }>();

  // Event listeners
  private stateChangeListeners = new Set<StateChangeCallback>();
  private errorListeners = new Set<ErrorCallback>();

  // Performance enhancements
  private messageBuffer: ClientMessage[] = [];
  private batchTimer: NodeJS.Timeout | null = null;
  private performanceMetrics = {
    totalMessages: 0,
    batchedMessages: 0,
    compressedMessages: 0,
    averageLatency: 0,
    lastMessageTime: 0
  };

  // Configuration
  private options: Required<WebSocketOptions> = {
    autoReconnect: true,
    reconnectInterval: 1000,
    maxReconnectAttempts: 5,
    heartbeatInterval: 30000,
    messageTimeout: 5000,
    enableMessageBatching: true,
    batchSize: 10,
    batchTimeout: 100,
    enableCompression: true,
    compressionThreshold: 1024
  };

  constructor(baseUrl: string, token: string, options?: WebSocketOptions) {
    this.baseUrl = baseUrl;
    this.token = token;

    if (options) {
      this.options = { ...this.options, ...options };
    }
  }

  /**
   * Connect to the WebSocket server
   */
  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.state === 'connected' || this.state === 'connecting') {
        resolve();
        return;
      }

      this.setState('connecting');

      const wsUrl = `${this.baseUrl.replace('http', 'ws')}/api/ws?token=${encodeURIComponent(this.token)}`;

      try {
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.setState('connected');
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          this.processMessageQueue();
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onclose = (event) => {
          console.log('WebSocket closed:', event.code, event.reason);
          this.cleanup();

          if (event.code !== 1000 && this.options.autoReconnect) {
            this.scheduleReconnect();
          } else {
            this.setState('disconnected');
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.setState('error');
          this.notifyError('WebSocket connection error');
          reject(new Error('WebSocket connection failed'));
        };

      } catch (error) {
        this.setState('error');
        reject(error);
      }
    });
  }

  /**
   * Disconnect from the WebSocket server
   */
  disconnect(): void {
    this.options.autoReconnect = false;

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
    }

    this.cleanup();
    this.setState('disconnected');
  }

  /**
   * Subscribe to events for a specific topic
   */
  subscribe(topic: string, callback: EventCallback): void {
    if (!this.subscriptions.has(topic)) {
      this.subscriptions.set(topic, new Set());

      // Send subscription message to server
      this.sendMessage({
        type: 'subscribe',
        topic,
        messageId: this.generateMessageId(),
      });
    }

    this.subscriptions.get(topic)!.add(callback);
  }

  /**
   * Unsubscribe from events for a specific topic
   */
  unsubscribe(topic: string, callback?: EventCallback): void {
    const callbacks = this.subscriptions.get(topic);
    if (!callbacks) return;

    if (callback) {
      callbacks.delete(callback);

      // If no more callbacks for this topic, unsubscribe from server
      if (callbacks.size === 0) {
        this.subscriptions.delete(topic);
        this.sendMessage({
          type: 'unsubscribe',
          topic,
          messageId: this.generateMessageId(),
        });
      }
    } else {
      // Unsubscribe all callbacks for this topic
      this.subscriptions.delete(topic);
      this.sendMessage({
        type: 'unsubscribe',
        topic,
        messageId: this.generateMessageId(),
      });
    }
  }

  /**
   * Send a ping message to the server
   */
  ping(): Promise<void> {
    return this.sendMessageWithResponse({
      type: 'ping',
      messageId: this.generateMessageId(),
    });
  }

  /**
   * Add a state change listener
   */
  onStateChange(callback: StateChangeCallback): () => void {
    this.stateChangeListeners.add(callback);
    return () => this.stateChangeListeners.delete(callback);
  }

  /**
   * Add an error listener
   */
  onError(callback: ErrorCallback): () => void {
    this.errorListeners.add(callback);
    return () => this.errorListeners.delete(callback);
  }

  /**
   * Get the current connection state
   */
  getState(): ConnectionState {
    return this.state;
  }

  /**
   * Check if the client is connected
   */
  isConnected(): boolean {
    return this.state === 'connected';
  }

  /**
   * Update the authentication token
   */
  updateToken(token: string): void {
    this.token = token;

    // Reconnect with new token if currently connected
    if (this.isConnected()) {
      this.disconnect();
      setTimeout(() => this.connect(), 100);
    }
  }

  /**
   * Get comprehensive performance statistics
   */
  getStats(): {
    subscriptions: number;
    queuedMessages: number;
    reconnectAttempts: number;
    performance: typeof this.performanceMetrics;
    bufferSize: number;
  } {
    return {
      subscriptions: this.subscriptions.size,
      queuedMessages: this.messageQueue.length,
      reconnectAttempts: this.reconnectAttempts,
      performance: { ...this.performanceMetrics },
      bufferSize: this.messageBuffer.length
    };
  }

  /**
   * Get performance metrics
   */
  getPerformanceMetrics() {
    return {
      ...this.performanceMetrics,
      messagesPerSecond: this.calculateMessagesPerSecond(),
      compressionRatio: this.performanceMetrics.totalMessages > 0
        ? this.performanceMetrics.compressedMessages / this.performanceMetrics.totalMessages
        : 0,
      batchingRatio: this.performanceMetrics.totalMessages > 0
        ? this.performanceMetrics.batchedMessages / this.performanceMetrics.totalMessages
        : 0
    };
  }

  private calculateMessagesPerSecond(): number {
    if (this.performanceMetrics.lastMessageTime === 0) return 0;

    const timeDiff = (Date.now() - this.performanceMetrics.lastMessageTime) / 1000;
    return timeDiff > 0 ? this.performanceMetrics.totalMessages / timeDiff : 0;
  }

  // Private methods

  private setState(state: ConnectionState): void {
    if (this.state !== state) {
      this.state = state;
      this.stateChangeListeners.forEach(callback => callback(state));
    }
  }

  private notifyError(error: string): void {
    this.errorListeners.forEach(callback => callback(error));
  }

  private handleMessage(data: string | ArrayBuffer): void {
    try {
      let messageStr: string;

      // Handle both text and binary messages
      if (data instanceof ArrayBuffer) {
        // Decompress if needed
        messageStr = this.decompressMessage(data);
      } else {
        messageStr = data;
      }

      const message: ServerMessage = JSON.parse(messageStr);

      // Calculate latency if timestamp is available
      if (message.timestamp) {
        const latency = Date.now() - message.timestamp;
        this.updateLatencyMetrics(latency);
      }

      // Handle batched messages
      if (message.type === 'batch' && message.data?.messages) {
        const batchedMessages = message.data.messages as ServerMessage[];
        batchedMessages.forEach(batchedMessage => {
          this.processMessage(batchedMessage);
        });
      } else {
        this.processMessage(message);
      }

      this.performanceMetrics.totalMessages++;
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
      this.notifyError('Failed to parse server message');
    }
  }

  private processMessage(message: ServerMessage): void {
    switch (message.type) {
      case 'event':
        this.handleEvent(message);
        break;
      case 'pong':
        this.handlePong(message);
        break;
      case 'error':
        this.handleError(message);
        break;
      case 'subscription_confirmed':
      case 'unsubscription_confirmed':
        this.handleConfirmation(message);
        break;
      default:
        console.warn('Unknown message type:', message.type);
    }
  }

  private decompressMessage(data: ArrayBuffer): string {
    // Simple decompression simulation - in real implementation use pako or similar
    const decoder = new TextDecoder();
    return decoder.decode(data);
  }

  private updateLatencyMetrics(latency: number): void {
    if (this.performanceMetrics.averageLatency === 0) {
      this.performanceMetrics.averageLatency = latency;
    } else {
      // Exponential moving average
      this.performanceMetrics.averageLatency =
        this.performanceMetrics.averageLatency * 0.9 + latency * 0.1;
    }
  }

  private handleEvent(message: ServerMessage): void {
    const eventData = message.data as EventData;
    const topic = message.topic;

    const callbacks = this.subscriptions.get(topic);
    if (callbacks) {
      callbacks.forEach(callback => {
        try {
          callback(eventData);
        } catch (error) {
          console.error('Error in event callback:', error);
        }
      });
    }

    // Also notify 'all' subscribers
    const allCallbacks = this.subscriptions.get('all');
    if (allCallbacks) {
      allCallbacks.forEach(callback => {
        try {
          callback(eventData);
        } catch (error) {
          console.error('Error in event callback:', error);
        }
      });
    }
  }

  private handlePong(message: ServerMessage): void {
    if (message.messageId) {
      this.resolvePendingMessage(message.messageId);
    }
  }

  private handleError(message: ServerMessage): void {
    const error = message.data?.error || 'Server error';
    console.error('Server error:', error);
    this.notifyError(error);

    if (message.messageId) {
      this.rejectPendingMessage(message.messageId, error);
    }
  }

  private handleConfirmation(message: ServerMessage): void {
    if (message.messageId) {
      this.resolvePendingMessage(message.messageId);
    }
  }

  private sendMessage(message: ClientMessage): void {
    // Add performance tracking
    message.timestamp = Date.now();

    if (this.options.enableMessageBatching && this.shouldBatchMessage(message)) {
      this.addToBatch(message);
    } else {
      this.sendImmediately(message);
    }
  }

  private sendImmediately(message: ClientMessage): void {
    if (this.isConnected() && this.ws) {
      try {
        const messageStr = JSON.stringify(message);
        let data: string | ArrayBuffer = messageStr;

        // Apply compression if enabled and message is large enough
        if (this.options.enableCompression && messageStr.length > this.options.compressionThreshold) {
          data = this.compressMessage(messageStr);
          this.performanceMetrics.compressedMessages++;
        }

        this.ws.send(data);
        this.performanceMetrics.totalMessages++;
        this.performanceMetrics.lastMessageTime = Date.now();
      } catch (error) {
        console.error('Failed to send message:', error);
        this.queueMessage(message);
      }
    } else {
      this.queueMessage(message);
    }
  }

  private shouldBatchMessage(message: ClientMessage): boolean {
    // Don't batch critical messages like ping/pong
    return message.type !== 'ping' && message.type !== 'ack';
  }

  private addToBatch(message: ClientMessage): void {
    this.messageBuffer.push(message);

    // Send batch if it reaches max size
    if (this.messageBuffer.length >= this.options.batchSize) {
      this.sendBatch();
    } else if (!this.batchTimer) {
      // Set timer to send batch after timeout
      this.batchTimer = setTimeout(() => {
        this.sendBatch();
      }, this.options.batchTimeout);
    }
  }

  private sendBatch(): void {
    if (this.messageBuffer.length === 0) return;

    const batchMessage: ClientMessage = {
      type: 'batch',
      data: {
        messages: this.messageBuffer.splice(0, this.options.batchSize),
        timestamp: Date.now()
      },
      messageId: this.generateMessageId()
    };

    this.sendImmediately(batchMessage);
    this.performanceMetrics.batchedMessages += batchMessage.data.messages.length;

    // Clear batch timer
    if (this.batchTimer) {
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }

    // Schedule next batch if buffer still has messages
    if (this.messageBuffer.length > 0) {
      this.batchTimer = setTimeout(() => {
        this.sendBatch();
      }, this.options.batchTimeout);
    }
  }

  private compressMessage(message: string): ArrayBuffer {
    // Simple compression simulation - in real implementation use pako or similar
    const encoder = new TextEncoder();
    const data = encoder.encode(message);
    // This would be actual compression in production
    return data.buffer;
  }

  private sendMessageWithResponse(message: ClientMessage): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!message.messageId) {
        message.messageId = this.generateMessageId();
      }

      const timeout = setTimeout(() => {
        this.pendingMessages.delete(message.messageId!);
        reject(new Error('Message timeout'));
      }, this.options.messageTimeout);

      this.pendingMessages.set(message.messageId, { resolve, reject, timeout });
      this.sendMessage(message);
    });
  }

  private queueMessage(message: ClientMessage): void {
    this.messageQueue.push(message);
  }

  private processMessageQueue(): void {
    while (this.messageQueue.length > 0 && this.isConnected()) {
      const message = this.messageQueue.shift()!;
      this.sendMessage(message);
    }
  }

  private resolvePendingMessage(messageId: string): void {
    const pending = this.pendingMessages.get(messageId);
    if (pending) {
      clearTimeout(pending.timeout);
      pending.resolve();
      this.pendingMessages.delete(messageId);
    }
  }

  private rejectPendingMessage(messageId: string, error: string): void {
    const pending = this.pendingMessages.get(messageId);
    if (pending) {
      clearTimeout(pending.timeout);
      pending.reject(new Error(error));
      this.pendingMessages.delete(messageId);
    }
  }

  private startHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
    }

    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.ping().catch(error => {
          console.error('Heartbeat failed:', error);
          this.setState('error');
        });
      }
    }, this.options.heartbeatInterval);
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      console.log('Max reconnect attempts reached');
      this.setState('disconnected');
      return;
    }

    this.setState('reconnecting');

    const delay = Math.min(
      this.options.reconnectInterval * Math.pow(2, this.reconnectAttempts),
      30000
    );

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++;
      console.log(`Reconnecting... (attempt ${this.reconnectAttempts})`);

      this.connect().catch(error => {
        console.error('Reconnect failed:', error);
        this.scheduleReconnect();
      });
    }, delay);
  }

  private cleanup(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }

    if (this.batchTimer) {
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }

    // Send any remaining batched messages
    if (this.messageBuffer.length > 0) {
      this.sendBatch();
    }

    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws = null;
    }

    // Reject all pending messages
    this.pendingMessages.forEach(({ reject, timeout }) => {
      clearTimeout(timeout);
      reject(new Error('Connection closed'));
    });
    this.pendingMessages.clear();
  }

  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Vue 3 composable for WebSocket
export function useWebSocket(baseUrl: string, token: string, options?: WebSocketOptions) {
  const client = new WebSocketClient(baseUrl, token, options);

  const state = ref<ConnectionState>('disconnected');
  const error = ref<string | null>(null);

  // Update reactive state
  client.onStateChange((newState) => {
    state.value = newState;
  });

  client.onError((errorMessage) => {
    error.value = errorMessage;
  });

  // Connect on mount
  onMounted(() => {
    client.connect().catch(err => {
      console.error('Failed to connect WebSocket:', err);
    });
  });

  // Disconnect on unmount
  onUnmounted(() => {
    client.disconnect();
  });

  return {
    client,
    state: readonly(state),
    error: readonly(error),
    isConnected: computed(() => state.value === 'connected'),
    subscribe: client.subscribe.bind(client),
    unsubscribe: client.unsubscribe.bind(client),
    ping: client.ping.bind(client),
  };
}

// Helper function to create a typed event subscription
export function createEventSubscription<T = any>(
  client: WebSocketClient,
  topic: string,
  callback: (data: T) => void
): () => void {
  client.subscribe(topic, callback as EventCallback);
  return () => client.unsubscribe(topic, callback as EventCallback);
}

// Default export
export default WebSocketClient;