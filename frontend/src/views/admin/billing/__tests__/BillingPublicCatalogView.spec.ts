import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import BillingPublicCatalogView from '../BillingPublicCatalogView.vue'

function localDateTimeToISOString(value: string) {
  return new Date(value).toISOString()
}

const messages: Record<string, string> = {
  'admin.billing.publicCatalog.header.eyebrow': 'Public Catalog',
  'admin.billing.publicCatalog.header.title': '对外模型展示',
  'admin.billing.publicCatalog.header.description': '按账号支持的模型实例维护公开目录。同一基础模型可以来自不同账号来源，发布后用户只调用唯一公开模型 ID，计费使用本条目的售卖价。',
  'admin.billing.publicCatalog.header.loading': '加载中...',
  'admin.billing.publicCatalog.header.refresh': '刷新候选',
  'admin.billing.publicCatalog.header.syncAvailable': '同步当前可用模型',
  'admin.billing.publicCatalog.header.saving': '保存中...',
  'admin.billing.publicCatalog.header.save': '保存草稿',
  'admin.billing.publicCatalog.header.publishing': '推送中...',
  'admin.billing.publicCatalog.header.publish': '推送更新',
  'admin.billing.publicCatalog.header.draftTitle': '草稿条目',
  'admin.billing.publicCatalog.header.draftValue': '{count} 个',
  'admin.billing.publicCatalog.header.pageSize': '每页 {count} 条',
  'admin.billing.publicCatalog.header.pageSizeTitle': '草稿分页',
  'admin.billing.publicCatalog.header.pageSizeValue': '{count} 条/页',
  'admin.billing.publicCatalog.header.draftSavedAt': '最近保存：{time}',
  'admin.billing.publicCatalog.header.availableTitle': '账号候选',
  'admin.billing.publicCatalog.header.availableValue': '{count} 个',
  'admin.billing.publicCatalog.header.accountAliasCount': '{count} 个来源别名',
  'admin.billing.publicCatalog.header.accountAliasTitle': '账号来源',
  'admin.billing.publicCatalog.header.accountAliasValue': '{count} 个别名',
  'admin.billing.publicCatalog.header.availableUpdatedAt': '更新时间：{time}',
  'admin.billing.publicCatalog.header.publishedTitle': '已发布',
  'admin.billing.publicCatalog.header.publishedValue': '{count} 个',
  'admin.billing.publicCatalog.header.publishedPageSizeTitle': '发布分页',
  'admin.billing.publicCatalog.header.publishedAt': '最近推送：{time}',
  'admin.billing.publicCatalog.header.sourceTitle': '候选来源',
  'admin.billing.publicCatalog.header.sourceDescription': '{source}。官方参考价来自模型定价快照，售卖价在本页随公开条目冻结。',
  'admin.billing.publicCatalog.controls.search': '搜索',
  'admin.billing.publicCatalog.controls.searchPlaceholder': '搜索展示名 / 模型 ID / 来源别名 / 厂商',
  'admin.billing.publicCatalog.controls.provider': '厂商',
  'admin.billing.publicCatalog.controls.allProviders': '全部厂商',
  'admin.billing.publicCatalog.controls.accountSource': '账号来源',
  'admin.billing.publicCatalog.controls.allSources': '全部来源',
  'admin.billing.publicCatalog.controls.pageSize': '每页',
  'admin.billing.publicCatalog.controls.addAll': '全部添加',
  'admin.billing.publicCatalog.controls.export': '导出快照',
  'admin.billing.publicCatalog.controls.batchTitle': '批量定价',
  'admin.billing.publicCatalog.controls.ratio': '比例',
  'admin.billing.publicCatalog.controls.scope': '范围',
  'admin.billing.publicCatalog.controls.scopeAria': '选择批量定价范围',
  'admin.billing.publicCatalog.controls.scopeGroups.global': '全局生效',
  'admin.billing.publicCatalog.controls.scopeGroups.source': '对单个账号单独设定',
  'admin.billing.publicCatalog.controls.scopes.filtered': '左侧当前筛选',
  'admin.billing.publicCatalog.controls.scopes.selected': '右侧已选展示',
  'admin.billing.publicCatalog.controls.scopes.all': '双侧全部条目',
  'admin.billing.publicCatalog.controls.scopes.source': '仅来源：{alias}',
  'admin.billing.publicCatalog.controls.applyOfficial': '按官方参考价应用',
  'admin.billing.publicCatalog.controls.batchHint': '只改草稿中的售卖价，官方参考价仍在“官方成本/价格源管理”页维护。',
  'admin.billing.publicCatalog.controls.duplicate': '公开模型 ID 不能重复：{ids}',
  'admin.billing.publicCatalog.controls.listSeparator': '、',
  'admin.billing.publicCatalog.columns.availableTitle': '账号支持的模型库',
  'admin.billing.publicCatalog.columns.availableDescription': '同一基础模型按账号/节点来源重复展示。',
  'admin.billing.publicCatalog.columns.currentCount': '当前 {count} 个',
  'admin.billing.publicCatalog.columns.emptyAvailable': '当前筛选下没有账号支持模型。',
  'admin.billing.publicCatalog.columns.selectedTitle': '已选对外展示模型',
  'admin.billing.publicCatalog.columns.selectedDescription': '顺序、公开 ID、来源别名和售卖价会随发布快照冻结。',
  'admin.billing.publicCatalog.columns.clear': '清空全部',
  'admin.billing.publicCatalog.columns.emptySelectedTitle': '展示列表为空',
  'admin.billing.publicCatalog.columns.emptySelected': '展示列表为空，请从左侧账号支持模型库添加。',
  'admin.billing.publicCatalog.card.dragLabel': '拖拽排序',
  'admin.billing.publicCatalog.card.add': '添加',
  'admin.billing.publicCatalog.card.added': '已添加',
  'admin.billing.publicCatalog.card.edit': '编辑',
  'admin.billing.publicCatalog.card.moveUp': '上移',
  'admin.billing.publicCatalog.card.moveDown': '下移',
  'admin.billing.publicCatalog.card.remove': '移除',
  'admin.billing.publicCatalog.card.copyPublicId': '复制公开模型 ID',
  'admin.billing.publicCatalog.card.provider': '厂商',
  'admin.billing.publicCatalog.card.protocolPlain': '协议',
  'admin.billing.publicCatalog.card.modePlain': '模式',
  'admin.billing.publicCatalog.card.baseModelPlain': '基础模型',
  'admin.billing.publicCatalog.card.publicModelId': '公开模型 ID',
  'admin.billing.publicCatalog.card.sourceAlias': '公开来源别名',
  'admin.billing.publicCatalog.card.baseModel': '基础模型：{value}',
  'admin.billing.publicCatalog.card.protocol': '协议：{value}',
  'admin.billing.publicCatalog.card.mode': '模式：{value}',
  'admin.billing.publicCatalog.card.account': '后台账号：{value}',
  'admin.billing.publicCatalog.card.missing': '当前条目已不在最新账号支持模型库中，发布时后端会阻止发布。',
  'admin.billing.publicCatalog.card.defaultSource': '默认来源',
  'admin.billing.publicCatalog.card.demo': '演示数据',
  'admin.billing.publicCatalog.card.contextSource': '上下文窗口来源',
  'admin.billing.publicCatalog.card.statuses.expired': '已失效',
  'admin.billing.publicCatalog.card.statuses.available': '可用',
  'admin.billing.publicCatalog.card.statuses.unavailable': '不可用',
  'admin.billing.publicCatalog.card.statuses.pending': '待验证',
  'admin.billing.publicCatalog.card.scheduleStatuses.scheduled': '预启用',
  'admin.billing.publicCatalog.card.scheduleStatuses.expired': '已过期',
  'admin.billing.publicCatalog.card.scheduleStatuses.outOfWindow': '窗口外',
  'admin.billing.publicCatalog.card.scheduleStatuses.invalid': '时间策略异常',
  'admin.billing.publicCatalog.card.lifecycle.stable': '高稳定',
  'admin.billing.publicCatalog.card.lifecycle.beta': 'Beta',
  'admin.billing.publicCatalog.card.lifecycle.deprecated': '即将下线',
  'admin.billing.publicCatalog.dialog.title': '编辑公开模型条目',
  'admin.billing.publicCatalog.dialog.publicModelId': '公开模型 ID',
  'admin.billing.publicCatalog.dialog.sourceAlias': '公开来源别名',
  'admin.billing.publicCatalog.dialog.pricingTitle': '定价设置',
  'admin.billing.publicCatalog.dialog.scheduleTitle': '预启用与限时调用',
  'admin.billing.publicCatalog.dialog.availableFrom': '预启用时间',
  'admin.billing.publicCatalog.dialog.availableUntil': '下架时间',
  'admin.billing.publicCatalog.dialog.timeAccess': '限制调用时间段',
  'admin.billing.publicCatalog.dialog.timeAccessHint': '发布后公开目录和运行时调用都会执行该时间策略。',
  'admin.billing.publicCatalog.dialog.cancel': '取消',
  'admin.billing.publicCatalog.dialog.save': '保存',
  'admin.billing.publicCatalog.price.official': '官方参考价',
  'admin.billing.publicCatalog.price.unit': '',
  'admin.billing.publicCatalog.price.noOfficial': '未配置官方参考价',
  'admin.billing.publicCatalog.price.sale': '售卖价',
  'admin.billing.publicCatalog.price.markup': '溢价',
  'admin.billing.publicCatalog.price.localRatio': '单卡快捷比例',
  'admin.billing.publicCatalog.price.applyLocalRatio': '应用单卡快捷比例',
  'admin.billing.publicCatalog.price.noSale': '暂无售价',
  'admin.billing.publicCatalog.price.supportedUnpriced': '缓存支持，价格未配置',
  'admin.billing.publicCatalog.price.units.perMillionTokens': '每百万 tokens',
  'admin.billing.publicCatalog.price.units.perImage': '每张图片',
  'admin.billing.publicCatalog.price.units.perRequest': '每次请求',
  'admin.billing.publicCatalog.price.units.perVideo': '每段视频',
  'admin.billing.publicCatalog.price.labels.input_price': '输入',
  'admin.billing.publicCatalog.price.labels.output_price': '输出',
  'admin.billing.publicCatalog.price.labels.cache_price': '缓存',
  'admin.billing.publicCatalog.price.labels.cache_creation': '缓存写入',
  'admin.billing.publicCatalog.price.labels.cache_read': '缓存读取',
  'admin.billing.publicCatalog.price.labels.cache_5m': '缓存写入 5 分钟',
  'admin.billing.publicCatalog.price.labels.cache_1h': '缓存写入 1 小时',
  'admin.billing.publicCatalog.price.labels.search_unit_price': '搜索',
  'common.timeAccess.start': '开始时间',
  'common.timeAccess.end': '结束时间',
  'common.timeAccess.notBefore': '最早启用',
  'common.timeAccess.notAfter': '最晚可用',
  'common.timeAccess.dailyAllowedMinutes': '每日窗口上限（分钟）',
  'common.timeAccess.windowHint': '默认使用站点统一时区 Asia/Singapore；结束时间早于开始时间时表示跨午夜。',
  'common.timeAccess.presets.daytime': '白天',
  'common.timeAccess.presets.deep_night': '深夜',
  'common.timeAccess.presets.eight_hours': '8 小时',
  'common.timeAccess.presets.twelve_hours': '12 小时',
  'common.timeAccess.presets.business_days_daytime': '工作日白天',
  'ui.modelCatalog.support.supported': '支持',
  'ui.modelCatalog.support.partial': '部分支持',
  'ui.modelCatalog.support.unsupported': '不支持',
  'ui.modelCatalog.support.unknown': '未验证',
  'ui.modelCatalog.source.verified': '已验证',
  'ui.modelCatalog.source.probe': '账号探测',
  'ui.modelCatalog.source.declared': '声明配置',
  'ui.modelCatalog.source.pricing': '价格目录',
  'ui.modelCatalog.source.snapshot': '发布快照',
  'ui.modelCatalog.source.inferred': '推断',
  'ui.modelCatalog.source.unknown': '未知来源',
  'admin.billing.publicCatalog.diagnostics.title': '模型健康度容量诊断',
  'admin.billing.publicCatalog.diagnostics.description': '仅管理员可见。汇总公开模型背后的账号调度、冷却、配额与供应商限流来源，普通目录保持脱敏。',
  'admin.billing.publicCatalog.diagnostics.refresh': '刷新诊断',
  'admin.billing.publicCatalog.diagnostics.loading': '刷新中...',
  'admin.billing.publicCatalog.diagnostics.total': '公开模型',
  'admin.billing.publicCatalog.diagnostics.available': '可用',
  'admin.billing.publicCatalog.diagnostics.limited': '受限',
  'admin.billing.publicCatalog.diagnostics.unschedulable': '不可调度',
  'admin.billing.publicCatalog.diagnostics.model': '模型',
  'admin.billing.publicCatalog.diagnostics.availability': '可用性',
  'admin.billing.publicCatalog.diagnostics.effectiveLimit': '有效限流',
  'admin.billing.publicCatalog.diagnostics.sources': '来源',
  'admin.billing.publicCatalog.diagnostics.empty': '暂无诊断数据',
  'admin.billing.publicCatalog.diagnostics.availabilityLabels.available': '可用',
  'admin.billing.publicCatalog.diagnostics.availabilityLabels.limited': '受限',
  'admin.billing.publicCatalog.diagnostics.restrictions.account_rate_limited': '账号限流中',
  'admin.billing.publicCatalog.diagnostics.restrictions.model_rate_limited': '模型限流中',
  'admin.billing.publicCatalog.source.persistedSnapshot': '已持久化候选快照',
  'admin.billing.publicCatalog.source.fallback': '候选快照',
  'admin.billing.publicCatalog.messages.loadFailed': '加载对外模型展示草稿失败',
  'admin.billing.publicCatalog.messages.draftSaved': '对外模型展示草稿已保存',
  'admin.billing.publicCatalog.messages.saveFailed': '保存对外模型展示草稿失败',
  'admin.billing.publicCatalog.messages.published': '公开模型库已推送更新',
  'admin.billing.publicCatalog.messages.publishFailed': '推送公开模型库失败',
  'admin.billing.publicCatalog.messages.diagnosticsLoadFailed': '加载模型健康度容量诊断失败',
  'admin.billing.publicCatalog.messages.billingIncomplete': '发布被阻止：{models} 缺少售卖价字段 {fields}，请补齐后重新发布。',
  'admin.billing.publicCatalog.messages.invalidBatchRatio': '批量比例必须是非负数字',
  'admin.billing.publicCatalog.messages.duplicatePublicId': '公开模型 ID 不能重复：{ids}',
  'admin.billing.publicCatalog.messages.unavailableEntries': '已选条目已不可用，请移除后再保存或发布',
  'admin.billing.publicCatalog.messages.availableSyncedToDraft': '已加入草稿，请保存并发布',
  'admin.billing.publicCatalog.messages.unsaved': '未保存',
}

const apiMocks = vi.hoisted(() => ({
  getBillingPublicCatalogDraft: vi.fn(),
  getBillingPublicCatalogCapacityDiagnostics: vi.fn(),
  getBillingPublicCatalogRevalidationState: vi.fn(),
  saveBillingPublicCatalogDraft: vi.fn(),
  publishBillingPublicCatalog: vi.fn(),
  revalidateBillingPublicCatalog: vi.fn(),
  updateBillingPublicCatalogRevalidationState: vi.fn(),
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

let clipboardText = ''

vi.mock('@/api/admin/billing', () => ({
  getBillingPublicCatalogDraft: apiMocks.getBillingPublicCatalogDraft,
  getBillingPublicCatalogCapacityDiagnostics: apiMocks.getBillingPublicCatalogCapacityDiagnostics,
  getBillingPublicCatalogRevalidationState: apiMocks.getBillingPublicCatalogRevalidationState,
  saveBillingPublicCatalogDraft: apiMocks.saveBillingPublicCatalogDraft,
  publishBillingPublicCatalog: apiMocks.publishBillingPublicCatalog,
  revalidateBillingPublicCatalog: apiMocks.revalidateBillingPublicCatalog,
  updateBillingPublicCatalogRevalidationState: apiMocks.updateBillingPublicCatalogRevalidationState,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: storeMocks.showError,
    showSuccess: storeMocks.showSuccess,
  }),
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: {
        value: 'zh',
      },
      t: (key: string, params?: Record<string, unknown>) => {
        const template = messages[key] ?? key
        return Object.entries(params || {}).reduce(
          (value, [param, replacement]) => value.replaceAll(`{${param}}`, String(replacement)),
          template,
        )
      },
      te: (key: string) => key in messages,
      setLocaleMessage: vi.fn(),
    },
  }),
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const template = messages[key] ?? key
      return Object.entries(params || {}).reduce(
        (value, [param, replacement]) => value.replaceAll(`{${param}}`, String(replacement)),
        template,
      )
    },
    te: (key: string) => key in messages,
  }),
}))

vi.mock('vue-draggable-plus', () => ({
  VueDraggable: defineComponent({
    name: 'VueDraggable',
    props: {
      modelValue: {
        type: Array,
        default: () => [],
      },
    },
    emits: ['update:modelValue'],
    setup(_, { slots }) {
      return () => h('div', { 'data-testid': 'vue-draggable-stub' }, slots.default?.())
    },
  }),
}))

function createCatalogEntry(
  entryId: string,
  model: string,
  sourceAlias: string,
  saleInput = 1.2e-6,
  overrides: Record<string, unknown> = {},
) {
  return {
    entry_id: entryId,
    public_model_id: `${model}@${sourceAlias.toLowerCase()}`,
    model: `${model}@${sourceAlias.toLowerCase()}`,
    base_model: model,
    source_model_id: model,
    source_protocol: 'openai',
    source_alias: sourceAlias,
    source_account_id: sourceAlias === 'primary' ? 10 : 20,
    display_name: 'GPT-5.4',
    provider: 'openai',
    provider_icon_key: 'openai',
    request_protocols: ['openai'],
    context_window: {
      tokens: 128000,
      source: 'account_probe',
      verified: true,
    },
    protocol_endpoints: [
      {
        key: 'openai.responses',
        protocol: 'openai',
        endpoint: '/v1/responses',
        support: 'supported',
        source: 'verified_probe',
        verified: true,
      },
    ],
    capability_matrix: [
      {
        capability: 'text',
        protocol: 'openai',
        endpoint: 'openai.responses',
        support: 'supported',
        source: 'verified_probe',
        verified: true,
      },
    ],
    lifecycle: {
      status: 'stable',
      source: 'official_registry',
      confidence: 'declared',
    },
    catalog_entry_source: 'real_account',
    mode: 'chat',
    currency: 'USD',
    price_display: {
      primary: [
        { id: 'input_price', unit: 'input_token', value: saleInput },
        { id: 'output_price', unit: 'output_token', value: 2.4e-6 },
      ],
    },
    official_price_display: {
      primary: [
        { id: 'input_price', unit: 'input_token', value: 1e-6 },
        { id: 'output_price', unit: 'output_token', value: 2e-6 },
      ],
    },
    sale_price_display: {
      primary: [
        { id: 'input_price', unit: 'input_token', value: saleInput },
        { id: 'output_price', unit: 'output_token', value: 2.4e-6 },
      ],
    },
    multiplier_summary: {
      enabled: false,
      kind: 'disabled',
    },
    ...overrides,
  }
}

function lastSavedEntries() {
  const payload = apiMocks.saveBillingPublicCatalogDraft.mock.calls.at(-1)?.[0]
  return payload?.selected_entries || []
}

function savedEntry(entryID: string) {
  return lastSavedEntries().find((entry: { entry_id: string }) => entry.entry_id === entryID)
}

function mountView() {
  return mount(BillingPublicCatalogView, {
    global: {
      stubs: {
        ModelIcon: true,
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<div v-if="show" data-testid="base-dialog"><slot /><slot name="footer" /></div>',
        },
      },
    },
  })
}

describe('BillingPublicCatalogView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clipboardText = ''
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn(async (value: string) => {
          clipboardText = value
        }),
      },
    })
    apiMocks.getBillingPublicCatalogDraft.mockResolvedValue({
      draft: {
        selected_entries: [
          {
            entry_id: 'acct_a',
            public_model_id: 'gpt-5.4@primary',
            source_alias: 'primary',
            source_model_id: 'gpt-5.4',
            base_model: 'gpt-5.4',
            source_protocol: 'openai',
            sale_price_display: {
              primary: [
                { id: 'input_price', unit: 'input_token', value: 1.2e-6 },
                { id: 'output_price', unit: 'output_token', value: 2.4e-6 },
              ],
            },
          },
        ],
        page_size: 10,
        updated_at: '2026-04-20T10:00:00Z',
      },
      available_entries: [
        createCatalogEntry('acct_a', 'gpt-5.4', 'primary'),
        createCatalogEntry('acct_b', 'gpt-5.4', 'backup', 1.1e-6),
        createCatalogEntry('acct_c', 'claude-3-haiku', 'bedrock', 0.5e-6, {
          display_name: 'Claude 3 Haiku',
          provider: 'anthropic',
          provider_icon_key: 'anthropic',
          source_protocol: 'anthropic',
          request_protocols: ['anthropic'],
        }),
      ],
      available_items: [],
      available_updated_at: '2026-04-20T10:30:00Z',
      available_source: 'persisted_snapshot',
      published: {
        etag: 'W/"published"',
        updated_at: '2026-04-19T09:00:00Z',
        page_size: 10,
        model_count: 1,
      },
    })
    apiMocks.getBillingPublicCatalogRevalidationState.mockResolvedValue({ auto_enabled: false })
    apiMocks.updateBillingPublicCatalogRevalidationState.mockResolvedValue({ auto_enabled: true })
    apiMocks.revalidateBillingPublicCatalog.mockResolvedValue({
      published: {
        etag: 'W/"published-revalidated"',
        updated_at: '2026-04-20T11:30:00Z',
        page_size: 10,
        model_count: 1,
      },
      checked_at: '2026-04-20T11:30:00Z',
      model_count: 1,
      stale_count: 0,
      reasons: {},
    })
    apiMocks.getBillingPublicCatalogCapacityDiagnostics.mockResolvedValue({
      updated_at: '2026-04-20T10:31:00Z',
      summary: {
        model_count: 1,
        available_count: 0,
        limited_count: 1,
        unschedulable_count: 0,
        restriction_counts: {
          account_rate_limited: 1,
        },
      },
      items: [
        {
          public_model_id: 'gpt-5.4@primary',
          model: 'gpt-5.4@primary',
          provider: 'openai',
          source_protocol: 'openai',
          source_account_id: 10,
          availability: 'limited',
          effective_rate_limit: { rpm: 60 },
          restrictions: [{ kind: 'account_rate_limited', scope: 'account' }],
          sources: [{ source: 'account_pool', scope: 'account' }],
        },
      ],
    })
    apiMocks.saveBillingPublicCatalogDraft.mockImplementation(async (payload) => ({
      ...payload,
      updated_at: '2026-04-20T11:00:00Z',
    }))
    apiMocks.publishBillingPublicCatalog.mockResolvedValue({
      etag: 'W/"published-next"',
      updated_at: '2026-04-20T11:00:00Z',
      page_size: 20,
      model_count: 2,
    })
  })

  it('loads account model entries and forces refresh when manually reloading', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(apiMocks.getBillingPublicCatalogDraft).toHaveBeenNthCalledWith(1, { force: false })
    expect(wrapper.text()).toContain('账号支持的模型库')
    expect(wrapper.text()).toContain('primary')
    expect(wrapper.text()).toContain('backup')
    expect(wrapper.text()).toContain('账号来源')
    expect(wrapper.text()).toContain('发布分页')
    expect(wrapper.text()).toContain('128K')
    expect(wrapper.text()).toContain('已验证')
    expect(wrapper.text()).toContain('openai.responses')
    expect(wrapper.text()).toContain('text')

    await wrapper.findAll('button').find((node) => node.text().includes('刷新候选'))!.trigger('click')
    await flushPromises()

    expect(apiMocks.getBillingPublicCatalogDraft).toHaveBeenLastCalledWith({ force: true })
  })

  it('syncs all currently available entries into the draft without saving or publishing', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="billing-public-catalog-sync-available"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).not.toHaveBeenCalled()
    expect(apiMocks.publishBillingPublicCatalog).not.toHaveBeenCalled()
    expect(storeMocks.showSuccess).toHaveBeenCalledWith('已加入草稿，请保存并发布')

    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(lastSavedEntries()).toEqual([
      expect.objectContaining({
        entry_id: 'acct_a',
        sale_price_display: expect.objectContaining({
          primary: expect.arrayContaining([
            expect.objectContaining({ id: 'input_price', value: 1.2e-6 }),
          ]),
        }),
      }),
      expect.objectContaining({
        entry_id: 'acct_b',
        public_model_id: 'gpt-5.4@backup',
        sale_price_display: expect.objectContaining({
          primary: expect.arrayContaining([
            expect.objectContaining({ id: 'input_price', value: 1.1e-6 }),
          ]),
        }),
      }),
      expect.objectContaining({
        entry_id: 'acct_c',
        public_model_id: 'claude-3-haiku@bedrock',
      }),
    ])
  })

  it('keeps export and publish disabled when no entries are selected', async () => {
    apiMocks.getBillingPublicCatalogDraft.mockResolvedValueOnce({
      draft: {
        selected_entries: [],
        page_size: 10,
        updated_at: '2026-04-20T10:00:00Z',
      },
      available_entries: [
        createCatalogEntry('acct_a', 'gpt-5.4', 'primary'),
      ],
      available_items: [],
      available_updated_at: '2026-04-20T10:30:00Z',
      available_source: 'persisted_snapshot',
      published: null,
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-testid="billing-public-catalog-publish"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="billing-public-catalog-export"]').attributes('disabled')).toBeDefined()
  })

  it('marks demo candidates in the admin draft list', async () => {
    apiMocks.getBillingPublicCatalogDraft.mockResolvedValueOnce({
      draft: {
        selected_entries: [],
        page_size: 10,
        updated_at: '2026-04-20T10:00:00Z',
      },
      available_entries: [
        createCatalogEntry('demo_a', 'demo-model', 'demo', 1e-6, {
          is_demo: true,
          catalog_entry_source: 'demo',
        }),
      ],
      available_items: [],
      available_updated_at: '2026-04-20T10:30:00Z',
      available_source: 'persisted_snapshot',
      published: null,
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('演示数据')
  })

  it('keeps legacy draft candidates usable when structured metadata is absent', async () => {
    apiMocks.getBillingPublicCatalogDraft.mockResolvedValueOnce({
      draft: {
        selected_entries: [],
        page_size: 10,
        updated_at: '2026-04-20T10:00:00Z',
      },
      available_entries: [
        createCatalogEntry('legacy_a', 'legacy-model', 'legacy', 1e-6, {
          context_window: undefined,
          context_window_tokens: 64000,
          protocol_endpoints: undefined,
          capability_matrix: undefined,
          lifecycle: undefined,
          lifecycle_status: 'beta',
          capabilities: ['tools'],
          request_protocols: ['openai'],
        }),
      ],
      available_items: [],
      available_updated_at: '2026-04-20T10:30:00Z',
      available_source: 'persisted_snapshot',
      published: null,
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('legacy-model')
    expect(wrapper.text()).toContain('64K')

    await wrapper.get('[data-testid="add-entry-legacy_a"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('legacy_a')).toMatchObject({
      entry_id: 'legacy_a',
      source_model_id: 'legacy-model',
    })
  })

  it('adds duplicate base models as separate public entries and publishes selected_entries', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="add-entry-acct_b"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-page-size"]').setValue('20')
    await wrapper.get('[data-testid="billing-public-catalog-publish"]').trigger('click')
    await flushPromises()

    expect(apiMocks.publishBillingPublicCatalog).toHaveBeenCalledWith(expect.objectContaining({
      page_size: 20,
      selected_models: ['gpt-5.4@primary', 'gpt-5.4@backup'],
      selected_entries: [
        expect.objectContaining({
          entry_id: 'acct_a',
          public_model_id: 'gpt-5.4@primary',
          source_alias: 'primary',
          source_model_id: 'gpt-5.4',
        }),
        expect.objectContaining({
          entry_id: 'acct_b',
          public_model_id: 'gpt-5.4@backup',
          source_alias: 'backup',
          source_model_id: 'gpt-5.4',
        }),
      ],
    }))
    expect(storeMocks.showSuccess).toHaveBeenCalledWith('公开模型库已推送更新')
  })

  it('edits public id, source alias, and sale price before saving', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="public-id-acct_a"]').setValue('gpt-5.4-premium')
    await wrapper.get('[data-testid="source-alias-acct_a"]').setValue('premium')
    await wrapper.get('[data-testid="price-acct_a-input_price"]').setValue('0.000003')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({
          entry_id: 'acct_a',
          public_model_id: 'gpt-5.4-premium',
          source_alias: 'premium',
          sale_price_display: expect.objectContaining({
            primary: expect.arrayContaining([
              expect.objectContaining({ id: 'input_price', value: 0.000003 }),
            ]),
          }),
        }),
      ],
    }))
  })

  it('edits selected entry in the dialog and can cancel without saving changes', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="edit-entry-acct_a"]').trigger('click')
    await wrapper.get('[data-testid="catalog-dialog-public-id"]').setValue('gpt-5.4-dialog')
    await wrapper.findAll('button').find((button) => button.text() === '取消')!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenLastCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({
          public_model_id: 'gpt-5.4@primary',
        }),
      ],
    }))

    await wrapper.get('[data-testid="edit-entry-acct_a"]').trigger('click')
    await wrapper.get('[data-testid="catalog-dialog-public-id"]').setValue('gpt-5.4-dialog')
    await wrapper.get('[data-testid="catalog-dialog-source-alias"]').setValue('dialog-source')
    await wrapper.get('[data-testid="catalog-dialog-price-input_price"]').setValue('0.000004')
    await wrapper.get('[data-testid="catalog-dialog-save"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenLastCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({
          entry_id: 'acct_a',
          public_model_id: 'gpt-5.4-dialog',
          source_alias: 'dialog-source',
          sale_price_display: expect.objectContaining({
            primary: expect.arrayContaining([
              expect.objectContaining({ id: 'input_price', value: 0.000004 }),
            ]),
          }),
        }),
      ],
    }))
  })

  it('saves scheduled availability and time access policy from the dialog', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="edit-entry-acct_a"]').trigger('click')
    await wrapper.get('[data-testid="catalog-dialog-available-from"]').setValue('2026-06-01T08:00')
    await wrapper.get('[data-testid="catalog-dialog-time-access-enabled"]').setValue(true)
    await flushPromises()
    await wrapper.get('[data-testid="catalog-dialog-save"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenLastCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({
          entry_id: 'acct_a',
          available_from: localDateTimeToISOString('2026-06-01T08:00'),
          access_time_policy: expect.objectContaining({
            enabled: true,
            timezone: 'Asia/Singapore',
            weekly_windows: expect.arrayContaining([
              expect.objectContaining({ start: '08:00', end: '20:00' }),
            ]),
          }),
        }),
      ],
    }))
  })

  it('reorders selected entries from drag updates and preserves keyboard move buttons', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="add-entry-acct_b"]').trigger('click')
    const draggable = wrapper.findComponent({ name: 'VueDraggable' })
    const selected = draggable.props('modelValue') as Array<{ entry_id: string }>
    await draggable.vm.$emit('update:modelValue', [selected[1], selected[0]])
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenLastCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({ entry_id: 'acct_b' }),
        expect.objectContaining({ entry_id: 'acct_a' }),
      ],
    }))

    const moveUpButtons = wrapper.findAll('button').filter((button) => button.text() === '上移')
    await moveUpButtons.at(-1)!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenLastCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({ entry_id: 'acct_a' }),
        expect.objectContaining({ entry_id: 'acct_b' }),
      ],
    }))
  })

  it('blocks duplicate public model ids before publishing', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="add-entry-acct_b"]').trigger('click')
    await wrapper.get('[data-testid="public-id-acct_b"]').setValue('gpt-5.4@primary')
    await wrapper.get('[data-testid="billing-public-catalog-publish"]').trigger('click')
    await flushPromises()

    expect(apiMocks.publishBillingPublicCatalog).not.toHaveBeenCalled()
    expect(storeMocks.showError).toHaveBeenCalledWith(expect.stringContaining('公开模型 ID 不能重复'))
  })

  it('shows publish billing coverage errors with repairable fields', async () => {
    apiMocks.publishBillingPublicCatalog.mockRejectedValueOnce({
      reason: 'PUBLIC_MODEL_BILLING_INCOMPLETE',
      message: 'public model billing price is incomplete',
      metadata: {
        public_model_ids: 'gpt-5.4@primary',
        missing_fields: 'cache_creation,cache_read',
      },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="billing-public-catalog-publish"]').trigger('click')
    await flushPromises()

    expect(storeMocks.showError).toHaveBeenCalledWith(expect.stringContaining('gpt-5.4@primary'))
    expect(storeMocks.showError).toHaveBeenCalledWith(expect.stringContaining('缓存写入'))
    expect(storeMocks.showError).toHaveBeenCalledWith(expect.stringContaining('缓存读取'))
  })

  it('renders admin-only capacity diagnostics', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(apiMocks.getBillingPublicCatalogCapacityDiagnostics).toHaveBeenCalled()
    expect(wrapper.text()).toContain('模型健康度容量诊断')
    expect(wrapper.text()).toContain('账号限流中')
    expect(wrapper.text()).toContain('RPM 60')
  })

  it('filters by provider chips and adds the filtered entries only', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text().includes('Anthropic'))!.trigger('click')
    await wrapper.findAll('button').find((button) => button.text().includes('全部添加'))!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({ entry_id: 'acct_a' }),
        expect.objectContaining({
          entry_id: 'acct_c',
          public_model_id: 'claude-3-haiku@bedrock',
          source_protocol: 'anthropic',
        }),
      ],
    }))
  })

  it('filters by search text and account source before adding current results', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="billing-public-catalog-search"]').setValue('claude')
    await wrapper.findAll('button').find((button) => button.text().includes('全部添加'))!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_c')).toEqual(expect.objectContaining({
      public_model_id: 'claude-3-haiku@bedrock',
    }))
    expect(savedEntry('acct_b')).toBeUndefined()

    await wrapper.get('[data-testid="billing-public-catalog-search"]').setValue('')
    await wrapper.get('[data-testid="billing-public-catalog-account-filter"]').setValue('backup')
    await wrapper.findAll('button').find((button) => button.text().includes('全部添加'))!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_b')).toEqual(expect.objectContaining({
      source_alias: 'backup',
    }))
  })

  it('removes entries and clears the selected catalog before saving', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="add-entry-acct_b"]').trigger('click')
    await wrapper.get('[data-testid="remove-entry-acct_b"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(lastSavedEntries()).toEqual([
      expect.objectContaining({ entry_id: 'acct_a' }),
    ])

    await wrapper.get('[data-testid="billing-public-catalog-clear"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(lastSavedEntries()).toEqual([])
  })

  it('applies batch ratio to a single source alias without changing other selected entries', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="add-entry-acct_b"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-batch-ratio"]').setValue('150')
    await wrapper.findAll('button').find((button) => button.text().includes('右侧已选展示'))!.trigger('click')
    await wrapper.findAll('button').find((button) => button.text().includes('仅来源：backup'))!.trigger('click')
    await wrapper.findAll('button').find((button) => button.text().includes('按官方参考价应用'))!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(apiMocks.saveBillingPublicCatalogDraft).toHaveBeenCalledWith(expect.objectContaining({
      selected_entries: [
        expect.objectContaining({
          entry_id: 'acct_a',
          sale_price_display: expect.objectContaining({
            primary: expect.arrayContaining([
              expect.objectContaining({ id: 'input_price', value: 1.2e-6 }),
            ]),
          }),
        }),
        expect.objectContaining({
          entry_id: 'acct_b',
          sale_price_display: expect.objectContaining({
            primary: expect.arrayContaining([
              expect.objectContaining({ id: 'input_price', value: 0.0000015 }),
              expect.objectContaining({ id: 'output_price', value: 0.000003 }),
            ]),
          }),
        }),
      ],
    }))
  })

  it('applies batch ratio only to selected entries in selected, filtered, and all scopes', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="add-entry-acct_b"]').trigger('click')
    await wrapper.get('[data-testid="add-entry-acct_c"]').trigger('click')

    await wrapper.get('[data-testid="billing-public-catalog-batch-ratio"]').setValue('110')
    await wrapper.findAll('button').find((button) => button.text().includes('按官方参考价应用'))!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_a').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000011 }),
    ]))
    expect(savedEntry('acct_b').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000011 }),
    ]))

    await wrapper.get('[data-testid="billing-public-catalog-search"]').setValue('claude')
    await wrapper.get('[data-testid="billing-public-catalog-batch-ratio"]').setValue('130')
    await wrapper.get('[data-testid="billing-public-catalog-scope"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-scope-filtered"]').trigger('click')
    await wrapper.findAll('button').find((button) => button.text().includes('按官方参考价应用'))!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_a').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000011 }),
    ]))
    expect(savedEntry('acct_c').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000013 }),
    ]))

    await wrapper.get('[data-testid="billing-public-catalog-batch-ratio"]').setValue('140')
    await wrapper.get('[data-testid="billing-public-catalog-scope"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-scope-all"]').trigger('click')
    await wrapper.findAll('button').find((button) => button.text().includes('按官方参考价应用'))!.trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_a').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000014 }),
    ]))
    expect(savedEntry('acct_b').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000014 }),
    ]))
    expect(savedEntry('acct_c').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000014 }),
    ]))
    expect(lastSavedEntries()).toHaveLength(3)
  })

  it('applies single-card ratios from selected cards and dialog without exposing candidate price inputs', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="price-acct_b-input_price"]').exists()).toBe(false)

    await wrapper.get('[data-testid="price-acct_a-ratio"]').setValue('125')
    await wrapper.get('[data-testid="price-acct_a-apply-ratio"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_a').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.00000125 }),
      expect.objectContaining({ id: 'output_price', value: 0.0000025 }),
    ]))

    await wrapper.get('[data-testid="edit-entry-acct_a"]').trigger('click')
    await wrapper.get('[data-testid="catalog-dialog-price-ratio"]').setValue('150')
    await wrapper.get('[data-testid="catalog-dialog-price-apply-ratio"]').trigger('click')
    await wrapper.get('[data-testid="catalog-dialog-save"]').trigger('click')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_a').sale_price_display.primary).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'input_price', value: 0.0000015 }),
      expect.objectContaining({ id: 'output_price', value: 0.000003 }),
    ]))
  })

  it('preserves secondary and extra price entries when editing compact price rows', async () => {
    apiMocks.getBillingPublicCatalogDraft.mockResolvedValueOnce({
      draft: {
        selected_entries: [
          {
            entry_id: 'acct_multi',
            public_model_id: 'gpt-5.4-multi',
            source_alias: 'multi',
            source_model_id: 'gpt-5.4',
            base_model: 'gpt-5.4',
            source_protocol: 'openai',
            sale_price_display: {
              primary: [
                { id: 'input_price', unit: 'input_token', value: 1.2e-6 },
                { id: 'output_price', unit: 'output_token', value: 2.4e-6 },
                { id: 'cache_price', unit: 'input_token', value: 0.2e-6 },
              ],
              secondary: [
                { id: 'search_unit_price', unit: 'request', value: 0.01 },
              ],
            },
          },
        ],
        page_size: 10,
        updated_at: '2026-04-20T10:00:00Z',
      },
      available_entries: [
        createCatalogEntry('acct_multi', 'gpt-5.4', 'multi', 1.2e-6, {
          official_price_display: {
            primary: [
              { id: 'input_price', unit: 'input_token', value: 1e-6 },
              { id: 'output_price', unit: 'output_token', value: 2e-6 },
              { id: 'cache_price', unit: 'input_token', value: 0.1e-6 },
            ],
            secondary: [
              { id: 'search_unit_price', unit: 'request', value: 0.008 },
            ],
          },
          sale_price_display: {
            primary: [
              { id: 'input_price', unit: 'input_token', value: 1.2e-6 },
              { id: 'output_price', unit: 'output_token', value: 2.4e-6 },
              { id: 'cache_price', unit: 'input_token', value: 0.2e-6 },
            ],
            secondary: [
              { id: 'search_unit_price', unit: 'request', value: 0.01 },
            ],
          },
        }),
      ],
      available_items: [],
      available_updated_at: '2026-04-20T10:30:00Z',
      available_source: 'persisted_snapshot',
      published: null,
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="price-acct_multi-cache_price"]').setValue('0.0000003')
    await wrapper.get('[data-testid="price-acct_multi-search_unit_price"]').setValue('0.02')
    await wrapper.get('[data-testid="billing-public-catalog-save"]').trigger('click')
    await flushPromises()

    expect(savedEntry('acct_multi').sale_price_display).toEqual({
      primary: [
        expect.objectContaining({ id: 'input_price', value: 1.2e-6 }),
        expect.objectContaining({ id: 'output_price', value: 2.4e-6 }),
        expect.objectContaining({ id: 'cache_price', value: 0.0000003 }),
      ],
      secondary: [
        expect.objectContaining({ id: 'search_unit_price', value: 0.02 }),
      ],
    })
  })

  it('copies the public model id from the card action', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('button[aria-label="复制公开模型 ID"]').trigger('click')

    expect(clipboardText).toBe('gpt-5.4@primary')
  })
})
