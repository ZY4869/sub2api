import { ref, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { CodexSessionImportResult } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'

type UseCodexSessionImportModalOptions = {
  show: Ref<boolean>
  onClose: () => void
  onImported: (result: CodexSessionImportResult) => void
}

const optionalNumber = (value: string) => {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : undefined
}

export function useCodexSessionImportModal(options: UseCodexSessionImportModalOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()
  const content = ref('')
  const name = ref('')
  const proxyId = ref('')
  const concurrency = ref('')
  const priority = ref('')
  const rateMultiplier = ref('')
  const loadFactor = ref('')
  const expiresAtLocal = ref('')
  const selectedGroupIds = ref<number[]>([])
  const updateExisting = ref(true)
  const autoPauseOnExpired = ref(true)
  const skipDefaultGroupBind = ref(false)
  const confirmMixedChannelRisk = ref(false)
  const submitting = ref(false)
  const result = ref<CodexSessionImportResult | null>(null)

  const reset = () => {
    content.value = ''
    name.value = ''
    proxyId.value = ''
    concurrency.value = ''
    priority.value = ''
    rateMultiplier.value = ''
    loadFactor.value = ''
    expiresAtLocal.value = ''
    selectedGroupIds.value = []
    updateExisting.value = true
    autoPauseOnExpired.value = true
    skipDefaultGroupBind.value = false
    confirmMixedChannelRisk.value = false
    result.value = null
  }

  watch(options.show, (open) => {
    if (open) reset()
  })

  const optionalUnixTime = () => {
    if (!expiresAtLocal.value) return undefined
    const parsed = new Date(expiresAtLocal.value).getTime()
    return Number.isFinite(parsed) ? Math.floor(parsed / 1000) : undefined
  }

  const handleClose = () => {
    if (!submitting.value) options.onClose()
  }

  const handleSubmit = async () => {
    if (!content.value.trim()) return
    submitting.value = true
    try {
      const imported = await adminAPI.accounts.importCodexSession({
        content: content.value,
        name: name.value.trim() || undefined,
        proxy_id: optionalNumber(proxyId.value) ?? null,
        group_ids: selectedGroupIds.value,
        concurrency: optionalNumber(concurrency.value),
        priority: optionalNumber(priority.value),
        rate_multiplier: optionalNumber(rateMultiplier.value),
        load_factor: optionalNumber(loadFactor.value),
        expires_at: optionalUnixTime(),
        auto_pause_on_expired: autoPauseOnExpired.value,
        update_existing: updateExisting.value,
        skip_default_group_bind: skipDefaultGroupBind.value,
        confirm_mixed_channel_risk: confirmMixedChannelRisk.value
      })
      result.value = imported
      options.onImported(imported)
      const key = imported.failed > 0 ? 'completedWithErrors' : 'success'
      const message = t(`admin.accounts.codexImport.${key}`, { ...imported })
      imported.failed > 0 ? appStore.showError(message) : appStore.showSuccess(message)
    } catch (error: any) {
      appStore.showError(error?.message || t('admin.accounts.codexImport.failed'))
    } finally {
      submitting.value = false
    }
  }

  return {
    content,
    name,
    proxyId,
    concurrency,
    priority,
    rateMultiplier,
    loadFactor,
    expiresAtLocal,
    selectedGroupIds,
    updateExisting,
    autoPauseOnExpired,
    skipDefaultGroupBind,
    confirmMixedChannelRisk,
    submitting,
    result,
    handleClose,
    handleSubmit
  }
}
