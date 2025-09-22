<template>
  <div class="updates-view">
    <!-- Header -->
    <div class="updates-header">
      <div class="header-content">
        <div class="header-info">
          <h1 class="page-title">
            <el-icon><Refresh /></el-icon>
            更新管理
          </h1>
          <p class="page-description">
            管理容器更新、安排维护计划并跟踪进度
          </p>
        </div>

        <div class="header-actions">
          <el-button-group>
            <el-button
              :icon="Refresh"
              :loading="checkingUpdates"
              @click="checkForUpdates(true)"
            >
              检查更新
            </el-button>
            <el-button
:icon="Calendar" @click="showScheduler = true"
>
              计划
            </el-button>
            <el-button
:icon="Setting" @click="showPolicies = true"
>
              策略
            </el-button>
          </el-button-group>

          <el-dropdown
trigger="click" @command="handleBulkAction"
>
            <el-button
type="primary" :disabled="!hasSelection"
>
              批量操作
              <el-icon class="el-icon--right">
                <ArrowDown />
              </el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="update-all">
                  <el-icon><Refresh /></el-icon>
                  更新所有已选择
                </el-dropdown-item>
                <el-dropdown-item command="schedule-all">
                  <el-icon><Calendar /></el-icon>
                  计划所有已选择
                </el-dropdown-item>
                <el-dropdown-item command="ignore-all">
                  <el-icon><CircleClose /></el-icon>
                  忽略所有已选择
                </el-dropdown-item>
                <el-dropdown-item divided command="export-selected">
                  <el-icon><Download /></el-icon>
                  导出已选择
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <!-- Quick Stats -->
      <div class="quick-stats">
        <div class="stat-card">
          <div class="stat-icon total">
            <el-icon><Refresh /></el-icon>
          </div>
          <div class="stat-content">
            <span class="stat-value">{{
              updateAnalytics.totalUpdatesAvailable
            }}</span>
            <span class="stat-label">可用更新</span>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon security">
            <el-icon><Warning /></el-icon>
          </div>
          <div class="stat-content">
            <span class="stat-value">{{
              updateAnalytics.securityUpdates
            }}</span>
            <span class="stat-label">安全更新</span>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon critical">
            <el-icon><CircleClose /></el-icon>
          </div>
          <div class="stat-content">
            <span class="stat-value">{{
              updateAnalytics.criticalUpdates
            }}</span>
            <span class="stat-label">重要更新</span>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon running">
            <el-icon><Loading /></el-icon>
          </div>
          <div class="stat-content">
            <span class="stat-value">{{ runningUpdatesCount }}</span>
            <span class="stat-label">运行中更新</span>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon scheduled">
            <el-icon><Clock /></el-icon>
          </div>
          <div class="stat-content">
            <span class="stat-value">{{ scheduledUpdatesCount }}</span>
            <span class="stat-label">计划更新</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Running Updates Panel -->
    <div v-if="runningUpdates.length > 0" class="running-updates-panel">
      <div class="panel-header">
        <h3>
          <el-icon><Loading /></el-icon>
          运行中更新 ({{ runningUpdates.length }})
        </h3>
        <el-button
          text
          type="primary"
          @click="showAllRunningUpdates = !showAllRunningUpdates"
        >
          {{ showAllRunningUpdates ? "显示更少" : "显示全部" }}
        </el-button>
      </div>

      <div class="running-updates-list">
        <UpdateProgress
          v-for="update in displayedRunningUpdates"
          :key="update.id"
          :update="update"
          @cancel="handleCancelUpdate"
        />
      </div>
    </div>

    <!-- Filters and Controls -->
    <div class="updates-controls">
      <div class="filters-section">
        <el-button
:icon="Filter" @click="showFilters = !showFilters"
>
          筛选器
          <el-badge
            v-if="activeFiltersCount > 0"
            :value="activeFiltersCount"
            class="filter-badge"
          />
        </el-button>

        <el-select
          v-model="quickFilter"
          placeholder="快速筛选"
          style="width: 150px"
          @change="applyQuickFilter"
        >
          <el-option label="所有更新" value="all" />
          <el-option label="仅安全更新" value="security" />
          <el-option label="仅重要更新" value="critical" />
          <el-option label="待处理" value="pending" />
          <el-option label="已忽略" value="ignored" />
          <el-option label="已计划" value="scheduled" />
        </el-select>

        <el-input
          v-model="searchQuery"
          placeholder="搜索容器..."
          :prefix-icon="Search"
          style="width: 250px"
          clearable
          @input="handleSearch"
        />
      </div>

      <div class="view-controls">
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button label="list">
            <el-icon><List /></el-icon>
            列表
          </el-radio-button>
          <el-radio-button label="grid">
            <el-icon><Grid /></el-icon>
            网格
          </el-radio-button>
        </el-radio-group>

        <el-select
          v-model="sortConfig.field"
          style="width: 150px"
          @change="handleSort"
        >
          <el-option label="可用日期" value="available_date" />
          <el-option label="发布日期" value="release_date" />
          <el-option label="容器名称" value="container_name" />
          <el-option label="风险等级" value="risk_level" />
          <el-option label="大小" value="size" />
        </el-select>

        <el-button
          :icon="sortConfig.direction === 'asc' ? 'SortUp' : 'SortDown'"
          @click="toggleSortDirection"
        />

        <el-checkbox
v-model="autoRefresh" @change="setAutoRefresh"
>
          自动刷新
        </el-checkbox>
      </div>
    </div>

    <!-- Advanced Filters Panel -->
    <el-collapse-transition>
      <div v-show="showFilters" class="filters-panel">
        <div class="filter-groups">
          <div class="filter-group">
            <label>更新类型</label>
            <el-checkbox-group v-model="filters.updateType">
              <el-checkbox label="major"> 主版本 </el-checkbox>
              <el-checkbox label="minor"> 次版本 </el-checkbox>
              <el-checkbox label="patch"> 补丁 </el-checkbox>
              <el-checkbox label="security"> 安全 </el-checkbox>
              <el-checkbox label="hotfix"> 热修复 </el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="filter-group">
            <label>风险等级</label>
            <el-checkbox-group v-model="filters.riskLevel">
              <el-checkbox label="low"> 低 </el-checkbox>
              <el-checkbox label="medium"> 中 </el-checkbox>
              <el-checkbox label="high"> 高 </el-checkbox>
              <el-checkbox label="critical"> 严重 </el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="filter-group">
            <label>大小范围</label>
            <el-slider
              v-model="sizeRange"
              range
              :min="0"
              :max="1000"
              :format-tooltip="formatSize"
              @change="handleSizeFilter"
            />
          </div>

          <div class="filter-group">
            <label>状态</label>
            <el-checkbox-group v-model="statusFilters">
              <el-checkbox label="available"> 可用 </el-checkbox>
              <el-checkbox label="ignored"> 已忽略 </el-checkbox>
              <el-checkbox label="scheduled"> 已计划 </el-checkbox>
            </el-checkbox-group>
          </div>
        </div>

        <div class="filter-actions">
          <el-button @click="clearAllFilters"> 清除全部 </el-button>
          <el-button type="primary" @click="applyFilters">
            应用筛选器
          </el-button>
        </div>
      </div>
    </el-collapse-transition>

    <!-- Selection Bar -->
    <div v-if="hasSelection" class="selection-bar">
      <div class="selection-info">
        <el-checkbox
          :model-value="isAllSelected"
          :indeterminate="hasSelection && !isAllSelected"
          @change="selectAll"
        />
        <span>{{ selectedUpdates.size }} of {{ filteredUpdates.length }} updates
          selected</span>
      </div>

      <div class="selection-actions">
        <el-button
size="small" @click="selectByType('security')"
>
          选择安全更新
        </el-button>
        <el-button
size="small" @click="selectByType('critical')"
>
          选择重要更新
        </el-button>
        <el-button
size="small" @click="clearSelection"
>
          清除选择
        </el-button>
      </div>
    </div>

    <!-- Updates List/Grid -->
    <div class="updates-content" :class="viewMode">
      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>

      <div v-else-if="filteredUpdates.length === 0" class="empty-state">
        <el-empty
:image-size="200" description="暂无可用更新"
>
          <el-button type="primary" @click="checkForUpdates(true)">
            检查更新
          </el-button>
        </el-empty>
      </div>

      <div v-else>
        <!-- List View -->
        <div v-if="viewMode === 'list'" class="updates-list">
          <UpdateCard
            v-for="update in paginatedUpdates"
            :key="update.id"
            :update="update"
            :selected="selectedUpdates.has(update.id)"
            :loading="isUpdateLoading(update.containerId)"
            view-mode="list"
            @select="handleSelectUpdate"
            @update="handleUpdateContainer"
            @schedule="handleScheduleUpdate"
            @ignore="handleIgnoreUpdate"
            @compare="handleCompareVersions"
            @details="handleShowDetails"
          />
        </div>

        <!-- Grid View -->
        <div v-else class="updates-grid">
          <UpdateCard
            v-for="update in paginatedUpdates"
            :key="update.id"
            :update="update"
            :selected="selectedUpdates.has(update.id)"
            :loading="isUpdateLoading(update.containerId)"
            view-mode="grid"
            @select="handleSelectUpdate"
            @update="handleUpdateContainer"
            @schedule="handleScheduleUpdate"
            @ignore="handleIgnoreUpdate"
            @compare="handleCompareVersions"
            @details="handleShowDetails"
          />
        </div>

        <!-- Pagination -->
        <div class="pagination-container">
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :total="totalUpdates"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handlePageSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </div>

    <!-- Update Scheduler Dialog -->
    <UpdateScheduler
      v-model="showScheduler"
      :selected-updates="Array.from(selectedUpdates)"
      @scheduled="handleUpdateScheduled"
    />

    <!-- Update Policies Dialog -->
    <UpdatePolicies
      v-model="showPolicies"
      @policy-applied="handlePolicyApplied"
    />

    <!-- Bulk Update Manager Dialog -->
    <BulkUpdateManager
      v-model="showBulkManager"
      :selected-updates="Array.from(selectedUpdates)"
      @bulk-update-started="handleBulkUpdateStarted"
    />

    <!-- Version Comparison Dialog -->
    <VersionComparison
      v-model="showVersionComparison"
      :comparison-data="versionComparisonData"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Refresh,
  Calendar,
  Setting,
  ArrowDown,
  CircleClose,
  Download,
  Warning,
  Loading,
  Clock,
  Filter,
  Search,
  List,
  Grid,
} from "@element-plus/icons-vue";

// Components
import UpdateCard from "@/components/update/UpdateCard.vue";
import UpdateProgress from "@/components/update/UpdateProgress.vue";
import UpdateScheduler from "@/components/update/UpdateScheduler.vue";
import UpdatePolicies from "@/components/update/UpdatePoliciesDialog.vue";
import BulkUpdateManager from "@/components/update/BulkUpdateManager.vue";
import VersionComparison from "@/components/update/VersionComparison.vue";

// Store
import { useUpdatesStore } from "@/store/updates";
import { storeToRefs } from "pinia";
import { useUpdateWebSocket } from "@/services/updateWebSocket";

// Types
import type {
  UpdateFilter,
  VersionComparison as VersionComparisonData,
} from "@/types/updates";

// Store instance
const updatesStore = useUpdatesStore();
const { updateWebSocket } = useUpdateWebSocket();

// Reactive state from store
const {
  availableUpdates,
  runningUpdates,
  updateAnalytics,
  loading,
  checkingUpdates,
  currentPage,
  pageSize,
  totalUpdates,
  filters,
  sortConfig,
  selectedUpdates,
  viewMode,
  showFilters,
  autoRefresh,
  hasSelection,
  isAllSelected,
  filteredUpdates,
  scheduledUpdatesCount,
  runningUpdatesCount,
} = storeToRefs(updatesStore);

// Local reactive state
const searchQuery = ref("");
const quickFilter = ref("all");
const sizeRange = ref([0, 1000]);
const statusFilters = ref(["available"]);
const showScheduler = ref(false);
const showPolicies = ref(false);
const showBulkManager = ref(false);
const showVersionComparison = ref(false);
const showAllRunningUpdates = ref(false);
const versionComparisonData = ref<VersionComparisonData | null>(null);

// Methods
const isUpdateLoading = (containerId: string): boolean => {
  return runningUpdates.value.some(
    (update) => update.containerId === containerId,
  );
};

// Computed properties
const activeFiltersCount = computed(() => {
  let count = 0;
  if (filters.value.updateType?.length) count++;
  if (filters.value.riskLevel?.length) count++;
  if (
    filters.value.size?.min !== undefined ||
    filters.value.size?.max !== undefined
  )
    count++;
  if (searchQuery.value) count++;
  if (quickFilter.value !== "all") count++;
  return count;
});

const paginatedUpdates = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  const end = start + pageSize.value;
  return filteredUpdates.value.slice(start, end);
});

const displayedRunningUpdates = computed(() => {
  return showAllRunningUpdates.value
    ? runningUpdates.value
    : runningUpdates.value.slice(0, 3);
});

// Methods
const checkForUpdates = async (force = false) => {
  try {
    await updatesStore.checkForUpdates(undefined, force);
  } catch (error) {
    console.error("Failed to check for updates:", error);
  }
};

const handleSelectUpdate = (updateId: string) => {
  updatesStore.toggleSelection(updateId);
};

const selectAll = () => {
  updatesStore.selectAll();
};

const clearSelection = () => {
  updatesStore.clearSelection();
};

const selectByType = (type: "security" | "critical" | "all") => {
  updatesStore.selectByType(type);
};

const handleUpdateContainer = async (updateId: string) => {
  try {
    await updatesStore.startUpdate(updateId);
  } catch (error) {
    console.error("Failed to start update:", error);
  }
};

const handleScheduleUpdate = (updateId: string) => {
  selectedUpdates.value.clear();
  selectedUpdates.value.add(updateId);
  showScheduler.value = true;
};

const handleIgnoreUpdate = async (updateId: string) => {
  try {
    const result = await ElMessageBox.prompt(
      "请提供忽略此更新的原因：",
      "忽略更新",
      {
        confirmButtonText: "忽略",
        cancelButtonText: "取消",
        inputType: "textarea",
        inputPlaceholder: "忽略此更新的原因...",
      },
    );

    await updatesStore.ignoreUpdate(updateId, result.value);
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to ignore update:", error);
    }
  }
};

const handleCompareVersions = async (updateId: string) => {
  try {
    const comparison = await updatesStore.compareVersions(updateId);
    if (comparison) {
      versionComparisonData.value = comparison;
      showVersionComparison.value = true;
    }
  } catch (error) {
    console.error("Failed to compare versions:", error);
  }
};

const handleShowDetails = (updateId: string) => {
  const update = availableUpdates.value.find((u) => u.id === updateId);
  if (update) {
    updatesStore.toggleExpanded(updateId);
  }
};

const handleCancelUpdate = async (runningUpdateId: string) => {
  try {
    await ElMessageBox.confirm(
      "您确定要取消此更新吗？这可能会使容器处于不一致状态。",
      "取消更新",
      {
        confirmButtonText: "是的，取消",
        cancelButtonText: "否",
        type: "warning",
      },
    );

    await updatesStore.cancelUpdate(runningUpdateId);
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to cancel update:", error);
    }
  }
};

const handleBulkAction = (command: string) => {
  switch (command) {
    case "update-all":
      showBulkManager.value = true;
      break;
    case "schedule-all":
      showScheduler.value = true;
      break;
    case "ignore-all":
      handleBulkIgnore();
      break;
    case "export-selected":
      handleExportSelected();
      break;
  }
};

const handleBulkIgnore = async () => {
  try {
    await ElMessageBox.confirm(
      `您确定要忽略选中的 ${selectedUpdates.value.size} 个更新吗？`,
      "批量忽略更新",
      {
        confirmButtonText: "是的，忽略全部",
        cancelButtonText: "取消",
        type: "warning",
      },
    );

    const promises = Array.from(selectedUpdates.value).map((updateId) =>
      updatesStore.ignoreUpdate(updateId, "批量忽略操作"),
    );

    await Promise.all(promises);
    clearSelection();
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to bulk ignore updates:", error);
    }
  }
};

const handleExportSelected = async () => {
  if (selectedUpdates.value.size === 0) {
    ElMessage.warning("请先选择要导出的更新记录");
    return;
  }

  try {
    // Ask for export format
    const formatChoice = await ElMessageBox.confirm(
      "请选择导出格式：CSV、JSON 或 Excel",
      "导出选中项",
      {
        distinguishCancelAndClose: true,
        confirmButtonText: "CSV格式",
        cancelButtonText: "取消",
        type: "info",
      }
    ).then(() => "csv")
    .catch((action) => {
      if (action === "close") {
        return ElMessageBox.confirm(
          "请选择导出格式：CSV、JSON 或 Excel",
          "导出选中项",
          {
            distinguishCancelAndClose: true,
            confirmButtonText: "JSON格式",
            cancelButtonText: "取消",
            type: "info",
          }
        ).then(() => "json")
        .catch(() => "excel");
      }
      throw action;
    });

    const selectedIds = Array.from(selectedUpdates.value);
    const exportFormat = formatChoice as "csv" | "json" | "excel";

    // Create filter for selected updates
    const exportFilters: UpdateFilter = {
      containerIds: selectedIds, // Use containerIds instead of updateIds
    };

    await updatesStore.exportUpdateHistory(exportFormat, exportFilters);
    clearSelection();
    ElMessage.success(`已导出 ${selectedIds.length} 项更新记录`);
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to export selected updates:", error);
      ElMessage.error("导出失败");
    }
  }
};

const handleUpdateScheduled = () => {
  showScheduler.value = false;
  clearSelection();
  ElMessage.success("更新计划安排成功");
};

const handlePolicyApplied = () => {
  showPolicies.value = false;
  checkForUpdates();
};

const handleBulkUpdateStarted = () => {
  showBulkManager.value = false;
  clearSelection();
};

const applyQuickFilter = () => {
  const newFilters: UpdateFilter = {};

  switch (quickFilter.value) {
    case "security":
      newFilters.securityOnly = true;
      break;
    case "critical":
      newFilters.riskLevel = ["critical"];
      break;
    case "pending":
      newFilters.ignored = false;
      newFilters.scheduled = false;
      break;
    case "ignored":
      newFilters.ignored = true;
      break;
    case "scheduled":
      newFilters.scheduled = true;
      break;
  }

  updatesStore.setFilters(newFilters);
};

const handleSearch = () => {
  if (searchQuery.value) {
    updatesStore.setFilters({
      ...filters.value,
      containerName: searchQuery.value,
    });
  } else {
    const newFilters = { ...filters.value };
    delete newFilters.containerName;
    updatesStore.setFilters(newFilters);
  }
};

const handleSort = () => {
  updatesStore.setSorting(sortConfig.value.field);
};

const toggleSortDirection = () => {
  updatesStore.setSorting(
    sortConfig.value.field,
    sortConfig.value.direction === "asc" ? "desc" : "asc",
  );
};

const handleSizeFilter = () => {
  updatesStore.setFilters({
    ...filters.value,
    size: {
      min: sizeRange.value[0] * 1024 * 1024, // Convert MB to bytes
      max: sizeRange.value[1] * 1024 * 1024,
    },
  });
};

const applyFilters = () => {
  updatesStore.setFilters(filters.value);
  showFilters.value = false;
};

const clearAllFilters = () => {
  updatesStore.clearFilters();
  searchQuery.value = "";
  quickFilter.value = "all";
  sizeRange.value = [0, 1000];
  statusFilters.value = ["available"];
};

const handlePageSizeChange = (newSize: number) => {
  pageSize.value = newSize;
};

const handlePageChange = (newPage: number) => {
  currentPage.value = newPage;
};

const formatSize = (value: number) => {
  return `${value} MB`;
};

const setAutoRefresh = (enabled: string | number | boolean) => {
  updatesStore.setAutoRefresh(Boolean(enabled));
};

// Lifecycle hooks
onMounted(async () => {
  // Initialize data
  await Promise.all([
    updatesStore.checkForUpdates(),
    updatesStore.fetchRunningUpdates(),
    updatesStore.getUpdateAnalytics(),
  ]);

  // Setup WebSocket connection
  try {
    await updateWebSocket.connect({
      onUpdateProgress: updatesStore.handleWebSocketMessage,
      onUpdateCompleted: updatesStore.handleWebSocketMessage,
      onUpdateFailed: updatesStore.handleWebSocketMessage,
      onUpdateAvailable: updatesStore.handleWebSocketMessage,
      onUpdateNotification: updatesStore.handleWebSocketMessage,
      onConnected: () => {
        updatesStore.setWebSocketConnected(true);
        updateWebSocket.subscribeToAllUpdates();
        updateWebSocket.subscribeToNotifications();
      },
      onDisconnected: () => {
        updatesStore.setWebSocketConnected(false);
      },
    });
  } catch (error) {
    console.error("Failed to connect to WebSocket:", error);
  }

  // Start auto-refresh if enabled
  if (autoRefresh) {
    updatesStore.startAutoRefresh();
  }
});

onUnmounted(() => {
  // Clean up
  updatesStore.stopAutoRefresh();
  updateWebSocket.disconnect();
});

// Watch for auto-refresh changes
watch(
  () => autoRefresh,
  (enabled) => {
    if (enabled) {
      updatesStore.startAutoRefresh();
    } else {
      updatesStore.stopAutoRefresh();
    }
  },
);
</script>

<style scoped lang="scss">
.updates-view {
  padding: 24px;
  background: var(--el-bg-color-page);
  min-height: 100vh;
}

.updates-header {
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.header-info {
  .page-title {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0 0 8px 0;
    font-size: 28px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .page-description {
    margin: 0;
    color: var(--el-text-color-regular);
    font-size: 14px;
  }
}

.header-actions {
  display: flex;
  gap: 12px;
}

.quick-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--el-color-primary);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }
}

.stat-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: white;

  &.total {
    background: linear-gradient(45deg, #409eff, #337ecc);
  }

  &.security {
    background: linear-gradient(45deg, #f56c6c, #e6a23c);
  }

  &.critical {
    background: linear-gradient(45deg, #f56c6c, #c73e1d);
  }

  &.running {
    background: linear-gradient(45deg, #67c23a, #529b2e);
  }

  &.scheduled {
    background: linear-gradient(45deg, #909399, #73767a);
  }
}

.stat-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: var(--el-text-color-regular);
  line-height: 1;
}

.running-updates-panel {
  margin-bottom: 24px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;

  h3 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 16px;
    font-weight: 600;
  }
}

.running-updates-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.updates-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.filters-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-badge {
  margin-left: 8px;
}

.view-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filters-panel {
  margin-bottom: 16px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.filter-groups {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 16px;
}

.filter-group {
  label {
    display: block;
    margin-bottom: 8px;
    font-weight: 500;
    color: var(--el-text-color-regular);
    font-size: 13px;
  }
}

.filter-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.selection-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: 6px;
}

.selection-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--el-color-primary);
  font-weight: 500;
}

.selection-actions {
  display: flex;
  gap: 8px;
}

.updates-content {
  &.list .updates-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  &.grid .updates-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 16px;
  }
}

.loading-container {
  padding: 20px;
}

.empty-state {
  padding: 60px 20px;
  text-align: center;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 24px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

@media (max-width: 768px) {
  .updates-view {
    padding: 16px;
  }

  .header-content {
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
  }

  .header-actions {
    justify-content: center;
  }

  .quick-stats {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  }

  .updates-controls {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .filters-section,
  .view-controls {
    flex-wrap: wrap;
    justify-content: center;
  }

  .filter-groups {
    grid-template-columns: 1fr;
  }

  .selection-bar {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .selection-actions {
    justify-content: center;
  }

  .updates-content.grid .updates-grid {
    grid-template-columns: 1fr;
  }
}
</style>
