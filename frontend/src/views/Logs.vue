<template>
  <div class="logs-page">
    <el-page-header @back="$router.back()" content="日志查看">
      <template #extra>
        <el-button type="primary" :icon="Refresh" @click="refreshLogs">
          刷新
        </el-button>
      </template>
    </el-page-header>

    <div class="page-content">
      <el-card>
        <template #header>
          <div class="card-header">
            <span>系统日志</span>
            <div>
              <el-select v-model="selectedLevel" placeholder="日志级别" @change="filterLogs">
                <el-option label="全部" value="" />
                <el-option label="ERROR" value="error" />
                <el-option label="WARN" value="warn" />
                <el-option label="INFO" value="info" />
                <el-option label="DEBUG" value="debug" />
              </el-select>
            </div>
          </div>
        </template>

        <div class="logs-container">
          <el-table :data="filteredLogs" v-loading="loading" empty-text="暂无日志数据" height="500">
            <el-table-column prop="timestamp" label="时间" width="180">
              <template #default="{ row }">
                {{ formatDate(row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="level" label="级别" width="100">
              <template #default="{ row }">
                <el-tag :type="getLevelType(row.level)">
                  {{ row.level.toUpperCase() }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="source" label="来源" width="150" />
            <el-table-column prop="message" label="消息" />
          </el-table>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ElMessage } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import { getLogs, type LogEntry } from "@/api/logs";

const logs = ref<LogEntry[]>([]);
const loading = ref(false);
const selectedLevel = ref("");

const filteredLogs = computed(() => {
  if (!selectedLevel.value) {
    return logs.value;
  }
  return logs.value.filter(log => log.level === selectedLevel.value);
});

const refreshLogs = async () => {
  loading.value = true;
  try {
    const response = await getLogs({
      level: selectedLevel.value || undefined,
      limit: 500,
    });
    logs.value = response.data || [];
  } catch (error) {
    console.error("Failed to fetch logs:", error);
    ElMessage.error("获取日志失败");
  } finally {
    loading.value = false;
  }
};

const filterLogs = () => {
  // Filtering is handled by computed property
};

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString();
};

const getLevelType = (level: string) => {
  switch (level.toLowerCase()) {
    case "error":
      return "danger";
    case "warn":
      return "warning";
    case "info":
      return "info";
    case "debug":
      return "info";
    default:
      return "info";
  }
};

onMounted(() => {
  refreshLogs();
});
</script>

<style scoped>
.logs-page {
  padding: 20px;
}

.page-content {
  margin-top: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logs-container {
  max-height: 600px;
  overflow-y: auto;
}
</style>