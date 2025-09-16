<template>
  <el-form
    ref="formRef"
    :model="formData"
    :rules="formRules"
    :label-width="labelWidth"
    :size="size"
    :disabled="disabled"
    class="config-form"
    @validate="handleValidate"
  >
    <slot :form="formData" :rules="formRules" />

    <div v-if="showActions" class="form-actions">
      <slot name="actions" :save="handleSave" :reset="handleReset" :test="handleTest">
        <el-button @click="handleReset" :disabled="!hasChanges || saving">
          Reset
        </el-button>
        <el-button
          v-if="testable"
          type="info"
          @click="handleTest"
          :loading="testing"
          :disabled="!isValid || saving"
        >
          Test Configuration
        </el-button>
        <el-button
          type="primary"
          @click="handleSave"
          :loading="saving"
          :disabled="!hasChanges || !isValid"
        >
          Save
        </el-button>
      </slot>
    </div>
  </el-form>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'

interface Props {
  modelValue: Record<string, any>
  rules?: FormRules
  labelWidth?: string
  size?: 'large' | 'default' | 'small'
  disabled?: boolean
  showActions?: boolean
  testable?: boolean
  saving?: boolean
  hasChanges?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: Record<string, any>): void
  (e: 'save', value: Record<string, any>): void
  (e: 'reset'): void
  (e: 'test', value: Record<string, any>): void
  (e: 'validate', field: string, valid: boolean, message?: string): void
  (e: 'field-change', field: string, value: any): void
}

const props = withDefaults(defineProps<Props>(), {
  labelWidth: '150px',
  size: 'default',
  disabled: false,
  showActions: true,
  testable: false,
  saving: false,
  hasChanges: false
})

const emit = defineEmits<Emits>()

const formRef = ref<FormInstance>()
const testing = ref(false)

const formData = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const formRules = computed(() => props.rules || {})

const isValid = ref(true)

const handleValidate = (prop: string, isValid: boolean, message: string) => {
  emit('validate', prop, isValid, message)
}

const handleSave = async () => {
  try {
    const valid = await formRef.value?.validate()
    if (valid) {
      emit('save', formData.value)
    }
  } catch (error) {
    console.error('Form validation failed:', error)
  }
}

const handleReset = () => {
  formRef.value?.resetFields()
  emit('reset')
}

const handleTest = async () => {
  try {
    testing.value = true
    const valid = await formRef.value?.validate()
    if (valid) {
      emit('test', formData.value)
    }
  } catch (error) {
    console.error('Form validation failed:', error)
  } finally {
    testing.value = false
  }
}

const updateField = (field: string, value: any) => {
  const newData = { ...formData.value }
  const fieldParts = field.split('.')
  let current = newData

  for (let i = 0; i < fieldParts.length - 1; i++) {
    if (!current[fieldParts[i]]) {
      current[fieldParts[i]] = {}
    }
    current = current[fieldParts[i]]
  }

  current[fieldParts[fieldParts.length - 1]] = value
  formData.value = newData
  emit('field-change', field, value)
}

watch(() => props.modelValue, () => {
  // Validate form when model changes
  if (formRef.value) {
    formRef.value.validate((valid) => {
      isValid.value = valid
    })
  }
}, { deep: true })

defineExpose({
  formRef,
  updateField,
  validate: () => formRef.value?.validate(),
  resetFields: () => formRef.value?.resetFields(),
  clearValidate: () => formRef.value?.clearValidate()
})
</script>

<style scoped lang="scss">
.config-form {
  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 24px;
    padding-top: 24px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

:deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

:deep(.el-form-item__content) {
  flex-direction: column;
  align-items: stretch;
}

:deep(.el-form-item__error) {
  margin-top: 4px;
}
</style>