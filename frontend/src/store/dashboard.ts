/**
 * Dashboard Store - Manages dashboard state, layouts, and widgets
 */
import { defineStore } from "pinia";
import { ref, computed, reactive } from "vue";

// Types for dashboard management
export interface WidgetPosition {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface WidgetSize {
  minW: number;
  minH: number;
  maxW?: number;
  maxH?: number;
}

export interface WidgetSettings {
  // Basic settings
  theme?: string;
  refreshInterval?: number;
  displayMode?: string;
  filters?: Record<string, any>;
  preferences?: Record<string, any>;

  // Appearance settings
  showHeader?: boolean;
  showFooter?: boolean;
  animations?: boolean;

  // Data settings
  dataSource?: string;
  cacheDuration?: number;
  maxDataPoints?: number;
  dateRange?: string;

  // Advanced settings
  errorHandling?: string;
  retryAttempts?: number;
  retryDelay?: number;
  debugMode?: boolean;
  performanceMonitoring?: boolean;
  customCSS?: string;
  customProperties?: Record<string, any>;

  // Permission settings
  minimumRole?: string;
  hideIfNoAccess?: boolean;
}

export interface DashboardWidget {
  id: string;
  type: string;
  title: string;
  component: string;
  position: WidgetPosition;
  size: WidgetSize;
  settings: WidgetSettings;
  refreshInterval: number;
  permissions: string[];
  enabled: boolean;
  draggable: boolean;
  resizable: boolean;
  configurable?: boolean;
}

export interface DashboardLayout {
  id: string;
  name: string;
  description?: string;
  isDefault: boolean;
  isShared: boolean;
  createdBy: string;
  createdAt: Date;
  updatedAt: Date;
  widgets: DashboardWidget[];
  gridSize: {
    cols: number;
    rows: number;
    margin: number[];
    containerPadding: number[];
  };
  permissions: string[];
}

export interface DashboardState {
  layouts: DashboardLayout[];
  currentLayoutId: string;
  widgets: DashboardWidget[];
  activeWidgets: string[];
  widgetData: Record<string, any>;
  refreshIntervals: Record<string, number>;
  isLoading: boolean;
  isEditMode: boolean;
  lastUpdated: Record<string, Date>;
  globalSettings: {
    autoRefresh: boolean;
    theme: string;
    animations: boolean;
    compactMode: boolean;
  };
}

// Widget registry for available widgets
export interface WidgetDefinition {
  type: string;
  name: string;
  description: string;
  component: string;
  icon: string;
  category: string;
  defaultSize: WidgetSize;
  defaultPosition: Partial<WidgetPosition>;
  defaultSettings: WidgetSettings;
  permissions: string[];
  configurable: boolean;
}

const defaultWidgets: WidgetDefinition[] = [
  {
    type: "system-overview",
    name: "System Overview",
    description: "Overall system health and key metrics",
    component: "SystemOverview",
    icon: "Monitor",
    category: "system",
    defaultSize: { minW: 4, minH: 3, maxW: 6, maxH: 4 },
    defaultPosition: { x: 0, y: 0, w: 4, h: 3 },
    defaultSettings: { refreshInterval: 30000 },
    permissions: ["dashboard:read"],
    configurable: true,
  },
  {
    type: "container-stats",
    name: "Container Statistics",
    description: "Container counts and status distribution",
    component: "ContainerStats",
    icon: "Box",
    category: "containers",
    defaultSize: { minW: 3, minH: 3, maxW: 4, maxH: 4 },
    defaultPosition: { x: 4, y: 0, w: 3, h: 3 },
    defaultSettings: { refreshInterval: 15000 },
    permissions: ["container:read"],
    configurable: true,
  },
  {
    type: "update-activity",
    name: "Update Activity",
    description: "Recent updates and update statistics",
    component: "UpdateActivity",
    icon: "Refresh",
    category: "updates",
    defaultSize: { minW: 3, minH: 3, maxW: 5, maxH: 4 },
    defaultPosition: { x: 7, y: 0, w: 3, h: 3 },
    defaultSettings: { refreshInterval: 30000 },
    permissions: ["update:read"],
    configurable: true,
  },
  {
    type: "realtime-monitor",
    name: "Real-time Monitor",
    description: "Live system activity and performance metrics",
    component: "RealtimeMonitor",
    icon: "DataLine",
    category: "monitoring",
    defaultSize: { minW: 5, minH: 4, maxW: 8, maxH: 6 },
    defaultPosition: { x: 0, y: 3, w: 5, h: 4 },
    defaultSettings: { refreshInterval: 5000 },
    permissions: ["monitor:read"],
    configurable: true,
  },
  {
    type: "health-monitor",
    name: "Health Monitor",
    description: "Service health and availability metrics",
    component: "HealthMonitor",
    icon: "CircleCheckFilled",
    category: "monitoring",
    defaultSize: { minW: 3, minH: 3, maxW: 4, maxH: 4 },
    defaultPosition: { x: 5, y: 3, w: 3, h: 3 },
    defaultSettings: { refreshInterval: 20000 },
    permissions: ["monitor:read"],
    configurable: true,
  },
  {
    type: "recent-activities",
    name: "Recent Activities",
    description: "Timeline of recent system activities",
    component: "RecentActivities",
    icon: "Document",
    category: "activities",
    defaultSize: { minW: 4, minH: 4, maxW: 6, maxH: 6 },
    defaultPosition: { x: 8, y: 3, w: 4, h: 4 },
    defaultSettings: { refreshInterval: 10000 },
    permissions: ["log:read"],
    configurable: true,
  },
  {
    type: "quick-actions",
    name: "Quick Actions",
    description: "Frequently used operations and shortcuts",
    component: "QuickActions",
    icon: "Lightning",
    category: "actions",
    defaultSize: { minW: 2, minH: 2, maxW: 3, maxH: 3 },
    defaultPosition: { x: 0, y: 7, w: 2, h: 2 },
    defaultSettings: {},
    permissions: ["dashboard:read"],
    configurable: true,
  },
  {
    type: "notification-center",
    name: "Notification Center",
    description: "Live notifications and alerts",
    component: "NotificationCenter",
    icon: "Bell",
    category: "notifications",
    defaultSize: { minW: 3, minH: 3, maxW: 4, maxH: 5 },
    defaultPosition: { x: 2, y: 7, w: 3, h: 3 },
    defaultSettings: { refreshInterval: 5000 },
    permissions: ["notification:read"],
    configurable: true,
  },
  {
    type: "resource-charts",
    name: "Resource Charts",
    description: "Historical resource usage charts",
    component: "ResourceCharts",
    icon: "DataAnalysis",
    category: "monitoring",
    defaultSize: { minW: 4, minH: 3, maxW: 8, maxH: 5 },
    defaultPosition: { x: 5, y: 7, w: 4, h: 3 },
    defaultSettings: { refreshInterval: 30000 },
    permissions: ["monitor:read"],
    configurable: true,
  },
  {
    type: "security-dashboard",
    name: "Security Dashboard",
    description: "Security status and vulnerability monitoring",
    component: "SecurityDashboard",
    icon: "Lock",
    category: "security",
    defaultSize: { minW: 3, minH: 3, maxW: 5, maxH: 4 },
    defaultPosition: { x: 9, y: 7, w: 3, h: 3 },
    defaultSettings: { refreshInterval: 60000 },
    permissions: ["security:read"],
    configurable: true,
  },
];

export const useDashboardStore = defineStore("dashboard", () => {
  // State
  const state = reactive<DashboardState>({
    layouts: [],
    currentLayoutId: "",
    widgets: [],
    activeWidgets: [],
    widgetData: {},
    refreshIntervals: {},
    isLoading: false,
    isEditMode: false,
    lastUpdated: {},
    globalSettings: {
      autoRefresh: true,
      theme: "auto",
      animations: true,
      compactMode: false,
    },
  });

  // Available widget definitions
  const availableWidgets = ref<WidgetDefinition[]>(defaultWidgets);

  // Computed properties
  const currentLayout = computed(() =>
    state.layouts.find((layout) => layout.id === state.currentLayoutId),
  );

  const activeWidgetData = computed(() => {
    const data: Record<string, any> = {};
    state.activeWidgets.forEach((widgetId) => {
      data[widgetId] = state.widgetData[widgetId] || {};
    });
    return data;
  });

  const widgetsByCategory = computed(() => {
    const categories: Record<string, WidgetDefinition[]> = {};
    availableWidgets.value.forEach((widget) => {
      if (!categories[widget.category]) {
        categories[widget.category] = [];
      }
      categories[widget.category].push(widget);
    });
    return categories;
  });

  // Actions
  const setLoading = (loading: boolean) => {
    state.isLoading = loading;
  };

  const setEditMode = (editMode: boolean) => {
    state.isEditMode = editMode;
  };

  const createDefaultLayout = (): DashboardLayout => {
    const defaultWidgets = availableWidgets.value
      .slice(0, 6)
      .map((def, index) => ({
        id: `widget-${Date.now()}-${index}`,
        type: def.type,
        title: def.name,
        component: def.component,
        position: {
          ...def.defaultPosition,
          w: def.defaultPosition.w || 3,
          h: def.defaultPosition.h || 3,
        } as WidgetPosition,
        size: def.defaultSize,
        settings: def.defaultSettings,
        refreshInterval: def.defaultSettings.refreshInterval || 30000,
        permissions: def.permissions,
        enabled: true,
        draggable: true,
        resizable: true,
      }));

    return {
      id: `layout-${Date.now()}`,
      name: "Default Dashboard",
      description: "Default system dashboard layout",
      isDefault: true,
      isShared: false,
      createdBy: "system",
      createdAt: new Date(),
      updatedAt: new Date(),
      widgets: defaultWidgets,
      gridSize: {
        cols: 12,
        rows: 12,
        margin: [10, 10],
        containerPadding: [20, 20],
      },
      permissions: ["dashboard:read"],
    };
  };

  const loadLayouts = async () => {
    try {
      setLoading(true);

      // Try to load layouts from localStorage first
      const savedLayouts = localStorage.getItem("dashboard-layouts");
      if (savedLayouts) {
        const layouts = JSON.parse(savedLayouts);
        state.layouts = layouts.map((layout: any) => ({
          ...layout,
          createdAt: new Date(layout.createdAt),
          updatedAt: new Date(layout.updatedAt),
        }));
      }

      // If no layouts exist, create default
      if (state.layouts.length === 0) {
        const defaultLayout = createDefaultLayout();
        state.layouts = [defaultLayout];
        state.currentLayoutId = defaultLayout.id;
        await saveLayoutsToStorage();
      } else if (!state.currentLayoutId) {
        // Set current layout to default or first available
        const defaultLayout = state.layouts.find((l) => l.isDefault);
        state.currentLayoutId = defaultLayout?.id || state.layouts[0].id;
      }

      // Load widgets for current layout
      loadWidgetsForCurrentLayout();
    } catch (error) {
      console.error("Failed to load dashboard layouts:", error);
      // Create emergency default layout
      const defaultLayout = createDefaultLayout();
      state.layouts = [defaultLayout];
      state.currentLayoutId = defaultLayout.id;
      loadWidgetsForCurrentLayout();
    } finally {
      setLoading(false);
    }
  };

  const saveLayoutsToStorage = async () => {
    try {
      localStorage.setItem("dashboard-layouts", JSON.stringify(state.layouts));
    } catch (error) {
      console.error("Failed to save layouts to storage:", error);
    }
  };

  const loadWidgetsForCurrentLayout = () => {
    const layout = currentLayout.value;
    if (!layout) return;

    state.widgets = layout.widgets;
    state.activeWidgets = layout.widgets
      .filter((w) => w.enabled)
      .map((w) => w.id);

    // Initialize refresh intervals
    layout.widgets.forEach((widget) => {
      if (widget.refreshInterval > 0) {
        state.refreshIntervals[widget.id] = widget.refreshInterval;
      }
    });
  };

  const switchLayout = async (layoutId: string) => {
    const layout = state.layouts.find((l) => l.id === layoutId);
    if (!layout) return;

    state.currentLayoutId = layoutId;
    loadWidgetsForCurrentLayout();

    // Save current layout preference
    localStorage.setItem("dashboard-current-layout", layoutId);
  };

  const createLayout = async (
    name: string,
    description?: string,
    copyFromId?: string,
  ) => {
    let newLayout: DashboardLayout;

    if (copyFromId) {
      const sourceLayout = state.layouts.find((l) => l.id === copyFromId);
      if (!sourceLayout) throw new Error("Source layout not found");

      newLayout = {
        ...sourceLayout,
        id: `layout-${Date.now()}`,
        name,
        description,
        isDefault: false,
        createdAt: new Date(),
        updatedAt: new Date(),
        widgets: sourceLayout.widgets.map((widget) => ({
          ...widget,
          id: `widget-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        })),
      };
    } else {
      newLayout = createDefaultLayout();
      newLayout.id = `layout-${Date.now()}`;
      newLayout.name = name;
      newLayout.description = description;
      newLayout.isDefault = false;
    }

    state.layouts.push(newLayout);
    await saveLayoutsToStorage();
    return newLayout;
  };

  const updateLayout = async (
    layoutId: string,
    updates: Partial<DashboardLayout>,
  ) => {
    const layoutIndex = state.layouts.findIndex((l) => l.id === layoutId);
    if (layoutIndex === -1) return;

    state.layouts[layoutIndex] = {
      ...state.layouts[layoutIndex],
      ...updates,
      updatedAt: new Date(),
    };

    await saveLayoutsToStorage();
  };

  const deleteLayout = async (layoutId: string) => {
    const layoutIndex = state.layouts.findIndex((l) => l.id === layoutId);
    if (layoutIndex === -1) return;

    // Can't delete the last layout
    if (state.layouts.length <= 1) {
      throw new Error("Cannot delete the last remaining layout");
    }

    // If deleting current layout, switch to another
    if (state.currentLayoutId === layoutId) {
      const newCurrentLayout =
        state.layouts.find((l) => l.id !== layoutId && l.isDefault) ||
        state.layouts.find((l) => l.id !== layoutId);
      if (newCurrentLayout) {
        await switchLayout(newCurrentLayout.id);
      }
    }

    state.layouts.splice(layoutIndex, 1);
    await saveLayoutsToStorage();
  };

  const addWidget = async (
    widgetType: string,
    position?: Partial<WidgetPosition>,
  ) => {
    const widgetDef = availableWidgets.value.find((w) => w.type === widgetType);
    if (!widgetDef) throw new Error("Widget type not found");

    const layout = currentLayout.value;
    if (!layout) throw new Error("No current layout");

    const newWidget: DashboardWidget = {
      id: `widget-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      type: widgetDef.type,
      title: widgetDef.name,
      component: widgetDef.component,
      position: {
        x: position?.x ?? widgetDef.defaultPosition.x ?? 0,
        y: position?.y ?? widgetDef.defaultPosition.y ?? 0,
        w: position?.w ?? widgetDef.defaultPosition.w ?? 3,
        h: position?.h ?? widgetDef.defaultPosition.h ?? 3,
      },
      size: widgetDef.defaultSize,
      settings: { ...widgetDef.defaultSettings },
      refreshInterval: widgetDef.defaultSettings.refreshInterval || 30000,
      permissions: widgetDef.permissions,
      enabled: true,
      draggable: true,
      resizable: true,
    };

    layout.widgets.push(newWidget);
    state.widgets.push(newWidget);
    state.activeWidgets.push(newWidget.id);

    await updateLayout(layout.id, { widgets: layout.widgets });
    return newWidget;
  };

  const removeWidget = async (widgetId: string) => {
    const layout = currentLayout.value;
    if (!layout) return;

    const widgetIndex = layout.widgets.findIndex((w) => w.id === widgetId);
    if (widgetIndex === -1) return;

    layout.widgets.splice(widgetIndex, 1);

    const stateWidgetIndex = state.widgets.findIndex((w) => w.id === widgetId);
    if (stateWidgetIndex !== -1) {
      state.widgets.splice(stateWidgetIndex, 1);
    }

    const activeIndex = state.activeWidgets.indexOf(widgetId);
    if (activeIndex !== -1) {
      state.activeWidgets.splice(activeIndex, 1);
    }

    // Clean up widget data
    delete state.widgetData[widgetId];
    delete state.refreshIntervals[widgetId];
    delete state.lastUpdated[widgetId];

    await updateLayout(layout.id, { widgets: layout.widgets });
  };

  const updateWidget = async (
    widgetId: string,
    updates: Partial<DashboardWidget>,
  ) => {
    const layout = currentLayout.value;
    if (!layout) return;

    const widgetIndex = layout.widgets.findIndex((w) => w.id === widgetId);
    if (widgetIndex === -1) return;

    layout.widgets[widgetIndex] = {
      ...layout.widgets[widgetIndex],
      ...updates,
    };

    const stateWidgetIndex = state.widgets.findIndex((w) => w.id === widgetId);
    if (stateWidgetIndex !== -1) {
      state.widgets[stateWidgetIndex] = layout.widgets[widgetIndex];
    }

    await updateLayout(layout.id, { widgets: layout.widgets });
  };

  const updateWidgetData = (widgetId: string, data: any) => {
    state.widgetData[widgetId] = data;
    state.lastUpdated[widgetId] = new Date();
  };

  const updateGlobalSettings = async (
    settings: Partial<typeof state.globalSettings>,
  ) => {
    Object.assign(state.globalSettings, settings);
    localStorage.setItem(
      "dashboard-global-settings",
      JSON.stringify(state.globalSettings),
    );
  };

  const loadGlobalSettings = () => {
    try {
      const saved = localStorage.getItem("dashboard-global-settings");
      if (saved) {
        Object.assign(state.globalSettings, JSON.parse(saved));
      }
    } catch (error) {
      console.error("Failed to load global settings:", error);
    }
  };

  // Initialize store
  const initialize = async () => {
    loadGlobalSettings();
    await loadLayouts();

    // Restore current layout preference
    const savedLayoutId = localStorage.getItem("dashboard-current-layout");
    if (savedLayoutId && state.layouts.find((l) => l.id === savedLayoutId)) {
      state.currentLayoutId = savedLayoutId;
      loadWidgetsForCurrentLayout();
    }
  };

  return {
    // State
    state,
    availableWidgets: availableWidgets as any,

    // Computed
    currentLayout,
    activeWidgetData,
    widgetsByCategory,

    // Actions
    setLoading,
    setEditMode,
    loadLayouts,
    switchLayout,
    createLayout,
    updateLayout,
    deleteLayout,
    addWidget,
    removeWidget,
    updateWidget,
    updateWidgetData,
    updateGlobalSettings,
    initialize,
  };
});

export type DashboardStore = ReturnType<typeof useDashboardStore>;
