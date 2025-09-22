<template>
  <div class="realtime-monitor-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="更新间隔">
        <el-select v-model="localConfig.updateInterval" style="width: 100%">
          <el-option label="1秒" :value="1000" />
          <el-option label="2秒" :value="2000" />
          <el-option label="5秒" :value="5000" />
          <el-option label="10秒" :value="10000" />
        </el-select>
      </el-form-item>

      <el-form-item label="图表类型">
        <el-radio-group v-model="localConfig.chartType">
          <el-radio label="line"> 折线图 </el-radio>
          <el-radio label="area"> 面积图 </el-radio>
          <el-radio label="bar"> 柱状图 </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="监控指标">
        <el-checkbox-group v-model="localConfig.metrics">
          <el-checkbox label="cpu"> CPU使用率 </el-checkbox>
          <el-checkbox label="memory"> 内存使用率 </el-checkbox>
          <el-checkbox label="network"> 网络I/O </el-checkbox>
          <el-checkbox label="disk"> 磁盘I/O </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="数据点数">
        <el-input-number
          v-model="localConfig.dataPoints"
          :min="10"
          :max="100"
          :step="10"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="显示网格">
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
