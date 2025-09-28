/**
 * Theme management composable with production logging
 */
import { ref, watch, computed, onMounted } from 'vue';
import { createLogger } from '@/services/logging';

export type ThemeMode = 'light' | 'dark' | 'auto';

export interface ThemeConfig {
  mode: ThemeMode;
  primaryColor?: string;
  accentColor?: string;
  fontSize?: 'small' | 'medium' | 'large';
  customColors?: Record<string, string>;
}

export interface ThemeState {
  mode: ThemeMode;
  actualTheme: 'light' | 'dark';
  config: ThemeConfig;
  isLoading: boolean;
  error: string | null;
}

const logger = createLogger('useTheme');
const STORAGE_KEY = 'theme-config';
const DEFAULT_CONFIG: ThemeConfig = {
  mode: 'auto',
  primaryColor: '#409eff',
  accentColor: '#67c23a',
  fontSize: 'medium',
};

export const useTheme = () => {
  const state = ref<ThemeState>({
    mode: 'auto',
    actualTheme: 'light',
    config: { ...DEFAULT_CONFIG },
    isLoading: false,
    error: null,
  });

  const prefersDark = ref(false);

  // Media query listener for system theme preference
  const mediaQuery = typeof window !== 'undefined'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : null;

  /**
   * Read theme configuration from localStorage with error handling
   */
  const readThemeFromStorage = (): ThemeConfig | null => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) return null;

      const parsed = JSON.parse(stored);

      // Validate the parsed data
      if (!parsed || typeof parsed !== 'object') {
        logger.warn('Invalid theme data format in localStorage', {
          action: 'read-storage',
          data: parsed,
        });
        return null;
      }

      logger.debug('Theme configuration loaded from storage', {
        action: 'read-storage',
        config: parsed,
      });

      return parsed as ThemeConfig;
    } catch (error) {
      logger.error('Failed to read theme from localStorage', error as Error, {
        action: 'read-storage',
        storageKey: STORAGE_KEY,
      });
      state.value.error = 'Failed to load theme preferences';
      return null;
    }
  };

  /**
   * Save theme configuration to localStorage with error handling
   */
  const saveThemeToStorage = async (config: ThemeConfig): Promise<void> => {
    try {
      const serialized = JSON.stringify(config);
      localStorage.setItem(STORAGE_KEY, serialized);

      logger.debug('Theme configuration saved to storage', {
        action: 'save-storage',
        config,
        size: `${serialized.length} bytes`,
      });

      state.value.error = null;
    } catch (error) {
      logger.error('Failed to save theme to localStorage', error as Error, {
        action: 'save-storage',
        config,
        storageKey: STORAGE_KEY,
      });
      state.value.error = 'Failed to save theme preferences';
      throw error;
    }
  };

  /**
   * Update system theme preference detection
   */
  const updateSystemPreference = (matches: boolean): void => {
    prefersDark.value = matches;

    logger.debug('System theme preference updated', {
      action: 'system-preference',
      prefersDark: matches,
      mode: state.value.mode,
    });

    if (state.value.mode === 'auto') {
      updateActualTheme();
    }
  };

  /**
   * Update the actual applied theme based on mode and system preference
   */
  const updateActualTheme = (): void => {
    const previousTheme = state.value.actualTheme;

    switch (state.value.mode) {
      case 'light':
        state.value.actualTheme = 'light';
        break;
      case 'dark':
        state.value.actualTheme = 'dark';
        break;
      case 'auto':
        state.value.actualTheme = prefersDark.value ? 'dark' : 'light';
        break;
    }

    if (previousTheme !== state.value.actualTheme) {
      logger.info('Theme changed', {
        action: 'theme-change',
        from: previousTheme,
        to: state.value.actualTheme,
        mode: state.value.mode,
      });

      applyThemeToDocument();
    }
  };

  /**
   * Apply theme classes and CSS variables to document
   */
  const applyThemeToDocument = (): void => {
    try {
      const root = document.documentElement;
      const { actualTheme, config } = state.value;

      // Remove previous theme classes
      root.classList.remove('light', 'dark');
      root.classList.add(actualTheme);

      // Apply CSS custom properties
      if (config.primaryColor) {
        root.style.setProperty('--el-color-primary', config.primaryColor);
      }

      if (config.accentColor) {
        root.style.setProperty('--el-color-success', config.accentColor);
      }

      // Apply font size
      const fontSizes = {
        small: '14px',
        medium: '16px',
        large: '18px',
      };

      if (config.fontSize) {
        root.style.setProperty('--el-font-size-base', fontSizes[config.fontSize]);
      }

      // Apply custom colors if any
      if (config.customColors) {
        Object.entries(config.customColors).forEach(([key, value]) => {
          root.style.setProperty(`--custom-${key}`, value);
        });
      }

      logger.debug('Theme applied to document', {
        action: 'apply-theme',
        theme: actualTheme,
        config,
      });
    } catch (error) {
      logger.error('Failed to apply theme to document', error as Error, {
        action: 'apply-theme',
        theme: state.value.actualTheme,
      });
    }
  };

  /**
   * Set theme mode with validation and persistence
   */
  const setMode = async (mode: ThemeMode): Promise<void> => {
    if (!['light', 'dark', 'auto'].includes(mode)) {
      logger.warn('Invalid theme mode provided', {
        action: 'set-mode',
        mode,
        validModes: ['light', 'dark', 'auto'],
      });
      return;
    }

    state.value.isLoading = true;

    try {
      const newConfig = { ...state.value.config, mode };
      await saveThemeToStorage(newConfig);

      state.value.mode = mode;
      state.value.config = newConfig;
      updateActualTheme();

      logger.info('Theme mode updated', {
        action: 'set-mode',
        mode,
        actualTheme: state.value.actualTheme,
      });
    } catch (error) {
      logger.error('Failed to set theme mode', error as Error, {
        action: 'set-mode',
        mode,
      });
      throw error;
    } finally {
      state.value.isLoading = false;
    }
  };

  /**
   * Update theme configuration
   */
  const updateConfig = async (updates: Partial<ThemeConfig>): Promise<void> => {
    state.value.isLoading = true;

    try {
      const newConfig = { ...state.value.config, ...updates };
      await saveThemeToStorage(newConfig);

      state.value.config = newConfig;
      if (updates.mode) {
        state.value.mode = updates.mode;
        updateActualTheme();
      } else {
        applyThemeToDocument();
      }

      logger.info('Theme configuration updated', {
        action: 'update-config',
        updates,
        config: newConfig,
      });
    } catch (error) {
      logger.error('Failed to update theme configuration', error as Error, {
        action: 'update-config',
        updates,
      });
      throw error;
    } finally {
      state.value.isLoading = false;
    }
  };

  /**
   * Toggle between light and dark mode (ignores auto mode)
   */
  const toggleMode = async (): Promise<void> => {
    const newMode = state.value.actualTheme === 'light' ? 'dark' : 'light';
    await setMode(newMode);
  };

  /**
   * Reset theme to defaults
   */
  const resetTheme = async (): Promise<void> => {
    try {
      localStorage.removeItem(STORAGE_KEY);
      state.value.config = { ...DEFAULT_CONFIG };
      state.value.mode = DEFAULT_CONFIG.mode;
      state.value.error = null;
      updateActualTheme();

      logger.info('Theme reset to defaults', {
        action: 'reset-theme',
        config: DEFAULT_CONFIG,
      });
    } catch (error) {
      logger.error('Failed to reset theme', error as Error, {
        action: 'reset-theme',
      });
      throw error;
    }
  };

  /**
   * Initialize theme system
   */
  const initializeTheme = (): void => {
    try {
      // Read saved configuration
      const savedConfig = readThemeFromStorage();
      if (savedConfig) {
        state.value.config = { ...DEFAULT_CONFIG, ...savedConfig };
        state.value.mode = savedConfig.mode || DEFAULT_CONFIG.mode;
      }

      // Initialize system preference
      if (mediaQuery) {
        prefersDark.value = mediaQuery.matches;
        mediaQuery.addEventListener('change', (e) => updateSystemPreference(e.matches));
      }

      // Apply initial theme
      updateActualTheme();

      logger.info('Theme system initialized', {
        action: 'initialize',
        config: state.value.config,
        systemPrefersDark: prefersDark.value,
      });
    } catch (error) {
      logger.error('Failed to initialize theme system', error as Error, {
        action: 'initialize',
      });
      // Apply default theme as fallback
      applyThemeToDocument();
    }
  };

  // Computed properties
  const isDark = computed(() => state.value.actualTheme === 'dark');
  const isLight = computed(() => state.value.actualTheme === 'light');
  const isAuto = computed(() => state.value.mode === 'auto');

  // Watch for theme changes to persist configuration
  watch(
    () => state.value.config,
    async (newConfig) => {
      if (newConfig !== DEFAULT_CONFIG) {
        try {
          await saveThemeToStorage(newConfig);
        } catch (error) {
          // Error already logged in saveThemeToStorage
        }
      }
    },
    { deep: true }
  );

  // Initialize on mount
  onMounted(() => {
    initializeTheme();
  });

  return {
    // State
    mode: computed(() => state.value.mode),
    actualTheme: computed(() => state.value.actualTheme),
    config: computed(() => state.value.config),
    isLoading: computed(() => state.value.isLoading),
    error: computed(() => state.value.error),

    // Computed flags
    isDark,
    isLight,
    isAuto,
    prefersDark: computed(() => prefersDark.value),

    // Actions
    setMode,
    updateConfig,
    toggleMode,
    resetTheme,

    // Utilities
    applyThemeToDocument,
  };
};