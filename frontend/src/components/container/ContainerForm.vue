<template>
  <div class="container-form">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="140px"
      @submit.prevent="handleSubmit"
    >
      <el-tabs v-model="activeTab" type="border-card">
        <!-- Basic Configuration -->
        <el-tab-pane label="Basic" name="basic">
          <div class="form-section">
            <h3 class="section-title">Container Information</h3>

            <el-form-item label="Container Name" prop="name" required>
              <el-input
                v-model="formData.name"
                placeholder="Enter container name"
                :disabled="isEditMode"
              />
              <div class="form-help">
                Must be unique and contain only alphanumeric characters, hyphens, and underscores
              </div>
            </el-form-item>

            <el-form-item label="Image" prop="image" required>
              <div class="image-input-group">
                <el-autocomplete
                  v-model="formData.image"
                  :fetch-suggestions="searchImages"
                  placeholder="nginx, postgres, node, etc."
                  style="flex: 1"
                />
                <el-input
                  v-model="formData.tag"
                  placeholder="Tag"
                  style="width: 120px"
                />
              </div>
              <div class="form-help">
                Image name and tag (e.g., nginx:latest, postgres:13)
              </div>
            </el-form-item>

            <el-form-item label="Registry">
              <el-select
                v-model="formData.registry"
                placeholder="Select registry (optional)"
                clearable
              >
                <el-option
                  v-for="registry in registries"
                  :key="registry.name"
                  :label="registry.name"
                  :value="registry.name"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="Command">
              <el-input
                v-model="commandString"
                type="textarea"
                :rows="2"
                placeholder="Override default command (one per line)"
              />
              <div class="form-help">
                Custom command to run in the container (optional)
              </div>
            </el-form-item>

            <el-form-item label="Working Directory">
              <el-input
                v-model="formData.securityOptions.workingDir"
                placeholder="/app"
              />
            </el-form-item>

            <el-form-item label="User">
              <el-input
                v-model="formData.securityOptions.user"
                placeholder="1000:1000 or username"
              />
            </el-form-item>

            <el-form-item label="Restart Policy">
              <el-select v-model="formData.restartPolicy">
                <el-option label="No restart" value="no" />
                <el-option label="Always restart" value="always" />
                <el-option label="Restart unless stopped" value="unless-stopped" />
                <el-option label="Restart on failure" value="on-failure" />
              </el-select>
            </el-form-item>
          </div>
        </el-tab-pane>

        <!-- Ports -->
        <el-tab-pane label="Ports" name="ports">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">Port Mappings</h3>
              <el-button
                type="primary"
                size="small"
                @click="addPort"
              >
                <el-icon><Plus /></el-icon>
                Add Port
              </el-button>
            </div>

            <div v-if="formData.ports.length === 0" class="empty-state">
              <el-icon class="empty-icon"><Connection /></el-icon>
              <p>No port mappings configured</p>
              <el-button type="primary" @click="addPort">Add First Port</el-button>
            </div>

            <div v-else class="ports-list">
              <div
                v-for="(port, index) in formData.ports"
                :key="index"
                class="port-item"
              >
                <el-form-item
                  :prop="`ports.${index}.hostPort`"
                  :rules="portRules.hostPort"
                  class="port-field"
                >
                  <template #label>Host Port</template>
                  <el-input-number
                    v-model="port.hostPort"
                    :min="1"
                    :max="65535"
                    placeholder="8080"
                  />
                </el-form-item>

                <el-form-item
                  :prop="`ports.${index}.containerPort`"
                  :rules="portRules.containerPort"
                  class="port-field"
                >
                  <template #label>Container Port</template>
                  <el-input-number
                    v-model="port.containerPort"
                    :min="1"
                    :max="65535"
                    placeholder="80"
                  />
                </el-form-item>

                <el-form-item class="port-field">
                  <template #label>Protocol</template>
                  <el-select v-model="port.protocol">
                    <el-option label="TCP" value="tcp" />
                    <el-option label="UDP" value="udp" />
                  </el-select>
                </el-form-item>

                <el-form-item class="port-field">
                  <template #label>Host IP</template>
                  <el-input
                    v-model="port.hostIp"
                    placeholder="0.0.0.0 (optional)"
                  />
                </el-form-item>

                <div class="port-actions">
                  <el-button
                    type="danger"
                    size="small"
                    @click="removePort(index)"
                  >
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
            </div>

            <div class="port-templates">
              <h4>Quick Templates</h4>
              <div class="template-buttons">
                <el-button size="small" @click="addPortTemplate('web')">
                  Web (80:80)
                </el-button>
                <el-button size="small" @click="addPortTemplate('https')">
                  HTTPS (443:443)
                </el-button>
                <el-button size="small" @click="addPortTemplate('ssh')">
                  SSH (22:22)
                </el-button>
                <el-button size="small" @click="addPortTemplate('db')">
                  Database (5432:5432)
                </el-button>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- Volumes -->
        <el-tab-pane label="Volumes" name="volumes">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">Volume Mounts</h3>
              <el-button
                type="primary"
                size="small"
                @click="addVolume"
              >
                <el-icon><Plus /></el-icon>
                Add Volume
              </el-button>
            </div>

            <div v-if="formData.volumes.length === 0" class="empty-state">
              <el-icon class="empty-icon"><FolderOpened /></el-icon>
              <p>No volume mounts configured</p>
              <el-button type="primary" @click="addVolume">Add First Volume</el-button>
            </div>

            <div v-else class="volumes-list">
              <div
                v-for="(volume, index) in formData.volumes"
                :key="index"
                class="volume-item"
              >
                <el-form-item
                  :prop="`volumes.${index}.source`"
                  :rules="volumeRules.source"
                  class="volume-field"
                >
                  <template #label>Source</template>
                  <el-input
                    v-model="volume.source"
                    placeholder="/host/path or volume-name"
                  />
                </el-form-item>

                <el-form-item
                  :prop="`volumes.${index}.target`"
                  :rules="volumeRules.target"
                  class="volume-field"
                >
                  <template #label>Target</template>
                  <el-input
                    v-model="volume.target"
                    placeholder="/container/path"
                  />
                </el-form-item>

                <el-form-item class="volume-field">
                  <template #label>Type</template>
                  <el-select v-model="volume.type">
                    <el-option label="Bind Mount" value="bind" />
                    <el-option label="Named Volume" value="volume" />
                    <el-option label="Tmpfs" value="tmpfs" />
                  </el-select>
                </el-form-item>

                <el-form-item class="volume-field">
                  <template #label>Options</template>
                  <el-checkbox v-model="volume.readOnly">
                    Read Only
                  </el-checkbox>
                </el-form-item>

                <div class="volume-actions">
                  <el-button
                    type="danger"
                    size="small"
                    @click="removeVolume(index)"
                  >
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- Environment -->
        <el-tab-pane label="Environment" name="environment">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">Environment Variables</h3>
              <el-button
                type="primary"
                size="small"
                @click="addEnvironmentVariable"
              >
                <el-icon><Plus /></el-icon>
                Add Variable
              </el-button>
            </div>

            <div v-if="environmentVariables.length === 0" class="empty-state">
              <el-icon class="empty-icon"><Setting /></el-icon>
              <p>No environment variables configured</p>
              <el-button type="primary" @click="addEnvironmentVariable">
                Add First Variable
              </el-button>
            </div>

            <div v-else class="env-list">
              <div
                v-for="(env, index) in environmentVariables"
                :key="index"
                class="env-item"
              >
                <el-form-item class="env-field">
                  <template #label>Key</template>
                  <el-input
                    v-model="env.key"
                    placeholder="VARIABLE_NAME"
                  />
                </el-form-item>

                <el-form-item class="env-field">
                  <template #label>Value</template>
                  <el-input
                    v-model="env.value"
                    :type="env.sensitive ? 'password' : 'text'"
                    placeholder="value"
                  />
                </el-form-item>

                <el-form-item class="env-field">
                  <template #label>Options</template>
                  <el-checkbox v-model="env.sensitive">
                    Sensitive
                  </el-checkbox>
                </el-form-item>

                <div class="env-actions">
                  <el-button
                    type="danger"
                    size="small"
                    @click="removeEnvironmentVariable(index)"
                  >
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
            </div>

            <div class="env-templates">
              <h4>Environment Templates</h4>
              <div class="template-buttons">
                <el-button size="small" @click="addEnvTemplate('node')">
                  Node.js
                </el-button>
                <el-button size="small" @click="addEnvTemplate('python')">
                  Python
                </el-button>
                <el-button size="small" @click="addEnvTemplate('postgres')">
                  PostgreSQL
                </el-button>
                <el-button size="small" @click="addEnvTemplate('mysql')">
                  MySQL
                </el-button>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- Labels -->
        <el-tab-pane label="Labels" name="labels">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">Container Labels</h3>
              <el-button
                type="primary"
                size="small"
                @click="addLabel"
              >
                <el-icon><Plus /></el-icon>
                Add Label
              </el-button>
            </div>

            <div v-if="labels.length === 0" class="empty-state">
              <el-icon class="empty-icon"><Tag /></el-icon>
              <p>No labels configured</p>
              <el-button type="primary" @click="addLabel">Add First Label</el-button>
            </div>

            <div v-else class="labels-list">
              <div
                v-for="(label, index) in labels"
                :key="index"
                class="label-item"
              >
                <el-form-item class="label-field">
                  <template #label>Key</template>
                  <el-input
                    v-model="label.key"
                    placeholder="app.name"
                  />
                </el-form-item>

                <el-form-item class="label-field">
                  <template #label>Value</template>
                  <el-input
                    v-model="label.value"
                    placeholder="my-app"
                  />
                </el-form-item>

                <div class="label-actions">
                  <el-button
                    type="danger"
                    size="small"
                    @click="removeLabel(index)"
                  >
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- Resources -->
        <el-tab-pane label="Resources" name="resources">
          <div class="form-section">
            <h3 class="section-title">Resource Limits</h3>

            <el-form-item label="CPU Limit">
              <el-input-number
                v-model="formData.resourceLimits.cpuLimit"
                :min="0"
                :step="0.1"
                :precision="1"
                placeholder="1.0"
              />
              <div class="form-help">
                Number of CPU cores (e.g., 1.5 for 1.5 cores)
              </div>
            </el-form-item>

            <el-form-item label="Memory Limit">
              <el-input-number
                v-model="formData.resourceLimits.memoryLimit"
                :min="0"
                :step="64"
                placeholder="512"
              />
              <div class="form-help">
                Memory limit in MB
              </div>
            </el-form-item>

            <el-form-item label="Swap Limit">
              <el-input-number
                v-model="formData.resourceLimits.swapLimit"
                :min="0"
                :step="64"
                placeholder="1024"
              />
              <div class="form-help">
                Swap limit in MB (optional)
              </div>
            </el-form-item>

            <h3 class="section-title">Security Options</h3>

            <el-form-item>
              <el-checkbox v-model="formData.securityOptions.readOnly">
                Read-only filesystem
              </el-checkbox>
            </el-form-item>

            <el-form-item>
              <el-checkbox v-model="formData.securityOptions.privileged">
                Privileged mode
              </el-checkbox>
            </el-form-item>

            <el-form-item label="Capabilities to Add">
              <el-select
                v-model="formData.securityOptions.capAdd"
                multiple
                placeholder="Select capabilities"
              >
                <el-option label="NET_ADMIN" value="NET_ADMIN" />
                <el-option label="SYS_ADMIN" value="SYS_ADMIN" />
                <el-option label="SETUID" value="SETUID" />
                <el-option label="SETGID" value="SETGID" />
              </el-select>
            </el-form-item>

            <el-form-item label="Capabilities to Drop">
              <el-select
                v-model="formData.securityOptions.capDrop"
                multiple
                placeholder="Select capabilities"
              >
                <el-option label="ALL" value="ALL" />
                <el-option label="NET_RAW" value="NET_RAW" />
                <el-option label="SYS_ADMIN" value="SYS_ADMIN" />
              </el-select>
            </el-form-item>
          </div>
        </el-tab-pane>

        <!-- Health Check -->
        <el-tab-pane label="Health Check" name="health">
          <div class="form-section">
            <h3 class="section-title">Health Check Configuration</h3>

            <el-form-item>
              <el-checkbox v-model="healthCheckEnabled">
                Enable health check
              </el-checkbox>
            </el-form-item>

            <div v-if="healthCheckEnabled">
              <el-form-item label="Command" required>
                <el-input
                  v-model="healthCheckCommand"
                  type="textarea"
                  :rows="2"
                  placeholder="curl -f http://localhost:8080/health || exit 1"
                />
                <div class="form-help">
                  Command to run for health check (one per line)
                </div>
              </el-form-item>

              <el-form-item label="Interval (seconds)">
                <el-input-number
                  v-model="formData.healthCheck.interval"
                  :min="1"
                  placeholder="30"
                />
              </el-form-item>

              <el-form-item label="Timeout (seconds)">
                <el-input-number
                  v-model="formData.healthCheck.timeout"
                  :min="1"
                  placeholder="10"
                />
              </el-form-item>

              <el-form-item label="Retries">
                <el-input-number
                  v-model="formData.healthCheck.retries"
                  :min="1"
                  placeholder="3"
                />
              </el-form-item>

              <el-form-item label="Start Period (seconds)">
                <el-input-number
                  v-model="formData.healthCheck.startPeriod"
                  :min="0"
                  placeholder="60"
                />
                <div class="form-help">
                  Grace period before health checks start
                </div>
              </el-form-item>
            </div>
          </div>
        </el-tab-pane>

        <!-- Update Policy -->
        <el-tab-pane label="Updates" name="updates">
          <div class="form-section">
            <h3 class="section-title">Update Policy</h3>

            <el-form-item>
              <el-checkbox v-model="formData.updatePolicy.enabled">
                Enable automatic updates
              </el-checkbox>
            </el-form-item>

            <div v-if="formData.updatePolicy.enabled">
              <el-form-item label="Update Strategy">
                <el-select v-model="formData.updatePolicy.strategy">
                  <el-option label="Recreate" value="recreate" />
                  <el-option label="Rolling Update" value="rolling" />
                  <el-option label="Blue-Green" value="blue-green" />
                </el-select>
              </el-form-item>

              <el-form-item>
                <el-checkbox v-model="formData.updatePolicy.autoUpdate">
                  Auto-update when new versions are available
                </el-checkbox>
              </el-form-item>

              <el-form-item label="Schedule" v-if="formData.updatePolicy.autoUpdate">
                <el-input
                  v-model="formData.updatePolicy.schedule"
                  placeholder="0 2 * * 0 (Every Sunday at 2 AM)"
                />
                <div class="form-help">
                  Cron expression for scheduled updates
                </div>
              </el-form-item>

              <el-form-item>
                <el-checkbox v-model="formData.updatePolicy.notifyOnUpdate">
                  Send notifications for updates
                </el-checkbox>
              </el-form-item>

              <el-form-item>
                <el-checkbox v-model="formData.updatePolicy.rollbackOnFailure">
                  Rollback on update failure
                </el-checkbox>
              </el-form-item>

              <el-form-item label="Max Retries">
                <el-input-number
                  v-model="formData.updatePolicy.maxUpdateRetries"
                  :min="0"
                  placeholder="3"
                />
              </el-form-item>

              <el-form-item label="Update Timeout (minutes)">
                <el-input-number
                  v-model="formData.updatePolicy.updateTimeout"
                  :min="1"
                  placeholder="30"
                />
              </el-form-item>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- Form Actions -->
      <div class="form-actions">
        <el-button @click="$emit('cancel')">
          Cancel
        </el-button>
        <el-button @click="resetForm">
          Reset
        </el-button>
        <el-button
          type="primary"
          @click="handleSubmit"
          :loading="submitting"
        >
          {{ isEditMode ? 'Update' : 'Create' }} Container
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Plus,
  Delete,
  Connection,
  FolderOpened,
  Setting,
  Tag
} from '@element-plus/icons-vue'

import type { Container, ContainerFormData, PortMapping, VolumeMount } from '@/types/container'

interface Props {
  container?: Container
}

interface Emits {
  (e: 'submit', data: ContainerFormData | Partial<ContainerFormData>): void
  (e: 'cancel'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// Local state
const formRef = ref()
const activeTab = ref('basic')
const submitting = ref(false)

// Form data
const formData = ref<ContainerFormData>({
  name: '',
  image: '',
  tag: 'latest',
  ports: [],
  volumes: [],
  environment: {},
  labels: {},
  networks: [],
  updatePolicy: {
    enabled: false,
    strategy: 'recreate',
    autoUpdate: false,
    notifyOnUpdate: true,
    rollbackOnFailure: true,
    maxUpdateRetries: 3,
    updateTimeout: 30
  },
  restartPolicy: 'unless-stopped',
  resourceLimits: {
    cpuLimit: undefined,
    memoryLimit: undefined,
    swapLimit: undefined,
    ioLimit: undefined
  },
  healthCheck: {
    command: [],
    interval: 30,
    timeout: 10,
    retries: 3,
    startPeriod: 60
  },
  securityOptions: {
    user: undefined,
    workingDir: undefined,
    readOnly: false,
    privileged: false,
    capAdd: [],
    capDrop: []
  },
  command: undefined,
  entrypoint: undefined
})

// Helper arrays for reactive form fields
const environmentVariables = ref<Array<{ key: string; value: string; sensitive: boolean }>>([])
const labels = ref<Array<{ key: string; value: string }>>([])
const commandString = ref('')
const healthCheckEnabled = ref(false)
const healthCheckCommand = ref('')

// Mock data
const registries = ref([
  { name: 'Docker Hub', url: 'https://registry-1.docker.io' },
  { name: 'GitHub Container Registry', url: 'https://ghcr.io' },
  { name: 'GitLab Registry', url: 'https://registry.gitlab.com' }
])

const commonImages = [
  'nginx', 'apache', 'node', 'python', 'postgres', 'mysql', 'redis',
  'mongo', 'rabbitmq', 'elasticsearch', 'ubuntu', 'alpine', 'debian'
]

// Computed
const isEditMode = computed(() => !!props.container)

// Form validation rules
const formRules = {
  name: [
    { required: true, message: 'Container name is required', trigger: 'blur' },
    { min: 1, max: 63, message: 'Name must be 1-63 characters', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/, message: 'Invalid container name format', trigger: 'blur' }
  ],
  image: [
    { required: true, message: 'Image is required', trigger: 'blur' }
  ]
}

const portRules = {
  hostPort: [
    { required: true, message: 'Host port is required', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: 'Port must be between 1-65535', trigger: 'blur' }
  ],
  containerPort: [
    { required: true, message: 'Container port is required', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: 'Port must be between 1-65535', trigger: 'blur' }
  ]
}

const volumeRules = {
  source: [
    { required: true, message: 'Source path is required', trigger: 'blur' }
  ],
  target: [
    { required: true, message: 'Target path is required', trigger: 'blur' }
  ]
}

// Methods
function searchImages(queryString: string, callback: (suggestions: any[]) => void) {
  const suggestions = commonImages
    .filter(image => image.includes(queryString.toLowerCase()))
    .map(image => ({ value: image }))
  callback(suggestions)
}

function addPort() {
  formData.value.ports.push({
    hostPort: 8080,
    containerPort: 80,
    protocol: 'tcp',
    hostIp: ''
  })
}

function removePort(index: number) {
  formData.value.ports.splice(index, 1)
}

function addPortTemplate(type: string) {
  const templates: Record<string, PortMapping> = {
    web: { hostPort: 80, containerPort: 80, protocol: 'tcp', hostIp: '' },
    https: { hostPort: 443, containerPort: 443, protocol: 'tcp', hostIp: '' },
    ssh: { hostPort: 22, containerPort: 22, protocol: 'tcp', hostIp: '' },
    db: { hostPort: 5432, containerPort: 5432, protocol: 'tcp', hostIp: '' }
  }

  if (templates[type]) {
    formData.value.ports.push({ ...templates[type] })
  }
}

function addVolume() {
  formData.value.volumes.push({
    source: '',
    target: '',
    type: 'bind',
    readOnly: false
  })
}

function removeVolume(index: number) {
  formData.value.volumes.splice(index, 1)
}

function addEnvironmentVariable() {
  environmentVariables.value.push({
    key: '',
    value: '',
    sensitive: false
  })
}

function removeEnvironmentVariable(index: number) {
  environmentVariables.value.splice(index, 1)
}

function addEnvTemplate(type: string) {
  const templates: Record<string, Array<{ key: string; value: string; sensitive: boolean }>> = {
    node: [
      { key: 'NODE_ENV', value: 'production', sensitive: false },
      { key: 'PORT', value: '3000', sensitive: false }
    ],
    python: [
      { key: 'PYTHONPATH', value: '/app', sensitive: false },
      { key: 'PYTHONUNBUFFERED', value: '1', sensitive: false }
    ],
    postgres: [
      { key: 'POSTGRES_DB', value: 'myapp', sensitive: false },
      { key: 'POSTGRES_USER', value: 'user', sensitive: false },
      { key: 'POSTGRES_PASSWORD', value: 'password', sensitive: true }
    ],
    mysql: [
      { key: 'MYSQL_DATABASE', value: 'myapp', sensitive: false },
      { key: 'MYSQL_USER', value: 'user', sensitive: false },
      { key: 'MYSQL_PASSWORD', value: 'password', sensitive: true },
      { key: 'MYSQL_ROOT_PASSWORD', value: 'rootpassword', sensitive: true }
    ]
  }

  if (templates[type]) {
    environmentVariables.value.push(...templates[type])
  }
}

function addLabel() {
  labels.value.push({
    key: '',
    value: ''
  })
}

function removeLabel(index: number) {
  labels.value.splice(index, 1)
}

function syncFormData() {
  // Sync environment variables
  formData.value.environment = {}
  environmentVariables.value.forEach(env => {
    if (env.key) {
      formData.value.environment[env.key] = env.value
    }
  })

  // Sync labels
  formData.value.labels = {}
  labels.value.forEach(label => {
    if (label.key) {
      formData.value.labels[label.key] = label.value
    }
  })

  // Sync command
  if (commandString.value) {
    formData.value.command = commandString.value.split('\n').filter(line => line.trim())
  } else {
    formData.value.command = undefined
  }

  // Sync health check
  if (healthCheckEnabled.value) {
    formData.value.healthCheck!.command = healthCheckCommand.value.split('\n').filter(line => line.trim())
  } else {
    formData.value.healthCheck = undefined
  }
}

function loadFormData() {
  if (props.container) {
    // Load existing container data
    Object.assign(formData.value, {
      name: props.container.name,
      image: props.container.image,
      tag: props.container.tag,
      ports: [...props.container.ports],
      volumes: [...props.container.volumes],
      environment: { ...props.container.environment },
      labels: { ...props.container.labels },
      updatePolicy: { ...props.container.updatePolicy },
      restartPolicy: props.container.restartPolicy
    })

    // Load environment variables
    environmentVariables.value = Object.entries(props.container.environment).map(([key, value]) => ({
      key,
      value,
      sensitive: key.toLowerCase().includes('password') || key.toLowerCase().includes('secret')
    }))

    // Load labels
    labels.value = Object.entries(props.container.labels).map(([key, value]) => ({
      key,
      value
    }))

    // Load command
    if (props.container.command) {
      commandString.value = props.container.command.join('\n')
    }

    // Load health check
    // This would need to be implemented based on container health check data
  }
}

async function handleSubmit() {
  try {
    await formRef.value.validate()
    syncFormData()

    submitting.value = true

    if (isEditMode.value) {
      emit('submit', formData.value as Partial<ContainerFormData>)
    } else {
      emit('submit', formData.value)
    }
  } catch (error) {
    console.error('Form validation failed:', error)
    ElMessage.error('Please fix the form errors before submitting')
  } finally {
    submitting.value = false
  }
}

function resetForm() {
  formRef.value.resetFields()
  environmentVariables.value = []
  labels.value = []
  commandString.value = ''
  healthCheckEnabled.value = false
  healthCheckCommand.value = ''
}

// Lifecycle
onMounted(() => {
  loadFormData()
})

// Watch for prop changes
watch(() => props.container, () => {
  loadFormData()
}, { deep: true })
</script>

<style scoped>
.container-form {
  max-width: 100%;
}

.form-section {
  padding: 20px;
}

.section-title {
  margin: 0 0 20px 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.form-help {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.image-input-group {
  display: flex;
  gap: 8px;
}

.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #909399;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #c0c4cc;
}

.empty-state p {
  margin: 0 0 16px 0;
}

.ports-list,
.volumes-list,
.env-list,
.labels-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.port-item,
.volume-item,
.env-item,
.label-item {
  display: flex;
  align-items: flex-end;
  gap: 16px;
  padding: 16px;
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
}

.port-field,
.volume-field,
.env-field,
.label-field {
  flex: 1;
  margin-bottom: 0;
}

.port-actions,
.volume-actions,
.env-actions,
.label-actions {
  display: flex;
  align-items: flex-end;
  padding-bottom: 8px;
}

.port-templates,
.env-templates {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e4e7ed;
}

.port-templates h4,
.env-templates h4 {
  margin: 0 0 12px 0;
  color: #606266;
  font-size: 14px;
}

.template-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid #e4e7ed;
  background: #f8f9fa;
}

/* Responsive Design */
@media (max-width: 768px) {
  .port-item,
  .volume-item,
  .env-item,
  .label-item {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .port-actions,
  .volume-actions,
  .env-actions,
  .label-actions {
    align-self: flex-end;
    padding-bottom: 0;
  }

  .form-actions {
    flex-direction: column-reverse;
    gap: 8px;
  }

  .image-input-group {
    flex-direction: column;
  }

  .template-buttons {
    flex-direction: column;
  }
}

/* Form field overrides */
:deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

:deep(.el-tabs__content) {
  padding: 0;
}

:deep(.el-tab-pane) {
  max-height: 60vh;
  overflow-y: auto;
}
</style>