<template>
  <div data-test="account-status-limit-badge" class="group relative inline-flex min-w-0 max-w-full">
    <span
      :class="[
        badgeBaseClass,
        toneClass,
      ]"
    >
      <ModelIcon
        v-if="model"
        :model="model"
        :display-name="modelDisplayName || label"
        size="12px"
      />
      <Icon v-else name="exclamationTriangle" size="xs" :stroke-width="2" />
      <span class="min-w-0 truncate">{{ label }}</span>
      <span v-if="countdown" class="shrink-0 text-[10px] opacity-70">{{ countdown }}</span>
    </span>
    <div
      v-if="tooltip"
      :class="tooltipClass"
    >
      {{ tooltip }}
      <div
        :class="tooltipArrowClass"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(
  defineProps<{
    tone: 'purple' | 'amber' | 'red'
    label: string
    countdown?: string | null
    tooltip?: string | null
    model?: string | null
    modelDisplayName?: string | null
    visualVariant?: 'default' | 'glass'
  }>(),
  {
    countdown: null,
    tooltip: null,
    model: null,
    modelDisplayName: null,
    visualVariant: 'default',
  },
)

const badgeBaseClass = computed(() => {
  if (props.visualVariant === 'glass') {
    return 'inline-flex w-full min-w-0 items-center justify-between gap-2 rounded-full border px-2.5 py-1.5 text-[10px] font-semibold tracking-tight'
  }
  return 'inline-flex max-w-full items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium'
})

const toneClass = computed(() => {
  if (props.visualVariant === 'glass') {
    switch (props.tone) {
      case 'amber':
        return 'border-amber-200/75 bg-amber-50 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-100'
      case 'red':
        return 'border-rose-200/75 bg-rose-50 text-rose-700 dark:border-rose-400/20 dark:bg-rose-400/10 dark:text-rose-100'
      default:
        return 'border-indigo-200/75 bg-indigo-50 text-indigo-700 dark:border-indigo-400/20 dark:bg-indigo-400/10 dark:text-indigo-100'
    }
  }
  switch (props.tone) {
    case 'amber':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'red':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
  }
})

const tooltipClass = computed(() => {
  if (props.visualVariant === 'glass') {
    return 'pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-56 -translate-x-1/2 whitespace-normal rounded-2xl border border-slate-200/80 bg-white px-3 py-2 text-center text-[11px] leading-relaxed text-slate-700 opacity-0 ring-1 ring-slate-200/70 transition-opacity group-hover:opacity-100 dark:border-slate-700/80 dark:bg-slate-900 dark:text-slate-100 dark:ring-slate-700/70'
  }
  return 'pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-56 -translate-x-1/2 whitespace-normal rounded bg-gray-900 px-3 py-2 text-center text-xs leading-relaxed text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700'
})

const tooltipArrowClass = computed(() => {
  if (props.visualVariant === 'glass') {
    return 'absolute left-1/2 top-full h-2 w-2 -translate-x-1/2 rotate-45 border-b border-r border-slate-200/70 bg-white dark:border-slate-700/80 dark:bg-slate-900'
  }
  return 'absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-gray-700'
})
</script>
