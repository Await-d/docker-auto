/**
 * Network status and offline management store
 */
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { ElNotification } from "element-plus";
import {
  useNetworkStatus,
  globalCache,
  globalActionQueue,
  checkNetworkStability,
  type NetworkStatus,
} from "@/utils/network";

export const useNetworkStore = defineStore("network", () => {
  // State
  const isOnline = ref(navigator.onLine);
  const networkQuality = ref<"excellent" | "good" | "fair" | "poor" | "offline">("good");
  const lastChecked = ref<Date>(new Date());
  const connectionDetails = ref<NetworkStatus>({
    isOnline: navigator.onLine,
    connectionType: "unknown",
    effectiveType: "unknown",
    downlink: 0,
    rtt: 0,
    lastChecked: new Date(),
  });

  const offlineActions = ref<Array<{
    id: string;
    type: string;
    payload: any;
    timestamp: string;
    retries: number;
  }>>([]);

  const cacheStats = ref({
    size: 0,
    maxSize: 100,
    hitRate: 0,
    oldestEntry: 0,
    newestEntry: 0,
  });

  // Getters
  const isConnected = computed(() => isOnline.value);
  const isOffline = computed(() => !isOnline.value);
  const hasGoodConnection = computed(() =>
    ["excellent", "good"].includes(networkQuality.value)
  );
  const shouldShowOfflineWarning = computed(() =>
    isOffline.value || networkQuality.value === "poor"
  );

  // Actions
  const updateNetworkStatus = () => {
    const { networkStatus, getConnectionQuality } = useNetworkStatus();

    isOnline.value = networkStatus.isOnline;
    networkQuality.value = getConnectionQuality();
    connectionDetails.value = networkStatus;
    lastChecked.value = new Date();

    // Update cache stats
    cacheStats.value = globalCache.getStats();
  };

  const handleNetworkChange = (online: boolean) => {
    isOnline.value = online;
    updateNetworkStatus();

    if (online) {
      handleNetworkReconnected();
    } else {
      handleNetworkDisconnected();
    }
  };

  const handleNetworkReconnected = async () => {
    ElNotification({
      title: "网络已恢复",
      message: "正在同步离线期间的操作...",
      type: "success",
      duration: 3000,
    });

    // Process queued offline actions
    await globalActionQueue.processQueue();

    // Check network stability
    try {
      const stability = await checkNetworkStability();
      if (!stability.stable) {
        ElNotification({
          title: "网络不稳定",
          message: `网络连接不稳定，延迟: ${stability.latency.toFixed(0)}ms，丢包率: ${(stability.packetLoss * 100).toFixed(1)}%`,
          type: "warning",
          duration: 5000,
        });
      }
    } catch (error) {
      console.warn("Failed to check network stability:", error);
    }
  };

  const handleNetworkDisconnected = () => {
    ElNotification({
      title: "网络连接断开",
      message: "应用已切换到离线模式，您的操作将在网络恢复后同步",
      type: "warning",
      duration: 5000,
    });
  };

  const addOfflineAction = (
    type: string,
    payload: any,
    action: () => Promise<any>
  ): string => {
    const id = `offline_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

    offlineActions.value.push({
      id,
      type,
      payload,
      timestamp: new Date().toISOString(),
      retries: 0,
    });

    globalActionQueue.add(id, action, payload);

    return id;
  };

  const removeOfflineAction = (id: string) => {
    const index = offlineActions.value.findIndex(action => action.id === id);
    if (index !== -1) {
      offlineActions.value.splice(index, 1);
    }
  };

  const clearOfflineActions = () => {
    offlineActions.value = [];
    globalActionQueue.clearQueue();
  };

  const getCachedData = (key: string): any => {
    return globalCache.get(key);
  };

  const setCachedData = (key: string, data: any, ttl?: number) => {
    globalCache.set(key, data, ttl);
    cacheStats.value = globalCache.getStats();
  };

  const clearCache = () => {
    globalCache.clear();
    cacheStats.value = globalCache.getStats();
  };

  const testNetworkSpeed = async (): Promise<{
    downloadSpeed: number;
    uploadSpeed: number;
    latency: number;
  }> => {
    try {
      // Simple network speed test using a small file
      const startTime = Date.now();

      const { request } = await import('@/utils/request');
      await request({
        url: "/api/ping",
        method: "GET",
        showLoading: false,
        showError: false,
      });

      const endTime = Date.now();
      const latency = endTime - startTime;

      // This is a simplified test - in real implementation,
      // you might want to use a larger file and calculate actual speeds
      return {
        downloadSpeed: 0, // Would need actual implementation
        uploadSpeed: 0, // Would need actual implementation
        latency,
      };
    } catch (error) {
      console.error("Network speed test failed:", error);
      throw error;
    }
  };

  const getNetworkInfo = () => {
    const connection = (navigator as any).connection ||
      (navigator as any).mozConnection ||
      (navigator as any).webkitConnection;

    return {
      type: connection?.type || "unknown",
      effectiveType: connection?.effectiveType || "unknown",
      downlink: connection?.downlink || 0,
      rtt: connection?.rtt || 0,
      saveData: connection?.saveData || false,
    };
  };

  const shouldReduceDataUsage = computed(() => {
    const connection = getNetworkInfo();
    return (
      connection.saveData ||
      ["slow-2g", "2g"].includes(connection.effectiveType) ||
      networkQuality.value === "poor"
    );
  });

  const getRecommendedPollingInterval = computed(() => {
    if (isOffline.value) return 0; // No polling when offline

    switch (networkQuality.value) {
      case "excellent":
        return 1000; // 1 second
      case "good":
        return 2000; // 2 seconds
      case "fair":
        return 5000; // 5 seconds
      case "poor":
        return 10000; // 10 seconds
      default:
        return 0;
    }
  });

  // Initialize
  const initialize = () => {
    updateNetworkStatus();

    // Listen for network changes
    window.addEventListener("online", () => handleNetworkChange(true));
    window.addEventListener("offline", () => handleNetworkChange(false));

    // Listen for connection changes
    const connection = (navigator as any).connection ||
      (navigator as any).mozConnection ||
      (navigator as any).webkitConnection;

    if (connection) {
      connection.addEventListener("change", updateNetworkStatus);
    }

    // Periodic status updates
    setInterval(updateNetworkStatus, 30000); // Every 30 seconds
  };

  return {
    // State
    isOnline,
    networkQuality,
    lastChecked,
    connectionDetails,
    offlineActions,
    cacheStats,

    // Getters
    isConnected,
    isOffline,
    hasGoodConnection,
    shouldShowOfflineWarning,
    shouldReduceDataUsage,
    getRecommendedPollingInterval,

    // Actions
    updateNetworkStatus,
    handleNetworkChange,
    addOfflineAction,
    removeOfflineAction,
    clearOfflineActions,
    getCachedData,
    setCachedData,
    clearCache,
    testNetworkSpeed,
    getNetworkInfo,
    initialize,
  };
});