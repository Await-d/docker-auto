/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_WS_URL: string
  readonly VITE_APP_TITLE: string
  readonly VITE_APP_VERSION: string
  readonly NODE_ENV: string
  // add more env variables here as needed
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}