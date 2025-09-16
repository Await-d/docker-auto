/**
 * HTTP request utilities with Axios
 */
import axios, {
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
  AxiosError,
  InternalAxiosRequestConfig
} from 'axios'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import NProgress from 'nprogress'
import {
  API_BASE_URL,
  REQUEST_TIMEOUT,
  HTTP_STATUS,
  NOTIFICATION_TYPES
} from './constants'
import { TokenManager, AuthUtils } from './auth'
import type { AuthResponse, TokenRefreshResponse } from '@/types/auth'

/**
 * Request/Response data interfaces
 */
export interface ApiResponse<T = any> {
  success: boolean
  message?: string
  data?: T
  error?: string
  code?: number
  timestamp?: string
}

export interface ApiError {
  code: number
  message: string
  details?: Record<string, any>
  timestamp: string
}

export interface RequestOptions extends AxiosRequestConfig {
  showLoading?: boolean
  showError?: boolean
  showSuccess?: boolean
  retryTimes?: number
  retryDelay?: number
}

/**
 * HTTP request class
 */
class HttpRequest {
  private instance: AxiosInstance
  private isRefreshing = false
  private failedQueue: Array<{
    resolve: (value: any) => void
    reject: (error: any) => void
  }> = []

  constructor() {
    this.instance = axios.create({
      baseURL: API_BASE_URL,
      timeout: REQUEST_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      }
    })

    this.setupInterceptors()
  }

  /**
   * Setup request and response interceptors
   */
  private setupInterceptors(): void {
    // Request interceptor
    this.instance.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        // Show loading progress
        if (config.metadata?.showLoading !== false) {
          NProgress.start()
        }

        // Add authentication token
        const token = TokenManager.getAccessToken()
        if (token && TokenManager.isTokenValid(token)) {
          config.headers.Authorization = `Bearer ${token}`
        }

        // Add CSRF token for state-changing requests
        if (['post', 'put', 'patch', 'delete'].includes(config.method?.toLowerCase() || '')) {
          const csrfToken = AuthUtils.generateCSRFToken()
          config.headers['X-CSRF-Token'] = csrfToken
        }

        // Add request timestamp
        config.headers['X-Request-Time'] = Date.now().toString()

        return config
      },
      (error: AxiosError) => {
        NProgress.done()
        console.error('Request interceptor error:', error)
        return Promise.reject(this.handleError(error))
      }
    )

    // Response interceptor
    this.instance.interceptors.response.use(
      (response: AxiosResponse) => {
        NProgress.done()

        // Show success message if configured
        if (response.config.metadata?.showSuccess && response.data?.message) {
          ElMessage.success(response.data.message)
        }

        return response
      },
      async (error: AxiosError) => {
        NProgress.done()

        const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

        // Handle token refresh for 401 errors
        if (error.response?.status === HTTP_STATUS.UNAUTHORIZED && !originalRequest._retry) {
          originalRequest._retry = true

          if (!this.isRefreshing) {
            this.isRefreshing = true

            try {
              const refreshToken = TokenManager.getRefreshToken()
              if (refreshToken) {
                const newTokens = await this.refreshAccessToken(refreshToken)
                TokenManager.setAccessToken(newTokens.accessToken)

                if (newTokens.refreshToken) {
                  TokenManager.setRefreshToken(newTokens.refreshToken)
                }

                // Process failed requests queue
                this.processQueue(null, newTokens.accessToken)

                // Retry original request with new token
                originalRequest.headers.Authorization = `Bearer ${newTokens.accessToken}`
                return this.instance(originalRequest)
              }
            } catch (refreshError) {
              this.processQueue(refreshError, null)
              AuthUtils.logout()
              return Promise.reject(refreshError)
            } finally {
              this.isRefreshing = false
            }
          }

          // Queue the request if token refresh is in progress
          return new Promise((resolve, reject) => {
            this.failedQueue.push({
              resolve: (token: string) => {
                originalRequest.headers.Authorization = `Bearer ${token}`
                resolve(this.instance(originalRequest))
              },
              reject: (err: any) => {
                reject(err)
              }
            })
          })
        }

        // Handle other errors
        const processedError = this.handleError(error)

        // Show error message if configured
        if (originalRequest.metadata?.showError !== false) {
          this.showErrorMessage(processedError)
        }

        return Promise.reject(processedError)
      }
    )
  }

  /**
   * Process failed requests queue
   */
  private processQueue(error: any, token: string | null): void {
    this.failedQueue.forEach(({ resolve, reject }) => {
      if (error) {
        reject(error)
      } else {
        resolve(token)
      }
    })

    this.failedQueue = []
  }

  /**
   * Refresh access token
   */
  private async refreshAccessToken(refreshToken: string): Promise<TokenRefreshResponse> {
    const response = await axios.post<ApiResponse<TokenRefreshResponse>>(
      `${API_BASE_URL}/auth/refresh`,
      { refresh_token: refreshToken },
      {
        headers: {
          'Content-Type': 'application/json'
        }
      }
    )

    if (!response.data.success || !response.data.data) {
      throw new Error('Failed to refresh token')
    }

    return response.data.data
  }

  /**
   * Handle and transform errors
   */
  private handleError(error: AxiosError): ApiError {
    const timestamp = new Date().toISOString()

    if (error.response) {
      // Server responded with error status
      const { status, data } = error.response

      return {
        code: status,
        message: (data as any)?.message || this.getStatusMessage(status),
        details: data as any,
        timestamp
      }
    } else if (error.request) {
      // Request was made but no response received
      return {
        code: 0,
        message: 'Network error - please check your connection',
        details: { request: error.request },
        timestamp
      }
    } else {
      // Something else happened
      return {
        code: -1,
        message: error.message || 'An unexpected error occurred',
        details: { error: error.toJSON() },
        timestamp
      }
    }
  }

  /**
   * Get human-readable status message
   */
  private getStatusMessage(status: number): string {
    const messages: Record<number, string> = {
      [HTTP_STATUS.BAD_REQUEST]: 'Bad request - please check your input',
      [HTTP_STATUS.UNAUTHORIZED]: 'Authentication required - please log in',
      [HTTP_STATUS.FORBIDDEN]: 'Access denied - insufficient permissions',
      [HTTP_STATUS.NOT_FOUND]: 'Resource not found',
      [HTTP_STATUS.INTERNAL_SERVER_ERROR]: 'Server error - please try again later'
    }

    return messages[status] || `HTTP Error ${status}`
  }

  /**
   * Show error message to user
   */
  private showErrorMessage(error: ApiError): void {
    const isAuthError = error.code === HTTP_STATUS.UNAUTHORIZED || error.code === HTTP_STATUS.FORBIDDEN

    if (isAuthError) {
      ElMessageBox.alert(error.message, 'Authentication Error', {
        type: 'error',
        callback: () => {
          if (error.code === HTTP_STATUS.UNAUTHORIZED) {
            AuthUtils.redirectToLogin()
          }
        }
      })
    } else {
      ElNotification({
        title: 'Error',
        message: error.message,
        type: NOTIFICATION_TYPES.ERROR,
        duration: 5000
      })
    }
  }

  /**
   * Generic request method
   */
  async request<T = any>(config: RequestOptions): Promise<ApiResponse<T>> {
    try {
      // Add metadata to config for interceptors
      config.metadata = {
        showLoading: config.showLoading ?? true,
        showError: config.showError ?? true,
        showSuccess: config.showSuccess ?? false
      }

      const response = await this.instance.request<ApiResponse<T>>(config)
      return response.data
    } catch (error) {
      // Retry logic
      if (config.retryTimes && config.retryTimes > 0) {
        await this.delay(config.retryDelay || 1000)
        return this.request({
          ...config,
          retryTimes: config.retryTimes - 1
        })
      }

      throw error
    }
  }

  /**
   * GET request
   */
  async get<T = any>(url: string, config?: RequestOptions): Promise<ApiResponse<T>> {
    return this.request<T>({
      ...config,
      method: 'GET',
      url
    })
  }

  /**
   * POST request
   */
  async post<T = any>(url: string, data?: any, config?: RequestOptions): Promise<ApiResponse<T>> {
    return this.request<T>({
      ...config,
      method: 'POST',
      url,
      data
    })
  }

  /**
   * PUT request
   */
  async put<T = any>(url: string, data?: any, config?: RequestOptions): Promise<ApiResponse<T>> {
    return this.request<T>({
      ...config,
      method: 'PUT',
      url,
      data
    })
  }

  /**
   * PATCH request
   */
  async patch<T = any>(url: string, data?: any, config?: RequestOptions): Promise<ApiResponse<T>> {
    return this.request<T>({
      ...config,
      method: 'PATCH',
      url,
      data
    })
  }

  /**
   * DELETE request
   */
  async delete<T = any>(url: string, config?: RequestOptions): Promise<ApiResponse<T>> {
    return this.request<T>({
      ...config,
      method: 'DELETE',
      url
    })
  }

  /**
   * Upload file
   */
  async upload<T = any>(
    url: string,
    file: File,
    config?: RequestOptions & {
      onUploadProgress?: (progressEvent: any) => void
    }
  ): Promise<ApiResponse<T>> {
    const formData = new FormData()
    formData.append('file', file)

    return this.request<T>({
      ...config,
      method: 'POST',
      url,
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      onUploadProgress: config?.onUploadProgress
    })
  }

  /**
   * Download file
   */
  async download(url: string, filename?: string, config?: RequestOptions): Promise<void> {
    try {
      const response = await this.instance.request({
        ...config,
        method: 'GET',
        url,
        responseType: 'blob'
      })

      // Create download link
      const blob = new Blob([response.data])
      const downloadUrl = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = downloadUrl
      link.download = filename || this.getFilenameFromResponse(response) || 'download'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(downloadUrl)

      ElMessage.success('File downloaded successfully')
    } catch (error) {
      console.error('Download error:', error)
      ElMessage.error('Failed to download file')
      throw error
    }
  }

  /**
   * Extract filename from response headers
   */
  private getFilenameFromResponse(response: AxiosResponse): string | null {
    const contentDisposition = response.headers['content-disposition']
    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename="?([^"]+)"?/)
      return filenameMatch ? filenameMatch[1] : null
    }
    return null
  }

  /**
   * Delay utility for retry logic
   */
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
  }

  /**
   * Cancel all pending requests
   */
  cancelAllRequests(): void {
    // This would require implementing a request cancellation system
    // For now, we'll just clear the failed queue
    this.failedQueue = []
  }

  /**
   * Get request instance for advanced usage
   */
  getInstance(): AxiosInstance {
    return this.instance
  }
}

// Create and export a singleton instance
export const http = new HttpRequest()

// Export convenient methods
export const { get, post, put, patch, delete: del, upload, download, request } = http

// Export types
export type { ApiResponse, ApiError, RequestOptions }

// Extend AxiosRequestConfig to include metadata
declare module 'axios' {
  interface InternalAxiosRequestConfig {
    metadata?: {
      showLoading?: boolean
      showError?: boolean
      showSuccess?: boolean
    }
  }
}