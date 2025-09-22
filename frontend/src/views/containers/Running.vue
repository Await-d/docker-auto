<template>
  <div class="running-containers">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-content">
        <div class="title-section">
          <h1 class="page-title">
            <el-icon><SuccessFilled /></el-icon>
            运行中的容器
          </h1>
          <p class="page-subtitle">
            管理正在运行的 Docker 容器
          </p>
        </div>

        <div class="header-actions">
          <el-button @click="refreshContainers">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <!-- Statistics -->
      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-icon running">
            <el-icon><SuccessFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ runningContainers.length }}</div>
            <div class="stat-label">运行中</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon healthy">
            <el-icon><CircleCheckFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ healthyContainers.length }}</div>
            <div class="stat-label">健康状态</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon updates">
            <el-icon><Refresh /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ containersWithUpdates.length }}</div>
            <div class="stat-label">有更新</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <el-skeleton :rows="3" animated />
    </div>

    <!-- Empty State -->
    <div v-else-if="runningContainers.length === 0" class="empty-state">
      <el-empty description="暂无运行中的容器">
        <el-button type="primary" @click="$router.push('/containers')">
          查看所有容器
        </el-button>
      </el-empty>
    </div>

    <!-- Container List -->
    <div v-else class="container-grid">
      <div
        v-for="container in runningContainers"
        :key="container.id"
        class="container-card"
        @click="goToContainerDetail(container.id)"
      >
        <div class="card-header">
          <div class="container-name">{{ container.name }}</div>
          <div class="container-status">
            <el-tag type="success" size="small">
              <el-icon><SuccessFilled /></el-icon>
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
              <span class="label">运行时间:</span>
              <span class="value">{{ formatUptime(container.uptime) }}</span>
            </div>

            <div class="info-item">
              <span class="label">端口:</span>
              <span class="value">{{ formatPorts(container.ports) }}</span>
            </div>
          </div>

          <!-- Resource Usage -->
          <div v-if="container.stats" class="resource-usage">
            <div class="resource-item">
              <span class="resource-label">CPU</span>
              <el-progress
                :percentage="container.stats.cpuPercent"
                :stroke-width="4"
                :show-text="false"
              />
              <span class="resource-value">{{ container.stats.cpuPercent.toFixed(1) }}%</span>
            </div>

            <div class="resource-item">
              <span class="resource-label">内存</span>
              <el-progress
                :percentage="container.stats.memoryPercent"
                :stroke-width="4"
                :show-text="false"
                color="#e6a23c"
              />
              <span class="resource-value">{{ formatMemory(container.stats.memoryUsage) }}</span>
            </div>
          </div>
        </div>

        <div class="card-actions">
          <el-button size="small" @click.stop="stopContainer(container.id)">
            <el-icon><VideoPause /></el-icon>
            停止
          </el-button>

          <el-button size="small" @click.stop="restartContainer(container.id)">
            <el-icon><Refresh /></el-icon>
            重启
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
                <el-dropdown-item command="stats">
                  <el-icon><TrendCharts /></el-icon>
                  性能监控
                </el-dropdown-item>
                <el-dropdown-item command="update" v-if="container.hasUpdate">
                  <el-icon><Refresh /></el-icon>
                  更新镜像
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
import { computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import {
  SuccessFilled,
  CircleCheckFilled,
  Refresh,
  Picture,
  VideoPause,
  MoreFilled,
  Document,
  TrendCharts
} from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import { useContainerStore } from '@/store/containers';
import { storeToRefs } from 'pinia';

const router = useRouter();
const containerStore = useContainerStore();

const { containers, loading } = storeToRefs(containerStore);

// Computed properties
const runningContainers = computed(() =>
  containers.value.filter(c => c.status === 'running')
);

const healthyContainers = computed(() =>
  runningContainers.value.filter(c => c.health === 'healthy')
);

const containersWithUpdates = computed(() =>
  runningContainers.value.filter(c => c.hasUpdate)
);

// Methods
const refreshContainers = () => {
  containerStore.fetchContainers();
};

const goToContainerDetail = (id: string) => {
  router.push(`/containers/${id}`);
};

const stopContainer = async (id: string) => {
  try {
    await containerStore.stopContainer(id);
    ElMessage.success('容器已停止');
  } catch (error) {
    ElMessage.error('停止容器失败');
  }
};

const restartContainer = async (id: string) => {
  try {
    await containerStore.restartContainer(id);
    ElMessage.success('容器已重启');
  } catch (error) {
    ElMessage.error('重启容器失败');
  }
};

const handleContainerAction = (command: string, containerId: string) => {
  switch (command) {
    case 'logs':
      router.push(`/containers/${containerId}?tab=logs`);
      break;
    case 'stats':
      router.push(`/containers/${containerId}?tab=stats`);
      break;
    case 'update':
      // Handle container update
      break;
  }
};

const formatUptime = (uptime: number): string => {
  const days = Math.floor(uptime / (24 * 60 * 60));
  const hours = Math.floor((uptime % (24 * 60 * 60)) / (60 * 60));
  const minutes = Math.floor((uptime % (60 * 60)) / 60);

  if (days > 0) return `${days}天 ${hours}小时`;
  if (hours > 0) return `${hours}小时 ${minutes}分钟`;
  return `${minutes}分钟`;
};

const formatPorts = (ports: any[]): string => {
  if (!ports || ports.length === 0) return '无';
  return ports.map(p => `${p.hostPort}:${p.containerPort}`).join(', ');
};

const formatMemory = (bytes: number): string => {
  const mb = bytes / 1024 / 1024;
  if (mb < 1024) return `${mb.toFixed(1)}MB`;
  return `${(mb / 1024).toFixed(1)}GB`;
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
.running-containers {
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
          color: var(--el-color-success);
        }
      }

      .page-subtitle {
        color: var(--el-text-color-secondary);
        margin: 0;
      }
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

        &.running {
          background: var(--el-color-success-light-8);
          color: var(--el-color-success);
        }

        &.healthy {
          background: var(--el-color-primary-light-8);
          color: var(--el-color-primary);
        }

        &.updates {
          background: var(--el-color-warning-light-8);
          color: var(--el-color-warning);
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

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

      .container-name {
        font-size: 18px;
        font-weight: 600;
        color: var(--el-text-color-primary);
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

      .container-info {
        margin-bottom: 16px;

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
          }
        }
      }

      .resource-usage {
        .resource-item {
          display: flex;
          align-items: center;
          gap: 12px;
          margin-bottom: 8px;

          .resource-label {
            width: 40px;
            font-size: 12px;
            color: var(--el-text-color-secondary);
          }

          .el-progress {
            flex: 1;
          }

          .resource-value {
            width: 50px;
            text-align: right;
            font-size: 12px;
            color: var(--el-text-color-primary);
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
}
</style>
