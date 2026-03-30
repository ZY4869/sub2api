<template>
  <section class="space-y-3 rounded-2xl border border-gray-200 bg-white/80 p-4 dark:border-dark-600 dark:bg-dark-700/60">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="space-y-1">
        <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('admin.accounts.apiKeyProbe.title') }}
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.apiKeyProbe.hint') }}
        </p>
      </div>

      <button
        type="button"
        class="inline-flex items-center justify-center gap-2 rounded-xl bg-primary-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-600 disabled:cursor-not-allowed disabled:bg-primary-300"
        :disabled="probing || !probeReady"
        @click="handleProbe"
      >
        <Icon v-if="probing" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
        <Icon v-else name="search" size="sm" :stroke-width="2" />
        <span>{{ t('admin.accounts.apiKeyProbe.action') }}</span>
      </button>
    </div>

    <div
      v-if="probeSource || probeNotice"
      class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:border-dark-500 dark:bg-dark-800 dark:text-gray-300"
    >
      <div v-if="probeSource">
        {{ t('admin.accounts.apiKeyProbe.probeSourceLabel') }}{{ probeSource }}
      </div>
      <div v-if="probeNotice">
        {{ t('admin.accounts.apiKeyProbe.probeNoticeLabel') }}{{ probeNotice }}
      </div>
    </div>

    <div
      v-if="probedModels.length > 0"
      class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400"
    >
      <span>{{ t('admin.accounts.apiKeyProbe.selectionHint') }}</span>
      <span>{{ t('admin.accounts.apiKeyProbe.selectedCount', { count: allowedModels.length }) }}</span>
    </div>

    <div v-if="probedModels.length > 0" class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      <button
        v-for="model in probedModels"
        :key="model.id"
        type="button"
        :title="model.id"
        :class="cardClasses(model)"
        @click="toggleModel(model)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="break-words text-sm font-semibold" :title="model.display_name || model.id">
              {{ model.display_name || model.id }}
            </div>
            <div class="break-words text-xs opacity-80" :title="model.id">{{ model.id }}</div>
            <div
              v-if="model.upstream_source || model.availability"
              class="mt-2 flex flex-wrap items-center gap-2"
            >
              <span
                v-if="model.upstream_source"
                class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
                :class="sourceBadgeClasses(model)"
              >
                {{ sourceBadgeText(model) }}
              </span>
              <span
                v-if="model.availability"
                class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
                :class="availabilityBadgeClasses(model)"
              >
                {{ availabilityBadgeText(model) }}
              </span>
            </div>
            <div
              v-if="model.availability === 'uncallable' && model.availability_reason"
              class="mt-2 break-words text-xs text-rose-700 dark:text-rose-200"
            >
              {{ model.availability_reason }}
            </div>
          </div>
          <span
            v-if="isSelected(model.id)"
            class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white/80 text-emerald-600 shadow-sm dark:bg-white/10 dark:text-emerald-300"
          >
            <Icon name="check" size="sm" :stroke-width="2" />
          </span>
        </div>

        <div class="mt-3 flex items-center justify-between gap-3 text-xs">
          <span>
            {{
              model.registry_state === 'existing'
                ? t('admin.accounts.apiKeyProbe.registryExisting')
                : t('admin.accounts.apiKeyProbe.registryMissing')
            }}
          </span>
          <span v-if="model.registry_model_id" class="break-words text-right opacity-80" :title="model.registry_model_id">
            {{ model.registry_model_id }}
          </span>
        </div>

        <div
          v-if="isSelected(model.id)"
          class="mt-3 space-y-2 rounded-xl border border-white/60 bg-white/60 p-3 dark:border-white/10 dark:bg-white/5"
          @click.stop
        >
          <div
            v-if="model.availability === 'uncallable'"
            class="rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200"
          >
            {{ t('admin.accounts.apiKeyProbe.selectedUncallableWarning') }}
          </div>
          <div class="grid gap-2 md:grid-cols-2">
            <label class="space-y-1 text-left">
              <span class="text-[11px] font-medium uppercase tracking-wide opacity-70">
                {{ t('admin.accounts.requestModel') }}
              </span>
              <input
                :value="currentAlias(model.id)"
                type="text"
                class="input h-10 bg-white/90 text-sm dark:bg-dark-900/60"
                :placeholder="model.id"
                @input="updateModelAlias(model, ($event.target as HTMLInputElement).value)"
                @click.stop
              />
            </label>
            <label class="space-y-1 text-left">
              <span class="text-[11px] font-medium uppercase tracking-wide opacity-70">
                {{ t('admin.accounts.actualModel') }}
              </span>
              <input
                :value="model.id"
                type="text"
                class="input h-10 cursor-not-allowed bg-gray-100/90 text-sm text-gray-500 dark:bg-dark-900/60 dark:text-gray-400"
                readonly
                @click.stop
              />
            </label>
          </div>
        </div>
      </button>
    </div>

    <div
      v-else
      class="rounded-xl border border-dashed border-gray-300 px-3 py-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400"
    >
      {{ t('admin.accounts.apiKeyProbe.empty') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { ProtocolGatewayProbeModel } from '@/api/admin/accounts'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { ModelMapping } from '@/utils/accountFormShared'
import { buildDefaultVertexAlias, isGeminiVertexSourceCredentials } from '@/utils/vertexAi'

const props = defineProps<{
  platform: string
  accountType: string
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
  proxyId?: number | null
  probeReady: boolean
}>()

const allowedModels = defineModel<string[]>('allowedModels', { required: true })
const modelMappings = defineModel<ModelMapping[]>('modelMappings', { required: true })
const probedModels = defineModel<ProtocolGatewayProbeModel[]>('probedModels', { required: true })

const { t } = useI18n()
const appStore = useAppStore()
const probing = ref(false)
const probeSource = ref('')
const probeNotice = ref('')
const aliasDrafts = ref<Record<string, string>>({})
const manualUncallableSelections = ref<string[]>([])
const hasInitializedFromMappings = ref(false)
const isVertexSource = () =>
  props.platform === 'gemini' && isGeminiVertexSourceCredentials(props.credentials)

const defaultAlias = (modelId: string) =>
  isVertexSource() ? buildDefaultVertexAlias(modelId) : modelId

watch(
  () => [probedModels.value.length, modelMappings.value.length] as const,
  ([modelCount, mappingCount]) => {
    if (hasInitializedFromMappings.value || modelCount > 0 || mappingCount === 0) return
    hasInitializedFromMappings.value = true
    const nextDrafts = { ...aliasDrafts.value }
    const seen = new Set<string>()
    for (const row of modelMappings.value) {
      const target = row.to.trim()
      if (!target || Object.prototype.hasOwnProperty.call(nextDrafts, target)) continue
      nextDrafts[target] = row.from
    }
    aliasDrafts.value = nextDrafts
    probedModels.value = modelMappings.value
      .map((row) => row.to.trim())
      .filter((target) => target && !seen.has(target) && (seen.add(target), true))
      .map((target) => ({
        id: target,
        display_name: target,
        registry_state: 'existing' as const,
        registry_model_id: target
      }))
  },
  { immediate: true }
)

watch(
  () => allowedModels.value.join('\x00'),
  () => {
    const selected = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
    const nextMappings = modelMappings.value.filter((row) => selected.has(row.to.trim()))
    if (nextMappings.length !== modelMappings.value.length) {
      modelMappings.value = nextMappings
    }
  },
  { immediate: true }
)

const isSelected = (modelId: string) => allowedModels.value.includes(modelId)
const isCallableModel = (model: ProtocolGatewayProbeModel) => model.availability !== 'uncallable'

const sourceBadgeText = (model: ProtocolGatewayProbeModel) =>
  model.upstream_source === 'verified_extra'
    ? t('admin.accounts.apiKeyProbe.sourceVerifiedExtra')
    : t('admin.accounts.apiKeyProbe.sourceOfficial')

const availabilityBadgeText = (model: ProtocolGatewayProbeModel) =>
  model.availability === 'uncallable'
    ? t('admin.accounts.apiKeyProbe.availabilityUncallable')
    : t('admin.accounts.apiKeyProbe.availabilityCallable')

const sourceBadgeClasses = (model: ProtocolGatewayProbeModel) =>
  model.upstream_source === 'verified_extra'
    ? 'bg-sky-100 text-sky-700 dark:bg-sky-950/40 dark:text-sky-200'
    : 'bg-slate-100 text-slate-700 dark:bg-slate-900/60 dark:text-slate-200'

const availabilityBadgeClasses = (model: ProtocolGatewayProbeModel) =>
  model.availability === 'uncallable'
    ? 'bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-200'
    : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-200'

const currentAlias = (modelId: string) => {
  if (Object.prototype.hasOwnProperty.call(aliasDrafts.value, modelId)) {
    return aliasDrafts.value[modelId]
  }
  return modelMappings.value.find((row) => row.to.trim() === modelId)?.from ?? defaultAlias(modelId)
}

const ensureMappingForModel = (model: ProtocolGatewayProbeModel) => {
  const index = modelMappings.value.findIndex((row) => row.to.trim() === model.id)
  const alias = currentAlias(model.id)
  if (index >= 0) {
    const nextMappings = [...modelMappings.value]
    nextMappings[index] = { from: alias, to: model.id }
    modelMappings.value = nextMappings
    return
  }
  modelMappings.value = [...modelMappings.value, { from: alias, to: model.id }]
}

const toggleModel = (model: ProtocolGatewayProbeModel) => {
  const nextManualSelections = new Set(manualUncallableSelections.value)
  if (isSelected(model.id)) {
    nextManualSelections.delete(model.id)
    manualUncallableSelections.value = [...nextManualSelections]
    allowedModels.value = allowedModels.value.filter((item) => item !== model.id)
    modelMappings.value = modelMappings.value.filter((row) => row.to.trim() !== model.id)
    return
  }
  if (!isCallableModel(model)) {
    nextManualSelections.add(model.id)
    manualUncallableSelections.value = [...nextManualSelections]
  }
  allowedModels.value = [...allowedModels.value, model.id]
  ensureMappingForModel(model)
}

const updateModelAlias = (model: ProtocolGatewayProbeModel, value: string) => {
  aliasDrafts.value = { ...aliasDrafts.value, [model.id]: value }
  ensureMappingForModel(model)
  modelMappings.value = modelMappings.value.map((row) =>
    row.to.trim() === model.id ? { from: value, to: model.id } : row
  )
}

const cardClasses = (model: ProtocolGatewayProbeModel) => [
  'rounded-2xl border px-4 py-3 text-left transition',
  model.availability === 'uncallable'
    ? 'border-rose-200 bg-rose-50/90 text-rose-950 dark:border-rose-900/70 dark:bg-rose-950/30 dark:text-rose-100'
    : model.registry_state === 'existing'
    ? 'border-emerald-200 bg-emerald-50/90 text-emerald-900 dark:border-emerald-900/70 dark:bg-emerald-950/30 dark:text-emerald-100'
    : 'border-amber-200 bg-amber-50/90 text-amber-900 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-100',
  isSelected(model.id)
    ? 'ring-2 ring-primary-400/60 shadow-md'
    : 'hover:border-primary-200 hover:shadow-sm dark:hover:border-primary-700/60'
]

const handleProbe = async () => {
  if (probing.value || !props.probeReady) return
  probing.value = true
  try {
    const result = await adminAPI.accounts.probeModels({
      platform: props.platform,
      type: props.accountType,
      credentials: props.credentials,
      extra: props.extra,
      proxy_id: props.proxyId ?? undefined
    })
    const aliasByTarget = new Map(
      modelMappings.value
        .map((row) => [row.to.trim(), currentAlias(row.to.trim())] as const)
        .filter(([target]) => Boolean(target))
    )
    const selectedUncallable = new Set(manualUncallableSelections.value)
    const nextAllowedModels = result.models
      .filter((model) => isCallableModel(model) || selectedUncallable.has(model.id))
      .map((model) => model.id)
    probedModels.value = result.models
    allowedModels.value = nextAllowedModels
    modelMappings.value = nextAllowedModels.map((modelId) => ({
      from: aliasByTarget.get(modelId) || defaultAlias(modelId),
      to: modelId
    }))
    manualUncallableSelections.value = result.models
      .filter((model) => model.availability === 'uncallable' && selectedUncallable.has(model.id))
      .map((model) => model.id)
    probeSource.value = result.probe_source || ''
    probeNotice.value = result.probe_notice || ''
  } catch (error: any) {
    console.error('Failed to probe account models:', error)
    appStore.showError(error?.message || t('admin.accounts.apiKeyProbe.failed'))
  } finally {
    probing.value = false
  }
}
</script>
