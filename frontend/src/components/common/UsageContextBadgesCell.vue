<template>
  <div
    v-if="badges.length"
    data-test="usage-context-badges-cell"
    class="flex min-h-8"
    :class="
      badges.length > 1
        ? 'flex-col items-start gap-1'
        : 'items-center gap-1'
    "
  >
    <UsageContextBadge
      v-for="badge in badges"
      :key="buildBadgeKey(badge)"
      :badge="badge"
    />
  </div>
  <span
    v-else
    data-test="usage-context-badges-empty"
    class="text-sm text-gray-400 dark:text-gray-500"
  >
    -
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import UsageContextBadge from '@/components/common/UsageContextBadge.vue'
import type { UsageContextBadgeDisplayMode, UsageLog } from '@/types'
import {
  buildUsageModelPresentation,
  normalizeUsageContextBadgeDisplayMode,
  resolveUsageContextBadges,
  type UsageContextBadgeInfo,
} from '@/utils/usageModelPresentation'

const props = defineProps<{
  row: Pick<UsageLog, 'model' | 'upstream_model' | 'million_context_requested' | 'million_context_effective'>
  mode?: UsageContextBadgeDisplayMode | null
}>()

const badgeMode = computed(() =>
  normalizeUsageContextBadgeDisplayMode(props.mode),
)

const badges = computed(() => {
  const presentation = buildUsageModelPresentation(props.row, 'model_only')
  return resolveUsageContextBadges(presentation.requested, badgeMode.value)
})

const buildBadgeKey = (badge: UsageContextBadgeInfo) =>
  `${badge.labelKey || badge.label}-${badge.tier}-${badge.tokens}-${badge.muted ? 'muted' : 'solid'}`
</script>
