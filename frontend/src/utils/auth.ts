/**
 * Authentication utilities
 */
import { jwtDecode } from "jwt-decode";
import Cookies from "js-cookie";
import { STORAGE_KEYS, TOKEN_REFRESH_THRESHOLD } from "./constants";
import type { UserInfo, TokenPayload } from "@/types/auth";

/**
 * JWT token management class
 */
export class TokenManager {
  /**
   * Store access token
   */
  static setAccessToken(token: string): void {
    localStorage.setItem(STORAGE_KEYS.ACCESS_TOKEN, token);
    Cookies.set(STORAGE_KEYS.ACCESS_TOKEN, token, {
      secure: true,
      sameSite: "strict",
    });
  }

  /**
   * Get access token
   */
  static getAccessToken(): string | null {
    return (
      localStorage.getItem(STORAGE_KEYS.ACCESS_TOKEN) ||
      Cookies.get(STORAGE_KEYS.ACCESS_TOKEN) ||
      null
    );
  }

  /**
   * Store refresh token
   */
  static setRefreshToken(token: string): void {
    localStorage.setItem(STORAGE_KEYS.REFRESH_TOKEN, token);
    Cookies.set(STORAGE_KEYS.REFRESH_TOKEN, token, {
      secure: true,
      sameSite: "strict",
      httpOnly: false, // Allow JavaScript access for refresh functionality
    });
  }

  /**
   * Get refresh token
   */
  static getRefreshToken(): string | null {
    return (
      localStorage.getItem(STORAGE_KEYS.REFRESH_TOKEN) ||
      Cookies.get(STORAGE_KEYS.REFRESH_TOKEN) ||
      null
    );
  }

  /**
   * Clear all tokens
   */
  static clearTokens(): void {
    localStorage.removeItem(STORAGE_KEYS.ACCESS_TOKEN);
    localStorage.removeItem(STORAGE_KEYS.REFRESH_TOKEN);
    Cookies.remove(STORAGE_KEYS.ACCESS_TOKEN);
    Cookies.remove(STORAGE_KEYS.REFRESH_TOKEN);
  }

  /**
   * Decode JWT token
   */
  static decodeToken(token: string): TokenPayload | null {
    try {
      return jwtDecode<TokenPayload>(token);
    } catch (error) {
      console.error("Failed to decode token:", error);
      return null;
    }
  }

  /**
   * Check if token is valid (not expired)
   */
  static isTokenValid(token: string): boolean {
    try {
      const decoded = this.decodeToken(token);
      if (!decoded || !decoded.exp) {
        return false;
      }

      const currentTime = Math.floor(Date.now() / 1000);
      return decoded.exp > currentTime;
    } catch (error) {
      return false;
    }
  }

  /**
   * Check if token needs refresh (expires within threshold)
   */
  static needsRefresh(token: string): boolean {
    try {
      const decoded = this.decodeToken(token);
      if (!decoded || !decoded.exp) {
        return true;
      }

      const currentTime = Math.floor(Date.now() / 1000);
      const thresholdTime = TOKEN_REFRESH_THRESHOLD * 60; // Convert minutes to seconds

      return decoded.exp - currentTime < thresholdTime;
    } catch (error) {
      return true;
    }
  }

  /**
   * Get token expiration time
   */
  static getTokenExpiration(token: string): Date | null {
    try {
      const decoded = this.decodeToken(token);
      if (!decoded || !decoded.exp) {
        return null;
      }

      return new Date(decoded.exp * 1000);
    } catch (error) {
      return null;
    }
  }

  /**
   * Get user info from token
   */
  static getUserInfoFromToken(token: string): Partial<UserInfo> | null {
    try {
      const decoded = this.decodeToken(token);
      if (!decoded) {
        return null;
      }

      return {
        id: decoded.sub,
        username: decoded.username,
        email: decoded.email,
        role: decoded.role,
        permissions: decoded.permissions,
      };
    } catch (error) {
      return null;
    }
  }
}

/**
 * User information management
 */
export class UserManager {
  /**
   * Store user information
   */
  static setUserInfo(userInfo: UserInfo): void {
    localStorage.setItem(STORAGE_KEYS.USER_INFO, JSON.stringify(userInfo));
  }

  /**
   * Get user information
   */
  static getUserInfo(): UserInfo | null {
    try {
      const userInfoStr = localStorage.getItem(STORAGE_KEYS.USER_INFO);
      return userInfoStr ? JSON.parse(userInfoStr) : null;
    } catch (error) {
      console.error("Failed to parse user info:", error);
      return null;
    }
  }

  /**
   * Clear user information
   */
  static clearUserInfo(): void {
    localStorage.removeItem(STORAGE_KEYS.USER_INFO);
  }

  /**
   * Check if user has permission
   */
  static hasPermission(
    permission: string,
    userInfo?: UserInfo | null,
  ): boolean {
    const user = userInfo || this.getUserInfo();
    if (!user || !user.permissions) {
      return false;
    }

    return user.permissions.includes(permission);
  }

  /**
   * Check if user has role
   */
  static hasRole(role: string, userInfo?: UserInfo | null): boolean {
    const user = userInfo || this.getUserInfo();
    if (!user) {
      return false;
    }

    return user.role === role;
  }

  /**
   * Check if user is admin
   */
  static isAdmin(userInfo?: UserInfo | null): boolean {
    return this.hasRole("admin", userInfo);
  }

  /**
   * Check if user can perform action
   */
  static canPerformAction(
    requiredRole?: string,
    requiredPermission?: string,
  ): boolean {
    const userInfo = this.getUserInfo();
    if (!userInfo) {
      return false;
    }

    // Check role if required
    if (requiredRole && !this.hasRole(requiredRole, userInfo)) {
      return false;
    }

    // Check permission if required
    if (
      requiredPermission &&
      !this.hasPermission(requiredPermission, userInfo)
    ) {
      return false;
    }

    return true;
  }
}

/**
 * Authentication utilities
 */
export class AuthUtils {
  /**
   * Check if user is authenticated
   */
  static isAuthenticated(): boolean {
    const token = TokenManager.getAccessToken();
    return token !== null && TokenManager.isTokenValid(token);
  }

  /**
   * Perform logout cleanup
   */
  static logout(): void {
    TokenManager.clearTokens();
    UserManager.clearUserInfo();

    // Clear any other auth-related data
    localStorage.removeItem(STORAGE_KEYS.THEME);
    localStorage.removeItem(STORAGE_KEYS.LANGUAGE);

    // Redirect to login page
    window.location.href = "/login";
  }

  /**
   * Redirect to login page
   */
  static redirectToLogin(returnUrl?: string): void {
    const loginUrl = "/login";
    const url = returnUrl
      ? `${loginUrl}?redirect=${encodeURIComponent(returnUrl)}`
      : loginUrl;
    window.location.href = url;
  }

  /**
   * Get redirect URL from query params
   */
  static getRedirectUrl(): string {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get("redirect") || "/";
  }

  /**
   * Generate CSRF token
   */
  static generateCSRFToken(): string {
    const array = new Uint8Array(32);
    crypto.getRandomValues(array);
    return Array.from(array, (byte) => byte.toString(16).padStart(2, "0")).join(
      "",
    );
  }

  /**
   * Sanitize input to prevent XSS
   */
  static sanitizeInput(input: string): string {
    const div = document.createElement("div");
    div.textContent = input;
    return div.innerHTML;
  }

  /**
   * Validate password strength
   */
  static validatePassword(password: string): {
    isValid: boolean;
    errors: string[];
  } {
    const errors: string[] = [];

    if (password.length < 8) {
      errors.push("Password must be at least 8 characters long");
    }

    if (!/[A-Z]/.test(password)) {
      errors.push("Password must contain at least one uppercase letter");
    }

    if (!/[a-z]/.test(password)) {
      errors.push("Password must contain at least one lowercase letter");
    }

    if (!/\d/.test(password)) {
      errors.push("Password must contain at least one number");
    }

    if (!/[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(password)) {
      errors.push("Password must contain at least one special character");
    }

    return {
      isValid: errors.length === 0,
      errors,
    };
  }

  /**
   * Format user display name
   */
  static formatUserDisplayName(userInfo: UserInfo): string {
    if (userInfo.firstName && userInfo.lastName) {
      return `${userInfo.firstName} ${userInfo.lastName}`;
    }

    if (userInfo.firstName) {
      return userInfo.firstName;
    }

    if (userInfo.email) {
      return userInfo.email;
    }

    return userInfo.username;
  }

  /**
   * Get user avatar URL or generate initials
   */
  static getUserAvatar(userInfo: UserInfo): string {
    if (userInfo.avatar) {
      return userInfo.avatar;
    }

    // Generate initials-based avatar
    const displayName = this.formatUserDisplayName(userInfo);
    const initials = displayName
      .split(" ")
      .map((word) => word.charAt(0).toUpperCase())
      .join("")
      .substring(0, 2);

    // Return a data URL for a simple avatar with initials
    const canvas = document.createElement("canvas");
    canvas.width = 40;
    canvas.height = 40;
    const ctx = canvas.getContext("2d");

    if (ctx) {
      // Background
      ctx.fillStyle = "#409EFF";
      ctx.fillRect(0, 0, 40, 40);

      // Text
      ctx.fillStyle = "#FFFFFF";
      ctx.font = "16px Arial";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(initials, 20, 20);
    }

    return canvas.toDataURL();
  }
}

/**
 * Security utilities
 */
export class SecurityUtils {
  /**
   * Generate secure random string
   */
  static generateRandomString(length: number = 32): string {
    const chars =
      "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    let result = "";
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);

    for (let i = 0; i < length; i++) {
      result += chars[array[i] % chars.length];
    }

    return result;
  }

  /**
   * Hash string using Web Crypto API
   */
  static async hashString(str: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(str);
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
  }

  /**
   * Constant-time string comparison
   */
  static constantTimeEquals(a: string, b: string): boolean {
    if (a.length !== b.length) {
      return false;
    }

    let result = 0;
    for (let i = 0; i < a.length; i++) {
      result |= a.charCodeAt(i) ^ b.charCodeAt(i);
    }

    return result === 0;
  }
}
