<template>
  <div class="health-monitor-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="检查间隔">
        <el-select v-model="localConfig.checkInterval" style="width: 100%">
          <el-option label="30秒" :value="30000" />
          <el-option label="1分钟" :value="60000" />
          <el-option label="2分钟" :value="120000" />
          <el-option label="5分钟" :value="300000" />
        </el-select>
      </el-form-item>

      <el-form-item label="服务">
        <el-checkbox-group v-model="localConfig.services">
          <el-checkbox label="docker"> Docker引擎 </el-checkbox>
          <el-checkbox label="web"> Web服务器 </el-checkbox>
          <el-checkbox label="database"> 数据库 </el-checkbox>
          <el-checkbox label="api"> API服务 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="警报级别">
        <el-radio-group v-model="localConfig.alertLevel">
          <el-radio label="critical"> 仅严重 </el-radio>
          <el-radio label="warning"> 警告和严重 </el-radio>
          <el-radio label="all"> 所有警报 </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="显示历史">
        <el-switch v-model="localConfig.showHistory" />
      </el-form-item>

      <el-form-item label="自动解决">
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
