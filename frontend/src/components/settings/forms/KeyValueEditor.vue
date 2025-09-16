<template>
  <div class="key-value-editor">
    <div class="editor-header">
      <div class="header-info">
        <h4 v-if="title" class="editor-title">{{ title }}</h4>
        <p v-if="description" class="editor-description">{{ description }}</p>
      </div>
      <div class="header-actions">
        <el-button
          type="primary"
          size="small"
          :disabled="disabled"
          @click="addPair"
        >
          <el-icon><Plus /></el-icon>
          Add {{ itemName }}
        </el-button>
      </div>
    </div>

    <div class="pairs-container">
      <div v-if="pairs.length === 0" class="empty-state">
        <el-empty
          :description="`No ${itemName.toLowerCase()}s configured`"
          :image-size="80"
        >
          <el-button
            type="primary"
            :disabled="disabled"
            @click="addPair"
          >
            Add First {{ itemName }}
          </el-button>
        </el-empty>
      </div>

      <div v-else class="pairs-list">
        <div
          v-for="(pair, index) in pairs"
          :key="pair.id"
          class="pair-item"
        >
          <div class="pair-inputs">
            <div class="input-group">
              <label class="input-label">{{ keyLabel }}</label>
              <el-input
                v-model="pair.key"
                :placeholder="keyPlaceholder"
                :disabled="disabled"
                :size="size"
                @input="updatePair(index, 'key', pair.key)"
                @blur="validatePair(index)"
                :class="{ 'is-error': pair.keyError }"
              />
              <div v-if="pair.keyError" class="input-error">
                {{ pair.keyError }}
              </div>
            </div>

            <div class="input-separator">
              <el-icon><Right /></el-icon>
            </div>

            <div class="input-group">
              <label class="input-label">{{ valueLabel }}</label>
              <el-input
                v-if="valueType === 'text'"
                v-model="pair.value"
                :placeholder="valuePlaceholder"
                :disabled="disabled"
                :size="size"
                @input="updatePair(index, 'value', pair.value)"
                @blur="validatePair(index)"
                :class="{ 'is-error': pair.valueError }"
              />
              <el-input
                v-else-if="valueType === 'textarea'"
                v-model="pair.value"
                type="textarea"
                :placeholder="valuePlaceholder"
                :disabled="disabled"
                :size="size"
                :rows="2"
                @input="updatePair(index, 'value', pair.value)"
                @blur="validatePair(index)"
                :class="{ 'is-error': pair.valueError }"
              />
              <el-input-number
                v-else-if="valueType === 'number'"
                v-model="pair.value"
                :disabled="disabled"
                :size="size"
                :min="valueMin"
                :max="valueMax"
                @change="updatePair(index, 'value', pair.value)"
                :class="{ 'is-error': pair.valueError }"
              />
              <el-switch
                v-else-if="valueType === 'boolean'"
                v-model="pair.value"
                :disabled="disabled"
                :size="size"
                @change="updatePair(index, 'value', pair.value)"
              />
              <el-select
                v-else-if="valueType === 'select'"
                v-model="pair.value"
                :placeholder="valuePlaceholder"
                :disabled="disabled"
                :size="size"
                @change="updatePair(index, 'value', pair.value)"
                :class="{ 'is-error': pair.valueError }"
              >
                <el-option
                  v-for="option in valueOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
              <div v-if="pair.valueError" class="input-error">
                {{ pair.valueError }}
              </div>
            </div>
          </div>

          <div class="pair-actions">
            <el-tooltip content="Duplicate" placement="top">
              <el-button
                type="text"
                size="small"
                :disabled="disabled"
                @click="duplicatePair(index)"
              >
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </el-tooltip>

            <el-tooltip content="Remove" placement="top">
              <el-button
                type="text"
                size="small"
                :disabled="disabled"
                @click="removePair(index)"
                class="danger-button"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>

            <div class="drag-handle" v-if="sortable && !disabled">
              <el-icon><Operation /></el-icon>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Bulk Operations -->
    <div v-if="pairs.length > 0 && showBulkActions" class="bulk-actions">
      <el-dropdown @command="handleBulkAction">
        <el-button type="text" size="small">
          Bulk Actions
          <el-icon><ArrowDown /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="clear">
              Clear All
            </el-dropdown-item>
            <el-dropdown-item command="export">
              Export as JSON
            </el-dropdown-item>
            <el-dropdown-item command="import">
              Import from JSON
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

      <span class="pairs-count">
        {{ pairs.length }} {{ itemName.toLowerCase() }}{{ pairs.length !== 1 ? 's' : '' }}
      </span>
    </div>

    <!-- Import Dialog -->
    <el-dialog
      v-model="importDialogVisible"
      :title="`Import ${itemName}s`"
      width="600px"
    >
      <div class="import-content">
        <el-alert
          type="info"
          :closable="false"
          show-icon
          class="import-info"
        >
          Paste JSON data to import. This will replace all existing {{ itemName.toLowerCase() }}s.
        </el-alert>

        <el-input
          v-model="importData"
          type="textarea"
          :rows="10"
          placeholder='{"key1": "value1", "key2": "value2"}'
          class="import-textarea"
        />

        <div v-if="importError" class="import-error">
          {{ importError }}
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="importDialogVisible = false">Cancel</el-button>
          <el-button
            type="primary"
            :disabled="!importData.trim()"
            @click="confirmImport"
          >
            Import
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Plus,
  Right,
  CopyDocument,
  Delete,
  Operation,
  ArrowDown
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

interface KeyValuePair {
  id: string
  key: string
  value: any
  keyError?: string
  valueError?: string
}

interface Props {
  modelValue: Record<string, any>
  title?: string
  description?: string
  itemName?: string
  keyLabel?: string
  valueLabel?: string
  keyPlaceholder?: string
  valuePlaceholder?: string
  valueType?: 'text' | 'textarea' | 'number' | 'boolean' | 'select'
  valueOptions?: Array<{ label: string; value: any }>
  valueMin?: number
  valueMax?: number
  disabled?: boolean
  size?: 'large' | 'default' | 'small'
  sortable?: boolean
  showBulkActions?: boolean
  keyPattern?: RegExp
  valuePattern?: RegExp
  requiredFields?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: Record<string, any>): void
  (e: 'change', value: Record<string, any>): void
  (e: 'add', key: string, value: any): void
  (e: 'remove', key: string): void
  (e: 'duplicate', originalKey: string, newKey: string): void
}

const props = withDefaults(defineProps<Props>(), {
  itemName: 'Item',
  keyLabel: 'Key',
  valueLabel: 'Value',
  keyPlaceholder: 'Enter key',
  valuePlaceholder: 'Enter value',
  valueType: 'text',
  disabled: false,
  size: 'default',
  sortable: false,
  showBulkActions: true,
  requiredFields: false
})

const emit = defineEmits<Emits>()

const pairs = ref<KeyValuePair[]>([])
const importDialogVisible = ref(false)
const importData = ref('')
const importError = ref('')

const currentValue = computed({
  get: () => props.modelValue,
  set: (value) => {
    emit('update:modelValue', value)
    emit('change', value)
  }
})

// Convert object to pairs array
const objectToPairs = (obj: Record<string, any>): KeyValuePair[] => {
  return Object.entries(obj || {}).map(([key, value]) => ({
    id: generateId(),
    key,
    value
  }))
}

// Convert pairs array to object
const pairsToObject = (pairsList: KeyValuePair[]): Record<string, any> => {
  const obj: Record<string, any> = {}
  pairsList.forEach(pair => {
    if (pair.key.trim()) {
      obj[pair.key.trim()] = pair.value
    }
  })
  return obj
}

const generateId = (): string => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9)
}

const addPair = () => {
  const newPair: KeyValuePair = {
    id: generateId(),
    key: '',
    value: props.valueType === 'boolean' ? false :
           props.valueType === 'number' ? 0 : ''
  }

  pairs.value.push(newPair)
  updateValue()
  emit('add', newPair.key, newPair.value)
}

const removePair = async (index: number) => {
  const pair = pairs.value[index]

  try {
    await ElMessageBox.confirm(
      `Remove ${props.itemName.toLowerCase()} "${pair.key}"?`,
      'Confirm Removal',
      {
        confirmButtonText: 'Remove',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    pairs.value.splice(index, 1)
    updateValue()
    emit('remove', pair.key)
  } catch {
    // User cancelled
  }
}

const duplicatePair = (index: number) => {
  const originalPair = pairs.value[index]
  const newPair: KeyValuePair = {
    id: generateId(),
    key: `${originalPair.key}_copy`,
    value: originalPair.value
  }

  pairs.value.splice(index + 1, 0, newPair)
  updateValue()
  emit('duplicate', originalPair.key, newPair.key)
}

const updatePair = (index: number, field: 'key' | 'value', value: any) => {
  if (pairs.value[index]) {
    pairs.value[index][field] = value
    updateValue()
  }
}

const validatePair = (index: number) => {
  const pair = pairs.value[index]
  if (!pair) return

  // Clear previous errors
  pair.keyError = undefined
  pair.valueError = undefined

  // Validate key
  if (props.requiredFields && !pair.key.trim()) {
    pair.keyError = 'Key is required'
  } else if (props.keyPattern && !props.keyPattern.test(pair.key)) {
    pair.keyError = 'Invalid key format'
  } else {
    // Check for duplicate keys
    const duplicateIndex = pairs.value.findIndex((p, i) =>
      i !== index && p.key.trim() === pair.key.trim()
    )
    if (duplicateIndex !== -1) {
      pair.keyError = 'Duplicate key'
    }
  }

  // Validate value
  if (props.requiredFields &&
      props.valueType !== 'boolean' &&
      (pair.value === '' || pair.value === null || pair.value === undefined)) {
    pair.valueError = 'Value is required'
  } else if (props.valuePattern &&
             typeof pair.value === 'string' &&
             !props.valuePattern.test(pair.value)) {
    pair.valueError = 'Invalid value format'
  }
}

const updateValue = () => {
  currentValue.value = pairsToObject(pairs.value)
}

const handleBulkAction = async (command: string) => {
  switch (command) {
    case 'clear':
      await clearAll()
      break
    case 'export':
      exportData()
      break
    case 'import':
      importDialogVisible.value = true
      break
  }
}

const clearAll = async () => {
  try {
    await ElMessageBox.confirm(
      `Remove all ${props.itemName.toLowerCase()}s?`,
      'Clear All',
      {
        confirmButtonText: 'Clear All',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    pairs.value = []
    updateValue()
  } catch {
    // User cancelled
  }
}

const exportData = () => {
  const data = JSON.stringify(currentValue.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${props.itemName.toLowerCase()}_export.json`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)

  ElMessage.success(`${props.itemName}s exported successfully`)
}

const confirmImport = () => {
  try {
    importError.value = ''
    const data = JSON.parse(importData.value)

    if (typeof data !== 'object' || Array.isArray(data)) {
      throw new Error('Data must be a JSON object')
    }

    pairs.value = objectToPairs(data)
    updateValue()
    importDialogVisible.value = false
    importData.value = ''

    ElMessage.success(`${props.itemName}s imported successfully`)
  } catch (error) {
    importError.value = `Invalid JSON: ${(error as Error).message}`
  }
}

// Initialize pairs from modelValue
watch(() => props.modelValue, (newValue) => {
  pairs.value = objectToPairs(newValue)
}, { immediate: true, deep: true })

// Validate all pairs when they change
watch(pairs, () => {
  pairs.value.forEach((_, index) => validatePair(index))
}, { deep: true })
</script>

<style scoped lang="scss">
.key-value-editor {
  .editor-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 16px;
    gap: 16px;

    .header-info {
      flex: 1;

      .editor-title {
        margin: 0 0 4px 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .editor-description {
        margin: 0;
        font-size: 13px;
        color: var(--el-text-color-regular);
        line-height: 1.4;
      }
    }

    .header-actions {
      flex-shrink: 0;
    }
  }

  .pairs-container {
    .empty-state {
      text-align: center;
      padding: 40px 20px;
    }

    .pairs-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .pair-item {
      display: flex;
      align-items: flex-start;
      gap: 12px;
      padding: 16px;
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 8px;
      transition: all 0.2s;

      &:hover {
        border-color: var(--el-border-color);
        background: var(--el-fill-color-light);
      }

      .pair-inputs {
        flex: 1;
        display: flex;
        align-items: flex-start;
        gap: 16px;

        .input-group {
          flex: 1;
          min-width: 0;

          .input-label {
            display: block;
            font-size: 12px;
            font-weight: 500;
            color: var(--el-text-color-primary);
            margin-bottom: 6px;
          }

          .input-error {
            font-size: 12px;
            color: var(--el-color-danger);
            margin-top: 4px;
            line-height: 1.3;
          }

          :deep(.el-input.is-error .el-input__wrapper),
          :deep(.el-select.is-error .el-select__wrapper) {
            border-color: var(--el-color-danger);
          }
        }

        .input-separator {
          display: flex;
          align-items: center;
          padding-top: 28px;
          color: var(--el-text-color-placeholder);
          flex-shrink: 0;
        }
      }

      .pair-actions {
        display: flex;
        align-items: center;
        gap: 4px;
        flex-shrink: 0;
        padding-top: 28px;

        .danger-button {
          color: var(--el-color-danger);

          &:hover {
            background: var(--el-color-danger-light-9);
          }
        }

        .drag-handle {
          cursor: move;
          color: var(--el-text-color-placeholder);
          padding: 4px;

          &:hover {
            color: var(--el-text-color-regular);
          }
        }
      }
    }
  }

  .bulk-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid var(--el-border-color-lighter);

    .pairs-count {
      font-size: 12px;
      color: var(--el-text-color-regular);
    }
  }

  .import-content {
    .import-info {
      margin-bottom: 16px;
    }

    .import-textarea {
      margin-bottom: 12px;
    }

    .import-error {
      color: var(--el-color-danger);
      font-size: 12px;
      background: var(--el-color-danger-light-9);
      padding: 8px 12px;
      border-radius: 4px;
      border: 1px solid var(--el-color-danger-light-7);
    }
  }

  .dialog-footer {
    text-align: right;
  }
}

@media (max-width: 768px) {
  .key-value-editor {
    .editor-header {
      flex-direction: column;
      gap: 12px;

      .header-actions {
        width: 100%;
        text-align: right;
      }
    }

    .pair-item {
      .pair-inputs {
        flex-direction: column;
        gap: 12px;

        .input-separator {
          display: none;
        }
      }

      .pair-actions {
        flex-direction: column;
        padding-top: 0;
      }
    }
  }
}
</style>