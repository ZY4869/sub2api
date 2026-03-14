import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import BulkEditAnthropicControlSection from '../BulkEditAnthropicControlSection.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function mountSection(showRpmLimit = true) {
  const enableInterceptWarmup = ref(false)
  const interceptWarmupRequests = ref(false)
  const enableRpmLimit = ref(true)
  const rpmLimitEnabled = ref(true)
  const bulkBaseRpm = ref<number | null>(null)
  const bulkRpmStrategy = ref<'tiered' | 'sticky_exempt'>('tiered')
  const bulkRpmStickyBuffer = ref<number | null>(null)
  const userMsgQueueMode = ref<string | null>(null)

  const wrapper = mount(
    defineComponent({
      components: { BulkEditAnthropicControlSection },
      setup() {
        return {
          enableInterceptWarmup,
          interceptWarmupRequests,
          enableRpmLimit,
          rpmLimitEnabled,
          bulkBaseRpm,
          bulkRpmStrategy,
          bulkRpmStickyBuffer,
          userMsgQueueMode,
          showRpmLimit
        }
      },
      template: `
        <BulkEditAnthropicControlSection
          v-model:enable-intercept-warmup="enableInterceptWarmup"
          v-model:intercept-warmup-requests="interceptWarmupRequests"
          v-model:enable-rpm-limit="enableRpmLimit"
          v-model:rpm-limit-enabled="rpmLimitEnabled"
          v-model:bulk-base-rpm="bulkBaseRpm"
          v-model:bulk-rpm-strategy="bulkRpmStrategy"
          v-model:bulk-rpm-sticky-buffer="bulkRpmStickyBuffer"
          v-model:user-msg-queue-mode="userMsgQueueMode"
          :show-rpm-limit="showRpmLimit"
        />
      `
    })
  )

  return {
    wrapper,
    bulkRpmStrategy,
    userMsgQueueMode
  }
}

describe('BulkEditAnthropicControlSection', () => {
  it('updates rpm strategy and toggles user message queue mode', async () => {
    const { wrapper, bulkRpmStrategy, userMsgQueueMode } = mountSection()

    const strategyButton = wrapper.findAll('button').find((button) =>
      button.text().includes('strategyStickyExempt')
    )
    expect(strategyButton).toBeTruthy()
    await strategyButton?.trigger('click')

    const umqButton = wrapper.findAll('button').find((button) =>
      button.text().includes('umqModeSerialize')
    )
    expect(umqButton).toBeTruthy()
    await umqButton?.trigger('click')
    await umqButton?.trigger('click')

    expect(bulkRpmStrategy.value).toBe('sticky_exempt')
    expect(userMsgQueueMode.value).toBeNull()
  })

  it('hides rpm controls when the platform combination does not support them', () => {
    const { wrapper } = mountSection(false)

    expect(wrapper.text()).toContain('admin.accounts.interceptWarmupRequests')
    expect(wrapper.text()).not.toContain('admin.accounts.quotaControl.rpmLimit.label')
  })
})
