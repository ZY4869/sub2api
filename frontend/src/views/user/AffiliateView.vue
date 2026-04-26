<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('affiliate.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('affiliate.description') }}
          </p>
        </div>

        <div class="flex gap-2">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            @click="reload()"
          >
            <Icon name="refresh" size="sm" class="mr-2" />
            {{ t('common.refresh') }}
          </button>

          <button
            type="button"
            class="btn btn-primary"
            :disabled="transferDisabled"
            @click="handleTransfer()"
          >
            <svg
              v-if="transferring"
              class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <Icon v-else name="creditCard" size="sm" class="mr-2" />
            {{ t('affiliate.transfer.button') }}
          </button>
        </div>
      </div>

      <div v-if="errorMessage" class="card border-red-200 bg-red-50 p-6 dark:border-red-800/50 dark:bg-red-900/20">
        <div class="flex items-start gap-3">
          <Icon name="exclamationCircle" size="md" class="mt-0.5 text-red-500" />
          <div class="text-sm text-red-700 dark:text-red-400">
            <p class="font-medium">{{ t('common.error') }}</p>
            <p class="mt-1 break-words">{{ errorMessage }}</p>
          </div>
        </div>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <template v-else-if="info">
        <div v-if="info.enabled !== true" class="card border-amber-200 bg-amber-50 p-6 dark:border-amber-800/50 dark:bg-amber-900/20">
          <div class="flex items-start gap-3">
            <Icon name="exclamationTriangle" size="md" class="mt-0.5 text-amber-500" />
            <div class="text-sm text-amber-700 dark:text-amber-300">
              <p class="font-medium">{{ t('affiliate.disabled.title') }}</p>
              <p class="mt-1">{{ t('affiliate.disabled.desc') }}</p>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-6 sm:grid-cols-4">
          <StatCard
            :title="t('affiliate.stats.invitees')"
            :value="info.invitee_count"
            :icon="UsersIcon"
            icon-variant="primary"
          />
          <StatCard
            :title="t('affiliate.stats.available')"
            :value="formatCurrency(info.rebate_balance)"
            :icon="WalletIcon"
            icon-variant="success"
          />
          <StatCard
            :title="t('affiliate.stats.frozen')"
            :value="formatCurrency(info.rebate_frozen_balance)"
            :icon="SnowflakeIcon"
            icon-variant="warning"
          />
          <StatCard
            :title="t('affiliate.stats.lifetime')"
            :value="formatCurrency(info.lifetime_rebate)"
            :icon="StarIcon"
            icon-variant="primary"
          />
        </div>

        <div class="card p-6">
          <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('affiliate.myCode') }}
              </label>
              <div class="flex items-center gap-2">
                <code class="select-all break-all rounded bg-gray-100 px-3 py-2 font-mono text-sm text-gray-900 dark:bg-dark-700 dark:text-gray-100">
                  {{ info.aff_code }}
                </code>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click="copyToClipboard(info.aff_code, t('affiliate.copied.code'))"
                >
                  {{ t('common.copy') }}
                </button>
              </div>
              <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
                {{ t('affiliate.myCodeHint') }}
              </p>
            </div>

            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('affiliate.inviteLink') }}
              </label>
              <div class="flex items-center gap-2">
                <code class="select-all break-all rounded bg-gray-100 px-3 py-2 font-mono text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                  {{ inviteLink }}
                </code>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click="copyToClipboard(inviteLink, t('affiliate.copied.link'))"
                >
                  {{ t('common.copy') }}
                </button>
              </div>
              <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
                {{ t('affiliate.inviteLinkHint') }}
              </p>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('affiliate.rulesTitle') }}
          </h2>
          <div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
            <div class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
              <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.rules.rate') }}</div>
              <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                {{ info.effective_rate_percent.toFixed(2) }}%
              </div>
            </div>
            <div class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
              <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.rules.freeze') }}</div>
              <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                {{ info.rebate_freeze_hours }}h
              </div>
            </div>
            <div class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
              <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.rules.duration') }}</div>
              <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                {{ info.rebate_duration_days === 0 ? t('affiliate.rules.durationForever') : `${info.rebate_duration_days}d` }}
              </div>
            </div>
            <div class="rounded-lg border border-gray-100 p-4 dark:border-dark-700">
              <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.rules.cap') }}</div>
              <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                {{ info.rebate_per_invitee_cap === 0 ? t('affiliate.rules.capUnlimited') : formatCurrency(info.rebate_per_invitee_cap) }}
              </div>
            </div>
          </div>

          <div class="mt-4 text-xs text-gray-500 dark:text-dark-400">
            <div>{{ t('affiliate.rules.switches', { usage: info.rebate_on_usage_enabled ? t('common.on') : t('common.off'), topup: info.rebate_on_topup_enabled ? t('common.on') : t('common.off') }) }}</div>
            <div v-if="info.transfer_enabled !== true" class="mt-1 text-amber-600 dark:text-amber-300">
              {{ t('affiliate.transfer.disabled') }}
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import StatCard from '@/components/common/StatCard.vue'
import Icon from '@/components/icons/Icon.vue'
import { affiliateAPI } from '@/api'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore, useAuthStore } from '@/stores'
import type { AffiliateUserInfo } from '@/api/affiliate'
import { buildAuthErrorMessage } from '@/utils/authError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(false)
const transferring = ref(false)
const errorMessage = ref('')
const info = ref<AffiliateUserInfo | null>(null)

const inviteLink = computed(() => {
  const code = info.value?.aff_code || ''
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  const url = new URL('/register', origin || 'http://localhost')
  if (code) {
    url.searchParams.set('aff_code', code)
  }
  return url.toString()
})

const transferDisabled = computed(() => {
  if (loading.value || transferring.value) return true
  if (!info.value) return true
  if (info.value.transfer_enabled !== true) return true
  return info.value.rebate_balance <= 0
})

function formatCurrency(v: number) {
  return `$${Number(v || 0).toFixed(2)}`
}

async function reload() {
  loading.value = true
  errorMessage.value = ''
  try {
    info.value = await affiliateAPI.getMyAffiliateInfo()
  } catch (err: any) {
    errorMessage.value = buildAuthErrorMessage(err, { fallback: t('common.unknownError') })
  } finally {
    loading.value = false
  }
}

async function handleTransfer() {
  if (transferDisabled.value) {
    if (info.value?.transfer_enabled !== true) {
      appStore.showError(t('affiliate.transfer.disabled'))
      return
    }
    if ((info.value?.rebate_balance || 0) <= 0) {
      appStore.showError(t('affiliate.transfer.noBalance'))
      return
    }
    return
  }

  transferring.value = true
  try {
    const result = await affiliateAPI.transferToBalance()
    if (authStore.user) {
      authStore.user.balance = result.new_balance
    }
    if (result.transferred_amount > 0) {
      appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(result.transferred_amount) }))
    } else {
      appStore.showSuccess(t('affiliate.transfer.none'))
    }
    await reload()
  } catch (err: any) {
    appStore.showError(buildAuthErrorMessage(err, { fallback: t('common.unknownError') }))
  } finally {
    transferring.value = false
  }
}

onMounted(async () => {
  // Ensure public settings are loaded so sidebar gating doesn't flicker.
  await appStore.fetchPublicSettings(false)
  await reload()
})

const UsersIcon = {
  render: () =>
    h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z' })
    ])
}

const WalletIcon = {
  render: () =>
    h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M21 12a2.25 2.25 0 00-2.25-2.25H15a3 3 0 11-6 0H5.25A2.25 2.25 0 003 12m18 0v6.75A2.25 2.25 0 0118.75 21H5.25A2.25 2.25 0 013 18.75V12m18 0V5.25A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25V12' })
    ])
}

const SnowflakeIcon = {
  render: () =>
    h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M12 3v18m0 0l-3-3m3 3l3-3m6-9H3m0 0l3 3m-3-3l3-3m12.364 9.364L6.636 6.636m0 0v4.243m0-4.243h4.243m12.728 12.728v-4.243m0 4.243h-4.243M17.364 6.636L6.636 17.364m0 0h4.243m-4.243 0v-4.243m12.728-6.485h-4.243m4.243 0v4.243' })
    ])
}

const StarIcon = {
  render: () =>
    h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [
      h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.563.563 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345l2.125-5.11z' })
    ])
}
</script>
