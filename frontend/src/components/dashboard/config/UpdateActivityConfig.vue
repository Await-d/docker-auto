<template>
  <div class="update-activity-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="时间范围">
        <el-select v-model="localConfig.timeRange" style="width: 100%">
          <el-option label="最近24小时" value="24h" />
          <el-option label="最近7天" value="7d" />
          <el-option label="最近30天" value="30d" />
          <el-option label="全部时间" value="all" />
        </el-select>
      </el-form-item>

      <el-form-item label="显示状态">
        <el-checkbox-group v-model="localConfig.statusFilter">
          <el-checkbox label="pending"> 等待中 </el-checkbox>
          <el-checkbox label="running"> 运行中 </el-checkbox>
          <el-checkbox label="completed"> 已完成 </el-checkbox>
          <el-checkbox label="failed"> 失败 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="最大项目数">
        <el-input-number
          v-model="localConfig.maxItems"
          :min="5"
          :max="50"
          :step="5"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="自动刷新">
        <el-switch v-model="localConfig.autoRefresh" />
      </el-form-item>

      <el-form-item label="显示进度">
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
