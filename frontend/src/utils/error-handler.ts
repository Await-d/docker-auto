/**
 * Global error handling utilities
 */
import { ElMessage, ElNotification } from "element-plus";

export interface ErrorContext {
  component?: string;
  action?: string;
  userId?: string;
  timestamp?: Date;
  userAgent?: string;
  url?: string;
  containerId?: string;
  sessionId?: string;
}

export interface ErrorDetails {
  code?: string | number;
  message: string;
  details?: string;
  stack?: string;
  context?: ErrorContext;
  severity?: 'info' | 'warning' | 'error' | 'critical';
  retryable?: boolean;
  userMessage?: string;
}

export class ErrorHandler {
  private static instance: ErrorHandler | null = null;
  private errorQueue: ErrorDetails[] = [];
  private isReporting = false;

  static getInstance(): ErrorHandler {
    if (!ErrorHandler.instance) {
      ErrorHandler.instance = new ErrorHandler();
    }
    return ErrorHandler.instance;
  }

  /**
   * Handle and display errors to the user
   */
  handle(error: Error | ErrorDetails, context?: ErrorContext): void {
    const errorDetails = this.normalizeError(error, context);

    // Log to console in development
    if (import.meta.env.DEV) {
      console.error('[ErrorHandler]', errorDetails);
    }

    // Show user notification
    this.showUserNotification(errorDetails);

    // Queue for reporting
    this.queueForReporting(errorDetails);
  }

  /**
   * Handle API errors specifically
   */
  handleApiError(error: any, context?: ErrorContext): void {
    const errorDetails: ErrorDetails = {
      code: error.response?.status || error.status || 'NETWORK_ERROR',
      message: error.response?.data?.message || error.message || 'Network error occurred',
      details: error.response?.data?.details || error.details,
      context: {
        ...context,
        url: error.config?.url || error.url,
        timestamp: new Date(),
      },
      severity: this.getErrorSeverity(error.response?.status || error.status),
      retryable: this.isRetryableError(error.response?.status || error.status),
      userMessage: this.getUserFriendlyMessage(error.response?.status || error.status, error.response?.data?.message || error.message),
    };

    this.handle(errorDetails, context);
  }

  /**
   * Handle WebSocket errors
   */
  handleWebSocketError(error: Event | CloseEvent | Error, context?: ErrorContext): void {
    let message = 'WebSocket connection error';
    let code = 'WS_ERROR';
    let severity: ErrorDetails['severity'] = 'warning';

    if (error instanceof CloseEvent) {
      message = `WebSocket closed: ${error.reason || 'Connection terminated'}`;
      code = `WS_CLOSE_${error.code}`;
      severity = error.code === 1000 ? 'info' : 'error'; // 1000 is normal close
    } else if (error instanceof Error) {
      message = error.message;
      code = 'WS_ERROR';
    }

    const errorDetails: ErrorDetails = {
      code,
      message,
      context: {
        ...context,
        timestamp: new Date(),
      },
      severity,
      retryable: true,
      userMessage: severity === 'info' ? undefined : '网络连接中断，正在尝试重新连接...',
    };

    this.handle(errorDetails);
  }

  /**
   * Handle Docker operation errors
   */
  handleDockerError(error: any, operation: string, containerId?: string): void {
    const errorDetails: ErrorDetails = {
      code: error.code || 'DOCKER_ERROR',
      message: error.message || 'Docker operation failed',
      details: error.details,
      context: {
        action: operation,
        containerId,
        timestamp: new Date(),
      },
      severity: 'error',
      retryable: this.isRetryableDockerError(error.code),
      userMessage: this.getDockerErrorMessage(error.code, operation),
    };

    this.handle(errorDetails);
  }

  /**
   * Handle terminal errors
   */
  handleTerminalError(error: any, sessionId?: string, containerId?: string): void {
    const errorDetails: ErrorDetails = {
      code: error.code || 'TERMINAL_ERROR',
      message: error.message || 'Terminal operation failed',
      context: {
        action: 'terminal_operation',
        sessionId,
        containerId,
        timestamp: new Date(),
      },
      severity: 'error',
      retryable: true,
      userMessage: '终端连接失败，请检查容器状态后重试',
    };

    this.handle(errorDetails);
  }

  /**
   * Show retry confirmation dialog
   */
  async showRetryDialog(error: ErrorDetails, retryAction: () => Promise<void>): Promise<void> {
    if (!error.retryable) {
      return;
    }

    const { ElMessageBox } = await import('element-plus');

    try {
      await ElMessageBox.confirm(
        error.userMessage || error.message,
        '操作失败',
        {
          type: 'warning',
          confirmButtonText: '重试',
          cancelButtonText: '取消',
          showClose: true,
        }
      );

      await retryAction();
    } catch (err) {
      // User cancelled or retry failed
      if (err !== 'cancel' && err !== 'close') {
        console.error('Retry failed:', err);
      }
    }
  }

  /**
   * Create error boundary for components
   */
  createErrorBoundary(componentName: string) {
    return {
      errorCaptured: (error: Error, instance: any, info: string) => {
        this.handle(error, {
          component: componentName,
          action: 'component_error',
          details: info,
          timestamp: new Date(),
        });

        // Return false to prevent the error from propagating
        return false;
      }
    };
  }

  private normalizeError(error: Error | ErrorDetails, context?: ErrorContext): ErrorDetails {
    if ('message' in error && typeof error.message === 'string') {
      // It's already an ErrorDetails object or has ErrorDetails-like structure
      const errorDetails = error as ErrorDetails;
      return {
        ...errorDetails,
        context: {
          ...errorDetails.context,
          ...context,
          timestamp: errorDetails.context?.timestamp || new Date(),
        },
        severity: errorDetails.severity || 'error',
      };
    }

    // It's a regular Error object
    const regularError = error as Error;
    return {
      message: regularError.message,
      stack: regularError.stack,
      context: {
        ...context,
        timestamp: new Date(),
        userAgent: navigator.userAgent,
        url: window.location.href,
      },
      severity: 'error',
    };
  }

  private showUserNotification(error: ErrorDetails): void {
    const message = error.userMessage || error.message;

    if (!message || error.severity === 'info') {
      return;
    }

    switch (error.severity) {
      case 'critical':
      case 'error':
        if (error.retryable) {
          ElNotification({
            title: '操作失败',
            message: message,
            type: 'error',
            duration: 0, // Don't auto-close for critical errors
            showClose: true,
          });
        } else {
          ElMessage.error(message);
        }
        break;

      case 'warning':
        ElMessage.warning(message);
        break;

      default:
        ElMessage.info(message);
    }
  }

  private queueForReporting(error: ErrorDetails): void {
    this.errorQueue.push(error);

    // Don't report in development
    if (import.meta.env.DEV) {
      return;
    }

    // Debounce reporting
    if (!this.isReporting) {
      this.isReporting = true;
      setTimeout(() => {
        this.reportErrors();
        this.isReporting = false;
      }, 1000);
    }
  }

  private async reportErrors(): Promise<void> {
    if (this.errorQueue.length === 0) {
      return;
    }

    const errorsToReport = [...this.errorQueue];
    this.errorQueue = [];

    try {
      // Send to error reporting service
      await fetch('/api/errors/report', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify({
          errors: errorsToReport,
          userAgent: navigator.userAgent,
          url: window.location.href,
          timestamp: new Date().toISOString(),
        }),
      });
    } catch (err) {
      // Failed to report errors - put them back in queue for later
      this.errorQueue.unshift(...errorsToReport);
      console.warn('Failed to report errors:', err);
    }
  }

  private getErrorSeverity(status?: number | string): ErrorDetails['severity'] {
    const statusCode = typeof status === 'string' ? parseInt(status, 10) : status;

    if (!statusCode || isNaN(statusCode)) {
      return 'error';
    }

    if (statusCode >= 500) {
      return 'critical';
    } else if (statusCode >= 400) {
      return 'error';
    } else if (statusCode >= 300) {
      return 'warning';
    } else {
      return 'info';
    }
  }

  private isRetryableError(status?: number | string): boolean {
    const statusCode = typeof status === 'string' ? parseInt(status, 10) : status;

    if (!statusCode || isNaN(statusCode)) {
      return true; // Network errors are usually retryable
    }

    // Retry on server errors and some client errors
    return (
      statusCode >= 500 || // Server errors
      statusCode === 408 || // Request timeout
      statusCode === 429 || // Too many requests
      statusCode === 503 || // Service unavailable
      statusCode === 504    // Gateway timeout
    );
  }

  private isRetryableDockerError(code?: string): boolean {
    if (!code) return true;

    const nonRetryableCodes = [
      'CONTAINER_NOT_FOUND',
      'IMAGE_NOT_FOUND',
      'INVALID_COMMAND',
      'PERMISSION_DENIED',
    ];

    return !nonRetryableCodes.includes(code);
  }

  private getUserFriendlyMessage(status?: number | string, originalMessage?: string): string {
    const statusCode = typeof status === 'string' ? parseInt(status, 10) : status;

    const messages: Record<number, string> = {
      400: '请求参数错误',
      401: '未授权，请重新登录',
      403: '权限不足',
      404: '资源未找到',
      408: '请求超时，请重试',
      409: '操作冲突，请刷新后重试',
      429: '请求过于频繁，请稍后重试',
      500: '服务器内部错误',
      502: '网关错误',
      503: '服务暂不可用',
      504: '网关超时',
    };

    if (statusCode && messages[statusCode]) {
      return messages[statusCode];
    }

    // Return original message if available, otherwise generic message
    return originalMessage || '操作失败，请重试';
  }

  private getDockerErrorMessage(code?: string, operation?: string): string {
    const operationText = operation || '操作';

    const messages: Record<string, string> = {
      'CONTAINER_NOT_FOUND': '容器不存在',
      'CONTAINER_NOT_RUNNING': '容器未运行',
      'IMAGE_NOT_FOUND': '镜像未找到',
      'NETWORK_ERROR': '网络连接失败',
      'PERMISSION_DENIED': '权限不足',
      'DOCKER_DAEMON_ERROR': 'Docker 服务异常',
      'CONTAINER_ALREADY_EXISTS': '容器已存在',
      'INVALID_COMMAND': '命令无效',
      'RESOURCE_EXHAUSTED': '资源不足',
    };

    if (code && messages[code]) {
      return `${operationText}失败: ${messages[code]}`;
    }

    return `Docker ${operationText}失败，请检查容器状态`;
  }
}

// Export singleton instance
export const errorHandler = ErrorHandler.getInstance();

// Export convenience functions
export const handleError = (error: Error | ErrorDetails, context?: ErrorContext) =>
  errorHandler.handle(error, context);

export const handleApiError = (error: any, context?: ErrorContext) =>
  errorHandler.handleApiError(error, context);

export const handleWebSocketError = (error: Event | CloseEvent | Error, context?: ErrorContext) =>
  errorHandler.handleWebSocketError(error, context);

export const handleDockerError = (error: any, operation: string, containerId?: string) =>
  errorHandler.handleDockerError(error, operation, containerId);

export const handleTerminalError = (error: any, sessionId?: string, containerId?: string) =>
  errorHandler.handleTerminalError(error, sessionId, containerId);

// Global error handler for unhandled errors
window.addEventListener('error', (event) => {
  errorHandler.handle(event.error || new Error(event.message), {
    component: 'global',
    action: 'unhandled_error',
    url: window.location.href,
  });
});

// Global handler for unhandled promise rejections
window.addEventListener('unhandledrejection', (event) => {
  errorHandler.handle(
    event.reason instanceof Error ? event.reason : new Error(String(event.reason)),
    {
      component: 'global',
      action: 'unhandled_rejection',
      url: window.location.href,
    }
  );
});

export default ErrorHandler;