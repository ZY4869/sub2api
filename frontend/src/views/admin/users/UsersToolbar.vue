<template>
  <div class="flex flex-wrap items-center gap-3">
    <div class="flex flex-1 flex-wrap items-center gap-3">
      <div class="relative w-full md:w-64">
        <Icon
          name="search"
          size="md"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
        />
        <input
          v-model="searchQueryModel"
          type="text"
          :placeholder="t('admin.users.searchUsers')"
          class="input pl-10"
          @input="emit('search')"
        />
      </div>

      <div v-if="visibleFilters.has('role')" class="w-full sm:w-32">
        <Select
          :model-value="filters.role"
          :options="roleOptions"
          @update:model-value="(value) => updateFilter('role', value)"
        />
      </div>

      <div v-if="visibleFilters.has('status')" class="w-full sm:w-32">
        <Select
          :model-value="filters.status"
          :options="statusOptions"
          @update:model-value="(value) => updateFilter('status', value)"
        />
      </div>

      <div v-if="visibleFilters.has('group')" class="w-full sm:w-44">
        <Select
          :model-value="filters.group"
          :options="groupFilterOptions"
          searchable
          creatable
          :creatable-prefix="t('admin.users.fuzzySearch')"
          :search-placeholder="t('admin.users.searchGroups')"
          @update:model-value="(value) => updateFilter('group', value)"
        />
      </div>

      <div v-if="visibleFilters.has('apiKeyGroup')" class="w-full sm:w-48">
        <Select
          :model-value="filters.apiKeyGroupId"
          :options="apiKeyGroupFilterOptions"
          searchable
          :search-placeholder="t('admin.users.searchApiKeyGroups')"
          @update:model-value="(value) => updateFilter('apiKeyGroupId', value)"
        />
      </div>

      <template v-for="(value, attrId) in activeAttributeFilters" :key="attrId">
        <div
          v-if="visibleFilters.has(`attr_${attrId}`)"
          class="relative w-full sm:w-36"
        >
          <input
            v-if="['text', 'textarea', 'email', 'url', 'date'].includes(getAttributeDefinition(Number(attrId))?.type || 'text')"
            :value="value"
            @input="(e) => emit('update-attribute-filter', Number(attrId), (e.target as HTMLInputElement).value)"
            @keyup.enter="emit('apply-filter')"
            :placeholder="getAttributeDefinitionName(Number(attrId))"
            class="input w-full"
          />
          <input
            v-else-if="getAttributeDefinition(Number(attrId))?.type === 'number'"
            :value="value"
            type="number"
            @input="(e) => emit('update-attribute-filter', Number(attrId), (e.target as HTMLInputElement).value)"
            @keyup.enter="emit('apply-filter')"
            :placeholder="getAttributeDefinitionName(Number(attrId))"
            class="input w-full"
          />
          <div
            v-else-if="['select', 'multi_select'].includes(getAttributeDefinition(Number(attrId))?.type || '')"
            class="w-full"
          >
            <Select
              :model-value="value"
              :options="[
                { value: '', label: getAttributeDefinitionName(Number(attrId)) },
                ...(getAttributeDefinition(Number(attrId))?.options || [])
              ]"
              @update:model-value="(val) => handleSelectAttribute(Number(attrId), val)"
            />
          </div>
          <input
            v-else
            :value="value"
            @input="(e) => emit('update-attribute-filter', Number(attrId), (e.target as HTMLInputElement).value)"
            @keyup.enter="emit('apply-filter')"
            :placeholder="getAttributeDefinitionName(Number(attrId))"
            class="input w-full"
          />
        </div>
      </template>
    </div>

    <div class="flex flex-wrap items-center justify-end gap-2">
      <div class="flex items-center gap-2 md:contents">
        <button
          @click="emit('load')"
          :disabled="loading"
          class="btn btn-secondary px-2 md:px-3"
          :title="t('common.refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
        <div class="relative" data-users-filter-dropdown>
          <button
            @click="emit('update:showFilterDropdown', !showFilterDropdown)"
            class="btn btn-secondary px-2 md:px-3"
            :title="t('admin.users.filterSettings')"
          >
            <Icon name="filter" size="sm" class="md:mr-1.5" />
            <span class="hidden md:inline">{{ t('admin.users.filterSettings') }}</span>
          </button>
          <div
            v-if="showFilterDropdown"
            class="absolute right-0 top-full z-50 mt-1 w-48 rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
          >
            <button
              v-for="filter in builtInFilters"
              :key="filter.key"
              @click="emit('toggle-built-in-filter', filter.key)"
              class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <span>{{ filter.name }}</span>
              <Icon
                v-if="visibleFilters.has(filter.key)"
                name="check"
                size="sm"
                class="text-primary-500"
                :stroke-width="2"
              />
            </button>
            <div
              v-if="filterableAttributes.length > 0"
              class="my-1 border-t border-gray-100 dark:border-dark-700"
            ></div>
            <button
              v-for="attr in filterableAttributes"
              :key="attr.id"
              @click="emit('toggle-attribute-filter', attr)"
              class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <span>{{ attr.name }}</span>
              <Icon
                v-if="visibleFilters.has(`attr_${attr.id}`)"
                name="check"
                size="sm"
                class="text-primary-500"
                :stroke-width="2"
              />
            </button>
          </div>
        </div>
        <div class="relative" data-users-column-dropdown>
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
            class="absolute right-0 top-full z-50 mt-1 max-h-80 w-48 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
          >
            <button
              v-for="col in toggleableColumns"
              :key="col.key"
              @click="emit('toggle-column', col.key)"
              class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            >
              <span>{{ col.label }}</span>
              <Icon
                v-if="isColumnVisible(col.key)"
                name="check"
                size="sm"
                class="text-primary-500"
                :stroke-width="2"
              />
            </button>
          </div>
        </div>
        <button
          @click="emit('open-attributes')"
          class="btn btn-secondary px-2 md:px-3"
          :title="t('admin.users.attributes.configButton')"
        >
          <Icon name="cog" size="sm" class="md:mr-1.5" />
          <span class="hidden md:inline">{{ t('admin.users.attributes.configButton') }}</span>
        </button>
      </div>

      <button
        @click="emit('open-batch-concurrency')"
        class="btn btn-secondary flex-1 md:flex-initial"
      >
        <Icon name="grid" size="md" class="mr-2" />
        {{ t('admin.users.batchConcurrencyAction') }}
      </button>
      <button @click="emit('create')" class="btn btn-primary flex-1 md:flex-initial">
        <Icon name="plus" size="md" class="mr-2" />
        {{ t('admin.users.createUser') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserAttributeDefinition } from '@/types'
import type { Column } from '@/components/common/types'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface UserFilters {
  role: string
  status: string
  group: string
  apiKeyGroupId: string
}

type BuiltInFilter = {
  key: string
  name: string
  type: 'select'
}

const props = defineProps<{
  searchQuery: string
  filters: UserFilters
  visibleFilters: Set<string>
  groupFilterOptions: Array<{ value: string; label: string }>
  apiKeyGroupFilterOptions: Array<{ value: string; label: string }>
  activeAttributeFilters: Record<number, string>
  getAttributeDefinition: (attrId: number) => UserAttributeDefinition | undefined
  getAttributeDefinitionName: (attrId: number) => string
  loading: boolean
  showFilterDropdown: boolean
  showColumnDropdown: boolean
  builtInFilters: BuiltInFilter[]
  filterableAttributes: UserAttributeDefinition[]
  toggleableColumns: Column[]
  isColumnVisible: (key: string) => boolean
}>()

const emit = defineEmits<{
  'update:searchQuery': [value: string]
  'update:showFilterDropdown': [value: boolean]
  'update:showColumnDropdown': [value: boolean]
  'update:filters': [value: UserFilters]
  search: []
  'apply-filter': []
  'update-attribute-filter': [attrId: number, value: string]
  load: []
  'toggle-built-in-filter': [key: string]
  'toggle-attribute-filter': [attr: UserAttributeDefinition]
  'toggle-column': [key: string]
  'open-attributes': []
  'open-batch-concurrency': []
  create: []
}>()

const { t } = useI18n()

const roleOptions = computed(() => [
  { value: '', label: t('admin.users.allRoles') },
  { value: 'admin', label: t('admin.users.admin') },
  { value: 'user', label: t('admin.users.user') }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.users.allStatus') },
  { value: 'active', label: t('common.active') },
  { value: 'disabled', label: t('admin.users.disabled') }
])

const searchQueryModel = computed({
  get: () => props.searchQuery,
  set: (value: string) => emit('update:searchQuery', value)
})

const handleSelectAttribute = (attrId: number, value: unknown) => {
  emit('update-attribute-filter', attrId, String(value ?? ''))
  emit('apply-filter')
}

const updateFilter = (key: keyof UserFilters, value: unknown) => {
  emit('update:filters', { ...props.filters, [key]: String(value ?? '') })
  emit('apply-filter')
}
</script>
