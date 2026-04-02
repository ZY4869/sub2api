import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import OpsAlertEventsCard from '../OpsAlertEventsCard.vue'

const listAlertEvents = vi.fn()
const getAlertEvent = vi.fn()
const createAlertSilence = vi.fn()
const updateAlertEventStatus = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listAlertEvents: (...args: any[]) => listAlertEvents(...args),
    getAlertEvent: (...args: any[]) => getAlertEvent(...args),
    createAlertSilence: (...args: any[]) => createAlertSilence(...args),
    updateAlertEventStatus: (...args: any[]) => updateAlertEventStatus(...args),
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => ({
        'common.all': '全部',
        'common.refresh': '刷新',
        'common.apply': '应用',
        'admin.ops.timeRange.7d': '7天',
        'admin.ops.timeRange.30d': '30天',
        'admin.ops.timeRange.1h': '1小时',
        'admin.ops.timeRange.24h': '24小时',
        'admin.ops.timeRange.5m': '5分钟',
        'admin.ops.timeRange.30m': '30分钟',
        'admin.ops.timeRange.6h': '6小时',
        'admin.ops.alertEvents.title': '告警事件',
        'admin.ops.alertEvents.description': '告警列表',
        'admin.ops.alertEvents.loading': '加载中',
        'admin.ops.alertEvents.empty': '暂无数据',
        'admin.ops.alertEvents.status.firing': '告警中',
        'admin.ops.alertEvents.status.resolved': '已恢复',
        'admin.ops.alertEvents.status.manualResolved': '手动恢复',
        'admin.ops.alertEvents.table.time': '时间',
        'admin.ops.alertEvents.table.severity': '级别',
        'admin.ops.alertEvents.table.platform': '平台',
        'admin.ops.alertEvents.table.ruleId': '规则',
        'admin.ops.alertEvents.table.title': '标题',
        'admin.ops.alertEvents.table.duration': '持续时间',
        'admin.ops.alertEvents.table.dimensions': '维度',
        'admin.ops.alertEvents.table.email': '邮件',
        'admin.ops.alertEvents.table.emailSent': '已发送',
        'admin.ops.alertEvents.table.emailIgnored': '未发送',
        'admin.ops.alertEvents.detail.title': '告警详情',
        'admin.ops.alertEvents.detail.loading': '加载中',
        'admin.ops.alertEvents.detail.empty': '暂无详情',
        'admin.ops.alertEvents.detail.silence': '静默',
        'admin.ops.alertEvents.detail.manualResolve': '手动恢复',
        'admin.ops.alertEvents.detail.firedAt': '触发时间',
        'admin.ops.alertEvents.detail.resolvedAt': '恢复时间',
        'admin.ops.alertEvents.detail.ruleId': '规则 ID',
        'admin.ops.alertEvents.detail.viewRule': '查看规则',
        'admin.ops.alertEvents.detail.viewLogs': '查看日志',
        'admin.ops.alertEvents.detail.dimensions': '维度',
        'admin.ops.alertEvents.detail.historyTitle': '历史',
        'admin.ops.alertEvents.detail.historyHint': '历史记录',
        'admin.ops.alertEvents.detail.historyLoading': '加载中',
        'admin.ops.alertEvents.detail.historyEmpty': '暂无历史',
        'admin.ops.alertEvents.table.metric': '指标',
        'ui.opsDimensions.platform': '平台',
        'ui.opsDimensions.groupId': '分组 ID',
        'ui.opsDimensions.region': '地区',
      }[key] || key)
    })
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  })
}))

const SelectStub = defineComponent({
  props: ['modelValue', 'options'],
  emits: ['change'],
  template: '<div class="select-stub"></div>'
})

const BaseDialogStub = defineComponent({
  props: ['show', 'title'],
  emits: ['close'],
  template: '<div class="dialog-stub"><div v-if="show"><slot /></div></div>'
})

const IconStub = defineComponent({
  template: '<span class="icon-stub">icon</span>'
})

const sampleEvent = {
  id: 1,
  rule_id: 9,
  severity: 'P1',
  status: 'firing',
  title: '429 激增',
  description: '最近 5 分钟出现大量 429',
  email_sent: true,
  fired_at: '2026-04-02T12:00:00Z',
  created_at: '2026-04-02T12:00:00Z',
  resolved_at: null,
  metric_value: 12,
  threshold_value: 5,
  dimensions: {
    platform: 'openai',
    group_id: 7,
    region: 'us-east-1'
  }
}

describe('OpsAlertEventsCard', () => {
  it('renders localized dimension labels in the summary row and detail dialog', async () => {
    listAlertEvents.mockResolvedValue([sampleEvent])
    getAlertEvent.mockResolvedValue(sampleEvent)

    const wrapper = mount(OpsAlertEventsCard, {
      global: {
        stubs: {
          Select: SelectStub,
          BaseDialog: BaseDialogStub,
          Icon: IconStub
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('平台：openai · 分组 ID：7 · 地区：美国东部 1（us-east-1）')

    await wrapper.find('tbody tr').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('平台：openai')
    expect(wrapper.text()).toContain('分组 ID：7')
    expect(wrapper.text()).toContain('地区：美国东部 1（us-east-1）')
  })
})
