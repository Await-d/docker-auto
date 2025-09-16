/**
 * Container management types and interfaces
 */

export interface Container {
  id: string
  name: string
  image: string
  tag: string
  status: ContainerStatus
  state: ContainerState
  ports: PortMapping[]
  volumes: VolumeMount[]
  environment: Record<string, string>
  labels: Record<string, string>
  networks: NetworkConfig[]
  createdAt: Date
  updatedAt: Date
  startedAt?: Date
  resourceUsage: ResourceMetrics
  health: HealthStatus
  updatePolicy: UpdatePolicy
  registryConfig?: RegistryConfig
  composeFile?: string
  workingDir?: string
  user?: string
  restartPolicy: RestartPolicy
  command?: string[]
  entrypoint?: string[]
}

export type ContainerStatus =
  | 'created'
  | 'running'
  | 'paused'
  | 'restarting'
  | 'removing'
  | 'exited'
  | 'dead'

export interface ContainerState {
  status: ContainerStatus
  running: boolean
  paused: boolean
  restarting: boolean
  oomKilled: boolean
  dead: boolean
  pid: number
  exitCode: number
  error: string
  startedAt: string
  finishedAt: string
}

export interface PortMapping {
  hostPort: number
  containerPort: number
  protocol: 'tcp' | 'udp'
  hostIp?: string
}

export interface VolumeMount {
  source: string
  target: string
  type: 'bind' | 'volume' | 'tmpfs'
  readOnly: boolean
  bindOptions?: {
    propagation: 'private' | 'rprivate' | 'shared' | 'rshared' | 'slave' | 'rslave'
  }
  volumeOptions?: {
    noCopy: boolean
    labels: Record<string, string>
    driverConfig?: {
      name: string
      options: Record<string, string>
    }
  }
  tmpfsOptions?: {
    sizeBytes: number
    mode: number
  }
}

export interface NetworkConfig {
  name: string
  id: string
  ipAddress: string
  gateway: string
  macAddress: string
  aliases: string[]
}

export interface ResourceMetrics {
  cpu: {
    usage: number // Percentage
    limit?: number
    cores?: number
  }
  memory: {
    usage: number // Bytes
    limit?: number
    percentage: number
  }
  network: {
    rxBytes: number
    txBytes: number
    rxPackets: number
    txPackets: number
  }
  disk: {
    readBytes: number
    writeBytes: number
    readOps: number
    writeOps: number
  }
  blkio: {
    read: number
    write: number
    total: number
  }
  timestamp: Date
}

export interface HealthStatus {
  status: 'healthy' | 'unhealthy' | 'starting' | 'none'
  failingStreak: number
  log: HealthLogEntry[]
}

export interface HealthLogEntry {
  start: Date
  end: Date
  exitCode: number
  output: string
}

export interface UpdatePolicy {
  enabled: boolean
  strategy: 'recreate' | 'rolling' | 'blue-green'
  schedule?: string // Cron expression
  autoUpdate: boolean
  notifyOnUpdate: boolean
  rollbackOnFailure: boolean
  preUpdateHook?: string
  postUpdateHook?: string
  maxUpdateRetries: number
  updateTimeout: number
}

export interface RegistryConfig {
  name: string
  url: string
  username?: string
  password?: string
  token?: string
  insecure: boolean
}

export type RestartPolicy =
  | 'no'
  | 'always'
  | 'unless-stopped'
  | 'on-failure'

export interface ContainerLog {
  timestamp: Date
  level: 'debug' | 'info' | 'warn' | 'error' | 'fatal'
  message: string
  stream: 'stdout' | 'stderr'
  container: string
}

export interface ContainerStats {
  container: string
  metrics: ResourceMetrics
  timestamp: Date
}

export interface ContainerEvent {
  id: string
  container: string
  action: string
  timestamp: Date
  attributes: Record<string, string>
}

export interface UpdateAvailable {
  container: string
  currentVersion: string
  availableVersion: string
  releaseNotes?: string
  publishedAt: Date
  size?: number
  critical: boolean
}

export interface ContainerFilter {
  status?: ContainerStatus[]
  image?: string
  registry?: string
  labels?: Record<string, string>
  updatePolicy?: string
  healthStatus?: string[]
  search?: string
}

export interface ContainerSort {
  field: 'name' | 'status' | 'createdAt' | 'updatedAt' | 'cpu' | 'memory'
  direction: 'asc' | 'desc'
}

export interface BulkOperation {
  action: 'start' | 'stop' | 'restart' | 'remove' | 'update' | 'pause' | 'unpause'
  containers: string[]
  options?: Record<string, any>
}

export interface ContainerTemplate {
  id: string
  name: string
  description: string
  image: string
  ports: PortMapping[]
  volumes: VolumeMount[]
  environment: Record<string, string>
  labels: Record<string, string>
  updatePolicy: UpdatePolicy
  restartPolicy: RestartPolicy
  createdBy: string
  createdAt: Date
  category: string
  tags: string[]
}

export interface ContainerFormData {
  name: string
  image: string
  tag: string
  registry?: string
  ports: PortMapping[]
  volumes: VolumeMount[]
  environment: Record<string, string>
  labels: Record<string, string>
  networks: string[]
  updatePolicy: UpdatePolicy
  restartPolicy: RestartPolicy
  resourceLimits: {
    cpuLimit?: number
    memoryLimit?: number
    swapLimit?: number
    ioLimit?: number
  }
  healthCheck?: {
    command: string[]
    interval: number
    timeout: number
    retries: number
    startPeriod: number
  }
  securityOptions: {
    user?: string
    workingDir?: string
    readOnly: boolean
    privileged: boolean
    capAdd: string[]
    capDrop: string[]
  }
  command?: string[]
  entrypoint?: string[]
}

// WebSocket message types
export interface WebSocketMessage {
  type: 'container_status' | 'container_stats' | 'container_logs' | 'container_event'
  data: any
  timestamp: Date
}

export interface ContainerStatusUpdate {
  container: string
  status: ContainerStatus
  state: ContainerState
  timestamp: Date
}

export interface ContainerStatsUpdate {
  container: string
  stats: ResourceMetrics
  timestamp: Date
}

export interface ContainerLogUpdate {
  container: string
  logs: ContainerLog[]
  timestamp: Date
}

// API Response types
export interface ContainerListResponse {
  containers: Container[]
  total: number
  page: number
  limit: number
  filters: ContainerFilter
}

export interface ContainerOperationResult {
  container: string
  success: boolean
  message?: string
  error?: string
}

export interface BulkOperationResult {
  operation: string
  results: ContainerOperationResult[]
  summary: {
    total: number
    successful: number
    failed: number
  }
}

export interface ContainerImage {
  id: string
  repository: string
  tag: string
  digest: string
  size: number
  createdAt: Date
  labels: Record<string, string>
  architecture: string
  os: string
}

export interface ImageUpdateCheck {
  image: string
  currentDigest: string
  latestDigest: string
  hasUpdate: boolean
  releaseNotes?: string
  publishedAt?: Date
}