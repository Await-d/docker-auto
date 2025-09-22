<template>
  <div class="recent-activities-widget">
    <div class="widget-placeholder">
      <el-icon :size="48">
        <Document />
      </el-icon>
      <h3>最近活动</h3>
      <p>最近系统活动的时间线将在此处显示。</p>
      <div class="mock-data">
        <div class="activity-item">
          <span class="activity-time">2分钟前</span>
          <span class="activity-desc">容器已启动: web-server</span>
        </div>
        <div class="activity-item">
          <span class="activity-time">5分钟前</span>
          <span class="activity-desc">更新已完成: nginx</span>
        </div>
        <div class="activity-item">
          <span class="activity-time">10分钟前</span>
          <span class="activity-desc">已创建备份</span>
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
