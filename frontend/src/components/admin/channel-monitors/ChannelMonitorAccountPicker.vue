<template>
  <div>
    <label class="input-label">
      {{ t('admin.channelMonitors.fields.accounts') }} <span class="text-red-500">*</span>
    </label>
    <div class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
      <div class="mb-3">
        <input
          :value="search"
          type="text"
          class="input"
          :placeholder="t('admin.channelMonitors.fields.accountSearchPlaceholder')"
          @input="emit('update:search', ($event.target as HTMLInputElement).value)"
        />
      </div>
      <div class="max-h-44 space-y-1 overflow-y-auto">
        <label
          v-for="account in accounts"
          :key="account.id"
          class="flex cursor-pointer items-center justify-between gap-3 rounded-lg px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-dark-700"
        >
          <span class="flex min-w-0 items-center gap-2">
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedIds.includes(account.id)"
              @change="emit('toggle', account.id)"
            />
            <ModelPlatformIcon :platform="account.platform" size="sm" />
            <span class="truncate text-gray-800 dark:text-gray-100">{{ account.name }}</span>
          </span>
          <span class="shrink-0 rounded px-2 py-0.5 text-xs" :class="accountStatusClass(account.status)">
            {{ accountStatusLabel(account.status) }}
          </span>
        </label>
        <div v-if="!loading && accounts.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.channelMonitors.fields.noAccounts') }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { Account } from '@/types'

defineProps<{
  accounts: Account[]
  selectedIds: number[]
  search: string
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle', id: number): void
  (e: 'update:search', value: string): void
}>()

const { t } = useI18n()

function accountStatusLabel(status?: string): string {
  if (status === 'active') return t('admin.channelMonitors.accountStatus.active')
  if (status === 'error') return t('admin.channelMonitors.accountStatus.error')
  return t('admin.channelMonitors.accountStatus.inactive')
}

function accountStatusClass(status?: string): string {
  if (status === 'active') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  if (status === 'error') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  return 'bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400'
}
</script>
