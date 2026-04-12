<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { getProtocolBadgeMeta } from '@/utils/protocolDisplay'

const props = defineProps<{
  value?: string | null
}>()

const badge = computed(() => getProtocolBadgeMeta(props.value))
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium"
    :class="badge.className"
  >
    <Icon :name="badge.iconName" size="xs" class="h-3 w-3" />
    <span>{{ badge.label }}</span>
  </span>
</template>
