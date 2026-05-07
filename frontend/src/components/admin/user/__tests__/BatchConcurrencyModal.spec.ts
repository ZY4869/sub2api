import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import BatchConcurrencyModal from '../BatchConcurrencyModal.vue'

const testState = vi.hoisted(() => ({
  batchUpdateConcurrencyMock: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const dictionary: Record<string, string> = {
          'admin.users.batchConcurrencyTitle': 'Batch Update Concurrency',
          'admin.users.batchConcurrencyValue': 'Target Concurrency',
          'admin.users.batchConcurrencyPlaceholder': 'Enter concurrency',
          'admin.users.batchConcurrencySubmit': 'Apply',
          'admin.users.batchConcurrencySummary': `Matched ${(params?.count as number) ?? 0} users`,
          'admin.users.batchConcurrencySummaryUnknown': 'Matched on server',
          'admin.users.batchConcurrencySuccess': `Success ${(params?.success as number) ?? 0}/${(params?.failed as number) ?? 0}`,
          'admin.users.batchConcurrencyPartial': `Partial ${(params?.success as number) ?? 0}/${(params?.failed as number) ?? 0}`,
          'admin.users.batchConcurrencyNoTargets': 'No targets',
          'admin.users.batchConcurrencyFailed': 'Request failed',
          'admin.users.batchConcurrencyFailureDetailUnknown': 'Unknown error',
          'admin.users.batchConcurrencyFailureDetail': `${params?.email}: ${params?.error}`,
          'admin.users.concurrencyMin': 'Concurrency must be at least 1',
          'common.cancel': 'Cancel',
          'common.saving': 'Saving',
        }
        return dictionary[key] || key
      },
    }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: testState.showError,
    showSuccess: testState.showSuccess,
    showWarning: testState.showWarning,
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      batchUpdateConcurrency: testState.batchUpdateConcurrencyMock,
    },
  },
}))

const BaseDialogStub = {
  props: ['show', 'title'],
  template: `
    <div v-if="show">
      <div data-testid="dialog-title">{{ title }}</div>
      <slot />
      <slot name="footer" />
    </div>
  `,
}

const InputStub = {
  props: ['modelValue', 'label', 'placeholder', 'type'],
  emits: ['update:modelValue'],
  template: `
    <label>
      <span>{{ label }}</span>
      <input
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        @input="$emit('update:modelValue', $event.target.value)"
      />
    </label>
  `,
}

describe('BatchConcurrencyModal', () => {
  beforeEach(() => {
    testState.batchUpdateConcurrencyMock.mockReset()
    testState.showError.mockReset()
    testState.showSuccess.mockReset()
    testState.showWarning.mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('submits with an idempotency key and shows partial failure details', async () => {
    vi.stubGlobal('crypto', {
      randomUUID: () => 'fixed-uuid',
    })
    testState.batchUpdateConcurrencyMock.mockResolvedValue({
      matched: 2,
      success_count: 1,
      failed_count: 1,
      concurrency: 4,
      results: [
        { user_id: 1, email: 'ok@example.com', success: true },
        { user_id: 2, email: 'fail@example.com', success: false, error: 'quota locked' },
      ],
    })

    const wrapper = mount(BatchConcurrencyModal, {
      props: {
        show: true,
        matchedCount: 2,
        search: 'demo',
        role: 'user',
        status: 'active',
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Input: InputStub,
        },
      },
    })

    await wrapper.find('input').setValue('4')
    await wrapper.find('form').trigger('submit.prevent')

    expect(testState.batchUpdateConcurrencyMock).toHaveBeenCalledWith(
      {
        concurrency: 4,
        search: 'demo',
        role: 'user',
        status: 'active',
        group_name: undefined,
        attributes: undefined,
      },
      'users-batch-concurrency-fixed-uuid',
    )
    expect(testState.showWarning).toHaveBeenCalledWith('Partial 1/1', {
      details: ['fail@example.com: quota locked'],
    })
    expect(wrapper.emitted('success')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('blocks invalid concurrency before sending the request', async () => {
    const wrapper = mount(BatchConcurrencyModal, {
      props: {
        show: true,
        matchedCount: 1,
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Input: InputStub,
        },
      },
    })

    await wrapper.find('input').setValue('0')
    await wrapper.find('form').trigger('submit.prevent')

    expect(testState.batchUpdateConcurrencyMock).not.toHaveBeenCalled()
    expect(testState.showError).toHaveBeenCalledWith('Concurrency must be at least 1')
  })
})
