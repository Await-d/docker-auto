<template>
  <div class="update-activity-widget">
    <div class="widget-placeholder">
      <el-icon :size="48"><Refresh /></el-icon>
      <h3>Update Activity</h3>
      <p>Recent updates and update statistics will be displayed here.</p>
      <div class="mock-data">
        <div class="update-item">
          <span class="update-name">nginx:latest</span>
          <span class="update-status success">Updated</span>
        </div>
        <div class="update-item">
          <span class="update-name">postgres:14</span>
          <span class="update-status pending">Pending</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Refresh } from '@element-plus/icons-vue'

defineProps<{
  widgetId: string
  widgetConfig: any
  widgetData?: any
}>()
</script>

<style scoped lang="scss">
.update-activity-widget {
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
    max-width: 200px;

    .update-item {
      display: flex;
      justify-content: space-between;
      padding: 4px 0;
      font-size: 12px;

      .update-status {
        &.success { color: var(--el-color-success); }
        &.pending { color: var(--el-color-warning); }
      }
    }
  }
}
</style>