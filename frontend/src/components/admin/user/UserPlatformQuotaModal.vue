<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.platformQuotasTitle')"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="user" class="space-y-5">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100">
          <span class="text-lg font-medium text-primary-700">
            {{ user.email.charAt(0).toUpperCase() }}
          </span>
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.users.platformQuotasHint') }}
          </p>
        </div>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-10">
        <LoadingSpinner />
      </div>

      <form v-else id="platform-quota-form" class="space-y-3" @submit.prevent="save">
        <div class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
          <div class="hidden grid-cols-[1.2fr_repeat(3,minmax(0,1fr))] gap-3 bg-gray-50 px-4 py-3 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400 md:grid">
            <span>{{ t('admin.users.quotaPlatform') }}</span>
            <span>{{ t('admin.users.quotaDaily') }}</span>
            <span>{{ t('admin.users.quotaWeekly') }}</span>
            <span>{{ t('admin.users.quotaMonthly') }}</span>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div
              v-for="row in rows"
              :key="row.platform"
              class="grid grid-cols-1 gap-4 px-4 py-4 md:grid-cols-[1.2fr_repeat(3,minmax(0,1fr))] md:items-start"
            >
              <div class="flex items-center gap-2">
                <PlatformIcon :platform="row.platform" size="md" />
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ getPlatformEnglishName(row.platform) }}
                </span>
              </div>
              <QuotaLimitInput
                v-model="row.dailyLimit"
                :label="t('admin.users.quotaDaily')"
                :used="row.quota?.daily.used"
                :reset-at="row.quota?.daily.reset_at"
              />
              <QuotaLimitInput
                v-model="row.weeklyLimit"
                :label="t('admin.users.quotaWeekly')"
                :used="row.quota?.weekly.used"
                :reset-at="row.quota?.weekly.reset_at"
              />
              <QuotaLimitInput
                v-model="row.monthlyLimit"
                :label="t('admin.users.quotaMonthly')"
                :used="row.quota?.monthly.used"
                :reset-at="row.quota?.monthly.reset_at"
              />
            </div>
          </div>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.users.quotaBlankUnlimited') }}
        </p>
      </form>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="platform-quota-form"
          class="btn btn-primary"
          :disabled="loading || saving"
        >
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { defineComponent, h, ref, watch, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { UserPlatformQuota, UserPlatformQuotaInput } from '@/api/admin/users'
import type { AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useAppStore } from '@/stores/app'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { FILTER_PLATFORM_ORDER, getPlatformEnglishName } from '@/utils/platformBranding'

interface Props {
  show: boolean
  user: AdminUser | null
}

interface QuotaRow {
  platform: string
  dailyLimit: string
  weeklyLimit: string
  monthlyLimit: string
  quota?: UserPlatformQuota
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'success'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const saving = ref(false)
const rows = ref<QuotaRow[]>([])

const QuotaLimitInput = defineComponent({
  props: {
    modelValue: { type: String, required: true },
    label: { type: String, required: true },
    used: { type: Number as PropType<number | undefined>, default: undefined },
    resetAt: { type: String as PropType<string | null | undefined>, default: null }
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
            onInput: (event: Event) => {
              inputEmit('update:modelValue', (event.target as HTMLInputElement).value)
            }
          })
        ]),
        h('div', { class: 'space-y-0.5 text-xs text-gray-500 dark:text-gray-400' }, [
          h('p', null, `${t('admin.users.quotaUsed')}: ${formatCurrency(inputProps.used || 0)}`),
          inputProps.resetAt
            ? h('p', null, `${t('admin.users.quotaResetAt')}: ${formatDateTime(inputProps.resetAt)}`)
            : h('p', null, `${t('admin.users.quotaResetAt')}: ${t('admin.users.quotaNoWindow')}`)
        ])
      ])
  }
})

watch(
  () => [props.show, props.user?.id] as const,
  ([show]) => {
    if (show) {
      void load()
    } else {
      rows.value = []
    }
  }
)

async function load() {
  if (!props.user) return
  loading.value = true
  try {
    const quotas = await adminAPI.users.getUserPlatformQuotas(props.user.id)
    rows.value = buildRows(quotas)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.users.platformQuotasLoadFailed'))
    rows.value = buildRows([])
  } finally {
    loading.value = false
  }
}

function buildRows(quotas: UserPlatformQuota[]): QuotaRow[] {
  const quotaMap = new Map(quotas.map((quota) => [quota.platform, quota]))
  const platforms: string[] = [...FILTER_PLATFORM_ORDER]
  for (const quota of quotas) {
    if (!platforms.includes(quota.platform)) {
      platforms.push(quota.platform)
    }
  }
  return platforms.map((platform) => {
    const quota = quotaMap.get(platform)
    return {
      platform,
      dailyLimit: limitToInput(quota?.daily.limit),
      weeklyLimit: limitToInput(quota?.weekly.limit),
      monthlyLimit: limitToInput(quota?.monthly.limit),
      quota
    }
  })
}

function limitToInput(value: number | null | undefined): string {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? String(value) : ''
}

function parseLimit(value: string): number | null {
  const trimmed = value.trim()
  if (!trimmed) return null
  const parsed = Number(trimmed)
  if (!Number.isFinite(parsed) || parsed < 0) {
    throw new Error('invalid')
  }
  return parsed === 0 ? null : parsed
}

function rowToInput(row: QuotaRow): UserPlatformQuotaInput | null {
  const item = {
    platform: row.platform,
    daily_limit_usd: parseLimit(row.dailyLimit),
    weekly_limit_usd: parseLimit(row.weeklyLimit),
    monthly_limit_usd: parseLimit(row.monthlyLimit)
  }
  if (!item.daily_limit_usd && !item.weekly_limit_usd && !item.monthly_limit_usd) {
    return null
  }
  return item
}

async function save() {
  if (!props.user) return
  let payload: UserPlatformQuotaInput[]
  try {
    payload = rows.value
      .map((row) => rowToInput(row))
      .filter((item): item is UserPlatformQuotaInput => item !== null)
  } catch {
    appStore.showError(t('admin.users.quotaInvalid'))
    return
  }
  saving.value = true
  try {
    const updated = await adminAPI.users.updateUserPlatformQuotas(props.user.id, payload)
    rows.value = buildRows(updated)
    appStore.showSuccess(t('admin.users.platformQuotasSaved'))
    emit('success')
    emit('close')
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.users.platformQuotasSaveFailed'))
  } finally {
    saving.value = false
  }
}
</script>
