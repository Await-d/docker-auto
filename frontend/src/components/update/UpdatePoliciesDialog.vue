<template>
  <el-dialog
    v-model="visible"
    title="更新策略"
    width="900px"
    :before-close="handleClose"
  >
    <div class="policies-manager">
      <el-alert
        title="更新策略配置"
        description="为容器组配置自动更新策略。这是一个简化版本 - 完整实现即将推出。"
        type="info"
        show-icon
        :closable="false"
      />

      <div class="policies-placeholder">
        <el-empty description="更新策略功能即将推出">
          <el-button
type="primary" @click="handleClose"> 关闭 </el-button>
        </el-empty>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from "vue";

// Props
interface Props {
  modelValue: boolean;
}

const props = defineProps<Props>();

// Emits
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  "policy-applied": [];
}>();

// Computed
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

// Methods
const handleClose = () => {
  visible.value = false;
};
</script>

<style scoped lang="scss">
.policies-manager {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.policies-placeholder {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
