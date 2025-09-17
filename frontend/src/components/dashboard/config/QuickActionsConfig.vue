<template>
  <div class="quick-actions-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Action Layout">
        <el-radio-group v-model="localConfig.layout">
          <el-radio label="grid"> Grid Layout </el-radio>
          <el-radio label="list"> List Layout </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Available Actions">
        <el-checkbox-group v-model="localConfig.actions">
          <el-checkbox label="start-all"> Start All Containers </el-checkbox>
          <el-checkbox label="stop-all"> Stop All Containers </el-checkbox>
          <el-checkbox label="restart-all">
            Restart All Containers
          </el-checkbox>
          <el-checkbox label="cleanup"> System Cleanup </el-checkbox>
          <el-checkbox label="backup"> Create Backup </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Confirmation">
        <el-radio-group v-model="localConfig.confirmationLevel">
          <el-radio label="none"> No Confirmation </el-radio>
          <el-radio label="critical"> Critical Actions Only </el-radio>
          <el-radio label="all"> All Actions </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Show Icons">
        <el-switch v-model="localConfig.showIcons" />
      </el-form-item>

      <el-form-item label="Show Labels">
        <el-switch v-model="localConfig.showLabels" />
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
  layout: "grid",
  actions: ["start-all", "stop-all", "restart-all"],
  confirmationLevel: "critical",
  showIcons: true,
  showLabels: true,
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
