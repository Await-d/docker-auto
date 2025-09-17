<template>
  <div class="docker-config">
    <ConfigForm
      v-model="formData"
      :rules="formRules"
      :saving="loading"
      :has-changes="hasChanges"
      :testable="true"
      @save="handleSave"
      @reset="handleReset"
      @test="handleTestConnection"
      @field-change="handleFieldChange"
    >
      <!-- Docker Connection -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Connection /></el-icon>
            <span>Docker Connection</span>
            <div class="header-status">
              <el-tag
                :type="connectionStatus.type"
                :icon="connectionStatus.icon"
                size="small"
              >
                {{ connectionStatus.text }}
              </el-tag>
            </div>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="Docker Socket/URL" prop="socketPath" required>
              <el-input
                v-model="formData.socketPath"
                placeholder="unix:///var/run/docker.sock or tcp://host:port"
                @input="handleFieldChange('socketPath', $event)"
              >
                <template #prepend>
                  <el-select
                    v-model="connectionType"
                    style="width: 120px"
                    @change="handleConnectionTypeChange"
                  >
                    <el-option label="Unix Socket" value="unix" />
                    <el-option label="TCP" value="tcp" />
                    <el-option label="SSH" value="ssh" />
                  </el-select>
                </template>
                <template #append>
                  <el-button
                    type="primary"
                    :loading="testing"
                    @click="testConnection"
                  >
                    Test
                  </el-button>
                </template>
              </el-input>
              <div class="field-help">
Connection string to Docker daemon
</div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="Connection Timeout"
              prop="connectionTimeout"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.connectionTimeout"
                  :min="5"
                  :max="300"
                  :step="5"
                  @change="handleFieldChange('connectionTimeout', $event)"
                />
                <span class="timeout-unit">seconds</span>
              </div>
              <div class="field-help">
Timeout for Docker API calls
</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Auto Reconnect">
              <el-switch
                v-model="formData.autoReconnect"
                @change="handleFieldChange('autoReconnect', $event)"
              />
              <div class="field-help">
                Automatically reconnect on connection loss
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- TLS Configuration -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Lock /></el-icon>
            <span>TLS Configuration</span>
            <el-switch
              v-model="formData.tlsEnabled"
              class="header-switch"
              @change="handleFieldChange('tlsEnabled', $event)"
            />
          </div>
        </template>

        <div v-if="formData.tlsEnabled">
          <el-row :gutter="24">
            <el-col :span="8">
              <el-form-item label="TLS Verification" prop="tlsVerify">
                <el-switch
                  v-model="formData.tlsVerify"
                  @change="handleFieldChange('tlsVerify', $event)"
                />
                <div class="field-help">
Verify TLS certificates
</div>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="Client Certificate Auth">
                <el-switch
                  v-model="formData.clientCertAuth"
                  @change="handleFieldChange('clientCertAuth', $event)"
                />
                <div class="field-help">
                  Use client certificate authentication
                </div>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="Skip Hostname Verification">
                <el-switch
                  v-model="formData.skipHostnameVerification"
                  @change="
                    handleFieldChange('skipHostnameVerification', $event)
                  "
                />
                <div class="field-help">
                  Skip hostname verification (insecure)
                </div>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="24">
            <el-col :span="8">
              <el-form-item label="CA Certificate" prop="tlsCa">
                <FileUpload
                  v-model="tlsCaValue"
                  accept=".pem,.crt,.cer"
                  placeholder="Upload CA certificate"
                  @change="handleFieldChange('tlsCa', $event)"
                />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item
                v-if="formData.clientCertAuth"
                label="Client Certificate"
                prop="tlsCert"
              >
                <FileUpload
                  v-model="tlsCertValue"
                  accept=".pem,.crt,.cer"
                  placeholder="Upload client certificate"
                  @change="handleFieldChange('tlsCert', $event)"
                />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item
                v-if="formData.clientCertAuth"
                label="Client Key"
                prop="tlsKey"
              >
                <FileUpload
                  v-model="tlsKeyValue"
                  accept=".pem,.key"
                  placeholder="Upload client private key"
                  @change="handleFieldChange('tlsKey', $event)"
                />
              </el-form-item>
            </el-col>
          </el-row>
        </div>

        <div v-else class="tls-disabled">
          <el-alert
type="warning" :closable="false"
show-icon
>
            <template #title>
TLS is disabled
</template>
            Connection to Docker daemon will not be encrypted. Enable TLS for
            secure communication.
          </el-alert>
        </div>
      </el-card>

      <!-- Default Container Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Box /></el-icon>
            <span>Default Container Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="Default Restart Policy"
              prop="defaultRestartPolicy"
              required
            >
              <el-select
                v-model="formData.defaultRestartPolicy"
                placeholder="Select restart policy"
                @change="handleFieldChange('defaultRestartPolicy', $event)"
              >
                <el-option
                  v-for="policy in restartPolicies"
                  :key="policy.value"
                  :label="policy.label"
                  :value="policy.value"
                >
                  <div class="policy-option">
                    <span class="policy-name">{{ policy.label }}</span>
                    <span class="policy-description">{{
                      policy.description
                    }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="Default Network Mode"
              prop="defaultNetworkMode"
              required
            >
              <el-select
                v-model="formData.defaultNetworkMode"
                placeholder="Select network mode"
                @change="handleFieldChange('defaultNetworkMode', $event)"
              >
                <el-option
                  v-for="mode in networkModes"
                  :key="mode.value"
                  :label="mode.label"
                  :value="mode.value"
                >
                  <div class="mode-option">
                    <span class="mode-name">{{ mode.label }}</span>
                    <span class="mode-description">{{ mode.description }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Default CPU Limit">
              <div class="resource-input">
                <el-input-number
                  v-model="formData.defaultCpuLimit"
                  :min="0.1"
                  :max="32"
                  :step="0.1"
                  :precision="1"
                  @change="handleFieldChange('defaultCpuLimit', $event)"
                />
                <span class="resource-unit">cores</span>
              </div>
              <div class="field-help">
                CPU limit for new containers (0 = unlimited)
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Default Memory Limit">
              <div class="resource-input">
                <el-input-number
                  v-model="formData.defaultMemoryLimit"
                  :min="0"
                  :max="64000"
                  :step="100"
                  @change="handleFieldChange('defaultMemoryLimit', $event)"
                />
                <span class="resource-unit">MB</span>
              </div>
              <div class="field-help">
                Memory limit for new containers (0 = unlimited)
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="Default Volume Mounts">
              <KeyValueEditor
                v-model="defaultVolumeMountsValue"
                title="Volume Mounts"
                item-name="Mount"
                key-label="Host Path"
                value-label="Container Path"
                key-placeholder="/host/path"
                value-placeholder="/container/path"
                @change="handleFieldChange('defaultVolumeMounts', $event)"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Image Management -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Picture /></el-icon>
            <span>Image Management</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="Default Pull Policy"
              prop="imagePullPolicy"
              required
            >
              <el-select
                v-model="formData.imagePullPolicy"
                placeholder="Select pull policy"
                @change="handleFieldChange('imagePullPolicy', $event)"
              >
                <el-option
                  v-for="policy in pullPolicies"
                  :key="policy.value"
                  :label="policy.label"
                  :value="policy.value"
                >
                  <div class="policy-option">
                    <span class="policy-name">{{ policy.label }}</span>
                    <span class="policy-description">{{
                      policy.description
                    }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
              label="Image Cleanup Schedule"
              prop="imageCleanupSchedule"
            >
              <CronEditor
                v-model="formData.imageCleanupSchedule"
                @change="handleFieldChange('imageCleanupSchedule', $event)"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Auto Remove Dangling Images">
              <el-switch
                v-model="formData.autoRemoveDanglingImages"
                @change="handleFieldChange('autoRemoveDanglingImages', $event)"
              />
              <div class="field-help">
Automatically remove untagged images
</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Image Verification">
              <el-switch
                v-model="formData.imageVerification"
                @change="handleFieldChange('imageVerification', $event)"
              />
              <div class="field-help">
                Verify image signatures and checksums
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="Registry Connection Timeout">
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.registryTimeout"
                  :min="10"
                  :max="300"
                  :step="10"
                  @change="handleFieldChange('registryTimeout', $event)"
                />
                <span class="timeout-unit">seconds</span>
              </div>
              <div class="field-help">
                Timeout for registry operations (pull, push, etc.)
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Advanced Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Tools /></el-icon>
            <span>Advanced Settings</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="Log Driver" prop="logDriver">
              <el-select
                v-model="formData.logDriver"
                placeholder="Select log driver"
                @change="handleFieldChange('logDriver', $event)"
              >
                <el-option
                  v-for="driver in logDrivers"
                  :key="driver.value"
                  :label="driver.label"
                  :value="driver.value"
                >
                  <div class="driver-option">
                    <span class="driver-name">{{ driver.label }}</span>
                    <span class="driver-description">{{
                      driver.description
                    }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Enable BuildKit">
              <el-switch
                v-model="formData.enableBuildKit"
                @change="handleFieldChange('enableBuildKit', $event)"
              />
              <div class="field-help">
                Use BuildKit for improved build performance
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="Docker CLI Options">
              <KeyValueEditor
                v-model="dockerCliOptionsValue"
                title="Docker CLI Options"
                item-name="Option"
                key-label="Option"
                value-label="Value"
                key-placeholder="--option"
                value-placeholder="value"
                @change="handleFieldChange('dockerCliOptions', $event)"
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
import { ElMessage } from "element-plus";
import {
  Connection,
  Lock,
  Box,
  Picture,
  Tools,
  CircleCheck,
  CircleClose,
  Warning,
} from "@element-plus/icons-vue";
import ConfigForm from "./forms/ConfigForm.vue";
import CronEditor from "./forms/CronEditor.vue";
import KeyValueEditor from "./forms/KeyValueEditor.vue";
import FileUpload from "./forms/FileUpload.vue";
import type { DockerSettings } from "@/store/settings";

interface Props {
  modelValue: DockerSettings;
  loading?: boolean;
  validationErrors?: Record<string, string[]>;
}

interface Emits {
  (e: "update:modelValue", value: DockerSettings): void;
  (e: "field-change", field: string, value: any): void;
  (e: "field-validate", field: string, value: any): void;
  (e: "test-configuration", config: DockerSettings): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const formData = ref<DockerSettings>({
  socketPath: "unix:///var/run/docker.sock",
  connectionTimeout: 30,
  autoReconnect: true,
  tlsEnabled: false,
  tlsVerify: true,
  clientCertAuth: false,
  skipHostnameVerification: false,
  tlsCa: "",
  tlsCert: "",
  tlsKey: "",
  defaultRestartPolicy: "unless-stopped",
  defaultNetworkMode: "bridge",
  defaultCpuLimit: 0,
  defaultMemoryLimit: 0,
  defaultVolumeMounts: {},
  imagePullPolicy: "missing",
  imageCleanupSchedule: "0 2 * * 0",
  autoRemoveDanglingImages: true,
  imageVerification: false,
  registryTimeout: 60,
  logDriver: "json-file",
  enableBuildKit: true,
  dockerCliOptions: {},
} as any);

const testing = ref(false);
const connectionType = ref("unix");
const lastConnectionTest = ref<Date | null>(null);
const connectionTestResult = ref<"success" | "error" | null>(null);

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

const connectionStatus = computed(
  (): {
    type: "success" | "warning" | "info" | "danger";
    icon: any;
    text: string;
  } => {
    if (testing.value) {
      return {
        type: "info",
        icon: "Loading",
        text: "Testing...",
      };
    }

    if (connectionTestResult.value === "success") {
      return {
        type: "success",
        icon: CircleCheck,
        text: "Connected",
      };
    }

    if (connectionTestResult.value === "error") {
      return {
        type: "danger",
        icon: CircleClose,
        text: "Connection Failed",
      };
    }

    return {
      type: "warning",
      icon: Warning,
      text: "Not Tested",
    };
  },
);

const tlsCaValue = computed({
  get: () => formData.value.tlsCa || "",
  set: (value: string) => {
    formData.value.tlsCa = value;
  },
});

const tlsCertValue = computed({
  get: () => formData.value.tlsCert || "",
  set: (value: string) => {
    formData.value.tlsCert = value;
  },
});

const tlsKeyValue = computed({
  get: () => formData.value.tlsKey || "",
  set: (value: string) => {
    formData.value.tlsKey = value;
  },
});

const defaultVolumeMountsValue = computed({
  get: () => formData.value.defaultVolumeMounts || [],
  set: (value: any[]) => {
    formData.value.defaultVolumeMounts = value;
  },
});

const dockerCliOptionsValue = computed({
  get: () => formData.value.dockerCliOptions || {},
  set: (value: Record<string, any>) => {
    formData.value.dockerCliOptions = value;
  },
});

const restartPolicies = ref([
  {
    label: "No",
    value: "no",
    description: "Do not restart container",
  },
  {
    label: "Always",
    value: "always",
    description: "Always restart container",
  },
  {
    label: "Unless Stopped",
    value: "unless-stopped",
    description: "Restart unless manually stopped",
  },
  {
    label: "On Failure",
    value: "on-failure",
    description: "Restart only on failure",
  },
]);

const networkModes = ref([
  {
    label: "Bridge",
    value: "bridge",
    description: "Default bridge network",
  },
  {
    label: "Host",
    value: "host",
    description: "Use host network",
  },
  {
    label: "None",
    value: "none",
    description: "No networking",
  },
  {
    label: "Container",
    value: "container",
    description: "Share another container network",
  },
]);

const pullPolicies = ref([
  {
    label: "Always",
    value: "always",
    description: "Always pull latest image",
  },
  {
    label: "Missing",
    value: "missing",
    description: "Pull only if image missing",
  },
  {
    label: "Never",
    value: "never",
    description: "Never pull images",
  },
]);

const logDrivers = ref([
  {
    label: "JSON File",
    value: "json-file",
    description: "Default JSON logging",
  },
  {
    label: "Syslog",
    value: "syslog",
    description: "Syslog daemon",
  },
  {
    label: "Journald",
    value: "journald",
    description: "Systemd journal",
  },
  {
    label: "None",
    value: "none",
    description: "Disable logging",
  },
]);

const formRules = computed(() => ({
  socketPath: [
    {
      required: true,
      message: "Docker socket path is required",
      trigger: "blur",
    },
  ],
  connectionTimeout: [
    {
      required: true,
      message: "Connection timeout is required",
      trigger: "blur",
    },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 5 || value > 300) {
          callback(new Error("Must be between 5 and 300 seconds"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  defaultRestartPolicy: [
    {
      required: true,
      message: "Default restart policy is required",
      trigger: "change",
    },
  ],
  defaultNetworkMode: [
    {
      required: true,
      message: "Default network mode is required",
      trigger: "change",
    },
  ],
  imagePullPolicy: [
    {
      required: true,
      message: "Image pull policy is required",
      trigger: "change",
    },
  ],
}));

const handleSave = (config: Record<string, any>) => {
  emit("update:modelValue", config as DockerSettings);
};

const handleReset = () => {
  formData.value = { ...props.modelValue };
};

const handleFieldChange = (field: string, value: any) => {
  emit("field-change", field, value);
};

const handleConnectionTypeChange = (type: string) => {
  connectionType.value = type;

  switch (type) {
    case "unix":
      formData.value.socketPath = "unix:///var/run/docker.sock";
      break;
    case "tcp":
      formData.value.socketPath = "tcp://localhost:2376";
      break;
    case "ssh":
      formData.value.socketPath = "ssh://user@host";
      break;
  }

  handleFieldChange("socketPath", formData.value.socketPath);
};

const testConnection = async () => {
  testing.value = true;
  connectionTestResult.value = null;

  try {
    // Simulate connection test
    await new Promise((resolve) => setTimeout(resolve, 2000));

    // For demo purposes, randomly succeed or fail
    const success = Math.random() > 0.3;

    if (success) {
      connectionTestResult.value = "success";
      ElMessage.success("Docker connection successful");
    } else {
      connectionTestResult.value = "error";
      ElMessage.error("Failed to connect to Docker daemon");
    }

    lastConnectionTest.value = new Date();
  } catch (error) {
    connectionTestResult.value = "error";
    ElMessage.error("Connection test failed");
  } finally {
    testing.value = false;
  }
};

const handleTestConnection = async (config: Record<string, any>) => {
  emit("test-configuration", config as DockerSettings);
  await testConnection();
};

// Initialize form data from props
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue) {
      formData.value = { ...newValue };

      // Determine connection type from socket path
      if (newValue.socketPath) {
        if (newValue.socketPath.startsWith("unix:")) {
          connectionType.value = "unix";
        } else if (newValue.socketPath.startsWith("tcp:")) {
          connectionType.value = "tcp";
        } else if (newValue.socketPath.startsWith("ssh:")) {
          connectionType.value = "ssh";
        }
      }
    }
  },
  { immediate: true, deep: true },
);
</script>

<style scoped lang="scss">
.docker-config {
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

        .header-status {
          margin-left: auto;
        }

        .header-switch {
          margin-left: auto;
        }
      }
    }

    :deep(.el-card__body) {
      padding: 24px;
    }
  }

  .timeout-input {
    display: flex;
    align-items: center;
    gap: 8px;

    .timeout-unit {
      color: var(--el-text-color-regular);
      font-size: 14px;
      white-space: nowrap;
    }
  }

  .resource-input {
    display: flex;
    align-items: center;
    gap: 8px;

    .resource-unit {
      color: var(--el-text-color-regular);
      font-size: 14px;
      white-space: nowrap;
    }
  }

  .field-help {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-top: 4px;
    line-height: 1.4;
  }

  .policy-option,
  .mode-option,
  .driver-option {
    display: flex;
    flex-direction: column;
    width: 100%;

    .policy-name,
    .mode-name,
    .driver-name {
      font-weight: 500;
    }

    .policy-description,
    .mode-description,
    .driver-description {
      color: var(--el-text-color-regular);
      font-size: 12px;
      margin-top: 2px;
    }
  }

  .tls-disabled {
    padding: 20px;
  }
}

@media (max-width: 768px) {
  .docker-config {
    .config-section {
      :deep(.el-card__body) {
        padding: 16px;
      }

      :deep(.section-header) {
        flex-direction: column;
        align-items: flex-start;
        gap: 12px;

        .header-status,
        .header-switch {
          margin-left: 0;
        }
      }
    }
  }
}
</style>
