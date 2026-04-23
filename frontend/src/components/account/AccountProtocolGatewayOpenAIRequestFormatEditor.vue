<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GatewayOpenAIRequestFormat, OpenAIImageProtocolMode } from '@/types'

const props = defineProps<{
  value: GatewayOpenAIRequestFormat
  imageProtocolMode: OpenAIImageProtocolMode
}>()

const emit = defineEmits<{
  (e: 'update:value', value: GatewayOpenAIRequestFormat): void
  (e: 'update:imageProtocolMode', value: OpenAIImageProtocolMode): void
}>()

const { t } = useI18n()

const options = computed<Array<{ value: GatewayOpenAIRequestFormat; label: string }>>(() => [
  {
    value: '/v1/chat/completions',
    label: t('admin.accounts.protocolGateway.openaiRequestFormat.options.chatCompletions')
  },
  {
    value: '/v1/responses',
    label: t('admin.accounts.protocolGateway.openaiRequestFormat.options.responses')
  }
])

const imageProtocolOptions = computed<Array<{ value: OpenAIImageProtocolMode; label: string }>>(() => [
  {
    value: 'native',
    label: t('admin.accounts.openai.imageProtocol.options.native')
  },
  {
    value: 'compat',
    label: t('admin.accounts.openai.imageProtocol.options.compat')
  }
])

function handleChange(event: Event) {
  emit('update:value', (event.target as HTMLSelectElement).value as GatewayOpenAIRequestFormat)
}

function handleImageProtocolChange(event: Event) {
  emit('update:imageProtocolMode', (event.target as HTMLSelectElement).value as OpenAIImageProtocolMode)
}
</script>

<template>
  <div class="rounded-lg border border-emerald-200 bg-emerald-50/60 p-4 dark:border-emerald-900/40 dark:bg-emerald-950/20">
    <div>
      <h3 class="text-sm font-semibold text-emerald-900 dark:text-emerald-200">
        {{ t('admin.accounts.protocolGateway.openaiRequestFormat.title') }}
      </h3>
      <p class="mt-1 text-xs leading-5 text-emerald-800/90 dark:text-emerald-100/80">
        {{ t('admin.accounts.protocolGateway.openaiRequestFormat.description') }}
      </p>
    </div>
    <div class="mt-4">
      <label class="input-label">{{ t('admin.accounts.protocolGateway.openaiRequestFormat.label') }}</label>
      <select class="input" :value="props.value" @change="handleChange">
        <option
          v-for="option in options"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </option>
      </select>
      <p class="input-hint">{{ t('admin.accounts.protocolGateway.openaiRequestFormat.hint') }}</p>
    </div>
    <div class="mt-4">
      <label class="input-label">{{ t('admin.accounts.protocolGateway.openaiImageProtocol.label') }}</label>
      <select class="input" :value="props.imageProtocolMode" @change="handleImageProtocolChange">
        <option
          v-for="option in imageProtocolOptions"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </option>
      </select>
      <p class="input-hint">{{ t('admin.accounts.protocolGateway.openaiImageProtocol.hint') }}</p>
    </div>
  </div>
</template>
