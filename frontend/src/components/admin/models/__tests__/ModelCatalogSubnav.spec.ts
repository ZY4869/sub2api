import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import ModelCatalogSubnav from '../ModelCatalogSubnav.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => {
        const labels: Record<string, string> = {
          'admin.models.pages.available.nav': '可用模型',
          'admin.models.pages.all.nav': '全部模型',
          'admin.models.pages.debug.nav': '模型调试'
        }
        return labels[key] || key
      }
    })
  }
})

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => ({ path: '/admin/models/all' }),
    RouterLink: defineComponent({
      props: { to: { type: String, required: true } },
      template: '<a :href="to"><slot /></a>'
    })
  }
})

describe('ModelCatalogSubnav', () => {
  it('renders available, all, and debug in the expected order', () => {
    const wrapper = mount(ModelCatalogSubnav)
    const labels = wrapper.findAll('a').map((link) => link.text())
    const destinations = wrapper.findAll('a').map((link) => link.attributes('href'))

    expect(labels).toEqual(['可用模型', '全部模型', '模型调试'])
    expect(destinations).toEqual(['/admin/models/available', '/admin/models/all', '/admin/models/debug'])
  })
})
