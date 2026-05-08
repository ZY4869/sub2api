<template>
  <span
    v-if="badge"
    :class="badgeClass"
    :title="badgeTitle"
  >
    <span class="mr-1 inline-flex h-3.5 w-3.5 items-center justify-center">
      <svg v-if="badge.tier === '1m'" viewBox="0 0 24 24" class="h-3.5 w-3.5 fill-current">
        <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
      </svg>
      <svg v-else-if="badge.tier === '2m'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <path d="m12 3-1.9 5.8a2 2 0 0 1-1.3 1.3L3 12l5.8 1.9a2 2 0 0 1 1.3 1.3L12 21l1.9-5.8a2 2 0 0 1 1.3-1.3L21 12l-5.8-1.9a2 2 0 0 1-1.3-1.3L12 3Z" />
      </svg>
      <svg v-else-if="badge.tier === '10m'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 12c-2-2.67-4-4-6-4a4 4 0 1 0 0 8c2 0 4-1.33 6-4Zm0 0c2 2.67 4 4 6 4a4 4 0 1 0 0-8c-2 0-4 1.33-6 4Z" />
      </svg>
      <svg v-else-if="badge.tier === '512k'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="16" y="16" width="6" height="6" rx="1" />
        <rect x="2" y="16" width="6" height="6" rx="1" />
        <rect x="9" y="2" width="6" height="6" rx="1" />
        <path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3" />
        <path d="M12 12V8" />
      </svg>
      <svg v-else-if="badge.tier === '200k'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
      </svg>
      <svg v-else-if="badge.tier === '128k'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <ellipse cx="12" cy="5" rx="9" ry="3" />
        <path d="M3 5V19A9 3 0 0 0 21 19V5" />
        <path d="M3 12A9 3 0 0 0 21 12" />
      </svg>
      <svg v-else-if="badge.tier === '64k'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <path d="m12 14 4-4" />
        <path d="M3.34 19a10 10 0 1 1 17.32 0" />
      </svg>
      <svg v-else-if="badge.tier === '32k'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" />
        <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" />
      </svg>
      <svg v-else-if="badge.tier === '16k'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="12 2 2 7 12 12 22 7 12 2" />
        <polyline points="2 17 12 22 22 17" />
        <polyline points="2 12 12 17 22 12" />
      </svg>
      <svg v-else-if="badge.tier === '8k'" viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="4 17 10 11 4 5" />
        <line x1="12" x2="20" y1="19" y2="19" />
      </svg>
      <svg v-else viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="4" y="4" width="16" height="16" rx="2" ry="2" />
        <rect x="9" y="9" width="6" height="6" />
      </svg>
    </span>
    {{ badgeLabel }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageContextBadgeInfo } from '@/utils/usageModelPresentation'

const props = defineProps<{
  badge: UsageContextBadgeInfo | null
}>()
const { t } = useI18n()

const badgeLabel = computed(() => {
  if (!props.badge) return ''
  return props.badge.labelKey ? t(props.badge.labelKey) : props.badge.label
})

const badgeTitle = computed(() => {
  if (!props.badge) return ''
  if (props.badge.titleKey) {
    return t(props.badge.titleKey, props.badge.titleParams || {})
  }
  return props.badge.title || badgeLabel.value
})

const badgeClass = computed(() => {
  if (!props.badge) return ''
  const muted = props.badge.muted
  const byTier: Record<UsageContextBadgeInfo['tier'], string> = {
    '4k': 'bg-slate-800 text-slate-400 border-slate-700',
    '8k': 'bg-slate-800 text-slate-200 border-slate-600',
    '16k': 'bg-slate-900 text-blue-400 border-blue-500/40',
    '32k': 'bg-slate-900 text-orange-400 border-orange-500/40',
    '64k': 'bg-slate-900 text-yellow-400 border-yellow-500/40',
    '128k': 'bg-slate-900 text-emerald-400 border-emerald-500/40',
    '200k': 'bg-slate-900 text-cyan-400 border-cyan-500/40',
    '512k': 'bg-slate-900 text-rose-400 border-rose-500/40',
    '1m': 'bg-slate-900 text-amber-400 border-amber-500/50 drop-shadow-[0_0_5px_rgba(251,191,36,0.4)]',
    '2m': 'bg-slate-900 text-purple-400 border-purple-500/50 drop-shadow-[0_0_5px_rgba(192,132,252,0.4)]',
    '10m': 'bg-slate-900 text-fuchsia-400 border-fuchsia-500/50 drop-shadow-[0_0_6px_rgba(232,121,249,0.5)]',
  }
  return [
    'inline-flex items-center rounded-full border px-2.5 py-0.5 text-[11px] font-black tracking-wide shadow-sm',
    byTier[props.badge.tier],
    muted ? 'opacity-60 saturate-75' : ''
  ].join(' ')
})
</script>
