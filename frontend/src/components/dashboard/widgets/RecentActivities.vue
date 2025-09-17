<template>
  <div class="recent-activities-widget">
    <div class="widget-placeholder">
      <el-icon :size="48">
        <Document />
      </el-icon>
      <h3>Recent Activities</h3>
      <p>Timeline of recent system activities will be displayed here.</p>
      <div class="mock-data">
        <div class="activity-item">
          <span class="activity-time">2m ago</span>
          <span class="activity-desc">Container started: web-server</span>
        </div>
        <div class="activity-item">
          <span class="activity-time">5m ago</span>
          <span class="activity-desc">Update completed: nginx</span>
        </div>
        <div class="activity-item">
          <span class="activity-time">10m ago</span>
          <span class="activity-desc">Backup created</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Document } from "@element-plus/icons-vue";

defineProps<{
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
}>();
</script>

<style scoped lang="scss">
.recent-activities-widget {
  padding: 16px;
  height: 100%;
}

.widget-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  color: var(--el-text-color-secondary);

  h3 {
    margin: 12px 0 8px 0;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 0 0 16px 0;
    font-size: 14px;
  }

  .mock-data {
    width: 100%;
    max-width: 250px;

    .activity-item {
      display: flex;
      gap: 8px;
      padding: 4px 0;
      font-size: 11px;
      text-align: left;

      .activity-time {
        color: var(--el-text-color-placeholder);
        min-width: 40px;
      }

      .activity-desc {
        color: var(--el-text-color-primary);
      }
    }
  }
}
</style>
