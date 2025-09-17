<template>
  <div class="security-dashboard-config">
    <el-form :model="localConfig" label-width="120px" size="small">
      <el-form-item label="Security Checks">
        <el-checkbox-group v-model="localConfig.securityChecks">
          <el-checkbox label="vulnerabilities">
            Vulnerability Scans
          </el-checkbox>
          <el-checkbox label="malware"> Malware Detection </el-checkbox>
          <el-checkbox label="compliance"> Compliance Checks </el-checkbox>
          <el-checkbox label="access"> Access Monitoring </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Severity Levels">
        <el-checkbox-group v-model="localConfig.severityLevels">
          <el-checkbox label="critical"> Critical </el-checkbox>
          <el-checkbox label="high"> High </el-checkbox>
          <el-checkbox label="medium"> Medium </el-checkbox>
          <el-checkbox label="low"> Low </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Scan Interval">
        <el-select v-model="localConfig.scanInterval" style="width: 100%">
          <el-option label="Every Hour" :value="3600000" />
          <el-option label="Every 6 Hours" :value="21600000" />
          <el-option label="Daily" :value="86400000" />
          <el-option label="Weekly" :value="604800000" />
        </el-select>
      </el-form-item>

      <el-form-item label="Auto Remediation">
        <el-switch v-model="localConfig.autoRemediation" />
      </el-form-item>

      <el-form-item label="Alert Notifications">
        <el-switch v-model="localConfig.alertNotifications" />
      </el-form-item>

      <el-form-item label="Show Risk Score">
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
