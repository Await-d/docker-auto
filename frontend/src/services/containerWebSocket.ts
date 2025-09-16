/**
 * Container-specific WebSocket service for real-time updates
 */
import { reactive, ref } from 'vue'
import { ElNotification } from 'element-plus'
import { WebSocketClient, type EventData } from '@/utils/websocket'
import { useContainerStore } from '@/store/containers'
import { useAuthStore } from '@/store/auth'
import type {
  ContainerStatusUpdate,
  ContainerStatsUpdate,
  ContainerLogUpdate,
  WebSocketMessage
} from '@/types/container'

export class ContainerWebSocketService {
  private client: WebSocketClient | null = null
  private containerStore = useContainerStore()
  private authStore = useAuthStore()
  private isInitialized = false

  // Connection state
  public state = ref<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected')
  public lastError = ref<string | null>(null)

  // Subscription tracking
  private subscriptions = new Set<string>()

  constructor() {
    // Initialize when auth state is available
    this.initialize()
  }

  private async initialize() {
    if (this.isInitialized) return

    try {
      const baseUrl = import.meta.env.VITE_API_BASE_URL || window.location.origin
      const token = this.authStore.token

      if (!token) {
        console.warn('No auth token available for WebSocket connection')
        return
      }

      this.client = new WebSocketClient(baseUrl, token, {
        autoReconnect: true,
        reconnectInterval: 1000,
        maxReconnectAttempts: 5,
        heartbeatInterval: 30000
      })

      // Set up state tracking
      this.client.onStateChange((newState) => {
        this.state.value = newState
        this.containerStore.setWebSocketConnected(newState === 'connected')
      })

      this.client.onError((error) => {
        this.lastError.value = error
        console.error('Container WebSocket error:', error)
      })

      await this.client.connect()
      this.isInitialized = true

      console.log('Container WebSocket service initialized')
    } catch (error) {
      console.error('Failed to initialize Container WebSocket service:', error)
      this.lastError.value = error instanceof Error ? error.message : 'Connection failed'
    }
  }

  /**
   * Subscribe to container status updates
   */
  subscribeToContainerStatus() {
    if (!this.client) return

    const topic = 'containers.status'
    if (this.subscriptions.has(topic)) return

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const update = event.data as ContainerStatusUpdate
        this.containerStore.handleWebSocketMessage({
          type: 'container_status',
          data: update,
          timestamp: new Date(event.timestamp)
        })

        // Show notification for important status changes
        if (update.status === 'running' || update.status === 'exited') {
          ElNotification({
            title: 'Container Status Update',
            message: `Container ${update.container} is now ${update.status}`,
            type: update.status === 'running' ? 'success' : 'warning',
            duration: 3000
          })
        }
      } catch (error) {
        console.error('Error handling container status update:', error)
      }
    })

    this.subscriptions.add(topic)
    console.log('Subscribed to container status updates')
  }

  /**
   * Subscribe to container stats updates
   */
  subscribeToContainerStats() {
    if (!this.client) return

    const topic = 'containers.stats'
    if (this.subscriptions.has(topic)) return

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const update = event.data as ContainerStatsUpdate
        this.containerStore.handleWebSocketMessage({
          type: 'container_stats',
          data: update,
          timestamp: new Date(event.timestamp)
        })
      } catch (error) {
        console.error('Error handling container stats update:', error)
      }
    })

    this.subscriptions.add(topic)
    console.log('Subscribed to container stats updates')
  }

  /**
   * Subscribe to container log updates
   */
  subscribeToContainerLogs(containerId?: string) {
    if (!this.client) return

    const topic = containerId ? `containers.logs.${containerId}` : 'containers.logs'
    if (this.subscriptions.has(topic)) return

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const update = event.data as ContainerLogUpdate
        this.containerStore.handleWebSocketMessage({
          type: 'container_logs',
          data: update,
          timestamp: new Date(event.timestamp)
        })
      } catch (error) {
        console.error('Error handling container logs update:', error)
      }
    })

    this.subscriptions.add(topic)
    console.log(`Subscribed to container logs: ${topic}`)
  }

  /**
   * Subscribe to container events (creation, deletion, etc.)
   */
  subscribeToContainerEvents() {
    if (!this.client) return

    const topic = 'containers.events'
    if (this.subscriptions.has(topic)) return

    this.client.subscribe(topic, (event: EventData) => {
      try {
        this.containerStore.handleWebSocketMessage({
          type: 'container_event',
          data: event.data,
          timestamp: new Date(event.timestamp)
        })

        // Show notifications for important events
        const eventType = event.data.action
        if (['create', 'start', 'stop', 'remove'].includes(eventType)) {
          const severity = eventType === 'remove' ? 'warning' : 'info'
          ElNotification({
            title: 'Container Event',
            message: `Container ${event.data.container} ${eventType}d`,
            type: severity,
            duration: 3000
          })
        }
      } catch (error) {
        console.error('Error handling container event:', error)
      }
    })

    this.subscriptions.add(topic)
    console.log('Subscribed to container events')
  }

  /**
   * Subscribe to update notifications
   */
  subscribeToUpdateNotifications() {
    if (!this.client) return

    const topic = 'containers.updates'
    if (this.subscriptions.has(topic)) return

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const updateData = event.data

        // Add to available updates in store
        this.containerStore.availableUpdates.push({
          container: updateData.container,
          currentVersion: updateData.currentVersion,
          availableVersion: updateData.availableVersion,
          releaseNotes: updateData.releaseNotes,
          publishedAt: new Date(updateData.publishedAt),
          size: updateData.size,
          critical: updateData.critical || false
        })

        // Show notification
        ElNotification({
          title: 'Update Available',
          message: `${updateData.container}: ${updateData.currentVersion} → ${updateData.availableVersion}`,
          type: updateData.critical ? 'warning' : 'info',
          duration: 5000
        })
      } catch (error) {
        console.error('Error handling update notification:', error)
      }
    })

    this.subscriptions.add(topic)
    console.log('Subscribed to update notifications')
  }

  /**
   * Subscribe to system alerts
   */
  subscribeToSystemAlerts() {
    if (!this.client) return

    const topic = 'system.alerts'
    if (this.subscriptions.has(topic)) return

    this.client.subscribe(topic, (event: EventData) => {
      try {
        const alert = event.data

        // Show system-wide notifications
        ElNotification({
          title: alert.title || 'System Alert',
          message: alert.message,
          type: alert.severity === 'error' ? 'error' :
                alert.severity === 'warning' ? 'warning' : 'info',
          duration: alert.severity === 'error' ? 0 : 5000 // Errors stay until dismissed
        })
      } catch (error) {
        console.error('Error handling system alert:', error)
      }
    })

    this.subscriptions.add(topic)
    console.log('Subscribed to system alerts')
  }

  /**
   * Subscribe to all container-related updates
   */
  subscribeToAll() {
    this.subscribeToContainerStatus()
    this.subscribeToContainerStats()
    this.subscribeToContainerLogs()
    this.subscribeToContainerEvents()
    this.subscribeToUpdateNotifications()
    this.subscribeToSystemAlerts()
  }

  /**
   * Unsubscribe from a specific topic
   */
  unsubscribe(topic: string) {
    if (!this.client || !this.subscriptions.has(topic)) return

    this.client.unsubscribe(topic)
    this.subscriptions.delete(topic)
    console.log(`Unsubscribed from ${topic}`)
  }

  /**
   * Unsubscribe from all topics
   */
  unsubscribeAll() {
    if (!this.client) return

    this.subscriptions.forEach(topic => {
      this.client!.unsubscribe(topic)
    })
    this.subscriptions.clear()
    console.log('Unsubscribed from all topics')
  }

  /**
   * Reconnect the WebSocket connection
   */
  async reconnect() {
    if (!this.client) {
      await this.initialize()
      return
    }

    try {
      this.client.disconnect()
      await this.client.connect()

      // Re-subscribe to all topics
      const currentSubscriptions = Array.from(this.subscriptions)
      this.subscriptions.clear()

      currentSubscriptions.forEach(topic => {
        if (topic === 'containers.status') this.subscribeToContainerStatus()
        else if (topic === 'containers.stats') this.subscribeToContainerStats()
        else if (topic.startsWith('containers.logs')) this.subscribeToContainerLogs()
        else if (topic === 'containers.events') this.subscribeToContainerEvents()
        else if (topic === 'containers.updates') this.subscribeToUpdateNotifications()
        else if (topic === 'system.alerts') this.subscribeToSystemAlerts()
      })
    } catch (error) {
      console.error('Failed to reconnect WebSocket:', error)
      this.lastError.value = error instanceof Error ? error.message : 'Reconnection failed'
    }
  }

  /**
   * Update authentication token
   */
  updateToken(token: string) {
    if (!this.client) return

    this.client.updateToken(token)
  }

  /**
   * Get connection statistics
   */
  getStats() {
    if (!this.client) {
      return {
        subscriptions: 0,
        queuedMessages: 0,
        reconnectAttempts: 0,
        activeTopics: []
      }
    }

    const stats = this.client.getStats()
    return {
      ...stats,
      activeTopics: Array.from(this.subscriptions)
    }
  }

  /**
   * Disconnect and cleanup
   */
  disconnect() {
    this.unsubscribeAll()

    if (this.client) {
      this.client.disconnect()
      this.client = null
    }

    this.state.value = 'disconnected'
    this.isInitialized = false
    console.log('Container WebSocket service disconnected')
  }

  /**
   * Check if connected
   */
  get isConnected() {
    return this.state.value === 'connected'
  }

  /**
   * Get active subscriptions
   */
  get activeSubscriptions() {
    return Array.from(this.subscriptions)
  }
}

// Create singleton instance
export const containerWebSocketService = new ContainerWebSocketService()

// Vue composable for easier usage in components
export function useContainerWebSocket() {
  return {
    service: containerWebSocketService,
    state: containerWebSocketService.state,
    lastError: containerWebSocketService.lastError,
    isConnected: containerWebSocketService.isConnected,
    subscribeToAll: () => containerWebSocketService.subscribeToAll(),
    subscribeToContainerLogs: (containerId?: string) =>
      containerWebSocketService.subscribeToContainerLogs(containerId),
    unsubscribe: (topic: string) => containerWebSocketService.unsubscribe(topic),
    reconnect: () => containerWebSocketService.reconnect(),
    getStats: () => containerWebSocketService.getStats()
  }
}

export default containerWebSocketService