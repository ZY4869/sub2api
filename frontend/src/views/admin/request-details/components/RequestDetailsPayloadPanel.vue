<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'

const props = withDefaults(defineProps<{
  title: string
  content?: string
  loading?: boolean
  emptyMessage?: string
  notice?: string
  canOpenFull?: boolean
}>(), {
  content: '',
  loading: false,
  emptyMessage: '',
  notice: '',
  canOpenFull: undefined
})

const emit = defineEmits<{
  (e: 'open-full'): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const hasContent = computed(() => String(props.content || '').trim().length > 0)
const canViewFull = computed(() => props.canOpenFull ?? hasContent.value)
const statusText = computed(() => {
  if (props.loading) return t('common.loading')
  return hasContent.value
    ? t('admin.requestDetails.drawer.payload.previewReady')
    : t('admin.requestDetails.drawer.payload.empty')
})

async function handleCopy() {
  if (!hasContent.value) return
  await copyToClipboard(props.content, t('common.copiedToClipboard'))
}
</script>

<template>
  <section class="rounded-2xl border border-gray-200 dark:border-dark-700">
    <div class="flex items-start justify-between gap-4 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
      <div class="min-w-0">
        <div class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ title }}
        </div>
        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ statusText }}
        </div>
      </div>

      <div class="flex shrink-0 items-center gap-2">
        <button
          class="btn btn-secondary btn-sm"
          type="button"
          :disabled="loading || !hasContent"
          @click="handleCopy"
        >
          {{ t('admin.requestDetails.drawer.copyContent') }}
        </button>
        <button
          class="btn btn-secondary btn-sm"
          type="button"
          :disabled="loading || !canViewFull"
          @click="emit('open-full')"
        >
          {{ t('admin.requestDetails.drawer.viewFull') }}
        </button>
      </div>
    </div>

    <div class="p-4">
      <div v-if="notice" class="mb-4 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200">
        {{ notice }}
      </div>

      <div v-if="loading" class="flex min-h-[240px] items-center justify-center rounded-2xl bg-gray-50 text-sm text-gray-500 dark:bg-dark-800 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div
        v-else-if="!hasContent"
        data-test="payload-empty"
        class="flex min-h-[240px] items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
      >
        {{ emptyMessage || t('common.noData') }}
      </div>

      <pre
        v-else
        class="max-h-[360px] overflow-auto rounded-2xl bg-gray-50 p-4 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"
      ><code>{{ content }}</code></pre>
    </div>
  </section>
</template>
