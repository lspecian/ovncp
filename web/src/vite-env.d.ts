/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_OIDC_ENABLED: string
  // Add other env variables as needed
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}