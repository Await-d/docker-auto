/**
 * Authentication store using Pinia
 */
import { defineStore } from "pinia";
import { ref, computed, nextTick } from "vue";
import { ElMessage, ElNotification } from "element-plus";
import router from "@/router";
import { http } from "@/utils/request";
import { TokenManager, UserManager, AuthUtils } from "@/utils/auth";
import { AUTH_ENDPOINTS, NOTIFICATION_TYPES } from "@/utils/constants";
import type {
  UserInfo,
  LoginForm,
  LoginResponseData,
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
      const response = await http.post<LoginResponseData>(AUTH_ENDPOINTS.LOGIN, {
        username: credentials.username,
        password: credentials.password,
        remember: credentials.remember,
      }, {
        showLoading: true,
        showError: false, // 让组件处理错误
        skipTokenRefresh: true, // 登录API的401错误不应该触发token刷新
      });

      let userData, accessToken, refreshToken;

      // Check if response has the expected structure
      if (!response || !response.success) {
        throw new Error(response?.message || "登录失败");
      }

      // Handle the standard API response format
      if (response.data && response.data.user && response.data.token_info) {
        // Our current backend format: { data: { user: {...}, token_info: {...} } }
        userData = response.data.user;
        accessToken = response.data.token_info.access_token;
        refreshToken = response.data.token_info.refresh_token;
      } else {
        throw new Error("登录响应格式错误");
      }

      // Store tokens
      TokenManager.setAccessToken(accessToken);
      if (refreshToken) {
        TokenManager.setRefreshToken(refreshToken);
      }

      // Store user info
      user.value = userData;
      UserManager.setUserInfo(userData);

      ElMessage.success("登录成功");

      // Wait for user data to be fully reactive and available
      await new Promise(resolve => setTimeout(resolve, 300));

      // Force reactive update by triggering a re-assignment
      user.value = { ...userData };

      // Wait for Vue reactivity to settle
      await nextTick();

      // Additional wait to ensure all computed properties are updated
      await new Promise(resolve => setTimeout(resolve, 100));

      // Redirect to intended page or dashboard
      const redirectUrl = AuthUtils.getRedirectUrl();
      await router.push(redirectUrl);
    } catch (err: any) {
      error.value = err.message || "登录失败";
      // Don't show error here, let the component handle it
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
      console.warn("退出登录API调用失败:", err);
    } finally {
      // Clear local state regardless of API success
      user.value = null;
      error.value = null;
      TokenManager.clearTokens();
      UserManager.clearUserInfo();

      ElMessage.info("退出登录成功");

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
      console.error("token刷新失败:", err);
      return false;
    }
  };

  const getCurrentUser = async (): Promise<void> => {
    try {
      const response = await http.get<{ user: UserInfo }>(AUTH_ENDPOINTS.PROFILE);

      if (response.success && response.data && response.data.user) {
        user.value = response.data.user;
        UserManager.setUserInfo(response.data.user);
      }
    } catch (err: any) {
      console.error("获取当前用户失败:", err);

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
        throw new Error(response.error || "个人资料更新失败");
      }

      user.value = response.data;
      UserManager.setUserInfo(response.data);

      ElNotification({
        title: "成功",
        message: "个人资料更新成功",
        type: NOTIFICATION_TYPES.SUCCESS,
      });
    } catch (err: any) {
      error.value = err.message || "个人资料更新失败";
      ElNotification({
        title: "错误",
        message: error.value || "个人资料更新失败",
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
        throw new Error(response.error || "密码修改失败");
      }

      ElNotification({
        title: "成功",
        message: "密码修改成功",
        type: NOTIFICATION_TYPES.SUCCESS,
      });
    } catch (err: any) {
      error.value = err.message || "密码修改失败";
      ElNotification({
        title: "错误",
        message: error.value || "密码修改失败",
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
      "/updates": ["update:create"],
      "/logs": ["system:logs"],
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

    // If no token, clear any existing user data
    if (!token || !TokenManager.isTokenValid(token)) {
      user.value = null;
      UserManager.clearUserInfo();
      return;
    }

    // Try to restore user from localStorage first
    const cachedUser = UserManager.getUserInfo();
    if (cachedUser && !user.value) {
      user.value = cachedUser;
    }

    // If we already have user data and token is valid, no need to fetch again
    if (user.value && TokenManager.isTokenValid(token)) {
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
        title: "会话过期",
        message: "您的会话已过期，请重新登录。",
        type: NOTIFICATION_TYPES.WARNING,
        duration: 0, // Don't auto close
      });
      logout();
    } else if (TokenManager.needsRefresh(token)) {
      refreshToken().catch(() => {
        ElNotification({
          title: "会话即将过期",
          message: "您的会话即将过期，请保存您的工作。",
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
