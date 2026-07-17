<template>
  <BaseDialog :show="show" :title="t('admin.users.batchPlatformQuotaTitle')" width="wide" @close="emit('close')">
    <form id="batch-platform-quota-form" class="space-y-5" @submit.prevent="handleSubmit">
      <div class="rounded-xl border border-amber-200 bg-amber-50/80 p-4 text-sm text-amber-900 dark:border-amber-700/40 dark:bg-amber-900/10 dark:text-amber-100">
        {{ summaryText }}
      </div>

      <div class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
        <div class="hidden grid-cols-[1.1fr_repeat(3,minmax(0,1fr))] gap-3 bg-gray-50 px-4 py-3 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400 md:grid">
          <span>{{ t('admin.users.quotaPlatform') }}</span>
          <span>{{ t('admin.users.quotaDaily') }}</span>
          <span>{{ t('admin.users.quotaWeekly') }}</span>
          <span>{{ t('admin.users.quotaMonthly') }}</span>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-dark-700">
          <div
            v-for="row in rows"
            :key="row.platform"
            class="grid grid-cols-1 gap-4 px-4 py-4 md:grid-cols-[1.1fr_repeat(3,minmax(0,1fr))] md:items-start"
          >
            <div class="flex items-center gap-2">
              <PlatformIcon :platform="row.platform" size="md" />
              <span class="font-medium text-gray-900 dark:text-white">{{ getPlatformEnglishName(row.platform) }}</span>
            </div>
            <QuotaInput v-model="row.dailyLimit" :label="t('admin.users.quotaDaily')" />
            <QuotaInput v-model="row.weeklyLimit" :label="t('admin.users.quotaWeekly')" />
            <QuotaInput v-model="row.monthlyLimit" :label="t('admin.users.quotaMonthly')" />
          </div>
        </div>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.users.batchPlatformQuotaReplaceHint') }}
      </p>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="submit" form="batch-platform-quota-form" :disabled="submitting" class="btn btn-primary">{{ submitting ? t('common.saving') : t('admin.users.batchPlatformQuotaSubmit') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, watch, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { UserPlatformQuotaInput } from '@/api/admin/users'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useAppStore } from '@/stores/app'
import { FILTER_PLATFORM_ORDER, getPlatformEnglishName } from '@/utils/platformBranding'

interface QuotaRow {
  platform: string
  dailyLimit: string
  weeklyLimit: string
  monthlyLimit: string
}

const props = defineProps<{
  show: boolean
  matchedCount: number
  search?: string
  role?: 'admin' | 'user' | ''
  status?: 'active' | 'disabled' | ''
  groupName?: string
  apiKeyGroupId?: string
  attributes?: Record<number, string>
}>()

const emit = defineEmits<{ (e: 'close'): void; (e: 'success'): void }>()

const { t } = useI18n()
const appStore = useAppStore()
const submitting = ref(false)
const rows = ref<QuotaRow[]>([])

const QuotaInput = defineComponent({
  props: {
    modelValue: { type: String, required: true },
    label: { type: String as PropType<string>, required: true }
  },
  emits: ['update:modelValue'],
  setup(inputProps, { emit: inputEmit }) {
    return () =>
      h('div', { class: 'space-y-1.5' }, [
        h('label', { class: 'block text-xs font-medium text-gray-500 md:hidden dark:text-gray-400' }, inputProps.label),
        h('div', { class: 'relative' }, [
          h('span', { class: 'pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-gray-400' }, '$'),
          h('input', {
            value: inputProps.modelValue,
            type: 'number',
            min: '0',
            step: '0.0001',
            class: 'input pl-7',
            placeholder: t('admin.users.quotaUnlimited'),
            'aria-label': inputProps.label,
            onInput: (event: Event) => inputEmit('update:modelValue', (event.target as HTMLInputElement).value)
          })
        ])
      ])
  }
})

const summaryText = computed(() =>
  props.matchedCount > 0
    ? t('admin.users.batchPlatformQuotaSummary', { count: props.matchedCount })
    : t('admin.users.batchPlatformQuotaSummaryUnknown')
)

watch(() => props.show, (visible) => {
  rows.value = visible ? FILTER_PLATFORM_ORDER.map((platform) => ({ platform, dailyLimit: '', weeklyLimit: '', monthlyLimit: '' })) : []
}, { immediate: true })

function parseLimit(value: string): number | null {
  const trimmed = value.trim()
  if (!trimmed) return null
  const parsed = Number(trimmed)
  if (!Number.isFinite(parsed) || parsed < 0) throw new Error('invalid')
  return parsed === 0 ? null : parsed
}

function buildPayload(): UserPlatformQuotaInput[] {
  return rows.value
    .map((row) => ({
      platform: row.platform,
      daily_limit_usd: parseLimit(row.dailyLimit),
      weekly_limit_usd: parseLimit(row.weeklyLimit),
      monthly_limit_usd: parseLimit(row.monthlyLimit)
    }))
    .filter((item) => item.daily_limit_usd || item.weekly_limit_usd || item.monthly_limit_usd)
}

function createIdempotencyKey() {
  const randomPart = globalThis.crypto?.randomUUID?.()
  return randomPart
    ? `users-batch-platform-quotas-${randomPart}`
    : `users-batch-platform-quotas-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`
}

function failureDetails(results: Array<{ email: string; success: boolean; error?: string }>) {
  return results
    .filter((item) => !item.success)
    .map((item) => t('admin.users.batchPlatformQuotaFailureDetail', {
      email: item.email,
      error: item.error || t('admin.users.batchPlatformQuotaFailureUnknown')
    }))
}

async function handleSubmit() {
  let items: UserPlatformQuotaInput[]
  try {
    items = buildPayload()
  } catch {
    appStore.showError(t('admin.users.quotaInvalid'))
    return
  }
  if (items.length === 0) {
    appStore.showError(t('admin.users.batchPlatformQuotaEmpty'))
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.users.batchUpdatePlatformQuotas({
      items,
      search: props.search || undefined,
      role: props.role || '',
      status: props.status || '',
      group_name: props.groupName || undefined,
      api_key_group_id: props.apiKeyGroupId || undefined,
      attributes: props.attributes && Object.keys(props.attributes).length > 0 ? props.attributes : undefined
    }, createIdempotencyKey())
    if (result.matched === 0) {
      appStore.showWarning(t('admin.users.batchPlatformQuotaNoTargets'))
    } else if (result.failed_count > 0) {
      appStore.showWarning(
        t('admin.users.batchPlatformQuotaPartial', { success: result.success_count, failed: result.failed_count }),
        { details: failureDetails(result.results) }
      )
    } else {
      appStore.showSuccess(t('admin.users.batchPlatformQuotaSuccess', { success: result.success_count, failed: result.failed_count }))
    }
    emit('success')
    emit('close')
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.users.batchPlatformQuotaFailed'))
  } finally {
    submitting.value = false
  }
}
</script>
