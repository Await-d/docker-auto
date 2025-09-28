/**
 * Enhanced WebSocket API for Docker container management
 */
import { WebSocketClient, createEventSubscription } from "@/utils/websocket";
import type {
  ResourceMetrics,
  ContainerStatusUpdate,
  ContainerStatsUpdate,
  ContainerLogUpdate
} from "@/types/container";

export interface DockerEventData {
  id: string;
  type: 'container' | 'image' | 'network' | 'volume';
  action: string;
  timestamp: number;
  attributes: Record<string, any>;
}

export interface TerminalData {
  sessionId: string;
  containerId: string;
  data: string | ArrayBuffer;
  timestamp: number;
}

export interface MonitoringData {
  containerId: string;
  metrics: ResourceMetrics;
  timestamp: number;
}

export class DockerWebSocketAPI {
  private client: WebSocketClient | null = null;
  private baseUrl: string;
  private token: string;
  private subscriptions = new Map<string, (() => void)[]>();

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl;
    this.token = token;
  }

  /**
   * Initialize WebSocket connection
   */
  async connect(): Promise<WebSocketClient> {
    if (this.client) {
      return this.client;
    }

    this.client = new WebSocketClient(this.baseUrl, this.token, {
      autoReconnect: true,
      reconnectInterval: 1000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
      enableMessageBatching: true,
      batchSize: 5,
      batchTimeout: 100,
    });

    await this.client.connect();
    return this.client;
  }

  /**
   * Disconnect WebSocket
   */
  disconnect(): void {
    if (this.client) {
      this.client.disconnect();
      this.client = null;
    }

    // Clear all subscriptions
    this.subscriptions.clear();
  }

  /**
   * Subscribe to container status updates
   */
  subscribeToContainerStatus(
    containerId: string | 'all',
    callback: (data: ContainerStatusUpdate) => void
  ): () => void {
    const topic = containerId === 'all'
      ? 'containers.status'
      : `containers.${containerId}.status`;

    return this.subscribe(topic, callback);
  }

  /**
   * Subscribe to container metrics/stats updates
   */
  subscribeToContainerStats(
    containerId: string,
    callback: (data: ContainerStatsUpdate) => void
  ): () => void {
    return this.subscribe(`containers.${containerId}.stats`, callback);
  }

  /**
   * Subscribe to container logs
   */
  subscribeToContainerLogs(
    containerId: string,
    callback: (data: ContainerLogUpdate) => void,
    options: {
      follow?: boolean;
      tail?: number;
      since?: Date;
      timestamps?: boolean;
    } = {}
  ): () => void {
    const topic = `containers.${containerId}.logs`;

    // Send subscription with options
    if (this.client) {
      this.client.subscribe(topic, callback as any);

      // Send log configuration
      this.client.sendMessage({
        type: 'subscribe',
        topic,
        data: {
          follow: options.follow !== false,
          tail: options.tail || 100,
          since: options.since?.toISOString(),
          timestamps: options.timestamps || true,
        },
      });
    }

    return this.subscribe(topic, callback);
  }

  /**
   * Subscribe to Docker events
   */
  subscribeToDockerEvents(
    callback: (event: DockerEventData) => void,
    filters?: {
      type?: ('container' | 'image' | 'network' | 'volume')[];
      action?: string[];
      containerId?: string[];
    }
  ): () => void {
    const topic = 'docker.events';

    // Send subscription with filters
    if (this.client) {
      this.client.subscribe(topic, callback as any);

      if (filters) {
        this.client.sendMessage({
          type: 'subscribe',
          topic,
          data: { filters },
        });
      }
    }

    return this.subscribe(topic, callback);
  }

  /**
   * Subscribe to system metrics
   */
  subscribeToSystemMetrics(
    callback: (metrics: any) => void,
    interval: number = 5000
  ): () => void {
    const topic = 'system.metrics';

    if (this.client) {
      this.client.subscribe(topic, callback as any);

      // Configure monitoring interval
      this.client.sendMessage({
        type: 'subscribe',
        topic,
        data: { interval },
      });
    }

    return this.subscribe(topic, callback);
  }

  /**
   * Start monitoring multiple containers
   */
  async startBulkMonitoring(
    containerIds: string[],
    options: {
      includeStats?: boolean;
      includeLogs?: boolean;
      includeEvents?: boolean;
      statsInterval?: number;
      logTail?: number;
    } = {}
  ): Promise<() => void> {
    await this.connect();

    const unsubscribeFunctions: (() => void)[] = [];

    for (const containerId of containerIds) {
      // Subscribe to status updates
      unsubscribeFunctions.push(
        this.subscribeToContainerStatus(containerId, (data) => {
          // This will be handled by the container store
          console.log(`Container ${containerId} status:`, data);
        })
      );

      // Subscribe to stats if requested
      if (options.includeStats) {
        unsubscribeFunctions.push(
          this.subscribeToContainerStats(containerId, (data) => {
            // This will be handled by the monitoring store
            console.log(`Container ${containerId} stats:`, data);
          })
        );
      }

      // Subscribe to logs if requested
      if (options.includeLogs) {
        unsubscribeFunctions.push(
          this.subscribeToContainerLogs(
            containerId,
            (data) => {
              console.log(`Container ${containerId} logs:`, data);
            },
            {
              tail: options.logTail || 50,
              follow: true,
              timestamps: true,
            }
          )
        );
      }
    }

    // Subscribe to Docker events if requested
    if (options.includeEvents) {
      unsubscribeFunctions.push(
        this.subscribeToDockerEvents((event) => {
          console.log('Docker event:', event);
        }, {
          type: ['container'],
          containerId: containerIds,
        })
      );
    }

    // Return function to unsubscribe from everything
    return () => {
      unsubscribeFunctions.forEach(fn => fn());
    };
  }

  /**
   * Generic subscribe method
   */
  private subscribe<T = any>(
    topic: string,
    callback: (data: T) => void
  ): () => void {
    if (!this.client) {
      throw new Error('WebSocket client not initialized');
    }

    const unsubscribe = createEventSubscription(this.client, topic, callback);

    // Track subscription for cleanup
    if (!this.subscriptions.has(topic)) {
      this.subscriptions.set(topic, []);
    }
    this.subscriptions.get(topic)!.push(unsubscribe);

    return unsubscribe;
  }

  /**
   * Subscribe to terminal data
   */
  subscribeToTerminalData(
    sessionId: string,
    callback: (data: TerminalData) => void
  ): () => void {
    const topic = `terminal.${sessionId}`;
    return this.subscribe(topic, callback);
  }

  /**
   * Send terminal input
   */
  sendTerminalInput(sessionId: string, data: string | ArrayBuffer): void {
    if (!this.client || !this.client.isConnected()) {
      throw new Error('WebSocket not connected');
    }

    this.client.sendMessage({
      type: 'terminal_input',
      data: {
        sessionId,
        data: typeof data === 'string' ? data : Array.from(new Uint8Array(data)),
      },
    });
  }

  /**
   * Resize terminal
   */
  resizeTerminal(sessionId: string, cols: number, rows: number): void {
    if (!this.client || !this.client.isConnected()) {
      throw new Error('WebSocket not connected');
    }

    this.client.sendMessage({
      type: 'terminal_resize',
      data: {
        sessionId,
        cols,
        rows,
      },
    });
  }

  /**
   * Request container stats
   */
  requestContainerStats(containerId: string): void {
    if (!this.client || !this.client.isConnected()) {
      throw new Error('WebSocket not connected');
    }

    this.client.sendMessage({
      type: 'request_stats',
      data: { containerId },
    });
  }

  /**
   * Start container monitoring
   */
  startContainerMonitoring(
    containerId: string,
    interval: number = 5000
  ): void {
    if (!this.client || !this.client.isConnected()) {
      throw new Error('WebSocket not connected');
    }

    this.client.sendMessage({
      type: 'start_monitoring',
      data: {
        containerId,
        interval,
      },
    });
  }

  /**
   * Stop container monitoring
   */
  stopContainerMonitoring(containerId: string): void {
    if (!this.client || !this.client.isConnected()) {
      throw new Error('WebSocket not connected');
    }

    this.client.sendMessage({
      type: 'stop_monitoring',
      data: { containerId },
    });
  }

  /**
   * Get WebSocket client instance
   */
  getClient(): WebSocketClient | null {
    return this.client;
  }

  /**
   * Get connection state
   */
  isConnected(): boolean {
    return this.client?.isConnected() || false;
  }

  /**
   * Get WebSocket statistics
   */
  getStats() {
    return this.client?.getStats() || null;
  }

  /**
   * Update authentication token
   */
  updateToken(token: string): void {
    this.token = token;
    if (this.client) {
      this.client.updateToken(token);
    }
  }
}

// Singleton instance
let dockerWSAPI: DockerWebSocketAPI | null = null;

export function createDockerWebSocketAPI(baseUrl: string, token: string): DockerWebSocketAPI {
  if (!dockerWSAPI) {
    dockerWSAPI = new DockerWebSocketAPI(baseUrl, token);
  } else {
    dockerWSAPI.updateToken(token);
  }
  return dockerWSAPI;
}

export function getDockerWebSocketAPI(): DockerWebSocketAPI | null {
  return dockerWSAPI;
}

// Vue composable for Docker WebSocket
export function useDockerWebSocket() {
  const connect = async (baseUrl: string, token: string) => {
    const api = createDockerWebSocketAPI(baseUrl, token);
    await api.connect();
    return api;
  };

  const disconnect = () => {
    if (dockerWSAPI) {
      dockerWSAPI.disconnect();
      dockerWSAPI = null;
    }
  };

  return {
    connect,
    disconnect,
    api: dockerWSAPI,
    isConnected: () => dockerWSAPI?.isConnected() || false,
  };
}

// DockerWebSocketAPI already exported above