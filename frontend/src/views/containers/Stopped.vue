<template>
  <div class="stopped-containers">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="title-section">
          <h1 class="page-title">
            <el-icon><Warning /></el-icon>
            已停止的容器
          </h1>
          <p class="page-subtitle">
            管理已停止的 Docker 容器
          </p>
        </div>

        <div class="header-actions">
          <el-button @click="refreshContainers">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>

          <el-button type="danger" :disabled="selectedContainers.size === 0" @click="handleBulkDelete">
            <el-icon><Delete /></el-icon>
            批量删除
          </el-button>
        </div>
      </div>

      <!-- Statistics -->
      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-icon stopped">
            <el-icon><Warning /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stoppedContainers.length }}</div>
            <div class="stat-label">已停止</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon exited">
            <el-icon><CircleCloseFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ exitedContainers.length }}</div>
            <div class="stat-label">异常退出</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon selected">
            <el-icon><Select /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ selectedContainers.size }}</div>
            <div class="stat-label">已选择</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Bulk Actions -->
    <div v-if="selectedContainers.size > 0" class="bulk-actions">
      <div class="selected-info">
        已选择 {{ selectedContainers.size }} 个容器
      </div>
      <div class="bulk-buttons">
        <el-button type="primary" @click="handleBulkStart">
          <el-icon><VideoPlay /></el-icon>
          批量启动
        </el-button>
        <el-button type="danger" @click="handleBulkDelete">
          <el-icon><Delete /></el-icon>
          批量删除
        </el-button>
        <el-button @click="clearSelection">
          取消选择
        </el-button>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <el-skeleton :rows="3" animated />
    </div>

    <!-- Empty State -->
    <div v-else-if="stoppedContainers.length === 0" class="empty-state">
      <el-empty description="暂无已停止的容器">
        <el-button type="primary" @click="$router.push('/containers')">
          查看所有容器
        </el-button>
      </el-empty>
    </div>

    <!-- Container List -->
    <div v-else class="container-grid">
      <div
        v-for="container in stoppedContainers"
        :key="container.id"
        class="container-card"
        :class="{ selected: selectedContainers.has(container.id) }"
        @click="toggleSelection(container.id)"
      >
        <div class="card-header">
          <div class="selection-area">
            <el-checkbox
              :model-value="selectedContainers.has(container.id)"
              @change="toggleSelection(container.id)"
              @click.stop
            />
            <div class="container-name">{{ container.name }}</div>
          </div>
          <div class="container-status">
            <el-tag type="danger" size="small">
              <el-icon><Warning /></el-icon>
              {{ container.status }}
            </el-tag>
          </div>
        </div>

        <div class="card-content">
          <div class="container-image">
            <el-icon><Picture /></el-icon>
            {{ container.image }}
          </div>

          <div class="container-info">
            <div class="info-item">
              <span class="label">停止时间:</span>
              <span class="value">{{ formatStopTime(container.state.finishedAt) }}</span>
            </div>

            <div class="info-item">
              <span class="label">退出码:</span>
              <span class="value" :class="{ error: container.state.exitCode !== 0 }">
                {{ container.state.exitCode || 0 }}
              </span>
            </div>

            <div v-if="container.state.error" class="info-item">
              <span class="label">退出原因:</span>
              <span class="value">{{ container.state.error }}</span>
            </div>
          </div>

          <!-- Last run info -->
          <div class="last-run-info">
            <div class="info-item">
              <span class="label">最后运行:</span>
              <span class="value">{{ formatRunDuration(container.state.startedAt, container.state.finishedAt) }}</span>
            </div>
          </div>
        </div>

        <div class="card-actions">
          <el-button type="primary" size="small" @click.stop="startContainer(container.id)">
            <el-icon><VideoPlay /></el-icon>
            启动
          </el-button>

          <el-button size="small" @click.stop="goToContainerDetail(container.id)">
            <el-icon><View /></el-icon>
            详情
          </el-button>

          <el-dropdown @command="(cmd) => handleContainerAction(cmd, container.id)" @click.stop>
            <el-button size="small">
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logs">
                  <el-icon><Document /></el-icon>
                  查看日志
                </el-dropdown-item>
                <el-dropdown-item command="inspect">
                  <el-icon><Search /></el-icon>
                  详细信息
                </el-dropdown-item>
                <el-dropdown-item command="delete" divided>
                  <el-icon><Delete /></el-icon>
                  删除容器
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import {
  Warning,
  CircleCloseFilled,
  Select,
  Refresh,
  Delete,
  VideoPlay,
  Picture,
  View,
  MoreFilled,
  Document,
  Search
} from '@element-plus/icons-vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { useContainerStore } from '@/store/containers';
import { storeToRefs } from 'pinia';

const router = useRouter();
const containerStore = useContainerStore();

const { containers, loading } = storeToRefs(containerStore);
const selectedContainers = ref<Set<string>>(new Set());

// Computed properties
const stoppedContainers = computed(() =>
  containers.value.filter(c => c.status === 'exited')
);

const exitedContainers = computed(() =>
  stoppedContainers.value.filter(c => c.state.exitCode !== 0)
);

// Methods
const refreshContainers = () => {
  containerStore.fetchContainers();
};

const toggleSelection = (id: string) => {
  if (selectedContainers.value.has(id)) {
    selectedContainers.value.delete(id);
  } else {
    selectedContainers.value.add(id);
  }
};

const clearSelection = () => {
  selectedContainers.value.clear();
};

const goToContainerDetail = (id: string) => {
  router.push(`/containers/${id}`);
};

const startContainer = async (id: string) => {
  try {
    await containerStore.startContainer(id);
    ElMessage.success('容器已启动');
  } catch (error) {
    ElMessage.error('启动容器失败');
  }
};

const deleteContainer = async (id: string) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除这个容器吗？此操作不可恢复。',
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    await containerStore.deleteContainer(id);
    ElMessage.success('容器已删除');
    selectedContainers.value.delete(id);
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除容器失败');
    }
  }
};

const handleBulkStart = async () => {
  try {
    const containerIds = Array.from(selectedContainers.value);
    await Promise.all(containerIds.map(id => containerStore.startContainer(id)));
    ElMessage.success(`成功启动 ${containerIds.length} 个容器`);
    clearSelection();
  } catch (error) {
    ElMessage.error('批量启动失败');
  }
};

const handleBulkDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedContainers.value.size} 个容器吗？此操作不可恢复。`,
      '确认批量删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    const containerIds = Array.from(selectedContainers.value);
    await Promise.all(containerIds.map(id => containerStore.deleteContainer(id)));
    ElMessage.success(`成功删除 ${containerIds.length} 个容器`);
    clearSelection();
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败');
    }
  }
};

const handleContainerAction = (command: string, containerId: string) => {
  switch (command) {
    case 'logs':
      router.push(`/containers/${containerId}?tab=logs`);
      break;
    case 'inspect':
      router.push(`/containers/${containerId}?tab=inspect`);
      break;
    case 'delete':
      deleteContainer(containerId);
      break;
  }
};

const formatStopTime = (finishedAt: string): string => {
  if (!finishedAt) return '未知';
  const date = new Date(finishedAt);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  const minutes = Math.floor(diff / (1000 * 60));
  const hours = Math.floor(diff / (1000 * 60 * 60));
  const days = Math.floor(diff / (1000 * 60 * 60 * 24));

  if (days > 0) return `${days}天前`;
  if (hours > 0) return `${hours}小时前`;
  if (minutes > 0) return `${minutes}分钟前`;
  return '刚刚';
};

const formatRunDuration = (startedAt: string, finishedAt: string): string => {
  if (!startedAt || !finishedAt) return '未知';

  const start = new Date(startedAt);
  const end = new Date(finishedAt);
  const duration = end.getTime() - start.getTime();

  const days = Math.floor(duration / (1000 * 60 * 60 * 24));
  const hours = Math.floor((duration % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const minutes = Math.floor((duration % (1000 * 60 * 60)) / (1000 * 60));

  if (days > 0) return `运行了 ${days}天 ${hours}小时`;
  if (hours > 0) return `运行了 ${hours}小时 ${minutes}分钟`;
  return `运行了 ${minutes}分钟`;
};

// Lifecycle
onMounted(() => {
  containerStore.fetchContainers();

  // Auto refresh every 30 seconds
  const interval = setInterval(() => {
    containerStore.fetchContainers();
  }, 30000);

  onUnmounted(() => {
    clearInterval(interval);
  });
});
</script>

<style scoped lang="scss">
.stopped-containers {
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;

  .header-content {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 20px;

    .title-section {
      .page-title {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 24px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        margin: 0 0 8px 0;

        .el-icon {
          color: var(--el-color-warning);
        }
      }

      .page-subtitle {
        color: var(--el-text-color-secondary);
        margin: 0;
      }
    }

    .header-actions {
      display: flex;
      gap: 12px;
    }
  }

  .stats-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;

    .stat-card {
      display: flex;
      align-items: center;
      padding: 20px;
      background: var(--el-bg-color);
      border-radius: 8px;
      border: 1px solid var(--el-border-color-light);

      .stat-icon {
        width: 48px;
        height: 48px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-right: 16px;

        &.stopped {
          background: var(--el-color-warning-light-8);
          color: var(--el-color-warning);
        }

        &.exited {
          background: var(--el-color-danger-light-8);
          color: var(--el-color-danger);
        }

        &.selected {
          background: var(--el-color-primary-light-8);
          color: var(--el-color-primary);
        }

        .el-icon {
          font-size: 24px;
        }
      }

      .stat-content {
        .stat-value {
          font-size: 28px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          line-height: 1;
        }

        .stat-label {
          font-size: 14px;
          color: var(--el-text-color-secondary);
          margin-top: 4px;
        }
      }
    }
  }
}

.bulk-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: 8px;
  margin-bottom: 20px;

  .selected-info {
    color: var(--el-color-primary);
    font-weight: 500;
  }

  .bulk-buttons {
    display: flex;
    gap: 8px;
  }
}

.loading-container {
  padding: 20px;
}

.empty-state {
  padding: 60px 20px;
  text-align: center;
}

.container-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 20px;

  .container-card {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    padding: 20px;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      border-color: var(--el-color-primary);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      transform: translateY(-2px);
    }

    &.selected {
      border-color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

      .selection-area {
        display: flex;
        align-items: center;
        gap: 12px;

        .container-name {
          font-size: 18px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }
      }
    }

    .card-content {
      .container-image {
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--el-text-color-secondary);
        font-size: 14px;
        margin-bottom: 12px;
      }

      .container-info,
      .last-run-info {
        margin-bottom: 12px;

        .info-item {
          display: flex;
          justify-content: space-between;
          margin-bottom: 4px;
          font-size: 14px;

          .label {
            color: var(--el-text-color-secondary);
          }

          .value {
            color: var(--el-text-color-primary);

            &.error {
              color: var(--el-color-danger);
              font-weight: 500;
            }
          }
        }
      }
    }

    .card-actions {
      display: flex;
      gap: 8px;
      padding-top: 16px;
      border-top: 1px solid var(--el-border-color-lighter);
    }
  }
}

@media (max-width: 768px) {
  .container-grid {
    grid-template-columns: 1fr;
  }

  .stats-row {
    grid-template-columns: 1fr;
  }

  .bulk-actions {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
}
</style>
