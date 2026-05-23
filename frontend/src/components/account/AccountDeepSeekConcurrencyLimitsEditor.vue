<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  DEEPSEEK_V4_FLASH_MODEL_ID,
  DEEPSEEK_V4_PRO_MODEL_ID,
  type DeepSeekModelConcurrencyLimitDraft
} from '@/utils/deepseekAccount'

const limits = defineModel<DeepSeekModelConcurrencyLimitDraft>('limits', { required: true })

const { t } = useI18n()

const rows = computed(() => [
  {
    model: DEEPSEEK_V4_PRO_MODEL_ID,
    label: t('admin.accounts.deepseek.modelConcurrency.pro'),
    defaultLimit: 500
  },
  {
    model: DEEPSEEK_V4_FLASH_MODEL_ID,
    label: t('admin.accounts.deepseek.modelConcurrency.flash'),
    defaultLimit: 2500
  }
])

function getInputValue(model: string): string | number {
  return limits.value[model] ?? ''
}

function updateLimit(model: string, value: string) {
  const trimmed = value.trim()
  limits.value = {
    ...limits.value,
    [model]: trimmed === '' ? '' : Math.max(0, Math.floor(Number(trimmed) || 0))
  }
}
</script>

<template>
  <section
    class="rounded-lg border border-indigo-200 bg-indigo-50/60 p-4 dark:border-indigo-900/50 dark:bg-indigo-950/20"
    data-testid="deepseek-concurrency-editor"
  >
    <div class="mb-4">
      <h3 class="text-sm font-semibold text-indigo-950 dark:text-indigo-100">
        {{ t('admin.accounts.deepseek.modelConcurrency.title') }}
      </h3>
      <p class="mt-1 text-xs leading-5 text-indigo-800/80 dark:text-indigo-200/80">
        {{ t('admin.accounts.deepseek.modelConcurrency.description') }}
      </p>
    </div>

    <div class="grid gap-3 md:grid-cols-2">
      <label
        v-for="row in rows"
        :key="row.model"
        class="rounded-lg border border-white/70 bg-white/80 p-3 dark:border-white/10 dark:bg-slate-950/40"
      >
        <span class="mb-2 block text-xs font-semibold text-slate-800 dark:text-slate-100">
          {{ row.label }}
        </span>
        <input
          :value="getInputValue(row.model)"
          type="number"
          min="1"
          step="1"
          inputmode="numeric"
          class="input"
          :data-testid="`deepseek-concurrency-${row.model}`"
          :placeholder="String(row.defaultLimit)"
          @input="updateLimit(row.model, ($event.target as HTMLInputElement).value)"
        />
        <span class="mt-1 block text-[11px] leading-4 text-slate-500 dark:text-slate-400">
          {{ t('admin.accounts.deepseek.modelConcurrency.officialDefault', { count: row.defaultLimit }) }}
        </span>
      </label>
    </div>

    <p class="mt-3 text-xs leading-5 text-indigo-800/80 dark:text-indigo-200/80">
      {{ t('admin.accounts.deepseek.modelConcurrency.fallbackHint') }}
    </p>
  </section>
</template>
