<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.batchTest.title')"
    width="account-wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div
        class="rounded-xl border border-gray-200 bg-gradient-to-r from-gray-50 to-gray-100 p-3 text-sm text-gray-700 dark:border-dark-500 dark:from-dark-700 dark:to-dark-600 dark:text-gray-200"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.accounts.batchTest.targetLabel') }}
            </div>
            <p class="mt-1">
              {{ targetSummary }}
            </p>
          </div>
          <div
            v-if="isFilterTarget"
            class="inline-flex rounded-lg border border-gray-200 bg-white p-1 dark:border-dark-500 dark:bg-dark-700"
          >
            <button
              type="button"
              class="rounded-md px-3 py-1.5 text-xs font-semibold transition"
              :class="targetScope === 'current' ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-600'"
              :disabled="running"
              @click="targetScope = 'current'"
            >
              {{ t('admin.accounts.batchTest.scopeCurrent') }}
            </button>
            <button
              type="button"
              class="rounded-md px-3 py-1.5 text-xs font-semibold transition"
              :class="targetScope === 'all_platforms' ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-600'"
              :disabled="running"
              @click="targetScope = 'all_platforms'"
            >
              {{ t('admin.accounts.batchTest.scopeAllPlatforms') }}
            </button>
          </div>
        </div>
      </div>

      <div
        v-if="!supportsRealForwardForAll"
        class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
      >
        {{ t('admin.accounts.batchTest.realForwardUnsupportedHint') }}
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.testModeLabel') }}
        </label>
        <div class="grid gap-2 sm:grid-cols-2">
          <button
            type="button"
            class="rounded-xl border px-4 py-3 text-left transition-all"
            :class="testModeButtonClass('health_check')"
            :disabled="running"
            data-test="batch-test-mode-health_check"
            @click="selectedTestMode = 'health_check'"
          >
            <div class="text-sm font-semibold">
              {{ t('admin.accounts.testModes.healthCheck') }}
            </div>
            <p class="mt-1 text-xs leading-5 opacity-80">
              {{ t('admin.accounts.testModes.healthCheckHint') }}
            </p>
          </button>
          <button
            type="button"
            class="rounded-xl border px-4 py-3 text-left transition-all"
            :class="testModeButtonClass('real_forward')"
            :disabled="running || !supportsRealForwardForAll"
            data-test="batch-test-mode-real_forward"
            @click="selectedTestMode = 'real_forward'"
          >
            <div class="text-sm font-semibold">
              {{ t('admin.accounts.testModes.realForward') }}
            </div>
            <p class="mt-1 text-xs leading-5 opacity-80">
              {{ t('admin.accounts.testModes.realForwardHint') }}
            </p>
          </button>
        </div>
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.batchTest.modelStrategyLabel') }}
        </label>
        <div class="grid gap-2 sm:grid-cols-2">
          <button
            type="button"
            class="rounded-xl border px-4 py-3 text-left transition-all"
            :class="modelStrategyButtonClass('auto')"
            :disabled="running"
            data-test="batch-model-strategy-auto"
            @click="selectModelStrategy('auto')"
          >
            <div class="text-sm font-semibold">
              {{ t('admin.accounts.batchTest.modelStrategies.auto') }}
            </div>
            <p class="mt-1 text-xs leading-5 opacity-80">
              {{ t('admin.accounts.batchTest.modelStrategies.autoHint') }}
            </p>
          </button>
          <button
            type="button"
            class="rounded-xl border px-4 py-3 text-left transition-all"
            :class="modelStrategyButtonClass('specified')"
            :disabled="running"
            data-test="batch-model-strategy-specified"
            @click="selectModelStrategy('specified')"
          >
            <div class="text-sm font-semibold">
              {{ t('admin.accounts.batchTest.modelStrategies.specified') }}
            </div>
            <p class="mt-1 text-xs leading-5 opacity-80">
              {{ t('admin.accounts.batchTest.modelStrategies.specifiedHint') }}
            </p>
          </button>
        </div>
      </div>

      <AccountTestModelSelectionFields
        v-if="modelStrategy === 'specified'"
        v-model:model-input-mode="modelInputMode"
        v-model:selected-model-key="selectedModelKey"
        v-model:manual-model-id="manualModelId"
        v-model:manual-source-protocol="manualSourceProtocol"
        :available-models="availableModels"
        :loading-models="loadingModels"
        :disabled="running"
        :show-manual-source-protocol-field="showManualSourceProtocolField"
        :empty-hint="t('admin.accounts.batchTest.noSharedModels')"
      />

      <div class="grid gap-3 sm:grid-cols-[1fr_10rem]">
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.batchTest.executionModeLabel') }}
          </label>
          <div class="grid gap-2 sm:grid-cols-2">
            <button
              type="button"
              class="rounded-xl border px-4 py-3 text-left transition-all"
              :class="executionModeButtonClass('sequential')"
              :disabled="running"
              @click="executionMode = 'sequential'"
            >
              <div class="text-sm font-semibold">
                {{ t('admin.accounts.batchTest.executionModes.sequential') }}
              </div>
              <p class="mt-1 text-xs leading-5 opacity-80">
                {{ t('admin.accounts.batchTest.executionModes.sequentialHint') }}
              </p>
            </button>
            <button
              type="button"
              class="rounded-xl border px-4 py-3 text-left transition-all"
              :class="executionModeButtonClass('concurrent')"
              :disabled="running"
              @click="executionMode = 'concurrent'"
            >
              <div class="text-sm font-semibold">
                {{ t('admin.accounts.batchTest.executionModes.concurrent') }}
              </div>
              <p class="mt-1 text-xs leading-5 opacity-80">
                {{ t('admin.accounts.batchTest.executionModes.concurrentHint') }}
              </p>
            </button>
          </div>
        </div>
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.batchTest.concurrencyLabel') }}
          </label>
          <input
            v-model.number="concurrency"
            type="number"
            min="1"
            max="10"
            class="input"
            :disabled="running || executionMode === 'sequential'"
          />
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.batchTest.concurrencyHint') }}
          </p>
        </div>
      </div>

      <TextArea
        v-if="selectedTestMode === 'real_forward'"
        v-model="testPrompt"
        :label="t('admin.accounts.batchTest.promptLabel')"
        :placeholder="t('admin.accounts.batchTest.promptPlaceholder')"
        :hint="t('admin.accounts.batchTest.promptHint')"
        :disabled="running"
        rows="3"
      />

      <div v-if="results.length > 0" class="space-y-3">
        <div class="grid gap-2 sm:grid-cols-4">
          <div class="rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-200">
            {{ t('admin.accounts.batchTest.summary.success', { count: successCount }) }}
          </div>
          <div class="rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200">
            {{ t('admin.accounts.batchTest.summary.failed', { count: failedCount }) }}
          </div>
          <div class="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
            {{ t('admin.accounts.batchTest.summary.autoBlacklisted', { count: autoBlacklistedCount }) }}
          </div>
          <div class="rounded-xl border border-orange-200 bg-orange-50 px-3 py-2 text-sm text-orange-700 dark:border-orange-900/60 dark:bg-orange-950/30 dark:text-orange-200">
            {{ t('admin.accounts.batchTest.summary.needsReauth', { count: needsReauthCount }) }}
          </div>
        </div>

        <div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-600">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
            <thead class="bg-gray-50 dark:bg-dark-800/80">
              <tr>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.account') }}</th>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.platform') }}</th>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.model') }}</th>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.mode') }}</th>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.result') }}</th>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.latency') }}</th>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.lifecycle') }}</th>
                <th class="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">{{ t('admin.accounts.batchTest.columns.detail') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="item in results" :key="item.account_id">
                <td class="px-3 py-2 font-medium text-gray-900 dark:text-gray-100">
                  {{ item.account_name || accountNameByID[item.account_id] || item.account_id }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                  {{ formatPlatform(item) }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                  <span class="max-w-[16rem] truncate" :title="item.resolved_model_id || '-'">
                    {{ item.resolved_model_id || '-' }}
                  </span>
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                  {{ formatMode(item) }}
                </td>
                <td class="px-3 py-2">
                  <span class="inline-flex rounded-full px-2.5 py-1 text-xs font-semibold" :class="resultBadgeClass(item)">
                    {{ formatResult(item) }}
                  </span>
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                  {{ item.latency_ms ? `${item.latency_ms} ms` : '-' }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                  {{ formatLifecycle(item.current_lifecycle_state) }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                  <span class="max-w-[22rem] truncate" :title="formatDetail(item)">
                    {{ formatDetail(item) }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="running" @click="handleClose">
        {{ t('common.close') }}
      </button>
      <button
        type="button"
        class="btn btn-primary"
        :disabled="running || !canSubmit"
        data-test="batch-test-submit"
        @click="handleSubmit"
      >
        {{ running ? t('admin.accounts.testing') : t('admin.accounts.batchTest.run') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  AccountTestMode,
  BatchAccountTestResponse,
  BatchAccountTestRequestPayload,
  BatchAccountTestResult,
  BulkUpdateAccountsFilters
} from '@/api/admin/accounts'
import type { Account, AdminAccountModelOption } from '@/types'
import { useAppStore } from '@/stores/app'
import { isCredentialsNeedReauthMessage } from '@/components/account/accountStatusPresentation'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TextArea from '@/components/common/TextArea.vue'
import AccountTestModelSelectionFields from './AccountTestModelSelectionFields.vue'
import { findAccountTestModelByKey } from '@/utils/accountTestModelOptions'
import { normalizeGatewayAcceptedProtocol, resolveGatewayProtocolLabel } from '@/utils/accountProtocolGateway'
import {
  resolveCatalogTargetFromModel,
  resolveGatewayTestSelectedModelKey
} from '@/utils/accountGatewayTestDefaults'

type ModelStrategy = 'auto' | 'specified'
type ExecutionMode = 'sequential' | 'concurrent'
type TargetScope = 'current' | 'all_platforms'

const props = withDefaults(defineProps<{
  show: boolean
  accounts: Account[]
  filters?: BulkUpdateAccountsFilters | null
  filtersTotal?: number | null
  defaultTestMode?: AccountTestMode
  defaultModelStrategy?: ModelStrategy
  defaultExecutionMode?: ExecutionMode
  defaultConcurrency?: number
}>(), {
  defaultTestMode: 'health_check',
  defaultModelStrategy: 'auto',
  defaultExecutionMode: 'concurrent',
  defaultConcurrency: 4
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'completed'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const availableModels = ref<AdminAccountModelOption[]>([])
const results = ref<BatchAccountTestResult[]>([])
const responseSummary = ref<Pick<
  BatchAccountTestResponse,
  'success_count' | 'failed_count' | 'needs_reauth_count' | 'auto_blacklisted_count'
> | null>(null)
const loadingModels = ref(false)
const running = ref(false)
const modelStrategy = ref<ModelStrategy>('auto')
const modelInputMode = ref<'catalog' | 'manual'>('catalog')
const selectedModelKey = ref('')
const manualModelId = ref('')
const manualSourceProtocol = ref<'openai' | 'anthropic' | 'gemini' | ''>('')
const selectedTestMode = ref<AccountTestMode>('health_check')
const testPrompt = ref('')
const resultTestMode = ref<AccountTestMode>('health_check')
const executionMode = ref<ExecutionMode>('concurrent')
const concurrency = ref(4)
const targetScope = ref<TargetScope>('current')
const resolvingTargetTotal = ref(false)
const filterTargetTotal = ref<number | null>(null)

const isFilterTarget = computed(() => Boolean(props.filters))
const targetAccountIds = computed(() => props.accounts.map((account) => account.id))
const scopedFilters = computed<BulkUpdateAccountsFilters | null>(() => {
  if (!props.filters) {
    return null
  }
  const filters: BulkUpdateAccountsFilters = { ...props.filters }
  if (targetScope.value === 'all_platforms') {
    delete filters.platform
  }
  return Object.fromEntries(
    Object.entries(filters).filter(([, value]) => String(value || '').trim() !== '')
  ) as BulkUpdateAccountsFilters
})
const targetPayload = computed<Pick<BatchAccountTestRequestPayload, 'account_ids' | 'filters'>>(() =>
  scopedFilters.value ? { filters: scopedFilters.value } : { account_ids: [...targetAccountIds.value] }
)
const accountNameByID = computed<Record<number, string>>(() =>
  props.accounts.reduce<Record<number, string>>((acc, account) => {
    acc[account.id] = account.name
    return acc
  }, {})
)
const selectedModelOption = computed(() => findAccountTestModelByKey(availableModels.value, selectedModelKey.value))
const selectedSourceProtocol = computed(() =>
  modelInputMode.value === 'manual'
    ? normalizeGatewayAcceptedProtocol(manualSourceProtocol.value)
    : normalizeGatewayAcceptedProtocol(selectedModelOption.value?.source_protocol)
)
const showManualSourceProtocolField = computed(() =>
  isFilterTarget.value || props.accounts.some((account) => account.platform === 'protocol_gateway')
)
const supportsRealForwardForAll = computed(() =>
  scopedFilters.value
    ? scopedFilters.value.platform !== 'grok'
    : props.accounts.every((account) => account.platform !== 'grok')
)
const targetSummary = computed(() => {
  if (isFilterTarget.value) {
    if (resolvingTargetTotal.value) {
      return t('admin.accounts.batchTest.targetResolving')
    }
    const count = filterTargetTotal.value ?? props.filtersTotal ?? 0
    return targetScope.value === 'all_platforms'
      ? t('admin.accounts.batchTest.targetAllPlatforms', { count })
      : t('admin.accounts.batchTest.targetCurrentPreview', { count })
  }
  if (props.accounts.length === 1) {
    return t('admin.accounts.batchTest.targetSingle', { name: props.accounts[0]?.name || '-' })
  }
  return t('admin.accounts.batchTest.targetBatch', { count: props.accounts.length })
})
const normalizedSummaryCount = (value: unknown) => {
  const count = Number(value)
  return Number.isFinite(count) && count >= 0 ? count : null
}
const localSuccessCount = computed(() => results.value.filter((item) => item.status === 'success').length)
const localAutoBlacklistedCount = computed(() =>
  results.value.filter((item) =>
    item.blacklist_advice_decision === 'auto_blacklisted' || item.current_lifecycle_state === 'blacklisted'
  ).length
)
const localNeedsReauthCount = computed(() =>
  results.value.filter((item) => isNeedsReauthResult(item)).length
)
const successCount = computed(() =>
  normalizedSummaryCount(responseSummary.value?.success_count) ?? localSuccessCount.value
)
const autoBlacklistedCount = computed(() =>
  normalizedSummaryCount(responseSummary.value?.auto_blacklisted_count) ?? localAutoBlacklistedCount.value
)
const needsReauthCount = computed(() =>
  normalizedSummaryCount(responseSummary.value?.needs_reauth_count) ?? localNeedsReauthCount.value
)
const failedCount = computed(() =>
  normalizedSummaryCount(responseSummary.value?.failed_count) ?? (results.value.length - localSuccessCount.value)
)
const canSubmit = computed(() => {
  if (isFilterTarget.value) {
    if (!scopedFilters.value || resolvingTargetTotal.value) {
      return false
    }
    if ((filterTargetTotal.value ?? props.filtersTotal ?? 0) <= 0) {
      return false
    }
  } else if (targetAccountIds.value.length === 0) {
    return false
  }
  if (modelStrategy.value === 'auto') {
    return true
  }
  if (modelInputMode.value === 'manual') {
    return manualModelId.value.trim().length > 0
  }
  return Boolean(selectedModelOption.value?.id)
})

const resetForm = () => {
  modelStrategy.value = props.defaultModelStrategy
  modelInputMode.value = 'catalog'
  selectedModelKey.value = ''
  manualModelId.value = ''
  manualSourceProtocol.value = ''
  selectedTestMode.value = supportsRealForwardForAll.value ? props.defaultTestMode : 'health_check'
  resultTestMode.value = selectedTestMode.value
  testPrompt.value = selectedTestMode.value === 'real_forward' ? 'Output exactly: OK' : ''
  executionMode.value = props.defaultExecutionMode
  concurrency.value = clampConcurrency(props.defaultConcurrency)
  targetScope.value = 'current'
  filterTargetTotal.value = props.filtersTotal ?? null
  availableModels.value = []
  results.value = []
  responseSummary.value = null
}

const clampConcurrency = (value: number) => Math.min(10, Math.max(1, Number(value) || 1))

const resolveFilterTargetTotal = async () => {
  if (!props.show || !scopedFilters.value) {
    filterTargetTotal.value = null
    return
  }
  resolvingTargetTotal.value = true
  try {
    const response = await adminAPI.accounts.list(1, 1, scopedFilters.value)
    filterTargetTotal.value = Number(response.total || 0)
  } catch (error) {
    console.error('Failed to resolve batch test targets:', error)
    filterTargetTotal.value = 0
    appStore.showError(t('admin.accounts.batchTest.resolveTargetsFailed'))
  } finally {
    resolvingTargetTotal.value = false
  }
}

const loadAvailableModels = async () => {
  if (modelStrategy.value !== 'specified') {
    return
  }
  if (!isFilterTarget.value && targetAccountIds.value.length === 0) {
    return
  }
  if (isFilterTarget.value && !scopedFilters.value) {
    return
  }
  loadingModels.value = true
  selectedModelKey.value = ''
  try {
    const models = await adminAPI.accounts.getBatchTestModels(targetPayload.value)
    availableModels.value = models
    selectedModelKey.value = resolveGatewayTestSelectedModelKey(props.accounts, models)
  } catch (error) {
    console.error('Failed to load batch test models:', error)
    availableModels.value = []
    selectedModelKey.value = ''
    appStore.showError(t('admin.accounts.batchTest.loadModelsFailed'))
  } finally {
    loadingModels.value = false
  }
}

watch(
  () => [
    props.show,
    targetAccountIds.value.join(','),
    JSON.stringify(props.filters || {}),
    props.defaultTestMode,
    props.defaultModelStrategy,
    props.defaultExecutionMode,
    props.defaultConcurrency
  ],
  async ([visible]) => {
    if (!visible) {
      return
    }
    resetForm()
    await resolveFilterTargetTotal()
    if (modelStrategy.value === 'specified') {
      await loadAvailableModels()
    }
  },
  { immediate: true }
)

watch(targetScope, async () => {
  if (!props.show || !isFilterTarget.value) {
    return
  }
  results.value = []
  responseSummary.value = null
  availableModels.value = []
  await resolveFilterTargetTotal()
  if (modelStrategy.value === 'specified') {
    await loadAvailableModels()
  }
})

watch(supportsRealForwardForAll, (supported) => {
  if (!supported && selectedTestMode.value === 'real_forward') {
    selectedTestMode.value = 'health_check'
  }
})

watch(selectedTestMode, (mode) => {
  if (mode === 'real_forward' && !testPrompt.value.trim()) {
    testPrompt.value = 'Output exactly: OK'
    return
  }
  if (mode !== 'real_forward') {
    testPrompt.value = ''
  }
})

const selectModelStrategy = async (value: ModelStrategy) => {
  if (running.value) {
    return
  }
  modelStrategy.value = value
  results.value = []
  responseSummary.value = null
  if (value === 'specified' && availableModels.value.length === 0) {
    await loadAvailableModels()
  }
}

const handleClose = () => {
  if (running.value) {
    return
  }
  emit('close')
}

const handleSubmit = async () => {
  if (!canSubmit.value || running.value) {
    return
  }

  const payload: BatchAccountTestRequestPayload = {
    ...targetPayload.value,
    model_input_mode: modelStrategy.value === 'auto' ? 'auto' : modelInputMode.value,
    test_mode: selectedTestMode.value,
    execution_mode: executionMode.value,
    concurrency: executionMode.value === 'sequential' ? 1 : clampConcurrency(concurrency.value)
  }

  if (modelStrategy.value === 'specified') {
    const catalogTarget = resolveCatalogTargetFromModel(selectedModelOption.value)
    if (modelInputMode.value === 'manual') {
      payload.manual_model_id = manualModelId.value.trim()
    } else if (selectedModelOption.value?.id) {
      payload.model_id = selectedModelOption.value.id
      payload.target_provider = catalogTarget.targetProvider
      payload.target_model_id = catalogTarget.targetModelId
    }
    if (modelInputMode.value === 'catalog') {
      payload.source_protocol = catalogTarget.sourceProtocol || selectedSourceProtocol.value || undefined
    } else if (selectedSourceProtocol.value) {
      payload.source_protocol = selectedSourceProtocol.value
    }
  }

  if (selectedTestMode.value === 'real_forward') {
    payload.prompt = testPrompt.value.trim()
  }

  running.value = true
  try {
    resultTestMode.value = selectedTestMode.value
    responseSummary.value = null
    const response = await adminAPI.accounts.batchTestAccounts(payload)
    results.value = response.results || []
    responseSummary.value = {
      success_count: response.success_count,
      failed_count: response.failed_count,
      needs_reauth_count: response.needs_reauth_count,
      auto_blacklisted_count: response.auto_blacklisted_count
    }
    appStore.showSuccess(
      t('admin.accounts.batchTest.completed', {
        success: successCount.value,
        failed: failedCount.value
      })
    )
    emit('completed')
  } catch (error: any) {
    console.error('Failed to batch test accounts:', error)
    appStore.showError(error?.message || t('admin.accounts.batchTest.runFailed'))
  } finally {
    running.value = false
  }
}

const testModeButtonClass = (mode: AccountTestMode) => [
  selectedTestMode.value === mode
    ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200 dark:hover:border-primary-500/60',
  mode === 'real_forward' && !supportsRealForwardForAll.value ? 'cursor-not-allowed opacity-60' : ''
]

const modelStrategyButtonClass = (value: ModelStrategy) => [
  modelStrategy.value === value
    ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200 dark:hover:border-primary-500/60'
]

const executionModeButtonClass = (value: ExecutionMode) => [
  executionMode.value === value
    ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200 dark:hover:border-primary-500/60'
]

const formatPlatform = (item: BatchAccountTestResult) => {
  const value = item.resolved_platform || item.platform || ''
  const key = `admin.accounts.platforms.${value}`
  const translated = t(key)
  return translated === key ? value || '-' : translated
}

const formatMode = (item: BatchAccountTestResult) => {
  const modeLabel = t(
    resultTestMode.value === 'real_forward'
      ? 'admin.accounts.testModes.realForward'
      : 'admin.accounts.testModes.healthCheck'
  )
  const sourceProtocolLabel = resolveGatewayProtocolLabel(item.resolved_source_protocol)
  return sourceProtocolLabel ? `${modeLabel} · ${sourceProtocolLabel}` : modeLabel
}

const formatLifecycle = (value?: string) => {
  if (!value) {
    return '-'
  }
  const key = `admin.accounts.lifecycle.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

const formatResult = (item: BatchAccountTestResult) => {
  if (isNeedsReauthResult(item)) {
    return t('admin.accounts.status.needsReauth')
  }
  if (item.blacklist_advice_decision === 'auto_blacklisted' || item.current_lifecycle_state === 'blacklisted') {
    return t('admin.accounts.batchTest.resultLabels.autoBlacklisted')
  }
  if (item.status === 'success') {
    return t('admin.accounts.batchTest.resultLabels.healthy')
  }
  return t('admin.accounts.batchTest.resultLabels.abnormal')
}

const resultBadgeClass = (item: BatchAccountTestResult) => {
  if (isNeedsReauthResult(item)) {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  }
  if (item.blacklist_advice_decision === 'auto_blacklisted' || item.current_lifecycle_state === 'blacklisted') {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  }
  if (item.status === 'success') {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
  }
  return 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300'
}

const isNeedsReauthResult = (item: BatchAccountTestResult) =>
  item.needs_reauth === true ||
  item.lifecycle_reason_code === 'credentials_need_reauth' ||
  isCredentialsNeedReauthMessage(item.error_message || item.response_text || '')

const formatDetail = (item: BatchAccountTestResult) =>
  isNeedsReauthResult(item)
    ? [
        t('admin.accounts.status.needsReauth'),
        item.reauth_deadline_at
          ? t('admin.accounts.batchTest.reauthDeadline', { deadline: item.reauth_deadline_at })
          : ''
      ].filter(Boolean).join(' · ')
    : (item.error_message || item.response_text || '-')
</script>
