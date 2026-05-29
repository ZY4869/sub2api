<template>
  <div class="flex flex-wrap items-center gap-3">
    <div class="relative w-full sm:w-64">
      <Icon
        name="search"
        size="md"
        class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
      />
      <input
        v-model="searchQueryModel"
        type="text"
        :placeholder="t('admin.proxies.searchProxies')"
        class="input pl-10"
        @input="emit('search')"
      />
    </div>

    <div class="w-full sm:w-40">
      <Select
        :model-value="filters.protocol"
        :options="protocolOptions"
        :placeholder="t('admin.proxies.allProtocols')"
        @update:model-value="(value) => updateFilter('protocol', value)"
      />
    </div>
    <div class="w-full sm:w-36">
      <Select
        :model-value="filters.status"
        :options="statusOptions"
        :placeholder="t('admin.proxies.allStatus')"
        @update:model-value="(value) => updateFilter('status', value)"
      />
    </div>

    <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
      <button
        @click="emit('load')"
        :disabled="loading"
        class="btn btn-secondary"
        :title="t('common.refresh')"
      >
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
      </button>
      <button
        @click="emit('batch-test')"
        :disabled="batchTesting || loading"
        class="btn btn-secondary"
        :title="t('admin.proxies.testConnection')"
      >
        <Icon name="play" size="md" class="mr-2" />
        {{ t('admin.proxies.testConnection') }}
      </button>
      <button
        @click="emit('batch-quality-check')"
        :disabled="batchQualityChecking || loading"
        class="btn btn-secondary"
        :title="t('admin.proxies.batchQualityCheck')"
      >
        <Icon name="shield" size="md" class="mr-2" :class="batchQualityChecking ? 'animate-pulse' : ''" />
        {{ t('admin.proxies.batchQualityCheck') }}
      </button>
      <button
        @click="emit('batch-delete')"
        :disabled="selectedCount === 0"
        class="btn btn-danger"
        :title="t('admin.proxies.batchDeleteAction')"
      >
        <Icon name="trash" size="md" class="mr-2" />
        {{ t('admin.proxies.batchDeleteAction') }}
      </button>
      <button @click="emit('import-data')" class="btn btn-secondary">
        {{ t('admin.proxies.dataImport') }}
      </button>
      <button @click="emit('export-data')" class="btn btn-secondary">
        {{ selectedCount > 0 ? t('admin.proxies.dataExportSelected') : t('admin.proxies.dataExport') }}
      </button>
      <button @click="emit('create')" class="btn btn-primary">
        <Icon name="plus" size="md" class="mr-2" />
        {{ t('admin.proxies.createProxy') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface ProxyFilters {
  protocol: string
  status: string
}

const props = defineProps<{
  searchQuery: string
  filters: ProxyFilters
  protocolOptions: Array<{ value: string; label: string }>
  statusOptions: Array<{ value: string; label: string }>
  loading: boolean
  batchTesting: boolean
  batchQualityChecking: boolean
  selectedCount: number
}>()

const emit = defineEmits<{
  'update:searchQuery': [value: string]
  'update:filters': [value: ProxyFilters]
  search: []
  load: []
  'batch-test': []
  'batch-quality-check': []
  'batch-delete': []
  'import-data': []
  'export-data': []
  create: []
}>()

const { t } = useI18n()

const searchQueryModel = computed({
  get: () => props.searchQuery,
  set: (value: string) => emit('update:searchQuery', value)
})

const updateFilter = (key: keyof ProxyFilters, value: unknown) => {
  emit('update:filters', { ...props.filters, [key]: String(value ?? '') })
  emit('load')
}
</script>
