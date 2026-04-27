import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ChannelMonitorsView from '../ChannelMonitorsView.vue'

const mocks = vi.hoisted(() => ({
  listMonitors: vi.fn(),
  listTemplates: vi.fn(),
  push: vi.fn(),
  appStore: {
    publicSettingsLoaded: true,
    cachedPublicSettings: {
      channel_monitor_enabled: false,
    },
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channelMonitors: {
      listMonitors: mocks.listMonitors,
      listTemplates: mocks.listTemplates,
      runMonitor: vi.fn(),
      updateMonitor: vi.fn(),
      deleteMonitor: vi.fn(),
      deleteTemplate: vi.fn(),
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => mocks.appStore,
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mocks.push,
  }),
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: {
        value: 'zh-CN',
      },
      t: (key: string) => key,
    },
  }),
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function mountView() {
  return mount(ChannelMonitorsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<section><slot name="actions" /><slot name="table" /></section>',
        },
        ChannelMonitorsTable: true,
        ChannelMonitorTemplatesTable: true,
        ChannelMonitorFormDialog: {
          props: ['show'],
          template: '<div v-if="show" data-test="monitor-dialog" />',
        },
        ChannelMonitorTemplateFormDialog: true,
        ChannelMonitorHistoryDialog: true,
        ChannelMonitorTemplateApplyDialog: true,
        ChannelMonitorAssociatedDialog: true,
        ConfirmDialog: true,
        Icon: true,
      },
    },
  })
}

describe('ChannelMonitorsView', () => {
  beforeEach(() => {
    mocks.listMonitors.mockReset()
    mocks.listTemplates.mockReset()
    mocks.push.mockReset()
    mocks.appStore.publicSettingsLoaded = true
    mocks.appStore.cachedPublicSettings = { channel_monitor_enabled: false }
    mocks.listMonitors.mockResolvedValue([])
    mocks.listTemplates.mockResolvedValue([])
  })

  it('shows a disabled feature notice with a settings shortcut', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="channel-monitor-disabled-notice"]').exists()).toBe(true)

    await wrapper.get('[data-test="channel-monitor-disabled-notice"] button').trigger('click')
    expect(mocks.push).toHaveBeenCalledWith('/admin/settings')
  })

  it('opens the create monitor dialog from the primary action', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="create-channel-monitor"]').trigger('click')

    expect(wrapper.find('[data-test="monitor-dialog"]').exists()).toBe(true)
  })
})
