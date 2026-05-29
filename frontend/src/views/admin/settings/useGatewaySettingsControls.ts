import { computed, reactive, ref } from 'vue'
import { adminAPI } from '@/api'
import type { BetaPolicyRule } from '@/api/admin/settings'

type Translate = (key: string) => string
type MessageStore = {
  showError: (message: string) => void
  showSuccess: (message: string) => void
}

export function useGatewaySettingsControls(t: Translate, appStore: MessageStore) {
  const overloadCooldownLoading = ref(true)
  const overloadCooldownSaving = ref(false)
  const overloadCooldownForm = reactive({
    enabled: true,
    cooldown_minutes: 10
  })

  const streamTimeoutLoading = ref(true)
  const streamTimeoutSaving = ref(false)
  const streamTimeoutForm = reactive({
    enabled: true,
    action: 'temp_unsched' as 'temp_unsched' | 'error' | 'none',
    temp_unsched_minutes: 5,
    threshold_count: 3,
    threshold_window_minutes: 10
  })

  const rectifierLoading = ref(true)
  const rectifierSaving = ref(false)
  const rectifierForm = reactive({
    enabled: true,
    thinking_signature_enabled: true,
    thinking_budget_enabled: true
  })

  const betaPolicyLoading = ref(true)
  const betaPolicySaving = ref(false)
  const betaPolicyForm = reactive({
    rules: [] as BetaPolicyRule[]
  })

  const betaPolicyActionOptions = computed(() => [
    { value: 'pass', label: t('admin.settings.betaPolicy.actionPass') },
    { value: 'filter', label: t('admin.settings.betaPolicy.actionFilter') },
    { value: 'block', label: t('admin.settings.betaPolicy.actionBlock') }
  ])

  const betaPolicyScopeOptions = computed(() => [
    { value: 'all', label: t('admin.settings.betaPolicy.scopeAll') },
    { value: 'oauth', label: t('admin.settings.betaPolicy.scopeOAuth') },
    { value: 'apikey', label: t('admin.settings.betaPolicy.scopeAPIKey') },
    { value: 'bedrock', label: t('admin.settings.betaPolicy.scopeBedrock') }
  ])

  const betaDisplayNames: Record<string, string> = {
    'fast-mode-2026-02-01': 'Fast Mode',
    'context-1m-2025-08-07': 'Context 1M'
  }

  function getBetaDisplayName(token: string): string {
    return betaDisplayNames[token] || token
  }

  async function loadOverloadCooldownSettings() {
    overloadCooldownLoading.value = true
    try {
      const settings = await adminAPI.settings.getOverloadCooldownSettings()
      Object.assign(overloadCooldownForm, settings)
    } catch (error: any) {
      console.error('Failed to load overload cooldown settings:', error)
    } finally {
      overloadCooldownLoading.value = false
    }
  }

  async function saveOverloadCooldownSettings() {
    overloadCooldownSaving.value = true
    try {
      const updated = await adminAPI.settings.updateOverloadCooldownSettings({
        enabled: overloadCooldownForm.enabled,
        cooldown_minutes: overloadCooldownForm.cooldown_minutes
      })
      Object.assign(overloadCooldownForm, updated)
      appStore.showSuccess(t('admin.settings.overloadCooldown.saved'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.overloadCooldown.saveFailed') +
          ': ' +
          (error.message || t('common.unknownError'))
      )
    } finally {
      overloadCooldownSaving.value = false
    }
  }

  async function loadStreamTimeoutSettings() {
    streamTimeoutLoading.value = true
    try {
      const settings = await adminAPI.settings.getStreamTimeoutSettings()
      Object.assign(streamTimeoutForm, settings)
    } catch (error: any) {
      console.error('Failed to load stream timeout settings:', error)
    } finally {
      streamTimeoutLoading.value = false
    }
  }

  async function saveStreamTimeoutSettings() {
    streamTimeoutSaving.value = true
    try {
      const updated = await adminAPI.settings.updateStreamTimeoutSettings({
        enabled: streamTimeoutForm.enabled,
        action: streamTimeoutForm.action,
        temp_unsched_minutes: streamTimeoutForm.temp_unsched_minutes,
        threshold_count: streamTimeoutForm.threshold_count,
        threshold_window_minutes: streamTimeoutForm.threshold_window_minutes
      })
      Object.assign(streamTimeoutForm, updated)
      appStore.showSuccess(t('admin.settings.streamTimeout.saved'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.streamTimeout.saveFailed') + ': ' + (error.message || t('common.unknownError'))
      )
    } finally {
      streamTimeoutSaving.value = false
    }
  }

  async function loadRectifierSettings() {
    rectifierLoading.value = true
    try {
      const settings = await adminAPI.settings.getRectifierSettings()
      Object.assign(rectifierForm, settings)
    } catch (error: any) {
      console.error('Failed to load rectifier settings:', error)
    } finally {
      rectifierLoading.value = false
    }
  }

  async function saveRectifierSettings() {
    rectifierSaving.value = true
    try {
      const updated = await adminAPI.settings.updateRectifierSettings({
        enabled: rectifierForm.enabled,
        thinking_signature_enabled: rectifierForm.thinking_signature_enabled,
        thinking_budget_enabled: rectifierForm.thinking_budget_enabled
      })
      Object.assign(rectifierForm, updated)
      appStore.showSuccess(t('admin.settings.rectifier.saved'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.rectifier.saveFailed') + ': ' + (error.message || t('common.unknownError'))
      )
    } finally {
      rectifierSaving.value = false
    }
  }

  async function loadBetaPolicySettings() {
    betaPolicyLoading.value = true
    try {
      const settings = await adminAPI.settings.getBetaPolicySettings()
      betaPolicyForm.rules = settings.rules
    } catch (error: any) {
      console.error('Failed to load beta policy settings:', error)
    } finally {
      betaPolicyLoading.value = false
    }
  }

  async function saveBetaPolicySettings() {
    betaPolicySaving.value = true
    try {
      const updated = await adminAPI.settings.updateBetaPolicySettings({
        rules: betaPolicyForm.rules
      })
      betaPolicyForm.rules = updated.rules
      appStore.showSuccess(t('admin.settings.betaPolicy.saved'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.betaPolicy.saveFailed') + ': ' + (error.message || t('common.unknownError'))
      )
    } finally {
      betaPolicySaving.value = false
    }
  }

  return {
    overloadCooldownLoading,
    overloadCooldownSaving,
    overloadCooldownForm,
    streamTimeoutLoading,
    streamTimeoutSaving,
    streamTimeoutForm,
    rectifierLoading,
    rectifierSaving,
    rectifierForm,
    betaPolicyLoading,
    betaPolicySaving,
    betaPolicyForm,
    betaPolicyActionOptions,
    betaPolicyScopeOptions,
    getBetaDisplayName,
    loadOverloadCooldownSettings,
    saveOverloadCooldownSettings,
    loadStreamTimeoutSettings,
    saveStreamTimeoutSettings,
    loadRectifierSettings,
    saveRectifierSettings,
    loadBetaPolicySettings,
    saveBetaPolicySettings
  }
}
