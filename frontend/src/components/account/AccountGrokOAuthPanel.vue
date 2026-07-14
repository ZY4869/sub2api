<template>
  <div class="space-y-4 rounded-lg border border-emerald-200 bg-emerald-50/70 p-4 dark:border-emerald-900/40 dark:bg-emerald-950/20">
    <div class="space-y-1">
      <h3 class="text-sm font-semibold text-emerald-950 dark:text-emerald-100">
        {{ t('admin.accounts.grokOauth.title') }}
      </h3>
      <p class="text-xs leading-5 text-emerald-800 dark:text-emerald-300">
        {{ t('admin.accounts.grokOauth.description') }}
      </p>
    </div>

    <div class="inline-flex rounded-lg border border-emerald-200 bg-white p-1 dark:border-emerald-900/40 dark:bg-slate-900">
      <button
        v-for="option in modeOptions"
        :key="option.value"
        type="button"
        :class="[
          'rounded-md px-3 py-1.5 text-xs font-semibold transition',
          mode === option.value
            ? 'bg-emerald-600 text-white shadow-sm'
            : 'text-emerald-800 hover:bg-emerald-50 dark:text-emerald-200 dark:hover:bg-emerald-950/40'
        ]"
        :aria-pressed="mode === option.value"
        @click="setMode(option.value)"
      >
        {{ option.label }}
      </button>
    </div>

    <GrokOAuthCallbackFlow
      v-if="mode === 'callback'"
      ref="callbackFlowRef"
      :proxy-id="proxyId"
      :submit-label="submitLabel"
      :submitting="submitting"
      @submit="emit('submit', $event)"
      @device-input="handleDeviceInput"
    />
    <GrokOAuthDeviceFlow
      v-else
      ref="deviceFlowRef"
      :proxy-id="proxyId"
      :submit-label="submitLabel"
      :submitting="submitting"
      :initial-message="deviceHint"
      @submit="emit('submit', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ParsedGrokOAuthPayload } from '@/utils/grokOAuth'
import GrokOAuthCallbackFlow from './GrokOAuthCallbackFlow.vue'
import GrokOAuthDeviceFlow from './GrokOAuthDeviceFlow.vue'

type GrokOAuthMode = 'callback' | 'device'

withDefaults(defineProps<{
  proxyId?: number | null
  submitLabel: string
  submitting?: boolean
}>(), {
  proxyId: null,
  submitting: false
})

const emit = defineEmits<{
  submit: [payload: ParsedGrokOAuthPayload]
}>()

const { t } = useI18n()

const mode = ref<GrokOAuthMode>('callback')
const callbackFlowRef = ref<{ reset: () => void } | null>(null)
const deviceFlowRef = ref<{ reset: () => void; showExternalCodeHint: () => void } | null>(null)
const deviceHint = ref('')

const modeOptions = computed(() => [
  { value: 'callback' as const, label: t('admin.accounts.grokOauth.callbackMode') },
  { value: 'device' as const, label: t('admin.accounts.grokOauth.deviceMode') }
])

function setMode(nextMode: GrokOAuthMode) {
  mode.value = nextMode
}

async function handleDeviceInput() {
  deviceHint.value = t('admin.accounts.grokOauth.externalDeviceCodeHint')
  mode.value = 'device'
  await nextTick()
  deviceFlowRef.value?.showExternalCodeHint()
}

function reset() {
  mode.value = 'callback'
  deviceHint.value = ''
  callbackFlowRef.value?.reset()
  deviceFlowRef.value?.reset()
}

defineExpose({ reset })
</script>
