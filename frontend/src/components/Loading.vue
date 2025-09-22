<!--
  通用加载组件
  支持多种加载样式：转圈、点状、进度条、骨架屏、遮罩层
  可配置全屏显示、自定义文本、进度显示等功能
-->
<template>
  <div :class="loadingClass">
    <div v-if="type === 'spinner'" class="loading-spinner">
      <el-icon class="is-loading" :size="size">
        <Loading />
      </el-icon>
      <p v-if="text" class="loading-text">
        {{ text }}
      </p>
    </div>

    <div v-else-if="type === 'dots'" class="loading-dots">
      <div class="dot" />
      <div class="dot" />
      <div class="dot" />
      <p v-if="text" class="loading-text">
        {{ text }}
      </p>
    </div>

    <div v-else-if="type === 'bar'" class="loading-bar">
      <div
class="loading-bar-fill" :style="{ width: `${progress}%` }" />
      <p v-if="text" class="loading-text">
        {{ text }}
      </p>
      <p
v-if="showProgress" class="loading-progress">{{ progress }}%</p>
    </div>

    <div v-else-if="type === 'skeleton'" class="loading-skeleton">
      <div
        v-for="line in skeletonLines"
        :key="line"
        class="skeleton-line"
        :style="{ width: `${Math.random() * 40 + 60}%` }"
      />
    </div>

    <div v-else class="loading-overlay">
      <el-icon class="is-loading" :size="size">
        <Loading />
      </el-icon>
      <p v-if="text" class="loading-text">
        {{ text }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Loading } from "@element-plus/icons-vue";

interface Props {
  /** 加载类型：转圈 | 点状 | 进度条 | 骨架屏 | 遮罩层 */
  type?: "spinner" | "dots" | "bar" | "skeleton" | "overlay";
  /** 图标大小 */
  size?: number;
  /** 加载提示文本 */
  text?: string;
  /** 是否全屏显示 */
  fullscreen?: boolean;
  /** 是否显示遮罩层 */
  overlay?: boolean;
  /** 进度百分比 */
  progress?: number;
  /** 是否显示进度百分比文本 */
  showProgress?: boolean;
  /** 骨架屏行数 */
  skeletonLines?: number;
  /** 背景颜色 */
  background?: string;
}

// 设置属性默认值
const props = withDefaults(defineProps<Props>(), {
  type: "spinner", // 默认使用转圈加载
  size: 24, // 默认图标大小24px
  text: "", // 默认无提示文本
  fullscreen: false, // 默认非全屏
  overlay: false, // 默认无遮罩层
  progress: 0, // 默认进度0%
  showProgress: false, // 默认不显示进度文本
  skeletonLines: 3, // 默认骨架屏3行
  background: "", // 默认无背景色
});

// 计算加载组件的CSS类名
const loadingClass = computed(() => [
  "loading-component", // 基础样式类
  `loading-${props.type}`, // 根据类型添加对应样式类
  {
    "loading-fullscreen": props.fullscreen, // 全屏模式样式
    "loading-overlay-mode": props.overlay, // 遮罩层模式样式
  },
]);
</script>

<style scoped lang="scss">
/* 加载组件基础样式 */
.loading-component {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60px;

  /* 全屏加载模式 */
  &.loading-fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 9999;
    background: rgba(255, 255, 255, 0.9);
    backdrop-filter: blur(2px);

    .dark & {
      background: rgba(0, 0, 0, 0.9);
    }
  }

  /* 遮罩层模式 */
  &.loading-overlay-mode {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(255, 255, 255, 0.8);
    backdrop-filter: blur(1px);

    .dark & {
      background: rgba(0, 0, 0, 0.8);
    }
  }
}

/* 转圈加载样式 */
.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;

  .el-icon {
    color: var(--el-color-primary);
  }
}

/* 点状加载样式 */
.loading-dots {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--el-color-primary);
    animation: dot-flashing 1.4s infinite ease-in-out;
    display: inline-block;

    &:nth-child(1) {
      animation-delay: -0.32s;
    }
    &:nth-child(2) {
      animation-delay: -0.16s;
    }
    &:nth-child(3) {
      animation-delay: 0s;
    }
  }
}

/* 进度条加载样式 */
.loading-bar {
  width: 100%;
  max-width: 300px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;

  .loading-bar-fill {
    height: 4px;
    background: var(--el-color-primary);
    border-radius: 2px;
    transition: width 0.3s ease;
    width: 100%;
    position: relative;
    overflow: hidden;

    &::before {
      content: "";
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: linear-gradient(
        90deg,
        transparent,
        rgba(255, 255, 255, 0.4),
        transparent
      );
      animation: shimmer 1.5s infinite;
    }
  }
}

/* 骨架屏加载样式 */
.loading-skeleton {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;

  .skeleton-line {
    height: 16px;
    background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
    background-size: 200% 100%;
    animation: skeleton-loading 1.5s infinite;
    border-radius: 4px;

    .dark & {
      background: linear-gradient(90deg, #2a2a2a 25%, #404040 50%, #2a2a2a 75%);
      background-size: 200% 100%;
    }
  }
}

.loading-text {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-regular);
  text-align: center;
}

.loading-progress {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
}

/* 动画效果 */
/* 点状闪烁动画 */
@keyframes dot-flashing {
  0%,
  80%,
  100% {
    opacity: 0.3;
    transform: scale(0.8);
  }
  40% {
    opacity: 1;
    transform: scale(1);
  }
}

/* 进度条光泽动画 */
@keyframes shimmer {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(100%);
  }
}

/* 骨架屏加载动画 */
@keyframes skeleton-loading {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

/* 响应式设计 */
/* 移动端适配 */
@media (max-width: 768px) {
  .loading-component {
    min-height: 40px;

    .loading-text {
      font-size: 12px;
    }

    .loading-bar {
      max-width: 250px;
    }
  }
}
</style>
