<template>
  <div class="container-stats-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Chart Type">
        <el-radio-group v-model="localConfig.chartType">
          <el-radio label="pie"> Pie Chart </el-radio>
          <el-radio label="donut"> Donut Chart </el-radio>
          <el-radio label="bar"> Bar Chart </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Display Metrics">
        <el-checkbox-group v-model="localConfig.displayMetrics">
          <el-checkbox label="running"> Running Containers </el-checkbox>
          <el-checkbox label="stopped"> Stopped Containers </el-checkbox>
          <el-checkbox label="paused"> Paused Containers </el-checkbox>
          <el-checkbox label="restarting"> Restarting Containers </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Show Labels">
        <el-switch v-model="localConfig.showLabels" />
      </el-form-item>

      <el-form-item label="Show Values">
        <el-switch v-model="localConfig.showValues" />
      </el-form-item>

      <el-form-item label="Animation">
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
