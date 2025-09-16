/**
 * Update WebSocket service for real-time updates
 */
import type { UpdateWebSocketMessage } from '@/types/updates'

export interface UpdateWebSocketCallbacks {
  onUpdateProgress?: (data: any) => void
  onUpdateCompleted?: (data: any) => void
  onUpdateFailed?: (data: any) => void
  onUpdateAvailable?: (data: any) => void
  onUpdateNotification?: (data: any) => void
  onConnected?: () => void
  onDisconnected?: () => void
  onError?: (error: any) => void
}

export class UpdateWebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private callbacks: UpdateWebSocketCallbacks = {}
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000 // Start with 1 second
  private reconnectTimer: NodeJS.Timeout | null = null
  private heartbeatTimer: NodeJS.Timeout | null = null
  private isIntentionallyClosed = false
  private subscriptions = new Set<string>()

  constructor(url?: string) {
    this.url = url || `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/updates`
  }

  /**
   * Connect to the WebSocket
   */
  connect(callbacks?: UpdateWebSocketCallbacks): Promise<void> {
    if (callbacks) {
      this.callbacks = { ...this.callbacks, ...callbacks }
    }

    return new Promise((resolve, reject) => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        resolve()
        return
      }

      this.isIntentionallyClosed = false

      try {
        const token = localStorage.getItem('authToken')
        const wsUrl = token ? `${this.url}?token=${token}` : this.url

        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
          console.log('Update WebSocket connected')
          this.reconnectAttempts = 0
          this.reconnectDelay = 1000
          this.startHeartbeat()
          this.resubscribe()
          this.callbacks.onConnected?.()
          resolve()
        }

        this.ws.onmessage = (event) => {
          try {
            const message: UpdateWebSocketMessage = JSON.parse(event.data)
            this.handleMessage(message)
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error, event.data)
          }
        }

        this.ws.onclose = (event) => {
          console.log('Update WebSocket disconnected:', event.code, event.reason)
          this.stopHeartbeat()
          this.callbacks.onDisconnected?.()

          if (!this.isIntentionallyClosed && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect()
          }
        }

        this.ws.onerror = (error) => {
          console.error('Update WebSocket error:', error)
          this.callbacks.onError?.(error)
          reject(error)
        }

      } catch (error) {
        console.error('Failed to create WebSocket connection:', error)
        reject(error)
      }
    })
  }

  /**
   * Disconnect from the WebSocket
   */
  disconnect(): void {
    this.isIntentionallyClosed = true
    this.stopHeartbeat()
    this.clearReconnectTimer()

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
  }

  /**
   * Subscribe to specific update events
   */
  subscribe(subscription: string): void {
    this.subscriptions.add(subscription)

    if (this.isConnected()) {
      this.send({
        type: 'subscribe',
        data: { subscription }
      })
    }
  }

  /**
   * Unsubscribe from specific update events
   */
  unsubscribe(subscription: string): void {
    this.subscriptions.delete(subscription)

    if (this.isConnected()) {
      this.send({
        type: 'unsubscribe',
        data: { subscription }
      })
    }
  }

  /**
   * Subscribe to container-specific updates
   */
  subscribeToContainer(containerId: string): void {
    this.subscribe(`container:${containerId}`)
  }

  /**
   * Unsubscribe from container-specific updates
   */
  unsubscribeFromContainer(containerId: string): void {
    this.unsubscribe(`container:${containerId}`)
  }

  /**
   * Subscribe to update operation
   */
  subscribeToUpdate(updateId: string): void {
    this.subscribe(`update:${updateId}`)
  }

  /**
   * Unsubscribe from update operation
   */
  unsubscribeFromUpdate(updateId: string): void {
    this.unsubscribe(`update:${updateId}`)
  }

  /**
   * Subscribe to bulk update operation
   */
  subscribeToBulkUpdate(operationId: string): void {
    this.subscribe(`bulk:${operationId}`)
  }

  /**
   * Unsubscribe from bulk update operation
   */
  unsubscribeFromBulkUpdate(operationId: string): void {
    this.unsubscribe(`bulk:${operationId}`)
  }

  /**
   * Subscribe to all updates
   */
  subscribeToAllUpdates(): void {
    this.subscribe('updates:all')
  }

  /**
   * Subscribe to security updates only
   */
  subscribeToSecurityUpdates(): void {
    this.subscribe('updates:security')
  }

  /**
   * Subscribe to update notifications
   */
  subscribeToNotifications(): void {
    this.subscribe('notifications')
  }

  /**
   * Check if WebSocket is connected
   */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN
  }

  /**
   * Get connection state
   */
  getState(): 'connecting' | 'connected' | 'disconnected' | 'error' {
    if (!this.ws) return 'disconnected'

    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return 'connecting'
      case WebSocket.OPEN:
        return 'connected'
      case WebSocket.CLOSING:
      case WebSocket.CLOSED:
        return 'disconnected'
      default:
        return 'error'
    }
  }

  /**
   * Send message to server
   */
  private send(message: any): void {
    if (this.isConnected()) {
      this.ws!.send(JSON.stringify(message))
    }
  }

  /**
   * Handle incoming WebSocket messages
   */
  private handleMessage(message: UpdateWebSocketMessage): void {
    const { type, data } = message

    switch (type) {
      case 'update_progress':
        this.callbacks.onUpdateProgress?.(data)
        break

      case 'update_completed':
        this.callbacks.onUpdateCompleted?.(data)
        break

      case 'update_failed':
        this.callbacks.onUpdateFailed?.(data)
        break

      case 'update_available':
        this.callbacks.onUpdateAvailable?.(data)
        break

      case 'update_notification':
        this.callbacks.onUpdateNotification?.(data)
        break

      default:
        console.warn('Unknown WebSocket message type:', type)
    }
  }

  /**
   * Schedule reconnection attempt
   */
  private scheduleReconnect(): void {
    this.clearReconnectTimer()

    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts), 30000)

    console.log(`Scheduling WebSocket reconnect in ${delay}ms (attempt ${this.reconnectAttempts + 1})`)

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++
      this.connect().catch(() => {
        // Reconnection failed, will be retried automatically
      })
    }, delay)
  }

  /**
   * Clear reconnection timer
   */
  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  /**
   * Start heartbeat to keep connection alive
   */
  private startHeartbeat(): void {
    this.stopHeartbeat()

    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.send({ type: 'ping' })
      }
    }, 30000) // 30 seconds
  }

  /**
   * Stop heartbeat
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  /**
   * Resubscribe to all subscriptions after reconnect
   */
  private resubscribe(): void {
    for (const subscription of this.subscriptions) {
      this.send({
        type: 'subscribe',
        data: { subscription }
      })
    }
  }

  /**
   * Update callbacks
   */
  setCallbacks(callbacks: UpdateWebSocketCallbacks): void {
    this.callbacks = { ...this.callbacks, ...callbacks }
  }

  /**
   * Get current subscriptions
   */
  getSubscriptions(): string[] {
    return Array.from(this.subscriptions)
  }

  /**
   * Clear all subscriptions
   */
  clearSubscriptions(): void {
    for (const subscription of this.subscriptions) {
      this.unsubscribe(subscription)
    }
    this.subscriptions.clear()
  }
}

// Export singleton instance
export const updateWebSocket = new UpdateWebSocketService()

// Export for Vue composition API
export function useUpdateWebSocket() {
  return {
    updateWebSocket,
    connect: updateWebSocket.connect.bind(updateWebSocket),
    disconnect: updateWebSocket.disconnect.bind(updateWebSocket),
    subscribe: updateWebSocket.subscribe.bind(updateWebSocket),
    unsubscribe: updateWebSocket.unsubscribe.bind(updateWebSocket),
    subscribeToContainer: updateWebSocket.subscribeToContainer.bind(updateWebSocket),
    unsubscribeFromContainer: updateWebSocket.unsubscribeFromContainer.bind(updateWebSocket),
    subscribeToUpdate: updateWebSocket.subscribeToUpdate.bind(updateWebSocket),
    unsubscribeFromUpdate: updateWebSocket.unsubscribeFromUpdate.bind(updateWebSocket),
    subscribeToBulkUpdate: updateWebSocket.subscribeToBulkUpdate.bind(updateWebSocket),
    unsubscribeFromBulkUpdate: updateWebSocket.unsubscribeFromBulkUpdate.bind(updateWebSocket),
    subscribeToAllUpdates: updateWebSocket.subscribeToAllUpdates.bind(updateWebSocket),
    subscribeToSecurityUpdates: updateWebSocket.subscribeToSecurityUpdates.bind(updateWebSocket),
    subscribeToNotifications: updateWebSocket.subscribeToNotifications.bind(updateWebSocket),
    isConnected: updateWebSocket.isConnected.bind(updateWebSocket),
    getState: updateWebSocket.getState.bind(updateWebSocket)
  }
}