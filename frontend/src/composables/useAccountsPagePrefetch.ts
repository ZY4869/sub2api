import { adminAPI } from '@/api/admin'
import type { AccountListRequestParams } from '@/utils/accountListSync'
import type { Account, PaginatedResponse } from '@/types'

type CachedAccountsPage = {
  response: PaginatedResponse<Account>
}

const isAbortError = (error: unknown) => {
  const maybeError = error as { name?: string; code?: string } | null
  return (
    maybeError?.name === 'AbortError' ||
    maybeError?.name === 'CanceledError' ||
    maybeError?.code === 'ERR_CANCELED'
  )
}

const normalizeParams = (params: AccountListRequestParams) => {
  const entries = Object.entries(params)
    .filter(([key, value]) => {
      if (key === 'lite') return false
      if (value === undefined || value === null) return false
      return !(typeof value === 'string' && value.trim() === '')
    })
    .sort(([left], [right]) => left.localeCompare(right))

  return Object.fromEntries(entries)
}

const buildQueryKey = (
  pageSize: number,
  params: AccountListRequestParams,
) => JSON.stringify({
  page_size: pageSize,
  params: normalizeParams(params),
})

const buildPageKey = (
  page: number,
  pageSize: number,
  params: AccountListRequestParams,
) => `${buildQueryKey(pageSize, params)}::page=${page}`

export function useAccountsPagePrefetch() {
  const cache = new Map<string, CachedAccountsPage>()
  const inflight = new Map<string, Promise<PaginatedResponse<Account> | null>>()
  const controllers = new Map<string, AbortController>()

  const clear = () => {
    controllers.forEach((controller) => controller.abort())
    controllers.clear()
    inflight.clear()
    cache.clear()
  }

  const storePageSnapshot = (
    response: PaginatedResponse<Account>,
    params: AccountListRequestParams,
  ) => {
    cache.set(
      buildPageKey(response.page, response.page_size, params),
      { response },
    )
  }

  const getCachedPage = (
    page: number,
    pageSize: number,
    params: AccountListRequestParams,
  ) => {
    const cached = cache.get(buildPageKey(page, pageSize, params))
    return cached?.response ?? null
  }

  const prefetchPage = async (
    page: number,
    pageSize: number,
    params: AccountListRequestParams,
  ) => {
    const pageKey = buildPageKey(page, pageSize, params)
    const cached = cache.get(pageKey)
    if (cached) return cached.response

    const pending = inflight.get(pageKey)
    if (pending) return pending

    const controller = new AbortController()
    controllers.set(pageKey, controller)

    const request = adminAPI.accounts
      .list(page, pageSize, normalizeParams(params), { signal: controller.signal })
      .then((response) => {
        cache.set(pageKey, { response })
        return response
      })
      .catch((error) => {
        if (!isAbortError(error)) {
          console.error('Failed to prefetch admin accounts page:', error)
        }
        return null
      })
      .finally(() => {
        inflight.delete(pageKey)
        controllers.delete(pageKey)
      })

    inflight.set(pageKey, request)
    return request
  }
  return {
    clear,
    storePageSnapshot,
    getCachedPage,
    prefetchPage,
  }
}
