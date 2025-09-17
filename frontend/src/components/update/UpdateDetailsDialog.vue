<template>
  <el-dialog
    v-model="visible"
    title="Update Details"
    width="60%"
    :before-close="handleClose"
  >
    <div
v-if="update" class="update-details"
>
      <el-card shadow="never">
        <template #header>
          <div class="detail-header">
            <h3>{{ update.containerName }}</h3>
            <el-tag :type="getStatusTagType(update.status)">
              {{ update.status.toUpperCase() }}
            </el-tag>
          </div>
        </template>

        <el-descriptions :column="2" border>
          <el-descriptions-item label="Update ID">
            {{ update.id }}
          </el-descriptions-item>
          <el-descriptions-item label="Container">
            {{ update.containerName }}
          </el-descriptions-item>
          <el-descriptions-item label="From Version">
            {{ update.fromVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="To Version">
            {{ update.toVersion }}
          </el-descriptions-item>
          <el-descriptions-item label="Status">
            <el-tag :type="getStatusTagType(update.status)">
              {{ update.status }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Started At">
            {{ formatDate(update.startedAt) }}
          </el-descriptions-item>
          <el-descriptions-item
v-if="update.completedAt" label="Completed At"
>
            {{ formatDate(update.completedAt) }}
          </el-descriptions-item>
          <el-descriptions-item
v-if="update.duration" label="Duration"
>
            {{ formatDuration(update.duration) }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card v-if="update.error" shadow="never" class="error-details">
        <template #header>
          <h4>Error Details</h4>
        </template>
        <el-alert
          :title="
            typeof update.error === 'string'
              ? update.error
              : update.error.message
          "
          type="error"
          :description="
            typeof update.error === 'string' ? '' : update.error.details
          "
          show-icon
          :closable="false"
        />
      </el-card>
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">Close</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { UpdateHistoryItem } from "@/types/updates";

interface Props {
  modelValue: boolean;
  update?: UpdateHistoryItem;
}

interface Emits {
  (e: "update:modelValue", value: boolean): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const visible = ref(false);

const getStatusTagType = (status: string) => {
  switch (status) {
    case "completed":
      return "success";
    case "failed":
      return "danger";
    case "cancelled":
      return "warning";
    case "running":
      return "primary";
    default:
      return "info";
  }
};

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString();
};

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
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
.update-details {
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    h3 {
      margin: 0;
    }
  }

  .error-details {
    margin-top: 16px;

    h4 {
      margin: 0;
      color: var(--el-color-danger);
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style>
