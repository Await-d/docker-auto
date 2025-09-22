<template>
  <div class="system-overview-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="指标显示">
        <el-checkbox-group v-model="localConfig.metrics">
          <el-checkbox label="cpu"> CPU使用率 </el-checkbox>
          <el-checkbox label="memory"> 内存使用率 </el-checkbox>
          <el-checkbox label="disk"> 磁盘使用率 </el-checkbox>
          <el-checkbox label="network"> 网络活动 </el-checkbox>
          <el-checkbox label="containers"> 容器数量 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="更新间隔">
        <el-select v-model="localConfig.updateInterval" style="width: 100%">
          <el-option label="5秒" :value="5000" />
          <el-option label="10秒" :value="10000" />
          <el-option label="30秒" :value="30000" />
          <el-option label="1分钟" :value="60000" />
        </el-select>
      </el-form-item>

      <el-form-item label="警报阈值">
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
            <span class="threshold-label">内存 %</span>
          </el-col>
        </el-row>
      </el-form-item>

      <el-form-item label="显示详情">
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
