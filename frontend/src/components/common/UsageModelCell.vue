<template>
  <div class="space-y-1 text-xs">
    <div class="flex items-start gap-2">
      <ModelIcon
        :model="presentation.requested.modelId"
        :provider="presentation.requested.provider"
        :display-name="presentation.requested.displayName"
        size="16px"
      />
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-1.5">
          <div class="break-all font-medium text-gray-900 dark:text-white">
            {{ presentation.requested.primaryText }}
          </div>
          <UsageContextBadge :badge="presentation.requested.badge" />
        </div>
        <div
          v-if="presentation.requested.secondaryText"
          class="break-all text-[11px] text-gray-500 dark:text-gray-400"
        >
          {{ presentation.requested.secondaryText }}
        </div>
      </div>
    </div>

    <div
      v-if="presentation.upstream"
      class="flex items-start gap-2 text-gray-500 dark:text-gray-400"
    >
      <ModelIcon
        :model="presentation.upstream.modelId"
        :provider="presentation.upstream.provider"
        :display-name="presentation.upstream.displayName"
        size="16px"
      />
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="mr-1 shrink-0">-></span>
          <div class="break-all">
            {{ presentation.upstream.primaryText }}
          </div>
          <UsageContextBadge :badge="presentation.upstream.badge" />
        </div>
        <div
          v-if="presentation.upstream.secondaryText"
          class="break-all text-[11px] text-gray-400 dark:text-gray-500"
        >
          {{ presentation.upstream.secondaryText }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import UsageContextBadge from '@/components/common/UsageContextBadge.vue'
import type { UsageLog, UsageModelDisplayMode } from '@/types'
import {
  buildUsageModelPresentation,
  normalizeUsageModelDisplayMode,
} from '@/utils/usageModelPresentation'

const props = defineProps<{
  row: Pick<UsageLog, 'model' | 'upstream_model' | 'million_context_requested' | 'million_context_effective'>
  mode?: UsageModelDisplayMode | null
}>()

const presentation = computed(() =>
  buildUsageModelPresentation(props.row, normalizeUsageModelDisplayMode(props.mode))
)
</script>
