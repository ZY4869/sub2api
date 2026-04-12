<script setup lang="ts">
import { computed } from 'vue'
import ProtocolFamilyBadge from '@/components/common/ProtocolFamilyBadge.vue'
import { resolveProtocolPairDisplay } from '@/utils/protocolDisplay'

const props = withDefaults(defineProps<{
  protocolIn?: string | null
  protocolOut?: string | null
  compact?: boolean
}>(), {
  compact: false
})

const display = computed(() =>
  resolveProtocolPairDisplay(props.protocolIn, props.protocolOut)
)
</script>

<template>
  <div class="space-y-1" :title="display.title">
    <div class="flex flex-wrap items-center gap-1.5">
      <ProtocolFamilyBadge :value="display.protocolIn" />
      <span class="text-xs text-gray-400 dark:text-gray-500">-></span>
      <ProtocolFamilyBadge :value="display.protocolOut" />
    </div>
    <div
      v-if="display.detailLabel || !compact"
      class="text-xs text-gray-500 dark:text-gray-400"
    >
      {{ display.detailLabel || display.label }}
    </div>
  </div>
</template>
