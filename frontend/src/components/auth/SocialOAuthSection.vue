<template>
  <div class="space-y-4">
    <div class="grid gap-3">
      <button
        v-if="showLinuxDo"
        type="button"
        :disabled="disabled"
        class="btn btn-secondary w-full"
        @click="startLinuxDo"
      >
        <span class="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-orange-500 text-[10px] font-bold text-white">L</span>
        {{ t('auth.social.continueWithLinuxDo') }}
      </button>

      <button
        v-if="showGitHub"
        type="button"
        :disabled="disabled"
        class="btn btn-secondary w-full"
        @click="startSocial('github')"
      >
        <LobeStaticIcon
          class="mr-2"
          :sources="githubIconSources"
          badge-text="GH"
          size="20px"
          variant="platform"
          alt="GitHub"
        />
        {{ t('auth.social.continueWithGitHub') }}
      </button>

      <button
        v-if="showGoogle"
        type="button"
        :disabled="disabled"
        class="btn btn-secondary w-full"
        @click="startSocial('google')"
      >
        <LobeStaticIcon
          class="mr-2"
          :sources="googleIconSources"
          badge-text="GO"
          size="20px"
          variant="platform"
          alt="Google"
        />
        {{ t('auth.social.continueWithGoogle') }}
      </button>

      <button
        v-if="showDingTalk"
        type="button"
        :disabled="disabled"
        class="btn btn-secondary w-full"
        @click="startSocial('dingtalk')"
      >
        <LobeStaticIcon
          class="mr-2"
          :sources="dingtalkIconSources"
          badge-text="DT"
          size="20px"
          variant="platform"
          alt="DingTalk"
        />
        {{ t('auth.social.continueWithDingTalk') }}
      </button>
    </div>

    <div v-if="showDivider" class="flex items-center gap-3">
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
      <span class="text-xs text-gray-500 dark:text-dark-400">
        {{ t('auth.social.orContinue') }}
      </span>
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { buildSocialOAuthStartURL } from '@/api/auth'
import LobeStaticIcon from '@/components/common/LobeStaticIcon.vue'
import type { SocialOAuthProvider } from '@/types'
import { buildLobeIconSources, resolveProviderIconSlugs } from '@/utils/lobeIconResolver'

const props = defineProps<{
  disabled?: boolean
  showLinuxDo?: boolean
  showGitHub?: boolean
  showGoogle?: boolean
  showDingTalk?: boolean
  mode?: 'login' | 'bind'
  redirect?: string
}>()

const route = useRoute()
const { t } = useI18n()

const showDivider = computed(
  () => props.showLinuxDo || props.showGitHub || props.showGoogle || props.showDingTalk
)
const githubIconSources = buildLobeIconSources(resolveProviderIconSlugs('github'))
const googleIconSources = buildLobeIconSources(resolveProviderIconSlugs('google'))
const dingtalkIconSources = buildLobeIconSources(resolveProviderIconSlugs('dingtalk'))

function getRedirectTarget(): string {
  return props.redirect || (route.query.redirect as string) || '/dashboard'
}

function startLinuxDo(): void {
  const redirectTo = getRedirectTarget()
  window.location.href = buildSocialOAuthStartURL('linuxdo', { redirect: redirectTo })
}

function startSocial(provider: SocialOAuthProvider): void {
  const redirectTo = getRedirectTarget()
  window.location.href = buildSocialOAuthStartURL(provider, {
    mode: props.mode || 'login',
    redirect: redirectTo
  })
}
</script>
