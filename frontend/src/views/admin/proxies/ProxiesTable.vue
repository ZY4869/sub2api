<template>
  <DataTable :columns="columns" :data="proxies" :loading="loading">
      <template #header-select>
        <input
          type="checkbox"
          class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="allVisibleSelected"
          @click.stop
          @change="emit('toggle-select-all', $event)"
        />
      </template>

      <template #cell-select="{ row }">
        <input
          type="checkbox"
          class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="selectedProxyIds.has(row.id)"
          @click.stop
          @change="emit('toggle-select-row', row.id, $event)"
        />
      </template>

      <template #cell-name="{ value }">
        <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
      </template>

      <template #cell-protocol="{ value }">
        <span
          v-if="value"
          :class="['badge', value.startsWith('socks5') ? 'badge-primary' : 'badge-gray']"
        >
          {{ value.toUpperCase() }}
        </span>
        <span v-else class="text-sm text-gray-400">-</span>
      </template>

      <template #cell-address="{ row }">
        <div class="flex items-center gap-1.5">
          <code class="code text-xs">{{ row.host }}:{{ row.port }}</code>
          <div class="relative">
            <button
              type="button"
              class="rounded p-0.5 text-gray-400 hover:text-primary-600 dark:hover:text-primary-400"
              :title="t('admin.proxies.copyProxyUrl')"
              @click.stop="emit('copy-proxy-url', row)"
              @contextmenu.prevent="emit('toggle-copy-menu', row.id)"
            >
              <Icon name="copy" size="sm" />
            </button>
            <div
              v-if="copyMenuProxyId === row.id"
              class="absolute left-0 top-full z-50 mt-1 w-auto min-w-[180px] rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-500 dark:bg-dark-700"
            >
              <button
                v-for="fmt in getCopyFormats(row)"
                :key="fmt.label"
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-gray-100 dark:hover:bg-dark-600"
                @click.stop="emit('copy-format', fmt.value)"
              >
                <span class="truncate font-mono text-gray-600 dark:text-gray-300">{{ fmt.label }}</span>
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #cell-auth="{ row }">
        <div v-if="row.username || row.password" class="flex items-center gap-1.5">
          <div class="flex flex-col text-xs">
            <span v-if="row.username" class="text-gray-700 dark:text-gray-200">{{ row.username }}</span>
            <span v-if="row.password" class="font-mono text-gray-500 dark:text-gray-400">
              {{ visiblePasswordIds.has(row.id) ? row.password : '••••••' }}
            </span>
          </div>
          <button
            v-if="row.password"
            type="button"
            class="ml-1 rounded p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            @click.stop="emit('toggle-password', row.id)"
          >
            <Icon :name="visiblePasswordIds.has(row.id) ? 'eyeOff' : 'eye'" size="sm" />
          </button>
        </div>
        <span v-else class="text-sm text-gray-400">-</span>
      </template>

      <template #cell-location="{ row }">
        <div class="flex items-center gap-2">
          <img
            v-if="row.country_code"
            :src="flagUrl(row.country_code)"
            :alt="row.country || row.country_code"
            class="h-4 w-6 rounded-sm"
          />
          <span v-if="formatProxyLocationLabel(row, locale)" class="text-sm text-gray-700 dark:text-gray-200">
            {{ formatProxyLocationLabel(row, locale) }}
          </span>
          <span v-else class="text-sm text-gray-400">-</span>
        </div>
      </template>

      <template #cell-account_count="{ row, value }">
        <button
          v-if="(value || 0) > 0"
          type="button"
          class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-primary-700 hover:bg-gray-200 dark:bg-dark-600 dark:text-primary-300 dark:hover:bg-dark-500"
          @click="emit('open-accounts', row)"
        >
          {{ t('admin.groups.accountsCount', { count: value || 0 }) }}
        </button>
        <span
          v-else
          class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800 dark:bg-dark-600 dark:text-gray-300"
        >
          {{ t('admin.groups.accountsCount', { count: 0 }) }}
        </span>
      </template>

      <template #cell-latency="{ row }">
        <div class="flex flex-col gap-1">
          <span
            v-if="row.latency_status === 'failed'"
            class="badge badge-danger"
            :title="row.latency_message || undefined"
          >
            {{ t('admin.proxies.latencyFailed') }}
          </span>
          <span
            v-else-if="typeof row.latency_ms === 'number'"
            :class="['badge', row.latency_ms < 200 ? 'badge-success' : 'badge-warning']"
          >
            {{ row.latency_ms }}ms
          </span>
          <span v-else class="text-sm text-gray-400">-</span>
          <div
            v-if="typeof row.quality_checked === 'number'"
            class="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400"
            :title="row.quality_summary || undefined"
          >
            <span>{{ t('admin.proxies.qualityInline', { grade: row.quality_grade || '-', score: row.quality_score ?? '-' }) }}</span>
            <span class="badge" :class="qualityOverallClass(row.quality_status)">
              {{ qualityOverallLabel(row.quality_status) }}
            </span>
          </div>
        </div>
      </template>

      <template #cell-expiry="{ row }">
        <div class="flex flex-col gap-1">
          <span
            v-if="proxyExpiryState(row) !== 'none'"
            :class="['badge', expiryBadgeClass(row)]"
            :title="formatProxyExpiryLabel(row, locale)"
          >
            {{ expiryBadgeLabel(row) }}
          </span>
          <span v-else class="text-sm text-gray-400">-</span>
          <span
            v-if="row.fallback_proxy_id"
            class="text-xs text-gray-500 dark:text-gray-400"
          >
            {{ t('admin.proxies.fallbackProxyShort', { id: row.fallback_proxy_id }) }}
          </span>
        </div>
      </template>

      <template #cell-status="{ value }">
        <span :class="['badge', value === 'active' ? 'badge-success' : 'badge-danger']">
          {{ t('admin.accounts.status.' + value) }}
        </span>
      </template>

      <template #cell-actions="{ row }">
        <div class="flex items-center gap-1">
          <button
            @click="emit('test-connection', row)"
            :disabled="testingProxyIds.has(row.id)"
            class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-emerald-50 hover:text-emerald-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400"
          >
            <svg
              v-if="testingProxyIds.has(row.id)"
              class="h-4 w-4 animate-spin"
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
            <Icon v-else name="checkCircle" size="sm" />
            <span class="text-xs">{{ t('admin.proxies.testConnection') }}</span>
          </button>
          <button
            @click="emit('quality-check', row)"
            :disabled="qualityCheckingProxyIds.has(row.id)"
            class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
          >
            <svg
              v-if="qualityCheckingProxyIds.has(row.id)"
              class="h-4 w-4 animate-spin"
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
            <Icon v-else name="shield" size="sm" />
            <span class="text-xs">{{ t('admin.proxies.qualityCheck') }}</span>
          </button>
          <button
            @click="emit('edit', row)"
            class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
          >
            <Icon name="edit" size="sm" />
            <span class="text-xs">{{ t('common.edit') }}</span>
          </button>
          <button
            @click="emit('delete', row)"
            class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          >
            <Icon name="trash" size="sm" />
            <span class="text-xs">{{ t('common.delete') }}</span>
          </button>
        </div>
      </template>

      <template #empty>
        <EmptyState
          :title="t('admin.proxies.noProxiesYet')"
          :description="t('admin.proxies.createFirstProxy')"
          :action-text="t('admin.proxies.createProxy')"
          @action="emit('create')"
        />
      </template>
  </DataTable>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Proxy } from '@/types'
import type { Column } from '@/components/common/types'
import { formatProxyLocationLabel } from '@/utils/displayLabels'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { flagUrl, getCopyFormats } from './utils'
import {
  formatProxyExpiryLabel,
  proxyExpiryState
} from './utils'

defineProps<{
  columns: Column[]
  proxies: Proxy[]
  loading: boolean
  allVisibleSelected: boolean
  selectedProxyIds: Set<number>
  visiblePasswordIds: Set<number>
  copyMenuProxyId: number | null
  testingProxyIds: Set<number>
  qualityCheckingProxyIds: Set<number>
  locale: string
  qualityOverallClass: (status?: string) => string
  qualityOverallLabel: (status?: string) => string
}>()

const emit = defineEmits<{
  create: []
  'toggle-select-all': [event: Event]
  'toggle-select-row': [id: number, event: Event]
  'toggle-password': [id: number]
  'copy-proxy-url': [proxy: Proxy]
  'toggle-copy-menu': [id: number]
  'copy-format': [value: string]
  'open-accounts': [proxy: Proxy]
  'test-connection': [proxy: Proxy]
  'quality-check': [proxy: Proxy]
  edit: [proxy: Proxy]
  delete: [proxy: Proxy]
}>()

const { t } = useI18n()

const expiryBadgeClass = (proxy: Proxy) => {
  const state = proxyExpiryState(proxy)
  if (state === 'expired') return 'badge-danger'
  if (state === 'expiring') return 'badge-warning'
  return 'badge-success'
}

const expiryBadgeLabel = (proxy: Proxy) => {
  const state = proxyExpiryState(proxy)
  if (state === 'expired') return t('admin.proxies.expired')
  if (state === 'expiring') return t('admin.proxies.expiringSoon')
  return t('admin.proxies.expiryActive')
}
</script>
