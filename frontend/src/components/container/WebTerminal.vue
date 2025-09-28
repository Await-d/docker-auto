<template>
  <div class="web-terminal" :class="{ 'fullscreen': isFullscreen }">
    <!-- Terminal Header -->
    <div class="terminal-header">
      <div class="terminal-title">
        <div class="terminal-controls">
          <div class="control-dot red" @click="closeTerminal"></div>
          <div class="control-dot yellow" @click="minimizeTerminal"></div>
          <div class="control-dot green" @click="toggleFullscreen"></div>
        </div>
        <div class="terminal-info">
          <el-icon><Monitor /></el-icon>
          <span>{{ containerName }}</span>
          <el-tag
            v-if="connectionStatus"
            :type="getStatusType(connectionStatus)"
            size="small"
            class="status-tag"
          >
            {{ getStatusText(connectionStatus) }}
          </el-tag>
        </div>
      </div>

      <div class="terminal-actions">
        <!-- Terminal Settings -->
        <el-dropdown @command="handleSettingCommand">
          <el-button size="small" text>
            <el-icon><Setting /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="clear">
                <el-icon><Delete /></el-icon>
                Clear Terminal
              </el-dropdown-item>
              <el-dropdown-item command="copy">
                <el-icon><CopyDocument /></el-icon>
                Copy Selection
              </el-dropdown-item>
              <el-dropdown-item command="paste">
                <el-icon><Document /></el-icon>
                Paste
              </el-dropdown-item>
              <el-dropdown-item command="font-size" divided>
                <el-icon><ZoomIn /></el-icon>
                Font Size
              </el-dropdown-item>
              <el-dropdown-item command="theme">
                <el-icon><View /></el-icon>
                Theme
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <!-- Connection Controls -->
        <el-button
          v-if="!connected"
          size="small"
          type="primary"
          :loading="connecting"
          @click="connect"
        >
          <el-icon><Connection /></el-icon>
          Connect
        </el-button>
        <el-button
          v-else
          size="small"
          type="danger"
          @click="disconnect"
        >
          <el-icon><Close /></el-icon>
          Disconnect
        </el-button>

        <!-- Fullscreen Toggle -->
        <el-button size="small" text @click="toggleFullscreen">
          <el-icon>
            <component :is="isFullscreen ? 'CloseFull' : 'FullScreen'" />
          </el-icon>
        </el-button>
      </div>
    </div>

    <!-- Terminal Body -->
    <div
      ref="terminalRef"
      class="terminal-body"
      :style="{
        fontSize: `${fontSize}px`,
        fontFamily: fontFamily,
        backgroundColor: themes[currentTheme].background,
        color: themes[currentTheme].foreground,
      }"
    ></div>

    <!-- Connection Status Overlay -->
    <div v-if="!connected && !connecting" class="terminal-overlay">
      <div class="overlay-content">
        <el-icon class="overlay-icon"><Monitor /></el-icon>
        <h3>Terminal Disconnected</h3>
        <p>
          {{ containerName }} terminal is not connected.
        </p>
        <el-button type="primary" @click="connect">
          <el-icon><Connection /></el-icon>
          Connect to Terminal
        </el-button>
      </div>
    </div>

    <!-- Connecting Overlay -->
    <div v-if="connecting" class="terminal-overlay">
      <div class="overlay-content">
        <el-icon class="loading-spinner"><Loading /></el-icon>
        <h3>Connecting...</h3>
        <p>Establishing terminal session with {{ containerName }}</p>
      </div>
    </div>

    <!-- Error Overlay -->
    <div v-if="error" class="terminal-overlay error">
      <div class="overlay-content">
        <el-icon class="overlay-icon error"><WarningFilled /></el-icon>
        <h3>Connection Error</h3>
        <p>{{ error.message }}</p>
        <div class="overlay-actions">
          <el-button @click="clearError">Dismiss</el-button>
          <el-button type="primary" @click="reconnect">
            <el-icon><Refresh /></el-icon>
            Retry Connection
          </el-button>
        </div>
      </div>
    </div>

    <!-- Size Selector Dialog -->
    <el-dialog
      v-model="showSizeDialog"
      title="Terminal Size"
      width="400px"
    >
      <div class="size-selector">
        <div class="size-option">
          <label>Columns:</label>
          <el-input-number
            v-model="terminalSize.cols"
            :min="40"
            :max="200"
            size="small"
          />
        </div>
        <div class="size-option">
          <label>Rows:</label>
          <el-input-number
            v-model="terminalSize.rows"
            :min="10"
            :max="60"
            size="small"
          />
        </div>
      </div>
      <template #footer>
        <el-button @click="showSizeDialog = false">Cancel</el-button>
        <el-button type="primary" @click="applyTerminalSize">Apply</el-button>
      </template>
    </el-dialog>

    <!-- Theme Selector Dialog -->
    <el-dialog
      v-model="showThemeDialog"
      title="Terminal Theme"
      width="500px"
    >
      <div class="theme-selector">
        <div
          v-for="(theme, name) in themes"
          :key="name"
          class="theme-option"
          :class="{ active: name === currentTheme }"
          @click="selectTheme(name)"
        >
          <div
            class="theme-preview"
            :style="{
              backgroundColor: theme.background,
              color: theme.foreground,
            }"
          >
            <div class="theme-name">{{ theme.name }}</div>
            <div class="theme-sample">$ echo "Hello World!"</div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showThemeDialog = false">Cancel</el-button>
        <el-button type="primary" @click="applyTheme">Apply</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Monitor,
  Setting,
  Connection,
  Close,
  Delete,
  CopyDocument,
  Document,
  ZoomIn,
  View,
  WarningFilled,
  Refresh,
  FullScreen,
  CloseFull,
  Loading,
} from "@element-plus/icons-vue";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebLinksAddon } from "xterm-addon-web-links";
import { SearchAddon } from "xterm-addon-search";
import { containerAPI } from "@/api/container";
import { useDockerWebSocket } from "@/api/websocket";

interface Props {
  containerId: string;
  containerName: string;
  autoConnect?: boolean;
  initialSize?: {
    cols: number;
    rows: number;
  };
}

interface TerminalTheme {
  name: string;
  background: string;
  foreground: string;
  cursor?: string;
  selection?: string;
  black?: string;
  red?: string;
  green?: string;
  yellow?: string;
  blue?: string;
  magenta?: string;
  cyan?: string;
  white?: string;
  brightBlack?: string;
  brightRed?: string;
  brightGreen?: string;
  brightYellow?: string;
  brightBlue?: string;
  brightMagenta?: string;
  brightCyan?: string;
  brightWhite?: string;
}

interface ConnectionError {
  message: string;
  code?: string;
  details?: string;
}

const props = withDefaults(defineProps<Props>(), {
  autoConnect: false,
  initialSize: () => ({ cols: 80, rows: 24 }),
});

const emit = defineEmits<{
  connected: [];
  disconnected: [];
  error: [error: ConnectionError];
  resize: [size: { cols: number; rows: number }];
}>();

// State
const terminalRef = ref<HTMLElement>();
const connected = ref(false);
const connecting = ref(false);
const connectionStatus = ref<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');
const error = ref<ConnectionError | null>(null);
const isFullscreen = ref(false);
const showSizeDialog = ref(false);
const showThemeDialog = ref(false);

// Terminal configuration
const fontSize = ref(14);
const fontFamily = ref("'Monaco', 'Menlo', 'Ubuntu Mono', monospace");
const terminalSize = ref({
  cols: props.initialSize.cols,
  rows: props.initialSize.rows,
});
const currentTheme = ref('dark');

// Terminal instances
let terminal: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let webLinksAddon: WebLinksAddon | null = null;
let searchAddon: SearchAddon | null = null;

// WebSocket and session management
const { api: wsAPI } = useDockerWebSocket();
let sessionId: string | null = null;
let wsUnsubscribe: (() => void) | null = null;

// Themes
const themes: Record<string, TerminalTheme> = {
  dark: {
    name: 'Dark',
    background: '#1e1e1e',
    foreground: '#d4d4d4',
    cursor: '#d4d4d4',
    selection: 'rgba(255, 255, 255, 0.3)',
    black: '#000000',
    red: '#f44747',
    green: '#608b4e',
    yellow: '#dcdcaa',
    blue: '#569cd6',
    magenta: '#c586c0',
    cyan: '#4ec9b0',
    white: '#d4d4d4',
    brightBlack: '#555555',
    brightRed: '#f44747',
    brightGreen: '#608b4e',
    brightYellow: '#dcdcaa',
    brightBlue: '#569cd6',
    brightMagenta: '#c586c0',
    brightCyan: '#4ec9b0',
    brightWhite: '#ffffff',
  },
  light: {
    name: 'Light',
    background: '#ffffff',
    foreground: '#000000',
    cursor: '#000000',
    selection: 'rgba(0, 0, 0, 0.3)',
    black: '#000000',
    red: '#cd3131',
    green: '#00bc00',
    yellow: '#949800',
    blue: '#0451a5',
    magenta: '#bc05bc',
    cyan: '#0598bc',
    white: '#555555',
    brightBlack: '#666666',
    brightRed: '#cd3131',
    brightGreen: '#14ce14',
    brightYellow: '#b5ba00',
    brightBlue: '#0451a5',
    brightMagenta: '#bc05bc',
    brightCyan: '#0598bc',
    brightWhite: '#a5a5a5',
  },
  monokai: {
    name: 'Monokai',
    background: '#272822',
    foreground: '#f8f8f2',
    cursor: '#f8f8f0',
    selection: 'rgba(255, 255, 255, 0.25)',
    black: '#272822',
    red: '#f92672',
    green: '#a6e22e',
    yellow: '#f4bf75',
    blue: '#66d9ef',
    magenta: '#ae81ff',
    cyan: '#a1efe4',
    white: '#f8f8f2',
    brightBlack: '#75715e',
    brightRed: '#f92672',
    brightGreen: '#a6e22e',
    brightYellow: '#f4bf75',
    brightBlue: '#66d9ef',
    brightMagenta: '#ae81ff',
    brightCyan: '#a1efe4',
    brightWhite: '#f9f8f5',
  },
};

// Computed
const getStatusType = (status: string) => {
  const types: Record<string, 'success' | 'warning' | 'danger' | 'info'> = {
    connected: 'success',
    connecting: 'warning',
    disconnected: 'info',
    error: 'danger',
  };
  return types[status] || 'info';
};

const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    connected: 'Connected',
    connecting: 'Connecting...',
    disconnected: 'Disconnected',
    error: 'Error',
  };
  return texts[status] || status;
};

// Methods
function initializeTerminal() {
  if (!terminalRef.value) return;

  // Create terminal instance
  terminal = new Terminal({
    cols: terminalSize.value.cols,
    rows: terminalSize.value.rows,
    fontSize: fontSize.value,
    fontFamily: fontFamily.value,
    theme: themes[currentTheme.value],
    cursorBlink: true,
    allowTransparency: false,
    convertEol: true,
    scrollback: 1000,
  });

  // Create addons
  fitAddon = new FitAddon();
  webLinksAddon = new WebLinksAddon();
  searchAddon = new SearchAddon();

  // Load addons
  terminal.loadAddon(fitAddon);
  terminal.loadAddon(webLinksAddon);
  terminal.loadAddon(searchAddon);

  // Open terminal
  terminal.open(terminalRef.value);

  // Set up event listeners
  terminal.onData((data) => {
    if (connected.value && sessionId && wsAPI?.isConnected()) {
      wsAPI.sendTerminalInput(sessionId, data);
    }
  });

  terminal.onResize((size) => {
    if (connected.value && sessionId && wsAPI?.isConnected()) {
      wsAPI.resizeTerminal(sessionId, size.cols, size.rows);
    }
    terminalSize.value = { cols: size.cols, rows: size.rows };
    emit('resize', size);
  });

  // Fit to container
  nextTick(() => {
    fitAddon?.fit();
  });

  console.log('Terminal initialized');
}

function destroyTerminal() {
  if (terminal) {
    terminal.dispose();
    terminal = null;
  }
  fitAddon = null;
  webLinksAddon = null;
  searchAddon = null;
}

async function connect() {
  if (connecting.value || connected.value) return;

  connecting.value = true;
  connectionStatus.value = 'connecting';
  error.value = null;

  try {
    console.log(`Creating terminal session for container ${props.containerId}`);

    // Create terminal session
    const sessionResponse = await containerAPI.createTerminalSession(
      props.containerId,
      {
        cols: terminalSize.value.cols,
        rows: terminalSize.value.rows,
        tty: true,
        command: ['/bin/bash', '-l'],
      }
    );

    sessionId = sessionResponse.sessionId;
    console.log(`Terminal session created: ${sessionId}`);

    // Initialize WebSocket connection if not already connected
    if (!wsAPI?.isConnected()) {
      const baseUrl = import.meta.env.VITE_API_BASE_URL || window.location.origin;
      const token = localStorage.getItem('token') || '';
      await wsAPI?.connect();
    }

    // Subscribe to terminal data
    if (wsAPI?.isConnected()) {
      wsUnsubscribe = wsAPI.subscribeToTerminalData(sessionId, (data) => {
        if (terminal && data.data) {
          if (typeof data.data === 'string') {
            terminal.write(data.data);
          } else {
            // Handle binary data
            terminal.write(new Uint8Array(data.data));
          }
        }
      });

      // Send initial connection message
      terminal?.writeln('\\r\\n🚀 Connected to container terminal..\\r\\n');

      connected.value = true;
      connectionStatus.value = 'connected';
      emit('connected');

      ElMessage.success('Terminal connected successfully');
    } else {
      throw new Error('WebSocket connection failed');
    }
  } catch (err: any) {
    console.error('Terminal connection failed:', err);
    const connectionError: ConnectionError = {
      message: err.message || 'Failed to connect to terminal',
      code: err.code,
      details: err.details,
    };
    error.value = connectionError;
    connectionStatus.value = 'error';
    emit('error', connectionError);
    ElMessage.error(`Terminal connection failed: ${connectionError.message}`);
  } finally {
    connecting.value = false;
  }
}

async function disconnect() {
  if (!connected.value && !connecting.value) return;

  try {
    // Unsubscribe from WebSocket
    if (wsUnsubscribe) {
      wsUnsubscribe();
      wsUnsubscribe = null;
    }

    // Close terminal session
    if (sessionId) {
      await containerAPI.closeTerminalSession(props.containerId, sessionId);
      sessionId = null;
    }

    // Clear terminal
    terminal?.clear();
    terminal?.writeln('\\r\\n❌ Terminal disconnected\\r\\n');

    connected.value = false;
    connectionStatus.value = 'disconnected';
    emit('disconnected');

    ElMessage.info('Terminal disconnected');
  } catch (err: any) {
    console.error('Error disconnecting terminal:', err);
    ElMessage.error('Error disconnecting terminal');
  }
}

async function reconnect() {
  if (connected.value) {
    await disconnect();
  }
  clearError();
  await connect();
}

function clearError() {
  error.value = null;
  connectionStatus.value = 'disconnected';
}

function closeTerminal() {
  emit('close');
}

function minimizeTerminal() {
  emit('minimize');
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value;
  nextTick(() => {
    fitAddon?.fit();
  });
}

function handleSettingCommand(command: string) {
  switch (command) {
    case 'clear':
      terminal?.clear();
      break;
    case 'copy':
      copySelection();
      break;
    case 'paste':
      pasteFromClipboard();
      break;
    case 'font-size':
      showFontSizeDialog();
      break;
    case 'theme':
      showThemeDialog.value = true;
      break;
  }
}

function copySelection() {
  if (terminal?.hasSelection()) {
    const selection = terminal.getSelection();
    navigator.clipboard.writeText(selection).then(() => {
      ElMessage.success('Text copied to clipboard');
    }).catch(() => {
      ElMessage.error('Failed to copy text');
    });
  } else {
    ElMessage.warning('No text selected');
  }
}

async function pasteFromClipboard() {
  try {
    const text = await navigator.clipboard.readText();
    if (text && connected.value && sessionId && wsAPI?.isConnected()) {
      wsAPI.sendTerminalInput(sessionId, text);
    }
  } catch (err) {
    ElMessage.error('Failed to paste from clipboard');
  }
}

function showFontSizeDialog() {
  ElMessageBox.prompt('Enter font size (px)', 'Font Size', {
    confirmButtonText: 'OK',
    cancelButtonText: 'Cancel',
    inputValue: fontSize.value.toString(),
    inputValidator: (value) => {
      const size = parseInt(value);
      if (isNaN(size) || size < 8 || size > 24) {
        return 'Font size must be between 8 and 24 pixels';
      }
      return true;
    },
  }).then(({ value }) => {
    fontSize.value = parseInt(value);
    if (terminal) {
      terminal.options.fontSize = fontSize.value;
      fitAddon?.fit();
    }
  });
}

function selectTheme(themeName: string) {
  currentTheme.value = themeName;
}

function applyTheme() {
  if (terminal) {
    terminal.options.theme = themes[currentTheme.value];
  }
  showThemeDialog.value = false;
  ElMessage.success(`Applied ${themes[currentTheme.value].name} theme`);
}

function applyTerminalSize() {
  if (terminal) {
    terminal.resize(terminalSize.value.cols, terminalSize.value.rows);
  }
  showSizeDialog.value = false;
  ElMessage.success('Terminal size updated');
}

// Handle window resize
function handleResize() {
  if (fitAddon && !isFullscreen.value) {
    fitAddon.fit();
  }
}

// Lifecycle
onMounted(() => {
  initializeTerminal();

  if (props.autoConnect) {
    nextTick(() => {
      connect();
    });
  }

  // Handle window resize
  window.addEventListener('resize', handleResize);
});

onUnmounted(() => {
  disconnect();
  destroyTerminal();
  window.removeEventListener('resize', handleResize);
});

// Watch for container changes
watch(
  () => props.containerId,
  async (newId, oldId) => {
    if (oldId && oldId !== newId) {
      await disconnect();
    }
    if (newId && props.autoConnect) {
      await connect();
    }
  }
);

// Watch for fullscreen changes
watch(isFullscreen, () => {
  nextTick(() => {
    fitAddon?.fit();
  });
});
</script>

<style scoped>
.web-terminal {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #333;
  position: relative;
}

.web-terminal.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  border-radius: 0;
  border: none;
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #2d2d30;
  border-bottom: 1px solid #333;
  min-height: 48px;
}

.terminal-title {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.terminal-controls {
  display: flex;
  gap: 6px;
}

.control-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  cursor: pointer;
  transition: opacity 0.2s;
}

.control-dot:hover {
  opacity: 0.8;
}

.control-dot.red {
  background: #ff5f57;
}

.control-dot.yellow {
  background: #ffbd2e;
}

.control-dot.green {
  background: #28ca42;
}

.terminal-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #cccccc;
  font-size: 14px;
}

.status-tag {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 3px;
}

.terminal-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.terminal-body {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.terminal-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.terminal-overlay.error {
  background: rgba(245, 108, 108, 0.1);
  backdrop-filter: blur(4px);
}

.overlay-content {
  text-align: center;
  color: #cccccc;
  max-width: 400px;
  padding: 32px;
}

.overlay-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #666;
}

.overlay-icon.error {
  color: #f56c6c;
}

.loading-spinner {
  font-size: 48px;
  margin-bottom: 16px;
  color: #409eff;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.overlay-content h3 {
  margin: 0 0 8px 0;
  color: #ffffff;
  font-size: 20px;
}

.overlay-content p {
  margin: 0 0 24px 0;
  color: #999;
  line-height: 1.5;
}

.overlay-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.size-selector {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.size-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.size-option label {
  font-weight: 500;
  color: #303133;
}

.theme-selector {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.theme-option {
  cursor: pointer;
  border: 2px solid transparent;
  border-radius: 6px;
  overflow: hidden;
  transition: border-color 0.2s;
}

.theme-option:hover {
  border-color: #409eff;
}

.theme-option.active {
  border-color: #409eff;
}

.theme-preview {
  padding: 16px;
  min-height: 80px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.theme-name {
  font-weight: 600;
  margin-bottom: 8px;
  font-size: 14px;
}

.theme-sample {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  opacity: 0.8;
}

/* Responsive Design */
@media (max-width: 768px) {
  .terminal-header {
    flex-direction: column;
    gap: 8px;
    padding: 12px 16px;
    min-height: auto;
  }

  .terminal-title {
    width: 100%;
    justify-content: center;
  }

  .terminal-actions {
    width: 100%;
    justify-content: center;
  }

  .theme-selector {
    grid-template-columns: 1fr;
  }

  .overlay-content {
    padding: 24px 16px;
  }

  .overlay-actions {
    flex-direction: column;
    align-items: stretch;
  }
}

/* XTerm styles override */
:deep(.xterm) {
  padding: 16px;
}

:deep(.xterm-viewport) {
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #666 #2d2d30;
}

:deep(.xterm-viewport::-webkit-scrollbar) {
  width: 8px;
}

:deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: #2d2d30;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: #666;
  border-radius: 4px;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: #888;
}
</style>