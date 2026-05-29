<template>
  <div class="md:col-span-2">
    <div class="rounded-2xl border border-gray-100 p-4 dark:border-dark-700">
      <div class="mb-4">
        <h3 class="font-medium text-gray-900 dark:text-white">
          {{ t('admin.settings.moderation.thresholdsTitle') }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.moderation.thresholdsHint') }}
        </p>
      </div>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <label
          v-for="category in categories"
          :key="category"
          class="rounded-xl bg-gray-50 p-3 dark:bg-dark-800/70"
        >
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ categoryLabel(category) }}
            </span>
            <span class="font-mono text-xs text-gray-500 dark:text-gray-400">
              {{ displayValue(category) }}
            </span>
          </div>
          <div class="flex items-center gap-3">
            <input
              :value="thresholds[category] ?? 1"
              type="range"
              min="0"
              max="1"
              step="0.01"
              class="w-full"
              :aria-label="categoryLabel(category)"
              @input="updateCategory(category, ($event.target as HTMLInputElement).value)"
            />
            <input
              :value="thresholds[category] ?? 1"
              type="number"
              min="0"
              max="1"
              step="0.01"
              class="input w-24"
              :aria-label="categoryLabel(category)"
              @input="updateCategory(category, ($event.target as HTMLInputElement).value)"
            />
          </div>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const categories = [
  'hate',
  'hate/threatening',
  'harassment',
  'harassment/threatening',
  'self-harm',
  'self-harm/intent',
  'self-harm/instructions',
  'sexual',
  'sexual/minors',
  'violence',
  'violence/graphic',
  'illicit',
  'illicit/violent'
] as const

const props = defineProps<{
  modelValue: Record<string, number>
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, number>): void
}>()

const { t, te } = useI18n()

const thresholds = computed(() => normalizeThresholds(props.modelValue))

function normalizeThresholds(value: Record<string, number> | undefined): Record<string, number> {
  const normalized: Record<string, number> = {}
  for (const category of categories) {
    const raw = value?.[category]
    normalized[category] = typeof raw === 'number' && Number.isFinite(raw) ? clampThreshold(raw) : 1
  }
  return normalized
}

function clampThreshold(value: number): number {
  return Math.min(1, Math.max(0, Math.round(value * 100) / 100))
}

function updateCategory(category: string, raw: string) {
  const parsed = Number(raw)
  const next = {
    ...thresholds.value,
    [category]: Number.isFinite(parsed) ? clampThreshold(parsed) : 1
  }
  emit('update:modelValue', next)
}

function displayValue(category: string): string {
  return (thresholds.value[category] ?? 1).toFixed(2)
}

function categoryLabel(category: string): string {
  const key = `admin.settings.moderation.thresholdCategories.${category}`
  return te(key) ? t(key) : category
}
</script>
