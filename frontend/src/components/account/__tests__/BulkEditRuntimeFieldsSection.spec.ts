import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import BulkEditRuntimeFieldsSection from '../BulkEditRuntimeFieldsSection.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const proxySelectorStub = {
  name: 'ProxySelector',
  props: ['modelValue', 'proxies'],
  emits: ['update:modelValue'],
  template:
    '<button type="button" data-testid="proxy-selector" @click="$emit(\'update:modelValue\', 9)">proxy</button>'
}

function mountSection() {
  const enableProxy = ref(true)
  const proxyId = ref<number | null>(null)
  const enableConcurrency = ref(true)
  const concurrency = ref(1)
  const enableLoadFactor = ref(true)
  const loadFactor = ref<number | null>(null)
  const enablePriority = ref(true)
  const priority = ref(1)
  const enableRateMultiplier = ref(true)
  const rateMultiplier = ref(1)

  const wrapper = mount(
    defineComponent({
      components: { BulkEditRuntimeFieldsSection },
      setup() {
        return {
          enableProxy,
          proxyId,
          enableConcurrency,
          concurrency,
          enableLoadFactor,
          loadFactor,
          enablePriority,
          priority,
          enableRateMultiplier,
          rateMultiplier,
          proxies: [],
        }
      },
      template: `
        <BulkEditRuntimeFieldsSection
          v-model:enable-proxy="enableProxy"
          v-model:proxy-id="proxyId"
          v-model:enable-concurrency="enableConcurrency"
          v-model:concurrency="concurrency"
          v-model:enable-load-factor="enableLoadFactor"
          v-model:load-factor="loadFactor"
          v-model:enable-priority="enablePriority"
          v-model:priority="priority"
          v-model:enable-rate-multiplier="enableRateMultiplier"
          v-model:rate-multiplier="rateMultiplier"
          :proxies="proxies"
        />
      `
    }),
    {
      global: {
        stubs: {
          ProxySelector: proxySelectorStub
        }
      }
    }
  )

  return {
    wrapper,
    proxyId,
    concurrency,
    loadFactor,
    priority,
    rateMultiplier
  }
}

describe('BulkEditRuntimeFieldsSection', () => {
  it('normalizes runtime numeric fields and forwards selector updates', async () => {
    const { wrapper, proxyId, concurrency, loadFactor, priority, rateMultiplier } = mountSection()

    await wrapper.get('[data-testid="proxy-selector"]').trigger('click')

    const inputs = wrapper.findAll('input[type="number"]')
    await inputs[0].setValue('0')
    await inputs[1].setValue('0')
    await inputs[2].setValue('4')
    await inputs[3].setValue('1.5')

    expect(proxyId.value).toBe(9)
    expect(concurrency.value).toBe(1)
    expect(loadFactor.value).toBeNull()
    expect(priority.value).toBe(4)
    expect(rateMultiplier.value).toBe(1.5)
  })
})
