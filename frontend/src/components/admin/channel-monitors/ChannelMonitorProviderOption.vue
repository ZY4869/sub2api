<template>
  <span v-if="option" class="flex min-w-0 items-center gap-2">
    <component :is="option.icon" v-if="option.icon" v-bind="option.iconProps || {}" size="sm" />
    <span class="truncate">{{ option.label }}</span>
  </span>
  <span v-else>{{ t('common.selectOption') }}</span>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SelectOptionItem } from '@/utils/channelMonitorPresentation'

defineProps<{
  option?: SelectOptionItem | Record<string, unknown>
}>()

const { t } = useI18n()
</script>
