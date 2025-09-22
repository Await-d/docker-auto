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
        <el-tab-pane label="基础配置" name="basic">
          <div class="form-section">
            <h3 class="section-title">容器信息</h3>

            <el-form-item label="容器名称" prop="name" required>
              <el-input
                v-model="formData.name"
                placeholder="输入容器名称"
                :disabled="isEditMode"
              />
              <div class="form-help">
                必须唯一且只能包含字母数字字符、连字符和下划线
              </div>
            </el-form-item>

            <el-form-item label="镜像" prop="image" required>
              <div class="image-input-group">
                <el-autocomplete
                  v-model="formData.image"
                  :fetch-suggestions="searchImages"
                  placeholder="nginx, postgres, node 等"
                  style="flex: 1"
                />
                <el-input
                  v-model="formData.tag"
                  placeholder="标签"
                  style="width: 120px"
                />
              </div>
              <div class="form-help">
                镜像名称和标签 (例如：nginx:latest, postgres:13)
              </div>
            </el-form-item>

            <el-form-item label="镜像仓库">
              <el-select
                v-model="formData.registry"
                placeholder="选择镜像仓库 (可选)"
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

            <el-form-item label="启动命令">
              <el-input
                v-model="commandString"
                type="textarea"
                :rows="2"
                placeholder="覆盖默认命令 (每行一个)"
              />
              <div class="form-help">
                在容器中运行的自定义命令 (可选)
              </div>
            </el-form-item>

            <el-form-item label="工作目录">
              <el-input
                v-model="formData.securityOptions.workingDir"
                placeholder="/app"
              />
            </el-form-item>

            <el-form-item label="用户">
              <el-input
                v-model="formData.securityOptions.user"
                placeholder="1000:1000 或用户名"
              />
            </el-form-item>

            <el-form-item label="重启策略">
              <el-select v-model="formData.restartPolicy">
                <el-option label="不重启" value="no" />
                <el-option label="总是重启" value="always" />
                <el-option
                  label="除非手动停止否则重启"
                  value="unless-stopped"
                />
                <el-option label="失败时重启" value="on-failure" />
              </el-select>
            </el-form-item>
          </div>
        </el-tab-pane>

        <!-- 端口映射 -->
        <el-tab-pane label="端口映射" name="ports">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">端口映射</h3>
              <el-button
type="primary" size="small"
@click="addPort"
>
                <el-icon><Plus /></el-icon>
                添加端口
              </el-button>
            </div>

            <div v-if="formData.ports.length === 0" class="empty-state">
              <el-icon class="empty-icon">
                <Connection />
              </el-icon>
              <p>未配置端口映射</p>
              <el-button type="primary" @click="addPort">
                添加第一个端口
              </el-button>
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
                  <template #label> 主机端口 </template>
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
                  <template #label> 容器端口 </template>
                  <el-input-number
                    v-model="port.containerPort"
                    :min="1"
                    :max="65535"
                    placeholder="80"
                  />
                </el-form-item>

                <el-form-item class="port-field">
                  <template #label> 协议 </template>
                  <el-select v-model="port.protocol">
                    <el-option label="TCP" value="tcp" />
                    <el-option label="UDP" value="udp" />
                  </el-select>
                </el-form-item>

                <el-form-item class="port-field">
                  <template #label> 主机IP </template>
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

        <!-- 数据卷 -->
        <el-tab-pane label="数据卷" name="volumes">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">数据卷挂载</h3>
              <el-button
type="primary" size="small"
@click="addVolume"
>
                <el-icon><Plus /></el-icon>
                添加数据卷
              </el-button>
            </div>

            <div v-if="formData.volumes.length === 0" class="empty-state">
              <el-icon class="empty-icon">
                <FolderOpened />
              </el-icon>
              <p>未配置数据卷挂载</p>
              <el-button type="primary" @click="addVolume">
                添加第一个数据卷
              </el-button>
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
                  <template #label> 源路径 </template>
                  <el-input
                    v-model="volume.source"
                    placeholder="/主机路径 或 数据卷名称"
                  />
                </el-form-item>

                <el-form-item
                  :prop="`volumes.${index}.target`"
                  :rules="volumeRules.target"
                  class="volume-field"
                >
                  <template #label> 目标路径 </template>
                  <el-input
                    v-model="volume.target"
                    placeholder="/容器路径"
                  />
                </el-form-item>

                <el-form-item class="volume-field">
                  <template #label> 类型 </template>
                  <el-select v-model="volume.type">
                    <el-option label="绑定挂载" value="bind" />
                    <el-option label="命名数据卷" value="volume" />
                    <el-option label="临时文件系统" value="tmpfs" />
                  </el-select>
                </el-form-item>

                <el-form-item class="volume-field">
                  <template #label> 选项 </template>
                  <el-checkbox v-model="volume.readOnly">
                    只读
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

        <!-- 环境变量 -->
        <el-tab-pane label="环境变量" name="environment">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">环境变量</h3>
              <el-button
                type="primary"
                size="small"
                @click="addEnvironmentVariable"
              >
                <el-icon><Plus /></el-icon>
                添加变量
              </el-button>
            </div>

            <div v-if="environmentVariables.length === 0" class="empty-state">
              <el-icon class="empty-icon">
                <Setting />
              </el-icon>
              <p>未配置环境变量</p>
              <el-button type="primary" @click="addEnvironmentVariable">
                添加第一个变量
              </el-button>
            </div>

            <div v-else class="env-list">
              <div
                v-for="(env, index) in environmentVariables"
                :key="index"
                class="env-item"
              >
                <el-form-item class="env-field">
                  <template #label> 键名 </template>
                  <el-input
v-model="env.key" placeholder="变量名称"
/>
                </el-form-item>

                <el-form-item class="env-field">
                  <template #label> 值 </template>
                  <el-input
                    v-model="env.value"
                    :type="env.sensitive ? 'password' : 'text'"
                    placeholder="值"
                  />
                </el-form-item>

                <el-form-item class="env-field">
                  <template #label> 选项 </template>
                  <el-checkbox v-model="env.sensitive">
敏感信息
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
              <h4>环境变量模板</h4>
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

        <!-- 标签 -->
        <el-tab-pane label="标签" name="labels">
          <div class="form-section">
            <div class="section-header">
              <h3 class="section-title">容器标签</h3>
              <el-button
type="primary" size="small"
@click="addLabel"
>
                <el-icon><Plus /></el-icon>
                添加标签
              </el-button>
            </div>

            <div v-if="labels.length === 0" class="empty-state">
              <el-icon class="empty-icon">
                <Tag />
              </el-icon>
              <p>未配置标签</p>
              <el-button type="primary" @click="addLabel">
                添加第一个标签
              </el-button>
            </div>

            <div v-else class="labels-list">
              <div
                v-for="(label, index) in labels"
                :key="index"
                class="label-item"
              >
                <el-form-item class="label-field">
                  <template #label> 键名 </template>
                  <el-input
v-model="label.key" placeholder="app.name"
/>
                </el-form-item>

                <el-form-item class="label-field">
                  <template #label> 值 </template>
                  <el-input
v-model="label.value" placeholder="我的应用"
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

        <!-- 资源限制 -->
        <el-tab-pane label="资源限制" name="resources">
          <div class="form-section">
            <h3 class="section-title">资源限制</h3>

            <el-form-item label="CPU限制">
              <el-input-number
                v-model="formData.resourceLimits.cpuLimit"
                :min="0"
                :step="0.1"
                :precision="1"
                placeholder="1.0"
              />
              <div class="form-help">
                CPU核数 (例如：1.5 表示1.5个核)
              </div>
            </el-form-item>

            <el-form-item label="内存限制">
              <el-input-number
                v-model="formData.resourceLimits.memoryLimit"
                :min="0"
                :step="64"
                placeholder="512"
              />
              <div class="form-help">
内存限制（MB）
</div>
            </el-form-item>

            <el-form-item label="交换区限制">
              <el-input-number
                v-model="formData.resourceLimits.swapLimit"
                :min="0"
                :step="64"
                placeholder="1024"
              />
              <div class="form-help">
交换区限制（MB）（可选）
</div>
            </el-form-item>

            <h3 class="section-title">安全选项</h3>

            <el-form-item>
              <el-checkbox v-model="formData.securityOptions.readOnly">
                只读文件系统
              </el-checkbox>
            </el-form-item>

            <el-form-item>
              <el-checkbox v-model="formData.securityOptions.privileged">
                特权模式
              </el-checkbox>
            </el-form-item>

            <el-form-item label="添加的权限">
              <el-select
                v-model="formData.securityOptions.capAdd"
                multiple
                placeholder="选择权限"
              >
                <el-option label="NET_ADMIN" value="NET_ADMIN" />
                <el-option label="SYS_ADMIN" value="SYS_ADMIN" />
                <el-option label="SETUID" value="SETUID" />
                <el-option label="SETGID" value="SETGID" />
              </el-select>
            </el-form-item>

            <el-form-item label="移除的权限">
              <el-select
                v-model="formData.securityOptions.capDrop"
                multiple
                placeholder="选择权限"
              >
                <el-option label="ALL" value="ALL" />
                <el-option label="NET_RAW" value="NET_RAW" />
                <el-option label="SYS_ADMIN" value="SYS_ADMIN" />
              </el-select>
            </el-form-item>
          </div>
        </el-tab-pane>

        <!-- 健康检查 -->
        <el-tab-pane label="健康检查" name="health">
          <div class="form-section">
            <h3 class="section-title">健康检查配置</h3>

            <el-form-item>
              <el-checkbox v-model="healthCheckEnabled">
                启用健康检查
              </el-checkbox>
            </el-form-item>

            <div v-if="healthCheckEnabled">
              <el-form-item label="命令" required>
                <el-input
                  v-model="healthCheckCommand"
                  type="textarea"
                  :rows="2"
                  placeholder="curl -f http://localhost:8080/health || exit 1"
                />
                <div class="form-help">
                  健康检查运行命令（每行一个）
                </div>
              </el-form-item>

              <el-form-item label="间隔时间（秒）">
                <el-input-number
                  v-model="formData.healthCheck!.interval"
                  :min="1"
                  placeholder="30"
                />
              </el-form-item>

              <el-form-item label="超时时间（秒）">
                <el-input-number
                  v-model="formData.healthCheck!.timeout"
                  :min="1"
                  placeholder="10"
                />
              </el-form-item>

              <el-form-item label="重试次数">
                <el-input-number
                  v-model="formData.healthCheck!.retries"
                  :min="1"
                  placeholder="3"
                />
              </el-form-item>

              <el-form-item label="启动等待时间（秒）">
                <el-input-number
                  v-model="formData.healthCheck!.startPeriod"
                  :min="0"
                  placeholder="60"
                />
                <div class="form-help">
                  健康检查启动前的宽限期
                </div>
              </el-form-item>
            </div>
          </div>
        </el-tab-pane>

        <!-- 更新策略 -->
        <el-tab-pane label="更新策略" name="updates">
          <div class="form-section">
            <h3 class="section-title">更新策略</h3>

            <el-form-item>
              <el-checkbox v-model="formData.updatePolicy.enabled">
                启用自动更新
              </el-checkbox>
            </el-form-item>

            <div v-if="formData.updatePolicy.enabled">
              <el-form-item label="更新策略">
                <el-select v-model="formData.updatePolicy.strategy">
                  <el-option label="重新创建" value="recreate" />
                  <el-option label="滚动更新" value="rolling" />
                  <el-option label="蓝绿部署" value="blue-green" />
                </el-select>
              </el-form-item>

              <el-form-item>
                <el-checkbox v-model="formData.updatePolicy.autoUpdate">
                  有新版本时自动更新
                </el-checkbox>
              </el-form-item>

              <el-form-item
                v-if="formData.updatePolicy.autoUpdate"
                label="计划时间"
              >
                <el-input
                  v-model="formData.updatePolicy.schedule"
                  placeholder="0 2 * * 0 (每周日凌晨2点)"
                />
                <div class="form-help">
                  定时更新的Cron表达式
                </div>
              </el-form-item>

              <el-form-item>
                <el-checkbox v-model="formData.updatePolicy.notifyOnUpdate">
                  发送更新通知
                </el-checkbox>
              </el-form-item>

              <el-form-item>
                <el-checkbox v-model="formData.updatePolicy.rollbackOnFailure">
                  更新失败时回滚
                </el-checkbox>
              </el-form-item>

              <el-form-item label="最大重试次数">
                <el-input-number
                  v-model="formData.updatePolicy.maxUpdateRetries"
                  :min="0"
                  placeholder="3"
                />
              </el-form-item>

              <el-form-item label="更新超时时间（分钟）">
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
取消
</el-button>
        <el-button @click="resetForm">
重置
</el-button>
        <el-button
type="primary" @click="handleSubmit"
:loading="submitting"
>
          {{ isEditMode ? "更新" : "创建" }} 容器
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { ElMessage } from "element-plus";
import {
  Plus,
  Delete,
  Connection,
  FolderOpened,
  Setting,
} from "@element-plus/icons-vue";

import type {
  Container,
  ContainerFormData,
  PortMapping,
} from "@/types/container";

interface Props {
  container?: Container;
}

interface Emits {
  (e: "submit", data: ContainerFormData | Partial<ContainerFormData>): void;
  (e: "cancel"): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

// Local state
const formRef = ref();
const activeTab = ref("basic");
const submitting = ref(false);

// Form data
const formData = ref<ContainerFormData>({
  name: "",
  image: "",
  tag: "latest",
  ports: [],
  volumes: [],
  environment: {},
  labels: {},
  networks: [],
  updatePolicy: {
    enabled: false,
    strategy: "recreate",
    autoUpdate: false,
    notifyOnUpdate: true,
    rollbackOnFailure: true,
    maxUpdateRetries: 3,
    updateTimeout: 30,
  },
  restartPolicy: "unless-stopped",
  resourceLimits: {
    cpuLimit: undefined,
    memoryLimit: undefined,
    swapLimit: undefined,
    ioLimit: undefined,
  },
  healthCheck: {
    command: [],
    interval: 30,
    timeout: 10,
    retries: 3,
    startPeriod: 60,
  },
  securityOptions: {
    user: undefined,
    workingDir: undefined,
    readOnly: false,
    privileged: false,
    capAdd: [],
    capDrop: [],
  },
  command: undefined,
  entrypoint: undefined,
});

// Helper arrays for reactive form fields
const environmentVariables = ref<
  Array<{ key: string; value: string; sensitive: boolean }>
>([]);
const labels = ref<Array<{ key: string; value: string }>>([]);
const commandString = ref("");
const healthCheckEnabled = ref(false);
const healthCheckCommand = ref("");

// Mock data
const registries = ref([
  { name: "Docker Hub", url: "https://registry-1.docker.io" },
  { name: "GitHub Container Registry", url: "https://ghcr.io" },
  { name: "GitLab Registry", url: "https://registry.gitlab.com" },
]);

const commonImages = [
  "nginx",
  "apache",
  "node",
  "python",
  "postgres",
  "mysql",
  "redis",
  "mongo",
  "rabbitmq",
  "elasticsearch",
  "ubuntu",
  "alpine",
  "debian",
];

// Computed
const isEditMode = computed(() => !!props.container);

// Form validation rules
const formRules = {
  name: [
    { required: true, message: "Container name is required", trigger: "blur" },
    {
      min: 1,
      max: 63,
      message: "Name must be 1-63 characters",
      trigger: "blur",
    },
    {
      pattern: /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/,
      message: "Invalid container name format",
      trigger: "blur",
    },
  ],
  image: [{ required: true, message: "Image is required", trigger: "blur" }],
};

const portRules = {
  hostPort: [
    { required: true, message: "Host port is required", trigger: "blur" },
    {
      type: "number" as const,
      min: 1,
      max: 65535,
      message: "Port must be between 1-65535",
      trigger: "blur",
    },
  ],
  containerPort: [
    { required: true, message: "Container port is required", trigger: "blur" },
    {
      type: "number" as const,
      min: 1,
      max: 65535,
      message: "Port must be between 1-65535",
      trigger: "blur",
    },
  ],
};

const volumeRules = {
  source: [
    { required: true, message: "Source path is required", trigger: "blur" },
  ],
  target: [
    { required: true, message: "Target path is required", trigger: "blur" },
  ],
};

// Methods
function searchImages(
  queryString: string,
  callback: (suggestions: any[]) => void,
) {
  const suggestions = commonImages
    .filter((image) => image.includes(queryString.toLowerCase()))
    .map((image) => ({ value: image }));
  callback(suggestions);
}

function addPort() {
  formData.value.ports.push({
    hostPort: 8080,
    containerPort: 80,
    protocol: "tcp",
    hostIp: "",
  });
}

function removePort(index: number) {
  formData.value.ports.splice(index, 1);
}

function addPortTemplate(type: string) {
  const templates: Record<string, PortMapping> = {
    web: { hostPort: 80, containerPort: 80, protocol: "tcp", hostIp: "" },
    https: { hostPort: 443, containerPort: 443, protocol: "tcp", hostIp: "" },
    ssh: { hostPort: 22, containerPort: 22, protocol: "tcp", hostIp: "" },
    db: { hostPort: 5432, containerPort: 5432, protocol: "tcp", hostIp: "" },
  };

  if (templates[type]) {
    formData.value.ports.push({ ...templates[type] });
  }
}

function addVolume() {
  formData.value.volumes.push({
    source: "",
    target: "",
    type: "bind",
    readOnly: false,
  });
}

function removeVolume(index: number) {
  formData.value.volumes.splice(index, 1);
}

function addEnvironmentVariable() {
  environmentVariables.value.push({
    key: "",
    value: "",
    sensitive: false,
  });
}

function removeEnvironmentVariable(index: number) {
  environmentVariables.value.splice(index, 1);
}

function addEnvTemplate(type: string) {
  const templates: Record<
    string,
    Array<{ key: string; value: string; sensitive: boolean }>
  > = {
    node: [
      { key: "NODE_ENV", value: "production", sensitive: false },
      { key: "PORT", value: "3000", sensitive: false },
    ],
    python: [
      { key: "PYTHONPATH", value: "/app", sensitive: false },
      { key: "PYTHONUNBUFFERED", value: "1", sensitive: false },
    ],
    postgres: [
      { key: "POSTGRES_DB", value: "myapp", sensitive: false },
      { key: "POSTGRES_USER", value: "user", sensitive: false },
      { key: "POSTGRES_PASSWORD", value: "password", sensitive: true },
    ],
    mysql: [
      { key: "MYSQL_DATABASE", value: "myapp", sensitive: false },
      { key: "MYSQL_USER", value: "user", sensitive: false },
      { key: "MYSQL_PASSWORD", value: "password", sensitive: true },
      { key: "MYSQL_ROOT_PASSWORD", value: "rootpassword", sensitive: true },
    ],
  };

  if (templates[type]) {
    environmentVariables.value.push(...templates[type]);
  }
}

function addLabel() {
  labels.value.push({
    key: "",
    value: "",
  });
}

function removeLabel(index: number) {
  labels.value.splice(index, 1);
}

function syncFormData() {
  // Sync environment variables
  formData.value.environment = {};
  environmentVariables.value.forEach((env) => {
    if (env.key) {
      formData.value.environment[env.key] = env.value;
    }
  });

  // Sync labels
  formData.value.labels = {};
  labels.value.forEach((label) => {
    if (label.key) {
      formData.value.labels[label.key] = label.value;
    }
  });

  // Sync command
  if (commandString.value) {
    formData.value.command = commandString.value
      .split("\n")
      .filter((line) => line.trim());
  } else {
    formData.value.command = undefined;
  }

  // Sync health check
  if (healthCheckEnabled.value) {
    formData.value.healthCheck!.command = healthCheckCommand.value
      .split("\n")
      .filter((line) => line.trim());
  } else {
    formData.value.healthCheck = undefined;
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
      restartPolicy: props.container.restartPolicy,
    });

    // Load environment variables
    environmentVariables.value = Object.entries(
      props.container.environment,
    ).map(([key, value]) => ({
      key,
      value,
      sensitive:
        key.toLowerCase().includes("password") ||
        key.toLowerCase().includes("secret"),
    }));

    // Load labels
    labels.value = Object.entries(props.container.labels).map(
      ([key, value]) => ({
        key,
        value,
      }),
    );

    // Load command
    if (props.container.command) {
      commandString.value = props.container.command.join("\n");
    }

    // Load health check
    // This would need to be implemented based on container health check data
  }
}

async function handleSubmit() {
  try {
    await formRef.value.validate();
    syncFormData();

    submitting.value = true;

    if (isEditMode.value) {
      emit("submit", formData.value as Partial<ContainerFormData>);
    } else {
      emit("submit", formData.value);
    }
  } catch (error) {
    console.error("Form validation failed:", error);
    ElMessage.error("Please fix the form errors before submitting");
  } finally {
    submitting.value = false;
  }
}

function resetForm() {
  formRef.value.resetFields();
  environmentVariables.value = [];
  labels.value = [];
  commandString.value = "";
  healthCheckEnabled.value = false;
  healthCheckCommand.value = "";
}

// Lifecycle
onMounted(() => {
  loadFormData();
});

// Watch for prop changes
watch(
  () => props.container,
  () => {
    loadFormData();
  },
  { deep: true },
);
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
