/**
 * Authentication composable with production logging and error handling
 */
import { ref, computed, watch, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { createLogger, initializeLogging } from '@/services/logging';
import { post, get, put } from '@/utils/request';
import { ElMessage } from 'element-plus';

export interface User {
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
}

export interface LoginCredentials {
  username: string;
  password: string;
  rememberMe?: boolean;
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
}

export interface AuthState {
  user: User | null;
  tokens: AuthTokens | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  loginAttempts: number;
  lastLoginAttempt?: Date;
}

const logger = createLogger('useAuth');
const TOKEN_STORAGE_KEY = 'auth-tokens';
const USER_STORAGE_KEY = 'auth-user';
const MAX_LOGIN_ATTEMPTS = 5;
const LOCKOUT_DURATION = 15 * 60 * 1000; // 15 minutes

export const useAuth = () => {
  const router = useRouter();

  const state = ref<AuthState>({
    user: null,
    tokens: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,
    loginAttempts: 0,
  });

  let refreshTimer: NodeJS.Timeout | null = null;

  /**
   * Load stored authentication data
   */
  const loadStoredAuth = (): void => {
    try {
      const tokensData = localStorage.getItem(TOKEN_STORAGE_KEY);
      const userData = localStorage.getItem(USER_STORAGE_KEY);

      if (tokensData && userData) {
        const tokens = JSON.parse(tokensData) as AuthTokens;
        const user = JSON.parse(userData) as User;

        // Check if tokens are still valid
        if (tokens.expiresAt > Date.now()) {
          state.value.tokens = tokens;
          state.value.user = user;
          state.value.isAuthenticated = true;

          // Initialize logging with user context
          initializeLogging({
            userId: user.id,
            username: user.username,
          });

          logger.info('Stored authentication loaded', {
            action: 'load-stored-auth',
            userId: user.id,
            expiresAt: new Date(tokens.expiresAt),
          });

          setupTokenRefresh();
        } else {
          logger.warn('Stored tokens expired', {
            action: 'load-stored-auth',
            expiresAt: new Date(tokens.expiresAt),
            now: new Date(),
          });
          clearStoredAuth();
        }
      }
    } catch (error) {
      logger.error('Failed to load stored authentication', error as Error, {
        action: 'load-stored-auth',
      });
      clearStoredAuth();
    }
  };

  /**
   * Store authentication data
   */
  const storeAuth = (tokens: AuthTokens, user: User): void => {
    try {
      localStorage.setItem(TOKEN_STORAGE_KEY, JSON.stringify(tokens));
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(user));

      logger.debug('Authentication data stored', {
        action: 'store-auth',
        userId: user.id,
        expiresAt: new Date(tokens.expiresAt),
      });
    } catch (error) {
      logger.error('Failed to store authentication data', error as Error, {
        action: 'store-auth',
        userId: user.id,
      });
    }
  };

  /**
   * Clear stored authentication data
   */
  const clearStoredAuth = (): void => {
    try {
      localStorage.removeItem(TOKEN_STORAGE_KEY);
      localStorage.removeItem(USER_STORAGE_KEY);

      logger.debug('Stored authentication cleared', {
        action: 'clear-stored-auth',
      });
    } catch (error) {
      logger.error('Failed to clear stored authentication', error as Error, {
        action: 'clear-stored-auth',
      });
    }
  };

  /**
   * Check if user is locked out due to too many failed attempts
   */
  const isLockedOut = (): boolean => {
    if (state.value.loginAttempts >= MAX_LOGIN_ATTEMPTS && state.value.lastLoginAttempt) {
      const timeSinceLastAttempt = Date.now() - state.value.lastLoginAttempt.getTime();
      return timeSinceLastAttempt < LOCKOUT_DURATION;
    }
    return false;
  };

  /**
   * Get remaining lockout time in minutes
   */
  const getRemainingLockoutTime = (): number => {
    if (!state.value.lastLoginAttempt) return 0;
    const elapsed = Date.now() - state.value.lastLoginAttempt.getTime();
    const remaining = Math.max(0, LOCKOUT_DURATION - elapsed);
    return Math.ceil(remaining / 60000); // Convert to minutes
  };

  /**
   * Login user with credentials
   */
  const login = async (credentials: LoginCredentials): Promise<void> => {
    if (isLockedOut()) {
      const remaining = getRemainingLockoutTime();
      const message = `Account locked due to too many failed attempts. Try again in ${remaining} minutes.`;

      logger.warn('Login attempt while locked out', {
        action: 'login-locked-out',
        username: credentials.username,
        attempts: state.value.loginAttempts,
        remainingMinutes: remaining,
      });

      state.value.error = message;
      throw new Error(message);
    }

    state.value.isLoading = true;
    state.value.error = null;

    try {
      const response = await post('/api/auth/login', {
        username: credentials.username,
        password: credentials.password,
        rememberMe: credentials.rememberMe,
      });

      const { user, tokens } = response.data;

      // Validate response structure
      if (!user || !tokens) {
        throw new Error('Invalid login response structure');
      }

      // Update state
      state.value.user = user;
      state.value.tokens = tokens;
      state.value.isAuthenticated = true;
      state.value.loginAttempts = 0;
      state.value.lastLoginAttempt = undefined;

      // Store authentication data
      storeAuth(tokens, user);

      // Initialize logging with user context
      initializeLogging({
        userId: user.id,
        username: user.username,
      });

      // Setup automatic token refresh
      setupTokenRefresh();

      logger.info('User logged in successfully', {
        action: 'login-success',
        userId: user.id,
        username: user.username,
        role: user.role,
        rememberMe: credentials.rememberMe,
      });

      ElMessage.success('Login successful');

    } catch (error) {
      state.value.loginAttempts++;
      state.value.lastLoginAttempt = new Date();

      const errorMessage = (error as any).response?.data?.message || 'Login failed';
      state.value.error = errorMessage;

      logger.warn('Login failed', {
        action: 'login-failed',
        username: credentials.username,
        attempts: state.value.loginAttempts,
        error: errorMessage,
        isLocked: isLockedOut(),
      });

      if (isLockedOut()) {
        ElMessage.error(`Too many failed attempts. Account locked for ${Math.ceil(LOCKOUT_DURATION / 60000)} minutes.`);
      } else {
        ElMessage.error(errorMessage);
      }

      throw error;
    } finally {
      state.value.isLoading = false;
    }
  };

  /**
   * Logout user
   */
  const logout = async (reason?: string): Promise<void> => {
    const currentUserId = state.value.user?.id;
    const currentUsername = state.value.user?.username;

    state.value.isLoading = true;

    try {
      // Call logout API if authenticated
      if (state.value.isAuthenticated && state.value.tokens) {
        await post('/api/auth/logout', {
          refreshToken: state.value.tokens.refreshToken,
        });

        logger.info('Logout API call successful', {
          action: 'logout-success',
          userId: currentUserId,
          username: currentUsername,
          reason,
        });
      }
    } catch (error) {
      // Log warning but don't prevent logout
      logger.warn('Logout API call failed', {
        action: 'logout-api-failed',
        userId: currentUserId,
        error: (error as Error).message,
        reason,
      });
    }

    // Clear state regardless of API call result
    state.value.user = null;
    state.value.tokens = null;
    state.value.isAuthenticated = false;
    state.value.error = null;
    clearStoredAuth();

    // Clear token refresh timer
    if (refreshTimer) {
      clearTimeout(refreshTimer);
      refreshTimer = null;
    }

    logger.info('User logged out', {
      action: 'logout',
      userId: currentUserId,
      username: currentUsername,
      reason,
    });

    state.value.isLoading = false;

    // Redirect to login page
    await router.push('/login');
  };

  /**
   * Refresh authentication tokens
   */
  const refreshTokens = async (): Promise<void> => {
    if (!state.value.tokens?.refreshToken) {
      logger.warn('No refresh token available', {
        action: 'refresh-no-token',
      });
      await logout('No refresh token');
      return;
    }

    try {
      const response = await post('/api/auth/refresh', {
        refreshToken: state.value.tokens.refreshToken,
      });

      const { tokens, user } = response.data;

      // Update tokens
      state.value.tokens = tokens;
      if (user) {
        state.value.user = user;
      }

      // Store updated auth data
      storeAuth(tokens, state.value.user!);

      // Setup next refresh
      setupTokenRefresh();

      logger.debug('Tokens refreshed successfully', {
        action: 'refresh-success',
        userId: state.value.user?.id,
        expiresAt: new Date(tokens.expiresAt),
      });

    } catch (error) {
      logger.error('Token refresh failed', error as Error, {
        action: 'refresh-failed',
        userId: state.value.user?.id,
      });

      // If refresh fails, logout the user
      await logout('Token refresh failed');
    }
  };

  /**
   * Setup automatic token refresh
   */
  const setupTokenRefresh = (): void => {
    if (refreshTimer) {
      clearTimeout(refreshTimer);
    }

    if (!state.value.tokens) return;

    // Refresh 5 minutes before expiry
    const refreshTime = state.value.tokens.expiresAt - Date.now() - (5 * 60 * 1000);

    if (refreshTime > 0) {
      refreshTimer = setTimeout(() => {
        refreshTokens();
      }, refreshTime);

      logger.debug('Token refresh scheduled', {
        action: 'schedule-refresh',
        refreshIn: `${Math.round(refreshTime / 60000)} minutes`,
        expiresAt: new Date(state.value.tokens!.expiresAt),
      });
    } else {
      // Token expires soon, refresh immediately
      refreshTokens();
    }
  };

  /**
   * Check if user has specific permission
   */
  const hasPermission = (permission: string): boolean => {
    return state.value.user?.permissions.includes(permission) ?? false;
  };

  /**
   * Check if user has specific role
   */
  const hasRole = (role: string): boolean => {
    return state.value.user?.role === role;
  };

  /**
   * Check if user has any of the specified roles
   */
  const hasAnyRole = (roles: string[]): boolean => {
    return state.value.user ? roles.includes(state.value.user.role) : false;
  };

  /**
   * Update user profile
   */
  const updateProfile = async (updates: Partial<User>): Promise<void> => {
    if (!state.value.user) throw new Error('Not authenticated');

    state.value.isLoading = true;

    try {
      const response = await put('/api/user/profile', updates);
      const updatedUser = response.data;

      state.value.user = updatedUser;
      storeAuth(state.value.tokens!, updatedUser);

      logger.info('User profile updated', {
        action: 'update-profile',
        userId: updatedUser.id,
        updates: Object.keys(updates),
      });

      ElMessage.success('Profile updated successfully');
    } catch (error) {
      logger.error('Failed to update profile', error as Error, {
        action: 'update-profile',
        userId: state.value.user.id,
      });
      throw error;
    } finally {
      state.value.isLoading = false;
    }
  };

  /**
   * Change user password
   */
  const changePassword = async (currentPassword: string, newPassword: string): Promise<void> => {
    if (!state.value.user) throw new Error('Not authenticated');

    state.value.isLoading = true;

    try {
      await post('/api/user/change-password', {
        currentPassword,
        newPassword,
      });

      logger.info('Password changed successfully', {
        action: 'change-password',
        userId: state.value.user.id,
      });

      ElMessage.success('Password changed successfully');
    } catch (error) {
      logger.error('Failed to change password', error as Error, {
        action: 'change-password',
        userId: state.value.user.id,
      });
      throw error;
    } finally {
      state.value.isLoading = false;
    }
  };

  // Computed properties
  const isAdmin = computed(() => hasRole('admin'));
  const isUser = computed(() => hasRole('user'));

  // Watch for authentication state changes
  watch(
    () => state.value.isAuthenticated,
    (isAuth) => {
      if (isAuth) {
        setupTokenRefresh();
      } else if (refreshTimer) {
        clearTimeout(refreshTimer);
        refreshTimer = null;
      }
    }
  );

  // Initialize authentication on mount
  onMounted(() => {
    loadStoredAuth();
  });

  return {
    // State
    user: computed(() => state.value.user),
    isAuthenticated: computed(() => state.value.isAuthenticated),
    isLoading: computed(() => state.value.isLoading),
    error: computed(() => state.value.error),
    loginAttempts: computed(() => state.value.loginAttempts),

    // Computed flags
    isAdmin,
    isUser,
    isLockedOut: computed(() => isLockedOut()),
    remainingLockoutTime: computed(() => getRemainingLockoutTime()),

    // Actions
    login,
    logout,
    refreshTokens,
    updateProfile,
    changePassword,

    // Permissions
    hasPermission,
    hasRole,
    hasAnyRole,
  };
};