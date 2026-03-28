import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

const showWarning = vi.fn()

const messages: Record<string, string> = {
  'common.total': '总计',
  'common.all': '全部',
  'common.loading': '加载中',
  'admin.models.registry.availableStatus': '已启用',
  'admin.models.registry.unavailableStatus': '未启用',
  'admin.models.registry.columns.model': '模型',
  'admin.models.registry.lifecycleLabels.stable': '稳定',
  'admin.models.registry.lifecycleLabels.beta': '测试版',
  'admin.models.registry.lifecycleLabels.deprecated': '已废弃',
  'admin.models.registry.actions.deactivate': '停用',
  'admin.models.registry.actions.activate': '启用',
  'admin.models.registry.actions.hardDelete': '彻底删除',
  'admin.models.registry.replacedByHint': '由 {model} 替代',
  'admin.models.registry.emptyTitle': '暂无注册表模型',
  'admin.models.registry.emptyDescription': '请调整筛选条件',
  'admin.models.pages.all.testOnly': '测试模型',
  'admin.models.pages.all.testBadge': '测试',
  'admin.models.pages.all.addToTest': '加入测试',
  'admin.models.pages.all.removeFromTest': '移出测试',
  'admin.models.pages.all.filterPlaceholder': '筛选当前提供商下的模型 ID',
  'admin.models.pages.all.loadMore': '加载更多',
  'admin.models.pages.all.hardDeleteSingleConfirm': '确认彻底删除 {model} 吗？',
  'admin.models.pages.all.categories.text': '大模型',
  'admin.models.pages.all.categories.image': '图像',
  'admin.models.pages.all.categories.video': '视频',
  'admin.models.pages.all.categories.audio': '音频',
  'admin.models.pages.all.categories.other': '其他',
  'admin.models.pages.all.bulk.selected': '已选择 {count} 个模型',
  'admin.models.pages.all.bulk.selectLoaded': '全选已加载',
  'admin.models.pages.all.bulk.clear': '清空',
  'admin.models.pages.all.bulk.addToTest': '选中加入测试',
  'admin.models.pages.all.bulk.removeFromTest': '选中移出测试',
  'admin.models.pages.all.bulk.deactivate': '停用选中模型',
  'admin.models.pages.all.bulk.hardDelete': '彻底删除选中模型',
  'admin.models.pages.all.bulk.moveProvider': '迁移到目标厂商',
  'admin.models.pages.all.bulk.moveProviderPlaceholder': '请选择目标厂商',
  'admin.models.pages.all.bulk.moveProviderHint': '已选模型后，请先选择目标厂商，再执行迁移。',
  'admin.models.pages.all.bulk.moveProviderSelectRequired': '请先选择目标厂商',
  'admin.models.pages.all.bulk.deactivateConfirm': '确认停用 {count} 个已启用模型吗？',
  'admin.models.pages.all.bulk.hardDeleteConfirm': '确认彻底删除 {count} 个模型吗？',
  'admin.models.pages.all.bulk.moveProviderConfirm': '确认将 {count} 个模型移动到“{provider}”吗？'
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string | number>) => {
      const template = messages[key] || key
      return Object.entries(params || {}).reduce(
        (text, [paramKey, value]) => text.replaceAll(`{${paramKey}}`, String(value)),
        template
      )
    }
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showWarning
  })
}))

import ModelProviderModelsPanel from '../ModelProviderModelsPanel.vue'

const SearchInputStub = {
  name: 'SearchInputStub',
  props: ['modelValue'],
  emits: ['update:modelValue', 'search'],
  template: '<input data-test="search-input" :value="modelValue" />'
}

const EmptyStateStub = { template: '<div data-test="empty-state" />' }
const LoadingSpinnerStub = { template: '<div data-test="loading-spinner" />' }
const ModelIconStub = { template: '<span data-test="model-icon" />' }
const ModelPlatformsInlineStub = { template: '<span data-test="platforms-inline" />' }

function createModel(id: string, available: boolean, extra?: Partial<Record<string, unknown>>) {
  return {
    id,
    display_name: id.toUpperCase(),
    provider: 'openai',
    platforms: ['openai'],
    protocol_ids: [id],
    aliases: [],
    pricing_lookup_ids: [id],
    preferred_protocol_ids: {},
    modalities: ['text'],
    capabilities: ['text'],
    ui_priority: 1,
    exposed_in: ['runtime'],
    source: 'runtime',
    hidden: false,
    tombstoned: false,
    available,
    ...extra
  }
}

function mountPanel(props?: Record<string, unknown>) {
  return mount(ModelProviderModelsPanel, {
    props: {
      provider: 'openai',
      models: [],
      selectedIds: [],
      moveTargetOptions: [],
      isActivating: () => false,
      isDeactivating: () => false,
      isDeleting: () => false,
      isMoving: () => false,
      isSyncingTestExposure: () => false,
      ...props
    },
    global: {
      stubs: {
        SearchInput: SearchInputStub,
        EmptyState: EmptyStateStub,
        LoadingSpinner: LoadingSpinnerStub,
        ModelIcon: ModelIconStub,
        ModelPlatformsInline: ModelPlatformsInlineStub
      }
    }
  })
}

describe('ModelProviderModelsPanel', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    showWarning.mockReset()
  })

  it('renders grouped categories and row actions based on availability', () => {
    const wrapper = mountPanel({
      models: [
        createModel('gpt-5.4', true),
        createModel('gpt-image', false, {
          modalities: ['text', 'image'],
          capabilities: ['image_generation']
        }),
        createModel('gpt-audio', false, {
          modalities: ['audio'],
          capabilities: ['audio_understanding']
        })
      ]
    })

    expect(wrapper.text()).toContain('大模型')
    expect(wrapper.text()).toContain('图像')
    expect(wrapper.text()).toContain('音频')
    expect(wrapper.text()).toContain('停用')
    expect(wrapper.text()).toContain('启用')
    expect(wrapper.text()).toContain('彻底删除')
  })

  it('emits bulk deactivate and hard delete actions after confirmation', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mountPanel({
      models: [
        createModel('gpt-5.4', true),
        createModel('gpt-image', false, {
          modalities: ['text', 'image'],
          capabilities: ['image_generation']
        })
      ],
      selectedIds: ['gpt-5.4', 'gpt-image']
    })

    const buttons = wrapper.findAll('button')
    const bulkDeactivate = buttons.find((button) => button.text() === '停用选中模型')
    const bulkHardDelete = buttons.find((button) => button.text() === '彻底删除选中模型')
    const rowHardDelete = buttons.find((button) => button.text() === '彻底删除')

    expect(bulkDeactivate).toBeDefined()
    expect(bulkHardDelete).toBeDefined()
    expect(rowHardDelete).toBeDefined()

    await bulkDeactivate!.trigger('click')
    await bulkHardDelete!.trigger('click')
    await rowHardDelete!.trigger('click')

    expect(wrapper.emitted('deactivate')).toEqual([[['gpt-5.4']]])
    expect(wrapper.emitted('hard-delete')).toEqual([
      [['gpt-5.4', 'gpt-image']],
      [['gpt-5.4']]
    ])
  })

  it('shows test and deprecated badges, and emits row add/remove test actions', async () => {
    const wrapper = mountPanel({
      models: [
        createModel('gpt-5.4', true, {
          exposed_in: ['runtime', 'test']
        }),
        createModel('gpt-legacy', false, {
          status: 'deprecated',
          replaced_by: 'gpt-5.4'
        })
      ]
    })

    expect(wrapper.text()).toContain('测试')
    expect(wrapper.text()).toContain('已废弃')
    expect(wrapper.text()).toContain('gpt-legacy')

    const buttons = wrapper.findAll('button')
    const removeFromTest = buttons.find((button) => button.text() === '移出测试')
    const addToTest = buttons.find((button) => button.text() === '加入测试')

    expect(removeFromTest).toBeDefined()
    expect(addToTest).toBeDefined()

    await removeFromTest!.trigger('click')
    await addToTest!.trigger('click')

    expect(wrapper.emitted('remove-from-test')).toEqual([[['gpt-5.4']]])
    expect(wrapper.emitted('add-to-test')).toEqual([[['gpt-legacy']]])
  })

  it('emits bulk add/remove test actions and filter updates', async () => {
    const wrapper = mountPanel({
      models: [
        createModel('gpt-test', true, { exposed_in: ['runtime', 'test'] }),
        createModel('gpt-runtime', true)
      ],
      selectedIds: ['gpt-test', 'gpt-runtime']
    })

    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(3)

    await selects[0].setValue('test')
    await selects[1].setValue('deprecated')

    const buttons = wrapper.findAll('button')
    const bulkAddToTest = buttons.find((button) => button.text() === '选中加入测试')
    const bulkRemoveFromTest = buttons.find((button) => button.text() === '选中移出测试')

    expect(bulkAddToTest).toBeDefined()
    expect(bulkRemoveFromTest).toBeDefined()

    await bulkAddToTest!.trigger('click')
    await bulkRemoveFromTest!.trigger('click')

    expect(wrapper.emitted('update:exposure')).toEqual([['test']])
    expect(wrapper.emitted('update:status')).toEqual([['deprecated']])
    expect(wrapper.emitted('add-to-test')).toEqual([[['gpt-runtime']]])
    expect(wrapper.emitted('remove-from-test')).toEqual([[['gpt-test']]])
  })

  it('shows chinese move-provider guidance and warns when no target provider is selected', async () => {
    const wrapper = mountPanel({
      models: [
        createModel('gpt-5.4', true),
        createModel('gpt-5.4-mini', true)
      ],
      selectedIds: ['gpt-5.4', 'gpt-5.4-mini'],
      moveTargetOptions: [
        { value: 'openai', label: 'OpenAI' },
        { value: 'anthropic', label: 'Anthropic' }
      ]
    })

    expect(wrapper.get('[data-test="bulk-move-provider-hint"]').text()).toBe('已选模型后，请先选择目标厂商，再执行迁移。')
    expect(wrapper.get('[data-test="bulk-move-provider-target"]').text()).toContain('请选择目标厂商')

    const moveButton = wrapper.get('[data-test="bulk-move-provider-button"]')
    expect(moveButton.attributes('disabled')).toBeUndefined()

    await moveButton.trigger('click')

    expect(showWarning).toHaveBeenCalledWith('请先选择目标厂商')
    expect(wrapper.emitted('move-provider')).toBeUndefined()
  })

  it('emits move-provider after selecting a target provider and confirming', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mountPanel({
      models: [
        createModel('gpt-5.4', true),
        createModel('gpt-5.4-mini', true)
      ],
      selectedIds: ['gpt-5.4', 'gpt-5.4-mini'],
      moveTargetOptions: [
        { value: 'openai', label: 'OpenAI' },
        { value: 'anthropic', label: 'Anthropic' }
      ]
    })

    const moveTargetSelect = wrapper.get('[data-test="bulk-move-provider-target"]')
    expect(moveTargetSelect.exists()).toBe(true)

    await moveTargetSelect.setValue('anthropic')

    const moveButton = wrapper.get('[data-test="bulk-move-provider-button"]')
    expect(moveButton.text()).toBe('迁移到目标厂商')

    await moveButton.trigger('click')

    expect(wrapper.emitted('move-provider')).toEqual([[
      {
        targetProvider: 'anthropic',
        modelIds: ['gpt-5.4', 'gpt-5.4-mini']
      }
    ]])
  })
})
