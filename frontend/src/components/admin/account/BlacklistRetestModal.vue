<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.blacklist.retestConfigTitle')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div
        class="rounded-xl border border-gray-200 bg-gradient-to-r from-gray-50 to-gray-100 p-3 text-sm text-gray-700 dark:border-dark-500 dark:from-dark-700 dark:to-dark-600 dark:text-gray-200"
      >
        <div class="font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.blacklist.retestTargetLabel') }}
        </div>
        <p class="mt-1">
          {{ targetSummary }}
        </p>
      </div>

      <div class="rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-700 dark:border-sky-700 dark:bg-sky-900/20 dark:text-sky-300">
        {{ t('admin.accounts.blacklist.retestConfigDescription') }}
      </div>

      <AccountTestModelSelectionFields
        v-model:model-input-mode="modelInputMode"
        v-model:selected-model-key="selectedModelKey"
        v-model:manual-model-id="manualModelId"
        v-model:manual-source-protocol="manualSourceProtocol"
        :available-models="availableModels"
        :loading-models="loadingModels"
        :disabled="submitting"
        :show-manual-source-protocol-field="showManualSourceProtocolField"
        :empty-hint="t('admin.accounts.blacklist.retestModelEmptyHint')"
      />
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="submitting" @click="handleClose">
        {{ t('common.cancel') }}
      </button>
      <button
        type="button"
        data-test="blacklist-retest-confirm"
        class="btn btn-primary"
        :disabled="submitting"
        @click="handleConfirm"
      >
        {{ t('admin.accounts.blacklist.retestRun') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { BlacklistRetestRequestPayload } from '@/api/admin/accounts'
import type { Account, AdminAccountModelOption } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AccountTestModelSelectionFields from './AccountTestModelSelectionFields.vue'
import {
  findAccountTestModelByKey
} from '@/utils/accountTestModelOptions'
import { normalizeGatewayAcceptedProtocol } from '@/utils/accountProtocolGateway'
import {
  resolveCatalogTargetFromModel,
  resolveGatewayTestSelectedModelKey
} from '@/utils/accountGatewayTestDefaults'

const props = withDefaults(defineProps<{
  show: boolean
  accounts: Account[]
  submitting?: boolean
}>(), {
  submitting: false
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'confirm', payload: BlacklistRetestRequestPayload): void
}>()

const { t } = useI18n()

const availableModels = ref<AdminAccountModelOption[]>([])
const loadingModels = ref(false)
const modelInputMode = ref<'catalog' | 'manual'>('catalog')
const selectedModelKey = ref('')
const manualModelId = ref('')
const manualSourceProtocol = ref<'openai' | 'anthropic' | 'gemini' | ''>('')

const targetAccountIds = computed(() => props.accounts.map((account) => account.id))
const selectedModelOption = computed(() => findAccountTestModelByKey(availableModels.value, selectedModelKey.value))
const effectiveSourceProtocol = computed(() =>
  modelInputMode.value === 'manual'
    ? normalizeGatewayAcceptedProtocol(manualSourceProtocol.value)
    : normalizeGatewayAcceptedProtocol(selectedModelOption.value?.source_protocol)
)
const showManualSourceProtocolField = computed(() =>
  props.accounts.some((account) => account.platform === 'protocol_gateway')
)
const targetSummary = computed(() => {
  if (props.accounts.length === 1) {
    return t('admin.accounts.blacklist.retestTargetSingle', { name: props.accounts[0]?.name || '-' })
  }
  return t('admin.accounts.blacklist.retestTargetBatch', { count: props.accounts.length })
})

const resetSelection = () => {
  modelInputMode.value = 'catalog'
  selectedModelKey.value = ''
  manualModelId.value = ''
  manualSourceProtocol.value = ''
}

const loadAvailableModels = async () => {
  if (targetAccountIds.value.length === 0) {
    availableModels.value = []
    selectedModelKey.value = ''
    return
  }

  loadingModels.value = true
  selectedModelKey.value = ''
  try {
    const models = await adminAPI.accounts.getBlacklistRetestModels(targetAccountIds.value)
    availableModels.value = models
    selectedModelKey.value = resolveGatewayTestSelectedModelKey(props.accounts, models)
  } catch (error) {
    console.error('Failed to load blacklist retest models:', error)
    availableModels.value = []
    selectedModelKey.value = ''
  } finally {
    loadingModels.value = false
  }
}

watch(
  () => [props.show, targetAccountIds.value.join(',')],
  async ([visible]) => {
    if (!visible) {
      return
    }
    resetSelection()
    await loadAvailableModels()
  },
  { immediate: true }
)

const handleClose = () => {
  if (props.submitting) {
    return
  }
  emit('close')
}

const handleConfirm = () => {
  const payload: BlacklistRetestRequestPayload = {
    account_ids: [...targetAccountIds.value],
    model_input_mode: modelInputMode.value
  }

  if (modelInputMode.value === 'manual') {
    if (manualModelId.value.trim()) {
      payload.manual_model_id = manualModelId.value.trim()
    }
  } else if (selectedModelOption.value?.id) {
    const catalogTarget = resolveCatalogTargetFromModel(selectedModelOption.value)
    payload.model_id = selectedModelOption.value.id
    payload.target_provider = catalogTarget.targetProvider
    payload.target_model_id = catalogTarget.targetModelId
    payload.source_protocol = catalogTarget.sourceProtocol || undefined
  }

  if (modelInputMode.value === 'manual' && effectiveSourceProtocol.value) {
    payload.source_protocol = effectiveSourceProtocol.value
  }

  emit('confirm', payload)
}
</script>
