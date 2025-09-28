/**
 * Home performance optimization composable with production logging
 */
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue';
import { createLogger, withPerformanceLogging } from '@/services/logging';

export interface PerformanceMetrics {
  loadTime: number;
  renderTime: number;
  interactionTime?: number;
  memoryUsage?: number;
  bundleSize?: number;
  criticalResourcesLoaded: boolean;
  performanceScore: number;
}

export interface ResourcePreloadConfig {
  criticalImages: string[];
  criticalFonts: string[];
  preloadData: string[];
  prefetchRoutes: string[];
}

export interface PerformanceBudget {
  maxLoadTime: number;
  maxRenderTime: number;
  maxMemoryUsage: number;
  maxBundleSize: number;
}

export interface PerformanceState {
  metrics: PerformanceMetrics | null;
  isMonitoring: boolean;
  isOptimizing: boolean;
  budget: PerformanceBudget;
  violations: string[];
  recommendations: string[];
}

const logger = createLogger('useHomePerformance');

const DEFAULT_BUDGET: PerformanceBudget = {
  maxLoadTime: 3000, // 3 seconds
  maxRenderTime: 1000, // 1 second
  maxMemoryUsage: 50 * 1024 * 1024, // 50MB
  maxBundleSize: 1024 * 1024, // 1MB
};

const DEFAULT_PRELOAD_CONFIG: ResourcePreloadConfig = {
  criticalImages: ['/images/logo.png', '/images/hero-bg.jpg'],
  criticalFonts: ['Inter', 'Roboto'],
  preloadData: ['/api/dashboard/overview'],
  prefetchRoutes: ['/containers', '/images'],
};

export const useHomePerformance = (
  customBudget?: Partial<PerformanceBudget>,
  preloadConfig?: Partial<ResourcePreloadConfig>
) => {
  const state = ref<PerformanceState>({
    metrics: null,
    isMonitoring: false,
    isOptimizing: false,
    budget: { ...DEFAULT_BUDGET, ...customBudget },
    violations: [],
    recommendations: [],
  });

  let performanceObserver: PerformanceObserver | null = null;
  let intersectionObserver: IntersectionObserver | null = null;
  let startTime = 0;

  const config: ResourcePreloadConfig = {
    ...DEFAULT_PRELOAD_CONFIG,
    ...preloadConfig,
  };

  /**
   * Initialize performance monitoring
   */
  const initializeMonitoring = (): void => {
    if (!('PerformanceObserver' in window)) {
      logger.warn('PerformanceObserver not supported', {
        action: 'init-monitoring',
        userAgent: navigator.userAgent,
      });
      return;
    }

    try {
      state.value.isMonitoring = true;
      startTime = performance.now();

      // Monitor paint and navigation timing
      performanceObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          handlePerformanceEntry(entry);
        }
      });

      // Observe different types of performance entries
      const entryTypes = ['navigation', 'paint', 'largest-contentful-paint', 'first-input'];
      entryTypes.forEach(type => {
        try {
          performanceObserver!.observe({ entryTypes: [type] });
        } catch (error) {
          logger.debug(`Performance entry type ${type} not supported`, {
            action: 'observe-entry-type',
            type,
            error: (error as Error).message,
          });
        }
      });

      logger.info('Performance monitoring initialized', {
        action: 'init-monitoring',
        entryTypes,
        budget: state.value.budget,
      });

    } catch (error) {
      logger.error('Performance monitoring setup failed', error as Error, {
        action: 'init-monitoring',
        budget: state.value.budget,
      });
      state.value.isMonitoring = false;
    }
  };

  /**
   * Handle performance entry data
   */
  const handlePerformanceEntry = (entry: PerformanceEntry): void => {
    try {
      switch (entry.entryType) {
        case 'navigation':
          handleNavigationTiming(entry as PerformanceNavigationTiming);
          break;
        case 'paint':
          handlePaintTiming(entry);
          break;
        case 'largest-contentful-paint':
          handleLCPTiming(entry);
          break;
        case 'first-input':
          handleFIDTiming(entry);
          break;
      }
    } catch (error) {
      logger.error('Error handling performance entry', error as Error, {
        action: 'handle-entry',
        entryType: entry.entryType,
        name: entry.name,
      });
    }
  };

  /**
   * Handle navigation timing data
   */
  const handleNavigationTiming = (entry: PerformanceNavigationTiming): void => {
    const loadTime = entry.loadEventEnd - entry.fetchStart;
    const renderTime = entry.domContentLoadedEventEnd - entry.domContentLoadedEventStart;

    const metrics: PerformanceMetrics = {
      loadTime,
      renderTime,
      criticalResourcesLoaded: checkCriticalResourcesLoaded(),
      performanceScore: calculatePerformanceScore(loadTime, renderTime),
    };

    // Get memory usage if available
    if ('memory' in performance) {
      const memInfo = (performance as any).memory;
      metrics.memoryUsage = memInfo.usedJSHeapSize;
    }

    state.value.metrics = metrics;
    analyzeBudgetViolations(metrics);

    logger.info('Navigation timing measured', {
      action: 'navigation-timing',
      loadTime: `${loadTime.toFixed(2)}ms`,
      renderTime: `${renderTime.toFixed(2)}ms`,
      performanceScore: metrics.performanceScore,
    });
  };

  /**
   * Handle paint timing
   */
  const handlePaintTiming = (entry: PerformanceEntry): void => {
    if (entry.name === 'first-contentful-paint') {
      logger.debug('First Contentful Paint measured', {
        action: 'fcp-timing',
        time: `${entry.startTime.toFixed(2)}ms`,
      });
    }
  };

  /**
   * Handle Largest Contentful Paint timing
   */
  const handleLCPTiming = (entry: PerformanceEntry): void => {
    logger.debug('Largest Contentful Paint measured', {
      action: 'lcp-timing',
      time: `${entry.startTime.toFixed(2)}ms`,
      element: (entry as any).element?.tagName,
    });
  };

  /**
   * Handle First Input Delay timing
   */
  const handleFIDTiming = (entry: PerformanceEntry): void => {
    const fid = entry.processingStart - entry.startTime;

    if (state.value.metrics) {
      state.value.metrics.interactionTime = fid;
    }

    logger.debug('First Input Delay measured', {
      action: 'fid-timing',
      delay: `${fid.toFixed(2)}ms`,
    });
  };

  /**
   * Check if critical resources are loaded
   */
  const checkCriticalResourcesLoaded = (): boolean => {
    try {
      const resourceEntries = performance.getEntriesByType('resource');
      const criticalResources = [...config.criticalImages, ...config.criticalFonts];

      return criticalResources.every(resource =>
        resourceEntries.some(entry => entry.name.includes(resource))
      );
    } catch (error) {
      logger.error('Failed to check critical resources', error as Error, {
        action: 'check-critical-resources',
      });
      return false;
    }
  };

  /**
   * Calculate performance score based on metrics
   */
  const calculatePerformanceScore = (loadTime: number, renderTime: number): number => {
    let score = 100;

    // Deduct points based on budget violations
    if (loadTime > state.value.budget.maxLoadTime) {
      score -= Math.min(30, (loadTime - state.value.budget.maxLoadTime) / 100);
    }

    if (renderTime > state.value.budget.maxRenderTime) {
      score -= Math.min(25, (renderTime - state.value.budget.maxRenderTime) / 50);
    }

    return Math.max(0, Math.round(score));
  };

  /**
   * Analyze budget violations
   */
  const analyzeBudgetViolations = (metrics: PerformanceMetrics): void => {
    const violations: string[] = [];
    const recommendations: string[] = [];

    if (metrics.loadTime > state.value.budget.maxLoadTime) {
      violations.push(`Load time (${metrics.loadTime.toFixed(0)}ms) exceeds budget (${state.value.budget.maxLoadTime}ms)`);
      recommendations.push('Consider code splitting or lazy loading non-critical resources');
    }

    if (metrics.renderTime > state.value.budget.maxRenderTime) {
      violations.push(`Render time (${metrics.renderTime.toFixed(0)}ms) exceeds budget (${state.value.budget.maxRenderTime}ms)`);
      recommendations.push('Optimize component rendering and reduce DOM operations');
    }

    if (metrics.memoryUsage && metrics.memoryUsage > state.value.budget.maxMemoryUsage) {
      violations.push(`Memory usage (${(metrics.memoryUsage / 1024 / 1024).toFixed(1)}MB) exceeds budget`);
      recommendations.push('Check for memory leaks and optimize data structures');
    }

    if (!metrics.criticalResourcesLoaded) {
      violations.push('Critical resources not properly loaded');
      recommendations.push('Ensure critical images and fonts are preloaded');
    }

    state.value.violations = violations;
    state.value.recommendations = recommendations;

    if (violations.length > 0) {
      logger.warn('Performance budget violations detected', {
        action: 'budget-violations',
        violations,
        metrics,
      });
    }
  };

  /**
   * Preload critical resources
   */
  const preloadCriticalResources = withPerformanceLogging(
    async (): Promise<void> => {
      const preloadPromises: Promise<void>[] = [];

      // Preload critical images
      config.criticalImages.forEach(src => {
        preloadPromises.push(preloadImage(src));
      });

      // Preload critical fonts
      config.criticalFonts.forEach(fontFamily => {
        preloadPromises.push(preloadFont(fontFamily));
      });

      // Preload critical data
      config.preloadData.forEach(url => {
        preloadPromises.push(preloadData(url));
      });

      try {
        await Promise.allSettled(preloadPromises);
        logger.info('Critical resources preloaded', {
          action: 'preload-resources',
          resourceCount: preloadPromises.length,
          config,
        });
      } catch (error) {
        logger.error('Resource preloading failed', error as Error, {
          action: 'preload-resources',
          config,
        });
      }
    },
    'useHomePerformance',
    'preload-resources'
  );

  /**
   * Preload a single image
   */
  const preloadImage = async (src: string): Promise<void> => {
    return new Promise((resolve, reject) => {
      const img = new Image();
      img.onload = () => resolve();
      img.onerror = () => reject(new Error(`Failed to preload image: ${src}`));
      img.src = src;

      // Timeout after 5 seconds
      setTimeout(() => reject(new Error(`Image preload timeout: ${src}`)), 5000);
    });
  };

  /**
   * Preload a font
   */
  const preloadFont = async (fontFamily: string): Promise<void> => {
    if ('fonts' in document) {
      try {
        await document.fonts.load(`16px ${fontFamily}`);
      } catch (error) {
        throw new Error(`Failed to preload font: ${fontFamily}`);
      }
    }
  };

  /**
   * Preload API data
   */
  const preloadData = async (url: string): Promise<void> => {
    try {
      const response = await fetch(url, {
        method: 'HEAD',
        credentials: 'include'
      });

      if (!response.ok) {
        throw new Error(`Preload request failed: ${response.statusText}`);
      }
    } catch (error) {
      throw new Error(`Failed to preload data: ${url} - ${(error as Error).message}`);
    }
  };

  /**
   * Setup lazy loading for images
   */
  const setupLazyLoading = (): void => {
    if (!('IntersectionObserver' in window)) {
      logger.debug('IntersectionObserver not supported', {
        action: 'lazy-loading-setup',
      });
      return;
    }

    try {
      intersectionObserver = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              const img = entry.target as HTMLImageElement;
              if (img.dataset.src) {
                img.src = img.dataset.src;
                img.removeAttribute('data-src');
                intersectionObserver!.unobserve(img);
              }
            }
          });
        },
        { threshold: 0.1 }
      );

      // Observe all images with data-src attribute
      const lazyImages = document.querySelectorAll('img[data-src]');
      lazyImages.forEach(img => intersectionObserver!.observe(img));

      logger.debug('Lazy loading setup complete', {
        action: 'lazy-loading-setup',
        imageCount: lazyImages.length,
      });
    } catch (error) {
      logger.error('Failed to setup lazy loading', error as Error, {
        action: 'lazy-loading-setup',
      });
    }
  };

  /**
   * Optimize component updates
   */
  const optimizeUpdates = async (): Promise<void> => {
    state.value.isOptimizing = true;

    try {
      // Wait for next tick to ensure DOM is ready
      await nextTick();

      // Force garbage collection if available
      if ('gc' in window) {
        (window as any).gc();
      }

      // Clear performance entries to prevent memory buildup
      if ('clearResourceTimings' in performance) {
        performance.clearResourceTimings();
      }

      logger.debug('Component optimization completed', {
        action: 'optimize-updates',
      });
    } catch (error) {
      logger.error('Performance measurement failed', error as Error, {
        action: 'optimize-updates',
      });
    } finally {
      state.value.isOptimizing = false;
    }
  };

  /**
   * Get performance recommendations
   */
  const getRecommendations = (): string[] => {
    const recommendations = [...state.value.recommendations];

    if (state.value.metrics) {
      const { performanceScore } = state.value.metrics;

      if (performanceScore < 50) {
        recommendations.push('Performance is poor - consider major optimizations');
      } else if (performanceScore < 80) {
        recommendations.push('Performance needs improvement - review critical resources');
      }
    }

    return recommendations;
  };

  /**
   * Clean up observers
   */
  const cleanup = (): void => {
    try {
      if (performanceObserver) {
        performanceObserver.disconnect();
        performanceObserver = null;
      }

      if (intersectionObserver) {
        intersectionObserver.disconnect();
        intersectionObserver = null;
      }

      logger.debug('Performance monitoring cleanup completed', {
        action: 'cleanup',
      });
    } catch (error) {
      logger.error('Failed to disconnect performance observer', error as Error, {
        action: 'cleanup',
      });
    }
  };

  // Computed properties
  const performanceScore = computed(() => state.value.metrics?.performanceScore || 0);
  const hasViolations = computed(() => state.value.violations.length > 0);
  const isGoodPerformance = computed(() => performanceScore.value >= 80);
  const loadTime = computed(() => state.value.metrics?.loadTime || 0);
  const renderTime = computed(() => state.value.metrics?.renderTime || 0);

  // Lifecycle
  onMounted(async () => {
    try {
      initializeMonitoring();
      setupLazyLoading();
      await preloadCriticalResources();
    } catch (error) {
      logger.error('Failed to initialize performance monitoring', error as Error, {
        action: 'mount',
      });
    }
  });

  onUnmounted(() => {
    cleanup();
  });

  return {
    // State
    metrics: computed(() => state.value.metrics),
    isMonitoring: computed(() => state.value.isMonitoring),
    isOptimizing: computed(() => state.value.isOptimizing),
    violations: computed(() => state.value.violations),
    budget: computed(() => state.value.budget),

    // Computed metrics
    performanceScore,
    hasViolations,
    isGoodPerformance,
    loadTime,
    renderTime,

    // Actions
    optimizeUpdates,
    preloadCriticalResources,
    getRecommendations,
    cleanup,

    // Configuration
    updateBudget: (newBudget: Partial<PerformanceBudget>) => {
      state.value.budget = { ...state.value.budget, ...newBudget };
    },
  };
};