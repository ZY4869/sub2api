import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { BulkUpdateAccountsTarget } from '@/api/admin/accounts'
import type { UpdateAccountRequest } from '@/types'

interface UseBulkEditAccountSubmitOptions {
  target: () => BulkUpdateAccountsTarget | number[]
  withConfirmFlag: <TPayload extends object>(payload: TPayload) => TPayload
  onMixedChannelWarning: (options: { message: string; retry: () => Promise<void> }) => void
  onUpdated: () => void
}

function getBulkUpdateCounts(payload: { success?: number; failed?: number }) {
  return {
    success: typeof payload.success === 'number' ? payload.success : 0,
    failed: typeof payload.failed === 'number' ? payload.failed : 0
  }
}

export function useBulkEditAccountSubmit(options: UseBulkEditAccountSubmitOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()
  const submitting = ref(false)

  const notifyUpdateResult = (success: number, failed: number) => {
    if (success > 0 && failed === 0) {
      appStore.showSuccess(t('admin.accounts.bulkEdit.success', { count: success }))
      return
    }
    if (success > 0) {
      appStore.showError(t('admin.accounts.bulkEdit.partialSuccess', { success, failed }))
      return
    }
    appStore.showError(t('admin.accounts.bulkEdit.failed'))
  }

  const submitBulkUpdate = async (
    baseUpdates: Partial<UpdateAccountRequest>,
    forceConfirmMixedChannelRisk = false
  ) => {
    submitting.value = true

    try {
      const payload = options.withConfirmFlag(baseUpdates) as Record<string, unknown>
      if (forceConfirmMixedChannelRisk) {
        payload.confirm_mixed_channel_risk = true
      }

      const target = options.target()
      const result = Array.isArray(target)
        ? await adminAPI.accounts.bulkUpdate(target, payload)
        : await adminAPI.accounts.bulkUpdate(target, payload)
      const { success, failed } = getBulkUpdateCounts(result)
      notifyUpdateResult(success, failed)
      if (success > 0) {
        options.onUpdated()
      }
    } catch (error: any) {
      if (error?.status === 409 && error?.error === 'mixed_channel_warning') {
        options.onMixedChannelWarning({
          message: error?.message || t('admin.accounts.bulkEdit.failed'),
          retry: async () => submitBulkUpdate(baseUpdates, true)
        })
        return
      }

      appStore.showError(error?.message || t('admin.accounts.bulkEdit.failed'))
      console.error('Error bulk updating accounts:', error)
    } finally {
      submitting.value = false
    }
  }

  return {
    submitting,
    submitBulkUpdate
  }
}
