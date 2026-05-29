<template>
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div class="flex flex-1 flex-wrap items-center gap-3">
      <div class="relative w-full sm:w-64" data-filter-user-search>
        <Icon
          name="search"
          size="md"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
        />
        <input
          v-model="filterUserKeywordModel"
          type="text"
          :placeholder="t('admin.users.searchUsers')"
          class="input pl-10 pr-8"
          @input="emit('search-filter-users')"
          @focus="emit('update:showFilterUserDropdown', true)"
        />
        <button
          v-if="selectedFilterUser"
          @click="emit('clear-filter-user')"
          type="button"
          class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          :title="t('common.clear')"
        >
          <Icon name="x" size="sm" :stroke-width="2" />
        </button>

        <div
          v-if="showFilterUserDropdown && (filterUserResults.length > 0 || filterUserKeyword)"
          class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
        >
          <div
            v-if="filterUserLoading"
            class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
          >
            {{ t('common.loading') }}
          </div>
          <div
            v-else-if="filterUserResults.length === 0 && filterUserKeyword"
            class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
          >
            {{ t('common.noOptionsFound') }}
          </div>
          <button
            v-for="user in filterUserResults"
            :key="user.id"
            type="button"
            @click="emit('select-filter-user', user)"
            class="w-full px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <span class="font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
            <span class="ml-2 text-gray-500 dark:text-gray-400">#{{ user.id }}</span>
          </button>
        </div>
      </div>

      <div class="w-full sm:w-40">
        <Select
          :model-value="filters.status"
          :options="statusOptions"
          :placeholder="t('admin.subscriptions.allStatus')"
          @update:model-value="(value) => updateFilter('status', value)"
        />
      </div>
      <div class="w-full sm:w-48">
        <Select
          :model-value="filters.group_id"
          :options="groupOptions"
          :placeholder="t('admin.subscriptions.allGroups')"
          @update:model-value="(value) => updateFilter('group_id', value)"
        />
      </div>
      <div class="w-full sm:w-40">
        <Select
          :model-value="filters.platform"
          :options="platformFilterOptions"
          :placeholder="t('admin.subscriptions.allPlatforms')"
          @update:model-value="(value) => updateFilter('platform', value)"
        />
      </div>
    </div>

    <div class="ml-auto flex flex-wrap items-center justify-end gap-3">
      <button
        @click="emit('load')"
        :disabled="loading"
        class="btn btn-secondary"
        :title="t('common.refresh')"
      >
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
      </button>
      <div class="relative" data-subscriptions-column-dropdown>
        <button
          @click="emit('update:showColumnDropdown', !showColumnDropdown)"
          class="btn btn-secondary px-2 md:px-3"
          :title="t('admin.users.columnSettings')"
        >
          <svg class="h-4 w-4 md:mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          <span class="hidden md:inline">{{ t('admin.users.columnSettings') }}</span>
        </button>
        <div
          v-if="showColumnDropdown"
          class="absolute right-0 z-50 mt-2 w-48 origin-top-right rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
        >
          <div class="p-2">
            <div class="mb-2 border-b border-gray-200 pb-2 dark:border-gray-700">
              <div class="px-3 py-1 text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.subscriptions.columns.user') }}
              </div>
              <button
                @click="emit('set-user-column-mode', 'email')"
                class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
              >
                <span>{{ t('admin.users.columns.email') }}</span>
                <Icon v-if="userColumnMode === 'email'" name="check" size="sm" class="text-primary-500" />
              </button>
              <button
                @click="emit('set-user-column-mode', 'username')"
                class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
              >
                <span>{{ t('admin.users.columns.username') }}</span>
                <Icon v-if="userColumnMode === 'username'" name="check" size="sm" class="text-primary-500" />
              </button>
            </div>
            <button
              v-for="col in toggleableColumns"
              :key="col.key"
              @click="emit('toggle-column', col.key)"
              class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            >
              <span>{{ col.label }}</span>
              <Icon v-if="isColumnVisible(col.key)" name="check" size="sm" class="text-primary-500" />
            </button>
          </div>
        </div>
      </div>
      <button
        @click="emit('show-guide')"
        class="btn btn-secondary"
        :title="t('admin.subscriptions.guide.showGuide')"
      >
        <Icon name="questionCircle" size="md" />
      </button>
      <button @click="emit('assign')" class="btn btn-primary">
        <Icon name="plus" size="md" class="mr-2" />
        {{ t('admin.subscriptions.assignSubscription') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SimpleUser } from '@/api/admin/usage'
import type { Column } from '@/components/common/types'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface SubscriptionFilters {
  status: string
  group_id: string
  platform: string
  user_id: number | null
}

const props = defineProps<{
  filterUserKeyword: string
  filterUserResults: SimpleUser[]
  filterUserLoading: boolean
  showFilterUserDropdown: boolean
  selectedFilterUser: SimpleUser | null
  filters: SubscriptionFilters
  statusOptions: Array<{ value: string; label: string }>
  groupOptions: Array<{ value: string | number; label: string }>
  platformFilterOptions: Array<{ value: string; label: string }>
  loading: boolean
  showColumnDropdown: boolean
  userColumnMode: 'email' | 'username'
  toggleableColumns: Column[]
  isColumnVisible: (key: string) => boolean
}>()

const emit = defineEmits<{
  'update:filterUserKeyword': [value: string]
  'update:showFilterUserDropdown': [value: boolean]
  'update:showColumnDropdown': [value: boolean]
  'update:filters': [value: SubscriptionFilters]
  'search-filter-users': []
  'select-filter-user': [user: SimpleUser]
  'clear-filter-user': []
  'apply-filters': []
  load: []
  'set-user-column-mode': [mode: 'email' | 'username']
  'toggle-column': [key: string]
  'show-guide': []
  assign: []
}>()

const { t } = useI18n()

const filterUserKeywordModel = computed({
  get: () => props.filterUserKeyword,
  set: (value: string) => emit('update:filterUserKeyword', value)
})

const updateFilter = (key: keyof SubscriptionFilters, value: unknown) => {
  emit('update:filters', { ...props.filters, [key]: String(value ?? '') })
  emit('apply-filters')
}
</script>
