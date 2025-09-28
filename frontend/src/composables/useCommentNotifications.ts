/**
 * Comment notifications composable with production error handling
 */
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { createLogger } from '@/services/logging';
import { post, get, put } from '@/utils/request';
import { useAuth } from './useAuth';

export interface NotificationSettings {
  enabled: boolean;
  sound: boolean;
  desktop: boolean;
  email: boolean;
  categories: {
    comments: boolean;
    mentions: boolean;
    updates: boolean;
    errors: boolean;
    system: boolean;
  };
  quietHours: {
    enabled: boolean;
    start: string; // HH:mm format
    end: string;   // HH:mm format
  };
}

export interface NotificationMessage {
  id: string;
  type: 'comment' | 'mention' | 'update' | 'error' | 'system';
  title: string;
  message: string;
  timestamp: Date;
  read: boolean;
  priority: 'low' | 'normal' | 'high' | 'urgent';
  actions?: Array<{
    label: string;
    action: string;
    primary?: boolean;
  }>;
  metadata?: Record<string, any>;
}

export interface NotificationState {
  permission: NotificationPermission;
  isSupported: boolean;
  settings: NotificationSettings;
  messages: NotificationMessage[];
  unreadCount: number;
  isLoading: boolean;
  error: string | null;
  soundEnabled: boolean;
  lastNotificationSound?: Date;
}

const logger = createLogger('useCommentNotifications');
const NOTIFICATION_SOUND_COOLDOWN = 2000; // 2 seconds between sounds
const MAX_NOTIFICATIONS = 100;
const SETTINGS_STORAGE_KEY = 'notification-settings';

const DEFAULT_SETTINGS: NotificationSettings = {
  enabled: true,
  sound: true,
  desktop: true,
  email: true,
  categories: {
    comments: true,
    mentions: true,
    updates: true,
    errors: true,
    system: false,
  },
  quietHours: {
    enabled: false,
    start: '22:00',
    end: '08:00',
  },
};

export const useCommentNotifications = () => {
  const { user } = useAuth();

  const state = ref<NotificationState>({
    permission: 'default',
    isSupported: 'Notification' in window,
    settings: { ...DEFAULT_SETTINGS },
    messages: [],
    unreadCount: 0,
    isLoading: false,
    error: null,
    soundEnabled: true,
  });

  let websocket: WebSocket | null = null;
  let reconnectTimer: NodeJS.Timeout | null = null;
  let soundCooldownTimer: NodeJS.Timeout | null = null;

  /**
   * Check browser notification support and permissions
   */
  const checkNotificationSupport = (): void => {
    if (!state.value.isSupported) {
      logger.warn('Browser does not support desktop notifications', {
        action: 'check-support',
        userAgent: navigator.userAgent,
        notificationAPI: 'Notification' in window,
      });
      return;
    }

    state.value.permission = Notification.permission;

    logger.debug('Notification support checked', {
      action: 'check-support',
      supported: state.value.isSupported,
      permission: state.value.permission,
    });
  };

  /**
   * Request notification permission from user
   */
  const requestPermission = async (): Promise<NotificationPermission> => {
    if (!state.value.isSupported) {
      throw new Error('Notifications not supported in this browser');
    }

    if (state.value.permission === 'granted') {
      return state.value.permission;
    }

    try {
      const permission = await Notification.requestPermission();
      state.value.permission = permission;

      logger.info('Notification permission requested', {
        action: 'request-permission',
        permission,
        userId: user.value?.id,
      });

      if (permission === 'denied') {
        logger.warn('User denied notification permission', {
          action: 'permission-denied',
          userId: user.value?.id,
        });
      }

      return permission;
    } catch (error) {
      logger.error('Error requesting notification permission', error as Error, {
        action: 'request-permission',
        userId: user.value?.id,
      });

      state.value.error = 'Failed to request notification permission';
      throw error;
    }
  };

  /**
   * Load notification settings from storage and API
   */
  const loadSettings = async (): Promise<void> => {
    try {
      // Load from localStorage first for immediate UI update
      const storedSettings = localStorage.getItem(SETTINGS_STORAGE_KEY);
      if (storedSettings) {
        const parsed = JSON.parse(storedSettings);
        state.value.settings = { ...DEFAULT_SETTINGS, ...parsed };
      }

      // Load from API for server-side settings
      if (user.value?.id) {
        const response = await get(`/api/users/${user.value.id}/notification-settings`);
        if (response.data) {
          state.value.settings = { ...DEFAULT_SETTINGS, ...response.data };
          // Sync with localStorage
          localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(state.value.settings));
        }
      }

      logger.debug('Notification settings loaded', {
        action: 'load-settings',
        userId: user.value?.id,
        settings: state.value.settings,
      });

    } catch (error) {
      logger.error('Failed to load notification settings', error as Error, {
        action: 'load-settings',
        userId: user.value?.id,
      });

      // Use default settings on error
      state.value.settings = { ...DEFAULT_SETTINGS };
    }
  };

  /**
   * Save notification settings
   */
  const saveSettings = async (newSettings: Partial<NotificationSettings>): Promise<void> => {
    const updatedSettings = { ...state.value.settings, ...newSettings };

    try {
      // Save to localStorage immediately
      localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(updatedSettings));
      state.value.settings = updatedSettings;

      // Save to API
      if (user.value?.id) {
        await put(`/api/users/${user.value.id}/notification-settings`, updatedSettings);
      }

      logger.info('Notification settings saved', {
        action: 'save-settings',
        userId: user.value?.id,
        changes: Object.keys(newSettings),
        settings: updatedSettings,
      });

    } catch (error) {
      logger.error('Failed to save notification settings', error as Error, {
        action: 'save-settings',
        userId: user.value?.id,
        changes: newSettings,
      });

      state.value.error = 'Failed to save notification settings';
      throw error;
    }
  };

  /**
   * Check if current time is in quiet hours
   */
  const isInQuietHours = (): boolean => {
    if (!state.value.settings.quietHours.enabled) {
      return false;
    }

    const now = new Date();
    const currentTime = now.getHours() * 60 + now.getMinutes();

    const [startHour, startMin] = state.value.settings.quietHours.start.split(':').map(Number);
    const [endHour, endMin] = state.value.settings.quietHours.end.split(':').map(Number);

    const startTime = startHour * 60 + startMin;
    const endTime = endHour * 60 + endMin;

    // Handle overnight quiet hours (e.g., 22:00 to 08:00)
    if (startTime > endTime) {
      return currentTime >= startTime || currentTime <= endTime;
    } else {
      return currentTime >= startTime && currentTime <= endTime;
    }
  };

  /**
   * Play notification sound with cooldown
   */
  const playNotificationSound = (priority: NotificationMessage['priority']): void => {
    if (!state.value.settings.sound || !state.value.soundEnabled) {
      return;
    }

    if (isInQuietHours()) {
      logger.debug('Notification sound suppressed during quiet hours', {
        action: 'play-sound',
        priority,
        quietHours: state.value.settings.quietHours,
      });
      return;
    }

    // Implement sound cooldown to prevent spam
    const now = new Date();
    if (state.value.lastNotificationSound) {
      const timeSinceLastSound = now.getTime() - state.value.lastNotificationSound.getTime();
      if (timeSinceLastSound < NOTIFICATION_SOUND_COOLDOWN) {
        return;
      }
    }

    try {
      // Use different sounds for different priorities
      const soundFile = priority === 'urgent' ? '/sounds/urgent.mp3' :
                       priority === 'high' ? '/sounds/high.mp3' :
                       '/sounds/notification.mp3';

      const audio = new Audio(soundFile);
      audio.volume = 0.6; // 60% volume

      audio.play().catch(error => {
        logger.warn('Failed to play notification sound', {
          action: 'play-sound',
          priority,
          soundFile,
          error: error.message,
        });
      });

      state.value.lastNotificationSound = now;

      logger.debug('Notification sound played', {
        action: 'play-sound',
        priority,
        soundFile,
      });

    } catch (error) {
      logger.error('Error playing notification sound', error as Error, {
        action: 'play-sound',
        priority,
      });
    }
  };

  /**
   * Show desktop notification
   */
  const showDesktopNotification = (notification: NotificationMessage): void => {
    if (!state.value.settings.desktop ||
        !state.value.isSupported ||
        state.value.permission !== 'granted') {
      return;
    }

    if (isInQuietHours() && notification.priority !== 'urgent') {
      logger.debug('Desktop notification suppressed during quiet hours', {
        action: 'show-desktop',
        notificationId: notification.id,
        priority: notification.priority,
      });
      return;
    }

    try {
      const options: NotificationOptions = {
        body: notification.message,
        icon: '/icons/notification-icon.png',
        badge: '/icons/notification-badge.png',
        tag: notification.id,
        requireInteraction: notification.priority === 'urgent',
        silent: !state.value.settings.sound,
        data: {
          notificationId: notification.id,
          type: notification.type,
          timestamp: notification.timestamp,
        },
      };

      // Add action buttons if provided
      if (notification.actions && notification.actions.length > 0) {
        options.actions = notification.actions.map(action => ({
          action: action.action,
          title: action.label,
          icon: undefined, // Could add icons for actions
        }));
      }

      const desktopNotification = new Notification(notification.title, options);

      // Handle notification click
      desktopNotification.onclick = () => {
        logger.info('Desktop notification clicked', {
          action: 'desktop-click',
          notificationId: notification.id,
          type: notification.type,
        });

        // Mark as read
        markAsRead(notification.id);

        // Focus window
        window.focus();

        // Close notification
        desktopNotification.close();
      };

      // Handle notification close
      desktopNotification.onclose = () => {
        logger.debug('Desktop notification closed', {
          action: 'desktop-close',
          notificationId: notification.id,
        });
      };

      // Auto-close after 5 seconds for non-urgent notifications
      if (notification.priority !== 'urgent') {
        setTimeout(() => {
          desktopNotification.close();
        }, 5000);
      }

      logger.debug('Desktop notification shown', {
        action: 'show-desktop',
        notificationId: notification.id,
        title: notification.title,
        priority: notification.priority,
      });

    } catch (error) {
      logger.error('Error showing desktop notification', error as Error, {
        action: 'show-desktop',
        notificationId: notification.id,
        title: notification.title,
      });
    }
  };

  /**
   * Add a new notification
   */
  const addNotification = (notification: Omit<NotificationMessage, 'id' | 'timestamp' | 'read'>): void => {
    // Check if notifications are enabled for this category
    if (!state.value.settings.enabled ||
        !state.value.settings.categories[notification.type]) {
      return;
    }

    const fullNotification: NotificationMessage = {
      ...notification,
      id: `notif-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date(),
      read: false,
    };

    // Add to messages array
    state.value.messages.unshift(fullNotification);

    // Keep only recent notifications
    if (state.value.messages.length > MAX_NOTIFICATIONS) {
      state.value.messages = state.value.messages.slice(0, MAX_NOTIFICATIONS);
    }

    // Update unread count
    state.value.unreadCount = state.value.messages.filter(msg => !msg.read).length;

    // Show desktop notification
    showDesktopNotification(fullNotification);

    // Play sound
    playNotificationSound(fullNotification.priority);

    logger.info('Notification added', {
      action: 'add-notification',
      notificationId: fullNotification.id,
      type: fullNotification.type,
      priority: fullNotification.priority,
      title: fullNotification.title,
      unreadCount: state.value.unreadCount,
    });

    // Send to analytics
    try {
      post('/api/analytics/notifications', {
        userId: user.value?.id,
        notificationId: fullNotification.id,
        type: fullNotification.type,
        priority: fullNotification.priority,
        timestamp: fullNotification.timestamp.toISOString(),
      });
    } catch (error) {
      logger.debug('Failed to send notification analytics', {
        action: 'analytics',
        error: (error as Error).message,
      });
    }
  };

  /**
   * Mark notification as read
   */
  const markAsRead = (notificationId: string): void => {
    const notification = state.value.messages.find(msg => msg.id === notificationId);
    if (notification && !notification.read) {
      notification.read = true;
      state.value.unreadCount = state.value.messages.filter(msg => !msg.read).length;

      logger.debug('Notification marked as read', {
        action: 'mark-read',
        notificationId,
        unreadCount: state.value.unreadCount,
      });

      // Update read status on server
      if (user.value?.id) {
        post(`/api/notifications/${notificationId}/read`).catch(error => {
          logger.warn('Failed to update read status on server', {
            action: 'update-read-status',
            notificationId,
            error: error.message,
          });
        });
      }
    }
  };

  /**
   * Mark all notifications as read
   */
  const markAllAsRead = (): void => {
    const unreadIds = state.value.messages
      .filter(msg => !msg.read)
      .map(msg => msg.id);

    state.value.messages.forEach(msg => {
      msg.read = true;
    });

    state.value.unreadCount = 0;

    logger.info('All notifications marked as read', {
      action: 'mark-all-read',
      count: unreadIds.length,
    });

    // Update on server
    if (user.value?.id && unreadIds.length > 0) {
      post('/api/notifications/mark-all-read', {
        notificationIds: unreadIds
      }).catch(error => {
        logger.warn('Failed to mark all as read on server', {
          action: 'mark-all-read-server',
          error: error.message,
        });
      });
    }
  };

  /**
   * Clear old notifications
   */
  const clearOldNotifications = (olderThanDays: number = 7): void => {
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - olderThanDays);

    const beforeCount = state.value.messages.length;
    state.value.messages = state.value.messages.filter(
      msg => msg.timestamp > cutoffDate
    );

    const removedCount = beforeCount - state.value.messages.length;
    state.value.unreadCount = state.value.messages.filter(msg => !msg.read).length;

    if (removedCount > 0) {
      logger.info('Old notifications cleared', {
        action: 'clear-old',
        removedCount,
        remainingCount: state.value.messages.length,
        cutoffDays: olderThanDays,
      });
    }
  };

  /**
   * Connect to WebSocket for real-time notifications
   */
  const connectWebSocket = (): void => {
    if (!user.value?.id) return;

    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/notifications`;

    try {
      websocket = new WebSocket(wsUrl);

      websocket.onopen = () => {
        logger.info('WebSocket connected for notifications', {
          action: 'websocket-connect',
          userId: user.value?.id,
          url: wsUrl,
        });

        // Send authentication
        websocket!.send(JSON.stringify({
          type: 'auth',
          token: localStorage.getItem('auth-token'),
        }));
      };

      websocket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'notification') {
            addNotification(data.notification);
          }
        } catch (error) {
          logger.error('Error processing WebSocket message', error as Error, {
            action: 'websocket-message',
            data: event.data,
          });
        }
      };

      websocket.onclose = (event) => {
        logger.warn('WebSocket connection closed', {
          action: 'websocket-close',
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean,
        });

        // Attempt reconnection after 5 seconds
        if (!event.wasClean) {
          reconnectTimer = setTimeout(connectWebSocket, 5000);
        }
      };

      websocket.onerror = (error) => {
        logger.error('WebSocket error', error as Event, {
          action: 'websocket-error',
          readyState: websocket?.readyState,
        });
      };

    } catch (error) {
      logger.error('Failed to establish WebSocket connection', error as Error, {
        action: 'websocket-init',
        url: wsUrl,
      });
    }
  };

  /**
   * Disconnect WebSocket
   */
  const disconnectWebSocket = (): void => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }

    if (websocket) {
      websocket.close();
      websocket = null;
    }
  };

  // Computed properties
  const hasUnread = computed(() => state.value.unreadCount > 0);
  const recentNotifications = computed(() =>
    state.value.messages.slice(0, 10)
  );
  const urgentNotifications = computed(() =>
    state.value.messages.filter(msg => !msg.read && msg.priority === 'urgent')
  );

  // Initialize on mount
  onMounted(async () => {
    checkNotificationSupport();
    await loadSettings();
    connectWebSocket();

    // Clear old notifications on startup
    clearOldNotifications();

    logger.info('Comment notifications initialized', {
      action: 'initialize',
      userId: user.value?.id,
      supported: state.value.isSupported,
      permission: state.value.permission,
    });
  });

  // Cleanup on unmount
  onUnmounted(() => {
    disconnectWebSocket();

    if (soundCooldownTimer) {
      clearTimeout(soundCooldownTimer);
    }

    logger.debug('Comment notifications cleanup completed', {
      action: 'cleanup',
    });
  });

  return {
    // State
    permission: computed(() => state.value.permission),
    isSupported: computed(() => state.value.isSupported),
    settings: computed(() => state.value.settings),
    messages: computed(() => state.value.messages),
    unreadCount: computed(() => state.value.unreadCount),
    isLoading: computed(() => state.value.isLoading),
    error: computed(() => state.value.error),

    // Computed
    hasUnread,
    recentNotifications,
    urgentNotifications,

    // Actions
    requestPermission,
    saveSettings,
    addNotification,
    markAsRead,
    markAllAsRead,
    clearOldNotifications,

    // Utils
    isInQuietHours: computed(() => isInQuietHours()),
  };
};