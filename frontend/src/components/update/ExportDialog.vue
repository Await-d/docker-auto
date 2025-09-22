<template>
  <el-dialog
    v-model="visible"
    title="导出更新历史"
    width="50%"
    :before-close="handleClose"
  >
    <el-form :model="exportForm" label-width="120px">
      <el-form-item label="导出格式">
        <el-radio-group v-model="exportForm.format">
          <el-radio value="csv"> CSV </el-radio>
          <el-radio value="json"> JSON </el-radio>
          <el-radio value="excel"> Excel </el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="日期范围">
        <el-date-picker
          v-model="exportForm.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="状态筛选">
        <el-checkbox-group v-model="exportForm.statusFilter">
          <el-checkbox value="completed"> 已完成 </el-checkbox>
          <el-checkbox value="failed"> 失败 </el-checkbox>
          <el-checkbox value="cancelled"> 已取消 </el-checkbox>
          <el-checkbox value="running"> 运行中 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="包含字段">
        <el-checkbox-group v-model="exportForm.includeFields">
          <el-checkbox value="container"> 容器名称 </el-checkbox>
          <el-checkbox value="versions"> 版本信息 </el-checkbox>
          <el-checkbox value="timestamps"> 时间戳 </el-checkbox>
          <el-checkbox value="duration"> 持续时间 </el-checkbox>
          <el-checkbox value="errors"> 错误详情 </el-checkbox>
          <el-checkbox value="logs"> 日志摘要 </el-checkbox>
        </el-checkbox-group>
      </el-form-item>

      <el-form-item label="选项">
        <el-checkbox v-model="exportForm.includeHeaders">
          包含表头
        </el-checkbox>
        <br>
        <el-checkbox v-model="exportForm.compressOutput">
          压缩输出
        </el-checkbox>
      </el-form-item>
    </el-form>

    <el-alert
title="导出预览" type="info"
:closable="false" show-icon
>
      <template #default>
        <p><strong>待导出记录：</strong> {{ estimatedRecords }}</p>
        <p><strong>预估文件大小：</strong> {{ estimatedSize }}</p>
        <p><strong>导出将包括：</strong> {{ selectedFieldsText }}</p>
      </template>
    </el-alert>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button
          type="primary"
          :loading="exporting"
          :disabled="!canExport"
          @click="handleExport"
        >
          <el-icon><Download /></el-icon>
          导出
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
    container: "容器名称",
    versions: "版本信息",
    timestamps: "时间戳",
    duration: "持续时间",
    errors: "错误详情",
    logs: "日志摘要",
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
      `导出完成！已导出 ${estimatedRecords.value} 条记录。`,
    );
    handleClose();
  } catch (error) {
    ElMessage.error("导出失败。请重试。");
    console.error("导出错误:", error);
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
