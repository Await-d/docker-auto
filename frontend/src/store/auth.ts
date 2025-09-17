/**
 * Authentication store using Pinia
 */
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { ElMessage, ElNotification } from "element-plus";
import router from "@/router";
import { http } from "@/utils/request";
import { TokenManager, UserManager, AuthUtils } from "@/utils/auth";
import { AUTH_ENDPOINTS, NOTIFICATION_TYPES } from "@/utils/constants";
import type {
  UserInfo,
  LoginForm,
  AuthResponse,
  ProfileUpdateForm,
  PasswordChangeForm,
  TokenRefreshResponse,
} from "@/types/auth";

export const useAuthStore = defineStore("auth", () => {
  // State
  const user = ref<UserInfo | null>(null);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  // Computed
  const isAuthenticated = computed(() => {
    const token = TokenManager.getAccessToken();
    return !!token && TokenManager.isTokenValid(token) && !!user.value;
  });

  const userRole = computed(() => user.value?.role || null);
  const userPermissions = computed(() => user.value?.permissions || []);
  const token = computed(() => TokenManager.getAccessToken());

  // Actions
  const login = async (credentials: LoginForm): Promise<void> => {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await http.post<AuthResponse>(AUTH_ENDPOINTS.LOGIN, {
        username: credentials.username,
        password: credentials.password,
        remember: credentials.remember,
      });

      if (!response.success || !response.data) {
        throw new Error(response.error || "Login failed");
      }

      const responseData = response.data.data!;
      const userData = responseData.user;
      const accessToken = responseData.accessToken;
      const refreshToken = responseData.refreshToken;

      // Store tokens
      TokenManager.setAccessToken(accessToken);
      if (refreshToken) {
        TokenManager.setRefreshToken(refreshToken);
      }

      // Store user info
      user.value = userData;
      UserManager.setUserInfo(userData);

      ElMessage.success("Login successful");

      // Redirect to intended page or dashboard
      const redirectUrl = AuthUtils.getRedirectUrl();
      await router.push(redirectUrl);
    } catch (err: any) {
      error.value = err.message || "Login failed";
      ElMessage.error(error.value || "Login failed");
      throw err;
    } finally {
      isLoading.value = false;
    }
  };

  const logout = async (): Promise<void> => {
    isLoading.value = true;

    try {
      // Call logout API if user is authenticated
      if (isAuthenticated.value) {
        await http.post(AUTH_ENDPOINTS.LOGOUT);
      }
    } catch (err) {
      console.warn("Logout API call failed:", err);
    } finally {
      // Clear local state regardless of API success
      user.value = null;
      error.value = null;
      TokenManager.clearTokens();
      UserManager.clearUserInfo();

      ElMessage.info("Logged out successfully");

      // Redirect to login page
      await router.push("/login");
      isLoading.value = false;
    }
  };

  const refreshToken = async (): Promise<boolean> => {
    try {
      const refreshToken = TokenManager.getRefreshToken();
      if (!refreshToken) {
        return false;
      }

      const response = await http.post<TokenRefreshResponse>(
        AUTH_ENDPOINTS.REFRESH,
        {
          refresh_token: refreshToken,
        },
      );

      if (!response.success || !response.data) {
        return false;
      }

      const { accessToken, refreshToken: newRefreshToken } = response.data;

      TokenManager.setAccessToken(accessToken);
      if (newRefreshToken) {
        TokenManager.setRefreshToken(newRefreshToken);
      }

      return true;
    } catch (err) {
      console.error("Token refresh failed:", err);
      return false;
    }
  };

  const getCurrentUser = async (): Promise<void> => {
    try {
      const response = await http.get<UserInfo>(AUTH_ENDPOINTS.PROFILE);

      if (response.success && response.data) {
        user.value = response.data;
        UserManager.setUserInfo(response.data);
      }
    } catch (err: any) {
      console.error("Failed to get current user:", err);

      // If token is invalid, logout
      if (err.code === 401) {
        await logout();
      }

      throw err;
    }
  };

  const updateProfile = async (data: ProfileUpdateForm): Promise<void> => {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await http.put<UserInfo>(AUTH_ENDPOINTS.PROFILE, data);

      if (!response.success || !response.data) {
        throw new Error(response.error || "Profile update failed");
      }

      user.value = response.data;
      UserManager.setUserInfo(response.data);

      ElNotification({
        title: "Success",
        message: "Profile updated successfully",
        type: NOTIFICATION_TYPES.SUCCESS,
      });
    } catch (err: any) {
      error.value = err.message || "Profile update failed";
      ElNotification({
        title: "Error",
        message: error.value || "Profile update failed",
        type: NOTIFICATION_TYPES.ERROR,
      });
      throw err;
    } finally {
      isLoading.value = false;
    }
  };

  const changePassword = async (data: PasswordChangeForm): Promise<void> => {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await http.post("/auth/change-password", {
        current_password: data.currentPassword,
        new_password: data.newPassword,
      });

      if (!response.success) {
        throw new Error(response.error || "Password change failed");
      }

      ElNotification({
        title: "Success",
        message: "Password changed successfully",
        type: NOTIFICATION_TYPES.SUCCESS,
      });
    } catch (err: any) {
      error.value = err.message || "Password change failed";
      ElNotification({
        title: "Error",
        message: error.value || "Password change failed",
        type: NOTIFICATION_TYPES.ERROR,
      });
      throw err;
    } finally {
      isLoading.value = false;
    }
  };

  const hasPermission = (permission: string): boolean => {
    return UserManager.hasPermission(permission, user.value);
  };

  const hasRole = (role: string): boolean => {
    return UserManager.hasRole(role, user.value);
  };

  const canAccess = (route: string): boolean => {
    // Basic route access check - can be enhanced based on route metadata
    if (!isAuthenticated.value) {
      return false;
    }

    // Admin can access everything
    if (hasRole("admin")) {
      return true;
    }

    // Define route permissions
    const routePermissions: Record<string, string[]> = {
      "/containers": ["container:read"],
      "/images": ["image:read"],
      "/updates": ["update:read"],
      "/logs": ["log:read"],
      "/settings": ["admin"],
      "/users": ["admin"],
    };

    const requiredPermissions = routePermissions[route];
    if (!requiredPermissions) {
      return true; // No specific permissions required
    }

    return requiredPermissions.some((permission) => hasPermission(permission));
  };

  const initialize = async (): Promise<void> => {
    const token = TokenManager.getAccessToken();
    if (!token || !TokenManager.isTokenValid(token)) {
      return;
    }

    try {
      await getCurrentUser();

      // Check if token needs refresh
      if (TokenManager.needsRefresh(token)) {
        await refreshToken();
      }
    } catch (err) {
      console.error("Failed to initialize auth store:", err);
      // Clear invalid session
      await logout();
    }
  };

  const checkTokenExpiration = (): void => {
    const token = TokenManager.getAccessToken();
    if (!token) {
      return;
    }

    if (!TokenManager.isTokenValid(token)) {
      ElNotification({
        title: "Session Expired",
        message: "Your session has expired. Please log in again.",
        type: NOTIFICATION_TYPES.WARNING,
        duration: 0, // Don't auto close
      });
      logout();
    } else if (TokenManager.needsRefresh(token)) {
      refreshToken().catch(() => {
        ElNotification({
          title: "Session Expiring",
          message: "Your session is about to expire. Please save your work.",
          type: NOTIFICATION_TYPES.WARNING,
        });
      });
    }
  };

  // Set up periodic token check
  let tokenCheckInterval: NodeJS.Timeout | null = null;

  const startTokenCheck = (): void => {
    if (tokenCheckInterval) {
      clearInterval(tokenCheckInterval);
    }

    tokenCheckInterval = setInterval(() => {
      if (isAuthenticated.value) {
        checkTokenExpiration();
      }
    }, 60000); // Check every minute
  };

  const stopTokenCheck = (): void => {
    if (tokenCheckInterval) {
      clearInterval(tokenCheckInterval);
      tokenCheckInterval = null;
    }
  };

  return {
    // State
    user: readonly(user),
    isLoading: readonly(isLoading),
    error: readonly(error),

    // Computed
    isAuthenticated,
    userRole,
    userPermissions,
    token,

    // Actions
    login,
    logout,
    refreshToken,
    getCurrentUser,
    updateProfile,
    changePassword,
    hasPermission,
    hasRole,
    canAccess,
    initialize,
    checkTokenExpiration,
    startTokenCheck,
    stopTokenCheck,
  };
});

// Export convenience composable
export const useAuth = () => {
  const authStore = useAuthStore();

  return {
    ...authStore,

    // Additional convenience methods
    isAdmin: computed(() => authStore.hasRole("admin")),
    isOperator: computed(() => authStore.hasRole("operator")),
    isViewer: computed(() => authStore.hasRole("viewer")),

    userDisplayName: computed(() => {
      if (!authStore.user) return "";
      return AuthUtils.formatUserDisplayName(authStore.user);
    }),

    userAvatar: computed(() => {
      if (!authStore.user) return "";
      return AuthUtils.getUserAvatar(authStore.user);
    }),
  };
};
