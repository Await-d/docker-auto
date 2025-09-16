<template>
  <div class="notification-center-widget">
    <div class="widget-placeholder">
      <el-icon :size="48"><Bell /></el-icon>
      <h3>Notification Center</h3>
      <p>Live notifications and alerts will be displayed here.</p>
      <div class="mock-data">
        <div class="notification-item warning">
          <el-icon><Warning /></el-icon>
          <span>High CPU usage detected</span>
        </div>
        <div class="notification-item info">
          <el-icon><InfoFilled /></el-icon>
          <span>Backup completed successfully</span>
        </div>
        <div class="notification-item success">
          <el-icon><SuccessFilled /></el-icon>
          <span>All systems operational</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bell, Warning, InfoFilled, SuccessFilled } from '@element-plus/icons-vue'

defineProps<{
  widgetId: string
  widgetConfig: any
  widgetData?: any
}>()
</script>

<style scoped lang="scss">
.notification-center-widget {
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

    .notification-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 8px;
      margin: 4px 0;
      border-radius: 4px;
      font-size: 12px;
      text-align: left;

      &.warning {
        background: rgba(var(--el-color-warning-rgb), 0.1);
        color: var(--el-color-warning);
      }

      &.info {
        background: rgba(var(--el-color-info-rgb), 0.1);
        color: var(--el-color-info);
      }

      &.success {
        background: rgba(var(--el-color-success-rgb), 0.1);
        color: var(--el-color-success);
      }
    }
  }
}
</style>