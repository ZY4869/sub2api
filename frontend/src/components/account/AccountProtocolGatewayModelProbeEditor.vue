<template>
  <section class="space-y-3 rounded-2xl border border-gray-200 bg-white/80 p-4 dark:border-dark-600 dark:bg-dark-700/60">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="space-y-1">
        <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('admin.accounts.protocolGateway.probeTitle') }}
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.protocolGateway.probeHint') }}
        </p>
      </div>

      <button
        type="button"
        class="inline-flex items-center justify-center gap-2 rounded-xl bg-primary-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-600 disabled:cursor-not-allowed disabled:bg-primary-300"
        :disabled="probing || !trimmedApiKey"
        @click="handleProbe"
      >
        <Icon v-if="probing" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
        <Icon v-else name="search" size="sm" :stroke-width="2" />
        <span>{{ t('admin.accounts.protocolGateway.probeAction') }}</span>
      </button>
    </div>

    <div
      v-if="probeSource || probeNotice"
      class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:border-dark-500 dark:bg-dark-800 dark:text-gray-300"
    >
      <div v-if="probeSource">
        {{ t('admin.accounts.protocolGateway.probeSourceLabel') }}{{ probeSource }}
      </div>
      <div v-if="probeNotice">
        {{ t('admin.accounts.protocolGateway.probeNoticeLabel') }}{{ probeNotice }}
      </div>
    </div>

    <div
      v-if="probedModels.length > 0"
      class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400"
    >
      <span>{{ t('admin.accounts.protocolGateway.probeSelectionHint') }}</span>
      <span>{{ t('admin.accounts.protocolGateway.probeSelectedCount', { count: allowedModels.length }) }}</span>
    </div>

    <div v-if="probedModels.length > 0" class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      <button
        v-for="model in probedModels"
        :key="model.id"
        type="button"
        :title="model.id"
        :class="cardClasses(model)"
        @click="toggleModel(model.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="truncate text-sm font-semibold">
              {{ model.display_name || model.id }}
            </div>
            <div class="truncate text-xs opacity-80">{{ model.id }}</div>
          </div>
          <span
            v-if="isSelected(model.id)"
            class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white/80 text-emerald-600 shadow-sm dark:bg-white/10 dark:text-emerald-300"
          >
            <Icon name="check" size="sm" :stroke-width="2" />
          </span>
        </div>
        <div class="mt-3 flex items-center justify-between gap-3 text-xs">
          <span class="truncate">
            {{
              model.registry_state === 'existing'
                ? t('admin.accounts.protocolGateway.registryExisting')
                : t('admin.accounts.protocolGateway.registryMissing')
            }}
          </span>
          <span v-if="model.registry_model_id" class="truncate opacity-80">
            {{ model.registry_model_id }}
          </span>
        </div>
      </button>
    </div>

    <div
      v-else
      class="rounded-xl border border-dashed border-gray-300 px-3 py-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400"
    >
      {{ t('admin.accounts.protocolGateway.probeEmpty') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  ProtocolGatewayProbeModel,
  ProtocolGatewayProbeResponse
} from '@/api/admin/accounts'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { GatewayProtocol } from '@/types'

const props = defineProps<{
  gatewayProtocol: GatewayProtocol
  baseUrl: string
  apiKey: string
  proxyId?: number | null
}>()

const emit = defineEmits<{
  probed: [result: ProtocolGatewayProbeResponse]
}>()

const allowedModels = defineModel<string[]>('allowedModels', { required: true })
const probedModels = defineModel<ProtocolGatewayProbeModel[]>('probedModels', { required: true })

const { t } = useI18n()
const appStore = useAppStore()

const probing = ref(false)
const probeSource = ref('')
const probeNotice = ref('')

const trimmedApiKey = computed(() => props.apiKey.trim())

watch(
  () => [props.gatewayProtocol, props.baseUrl, props.apiKey, props.proxyId, probedModels.value.length] as const,
  ([, , , , modelCount]) => {
    if (modelCount === 0) {
      probeSource.value = ''
      probeNotice.value = ''
    }
  }
)

const isSelected = (modelId: string) => allowedModels.value.includes(modelId)

const toggleModel = (modelId: string) => {
  if (isSelected(modelId)) {
    allowedModels.value = allowedModels.value.filter((item) => item !== modelId)
    return
  }
  allowedModels.value = [...allowedModels.value, modelId]
}

const cardClasses = (model: ProtocolGatewayProbeModel) => {
  const selected = isSelected(model.id)
  const palette =
    model.registry_state === 'existing'
      ? 'border-emerald-200 bg-emerald-50/90 text-emerald-900 dark:border-emerald-900/70 dark:bg-emerald-950/30 dark:text-emerald-100'
      : 'border-amber-200 bg-amber-50/90 text-amber-900 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-100'
  const state = selected
    ? 'ring-2 ring-primary-400/60 shadow-md'
    : 'hover:border-primary-200 hover:shadow-sm dark:hover:border-primary-700/60'
  return [
    'rounded-2xl border px-4 py-3 text-left transition',
    palette,
    state
  ]
}

const handleProbe = async () => {
  if (probing.value) {
    return
  }
  if (!trimmedApiKey.value) {
    appStore.showWarning(t('admin.accounts.protocolGateway.probeRequiredApiKey'))
    return
  }

  probing.value = true
  try {
    const result = await adminAPI.accounts.probeProtocolGatewayModels({
      gateway_protocol: props.gatewayProtocol,
      base_url: props.baseUrl.trim() || undefined,
      api_key: trimmedApiKey.value,
      proxy_id: props.proxyId ?? undefined
    })
    probedModels.value = result.models
    allowedModels.value = result.models.map((model) => model.id)
    probeSource.value = result.probe_source || ''
    probeNotice.value = result.probe_notice || ''
    emit('probed', result)
  } catch (error: any) {
    console.error('Failed to probe protocol gateway models:', error)
    appStore.showError(error?.message || t('admin.accounts.protocolGateway.probeFailed'))
  } finally {
    probing.value = false
  }
}
</script>
