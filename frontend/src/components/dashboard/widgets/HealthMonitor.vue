<template>
  <div class="health-monitor-widget">
    <div class="widget-placeholder">
      <el-icon :size="48">
        <CircleCheckFilled />
      </el-icon>
      <h3>健康监控</h3>
      <p>服务健康状态和可用性指标将在此处显示。</p>
      <div class="mock-data">
        <div class="health-item">
          <span class="service-name">Web服务器</span>
          <span class="health-status healthy">健康</span>
        </div>
        <div class="health-item">
          <span class="service-name">数据库</span>
          <span class="health-status healthy">健康</span>
        </div>
        <div class="health-item">
          <span class="service-name">缓存</span>
          <span class="health-status warning">警告</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CircleCheckFilled } from "@element-plus/icons-vue";

defineProps<{
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
}>();
</script>

<style scoped lang="scss">
.health-monitor-widget {
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

    .health-item {
      display: flex;
      justify-content: space-between;
      padding: 4px 0;
      font-size: 12px;

      .health-status {
        &.healthy {
          color: var(--el-color-success);
        }
        &.warning {
          color: var(--el-color-warning);
        }
        &.error {
          color: var(--el-color-danger);
        }
      }
    }
  }
}
</style>
