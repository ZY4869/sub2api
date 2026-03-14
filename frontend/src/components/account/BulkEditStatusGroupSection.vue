<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import type { AdminGroup } from '@/types'

defineProps<{
  groups: AdminGroup[]
}>()

const enableStatus = defineModel<boolean>('enableStatus', { required: true })
const status = defineModel<'active' | 'inactive'>('status', { required: true })
const enableGroups = defineModel<boolean>('enableGroups', { required: true })
const groupIds = defineModel<number[]>('groupIds', { required: true })

const { t } = useI18n()

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <label
        id="bulk-edit-status-label"
        class="input-label mb-0"
        for="bulk-edit-status-enabled"
      >
        {{ t('common.status') }}
      </label>
      <input
        v-model="enableStatus"
        id="bulk-edit-status-enabled"
        type="checkbox"
        aria-controls="bulk-edit-status"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>
    <div id="bulk-edit-status" :class="!enableStatus && 'pointer-events-none opacity-50'">
      <Select
        v-model="status"
        :options="statusOptions"
        aria-labelledby="bulk-edit-status-label"
      />
    </div>
  </div>

  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <label
        id="bulk-edit-groups-label"
        class="input-label mb-0"
        for="bulk-edit-groups-enabled"
      >
        {{ t('nav.groups') }}
      </label>
      <input
        v-model="enableGroups"
        id="bulk-edit-groups-enabled"
        type="checkbox"
        aria-controls="bulk-edit-groups"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>
    <div id="bulk-edit-groups" :class="!enableGroups && 'pointer-events-none opacity-50'">
      <GroupSelector
        v-model="groupIds"
        :groups="groups"
        aria-labelledby="bulk-edit-groups-label"
      />
    </div>
  </div>
</template>
