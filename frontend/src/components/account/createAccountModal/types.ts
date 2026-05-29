import type { AdminGroup, Proxy } from '@/types'

export interface CreateAccountModalProps {
  show: boolean
  proxies: Proxy[]
  groups: AdminGroup[]
}

export interface CreateAccountModalEmit {
  (event: 'close'): void
  (event: 'created'): void
  (event: 'models-imported', result: import('@/api/admin/accounts').AccountModelImportResult): void
}

export interface CreateAccountModalEmits {
  close: []
  created: []
  'models-imported': [result: import('@/api/admin/accounts').AccountModelImportResult]
}
