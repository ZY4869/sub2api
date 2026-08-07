<template>
  <div v-if="enabled" class="space-y-2">
    <TurnstileWidget
      v-if="props.provider === 'turnstile' && props.turnstileSiteKey"
      ref="turnstileRef"
      :site-key="props.turnstileSiteKey"
      @verify="handleTurnstileVerify"
      @expire="handleExpire"
      @error="handleError"
    />

    <button
      v-else-if="props.provider === 'tencent'"
      type="button"
      class="btn btn-secondary w-full"
      :disabled="props.disabled || !props.tencentCaptchaAppId || loading"
      @click="startTencentCaptcha"
    >
      <Icon name="shield" size="md" class="mr-2" />
      {{ verified ? '已完成人机验证' : loading ? '正在加载验证' : '完成人机验证' }}
    </button>

    <div v-else-if="props.provider === 'aliyun'" class="space-y-2">
      <div :id="aliyunElementId" class="min-h-0"></div>
      <button
        :id="aliyunButtonId"
        type="button"
        class="btn btn-secondary w-full"
        :disabled="props.disabled || !props.aliyunCaptchaSceneId || !props.aliyunPrefix || loading"
        @click="startAliyunCaptcha"
      >
        <Icon name="shield" size="md" class="mr-2" />
        {{ verified ? '已完成人机验证' : loading ? '正在加载验证' : '完成人机验证' }}
      </button>
    </div>

    <p v-if="error" class="input-error-text text-center">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import Icon from '@/components/icons/Icon.vue'
import type { CaptchaProof, CaptchaProvider } from '@/types'

type TencentCaptchaCallbackResult = {
  ret: number
  ticket?: string
  randstr?: string
}

type TencentCaptchaInstance = {
  show: () => void
  destroy?: () => void
}

type AliyunCaptchaInstance = {
  show?: () => void
  hide?: () => void
  startTracelessVerification?: () => void
}

declare global {
  interface Window {
    TencentCaptcha?: new (
      appId: string,
      callback: (result: TencentCaptchaCallbackResult) => void
    ) => TencentCaptchaInstance
    AliyunCaptchaConfig?: { region: string; prefix: string }
    initAliyunCaptcha?: (options: Record<string, unknown>) => void
  }
}

const props = defineProps<{
  provider: CaptchaProvider
  disabled?: boolean
  turnstileSiteKey?: string
  tencentCaptchaAppId?: string
  aliyunCaptchaSceneId?: string
  aliyunPrefix?: string
  aliyunRegion?: string
}>()

const emit = defineEmits<{
  (e: 'verify', proof: CaptchaProof): void
  (e: 'expire'): void
  (e: 'error'): void
}>()

const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const loading = ref(false)
const verified = ref(false)
const error = ref('')
const aliyunCaptcha = ref<AliyunCaptchaInstance | null>(null)
const aliyunElementId = `aliyun-captcha-${Math.random().toString(36).slice(2)}`
const aliyunButtonId = `aliyun-captcha-button-${Math.random().toString(36).slice(2)}`

const enabled = computed(() => props.provider !== 'none')

function emitProof(proof: CaptchaProof): void {
  verified.value = true
  error.value = ''
  emit('verify', proof)
}

function handleTurnstileVerify(token: string): void {
  emitProof({ turnstile_token: token })
}

function handleExpire(): void {
  verified.value = false
  emit('expire')
}

function handleError(): void {
  verified.value = false
  error.value = '人机验证加载失败，请稍后重试'
  emit('error')
}

function loadScript(src: string, marker: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[data-captcha="${marker}"]`)) {
      resolve()
      return
    }
    const script = document.createElement('script')
    script.src = src
    script.async = true
    script.defer = true
    script.dataset.captcha = marker
    script.onload = () => resolve()
    script.onerror = () => reject(new Error(`Failed to load ${marker}`))
    document.head.appendChild(script)
  })
}

async function startTencentCaptcha(): Promise<void> {
  if (!props.tencentCaptchaAppId || props.disabled) return
  loading.value = true
  error.value = ''
  try {
    await loadScript('https://ssl.captcha.qq.com/TCaptcha.js', 'tencent')
    if (!window.TencentCaptcha) throw new Error('TencentCaptcha unavailable')
    const captcha = new window.TencentCaptcha(props.tencentCaptchaAppId, (result) => {
      loading.value = false
      if (result.ret === 0 && result.ticket && result.randstr) {
        emitProof({
          tencent_captcha_ticket: result.ticket,
          tencent_captcha_randstr: result.randstr
        })
        return
      }
      if (result.ret !== 2) {
        handleError()
      }
    })
    captcha.show()
  } catch {
    loading.value = false
    handleError()
  }
}

async function initAliyunCaptcha(): Promise<void> {
  if (!props.aliyunCaptchaSceneId || !props.aliyunPrefix) return
  window.AliyunCaptchaConfig = {
    region: props.aliyunRegion || 'cn',
    prefix: props.aliyunPrefix
  }
  await loadScript('https://o.alicdn.com/captcha-frontend/aliyunCaptcha/AliyunCaptcha.js', 'aliyun')
  if (!window.initAliyunCaptcha) throw new Error('initAliyunCaptcha unavailable')
  window.initAliyunCaptcha({
    SceneId: props.aliyunCaptchaSceneId,
    mode: 'popup',
    element: `#${aliyunElementId}`,
    button: `#${aliyunButtonId}`,
    success: (captchaVerifyParam: string) => {
      loading.value = false
      emitProof({ aliyun_captcha_verify_param: captchaVerifyParam })
    },
    fail: () => {
      loading.value = false
      handleError()
    },
    getInstance: (instance: AliyunCaptchaInstance) => {
      aliyunCaptcha.value = instance
    },
    onError: () => {
      loading.value = false
      handleError()
    },
    slideStyle: {
      width: Math.max(320, Math.min(360, window.innerWidth - 48)),
      height: 40
    },
    language: 'cn'
  })
}

async function startAliyunCaptcha(): Promise<void> {
  if (props.disabled) return
  loading.value = true
  error.value = ''
  try {
    if (!aliyunCaptcha.value) {
      await initAliyunCaptcha()
    }
    aliyunCaptcha.value?.show?.()
    aliyunCaptcha.value?.startTracelessVerification?.()
  } catch {
    loading.value = false
    handleError()
  }
}

function reset(): void {
  verified.value = false
  error.value = ''
  if (props.provider === 'turnstile') {
    turnstileRef.value?.reset()
  }
}

defineExpose({ reset })

onMounted(() => {
  if (props.provider === 'aliyun' && props.aliyunCaptchaSceneId && props.aliyunPrefix) {
    initAliyunCaptcha().catch(() => handleError())
  }
})

watch(
  () => props.provider,
  () => reset()
)
</script>
