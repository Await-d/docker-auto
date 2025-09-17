<template>
  <div class="update-activity-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Time Range">
        <el-select v-model="localConfig.timeRange" style="width: 100%">
          <el-option label="Last 24 hours" value="24h" />
          <el-option label="Last 7 days" value="7d" />
          <el-option label="Last 30 days" value="30d" />
          <el-option label="All time" value="all" />
        </el-select>
      </el-form-item>

      <el-form-item label="Show Status">
        <el-checkbox-group v-model="localConfig.statusFilter">
          <el-checkbox label="pending"> Pending </el-checkbox>
          <el-checkbox label="running"> Running </el-checkbox>
          <el-checkbox label="completed"> Completed </el-checkbox>
          <el-checkbox label="failed"> Failed </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Max Items">
        <el-input-number
          v-model="localConfig.maxItems"
          :min="5"
          :max="50"
          :step="5"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="Auto Refresh">
        <el-switch v-model="localConfig.autoRefresh" />
      </el-form-item>

      <el-form-item label="Show Progress">
        <el-switch v-model="localConfig.showProgress" />
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
  timeRange: "24h",
  statusFilter: ["pending", "running", "completed", "failed"],
  maxItems: 20,
  autoRefresh: true,
  showProgress: true,
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
