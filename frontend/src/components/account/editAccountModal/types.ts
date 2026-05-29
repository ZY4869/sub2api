import type { Account, AdminGroup, Proxy } from '@/types'

export interface EditAccountModalProps {
  show: boolean
  loading: boolean
  account: Account | null
  proxies: Proxy[]
  groups: AdminGroup[]
}

export interface EditAccountModalEmit {
  (event: 'close'): void
  (event: 'updated', account: Account): void
}

export interface EditAccountModalEmits {
  close: []
  updated: [account: Account]
}
