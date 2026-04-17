import { describe, expect, it } from 'vitest'
import {
  buildBulkEditAccountPayload,
  canBulkEditAccountPreCheck,
  createDefaultBulkEditAccountFormState,
  hasBulkEditAccountFieldEnabled
} from '../bulkEditAccountForm'
import { OPENAI_WS_MODE_CTX_POOL, OPENAI_WS_MODE_OFF } from '../openaiWsMode'

describe('bulkEditAccountForm', () => {
  it('builds the bulk-update payload and omits legacy queue flags', () => {
    const state = createDefaultBulkEditAccountFormState()
    state.enableBaseUrl = true
    state.baseUrl = ' https://example.com '
    state.enableModelRestriction = true
    state.enableCustomErrorCodes = true
    state.selectedErrorCodes = [429, 503]
    state.enableInterceptWarmup = true
    state.interceptWarmupRequests = true
    state.enableProxy = true
    state.enableLoadFactor = true
    state.loadFactor = 0
    state.enableStatus = true
    state.status = 'inactive'
    state.enableRpmLimit = true
    state.userMsgQueueMode = ''

    const payload = buildBulkEditAccountPayload(state, () => ({
      'claude-sonnet-4.5': 'claude-sonnet-4.5'
    }))

    expect(payload).toEqual({
      proxy_id: 0,
      load_factor: 0,
      status: 'inactive',
      credentials: {
        base_url: 'https://example.com',
        model_mapping: {
          'claude-sonnet-4.5': 'claude-sonnet-4.5'
        },
        custom_error_codes_enabled: true,
        custom_error_codes: [429, 503],
        intercept_warmup_requests: true
      },
      extra: {
        base_rpm: 0,
        rpm_strategy: '',
        rpm_sticky_buffer: 0,
        user_msg_queue_mode: ''
      }
    })
    expect(payload?.extra).not.toHaveProperty('user_msg_queue_enabled')
  })

  it('treats explicit queue-mode changes as enabled and only prechecks supported platforms', () => {
    const state = createDefaultBulkEditAccountFormState()
    state.userMsgQueueMode = ''
    state.enableGroups = true
    state.groupIds = [7]

    expect(hasBulkEditAccountFieldEnabled(state)).toBe(true)
    expect(canBulkEditAccountPreCheck(state, ['anthropic'])).toBe(true)
    expect(canBulkEditAccountPreCheck(state, ['openai'])).toBe(false)
    expect(canBulkEditAccountPreCheck(state, ['anthropic', 'openai'])).toBe(false)
  })

  it('returns null when enabled fields do not produce an effective payload', () => {
    const state = createDefaultBulkEditAccountFormState()
    state.enableBaseUrl = true
    state.baseUrl = '   '
    state.enableModelRestriction = true

    expect(buildBulkEditAccountPayload(state, () => null)).toBeNull()
  })

  it('writes openai ws mode updates only for the selected account types', () => {
    const state = createDefaultBulkEditAccountFormState()
    state.enableOpenAIWSMode = true
    state.openAIWSMode = OPENAI_WS_MODE_CTX_POOL

    const payload = buildBulkEditAccountPayload(
      state,
      () => null,
      { applyOpenAIOAuthWSMode: true, applyOpenAIAPIKeyWSMode: false }
    )

    expect(payload).toEqual({
      extra: {
        openai_oauth_responses_websockets_v2_mode: OPENAI_WS_MODE_CTX_POOL,
        openai_oauth_responses_websockets_v2_enabled: true
      }
    })

    state.openAIWSMode = OPENAI_WS_MODE_OFF
    const bothPayload = buildBulkEditAccountPayload(
      state,
      () => null,
      { applyOpenAIOAuthWSMode: true, applyOpenAIAPIKeyWSMode: true }
    )

    expect(bothPayload).toEqual({
      extra: {
        openai_oauth_responses_websockets_v2_mode: OPENAI_WS_MODE_OFF,
        openai_oauth_responses_websockets_v2_enabled: false,
        openai_apikey_responses_websockets_v2_mode: OPENAI_WS_MODE_OFF,
        openai_apikey_responses_websockets_v2_enabled: false
      }
    })
  })
})
