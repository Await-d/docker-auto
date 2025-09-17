<template>
  <div class="log-viewer" :class="{ 'full-height': fullHeight }">
    <!-- Log Controls -->
    <div class="log-controls">
      <div class="control-group">
        <el-button-group size="small">
          <el-button
            :type="isFollowing ? 'primary' : 'default'"
            :disabled="loading"
            @click="toggleFollow"
          >
            <el-icon>
              <VideoPlay v-if="!isFollowing" /><VideoPause v-else />
            </el-icon>
            {{ isFollowing ? "Following" : "Follow" }}
          </el-button>

          <el-button
:loading="loading" @click="refreshLogs"
>
            <el-icon><Refresh /></el-icon>
            Refresh
          </el-button>

          <el-button @click="clearLogs">
            <el-icon><Delete /></el-icon>
            Clear
          </el-button>
        </el-button-group>
      </div>

      <div class="control-group">
        <el-select
          v-model="logLevel"
          placeholder="Log Level"
          size="small"
          clearable
          @change="filterLogs"
        >
          <el-option label="All Levels" value="" />
          <el-option label="Debug" value="debug" />
          <el-option label="Info" value="info" />
          <el-option label="Warning" value="warn" />
          <el-option label="Error" value="error" />
          <el-option label="Fatal" value="fatal" />
        </el-select>

        <el-input
          v-model="searchQuery"
          placeholder="Search logs..."
          size="small"
          clearable
          style="width: 200px"
          @input="filterLogs"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="control-group">
        <el-select
          v-model="tailLines"
          placeholder="Lines"
          size="small"
          @change="changeTailLines"
        >
          <el-option label="50 lines" :value="50" />
          <el-option label="100 lines" :value="100" />
          <el-option label="500 lines" :value="500" />
          <el-option label="1000 lines" :value="1000" />
          <el-option label="All lines" :value="0" />
        </el-select>

        <el-checkbox
          v-model="showTimestamps"
          size="small"
          @change="toggleTimestamps"
        >
          Timestamps
        </el-checkbox>

        <el-checkbox
v-model="wrapLines" size="small"
@change="toggleWrapLines"
>
          Wrap Lines
        </el-checkbox>
      </div>

      <div class="control-group">
        <el-dropdown @command="handleAction">
          <el-button size="small">
            <el-icon><MoreFilled /></el-icon>
            Actions
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="download">
                <el-icon><Download /></el-icon>
                Download Logs
              </el-dropdown-item>
              <el-dropdown-item command="copy">
                <el-icon><CopyDocument /></el-icon>
                Copy Visible Logs
              </el-dropdown-item>
              <el-dropdown-item command="share">
                <el-icon><Share /></el-icon>
                Share Log Link
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- Log Stats -->
    <div
v-if="containerLogs.length > 0" class="log-stats"
>
      <div class="stat-item">
        <span class="stat-label">Total Lines:</span>
        <span class="stat-value">{{ formatNumber(containerLogs.length) }}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">Filtered:</span>
        <span class="stat-value">{{ formatNumber(filteredLogs.length) }}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">Errors:</span>
        <span class="stat-value error-count">{{ errorCount }}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">Last Update:</span>
        <span class="stat-value">{{ formatTime(lastUpdate) }}</span>
      </div>
    </div>

    <!-- Log Content -->
    <div
ref="logContentRef" class="log-content"
>
      <!-- Loading State -->
      <div v-if="loading && containerLogs.length === 0" class="log-loading">
        <el-icon class="loading-spinner">
          <Loading />
        </el-icon>
        <span>Loading logs...</span>
      </div>

      <!-- Empty State -->
      <div v-else-if="containerLogs.length === 0" class="log-empty">
        <el-icon class="empty-icon">
          <Document />
        </el-icon>
        <h3>No logs available</h3>
        <p>This container hasn't generated any logs yet.</p>
      </div>

      <!-- No Results -->
      <div v-else-if="filteredLogs.length === 0" class="log-empty">
        <el-icon class="empty-icon">
          <Search />
        </el-icon>
        <h3>No matching logs</h3>
        <p>No logs match your current filters.</p>
      </div>

      <!-- Log Lines -->
      <div v-else class="log-lines" :class="{ 'wrap-lines': wrapLines }">
        <VirtualList
          v-slot="{ item, index }"
          :items="filteredLogs"
          :item-height="lineHeight"
          :container-height="containerHeight"
        >
          <div
            :key="index"
            class="log-line"
            :class="{
              'log-error': item.level === 'error' || item.level === 'fatal',
              'log-warning': item.level === 'warn',
              'log-info': item.level === 'info',
              'log-debug': item.level === 'debug',
              highlighted: highlightedLines.has(index),
            }"
            @click="toggleLineHighlight(index)"
          >
            <span v-if="showTimestamps" class="log-timestamp">
              {{ formatLogTime(item.timestamp) }}
            </span>

            <span class="log-level" :class="`level-${item.level}`">
              {{ item.level.toUpperCase() }}
            </span>

            <span class="log-stream" :class="`stream-${item.stream}`">
              {{ item.stream }}
            </span>

            <span class="log-message" v-html="formatLogMessage(item.message)" />
          </div>
        </VirtualList>
      </div>

      <!-- Auto-scroll indicator -->
      <div v-if="!isAtBottom && isFollowing" class="scroll-indicator">
        <el-button size="small" @click="scrollToBottom">
          <el-icon><ArrowDown /></el-icon>
          Scroll to bottom
        </el-button>
      </div>
    </div>

    <!-- Log Footer -->
    <div
v-if="containerLogs.length > 0" class="log-footer"
>
      <div class="footer-info">
        <span>Line {{ currentLine }} of
          {{ formatNumber(filteredLogs.length) }}</span>
        <span v-if="selectedLinesCount > 0">
          | {{ selectedLinesCount }} lines selected
        </span>
      </div>

      <div class="footer-controls">
        <el-button-group size="small">
          <el-button
:disabled="currentLine <= 1" @click="goToTop"
>
            <el-icon><Top /></el-icon>
          </el-button>
          <el-button
            :disabled="currentLine >= filteredLogs.length"
            @click="goToBottom"
          >
            <el-icon><Bottom /></el-icon>
          </el-button>
        </el-button-group>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from "vue";
import { storeToRefs } from "pinia";
import { ElMessage, ElNotification } from "element-plus";
import {
  VideoPlay,
  VideoPause,
  Refresh,
  Delete,
  Search,
  MoreFilled,
  Download,
  CopyDocument,
  Share,
  Loading,
  Document,
  ArrowDown,
  Top,
  Bottom,
} from "@element-plus/icons-vue";

import { useContainerStore } from "@/store/containers";
import VirtualList from "@/components/common/VirtualList.vue";

interface Props {
  containerId: string;
  containerName: string;
  fullHeight?: boolean;
  autoStart?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  fullHeight: false,
  autoStart: true,
});

const containerStore = useContainerStore();
const { logs } = storeToRefs(containerStore);

// Local state
const loading = ref(false);
const isFollowing = ref(false);
const logLevel = ref("");
const searchQuery = ref("");
const showTimestamps = ref(true);
const wrapLines = ref(false);
const tailLines = ref(100);
const lastUpdate = ref(new Date());
const highlightedLines = ref(new Set<number>());
const currentLine = ref(1);
const isAtBottom = ref(true);

// Refs
const logContentRef = ref<HTMLElement>();

// Virtual list settings
const lineHeight = ref(24);
const containerHeight = ref(400);

// Computed
const containerLogs = computed(() => {
  return logs.value.get(props.containerId) || [];
});

const filteredLogs = computed(() => {
  let filtered = containerLogs.value;

  // Filter by log level
  if (logLevel.value) {
    filtered = filtered.filter((log) => log.level === logLevel.value);
  }

  // Filter by search query
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter((log) =>
      log.message.toLowerCase().includes(query),
    );
  }

  return filtered;
});

const errorCount = computed(() => {
  return containerLogs.value.filter(
    (log) => log.level === "error" || log.level === "fatal",
  ).length;
});

const selectedLinesCount = computed(() => {
  return highlightedLines.value.size;
});

// WebSocket connection
let wsConnection: WebSocket | null = null;
let followInterval: NodeJS.Timeout | null = null;

// Methods
function formatNumber(num: number): string {
  return num.toLocaleString();
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString();
}

function formatLogTime(timestamp: Date | string): string {
  const date = new Date(timestamp);
  return date.toISOString().split("T")[1].split(".")[0];
}

function formatLogMessage(message: string): string {
  // Escape HTML and highlight search terms
  let escaped = message
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");

  // Highlight search query
  if (searchQuery.value) {
    const regex = new RegExp(`(${searchQuery.value})`, "gi");
    escaped = escaped.replace(regex, "<mark>$1</mark>");
  }

  // Highlight common patterns
  escaped = escaped
    .replace(/\b(ERROR|FATAL)\b/g, '<span class="text-error">$1</span>')
    .replace(/\b(WARN|WARNING)\b/g, '<span class="text-warning">$1</span>')
    .replace(/\b(INFO)\b/g, '<span class="text-info">$1</span>')
    .replace(/\b(DEBUG)\b/g, '<span class="text-debug">$1</span>');

  return escaped;
}

async function refreshLogs() {
  loading.value = true;
  try {
    await containerStore.fetchLogs(props.containerId, {
      tail: tailLines.value || undefined,
      timestamps: showTimestamps.value,
    });
    lastUpdate.value = new Date();

    if (isFollowing.value) {
      await nextTick();
      scrollToBottom();
    }
  } catch (error) {
    console.error("Failed to refresh logs:", error);
    ElMessage.error("Failed to refresh logs");
  } finally {
    loading.value = false;
  }
}

function clearLogs() {
  logs.value.set(props.containerId, []);
  highlightedLines.value.clear();
  ElMessage.success("Logs cleared");
}

function toggleFollow() {
  isFollowing.value = !isFollowing.value;

  if (isFollowing.value) {
    startFollowing();
    scrollToBottom();
  } else {
    stopFollowing();
  }
}

function startFollowing() {
  if (followInterval) return;

  // Start WebSocket connection for real-time logs
  connectWebSocket();

  // Fallback polling
  followInterval = setInterval(() => {
    if (!wsConnection || wsConnection.readyState !== WebSocket.OPEN) {
      refreshLogs();
    }
  }, 5000);
}

function stopFollowing() {
  if (followInterval) {
    clearInterval(followInterval);
    followInterval = null;
  }

  if (wsConnection) {
    wsConnection.close();
    wsConnection = null;
  }
}

function connectWebSocket() {
  try {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/api/containers/${props.containerId}/logs/stream`;

    wsConnection = new WebSocket(wsUrl);

    wsConnection.onopen = () => {
      console.log("WebSocket connected for log streaming");
    };

    wsConnection.onmessage = (event) => {
      const logData = JSON.parse(event.data);
      const currentLogs = logs.value.get(props.containerId) || [];
      logs.value.set(props.containerId, [...currentLogs, ...logData.logs]);

      lastUpdate.value = new Date();

      if (isFollowing.value && isAtBottom.value) {
        nextTick(() => scrollToBottom());
      }
    };

    wsConnection.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    wsConnection.onclose = () => {
      console.log("WebSocket connection closed");
      if (isFollowing.value) {
        // Try to reconnect after a delay
        setTimeout(connectWebSocket, 3000);
      }
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
}

function filterLogs() {
  // Reset to top when filters change
  currentLine.value = 1;
  scrollToTop();
}

function changeTailLines() {
  refreshLogs();
}

function toggleTimestamps() {
  // Timestamps preference saved in local storage
  localStorage.setItem(
    "logViewer.showTimestamps",
    showTimestamps.value.toString(),
  );
}

function toggleWrapLines() {
  localStorage.setItem("logViewer.wrapLines", wrapLines.value.toString());
}

function toggleLineHighlight(index: number) {
  if (highlightedLines.value.has(index)) {
    highlightedLines.value.delete(index);
  } else {
    highlightedLines.value.add(index);
  }
}

function scrollToBottom() {
  if (logContentRef.value) {
    const scrollContainer = logContentRef.value.querySelector(".log-lines");
    if (scrollContainer) {
      scrollContainer.scrollTop = scrollContainer.scrollHeight;
      isAtBottom.value = true;
      currentLine.value = filteredLogs.value.length;
    }
  }
}

function scrollToTop() {
  if (logContentRef.value) {
    const scrollContainer = logContentRef.value.querySelector(".log-lines");
    if (scrollContainer) {
      scrollContainer.scrollTop = 0;
      isAtBottom.value = false;
      currentLine.value = 1;
    }
  }
}

function goToTop() {
  scrollToTop();
}

function goToBottom() {
  scrollToBottom();
}

async function handleAction(command: string) {
  switch (command) {
    case "download":
      await downloadLogs();
      break;
    case "copy":
      await copyLogs();
      break;
    case "share":
      await shareLogs();
      break;
  }
}

async function downloadLogs() {
  try {
    const logText = filteredLogs.value
      .map((log) => {
        const parts = [];
        if (showTimestamps.value) {
          parts.push(formatLogTime(log.timestamp));
        }
        parts.push(log.level.toUpperCase());
        parts.push(log.stream);
        parts.push(log.message);
        return parts.join(" ");
      })
      .join("\n");

    const blob = new Blob([logText], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${props.containerName}-logs-${Date.now()}.txt`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);

    ElMessage.success("Logs downloaded successfully");
  } catch (error) {
    console.error("Failed to download logs:", error);
    ElMessage.error("Failed to download logs");
  }
}

async function copyLogs() {
  try {
    const logText = filteredLogs.value
      .map((log) => {
        const parts = [];
        if (showTimestamps.value) {
          parts.push(formatLogTime(log.timestamp));
        }
        parts.push(log.level.toUpperCase());
        parts.push(log.stream);
        parts.push(log.message);
        return parts.join(" ");
      })
      .join("\n");

    await navigator.clipboard.writeText(logText);
    ElMessage.success("Logs copied to clipboard");
  } catch (error) {
    console.error("Failed to copy logs:", error);
    ElMessage.error("Failed to copy logs");
  }
}

async function shareLogs() {
  // Generate a shareable link or create a log snippet
  ElNotification({
    title: "Share Logs",
    message: "Log sharing functionality will be implemented here",
    type: "info",
  });
}

function updateContainerHeight() {
  if (props.fullHeight && logContentRef.value) {
    const rect = logContentRef.value.getBoundingClientRect();
    const availableHeight = window.innerHeight - rect.top - 100;
    containerHeight.value = Math.max(200, availableHeight);
  }
}

// Load preferences
function loadPreferences() {
  const savedTimestamps = localStorage.getItem("logViewer.showTimestamps");
  if (savedTimestamps !== null) {
    showTimestamps.value = savedTimestamps === "true";
  }

  const savedWrapLines = localStorage.getItem("logViewer.wrapLines");
  if (savedWrapLines !== null) {
    wrapLines.value = savedWrapLines === "true";
  }
}

// Lifecycle
onMounted(() => {
  loadPreferences();
  updateContainerHeight();

  if (props.autoStart) {
    refreshLogs();
  }

  window.addEventListener("resize", updateContainerHeight);
});

onUnmounted(() => {
  stopFollowing();
  window.removeEventListener("resize", updateContainerHeight);
});

// Watch for container changes
watch(
  () => props.containerId,
  (newId) => {
    if (newId) {
      clearLogs();
      refreshLogs();
    }
  },
);
</script>

<style scoped>
.log-viewer {
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: "Courier New", "Consolas", monospace;
  font-size: 13px;
  line-height: 1.5;
  border-radius: 4px;
  overflow: hidden;
}

.log-viewer.full-height {
  height: calc(100vh - 200px);
  min-height: 400px;
}

.log-controls {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: #2d2d30;
  border-bottom: 1px solid #3e3e42;
  flex-wrap: wrap;
}

.control-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-stats {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 8px 16px;
  background: #252526;
  border-bottom: 1px solid #3e3e42;
  font-size: 11px;
}

.stat-item {
  display: flex;
  gap: 4px;
}

.stat-label {
  color: #969696;
}

.stat-value {
  color: #d4d4d4;
  font-weight: 500;
}

.error-count {
  color: #f48771 !important;
}

.log-content {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.log-loading,
.log-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #969696;
}

.loading-spinner {
  font-size: 24px;
  margin-bottom: 8px;
  animation: spin 1s linear infinite;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #616161;
}

.log-empty h3 {
  margin: 0 0 8px 0;
  color: #cccccc;
}

.log-empty p {
  margin: 0;
  font-size: 12px;
}

.log-lines {
  height: 100%;
  overflow-y: auto;
  padding: 8px 0;
}

.log-lines.wrap-lines .log-message {
  white-space: pre-wrap;
  word-break: break-word;
}

.log-line {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 2px 16px;
  min-height: 24px;
  cursor: pointer;
  border-left: 3px solid transparent;
  transition: background-color 0.2s;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.05);
}

.log-line.highlighted {
  background: rgba(0, 122, 204, 0.2);
  border-left-color: #007acc;
}

.log-line.log-error {
  border-left-color: #f48771;
}

.log-line.log-warning {
  border-left-color: #dcdcaa;
}

.log-line.log-info {
  border-left-color: #9cdcfe;
}

.log-line.log-debug {
  border-left-color: #969696;
}

.log-timestamp {
  color: #969696;
  font-size: 11px;
  min-width: 70px;
  flex-shrink: 0;
}

.log-level {
  font-size: 10px;
  font-weight: 600;
  min-width: 50px;
  flex-shrink: 0;
  text-align: center;
  padding: 1px 4px;
  border-radius: 2px;
}

.level-error,
.level-fatal {
  background: rgba(244, 135, 113, 0.2);
  color: #f48771;
}

.level-warn {
  background: rgba(220, 220, 170, 0.2);
  color: #dcdcaa;
}

.level-info {
  background: rgba(156, 220, 254, 0.2);
  color: #9cdcfe;
}

.level-debug {
  background: rgba(150, 150, 150, 0.2);
  color: #969696;
}

.log-stream {
  font-size: 10px;
  min-width: 45px;
  flex-shrink: 0;
  color: #969696;
}

.stream-stdout {
  color: #9cdcfe;
}

.stream-stderr {
  color: #f48771;
}

.log-message {
  flex: 1;
  white-space: pre;
  overflow: hidden;
  word-break: break-all;
}

.scroll-indicator {
  position: absolute;
  bottom: 20px;
  right: 20px;
  z-index: 10;
}

.log-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #2d2d30;
  border-top: 1px solid #3e3e42;
  font-size: 11px;
}

.footer-info {
  color: #969696;
  display: flex;
  gap: 8px;
}

.footer-controls {
  display: flex;
  gap: 8px;
}

/* Syntax highlighting */
:deep(.text-error) {
  color: #f48771;
  font-weight: 600;
}

:deep(.text-warning) {
  color: #dcdcaa;
  font-weight: 600;
}

:deep(.text-info) {
  color: #9cdcfe;
  font-weight: 600;
}

:deep(.text-debug) {
  color: #969696;
}

:deep(mark) {
  background: rgba(255, 255, 0, 0.3);
  color: inherit;
  padding: 0 2px;
  border-radius: 2px;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Responsive */
@media (max-width: 768px) {
  .log-controls {
    padding: 8px;
    gap: 8px;
  }

  .control-group {
    flex-wrap: wrap;
    gap: 4px;
  }

  .log-stats {
    flex-wrap: wrap;
    gap: 12px;
  }

  .log-line {
    padding: 2px 8px;
    gap: 4px;
  }

  .log-timestamp {
    min-width: 60px;
  }

  .log-level {
    min-width: 40px;
  }

  .log-stream {
    min-width: 35px;
  }
}

/* Dark theme overrides for Element Plus components */
:deep(.el-button) {
  border-color: #484848;
}

:deep(.el-button:hover) {
  border-color: #007acc;
}

:deep(.el-select .el-input__inner) {
  background: #3c3c3c;
  border-color: #484848;
  color: #cccccc;
}

:deep(.el-input__inner) {
  background: #3c3c3c;
  border-color: #484848;
  color: #cccccc;
}

:deep(.el-checkbox__label) {
  color: #cccccc;
}
</style>
