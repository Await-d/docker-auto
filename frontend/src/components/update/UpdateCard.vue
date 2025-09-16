<template>
  <div
    class="update-card"
    :class="[
      viewMode,
      {
        selected,
        ignored: update.ignored,
        scheduled: update.scheduled,
        'requires-approval': update.requiresApproval,
        'security-update': isSecurityUpdate,
        'critical-update': update.riskLevel === 'critical'
      }
    ]"
  >
    <!-- Selection Checkbox -->
    <div v-if="!update.ignored" class="card-checkbox">
      <el-checkbox
        :model-value="selected"
        @change="$emit('select', update.id)"
      />
    </div>

    <!-- Card Header -->
    <div class="card-header">
      <div class="container-info">
        <div class="container-name">
          <el-icon><Box /></el-icon>
          <span class="name">{{ update.containerName }}</span>
          <el-tag
            v-if="update.scheduled"
            size="small"
            type="info"
            effect="plain"
          >
            <el-icon><Clock /></el-icon>
            Scheduled
          </el-tag>
        </div>

        <div class="image-info">
          <span class="image-name">{{ update.imageName }}</span>
        </div>
      </div>

      <div class="update-status">
        <!-- Risk Level Badge -->
        <el-tag
          :type="getRiskLevelType(update.riskLevel)"
          :effect="update.riskLevel === 'critical' ? 'dark' : 'light'"
          size="small"
        >
          <el-icon>
            <component :is="getRiskIcon(update.riskLevel)" />
          </el-icon>
          {{ update.riskLevel.toUpperCase() }}
        </el-tag>

        <!-- Update Type Badge -->
        <el-tag
          :type="getUpdateTypeColor(update.updateType)"
          size="small"
          effect="plain"
        >
          {{ update.updateType }}
        </el-tag>

        <!-- Security Badge -->
        <el-tag
          v-if="isSecurityUpdate"
          type="danger"
          size="small"
          effect="dark"
        >
          <el-icon><Warning /></el-icon>
          SECURITY
        </el-tag>
      </div>
    </div>

    <!-- Version Information -->
    <div class="version-info">
      <div class="version-change">
        <div class="current-version">
          <span class="version-label">Current</span>
          <el-tag size="small" type="info">{{ update.currentVersion }}</el-tag>
        </div>
        <div class="version-arrow">
          <el-icon><Right /></el-icon>
        </div>
        <div class="new-version">
          <span class="version-label">Available</span>
          <el-tag size="small" type="primary">{{ update.availableVersion }}</el-tag>
        </div>
      </div>

      <div class="update-metadata">
        <div class="metadata-item">
          <el-icon><Calendar /></el-icon>
          <span>{{ formatReleaseDate(update.releaseDate) }}</span>
        </div>
        <div class="metadata-item">
          <el-icon><Folder /></el-icon>
          <span>{{ formatSize(update.size) }}</span>
        </div>
        <div v-if="update.estimatedDowntime > 0" class="metadata-item">
          <el-icon><Timer /></el-icon>
          <span>~{{ formatDuration(update.estimatedDowntime) }}</span>
        </div>
      </div>
    </div>

    <!-- Security Patches -->
    <div v-if="update.securityPatches.length > 0" class="security-patches">
      <h4>Security Patches</h4>
      <div class="patches-list">
        <div
          v-for="patch in update.securityPatches.slice(0, showAllPatches ? undefined : 3)"
          :key="patch.id"
          class="patch-item"
        >
          <div class="patch-header">
            <el-tag
              :type="getPatchSeverityType(patch.severity)"
              size="small"
              effect="dark"
            >
              {{ patch.severity.toUpperCase() }}
            </el-tag>
            <span v-if="patch.cveId" class="cve-id">{{ patch.cveId }}</span>
            <span v-if="patch.score" class="cvss-score">
              CVSS: {{ patch.score.toFixed(1) }}
            </span>
          </div>
          <p class="patch-description">{{ patch.description }}</p>
        </div>

        <el-button
          v-if="update.securityPatches.length > 3"
          text
          type="primary"
          size="small"
          @click="showAllPatches = !showAllPatches"
        >
          {{ showAllPatches ? 'Show Less' : `Show ${update.securityPatches.length - 3} More` }}
        </el-button>
      </div>
    </div>

    <!-- Changelog Preview -->
    <div v-if="update.changelog.length > 0 && expanded" class="changelog-preview">
      <h4>What's New</h4>
      <div class="changelog-items">
        <div
          v-for="item in update.changelog.slice(0, 5)"
          :key="item.description"
          class="changelog-item"
          :class="item.type"
        >
          <div class="change-icon">
            <el-icon>
              <component :is="getChangeIcon(item.type)" />
            </el-icon>
          </div>
          <div class="change-content">
            <span class="change-description">{{ item.description }}</span>
            <el-tag
              v-if="item.breaking"
              type="danger"
              size="small"
              effect="dark"
            >
              BREAKING
            </el-tag>
          </div>
        </div>
      </div>

      <div v-if="update.releaseNotesUrl" class="release-notes-link">
        <el-button
          text
          type="primary"
          size="small"
          @click="openReleaseNotes"
        >
          <el-icon><Link /></el-icon>
          View Full Release Notes
        </el-button>
      </div>
    </div>

    <!-- Dependencies and Conflicts -->
    <div v-if="(update.dependencies.length > 0 || update.conflicts.length > 0) && expanded" class="dependencies-section">
      <div v-if="update.dependencies.length > 0" class="dependencies">
        <h5>Dependencies</h5>
        <div class="dependency-list">
          <el-tag
            v-for="dep in update.dependencies"
            :key="dep"
            size="small"
            type="warning"
            effect="plain"
          >
            <el-icon><Connection /></el-icon>
            {{ getContainerName(dep) }}
          </el-tag>
        </div>
      </div>

      <div v-if="update.conflicts.length > 0" class="conflicts">
        <h5>Conflicts</h5>
        <div class="conflict-list">
          <el-tag
            v-for="conflict in update.conflicts"
            :key="conflict"
            size="small"
            type="danger"
            effect="plain"
          >
            <el-icon><CircleClose /></el-icon>
            {{ getContainerName(conflict) }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- Ignored Information -->
    <div v-if="update.ignored" class="ignored-info">
      <div class="ignored-banner">
        <el-icon><CircleClose /></el-icon>
        <span>This update has been ignored</span>
      </div>
      <div v-if="update.ignoredReason" class="ignored-reason">
        <strong>Reason:</strong> {{ update.ignoredReason }}
      </div>
      <div class="ignored-date">
        <strong>Ignored on:</strong> {{ formatDateTime(update.ignoredAt!) }}
      </div>
    </div>

    <!-- Approval Information -->
    <div v-if="update.requiresApproval" class="approval-info">
      <div class="approval-banner">
        <el-icon><Lock /></el-icon>
        <span>Requires approval before update</span>
        <el-tag
          v-if="update.approvalStatus"
          :type="getApprovalStatusType(update.approvalStatus)"
          size="small"
        >
          {{ update.approvalStatus }}
        </el-tag>
      </div>
    </div>

    <!-- Card Footer / Actions -->
    <div class="card-footer">
      <div class="action-buttons">
        <!-- Primary Actions -->
        <div class="primary-actions">
          <el-button
            v-if="!update.ignored && !update.scheduled"
            type="primary"
            :loading="loading"
            :disabled="update.requiresApproval && update.approvalStatus !== 'approved'"
            @click="$emit('update', update.id)"
          >
            <el-icon><UpdateFilled /></el-icon>
            Update Now
          </el-button>

          <el-button
            v-if="update.ignored"
            type="primary"
            @click="$emit('unignore', update.id)"
          >
            <el-icon><Check /></el-icon>
            Unignore
          </el-button>

          <el-button
            v-if="update.scheduled && update.scheduledAt"
            type="warning"
            @click="$emit('reschedule', update.id)"
          >
            <el-icon><Clock /></el-icon>
            Reschedule
          </el-button>
        </div>

        <!-- Secondary Actions -->
        <div class="secondary-actions">
          <el-dropdown trigger="click" placement="top-end">
            <el-button :icon="MoreFilled" circle />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  @click="$emit('details', update.id)"
                >
                  <el-icon><View /></el-icon>
                  View Details
                </el-dropdown-item>
                <el-dropdown-item
                  @click="$emit('compare', update.id)"
                >
                  <el-icon><Compare /></el-icon>
                  Compare Versions
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="!update.ignored && !update.scheduled"
                  @click="$emit('schedule', update.id)"
                >
                  <el-icon><Calendar /></el-icon>
                  Schedule Update
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="!update.ignored"
                  divided
                  @click="$emit('ignore', update.id)"
                >
                  <el-icon><CircleClose /></el-icon>
                  Ignore Update
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <!-- Expand/Collapse Toggle -->
      <div class="expand-toggle">
        <el-button
          text
          :icon="expanded ? 'ArrowUp' : 'ArrowDown'"
          @click="toggleExpanded"
        >
          {{ expanded ? 'Less Info' : 'More Info' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Box,
  Clock,
  Warning,
  Right,
  Calendar,
  Folder,
  Timer,
  Link,
  Connection,
  CircleClose,
  Lock,
  UpdateFilled,
  Check,
  MoreFilled,
  View,
  Compare
} from '@element-plus/icons-vue'

// Types
import type { ContainerUpdate } from '@/types/updates'

// Props
interface Props {
  update: ContainerUpdate
  selected?: boolean
  loading?: boolean
  viewMode?: 'list' | 'grid'
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
  loading: false,
  viewMode: 'list'
})

// Emits
defineEmits<{
  select: [updateId: string]
  update: [updateId: string]
  schedule: [updateId: string]
  ignore: [updateId: string]
  unignore: [updateId: string]
  compare: [updateId: string]
  details: [updateId: string]
  reschedule: [updateId: string]
}>()

// Local state
const expanded = ref(false)
const showAllPatches = ref(false)

// Computed
const isSecurityUpdate = computed(() =>
  props.update.updateType === 'security' || props.update.securityPatches.length > 0
)

// Methods
const toggleExpanded = () => {
  expanded.value = !expanded.value
}

const openReleaseNotes = () => {
  if (props.update.releaseNotesUrl) {
    window.open(props.update.releaseNotesUrl, '_blank')
  }
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

const getRiskIcon = (riskLevel: string) => {
  switch (riskLevel) {
    case 'low': return 'Check'
    case 'medium': return 'Warning'
    case 'high': return 'Close'
    case 'critical': return 'CircleClose'
    default: return 'InfoFilled'
  }
}

const getUpdateTypeColor = (updateType: string) => {
  switch (updateType) {
    case 'major': return 'danger'
    case 'minor': return 'warning'
    case 'patch': return 'success'
    case 'security': return 'danger'
    case 'hotfix': return 'warning'
    default: return 'info'
  }
}

const getPatchSeverityType = (severity: string) => {
  switch (severity) {
    case 'low': return 'success'
    case 'medium': return 'warning'
    case 'high': return 'danger'
    case 'critical': return 'danger'
    default: return 'info'
  }
}

const getChangeIcon = (type: string) => {
  switch (type) {
    case 'added': return 'Plus'
    case 'changed': return 'Edit'
    case 'deprecated': return 'Warning'
    case 'removed': return 'Minus'
    case 'fixed': return 'Tools'
    case 'security': return 'Lock'
    default: return 'InfoFilled'
  }
}

const getApprovalStatusType = (status: string) => {
  switch (status) {
    case 'approved': return 'success'
    case 'rejected': return 'danger'
    case 'pending': return 'warning'
    default: return 'info'
  }
}

const getContainerName = (containerId: string) => {
  // This would typically lookup the container name from a store
  // For now, return a truncated ID
  return containerId.substring(0, 8)
}

const formatReleaseDate = (dateString: string) => {
  const date = new Date(dateString)
  const now = new Date()
  const diffTime = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24))

  if (diffDays === 0) return 'Today'
  if (diffDays === 1) return 'Yesterday'
  if (diffDays < 7) return `${diffDays} days ago`
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`
  if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`

  return date.toLocaleDateString()
}

const formatSize = (bytes: number) => {
  const sizes = ['B', 'KB', 'MB', 'GB']
  if (bytes === 0) return '0 B'
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`
}

const formatDuration = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

const formatDateTime = (dateString: string) => {
  return new Date(dateString).toLocaleString()
}
</script>

<style scoped lang="scss">
.update-card {
  position: relative;
  padding: 20px;
  background: var(--el-bg-color);
  border: 2px solid var(--el-border-color);
  border-radius: 12px;
  transition: all 0.3s ease;
  overflow: hidden;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    transform: translateY(-2px);
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
  }

  &.selected {
    border-color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &.ignored {
    opacity: 0.7;
    background: var(--el-color-info-light-9);
    border-color: var(--el-color-info-light-5);

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 4px;
      background: var(--el-color-info);
    }
  }

  &.scheduled {
    border-color: var(--el-color-warning-light-5);
    background: var(--el-color-warning-light-9);

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 4px;
      background: var(--el-color-warning);
    }
  }

  &.security-update {
    border-color: var(--el-color-danger-light-5);

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 4px;
      background: linear-gradient(90deg, var(--el-color-danger), var(--el-color-warning));
    }
  }

  &.critical-update {
    border-color: var(--el-color-danger);
    box-shadow: 0 0 0 1px var(--el-color-danger-light-8);

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 4px;
      background: var(--el-color-danger);
      animation: pulse 2s infinite;
    }
  }

  &.requires-approval {
    &::after {
      content: '🔒';
      position: absolute;
      top: 12px;
      right: 12px;
      font-size: 18px;
      opacity: 0.6;
    }
  }

  &.grid {
    min-height: 280px;
    display: flex;
    flex-direction: column;
  }

  &.list {
    .card-header {
      margin-bottom: 12px;
    }

    .version-info {
      margin-bottom: 12px;
    }
  }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.card-checkbox {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 2;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding-left: 32px; // Space for checkbox
}

.container-info {
  flex: 1;

  .container-name {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;

    .name {
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .image-info {
    .image-name {
      font-size: 14px;
      color: var(--el-text-color-regular);
      font-family: monospace;
    }
  }
}

.update-status {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.version-info {
  margin-bottom: 16px;

  .version-change {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
    justify-content: center;
    padding: 12px;
    background: var(--el-bg-color-page);
    border-radius: 8px;
    border: 1px solid var(--el-border-color-lighter);

    .current-version,
    .new-version {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 4px;
    }

    .version-label {
      font-size: 12px;
      color: var(--el-text-color-regular);
      font-weight: 500;
    }

    .version-arrow {
      color: var(--el-color-primary);
      font-size: 18px;
    }
  }

  .update-metadata {
    display: flex;
    justify-content: center;
    gap: 16px;
    flex-wrap: wrap;
  }

  .metadata-item {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--el-text-color-regular);
  }
}

.security-patches {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: 6px;

  h4 {
    margin: 0 0 8px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-color-danger);
    display: flex;
    align-items: center;
    gap: 6px;

    &::before {
      content: '🔒';
    }
  }

  .patches-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .patch-item {
    padding: 8px;
    background: var(--el-bg-color);
    border-radius: 4px;
    border: 1px solid var(--el-color-danger-light-8);
  }

  .patch-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;

    .cve-id {
      font-family: monospace;
      font-size: 11px;
      padding: 2px 4px;
      background: var(--el-color-danger-light-8);
      border-radius: 2px;
      color: var(--el-color-danger);
    }

    .cvss-score {
      font-size: 11px;
      color: var(--el-text-color-regular);
    }
  }

  .patch-description {
    margin: 0;
    font-size: 12px;
    color: var(--el-text-color-regular);
    line-height: 1.4;
  }
}

.changelog-preview {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;

  h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .changelog-items {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 12px;
  }

  .changelog-item {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 6px 0;

    .change-icon {
      width: 16px;
      height: 16px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 10px;
      color: white;
      flex-shrink: 0;
      margin-top: 2px;

      .el-icon {
        font-size: 10px;
      }
    }

    &.added .change-icon {
      background: var(--el-color-success);
    }

    &.changed .change-icon {
      background: var(--el-color-warning);
    }

    &.fixed .change-icon {
      background: var(--el-color-primary);
    }

    &.deprecated .change-icon {
      background: var(--el-color-warning);
    }

    &.removed .change-icon {
      background: var(--el-color-danger);
    }

    &.security .change-icon {
      background: var(--el-color-danger);
    }

    .change-content {
      flex: 1;
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;

      .change-description {
        font-size: 13px;
        color: var(--el-text-color-regular);
        line-height: 1.4;
      }
    }
  }

  .release-notes-link {
    padding-top: 8px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

.dependencies-section {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-bg-color-page);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;

  h5 {
    margin: 0 0 8px 0;
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .dependencies {
    margin-bottom: 12px;

    &:last-child {
      margin-bottom: 0;
    }

    h5 {
      color: var(--el-color-warning);
    }
  }

  .conflicts {
    h5 {
      color: var(--el-color-danger);
    }
  }

  .dependency-list,
  .conflict-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
}

.ignored-info,
.approval-info {
  margin-bottom: 16px;
  padding: 12px;
  border-radius: 6px;
}

.ignored-info {
  background: var(--el-color-info-light-9);
  border: 1px solid var(--el-color-info-light-7);

  .ignored-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--el-color-info);
    font-weight: 600;
    margin-bottom: 8px;
  }

  .ignored-reason,
  .ignored-date {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-bottom: 4px;
  }
}

.approval-info {
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-7);

  .approval-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--el-color-warning-dark-2);
    font-weight: 600;
  }
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.primary-actions {
  display: flex;
  gap: 8px;
}

.secondary-actions {
  margin-left: auto;
}

.expand-toggle {
  margin-left: 12px;
}

@media (max-width: 768px) {
  .update-card {
    padding: 16px;

    &.grid {
      min-height: auto;
    }

    .card-header {
      flex-direction: column;
      gap: 12px;
      align-items: stretch;
      padding-left: 24px;
    }

    .update-status {
      align-items: flex-start;
      flex-direction: row;
      flex-wrap: wrap;
    }

    .version-info {
      .update-metadata {
        justify-content: flex-start;
      }
    }

    .card-footer {
      flex-direction: column;
      gap: 12px;
      align-items: stretch;

      .action-buttons {
        justify-content: center;
      }

      .expand-toggle {
        margin-left: 0;
        text-align: center;
      }
    }
  }
}
</style>