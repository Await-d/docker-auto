<template>
  <div class="resource-charts-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="图表类型">
        <el-radio-group v-model="localConfig.chartType">
          <el-radio label="line"> 折线图 </el-radio>
          <el-radio label="area"> 面积图 </el-radio>
          <el-radio label="bar"> 柱状图 </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="时间范围">
        <el-select v-model="localConfig.timeRange" style="width: 100%">
          <el-option label="最近1小时" value="1h" />
          <el-option label="最近6小时" value="6h" />
          <el-option label="最近24小时" value="24h" />
          <el-option label="最近7天" value="7d" />
        </el-select>
      </el-form-item>

      <el-form-item label="资源类型">
        <el-checkbox-group v-model="localConfig.resources">
          <el-checkbox label="cpu"> CPU使用率 </el-checkbox>
          <el-checkbox label="memory"> 内存使用率 </el-checkbox>
          <el-checkbox label="disk"> 磁盘使用率 </el-checkbox>
          <el-checkbox label="network"> 网络使用率 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="数据聚合">
        <el-radio-group v-model="localConfig.aggregation">
          <el-radio label="avg"> 平均值 </el-radio>
          <el-radio label="max"> 最大值 </el-radio>
          <el-radio label="min"> 最小值 </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="显示图例">
        <el-switch v-model="localConfig.showLegend" />
      </el-form-item>

      <el-form-item label="平滑曲线">
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
