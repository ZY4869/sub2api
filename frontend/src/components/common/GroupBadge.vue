<template>
  <span
    :class="[
      'inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-medium transition-colors',
      badgeClass
    ]"
    :title="name"
    :aria-label="name"
  >
    <span
      v-if="isAiry"
      class="font-mono text-[10px] font-black opacity-75"
      aria-hidden="true"
    >
      #
    </span>
    <!-- Platform logo -->
    <PlatformIcon v-else-if="platform" :platform="platform" size="sm" />
    <!-- Group name -->
    <span class="truncate">{{ name }}</span>
    <!-- Right side label -->
    <span v-if="showLabel" :class="labelClass">
      <template v-if="hasCustomRate">
        <!-- 原倍率删除线 + 专属倍率高亮 -->
        <span class="line-through opacity-50 mr-0.5">{{ rateMultiplier }}x</span>
        <span class="font-bold">{{ userRateMultiplier }}x</span>
      </template>
      <template v-else>
        {{ labelText }}
      </template>
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionType, GroupPlatform } from '@/types'
import PlatformIcon from './PlatformIcon.vue'

interface Props {
  name: string
  platform?: GroupPlatform
  subscriptionType?: SubscriptionType
  rateMultiplier?: number
  userRateMultiplier?: number | null // 用户专属倍率
  showRate?: boolean
  daysRemaining?: number | null // 剩余天数（订阅类型时使用）
  visualVariant?: 'default' | 'airy'
}

const props = withDefaults(defineProps<Props>(), {
  subscriptionType: 'standard',
  showRate: true,
  daysRemaining: null,
  userRateMultiplier: null,
  visualVariant: 'default'
})

const { t } = useI18n()

const isSubscription = computed(() => props.subscriptionType === 'subscription')
const isAiry = computed(() => props.visualVariant === 'airy')

// 是否有专属倍率（且与默认倍率不同）
const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

// 是否显示右侧标签
const showLabel = computed(() => {
  if (!props.showRate) return false
  // 订阅类型：显示天数或"订阅"
  if (isSubscription.value) return true
  // 标准类型：显示倍率（包括专属倍率）
  return props.rateMultiplier !== undefined || hasCustomRate.value
})

// Label text
const labelText = computed(() => {
  if (isSubscription.value) {
    // 如果有剩余天数，显示天数
    if (props.daysRemaining !== null && props.daysRemaining !== undefined) {
      if (props.daysRemaining <= 0) {
        return t('admin.users.expired')
      }
      return t('admin.users.daysRemaining', { days: props.daysRemaining })
    }
    // 否则显示"订阅"
    return t('groups.subscription')
  }
  return props.rateMultiplier !== undefined ? `${props.rateMultiplier}x` : ''
})

// Label style based on type and days remaining
const labelClass = computed(() => {
  const base = 'px-1.5 py-0.5 rounded text-[10px] font-semibold'

  if (!isSubscription.value) {
    // Standard: subtle background (不再为专属倍率使用不同的背景色)
    return `${base} bg-black/10 dark:bg-white/10`
  }

  // 订阅类型：根据剩余天数显示不同颜色
  if (props.daysRemaining !== null && props.daysRemaining !== undefined) {
    if (props.daysRemaining <= 0 || props.daysRemaining <= 3) {
      // 已过期或紧急（<=3天）：红色
      return `${base} bg-red-200/80 text-red-800 dark:bg-red-800/50 dark:text-red-300`
    }
    if (props.daysRemaining <= 7) {
      // 警告（<=7天）：橙色
      return `${base} bg-amber-200/80 text-amber-800 dark:bg-amber-800/50 dark:text-amber-300`
    }
  }

  // 正常状态或无天数：根据平台显示主题色
  if (props.platform === 'anthropic') {
    return `${base} bg-orange-200/60 text-orange-800 dark:bg-orange-800/40 dark:text-orange-300`
  }
  if (props.platform === 'kiro') {
    return `${base} bg-amber-200/60 text-amber-800 dark:bg-amber-800/40 dark:text-amber-300`
  }
  if (props.platform === 'openai') {
    return `${base} bg-emerald-200/60 text-emerald-800 dark:bg-emerald-800/40 dark:text-emerald-300`
  }
  if (props.platform === 'gemini') {
    return `${base} bg-blue-200/60 text-blue-800 dark:bg-blue-800/40 dark:text-blue-300`
  }
  return `${base} bg-violet-200/60 text-violet-800 dark:bg-violet-800/40 dark:text-violet-300`
})

const hashGroupName = (value: string) => {
  let hash = 0
  for (let index = 0; index < value.length; index += 1) {
    hash = value.charCodeAt(index) + ((hash << 5) - hash)
  }
  return Math.abs(hash)
}

const airyBadgeClass = computed(() => {
  const normalizedName = props.name.trim()
  const presets: Record<string, string> = {
    Admin: 'bg-red-100 border-red-300 text-red-800 dark:bg-red-500/15 dark:border-red-400/30 dark:text-red-200',
    管理员: 'bg-red-100 border-red-300 text-red-800 dark:bg-red-500/15 dark:border-red-400/30 dark:text-red-200',
    Dev: 'bg-blue-100 border-blue-300 text-blue-800 dark:bg-blue-500/15 dark:border-blue-400/30 dark:text-blue-200',
    开发组: 'bg-blue-100 border-blue-300 text-blue-800 dark:bg-blue-500/15 dark:border-blue-400/30 dark:text-blue-200',
    Test: 'bg-amber-100 border-amber-300 text-amber-800 dark:bg-amber-500/15 dark:border-amber-400/30 dark:text-amber-100',
    测试池: 'bg-amber-100 border-amber-300 text-amber-800 dark:bg-amber-500/15 dark:border-amber-400/30 dark:text-amber-100',
    VIP: 'bg-fuchsia-100 border-fuchsia-300 text-fuchsia-800 dark:bg-fuchsia-500/15 dark:border-fuchsia-400/30 dark:text-fuchsia-200',
    专属: 'bg-fuchsia-100 border-fuchsia-300 text-fuchsia-800 dark:bg-fuchsia-500/15 dark:border-fuchsia-400/30 dark:text-fuchsia-200',
    Audit: 'bg-teal-100 border-teal-300 text-teal-800 dark:bg-teal-500/15 dark:border-teal-400/30 dark:text-teal-200',
    审计: 'bg-teal-100 border-teal-300 text-teal-800 dark:bg-teal-500/15 dark:border-teal-400/30 dark:text-teal-200',
    Billing: 'bg-emerald-100 border-emerald-300 text-emerald-800 dark:bg-emerald-500/15 dark:border-emerald-400/30 dark:text-emerald-200',
    财务: 'bg-emerald-100 border-emerald-300 text-emerald-800 dark:bg-emerald-500/15 dark:border-emerald-400/30 dark:text-emerald-200',
    Overseas: 'bg-indigo-100 border-indigo-300 text-indigo-800 dark:bg-indigo-500/15 dark:border-indigo-400/30 dark:text-indigo-200',
    海外代理: 'bg-indigo-100 border-indigo-300 text-indigo-800 dark:bg-indigo-500/15 dark:border-indigo-400/30 dark:text-indigo-200'
  }
  const fallbackPalettes = [
    'bg-orange-100 border-orange-300 text-orange-800 dark:bg-orange-500/15 dark:border-orange-400/30 dark:text-orange-200',
    'bg-cyan-100 border-cyan-300 text-cyan-800 dark:bg-cyan-500/15 dark:border-cyan-400/30 dark:text-cyan-200',
    'bg-lime-100 border-lime-300 text-lime-800 dark:bg-lime-500/15 dark:border-lime-400/30 dark:text-lime-200',
    'bg-pink-100 border-pink-300 text-pink-800 dark:bg-pink-500/15 dark:border-pink-400/30 dark:text-pink-200',
    'bg-violet-100 border-violet-300 text-violet-800 dark:bg-violet-500/15 dark:border-violet-400/30 dark:text-violet-200'
  ]
  return [
    'rounded border px-1.5 py-[2.5px] text-[9px] font-extrabold tracking-normal shadow-sm',
    presets[normalizedName] ?? fallbackPalettes[hashGroupName(normalizedName) % fallbackPalettes.length]
  ].join(' ')
})

// Badge color based on platform and subscription type
const badgeClass = computed(() => {
  if (isAiry.value) {
    return airyBadgeClass.value
  }
  if (props.platform === 'anthropic') {
    // Claude: orange theme
    return isSubscription.value
      ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
      : 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400'
  } else if (props.platform === 'kiro') {
    return isSubscription.value
      ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
      : 'bg-orange-50 text-orange-700 dark:bg-orange-900/20 dark:text-orange-400'
  } else if (props.platform === 'openai') {
    // OpenAI: green theme
    return isSubscription.value
      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
      : 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400'
  } else if (props.platform === 'gemini') {
    return isSubscription.value
      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
      : 'bg-sky-50 text-sky-700 dark:bg-sky-900/20 dark:text-sky-400'
  }
  // Fallback: original colors
  return isSubscription.value
    ? 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400'
    : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
})
</script>
