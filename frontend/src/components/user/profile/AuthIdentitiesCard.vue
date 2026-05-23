<template>
  <div class="card overflow-hidden">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('profile.identities.title') }}
      </h3>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.identities.description') }}
      </p>
    </div>

    <div class="space-y-4 p-6">
      <div class="flex flex-wrap gap-3">
        <button
          v-if="githubEnabled"
          type="button"
          class="btn btn-secondary btn-sm inline-flex items-center"
          @click="startBind('github')"
        >
          <LobeStaticIcon
            class="mr-2"
            :sources="githubIconSources"
            badge-text="GH"
            size="18px"
            variant="platform"
            alt="GitHub"
          />
          {{ t('profile.identities.bindGitHub') }}
        </button>
        <button
          v-if="googleEnabled"
          type="button"
          class="btn btn-secondary btn-sm inline-flex items-center"
          @click="startBind('google')"
        >
          <LobeStaticIcon
            class="mr-2"
            :sources="googleIconSources"
            badge-text="GO"
            size="18px"
            variant="platform"
            alt="Google"
          />
          {{ t('profile.identities.bindGoogle') }}
        </button>
        <button
          v-if="dingtalkEnabled"
          type="button"
          class="btn btn-secondary btn-sm inline-flex items-center"
          @click="startBind('dingtalk')"
        >
          <LobeStaticIcon
            class="mr-2"
            :sources="dingtalkIconSources"
            badge-text="DT"
            size="18px"
            variant="platform"
            alt="DingTalk"
          />
          {{ t('profile.identities.bindDingTalk') }}
        </button>
      </div>

      <div v-if="loading" class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div v-else-if="identities.length === 0" class="rounded-xl bg-gray-50 px-4 py-5 text-sm text-gray-500 dark:bg-dark-800 dark:text-gray-400">
        {{ t('profile.identities.empty') }}
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="identity in identities"
          :key="`${identity.provider}-${identity.id}`"
          class="flex items-center justify-between rounded-xl border border-gray-100 px-4 py-3 dark:border-dark-700"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <LobeStaticIcon
                :sources="getProviderIconSources(identity.provider)"
                :badge-text="providerBadge(identity.provider)"
                size="18px"
                variant="platform"
                :alt="providerLabel(identity.provider)"
              />
              <span class="font-medium text-gray-900 dark:text-white">
                {{ providerLabel(identity.provider) }}
              </span>
              <span
                :class="[
                  'badge',
                  identity.email_verified ? 'badge-success' : 'badge-warning'
                ]"
              >
                {{
                  identity.email_verified
                    ? t('profile.identities.verified')
                    : t('profile.identities.unverified')
                }}
              </span>
            </div>
            <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ identity.email || identity.display_name || identity.provider_user_id }}
            </div>
          </div>

          <button
            type="button"
            class="btn btn-secondary btn-sm"
            @click="removeIdentity(identity.provider)"
          >
            {{ t('profile.identities.unbind') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { buildSocialOAuthStartURL } from '@/api/auth'
import { userAPI } from '@/api'
import LobeStaticIcon from '@/components/common/LobeStaticIcon.vue'
import { useAppStore } from '@/stores'
import type { AuthIdentity, SocialOAuthProvider } from '@/types'
import {
  buildLobeIconSources,
  resolveLobeBadgeText,
  resolveProviderIconSlugs,
} from '@/utils/lobeIconResolver'

defineProps<{
  identities: AuthIdentity[]
  loading?: boolean
  githubEnabled?: boolean
  googleEnabled?: boolean
  dingtalkEnabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const githubIconSources = buildLobeIconSources(resolveProviderIconSlugs('github'))
const googleIconSources = buildLobeIconSources(resolveProviderIconSlugs('google'))
const dingtalkIconSources = buildLobeIconSources(resolveProviderIconSlugs('dingtalk'))

function providerLabel(provider: string): string {
  switch (provider) {
    case 'github':
      return 'GitHub'
    case 'google':
      return 'Google'
    case 'dingtalk':
      return 'DingTalk'
    default:
      return provider
  }
}

function providerBadge(provider: string): string {
  switch (provider) {
    case 'github':
      return 'GH'
    case 'google':
      return 'GO'
    case 'dingtalk':
      return 'DT'
    default:
      return resolveLobeBadgeText(provider)
  }
}

function getProviderIconSources(provider: string): string[] {
  return buildLobeIconSources(resolveProviderIconSlugs(provider))
}

function startBind(provider: SocialOAuthProvider): void {
  window.location.href = buildSocialOAuthStartURL(provider, {
    mode: 'bind',
    redirect: '/profile'
  })
}

async function removeIdentity(provider: string): Promise<void> {
  try {
    await userAPI.deleteAuthIdentity(provider as SocialOAuthProvider)
    appStore.showSuccess(t('profile.identities.unbindSuccess'))
    emit('refresh')
  } catch (error: any) {
    appStore.showError(error?.message || t('profile.identities.unbindFailed'))
  }
}
</script>
