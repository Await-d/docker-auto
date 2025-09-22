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
            <span>注册表连接</span>
            <el-button type="primary" size="small" @click="addRegistry">
              <el-icon><Plus /></el-icon>
              添加注册表
            </el-button>
          </div>
        </template>

        <div v-if="!formData.registries || formData.registries.length === 0" class="empty-state">
          <el-empty description="未配置注册表">
            <el-button type="primary" @click="addRegistry">
              添加第一个注册表
            </el-button>
          </el-empty>
        </div>

        <div v-else class="registries-list">
          <div
            v-for="(registry, index) in (formData.registries || [])"
            :key="registry.id"
            class="registry-item"
          >
            <div class="registry-header">
              <el-input
                v-model="registry.name"
                placeholder="注册表名称"
                class="registry-name"
                @input="updateRegistry(index)"
              />
              <el-tag
                :type="registry.enabled ? 'success' : 'info'"
                size="small"
              >
                {{ registry.enabled ? "已启用" : "已禁用" }}
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
                <el-form-item label="类型">
                  <el-select
                    v-model="registry.type"
                    placeholder="选择类型"
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
                <el-form-item label="用户名">
                  <el-input
                    v-model="registry.username"
                    placeholder="用户名（可选）"
                    @input="updateRegistry(index)"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="密码/令牌">
                  <el-input
                    v-model="registry.password"
                    type="password"
                    placeholder="密码或访问令牌"
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
            <span>镜像搜索设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="搜索结果限制"
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
            <el-form-item label="默认注册表" prop="defaultRegistry">
              <el-select
                v-model="formData.defaultRegistry"
                placeholder="选择默认注册表"
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
            <el-form-item label="信任策略" prop="trustPolicy" required>
              <el-select
                v-model="formData.trustPolicy"
                placeholder="选择信任策略"
                @change="handleFieldChange('trustPolicy', $event)"
              >
                <el-option label="始终信任" value="always" />
                <el-option label="仅签名镜像" value="signed" />
                <el-option label="从不信任" value="never" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="安全扫描">
              <el-switch
                v-model="formData.securityScanEnabled"
                @change="handleFieldChange('securityScanEnabled', $event)"
              />
              <div class="field-help">
                启用安全漏洞扫描
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

// Initialize form data from props
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      formData.value = {
        ...formData.value, // Keep existing defaults
        ...newValue, // Override with new values
        // Ensure required arrays are always present
        registries: newValue.registries || [],
      };
    }
  },
  { immediate: true, deep: true },
);

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

const enabledRegistries = computed(() => {
  return (formData.value.registries || []).filter((registry) => registry.enabled);
});

const formRules = computed(() => ({
  searchLimit: [
    { required: true, message: "搜索限制为必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 10 || value > 100) {
          callback(new Error("必须在10到6100之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  trustPolicy: [
    { required: true, message: "信任策略为必填项", trigger: "change" },
  ],
}));

const generateId = (): string => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9);
};

const addRegistry = () => {
  // Ensure registries array exists
  if (!formData.value.registries) {
    formData.value.registries = [];
  }
  
  const newRegistry: DockerRegistry = {
    id: generateId(),
    name: `注册表 ${formData.value.registries.length + 1}`,
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
  if (formData.value.registries && formData.value.registries.length > index) {
    formData.value.registries.splice(index, 1);
    updateRegistries();
  }
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
