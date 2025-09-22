<template>
  <div class="security-dashboard-widget">
    <div class="widget-placeholder">
      <el-icon :size="48">
        <Lock />
      </el-icon>
      <h3>安全仪表盘</h3>
      <p>
        安全状态和漏洞监控将在此处显示。
      </p>
      <div class="mock-data">
        <div class="security-metric">
          <span class="metric-label">安全评分</span>
          <span class="metric-value good">85%</span>
        </div>
        <div class="security-metric">
          <span class="metric-label">漏洞</span>
          <span class="metric-value warning">3个中等</span>
        </div>
        <div class="security-metric">
          <span class="metric-label">最后扫描</span>
          <span class="metric-value">2小时前</span>
        </div>
        <div class="security-status">
          <div class="status-item">
            <el-icon><SuccessFilled /></el-icon>
            <span>SSL证书有效</span>
          </div>
          <div class="status-item">
            <el-icon><Warning /></el-icon>
            <span>依赖项过时</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Lock, SuccessFilled, Warning } from "@element-plus/icons-vue";

defineProps<{
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
}>();
</script>

<style scoped lang="scss">
.security-dashboard-widget {
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

    .security-metric {
      display: flex;
      justify-content: space-between;
      padding: 4px 0;
      font-size: 12px;
      margin-bottom: 4px;

      .metric-label {
        color: var(--el-text-color-secondary);
      }

      .metric-value {
        font-weight: 600;

        &.good {
          color: var(--el-color-success);
        }
        &.warning {
          color: var(--el-color-warning);
        }
        &.danger {
          color: var(--el-color-danger);
        }
      }
    }

    .security-status {
      margin-top: 12px;
      text-align: left;

      .status-item {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 0;
        font-size: 11px;

        .el-icon {
          font-size: 12px;
        }

        &:first-child .el-icon {
          color: var(--el-color-success);
        }

        &:last-child .el-icon {
          color: var(--el-color-warning);
        }
      }
    }
  }
}
</style>
