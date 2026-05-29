<template>
  <div class="card p-5">
    <div class="mb-4 flex items-center justify-between gap-3">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('dashboard.platformQuotas') }}
        </h2>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('dashboard.platformQuotasHint') }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="loading"
        @click="load"
      >
        {{ t('dashboard.retry') }}
      </button>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-8">
      <LoadingSpinner />
    </div>
    <div
      v-else-if="error"
      class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300"
    >
      {{ t('dashboard.platformQuotasLoadFailed') }}
    </div>
    <div v-else-if="sortedQuotas.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('dashboard.platformQuotasEmpty') }}
    </div>
    <div v-else class="grid grid-cols-1 gap-3 lg:grid-cols-2">
      <div
        v-for="quota in sortedQuotas"
        :key="quota.platform"
        class="rounded-xl border border-gray-100 p-4 dark:border-dark-700"
      >
        <div class="mb-3 flex items-center gap-2">
          <PlatformIcon :platform="quota.platform" size="md" />
          <span class="font-medium text-gray-900 dark:text-white">
            {{ getPlatformEnglishName(quota.platform) }}
          </span>
        </div>
        <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
          <QuotaCycle
            :label="t('dashboard.daily')"
            :limit="quota.daily.limit"
            :used="quota.daily.used"
            :reset-at="quota.daily.reset_at"
          />
          <QuotaCycle
            :label="t('dashboard.weekly')"
            :limit="quota.weekly.limit"
            :used="quota.weekly.used"
            :reset-at="quota.weekly.reset_at"
          />
          <QuotaCycle
            :label="t('dashboard.monthly')"
            :limit="quota.monthly.limit"
            :used="quota.monthly.used"
            :reset-at="quota.monthly.reset_at"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { userAPI } from '@/api/user'
import type { UserPlatformQuota } from '@/api/admin/users'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { getPlatformEnglishName, getPlatformOrderIndex } from '@/utils/platformBranding'

const { t } = useI18n()
const quotas = ref<UserPlatformQuota[]>([])
const loading = ref(false)
const error = ref(false)

const QuotaCycle = defineComponent({
  props: {
    label: { type: String, required: true },
    limit: { type: Number as PropType<number | null>, default: null },
    used: { type: Number, required: true },
    resetAt: { type: String as PropType<string | null | undefined>, default: null }
  },
  setup(props) {
    return () =>
      h('div', { class: 'rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60' }, [
        h('div', { class: 'text-xs font-medium text-gray-500 dark:text-gray-400' }, props.label),
        h('div', { class: 'mt-1 text-sm font-semibold text-gray-900 dark:text-white' }, [
          `${formatCurrency(props.used)} / `,
          props.limit === null ? t('dashboard.unlimited') : formatCurrency(props.limit)
        ]),
        h(
          'div',
          { class: 'mt-1 text-xs text-gray-500 dark:text-gray-400' },
          `${t('dashboard.resetAt')}: ${props.resetAt ? formatDateTime(props.resetAt) : t('dashboard.noResetWindow')}`
        )
      ])
  }
})

const sortedQuotas = computed(() =>
  [...quotas.value].sort((left, right) => {
    const order = getPlatformOrderIndex(left.platform) - getPlatformOrderIndex(right.platform)
    return order === 0 ? left.platform.localeCompare(right.platform) : order
  })
)

async function load() {
  loading.value = true
  error.value = false
  try {
    quotas.value = await userAPI.getPlatformQuotas()
  } catch (loadError) {
    console.error('Failed to load platform quotas:', loadError)
    error.value = true
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
