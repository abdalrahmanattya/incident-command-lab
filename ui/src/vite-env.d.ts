/// <reference types="vite/client" />

declare global {
  interface Window {
    __INCIDENTLAB_CONFIG__?: RuntimeConfig
  }
}

export interface RuntimeConfig {
  authMode: 'disabled' | 'entra'
  clientId: string
  tenantId: string
  operatorGroupId: string
  apiScope: string
}

export {}
