<template>
  <div class="registry-config">
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
      <!-- Registry Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Upload /></el-icon>
            <span>Registry Connections</span>
            <el-button type="primary" size="small" @click="addRegistry">
              <el-icon><Plus /></el-icon>
              Add Registry
            </el-button>
          </div>
        </template>

        <div v-if="formData.registries.length === 0" class="empty-state">
          <el-empty description="No registries configured">
            <el-button type="primary" @click="addRegistry">
              Add First Registry
            </el-button>
          </el-empty>
        </div>

        <div v-else class="registries-list">
          <div
            v-for="(registry, index) in formData.registries"
            :key="registry.id"
            class="registry-item"
          >
            <div class="registry-header">
              <el-input
                v-model="registry.name"
                placeholder="Registry name"
                class="registry-name"
                @input="updateRegistry(index)"
              />
              <el-tag
                :type="registry.enabled ? 'success' : 'info'"
                size="small"
              >
                {{ registry.enabled ? "Enabled" : "Disabled" }}
              </el-tag>
              <el-switch
                v-model="registry.enabled"
                @change="updateRegistry(index)"
              />
              <el-button
                type="text"
                size="small"
                class="danger-button"
                @click="removeRegistry(index)"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>

            <el-row :gutter="16">
              <el-col :span="8">
                <el-form-item label="Type">
                  <el-select
                    v-model="registry.type"
                    placeholder="Select type"
                    @change="updateRegistry(index)"
                  >
                    <el-option label="Docker Hub" value="dockerhub" />
                    <el-option label="Harbor" value="harbor" />
                    <el-option label="AWS ECR" value="ecr" />
                    <el-option label="Azure ACR" value="acr" />
                    <el-option label="Google GCR" value="gcr" />
                    <el-option label="Generic" value="generic" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="16">
                <el-form-item label="URL">
                  <el-input
                    v-model="registry.url"
                    placeholder="https://registry.example.com"
                    @input="updateRegistry(index)"
                  />
                </el-form-item>
              </el-col>
            </el-row>

            <el-row :gutter="16">
              <el-col :span="12">
                <el-form-item label="Username">
                  <el-input
                    v-model="registry.username"
                    placeholder="Username (optional)"
                    @input="updateRegistry(index)"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Password/Token">
                  <el-input
                    v-model="registry.password"
                    type="password"
                    placeholder="Password or access token"
                    show-password
                    @input="updateRegistry(index)"
                  />
                </el-form-item>
              </el-col>
            </el-row>
          </div>
        </div>
      </el-card>

      <!-- Search Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Search /></el-icon>
            <span>Image Search Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="Search Result Limit"
              prop="searchLimit"
              required
            >
              <el-input-number
                v-model="formData.searchLimit"
                :min="10"
                :max="100"
                @change="handleFieldChange('searchLimit', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Default Registry" prop="defaultRegistry">
              <el-select
                v-model="formData.defaultRegistry"
                placeholder="Select default registry"
                @change="handleFieldChange('defaultRegistry', $event)"
              >
                <el-option
                  v-for="registry in enabledRegistries"
                  :key="registry.id"
                  :label="registry.name"
                  :value="registry.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Trust Policy" prop="trustPolicy" required>
              <el-select
                v-model="formData.trustPolicy"
                placeholder="Select trust policy"
                @change="handleFieldChange('trustPolicy', $event)"
              >
                <el-option label="Always Trust" value="always" />
                <el-option label="Signed Images Only" value="signed" />
                <el-option label="Never Trust" value="never" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Security Scanning">
              <el-switch
                v-model="formData.securityScanEnabled"
                @change="handleFieldChange('securityScanEnabled', $event)"
              />
              <div class="field-help">
                Enable security vulnerability scanning
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { Upload, Search, Plus, Delete } from "@element-plus/icons-vue";
import ConfigForm from "./forms/ConfigForm.vue";
import type { RegistrySettings, DockerRegistry } from "@/store/settings";

interface Props {
  modelValue: RegistrySettings;
  loading?: boolean;
  validationErrors?: Record<string, string[]>;
}

interface Emits {
  (e: "update:modelValue", value: RegistrySettings): void;
  (e: "field-change", field: string, value: any): void;
  (e: "field-validate", field: string, value: any): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const formData = ref<RegistrySettings>({
  defaultRegistry: "",
  registries: [],
  searchLimit: 50,
  trustPolicy: "signed",
  securityScanEnabled: true,
} as any);

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

const enabledRegistries = computed(() => {
  return formData.value.registries.filter((registry) => registry.enabled);
});

const formRules = computed(() => ({
  searchLimit: [
    { required: true, message: "Search limit is required", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 10 || value > 100) {
          callback(new Error("Must be between 10 and 100"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  trustPolicy: [
    { required: true, message: "Trust policy is required", trigger: "change" },
  ],
}));

const generateId = (): string => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9);
};

const addRegistry = () => {
  const newRegistry: DockerRegistry = {
    id: generateId(),
    name: `Registry ${formData.value.registries.length + 1}`,
    url: "",
    type: "generic",
    username: "",
    password: "",
    isDefault: false,
    healthCheckInterval: 300,
    enabled: true,
  };

  formData.value.registries.push(newRegistry);
  updateRegistries();
};

const removeRegistry = (index: number) => {
  formData.value.registries.splice(index, 1);
  updateRegistries();
};

const updateRegistry = (_index: number) => {
  updateRegistries();
};

const updateRegistries = () => {
  handleFieldChange("registries", formData.value.registries);
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

// Initialize form data from props
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      formData.value = { ...newValue };
    }
  },
  { immediate: true, deep: true },
);
</script>

<style scoped lang="scss">
.registry-config {
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
        justify-content: space-between;
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

  .registries-list {
    .registry-item {
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
      padding: 16px;
      margin-bottom: 16px;

      .registry-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;

        .registry-name {
          flex: 1;
        }

        .danger-button {
          color: var(--el-color-danger);
        }
      }
    }
  }
}

@media (max-width: 768px) {
  .registry-config {
    .config-section {
      :deep(.el-card__body) {
        padding: 16px;
      }

      :deep(.section-header) {
        flex-direction: column;
        gap: 12px;
      }
    }

    .registries-list {
      .registry-item {
        .registry-header {
          flex-direction: column;
          align-items: stretch;
          gap: 8px;
        }
      }
    }
  }
}
</style>
