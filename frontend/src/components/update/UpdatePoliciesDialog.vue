<template>
  <el-dialog
    v-model="visible"
    title="Update Policies"
    width="900px"
    :before-close="handleClose"
  >
    <div class="policies-manager">
      <el-alert
        title="Update Policies Configuration"
        description="Configure automated update policies for container groups. This is a simplified version - full implementation coming soon."
        type="info"
        show-icon
        :closable="false"
      />

      <div class="policies-placeholder">
        <el-empty description="Update Policies feature coming soon">
          <el-button
type="primary" @click="handleClose"> Close </el-button>
        </el-empty>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from "vue";

// Props
interface Props {
  modelValue: boolean;
}

const props = defineProps<Props>();

// Emits
const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  "policy-applied": [];
}>();

// Computed
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

// Methods
const handleClose = () => {
  visible.value = false;
};
</script>

<style scoped lang="scss">
.policies-manager {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.policies-placeholder {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
