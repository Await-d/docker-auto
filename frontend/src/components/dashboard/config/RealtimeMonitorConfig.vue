<template>
  <div class="realtime-monitor-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Update Interval">
        <el-select v-model="localConfig.updateInterval" style="width: 100%">
          <el-option label="1 second" :value="1000" />
          <el-option label="2 seconds" :value="2000" />
          <el-option label="5 seconds" :value="5000" />
          <el-option label="10 seconds" :value="10000" />
        </el-select>
      </el-form-item>

      <el-form-item label="Chart Type">
        <el-radio-group v-model="localConfig.chartType">
          <el-radio label="line"> Line Chart </el-radio>
          <el-radio label="area"> Area Chart </el-radio>
          <el-radio label="bar"> Bar Chart </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Metrics">
        <el-checkbox-group v-model="localConfig.metrics">
          <el-checkbox label="cpu"> CPU Usage </el-checkbox>
          <el-checkbox label="memory"> Memory Usage </el-checkbox>
          <el-checkbox label="network"> Network I/O </el-checkbox>
          <el-checkbox label="disk"> Disk I/O </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Data Points">
        <el-input-number
          v-model="localConfig.dataPoints"
          :min="10"
          :max="100"
          :step="10"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="Show Grid">
        <el-switch v-model="localConfig.showGrid" />
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
  updateInterval: 5000,
  chartType: "line",
  metrics: ["cpu", "memory"],
  dataPoints: 50,
  showGrid: true,
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
