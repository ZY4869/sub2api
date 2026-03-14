import type { Ref } from 'vue'
import { accountMatchesFilters, mergeRuntimeFields, type AccountListRequestParams } from '@/utils/accountListSync'
import type { Account } from '@/types'

interface PaginationState {
  page: number
  page_size: number
  total: number
  pages: number
}

interface UseAccountsViewListPatchingOptions {
  accounts: Ref<Account[]>
  params: AccountListRequestParams
  pagination: PaginationState
  hasPendingListSync: Ref<boolean>
  removeSelectedAccounts: (ids: number[]) => void
  syncAccountRefs: (account: Account) => void
  clearRemovedAccount: (accountId: number) => void
}

export function useAccountsViewListPatching({
  accounts,
  params,
  pagination,
  hasPendingListSync,
  removeSelectedAccounts,
  syncAccountRefs,
  clearRemovedAccount
}: UseAccountsViewListPatchingOptions) {
  const syncPaginationAfterLocalRemoval = () => {
    const nextTotal = Math.max(0, pagination.total - 1)
    pagination.total = nextTotal
    pagination.pages = nextTotal > 0 ? Math.ceil(nextTotal / pagination.page_size) : 0

    const maxPage = Math.max(1, pagination.pages || 1)
    if (pagination.page > maxPage) {
      pagination.page = maxPage
    }

    hasPendingListSync.value = nextTotal > 0
  }

  const patchAccountInList = (updatedAccount: Account) => {
    const index = accounts.value.findIndex(account => account.id === updatedAccount.id)
    if (index === -1) return

    const mergedAccount = mergeRuntimeFields(accounts.value[index], updatedAccount)
    if (!accountMatchesFilters(mergedAccount, params)) {
      accounts.value = accounts.value.filter(account => account.id !== mergedAccount.id)
      syncPaginationAfterLocalRemoval()
      removeSelectedAccounts([mergedAccount.id])
      clearRemovedAccount(mergedAccount.id)
      return
    }

    const nextAccounts = [...accounts.value]
    nextAccounts[index] = mergedAccount
    accounts.value = nextAccounts
    syncAccountRefs(mergedAccount)
  }

  return {
    patchAccountInList
  }
}
