import { beforeEach, describe, expect, it, vi } from 'vitest'

const getMock = vi.fn()
const postMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    post: postMock
  }
}))

describe('admin accounts api', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('normalizes archived group summaries from legacy PascalCase fields', async () => {
    getMock.mockResolvedValue({
      data: [
        {
          GroupID: 9,
          GroupName: 'OpenAI Archive',
          TotalCount: 12,
          AvailableCount: 7,
          InvalidCount: 5,
          LatestUpdatedAt: '2026-03-23T01:02:03Z'
        }
      ]
    })

    const { listArchivedGroups } = await import('../accounts')
    const result = await listArchivedGroups()

    expect(result).toEqual([
      {
        group_id: 9,
        group_name: 'OpenAI Archive',
        total_count: 12,
        available_count: 7,
        invalid_count: 5,
        latest_updated_at: '2026-03-23T01:02:03Z'
      }
    ])
  })

  it('exposes Grok quota, billing probe, reset, reconcile, and runtime sanity endpoints', async () => {
    getMock
      .mockResolvedValueOnce({
        data: {
          source: 'billing_probe',
          headers_observed: true,
          reset_supported: false,
          fetched_at: 1,
          persisted: true,
          billing: {
            plan: 'SuperGrok',
            used_cents: 123,
            monthly_limit_cents: 3000,
            updated_at: '2026-07-15T00:00:00Z'
          },
          snapshot: {
            requests: { limit: 100, remaining: 75, reset_unix: 1784073600 },
            tokens: { limit: 1000, remaining: 850, reset_at: '2026-07-15T01:00:00Z' },
            retry_after_seconds: null,
            observation_source: 'billing_probe',
            headers_observed: true
          }
        }
      })
      .mockResolvedValueOnce({ data: { valid: true, cli_base_url: 'https://cli-chat-proxy.grok.com/v1' } })
    postMock
      .mockResolvedValueOnce({ data: { account_id: 42, platform: 'grok', supported: true, status: 'success', source: 'billing_probe', fetched_at: 1, persisted: true } })
      .mockResolvedValueOnce({ data: { total: 1, succeeded: 1, failed: 0, unsupported: 0, items: [{ account_id: 42, supported: true, status: 'success', persisted: true }] } })
      .mockResolvedValueOnce({ data: { supported: false, code: 'GROK_QUOTA_RESET_UNSUPPORTED', message: 'unsupported' } })
      .mockResolvedValueOnce({ data: { dry_run: true, scanned: 1, actionable: 1, would_block: 1, would_refresh: 0, blocked: 0, refreshed: 0, skipped: 0, failed: 0, items: [], next_after_id: 42, has_more: false } })

    const {
      accountsAPI,
      queryGrokQuota,
      probeAccountBilling,
      batchProbeAccountBilling,
      resetGrokQuota,
      reconcileGrokOAuth,
      getGrokRuntimeSanity
    } = await import('../accounts')

    await expect(queryGrokQuota(42)).resolves.toMatchObject({
      source: 'billing_probe',
      billing: { plan: 'SuperGrok', used_cents: 123 },
      snapshot: {
        requests: { limit: 100, remaining: 75 },
        tokens: { limit: 1000, remaining: 850 },
        observation_source: 'billing_probe'
      }
    })
    await expect(probeAccountBilling(42)).resolves.toMatchObject({ account_id: 42, status: 'success' })
    await expect(batchProbeAccountBilling({ account_ids: [42] })).resolves.toMatchObject({ succeeded: 1 })
    await expect(resetGrokQuota(42)).resolves.toMatchObject({ supported: false })
    await expect(reconcileGrokOAuth({ dry_run: true, limit: 10 })).resolves.toMatchObject({ next_after_id: 42 })
    await expect(getGrokRuntimeSanity()).resolves.toMatchObject({ valid: true })

    expect(getMock).toHaveBeenNthCalledWith(1, '/admin/grok/accounts/42/quota')
    expect(postMock).toHaveBeenNthCalledWith(1, '/admin/accounts/42/billing-probe', {})
    expect(postMock).toHaveBeenNthCalledWith(2, '/admin/accounts/billing-probe', { account_ids: [42] })
    expect(postMock).toHaveBeenNthCalledWith(3, '/admin/grok/accounts/42/reset-quota')
    expect(postMock).toHaveBeenNthCalledWith(4, '/admin/grok/oauth/reconcile', { dry_run: true, limit: 10 })
    expect(getMock).toHaveBeenNthCalledWith(2, '/admin/grok/runtime-sanity')
    expect(accountsAPI.queryGrokQuota).toBe(queryGrokQuota)
    expect(accountsAPI.probeAccountBilling).toBe(probeAccountBilling)
    expect(accountsAPI.batchProbeAccountBilling).toBe(batchProbeAccountBilling)
    expect(accountsAPI.resetGrokQuota).toBe(resetGrokQuota)
    expect(accountsAPI.reconcileGrokOAuth).toBe(reconcileGrokOAuth)
    expect(accountsAPI.getGrokRuntimeSanity).toBe(getGrokRuntimeSanity)
  })

  it('exposes Grok OAuth callback, device, create, reauth, and refresh endpoints', async () => {
    postMock
      .mockResolvedValueOnce({ data: { auth_url: 'https://auth.x.ai/oauth2/authorize', session_id: 's1', redirect_uri: 'http://127.0.0.1/callback', state: 'state-1' } })
      .mockResolvedValueOnce({ data: { access_token: 'access-1', refresh_token: 'refresh-1', base_url: 'https://cli-chat-proxy.grok.com/v1' } })
      .mockResolvedValueOnce({ data: { session_id: 'd1', user_code: 'ABCD-EFGH', verification_uri: 'https://auth.x.ai/activate', interval: 5, expires_at: 1784073600 } })
      .mockResolvedValueOnce({ data: { status: 'authorized', interval: 5, expires_at: 1784073600, token_info: { access_token: 'device-access' } } })
      .mockResolvedValueOnce({ data: { id: 7, platform: 'grok', type: 'oauth', name: 'Grok Build' } })
      .mockResolvedValueOnce({ data: { id: 7, platform: 'grok', type: 'oauth', name: 'Grok Build' } })
      .mockResolvedValueOnce({ data: { id: 7, platform: 'grok', type: 'oauth', name: 'Grok Build' } })

    const {
      generateGrokAuthUrl,
      exchangeGrokAuthCode,
      startGrokDeviceFlow,
      pollGrokDeviceToken,
      createGrokAccountFromOAuth,
      reauthorizeGrokAccountFromOAuth,
      refreshGrokAccount
    } = await import('../accounts')

    await expect(generateGrokAuthUrl({ proxy_id: 2 })).resolves.toMatchObject({ session_id: 's1' })
    await expect(exchangeGrokAuthCode({ session_id: 's1', code: 'code-1', state: 'state-1' })).resolves.toMatchObject({ access_token: 'access-1' })
    await expect(startGrokDeviceFlow({ proxy_id: 2 })).resolves.toMatchObject({ user_code: 'ABCD-EFGH' })
    await expect(pollGrokDeviceToken({ session_id: 'd1' })).resolves.toMatchObject({ status: 'authorized' })
    await expect(createGrokAccountFromOAuth({ session_id: 's1', code: 'code-1', state: 'state-1', name: 'Grok Build' })).resolves.toMatchObject({ id: 7 })
    await expect(reauthorizeGrokAccountFromOAuth(7, { session_id: 's2', code: 'code-2', state: 'state-2' })).resolves.toMatchObject({ id: 7 })
    await expect(refreshGrokAccount(7)).resolves.toMatchObject({ id: 7 })

    expect(postMock).toHaveBeenNthCalledWith(1, '/admin/grok/oauth/auth-url', { proxy_id: 2 })
    expect(postMock).toHaveBeenNthCalledWith(2, '/admin/grok/oauth/exchange-code', { session_id: 's1', code: 'code-1', state: 'state-1' })
    expect(postMock).toHaveBeenNthCalledWith(3, '/admin/grok/oauth/device/start', { proxy_id: 2 })
    expect(postMock).toHaveBeenNthCalledWith(4, '/admin/grok/oauth/device/poll', { session_id: 'd1' })
    expect(postMock).toHaveBeenNthCalledWith(5, '/admin/grok/create-from-oauth', { session_id: 's1', code: 'code-1', state: 'state-1', name: 'Grok Build' })
    expect(postMock).toHaveBeenNthCalledWith(6, '/admin/grok/accounts/7/reauthorize-from-oauth', { session_id: 's2', code: 'code-2', state: 'state-2' })
    expect(postMock).toHaveBeenNthCalledWith(7, '/admin/grok/accounts/7/refresh')
  })
})
