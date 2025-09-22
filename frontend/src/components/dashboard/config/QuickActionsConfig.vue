<template>
  <div class="quick-actions-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="操作布局">
        <el-radio-group v-model="localConfig.layout">
          <el-radio label="grid"> 网格布局 </el-radio>
          <el-radio label="list"> 列表布局 </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="可用操作">
        <el-checkbox-group v-model="localConfig.actions">
          <el-checkbox label="start-all"> 启动所有容器 </el-checkbox>
          <el-checkbox label="stop-all"> 停止所有容器 </el-checkbox>
          <el-checkbox label="restart-all">
            重启所有容器
          </el-checkbox>
          <el-checkbox label="cleanup"> 系统清理 </el-checkbox>
          <el-checkbox label="backup"> 创建备份 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="确认设置">
        <el-radio-group v-model="localConfig.confirmationLevel">
          <el-radio label="none"> 无需确认 </el-radio>
          <el-radio label="critical"> 仅关键操作 </el-radio>
          <el-radio label="all"> 所有操作 </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="显示图标">
        <el-switch v-model="localConfig.showIcons" />
      </el-form-item>

      <el-form-item label="显示标签">
        <el-switch v-model="localConfig.showLabels" />
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";

interface Props {
  modelValue: Record<string, any>;
}

interface Emits {
  (e: "update:modelValue", value: Record<string, any>): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const localConfig = ref({
  layout: "grid",
  actions: ["start-all", "stop-all", "restart-all"],
  confirmationLevel: "critical",
  showIcons: true,
  showLabels: true,
  ...props.modelValue,
});

watch(
  localConfig,
  (newConfig) => {
    emit("update:modelValue", { ...newConfig });
  },
  { deep: true },
);
</script>
