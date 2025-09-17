<template>
  <el-dialog
    v-model="visible"
    title="Export Update History"
    width="50%"
    :before-close="handleClose"
  >
    <el-form :model="exportForm" label-width="120px">
      <el-form-item label="Export Format">
        <el-radio-group v-model="exportForm.format">
          <el-radio value="csv"> CSV </el-radio>
          <el-radio value="json"> JSON </el-radio>
          <el-radio value="excel"> Excel </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="Date Range">
        <el-date-picker
          v-model="exportForm.dateRange"
          type="daterange"
          range-separator="To"
          start-placeholder="Start date"
          end-placeholder="End date"
          value-format="YYYY-MM-DD"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="Status Filter">
        <el-checkbox-group v-model="exportForm.statusFilter">
          <el-checkbox value="completed"> Completed </el-checkbox>
          <el-checkbox value="failed"> Failed </el-checkbox>
          <el-checkbox value="cancelled"> Cancelled </el-checkbox>
          <el-checkbox value="running"> Running </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Include Fields">
        <el-checkbox-group v-model="exportForm.includeFields">
          <el-checkbox value="container"> Container Name </el-checkbox>
          <el-checkbox value="versions"> Version Information </el-checkbox>
          <el-checkbox value="timestamps"> Timestamps </el-checkbox>
          <el-checkbox value="duration"> Duration </el-checkbox>
          <el-checkbox value="errors"> Error Details </el-checkbox>
          <el-checkbox value="logs"> Logs Summary </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="Options">
        <el-checkbox v-model="exportForm.includeHeaders">
          Include Headers
        </el-checkbox>
        <br>
        <el-checkbox v-model="exportForm.compressOutput">
          Compress Output
        </el-checkbox>
      </el-form-item>
    </el-form>

    <el-alert
title="Export Preview" type="info"
:closable="false" show-icon
>
      <template #default>
        <p><strong>Records to export:</strong> {{ estimatedRecords }}</p>
        <p><strong>Estimated file size:</strong> {{ estimatedSize }}</p>
        <p><strong>Export will include:</strong> {{ selectedFieldsText }}</p>
      </template>
    </el-alert>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">Cancel</el-button>
        <el-button
          type="primary"
          :loading="exporting"
          :disabled="!canExport"
          @click="handleExport"
        >
          <el-icon><Download /></el-icon>
          Export
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { Download } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";

interface ExportForm {
  format: "csv" | "json" | "excel";
  dateRange: [string, string] | null;
  statusFilter: string[];
  includeFields: string[];
  includeHeaders: boolean;
  compressOutput: boolean;
}

interface Props {
  modelValue: boolean;
  totalRecords?: number;
}

interface Emits {
  (e: "update:modelValue", value: boolean): void;
  (e: "export", options: ExportForm): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const visible = ref(false);
const exporting = ref(false);

const exportForm = ref<ExportForm>({
  format: "csv",
  dateRange: null,
  statusFilter: ["completed", "failed"],
  includeFields: ["container", "versions", "timestamps", "duration"],
  includeHeaders: true,
  compressOutput: false,
});

const canExport = computed(() => {
  return exportForm.value.includeFields.length > 0;
});

const estimatedRecords = computed(() => {
  // Simulate filtering logic
  const total = props.totalRecords || 100;
  const statusFilterRatio = exportForm.value.statusFilter.length / 4;
  return Math.floor(total * statusFilterRatio);
});

const estimatedSize = computed(() => {
  const records = estimatedRecords.value;
  const fieldsCount = exportForm.value.includeFields.length;
  let sizeInKB = 0;

  switch (exportForm.value.format) {
    case "csv":
      sizeInKB = records * fieldsCount * 0.05;
      break;
    case "json":
      sizeInKB = records * fieldsCount * 0.1;
      break;
    case "excel":
      sizeInKB = records * fieldsCount * 0.08;
      break;
  }

  if (exportForm.value.compressOutput) {
    sizeInKB *= 0.3; // Assume 70% compression
  }

  if (sizeInKB < 1) {
    return "< 1 KB";
  } else if (sizeInKB < 1024) {
    return `${Math.round(sizeInKB)} KB`;
  } else {
    return `${(sizeInKB / 1024).toFixed(1)} MB`;
  }
});

const selectedFieldsText = computed(() => {
  const fieldLabels: Record<string, string> = {
    container: "Container Names",
    versions: "Version Info",
    timestamps: "Timestamps",
    duration: "Duration",
    errors: "Error Details",
    logs: "Logs Summary",
  };

  return exportForm.value.includeFields
    .map((field) => fieldLabels[field])
    .join(", ");
});

const handleExport = async () => {
  exporting.value = true;

  try {
    // Simulate export process
    await new Promise((resolve) => setTimeout(resolve, 2000));

    // Emit export event with form data
    emit("export", { ...exportForm.value });

    ElMessage.success(
      `Export completed! ${estimatedRecords.value} records exported.`,
    );
    handleClose();
  } catch (error) {
    ElMessage.error("Export failed. Please try again.");
    console.error("Export error:", error);
  } finally {
    exporting.value = false;
  }
};

const handleClose = () => {
  emit("update:modelValue", false);
};

watch(
  () => props.modelValue,
  (newValue) => {
    visible.value = newValue;
  },
);

watch(visible, (newValue) => {
  emit("update:modelValue", newValue);
});
</script>

<style scoped lang="scss">
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.el-checkbox-group) {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .el-checkbox {
    margin-right: 0;
  }
}

:deep(.el-alert) {
  margin-top: 16px;

  .el-alert__content {
    p {
      margin: 4px 0;

      &:last-child {
        margin-bottom: 0;
      }
    }
  }
}
</style>
