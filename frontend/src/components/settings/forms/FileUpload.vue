<template>
  <div class="file-upload">
    <el-upload
      ref="uploadRef"
      :auto-upload="false"
      :show-file-list="showFileList"
      :accept="accept"
      :multiple="multiple"
      :limit="limit"
      :disabled="disabled"
      class="upload-component"
      @change="handleChange"
      @remove="handleRemove"
      @exceed="handleExceed"
    >
      <el-button type="primary" :disabled="disabled" :size="size">
        <el-icon><FolderOpened /></el-icon>
        {{ buttonText }}
      </el-button>

      <template v-if="showTip" #tip>
        <div class="upload-tip">
          {{ tipText }}
        </div>
      </template>
    </el-upload>

    <div v-if="modelValue && !showFileList" class="current-file">
      <div class="file-info">
        <el-icon><Document /></el-icon>
        <span class="file-name">{{ currentFileName }}</span>
        <el-button
          type="text"
          size="small"
          :disabled="disabled"
          @click="clearFile"
        >
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
    </div>

    <div v-if="error" class="upload-error">
      {{ error }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { ElMessage } from "element-plus";
import { FolderOpened, Document, Delete } from "@element-plus/icons-vue";
import type { UploadFile, UploadFiles } from "element-plus";

interface Props {
  modelValue: string | string[];
  accept?: string;
  multiple?: boolean;
  limit?: number;
  disabled?: boolean;
  size?: "large" | "default" | "small";
  placeholder?: string;
  showFileList?: boolean;
  showTip?: boolean;
  maxSize?: number; // in MB
}

interface Emits {
  (e: "update:modelValue", value: string | string[]): void;
  (e: "change", value: string | string[]): void;
  (e: "error", error: string): void;
}

const props = withDefaults(defineProps<Props>(), {
  accept: "*",
  multiple: false,
  limit: 1,
  disabled: false,
  size: "default",
  placeholder: "Select file",
  showFileList: true,
  showTip: true,
  maxSize: 10,
});

const emit = defineEmits<Emits>();

const uploadRef = ref();
const error = ref("");

const buttonText = computed(() => {
  if (props.multiple) {
    return props.modelValue &&
      Array.isArray(props.modelValue) &&
      props.modelValue.length > 0
      ? "Change Files"
      : "Select Files";
  }
  return props.modelValue ? "Change File" : props.placeholder;
});

const tipText = computed(() => {
  const acceptText =
    props.accept !== "*" ? `Accepted: ${props.accept}` : "All file types";
  const sizeText = `Max size: ${props.maxSize}MB`;
  return `${acceptText}, ${sizeText}`;
});

const currentFileName = computed(() => {
  if (!props.modelValue || typeof props.modelValue !== "string") return "";

  // Extract filename from path or use as-is
  const parts = props.modelValue.split("/");
  return parts[parts.length - 1] || props.modelValue;
});

const handleChange = (uploadFile: UploadFile, _uploadFiles: UploadFiles) => {
  error.value = "";

  if (!uploadFile.raw) return;

  // Validate file size
  const fileSizeMB = uploadFile.raw.size / 1024 / 1024;
  if (fileSizeMB > props.maxSize) {
    error.value = `File size exceeds ${props.maxSize}MB limit`;
    emit("error", error.value);
    return;
  }

  // Read file content
  const reader = new FileReader();
  reader.onload = (e) => {
    const content = e.target?.result as string;

    if (props.multiple) {
      const currentValues = Array.isArray(props.modelValue)
        ? [...props.modelValue]
        : [];
      currentValues.push(content);
      emit("update:modelValue", currentValues);
      emit("change", currentValues);
    } else {
      emit("update:modelValue", content);
      emit("change", content);
    }
  };

  reader.onerror = () => {
    error.value = "Failed to read file";
    emit("error", error.value);
  };

  // Read as text for certificates, data URL for images, etc.
  if (props.accept.includes("image/")) {
    reader.readAsDataURL(uploadFile.raw);
  } else {
    reader.readAsText(uploadFile.raw);
  }
};

const handleRemove = (uploadFile: UploadFile, uploadFiles: UploadFiles) => {
  if (props.multiple) {
    // For multiple files, remove the specific file
    const currentValues = Array.isArray(props.modelValue)
      ? [...props.modelValue]
      : [];
    const index = uploadFiles.findIndex((f) => f.uid === uploadFile.uid);
    if (index !== -1) {
      currentValues.splice(index, 1);
      emit("update:modelValue", currentValues);
      emit("change", currentValues);
    }
  } else {
    // For single file, clear the value
    clearFile();
  }
};

const handleExceed = () => {
  ElMessage.warning(`Maximum ${props.limit} file(s) allowed`);
};

const clearFile = () => {
  if (props.multiple) {
    emit("update:modelValue", []);
    emit("change", []);
  } else {
    emit("update:modelValue", "");
    emit("change", "");
  }

  uploadRef.value?.clearFiles();
  error.value = "";
};

defineExpose({
  clearFiles: () => uploadRef.value?.clearFiles(),
  upload: () => uploadRef.value?.submit(),
});
</script>

<style scoped lang="scss">
.file-upload {
  .upload-component {
    width: 100%;
  }

  .upload-tip {
    color: var(--el-text-color-regular);
    font-size: 12px;
    margin-top: 8px;
    line-height: 1.4;
  }

  .current-file {
    margin-top: 8px;

    .file-info {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 6px;
      font-size: 13px;

      .file-name {
        flex: 1;
        color: var(--el-text-color-primary);
        word-break: break-all;
      }

      .el-icon {
        color: var(--el-text-color-regular);

        &:last-child {
          color: var(--el-color-danger);
          cursor: pointer;

          &:hover {
            color: var(--el-color-danger-light-3);
          }
        }
      }
    }
  }

  .upload-error {
    color: var(--el-color-danger);
    font-size: 12px;
    margin-top: 8px;
    line-height: 1.4;
  }
}

:deep(.el-upload) {
  width: 100%;
}

:deep(.el-upload-dragger) {
  width: 100%;
  height: auto;
  min-height: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

:deep(.el-upload-list) {
  margin-top: 8px;
}
</style>
