<template>
  <div class="health-monitor-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Check Interval">
        <el-select v-model="localConfig.checkInterval" style="width: 100%">
          <el-option label="30 seconds" :value="30000" />
          <el-option label="1 minute" :value="60000" />
          <el-option label="2 minutes" :value="120000" />
          <el-option label="5 minutes" :value="300000" />
        </el-select>
      </el-form-item>

      <el-form-item label="Services">
        <el-checkbox-group v-model="localConfig.services">
          <el-checkbox label="docker"> Docker Engine </el-checkbox>
          <el-checkbox label="web"> Web Server </el-checkbox>
          <el-checkbox label="database"> Database </el-checkbox>
          <el-checkbox label="api"> API Service </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Alert Level">
        <el-radio-group v-model="localConfig.alertLevel">
          <el-radio label="critical"> Critical Only </el-radio>
          <el-radio label="warning"> Warning & Critical </el-radio>
          <el-radio label="all"> All Alerts </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Show History">
        <el-switch v-model="localConfig.showHistory" />
      </el-form-item>

      <el-form-item label="Auto Resolve">
        <el-switch v-model="localConfig.autoResolve" />
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
  checkInterval: 60000,
  services: ["docker", "web", "api"],
  alertLevel: "warning",
  showHistory: true,
  autoResolve: false,
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
