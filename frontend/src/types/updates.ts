/**
 * Update management types and interfaces
 */

// Base update types
export type UpdateType = 'major' | 'minor' | 'patch' | 'security' | 'hotfix' | 'rollback'
export type UpdateStatus = 'available' | 'queued' | 'running' | 'completed' | 'failed' | 'cancelled' | 'rollback'
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical'
export type UpdatePriority = 'low' | 'normal' | 'high' | 'critical'
export type UpdateStrategy = 'recreate' | 'rolling' | 'blue-green' | 'canary'
export type ApprovalStatus = 'pending' | 'approved' | 'rejected'

// Container Update
export interface ContainerUpdate {
  id: string
  containerId: string
  containerName: string
  imageName: string
  currentVersion: string
  availableVersion: string
  updateType: UpdateType
  riskLevel: RiskLevel
  priority: UpdatePriority
  size: number // in bytes
  releaseDate: string
  availableDate: string
  securityPatches: SecurityPatch[]
  releaseNotes: string
  releaseNotesUrl?: string
  changelog: ChangelogItem[]
  estimatedDowntime: number // in seconds
  dependencies: string[] // container IDs that depend on this
  conflicts: string[] // container IDs that conflict with this update
  tags: string[]
  labels: Record<string, string>

  // Status flags
  ignored: boolean
  ignoredReason?: string
  ignoredAt?: string
  scheduled: boolean
  scheduledAt?: string
  requiresApproval: boolean
  approvalStatus?: ApprovalStatus

  // Metadata
  createdAt: string
  updatedAt: string
  lastChecked: string
}

// Security Patch
export interface SecurityPatch {
  id: string
  cveId?: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  description: string
  affectedVersions: string[]
  patchedVersion: string
  publishedDate: string
  score?: number // CVSS score
  vector?: string // Attack vector
  references: Array<{
    type: 'advisory' | 'patch' | 'exploit' | 'article'
    url: string
    title: string
  }>
}

// Changelog Item
export interface ChangelogItem {
  type: 'added' | 'changed' | 'deprecated' | 'removed' | 'fixed' | 'security'
  description: string
  breaking: boolean
  author?: string
  pullRequest?: string
  issue?: string
}

// Update History
export interface UpdateHistoryItem {
  id: string
  containerId: string
  containerName: string
  imageName?: string
  fromVersion: string
  toVersion: string
  updateType: UpdateType
  status: UpdateStatus
  strategy?: UpdateStrategy
  triggeredBy: 'manual' | 'scheduled' | 'policy' | 'webhook' | 'api'
  triggeredById?: string
  startedAt: string
  completedAt?: string
  duration?: number // in seconds
  reason?: string
  error?: string
  errorCode?: string
  rollbackReason?: string

  // Update details
  steps?: UpdateStep[]
  currentStep?: number
  logs?: UpdateLog[]
  healthChecks?: HealthCheck[]

  // Metadata
  size?: number
  downloadTime?: number
  restartTime?: number
  verificationTime?: number

  // Rollback info
  canRollback: boolean
  rollbackAvailable: boolean
  rollbackTo?: string
}

// Running Update
export interface RunningUpdate {
  id: string
  updateId: string
  containerId: string
  containerName: string
  fromVersion: string
  toVersion: string
  strategy: UpdateStrategy
  status: 'queued' | 'running' | 'stopping' | 'paused'
  progress: number // 0-100
  currentStep: number
  totalSteps: number
  startedAt: string
  estimatedDuration: number
  elapsedTime: number
  remainingTime?: number

  // Progress details
  steps: UpdateStep[]
  logs: UpdateLog[]
  metrics?: UpdateMetrics

  // Control
  canCancel: boolean
  canPause: boolean
  canRetry: boolean
}

// Update Step
export interface UpdateStep {
  id: number
  name: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped'
  startedAt?: string
  completedAt?: string
  duration?: number
  progress: number
  error?: string
  logs: string[]
  skippable: boolean
  retryable: boolean
  retryCount?: number
  maxRetries?: number
}

// Update Log
export interface UpdateLog {
  timestamp: string
  level: 'debug' | 'info' | 'warn' | 'error'
  message: string
  source: string
  step?: number
  data?: any
}

// Update Metrics
export interface UpdateMetrics {
  downloadSpeed: number // bytes/sec
  downloadProgress: number // 0-100
  extractionProgress: number // 0-100
  cpuUsage: number // percentage
  memoryUsage: number // bytes
  diskIo: number // bytes/sec
  networkIo: number // bytes/sec
}

// Health Check
export interface HealthCheck {
  name: string
  status: 'passed' | 'failed' | 'warning' | 'skipped'
  message: string
  duration: number
  timestamp: string
  details?: any
}

// Scheduled Update
export interface ScheduledUpdate {
  id: string
  updateId: string
  containerId: string
  containerName: string
  fromVersion: string
  toVersion: string
  scheduledAt: string
  status: 'scheduled' | 'running' | 'completed' | 'failed' | 'cancelled'
  strategy: UpdateStrategy
  recurring: boolean
  recurringPattern?: string // cron pattern
  nextRun?: string
  notifyBefore: number // milliseconds
  notifications: NotificationSettings

  // Options
  rollbackOnFailure: boolean
  runTests: boolean
  requireApproval: boolean

  // Metadata
  createdAt: string
  createdBy: string
  lastModified?: string
  modifiedBy?: string
  runCount: number
  lastRun?: string
  lastStatus?: UpdateStatus
}

// Update Policy
export interface UpdatePolicy {
  id: string
  name: string
  description?: string
  enabled: boolean

  // Matching criteria
  containerSelector: ContainerSelector
  updateTypeFilter: UpdateType[]
  riskLevelFilter: RiskLevel[]
  securityOnly: boolean
  autoApprove: boolean

  // Update settings
  strategy: UpdateStrategy
  schedule?: ScheduleSettings
  rollbackOnFailure: boolean
  runTests: boolean
  requireApproval: boolean
  maxConcurrent: number

  // Notification settings
  notifications: NotificationSettings

  // Advanced settings
  maintenanceWindow?: MaintenanceWindow
  excludeDays?: string[] // days of week
  cooldownPeriod?: number // seconds between updates

  // Metadata
  createdAt: string
  createdBy: string
  lastModified?: string
  modifiedBy?: string
  lastApplied?: string
  applicationCount: number
}

// Container Selector
export interface ContainerSelector {
  names?: string[] // exact names
  patterns?: string[] // regex patterns
  labels?: Record<string, string>
  images?: string[] // image patterns
  tags?: string[]
  excludeNames?: string[]
  excludeLabels?: Record<string, string>
}

// Schedule Settings
export interface ScheduleSettings {
  type: 'immediate' | 'scheduled' | 'recurring'
  scheduledAt?: string
  cronPattern?: string
  timezone?: string
  maxDelay?: number // seconds
}

// Maintenance Window
export interface MaintenanceWindow {
  enabled: boolean
  start: string // HH:MM
  end: string // HH:MM
  timezone: string
  days: string[] // days of week
  buffer: number // minutes before/after
}

// Notification Settings
export interface NotificationSettings {
  enabled: boolean
  channels: NotificationChannel[]
  events: NotificationEvent[]
  beforeUpdate?: number // minutes
  onSuccess: boolean
  onFailure: boolean
  onRollback: boolean
}

// Notification Channel
export interface NotificationChannel {
  type: 'email' | 'slack' | 'webhook' | 'in-app'
  enabled: boolean
  config: Record<string, any>
}

// Notification Event
export type NotificationEvent =
  | 'update_available'
  | 'update_started'
  | 'update_completed'
  | 'update_failed'
  | 'rollback_started'
  | 'rollback_completed'
  | 'approval_required'

// Update Template
export interface UpdateTemplate {
  id: string
  name: string
  description?: string

  // Template settings
  strategy: UpdateStrategy
  rollbackOnFailure: boolean
  runTests: boolean
  requireApproval: boolean
  notifications: NotificationSettings
  schedule?: ScheduleSettings
  maintenanceWindow?: MaintenanceWindow

  // Metadata
  createdAt: string
  createdBy: string
  lastUsed?: string
  usageCount: number
}

// Bulk Update Operation
export interface BulkUpdateOperation {
  updateIds: string[]
  strategy: 'sequential' | 'parallel' | 'rolling'
  maxConcurrent: number
  continueOnError: boolean
  rollbackOnFailure: boolean
  runTests: boolean
  estimatedDuration?: number

  // Dependencies
  respectDependencies: boolean
  dependencyStrategy: 'strict' | 'loose' | 'ignore'

  // Progress tracking
  operationId?: string
  status?: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress?: number
  startedAt?: string
  completedAt?: string
}

// Rollback Operation
export interface RollbackOperation {
  historyItemId: string
  containerId: string
  targetVersion: string
  reason: string
  strategy?: UpdateStrategy
  force?: boolean
}

// Version Comparison
export interface VersionComparison {
  containerId: string
  containerName: string
  imageName: string
  fromVersion: string
  toVersion: string

  // Version analysis
  versionType: UpdateType
  breaking: boolean
  compatibility: CompatibilityInfo

  // Changes
  changelog: ChangelogItem[]
  securityPatches: SecurityPatch[]

  // Size analysis
  sizeDiff: number // bytes
  layerChanges: LayerChange[]

  // Recommendations
  recommendations: string[]
  warnings: string[]

  // Generated at
  generatedAt: string
}

// Compatibility Info
export interface CompatibilityInfo {
  overall: 'compatible' | 'warning' | 'incompatible'
  api: {
    status: 'compatible' | 'warning' | 'incompatible'
    changes: string[]
  }
  config: {
    status: 'compatible' | 'warning' | 'incompatible'
    changes: string[]
  }
  dependencies: {
    status: 'compatible' | 'warning' | 'incompatible'
    changes: string[]
  }
}

// Layer Change
export interface LayerChange {
  type: 'added' | 'removed' | 'modified'
  digest: string
  size: number
  description?: string
}

// Update Analytics
export interface UpdateAnalytics {
  totalUpdatesAvailable: number
  securityUpdates: number
  criticalUpdates: number
  successRate: number // percentage
  averageUpdateTime: number // seconds
  failedUpdatesLast30Days: number
  updatesThisMonth: number

  // Trends (arrays for charting)
  updateTrend?: Array<{
    date: string
    available: number
    completed: number
    failed: number
  }>

  // Distribution
  updatesByType?: Record<UpdateType, number>
  updatesByRisk?: Record<RiskLevel, number>
  updatesByDay?: Record<string, number>

  // Performance
  averageDowntime?: number
  largestUpdate?: {
    size: number
    container: string
    duration: number
  }
}

// Update Settings
export interface UpdateSettings {
  autoCheckInterval: number // milliseconds
  enableNotifications: boolean
  requireApproval: boolean
  maxConcurrentUpdates: number
  updateStrategy: UpdateStrategy
  rollbackOnFailure: boolean
  testBeforeUpdate: boolean

  // Maintenance window
  maintenanceWindow: MaintenanceWindow

  // Advanced settings
  downloadTimeout: number
  healthCheckTimeout: number
  rollbackTimeout: number
  maxRetries: number

  // Cleanup settings
  cleanupOldImages: boolean
  cleanupAfterDays: number
  keepVersions: number
}

// Update Notification
export interface UpdateNotification {
  id: string
  type: 'info' | 'warning' | 'error' | 'success'
  title: string
  message: string
  timestamp: string
  read: boolean
  duration?: number
  data?: {
    containerId?: string
    containerName?: string
    updateId?: string
    action?: string
    [key: string]: any
  }
}

// Filter and Sort types
export interface UpdateFilter {
  updateType?: UpdateType[]
  riskLevel?: RiskLevel[]
  status?: UpdateStatus[]
  containerName?: string
  imageName?: string
  securityOnly?: boolean
  ignored?: boolean
  scheduled?: boolean
  requiresApproval?: boolean
  dateRange?: {
    start: string
    end: string
  }
  size?: {
    min?: number
    max?: number
  }
}

export interface UpdateSort {
  field: 'available_date' | 'release_date' | 'container_name' | 'image_name' | 'size' | 'risk_level' | 'update_type'
  direction: 'asc' | 'desc'
}

// WebSocket message types
export interface UpdateWebSocketMessage {
  type: 'update_progress' | 'update_completed' | 'update_failed' | 'update_available' | 'update_notification'
  data: any
  timestamp: string
}

// Export types
export interface ExportOptions {
  format: 'csv' | 'json' | 'pdf'
  filters?: UpdateFilter
  dateRange?: {
    start: string
    end: string
  }
  includeFields?: string[]
}

// Validation result
export interface ValidationResult {
  valid: boolean
  errors: string[]
  warnings: string[]
  recommendations: string[]
}

// Update simulation result
export interface SimulationResult {
  steps: Array<{
    name: string
    description: string
    estimatedDuration: number
    riskLevel: RiskLevel
    reversible: boolean
  }>
  totalEstimatedTime: number
  overallRiskLevel: RiskLevel
  requiredDowntime: number
  potentialIssues: Array<{
    issue: string
    probability: number // 0-1
    impact: RiskLevel
    mitigation: string
  }>
}