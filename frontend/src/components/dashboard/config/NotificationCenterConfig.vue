<template>
  <div class="notification-center-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="通知类型">
        <el-checkbox-group v-model="localConfig.notificationTypes">
          <el-checkbox label="info"> 信息 </el-checkbox>
          <el-checkbox label="success"> 成功 </el-checkbox>
          <el-checkbox label="warning"> 警告 </el-checkbox>
          <el-checkbox label="error"> 错误 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="最大通知数">
        <el-input-number
          v-model="localConfig.maxNotifications"
          :min="5"
          :max="50"
          :step="5"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="自动关闭">
        <el-switch v-model="localConfig.autoDismiss" />
      </el-form-item>

      <el-form-item
v-if="localConfig.autoDismiss" label="关闭超时"
>
        <el-input-number
          v-model="localConfig.dismissTimeout"
          :min="1"
          :max="30"
          :step="1"
          style="width: 100%"
        />
        <span class="timeout-label">秒</span>
      </el-form-item>

      <el-form-item label="显示时间戳">
        <el-switch v-model="localConfig.showTimestamps" />
      </el-form-item>

      <el-form-item label="分组相似">
        <el-switch v-model="localConfig.groupSimilar" />
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
  notificationTypes: ["info", "success", "warning", "error"],
  maxNotifications: 20,
  autoDismiss: true,
  dismissTimeout: 5,
  showTimestamps: true,
  groupSimilar: false,
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
.notification-center-config {
  .timeout-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-left: 8px;
  }
}
</style>
