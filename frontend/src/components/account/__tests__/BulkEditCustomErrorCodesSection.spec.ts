import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BulkEditCustomErrorCodesSection from '../BulkEditCustomErrorCodesSection.vue'

const showError = vi.fn()
const showInfo = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showInfo
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function mountSection() {
  const enabled = ref(true)
  const selectedCodes = ref<number[]>([])
  const input = ref<number | null>(null)

  const wrapper = mount(
    defineComponent({
      components: { BulkEditCustomErrorCodesSection },
      setup() {
        return {
          enabled,
          selectedCodes,
          input,
          errorCodeOptions: [
            { value: 429, label: 'Rate Limit' },
            { value: 500, label: 'Server Error' }
          ]
        }
      },
      template: `
        <BulkEditCustomErrorCodesSection
          v-model:enabled="enabled"
          v-model:selected-codes="selectedCodes"
          v-model:input="input"
          :error-code-options="errorCodeOptions"
        />
      `
    }),
    {
      global: {
        stubs: {
          Icon: true
        }
      }
    }
  )

  return { wrapper, enabled, selectedCodes, input }
}

describe('BulkEditCustomErrorCodesSection', () => {
  beforeEach(() => {
    showError.mockReset()
    showInfo.mockReset()
    vi.stubGlobal('confirm', vi.fn(() => true))
  })

  it('toggles common codes and supports manual input', async () => {
    const { wrapper, selectedCodes, input } = mountSection()

    const toggle429 = wrapper.findAll('button').find((button) => button.text().includes('429'))
    expect(toggle429).toBeTruthy()

    await toggle429?.trigger('click')
    expect(selectedCodes.value).toContain(429)

    await wrapper.get('#bulk-edit-custom-error-code-input').setValue('500')
    expect(input.value).toBe(500)

    const addButton = wrapper
      .findAll('button')
      .find((button) => (button.attributes('class') || '').includes('btn-secondary'))
    await addButton?.trigger('click')

    expect(selectedCodes.value).toEqual([429, 500])
  })

  it('shows validation feedback for invalid or duplicate input', async () => {
    const { wrapper, selectedCodes } = mountSection()
    selectedCodes.value = [500]

    await wrapper.get('#bulk-edit-custom-error-code-input').setValue('99')
    const addButton = wrapper
      .findAll('button')
      .find((button) => (button.attributes('class') || '').includes('btn-secondary'))
    await addButton?.trigger('click')
    expect(showError).toHaveBeenCalledWith('admin.accounts.invalidErrorCode')

    await wrapper.get('#bulk-edit-custom-error-code-input').setValue('500')
    await addButton?.trigger('click')
    expect(showInfo).toHaveBeenCalledWith('admin.accounts.errorCodeExists')
  })
})
