/**
 * API Contracts for Production Composables
 *
 * This file defines the API contracts for all the production-grade composables
 * and the backend services they integrate with.
 */

// ===== LOGGING SERVICE CONTRACTS =====

export interface LoggingAPIContracts {
  // POST /api/logs - Batch log upload
  createLogs: {
    request: {
      entries: Array<{
        level: 'debug' | 'info' | 'warn' | 'error';
        source: string;
        message: string;
        details?: {
          correlationId?: string;
          context?: Record<string, any>;
          error?: {
            name: string;
            message: string;
            stack?: string;
          };
        };
        userId?: number;
        username?: string;
        ipAddress?: string;
        userAgent?: string;
        timestamp: string;
      }>;
    };
    response: {
      success: boolean;
      processed: number;
      errors?: string[];
    };
  };
}

// ===== PERSONALIZATION SERVICE CONTRACTS =====

export interface PersonalizationAPIContracts {
  // GET /api/users/:userId/personalization
  getPersonalization: {
    params: {
      userId: number;
    };
    response: {
      userId: number;
      preferences: {
        theme: string;
        language: string;
        timezone: string;
        dateFormat: string;
        numberFormat: string;
        dashboardLayout: string[];
        widgetSettings: Record<string, any>;
        notifications: {
          email: boolean;
          push: boolean;
          desktop: boolean;
          categories: string[];
        };
      };
      customizations: {
        shortcuts: Array<{
          id: string;
          name: string;
          action: string;
          icon?: string;
          hotkey?: string;
        }>;
        quickActions: string[];
        favoriteContainers: string[];
        recentlyViewed: Array<{
          type: 'container' | 'image' | 'volume';
          id: string;
          name: string;
          timestamp: string;
        }>;
      };
      analytics: {
        mostUsedFeatures: Record<string, number>;
        sessionDuration: number[];
        clickHeatmap: Record<string, number>;
        errorPatterns: string[];
      };
    };
  };

  // POST /api/users/personalization - Create personalization data
  createPersonalization: {
    request: PersonalizationAPIContracts['getPersonalization']['response'];
    response: PersonalizationAPIContracts['getPersonalization']['response'];
  };

  // PUT /api/users/:userId/personalization - Update personalization data
  updatePersonalization: {
    params: {
      userId: number;
    };
    request: Partial<PersonalizationAPIContracts['getPersonalization']['response']>;
    response: PersonalizationAPIContracts['getPersonalization']['response'];
  };
}

// ===== ANALYTICS SERVICE CONTRACTS =====

export interface AnalyticsAPIContracts {
  // POST /api/analytics/interactions - Upload user interactions
  uploadInteractions: {
    request: {
      userId: number;
      sessionId: string;
      interactions: Array<{
        type: 'click' | 'view' | 'search' | 'action';
        target: string;
        context?: Record<string, any>;
        timestamp: string;
        sessionId: string;
      }>;
    };
    response: {
      success: boolean;
      processed: number;
    };
  };

  // POST /api/analytics/notifications - Track notification events
  trackNotifications: {
    request: {
      userId?: number;
      notificationId: string;
      type: 'comment' | 'mention' | 'update' | 'error' | 'system';
      priority: 'low' | 'normal' | 'high' | 'urgent';
      timestamp: string;
    };
    response: {
      success: boolean;
    };
  };
}

// ===== AUTHENTICATION SERVICE CONTRACTS =====

export interface AuthenticationAPIContracts {
  // POST /api/auth/login - User login
  login: {
    request: {
      username: string;
      password: string;
      rememberMe?: boolean;
    };
    response: {
      user: {
        id: number;
        username: string;
        email?: string;
        role: string;
        permissions: string[];
        profile?: {
          avatar?: string;
          displayName?: string;
          lastLoginAt?: string;
        };
        preferences?: Record<string, any>;
      };
      tokens: {
        accessToken: string;
        refreshToken: string;
        expiresAt: number;
      };
    };
  };

  // POST /api/auth/logout - User logout
  logout: {
    request: {
      refreshToken: string;
    };
    response: {
      success: boolean;
    };
  };

  // POST /api/auth/refresh - Refresh tokens
  refreshTokens: {
    request: {
      refreshToken: string;
    };
    response: {
      tokens: {
        accessToken: string;
        refreshToken: string;
        expiresAt: number;
      };
      user?: AuthenticationAPIContracts['login']['response']['user'];
    };
  };

  // PUT /api/user/profile - Update user profile
  updateProfile: {
    request: Partial<{
      username: string;
      email: string;
      profile: {
        avatar?: string;
        displayName?: string;
      };
      preferences: Record<string, any>;
    }>;
    response: AuthenticationAPIContracts['login']['response']['user'];
  };

  // POST /api/user/change-password - Change password
  changePassword: {
    request: {
      currentPassword: string;
      newPassword: string;
    };
    response: {
      success: boolean;
    };
  };
}

// ===== NOTIFICATION SERVICE CONTRACTS =====

export interface NotificationAPIContracts {
  // GET /api/users/:userId/notification-settings
  getNotificationSettings: {
    params: {
      userId: number;
    };
    response: {
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
        start: string;
        end: string;
      };
    };
  };

  // PUT /api/users/:userId/notification-settings
  updateNotificationSettings: {
    params: {
      userId: number;
    };
    request: Partial<NotificationAPIContracts['getNotificationSettings']['response']>;
    response: NotificationAPIContracts['getNotificationSettings']['response'];
  };

  // POST /api/notifications/:notificationId/read
  markNotificationRead: {
    params: {
      notificationId: string;
    };
    response: {
      success: boolean;
    };
  };

  // POST /api/notifications/mark-all-read
  markAllNotificationsRead: {
    request: {
      notificationIds: string[];
    };
    response: {
      success: boolean;
      updated: number;
    };
  };
}

// ===== WEBSOCKET CONTRACTS =====

export interface WebSocketContracts {
  // WebSocket connection for notifications
  notifications: {
    url: '/ws/notifications';
    messages: {
      // Client to server
      auth: {
        type: 'auth';
        token: string;
      };

      // Server to client
      notification: {
        type: 'notification';
        notification: {
          id: string;
          type: 'comment' | 'mention' | 'update' | 'error' | 'system';
          title: string;
          message: string;
          priority: 'low' | 'normal' | 'high' | 'urgent';
          actions?: Array<{
            label: string;
            action: string;
            primary?: boolean;
          }>;
          metadata?: Record<string, any>;
        };
      };
    };
  };
}

// ===== CONSOLIDATED API CONTRACTS =====

export interface APIContracts extends
  LoggingAPIContracts,
  PersonalizationAPIContracts,
  AnalyticsAPIContracts,
  AuthenticationAPIContracts,
  NotificationAPIContracts {
}

// ===== ERROR RESPONSE FORMAT =====

export interface APIError {
  success: false;
  error: {
    code: string;
    message: string;
    details?: Record<string, any>;
    timestamp: string;
    correlationId?: string;
  };
}

// ===== SUCCESS RESPONSE FORMAT =====

export interface APISuccess<T = any> {
  success: true;
  data: T;
  timestamp: string;
  correlationId?: string;
}

// ===== COMMON RESPONSE WRAPPER =====

export type APIResponse<T> = APISuccess<T> | APIError;

// ===== VALIDATION SCHEMAS =====

export interface ValidationSchemas {
  personalization: {
    preferences: {
      theme: string[];
      language: string[];
      dateFormat: string[];
      numberFormat: string[];
    };
    customizations: {
      maxShortcuts: number;
      maxFavorites: number;
      maxRecentlyViewed: number;
    };
  };

  notifications: {
    categories: string[];
    priorities: string[];
    maxMessageLength: number;
  };

  authentication: {
    usernameMinLength: number;
    passwordMinLength: number;
    passwordRequirements: {
      uppercase: boolean;
      lowercase: boolean;
      numbers: boolean;
      specialChars: boolean;
    };
  };

  logging: {
    levels: string[];
    maxMessageLength: number;
    maxContextSize: number;
    rateLimits: {
      perSecond: number;
      perMinute: number;
      perHour: number;
    };
  };
}

// ===== TYPE GUARDS =====

export function isAPIError(response: APIResponse<any>): response is APIError {
  return response.success === false;
}

export function isAPISuccess<T>(response: APIResponse<T>): response is APISuccess<T> {
  return response.success === true;
}

// ===== CONSTANTS =====

export const API_CONSTANTS = {
  MAX_RETRY_ATTEMPTS: 3,
  REQUEST_TIMEOUT: 30000, // 30 seconds
  BATCH_SIZE: {
    LOGS: 100,
    INTERACTIONS: 50,
    NOTIFICATIONS: 25,
  },
  RATE_LIMITS: {
    LOGIN_ATTEMPTS: 5,
    API_CALLS_PER_MINUTE: 1000,
    LOG_ENTRIES_PER_SECOND: 10,
  },
} as const;