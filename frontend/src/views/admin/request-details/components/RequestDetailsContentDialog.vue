<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useClipboard } from '@/composables/useClipboard'

const props = withDefaults(defineProps<{
  show: boolean
  title: string
  content?: string
  notice?: string
  loading?: boolean
  emptyMessage?: string
}>(), {
  content: '',
  notice: '',
  loading: false,
  emptyMessage: ''
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const hasContent = computed(() => String(props.content || '').trim().length > 0)

async function handleCopy() {
  if (!hasContent.value) return
  await copyToClipboard(props.content, t('common.copiedToClipboard'))
}
</script>

<template>
  <BaseDialog
    :show="show"
    :title="title"
    width="full"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="flex min-h-0 flex-col gap-4">
      <div v-if="notice" class="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200">
        {{ notice }}
      </div>

      <div class="flex items-center justify-end gap-3">
        <button
          class="btn btn-secondary btn-sm"
          type="button"
          :disabled="loading || !hasContent"
          @click="handleCopy"
        >
          {{ t('admin.requestDetails.drawer.copyContent') }}
        </button>
      </div>

      <div v-if="loading" class="flex min-h-[320px] items-center justify-center rounded-2xl border border-gray-200 bg-gray-50 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div
        v-else-if="!hasContent"
        class="flex min-h-[320px] items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
      >
        {{ emptyMessage || t('common.noData') }}
      </div>

      <pre
        v-else
        data-test="request-details-full-dialog"
        class="max-h-[72vh] overflow-auto rounded-2xl border border-gray-200 bg-gray-50 p-4 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200"
      ><code>{{ content }}</code></pre>
    </div>
  </BaseDialog>
</template>
