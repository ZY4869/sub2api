import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RequestDetailsView from '../RequestDetailsView.vue'

const replace = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: { tab: 'trace', account_id: '12' }
  }),
  useRouter: () => ({
    replace
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => ({
        'admin.requestDetails.title': 'Request Details',
        'admin.requestDetails.description': 'Description',
        'admin.requestDetails.pageTabs.trace': 'Trace',
        'admin.requestDetails.pageTabs.subject': 'Subject'
      }[key] || key)
    })
  }
})

describe('RequestDetailsView', () => {
  it('renders page tabs and switches to the subject tab', async () => {
    const wrapper = mount(RequestDetailsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          RequestDetailsTraceTab: { template: '<div data-test="trace-tab" />' },
          RequestDetailsSubjectTab: { template: '<div data-test="subject-tab" />' },
        }
      }
    })

    expect(wrapper.find('[data-test="trace-tab"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Trace')
    expect(wrapper.text()).toContain('Subject')

    await wrapper.findAll('button')[1].trigger('click')

    expect(replace).toHaveBeenCalledWith({
      query: {
        tab: 'subject',
        account_id: '12'
      }
    })
  })
})
