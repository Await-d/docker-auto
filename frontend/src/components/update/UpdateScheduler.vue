<template>
  <el-dialog
    v-model="visible"
    title="Schedule Updates"
    width="800px"
    :before-close="handleClose"
  >
    <div class="scheduler-container">
      <!-- Selected Updates Summary -->
      <div v-if="selectedUpdates.length > 0" class="selected-summary">
        <h4>Selected Updates ({{ selectedUpdates.length }})</h4>
        <div class="updates-list">
          <div
            v-for="update in getSelectedUpdateDetails()"
            :key="update.id"
            class="update-item"
          >
            <div class="update-info">
              <span class="container-name">{{ update.containerName }}</span>
              <span class="version-change">
                {{ update.currentVersion }} → {{ update.availableVersion }}
              </span>
            </div>
            <el-tag
              :type="getRiskLevelType(update.riskLevel)"
              size="small"
            >
              {{ update.riskLevel }}
            </el-tag>
          </div>
        </div>
      </div>

      <!-- Schedule Configuration -->
      <div class="schedule-config">
        <el-form
          ref="formRef"
          :model="scheduleForm"
          :rules="formRules"
          label-width="140px"
        >
          <!-- Schedule Type -->
          <el-form-item label="Schedule Type" prop="scheduleType">
            <el-radio-group v-model="scheduleForm.scheduleType">
              <el-radio label="once">One-time</el-radio>
              <el-radio label="recurring">Recurring</el-radio>
            </el-radio-group>
          </el-form-item>

          <!-- Date and Time -->
          <el-form-item
            v-if="scheduleForm.scheduleType === 'once'"
            label="Date & Time"
            prop="scheduledAt"
          >
            <el-date-picker
              v-model="scheduleForm.scheduledAt"
              type="datetime"
              placeholder="Select date and time"
              format="YYYY-MM-DD HH:mm"
              value-format="YYYY-MM-DD HH:mm:ss"
              :disabled-date="disabledDate"
              style="width: 100%"
            />
          </el-form-item>

          <!-- Recurring Pattern -->
          <el-form-item
            v-if="scheduleForm.scheduleType === 'recurring'"
            label="Pattern"
            prop="recurringPattern"
          >
            <div class="recurring-options">
              <el-select
                v-model="recurringType"
                placeholder="Select pattern"
                style="width: 150px"
                @change="updateRecurringPattern"
              >
                <el-option label="Daily" value="daily" />
                <el-option label="Weekly" value="weekly" />
                <el-option label="Monthly" value="monthly" />
                <el-option label="Custom" value="custom" />
              </el-select>

              <!-- Daily Options -->
              <div v-if="recurringType === 'daily'" class="pattern-details">
                <el-time-picker
                  v-model="dailyTime"
                  placeholder="Select time"
                  format="HH:mm"
                  @change="updateRecurringPattern"
                />
              </div>

              <!-- Weekly Options -->
              <div v-if="recurringType === 'weekly'" class="pattern-details">
                <el-select
                  v-model="weeklyDays"
                  multiple
                  placeholder="Select days"
                  style="width: 200px"
                  @change="updateRecurringPattern"
                >
                  <el-option label="Monday" value="1" />
                  <el-option label="Tuesday" value="2" />
                  <el-option label="Wednesday" value="3" />
                  <el-option label="Thursday" value="4" />
                  <el-option label="Friday" value="5" />
                  <el-option label="Saturday" value="6" />
                  <el-option label="Sunday" value="0" />
                </el-select>
                <el-time-picker
                  v-model="weeklyTime"
                  placeholder="Select time"
                  format="HH:mm"
                  @change="updateRecurringPattern"
                />
              </div>

              <!-- Custom Cron -->
              <div v-if="recurringType === 'custom'" class="pattern-details">
                <el-input
                  v-model="scheduleForm.recurringPattern"
                  placeholder="0 2 * * 0 (Every Sunday at 2:00 AM)"
                  @input="validateCronExpression"
                />
                <div class="cron-help">
                  <el-tooltip content="Cron expression format: minute hour day month day-of-week">
                    <el-button text type="primary" size="small">
                      <el-icon><QuestionFilled /></el-icon>
                      Cron Help
                    </el-button>
                  </el-tooltip>
                  <span v-if="cronDescription" class="cron-description">
                    {{ cronDescription }}
                  </span>
                </div>
              </div>
            </div>
          </el-form-item>

          <!-- Timezone -->
          <el-form-item label="Timezone" prop="timezone">
            <el-select
              v-model="scheduleForm.timezone"
              filterable
              placeholder="Select timezone"
              style="width: 100%"
            >
              <el-option
                v-for="tz in timezones"
                :key="tz.value"
                :label="tz.label"
                :value="tz.value"
              />
            </el-select>
          </el-form-item>

          <!-- Update Strategy -->
          <el-form-item label="Update Strategy" prop="strategy">
            <el-select v-model="scheduleForm.strategy" style="width: 100%">
              <el-option label="Recreate" value="recreate">
                <div class="strategy-option">
                  <span>Recreate</span>
                  <small>Stop container, pull image, create new container</small>
                </div>
              </el-option>
              <el-option label="Rolling Update" value="rolling">
                <div class="strategy-option">
                  <span>Rolling Update</span>
                  <small>Zero-downtime update using load balancer</small>
                </div>
              </el-option>
              <el-option label="Blue-Green" value="blue-green">
                <div class="strategy-option">
                  <span>Blue-Green</span>
                  <small>Deploy alongside existing, switch when ready</small>
                </div>
              </el-option>
              <el-option label="Canary" value="canary">
                <div class="strategy-option">
                  <span>Canary</span>
                  <small>Gradual rollout with monitoring</small>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <!-- Advanced Options -->
          <el-form-item>
            <el-checkbox v-model="scheduleForm.rollbackOnFailure">
              Rollback automatically on failure
            </el-checkbox>
          </el-form-item>

          <el-form-item>
            <el-checkbox v-model="scheduleForm.runTests">
              Run tests before update
            </el-checkbox>
          </el-form-item>

          <!-- Notification Settings -->
          <el-form-item label="Notifications">
            <div class="notification-settings">
              <el-checkbox v-model="scheduleForm.notifications.enabled">
                Enable notifications
              </el-checkbox>

              <div v-if="scheduleForm.notifications.enabled" class="notification-options">
                <el-form-item label="Notify Before">
                  <el-select
                    v-model="scheduleForm.notifyBefore"
                    style="width: 150px"
                  >
                    <el-option label="5 minutes" :value="300000" />
                    <el-option label="15 minutes" :value="900000" />
                    <el-option label="30 minutes" :value="1800000" />
                    <el-option label="1 hour" :value="3600000" />
                    <el-option label="2 hours" :value="7200000" />
                  </el-select>
                </el-form-item>

                <el-checkbox-group v-model="scheduleForm.notifications.events">
                  <el-checkbox label="update_started">Update started</el-checkbox>
                  <el-checkbox label="update_completed">Update completed</el-checkbox>
                  <el-checkbox label="update_failed">Update failed</el-checkbox>
                  <el-checkbox label="rollback_started">Rollback started</el-checkbox>
                </el-checkbox-group>
              </div>
            </div>
          </el-form-item>

          <!-- Dependencies -->
          <el-form-item v-if="hasDependencies" label="Dependencies">
            <div class="dependencies-info">
              <el-alert
                title="Some containers have dependencies"
                type="warning"
                :closable="false"
                show-icon
              />
              <el-checkbox v-model="scheduleForm.respectDependencies">
                Update containers in dependency order
              </el-checkbox>
              <el-select
                v-model="scheduleForm.dependencyStrategy"
                style="width: 200px"
                :disabled="!scheduleForm.respectDependencies"
              >
                <el-option label="Strict Order" value="strict" />
                <el-option label="Loose Order" value="loose" />
                <el-option label="Ignore Dependencies" value="ignore" />
              </el-select>
            </div>
          </el-form-item>

          <!-- Preview -->
          <el-form-item label="Preview">
            <div class="schedule-preview">
              <div class="preview-item">
                <strong>Next Execution:</strong>
                <span>{{ getNextExecutionTime() }}</span>
              </div>
              <div v-if="scheduleForm.scheduleType === 'recurring'" class="preview-item">
                <strong>Pattern:</strong>
                <span>{{ getPatternDescription() }}</span>
              </div>
              <div class="preview-item">
                <strong>Estimated Duration:</strong>
                <span>{{ getEstimatedDuration() }}</span>
              </div>
            </div>
          </el-form-item>
        </el-form>
      </div>

      <!-- Calendar View -->
      <div v-if="scheduleForm.scheduleType === 'once'" class="calendar-view">
        <h4>Calendar</h4>
        <el-calendar v-model="calendarValue">
          <template #date-cell="{ data }">
            <div class="calendar-day">
              <span>{{ data.day.split('-').slice(-1)[0] }}</span>
              <div v-if="hasScheduledUpdates(data.day)" class="scheduled-indicator">
                <el-badge :value="getScheduledCount(data.day)" class="scheduled-badge" />
              </div>
            </div>
          </template>
        </el-calendar>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">Cancel</el-button>
        <el-button
          type="primary"
          :loading="scheduling"
          :disabled="!isFormValid"
          @click="handleSchedule"
        >
          Schedule Update{{ selectedUpdates.length > 1 ? 's' : '' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { QuestionFilled } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'

// Store
import { useUpdatesStore } from '@/store/updates'

// Types
import type { UpdateStrategy, NotificationSettings, ContainerUpdate } from '@/types/updates'

// Props
interface Props {
  modelValue: boolean
  selectedUpdates: string[]
}

const props = defineProps<Props>()

// Emits
defineEmits<{
  'update:modelValue': [value: boolean]
  scheduled: []
}>()

// Store
const updatesStore = useUpdatesStore()

// Local state
const formRef = ref<FormInstance>()
const scheduling = ref(false)
const recurringType = ref('daily')
const dailyTime = ref(new Date())
const weeklyDays = ref(['1'])
const weeklyTime = ref(new Date())
const calendarValue = ref(new Date())
const cronDescription = ref('')

const scheduleForm = ref({
  scheduleType: 'once' as 'once' | 'recurring',
  scheduledAt: '',
  recurringPattern: '',
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  strategy: 'recreate' as UpdateStrategy,
  rollbackOnFailure: true,
  runTests: false,
  notifyBefore: 300000, // 5 minutes
  respectDependencies: true,
  dependencyStrategy: 'strict' as 'strict' | 'loose' | 'ignore',
  notifications: {
    enabled: true,
    events: ['update_started', 'update_completed', 'update_failed']
  } as NotificationSettings
})

// Computed
const visible = computed({
  get: () => props.modelValue,
  set: (value) => $emit('update:modelValue', value)
})

const timezones = computed(() => [
  { label: 'UTC', value: 'UTC' },
  { label: 'America/New_York (EST/EDT)', value: 'America/New_York' },
  { label: 'America/Chicago (CST/CDT)', value: 'America/Chicago' },
  { label: 'America/Denver (MST/MDT)', value: 'America/Denver' },
  { label: 'America/Los_Angeles (PST/PDT)', value: 'America/Los_Angeles' },
  { label: 'Europe/London (GMT/BST)', value: 'Europe/London' },
  { label: 'Europe/Berlin (CET/CEST)', value: 'Europe/Berlin' },
  { label: 'Asia/Tokyo (JST)', value: 'Asia/Tokyo' },
  { label: 'Asia/Shanghai (CST)', value: 'Asia/Shanghai' },
  { label: 'Australia/Sydney (AEDT/AEST)', value: 'Australia/Sydney' }
])

const hasDependencies = computed(() => {
  const selectedUpdateDetails = getSelectedUpdateDetails()
  return selectedUpdateDetails.some(update =>
    update.dependencies.length > 0 || update.conflicts.length > 0
  )
})

const isFormValid = computed(() => {
  if (scheduleForm.value.scheduleType === 'once') {
    return !!scheduleForm.value.scheduledAt
  } else {
    return !!scheduleForm.value.recurringPattern
  }
})

// Form validation rules
const formRules: FormRules = {
  scheduledAt: [
    { required: true, message: 'Please select a date and time', trigger: 'blur' }
  ],
  recurringPattern: [
    { required: true, message: 'Please set a recurring pattern', trigger: 'blur' }
  ],
  strategy: [
    { required: true, message: 'Please select an update strategy', trigger: 'change' }
  ]
}

// Methods
const getSelectedUpdateDetails = (): ContainerUpdate[] => {
  return props.selectedUpdates.map(id =>
    updatesStore.availableUpdates.find(u => u.id === id)
  ).filter(Boolean) as ContainerUpdate[]
}

const getRiskLevelType = (riskLevel: string) => {
  switch (riskLevel) {
    case 'low': return 'success'
    case 'medium': return 'warning'
    case 'high': return 'danger'
    case 'critical': return 'danger'
    default: return 'info'
  }
}

const disabledDate = (time: Date) => {
  return time.getTime() < Date.now() - 24 * 60 * 60 * 1000
}

const updateRecurringPattern = () => {
  if (recurringType.value === 'daily') {
    const time = dailyTime.value
    const hour = time.getHours()
    const minute = time.getMinutes()
    scheduleForm.value.recurringPattern = `${minute} ${hour} * * *`
  } else if (recurringType.value === 'weekly') {
    const time = weeklyTime.value
    const hour = time.getHours()
    const minute = time.getMinutes()
    const days = weeklyDays.value.join(',')
    scheduleForm.value.recurringPattern = `${minute} ${hour} * * ${days}`
  }
  updateCronDescription()
}

const validateCronExpression = () => {
  // Basic cron validation
  const pattern = scheduleForm.value.recurringPattern
  const parts = pattern.split(' ')

  if (parts.length !== 5) {
    cronDescription.value = 'Invalid cron expression'
    return false
  }

  updateCronDescription()
  return true
}

const updateCronDescription = () => {
  const pattern = scheduleForm.value.recurringPattern
  if (!pattern) return

  try {
    // This is a simplified description generator
    // In a real app, you'd use a cron parser library
    cronDescription.value = parseCronExpression(pattern)
  } catch (error) {
    cronDescription.value = 'Invalid cron expression'
  }
}

const parseCronExpression = (pattern: string): string => {
  const parts = pattern.split(' ')
  const [minute, hour, day, month, dayOfWeek] = parts

  let description = 'Runs '

  if (dayOfWeek !== '*') {
    const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
    const selectedDays = dayOfWeek.split(',').map(d => days[parseInt(d)])
    description += `every ${selectedDays.join(', ')} `
  } else if (day !== '*') {
    description += `on day ${day} of each month `
  } else {
    description += 'daily '
  }

  if (hour !== '*' && minute !== '*') {
    description += `at ${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`
  }

  return description
}

const getNextExecutionTime = () => {
  if (scheduleForm.value.scheduleType === 'once' && scheduleForm.value.scheduledAt) {
    return new Date(scheduleForm.value.scheduledAt).toLocaleString()
  } else if (scheduleForm.value.scheduleType === 'recurring' && scheduleForm.value.recurringPattern) {
    // Calculate next execution based on cron pattern
    // This is simplified - in reality you'd use a cron library
    return calculateNextCronExecution(scheduleForm.value.recurringPattern)
  }
  return 'Not configured'
}

const calculateNextCronExecution = (pattern: string): string => {
  // Simplified next execution calculation
  // In a real app, use a proper cron library
  const now = new Date()
  const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000)
  return tomorrow.toLocaleString()
}

const getPatternDescription = () => {
  if (scheduleForm.value.recurringPattern) {
    return cronDescription.value || scheduleForm.value.recurringPattern
  }
  return 'Not configured'
}

const getEstimatedDuration = () => {
  const selectedUpdateDetails = getSelectedUpdateDetails()
  const totalTime = selectedUpdateDetails.reduce((sum, update) => sum + update.estimatedDowntime, 0)

  if (totalTime < 60) return `~${totalTime}s`
  if (totalTime < 3600) return `~${Math.floor(totalTime / 60)}m`
  return `~${Math.floor(totalTime / 3600)}h ${Math.floor((totalTime % 3600) / 60)}m`
}

const hasScheduledUpdates = (date: string): boolean => {
  // Check if there are scheduled updates on this date
  // This would come from the store in a real app
  return false
}

const getScheduledCount = (date: string): number => {
  // Return count of scheduled updates on this date
  return 0
}

const handleSchedule = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()

    scheduling.value = true

    // Schedule each selected update
    const promises = props.selectedUpdates.map(updateId => {
      const scheduledAt = scheduleForm.value.scheduleType === 'once'
        ? new Date(scheduleForm.value.scheduledAt)
        : new Date() // For recurring, calculate next execution

      return updatesStore.scheduleUpdate(updateId, scheduledAt, {
        recurring: scheduleForm.value.scheduleType === 'recurring',
        recurringPattern: scheduleForm.value.recurringPattern,
        notifyBefore: scheduleForm.value.notifyBefore
      })
    })

    await Promise.all(promises)

    ElMessage.success(`Successfully scheduled ${props.selectedUpdates.length} update(s)`)
    $emit('scheduled')
    handleClose()

  } catch (error) {
    console.error('Failed to schedule updates:', error)
    ElMessage.error('Failed to schedule updates')
  } finally {
    scheduling.value = false
  }
}

const handleClose = () => {
  visible.value = false
}

// Watch for changes to update cron description
watch(() => scheduleForm.value.recurringPattern, updateCronDescription)

// Initialize daily time to 2 AM
dailyTime.value.setHours(2, 0, 0, 0)
weeklyTime.value.setHours(2, 0, 0, 0)

// Set default recurring pattern
updateRecurringPattern()
</script>

<style scoped lang="scss">
.scheduler-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

.selected-summary {
  padding: 16px;
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;

  h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .updates-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 150px;
    overflow-y: auto;
  }

  .update-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;

    .update-info {
      display: flex;
      flex-direction: column;
      gap: 2px;

      .container-name {
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .version-change {
        font-size: 12px;
        color: var(--el-text-color-regular);
        font-family: monospace;
      }
    }
  }
}

.schedule-config {
  .recurring-options {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;

    .pattern-details {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .cron-help {
      display: flex;
      align-items: center;
      gap: 8px;

      .cron-description {
        font-size: 12px;
        color: var(--el-text-color-regular);
        font-style: italic;
      }
    }
  }

  .strategy-option {
    display: flex;
    flex-direction: column;

    small {
      color: var(--el-text-color-regular);
      font-size: 11px;
    }
  }

  .notification-settings {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .notification-options {
      margin-left: 24px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
  }

  .dependencies-info {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .schedule-preview {
    padding: 12px;
    background: var(--el-bg-color-page);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;

    .preview-item {
      display: flex;
      justify-content: space-between;
      margin-bottom: 8px;

      &:last-child {
        margin-bottom: 0;
      }

      strong {
        color: var(--el-text-color-primary);
      }

      span {
        color: var(--el-text-color-regular);
      }
    }
  }
}

.calendar-view {
  h4 {
    margin: 0 0 16px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .calendar-day {
    position: relative;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;

    .scheduled-indicator {
      position: absolute;
      top: 4px;
      right: 4px;

      .scheduled-badge {
        .el-badge__content {
          font-size: 10px;
          min-width: 16px;
          height: 16px;
          line-height: 16px;
        }
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 768px) {
  .scheduler-container {
    max-height: 60vh;
  }

  .recurring-options {
    flex-direction: column;
    align-items: stretch;

    .pattern-details {
      justify-content: center;
    }
  }

  .schedule-preview {
    .preview-item {
      flex-direction: column;
      gap: 4px;
    }
  }
}
</style>