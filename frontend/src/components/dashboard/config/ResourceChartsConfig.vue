<template>
  <div class="resource-charts-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Chart Type">
        <el-radio-group v-model="localConfig.chartType">
          <el-radio label="line"> Line Chart </el-radio>
          <el-radio label="area"> Area Chart </el-radio>
          <el-radio label="bar"> Bar Chart </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Time Range">
        <el-select v-model="localConfig.timeRange" style="width: 100%">
          <el-option label="Last Hour" value="1h" />
          <el-option label="Last 6 Hours" value="6h" />
          <el-option label="Last 24 Hours" value="24h" />
          <el-option label="Last 7 Days" value="7d" />
        </el-select>
      </el-form-item>

      <el-form-item label="Resources">
        <el-checkbox-group v-model="localConfig.resources">
          <el-checkbox label="cpu"> CPU Usage </el-checkbox>
          <el-checkbox label="memory"> Memory Usage </el-checkbox>
          <el-checkbox label="disk"> Disk Usage </el-checkbox>
          <el-checkbox label="network"> Network Usage </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Data Aggregation">
        <el-radio-group v-model="localConfig.aggregation">
          <el-radio label="avg"> Average </el-radio>
          <el-radio label="max"> Maximum </el-radio>
          <el-radio label="min"> Minimum </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Show Legend">
        <el-switch v-model="localConfig.showLegend" />
      </el-form-item>

      <el-form-item label="Smooth Lines">
        <el-switch v-model="localConfig.smoothLines" />
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
  chartType: "line",
  timeRange: "24h",
  resources: ["cpu", "memory"],
  aggregation: "avg",
  showLegend: true,
  smoothLines: true,
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
