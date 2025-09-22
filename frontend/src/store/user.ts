import { defineStore } from "pinia";
import { ref, readonly } from "vue";
import { ElMessage } from "element-plus";
import type { User, LoginRequest } from "@/types/user";
import { userAPI } from "@/api/user";
import { TokenManager } from "@/utils/auth";
import router from "@/router";

export const useUserStore = defineStore("user", () => {
  const user = ref<User | null>(null);
  const token = ref<string>(TokenManager.getAccessToken() || "");

  // Login action
  const login = async (loginData: LoginRequest): Promise<void> => {
    try {
      const response = await userAPI.login(loginData);

      // Store tokens
      token.value = response.token;
      TokenManager.setAccessToken(response.token);
      if (response.refresh_token) {
        TokenManager.setRefreshToken(response.refresh_token);
      }

      // Store user info
      user.value = response.user;

      // Login success message is handled by auth store
    } catch (error) {
      console.error("Login failed:", error);
      throw error;
    }
  };

  // Get current user info
  const getCurrentUser = async (): Promise<void> => {
    try {
      const userData = await userAPI.getCurrentUser();
      user.value = userData;
    } catch (error) {
      console.error("Failed to get user info:", error);
      // If token is invalid, logout
      logout();
      throw error;
    }
  };

  // Logout action
  const logout = async (): Promise<void> => {
    try {
      if (token.value) {
        await userAPI.logout();
      }
    } catch (error) {
      console.error("Logout API failed:", error);
    } finally {
      // Clear local state regardless of API success
      user.value = null;
      token.value = "";
      TokenManager.clearTokens();

      // Redirect to login page
      await router.push("/login");
      ElMessage.info("已退出登录");
    }
  };

  // Check if user has permission
  const hasPermission = (permission: string): boolean => {
    if (!user.value) return false;

    // Admin has all permissions
    if (user.value.role === "admin") return true;

    // Define role permissions
    const rolePermissions: Record<string, string[]> = {
      operator: [
        "container:read",
        "container:create",
        "container:update",
        "container:delete",
        "container:start",
        "container:stop",
        "update:read",
        "update:create",
        "logs:read",
      ],
      viewer: ["container:read", "update:read", "logs:read"],
    };

    const userPermissions = rolePermissions[user.value.role] || [];
    return userPermissions.includes(permission);
  };

  // Check if user has role
  const hasRole = (role: string): boolean => {
    return user.value?.role === role;
  };

  // Update user profile
  const updateProfile = async (profileData: import("@/types/user").UpdateProfileRequest): Promise<void> => {
    try {
      const updatedUser = await userAPI.updateProfile(profileData);
      user.value = updatedUser;
      ElMessage.success("用户资料更新成功");
    } catch (error) {
      console.error("Failed to update profile:", error);
      throw error;
    }
  };

  // Change password
  const changePassword = async (passwordData: import("@/types/user").ChangePasswordRequest): Promise<void> => {
    try {
      await userAPI.changePassword(passwordData);
      ElMessage.success("密码修改成功");
    } catch (error) {
      console.error("Failed to change password:", error);
      throw error;
    }
  };

  // Initialize store (called when app starts)
  const initialize = async (): Promise<void> => {
    if (token.value) {
      try {
        await getCurrentUser();
      } catch (error) {
        // If token is invalid, clear it
        logout();
      }
    }
  };

  return {
    user: readonly(user),
    token: readonly(token),
    login,
    getCurrentUser,
    logout,
    updateProfile,
    changePassword,
    hasPermission,
    hasRole,
    initialize,
  };
});
