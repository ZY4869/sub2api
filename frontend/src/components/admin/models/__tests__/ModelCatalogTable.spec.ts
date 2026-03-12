import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import ModelCatalogTable from '../ModelCatalogTable.vue'

const translations: Record<string, string> = {
  'admin.models.columns.model': 'Model',
  'admin.models.columns.provider': 'Provider',
  'admin.models.columns.mode': 'Mode',
  'admin.models.columns.defaultProtocol': 'Default Protocol',
  'admin.models.columns.accessSource': 'Source',
  'admin.models.columns.pricingSource': 'Pricing Source',
  'admin.models.columns.inputCost': 'Input Price',
  'admin.models.columns.outputCost': 'Output Price',
  'admin.models.columns.cacheCreationCost': 'Cache Create Price',
  'admin.models.columns.cacheReadCost': 'Cache Read Price',
  'admin.models.columns.imageCost': 'Image Price',
  'admin.models.accessSources.login': 'Login',
  'admin.models.accessSources.key': 'Key',
  'admin.models.sources.dynamic': 'Dynamic',
  'admin.models.modes.chat': 'Chat',
  'common.actions': 'Actions',
  'admin.models.viewDetails': 'View Details',
  'admin.models.emptyTitle': 'Empty',
  'admin.models.emptyDescription': 'Empty'
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] ?? key
    })
  }
})

const DataTableStub = defineComponent({
  props: {
    columns: { type: Array, required: true },
    data: { type: Array, required: true },
    loading: { type: Boolean, default: false }
  },
  template: `
    <table>
      <thead>
        <tr>
          <th v-for="column in columns" :key="column.key">
            <slot :name="'header-' + column.key" :column="column">{{ column.label }}</slot>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, rowIndex) in data" :key="row.model || rowIndex">
          <td v-for="column in columns" :key="column.key">
            <slot :name="'cell-' + column.key" :row="row" :value="row[column.key]">{{ row[column.key] }}</slot>
          </td>
        </tr>
      </tbody>
    </table>
  `
})

describe('ModelCatalogTable', () => {
  it('renders default protocols, access sources, and keeps only inspect action', () => {
    const wrapper = mount(ModelCatalogTable, {
      props: {
        items: [
          {
            model: 'claude-sonnet-4.5',
            provider: 'anthropic',
            mode: 'chat',
            default_available: true,
            default_platforms: ['anthropic', 'antigravity'],
            access_sources: ['login', 'key'],
            pricing_source: 'dynamic',
            official_pricing: {
              input_cost_per_token: 0.1,
              output_cost_per_token: 0.2,
              cache_creation_input_token_cost: 0.3,
              cache_read_input_token_cost: 0.4,
              output_cost_per_image: 0.5
            }
          }
        ],
        loading: false,
        pricingLayer: 'official'
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          ModelCatalogModelLabel: defineComponent({
            props: ['model'],
            template: '<span>{{ model }}</span>'
          }),
          ModelCatalogPriceValue: defineComponent({
            props: ['value'],
            template: '<span>{{ value }}</span>'
          })
        }
      }
    })

    expect(wrapper.text()).toContain('Anthropic')
    expect(wrapper.text()).toContain('Antigravity')
    expect(wrapper.text()).toContain('Login')
    expect(wrapper.text()).toContain('Key')
    expect(wrapper.text()).toContain('View Details')
    expect(wrapper.text()).not.toContain('Delete')

    const headerChips = wrapper.findAll('thead th span.rounded-full')
    expect(headerChips).toHaveLength(5)
    expect(headerChips.map((chip) => chip.text())).toEqual([
      'Input Price',
      'Output Price',
      'Cache Create Price',
      'Cache Read Price',
      'Image Price'
    ])
  })
})
