import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import AccountActionMenu from '../AccountActionMenu.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 1,
    name: 'openai-1',
    platform: 'openai',
    type: 'apikey',
    status: 'active',
    schedulable: true,
    lifecycle_state: 'normal',
    ...overrides
  } as Account
}

describe('AccountActionMenu', () => {
  it('shows the blacklist action for non-blacklisted accounts and emits the event', async () => {
    const wrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount(),
        position: { top: 12, left: 34 }
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          Icon: true,
          Teleport: true
        }
      }
    })

    const blacklistButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.blacklist.addNow')
    )

    expect(blacklistButton).toBeTruthy()

    await blacklistButton!.trigger('click')

    expect(wrapper.emitted('blacklist')?.[0]?.[0]).toMatchObject({ id: 1, name: 'openai-1' })
    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('hides the blacklist action for blacklisted accounts', () => {
    const wrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount({ lifecycle_state: 'blacklisted' }),
        position: { top: 12, left: 34 }
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          Icon: true,
          Teleport: true
        }
      }
    })

    expect(wrapper.text()).not.toContain('admin.accounts.blacklist.addNow')
  })

  it('shows the downstream model diagnostics action and emits the event', async () => {
    const wrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount({ platform: 'grok', type: 'apikey' }),
        position: { top: 12, left: 34 }
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          Icon: true,
          Teleport: true
        }
      }
    })

    const diagnosticsButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.modelDiagnostics.action')
    )

    expect(diagnosticsButton).toBeTruthy()

    await diagnosticsButton!.trigger('click')

    expect(wrapper.emitted('diagnose-models')?.[0]?.[0]).toMatchObject({ id: 1, name: 'openai-1' })
    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('shows the quick-test action and emits the event', async () => {
    const wrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount(),
        position: { top: 12, left: 34 }
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          Icon: true,
          Teleport: true
        }
      }
    })

    const quickTestButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.batchTest.quickCheck')
    )

    expect(quickTestButton).toBeTruthy()

    await quickTestButton!.trigger('click')

    expect(wrapper.emitted('quick-test')?.[0]?.[0]).toMatchObject({ id: 1, name: 'openai-1' })
    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('hides generic quota reset for openai accounts', () => {
    const wrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount({
          platform: 'openai',
          type: 'apikey',
          quota_limit: 100
        }),
        position: { top: 12, left: 34 }
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          Icon: true,
          Teleport: true
        }
      }
    })

    expect(wrapper.text()).not.toContain('admin.accounts.resetQuota')
  })

  it('keeps generic quota reset for non-openai local quota accounts', async () => {
    const wrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount({
          platform: 'anthropic',
          type: 'apikey',
          quota_limit: 100
        }),
        position: { top: 12, left: 34 }
      },
      global: {
        plugins: [createPinia()],
        stubs: {
          Icon: true,
          Teleport: true
        }
      }
    })

    const resetButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.resetQuota')
    )

    expect(resetButton).toBeTruthy()

    await resetButton!.trigger('click')

    expect(wrapper.emitted('reset-quota')?.[0]?.[0]).toMatchObject({ platform: 'anthropic' })
    expect(wrapper.emitted('close')).toEqual([[]])
  })
})
