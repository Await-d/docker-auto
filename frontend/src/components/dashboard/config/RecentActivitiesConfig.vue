<template>
  <div class="recent-activities-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="活动类型">
        <el-checkbox-group v-model="localConfig.activityTypes">
          <el-checkbox label="container"> 容器事件 </el-checkbox>
          <el-checkbox label="update"> 更新事件 </el-checkbox>
          <el-checkbox label="system"> 系统事件 </el-checkbox>
          <el-checkbox label="user"> 用户操作 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="最大活动数">
        <el-input-number
          v-model="localConfig.maxActivities"
          :min="10"
          :max="100"
          :step="10"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="时间格式">
        <el-radio-group v-model="localConfig.timeFormat">
          <el-radio label="relative"> 相对时间 (2小时前) </el-radio>
          <el-radio label="absolute"> 绝对时间 (14:30) </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="显示详情">
        <el-switch v-model="localConfig.showDetails" />
      </el-form-item>

      <el-form-item label="按日期分组">
        <el-switch v-model="localConfig.groupByDate" />
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
  activityTypes: ["container", "update", "system"],
  maxActivities: 50,
  timeFormat: "relative",
  showDetails: true,
  groupByDate: false,
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
