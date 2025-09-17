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
  type?: "spinner" | "dots" | "bar" | "skeleton" | "overlay";
  size?: number;
  text?: string;
  fullscreen?: boolean;
  overlay?: boolean;
  progress?: number;
  showProgress?: boolean;
  skeletonLines?: number;
  background?: string;
}

const props = withDefaults(defineProps<Props>(), {
  type: "spinner",
  size: 24,
  text: "",
  fullscreen: false,
  overlay: false,
  progress: 0,
  showProgress: false,
  skeletonLines: 3,
  background: "",
});

const loadingClass = computed(() => [
  "loading-component",
  `loading-${props.type}`,
  {
    "loading-fullscreen": props.fullscreen,
    "loading-overlay-mode": props.overlay,
  },
]);
</script>

<style scoped lang="scss">
.loading-component {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60px;

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

.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;

  .el-icon {
    color: var(--el-color-primary);
  }
}

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

// Animations
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

@keyframes shimmer {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(100%);
  }
}

@keyframes skeleton-loading {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

// Responsive design
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
