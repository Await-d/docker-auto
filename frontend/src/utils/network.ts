/**
 * Network status management and offline handling utilities
 */
import { ref, onMounted, onUnmounted } from "vue";
import { ElMessage, ElNotification } from "element-plus";

export interface NetworkStatus {
  isOnline: boolean;
  connectionType: string;
  effectiveType: string;
  downlink: number;
  rtt: number;
  lastChecked: Date;
}

export interface RetryConfig {
  maxRetries: number;
  baseDelay: number;
  maxDelay: number;
  backoffFactor: number;
}

export interface CacheConfig {
  maxAge: number; // milliseconds
  maxSize: number; // number of items
  strategy: "lru" | "fifo";
}

// Global network status
const networkStatus = ref<NetworkStatus>({
  isOnline: navigator.onLine,
  connectionType: "unknown",
  effectiveType: "unknown",
  downlink: 0,
  rtt: 0,
  lastChecked: new Date(),
});

// Offline cache
const offlineCache = new Map<string, {
  data: any;
  timestamp: number;
  ttl: number;
}>();

/**
 * Network status composable
 */
export const useNetworkStatus = () => {
  const updateNetworkStatus = () => {
    const connection = (navigator as any).connection ||
      (navigator as any).mozConnection ||
      (navigator as any).webkitConnection;

    networkStatus.value = {
      isOnline: navigator.onLine,
      connectionType: connection?.type || "unknown",
      effectiveType: connection?.effectiveType || "unknown",
      downlink: connection?.downlink || 0,
      rtt: connection?.rtt || 0,
      lastChecked: new Date(),
    };
  };

  const handleOnline = () => {
    updateNetworkStatus();
    ElMessage.success("网络连接已恢复");

    ElNotification({
      title: "网络状态",
      message: "已重新连接到网络，正在同步数据...",
      type: "success",
      duration: 3000,
    });
  };

  const handleOffline = () => {
    updateNetworkStatus();
    ElMessage.warning("网络连接已断开");

    ElNotification({
      title: "网络状态",
      message: "网络连接已断开，应用将进入离线模式",
      type: "warning",
      duration: 5000,
    });
  };

  onMounted(() => {
    updateNetworkStatus();
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    // Update network info periodically
    const interval = setInterval(updateNetworkStatus, 30000);

    onUnmounted(() => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
      clearInterval(interval);
    });
  });

  return {
    networkStatus: networkStatus.value,
    isOnline: () => networkStatus.value.isOnline,
    getConnectionQuality: () => {
      const { effectiveType, downlink, rtt } = networkStatus.value;

      if (!networkStatus.value.isOnline) return "offline";
      if (effectiveType === "4g" && downlink > 5) return "excellent";
      if (effectiveType === "4g" || (effectiveType === "3g" && downlink > 1.5)) return "good";
      if (effectiveType === "3g" || rtt < 500) return "fair";
      return "poor";
    },
  };
};

/**
 * Retry with exponential backoff
 */
export const retryWithBackoff = async <T>(
  fn: () => Promise<T>,
  config: RetryConfig = {
    maxRetries: 3,
    baseDelay: 1000,
    maxDelay: 10000,
    backoffFactor: 2,
  }
): Promise<T> => {
  let lastError: Error;

  for (let attempt = 0; attempt <= config.maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;

      if (attempt === config.maxRetries) {
        throw lastError;
      }

      // Calculate delay with exponential backoff
      const delay = Math.min(
        config.baseDelay * Math.pow(config.backoffFactor, attempt),
        config.maxDelay
      );

      console.warn(`Attempt ${attempt + 1} failed, retrying in ${delay}ms:`, error);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError!;
};

/**
 * Cache manager for offline functionality
 */
export class OfflineCache {
  private maxAge: number;
  private maxSize: number;
  private strategy: "lru" | "fifo";

  constructor(config: CacheConfig = {
    maxAge: 300000, // 5 minutes
    maxSize: 100,
    strategy: "lru",
  }) {
    this.maxAge = config.maxAge;
    this.maxSize = config.maxSize;
    this.strategy = config.strategy;
  }

  set(key: string, data: any, ttl?: number): void {
    // Clean expired entries
    this.cleanExpired();

    // Remove oldest entries if cache is full
    if (offlineCache.size >= this.maxSize) {
      this.evictOldest();
    }

    offlineCache.set(key, {
      data,
      timestamp: Date.now(),
      ttl: ttl || this.maxAge,
    });
  }

  get(key: string): any | null {
    const entry = offlineCache.get(key);

    if (!entry) {
      return null;
    }

    // Check if entry has expired
    if (Date.now() - entry.timestamp > entry.ttl) {
      offlineCache.delete(key);
      return null;
    }

    // Update timestamp for LRU strategy
    if (this.strategy === "lru") {
      entry.timestamp = Date.now();
      offlineCache.set(key, entry);
    }

    return entry.data;
  }

  has(key: string): boolean {
    return this.get(key) !== null;
  }

  delete(key: string): void {
    offlineCache.delete(key);
  }

  clear(): void {
    offlineCache.clear();
  }

  private cleanExpired(): void {
    const now = Date.now();
    for (const [key, entry] of offlineCache.entries()) {
      if (now - entry.timestamp > entry.ttl) {
        offlineCache.delete(key);
      }
    }
  }

  private evictOldest(): void {
    if (offlineCache.size === 0) return;

    let oldestKey = "";
    let oldestTime = Date.now();

    for (const [key, entry] of offlineCache.entries()) {
      if (entry.timestamp < oldestTime) {
        oldestTime = entry.timestamp;
        oldestKey = key;
      }
    }

    if (oldestKey) {
      offlineCache.delete(oldestKey);
    }
  }

  getStats(): {
    size: number;
    maxSize: number;
    hitRate: number;
    oldestEntry: number;
    newestEntry: number;
  } {
    const entries = Array.from(offlineCache.values());
    const timestamps = entries.map(e => e.timestamp);

    return {
      size: offlineCache.size,
      maxSize: this.maxSize,
      hitRate: 0, // Would need to track hits/misses
      oldestEntry: Math.min(...timestamps) || 0,
      newestEntry: Math.max(...timestamps) || 0,
    };
  }
}

// Global cache instance
export const globalCache = new OfflineCache();

/**
 * Enhanced fetch with offline support
 */
export const fetchWithOfflineSupport = async <T>(
  url: string,
  options: any = {},
  cacheKey?: string
): Promise<T> => {
  const key = cacheKey || url;

  // Try to fetch from network first
  if (networkStatus.value.isOnline) {
    try {
      const { request } = await import('@/utils/request');
      const data = await request({
        url,
        method: options.method || 'GET',
        data: options.body,
        headers: options.headers,
        showLoading: false,
        showError: false,
      });

      // Cache successful responses
      if (options.method === "GET" || !options.method) {
        globalCache.set(key, data);
      }

      return data.data;
    } catch (error) {
      console.warn("Network request failed, checking cache:", error);
    }
  }

  // Fallback to cache
  const cachedData = globalCache.get(key);
  if (cachedData) {
    console.info("Serving from cache:", key);
    return cachedData;
  }

  // No network and no cache
  throw new Error("Data not available offline");
};

/**
 * Queue for offline actions
 */
export class OfflineActionQueue {
  private queue: Array<{
    id: string;
    action: () => Promise<any>;
    payload: any;
    timestamp: number;
    retries: number;
  }> = [];

  private isProcessing = false;
  private maxRetries = 3;

  add(id: string, action: () => Promise<any>, payload: any): void {
    this.queue.push({
      id,
      action,
      payload,
      timestamp: Date.now(),
      retries: 0,
    });

    this.processQueue();
  }

  async processQueue(): Promise<void> {
    if (this.isProcessing || !networkStatus.value.isOnline || this.queue.length === 0) {
      return;
    }

    this.isProcessing = true;

    while (this.queue.length > 0 && networkStatus.value.isOnline) {
      const item = this.queue.shift()!;

      try {
        await item.action();
        console.info("Offline action processed successfully:", item.id);
      } catch (error) {
        console.error("Failed to process offline action:", item.id, error);

        item.retries++;
        if (item.retries < this.maxRetries) {
          // Re-queue with exponential backoff
          setTimeout(() => {
            this.queue.unshift(item);
            this.processQueue();
          }, Math.pow(2, item.retries) * 1000);
        } else {
          console.error("Max retries exceeded for action:", item.id);
        }
      }
    }

    this.isProcessing = false;
  }

  getQueueSize(): number {
    return this.queue.length;
  }

  clearQueue(): void {
    this.queue = [];
  }
}

// Global action queue
export const globalActionQueue = new OfflineActionQueue();

/**
 * Debounce function for reducing API calls
 */
export const debounce = <T extends (...args: any[]) => any>(
  func: T,
  delay: number
): ((...args: Parameters<T>) => void) => {
  let timeoutId: NodeJS.Timeout;

  return (...args: Parameters<T>) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => func(...args), delay);
  };
};

/**
 * Throttle function for limiting API calls
 */
export const throttle = <T extends (...args: any[]) => any>(
  func: T,
  delay: number
): ((...args: Parameters<T>) => void) => {
  let lastCall = 0;

  return (...args: Parameters<T>) => {
    const now = Date.now();
    if (now - lastCall >= delay) {
      lastCall = now;
      func(...args);
    }
  };
};

/**
 * Check if network connection is stable
 */
export const checkNetworkStability = async (): Promise<{
  stable: boolean;
  latency: number;
  packetLoss: number;
}> => {
  const pings: number[] = [];

  for (let i = 0; i < 5; i++) {
    try {
      const pingStart = Date.now();
      const { request } = await import('@/utils/request');
      await request({
        url: "/api/ping",
        method: "HEAD",
        showLoading: false,
        showError: false,
      });
      const pingEnd = Date.now();
      pings.push(pingEnd - pingStart);
    } catch (error) {
      pings.push(-1); // Failed ping
    }

    // Wait 200ms between pings
    await new Promise(resolve => setTimeout(resolve, 200));
  }

  const successfulPings = pings.filter(p => p > 0);
  const averageLatency = successfulPings.length > 0
    ? successfulPings.reduce((a, b) => a + b, 0) / successfulPings.length
    : -1;

  const packetLoss = (5 - successfulPings.length) / 5;
  const stable = packetLoss < 0.2 && averageLatency < 1000;

  return {
    stable,
    latency: averageLatency,
    packetLoss,
  };
};