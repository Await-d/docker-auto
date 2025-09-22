<template>
  <div class="security-dashboard-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="安全检查">
        <el-checkbox-group v-model="localConfig.securityChecks">
          <el-checkbox label="vulnerabilities">
            漏洞扫描
          </el-checkbox>
          <el-checkbox label="malware"> 恶意软件检测 </el-checkbox>
          <el-checkbox label="compliance"> 合规性检查 </el-checkbox>
          <el-checkbox label="access"> 访问监控 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="严重程度">
        <el-checkbox-group v-model="localConfig.severityLevels">
          <el-checkbox label="critical"> 严重 </el-checkbox>
          <el-checkbox label="high"> 高 </el-checkbox>
          <el-checkbox label="medium"> 中等 </el-checkbox>
          <el-checkbox label="low"> 低 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="扫描间隔">
        <el-select v-model="localConfig.scanInterval" style="width: 100%">
          <el-option label="每小时" :value="3600000" />
          <el-option label="每6小时" :value="21600000" />
          <el-option label="每日" :value="86400000" />
          <el-option label="每周" :value="604800000" />
        </el-select>
      </el-form-item>

      <el-form-item label="自动修复">
        <el-switch v-model="localConfig.autoRemediation" />
      </el-form-item>

      <el-form-item label="警报通知">
        <el-switch v-model="localConfig.alertNotifications" />
      </el-form-item>

      <el-form-item label="显示风险评分">
        <el-switch v-model="localConfig.showRiskScore" />
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
  securityChecks: ["vulnerabilities", "compliance"],
  severityLevels: ["critical", "high", "medium"],
  scanInterval: 86400000, // Daily
  autoRemediation: false,
  alertNotifications: true,
  showRiskScore: true,
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
