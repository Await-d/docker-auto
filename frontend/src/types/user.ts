// User related types

export interface User {
  id: number;
  username: string;
  email: string;
  role: "admin" | "operator" | "viewer";
  avatar_url?: string;
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
  remember?: boolean;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface UpdateProfileRequest {
  username?: string;
  email?: string;
  avatar_url?: string;
}
