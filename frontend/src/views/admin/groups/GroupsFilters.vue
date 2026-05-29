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
          v-model="searchQuery"
          type="text"
          :placeholder="t('admin.groups.searchGroups')"
          class="input pl-10"
          @input="handleSearch"
        />
      </div>
          <Select
            v-model="filters.platform"
            :options="platformFilterOptions"
            :placeholder="t('admin.groups.allPlatforms')"
            class="w-44"
            @change="loadGroups"
          >
            <template #selected="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="selectOption(option).platform"
                :label="selectOption(option).label"
              />
            </template>
            <template #option="{ option }">
              <PlatformLabel
                v-if="isPlatformSelectOption(option)"
                :platform="selectOption(option).platform"
                :label="selectOption(option).label"
              />
            </template>
          </Select>
          <Select
            v-model="filters.status"
            :options="statusOptions"
            :placeholder="t('admin.groups.allStatus')"
            class="w-40"
            @change="loadGroups"
          />
          <Select
            v-model="filters.is_exclusive"
            :options="exclusiveOptions"
            :placeholder="t('admin.groups.allGroups')"
            class="w-44"
            @change="loadGroups"
          />
    </div>

    <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
      <button
        @click="loadGroups"
        :disabled="loading"
        class="btn btn-secondary"
        :title="t('common.refresh')"
      >
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
      </button>
      <button
        @click="openSortModal"
        class="btn btn-secondary"
        :title="t('admin.groups.sortOrder')"
      >
        <Icon name="arrowsUpDown" size="md" class="mr-2" />
        {{ t('admin.groups.sortOrder') }}
      </button>
      <button
        @click="showCreateModal = true"
        class="btn btn-primary"
        data-tour="groups-create-btn"
      >
        <Icon name="plus" size="md" class="mr-2" />
        {{ t('admin.groups.createGroup') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import PlatformLabel from '@/components/common/PlatformLabel.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ ctx: any }>()
const {
  t,
  searchQuery,
  filters,
  platformFilterOptions,
  statusOptions,
  exclusiveOptions,
  isPlatformSelectOption,
  loadGroups,
  loading,
  handleSearch,
  openSortModal,
  showCreateModal
} = props.ctx

const selectOption = (option: unknown) => (option ?? {}) as Record<string, any>
</script>
