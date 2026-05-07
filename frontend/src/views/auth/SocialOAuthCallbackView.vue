<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('auth.social.callbackTitle') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ isProcessing ? t('auth.social.callbackProcessing') : t('auth.social.callbackHint') }}
        </p>
      </div>

      <div v-if="needsInvitation" class="space-y-4">
        <p class="text-sm text-gray-700 dark:text-gray-300">
          {{ t('auth.social.invitationRequired') }}
        </p>
        <input
          v-model="invitationCode"
          type="text"
          class="input w-full"
          :placeholder="t('auth.invitationCodePlaceholder')"
          :disabled="isSubmitting"
          @keyup.enter="handleSubmitInvitation"
        />
        <p v-if="invitationError" class="text-sm text-red-600 dark:text-red-400">
          {{ invitationError }}
        </p>
        <button
          class="btn btn-primary w-full"
          :disabled="isSubmitting || !invitationCode.trim()"
          @click="handleSubmitInvitation"
        >
          {{ isSubmitting ? t('auth.social.completing') : t('auth.social.completeRegistration') }}
        </button>
      </div>

      <div
        v-if="errorMessage"
        class="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
      >
        <p class="text-sm text-red-700 dark:text-red-400">
          {{ errorMessage }}
        </p>
      </div>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import { completeSocialOAuthRegistration } from '@/api/auth'
import { useAuthStore, useAppStore } from '@/stores'
import type { SocialOAuthProvider } from '@/types'

const router = useRouter()
const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

const isProcessing = ref(true)
const errorMessage = ref('')
const needsInvitation = ref(false)
const isSubmitting = ref(false)
const invitationCode = ref('')
const invitationError = ref('')
const pendingOAuthToken = ref('')
const provider = ref<SocialOAuthProvider>('github')
const redirectTo = ref('/dashboard')

function parseFragmentParams(): URLSearchParams {
  const raw = typeof window !== 'undefined' ? window.location.hash : ''
  const hash = raw.startsWith('#') ? raw.slice(1) : raw
  return new URLSearchParams(hash)
}

function sanitizeRedirectPath(path: string | null | undefined): string {
  if (!path) return '/dashboard'
  if (!path.startsWith('/')) return '/dashboard'
  if (path.startsWith('//')) return '/dashboard'
  if (path.includes('://')) return '/dashboard'
  if (path.includes('\n') || path.includes('\r')) return '/dashboard'
  return path
}

async function handleSubmitInvitation() {
  invitationError.value = ''
  if (!invitationCode.value.trim()) return

  isSubmitting.value = true
  try {
    const tokenData = await completeSocialOAuthRegistration(
      provider.value,
      pendingOAuthToken.value,
      invitationCode.value.trim()
    )
    if (tokenData.refresh_token) {
      localStorage.setItem('refresh_token', tokenData.refresh_token)
    }
    if (tokenData.expires_in) {
      localStorage.setItem('token_expires_at', String(Date.now() + tokenData.expires_in * 1000))
    }
    await authStore.setToken(tokenData.access_token)
    appStore.showSuccess(t('auth.loginSuccess'))
    await router.replace(redirectTo.value)
  } catch (error: any) {
    invitationError.value = error?.message || t('auth.social.completeRegistrationFailed')
  } finally {
    isSubmitting.value = false
  }
}

onMounted(async () => {
  const params = parseFragmentParams()
  const token = params.get('access_token') || ''
  const refreshToken = params.get('refresh_token') || ''
  const expiresIn = params.get('expires_in') || ''
  const mode = params.get('mode') || 'login'
  const result = params.get('result') || ''
  const redirect = sanitizeRedirectPath(params.get('redirect') || '/dashboard')
  const callbackProvider = params.get('provider')
  const error = params.get('error')
  const errorDesc = params.get('error_description') || params.get('error_message') || ''

  if (callbackProvider === 'google' || callbackProvider === 'github') {
    provider.value = callbackProvider
  }
  redirectTo.value = redirect

  if (error) {
    if (error === 'invitation_required') {
      pendingOAuthToken.value = params.get('pending_oauth_token') || ''
      if (!pendingOAuthToken.value) {
        errorMessage.value = t('auth.social.invalidPendingToken')
        appStore.showError(errorMessage.value)
        isProcessing.value = false
        return
      }
      needsInvitation.value = true
      isProcessing.value = false
      return
    }
    errorMessage.value = errorDesc || error
    appStore.showError(errorMessage.value)
    isProcessing.value = false
    return
  }

  if (mode === 'bind' && result === 'bind_success') {
    appStore.showSuccess(t('auth.social.bindSuccess'))
    await router.replace(redirect)
    return
  }

  if (!token) {
    errorMessage.value = t('auth.social.callbackMissingToken')
    appStore.showError(errorMessage.value)
    isProcessing.value = false
    return
  }

  try {
    if (refreshToken) {
      localStorage.setItem('refresh_token', refreshToken)
    }
    if (expiresIn) {
      const parsed = Number.parseInt(expiresIn, 10)
      if (!Number.isNaN(parsed)) {
        localStorage.setItem('token_expires_at', String(Date.now() + parsed * 1000))
      }
    }
    await authStore.setToken(token)
    appStore.showSuccess(t('auth.loginSuccess'))
    await router.replace(redirect)
  } catch (error: any) {
    errorMessage.value = error?.message || t('auth.loginFailed')
    appStore.showError(errorMessage.value)
    isProcessing.value = false
  }
})
</script>
