/**
 * Production-grade logging service with correlation IDs and error tracking
 */
import { ref } from 'vue';
import { post } from '@/utils/request';
import type { LogEntry } from '@/api/logs';

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LogContext {
  userId?: number;
  username?: string;
  ipAddress?: string;
  userAgent?: string;
  sessionId?: string;
  requestId?: string;
  component?: string;
  action?: string;
  metadata?: Record<string, any>;
}

export interface LogMessage {
  level: LogLevel;
  message: string;
  source: string;
  context?: LogContext;
  error?: Error;
  correlationId?: string;
}

export interface LogBuffer {
  messages: LogMessage[];
  maxSize: number;
  flushInterval: number;
}

class Logger {
  private buffer: LogBuffer;
  private flushTimer: NodeJS.Timeout | null = null;
  private correlationId: string = '';
  private globalContext: LogContext = {};
  private isEnabled = ref(true);
  private isFlushInProgress = ref(false);

  constructor() {
    this.buffer = {
      messages: [],
      maxSize: 100,
      flushInterval: 5000, // 5 seconds
    };
    this.generateCorrelationId();
    this.initializeGlobalContext();
    this.startAutoFlush();
  }

  /**
   * Generate a unique correlation ID for request tracking
   */
  private generateCorrelationId(): void {
    this.correlationId = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Initialize global context with browser and session information
   */
  private initializeGlobalContext(): void {
    this.globalContext = {
      userAgent: navigator.userAgent,
      sessionId: sessionStorage.getItem('sessionId') || this.correlationId,
    };
  }

  /**
   * Start automatic buffer flush
   */
  private startAutoFlush(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
    }
    this.flushTimer = setInterval(() => {
      this.flush();
    }, this.buffer.flushInterval);
  }

  /**
   * Set global context for all log messages
   */
  public setGlobalContext(context: Partial<LogContext>): void {
    this.globalContext = { ...this.globalContext, ...context };
  }

  /**
   * Update correlation ID (useful for new requests/operations)
   */
  public updateCorrelationId(): string {
    this.generateCorrelationId();
    return this.correlationId;
  }

  /**
   * Create a scoped logger for a specific component
   */
  public createScope(source: string, context?: Partial<LogContext>): ScopedLogger {
    return new ScopedLogger(this, source, context);
  }

  /**
   * Log a debug message
   */
  public debug(message: string, source: string, context?: LogContext): void {
    this.log('debug', message, source, context);
  }

  /**
   * Log an info message
   */
  public info(message: string, source: string, context?: LogContext): void {
    this.log('info', message, source, context);
  }

  /**
   * Log a warning message
   */
  public warn(message: string, source: string, context?: LogContext): void {
    this.log('warn', message, source, context);
  }

  /**
   * Log an error message
   */
  public error(message: string, source: string, error?: Error, context?: LogContext): void {
    this.log('error', message, source, { ...context, error });
  }

  /**
   * Log a message with specified level
   */
  public log(level: LogLevel, message: string, source: string, context?: LogContext): void {
    if (!this.isEnabled.value) return;

    const logMessage: LogMessage = {
      level,
      message,
      source,
      correlationId: this.correlationId,
      context: {
        ...this.globalContext,
        ...context,
        timestamp: new Date().toISOString(),
      },
    };

    // Add to buffer
    this.buffer.messages.push(logMessage);

    // If buffer is full, flush immediately
    if (this.buffer.messages.length >= this.buffer.maxSize) {
      this.flush();
    }

    // Also log to console in development for debugging
    if (process.env.NODE_ENV === 'development') {
      this.logToConsole(logMessage);
    }
  }

  /**
   * Log to console for development debugging
   */
  private logToConsole(logMessage: LogMessage): void {
    const { level, message, source, correlationId, context } = logMessage;
    const prefix = `[${correlationId?.slice(-8)}] [${source}]`;

    switch (level) {
      case 'debug':
        console.debug(prefix, message, context);
        break;
      case 'info':
        console.info(prefix, message, context);
        break;
      case 'warn':
        console.warn(prefix, message, context);
        break;
      case 'error':
        console.error(prefix, message, context?.error || context);
        break;
    }
  }

  /**
   * Flush buffer to remote logging service
   */
  public async flush(): Promise<void> {
    if (this.isFlushInProgress.value || this.buffer.messages.length === 0) {
      return;
    }

    this.isFlushInProgress.value = true;
    const messagesToSend = [...this.buffer.messages];
    this.buffer.messages = [];

    try {
      // Convert to API format
      const logEntries: Partial<LogEntry>[] = messagesToSend.map(msg => ({
        level: msg.level,
        source: msg.source,
        message: msg.message,
        details: {
          correlationId: msg.correlationId,
          context: msg.context,
          error: msg.context?.error ? {
            name: msg.context.error.name,
            message: msg.context.error.message,
            stack: msg.context.error.stack,
          } : undefined,
        },
        userId: msg.context?.userId,
        username: msg.context?.username,
        ipAddress: msg.context?.ipAddress,
        userAgent: msg.context?.userAgent,
        timestamp: new Date().toISOString(),
      }));

      // Send to backend logging service
      await post('/api/logs', { entries: logEntries });
    } catch (error) {
      // Fallback: put messages back in buffer if send failed
      this.buffer.messages.unshift(...messagesToSend);

      // Only log to console in development to avoid recursive errors
      if (process.env.NODE_ENV === 'development') {
        console.error('[Logger] Failed to flush logs:', error);
      }
    } finally {
      this.isFlushInProgress.value = false;
    }
  }

  /**
   * Enable or disable logging
   */
  public setEnabled(enabled: boolean): void {
    this.isEnabled.value = enabled;
  }

  /**
   * Get current logging status
   */
  public get enabled(): boolean {
    return this.isEnabled.value;
  }

  /**
   * Get current correlation ID
   */
  public getCorrelationId(): string {
    return this.correlationId;
  }

  /**
   * Clean up resources
   */
  public destroy(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
    this.flush(); // Final flush
  }
}

/**
 * Scoped logger for specific components
 */
class ScopedLogger {
  constructor(
    private logger: Logger,
    private source: string,
    private scopedContext?: Partial<LogContext>
  ) {}

  private mergeContext(context?: LogContext): LogContext {
    return {
      ...this.scopedContext,
      ...context,
      component: this.source,
    };
  }

  public debug(message: string, context?: LogContext): void {
    this.logger.debug(message, this.source, this.mergeContext(context));
  }

  public info(message: string, context?: LogContext): void {
    this.logger.info(message, this.source, this.mergeContext(context));
  }

  public warn(message: string, context?: LogContext): void {
    this.logger.warn(message, this.source, this.mergeContext(context));
  }

  public error(message: string, error?: Error, context?: LogContext): void {
    this.logger.error(message, this.source, error, this.mergeContext(context));
  }

  public updateContext(context: Partial<LogContext>): void {
    this.scopedContext = { ...this.scopedContext, ...context };
  }
}

/**
 * Performance measurement decorator
 */
export function withPerformanceLogging<T extends any[], R>(
  fn: (...args: T) => Promise<R>,
  source: string,
  operation: string
): (...args: T) => Promise<R> {
  return async (...args: T): Promise<R> => {
    const logger = getLogger().createScope(source);
    const start = performance.now();
    const correlationId = getLogger().updateCorrelationId();

    logger.debug(`Starting ${operation}`, {
      action: operation,
      correlationId,
      args: args.length > 0 ? 'provided' : 'none',
    });

    try {
      const result = await fn(...args);
      const duration = performance.now() - start;

      logger.info(`Completed ${operation}`, {
        action: operation,
        correlationId,
        duration: `${duration.toFixed(2)}ms`,
        status: 'success',
      });

      return result;
    } catch (error) {
      const duration = performance.now() - start;

      logger.error(`Failed ${operation}`, error as Error, {
        action: operation,
        correlationId,
        duration: `${duration.toFixed(2)}ms`,
        status: 'error',
      });

      throw error;
    }
  };
}

/**
 * Error boundary wrapper for component errors
 */
export function withErrorBoundary<T extends any[], R>(
  fn: (...args: T) => R,
  source: string,
  fallbackValue?: R
): (...args: T) => R {
  return (...args: T): R => {
    try {
      return fn(...args);
    } catch (error) {
      const logger = getLogger().createScope(source);
      logger.error('Component error caught by boundary', error as Error, {
        action: 'error-boundary',
        args: args.length > 0 ? 'provided' : 'none',
      });

      if (fallbackValue !== undefined) {
        return fallbackValue;
      }
      throw error;
    }
  };
}

// Global logger instance
const globalLogger = new Logger();

/**
 * Get the global logger instance
 */
export function getLogger(): Logger {
  return globalLogger;
}

/**
 * Create a scoped logger for a specific source
 */
export function createLogger(source: string, context?: Partial<LogContext>): ScopedLogger {
  return globalLogger.createScope(source, context);
}

/**
 * Initialize logging with user context (call after authentication)
 */
export function initializeLogging(userContext: Partial<LogContext>): void {
  globalLogger.setGlobalContext(userContext);
}

// Cleanup on page unload
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => {
    globalLogger.destroy();
  });
}

export default globalLogger;