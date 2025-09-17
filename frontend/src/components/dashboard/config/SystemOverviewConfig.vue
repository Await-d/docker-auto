<template>
  <div class="system-overview-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Metrics Display">
        <el-checkbox-group v-model="localConfig.metrics">
          <el-checkbox label="cpu"> CPU Usage </el-checkbox>
          <el-checkbox label="memory"> Memory Usage </el-checkbox>
          <el-checkbox label="disk"> Disk Usage </el-checkbox>
          <el-checkbox label="network"> Network Activity </el-checkbox>
          <el-checkbox label="containers"> Container Count </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Update Interval">
        <el-select v-model="localConfig.updateInterval" style="width: 100%">
          <el-option label="5 seconds" :value="5000" />
          <el-option label="10 seconds" :value="10000" />
          <el-option label="30 seconds" :value="30000" />
          <el-option label="1 minute" :value="60000" />
        </el-select>
      </el-form-item>

      <el-form-item label="Alert Thresholds">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-input-number
              v-model="localConfig.thresholds.cpu"
              :min="0"
              :max="100"
              :precision="0"
              style="width: 100%"
            />
            <span class="threshold-label">CPU %</span>
          </el-col>
          <el-col :span="12">
            <el-input-number
              v-model="localConfig.thresholds.memory"
              :min="0"
              :max="100"
              :precision="0"
              style="width: 100%"
            />
            <span class="threshold-label">Memory %</span>
          </el-col>
        </el-row>
      </el-form-item>

      <el-form-item label="Show Details">
        <el-switch v-model="localConfig.showDetails" />
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
  metrics: ["cpu", "memory", "disk", "containers"],
  updateInterval: 30000,
  thresholds: {
    cpu: 80,
    memory: 85,
  },
  showDetails: true,
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

<style scoped lang="scss">
.system-overview-config {
  .threshold-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-top: 4px;
    display: block;
    text-align: center;
  }
}
</style>
