<template>
  <div :class="['flex min-w-0 items-center', compact ? 'gap-2' : 'gap-3']">
    <div
      :class="[
        'relative flex shrink-0 items-center justify-center border border-white/80 shadow-sm',
        compact ? 'h-9 w-9 rounded-xl' : 'h-10 w-10 rounded-2xl',
        avatarClass
      ]"
    >
      <PlatformIcon
        v-if="resolvedPlatform"
        :platform="resolvedPlatform"
        size="lg"
      />
      <span
        v-else
        class="text-[11px] font-black uppercase tracking-wide"
      >
        {{ fallbackPlatformLabel }}
      </span>
      <span
        v-if="showHealthyDot"
        class="absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-white bg-emerald-500"
      ></span>
      <AccountErrorTooltipButton
        v-else-if="showErrorDot"
        :message="errorDotMessage"
        :ariaLabel="t('admin.accounts.status.viewIssueDetails')"
        wrapper-class="absolute -bottom-0.5 -right-0.5"
        button-class="relative flex h-3.5 w-3.5 items-center justify-center rounded-full border-2 border-white bg-rose-500 text-white shadow-sm transition hover:bg-rose-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400/70"
        icon-name="exclamationCircle"
        icon-size="xs"
      >
        <span class="h-full w-full animate-ping rounded-full bg-rose-500 opacity-60"></span>
      </AccountErrorTooltipButton>
    </div>

    <div class="flex min-w-0 flex-1 flex-col justify-center">
      <div class="flex min-w-0 items-center gap-1.5">
        <span
          class="truncate text-[13px] font-extrabold leading-tight tracking-tight"
          :class="nameClass"
          :title="account.name"
        >
          {{ account.name }}
        </span>
        <span
          v-if="showProbeIndicator"
          class="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-full border"
          :class="probeBadgeClass"
          :title="probeBadgeTitle"
          :aria-label="probeBadgeTitle"
        >
          <svg
            v-if="probeStatus === 'retry_scheduled'"
            class="h-2.5 w-2.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M12 6v6l4 2M21 12a9 9 0 11-3.2-6.9"
            />
          </svg>
          <svg
            v-else-if="probeStatus === 'blacklisted'"
            class="h-2.5 w-2.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M18.364 5.636l-12.728 12.728M8 8h.01M16 16h.01M6.343 17.657A8 8 0 1117.657 6.343 8 8 0 016.343 17.657z"
            />
          </svg>
        </span>
        <span
          v-if="showSuccessIndicator"
          class="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-full border border-emerald-200/50 bg-emerald-100/80 text-emerald-600"
          :title="t('admin.accounts.autoRecoveryProbe.successIndicator')"
          :aria-label="t('admin.accounts.autoRecoveryProbe.successIndicator')"
        >
          <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        </span>
      </div>

      <div class="mt-1 flex min-w-0 items-center gap-1.5">
        <span
          class="min-w-0 truncate font-mono text-[10.5px] font-medium leading-none"
          :class="metaClass"
          :title="accountIdentity"
        >
          {{ accountIdentity }}
        </span>
        <span class="h-[3px] w-[3px] shrink-0 rounded-full" :class="dotClass"></span>
        <span
          class="inline-flex max-w-[86px] shrink-0 items-center rounded border px-[5px] py-[1.5px] text-[8.5px] font-black leading-none tracking-wider"
          :class="badgeClass"
          :title="platformBadgeLabel"
        >
          <span class="truncate">{{ platformBadgeLabel }}</span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import AccountErrorTooltipButton from '@/components/account/AccountErrorTooltipButton.vue'
import { resolveEffectiveAccountPlatformFromAccount } from '@/utils/accountProtocolGateway'
import { getPlatformEnglishName } from '@/utils/platformBranding'

const props = defineProps<{
  account: Account
  compact?: boolean
}>()

const { t } = useI18n()

const platformThemeMap: Record<string, {
  avatar: string
  name: string
  meta: string
  dot: string
  badge: string
}> = {
  openai: {
    avatar: 'bg-emerald-50 text-emerald-600',
    name: 'text-emerald-800 dark:text-emerald-100',
    meta: 'text-emerald-600/90 dark:text-emerald-200/90',
    dot: 'bg-emerald-300 dark:bg-emerald-400/70',
    badge: 'border-emerald-200 bg-emerald-100/80 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-100'
  },
  anthropic: {
    avatar: 'bg-amber-50 text-amber-600',
    name: 'text-amber-900 dark:text-amber-100',
    meta: 'text-amber-600/90 dark:text-amber-200/90',
    dot: 'bg-amber-300 dark:bg-amber-400/70',
    badge: 'border-amber-200 bg-amber-100/80 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-100'
  },
  gemini: {
    avatar: 'bg-blue-50 text-blue-600',
    name: 'text-blue-800 dark:text-blue-100',
    meta: 'text-blue-600/90 dark:text-blue-200/90',
    dot: 'bg-blue-300 dark:bg-blue-400/70',
    badge: 'border-blue-200 bg-blue-100/80 text-blue-700 dark:border-blue-400/20 dark:bg-blue-400/10 dark:text-blue-100'
  },
  grok: {
    avatar: 'bg-zinc-100 text-zinc-700',
    name: 'text-zinc-800 dark:text-zinc-100',
    meta: 'text-zinc-500/90 dark:text-zinc-300/90',
    dot: 'bg-zinc-300 dark:bg-zinc-400/70',
    badge: 'border-zinc-300 bg-zinc-200/80 text-zinc-700 dark:border-zinc-500/30 dark:bg-zinc-500/10 dark:text-zinc-100'
  },
  antigravity: {
    avatar: 'bg-violet-50 text-violet-600',
    name: 'text-violet-800 dark:text-violet-100',
    meta: 'text-violet-600/90 dark:text-violet-200/90',
    dot: 'bg-violet-300 dark:bg-violet-400/70',
    badge: 'border-violet-200 bg-violet-100/80 text-violet-700 dark:border-violet-400/20 dark:bg-violet-400/10 dark:text-violet-100'
  },
  deepseek: {
    avatar: 'bg-cyan-50 text-cyan-600',
    name: 'text-cyan-800 dark:text-cyan-100',
    meta: 'text-cyan-600/90 dark:text-cyan-200/90',
    dot: 'bg-cyan-300 dark:bg-cyan-400/70',
    badge: 'border-cyan-200 bg-cyan-100/80 text-cyan-700 dark:border-cyan-400/20 dark:bg-cyan-400/10 dark:text-cyan-100'
  },
  kiro: {
    avatar: 'bg-fuchsia-50 text-fuchsia-600',
    name: 'text-fuchsia-800 dark:text-fuchsia-100',
    meta: 'text-fuchsia-600/90 dark:text-fuchsia-200/90',
    dot: 'bg-fuchsia-300 dark:bg-fuchsia-400/70',
    badge: 'border-fuchsia-200 bg-fuchsia-100/80 text-fuchsia-700 dark:border-fuchsia-400/20 dark:bg-fuchsia-400/10 dark:text-fuchsia-100'
  },
  baidu_document_ai: {
    avatar: 'bg-indigo-50 text-indigo-600',
    name: 'text-indigo-800 dark:text-indigo-100',
    meta: 'text-indigo-600/90 dark:text-indigo-200/90',
    dot: 'bg-indigo-300 dark:bg-indigo-400/70',
    badge: 'border-indigo-200 bg-indigo-100/80 text-indigo-700 dark:border-indigo-400/20 dark:bg-indigo-400/10 dark:text-indigo-100'
  },
  protocol_gateway: {
    avatar: 'bg-sky-50 text-sky-600',
    name: 'text-sky-800 dark:text-sky-100',
    meta: 'text-sky-600/90 dark:text-sky-200/90',
    dot: 'bg-sky-300 dark:bg-sky-400/70',
    badge: 'border-sky-200 bg-sky-100/80 text-sky-700 dark:border-sky-400/20 dark:bg-sky-400/10 dark:text-sky-100'
  },
}

const resolvedPlatform = computed(() => {
  return resolveEffectiveAccountPlatformFromAccount(props.account) || props.account.platform || undefined
})

const platformTheme = computed(() => {
  const key = String(resolvedPlatform.value || '').trim().toLowerCase()
  return platformThemeMap[key] || {
    avatar: 'bg-slate-50 text-slate-600',
    name: 'text-slate-800 dark:text-slate-100',
    meta: 'text-slate-500/90 dark:text-slate-300/90',
    dot: 'bg-slate-300 dark:bg-slate-400/70',
    badge: 'border-slate-200 bg-slate-100 text-slate-700 dark:border-slate-500/30 dark:bg-slate-500/10 dark:text-slate-100'
  }
})

const hasRestoredFromBlacklisted = computed(() => {
  const lifecycleState = String(props.account.lifecycle_state || '').trim().toLowerCase()
  if (!lifecycleState || lifecycleState === 'blacklisted') {
    return false
  }
  return props.account.auto_recovery_probe?.blacklisted || props.account.auto_recovery_probe?.status === 'blacklisted'
})

const probeStatus = computed(() => {
  const status = props.account.auto_recovery_probe?.status
  if (status === 'success') return 'success'
  if (hasRestoredFromBlacklisted.value) return null
  if (status === 'retry_scheduled') return 'retry_scheduled'
  if (status === 'blacklisted') return 'blacklisted'
  return null
})

const showProbeIndicator = computed(() => Boolean(probeStatus.value && probeStatus.value !== 'success'))

const probeBadgeTitle = computed(() => {
  if (!probeStatus.value || probeStatus.value === 'success') {
    return ''
  }
  return t('admin.accounts.autoRecoveryProbe.headline', {
    status: t(`admin.accounts.autoRecoveryProbe.statuses.${probeStatus.value}`)
  })
})

const probeBadgeClass = computed(() => {
  if (probeStatus.value === 'blacklisted') {
    return 'border-red-200/80 bg-red-50/90 text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200'
  }
  if (probeStatus.value === 'retry_scheduled') {
    return 'border-amber-200/80 bg-amber-50/90 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100'
  }
  return ''
})

const showSuccessIndicator = computed(() => props.account.auto_recovery_probe?.status === 'success')
const accountIdentity = computed(() => {
  const email = String(props.account.extra?.email_address || '').trim()
  if (email) {
    return email
  }
  return `#${props.account.id}`
})
const platformBadgeLabel = computed(() => {
  const key = String(resolvedPlatform.value || '').trim().toLowerCase()
  return getPlatformEnglishName(key) || 'Account'
})
const fallbackPlatformLabel = computed(() => platformBadgeLabel.value.slice(0, 2).toUpperCase())
const showHealthyDot = computed(() => {
  return props.account.status === 'active' && props.account.lifecycle_state !== 'blacklisted'
})
const showErrorDot = computed(() => {
  return props.account.status === 'error' || props.account.lifecycle_state === 'blacklisted'
})
const errorDotMessage = computed(() => {
  const candidates = [
    props.account.error_message,
    props.account.lifecycle_reason_message,
    props.account.auto_recovery_probe?.summary,
    props.account.auto_recovery_probe?.error_code,
    props.account.lifecycle_reason_code,
  ]
  for (const value of candidates) {
    const text = String(value || '').trim()
    if (text) return text
  }
  return t('admin.accounts.status.issueSummaries.error')
})
const avatarClass = computed(() => platformTheme.value.avatar)
const nameClass = computed(() => platformTheme.value.name)
const metaClass = computed(() => platformTheme.value.meta)
const dotClass = computed(() => platformTheme.value.dot)
const badgeClass = computed(() => platformTheme.value.badge)
</script>
