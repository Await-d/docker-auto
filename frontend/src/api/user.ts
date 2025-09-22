/**
 * User authentication and management API service
 */
import { get, post, put, del } from "@/utils/request";
import type {
  User,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  ChangePasswordRequest,
  UpdateProfileRequest,
} from "@/types/user";
import type { TokenRefreshResponse } from "@/types/auth";

export interface UserListResponse {
  users: User[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  role: "admin" | "operator" | "viewer";
  is_active?: boolean;
}

export interface UserStatsResponse {
  totalUsers: number;
  activeUsers: number;
  adminUsers: number;
  operatorUsers: number;
  viewerUsers: number;
  recentLogins: number;
}

export const userAPI = {
  /**
   * User authentication
   */
  async login(loginData: LoginRequest): Promise<LoginResponse> {
    return post<LoginResponse>("/api/auth/login", loginData, {
      showLoading: true,
      showError: false, // Let login component handle errors for better UX
      skipTokenRefresh: true, // 登录API的401错误不应该触发token刷新
    });
  },

  /**
   * Logout user
   */
  async logout(): Promise<void> {
    return post<void>("/api/auth/logout", {}, {
      showLoading: false,
      showError: false,
    });
  },

  /**
   * Refresh access token
   */
  async refreshToken(refreshTokenData: RefreshTokenRequest): Promise<TokenRefreshResponse> {
    return post<TokenRefreshResponse>("/api/auth/refresh", refreshTokenData, {
      showLoading: false,
      showError: false,
      skipTokenRefresh: true, // 刷新token API的401错误不应该触发token刷新（避免无限循环）
    });
  },

  /**
   * Get current user information
   */
  async getCurrentUser(): Promise<User> {
    return get<User>("/api/auth/me", {
      showLoading: false,
      showError: true,
    });
  },

  /**
   * Update current user profile
   */
  async updateProfile(profileData: UpdateProfileRequest): Promise<User> {
    return put<User>("/api/auth/profile", profileData, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Change password
   */
  async changePassword(passwordData: ChangePasswordRequest): Promise<void> {
    return post<void>("/api/auth/change-password", passwordData, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Reset password (forgot password)
   */
  async resetPassword(email: string): Promise<void> {
    return post<void>("/api/auth/reset-password", { email }, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Confirm password reset
   */
  async confirmPasswordReset(token: string, newPassword: string): Promise<void> {
    return post<void>("/api/auth/confirm-reset", { token, new_password: newPassword }, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * User management (admin only)
   */
  async getUserList(
    page = 1,
    pageSize = 20,
    search?: string,
    role?: string,
    isActive?: boolean
  ): Promise<UserListResponse> {
    const params = new URLSearchParams({
      page: page.toString(),
      pageSize: pageSize.toString(),
    });

    if (search) params.set("search", search);
    if (role) params.set("role", role);
    if (isActive !== undefined) params.set("is_active", isActive.toString());

    return get<UserListResponse>(`/api/users?${params}`, {
      showLoading: true,
    });
  },

  /**
   * Get user by ID
   */
  async getUserById(userId: number): Promise<User> {
    return get<User>(`/api/users/${userId}`, {
      showLoading: true,
    });
  },

  /**
   * Create new user (admin only)
   */
  async createUser(userData: CreateUserRequest): Promise<User> {
    return post<User>("/api/users", userData, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Update user (admin only)
   */
  async updateUser(userId: number, userData: Partial<CreateUserRequest>): Promise<User> {
    return put<User>(`/api/users/${userId}`, userData, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Delete user (admin only)
   */
  async deleteUser(userId: number): Promise<void> {
    return del<void>(`/api/users/${userId}`, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Activate/Deactivate user (admin only)
   */
  async toggleUserStatus(userId: number, isActive: boolean): Promise<User> {
    return put<User>(`/api/users/${userId}/status`, { is_active: isActive }, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Get user statistics
   */
  async getUserStats(): Promise<UserStatsResponse> {
    return get<UserStatsResponse>("/api/users/stats", {
      showLoading: false,
    });
  },

  /**
   * Get user activity log
   */
  async getUserActivity(
    userId?: number,
    page = 1,
    pageSize = 50,
    dateFrom?: string,
    dateTo?: string
  ): Promise<{
    activities: Array<{
      id: string;
      userId: number;
      username: string;
      action: string;
      resource: string;
      resourceId?: string;
      ipAddress: string;
      userAgent: string;
      timestamp: string;
      details?: Record<string, any>;
    }>;
    total: number;
    page: number;
    pageSize: number;
  }> {
    const params = new URLSearchParams({
      page: page.toString(),
      pageSize: pageSize.toString(),
    });

    if (userId) params.set("userId", userId.toString());
    if (dateFrom) params.set("dateFrom", dateFrom);
    if (dateTo) params.set("dateTo", dateTo);

    return get<{
      activities: Array<{
        id: string;
        userId: number;
        username: string;
        action: string;
        resource: string;
        resourceId?: string;
        ipAddress: string;
        userAgent: string;
        timestamp: string;
        details?: Record<string, any>;
      }>;
      total: number;
      page: number;
      pageSize: number;
    }>(`/api/users/activity?${params}`, {
      showLoading: true,
    });
  },

  /**
   * Get user permissions
   */
  async getUserPermissions(userId?: number): Promise<{
    permissions: string[];
    role: string;
    effectivePermissions: string[];
  }> {
    const url = userId ? `/api/users/${userId}/permissions` : "/api/auth/permissions";
    return get<{
      permissions: string[];
      role: string;
      effectivePermissions: string[];
    }>(url, {
      showLoading: false,
    });
  },

  /**
   * Update user permissions (admin only)
   */
  async updateUserPermissions(
    userId: number,
    permissions: string[]
  ): Promise<void> {
    return put<void>(`/api/users/${userId}/permissions`, { permissions }, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Get user sessions
   */
  async getUserSessions(userId?: number): Promise<Array<{
    id: string;
    userId: number;
    deviceInfo: string;
    ipAddress: string;
    location?: string;
    createdAt: string;
    lastActivity: string;
    isActive: boolean;
  }>> {
    const url = userId ? `/api/users/${userId}/sessions` : "/api/auth/sessions";
    return get<Array<{
      id: string;
      userId: number;
      deviceInfo: string;
      ipAddress: string;
      location?: string;
      createdAt: string;
      lastActivity: string;
      isActive: boolean;
    }>>(url, {
      showLoading: true,
    });
  },

  /**
   * Revoke user session
   */
  async revokeSession(sessionId: string): Promise<void> {
    return del<void>(`/api/auth/sessions/${sessionId}`, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Revoke all sessions except current
   */
  async revokeAllOtherSessions(): Promise<void> {
    return post<void>("/api/auth/revoke-sessions", {}, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Enable/Disable two-factor authentication
   */
  async setupTwoFactor(): Promise<{
    secret: string;
    qrCode: string;
    backupCodes: string[];
  }> {
    return post<{
      secret: string;
      qrCode: string;
      backupCodes: string[];
    }>("/api/auth/2fa/setup", {}, {
      showLoading: true,
    });
  },

  /**
   * Verify and enable two-factor authentication
   */
  async enableTwoFactor(token: string): Promise<{
    backupCodes: string[];
  }> {
    return post<{
      backupCodes: string[];
    }>("/api/auth/2fa/enable", { token }, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Disable two-factor authentication
   */
  async disableTwoFactor(password: string): Promise<void> {
    return post<void>("/api/auth/2fa/disable", { password }, {
      showLoading: true,
      showSuccess: true,
    });
  },

  /**
   * Generate new backup codes
   */
  async generateBackupCodes(): Promise<{
    backupCodes: string[];
  }> {
    return post<{
      backupCodes: string[];
    }>("/api/auth/2fa/backup-codes", {}, {
      showLoading: true,
      showSuccess: true,
    });
  },
};