import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import AccountsViewAiryRowActions from '../AccountsViewAiryRowActions.vue'
import type { Account } from '@/types'

const writeText = vi.fn()

Object.assign(navigator, {
  clipboard: {
    writeText,
  },
})

Object.defineProperty(window, 'isSecureContext', {
  value: true,
  configurable: true,
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const makeAccount = (): Account => ({
  id: 9,
  name: 'Primary',
  platform: 'openai',
  type: 'oauth',
  proxy_id: 1,
  concurrency: 4,
  priority: 2,
  status: 'active',
  schedulable: true,
  credentials: {
    access_token: 'secret-access-token',
    refresh_token: 'secret-refresh-token',
  },
  extra: {
    email_address: 'owner@example.com',
  },
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  auto_renew_enabled: false,
  auto_renew_period: 'month',
  created_at: '2026-05-22T00:00:00Z',
  updated_at: '2026-05-22T00:00:00Z',
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null,
  groups: [{ id: 1, name: 'Admin', platform: 'openai' } as any],
} as Account)

enableAutoUnmount(afterEach)

describe('AccountsViewAiryRowActions', () => {
  afterEach(() => {
    writeText.mockReset()
  })

  it('emits scheduling and row action events', async () => {
    const wrapper = mount(AccountsViewAiryRowActions, {
      props: {
        account: makeAccount(),
        togglingSchedulable: null,
      },
      global: {
        plugins: [createPinia()],
      },
    })

    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    await buttons[2].trigger('click')
    await buttons[3].trigger('click')
    await buttons[4].trigger('click')

    expect(wrapper.emitted('toggle-schedulable')).toEqual([[]])
    expect(wrapper.emitted('edit')).toEqual([[]])
    expect(wrapper.emitted('delete')).toEqual([[]])
    expect(wrapper.emitted('more')).toHaveLength(1)
  })

  it('copies only a sanitized account summary', async () => {
    writeText.mockResolvedValue(undefined)

    const wrapper = mount(AccountsViewAiryRowActions, {
      props: {
        account: makeAccount(),
        togglingSchedulable: null,
      },
      global: {
        plugins: [createPinia()],
      },
    })

    await wrapper.findAll('button')[1].trigger('click')

    expect(writeText).toHaveBeenCalledTimes(1)
    const copiedPayload = String(writeText.mock.calls[0][0])
    expect(copiedPayload).toContain('"name": "Primary"')
    expect(copiedPayload).toContain('"group_names"')
    expect(copiedPayload).not.toContain('secret-access-token')
    expect(copiedPayload).not.toContain('secret-refresh-token')
    expect(copiedPayload).not.toContain('owner@example.com')
  })
})
