<script setup lang="ts">
import { computed } from 'vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
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
    <div class="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
      <PlatformIcon :platform="display.iconPlatform" size="sm" />
      <span>{{ display.badge.label }}</span>
    </div>
    <div class="text-xs text-gray-500 dark:text-gray-400">
      <span class="break-all">{{ display.requestPath }}</span>
      <span class="ml-1 inline-flex rounded-full bg-gray-100 px-1.5 py-0.5 dark:bg-dark-700">
        {{ display.modeLabel }}
      </span>
    </div>
  </div>
  <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
</template>
