import { computed, ref, type ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import type { Account, AdminAccountModelOption } from '@/types'
import type { ChannelMonitorFormState, MonitorModelSelectOption } from './channelMonitorFormTypes'

export function useChannelMonitorAccountModels(
  form: ChannelMonitorFormState,
  isAccountMode: ComputedRef<boolean>
) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const accounts = ref<Account[]>([])
  const accountSearch = ref('')
  const loadingAccounts = ref(false)
  const availableModels = ref<AdminAccountModelOption[]>([])
  const loadingModels = ref(false)
  const additionalModels = ref<string[]>([])
  const additionalModelToAdd = ref<string | number | boolean | null>(null)

  const filteredAccounts = computed(() => {
    const query = accountSearch.value.trim().toLowerCase()
    const list = Array.isArray(accounts.value) ? accounts.value : []
    if (!query) return list
    return list.filter((account) =>
      `${account.name} ${account.platform} ${account.status}`.toLowerCase().includes(query)
    )
  })

  const modelOptions = computed<MonitorModelSelectOption[]>(() =>
    availableModels.value.map((model) => ({
      value: model.id,
      label: model.display_name || model.id,
      provider: model.provider,
      provider_label: model.provider_label,
      display_name: model.display_name,
      source_protocol: model.source_protocol,
      disabled: model.availability_state === 'unavailable' || model.status === 'deprecated'
    }))
  )

  const additionalModelOptions = computed(() =>
    modelOptions.value.filter((model) =>
      model.value !== form.primary_model_id && !additionalModels.value.includes(String(model.value))
    )
  )

  const modelSelectHint = computed(() => {
    if (form.account_ids.length === 0) return t('admin.channelMonitors.fields.selectAccountsFirst')
    if (modelOptions.value.length === 0) return t('admin.channelMonitors.fields.noSharedModels')
    return t('admin.channelMonitors.fields.sharedModelsHint', { count: modelOptions.value.length })
  })

  async function loadAccounts() {
    if (loadingAccounts.value || accounts.value.length > 0) return
    loadingAccounts.value = true
    try {
      const result = await adminAPI.accounts.list(1, 200, { status: 'active', lite: 'true' })
      accounts.value = Array.isArray(result?.items) ? result.items : []
    } catch {
      appStore.showError(t('admin.channelMonitors.messages.loadAccountsFailed'))
    } finally {
      loadingAccounts.value = false
    }
  }

  async function loadSharedModels() {
    if (!isAccountMode.value || form.account_ids.length === 0) {
      availableModels.value = []
      return
    }
    loadingModels.value = true
    try {
      availableModels.value = await adminAPI.accounts.getBatchTestModels({ account_ids: form.account_ids })
      if (!availableModels.value.some((model) => model.id === form.primary_model_id)) {
        form.primary_model_id = availableModels.value[0]?.id || ''
      }
      additionalModels.value = additionalModels.value.filter((id) =>
        availableModels.value.some((model) => model.id === id)
      )
    } catch {
      appStore.showError(t('admin.channelMonitors.messages.loadModelsFailed'))
    } finally {
      loadingModels.value = false
    }
  }

  function toggleAccount(id: number) {
    if (form.account_ids.includes(id)) {
      form.account_ids = form.account_ids.filter((item) => item !== id)
      return
    }
    form.account_ids = [...form.account_ids, id]
  }

  function addAdditionalModel(value: string | number | boolean | null) {
    const model = String(value || '').trim()
    if (model && model !== form.primary_model_id && !additionalModels.value.includes(model)) {
      additionalModels.value = [...additionalModels.value, model]
    }
    additionalModelToAdd.value = null
  }

  function removeAdditionalModel(model: string) {
    additionalModels.value = additionalModels.value.filter((item) => item !== model)
  }

  return {
    additionalModelOptions,
    additionalModelToAdd,
    additionalModels,
    addAdditionalModel,
    accountSearch,
    availableModels,
    filteredAccounts,
    loadAccounts,
    loadSharedModels,
    loadingAccounts,
    loadingModels,
    modelOptions,
    modelSelectHint,
    removeAdditionalModel,
    toggleAccount
  }
}
