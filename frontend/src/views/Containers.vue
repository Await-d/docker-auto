<template>
  <div class="containers-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="title-section">
          <h1 class="page-title">
            <el-icon><Box /></el-icon>
            容器管理
          </h1>
          <p class="page-subtitle">
            管理您的 Docker 容器及其生命周期
          </p>
        </div>

        <div class="header-actions">
          <el-button
            v-if="hasPermission('container:create')"
            type="primary"
            @click="showCreateDialog = true"
          >
            <el-icon><Plus /></el-icon>
            新建容器
          </el-button>

          <el-dropdown @command="handleHeaderAction">
            <el-button>
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="refresh">
                  <el-icon><Refresh /></el-icon>
                  刷新
                </el-dropdown-item>
                <el-dropdown-item command="check-updates">
                  <el-icon><Download /></el-icon>
                  检查更新
                </el-dropdown-item>
                <el-dropdown-item command="templates">
                  <el-icon><Document /></el-icon>
                  模板
                </el-dropdown-item>
                <el-dropdown-item command="export">
                  <el-icon><Upload /></el-icon>
                  导出配置
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <!-- Stats Cards -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon running">
            <el-icon><SuccessFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">
              {{ runningContainers.length }}
            </div>
            <div class="stat-label">运行中</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon stopped">
            <el-icon><CircleCloseFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">
              {{ stoppedContainers.length }}
            </div>
            <div class="stat-label">已停止</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon updates">
            <el-icon><Download /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">
              {{ containersWithUpdates.length }}
            </div>
            <div class="stat-label">可用更新</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon unhealthy">
            <el-icon><WarningFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">
              {{ unhealthyContainers.length }}
            </div>
            <div class="stat-label">不健康</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Filters and Controls -->
    <div class="filters-section">
      <div class="filters-row">
        <div class="left-controls">
          <!-- Search -->
          <el-input
            v-model="searchQuery"
            placeholder="搜索容器..."
            :prefix-icon="Search"
            clearable
            class="search-input"
          />

          <!-- Quick Filters -->
          <el-select
            v-model="statusFilter"
            placeholder="状态"
            multiple
            collapse-tags
            collapse-tags-tooltip
            clearable
            class="status-filter"
          >
            <el-option label="运行中" value="running" />
            <el-option label="已停止" value="exited" />
            <el-option label="已暂停" value="paused" />
            <el-option label="重启中" value="restarting" />
          </el-select>

          <!-- Advanced Filters Toggle -->
          <el-button
            :type="showFilters ? 'primary' : 'default'"
            @click="showFilters = !showFilters"
          >
            <el-icon><Filter /></el-icon>
            过滤器
          </el-button>
        </div>

        <div class="right-controls">
          <!-- View Mode Toggle -->
          <el-radio-group v-model="viewMode" size="small">
            <el-radio-button value="grid">
              <el-icon><Grid /></el-icon>
            </el-radio-button>
            <el-radio-button value="list">
              <el-icon><List /></el-icon>
            </el-radio-button>
          </el-radio-group>

          <!-- Bulk Actions -->
          <el-dropdown
v-if="hasSelection" @command="handleBulkAction"
>
            <el-button type="warning">
              <el-icon><Operation /></el-icon>
              批量操作 ({{ selectedContainers.size }})
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="start">
                  <el-icon><VideoPlay /></el-icon>
                  启动所选
                </el-dropdown-item>
                <el-dropdown-item command="stop">
                  <el-icon><VideoPause /></el-icon>
                  停止所选
                </el-dropdown-item>
                <el-dropdown-item command="restart">
                  <el-icon><Refresh /></el-icon>
                  重启所选
                </el-dropdown-item>
                <el-dropdown-item command="update">
                  <el-icon><Download /></el-icon>
                  更新所选
                </el-dropdown-item>
                <el-dropdown-item command="delete" divided>
                  <el-icon><Delete /></el-icon>
                  删除所选
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <!-- Advanced Filters Panel -->
      <el-collapse-transition>
        <div v-show="showFilters" class="advanced-filters">
          <div class="filters-grid">
            <div class="filter-group">
              <label>镜像</label>
              <el-input
                v-model="imageFilter"
                placeholder="按镜像过滤..."
                clearable
              />
            </div>

            <div class="filter-group">
              <label>注册表</label>
              <el-select
                v-model="registryFilter"
                placeholder="选择注册表"
                clearable
              >
                <el-option
                  v-for="registry in registries"
                  :key="registry"
                  :label="registry"
                  :value="registry"
                />
              </el-select>
            </div>

            <div class="filter-group">
              <label>标签</label>
              <el-input
                v-model="labelFilter"
                placeholder="key=value"
                clearable
              />
            </div>

            <div class="filter-group">
              <label>更新策略</label>
              <el-select
                v-model="updatePolicyFilter"
                placeholder="选择策略"
                clearable
              >
                <el-option label="自动更新" value="auto" />
                <el-option label="手动更新" value="manual" />
                <el-option label="已禁用" value="disabled" />
              </el-select>
            </div>
          </div>

          <div class="filters-actions">
            <el-button
type="primary" @click="applyFilters"
>
              应用过滤器
            </el-button>
            <el-button @click="clearAllFilters">
清除全部
</el-button>
          </div>
        </div>
      </el-collapse-transition>
    </div>

    <!-- Container List/Grid -->
    <div class="containers-content">
      <el-loading-directive v-loading="loading">
        <!-- Selection Header -->
        <div v-if="containers.length > 0" class="selection-header">
          <el-checkbox
            v-model="isAllSelected"
            :indeterminate="hasSelection && !isAllSelected"
            @change="selectAll"
          >
            全选 ({{ containers.length }})
          </el-checkbox>

          <div class="sort-controls">
            <span>排序方式:</span>
            <el-select
              :model-value="sortConfig.field"
              size="small"
              @update:model-value="handleSortChange"
            >
              <el-option label="名称" value="name" />
              <el-option label="状态" value="status" />
              <el-option label="创建时间" value="createdAt" />
              <el-option label="更新时间" value="updatedAt" />
              <el-option label="CPU 使用率" value="cpu" />
              <el-option label="内存使用率" value="memory" />
            </el-select>

            <el-button
size="small" @click="toggleSortDirection"
>
              <el-icon>
                <component
                  :is="sortConfig.direction === 'asc' ? 'SortUp' : 'SortDown'"
                />
              </el-icon>
            </el-button>
          </div>
        </div>

        <!-- Grid View -->
        <div v-if="viewMode === 'grid'" class="containers-grid">
          <ContainerCard
            v-for="container in filteredContainers"
            :key="container.id"
            :container="container"
            :selected="selectedContainers.has(container.id)"
            :loading="isOperationLoading(container.id)"
            @select="toggleSelection(container.id)"
            @action="handleContainerAction"
            @click="goToContainer(container.id)"
          />
        </div>

        <!-- List View -->
        <div v-if="viewMode === 'list'" class="containers-list">
          <el-table
            :data="filteredContainers"
            row-key="id"
            stripe
            @selection-change="handleSelectionChange"
          >
            <el-table-column type="selection" width="55" />

            <el-table-column prop="name" label="名称" min-width="200">
              <template #default="{ row }">
                <div class="container-name">
                  <el-tag
                    :type="getStatusType(row.status)"
                    size="small"
                    effect="dark"
                    class="status-tag"
                  >
                    {{ translateStatus(row.status) }}
                  </el-tag>
                  <span
class="name-link" @click="goToContainer(row.id)"
>
                    {{ row.name }}
                  </span>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="image" label="镜像" min-width="250">
              <template #default="{ row }">
                <div class="image-info">
                  <span class="image-name">{{ row.image }}</span>
                  <el-tag size="small" class="tag-badge">
                    {{ row.tag }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="资源" min-width="150">
              <template #default="{ row }">
                <div class="resource-info">
                  <div class="resource-item">
                    <span class="resource-label">CPU:</span>
                    <span class="resource-value">{{
                      formatPercentage(row.resourceUsage.cpu.usage)
                    }}</span>
                  </div>
                  <div class="resource-item">
                    <span class="resource-label">内存:</span>
                    <span class="resource-value">{{
                      formatPercentage(row.resourceUsage.memory.percentage)
                    }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="健康状态" width="100">
              <template #default="{ row }">
                <el-tag
:type="getHealthType(row.health.status)" size="small"
>
                  {{ translateHealthStatus(row.health.status) }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column label="更新" width="80">
              <template #default="{ row }">
                <el-badge
                  v-if="hasAvailableUpdate(row.id)"
                  is-dot
                  type="warning"
                >
                  <el-icon><Download /></el-icon>
                </el-badge>
              </template>
            </el-table-column>

            <el-table-column label="创建时间" width="120">
              <template #default="{ row }">
                {{ formatDate(row.createdAt) }}
              </template>
            </el-table-column>

            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <div class="table-actions">
                  <el-button-group size="small">
                    <el-button
                      v-if="row.status === 'exited'"
                      :loading="isOperationLoading(row.id)"
                      type="success"
                      @click="handleContainerAction('start', row.id)"
                    >
                      <el-icon><VideoPlay /></el-icon>
                    </el-button>

                    <el-button
                      v-if="row.status === 'running'"
                      :loading="isOperationLoading(row.id)"
                      type="warning"
                      @click="handleContainerAction('stop', row.id)"
                    >
                      <el-icon><VideoPause /></el-icon>
                    </el-button>

                    <el-button
                      :loading="isOperationLoading(row.id)"
                      @click="handleContainerAction('restart', row.id)"
                    >
                      <el-icon><Refresh /></el-icon>
                    </el-button>
                  </el-button-group>

                  <el-dropdown
                    @command="
                      (cmd: string) => handleContainerAction(cmd, row.id)
                    "
                  >
                    <el-button size="small">
                      <el-icon><MoreFilled /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="logs">
                          <el-icon><Document /></el-icon>
                          查看日志
                        </el-dropdown-item>
                        <el-dropdown-item command="terminal">
                          <el-icon><Monitor /></el-icon>
                          终端
                        </el-dropdown-item>
                        <el-dropdown-item command="update">
                          <el-icon><Download /></el-icon>
                          更新
                        </el-dropdown-item>
                        <el-dropdown-item command="edit">
                          <el-icon><Edit /></el-icon>
                          编辑
                        </el-dropdown-item>
                        <el-dropdown-item command="delete" divided>
                          <el-icon><Delete /></el-icon>
                          删除
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- Empty State -->
        <div v-if="!loading && containers.length === 0" class="empty-state">
          <div class="empty-icon">
            <el-icon><Box /></el-icon>
          </div>
          <h3>未找到容器</h3>
          <p>创建您的第一个容器以开始使用</p>
          <el-button
            v-if="hasPermission('container:create')"
            type="primary"
            @click="showCreateDialog = true"
          >
            <el-icon><Plus /></el-icon>
            创建容器
          </el-button>
        </div>
      </el-loading-directive>
    </div>

    <!-- Pagination -->
    <div v-if="totalPages > 1" class="pagination-section">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="totalContainers"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="handlePageChange"
        @size-change="handlePageSizeChange"
      />
    </div>

    <!-- Create Container Dialog -->
    <el-dialog
      v-model="showCreateDialog"
      title="创建新容器"
      width="80%"
      :before-close="handleCreateDialogClose"
    >
      <ContainerForm
        v-if="showCreateDialog"
        @submit="handleCreateContainer"
        @cancel="showCreateDialog = false"
      />
    </el-dialog>

    <!-- Edit Container Dialog -->
    <el-dialog
      v-model="showEditDialog"
      title="编辑容器"
      width="80%"
      :before-close="handleEditDialogClose"
    >
      <ContainerForm
        v-if="showEditDialog && editingContainer"
        :container="editingContainer"
        @submit="handleEditContainer"
        @cancel="showEditDialog = false"
      />
    </el-dialog>

    <!-- Log Viewer Dialog -->
    <el-dialog
      v-model="showLogsDialog"
      :title="`日志 - ${logsContainer?.name}`"
      width="90%"
      fullscreen
    >
      <LogViewer
        v-if="showLogsDialog && logsContainer"
        :container-id="logsContainer.id"
        :container-name="logsContainer.name"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useRouter } from "vue-router";
import { storeToRefs } from "pinia";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Box,
  Plus,
  MoreFilled,
  Refresh,
  Download,
  Document,
  Upload,
  SuccessFilled,
  CircleCloseFilled,
  WarningFilled,
  Search,
  Filter,
  Grid,
  List,
  Operation,
  VideoPlay,
  VideoPause,
  Delete,
  Edit,
  Monitor,
} from "@element-plus/icons-vue";

import { useContainerStore } from "@/store/containers";
import { useAuth } from "@/store/auth";
import ContainerCard from "@/components/container/ContainerCard.vue";
import ContainerForm from "@/components/container/ContainerForm.vue";
import LogViewer from "@/components/container/LogViewer.vue";

import type { Container, ContainerFormData } from "@/types/container";

const router = useRouter();
const containerStore = useContainerStore();
const { hasPermission } = useAuth();

// Store refs
const {
  containers,
  loading,
  selectedContainers,
  viewMode,
  showFilters,
  currentPage,
  pageSize,
  totalContainers,
  totalPages,
  sortConfig,
  runningContainers,
  stoppedContainers,
  unhealthyContainers,
  containersWithUpdates,
  isAllSelected,
  hasSelection,
  filteredContainers,
} = storeToRefs(containerStore);

// Local state
const searchQuery = ref("");
const statusFilter = ref<string[]>([]);
const imageFilter = ref("");
const registryFilter = ref("");
const labelFilter = ref("");
const updatePolicyFilter = ref("");
const showCreateDialog = ref(false);
const showEditDialog = ref(false);
const showLogsDialog = ref(false);
const editingContainer = ref<Container | null>(null);
const logsContainer = ref<Container | null>(null);
const registries = ref<string[]>([]);

// Auto-refresh
let autoRefreshInterval: NodeJS.Timeout | null = null;

// Computed
const hasAvailableUpdate = computed(() => (containerId: string) => {
  return containersWithUpdates.value.some((c) => c.id === containerId);
});

// Methods
function getStatusType(
  status: string,
): "success" | "info" | "warning" | "primary" | "danger" {
  const types: Record<
    string,
    "success" | "info" | "warning" | "primary" | "danger"
  > = {
    running: "success",
    exited: "info",
    paused: "warning",
    restarting: "warning",
    removing: "danger",
    dead: "danger",
  };
  return types[status] || "info";
}

function translateStatus(status: string): string {
  const translations: Record<string, string> = {
    running: "运行中",
    exited: "已停止",
    paused: "已暂停",
    restarting: "重启中",
    removing: "删除中",
    dead: "已终止",
    created: "已创建",
  };
  return translations[status] || status;
}

function getHealthType(
  health: string,
): "success" | "info" | "warning" | "primary" | "danger" {
  const types: Record<
    string,
    "success" | "info" | "warning" | "primary" | "danger"
  > = {
    healthy: "success",
    unhealthy: "danger",
    starting: "warning",
    none: "info",
  };
  return types[health] || "info";
}

function translateHealthStatus(health: string): string {
  const translations: Record<string, string> = {
    healthy: "健康",
    unhealthy: "不健康",
    starting: "启动中",
    none: "无",
  };
  return translations[health] || health;
}

function formatPercentage(value: number) {
  return `${Math.round(value)}%`;
}

function formatDate(date: Date | string) {
  return new Date(date).toLocaleDateString();
}

function handleHeaderAction(command: string) {
  switch (command) {
    case "refresh":
      containerStore.refreshData();
      break;
    case "check-updates":
      containerStore.checkUpdates();
      break;
    case "templates":
      // Navigate to templates
      break;
    case "export":
      // Handle export
      break;
  }
}

function handleBulkAction(command: string) {
  const selectedIds = Array.from(selectedContainers.value);

  if (selectedIds.length === 0) {
    ElMessage.warning("未选择容器");
    return;
  }

  const confirmAction = async () => {
    try {
      await containerStore.performBulkOperation({
        action: command as any,
        containers: selectedIds,
      });
    } catch (error) {
      console.error("批量操作失败:", error);
    }
  };

  if (command === "delete") {
    ElMessageBox.confirm(
      `确定要删除 ${selectedIds.length} 个容器吗?`,
      "确认删除",
      {
        type: "warning",
        confirmButtonText: "删除",
        cancelButtonText: "取消",
      },
    ).then(confirmAction);
  } else {
    confirmAction();
  }
}

function handleContainerAction(action: string, containerId: string) {
  const container = containers.value.find((c) => c.id === containerId);
  if (!container) return;

  switch (action) {
    case "start":
    case "stop":
    case "restart":
    case "pause":
    case "unpause":
      containerStore.performOperation(containerId, action);
      break;
    case "logs":
      logsContainer.value = container;
      showLogsDialog.value = true;
      break;
    case "terminal":
      // Handle terminal access
      break;
    case "update":
      containerStore.updateContainerImage(containerId);
      break;
    case "edit":
      editingContainer.value = container;
      showEditDialog.value = true;
      break;
    case "delete":
      ElMessageBox.confirm(
        `确定要删除容器 "${container.name}" 吗?`,
        "确认删除",
        {
          type: "warning",
          confirmButtonText: "删除",
          cancelButtonText: "取消",
        },
      ).then(() => {
        containerStore.deleteContainer(containerId);
      });
      break;
  }
}

function handleSelectionChange(selection: Container[]) {
  containerStore.clearSelection();
  selection.forEach((container) => {
    containerStore.toggleSelection(container.id);
  });
}

function selectAll() {
  containerStore.selectAll();
}

function toggleSelection(id: string) {
  containerStore.toggleSelection(id);
}

function isOperationLoading(id: string) {
  return containerStore.isOperationLoading(id);
}

function goToContainer(id: string) {
  router.push(`/containers/${id}`);
}

function handleSortChange(field: string) {
  containerStore.setSorting(field as any);
}

function toggleSortDirection() {
  containerStore.setSorting(
    sortConfig.value.field,
    sortConfig.value.direction === "asc" ? "desc" : "asc",
  );
}

function applyFilters() {
  const newFilters = {
    search: searchQuery.value,
    status:
      statusFilter.value.length > 0 ? (statusFilter.value as any) : undefined,
    image: imageFilter.value || undefined,
    registry: registryFilter.value || undefined,
    updatePolicy: updatePolicyFilter.value || undefined,
  };

  containerStore.setFilters(newFilters);
}

function clearAllFilters() {
  searchQuery.value = "";
  statusFilter.value = [];
  imageFilter.value = "";
  registryFilter.value = "";
  labelFilter.value = "";
  updatePolicyFilter.value = "";
  containerStore.clearFilters();
}

function handlePageChange(page: number) {
  containerStore.fetchContainers(page);
}

function handlePageSizeChange(size: number) {
  pageSize.value = size;
  containerStore.fetchContainers(1);
}

function handleCreateContainer(
  data: ContainerFormData | Partial<ContainerFormData>,
) {
  // Ensure we have complete data for creation
  if (!data.name || !data.image) {
    throw new Error("容器名称和镜像是必需的");
  }

  containerStore.createContainer(data as ContainerFormData).then(() => {
    showCreateDialog.value = false;
  });
}

function handleEditContainer(data: Partial<ContainerFormData>) {
  if (!editingContainer.value) return;

  containerStore.updateContainer(editingContainer.value.id, data).then(() => {
    showEditDialog.value = false;
    editingContainer.value = null;
  });
}

function handleCreateDialogClose(done: () => void) {
  ElMessageBox.confirm("放弃更改并关闭?")
    .then(() => done())
    .catch(() => {
      // User cancelled the dialog close
      console.log('User cancelled closing dialog');
    });
}

function handleEditDialogClose(done: () => void) {
  ElMessageBox.confirm("放弃更改并关闭?")
    .then(() => done())
    .catch(() => {
      // User cancelled the dialog close
      console.log('User cancelled closing dialog');
    });
}

function startAutoRefresh() {
  if (autoRefreshInterval) {
    clearInterval(autoRefreshInterval);
  }

  autoRefreshInterval = setInterval(() => {
    if (!loading.value) {
      containerStore.refreshData();
    }
  }, 30000); // Refresh every 30 seconds
}

function stopAutoRefresh() {
  if (autoRefreshInterval) {
    clearInterval(autoRefreshInterval);
    autoRefreshInterval = null;
  }
}

// Watch for search query changes
watch(searchQuery, (newValue) => {
  // Debounce search
  setTimeout(() => {
    if (searchQuery.value === newValue) {
      applyFilters();
    }
  }, 500);
});

// Lifecycle
onMounted(() => {
  containerStore.fetchContainers();
  containerStore.fetchTemplates();
  containerStore.checkUpdates();
  startAutoRefresh();
});

onUnmounted(() => {
  stopAutoRefresh();
});
</script>

<style scoped>
.containers-page {
  padding: 24px;
  min-height: 100vh;
  background-color: #f5f7fa;
}

.page-header {
  background: white;
  padding: 24px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.title-section h1 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 600;
  color: #303133;
}

.page-subtitle {
  margin: 0;
  color: #606266;
  font-size: 14px;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.stat-icon.running {
  background: #f0f9ff;
  color: #67c23a;
}

.stat-icon.stopped {
  background: #f5f5f5;
  color: #909399;
}

.stat-icon.updates {
  background: #fdf6ec;
  color: #e6a23c;
}

.stat-icon.unhealthy {
  background: #fef0f0;
  color: #f56c6c;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: #606266;
  margin-top: 4px;
}

.filters-section {
  background: white;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 24px;
}

.filters-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.left-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.right-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input {
  width: 300px;
}

.status-filter {
  width: 200px;
}

.advanced-filters {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e4e7ed;
}

.filters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.filter-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: #303133;
}

.filters-actions {
  display: flex;
  gap: 12px;
}

.containers-content {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.selection-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #e4e7ed;
  background: #f8f9fa;
}

.sort-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  color: #606266;
}

.containers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
  padding: 20px;
}

.containers-list {
  padding: 0;
}

.container-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.name-link {
  color: #409eff;
  cursor: pointer;
  text-decoration: none;
}

.name-link:hover {
  text-decoration: underline;
}

.status-tag {
  flex-shrink: 0;
}

.image-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.image-name {
  font-family: monospace;
  font-size: 13px;
}

.tag-badge {
  flex-shrink: 0;
}

.resource-info {
  font-size: 12px;
}

.resource-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.resource-label {
  color: #909399;
}

.resource-value {
  font-weight: 500;
}

.table-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #909399;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
  color: #c0c4cc;
}

.empty-state h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  color: #606266;
}

.empty-state p {
  margin: 0 0 24px 0;
  color: #909399;
}

.pagination-section {
  display: flex;
  justify-content: center;
  padding: 24px;
}

@media (max-width: 768px) {
  .containers-page {
    padding: 16px;
  }

  .header-content {
    flex-direction: column;
    gap: 16px;
  }

  .filters-row {
    flex-direction: column;
    align-items: stretch;
  }

  .left-controls,
  .right-controls {
    flex-wrap: wrap;
  }

  .search-input {
    width: 100%;
  }

  .containers-grid {
    grid-template-columns: 1fr;
    padding: 16px;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
