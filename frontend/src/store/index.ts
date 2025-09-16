/**
 * Pinia store configuration and exports
 */
import { createPinia } from 'pinia'

// Create pinia instance
export const pinia = createPinia()

// Export stores
export { useAuthStore, useAuth } from './auth'
export { useUserStore } from './user'
export { useContainerStore } from './containers'
export { useSettingsStore, useSettings } from './settings'
export { useDashboardStore } from './dashboard'
export { useUpdatesStore } from './updates'

// Application store for global state
export { useAppStore } from './app'

// Default export
export default pinia