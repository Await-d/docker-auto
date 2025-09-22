<template>
  <el-dialog
    v-model="visible"
    title="版本比较"
    width="800px"
    :before-close="handleClose"
  >
    <div class="version-comparison">
      <el-alert
        title="版本比较工具"
        description="比较容器版本，查看更改日志，并分析差异。这是一个简化版本 - 完整实现即将推出。"
        type="info"
        show-icon
        :closable="false"
      />

      <div v-if="comparisonData" class="comparison-data">
        <div class="comparison-header">
          <h3>{{ comparisonData.containerName }}</h3>
          <div class="version-badges">
            <el-tag type="info">
              {{ comparisonData.fromVersion }}
            </el-tag>
            <el-icon><Right /></el-icon>
            <el-tag type="primary">
              {{ comparisonData.toVersion }}
            </el-tag>
          </div>
        </div>

        <div class="comparison-placeholder">
          <el-empty description="详细比较视图即将推出">
            <p>容器: {{ comparisonData.containerName }}</p>
            <p>源版本: {{ comparisonData.fromVersion }}</p>
            <p>目标版本: {{ comparisonData.toVersion }}</p>
          </el-empty>
        </div>
      </div>

      <div v-else class="no-data">
        <el-empty description="没有可用的比较数据" />
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose"> 关闭 </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Right } from "@element-plus/icons-vue";
import type { VersionComparison } from "@/types/updates";

// Props
interface Props {
  modelValue: boolean;
  comparisonData?: VersionComparison | null;
}

const props = defineProps<Props>();

// Emits
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
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
.version-comparison {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.comparison-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-bg-color-page);
  border-radius: 8px;

  h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
  }

  .version-badges {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

.comparison-placeholder,
.no-data {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style>
