import { reactive } from 'vue'
import { resolveAccountActionMenuPosition } from '@/utils/accountActionMenuPosition'
import type { Account } from '@/types'

interface OpenAccountActionMenuOptions {
  account: Account
  event: MouseEvent
}

export function useAccountActionMenu() {
  const menu = reactive<{
    show: boolean
    acc: Account | null
    pos: { top: number; left: number } | null
  }>({
    show: false,
    acc: null,
    pos: null
  })

  const openMenu = ({ account, event }: OpenAccountActionMenuOptions) => {
    menu.acc = account
    menu.pos = resolveAccountActionMenuPosition({
      event,
      target: event.currentTarget instanceof HTMLElement ? event.currentTarget : null
    })
    menu.show = true
  }

  const closeMenu = () => {
    menu.show = false
  }

  const syncMenuAccount = (account: Account) => {
    if (menu.acc?.id === account.id) {
      menu.acc = account
    }
  }

  const clearMenuAccount = (accountId: number) => {
    if (menu.acc?.id !== accountId) return
    menu.show = false
    menu.acc = null
  }

  return {
    menu,
    openMenu,
    closeMenu,
    syncMenuAccount,
    clearMenuAccount
  }
}
