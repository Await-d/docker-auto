/**
 * Settings store for system configuration management
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { request } from '@/utils/request'
import { useApp } from '@/store/app'

// Type definitions for different settings sections
export interface GeneralSettings {
  systemName: string
  systemDescription: string
  timezone: string
  language: string
  dateFormat: string
  timeFormat: string
  sessionTimeout: number
}

export interface DockerSettings {
  socketPath: string
  connectionTimeout: number
  tlsEnabled: boolean
  tlsCert?: string
  tlsKey?: string
  tlsCa?: string
  defaultRestartPolicy: string
  defaultCpuLimit?: number
  defaultMemoryLimit?: number
  defaultNetworkMode: string
  imagePullPolicy: string
  imageCleanupSchedule: string
}

export interface UpdateSettings {
  defaultStrategy: 'auto' | 'manual'
  maintenanceWindows: MaintenanceWindow[]
  maxConcurrentUpdates: number
  retryAttempts: number
  retryDelay: number
  rollbackTimeout: number
  semanticVersioning: boolean
  allowPrerelease: boolean
  securityUpdatePriority: boolean
  notifyOnAvailable: boolean
  notifyOnComplete: boolean
  notifyOnFailure: boolean
}

export interface MaintenanceWindow {
  id: string
  name: string
  dayOfWeek: number[]
  startTime: string
  endTime: string
  timezone: string
  enabled: boolean
}

export interface RegistrySettings {
  defaultRegistry: string
  registries: DockerRegistry[]
  searchLimit: number
  trustPolicy: 'always' | 'signed' | 'never'
  securityScanEnabled: boolean
}

export interface DockerRegistry {
  id: string
  name: string
  url: string
  type: 'dockerhub' | 'harbor' | 'ecr' | 'acr' | 'gcr' | 'generic'
  username?: string
  password?: string
  accessToken?: string
  isDefault: boolean
  healthCheckInterval: number
  rateLimit?: number
  enabled: boolean
}

export interface UserSettings {
  passwordPolicy: PasswordPolicy
  sessionPolicy: SessionPolicy
  roles: UserRole[]
  defaultRole: string
  jwtExpiration: number
  refreshTokenExpiration: number
  twoFactorEnabled: boolean
  accountLockoutEnabled: boolean
  maxLoginAttempts: number
  lockoutDuration: number
}

export interface PasswordPolicy {
  minLength: number
  requireUppercase: boolean
  requireLowercase: boolean
  requireNumbers: boolean
  requireSpecialChars: boolean
  maxAge: number
  preventReuse: number
}

export interface SessionPolicy {
  maxConcurrentSessions: number
  idleTimeout: number
  absoluteTimeout: number
  requireReauth: boolean
}

export interface UserRole {
  id: string
  name: string
  description: string
  permissions: string[]
  isSystem: boolean
  isDefault: boolean
}

export interface NotificationSettings {
  channels: NotificationChannel[]
  rules: NotificationRule[]
  templates: NotificationTemplate[]
  rateLimiting: RateLimitConfig
  quietHours: QuietHours
}

export interface NotificationChannel {
  id: string
  type: 'email' | 'webhook' | 'slack' | 'discord' | 'teams'
  name: string
  config: Record<string, any>
  enabled: boolean
  testEndpoint?: string
}

export interface NotificationRule {
  id: string
  name: string
  events: string[]
  severity: 'low' | 'medium' | 'high' | 'critical'
  channels: string[]
  userGroups: string[]
  enabled: boolean
}

export interface NotificationTemplate {
  id: string
  name: string
  type: 'email' | 'webhook' | 'slack' | 'discord' | 'teams'
  subject: string
  body: string
  variables: string[]
}

export interface RateLimitConfig {
  enabled: boolean
  maxPerMinute: number
  maxPerHour: number
  maxPerDay: number
}

export interface QuietHours {
  enabled: boolean
  startTime: string
  endTime: string
  timezone: string
  days: number[]
}

export interface SchedulerSettings {
  maxConcurrentTasks: number
  defaultTimeout: number
  retryPolicy: RetryPolicy
  deadLetterQueueEnabled: boolean
  timezone: string
  taskTemplates: TaskTemplate[]
  performanceMonitoring: PerformanceConfig
}

export interface RetryPolicy {
  enabled: boolean
  maxAttempts: number
  backoffStrategy: 'linear' | 'exponential' | 'fixed'
  initialDelay: number
  maxDelay: number
  multiplier: number
}

export interface TaskTemplate {
  id: string
  name: string
  description: string
  cronExpression: string
  command: string
  timeout: number
  retryPolicy: RetryPolicy
  dependencies: string[]
  resourceLimits: ResourceLimits
}

export interface ResourceLimits {
  cpuLimit?: number
  memoryLimit?: number
  diskLimit?: number
}

export interface PerformanceConfig {
  metricsRetention: number
  alertThresholds: AlertThresholds
  healthCheckInterval: number
}

export interface AlertThresholds {
  taskFailureRate: number
  avgExecutionTime: number
  queueSize: number
  resourceUsage: number
}

export interface SecuritySettings {
  accessControl: AccessControlConfig
  auditSettings: AuditConfig
  encryption: EncryptionConfig
  apiSecurity: ApiSecurityConfig
}

export interface AccessControlConfig {
  ipWhitelist: string[]
  ipBlacklist: string[]
  corsPolicy: CorsPolicy
  rateLimiting: RateLimitConfig
}

export interface CorsPolicy {
  enabled: boolean
  allowedOrigins: string[]
  allowedMethods: string[]
  allowedHeaders: string[]
  allowCredentials: boolean
  maxAge: number
}

export interface AuditConfig {
  enabled: boolean
  retentionDays: number
  eventFilters: string[]
  exportEnabled: boolean
  complianceReporting: boolean
}

export interface EncryptionConfig {
  algorithm: string
  keyRotationEnabled: boolean
  keyRotationInterval: number
  certificateAutoRenewal: boolean
}

export interface ApiSecurityConfig {
  apiKeysEnabled: boolean
  apiKeyExpiration: number
  requestSigning: boolean
  webhookVerification: boolean
}

export interface MonitoringSettings {
  systemMonitoring: SystemMonitoringConfig
  logging: LoggingConfig
  externalIntegrations: ExternalIntegration[]
}

export interface SystemMonitoringConfig {
  resourceThresholds: ResourceThresholds
  healthCheckInterval: number
  alertingEnabled: boolean
  metricsCollection: MetricsCollection
}

export interface ResourceThresholds {
  cpuWarning: number
  cpuCritical: number
  memoryWarning: number
  memoryCritical: number
  diskWarning: number
  diskCritical: number
  networkWarning: number
  networkCritical: number
}

export interface MetricsCollection {
  enabled: boolean
  interval: number
  retention: number
  aggregation: string[]
}

export interface LoggingConfig {
  level: 'debug' | 'info' | 'warn' | 'error'
  components: Record<string, string>
  rotation: LogRotationConfig
  export: LogExportConfig
  structured: boolean
}

export interface LogRotationConfig {
  maxSize: number
  maxFiles: number
  maxAge: number
  compress: boolean
}

export interface LogExportConfig {
  enabled: boolean
  destination: string
  format: 'json' | 'text' | 'syslog'
  filters: string[]
}

export interface ExternalIntegration {
  id: string
  type: 'prometheus' | 'grafana' | 'elasticsearch' | 'splunk' | 'snmp'
  name: string
  config: Record<string, any>
  enabled: boolean
}

export interface SystemSettings {
  general: GeneralSettings
  docker: DockerSettings
  updates: UpdateSettings
  registries: RegistrySettings
  users: UserSettings
  notifications: NotificationSettings
  scheduler: SchedulerSettings
  security: SecuritySettings
  monitoring: MonitoringSettings
}

export interface SettingsSection {
  title: string
  description: string
  icon: string
  key: keyof SystemSettings
  permissions: string[]
  isDirty: boolean
  isValid: boolean
  hasChanges: boolean
}

export const useSettingsStore = defineStore('settings', () => {
  const app = useApp()

  // State
  const settings = ref<SystemSettings | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const currentSection = ref<string>('general')
  const searchQuery = ref('')
  const dirtyFields = ref<Set<string>>(new Set())
  const validationErrors = ref<Record<string, string[]>>({})

  // Settings backup for reset functionality
  const originalSettings = ref<SystemSettings | null>(null)
  const sectionBackups = ref<Record<string, any>>({})

  // Computed
  const isDirty = computed(() => dirtyFields.value.size > 0)

  const hasValidationErrors = computed(() =>
    Object.keys(validationErrors.value).length > 0
  )

  const canSave = computed(() =>
    isDirty.value && !hasValidationErrors.value && !saving.value
  )

  const filteredSections = computed(() => {
    if (!searchQuery.value.trim()) return settingsSections.value

    const query = searchQuery.value.toLowerCase()
    return settingsSections.value.filter(section =>
      section.title.toLowerCase().includes(query) ||
      section.description.toLowerCase().includes(query)
    )
  })

  const settingsSections = computed((): SettingsSection[] => [
    {
      title: 'System Configuration',
      description: 'General system settings and configuration',
      icon: 'Setting',
      key: 'general',
      permissions: ['settings:general:read'],
      isDirty: isDirtySection('general'),
      isValid: isValidSection('general'),
      hasChanges: hasChangesInSection('general')
    },
    {
      title: 'Docker Configuration',
      description: 'Docker connection and container settings',
      icon: 'Box',
      key: 'docker',
      permissions: ['settings:docker:read'],
      isDirty: isDirtySection('docker'),
      isValid: isValidSection('docker'),
      hasChanges: hasChangesInSection('docker')
    },
    {
      title: 'Update Policies',
      description: 'Container update strategies and schedules',
      icon: 'Refresh',
      key: 'updates',
      permissions: ['settings:updates:read'],
      isDirty: isDirtySection('updates'),
      isValid: isValidSection('updates'),
      hasChanges: hasChangesInSection('updates')
    },
    {
      title: 'Registry Management',
      description: 'Container registry connections and settings',
      icon: 'CloudUpload',
      key: 'registries',
      permissions: ['settings:registries:read'],
      isDirty: isDirtySection('registries'),
      isValid: isValidSection('registries'),
      hasChanges: hasChangesInSection('registries')
    },
    {
      title: 'User Management',
      description: 'User accounts, roles and permissions',
      icon: 'User',
      key: 'users',
      permissions: ['settings:users:read', 'admin'],
      isDirty: isDirtySection('users'),
      isValid: isValidSection('users'),
      hasChanges: hasChangesInSection('users')
    },
    {
      title: 'Notifications',
      description: 'Notification channels and rules',
      icon: 'Bell',
      key: 'notifications',
      permissions: ['settings:notifications:read'],
      isDirty: isDirtySection('notifications'),
      isValid: isValidSection('notifications'),
      hasChanges: hasChangesInSection('notifications')
    },
    {
      title: 'Scheduler',
      description: 'Task scheduling and execution settings',
      icon: 'Timer',
      key: 'scheduler',
      permissions: ['settings:scheduler:read'],
      isDirty: isDirtySection('scheduler'),
      isValid: isValidSection('scheduler'),
      hasChanges: hasChangesInSection('scheduler')
    },
    {
      title: 'Security',
      description: 'Security policies and access controls',
      icon: 'Lock',
      key: 'security',
      permissions: ['settings:security:read', 'admin'],
      isDirty: isDirtySection('security'),
      isValid: isValidSection('security'),
      hasChanges: hasChangesInSection('security')
    },
    {
      title: 'Monitoring',
      description: 'System monitoring and logging configuration',
      icon: 'Monitor',
      key: 'monitoring',
      permissions: ['settings:monitoring:read'],
      isDirty: isDirtySection('monitoring'),
      isValid: isValidSection('monitoring'),
      hasChanges: hasChangesInSection('monitoring')
    }
  ])

  // Helper functions for computed properties
  function isDirtySection(section: string): boolean {
    return Array.from(dirtyFields.value).some(field => field.startsWith(`${section}.`))
  }

  function isValidSection(section: string): boolean {
    return !Object.keys(validationErrors.value).some(field => field.startsWith(`${section}.`))
  }

  function hasChangesInSection(section: string): boolean {
    if (!settings.value || !originalSettings.value) return false
    return JSON.stringify(settings.value[section as keyof SystemSettings]) !==
           JSON.stringify(originalSettings.value[section as keyof SystemSettings])
  }

  // Actions
  const loadSettings = async () => {
    loading.value = true
    try {
      const response = await request.get('/api/system/config')
      settings.value = response.data
      originalSettings.value = JSON.parse(JSON.stringify(response.data))
      dirtyFields.value.clear()
      validationErrors.value = {}
    } catch (error) {
      app.handleError(error as Error, 'Settings')
      throw error
    } finally {
      loading.value = false
    }
  }

  const saveSettings = async (section?: keyof SystemSettings) => {
    if (!settings.value) return

    saving.value = true
    try {
      let payload = settings.value

      if (section) {
        payload = { [section]: settings.value[section] } as any
      }

      await request.put('/api/system/config', payload)

      // Update original settings to reflect saved state
      if (section) {
        if (originalSettings.value) {
          originalSettings.value[section] = JSON.parse(JSON.stringify(settings.value[section]))
        }
        // Clear dirty fields for this section
        const sectionFields = Array.from(dirtyFields.value).filter(field => field.startsWith(`${section}.`))
        sectionFields.forEach(field => dirtyFields.value.delete(field))
      } else {
        originalSettings.value = JSON.parse(JSON.stringify(settings.value))
        dirtyFields.value.clear()
      }

      app.showSuccess(`Settings ${section ? `for ${section}` : ''} saved successfully`)
    } catch (error) {
      app.handleError(error as Error, 'Settings Save')
      throw error
    } finally {
      saving.value = false
    }
  }

  const resetSettings = async (section?: keyof SystemSettings) => {
    if (!originalSettings.value) return

    if (section) {
      if (settings.value) {
        settings.value[section] = JSON.parse(JSON.stringify(originalSettings.value[section]))

        // Clear dirty fields for this section
        const sectionFields = Array.from(dirtyFields.value).filter(field => field.startsWith(`${section}.`))
        sectionFields.forEach(field => dirtyFields.value.delete(field))

        // Clear validation errors for this section
        Object.keys(validationErrors.value).forEach(field => {
          if (field.startsWith(`${section}.`)) {
            delete validationErrors.value[field]
          }
        })
      }
    } else {
      settings.value = JSON.parse(JSON.stringify(originalSettings.value))
      dirtyFields.value.clear()
      validationErrors.value = {}
    }

    app.showInfo(`Settings ${section ? `for ${section}` : ''} reset to saved values`)
  }

  const testConfiguration = async (section: keyof SystemSettings, config: any) => {
    try {
      const response = await request.post(`/api/system/config/test/${section}`, config)
      return response.data
    } catch (error) {
      app.handleError(error as Error, 'Configuration Test')
      throw error
    }
  }

  const exportSettings = async (sections?: (keyof SystemSettings)[]) => {
    try {
      const payload = sections ? { sections } : {}
      const response = await request.post('/api/system/config/export', payload, {
        responseType: 'blob'
      })

      // Create download link
      const blob = new Blob([response.data], { type: 'application/json' })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `docker-auto-settings-${new Date().toISOString().split('T')[0]}.json`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      app.showSuccess('Settings exported successfully')
    } catch (error) {
      app.handleError(error as Error, 'Settings Export')
      throw error
    }
  }

  const importSettings = async (file: File) => {
    try {
      const formData = new FormData()
      formData.append('file', file)

      const response = await request.post('/api/system/config/import', formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })

      // Reload settings after import
      await loadSettings()

      app.showSuccess('Settings imported successfully')
      return response.data
    } catch (error) {
      app.handleError(error as Error, 'Settings Import')
      throw error
    }
  }

  const updateField = (field: string, value: any) => {
    if (!settings.value) return

    // Update the nested field
    const fieldParts = field.split('.')
    let current: any = settings.value

    for (let i = 0; i < fieldParts.length - 1; i++) {
      if (!current[fieldParts[i]]) {
        current[fieldParts[i]] = {}
      }
      current = current[fieldParts[i]]
    }

    current[fieldParts[fieldParts.length - 1]] = value

    // Mark field as dirty
    dirtyFields.value.add(field)

    // Clear validation error for this field
    if (validationErrors.value[field]) {
      delete validationErrors.value[field]
    }
  }

  const validateField = (field: string, value: any): string[] => {
    const errors: string[] = []

    // Add validation logic based on field type and constraints
    // This is a simplified version - you would implement full validation
    if (field.includes('email') && value && !isValidEmail(value)) {
      errors.push('Invalid email format')
    }

    if (field.includes('url') && value && !isValidUrl(value)) {
      errors.push('Invalid URL format')
    }

    if (field.includes('port') && value && (!Number.isInteger(value) || value < 1 || value > 65535)) {
      errors.push('Port must be between 1 and 65535')
    }

    return errors
  }

  const setValidationErrors = (field: string, errors: string[]) => {
    if (errors.length > 0) {
      validationErrors.value[field] = errors
    } else {
      delete validationErrors.value[field]
    }
  }

  const createSectionBackup = (section: keyof SystemSettings) => {
    if (settings.value) {
      sectionBackups.value[section] = JSON.parse(JSON.stringify(settings.value[section]))
    }
  }

  const restoreSectionBackup = (section: keyof SystemSettings) => {
    if (sectionBackups.value[section] && settings.value) {
      settings.value[section] = JSON.parse(JSON.stringify(sectionBackups.value[section]))
      delete sectionBackups.value[section]
    }
  }

  // Validation helpers
  function isValidEmail(email: string): boolean {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return emailRegex.test(email)
  }

  function isValidUrl(url: string): boolean {
    try {
      new URL(url)
      return true
    } catch {
      return false
    }
  }

  return {
    // State
    settings: computed(() => settings.value),
    loading: computed(() => loading.value),
    saving: computed(() => saving.value),
    currentSection: computed(() => currentSection.value),
    searchQuery: computed(() => searchQuery.value),
    dirtyFields: computed(() => dirtyFields.value),
    validationErrors: computed(() => validationErrors.value),

    // Computed
    isDirty,
    hasValidationErrors,
    canSave,
    settingsSections,
    filteredSections,

    // Actions
    loadSettings,
    saveSettings,
    resetSettings,
    testConfiguration,
    exportSettings,
    importSettings,
    updateField,
    validateField,
    setValidationErrors,
    createSectionBackup,
    restoreSectionBackup,

    // Setters
    setCurrentSection: (section: string) => { currentSection.value = section },
    setSearchQuery: (query: string) => { searchQuery.value = query }
  }
})

export const useSettings = () => {
  const settingsStore = useSettingsStore()
  const app = useApp()

  return {
    ...settingsStore,

    // Convenience methods
    getSectionSettings: <T extends keyof SystemSettings>(section: T): SystemSettings[T] | null => {
      return settingsStore.settings?.[section] || null
    },

    updateSectionField: <T extends keyof SystemSettings>(
      section: T,
      field: string,
      value: any
    ) => {
      settingsStore.updateField(`${section}.${field}`, value)
    },

    saveSectionWithConfirmation: async (section: keyof SystemSettings) => {
      try {
        await app.showConfirmation({
          title: 'Save Settings',
          message: `Are you sure you want to save ${section} settings?`,
          confirmText: 'Save',
          type: 'warning'
        })

        await settingsStore.saveSettings(section)
      } catch (error) {
        // User cancelled or error occurred
        if (error !== 'cancelled') {
          throw error
        }
      }
    },

    resetSectionWithConfirmation: async (section: keyof SystemSettings) => {
      try {
        await app.showConfirmation({
          title: 'Reset Settings',
          message: `Are you sure you want to reset ${section} settings? All unsaved changes will be lost.`,
          confirmText: 'Reset',
          type: 'danger'
        })

        await settingsStore.resetSettings(section)
      } catch (error) {
        // User cancelled or error occurred
        if (error !== 'cancelled') {
          throw error
        }
      }
    }
  }
}