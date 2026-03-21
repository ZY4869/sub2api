import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const { listModelRegistry } = vi.hoisted(() => ({
  listModelRegistry: vi.fn()
}))

vi.mock('@/api/admin/modelRegistry', () => ({
  listModelRegistry
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

import ActivateAvailableModelsDialog from '../ActivateAvailableModelsDialog.vue'

const BaseDialogStub = {
  name: 'BaseDialogStub',
  props: ['show'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
}

const PaginationStub = {
  name: 'PaginationStub',
  emits: ['update:page', 'update:pageSize'],
  template: `
    <div data-test="pagination">
      <button type="button" class="go-page-2" @click="$emit('update:page', 2)">page2</button>
    </div>
  `
}

const ModelIconStub = { template: '<span data-test="model-icon" />' }
const LoadingSpinnerStub = { template: '<div data-test="loading-spinner" />' }
const EmptyStateStub = { template: '<div data-test="empty-state" />' }

function createRegistryModel(id: string, provider = 'openai') {
  return {
    id,
    display_name: id.toUpperCase(),
    provider,
    platforms: [provider],
    protocol_ids: [id],
    aliases: [],
    pricing_lookup_ids: [id],
    preferred_protocol_ids: {},
    modalities: ['text'],
    capabilities: [],
    ui_priority: 0,
    exposed_in: ['runtime'],
    source: 'runtime',
    hidden: false,
    tombstoned: false,
    available: false
  }
}

describe('ActivateAvailableModelsDialog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    listModelRegistry.mockReset()
  })

  it('loads unavailable models remotely and keeps selections across pages', async () => {
    listModelRegistry
      .mockResolvedValueOnce({
        items: [createRegistryModel('gpt-5.4'), createRegistryModel('gpt-5.4-mini')],
        total: 3,
        page: 1,
        page_size: 50,
        pages: 2
      })
      .mockResolvedValueOnce({
        items: [createRegistryModel('claude-sonnet-4.5', 'anthropic')],
        total: 3,
        page: 2,
        page_size: 50,
        pages: 2
      })

    const wrapper = mount(ActivateAvailableModelsDialog, {
      props: {
        show: false
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Pagination: PaginationStub,
          ModelIcon: ModelIconStub,
          LoadingSpinner: LoadingSpinnerStub,
          EmptyState: EmptyStateStub
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(listModelRegistry).toHaveBeenCalledWith({
      search: undefined,
      availability: 'unavailable',
      include_hidden: false,
      include_tombstoned: false,
      page: 1,
      page_size: 50
    })

    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.get('.go-page-2').trigger('click')
    await flushPromises()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    await checkboxes[0].setValue(true)
    await wrapper.get('button.btn-primary').trigger('click')

    expect(wrapper.emitted('submit')).toEqual([
      [['gpt-5.4', 'claude-sonnet-4.5']]
    ])
  })

  it('debounces remote search and resets back to the first page', async () => {
    listModelRegistry
      .mockResolvedValueOnce({
        items: [createRegistryModel('gpt-5.4'), createRegistryModel('gpt-5.4-mini')],
        total: 3,
        page: 1,
        page_size: 50,
        pages: 2
      })
      .mockResolvedValueOnce({
        items: [createRegistryModel('claude-sonnet-4.5', 'anthropic')],
        total: 3,
        page: 2,
        page_size: 50,
        pages: 2
      })
      .mockResolvedValueOnce({
        items: [createRegistryModel('gpt-5.4')],
        total: 1,
        page: 1,
        page_size: 50,
        pages: 1
      })

    const wrapper = mount(ActivateAvailableModelsDialog, {
      props: {
        show: false
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Pagination: PaginationStub,
          ModelIcon: ModelIconStub,
          LoadingSpinner: LoadingSpinnerStub,
          EmptyState: EmptyStateStub
        }
      }
    })

    await wrapper.setProps({ show: true })
    await flushPromises()
    await wrapper.get('.go-page-2').trigger('click')
    await flushPromises()

    await wrapper.get('input[type="text"]').setValue('gpt')
    vi.advanceTimersByTime(249)
    await flushPromises()

    expect(listModelRegistry).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(1)
    await flushPromises()

    expect(listModelRegistry).toHaveBeenLastCalledWith({
      search: 'gpt',
      availability: 'unavailable',
      include_hidden: false,
      include_tombstoned: false,
      page: 1,
      page_size: 50
    })
  })
})
