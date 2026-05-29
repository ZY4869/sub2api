export type OpsRetryMode = 'client' | 'upstream'
export type OpsQueryMode = 'auto' | 'raw' | 'preagg'

export interface OpsRequestOptions {
  signal?: AbortSignal
}

export type OpsWSStatus = 'connecting' | 'connected' | 'reconnecting' | 'offline' | 'closed'

export type OpsSeverity = string
export type OpsPhase = string
