<template>
  <span v-if="option" class="flex min-w-0 items-center gap-2">
    <ModelIcon
      :model="modelId"
      :provider="option.provider || ''"
      :display-name="option.display_name || modelId"
      size="14px"
    />
    <span class="min-w-0 truncate font-mono text-xs">{{ option.label || modelId }}</span>
    <span v-if="option.provider_label" class="shrink-0 text-xs text-gray-400">{{ option.provider_label }}</span>
  </span>
  <span v-else>{{ t('common.selectOption') }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'

interface MonitorModelSelectOption {
  value?: string | number | boolean | null
  id?: string
  label?: string
  provider?: string
  provider_label?: string
  display_name?: string
}

const props = defineProps<{
  option?: MonitorModelSelectOption | null
}>()

const { t } = useI18n()

const modelId = computed(() => String(props.option?.value || props.option?.id || ''))
</script>
