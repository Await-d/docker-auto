<template>
  <div class="security-dashboard-widget">
    <!-- Loading State -->
    <div v-if="isLoading && !hasSecurityData" class="loading-container">
      <el-skeleton :rows="3" animated />
      <div class="loading-text">正在加载安全信息...</div>
    </div>

    <!-- Error State -->
    <div v-else-if="error && !hasSecurityData" class="error-container">
      <el-icon :size="48" class="error-icon">
        <Warning />
      </el-icon>
      <h3 class="error-title">无法加载安全信息</h3>
      <p class="error-message">{{ error }}</p>
      <el-button @click="retryLoad" type="primary" size="small">
        <el-icon><Refresh /></el-icon>
        重试
      </el-button>
    </div>

    <!-- Main Content -->
    <div v-else class="security-content">
      <!-- Security Overview Header -->
      <div class="security-header">
        <div class="security-score">
          <div class="score-circle" :class="getScoreClass(overallScore)">
            <span class="score-value">{{ overallScore }}</span>
            <span class="score-unit">分</span>
          </div>
          <div class="score-info">
            <h3 class="score-title">安全评分</h3>
            <p class="score-subtitle">{{ getScoreDescription(overallScore) }}</p>
          </div>
        </div>

        <div class="header-actions">
          <el-button-group size="small">
            <el-button @click="startSecurityScan" :loading="isScanning" type="primary">
              <el-icon><Search /></el-icon>
              扫描
            </el-button>
            <el-button @click="viewSecurityReport" type="default">
              <el-icon><Document /></el-icon>
              报告
            </el-button>
          </el-button-group>
        </div>
      </div>

      <!-- Critical Vulnerabilities Alert -->
      <div v-if="criticalVulnerabilities.length > 0" class="critical-alert">
        <el-alert
          title="发现严重漏洞"
          type="error"
          :description="`发现 ${criticalVulnerabilities.length} 个严重安全漏洞，需要立即处理`"
          show-icon
          :closable="false"
        >
          <template #default>
            <div class="critical-actions">
              <el-button size="small" type="danger" @click="showCriticalVulnerabilities">
                查看详情
              </el-button>
              <el-button size="small" type="primary" @click="autoFixVulnerabilities">
                自动修复
              </el-button>
            </div>
          </template>
        </el-alert>
      </div>

      <!-- Security Metrics Grid -->
      <div class="metrics-grid">
        <div class="metric-card vulnerabilities">
          <div class="metric-header">
            <el-icon class="metric-icon vulnerability-icon">
              <Warning />
            </el-icon>
            <span class="metric-label">漏洞</span>
          </div>
          <div class="metric-content">
            <div class="metric-main">
              <span class="metric-value">{{ securityMetrics.totalVulnerabilities }}</span>
              <span class="metric-unit">个</span>
            </div>
            <div class="vulnerability-breakdown">
              <div class="vuln-item critical" v-if="securityMetrics.vulnerabilities.critical > 0">
                <span class="vuln-count">{{ securityMetrics.vulnerabilities.critical }}</span>
                <span class="vuln-label">严重</span>
              </div>
              <div class="vuln-item high" v-if="securityMetrics.vulnerabilities.high > 0">
                <span class="vuln-count">{{ securityMetrics.vulnerabilities.high }}</span>
                <span class="vuln-label">高危</span>
              </div>
              <div class="vuln-item medium" v-if="securityMetrics.vulnerabilities.medium > 0">
                <span class="vuln-count">{{ securityMetrics.vulnerabilities.medium }}</span>
                <span class="vuln-label">中等</span>
              </div>
              <div class="vuln-item low" v-if="securityMetrics.vulnerabilities.low > 0">
                <span class="vuln-count">{{ securityMetrics.vulnerabilities.low }}</span>
                <span class="vuln-label">低危</span>
              </div>
            </div>
          </div>
        </div>

        <div class="metric-card compliance">
          <div class="metric-header">
            <el-icon class="metric-icon compliance-icon">
              <Lock />
            </el-icon>
            <span class="metric-label">合规检查</span>
          </div>
          <div class="metric-content">
            <div class="metric-main">
              <span class="metric-value">{{ compliancePercentage }}%</span>
              <span class="metric-unit">通过</span>
            </div>
            <div class="compliance-details">
              <div class="compliance-item">
                <span class="compliance-label">已通过</span>
                <span class="compliance-count passed">{{ securityMetrics.compliance.passed }}</span>
              </div>
              <div class="compliance-item">
                <span class="compliance-label">失败</span>
                <span class="compliance-count failed">{{ securityMetrics.compliance.failed }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="metric-card last-scan">
          <div class="metric-header">
            <el-icon class="metric-icon scan-icon">
              <Clock />
            </el-icon>
            <span class="metric-label">最后扫描</span>
          </div>
          <div class="metric-content">
            <div class="metric-main">
              <span class="metric-value">{{ formatLastScan(lastScanTime) }}</span>
            </div>
            <div class="scan-details">
              <div class="scan-info">
                <span class="scan-label">扫描时间</span>
                <span class="scan-duration">{{ lastScanDuration }}s</span>
              </div>
              <div class="scan-info">
                <span class="scan-label">容器数</span>
                <span class="scan-count">{{ scannedContainers }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Container Security Status -->
      <div class="container-security">
        <div class="section-header">
          <el-icon class="section-icon"><Box /></el-icon>
          <span class="section-title">容器安全状态</span>
          <el-badge :value="containerSecurityIssues" v-if="containerSecurityIssues > 0" type="danger" />
        </div>

        <div class="container-list">
          <div
            v-for="container in containerSecurity"
            :key="container.id"
            class="container-item"
            :class="getContainerSecurityClass(container.securityLevel)"
            @click="viewContainerSecurity(container.id)"
          >
            <div class="container-info">
              <div class="container-header">
                <span class="container-name">{{ container.name }}</span>
                <el-tag :type="getSecurityTagType(container.securityLevel)" size="small">
                  {{ getSecurityLevelText(container.securityLevel) }}
                </el-tag>
              </div>
              <div class="container-image">{{ container.image }}</div>
            </div>

            <div class="security-indicators">
              <div class="indicator" v-if="container.vulnerabilities.critical > 0">
                <el-icon class="critical-icon"><Warning /></el-icon>
                <span class="indicator-count">{{ container.vulnerabilities.critical }}</span>
              </div>
              <div class="indicator" v-if="container.vulnerabilities.high > 0">
                <el-icon class="high-icon"><InfoFilled /></el-icon>
                <span class="indicator-count">{{ container.vulnerabilities.high }}</span>
              </div>
              <div class="indicator compliance" v-if="!container.isCompliant">
                <el-icon class="compliance-icon"><Lock /></el-icon>
                <span class="indicator-text">不合规</span>
              </div>
            </div>
          </div>

          <div v-if="containerSecurity.length === 0" class="empty-state">
            <el-icon :size="32"><Box /></el-icon>
            <p>暂无容器安全数据</p>
          </div>
        </div>
      </div>

      <!-- Security Recommendations -->
      <div v-if="securityRecommendations.length > 0" class="recommendations">
        <div class="section-header">
          <el-icon class="section-icon"><Star /></el-icon>
          <span class="section-title">安全建议</span>
        </div>

        <div class="recommendations-list">
          <div
            v-for="(recommendation, index) in displayedRecommendations"
            :key="index"
            class="recommendation-item"
            :class="recommendation.priority"
          >
            <div class="recommendation-icon">
              <el-icon>
                <component :is="getRecommendationIcon(recommendation.type)" />
              </el-icon>
            </div>
            <div class="recommendation-content">
              <h4 class="recommendation-title">{{ recommendation.title }}</h4>
              <p class="recommendation-description">{{ recommendation.description }}</p>
            </div>
            <div class="recommendation-actions">
              <el-button
                size="small"
                v-if="recommendation.actionable"
                @click="executeRecommendation(recommendation)"
                type="primary"
              >
                {{ recommendation.actionText || '执行' }}
              </el-button>
              <el-button size="small" type="text" @click="dismissRecommendation(index)">
                忽略
              </el-button>
            </div>
          </div>
        </div>

        <div v-if="securityRecommendations.length > 3" class="show-more">
          <el-button size="small" type="text" @click="showAllRecommendations = !showAllRecommendations">
            {{ showAllRecommendations ? '收起' : `查看全部 ${securityRecommendations.length} 条建议` }}
            <el-icon><component :is="showAllRecommendations ? 'ArrowUp' : 'ArrowDown'" /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- Security Timeline -->
      <div class="security-timeline">
        <div class="section-header">
          <el-icon class="section-icon"><List /></el-icon>
          <span class="section-title">安全事件</span>
        </div>

        <div class="timeline-list">
          <div
            v-for="event in securityEvents"
            :key="event.id"
            class="timeline-item"
            :class="event.severity"
          >
            <div class="timeline-indicator">
              <div class="indicator-dot" />
            </div>

            <div class="timeline-content">
              <div class="event-header">
                <span class="event-title">{{ event.title }}</span>
                <span class="event-time">{{ formatRelativeTime(event.timestamp) }}</span>
              </div>
              <div class="event-details">
                <span class="event-container" v-if="event.containerName">
                  {{ event.containerName }}
                </span>
                <span class="event-description">{{ event.description }}</span>
              </div>
            </div>

            <div class="event-status">
              <el-icon class="status-icon" :class="`status-${event.severity}`">
                <component :is="getEventStatusIcon(event.severity)" />
              </el-icon>
            </div>
          </div>

          <div v-if="securityEvents.length === 0" class="empty-timeline">
            <p>暂无安全事件</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Vulnerability Details Dialog -->
    <el-dialog
      v-model="showVulnerabilityDialog"
      title="漏洞详情"
      width="800px"
      :close-on-click-modal="false"
    >
      <div v-if="selectedVulnerabilities.length > 0" class="vulnerability-dialog-content">
        <div
          v-for="vuln in selectedVulnerabilities"
          :key="vuln.id"
          class="vulnerability-item"
        >
          <div class="vuln-header">
            <span class="vuln-title">{{ vuln.title }}</span>
            <el-tag :type="getVulnerabilityTagType(vuln.severity)" size="small">
              {{ vuln.severity.toUpperCase() }}
            </el-tag>
          </div>
          <div class="vuln-details">
            <p class="vuln-description">{{ vuln.description }}</p>
            <div class="vuln-meta">
              <span class="vuln-cve" v-if="vuln.cve">CVE: {{ vuln.cve }}</span>
              <span class="vuln-score" v-if="vuln.cvssScore">CVSS: {{ vuln.cvssScore }}</span>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showVulnerabilityDialog = false">关闭</el-button>
        <el-button type="primary" @click="fixSelectedVulnerabilities">批量修复</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import {
  Warning,
  Refresh,
  Search,
  Document,
  Lock,
  Clock,
  Box,
  InfoFilled,
  Star,
  List,
  ArrowUp,
  ArrowDown,
  SuccessFilled,
  CircleCloseFilled,
} from '@element-plus/icons-vue';

// Icons for dynamic components
// @ts-ignore: _dynamicIcons is intentionally unused - exists to prevent unused import warnings
const _dynamicIcons = {
  ArrowUp,
  ArrowDown,
  SuccessFilled,
  CircleCloseFilled,
  Warning,
  InfoFilled,
  Lock,
  Star,
};

// Props
interface Props {
  widgetId: string;
  widgetConfig: any;
  widgetData?: any;
  displayMode?: 'default' | 'compact' | 'detailed' | 'minimal';
}

const props = withDefaults(defineProps<Props>(), {
  displayMode: 'default',
});

// Emits
const emit = defineEmits<{
  'data-updated': [data: any];
  error: [error: any];
  loading: [loading: boolean];
}>();

// Types
interface SecurityMetrics {
  totalVulnerabilities: number;
  vulnerabilities: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
  compliance: {
    passed: number;
    failed: number;
    total: number;
  };
}

interface ContainerSecurity {
  id: string;
  name: string;
  image: string;
  securityLevel: 'secure' | 'warning' | 'danger';
  vulnerabilities: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
  isCompliant: boolean;
  lastScan: Date;
}

interface SecurityEvent {
  id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  timestamp: Date;
  containerName?: string;
  resolved: boolean;
}

interface SecurityRecommendation {
  title: string;
  description: string;
  type: 'security' | 'compliance' | 'performance' | 'configuration';
  priority: 'high' | 'medium' | 'low';
  actionable: boolean;
  actionText?: string;
}

interface Vulnerability {
  id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  cve?: string;
  cvssScore?: number;
  containerName: string;
  fixAvailable: boolean;
}

// Reactive state
const isLoading = ref(false);
const isScanning = ref(false);
const error = ref<string | null>(null);
const showAllRecommendations = ref(false);
const showVulnerabilityDialog = ref(false);

// Data state
const overallScore = ref(0);
const securityMetrics = ref<SecurityMetrics>({
  totalVulnerabilities: 0,
  vulnerabilities: {
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
  },
  compliance: {
    passed: 0,
    failed: 0,
    total: 0,
  },
});

const containerSecurity = ref<ContainerSecurity[]>([]);
const securityEvents = ref<SecurityEvent[]>([]);
const securityRecommendations = ref<SecurityRecommendation[]>([]);
const criticalVulnerabilities = ref<Vulnerability[]>([]);
const selectedVulnerabilities = ref<Vulnerability[]>([]);

const lastScanTime = ref<Date | null>(null);
const lastScanDuration = ref(0);
const scannedContainers = ref(0);

// Update intervals
const refreshInterval = ref<NodeJS.Timeout>();

// Computed properties
const hasSecurityData = computed(() => {
  return containerSecurity.value.length > 0 ||
         securityEvents.value.length > 0 ||
         securityMetrics.value.totalVulnerabilities > 0;
});

const compliancePercentage = computed(() => {
  const total = securityMetrics.value.compliance.total;
  if (total === 0) return 100;
  return Math.round((securityMetrics.value.compliance.passed / total) * 100);
});

const containerSecurityIssues = computed(() => {
  return containerSecurity.value.filter(c =>
    c.securityLevel === 'danger' || c.securityLevel === 'warning'
  ).length;
});

const displayedRecommendations = computed(() => {
  if (showAllRecommendations.value || securityRecommendations.value.length <= 3) {
    return securityRecommendations.value;
  }
  return securityRecommendations.value.slice(0, 3);
});

// Methods
const loadSecurityData = async () => {
  try {
    isLoading.value = true;
    emit('loading', true);
    error.value = null;

    // Import API modules dynamically to avoid dependency issues
    const { securityAPI } = await import('@/api/security');

    // Load all security data in parallel
    const [
      securityOverview,
      containerSecurityData,
      securityEventsData,
      recommendationsData,
      criticalVulnData
    ] = await Promise.all([
      securityAPI.getSecurityOverview(),
      securityAPI.getContainerSecurity(),
      securityAPI.getSecurityEvents(10),
      securityAPI.getSecurityRecommendations(),
      securityAPI.getCriticalVulnerabilities(),
    ]);

    // Update state with real API data
    overallScore.value = securityOverview.overallScore || 0;
    securityMetrics.value = {
      totalVulnerabilities: securityOverview.totalVulnerabilities || 0,
      vulnerabilities: securityOverview.vulnerabilities || {
        critical: 0,
        high: 0,
        medium: 0,
        low: 0,
      },
      compliance: securityOverview.compliance || {
        passed: 0,
        failed: 0,
        total: 0,
      },
    };

    containerSecurity.value = (containerSecurityData || []).map((container: any) => ({
      id: container.id,
      name: container.name,
      image: container.image,
      securityLevel: container.securityLevel,
      vulnerabilities: container.vulnerabilities || {
        critical: 0,
        high: 0,
        medium: 0,
        low: 0,
      },
      isCompliant: container.isCompliant ?? true,
      lastScan: new Date(container.lastScan),
    }));

    securityEvents.value = (securityEventsData || []).map((event: any) => ({
      id: event.id,
      title: event.title,
      description: event.description,
      severity: event.severity,
      timestamp: new Date(event.timestamp),
      containerName: event.containerName,
      resolved: event.resolved ?? false,
    }));

    securityRecommendations.value = recommendationsData || [];
    criticalVulnerabilities.value = criticalVulnData || [];

    lastScanTime.value = securityOverview.lastScan ? new Date(securityOverview.lastScan) : null;
    lastScanDuration.value = securityOverview.lastScanDuration || 0;
    scannedContainers.value = securityOverview.scannedContainers || 0;

    emit('data-updated', {
      overallScore: overallScore.value,
      vulnerabilities: securityMetrics.value.totalVulnerabilities,
      criticalIssues: criticalVulnerabilities.value.length,
      compliance: compliancePercentage.value,
    });

  } catch (err: any) {
    console.error('Failed to load security data:', err);
    error.value = err.message || '加载安全数据失败';
    emit('error', err);
  } finally {
    isLoading.value = false;
    emit('loading', false);
  }
};

const startSecurityScan = async () => {
  try {
    isScanning.value = true;

    const { securityAPI } = await import('@/api/security');
    const scanResult = await securityAPI.startSecurityScan();

    ElMessage.success(`安全扫描已开始，扫描ID: ${scanResult.scanId}`);

    // Refresh data after scan starts
    setTimeout(() => {
      loadSecurityData();
    }, 2000);

  } catch (err: any) {
    console.error('Failed to start security scan:', err);
    ElMessage.error('启动安全扫描失败: ' + (err.message || '未知错误'));
  } finally {
    isScanning.value = false;
  }
};

const viewSecurityReport = () => {
  ElMessage.info('跳转到安全报告页面...');
};

const showCriticalVulnerabilities = () => {
  selectedVulnerabilities.value = criticalVulnerabilities.value;
  showVulnerabilityDialog.value = true;
};

const autoFixVulnerabilities = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要自动修复所有严重漏洞吗？这可能会重启相关容器。',
      '确认修复',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    const { securityAPI } = await import('@/api/security');
    const fixResult = await securityAPI.autoFixVulnerabilities(
      criticalVulnerabilities.value.map(v => v.id)
    );

    ElMessage.success(`已启动自动修复，操作ID: ${fixResult.operationId}`);

    // Refresh data after fix starts
    setTimeout(() => {
      loadSecurityData();
    }, 3000);

  } catch (err: any) {
    if (err === 'cancel') return;
    console.error('Failed to auto-fix vulnerabilities:', err);
    ElMessage.error('自动修复失败: ' + (err.message || '未知错误'));
  }
};

const viewContainerSecurity = (containerId: string) => {
  ElMessage.info(`查看容器 ${containerId} 的安全详情...`);
};

const executeRecommendation = async (recommendation: SecurityRecommendation) => {
  try {
    const { securityAPI } = await import('@/api/security');
    await securityAPI.executeRecommendation(recommendation);
    ElMessage.success('安全建议已执行');

    // Refresh data
    await loadSecurityData();
  } catch (err: any) {
    console.error('Failed to execute recommendation:', err);
    ElMessage.error('执行建议失败: ' + (err.message || '未知错误'));
  }
};

const dismissRecommendation = (index: number) => {
  securityRecommendations.value.splice(index, 1);
  ElMessage.info('已忽略该建议');
};

const fixSelectedVulnerabilities = async () => {
  try {
    const { securityAPI } = await import('@/api/security');
    const vulnerabilityIds = selectedVulnerabilities.value.map(v => v.id);

    const fixResult = await securityAPI.fixVulnerabilities(vulnerabilityIds);
    ElMessage.success(`已启动修复，操作ID: ${fixResult.operationId}`);

    showVulnerabilityDialog.value = false;

    // Refresh data after fix starts
    setTimeout(() => {
      loadSecurityData();
    }, 3000);

  } catch (err: any) {
    console.error('Failed to fix vulnerabilities:', err);
    ElMessage.error('修复漏洞失败: ' + (err.message || '未知错误'));
  }
};

const retryLoad = () => {
  loadSecurityData();
};

// Helper functions
const getScoreClass = (score: number): string => {
  if (score >= 80) return 'excellent';
  if (score >= 60) return 'good';
  if (score >= 40) return 'warning';
  return 'danger';
};

const getScoreDescription = (score: number): string => {
  if (score >= 80) return '安全状况良好';
  if (score >= 60) return '安全状况一般';
  if (score >= 40) return '存在安全风险';
  return '存在严重安全风险';
};

const getSecurityTagType = (level: string) => {
  switch (level) {
    case 'secure':
      return 'success';
    case 'warning':
      return 'warning';
    case 'danger':
      return 'danger';
    default:
      return 'info';
  }
};

const getSecurityLevelText = (level: string): string => {
  switch (level) {
    case 'secure':
      return '安全';
    case 'warning':
      return '警告';
    case 'danger':
      return '危险';
    default:
      return '未知';
  }
};

const getContainerSecurityClass = (level: string): string => {
  return `security-${level}`;
};

const getVulnerabilityTagType = (severity: string) => {
  switch (severity) {
    case 'critical':
      return 'danger';
    case 'high':
      return 'danger';
    case 'medium':
      return 'warning';
    case 'low':
      return 'info';
    default:
      return 'info';
  }
};

const getRecommendationIcon = (type: string): string => {
  switch (type) {
    case 'security':
      return 'Lock';
    case 'compliance':
      return 'Lock';
    case 'performance':
      return 'TrendCharts';
    case 'configuration':
      return 'Setting';
    default:
      return 'Star';
  }
};

const getEventStatusIcon = (severity: string): string => {
  switch (severity) {
    case 'critical':
    case 'high':
      return 'CircleCloseFilled';
    case 'medium':
      return 'Warning';
    case 'low':
    case 'info':
      return 'InfoFilled';
    default:
      return 'InfoFilled';
  }
};

const formatLastScan = (date: Date | null): string => {
  if (!date) return '从未';
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  return date.toLocaleDateString();
};

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return date.toLocaleDateString();
};

// Lifecycle hooks
onMounted(async () => {
  await loadSecurityData();

  // Set up periodic refresh
  const refreshIntervalMs = props.widgetConfig?.refreshInterval || 60000; // 1 minute
  if (refreshIntervalMs > 0) {
    refreshInterval.value = setInterval(loadSecurityData, refreshIntervalMs);
  }
});

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
  }
});
</script>

<style scoped lang="scss">
.security-dashboard-widget {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.loading-container,
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  gap: 16px;

  .loading-text {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .error-icon {
    color: var(--el-color-warning);
  }

  .error-title {
    color: var(--el-text-color-primary);
    margin: 0;
  }

  .error-message {
    color: var(--el-text-color-secondary);
    margin: 0;
    font-size: 14px;
  }
}

.security-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  height: 100%;
}

.security-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .security-score {
    display: flex;
    align-items: center;
    gap: 16px;

    .score-circle {
      width: 80px;
      height: 80px;
      border-radius: 50%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      position: relative;
      border: 4px solid;

      &.excellent {
        border-color: var(--el-color-success);
        background: rgba(var(--el-color-success-rgb), 0.1);
      }

      &.good {
        border-color: var(--el-color-primary);
        background: rgba(var(--el-color-primary-rgb), 0.1);
      }

      &.warning {
        border-color: var(--el-color-warning);
        background: rgba(var(--el-color-warning-rgb), 0.1);
      }

      &.danger {
        border-color: var(--el-color-danger);
        background: rgba(var(--el-color-danger-rgb), 0.1);
      }

      .score-value {
        font-size: 24px;
        font-weight: 700;
        color: var(--el-text-color-primary);
        line-height: 1;
      }

      .score-unit {
        font-size: 12px;
        color: var(--el-text-color-secondary);
      }
    }

    .score-info {
      .score-title {
        margin: 0 0 4px 0;
        font-size: 18px;
        color: var(--el-text-color-primary);
      }

      .score-subtitle {
        margin: 0;
        font-size: 14px;
        color: var(--el-text-color-secondary);
      }
    }
  }
}

.critical-alert {
  margin-bottom: 16px;

  .critical-actions {
    display: flex;
    gap: 8px;
    margin-top: 8px;
  }
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;

  .metric-card {
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    padding: 16px;

    .metric-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;

      .metric-icon {
        font-size: 20px;

        &.vulnerability-icon {
          color: var(--el-color-warning);
        }

        &.compliance-icon {
          color: var(--el-color-success);
        }

        &.scan-icon {
          color: var(--el-color-info);
        }
      }

      .metric-label {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    .metric-content {
      .metric-main {
        margin-bottom: 12px;

        .metric-value {
          font-size: 28px;
          font-weight: 700;
          color: var(--el-color-primary);
        }

        .metric-unit {
          font-size: 14px;
          color: var(--el-text-color-secondary);
          margin-left: 4px;
        }
      }

      .vulnerability-breakdown {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;

        .vuln-item {
          display: flex;
          align-items: center;
          gap: 4px;
          padding: 2px 8px;
          border-radius: 12px;
          font-size: 12px;

          &.critical {
            background: rgba(var(--el-color-danger-rgb), 0.1);
            color: var(--el-color-danger);
          }

          &.high {
            background: rgba(var(--el-color-danger-rgb), 0.1);
            color: var(--el-color-danger);
          }

          &.medium {
            background: rgba(var(--el-color-warning-rgb), 0.1);
            color: var(--el-color-warning);
          }

          &.low {
            background: rgba(var(--el-color-info-rgb), 0.1);
            color: var(--el-color-info);
          }

          .vuln-count {
            font-weight: 600;
          }
        }
      }

      .compliance-details,
      .scan-details {
        display: flex;
        flex-direction: column;
        gap: 4px;

        .compliance-item,
        .scan-info {
          display: flex;
          justify-content: space-between;
          font-size: 12px;

          .compliance-label,
          .scan-label {
            color: var(--el-text-color-secondary);
          }

          .compliance-count,
          .scan-duration,
          .scan-count {
            font-weight: 600;

            &.passed {
              color: var(--el-color-success);
            }

            &.failed {
              color: var(--el-color-danger);
            }
          }
        }
      }
    }
  }
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;

  .section-icon {
    color: var(--el-color-primary);
    font-size: 16px;
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.container-security {
  .container-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .container-item {
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      padding: 12px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      cursor: pointer;
      transition: all 0.3s ease;

      &:hover {
        border-color: var(--el-color-primary);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      }

      &.security-secure {
        border-left: 4px solid var(--el-color-success);
      }

      &.security-warning {
        border-left: 4px solid var(--el-color-warning);
      }

      &.security-danger {
        border-left: 4px solid var(--el-color-danger);
      }

      .container-info {
        flex: 1;
        min-width: 0;

        .container-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 4px;

          .container-name {
            font-size: 14px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }
        }

        .container-image {
          font-size: 12px;
          color: var(--el-text-color-secondary);
          font-family: monospace;
        }
      }

      .security-indicators {
        display: flex;
        gap: 8px;
        flex-shrink: 0;

        .indicator {
          display: flex;
          align-items: center;
          gap: 4px;
          padding: 2px 6px;
          border-radius: 12px;
          font-size: 11px;
          background: var(--el-fill-color-light);

          .critical-icon {
            color: var(--el-color-danger);
          }

          .high-icon {
            color: var(--el-color-warning);
          }

          .compliance-icon {
            color: var(--el-color-info);
          }

          .indicator-count,
          .indicator-text {
            font-weight: 600;
          }
        }
      }
    }

    .empty-state {
      text-align: center;
      padding: 32px;
      color: var(--el-text-color-secondary);

      p {
        margin: 8px 0 0 0;
        font-size: 14px;
      }
    }
  }
}

.recommendations {
  .recommendations-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .recommendation-item {
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      padding: 12px;
      display: flex;
      gap: 12px;

      &.high {
        border-left: 4px solid var(--el-color-danger);
      }

      &.medium {
        border-left: 4px solid var(--el-color-warning);
      }

      &.low {
        border-left: 4px solid var(--el-color-info);
      }

      .recommendation-icon {
        flex-shrink: 0;
        width: 32px;
        height: 32px;
        border-radius: 50%;
        background: var(--el-color-primary-light-9);
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--el-color-primary);
      }

      .recommendation-content {
        flex: 1;
        min-width: 0;

        .recommendation-title {
          margin: 0 0 4px 0;
          font-size: 14px;
          color: var(--el-text-color-primary);
        }

        .recommendation-description {
          margin: 0;
          font-size: 12px;
          color: var(--el-text-color-secondary);
          line-height: 1.4;
        }
      }

      .recommendation-actions {
        display: flex;
        gap: 8px;
        flex-shrink: 0;
      }
    }
  }

  .show-more {
    text-align: center;
    margin-top: 8px;
  }
}

.security-timeline {
  .timeline-list {
    position: relative;

    .timeline-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 8px 0;
      position: relative;

      &:not(:last-child)::after {
        content: '';
        position: absolute;
        left: 6px;
        top: 32px;
        width: 1px;
        height: 20px;
        background: var(--el-border-color-light);
      }

      .timeline-indicator {
        flex-shrink: 0;

        .indicator-dot {
          width: 12px;
          height: 12px;
          border-radius: 50%;
          background: var(--el-color-info);
        }
      }

      &.critical .indicator-dot {
        background: var(--el-color-danger);
      }

      &.high .indicator-dot {
        background: var(--el-color-danger);
      }

      &.medium .indicator-dot {
        background: var(--el-color-warning);
      }

      &.low .indicator-dot {
        background: var(--el-color-success);
      }

      .timeline-content {
        flex: 1;
        min-width: 0;

        .event-header {
          display: flex;
          justify-content: space-between;
          align-items: baseline;
          margin-bottom: 4px;

          .event-title {
            font-size: 13px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          .event-time {
            font-size: 11px;
            color: var(--el-text-color-placeholder);
          }
        }

        .event-details {
          display: flex;
          gap: 8px;
          font-size: 12px;
          color: var(--el-text-color-secondary);

          .event-container {
            font-family: monospace;
            background: var(--el-fill-color-light);
            padding: 2px 6px;
            border-radius: 4px;
          }
        }
      }

      .event-status {
        flex-shrink: 0;

        .status-icon {
          font-size: 16px;

          &.status-critical,
          &.status-high {
            color: var(--el-color-danger);
          }

          &.status-medium {
            color: var(--el-color-warning);
          }

          &.status-low,
          &.status-info {
            color: var(--el-color-info);
          }
        }
      }
    }

    .empty-timeline {
      text-align: center;
      padding: 24px;
      color: var(--el-text-color-secondary);
      font-size: 14px;
    }
  }
}

.vulnerability-dialog-content {
  .vulnerability-item {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    padding: 12px;
    margin-bottom: 12px;

    .vuln-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 8px;

      .vuln-title {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    .vuln-details {
      .vuln-description {
        margin: 0 0 8px 0;
        font-size: 13px;
        color: var(--el-text-color-regular);
        line-height: 1.4;
      }

      .vuln-meta {
        display: flex;
        gap: 16px;
        font-size: 12px;
        color: var(--el-text-color-secondary);

        .vuln-cve,
        .vuln-score {
          font-family: monospace;
          background: var(--el-fill-color-light);
          padding: 2px 6px;
          border-radius: 4px;
        }
      }
    }
  }
}

// Responsive design
@media (max-width: 768px) {
  .security-dashboard-widget {
    padding: 12px;
  }

  .security-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .container-item,
  .recommendation-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;

    .security-indicators,
    .recommendation-actions {
      align-self: flex-end;
    }
  }
}
</style>