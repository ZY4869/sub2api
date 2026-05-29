<template>
  <BaseDialog
    :show="show"
    :title="t('admin.proxies.accountsTitle', { name: accountsProxy?.name || '' })"
    width="normal"
    @close="emit('close')"
  >
    <div v-if="accountsLoading" class="flex items-center justify-center py-8 text-sm text-gray-500">
      <Icon name="refresh" size="md" class="mr-2 animate-spin" />
      {{ t('common.loading') }}
    </div>
    <div v-else-if="proxyAccounts.length === 0" class="py-6 text-center text-sm text-gray-500">
      {{ t('admin.proxies.accountsEmpty') }}
    </div>
    <div v-else class="max-h-80 overflow-auto">
      <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
        <thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-800 dark:text-dark-400">
          <tr>
            <th class="px-4 py-2 text-left">{{ t('admin.proxies.accountName') }}</th>
            <th class="px-4 py-2 text-left">{{ t('admin.accounts.columns.platformType') }}</th>
            <th class="px-4 py-2 text-left">{{ t('admin.proxies.accountNotes') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
          <tr v-for="account in proxyAccounts" :key="account.id">
            <td class="px-4 py-2 font-medium text-gray-900 dark:text-white">{{ account.name }}</td>
            <td class="px-4 py-2">
              <PlatformTypeBadge
                :platform="account.platform"
                :gateway-protocol="account.gateway_protocol"
                :type="account.type"
              />
            </td>
            <td class="px-4 py-2 text-gray-600 dark:text-gray-300">
              {{ account.notes || '-' }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <button @click="emit('close')" class="btn btn-secondary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Proxy, ProxyAccountSummary } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'

defineProps<{
  show: boolean
  accountsProxy: Proxy | null
  proxyAccounts: ProxyAccountSummary[]
  accountsLoading: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
</script>
