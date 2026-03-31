import { ref, type ComputedRef, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type {
  Account,
  AccountPlatform,
  AccountType,
  CreateAccountRequest,
  GatewayProtocol
} from '@/types'
import { buildAccountModelScopeExtra } from '@/utils/accountModelScope'
import {
  isProtocolGatewayPlatform,
  resolveEffectiveAccountPlatform
} from '@/utils/accountProtocolGateway'
import type { ModelMapping } from '@/utils/accountFormShared'
import {
  applyGoogleBatchArchiveExtra,
  resolveGoogleBatchArchiveTargetKind,
  type GoogleBatchArchiveBillingMode
} from '@/utils/accountGoogleBatchArchive'

interface UseCreateAccountSubmitOptions {
  withConfirmFlag: <TPayload extends object>(payload: TPayload) => TPayload
  ensureMixedChannelConfirmed: (onConfirm: () => Promise<unknown> | unknown) => Promise<boolean>
  requiresMixedChannelCheck: Ref<boolean>
  openMixedChannelDialog: (options: {
    message?: string
    onConfirm: () => Promise<unknown> | unknown
  }) => void
  isOpenAIModelRestrictionDisabled: ComputedRef<boolean>
  modelRestrictionMode: Ref<'whitelist' | 'mapping'>
  allowedModels: Ref<string[]>
  modelMappings: Ref<ModelMapping[]>
  antigravityModelMappings: Ref<ModelMapping[]>
  applyTempUnschedConfig: (credentials: Record<string, unknown>) => boolean
  form: {
    name: string
    notes: string
    proxy_id: number | null
    concurrency: number
    load_factor: number | null
    priority: number
    rate_multiplier: number
    group_ids: number[]
    expires_at: number | null
  }
  autoPauseOnExpired: Ref<boolean>
  editQuotaLimit: Ref<number | null>
  editQuotaDailyLimit: Ref<number | null>
  editQuotaWeeklyLimit: Ref<number | null>
  editQuotaDailyResetMode: Ref<'rolling' | 'fixed' | null>
  editQuotaDailyResetHour: Ref<number | null>
  editQuotaWeeklyResetMode: Ref<'rolling' | 'fixed' | null>
  editQuotaWeeklyResetDay: Ref<number | null>
  editQuotaWeeklyResetHour: Ref<number | null>
  editQuotaResetTimezone: Ref<string | null>
  batchArchiveEnabled: Ref<boolean>
  batchArchiveAutoPrefetchEnabled: Ref<boolean>
  batchArchiveRetentionDays: Ref<number>
  batchArchiveBillingMode: Ref<GoogleBatchArchiveBillingMode>
  batchArchiveDownloadPriceUSD: Ref<number>
  allowVertexBatchOverflow: Ref<boolean>
  acceptAIStudioBatchOverflow: Ref<boolean>
  afterCreateImportModels: (accounts: Account[]) => Promise<void>
  emitCreated: () => void
  onClose: () => void
}

export function useCreateAccountSubmit(options: UseCreateAccountSubmitOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()
  const submitting = ref(false)

  const buildPayloadWithModelScope = (payload: CreateAccountRequest): CreateAccountRequest => {
    const runtimePlatform = resolveEffectiveAccountPlatform(
      payload.platform,
      payload.gateway_protocol
    )
    return {
      ...payload,
      extra: buildAccountModelScopeExtra(payload.extra as Record<string, unknown> | undefined, {
        platform: runtimePlatform,
        enabled:
          runtimePlatform === 'antigravity'
            ? true
            : !(runtimePlatform === 'openai' && options.isOpenAIModelRestrictionDisabled.value),
        mode: runtimePlatform === 'antigravity' ? 'mapping' : options.modelRestrictionMode.value,
        allowedModels: options.allowedModels.value,
        modelMappings:
          runtimePlatform === 'antigravity'
            ? options.antigravityModelMappings.value
            : options.modelMappings.value
      })
    }
  }

  const submitCreateAccount = async (payload: CreateAccountRequest): Promise<Account | null> => {
    submitting.value = true
    try {
      const payloadWithScope = buildPayloadWithModelScope(payload)
      const createdAccount = await adminAPI.accounts.create(
        options.withConfirmFlag(payloadWithScope)
      )
      appStore.showSuccess(t('admin.accounts.accountCreated'))
      await options.afterCreateImportModels([createdAccount])
      options.emitCreated()
      options.onClose()
      return createdAccount
    } catch (error: any) {
      if (
        error?.status === 409 &&
        error?.error === 'mixed_channel_warning' &&
        options.requiresMixedChannelCheck.value
      ) {
        options.openMixedChannelDialog({
          message: error?.message,
          onConfirm: async () => submitCreateAccount(payload)
        })
        return null
      }

      appStore.showError(error?.message || t('admin.accounts.failedToCreate'))
      return null
    } finally {
      submitting.value = false
    }
  }

  const doCreateAccount = async (payload: CreateAccountRequest): Promise<Account | null> => {
    const canContinue = await options.ensureMixedChannelConfirmed(async () => {
      await submitCreateAccount(payload)
    })
    if (!canContinue) {
      return null
    }
    return submitCreateAccount(payload)
  }

  const applyQuotaLimits = (base?: Record<string, unknown>): Record<string, unknown> | undefined => {
    const extra: Record<string, unknown> = { ...(base || {}) }
    if (options.editQuotaLimit.value != null && options.editQuotaLimit.value > 0) {
      extra.quota_limit = options.editQuotaLimit.value
    }
    if (options.editQuotaDailyLimit.value != null && options.editQuotaDailyLimit.value > 0) {
      extra.quota_daily_limit = options.editQuotaDailyLimit.value
    }
    if (options.editQuotaWeeklyLimit.value != null && options.editQuotaWeeklyLimit.value > 0) {
      extra.quota_weekly_limit = options.editQuotaWeeklyLimit.value
    }

    if (options.editQuotaDailyResetMode.value != null) {
      extra.quota_daily_reset_mode = options.editQuotaDailyResetMode.value
    }
    if (options.editQuotaDailyResetHour.value != null) {
      extra.quota_daily_reset_hour = options.editQuotaDailyResetHour.value
    }
    if (options.editQuotaWeeklyResetMode.value != null) {
      extra.quota_weekly_reset_mode = options.editQuotaWeeklyResetMode.value
    }
    if (options.editQuotaWeeklyResetDay.value != null) {
      extra.quota_weekly_reset_day = options.editQuotaWeeklyResetDay.value
    }
    if (options.editQuotaWeeklyResetHour.value != null) {
      extra.quota_weekly_reset_hour = options.editQuotaWeeklyResetHour.value
    }
    if (options.editQuotaResetTimezone.value != null) {
      extra.quota_reset_timezone = options.editQuotaResetTimezone.value
    }

    return Object.keys(extra).length > 0 ? extra : undefined
  }

  const createAccountAndFinish = async (
    platform: AccountPlatform,
    type: AccountType,
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>,
    gatewayProtocol?: GatewayProtocol
  ): Promise<Account | null> => {
    if (!options.applyTempUnschedConfig(credentials)) {
      return null
    }

    const extraWithGateway = (() => {
      const nextExtra = { ...(extra || {}) }
      if (!isProtocolGatewayPlatform(platform)) {
        delete nextExtra.gateway_protocol
      } else if (gatewayProtocol) {
        nextExtra.gateway_protocol = gatewayProtocol
      }
      return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
    })()

    const finalExtra = applyGoogleBatchArchiveExtra(
      applyQuotaLimits(extraWithGateway),
      resolveGoogleBatchArchiveTargetKind(platform, type, credentials),
      {
        enabled: options.batchArchiveEnabled.value,
        autoPrefetchEnabled: options.batchArchiveAutoPrefetchEnabled.value,
        retentionDays: options.batchArchiveRetentionDays.value,
        billingMode: options.batchArchiveBillingMode.value,
        downloadPriceUSD: options.batchArchiveDownloadPriceUSD.value,
        allowVertexBatchOverflow: options.allowVertexBatchOverflow.value,
        acceptAIStudioBatchOverflow: options.acceptAIStudioBatchOverflow.value
      }
    )

    return doCreateAccount({
      name: options.form.name,
      notes: options.form.notes,
      platform,
      gateway_protocol: isProtocolGatewayPlatform(platform) ? gatewayProtocol : undefined,
      type,
      credentials,
      extra: finalExtra,
      proxy_id: options.form.proxy_id,
      concurrency: options.form.concurrency,
      load_factor: options.form.load_factor ?? undefined,
      priority: options.form.priority,
      rate_multiplier: options.form.rate_multiplier,
      group_ids: options.form.group_ids,
      expires_at: options.form.expires_at,
      auto_pause_on_expired: options.autoPauseOnExpired.value
    })
  }

  return {
    submitting,
    submitCreateAccount,
    doCreateAccount,
    createAccountAndFinish
  }
}
