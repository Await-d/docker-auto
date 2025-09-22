<template>
  <div class="container-stats-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="图表类型">
        <el-radio-group v-model="localConfig.chartType">
          <el-radio label="pie"> 饼图 </el-radio>
          <el-radio label="donut"> 环形图 </el-radio>
          <el-radio label="bar"> 柱状图 </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="显示指标">
        <el-checkbox-group v-model="localConfig.displayMetrics">
          <el-checkbox label="running"> 运行中的容器 </el-checkbox>
          <el-checkbox label="stopped"> 已停止的容器 </el-checkbox>
          <el-checkbox label="paused"> 已暂停的容器 </el-checkbox>
          <el-checkbox label="restarting"> 重启中的容器 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="显示标签">
        <el-switch v-model="localConfig.showLabels" />
      </el-form-item>

      <el-form-item label="显示数值">
        <el-switch v-model="localConfig.showValues" />
      </el-form-item>

      <el-form-item label="启用动画">
        <el-switch v-model="localConfig.animated" />
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
  chartType: "pie",
  displayMetrics: ["running", "stopped", "paused"],
  showLabels: true,
  showValues: true,
  animated: true,
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
