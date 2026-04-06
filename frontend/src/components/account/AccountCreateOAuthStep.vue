<script setup lang="ts">
import { computed, ref } from 'vue'
import type { AddMethod } from '@/composables/useAccountOAuth'
import type { AccountPlatform } from '@/types'
import OAuthAuthorizationFlow from './OAuthAuthorizationFlow.vue'
import type { OAuthFlowExposed } from './oauthFlow.types'

const props = defineProps<{
  addMethod: AddMethod
  authUrl: string
  sessionId: string
  loading: boolean
  error: string
  showHelp: boolean
  showProxyWarning: boolean
  allowMultiple: boolean
  showCookieOption: boolean
  showRefreshTokenOption: boolean
  platform: AccountPlatform
  showProjectId: boolean
}>()

const emit = defineEmits<{
  generateUrl: []
  cookieAuth: [sessionKey: string]
  validateRefreshToken: [refreshToken: string]
}>()

const flowRef = ref<OAuthFlowExposed | null>(null)

const authCode = computed(() => flowRef.value?.authCode || '')
const oauthState = computed(() => flowRef.value?.oauthState || '')
const projectId = computed(() => flowRef.value?.projectId || '')
const sessionKey = computed(() => flowRef.value?.sessionKey || '')
const refreshToken = computed(() => flowRef.value?.refreshToken || '')
const inputMethod = computed(() => flowRef.value?.inputMethod || 'manual')

defineExpose({
  authCode,
  oauthState,
  projectId,
  sessionKey,
  refreshToken,
  inputMethod,
  reset: () => flowRef.value?.reset()
})
</script>

<template>
  <div class="space-y-5">
    <OAuthAuthorizationFlow
      ref="flowRef"
      :add-method="props.addMethod"
      :auth-url="props.authUrl"
      :session-id="props.sessionId"
      :loading="props.loading"
      :error="props.error"
      :show-help="props.showHelp"
      :show-proxy-warning="props.showProxyWarning"
      :allow-multiple="props.allowMultiple"
      :show-cookie-option="props.showCookieOption"
      :show-refresh-token-option="props.showRefreshTokenOption"
      :platform="props.platform"
      :show-project-id="props.showProjectId"
      @generate-url="emit('generateUrl')"
      @cookie-auth="emit('cookieAuth', $event)"
      @validate-refresh-token="emit('validateRefreshToken', $event)"
    />
  </div>
</template>
