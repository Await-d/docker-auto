<template>
  <div class="security-config">
    <ConfigForm
      v-model="formData"
      :rules="formRules"
      :saving="loading"
      :has-changes="hasChanges"
      :testable="false"
      @save="handleSave"
      @reset="handleReset"
      @field-change="handleFieldChange"
    >
      <!-- Access Control -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Lock /></el-icon>
            <span>Access Control</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="IP Whitelist">
              <KeyValueEditor
                v-model="ipWhitelistObject"
                title="IP Whitelist"
                item-name="IP Range"
                key-label="Name"
                value-label="IP/CIDR"
                key-placeholder="Office Network"
                value-placeholder="192.168.1.0/24"
                @change="updateIpWhitelist"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Rate Limiting">
              <el-switch
                v-model="formData.accessControl.rateLimiting.enabled"
                @change="updateAccessControl"
              />
              <div class="field-help">
Limit API requests per IP
</div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Audit Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Document /></el-icon>
            <span>Audit Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="Enable Auditing">
              <el-switch
                v-model="formData.auditSettings.enabled"
                @change="updateAuditSettings"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item
              label="Retention Days"
              prop="auditSettings.retentionDays"
            >
              <el-input-number
                v-model="formData.auditSettings.retentionDays"
                :min="1"
                :max="2555"
                :disabled="!formData.auditSettings.enabled"
                @change="updateAuditSettings"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Export Enabled">
              <el-switch
                v-model="formData.auditSettings.exportEnabled"
                :disabled="!formData.auditSettings.enabled"
                @change="updateAuditSettings"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Encryption -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Key /></el-icon>
            <span>Encryption</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="Encryption Algorithm"
              prop="encryption.algorithm"
              required
            >
              <el-select
                v-model="formData.encryption.algorithm"
                placeholder="Select algorithm"
                @change="updateEncryption"
              >
                <el-option label="AES-256-GCM" value="AES-256-GCM" />
                <el-option
                  label="ChaCha20-Poly1305"
                  value="ChaCha20-Poly1305"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Key Rotation">
              <el-switch
                v-model="formData.encryption.keyRotationEnabled"
                @change="updateEncryption"
              />
              <div class="field-help">
Automatically rotate encryption keys
</div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- API Security -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Connection /></el-icon>
            <span>API Security</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="API Keys Enabled">
              <el-switch
                v-model="formData.apiSecurity.apiKeysEnabled"
                @change="updateApiSecurity"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Request Signing">
              <el-switch
                v-model="formData.apiSecurity.requestSigning"
                @change="updateApiSecurity"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Webhook Verification">
              <el-switch
                v-model="formData.apiSecurity.webhookVerification"
                @change="updateApiSecurity"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { Lock, Document, Key, Connection } from "@element-plus/icons-vue";
import ConfigForm from "./forms/ConfigForm.vue";
import KeyValueEditor from "./forms/KeyValueEditor.vue";
import type { SecuritySettings } from "@/store/settings";

interface Props {
  modelValue: SecuritySettings;
  loading?: boolean;
  validationErrors?: Record<string, string[]>;
}

interface Emits {
  (e: "update:modelValue", value: SecuritySettings): void;
  (e: "field-change", field: string, value: any): void;
  (e: "field-validate", field: string, value: any): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const formData = ref<SecuritySettings>({
  accessControl: {
    ipWhitelist: [],
    ipBlacklist: [],
    corsPolicy: {
      enabled: true,
      allowedOrigins: ["*"],
      allowedMethods: ["GET", "POST", "PUT", "DELETE"],
      allowedHeaders: ["*"],
      allowCredentials: false,
      maxAge: 3600,
    },
    rateLimiting: {
      enabled: true,
      maxPerMinute: 100,
      maxPerHour: 1000,
      maxPerDay: 10000,
    },
  },
  auditSettings: {
    enabled: true,
    retentionDays: 90,
    eventFilters: [],
    exportEnabled: true,
    complianceReporting: false,
  },
  encryption: {
    algorithm: "AES-256-GCM",
    keyRotationEnabled: true,
    keyRotationInterval: 90,
    certificateAutoRenewal: true,
  },
  apiSecurity: {
    apiKeysEnabled: true,
    apiKeyExpiration: 365,
    requestSigning: false,
    webhookVerification: true,
  },
} as any);

const ipWhitelistObject = ref<Record<string, string>>({});

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

const formRules = computed(() => ({
  "encryption.algorithm": [
    {
      required: true,
      message: "Encryption algorithm is required",
      trigger: "change",
    },
  ],
}));

const updateIpWhitelist = (obj: Record<string, string>) => {
  formData.value.accessControl.ipWhitelist = Object.values(obj);
  updateAccessControl();
};

const updateAccessControl = () => {
  handleFieldChange("accessControl", formData.value.accessControl);
};

const updateAuditSettings = () => {
  handleFieldChange("auditSettings", formData.value.auditSettings);
};

const updateEncryption = () => {
  handleFieldChange("encryption", formData.value.encryption);
};

const updateApiSecurity = () => {
  handleFieldChange("apiSecurity", formData.value.apiSecurity);
};

const handleSave = () => {
  emit("update:modelValue", formData.value);
};

const handleReset = () => {
  formData.value = { ...props.modelValue };
};

const handleFieldChange = (field: string, value: any) => {
  emit("field-change", field, value);
};

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      formData.value = { ...newValue };

      // Convert IP whitelist array to object for KeyValueEditor
      const ipObj: Record<string, string> = {};
      newValue.accessControl?.ipWhitelist?.forEach((ip, index) => {
        ipObj[`ip_${index + 1}`] = ip;
      });
      ipWhitelistObject.value = ipObj;
    }
  },
  { immediate: true, deep: true },
);
</script>

<style scoped lang="scss">
.security-config {
  .config-section {
    margin-bottom: 24px;
    border: 1px solid var(--el-border-color-lighter);

    :deep(.el-card__header) {
      background: var(--el-fill-color-extra-light);
      border-bottom: 1px solid var(--el-border-color-lighter);
      padding: 16px 20px;

      .section-header {
        display: flex;
        align-items: center;
        gap: 8px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    :deep(.el-card__body) {
      padding: 24px;
    }
  }

  .field-help {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-top: 4px;
    line-height: 1.4;
  }
}
</style>
