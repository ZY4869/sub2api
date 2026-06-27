<template>
  <div class="flex min-w-0 max-w-full flex-col gap-1.5 overflow-hidden">
    <div class="flex min-w-0 items-center gap-1.5">
      <div
        :class="[
          'inline-flex w-fit min-w-0 max-w-full items-center gap-1.5 rounded-full border py-[3px]',
          compact ? 'px-2' : 'px-2.5',
          planBadgeClass
        ]"
        :title="planLabel"
        data-test="account-service-plan-badge"
      >
        <span
          v-if="isApiKeyAccount"
          :class="['flex h-4 w-4 shrink-0 items-center justify-center rounded-full', keyIconClass]"
          data-test="account-key-type-icon"
        >
          <Icon name="key" size="xs" :stroke-width="2.4" />
        </span>
        <PlatformIcon v-else :platform="platform" size="xs" />
        <span class="min-w-0 max-w-[5.5rem] truncate text-[11px] font-bold leading-none tracking-tight">
          {{ planLabel }}
        </span>
      </div>

      <div class="flex shrink-0 items-center gap-1">
        <div
          :class="[
            'flex h-6 w-6 items-center justify-center rounded-full border',
            authTypeIconClass
          ]"
          :title="typeTitle"
          data-test="account-auth-type-icon"
        >
          <svg
            v-if="type === 'oauth'"
            class="h-3 w-3"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
            />
          </svg>
          <svg
            v-else-if="type === 'setup-token'"
            class="h-3 w-3"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.1"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M12 3l7 4v5c0 5-3.5 8-7 9-3.5-1-7-4-7-9V7l7-4z"
            />
          </svg>
          <svg
            v-else
            class="h-3 w-3"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.1"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M15.75 5.25a3 3 0 114.243 4.243l-7.72 7.72a3 3 0 01-1.06.688l-3.08 1.026 1.026-3.08a3 3 0 01.688-1.06l7.72-7.72z"
            />
          </svg>
        </div>

        <div
          v-if="privacyBadge"
          :class="[
            'flex h-6 w-6 items-center justify-center rounded-full border',
            privacyBadge.className
          ]"
          :title="privacyBadge.title"
        >
          <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.1">
            <path stroke-linecap="round" stroke-linejoin="round" :d="privacyBadge.icon" />
          </svg>
        </div>
      </div>
    </div>

    <div class="flex min-w-0 flex-wrap items-center gap-1">
      <span :class="[metaBadgeClass, platformMetaBadgeClass]" :title="platformLabel">
        {{ platformLabel }}
      </span>
      <span :class="[metaBadgeClass, typeMetaBadgeClass]" :title="typeLabel">
        {{ typeLabel }}
      </span>
      <span
        v-if="isApiKeyAccount && keyTierDisplay.tierLabel"
        :class="[metaBadgeClass, tierMetaBadgeClass]"
        :title="keyTierDisplay.tierLabel"
      >
        {{ keyTierDisplay.tierLabel }}
      </span>
      <span v-if="gatewayProtocolLabel" :class="[metaBadgeClass, platformMetaBadgeClass]" :title="gatewayProtocolLabel">
        {{ gatewayProtocolLabel }}
      </span>
      <span v-if="expiresLabel" :class="[metaBadgeClass, tierMetaBadgeClass]" :title="expiresLabel">
        {{ expiresLabel }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountPlatform, AccountType, GatewayProtocol } from '@/types'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { isProtocolGatewayPlatform, resolveGatewayProtocolLabel } from '@/utils/accountProtocolGateway'
import Icon from '@/components/icons/Icon.vue'
import { resolveAccountKeyTierDisplay } from '@/utils/accountKeyTierDisplay'

const props = defineProps<{
  platform: AccountPlatform
  gatewayProtocol?: GatewayProtocol
  type: AccountType
  planType?: string
  planTypeLabel?: string
  proMultiplier?: number | null
  extra?: Record<string, unknown>
  privacyMode?: string
  subscriptionExpiresAt?: string
  compact?: boolean
}>()

const { t } = useI18n()

const platformLabel = computed(() => t(`admin.accounts.platforms.${props.platform}`))
const isApiKeyAccount = computed(() => props.type === 'apikey')

const keyTierDisplay = computed(() =>
  resolveAccountKeyTierDisplay({
    platform: props.platform,
    type: props.type,
    credentials: {
      plan_type: props.planType,
      plan_type_label: props.planTypeLabel,
      pro_multiplier: props.proMultiplier,
    },
    extra: props.extra,
  })
)

const typeLabel = computed(() => {
  switch (props.type) {
    case 'oauth':
      return t('ui.platformType.oauth')
    case 'setup-token':
      return t('ui.platformType.token')
    case 'apikey':
      return t('ui.platformType.key')
    case 'sso':
      return t('ui.platformType.sso')
    case 'bedrock':
      return t('ui.platformType.aws')
    default:
      return props.type
  }
})

const typeTitle = computed(() => `${platformLabel.value} / ${typeLabel.value}`)
const normalizedPlatform = computed(() => String(props.platform || '').trim().toLowerCase())

const gatewayProtocolLabel = computed(() => {
  if (!isProtocolGatewayPlatform(props.platform)) return ''
  return resolveGatewayProtocolLabel(props.gatewayProtocol)
})

const planLabel = computed<string>(() => {
  if (isApiKeyAccount.value) return keyTierDisplay.value.primaryLabel
  const lower = String(props.planType || '').trim().toLowerCase()
  if (lower === 'pro' || lower === 'chatgptpro') {
    return typeof props.proMultiplier === 'number' && props.proMultiplier > 0
      ? `Pro${props.proMultiplier}x`
      : 'Pro'
  }
  const explicitLabel = props.planTypeLabel?.trim()
  if (explicitLabel) return explicitLabel
  if (!lower) return platformLabel.value
  switch (lower) {
    case 'plus':
      return 'Plus'
    case 'team':
      return 'Team'
    case 'free':
      return 'Free'
    default:
      return String(props.planType)
  }
})

const expiresLabel = computed(() => {
  if (!props.subscriptionExpiresAt || !props.planType) return ''
  if (props.planType.toLowerCase() === 'free') return ''
  const expiresAt = new Date(props.subscriptionExpiresAt)
  if (Number.isNaN(expiresAt.getTime())) return ''
  const yyyy = expiresAt.getFullYear()
  const mm = String(expiresAt.getMonth() + 1).padStart(2, '0')
  const dd = String(expiresAt.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
})

const planBadgeClass = computed(() => {
  if (isApiKeyAccount.value) return keyTierDisplay.value.className
  if (planLabel.value === 'Free') {
    return 'border-slate-300/80 bg-slate-100 text-slate-600'
  }
  if (planLabel.value === 'Plus') {
    return 'border-emerald-300/80 bg-emerald-50 text-emerald-700'
  }
  if (planLabel.value === 'Team') {
    return 'border-blue-300/80 bg-blue-50 text-blue-700'
  }
  if (planLabel.value.startsWith('Pro')) {
    if (planLabel.value.includes('20x')) {
      return 'border-slate-700 bg-slate-800 text-amber-400 ring-1 ring-slate-900'
    }
    return 'border-cyan-200 bg-cyan-50 text-cyan-700'
  }
  if (props.platform === 'openai') {
    return 'border-emerald-200/60 bg-emerald-50/85 text-emerald-700'
  }
  if (props.platform === 'anthropic') {
    return 'border-orange-200/60 bg-orange-50/85 text-orange-700'
  }
  if (props.platform === 'grok') {
    return 'border-slate-200/60 bg-slate-100/90 text-slate-700'
  }
  if (props.platform === 'antigravity') {
    return 'border-purple-200/60 bg-purple-50/85 text-purple-700'
  }
  return 'border-blue-200/60 bg-blue-50/85 text-blue-700'
})

const metaBadgeClass =
  'inline-flex max-w-[76px] items-center rounded px-1.5 py-[2px] text-[10px] font-semibold truncate'

const keyIconClass = computed(() => {
  if (keyTierDisplay.value.className.includes('bg-slate-800')) {
    return 'bg-amber-400/20 text-amber-300 ring-1 ring-amber-300/40'
  }
  if (keyTierDisplay.value.className.includes('emerald')) {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-200'
  }
  if (keyTierDisplay.value.className.includes('blue')) {
    return 'bg-blue-100 text-blue-700 dark:bg-blue-400/15 dark:text-blue-200'
  }
  return 'bg-sky-100 text-sky-700 dark:bg-sky-400/15 dark:text-sky-200'
})

const authTypeIconClass = computed(() => {
  switch (props.type) {
    case 'oauth':
      return 'border-blue-200 bg-blue-50 text-blue-600 dark:border-blue-400/20 dark:bg-blue-400/10 dark:text-blue-200'
    case 'setup-token':
      return 'border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-200'
    case 'apikey':
      return 'border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-200'
    case 'sso':
      return 'border-violet-200 bg-violet-50 text-violet-600 dark:border-violet-400/20 dark:bg-violet-400/10 dark:text-violet-200'
    default:
      return 'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-500/30 dark:bg-slate-500/10 dark:text-slate-200'
  }
})

const platformMetaBadgeClass = computed(() => {
  switch (normalizedPlatform.value) {
    case 'openai':
      return 'border border-emerald-200/80 bg-emerald-50/90 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-100'
    case 'anthropic':
      return 'border border-orange-200/80 bg-orange-50/90 text-orange-700 dark:border-orange-400/20 dark:bg-orange-400/10 dark:text-orange-100'
    case 'gemini':
      return 'border border-blue-200/80 bg-blue-50/90 text-blue-700 dark:border-blue-400/20 dark:bg-blue-400/10 dark:text-blue-100'
    case 'antigravity':
      return 'border border-violet-200/80 bg-violet-50/90 text-violet-700 dark:border-violet-400/20 dark:bg-violet-400/10 dark:text-violet-100'
    default:
      return 'border border-slate-200/80 bg-slate-50/90 text-slate-700 dark:border-slate-600/40 dark:bg-slate-700/40 dark:text-slate-100'
  }
})

const typeMetaBadgeClass = computed(() => {
  switch (props.type) {
    case 'oauth':
      return 'border border-blue-200/80 bg-blue-50/90 text-blue-700 dark:border-blue-400/20 dark:bg-blue-400/10 dark:text-blue-100'
    case 'apikey':
      return 'border border-amber-200/80 bg-amber-50/90 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-100'
    case 'setup-token':
      return 'border border-emerald-200/80 bg-emerald-50/90 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-100'
    default:
      return 'border border-slate-200/80 bg-slate-50/90 text-slate-700 dark:border-slate-600/40 dark:bg-slate-700/40 dark:text-slate-100'
  }
})

const tierMetaBadgeClass = computed(() => {
  if (keyTierDisplay.value.className.includes('bg-slate-800')) {
    return 'border border-slate-700 bg-slate-800 text-amber-300 dark:border-amber-400/20'
  }
  return typeMetaBadgeClass.value
})

const privacyBadge = computed(() => {
  if (props.platform !== 'openai' || props.type !== 'oauth' || !props.privacyMode) return null
  const shieldCheck = 'M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z'
  const shieldX = 'M12 9v3.75m0-10.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285zM12 18h.008v.008H12V18z'
  switch (props.privacyMode) {
    case 'training_off':
      return {
        title: t('admin.accounts.privacyTrainingOff'),
        icon: shieldCheck,
        className: 'border-emerald-200/70 bg-emerald-50 text-emerald-500 dark:border-emerald-400/20 dark:bg-emerald-500/10'
      }
    case 'training_set_cf_blocked':
      return {
        title: t('admin.accounts.privacyCfBlocked'),
        icon: shieldX,
        className: 'border-amber-200/70 bg-amber-50 text-amber-500 dark:border-amber-400/20 dark:bg-amber-500/10'
      }
    case 'training_set_failed':
      return {
        title: t('admin.accounts.privacyFailed'),
        icon: shieldX,
        className: 'border-red-200/70 bg-red-50 text-red-500 dark:border-red-400/20 dark:bg-red-500/10'
      }
    default:
      return null
  }
})
</script>
