<template>
  <div v-if="shouldShowQuota">
    <!-- First line: Platform + Tier Badge -->
    <div class="mb-1 flex items-center gap-1">
      <span :class="['badge text-xs px-2 py-0.5 rounded font-medium', tierBadgeClass]">
        {{ tierLabel }}
      </span>
    </div>

    <!-- Usage status: unlimited flow or rate limit -->
    <div class="text-xs text-gray-400 dark:text-gray-500">
      <span v-if="!isRateLimited">
        {{ t('admin.accounts.gemini.rateLimit.unlimited') }}
      </span>
      <span
        v-else
        :class="[
          'font-medium',
          isUrgent
            ? 'text-red-600 dark:text-red-400 animate-pulse'
            : 'text-amber-600 dark:text-amber-400'
        ]"
      >
        {{ t('admin.accounts.gemini.rateLimit.limited', { time: resetCountdown }) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account, GeminiCredentials } from '@/types'
import { resolveEffectiveAccountPlatformFromAccount } from '@/utils/accountProtocolGateway'
import { normalizeGeminiAIStudioTier, resolveGeminiChannel } from '@/utils/geminiAccount'

const props = defineProps<{
  account: Account
}>()

const { t } = useI18n()
const runtimePlatform = computed(() => resolveEffectiveAccountPlatformFromAccount(props.account))

const now = ref(new Date())
let timer: ReturnType<typeof setInterval> | null = null

// 是否为 Code Assist OAuth
// 判断逻辑与后端保持一致：project_id 存在即为 Code Assist
const isCodeAssist = computed(() => {
  return geminiChannel.value === 'gcp'
})

// 是否为 Google One OAuth
const isGoogleOne = computed(() => {
  return geminiChannel.value === 'google_one'
})

const geminiChannel = computed(() =>
  resolveGeminiChannel({
    type: props.account.type,
    credentials: props.account.credentials as GeminiCredentials | undefined
  })
)

// 是否应该显示配额信息
const shouldShowQuota = computed(() => {
  return runtimePlatform.value === 'gemini'
})

const normalizedAIStudioTier = computed(() => {
  const creds = props.account.credentials as GeminiCredentials | undefined
  return normalizeGeminiAIStudioTier(creds?.tier_id)
})

// Tier 标签文本
const tierLabel = computed(() => {
  const creds = props.account.credentials as GeminiCredentials | undefined

  if (isCodeAssist.value) {
    const tier = (creds?.tier_id || '').toString().trim().toLowerCase()
    if (tier === 'gcp_enterprise') return 'GCP Enterprise'
    if (tier === 'gcp_standard') return 'GCP Standard'
    // Backward compatibility
    const upper = (creds?.tier_id || '').toString().trim().toUpperCase()
    if (upper.includes('ULTRA') || upper.includes('ENTERPRISE')) return 'GCP Enterprise'
    if (upper) return `GCP ${upper}`
    return 'GCP'
  }

  if (isGoogleOne.value) {
    const tier = (creds?.tier_id || '').toString().trim().toLowerCase()
    if (tier === 'google_ai_ultra') return 'Google AI Ultra'
    if (tier === 'google_ai_pro') return 'Google AI Pro'
    if (tier === 'google_one_free') return 'Google One Free'
    // Backward compatibility
    const upper = (creds?.tier_id || '').toString().trim().toUpperCase()
    if (upper === 'AI_PREMIUM') return 'Google AI Pro'
    if (upper === 'GOOGLE_ONE_UNLIMITED') return 'Google AI Ultra'
    if (upper) return `Google One ${upper}`
    return 'Google One'
  }

  if (geminiChannel.value === 'vertex_ai') {
    return 'Vertex AI'
  }

  if (geminiChannel.value === 'ai_studio_client') {
    switch (normalizedAIStudioTier.value) {
      case 'aistudio_tier_3':
        return 'AI Studio Client Tier 3'
      case 'aistudio_tier_2':
        return 'AI Studio Client Tier 2'
      case 'aistudio_tier_1':
        return 'AI Studio Client Tier 1'
      default:
        return 'AI Studio Client Free Tier'
    }
  }

  switch (normalizedAIStudioTier.value) {
    case 'aistudio_tier_3':
      return 'AI Studio Tier 3'
    case 'aistudio_tier_2':
      return 'AI Studio Tier 2'
    case 'aistudio_tier_1':
      return 'AI Studio Tier 1'
    default:
      return 'AI Studio Free Tier'
  }
})

// Tier Badge 样式（统一样式）
const tierBadgeClass = computed(() => {
  const creds = props.account.credentials as GeminiCredentials | undefined

  if (isCodeAssist.value) {
    const tier = (creds?.tier_id || '').toString().trim().toLowerCase()
    if (tier === 'gcp_enterprise') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    if (tier === 'gcp_standard') return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    // Backward compatibility
    const upper = (creds?.tier_id || '').toString().trim().toUpperCase()
    if (upper.includes('ULTRA') || upper.includes('ENTERPRISE')) return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
  }

  if (isGoogleOne.value) {
    const tier = (creds?.tier_id || '').toString().trim().toLowerCase()
    if (tier === 'google_ai_ultra') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    if (tier === 'google_ai_pro') return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    if (tier === 'google_one_free') return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    // Backward compatibility
    const upper = (creds?.tier_id || '').toString().trim().toUpperCase()
    if (upper === 'GOOGLE_ONE_UNLIMITED') return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    if (upper === 'AI_PREMIUM') return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
  }

  if (geminiChannel.value === 'vertex_ai') {
    return 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300'
  }

  switch (normalizedAIStudioTier.value) {
    case 'aistudio_tier_3':
      return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    case 'aistudio_tier_2':
      return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
    case 'aistudio_tier_1':
      return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
  }
})

// 是否限流
const isRateLimited = computed(() => {
  if (!props.account.rate_limit_reset_at) return false
  const resetTime = Date.parse(props.account.rate_limit_reset_at)
  // 防护：如果日期解析失败（NaN），则认为未限流
  if (Number.isNaN(resetTime)) return false
  return resetTime > now.value.getTime()
})

// 倒计时文本
const resetCountdown = computed(() => {
  if (!props.account.rate_limit_reset_at) return ''
  const resetTime = Date.parse(props.account.rate_limit_reset_at)
  // 防护：如果日期解析失败，显示 "-"
  if (Number.isNaN(resetTime)) return '-'

  const diffMs = resetTime - now.value.getTime()
  if (diffMs <= 0) return t('admin.accounts.gemini.rateLimit.now')

  const diffSeconds = Math.floor(diffMs / 1000)
  const diffMinutes = Math.floor(diffSeconds / 60)
  const diffHours = Math.floor(diffMinutes / 60)

  if (diffMinutes < 1) return `${diffSeconds}s`
  if (diffHours < 1) {
    const secs = diffSeconds % 60
    return `${diffMinutes}m ${secs}s`
  }
  const mins = diffMinutes % 60
  return `${diffHours}h ${mins}m`
})

// 是否紧急（< 1分钟）
const isUrgent = computed(() => {
  if (!props.account.rate_limit_reset_at) return false
  const resetTime = Date.parse(props.account.rate_limit_reset_at)
  // 防护：如果日期解析失败，返回 false
  if (Number.isNaN(resetTime)) return false

  const diffMs = resetTime - now.value.getTime()
  return diffMs > 0 && diffMs < 60000
})

// 监听限流状态，动态启动/停止定时器
watch(
  () => isRateLimited.value,
  (limited) => {
    if (limited && !timer) {
      // 进入限流状态，启动定时器
      timer = setInterval(() => {
        now.value = new Date()
      }, 1000)
    } else if (!limited && timer) {
      // 解除限流，停止定时器
      clearInterval(timer)
      timer = null
    }
  },
  { immediate: true } // 立即执行，确保挂载时已限流的情况也能启动定时器
)

onUnmounted(() => {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
})
</script>
