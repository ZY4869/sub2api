import { computed, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { AccountPlatform, CheckMixedChannelResponse } from '@/types'
import {
  buildMixedChannelWarningDetails,
  supportsMixedChannelCheck,
  type MixedChannelWarningDetails
} from '@/utils/accountFormShared'

interface MixedChannelRiskPayload {
  platform: AccountPlatform
  group_ids: number[]
  account_id?: number
}

interface OpenMixedChannelDialogOptions {
  response?: CheckMixedChannelResponse
  message?: string
  onConfirm: () => Promise<unknown> | unknown
}

interface UseAccountMixedChannelRiskOptions {
  currentPlatform: () => AccountPlatform | null | undefined
  buildCheckPayload: () => MixedChannelRiskPayload | null
  buildWarningText: (details: MixedChannelWarningDetails) => string
  fallbackMessage: () => string
  showError: (message: string) => void
}

/**
 * Keeps the mixed-channel confirmation flow in one place so create/edit
 * modals do not duplicate the risk-check API orchestration.
 */
export function useAccountMixedChannelRisk(options: UseAccountMixedChannelRiskOptions) {
  const showWarning = ref(false)
  const warningDetails = ref<MixedChannelWarningDetails | null>(null)
  const warningRawMessage = ref('')
  const warningAction = ref<(() => Promise<unknown> | unknown) | null>(null)
  const confirmed = ref(false)

  const requiresCheck = computed(() =>
    supportsMixedChannelCheck(options.currentPlatform())
  )

  const warningMessageText = computed(() => {
    if (warningDetails.value) {
      return options.buildWarningText(warningDetails.value)
    }
    return warningRawMessage.value
  })

  const clearDialog = () => {
    showWarning.value = false
    warningDetails.value = null
    warningRawMessage.value = ''
    warningAction.value = null
  }

  const openDialog = (dialogOptions: OpenMixedChannelDialogOptions) => {
    warningDetails.value = buildMixedChannelWarningDetails(dialogOptions.response)
    warningRawMessage.value =
      dialogOptions.message ||
      dialogOptions.response?.message ||
      options.fallbackMessage()
    warningAction.value = dialogOptions.onConfirm
    showWarning.value = true
  }

  const withConfirmFlag = <TPayload extends object>(
    payload: TPayload & { confirm_mixed_channel_risk?: boolean }
  ): TPayload & { confirm_mixed_channel_risk?: boolean } => {
    if (requiresCheck.value && confirmed.value) {
      return {
        ...payload,
        confirm_mixed_channel_risk: true
      } as TPayload & { confirm_mixed_channel_risk?: boolean }
    }

    const cloned = { ...payload }
    delete cloned.confirm_mixed_channel_risk
    return cloned as TPayload & { confirm_mixed_channel_risk?: boolean }
  }

  const ensureConfirmed = async (
    onConfirm: () => Promise<unknown> | unknown
  ): Promise<boolean> => {
    if (!requiresCheck.value) {
      return true
    }
    if (confirmed.value) {
      return true
    }

    const payload = options.buildCheckPayload()
    if (!payload) {
      return false
    }

    try {
      const result = await adminAPI.accounts.checkMixedChannelRisk(payload)
      if (!result.has_risk) {
        return true
      }

      openDialog({
        response: result,
        onConfirm
      })
      return false
    } catch (error: any) {
      options.showError(resolveErrorMessage(error, options.fallbackMessage()))
      return false
    }
  }

  const handleConfirm = async () => {
    const action = warningAction.value
    clearDialog()
    if (!action) {
      return
    }

    confirmed.value = true
    await action()
  }

  const handleCancel = () => {
    clearDialog()
  }

  const reset = () => {
    confirmed.value = false
    clearDialog()
  }

  return {
    showWarning,
    warningDetails,
    warningRawMessage,
    warningMessageText,
    confirmed,
    requiresCheck,
    openDialog,
    withConfirmFlag,
    ensureConfirmed,
    handleConfirm,
    handleCancel,
    reset
  }
}

function resolveErrorMessage(error: any, fallbackMessage: string): string {
  return (
    error?.response?.data?.message ||
    error?.response?.data?.detail ||
    error?.message ||
    fallbackMessage
  )
}
