<template>
  <div class="recent-activities-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Activity Types">
        <el-checkbox-group v-model="localConfig.activityTypes">
          <el-checkbox label="container"> Container Events </el-checkbox>
          <el-checkbox label="update"> Update Events </el-checkbox>
          <el-checkbox label="system"> System Events </el-checkbox>
          <el-checkbox label="user"> User Actions </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Max Activities">
        <el-input-number
          v-model="localConfig.maxActivities"
          :min="10"
          :max="100"
          :step="10"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="Time Format">
        <el-radio-group v-model="localConfig.timeFormat">
          <el-radio label="relative"> Relative (2 hours ago) </el-radio>
          <el-radio label="absolute"> Absolute (14:30) </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Show Details">
        <el-switch v-model="localConfig.showDetails" />
      </el-form-item>

      <el-form-item label="Group by Date">
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
