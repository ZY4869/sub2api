<template>
  <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
    <div class="flex flex-1 flex-wrap items-center gap-3">
      <div class="relative w-full sm:w-64">
        <Icon
          name="search"
          size="md"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
        />
        <input
          :value="searchQuery"
          type="text"
          :placeholder="t('admin.channels.searchChannels', 'Search channels...')"
          class="input pl-10"
          @input="emit('search-change', ($event.target as HTMLInputElement).value)"
        />
      </div>

      <Select
        :model-value="status"
        :options="statusFilterOptions"
        :placeholder="t('admin.channels.allStatus', 'All Status')"
        class="w-40"
        @change="emit('status-change', $event as ChannelStatus | '')"
      />
    </div>

    <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
      <button
        @click="emit('refresh')"
        :disabled="loading"
        class="btn btn-secondary"
        :title="t('common.refresh', 'Refresh')"
      >
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
      </button>
      <button @click="emit('create')" class="btn btn-primary">
        <Icon name="plus" size="md" class="mr-2" />
        {{ t('admin.channels.createChannel', 'Create Channel') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Channel } from '@/api/admin/channels'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

type ChannelStatus = Channel['status']

defineProps<{
  searchQuery: string
  status: ChannelStatus | ''
  statusFilterOptions: Array<{ value: ChannelStatus | ''; label: string }>
  loading: boolean
}>()

const emit = defineEmits<{
  'search-change': [value: string]
  'status-change': [value: ChannelStatus | '']
  refresh: []
  create: []
}>()

const { t } = useI18n()
</script>
