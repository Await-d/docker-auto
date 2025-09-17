<template>
  <div class="update-manager">
    <!-- Update Status -->
    <div class="update-status">
      <div class="status-header">
        <div class="status-info">
          <el-icon class="status-icon" :class="statusIconClass">
            <component :is="statusIcon" />
          </el-icon>
          <div class="status-text">
            <h3>{{ statusTitle }}</h3>
            <p>{{ statusDescription }}</p>
          </div>
        </div>
        <div class="status-actions">
          <el-button
            v-if="!updateAvailable"
            :loading="checking"
            size="small"
            @click="checkForUpdates"
          >
            <el-icon><Refresh /></el-icon>
            Check for Updates
          </el-button>
        </div>
      </div>

      <!-- Update Available -->
      <div v-if="updateAvailable" class="update-available">
        <div class="update-info">
          <div class="version-comparison">
            <div class="version-item current">
              <span class="version-label">Current Version</span>
              <span class="version-value">{{ currentVersion }}</span>
            </div>
            <el-icon class="arrow-icon">
              <ArrowRight />
            </el-icon>
            <div class="version-item new">
              <span class="version-label">Available Version</span>
              <span class="version-value">{{
                availableUpdate!.availableVersion
              }}</span>
            </div>
          </div>

          <div class="update-details">
            <div class="detail-item">
              <span class="detail-label">Published:</span>
              <span class="detail-value">{{
                formatDate(availableUpdate!.publishedAt)
              }}</span>
            </div>
            <div
v-if="availableUpdate!.size" class="detail-item"
>
              <span class="detail-label">Size:</span>
              <span class="detail-value">{{
                formatBytes(availableUpdate!.size)
              }}</span>
            </div>
            <div
v-if="availableUpdate!.critical" class="detail-item"
>
              <span class="detail-label">Priority:</span>
              <el-tag
type="danger" size="small"> Critical </el-tag>
            </div>
          </div>
        </div>

        <!-- Update Actions -->
        <div class="update-actions">
          <el-button
            type="primary"
            :disabled="updating"
            @click="showUpdateDialog = true"
          >
            <el-icon><Download /></el-icon>
            Update Now
          </el-button>
          <el-button :disabled="updating"
@click="scheduleUpdate">
            <el-icon><Clock /></el-icon>
            Schedule Update
          </el-button>
          <el-dropdown @command="handleMoreAction">
            <el-button>
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="release-notes">
                  <el-icon><Document /></el-icon>
                  View Release Notes
                </el-dropdown-item>
                <el-dropdown-item command="ignore">
                  <el-icon><Close /></el-icon>
                  Ignore This Update
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <!-- Update History -->
    <div class="update-history">
      <div class="history-header">
        <h4>Update History</h4>
        <el-button size="small" @click="refreshHistory">
          <el-icon><Refresh /></el-icon>
          Refresh
        </el-button>
      </div>

      <div class="history-timeline">
        <div v-if="updateHistory.length === 0" class="no-history">
          <el-icon class="no-history-icon">
            <DocumentRemove />
          </el-icon>
          <p>No update history available</p>
        </div>

        <div
          v-for="(entry, index) in updateHistory"
          :key="index"
          class="timeline-item"
          :class="{
            'timeline-success': entry.status === 'success',
            'timeline-failed': entry.status === 'failed',
            'timeline-pending': entry.status === 'pending',
          }"
        >
          <div class="timeline-dot">
            <el-icon>
              <component :is="getHistoryIcon(entry.status)" />
            </el-icon>
          </div>
          <div class="timeline-content">
            <div class="timeline-header">
              <div class="timeline-title">
                {{ entry.fromVersion }} → {{ entry.toVersion }}
              </div>
              <div class="timeline-date">
                {{ formatDateTime(entry.timestamp) }}
              </div>
            </div>
            <div class="timeline-details">
              <div class="timeline-status">
                <el-tag
:type="getStatusTagType(entry.status)" size="small"
>
                  {{ formatStatus(entry.status) }}
                </el-tag>
                <span v-if="entry.duration" class="timeline-duration">
                  {{ formatDuration(entry.duration) }}
                </span>
              </div>
              <div v-if="entry.message" class="timeline-message">
                {{ entry.message }}
              </div>
            </div>
            <div v-if="entry.status === 'failed'" class="timeline-actions">
              <el-button size="small" @click="retryUpdate(entry)">
                <el-icon><Refresh /></el-icon>
                Retry
              </el-button>
              <el-button size="small" @click="viewLogs(entry)">
                <el-icon><Document /></el-icon>
                View Logs
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Update Settings -->
    <div class="update-settings">
      <div class="settings-header">
        <h4>Update Settings</h4>
        <el-button size="small" @click="showSettingsDialog = true">
          <el-icon><Setting /></el-icon>
          Configure
        </el-button>
      </div>

      <div class="settings-summary">
        <div class="setting-item">
          <span class="setting-label">Auto Update:</span>
          <el-tag :type="autoUpdateEnabled ? 'success' : 'info'">
            {{ autoUpdateEnabled ? "Enabled" : "Disabled" }}
          </el-tag>
        </div>
        <div
v-if="updateSchedule" class="setting-item"
>
          <span class="setting-label">Schedule:</span>
          <span class="setting-value">{{ updateSchedule }}</span>
        </div>
        <div class="setting-item">
          <span class="setting-label">Strategy:</span>
          <span class="setting-value">{{ updateStrategy }}</span>
        </div>
      </div>
    </div>

    <!-- Update Dialog -->
    <el-dialog
      v-model="showUpdateDialog"
      title="Update Container"
      width="600px"
      :before-close="handleUpdateDialogClose"
    >
      <div class="update-dialog-content">
        <div class="update-confirmation">
          <div class="confirmation-icon">
            <el-icon><Download /></el-icon>
          </div>
          <div class="confirmation-text">
            <h3>Confirm Update</h3>
            <p>
              Are you sure you want to update {{ containerName }} from
              {{ currentVersion }} to {{ availableUpdate?.availableVersion }}?
            </p>
          </div>
        </div>

        <div class="update-options">
          <h4>Update Options</h4>
          <el-form :model="updateOptions" label-width="140px">
            <el-form-item label="Update Strategy">
              <el-select v-model="updateOptions.strategy">
                <el-option label="Recreate" value="recreate" />
                <el-option label="Rolling Update" value="rolling" />
                <el-option label="Blue-Green" value="blue-green" />
              </el-select>
            </el-form-item>

            <el-form-item label="Pull Policy">
              <el-select v-model="updateOptions.pullPolicy">
                <el-option label="Always" value="always" />
                <el-option label="If Not Present" value="missing" />
                <el-option label="Never" value="never" />
              </el-select>
            </el-form-item>

            <el-form-item>
              <el-checkbox v-model="updateOptions.preserveVolumes">
                Preserve volumes
              </el-checkbox>
            </el-form-item>

            <el-form-item>
              <el-checkbox v-model="updateOptions.recreate">
                Force recreate container
              </el-checkbox>
            </el-form-item>
          </el-form>
        </div>

        <!-- Update Progress -->
        <div v-if="updating" class="update-progress">
          <el-progress
            :percentage="updateProgress"
            :status="updateStatus === 'failed' ? 'exception' : undefined"
          />
          <p class="progress-text">
            {{ updateStatusText }}
          </p>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button
:disabled="updating" @click="showUpdateDialog = false"
>
            Cancel
          </el-button>
          <el-button
            type="primary"
            :loading="updating"
            :disabled="!availableUpdate"
            @click="performUpdate"
          >
            <el-icon><Download /></el-icon>
            Update Container
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Settings Dialog -->
    <el-dialog
      v-model="showSettingsDialog"
      title="Update Settings"
      width="500px"
    >
      <el-form :model="settings" label-width="120px">
        <el-form-item label="Auto Update">
          <el-switch v-model="settings.autoUpdate" />
        </el-form-item>

        <el-form-item label="Strategy">
          <el-select v-model="settings.strategy">
            <el-option label="Recreate" value="recreate" />
            <el-option label="Rolling" value="rolling" />
            <el-option label="Blue-Green" value="blue-green" />
          </el-select>
        </el-form-item>

        <el-form-item
v-if="settings.autoUpdate" label="Schedule"
>
          <el-input
            v-model="settings.schedule"
            placeholder="0 2 * * 0 (Every Sunday at 2 AM)"
          />
        </el-form-item>

        <el-form-item>
          <el-checkbox v-model="settings.notifyOnUpdate">
            Send notifications
          </el-checkbox>
        </el-form-item>

        <el-form-item>
          <el-checkbox v-model="settings.rollbackOnFailure">
            Auto rollback on failure
          </el-checkbox>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showSettingsDialog = false"> Cancel </el-button>
          <el-button type="primary" @click="saveSettings">
            Save Settings
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Release Notes Dialog -->
    <el-dialog
      v-model="showReleaseNotesDialog"
      title="Release Notes"
      width="70%"
    >
      <div class="release-notes-content">
        <div v-if="availableUpdate?.releaseNotes">
          <div
            class="release-notes"
            v-html="formatReleaseNotes(availableUpdate.releaseNotes)"
          />
        </div>
        <div v-else class="no-release-notes">
          <p>No release notes available for this update.</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { storeToRefs } from "pinia";
import { ElMessage, ElMessageBox, ElNotification } from "element-plus";
import {
  Refresh,
  ArrowRight,
  Download,
  Clock,
  MoreFilled,
  Document,
  Close,
  DocumentRemove,
  Setting,
  SuccessFilled,
  CircleCloseFilled,
  Warning,
  Loading,
} from "@element-plus/icons-vue";

import { useContainerStore } from "@/store/containers";

interface Props {
  containerId: string;
  containerName?: string;
  currentVersion: string;
}

interface UpdateHistoryEntry {
  fromVersion: string;
  toVersion: string;
  timestamp: Date;
  status: "success" | "failed" | "pending";
  duration?: number;
  message?: string;
}

interface UpdateOptions {
  strategy: "recreate" | "rolling" | "blue-green";
  pullPolicy: "always" | "missing" | "never";
  preserveVolumes: boolean;
  recreate: boolean;
}

interface UpdateSettings {
  autoUpdate: boolean;
  strategy: "recreate" | "rolling" | "blue-green";
  schedule?: string;
  notifyOnUpdate: boolean;
  rollbackOnFailure: boolean;
}

const props = defineProps<Props>();

const containerStore = useContainerStore();
const { availableUpdates } = storeToRefs(containerStore);

// Local state
const checking = ref(false);
const updating = ref(false);
const updateProgress = ref(0);
const updateStatus = ref<"pending" | "success" | "failed">("pending");
const updateStatusText = ref("");
const showUpdateDialog = ref(false);
const showSettingsDialog = ref(false);
const showReleaseNotesDialog = ref(false);

// Mock data for demonstration
const updateHistory = ref<UpdateHistoryEntry[]>([
  {
    fromVersion: "1.0.0",
    toVersion: "1.1.0",
    timestamp: new Date(Date.now() - 24 * 60 * 60 * 1000),
    status: "success",
    duration: 120,
    message: "Update completed successfully",
  },
  {
    fromVersion: "0.9.5",
    toVersion: "1.0.0",
    timestamp: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000),
    status: "success",
    duration: 180,
  },
]);

const updateOptions = ref<UpdateOptions>({
  strategy: "recreate",
  pullPolicy: "always",
  preserveVolumes: true,
  recreate: false,
});

const settings = ref<UpdateSettings>({
  autoUpdate: false,
  strategy: "recreate",
  schedule: "0 2 * * 0",
  notifyOnUpdate: true,
  rollbackOnFailure: true,
});

// Computed
const availableUpdate = computed(() => {
  return availableUpdates.value.find(
    (update) => update.container === props.containerId,
  );
});

const updateAvailable = computed(() => !!availableUpdate.value);

const statusIcon = computed(() => {
  if (updateAvailable.value && availableUpdate.value) {
    return availableUpdate.value.critical ? Warning : Download;
  }
  return SuccessFilled;
});

const statusIconClass = computed(() => {
  if (updateAvailable.value && availableUpdate.value) {
    return availableUpdate.value.critical ? "status-critical" : "status-update";
  }
  return "status-up-to-date";
});

const statusTitle = computed(() => {
  if (updateAvailable.value && availableUpdate.value) {
    return availableUpdate.value.critical
      ? "Critical Update Available"
      : "Update Available";
  }
  return "Up to Date";
});

const statusDescription = computed(() => {
  if (updateAvailable.value && availableUpdate.value) {
    return `Version ${availableUpdate.value.availableVersion} is available`;
  }
  return "Container is running the latest version";
});

const autoUpdateEnabled = computed(() => settings.value.autoUpdate);
const updateSchedule = computed(() => settings.value.schedule);
const updateStrategy = computed(() => settings.value.strategy);

// Methods
function formatDate(date: Date | string): string {
  return new Date(date).toLocaleDateString();
}

function formatDateTime(date: Date | string): string {
  return new Date(date).toLocaleString();
}

function formatDuration(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`;
}

function formatStatus(status: string): string {
  const statusMap: Record<string, string> = {
    success: "Success",
    failed: "Failed",
    pending: "In Progress",
  };
  return statusMap[status] || status;
}

function getStatusTagType(
  status: string,
): "success" | "info" | "warning" | "danger" {
  const typeMap: Record<string, "success" | "info" | "warning" | "danger"> = {
    success: "success",
    failed: "danger",
    pending: "warning",
  };
  return typeMap[status] || "info";
}

function getHistoryIcon(status: string) {
  const iconMap: Record<string, any> = {
    success: SuccessFilled,
    failed: CircleCloseFilled,
    pending: Loading,
  };
  return iconMap[status] || Warning;
}

async function checkForUpdates() {
  checking.value = true;
  try {
    await containerStore.checkUpdates(props.containerId);
    ElMessage.success("Update check completed");
  } catch (error) {
    console.error("Failed to check for updates:", error);
    ElMessage.error("Failed to check for updates");
  } finally {
    checking.value = false;
  }
}

async function performUpdate() {
  if (!availableUpdate.value) return;

  updating.value = true;
  updateProgress.value = 0;
  updateStatus.value = "pending";
  updateStatusText.value = "Starting update...";

  try {
    // Simulate update progress
    const progressSteps = [
      { progress: 10, text: "Pulling new image..." },
      { progress: 30, text: "Stopping container..." },
      { progress: 50, text: "Creating new container..." },
      { progress: 70, text: "Starting container..." },
      { progress: 90, text: "Verifying update..." },
      { progress: 100, text: "Update completed!" },
    ];

    for (const step of progressSteps) {
      updateProgress.value = step.progress;
      updateStatusText.value = step.text;
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }

    await containerStore.updateContainerImage(
      props.containerId,
      updateOptions.value,
    );

    updateStatus.value = "success";
    updateStatusText.value = "Update completed successfully!";

    // Add to history
    updateHistory.value.unshift({
      fromVersion: props.currentVersion,
      toVersion: availableUpdate.value.availableVersion,
      timestamp: new Date(),
      status: "success",
      duration: 6,
      message: "Update completed successfully",
    });

    ElNotification({
      title: "Update Complete",
      message: `Container has been updated to ${availableUpdate.value.availableVersion}`,
      type: "success",
    });

    // Close dialog after success
    setTimeout(() => {
      showUpdateDialog.value = false;
    }, 2000);
  } catch (error) {
    console.error("Update failed:", error);
    updateStatus.value = "failed";
    updateStatusText.value = "Update failed!";

    updateHistory.value.unshift({
      fromVersion: props.currentVersion,
      toVersion: availableUpdate.value.availableVersion,
      timestamp: new Date(),
      status: "failed",
      message: "Update failed: " + (error as any).message,
    });

    ElMessage.error("Update failed");
  } finally {
    updating.value = false;
  }
}

function scheduleUpdate() {
  ElNotification({
    title: "Update Scheduled",
    message: "Update has been scheduled for the next maintenance window",
    type: "info",
  });
}

function handleMoreAction(command: string) {
  switch (command) {
    case "release-notes":
      showReleaseNotesDialog.value = true;
      break;
    case "ignore":
      ignoreUpdate();
      break;
  }
}

function ignoreUpdate() {
  ElMessageBox.confirm(
    "Are you sure you want to ignore this update? You will not be notified about it again.",
    "Ignore Update",
    {
      type: "warning",
      confirmButtonText: "Ignore",
      cancelButtonText: "Cancel",
    },
  ).then(() => {
    // Remove from available updates
    const index = availableUpdates.value.findIndex(
      (u) => u.container === props.containerId,
    );
    if (index !== -1) {
      availableUpdates.value.splice(index, 1);
    }
    ElMessage.success("Update ignored");
  });
}

function retryUpdate(entry: UpdateHistoryEntry) {
  ElMessageBox.confirm(
    `Retry updating from ${entry.fromVersion} to ${entry.toVersion}?`,
    "Retry Update",
    {
      type: "info",
      confirmButtonText: "Retry",
      cancelButtonText: "Cancel",
    },
  ).then(() => {
    showUpdateDialog.value = true;
  });
}

function viewLogs(_entry: UpdateHistoryEntry) {
  ElNotification({
    title: "View Logs",
    message: "Update logs functionality will be implemented here",
    type: "info",
  });
}

function refreshHistory() {
  ElMessage.success("Update history refreshed");
}

function saveSettings() {
  // Save settings logic would go here
  ElMessage.success("Settings saved successfully");
  showSettingsDialog.value = false;
}

function formatReleaseNotes(notes: string): string {
  // Convert markdown-like text to HTML
  return notes
    .replace(/## (.*)/g, "<h3>$1</h3>")
    .replace(/### (.*)/g, "<h4>$1</h4>")
    .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
    .replace(/\*(.*?)\*/g, "<em>$1</em>")
    .replace(/\n/g, "<br>");
}

function handleUpdateDialogClose(done: () => void) {
  if (updating.value) {
    ElMessageBox.confirm(
      "Update is in progress. Are you sure you want to close?",
    )
      .then(() => done())
      .catch(() => {});
  } else {
    done();
  }
}

// Lifecycle
onMounted(() => {
  checkForUpdates();
});

// Watch for container changes
watch(
  () => props.containerId,
  () => {
    checkForUpdates();
  },
);
</script>

<style scoped>
.update-manager {
  background: white;
  border-radius: 8px;
  overflow: hidden;
}

.update-status {
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;
}

.status-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.status-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.status-icon {
  font-size: 24px;
  margin-top: 4px;
}

.status-icon.status-up-to-date {
  color: #67c23a;
}

.status-icon.status-update {
  color: #409eff;
}

.status-icon.status-critical {
  color: #f56c6c;
}

.status-text h3 {
  margin: 0 0 4px 0;
  color: #303133;
}

.status-text p {
  margin: 0;
  color: #606266;
  font-size: 14px;
}

.update-available {
  background: #f0f9ff;
  border: 1px solid #b3d8ff;
  border-radius: 8px;
  padding: 16px;
  margin-top: 16px;
}

.update-info {
  margin-bottom: 16px;
}

.version-comparison {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.version-item {
  text-align: center;
}

.version-label {
  display: block;
  font-size: 12px;
  color: #606266;
  margin-bottom: 4px;
}

.version-value {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.version-item.current .version-value {
  color: #909399;
}

.version-item.new .version-value {
  color: #409eff;
}

.arrow-icon {
  font-size: 20px;
  color: #409eff;
}

.update-details {
  display: flex;
  gap: 24px;
  font-size: 12px;
}

.detail-item {
  display: flex;
  gap: 4px;
}

.detail-label {
  color: #606266;
  font-weight: 500;
}

.detail-value {
  color: #303133;
}

.update-actions {
  display: flex;
  gap: 8px;
}

.update-history {
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.history-header h4 {
  margin: 0;
  color: #303133;
}

.history-timeline {
  position: relative;
}

.no-history {
  text-align: center;
  padding: 40px 20px;
  color: #909399;
}

.no-history-icon {
  font-size: 48px;
  margin-bottom: 8px;
  color: #c0c4cc;
}

.timeline-item {
  position: relative;
  padding-left: 40px;
  margin-bottom: 20px;
}

.timeline-item:before {
  content: "";
  position: absolute;
  left: 15px;
  top: 30px;
  bottom: -20px;
  width: 2px;
  background: #e4e7ed;
}

.timeline-item:last-child:before {
  display: none;
}

.timeline-dot {
  position: absolute;
  left: 0;
  top: 8px;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: white;
}

.timeline-success .timeline-dot {
  background: #67c23a;
}

.timeline-failed .timeline-dot {
  background: #f56c6c;
}

.timeline-pending .timeline-dot {
  background: #e6a23c;
}

.timeline-content {
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 12px;
}

.timeline-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.timeline-title {
  font-weight: 600;
  color: #303133;
}

.timeline-date {
  font-size: 12px;
  color: #909399;
}

.timeline-details {
  margin-bottom: 8px;
}

.timeline-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.timeline-duration {
  font-size: 12px;
  color: #606266;
}

.timeline-message {
  font-size: 12px;
  color: #606266;
}

.timeline-actions {
  display: flex;
  gap: 8px;
}

.update-settings {
  padding: 20px;
}

.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.settings-header h4 {
  margin: 0;
  color: #303133;
}

.settings-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
}

.setting-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.setting-label {
  color: #606266;
  font-weight: 500;
}

.setting-value {
  color: #303133;
}

.update-dialog-content {
  padding: 0;
}

.update-confirmation {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 24px;
  padding: 16px;
  background: #f0f9ff;
  border-radius: 6px;
}

.confirmation-icon {
  font-size: 32px;
  color: #409eff;
}

.confirmation-text h3 {
  margin: 0 0 8px 0;
  color: #303133;
}

.confirmation-text p {
  margin: 0;
  color: #606266;
}

.update-options {
  margin-bottom: 24px;
}

.update-options h4 {
  margin: 0 0 16px 0;
  color: #303133;
}

.update-progress {
  margin-top: 24px;
}

.progress-text {
  margin-top: 8px;
  text-align: center;
  color: #606266;
  font-size: 14px;
}

.release-notes-content {
  max-height: 400px;
  overflow-y: auto;
}

.release-notes {
  line-height: 1.6;
}

.release-notes :deep(h3) {
  margin: 16px 0 8px 0;
  color: #303133;
}

.release-notes :deep(h4) {
  margin: 12px 0 6px 0;
  color: #606266;
}

.no-release-notes {
  text-align: center;
  padding: 40px;
  color: #909399;
}

/* Responsive Design */
@media (max-width: 768px) {
  .status-header {
    flex-direction: column;
    gap: 16px;
  }

  .version-comparison {
    flex-direction: column;
    gap: 8px;
  }

  .arrow-icon {
    transform: rotate(90deg);
  }

  .update-details {
    flex-direction: column;
    gap: 8px;
  }

  .update-actions {
    flex-wrap: wrap;
  }

  .settings-summary {
    flex-direction: column;
    gap: 12px;
  }

  .timeline-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}
</style>
