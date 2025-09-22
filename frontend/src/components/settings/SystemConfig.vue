<template>
  <div class="system-config">
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
      <!-- System Information (Read-only) -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><InfoFilled /></el-icon>
            <span>系统信息</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="版本">
              <el-input
                :value="systemInfo.version"
                readonly
                class="readonly-input"
              >
                <template #suffix>
                  <el-tag
type="success" size="small"> 当前版本 </el-tag>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="构建日期">
              <el-input
                :value="formatDate(systemInfo.buildDate)"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="运行时">
              <el-input
                :value="systemInfo.runtime"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="平台">
              <el-input
                :value="systemInfo.platform"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="安装路径">
              <el-input
                :value="systemInfo.installPath"
                readonly
                class="readonly-input"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- General Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Setting /></el-icon>
            <span>常规设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="系统名称" prop="systemName" required>
              <el-input
                v-model="formData.systemName"
                placeholder="输入系统名称"
                maxlength="100"
                show-word-limit
                @input="handleFieldChange('systemName', $event)"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="默认时区" prop="timezone" required>
              <el-select
                v-model="formData.timezone"
                placeholder="选择时区"
                filterable
                @change="handleFieldChange('timezone', $event)"
              >
                <el-option
                  v-for="tz in timezones"
                  :key="tz.value"
                  :label="tz.label"
                  :value="tz.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item label="系统描述" prop="systemDescription">
              <el-input
                v-model="formData.systemDescription"
                type="textarea"
                :rows="3"
                placeholder="输入系统描述"
                maxlength="500"
                show-word-limit
                @input="handleFieldChange('systemDescription', $event)"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item label="语言" prop="language" required>
              <el-select
                v-model="formData.language"
                placeholder="选择语言"
                @change="handleFieldChange('language', $event)"
              >
                <el-option
                  v-for="lang in languages"
                  :key="lang.value"
                  :label="lang.label"
                  :value="lang.value"
                >
                  <div class="language-option">
                    <span class="language-flag">{{ lang.flag }}</span>
                    <span class="language-name">{{ lang.label }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="日期格式" prop="dateFormat" required>
              <el-select
                v-model="formData.dateFormat"
                placeholder="选择日期格式"
                @change="handleFieldChange('dateFormat', $event)"
              >
                <el-option
                  v-for="format in dateFormats"
                  :key="format.value"
                  :label="format.label"
                  :value="format.value"
                >
                  <div class="format-option">
                    <span class="format-pattern">{{ format.label }}</span>
                    <span class="format-example">{{ format.example }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="时间格式" prop="timeFormat" required>
              <el-select
                v-model="formData.timeFormat"
                placeholder="选择时间格式"
                @change="handleFieldChange('timeFormat', $event)"
              >
                <el-option
                  v-for="format in timeFormats"
                  :key="format.value"
                  :label="format.label"
                  :value="format.value"
                >
                  <div class="format-option">
                    <span class="format-pattern">{{ format.label }}</span>
                    <span class="format-example">{{ format.example }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Session Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Clock /></el-icon>
            <span>会话设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
              label="会话超时"
              prop="sessionTimeout"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.sessionTimeout"
                  :min="5"
                  :max="1440"
                  :step="5"
                  @change="handleFieldChange('sessionTimeout', $event)"
                />
                <span class="timeout-unit">分钟</span>
              </div>
              <div class="field-help">
                用户无活动后会话将过期的时间
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="自动登出警告">
              <el-switch
                v-model="formData.autoLogoutWarning"
                @change="handleFieldChange('autoLogoutWarning', $event)"
              />
              <div class="field-help">
在会话过期前显示警告
</div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Performance Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Monitor /></el-icon>
            <span>性能设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="8">
            <el-form-item
              label="最大并发操作数"
              prop="maxConcurrentOperations"
              required
            >
              <el-input-number
                v-model="formData.maxConcurrentOperations"
                :min="1"
                :max="100"
                @change="handleFieldChange('maxConcurrentOperations', $event)"
              />
              <div class="field-help">
                同时运行的最大操作数量
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item
              label="请求超时时间"
              prop="requestTimeout"
              required
            >
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.requestTimeout"
                  :min="5"
                  :max="300"
                  :step="5"
                  @change="handleFieldChange('requestTimeout', $event)"
                />
                <span class="timeout-unit">秒</span>
              </div>
              <div class="field-help">
API请求的超时时间
</div>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item
label="缓存生存时间" prop="cacheTtl"
required
>
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.cacheTtl"
                  :min="1"
                  :max="1440"
                  @change="handleFieldChange('cacheTtl', $event)"
                />
                <span class="timeout-unit">分钟</span>
              </div>
              <div class="field-help">
缓存数据的生存时间
</div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item
label="资源使用限制" prop="resourceLimits"
>
              <div class="resource-limits">
                <div class="limit-item">
                  <label>CPU限制 (%)</label>
                  <el-slider
                    v-model="resourceLimitsCpu"
                    :min="10"
                    :max="100"
                    :step="5"
                    show-input
                    @change="handleResourceLimitChange"
                  />
                </div>
                <div class="limit-item">
                  <label>内存限制 (%)</label>
                  <el-slider
                    v-model="resourceLimitsMemory"
                    :min="10"
                    :max="100"
                    :step="5"
                    show-input
                    @change="handleResourceLimitChange"
                  />
                </div>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item
label="日志保留时间" prop="logRetention"
required
>
              <div class="timeout-input">
                <el-input-number
                  v-model="formData.logRetention"
                  :min="1"
                  :max="365"
                  @change="handleFieldChange('logRetention', $event)"
                />
                <span class="timeout-unit">天</span>
              </div>
              <div class="field-help">
系统日志的保留时长
</div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- Maintenance Settings -->
      <el-card class="config-section" shadow="never">
        <template #header>
          <div class="section-header">
            <el-icon><Tools /></el-icon>
            <span>维护设置</span>
          </div>
        </template>

        <el-row :gutter="24">
          <el-col :span="12">
            <el-form-item label="自动清理">
              <el-switch
                v-model="formData.autoCleanup"
                @change="handleFieldChange('autoCleanup', $event)"
              />
              <div class="field-help">
                自动清理临时文件和日志
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="自动更新">
              <el-switch
                v-model="formData.autoUpdates"
                @change="handleFieldChange('autoUpdates', $event)"
              />
              <div class="field-help">
                自动更新系统组件
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="24">
          <el-col :span="24">
            <el-form-item
label="维护时间窗口" prop="maintenanceWindow"
>
              <div class="maintenance-window">
                <div class="window-item">
                  <label>开始时间</label>
                  <el-time-picker
                    v-model="maintenanceWindowStart"
                    format="HH:mm"
                    value-format="HH:mm"
                    @change="handleMaintenanceWindowChange"
                  />
                </div>
                <div class="window-item">
                  <label>结束时间</label>
                  <el-time-picker
                    v-model="maintenanceWindowEnd"
                    format="HH:mm"
                    value-format="HH:mm"
                    @change="handleMaintenanceWindowChange"
                  />
                </div>
                <div class="window-item">
                  <label>维护日期</label>
                  <el-checkbox-group
                    v-model="maintenanceWindowDays"
                    @change="handleMaintenanceWindowChange"
                  >
                    <el-checkbox :label="0"> 周日 </el-checkbox>
                    <el-checkbox :label="1"> 周一 </el-checkbox>
                    <el-checkbox :label="2"> 周二 </el-checkbox>
                    <el-checkbox :label="3"> 周三 </el-checkbox>
                    <el-checkbox :label="4"> 周四 </el-checkbox>
                    <el-checkbox :label="5"> 周五 </el-checkbox>
                    <el-checkbox :label="6"> 周六 </el-checkbox>
                  </el-checkbox-group>
                </div>
              </div>
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>
    </ConfigForm>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import {
  InfoFilled,
  Setting,
  Clock,
  Monitor,
  Tools,
} from "@element-plus/icons-vue";
import dayjs from "dayjs";
import ConfigForm from "./forms/ConfigForm.vue";
import type { GeneralSettings } from "@/store/settings";

interface Props {
  modelValue: GeneralSettings;
  loading?: boolean;
  validationErrors?: Record<string, string[]>;
}

interface Emits {
  (e: "update:modelValue", value: GeneralSettings): void;
  (e: "field-change", field: string, value: any): void;
  (e: "field-validate", field: string, value: any): void;
  (e: "test-configuration", config: GeneralSettings): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const formData = ref<GeneralSettings>({
  systemName: "",
  systemDescription: "",
  timezone: "UTC",
  language: "en",
  dateFormat: "YYYY-MM-DD",
  timeFormat: "24",
  sessionTimeout: 30,
  autoLogoutWarning: true,
  maxConcurrentOperations: 10,
  requestTimeout: 30,
  cacheTtl: 60,
  resourceLimits: {
    cpu: 80,
    memory: 80,
  },
  logRetention: 30,
  autoCleanup: true,
  autoUpdates: false,
  maintenanceWindow: {
    start: "02:00",
    end: "04:00",
    days: [0], // Sunday
  },
} as any);

const systemInfo = ref({
  version: "2.1.0",
  buildDate: new Date().toISOString(),
  runtime: "Node.js 20.x",
  platform: "Linux x64",
  installPath: "/opt/docker-auto",
});

const hasChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(props.modelValue);
});

const resourceLimitsCpu = computed({
  get: () => formData.value.resourceLimits?.cpu || 80,
  set: (value: number) => {
    if (!formData.value.resourceLimits) {
      formData.value.resourceLimits = {
        maxMemoryUsage: 80,
        maxCpuUsage: 80,
        maxDiskUsage: 80,
        cpu: 80,
        memory: 80,
      };
    }
    formData.value.resourceLimits!.cpu = value;
  },
});

const resourceLimitsMemory = computed({
  get: () => formData.value.resourceLimits?.memory || 80,
  set: (value: number) => {
    if (!formData.value.resourceLimits) {
      formData.value.resourceLimits = {
        maxMemoryUsage: 80,
        maxCpuUsage: 80,
        maxDiskUsage: 80,
        cpu: 80,
        memory: 80,
      };
    }
    formData.value.resourceLimits!.memory = value;
  },
});

const maintenanceWindowStart = computed({
  get: () => formData.value.maintenanceWindow?.start || "02:00",
  set: (value: string) => {
    if (!formData.value.maintenanceWindow) {
      formData.value.maintenanceWindow = {
        enabled: true,
        startTime: "02:00",
        endTime: "04:00",
        daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
        start: "02:00",
        end: "04:00",
        days: [0, 1, 2, 3, 4, 5, 6],
      };
    }
    formData.value.maintenanceWindow!.start = value;
  },
});

const maintenanceWindowEnd = computed({
  get: () => formData.value.maintenanceWindow?.end || "04:00",
  set: (value: string) => {
    if (!formData.value.maintenanceWindow) {
      formData.value.maintenanceWindow = {
        enabled: true,
        startTime: "02:00",
        endTime: "04:00",
        daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
        start: "02:00",
        end: "04:00",
        days: [0, 1, 2, 3, 4, 5, 6],
      };
    }
    formData.value.maintenanceWindow!.end = value;
  },
});

const maintenanceWindowDays = computed({
  get: () => formData.value.maintenanceWindow?.days || [0, 1, 2, 3, 4, 5, 6],
  set: (value: number[]) => {
    if (!formData.value.maintenanceWindow) {
      formData.value.maintenanceWindow = {
        enabled: true,
        startTime: "02:00",
        endTime: "04:00",
        daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
        start: "02:00",
        end: "04:00",
        days: [0, 1, 2, 3, 4, 5, 6],
      };
    }
    formData.value.maintenanceWindow!.days = value;
  },
});

const timezones = ref([
  { label: "UTC", value: "UTC" },
  { label: "America/New_York (EST/EDT)", value: "America/New_York" },
  { label: "America/Chicago (CST/CDT)", value: "America/Chicago" },
  { label: "America/Denver (MST/MDT)", value: "America/Denver" },
  { label: "America/Los_Angeles (PST/PDT)", value: "America/Los_Angeles" },
  { label: "Europe/London (GMT/BST)", value: "Europe/London" },
  { label: "Europe/Paris (CET/CEST)", value: "Europe/Paris" },
  { label: "Asia/Tokyo (JST)", value: "Asia/Tokyo" },
  { label: "Asia/Shanghai (CST)", value: "Asia/Shanghai" },
  { label: "Australia/Sydney (AEST/AEDT)", value: "Australia/Sydney" },
]);

const languages = ref([
  { label: "English", value: "en", flag: "🇺🇸" },
  { label: "Español", value: "es", flag: "🇪🇸" },
  { label: "Français", value: "fr", flag: "🇫🇷" },
  { label: "Deutsch", value: "de", flag: "🇩🇪" },
  { label: "中文", value: "zh", flag: "🇨🇳" },
  { label: "日本語", value: "ja", flag: "🇯🇵" },
]);

const dateFormats = ref([
  { label: "YYYY-MM-DD", value: "YYYY-MM-DD", example: "2024-01-15" },
  { label: "MM/DD/YYYY", value: "MM/DD/YYYY", example: "01/15/2024" },
  { label: "DD/MM/YYYY", value: "DD/MM/YYYY", example: "15/01/2024" },
  { label: "DD.MM.YYYY", value: "DD.MM.YYYY", example: "15.01.2024" },
  { label: "MMM DD, YYYY", value: "MMM DD, YYYY", example: "Jan 15, 2024" },
]);

const timeFormats = ref([
  { label: "24-hour", value: "24", example: "14:30" },
  { label: "12-hour", value: "12", example: "2:30 PM" },
]);

const formRules = computed(() => ({
  systemName: [
    { required: true, message: "系统名称是必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value.length < 3 || value.length > 100) {
          callback(new Error("长度应在3到100个字符之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  timezone: [
    { required: true, message: "时区是必选项", trigger: "change" },
  ],
  language: [
    { required: true, message: "语言是必选项", trigger: "change" },
  ],
  dateFormat: [
    { required: true, message: "日期格式是必选项", trigger: "change" },
  ],
  timeFormat: [
    { required: true, message: "时间格式是必选项", trigger: "change" },
  ],
  sessionTimeout: [
    { required: true, message: "会话超时是必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 5 || value > 1440) {
          callback(new Error("必须在5到1440分钟之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  maxConcurrentOperations: [
    {
      required: true,
      message: "最大并发操作数是必填项",
      trigger: "blur",
    },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 1 || value > 100) {
          callback(new Error("必须在1到100之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  requestTimeout: [
    { required: true, message: "请求超时是必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 5 || value > 300) {
          callback(new Error("必须在5到300秒之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  cacheTtl: [
    { required: true, message: "缓存生存时间是必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 1 || value > 1440) {
          callback(new Error("必须在1到1440分钟之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
  logRetention: [
    { required: true, message: "日志保留时间是必填项", trigger: "blur" },
    {
      validator: (
        _rule: any,
        value: any,
        callback: (error?: Error) => void,
      ) => {
        if (value < 1 || value > 365) {
          callback(new Error("必须在1到365天之间"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
  ],
}));

const formatDate = (dateString: string): string => {
  return dayjs(dateString).format("YYYY-MM-DD HH:mm:ss");
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

const handleResourceLimitChange = () => {
  emit("field-change", "resourceLimits", formData.value.resourceLimits);
};

const handleMaintenanceWindowChange = () => {
  emit("field-change", "maintenanceWindow", formData.value.maintenanceWindow);
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

onMounted(() => {
  // Load system info
  // This would typically come from an API call
  console.log("系统配置组件已挂载");
});
</script>

<style scoped lang="scss">
.system-config {
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

  .readonly-input {
    :deep(.el-input__wrapper) {
      background: var(--el-fill-color-extra-light);
      cursor: not-allowed;
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

  .field-help {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-top: 4px;
    line-height: 1.4;
  }

  .language-option {
    display: flex;
    align-items: center;
    gap: 8px;

    .language-flag {
      font-size: 16px;
    }
  }

  .format-option {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;

    .format-example {
      color: var(--el-text-color-regular);
      font-size: 12px;
    }
  }

  .resource-limits {
    .limit-item {
      margin-bottom: 16px;

      label {
        display: block;
        font-size: 12px;
        font-weight: 500;
        color: var(--el-text-color-primary);
        margin-bottom: 8px;
      }

      &:last-child {
        margin-bottom: 0;
      }
    }
  }

  .maintenance-window {
    display: flex;
    flex-direction: column;
    gap: 16px;

    .window-item {
      label {
        display: block;
        font-size: 12px;
        font-weight: 500;
        color: var(--el-text-color-primary);
        margin-bottom: 8px;
      }

      :deep(.el-checkbox-group) {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;

        .el-checkbox {
          margin-right: 0;
        }
      }
    }
  }
}

// Large screen optimizations
@media (min-width: 1200px) {
  .system-config {
    .config-section {
      :deep(.el-card__body) {
        padding: 32px;
        max-width: 1000px;
        margin: 0 auto;
      }
    }
  }
}

@media (max-width: 1024px) {
  .system-config {
    .config-section {
      :deep(.el-card__body) {
        padding: 20px;
      }
    }
  }
}

@media (max-width: 768px) {
  .system-config {
    .config-section {
      :deep(.el-card__body) {
        padding: 16px;
      }
    }

    .maintenance-window {
      .window-item {
        :deep(.el-checkbox-group) {
          .el-checkbox {
            flex: 1;
            min-width: 60px;
            justify-content: center;
          }
        }
      }
    }
  }
}
</style>
