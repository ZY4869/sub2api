<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import GroupSelector from '@/components/common/GroupSelector.vue'
import type { AdminGroup, GroupPlatform } from '@/types'

const props = withDefaults(defineProps<{
  groups: AdminGroup[]
  platform: GroupPlatform | null
  simpleMode: boolean
  showMixedScheduling?: boolean
  mixedSchedulingReadonly?: boolean
}>(), {
  showMixedScheduling: false,
  mixedSchedulingReadonly: false
})

const groupIds = defineModel<number[]>('groupIds', { required: true })
const mixedScheduling = defineModel<boolean>('mixedScheduling', { required: true })

const { t } = useI18n()

const mixedSchedulingLabelClass = computed(() =>
  props.mixedSchedulingReadonly
    ? 'flex cursor-not-allowed items-center gap-2 opacity-60'
    : 'flex cursor-pointer items-center gap-2'
)

const mixedSchedulingInputClass = computed(() =>
  props.mixedSchedulingReadonly
    ? 'h-4 w-4 cursor-not-allowed rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500'
    : 'h-4 w-4 rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500'
)

const groupPlatform = computed(() => props.platform || undefined)
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div v-if="showMixedScheduling" class="flex items-center gap-2">
      <label :class="mixedSchedulingLabelClass">
        <input
          v-model="mixedScheduling"
          type="checkbox"
          :disabled="mixedSchedulingReadonly"
          :class="mixedSchedulingInputClass"
        />
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.mixedScheduling') }}
        </span>
      </label>
      <div class="group relative">
        <span
          class="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full bg-gray-200 text-xs text-gray-500 hover:bg-gray-300 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500"
        >
          ?
        </span>
        <div
          class="pointer-events-none absolute left-0 top-full z-[100] mt-1.5 w-72 rounded bg-gray-900 px-3 py-2 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
        >
          {{ t('admin.accounts.mixedSchedulingTooltip') }}
          <div
            class="absolute bottom-full left-3 border-4 border-transparent border-b-gray-900 dark:border-b-gray-700"
          ></div>
        </div>
      </div>
    </div>

    <GroupSelector
      v-if="!simpleMode"
      v-model="groupIds"
      :groups="groups"
      :platform="groupPlatform"
      :mixed-scheduling="mixedScheduling"
      data-tour="account-form-groups"
    />
  </div>
</template>
