/**
 * Authentication and authorization types
 */

// Re-export user types for convenience
export type {
  User,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  ChangePasswordRequest,
  UpdateProfileRequest,
} from "./user";

/**
 * Extended user information interface
 */
export interface UserInfo {
  id: string | number;
  username: string;
  email: string;
  firstName?: string;
  lastName?: string;
  avatar?: string;
  role: string;
  permissions: readonly string[];
  isActive: boolean;
  lastLoginAt?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * JWT token payload interface
 */
export interface TokenPayload {
  sub: string | number; // Subject (user ID)
  username: string;
  email: string;
  role: string;
  permissions: string[];
  iat: number; // Issued at
  exp: number; // Expiration time
  iss?: string; // Issuer
  aud?: string; // Audience
}

/**
 * Authentication state interface
 */
export interface AuthState {
  isAuthenticated: boolean;
  user: UserInfo | null;
  accessToken: string | null;
  refreshToken: string | null;
  isLoading: boolean;
  error: string | null;
}

/**
 * Login form data interface
 */
export interface LoginForm {
  username: string;
  password: string;
  remember: boolean;
}

/**
 * Login form validation rules
 */
export interface LoginValidationRules {
  username: Array<{
    required?: boolean;
    message: string;
    trigger?: string;
  }>;
  password: Array<{
    required?: boolean;
    min?: number;
    message: string;
    trigger?: string;
  }>;
}

/**
 * Registration form data interface
 */
export interface RegisterForm {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
  firstName?: string;
  lastName?: string;
  agreeToTerms: boolean;
}

/**
 * Password reset form interface
 */
export interface PasswordResetForm {
  email: string;
}

/**
 * Password reset confirmation form interface
 */
export interface PasswordResetConfirmForm {
  token: string;
  password: string;
  confirmPassword: string;
}

/**
 * Profile update form interface
 */
export interface ProfileUpdateForm {
  username?: string;
  email?: string;
  firstName?: string;
  lastName?: string;
  avatar?: string;
}

/**
 * Password change form interface
 */
export interface PasswordChangeForm {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

/**
 * Authentication response interface
 */
export interface AuthResponse {
  success: boolean;
  message?: string;
  data?: {
    user: UserInfo;
    accessToken: string;
    refreshToken: string;
    expiresIn: number;
  };
  error?: string;
}

/**
 * Token refresh response interface
 */
export interface TokenRefreshResponse {
  accessToken: string;
  refreshToken?: string;
  expiresIn: number;
}

/**
 * Permission check interface
 */
export interface PermissionCheck {
  hasPermission: boolean;
  requiredPermission?: string;
  userPermissions?: string[];
}

/**
 * Role check interface
 */
export interface RoleCheck {
  hasRole: boolean;
  requiredRole?: string;
  userRole?: string;
}

/**
 * Route access interface
 */
export interface RouteAccess {
  isAllowed: boolean;
  reason?: string;
  redirectTo?: string;
}

/**
 * Authentication error interface
 */
export interface AuthError {
  code: string;
  message: string;
  details?: Record<string, any>;
}

/**
 * Session information interface
 */
export interface SessionInfo {
  id: string;
  userId: string | number;
  ipAddress: string;
  userAgent: string;
  lastActivity: string;
  isActive: boolean;
  expiresAt: string;
}

/**
 * Two-factor authentication interface
 */
export interface TwoFactorAuth {
  isEnabled: boolean;
  qrCode?: string;
  backupCodes?: string[];
  lastUsed?: string;
}

/**
 * OAuth provider interface
 */
export interface OAuthProvider {
  name: string;
  displayName: string;
  icon: string;
  enabled: boolean;
  clientId?: string;
}

/**
 * Security settings interface
 */
export interface SecuritySettings {
  twoFactorEnabled: boolean;
  passwordLastChanged: string;
  activeSessions: SessionInfo[];
  recentActivity: Array<{
    action: string;
    timestamp: string;
    ipAddress: string;
    userAgent: string;
  }>;
}

/**
 * User preferences interface
 */
export interface UserPreferences {
  theme: "light" | "dark" | "auto";
  language: string;
  timezone: string;
  notifications: {
    email: boolean;
    push: boolean;
    desktop: boolean;
  };
  dashboard: {
    layout: string;
    widgets: string[];
  };
}

/**
 * API key interface
 */
export interface ApiKey {
  id: string;
  name: string;
  key: string;
  permissions: string[];
  expiresAt?: string;
  lastUsed?: string;
  isActive: boolean;
  createdAt: string;
}

/**
 * Audit log interface
 */
export interface AuditLog {
  id: string;
  userId: string | number;
  action: string;
  resource: string;
  resourceId?: string;
  details: Record<string, any>;
  ipAddress: string;
  userAgent: string;
  timestamp: string;
}

/**
 * Login attempt interface
 */
export interface LoginAttempt {
  id: string;
  username: string;
  ipAddress: string;
  userAgent: string;
  success: boolean;
  failureReason?: string;
  timestamp: string;
}

/**
 * Account lockout interface
 */
export interface AccountLockout {
  isLocked: boolean;
  lockedAt?: string;
  lockoutDuration: number;
  failedAttempts: number;
  maxAttempts: number;
}

/**
 * Authentication context interface
 */
export interface AuthContext {
  user: UserInfo | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (credentials: LoginForm) => Promise<AuthResponse>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<boolean>;
  updateProfile: (data: ProfileUpdateForm) => Promise<boolean>;
  changePassword: (data: PasswordChangeForm) => Promise<boolean>;
  hasPermission: (permission: string) => boolean;
  hasRole: (role: string) => boolean;
  canAccess: (route: string) => boolean;
}

/**
 * Route meta interface for authentication
 */
export interface RouteMeta {
  requiresAuth?: boolean;
  requiredRole?: string;
  requiredPermissions?: string[];
  allowAnonymous?: boolean;
  title?: string;
  icon?: string;
}

/**
 * Navigation guard context interface
 */
export interface NavigationGuardContext {
  to: {
    path: string;
    name?: string;
    meta?: RouteMeta;
    query?: Record<string, string>;
    params?: Record<string, string>;
  };
  from: {
    path: string;
    name?: string;
  };
  next: (route?: string | boolean) => void;
}
