import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ManualAddModelDialog from '../ManualAddModelDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const BaseDialogStub = {
  name: 'BaseDialogStub',
  props: ['show', 'title'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
}

describe('ManualAddModelDialog', () => {
  it('validates the required model id before submitting', async () => {
    const wrapper = mount(ManualAddModelDialog, {
      props: {
        show: true
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub
        }
      }
    })

    const confirmButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.models.available.manualAddDialog.confirm')
    )
    expect(confirmButton).toBeTruthy()

    await confirmButton!.trigger('click')

    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(wrapper.text()).toContain('admin.models.available.manualAddDialog.modelIdRequired')
  })

  it('emits a trimmed payload and resets the form on close', async () => {
    const wrapper = mount(ManualAddModelDialog, {
      props: {
        show: true
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub
        }
      }
    })

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('  gpt-5.4-mini  ')
    await inputs[1].setValue('  GPT-5.4 Mini  ')

    const confirmButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.models.available.manualAddDialog.confirm')
    )
    expect(confirmButton).toBeTruthy()

    await confirmButton!.trigger('click')

    expect(wrapper.emitted('submit')).toEqual([[
      {
        id: 'gpt-5.4-mini',
        display_name: 'GPT-5.4 Mini'
      }
    ]])

    const cancelButton = wrapper.findAll('button').find((button) =>
      button.text().includes('common.cancel')
    )
    expect(cancelButton).toBeTruthy()

    await cancelButton!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('close')).toHaveLength(1)
    expect((inputs[0].element as HTMLInputElement).value).toBe('')
    expect((inputs[1].element as HTMLInputElement).value).toBe('')
  })
})
