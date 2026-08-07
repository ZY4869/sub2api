<template>
  <div class="space-y-4">
    <CaptchaChallenge
      v-if="captchaRequired"
      :provider="captchaProvider"
      :turnstile-site-key="props.turnstileSiteKey"
      :tencent-captcha-app-id="props.tencentCaptchaAppId"
      :aliyun-captcha-scene-id="props.aliyunCaptchaSceneId"
      :aliyun-prefix="props.aliyunCaptchaPrefix"
      :aliyun-region="props.aliyunCaptchaRegion"
      :disabled="disabled || starting"
      @verify="handleCaptchaVerify"
      @expire="handleCaptchaExpire"
      @error="handleCaptchaExpire"
    />

    <div class="grid gap-3">
      <button
        v-if="showLinuxDo"
        type="button"
        :disabled="buttonsDisabled"
        class="btn btn-secondary w-full"
        @click="startLinuxDo"
      >
        <span class="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-orange-500 text-[10px] font-bold text-white">L</span>
        {{ t('auth.social.continueWithLinuxDo') }}
      </button>

      <button
        v-if="showGitHub"
        type="button"
        :disabled="buttonsDisabled"
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
        :disabled="buttonsDisabled"
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
        :disabled="buttonsDisabled"
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
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { startSocialOAuth } from '@/api/auth'
import CaptchaChallenge from '@/components/auth/CaptchaChallenge.vue'
import LobeStaticIcon from '@/components/common/LobeStaticIcon.vue'
import type { CaptchaProof, CaptchaProvider, SocialOAuthProvider } from '@/types'
import { buildLobeIconSources, resolveProviderIconSlugs } from '@/utils/lobeIconResolver'

const props = defineProps<{
  disabled?: boolean
  showLinuxDo?: boolean
  showGitHub?: boolean
  showGoogle?: boolean
  showDingTalk?: boolean
  mode?: 'login' | 'bind'
  redirect?: string
  captchaProvider?: CaptchaProvider
  turnstileSiteKey?: string
  tencentCaptchaAppId?: string
  aliyunCaptchaSceneId?: string
  aliyunCaptchaPrefix?: string
  aliyunCaptchaRegion?: string
}>()

const route = useRoute()
const { t } = useI18n()

const showDivider = computed(
  () => props.showLinuxDo || props.showGitHub || props.showGoogle || props.showDingTalk
)
const starting = ref(false)
const captchaProof = ref<CaptchaProof>({})
const captchaVerified = ref(false)
const captchaProvider = computed<CaptchaProvider>(() => props.captchaProvider || 'none')
const captchaRequired = computed(() => captchaProvider.value !== 'none')
const buttonsDisabled = computed(() => props.disabled || starting.value || (captchaRequired.value && !captchaVerified.value))
const githubIconSources = buildLobeIconSources(resolveProviderIconSlugs('github'))
const googleIconSources = buildLobeIconSources(resolveProviderIconSlugs('google'))
const dingtalkIconSources = buildLobeIconSources(resolveProviderIconSlugs('dingtalk'))

function getRedirectTarget(): string {
  return props.redirect || (route.query.redirect as string) || '/dashboard'
}

function handleCaptchaVerify(proof: CaptchaProof): void {
  captchaProof.value = proof
  captchaVerified.value = true
}

function handleCaptchaExpire(): void {
  captchaProof.value = {}
  captchaVerified.value = false
}

async function startLinuxDo(): Promise<void> {
  await startSocial('linuxdo')
}

async function startSocial(provider: SocialOAuthProvider): Promise<void> {
  if (buttonsDisabled.value) return
  starting.value = true
  try {
    const redirectTo = getRedirectTarget()
    const result = await startSocialOAuth(provider, {
      ...captchaProof.value,
      mode: props.mode || 'login',
      redirect: redirectTo,
      aff_code: typeof route.query.aff_code === 'string' ? route.query.aff_code : undefined
    })
    window.location.href = result.authorize_url
  } finally {
    starting.value = false
  }
}
</script>
