<script setup lang="ts">
import { computed } from 'vue'
import ProtocolFamilyBadge from '@/components/common/ProtocolFamilyBadge.vue'
import { resolveUsageProtocolDisplay } from '@/utils/protocolDisplay'

const props = defineProps<{
  inboundPath?: string | null
  upstreamPath?: string | null
}>()

const display = computed(() =>
  resolveUsageProtocolDisplay(props.inboundPath, props.upstreamPath)
)

const titleText = computed(() => display.value?.tooltip || display.value?.pathTitle || '')
</script>

<template>
  <div v-if="display" class="space-y-1" :title="titleText">
    <ProtocolFamilyBadge :value="display.requestPath" />
    <div class="text-xs text-gray-500 dark:text-gray-400">
      <span class="break-all">{{ display.requestPath }}</span>
      <span class="ml-1">{{ display.modeLabel }}</span>
    </div>
  </div>
  <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
</template>
