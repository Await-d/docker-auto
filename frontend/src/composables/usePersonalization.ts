/**
 * Personalization composable with real API integration and analytics
 */
import { ref, computed, watch, onMounted } from 'vue';
import { createLogger } from '@/services/logging';
import { get, post, put } from '@/utils/request';
import { useAuth } from './useAuth';

export interface PersonalizationData {
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
}

export interface UserInteraction {
  type: 'click' | 'view' | 'search' | 'action';
  target: string;
  context?: Record<string, any>;
  timestamp: string;
  sessionId: string;
}

export interface PersonalizationState {
  data: PersonalizationData | null;
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
  lastSync: Date | null;
  isDirty: boolean;
  interactions: UserInteraction[];
}

const logger = createLogger('usePersonalization');
const SYNC_INTERVAL = 30000; // 30 seconds
const INTERACTION_BATCH_SIZE = 50;

export const usePersonalization = () => {
  const { user } = useAuth();

  const state = ref<PersonalizationState>({
    data: null,
    isLoading: false,
    isSaving: false,
    error: null,
    lastSync: null,
    isDirty: false,
    interactions: [],
  });

  let syncTimer: NodeJS.Timeout | null = null;
  let sessionId: string = generateSessionId();

  /**
   * Generate a unique session ID
   */
  function generateSessionId(): string {
    return `session-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Load personalization data from API
   */
  const loadPersonalizationData = async (): Promise<void> => {
    if (!user.value?.id) {
      logger.debug('No authenticated user, skipping personalization load', {
        action: 'load-personalization',
      });
      return;
    }

    state.value.isLoading = true;
    state.value.error = null;

    try {
      const response = await get(`/api/users/${user.value.id}/personalization`);

      if (!response.data) {
        throw new Error('No personalization data received');
      }

      state.value.data = response.data;
      state.value.lastSync = new Date();
      state.value.isDirty = false;

      logger.info('Personalization data loaded successfully', {
        action: 'load-personalization',
        userId: user.value.id,
        dataSize: JSON.stringify(response.data).length,
        preferences: Object.keys(response.data.preferences || {}),
      });

    } catch (error) {
      const errorMessage = (error as any).response?.data?.message || 'Failed to load personalization data';
      state.value.error = errorMessage;

      logger.error('Failed to load personalization data', error as Error, {
        action: 'load-personalization',
        userId: user.value?.id,
        error: errorMessage,
      });

      // Initialize with default data if loading fails
      await initializeDefaultData();
    } finally {
      state.value.isLoading = false;
    }
  };

  /**
   * Initialize default personalization data
   */
  const initializeDefaultData = async (): Promise<void> => {
    if (!user.value?.id) return;

    const defaultData: PersonalizationData = {
      userId: user.value.id,
      preferences: {
        theme: 'auto',
        language: 'zh-CN',
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        dateFormat: 'YYYY-MM-DD',
        numberFormat: 'en-US',
        dashboardLayout: ['system-overview', 'container-stats', 'recent-activities'],
        widgetSettings: {},
        notifications: {
          email: true,
          push: false,
          desktop: true,
          categories: ['errors', 'updates', 'system'],
        },
      },
      customizations: {
        shortcuts: [],
        quickActions: ['start-container', 'stop-container', 'view-logs'],
        favoriteContainers: [],
        recentlyViewed: [],
      },
      analytics: {
        mostUsedFeatures: {},
        sessionDuration: [],
        clickHeatmap: {},
        errorPatterns: [],
      },
    };

    try {
      // Save default data to API
      await post('/api/users/personalization', defaultData);
      state.value.data = defaultData;
      state.value.lastSync = new Date();
      state.value.isDirty = false;

      logger.info('Default personalization data initialized', {
        action: 'init-default',
        userId: user.value.id,
      });
    } catch (error) {
      logger.error('Failed to initialize default personalization data', error as Error, {
        action: 'init-default',
        userId: user.value.id,
      });

      // Use local data even if API save fails
      state.value.data = defaultData;
    }
  };

  /**
   * Save personalization data to API
   */
  const savePersonalizationData = async (): Promise<void> => {
    if (!state.value.data || !state.value.isDirty || state.value.isSaving) {
      return;
    }

    state.value.isSaving = true;
    state.value.error = null;

    try {
      const response = await put(
        `/api/users/${state.value.data.userId}/personalization`,
        state.value.data
      );

      if (response.data) {
        state.value.data = response.data;
      }

      state.value.lastSync = new Date();
      state.value.isDirty = false;

      logger.info('Personalization data saved successfully', {
        action: 'save-personalization',
        userId: state.value.data.userId,
        dataSize: JSON.stringify(state.value.data).length,
      });

    } catch (error) {
      const errorMessage = (error as any).response?.data?.message || 'Failed to save personalization data';
      state.value.error = errorMessage;

      logger.error('Failed to save personalization data', error as Error, {
        action: 'save-personalization',
        userId: state.value.data?.userId,
        error: errorMessage,
      });

      throw error;
    } finally {
      state.value.isSaving = false;
    }
  };

  /**
   * Record user interaction for analytics
   */
  const recordInteraction = (
    type: UserInteraction['type'],
    target: string,
    context?: Record<string, any>
  ): void => {
    if (!state.value.data) return;

    const interaction: UserInteraction = {
      type,
      target,
      context,
      timestamp: new Date().toISOString(),
      sessionId,
    };

    state.value.interactions.push(interaction);

    // Update analytics in personalization data
    if (state.value.data.analytics) {
      // Track feature usage
      const featureKey = `${type}:${target}`;
      state.value.data.analytics.mostUsedFeatures[featureKey] =
        (state.value.data.analytics.mostUsedFeatures[featureKey] || 0) + 1;

      // Track click heatmap
      if (type === 'click') {
        state.value.data.analytics.clickHeatmap[target] =
          (state.value.data.analytics.clickHeatmap[target] || 0) + 1;
      }

      state.value.isDirty = true;
    }

    logger.debug('User interaction recorded', {
      action: 'record-interaction',
      interaction: {
        type,
        target,
        context: context ? Object.keys(context) : 'none',
        sessionId,
      },
      totalInteractions: state.value.interactions.length,
    });

    // Batch upload interactions when buffer is full
    if (state.value.interactions.length >= INTERACTION_BATCH_SIZE) {
      uploadInteractions();
    }
  };

  /**
   * Upload interactions to analytics service
   */
  const uploadInteractions = async (): Promise<void> => {
    if (state.value.interactions.length === 0 || !state.value.data) {
      return;
    }

    const interactionsToUpload = [...state.value.interactions];
    state.value.interactions = [];

    try {
      await post('/api/analytics/interactions', {
        userId: state.value.data.userId,
        sessionId,
        interactions: interactionsToUpload,
      });

      logger.debug('Interactions uploaded to analytics service', {
        action: 'upload-interactions',
        count: interactionsToUpload.length,
        userId: state.value.data.userId,
        sessionId,
      });

    } catch (error) {
      // Put interactions back on error for retry
      state.value.interactions.unshift(...interactionsToUpload);

      logger.error('Failed to upload interactions', error as Error, {
        action: 'upload-interactions',
        count: interactionsToUpload.length,
        userId: state.value.data?.userId,
      });
    }
  };

  /**
   * Update user preferences
   */
  const updatePreferences = async (updates: Partial<PersonalizationData['preferences']>): Promise<void> => {
    if (!state.value.data) {
      throw new Error('Personalization data not loaded');
    }

    const oldPreferences = { ...state.value.data.preferences };
    state.value.data.preferences = { ...state.value.data.preferences, ...updates };
    state.value.isDirty = true;

    try {
      await savePersonalizationData();

      logger.info('User preferences updated', {
        action: 'update-preferences',
        userId: state.value.data.userId,
        updates: Object.keys(updates),
        changes: Object.entries(updates).reduce((acc, [key, value]) => {
          acc[key] = { from: oldPreferences[key as keyof typeof oldPreferences], to: value };
          return acc;
        }, {} as Record<string, any>),
      });

    } catch (error) {
      // Revert changes on error
      state.value.data.preferences = oldPreferences;
      state.value.isDirty = false;
      throw error;
    }
  };

  /**
   * Add to recently viewed items
   */
  const addToRecentlyViewed = (item: PersonalizationData['customizations']['recentlyViewed'][0]): void => {
    if (!state.value.data) return;

    // Remove existing entry if present
    state.value.data.customizations.recentlyViewed =
      state.value.data.customizations.recentlyViewed.filter(
        recent => !(recent.type === item.type && recent.id === item.id)
      );

    // Add to beginning
    state.value.data.customizations.recentlyViewed.unshift(item);

    // Keep only last 20 items
    if (state.value.data.customizations.recentlyViewed.length > 20) {
      state.value.data.customizations.recentlyViewed =
        state.value.data.customizations.recentlyViewed.slice(0, 20);
    }

    state.value.isDirty = true;

    recordInteraction('view', `${item.type}:${item.id}`, {
      name: item.name,
      type: item.type,
    });
  };

  /**
   * Toggle favorite container
   */
  const toggleFavoriteContainer = (containerId: string): void => {
    if (!state.value.data) return;

    const favorites = state.value.data.customizations.favoriteContainers;
    const index = favorites.indexOf(containerId);

    if (index === -1) {
      favorites.push(containerId);
      recordInteraction('action', 'add-favorite', { containerId });
    } else {
      favorites.splice(index, 1);
      recordInteraction('action', 'remove-favorite', { containerId });
    }

    state.value.isDirty = true;

    logger.info('Favorite container toggled', {
      action: 'toggle-favorite',
      containerId,
      isFavorite: index === -1,
      totalFavorites: favorites.length,
    });
  };

  /**
   * Add custom shortcut
   */
  const addShortcut = (shortcut: PersonalizationData['customizations']['shortcuts'][0]): void => {
    if (!state.value.data) return;

    // Check for duplicate IDs
    const existingIndex = state.value.data.customizations.shortcuts.findIndex(s => s.id === shortcut.id);

    if (existingIndex !== -1) {
      state.value.data.customizations.shortcuts[existingIndex] = shortcut;
    } else {
      state.value.data.customizations.shortcuts.push(shortcut);
    }

    state.value.isDirty = true;

    recordInteraction('action', 'add-shortcut', {
      shortcutId: shortcut.id,
      name: shortcut.name,
      action: shortcut.action,
    });

    logger.info('Custom shortcut added', {
      action: 'add-shortcut',
      shortcut: {
        id: shortcut.id,
        name: shortcut.name,
        action: shortcut.action,
        hotkey: shortcut.hotkey,
      },
      totalShortcuts: state.value.data.customizations.shortcuts.length,
    });
  };

  /**
   * Remove custom shortcut
   */
  const removeShortcut = (shortcutId: string): void => {
    if (!state.value.data) return;

    const index = state.value.data.customizations.shortcuts.findIndex(s => s.id === shortcutId);

    if (index !== -1) {
      const removed = state.value.data.customizations.shortcuts.splice(index, 1)[0];
      state.value.isDirty = true;

      recordInteraction('action', 'remove-shortcut', {
        shortcutId: removed.id,
        name: removed.name,
      });

      logger.info('Custom shortcut removed', {
        action: 'remove-shortcut',
        shortcutId,
        name: removed.name,
        remainingShortcuts: state.value.data.customizations.shortcuts.length,
      });
    }
  };

  /**
   * Setup automatic sync
   */
  const setupAutoSync = (): void => {
    if (syncTimer) {
      clearInterval(syncTimer);
    }

    syncTimer = setInterval(async () => {
      if (state.value.isDirty && !state.value.isSaving) {
        try {
          await savePersonalizationData();
          await uploadInteractions();
        } catch (error) {
          logger.debug('Auto-sync failed', {
            action: 'auto-sync',
            error: (error as Error).message,
          });
        }
      }
    }, SYNC_INTERVAL);

    logger.debug('Auto-sync setup completed', {
      action: 'setup-auto-sync',
      interval: SYNC_INTERVAL,
    });
  };

  /**
   * Cleanup function
   */
  const cleanup = async (): Promise<void> => {
    if (syncTimer) {
      clearInterval(syncTimer);
      syncTimer = null;
    }

    // Final save and interaction upload
    try {
      if (state.value.isDirty) {
        await savePersonalizationData();
      }
      if (state.value.interactions.length > 0) {
        await uploadInteractions();
      }
    } catch (error) {
      logger.debug('Cleanup operations failed', {
        action: 'cleanup',
        error: (error as Error).message,
      });
    }

    logger.debug('Personalization cleanup completed', {
      action: 'cleanup',
    });
  };

  // Computed properties
  const isLoaded = computed(() => state.value.data !== null);
  const preferences = computed(() => state.value.data?.preferences || null);
  const customizations = computed(() => state.value.data?.customizations || null);
  const analytics = computed(() => state.value.data?.analytics || null);
  const favoriteContainers = computed(() => state.value.data?.customizations.favoriteContainers || []);
  const shortcuts = computed(() => state.value.data?.customizations.shortcuts || []);
  const recentlyViewed = computed(() => state.value.data?.customizations.recentlyViewed || []);

  // Watch for user changes
  watch(
    () => user.value?.id,
    async (newUserId) => {
      if (newUserId) {
        sessionId = generateSessionId();
        await loadPersonalizationData();
        setupAutoSync();
      } else {
        await cleanup();
        state.value.data = null;
        state.value.error = null;
        state.value.isDirty = false;
      }
    },
    { immediate: false }
  );

  // Initialize on mount
  onMounted(async () => {
    if (user.value?.id) {
      await loadPersonalizationData();
      setupAutoSync();
    }
  });

  // Cleanup on unmount
  window.addEventListener('beforeunload', cleanup);

  return {
    // State
    data: computed(() => state.value.data),
    isLoading: computed(() => state.value.isLoading),
    isSaving: computed(() => state.value.isSaving),
    error: computed(() => state.value.error),
    lastSync: computed(() => state.value.lastSync),
    isDirty: computed(() => state.value.isDirty),

    // Computed data
    isLoaded,
    preferences,
    customizations,
    analytics,
    favoriteContainers,
    shortcuts,
    recentlyViewed,

    // Actions
    loadPersonalizationData,
    savePersonalizationData,
    updatePreferences,
    recordInteraction,
    addToRecentlyViewed,
    toggleFavoriteContainer,
    addShortcut,
    removeShortcut,
    cleanup,

    // Analytics
    uploadInteractions,
  };
};