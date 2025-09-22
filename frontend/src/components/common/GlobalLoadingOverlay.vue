<template>
  <Teleport to="body">
    <Transition name="loading-fade">
      <div v-if="isVisible" class="global-loading-overlay">
        <div class="loading-content">
          <div class="loading-spinner">
            <el-icon class="loading-icon" :size="40">
              <Loading />
            </el-icon>
          </div>
          <div class="loading-text">
            {{ message }}
          </div>
          <div v-if="progress >= 0" class="loading-progress">
            <el-progress
              :percentage="progress"
              :stroke-width="4"
              :show-text="false"
            />
          </div>
          <div v-if="canCancel" class="loading-actions">
            <el-button
              size="small"
              type="text"
              @click="handleCancel"
            >
              取消
            </el-button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Loading } from "@element-plus/icons-vue";

interface Props {
  visible?: boolean;
  message?: string;
  progress?: number;
  canCancel?: boolean;
}

interface Emits {
  (e: "cancel"): void;
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  message: "加载中...",
  progress: -1,
  canCancel: false,
});

const emit = defineEmits<Emits>();

const isVisible = computed(() => props.visible);

const handleCancel = () => {
  emit("cancel");
};
</script>

<style scoped lang="scss">
.global-loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.loading-content {
  background: white;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  text-align: center;
  min-width: 280px;
  max-width: 400px;
}

.loading-spinner {
  margin-bottom: 16px;
}

.loading-icon {
  animation: rotate 1.5s linear infinite;
  color: var(--el-color-primary);
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.loading-text {
  font-size: 16px;
  color: var(--el-text-color-primary);
  margin-bottom: 16px;
  font-weight: 500;
}

.loading-progress {
  margin-bottom: 16px;
}

.loading-actions {
  margin-top: 8px;
}

.loading-fade-enter-active,
.loading-fade-leave-active {
  transition: opacity 0.3s ease;
}

.loading-fade-enter-from,
.loading-fade-leave-to {
  opacity: 0;
}
</style>