<template>
  <div class="cron-editor">
    <!-- Expression Display -->
    <div class="cron-expression">
      <el-input
        v-model="cronExpression"
        placeholder="Enter cron expression"
        :disabled="disabled"
        :size="size"
        @input="handleExpressionChange"
        @blur="validateExpression"
      >
        <template #prepend>
          <span class="cron-label">Cron</span>
        </template>
        <template #append>
          <el-button
            type="primary"
            :disabled="!isValidExpression"
            @click="showBuilder = !showBuilder"
          >
            <el-icon><Setting /></el-icon>
          </el-button>
        </template>
      </el-input>

      <div v-if="description" class="cron-description">
        {{ description }}
      </div>

      <div v-if="nextRuns.length > 0" class="next-runs">
        <div class="next-runs-title">Next runs:</div>
        <div class="next-runs-list">
          <div
v-for="(run, index) in nextRuns" :key="index"
class="next-run"
>
            {{ formatDateTime(run) }}
          </div>
        </div>
      </div>

      <div v-if="!isValidExpression && cronExpression" class="cron-error">
        Invalid cron expression
      </div>
    </div>

    <!-- Visual Builder -->
    <el-collapse-transition>
      <div v-show="showBuilder" class="cron-builder">
        <el-card>
          <template #header>
            <div class="builder-header">
              <span>Cron Expression Builder</span>
              <el-button type="text" @click="showBuilder = false">
                <el-icon><Close /></el-icon>
              </el-button>
            </div>
          </template>

          <el-tabs v-model="builderMode" @tab-change="handleModeChange">
            <!-- Simple Mode -->
            <el-tab-pane label="Simple" name="simple">
              <div class="simple-builder">
                <el-form :model="simpleConfig" label-width="120px">
                  <el-form-item label="Schedule Type">
                    <el-select
                      v-model="simpleConfig.type"
                      @change="handleSimpleTypeChange"
                    >
                      <el-option label="Every minute" value="minute" />
                      <el-option label="Hourly" value="hourly" />
                      <el-option label="Daily" value="daily" />
                      <el-option label="Weekly" value="weekly" />
                      <el-option label="Monthly" value="monthly" />
                    </el-select>
                  </el-form-item>

                  <el-form-item
                    v-if="simpleConfig.type === 'hourly'"
                    label="At minute"
                  >
                    <el-input-number
                      v-model="simpleConfig.minute"
                      :min="0"
                      :max="59"
                      @change="updateSimpleExpression"
                    />
                  </el-form-item>

                  <el-form-item
                    v-if="
                      ['daily', 'weekly', 'monthly'].includes(simpleConfig.type)
                    "
                    label="At time"
                  >
                    <el-time-picker
                      v-model="simpleConfig.time"
                      format="HH:mm"
                      value-format="HH:mm"
                      @change="updateSimpleExpression"
                    />
                  </el-form-item>

                  <el-form-item
                    v-if="simpleConfig.type === 'weekly'"
                    label="On days"
                  >
                    <el-checkbox-group
                      v-model="simpleConfig.weekdays"
                      @change="updateSimpleExpression"
                    >
                      <el-checkbox :label="1"> Monday </el-checkbox>
                      <el-checkbox :label="2"> Tuesday </el-checkbox>
                      <el-checkbox :label="3"> Wednesday </el-checkbox>
                      <el-checkbox :label="4"> Thursday </el-checkbox>
                      <el-checkbox :label="5"> Friday </el-checkbox>
                      <el-checkbox :label="6"> Saturday </el-checkbox>
                      <el-checkbox :label="0"> Sunday </el-checkbox>
                    </el-checkbox-group>
                  </el-form-item>

                  <el-form-item
                    v-if="simpleConfig.type === 'monthly'"
                    label="On day"
                  >
                    <el-input-number
                      v-model="simpleConfig.dayOfMonth"
                      :min="1"
                      :max="31"
                      @change="updateSimpleExpression"
                    />
                  </el-form-item>
                </el-form>
              </div>
            </el-tab-pane>

            <!-- Advanced Mode -->
            <el-tab-pane label="Advanced" name="advanced">
              <div class="advanced-builder">
                <div class="cron-fields">
                  <div class="field-group">
                    <label>Minute (0-59)</label>
                    <el-input
                      v-model="advancedConfig.minute"
                      placeholder="*"
                      @input="updateAdvancedExpression"
                    />
                  </div>

                  <div class="field-group">
                    <label>Hour (0-23)</label>
                    <el-input
                      v-model="advancedConfig.hour"
                      placeholder="*"
                      @input="updateAdvancedExpression"
                    />
                  </div>

                  <div class="field-group">
                    <label>Day of Month (1-31)</label>
                    <el-input
                      v-model="advancedConfig.dayOfMonth"
                      placeholder="*"
                      @input="updateAdvancedExpression"
                    />
                  </div>

                  <div class="field-group">
                    <label>Month (1-12)</label>
                    <el-input
                      v-model="advancedConfig.month"
                      placeholder="*"
                      @input="updateAdvancedExpression"
                    />
                  </div>

                  <div class="field-group">
                    <label>Day of Week (0-6)</label>
                    <el-input
                      v-model="advancedConfig.dayOfWeek"
                      placeholder="*"
                      @input="updateAdvancedExpression"
                    />
                  </div>
                </div>

                <div class="cron-help">
                  <h4>Cron Expression Format</h4>
                  <div class="help-content">
                    <div class="help-item">
<strong>*</strong> - Any value
</div>
                    <div class="help-item">
                      <strong>,</strong> - Value list separator (e.g., 1,3,5)
                    </div>
                    <div class="help-item">
                      <strong>-</strong> - Range of values (e.g., 1-5)
                    </div>
                    <div class="help-item">
                      <strong>/</strong> - Step values (e.g., */5 = every 5)
                    </div>
                  </div>
                </div>
              </div>
            </el-tab-pane>

            <!-- Presets -->
            <el-tab-pane label="Presets" name="presets">
              <div class="presets">
                <div
                  v-for="preset in cronPresets"
                  :key="preset.name"
                  class="preset-item"
                  @click="selectPreset(preset)"
                >
                  <div class="preset-name">
                    {{ preset.name }}
                  </div>
                  <div class="preset-expression">
                    {{ preset.expression }}
                  </div>
                  <div class="preset-description">
                    {{ preset.description }}
                  </div>
                </div>
              </div>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { Setting, Close } from "@element-plus/icons-vue";
import dayjs from "dayjs";

interface Props {
  modelValue: string;
  disabled?: boolean;
  size?: "large" | "default" | "small";
  showNextRuns?: boolean;
  timezone?: string;
}

interface Emits {
  (e: "update:modelValue", value: string): void;
  (e: "change", value: string, description: string): void;
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  size: "default",
  showNextRuns: true,
  timezone: "UTC",
});

const emit = defineEmits<Emits>();

const cronExpression = ref(props.modelValue || "0 0 * * *");
const showBuilder = ref(false);
const builderMode = ref("simple");

// Simple configuration
const simpleConfig = ref({
  type: "daily",
  minute: 0,
  time: "00:00",
  weekdays: [1, 2, 3, 4, 5] as number[],
  dayOfMonth: 1,
});

// Advanced configuration
const advancedConfig = ref({
  minute: "*",
  hour: "*",
  dayOfMonth: "*",
  month: "*",
  dayOfWeek: "*",
});

const cronPresets = ref([
  {
    name: "Every minute",
    expression: "* * * * *",
    description: "Runs every minute",
  },
  {
    name: "Every 5 minutes",
    expression: "*/5 * * * *",
    description: "Runs every 5 minutes",
  },
  {
    name: "Every hour",
    expression: "0 * * * *",
    description: "Runs at the beginning of every hour",
  },
  {
    name: "Every day at midnight",
    expression: "0 0 * * *",
    description: "Runs once a day at midnight",
  },
  {
    name: "Every day at 6 AM",
    expression: "0 6 * * *",
    description: "Runs once a day at 6:00 AM",
  },
  {
    name: "Every weekday at 9 AM",
    expression: "0 9 * * 1-5",
    description: "Runs Monday through Friday at 9:00 AM",
  },
  {
    name: "Every Sunday at noon",
    expression: "0 12 * * 0",
    description: "Runs every Sunday at 12:00 PM",
  },
  {
    name: "First day of month at midnight",
    expression: "0 0 1 * *",
    description: "Runs on the first day of every month at midnight",
  },
]);

const isValidExpression = computed(() => {
  return validateCronExpression(cronExpression.value);
});

const description = computed(() => {
  if (!isValidExpression.value) return "";
  return describeCronExpression(cronExpression.value);
});

const nextRuns = computed(() => {
  if (!isValidExpression.value || !props.showNextRuns) return [];
  return calculateNextRuns(cronExpression.value, 5);
});

const handleExpressionChange = (value: string) => {
  cronExpression.value = value;
  emit("update:modelValue", value);
  if (isValidExpression.value) {
    emit("change", value, description.value);
  }
};

const validateExpression = () => {
  // Additional validation logic if needed
};

const handleModeChange = (mode: string | number) => {
  builderMode.value = mode.toString();
  const modeString = mode.toString();
  if (modeString === "advanced") {
    parseExpressionToAdvanced();
  } else if (modeString === "simple") {
    parseExpressionToSimple();
  }
};

const handleSimpleTypeChange = () => {
  updateSimpleExpression();
};

const updateSimpleExpression = () => {
  let expression = "";
  const { type, minute, time, weekdays, dayOfMonth } = simpleConfig.value;

  switch (type) {
    case "minute":
      expression = "* * * * *";
      break;
    case "hourly":
      expression = `${minute} * * * *`;
      break;
    case "daily": {
      const [hour, min] = (time || "00:00").split(":");
      expression = `${min} ${hour} * * *`;
      break;
    }
    case "weekly": {
      const [weekHour, weekMin] = (time || "00:00").split(":");
      const days = weekdays.length > 0 ? weekdays.join(",") : "*";
      expression = `${weekMin} ${weekHour} * * ${days}`;
      break;
    }
    case "monthly": {
      const [monthHour, monthMin] = (time || "00:00").split(":");
      expression = `${monthMin} ${monthHour} ${dayOfMonth} * *`;
      break;
    }
  }

  cronExpression.value = expression;
  emit("update:modelValue", expression);
  emit("change", expression, description.value);
};

const updateAdvancedExpression = () => {
  const { minute, hour, dayOfMonth, month, dayOfWeek } = advancedConfig.value;
  const expression = `${minute} ${hour} ${dayOfMonth} ${month} ${dayOfWeek}`;

  cronExpression.value = expression;
  emit("update:modelValue", expression);
  emit("change", expression, description.value);
};

const parseExpressionToSimple = () => {
  // Parse current expression to simple config
  const parts = cronExpression.value.split(" ");
  if (parts.length >= 5) {
    const [min, hour, dayOfMonth, month, dayOfWeek] = parts;

    if (
      min === "*" &&
      hour === "*" &&
      dayOfMonth === "*" &&
      month === "*" &&
      dayOfWeek === "*"
    ) {
      simpleConfig.value.type = "minute";
    } else if (
      hour === "*" &&
      dayOfMonth === "*" &&
      month === "*" &&
      dayOfWeek === "*"
    ) {
      simpleConfig.value.type = "hourly";
      simpleConfig.value.minute = parseInt(min) || 0;
    } else if (dayOfMonth === "*" && month === "*" && dayOfWeek === "*") {
      simpleConfig.value.type = "daily";
      simpleConfig.value.time = `${hour.padStart(2, "0")}:${min.padStart(2, "0")}`;
    }
    // Add more parsing logic as needed
  }
};

const parseExpressionToAdvanced = () => {
  const parts = cronExpression.value.split(" ");
  if (parts.length >= 5) {
    advancedConfig.value = {
      minute: parts[0] || "*",
      hour: parts[1] || "*",
      dayOfMonth: parts[2] || "*",
      month: parts[3] || "*",
      dayOfWeek: parts[4] || "*",
    };
  }
};

const selectPreset = (preset: any) => {
  cronExpression.value = preset.expression;
  emit("update:modelValue", preset.expression);
  emit("change", preset.expression, preset.description);
  showBuilder.value = false;
};

const validateCronExpression = (expression: string): boolean => {
  if (!expression || typeof expression !== "string") return false;

  const parts = expression.trim().split(/\s+/);
  if (parts.length !== 5) return false;

  // Basic validation for each field
  const patterns = [
    /^(\*|([0-5]?\d)(,([0-5]?\d))*|\*\/[0-5]?\d|([0-5]?\d)-([0-5]?\d))$/, // minute
    /^(\*|([01]?\d|2[0-3])(,([01]?\d|2[0-3]))*|\*\/([01]?\d|2[0-3])|([01]?\d|2[0-3])-([01]?\d|2[0-3]))$/, // hour
    /^(\*|([1-2]?\d|3[01])(,([1-2]?\d|3[01]))*|\*\/([1-2]?\d|3[01])|([1-2]?\d|3[01])-([1-2]?\d|3[01]))$/, // day of month
    /^(\*|([1-9]|1[0-2])(,([1-9]|1[0-2]))*|\*\/([1-9]|1[0-2])|([1-9]|1[0-2])-([1-9]|1[0-2]))$/, // month
    /^(\*|[0-6](,[0-6])*|\*\/[0-6]|[0-6]-[0-6])$/, // day of week
  ];

  return parts.every((part, index) => patterns[index]?.test(part));
};

const describeCronExpression = (expression: string): string => {
  // This is a simplified description generator
  // A full implementation would require a comprehensive cron parser
  const parts = expression.split(" ");
  if (parts.length !== 5) return "Invalid expression";

  const [min, hour, dayOfMonth, month, dayOfWeek] = parts;

  // Handle some common patterns
  if (expression === "* * * * *") return "Every minute";
  if (expression === "0 * * * *") return "Every hour";
  if (expression === "0 0 * * *") return "Every day at midnight";
  if (expression === "0 0 * * 0") return "Every Sunday at midnight";
  if (expression === "0 0 1 * *") return "First day of every month at midnight";

  // Basic description
  let desc = "At ";
  if (min !== "*") desc += `minute ${min} `;
  if (hour !== "*") desc += `hour ${hour} `;
  if (dayOfMonth !== "*") desc += `on day ${dayOfMonth} `;
  if (month !== "*") desc += `of month ${month} `;
  if (dayOfWeek !== "*") desc += `on weekday ${dayOfWeek}`;

  return desc.trim();
};

const calculateNextRuns = (_expression: string, count: number): Date[] => {
  // This is a simplified implementation
  // A full implementation would require a proper cron scheduler
  const runs: Date[] = [];
  const now = new Date();

  // For demo purposes, add some future times
  for (let i = 1; i <= count; i++) {
    const nextRun = new Date(now.getTime() + i * 24 * 60 * 60 * 1000); // Add i days
    runs.push(nextRun);
  }

  return runs;
};

const formatDateTime = (date: Date): string => {
  return dayjs(date).format("YYYY-MM-DD HH:mm:ss");
};

watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue !== cronExpression.value) {
      cronExpression.value = newValue || "0 0 * * *";
    }
  },
);

onMounted(() => {
  if (!cronExpression.value) {
    cronExpression.value = "0 0 * * *";
    emit("update:modelValue", cronExpression.value);
  }
});
</script>

<style scoped lang="scss">
.cron-editor {
  .cron-expression {
    margin-bottom: 12px;

    .cron-label {
      font-weight: 500;
      color: var(--el-text-color-primary);
    }

    .cron-description {
      margin-top: 8px;
      font-size: 13px;
      color: var(--el-color-primary);
      font-weight: 500;
    }

    .next-runs {
      margin-top: 12px;
      padding: 12px;
      background: var(--el-fill-color-extra-light);
      border-radius: 6px;

      .next-runs-title {
        font-size: 12px;
        font-weight: 500;
        color: var(--el-text-color-primary);
        margin-bottom: 6px;
      }

      .next-runs-list {
        display: flex;
        flex-direction: column;
        gap: 2px;

        .next-run {
          font-size: 12px;
          color: var(--el-text-color-regular);
          font-family: monospace;
        }
      }
    }

    .cron-error {
      margin-top: 8px;
      font-size: 12px;
      color: var(--el-color-danger);
    }
  }

  .cron-builder {
    margin-top: 16px;

    .builder-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .simple-builder {
      :deep(.el-form-item) {
        margin-bottom: 18px;
      }

      :deep(.el-checkbox-group) {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
      }
    }

    .advanced-builder {
      .cron-fields {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
        gap: 16px;
        margin-bottom: 24px;

        .field-group {
          label {
            display: block;
            font-size: 12px;
            font-weight: 500;
            color: var(--el-text-color-primary);
            margin-bottom: 6px;
          }
        }
      }

      .cron-help {
        background: var(--el-fill-color-extra-light);
        border-radius: 6px;
        padding: 16px;

        h4 {
          margin: 0 0 12px 0;
          font-size: 14px;
          color: var(--el-text-color-primary);
        }

        .help-content {
          display: flex;
          flex-direction: column;
          gap: 6px;

          .help-item {
            font-size: 12px;
            color: var(--el-text-color-regular);

            strong {
              color: var(--el-text-color-primary);
              font-family: monospace;
              background: var(--el-fill-color);
              padding: 2px 4px;
              border-radius: 3px;
              margin-right: 6px;
            }
          }
        }
      }
    }

    .presets {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
      gap: 12px;

      .preset-item {
        padding: 16px;
        border: 1px solid var(--el-border-color-lighter);
        border-radius: 6px;
        cursor: pointer;
        transition: all 0.2s;

        &:hover {
          border-color: var(--el-color-primary);
          background: var(--el-color-primary-light-9);
        }

        .preset-name {
          font-weight: 500;
          color: var(--el-text-color-primary);
          margin-bottom: 4px;
        }

        .preset-expression {
          font-family: monospace;
          font-size: 12px;
          color: var(--el-color-primary);
          margin-bottom: 6px;
          background: var(--el-fill-color);
          padding: 4px 6px;
          border-radius: 3px;
          display: inline-block;
        }

        .preset-description {
          font-size: 12px;
          color: var(--el-text-color-regular);
          line-height: 1.4;
        }
      }
    }
  }
}

@media (max-width: 768px) {
  .cron-editor {
    .cron-builder {
      .advanced-builder {
        .cron-fields {
          grid-template-columns: 1fr;
        }
      }

      .presets {
        grid-template-columns: 1fr;
      }
    }
  }
}
</style>
