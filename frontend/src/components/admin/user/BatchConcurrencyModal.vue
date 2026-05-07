<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.batchConcurrencyTitle')"
    width="normal"
    @close="$emit('close')"
  >
    <form id="batch-concurrency-form" class="space-y-5" @submit.prevent="handleSubmit">
      <div class="rounded-xl border border-sky-200 bg-sky-50/80 p-4 text-sm text-sky-900 dark:border-sky-700/40 dark:bg-sky-900/10 dark:text-sky-100">
        {{ summaryText }}
      </div>

      <Input
        v-model="concurrencyInput"
        type="number"
        :label="t('admin.users.batchConcurrencyValue')"
        :placeholder="t('admin.users.batchConcurrencyPlaceholder')"
      />
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="batch-concurrency-form"
          :disabled="submitting"
          class="btn btn-primary"
        >
          {{ submitting ? t('common.saving') : t('admin.users.batchConcurrencySubmit') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Input from '@/components/common/Input.vue'

const props = defineProps<{
  show: boolean
  matchedCount: number
  search?: string
  role?: 'admin' | 'user' | ''
  status?: 'active' | 'disabled' | ''
  groupName?: string
  attributes?: Record<number, string>
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'success'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const submitting = ref(false)
const concurrencyInput = ref('1')

const summaryText = computed(() => {
  if (props.matchedCount > 0) {
    return t('admin.users.batchConcurrencySummary', { count: props.matchedCount })
  }
  return t('admin.users.batchConcurrencySummaryUnknown')
})

function createIdempotencyKey() {
  const randomPart = globalThis.crypto?.randomUUID?.()
  if (randomPart) {
    return `users-batch-concurrency-${randomPart}`
  }
  return `users-batch-concurrency-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`
}

function buildFailureDetails(
  results: Array<{ email: string; success: boolean; error?: string }>
) {
  return results
    .filter((item) => !item.success)
    .map((item) =>
      t('admin.users.batchConcurrencyFailureDetail', {
        email: item.email,
        error: item.error || t('admin.users.batchConcurrencyFailureDetailUnknown'),
      }),
    )
}

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      concurrencyInput.value = '1'
    }
  },
)

async function handleSubmit() {
  const concurrency = Number.parseInt(concurrencyInput.value, 10)
  if (!Number.isFinite(concurrency) || concurrency < 1) {
    appStore.showError(t('admin.users.concurrencyMin'))
    return
  }

  submitting.value = true
  try {
    const idempotencyKey = createIdempotencyKey()
    const result = await adminAPI.users.batchUpdateConcurrency({
      concurrency,
      search: props.search || undefined,
      role: props.role || '',
      status: props.status || '',
      group_name: props.groupName || undefined,
      attributes: props.attributes && Object.keys(props.attributes).length > 0 ? props.attributes : undefined,
    }, idempotencyKey)

    const failureDetails = buildFailureDetails(result.results)

    if (result.matched === 0) {
      appStore.showWarning(t('admin.users.batchConcurrencyNoTargets'))
    } else if (result.failed_count > 0) {
      appStore.showWarning(
        t('admin.users.batchConcurrencyPartial', {
          success: result.success_count,
          failed: result.failed_count,
        }),
        failureDetails.length > 0 ? { details: failureDetails } : undefined,
      )
    } else {
      appStore.showSuccess(
        t('admin.users.batchConcurrencySuccess', {
          success: result.success_count,
          failed: result.failed_count,
        }),
      )
    }
    emit('success')
    emit('close')
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.users.batchConcurrencyFailed'))
  } finally {
    submitting.value = false
  }
}
</script>
