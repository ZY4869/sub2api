<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
import type { RequestPreviewRenderState } from '@/utils/requestPreview'

const props = withDefaults(defineProps<{
  title: string
  content?: string
  loading?: boolean
  emptyMessage?: string
  notice?: string
  canOpenFull?: boolean
  source?: string
  truncated?: boolean
  state?: RequestPreviewRenderState
}>(), {
  content: '',
  loading: false,
  emptyMessage: '',
  notice: '',
  canOpenFull: undefined,
  source: '',
  truncated: false,
  state: 'uncollected'
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
  switch (props.state) {
    case 'ready':
      return t('admin.requestDetails.drawer.payload.previewReady')
    case 'raw_only':
      return t('admin.requestDetails.drawer.payload.rawOnlyStatus')
    case 'empty':
      return t('admin.requestDetails.drawer.payload.collectedEmpty')
    default:
      return t('admin.requestDetails.drawer.payload.empty')
  }
})
const emptyText = computed(() => {
  switch (props.state) {
    case 'empty':
      return t('admin.requestDetails.drawer.payload.collectedEmpty')
    case 'raw_only':
      return t('admin.requestDetails.drawer.payload.rawOnlyEmpty')
    default:
      return props.emptyMessage || t('common.noData')
  }
})
const notices = computed(() => {
  const messages: string[] = []
  if (props.notice) {
    messages.push(props.notice)
  }
  if (props.state === 'raw_only') {
    messages.push(t('admin.requestDetails.drawer.payload.rawOnlyNotice'))
  }
  if (props.truncated) {
    messages.push(t('admin.requestDetails.drawer.payload.truncatedNotice'))
  }
  return messages
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
        <div v-if="source || truncated || state === 'raw_only'" class="mt-2 flex flex-wrap gap-2 text-[11px]">
          <span
            v-if="source"
            class="inline-flex items-center rounded-full bg-sky-100 px-2 py-0.5 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300"
          >
            {{ t('admin.requestDetails.drawer.payload.sourceLabel', { source }) }}
          </span>
          <span
            v-if="state === 'raw_only'"
            class="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200"
          >
            {{ t('admin.requestDetails.drawer.payload.rawOnlyBadge') }}
          </span>
          <span
            v-if="truncated"
            class="inline-flex items-center rounded-full bg-rose-100 px-2 py-0.5 text-rose-700 dark:bg-rose-900/30 dark:text-rose-200"
          >
            {{ t('admin.requestDetails.drawer.payload.truncatedBadge') }}
          </span>
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
      <div
        v-for="message in notices"
        :key="message"
        class="mb-4 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200"
      >
        {{ message }}
      </div>

      <div v-if="loading" class="flex min-h-[240px] items-center justify-center rounded-2xl bg-gray-50 text-sm text-gray-500 dark:bg-dark-800 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div
        v-else-if="!hasContent"
        data-test="payload-empty"
        class="flex min-h-[240px] items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
      >
        {{ emptyText }}
      </div>

      <pre
        v-else
        class="max-h-[360px] overflow-auto rounded-2xl bg-gray-50 p-4 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"
      ><code>{{ content }}</code></pre>
    </div>
  </section>
</template>
