<template>
  <div class="resource-charts-widget">
    <div class="widget-placeholder">
      <el-icon :size="48"><DataAnalysis /></el-icon>
      <h3>Resource Charts</h3>
      <p>Historical resource usage charts will be displayed here.</p>
      <div class="mock-chart">
        <div class="chart-legend">
          <div class="legend-item cpu">
            <div class="legend-color"></div>
            <span>CPU Usage</span>
          </div>
          <div class="legend-item memory">
            <div class="legend-color"></div>
            <span>Memory Usage</span>
          </div>
          <div class="legend-item disk">
            <div class="legend-color"></div>
            <span>Disk I/O</span>
          </div>
        </div>
        <div class="chart-area">
          <svg width="100%" height="80" viewBox="0 0 200 80">
            <polyline
              points="0,60 40,45 80,55 120,40 160,50 200,35"
              fill="none"
              stroke="var(--el-color-primary)"
              stroke-width="2"
            />
            <polyline
              points="0,65 40,55 80,60 120,50 160,45 200,40"
              fill="none"
              stroke="var(--el-color-success)"
              stroke-width="2"
            />
            <polyline
              points="0,70 40,68 80,65 120,70 160,60 200,55"
              fill="none"
              stroke="var(--el-color-warning)"
              stroke-width="2"
            />
          </svg>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { DataAnalysis } from '@element-plus/icons-vue'

defineProps<{
  widgetId: string
  widgetConfig: any
  widgetData?: any
}>()
</script>

<style scoped lang="scss">
.resource-charts-widget {
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

  .mock-chart {
    width: 100%;
    max-width: 250px;

    .chart-legend {
      display: flex;
      justify-content: center;
      gap: 12px;
      margin-bottom: 12px;
      font-size: 11px;

      .legend-item {
        display: flex;
        align-items: center;
        gap: 4px;

        .legend-color {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }

        &.cpu .legend-color {
          background: var(--el-color-primary);
        }

        &.memory .legend-color {
          background: var(--el-color-success);
        }

        &.disk .legend-color {
          background: var(--el-color-warning);
        }
      }
    }

    .chart-area {
      background: var(--el-fill-color-extra-light);
      border-radius: 4px;
      padding: 8px;
    }
  }
}
</style>