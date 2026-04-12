import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import OpsAlertRulesCard from '../OpsAlertRulesCard.vue'

const listAlertRules = vi.fn()
const createAlertRule = vi.fn()
const updateAlertRule = vi.fn()
const deleteAlertRule = vi.fn()
const getAllGroups = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listAlertRules: (...args: any[]) => listAlertRules(...args),
    createAlertRule: (...args: any[]) => createAlertRule(...args),
    updateAlertRule: (...args: any[]) => updateAlertRule(...args),
    deleteAlertRule: (...args: any[]) => deleteAlertRule(...args)
  }
}))

vi.mock('@/api', () => ({
  adminAPI: {
    groups: {
      getAll: (...args: any[]) => getAllGroups(...args)
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('en'),
      t: (key: string, params?: Record<string, string | number>) => {
        const translations: Record<string, string> = {
          'common.refresh': 'Refresh',
          'common.enabled': 'Enabled',
          'common.disabled': 'Disabled',
          'common.edit': 'Edit',
          'common.delete': 'Delete',
          'common.cancel': 'Cancel',
          'common.save': 'Save',
          'common.saving': 'Saving',
          'admin.ops.alertRules.title': 'Alert Rules',
          'admin.ops.alertRules.description': 'Manage alert rules',
          'admin.ops.alertRules.loading': 'Loading',
          'admin.ops.alertRules.empty': 'No rules',
          'admin.ops.alertRules.create': 'Create Rule',
          'admin.ops.alertRules.createTitle': 'Create Alert Rule',
          'admin.ops.alertRules.editTitle': 'Edit Alert Rule',
          'admin.ops.alertRules.table.name': 'Name',
          'admin.ops.alertRules.table.metric': 'Metric',
          'admin.ops.alertRules.table.severity': 'Severity',
          'admin.ops.alertRules.table.enabled': 'Enabled',
          'admin.ops.alertRules.table.actions': 'Actions',
          'admin.ops.alertRules.metricGroups.system': 'System Metrics',
          'admin.ops.alertRules.metricGroups.group': 'Group Metrics',
          'admin.ops.alertRules.metricGroups.account': 'Account Metrics',
          'admin.ops.alertRules.metrics.successRate': 'Success Rate',
          'admin.ops.alertRules.metrics.errorRate': 'Error Rate',
          'admin.ops.alertRules.metrics.upstreamErrorRate': 'Upstream Error Rate',
          'admin.ops.alertRules.metrics.cpu': 'CPU',
          'admin.ops.alertRules.metrics.memory': 'Memory',
          'admin.ops.alertRules.metrics.queueDepth': 'Queue Depth',
          'admin.ops.alertRules.metrics.groupAvailableAccounts': 'Group Available Accounts',
          'admin.ops.alertRules.metrics.groupAvailableRatio': 'Group Available Ratio',
          'admin.ops.alertRules.metrics.groupRateLimitRatio': 'Group Rate Limit Ratio',
          'admin.ops.alertRules.metrics.accountRateLimitedCount': 'Account Rate Limited Count',
          'admin.ops.alertRules.metrics.accountErrorCount': 'Account Error Count',
          'admin.ops.alertRules.metrics.accountErrorRatio': 'Account Error Ratio',
          'admin.ops.alertRules.metrics.overloadAccountCount': 'Overload Account Count',
          'admin.ops.alertRules.metrics.recoveryProbeStartedCount': 'Recovery Probe Started',
          'admin.ops.alertRules.metrics.recoveryProbeSuccessCount': 'Recovery Probe Success',
          'admin.ops.alertRules.metrics.recoveryProbeRetryCount': 'Recovery Probe Retry',
          'admin.ops.alertRules.metrics.recoveryProbeBlacklistedCount': 'Recovery Probe Blacklisted',
          'admin.ops.alertRules.metrics.geminiBillingFallbackAppliedCount': 'Gemini Fallback Applied',
          'admin.ops.alertRules.metrics.geminiBillingFallbackMissCount': 'Gemini Fallback Miss',
          'admin.ops.alertRules.metricDescriptions.successRate': 'success rate',
          'admin.ops.alertRules.metricDescriptions.errorRate': 'error rate',
          'admin.ops.alertRules.metricDescriptions.upstreamErrorRate': 'upstream error rate',
          'admin.ops.alertRules.metricDescriptions.cpu': 'cpu',
          'admin.ops.alertRules.metricDescriptions.memory': 'memory',
          'admin.ops.alertRules.metricDescriptions.queueDepth': 'queue depth',
          'admin.ops.alertRules.metricDescriptions.groupAvailableAccounts': 'group accounts',
          'admin.ops.alertRules.metricDescriptions.groupAvailableRatio': 'group ratio',
          'admin.ops.alertRules.metricDescriptions.groupRateLimitRatio': 'group rate limit ratio',
          'admin.ops.alertRules.metricDescriptions.accountRateLimitedCount': 'account rate limited count',
          'admin.ops.alertRules.metricDescriptions.accountErrorCount': 'account error count',
          'admin.ops.alertRules.metricDescriptions.accountErrorRatio': 'account error ratio',
          'admin.ops.alertRules.metricDescriptions.overloadAccountCount': 'overload account count',
          'admin.ops.alertRules.metricDescriptions.recoveryProbeStartedCount': 'recovery probe started',
          'admin.ops.alertRules.metricDescriptions.recoveryProbeSuccessCount': 'recovery probe success',
          'admin.ops.alertRules.metricDescriptions.recoveryProbeRetryCount': 'recovery probe retry',
          'admin.ops.alertRules.metricDescriptions.recoveryProbeBlacklistedCount': 'recovery probe blacklisted',
          'admin.ops.alertRules.metricDescriptions.geminiBillingFallbackAppliedCount': 'gemini fallback applied',
          'admin.ops.alertRules.metricDescriptions.geminiBillingFallbackMissCount': 'gemini fallback miss',
          'admin.ops.alertRules.hints.recommended': 'Recommended {operator} {threshold}{unit}',
          'admin.ops.alertRules.hints.groupRequired': 'Group required',
          'admin.ops.alertRules.hints.groupOptional': 'Group optional',
          'admin.ops.alertRules.hints.reasonOptional': 'Reason optional',
          'admin.ops.alertRules.form.name': 'Name',
          'admin.ops.alertRules.form.description': 'Description',
          'admin.ops.alertRules.form.metric': 'Metric',
          'admin.ops.alertRules.form.operator': 'Operator',
          'admin.ops.alertRules.form.groupId': 'Group',
          'admin.ops.alertRules.form.groupPlaceholder': 'Pick group',
          'admin.ops.alertRules.form.allGroups': 'All Groups',
          'admin.ops.alertRules.form.reason': 'Reason',
          'admin.ops.alertRules.form.reasonPlaceholder': 'Exact reason bucket',
          'admin.ops.alertRules.form.threshold': 'Threshold',
          'admin.ops.alertRules.form.severity': 'Severity',
          'admin.ops.alertRules.form.window': 'Window',
          'admin.ops.alertRules.form.sustained': 'Sustained',
          'admin.ops.alertRules.form.cooldown': 'Cooldown',
          'admin.ops.alertRules.form.enabled': 'Enabled',
          'admin.ops.alertRules.form.notifyEmail': 'Notify Email',
          'admin.ops.alertRules.validation.title': 'Validation',
          'admin.ops.alertRules.validation.invalid': 'Invalid',
          'admin.ops.alertRules.validation.nameRequired': 'Name required',
          'admin.ops.alertRules.validation.metricRequired': 'Metric required',
          'admin.ops.alertRules.validation.groupIdRequired': 'Group required',
          'admin.ops.alertRules.validation.operatorRequired': 'Operator required',
          'admin.ops.alertRules.validation.thresholdRequired': 'Threshold required',
          'admin.ops.alertRules.validation.windowRange': 'Window range',
          'admin.ops.alertRules.validation.sustainedRange': 'Sustained range',
          'admin.ops.alertRules.validation.cooldownRange': 'Cooldown range'
        }
        const template = translations[key] ?? key
        return Object.entries(params || {}).reduce(
          (result, [paramKey, value]) => result.replace(`{${paramKey}}`, String(value)),
          template
        )
      }
    })
  }
})

const SelectStub = defineComponent({
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: null
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue ?? ''"
      @change="$emit('update:modelValue', $event.target ? $event.target.value : '')"
    >
      <option
        v-for="option in options"
        :key="String(option.value)"
        :value="option.value ?? ''"
        :disabled="option.disabled"
      >
        {{ option.label }}
      </option>
    </select>
  `
})

const BaseDialogStub = defineComponent({
  props: ['show', 'title'],
  emits: ['close'],
  template: `
    <div v-if="show" data-test="base-dialog">
      <h3>{{ title }}</h3>
      <slot />
      <slot name="footer" />
    </div>
  `
})

const ConfirmDialogStub = defineComponent({
  template: '<div data-test="confirm-dialog" />'
})

describe('OpsAlertRulesCard', () => {
  it('shows the reason filter only for the new reason-aware metrics', async () => {
    listAlertRules.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])

    const wrapper = mount(OpsAlertRulesCard, {
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          ConfirmDialog: ConfirmDialogStub,
          Select: SelectStub
        }
      }
    })

    await flushPromises()

    await wrapper.find('button.btn.btn-sm.btn-primary').trigger('click')

    const metricSelect = wrapper.find('[data-test="alert-rule-metric-select"]')
    expect(metricSelect.exists()).toBe(true)

    await metricSelect.setValue('gemini_billing_fallback_miss_count')
    await flushPromises()

    const reasonInput = wrapper.find('[data-test="alert-rule-reason-input"]')
    expect(reasonInput.exists()).toBe(true)

    await reasonInput.setValue('no_billing_rule_match')
    expect((reasonInput.element as HTMLInputElement).value).toBe('no_billing_rule_match')

    const metricOptionsText = metricSelect.text()
    expect(metricOptionsText).toContain('Recovery Probe Started')
    expect(metricOptionsText).toContain('Gemini Fallback Applied')
    expect(metricOptionsText).toContain('Gemini Fallback Miss')

    await metricSelect.setValue('error_rate')
    await flushPromises()

    expect(wrapper.find('[data-test="alert-rule-reason-input"]').exists()).toBe(false)
  })
})
