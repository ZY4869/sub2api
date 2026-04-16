<template>
  <section class="space-y-3 rounded-2xl border border-gray-200 bg-white/80 p-4 dark:border-dark-600 dark:bg-dark-700/60">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="space-y-1">
        <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ title || t('admin.accounts.probeFinalize.manualModelsTitle') }}
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ hint || t(allowSourceProtocol ? 'admin.accounts.probeFinalize.manualModelsHintGateway' : 'admin.accounts.probeFinalize.manualModelsHint') }}
        </p>
      </div>
      <button
        type="button"
        class="inline-flex items-center justify-center rounded-xl border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
        @click="appendRow"
      >
        {{ t('admin.accounts.probeFinalize.addManualModel') }}
      </button>
    </div>

    <div v-if="rows.length === 0" class="rounded-xl border border-dashed border-gray-300 px-3 py-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400">
      {{ t('admin.accounts.probeFinalize.manualModelsEmpty') }}
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="(row, index) in rows"
        :key="`${index}-${row.model_id}-${row.source_protocol || 'default'}`"
        class="space-y-3 rounded-2xl border border-gray-200 bg-gray-50/80 p-3 dark:border-dark-500 dark:bg-dark-800/60"
      >
        <div :class="gridClass">
          <label class="space-y-1">
            <span class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.probeFinalize.manualModelId') }}
            </span>
            <input
              :value="row.model_id"
              type="text"
              class="input h-10"
              :placeholder="t('admin.accounts.probeFinalize.manualModelIdPlaceholder')"
              @input="updateRow(index, 'model_id', ($event.target as HTMLInputElement).value)"
            />
          </label>
          <label class="space-y-1">
            <span class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.probeFinalize.manualRequestAlias') }}
            </span>
            <input
              :value="row.request_alias || ''"
              type="text"
              class="input h-10"
              :placeholder="row.model_id || t('admin.accounts.probeFinalize.manualRequestAliasPlaceholder')"
              @input="updateRow(index, 'request_alias', ($event.target as HTMLInputElement).value)"
            />
          </label>
          <label class="space-y-1">
            <span class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.probeFinalize.manualProvider') }}
            </span>
            <select
              :value="row.provider || ''"
              class="input h-10"
              @change="updateRow(index, 'provider', ($event.target as HTMLSelectElement).value)"
            >
              <option value="">{{ t('admin.accounts.probeFinalize.manualProviderAuto') }}</option>
              <option
                v-for="option in providerOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
          </label>
          <label v-if="allowSourceProtocol" class="space-y-1">
            <span class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.probeFinalize.manualSourceProtocol') }}
            </span>
            <select
              :value="row.source_protocol || ''"
              class="input h-10"
              @change="updateRow(index, 'source_protocol', ($event.target as HTMLSelectElement).value)"
            >
              <option value="">{{ t('admin.accounts.probeFinalize.manualSourceProtocolAuto') }}</option>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="gemini">Gemini</option>
            </select>
          </label>
        </div>

        <div class="flex justify-end">
          <button
            type="button"
            class="rounded-lg border border-rose-200 px-3 py-1.5 text-xs font-medium text-rose-700 transition hover:border-rose-300 hover:text-rose-800 dark:border-rose-900/60 dark:text-rose-200 dark:hover:border-rose-700"
            @click="removeRow(index)"
          >
            {{ t('admin.accounts.probeFinalize.removeManualModel') }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountManualModel } from '@/api/admin/accounts'
import { normalizeAccountManualModels } from '@/utils/accountProbeDraft'
import { formatProviderLabel, listKnownProviders } from '@/utils/providerLabels'

const props = withDefaults(defineProps<{
  allowSourceProtocol?: boolean
  title?: string
  hint?: string
}>(), {
  allowSourceProtocol: false,
  title: '',
  hint: ''
})

const rows = defineModel<AccountManualModel[]>('rows', { required: true })
const { t } = useI18n()

const normalizedRows = computed(() =>
  normalizeAccountManualModels(rows.value, props.allowSourceProtocol)
)
const providerOptions = computed(() =>
  listKnownProviders(rows.value.map((row) => row.provider)).map((value) => ({
    value,
    label: formatProviderLabel(value)
  }))
)
const gridClass = computed(() =>
  props.allowSourceProtocol
    ? 'grid gap-3 md:grid-cols-2 xl:grid-cols-4'
    : 'grid gap-3 md:grid-cols-2 xl:grid-cols-3'
)

function emitRows(nextRows: AccountManualModel[]) {
  rows.value = normalizeAccountManualModels(nextRows, props.allowSourceProtocol)
}

function appendRow() {
  rows.value = [
    ...normalizedRows.value,
    {
      model_id: '',
      request_alias: undefined,
      provider: undefined,
      source_protocol: undefined
    }
  ]
}

function updateRow(
  index: number,
  key: keyof AccountManualModel,
  value: string
) {
  const nextRows = [...rows.value]
  const currentRow = nextRows[index] || { model_id: '' }
  nextRows[index] = {
    ...currentRow,
    [key]: value
  }
  rows.value = nextRows
}

function removeRow(index: number) {
  const nextRows = [...rows.value]
  nextRows.splice(index, 1)
  emitRows(nextRows)
}
</script>
