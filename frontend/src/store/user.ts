import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User, LoginRequest, LoginResponse } from '@/types/user'
import { login as apiLogin, getCurrentUser as apiGetCurrentUser, logout as apiLogout } from '@/api/user'
import { getToken, setToken, removeToken } from '@/utils/auth'
import router from '@/router'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const token = ref<string>(getToken() || '')

  // Login action
  const login = async (loginData: LoginRequest): Promise<void> => {
    try {
      const response: LoginResponse = await apiLogin(loginData)
      token.value = response.token
      setToken(response.token)

      // Get user info after login
      await getCurrentUser()

      ElMessage.success('登录成功')
    } catch (error) {
      console.error('Login failed:', error)
      throw error
    }
  }

  // Get current user info
  const getCurrentUser = async (): Promise<void> => {
    try {
      const userData = await apiGetCurrentUser()
      user.value = userData
    } catch (error) {
      console.error('Failed to get user info:', error)
      // If token is invalid, logout
      logout()
      throw error
    }
  }

  // Logout action
  const logout = async (): Promise<void> => {
    try {
      if (token.value) {
        await apiLogout()
      }
    } catch (error) {
      console.error('Logout API failed:', error)
    } finally {
      // Clear local state regardless of API success
      user.value = null
      token.value = ''
      removeToken()

      // Redirect to login page
      await router.push('/login')
      ElMessage.info('已退出登录')
    }
  }

  // Check if user has permission
  const hasPermission = (permission: string): boolean => {
    if (!user.value) return false

    // Admin has all permissions
    if (user.value.role === 'admin') return true

    // Define role permissions
    const rolePermissions: Record<string, string[]> = {
      operator: [
        'container:read',
        'container:create',
        'container:update',
        'container:delete',
        'container:start',
        'container:stop',
        'update:read',
        'update:create',
        'logs:read'
      ],
      viewer: [
        'container:read',
        'update:read',
        'logs:read'
      ]
    }

    const userPermissions = rolePermissions[user.value.role] || []
    return userPermissions.includes(permission)
  }

  // Check if user has role
  const hasRole = (role: string): boolean => {
    return user.value?.role === role
  }

  // Initialize store (called when app starts)
  const initialize = async (): Promise<void> => {
    if (token.value) {
      try {
        await getCurrentUser()
      } catch (error) {
        // If token is invalid, clear it
        logout()
      }
    }
  }

  return {
    user: readonly(user),
    token: readonly(token),
    login,
    getCurrentUser,
    logout,
    hasPermission,
    hasRole,
    initialize
  }
})