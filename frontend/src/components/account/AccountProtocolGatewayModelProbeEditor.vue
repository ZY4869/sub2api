<template>
  <section class="space-y-3 rounded-2xl border border-gray-200 bg-white/80 p-4 dark:border-dark-600 dark:bg-dark-700/60">
    <div
      v-if="gatewayProtocol === 'mixed'"
      class="space-y-3 rounded-2xl border border-dashed border-gray-200 bg-gray-50/70 p-4 dark:border-dark-500 dark:bg-dark-800/60"
    >
      <div class="space-y-1">
        <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('admin.accounts.protocolGateway.acceptedProtocolsTitle') }}
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.protocolGateway.acceptedProtocolsHint') }}
        </p>
      </div>

      <div class="grid gap-2 sm:grid-cols-3">
        <button
          v-for="protocol in acceptedProtocolOptions"
          :key="protocol.value"
          type="button"
          :class="selectionCardClass(isAcceptedProtocolSelected(protocol.value))"
          @click="toggleAcceptedProtocol(protocol.value)"
        >
          <span class="font-medium">{{ protocol.label }}</span>
          <span class="break-words text-left text-xs opacity-80" :title="protocol.requestFormats">{{ protocol.requestFormats }}</span>
        </button>
      </div>
    </div>

    <div
      v-if="availableClientProfiles.length > 0"
      class="space-y-3 rounded-2xl border border-dashed border-gray-200 bg-gray-50/70 p-4 dark:border-dark-500 dark:bg-dark-800/60"
    >
      <div class="space-y-1">
        <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('admin.accounts.protocolGateway.clientProfilesTitle') }}
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.protocolGateway.clientProfilesHint') }}
        </p>
      </div>

      <div class="grid gap-2 sm:grid-cols-2">
        <button
          v-for="profile in availableClientProfiles"
          :key="profile"
          type="button"
          :class="selectionCardClass(isClientProfileSelected(profile))"
          @click="toggleClientProfile(profile)"
        >
          <span class="font-medium">{{ clientProfileLabel(profile) }}</span>
          <span class="text-xs opacity-80">{{ clientProfileHint(profile) }}</span>
        </button>
      </div>
    </div>

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
      v-if="probedModels.length > 0 && availableClientProfiles.length > 0"
      class="flex flex-wrap items-center gap-2"
    >
      <button
        v-if="supportsProfile('codex')"
        type="button"
        class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
        @click="applyProfileToAll('codex')"
      >
        {{ t('admin.accounts.protocolGateway.applyAllCodex') }}
      </button>
      <button
        v-if="supportsProfile('gemini_cli')"
        type="button"
        class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
        @click="applyProfileToAll('gemini_cli')"
      >
        {{ t('admin.accounts.protocolGateway.applyAllGeminiCli') }}
      </button>
      <button
        type="button"
        class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 transition hover:border-rose-300 hover:text-rose-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-rose-500 dark:hover:text-rose-300"
        @click="clearAllProfiles"
      >
        {{ t('admin.accounts.protocolGateway.clearSimulations') }}
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
      v-if="providerOptions.length > 0"
      class="space-y-3 rounded-2xl border border-dashed border-gray-200 bg-gray-50/70 p-4 dark:border-dark-500 dark:bg-dark-800/60"
    >
      <div class="space-y-1">
        <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {{ t('admin.accounts.protocolGateway.defaultTestTargetTitle') }}
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.protocolGateway.defaultTestTargetHint') }}
        </p>
      </div>

      <div class="grid gap-3 md:grid-cols-2">
        <label class="space-y-1.5">
          <span class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.protocolGateway.testProviderLabel') }}
          </span>
          <select
            :value="gatewayTestProvider"
            class="input h-11"
            @change="updateGatewayTestProvider(($event.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t('admin.accounts.protocolGateway.testProviderAutoOption') }}</option>
            <option
              v-for="option in providerOptions"
              :key="option.provider"
              :value="option.provider"
            >
              {{ option.label }}
            </option>
          </select>
        </label>

        <label class="space-y-1.5">
          <span class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.protocolGateway.testModelLabel') }}
          </span>
          <select
            :value="gatewayTestModelId"
            class="input h-11"
            :disabled="!gatewayTestProvider"
            @change="updateGatewayTestModelId(($event.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t('admin.accounts.protocolGateway.testModelAutoOption') }}</option>
            <option
              v-for="option in testModelOptions"
              :key="option.id"
              :value="option.id"
            >
              {{ option.label }}
            </option>
          </select>
        </label>
      </div>
    </div>

    <AccountResolvedUpstreamPanel
      :upstream-url="resolvedUpstream?.upstream_url"
      :upstream-host="resolvedUpstream?.upstream_host"
      :upstream-service="resolvedUpstream?.upstream_service"
      :upstream-region="resolvedUpstream?.upstream_region"
      :probe-source="resolvedUpstream?.upstream_probe_source"
      :probed-at="resolvedUpstream?.upstream_probed_at"
    />

    <div
      v-if="probedModels.length > 0"
      class="flex flex-wrap items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400"
    >
      <span>{{ t('admin.accounts.protocolGateway.probeSelectionHint') }}</span>
      <div class="flex flex-wrap items-center gap-2">
        <span>{{ t('admin.accounts.protocolGateway.probeSelectedCount', { count: allowedModels.length }) }}</span>
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-3 py-1.5 font-medium text-gray-700 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
          @click="selectAllCurrentResults"
        >
          {{ t('admin.accounts.protocolGateway.selectAllCurrentResults') }}
        </button>
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-3 py-1.5 font-medium text-gray-700 transition hover:border-rose-300 hover:text-rose-600 dark:border-dark-500 dark:text-gray-200 dark:hover:border-rose-500 dark:hover:text-rose-300"
          @click="clearSelectedModels"
        >
          {{ t('admin.accounts.protocolGateway.clearSelection') }}
        </button>
      </div>
    </div>

    <div v-if="probedModels.length > 0" class="space-y-4">
      <section
        v-for="group in groupedProbedModels"
        :key="group.protocol"
        class="space-y-3"
      >
        <header
          v-if="shouldShowProtocolGrouping"
          class="flex items-center justify-between gap-3"
        >
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ group.label }}
            </span>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ group.requestFormats }}
            </span>
          </div>
          <span class="text-xs text-gray-400 dark:text-gray-500">
            {{ group.models.length }}
          </span>
        </header>

        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <button
            v-for="model in group.models"
            :key="`${group.protocol}:${model.id}`"
            type="button"
            :title="model.id"
            :class="cardClasses(model)"
            @click="toggleModel(model)"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <AccountProbeModelIdentity
                  :model-id="model.id"
                  :display-name="displayModelName(model)"
                  :provider="resolveProvider(model)"
                  :provider-text="resolveProviderLabel(model)"
                />
                <div class="mt-2 flex flex-wrap items-center gap-1.5 text-[11px]">
                  <span
                    v-if="resolveProviderLabel(model)"
                    class="inline-flex items-center rounded-full bg-white/70 px-2 py-0.5 font-medium text-slate-700 dark:bg-white/10 dark:text-slate-200"
                  >
                    {{ resolveProviderLabel(model) }}
                  </span>
                  <span
                    v-if="resolveModelProtocol(model)"
                    class="inline-flex items-center rounded-full bg-white/70 px-2 py-0.5 font-medium text-slate-700 dark:bg-white/10 dark:text-slate-200"
                  >
                    {{ protocolLabel(resolveModelProtocol(model)) }}
                  </span>
                  <span
                    v-if="resolveModelProtocol(model) === 'gemini'"
                    class="inline-flex items-center rounded-full bg-sky-500/15 px-2 py-0.5 font-medium text-sky-700 dark:text-sky-200"
                  >
                    {{ t('admin.accounts.protocolGateway.geminiCompatibilityBadge') }}
                  </span>
                  <span
                    v-if="currentRouteProfile(model)"
                    class="inline-flex items-center rounded-full bg-primary-500/15 px-2 py-0.5 font-medium text-primary-700 dark:text-primary-200"
                  >
                    {{ clientProfileLabel(currentRouteProfile(model)!) }}
                  </span>
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
              <span class="break-words text-left">
                {{
                  model.registry_state === 'existing'
                    ? t('admin.accounts.protocolGateway.registryExisting')
                    : t('admin.accounts.protocolGateway.registryMissing')
                }}
              </span>
              <span v-if="model.registry_model_id" class="break-words text-right opacity-80" :title="model.registry_model_id">
                {{ model.registry_model_id }}
              </span>
            </div>

            <div
              v-if="isSelected(model.id) && availableProfilesForModel(model).length > 0"
              class="mt-3 flex flex-wrap items-center gap-2"
              @click.stop
            >
              <button
                v-for="profile in availableProfilesForModel(model)"
                :key="`${model.id}-${profile}`"
                type="button"
                class="rounded-lg border px-2 py-1 text-[11px] font-medium transition"
                :class="routeButtonClass(model, profile)"
                @click.stop="setModelClientProfile(model, profile)"
              >
                {{ clientProfileLabel(profile) }}
              </button>
              <button
                v-if="currentRouteProfile(model)"
                type="button"
                class="rounded-lg border border-white/60 px-2 py-1 text-[11px] font-medium text-gray-700 transition hover:border-rose-300 hover:text-rose-600 dark:border-white/10 dark:text-gray-200 dark:hover:border-rose-500 dark:hover:text-rose-300"
                @click.stop="setModelClientProfile(model, '')"
              >
                {{ t('admin.accounts.protocolGateway.clearSimulation') }}
              </button>
            </div>

            <div
              v-if="isSelected(model.id)"
              class="mt-3 space-y-2 rounded-xl border border-white/60 bg-white/60 p-3 dark:border-white/10 dark:bg-white/5"
              @click.stop
            >
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
      </section>
    </div>

    <div
      v-else
      class="rounded-xl border border-dashed border-gray-300 px-3 py-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400"
    >
      {{ t('admin.accounts.protocolGateway.probeEmpty') }}
    </div>

    <AccountManualModelsEditor
      v-model:rows="manualModels"
      :allow-source-protocol="true"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  AccountManualModel,
  ProtocolGatewayProbeModel,
  ProtocolGatewayProbeResponse
} from '@/api/admin/accounts'
import AccountManualModelsEditor from '@/components/account/AccountManualModelsEditor.vue'
import AccountResolvedUpstreamPanel from '@/components/account/AccountResolvedUpstreamPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type {
  GatewayAcceptedProtocol,
  GatewayClientProfile,
  GatewayClientRoute,
  GatewayProtocol
} from '@/types'
import type { ModelMapping } from '@/utils/accountFormShared'
import {
  PROTOCOL_GATEWAY_ACCEPTED_PROTOCOLS,
  normalizeGatewayAcceptedProtocol,
  normalizeGatewayAcceptedProtocols,
  normalizeGatewayClientRoutes,
  resolveGatewayProtocolDescriptor,
  supportedGatewayClientProfilesForProtocol
} from '@/utils/accountProtocolGateway'
import {
  createAccountModelProbeSnapshotDraft,
  createResolvedUpstreamDraft,
  type AccountModelProbeSnapshotDraft,
  type AccountResolvedUpstreamDraft
} from '@/utils/accountProbeDraft'
import { checkProtocolGatewayBaseUrl } from '@/utils/protocolGatewayBaseUrl'
import AccountProbeModelIdentity from '@/components/account/AccountProbeModelIdentity.vue'
import { formatModelDisplayName } from '@/utils/modelDisplayName'
import { formatProviderLabel, normalizeProviderSlug } from '@/utils/providerLabels'

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
const modelMappings = defineModel<ModelMapping[]>('modelMappings', { required: true })
const probedModels = defineModel<ProtocolGatewayProbeModel[]>('probedModels', { required: true })
const acceptedProtocols = defineModel<GatewayAcceptedProtocol[]>('acceptedProtocols', { required: true })
const clientProfiles = defineModel<GatewayClientProfile[]>('clientProfiles', { required: true })
const clientRoutes = defineModel<GatewayClientRoute[]>('clientRoutes', { required: true })
const manualModels = defineModel<AccountManualModel[]>('manualModels', { required: true })
const resolvedUpstream = defineModel<AccountResolvedUpstreamDraft | null>('resolvedUpstream', { required: true })
const probeSnapshot = defineModel<AccountModelProbeSnapshotDraft | null>('probeSnapshot', { default: null })
const gatewayTestProvider = defineModel<string>('gatewayTestProvider', { default: '' })
const gatewayTestModelId = defineModel<string>('gatewayTestModelId', { default: '' })

const { t } = useI18n()
const appStore = useAppStore()

const probing = ref(false)
const probeSource = ref('')
const probeNotice = ref('')
const aliasDrafts = ref<Record<string, string>>({})
const hasInitializedFromMappings = ref(false)

const trimmedApiKey = computed(() => props.apiKey.trim())
const acceptedProtocolOptions = computed(() =>
  ['openai', 'anthropic', 'gemini'].map((value) => {
    const descriptor = resolveGatewayProtocolDescriptor(value)
    return {
      value: value as GatewayAcceptedProtocol,
      label: descriptor?.displayName || value,
      requestFormats: (descriptor?.requestFormats || []).join(', ')
    }
  })
)
const normalizedAcceptedProtocols = computed(() =>
  normalizeGatewayAcceptedProtocols(props.gatewayProtocol, acceptedProtocols.value)
)
const availableClientProfiles = computed<GatewayClientProfile[]>(() => {
  const values = normalizedAcceptedProtocols.value.flatMap((protocol) =>
    supportedGatewayClientProfilesForProtocol(protocol)
  )
  return [...new Set(values)]
})
const shouldShowProtocolGrouping = computed(() => props.gatewayProtocol === 'mixed')
const selectedModelTargets = computed(() => {
  const values = new Set<string>()
  for (const modelId of allowedModels.value) {
    const normalized = String(modelId || '').trim()
    if (normalized) {
      values.add(normalized)
    }
  }
  for (const mapping of modelMappings.value) {
    const normalized = String(mapping.to || '').trim()
    if (normalized) {
      values.add(normalized)
    }
  }
  return values
})
const selectedProbedModels = computed(() =>
  probedModels.value.filter((model) => selectedModelTargets.value.has(String(model.id || '').trim()))
)
const providerOptions = computed(() => {
  const seen = new Set<string>()
  return [...selectedProbedModels.value]
    .filter((model) => {
      const provider = normalizeProviderSlug(model.provider)
      if (!provider || seen.has(provider)) {
        return false
      }
      seen.add(provider)
      return true
    })
    .map((model) => ({
      provider: normalizeProviderSlug(model.provider),
      label: resolveProviderLabel(model)
    }))
    .sort((left, right) => left.label.localeCompare(right.label))
})
const testModelOptions = computed(() => {
  const provider = normalizeProviderSlug(gatewayTestProvider.value)
  if (!provider) {
    return []
  }
  return [...selectedProbedModels.value]
    .filter((model) => normalizeProviderSlug(model.provider) === provider)
    .sort((left, right) => displayModelName(left).localeCompare(displayModelName(right)))
    .map((model) => ({
      id: model.id,
      label: displayModelName(model)
    }))
})

watch(
  () => [props.gatewayProtocol, props.baseUrl, props.apiKey, props.proxyId, probedModels.value.length] as const,
  ([, , , , modelCount]) => {
    if (modelCount === 0) {
      probeSource.value = ''
      probeNotice.value = ''
    }
  }
)

watch(
  () => props.gatewayProtocol,
  () => {
    const nextProtocols = normalizeGatewayAcceptedProtocols(props.gatewayProtocol, acceptedProtocols.value)
    if (JSON.stringify(nextProtocols) !== JSON.stringify(acceptedProtocols.value)) {
      acceptedProtocols.value = nextProtocols
    }
  },
  { immediate: true }
)

watch(
  [normalizedAcceptedProtocols, availableClientProfiles],
  () => {
    const supportedProfiles = new Set(availableClientProfiles.value)
    const nextProfiles = clientProfiles.value.filter((profile) => supportedProfiles.has(profile))
    if (JSON.stringify(nextProfiles) !== JSON.stringify(clientProfiles.value)) {
      clientProfiles.value = nextProfiles
    }

    const selectedProfiles = new Set(nextProfiles)
    const nextRoutes = normalizeGatewayClientRoutes(clientRoutes.value).filter((route) => {
      return (
        normalizedAcceptedProtocols.value.includes(route.protocol) &&
        selectedProfiles.has(route.client_profile) &&
        supportedGatewayClientProfilesForProtocol(route.protocol).includes(route.client_profile)
      )
    })
    if (JSON.stringify(nextRoutes) !== JSON.stringify(clientRoutes.value)) {
      clientRoutes.value = nextRoutes
    }
  },
  { immediate: true }
)

watch(
  probeSnapshot,
  (snapshot) => {
    if (!snapshot) {
      return
    }
    if (probedModels.value.length === 0) {
      probedModels.value = snapshot.models.map((modelId) => ({
        id: modelId,
        display_name: formatModelDisplayName(modelId) || modelId,
        registry_state: 'existing' as const,
        registry_model_id: modelId
      }))
    }
    if (!probeSource.value) {
      probeSource.value = snapshot.probe_source || snapshot.source || ''
    }
    if (!resolvedUpstream.value?.upstream_probed_at && snapshot.updated_at) {
      resolvedUpstream.value = {
        ...(resolvedUpstream.value || {}),
        upstream_probed_at: snapshot.updated_at
      }
    }
  },
  { immediate: true, deep: true }
)

watch(
  [providerOptions, testModelOptions],
  () => {
    const provider = normalizeProviderSlug(gatewayTestProvider.value)
    if (provider && !providerOptions.value.some((option) => option.provider === provider)) {
      gatewayTestProvider.value = ''
      gatewayTestModelId.value = ''
      return
    }
    if (gatewayTestModelId.value && !testModelOptions.value.some((option) => option.id === gatewayTestModelId.value)) {
      gatewayTestModelId.value = ''
    }
  },
  { immediate: true }
)

watch(
  () => [allowedModels.value.join('\x00'), modelMappings.value.length] as const,
  () => {
    const selected = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
    const nextMappings = modelMappings.value.filter((row) => selected.has(row.to.trim()))
    if (nextMappings.length !== modelMappings.value.length) {
      modelMappings.value = nextMappings
    }
    const nextRoutes = clientRoutes.value.filter((route) => selected.has(route.match_value.trim()))
    if (nextRoutes.length !== clientRoutes.value.length) {
      clientRoutes.value = normalizeGatewayClientRoutes(nextRoutes)
    }
  },
  { immediate: true }
)

watch(
  () => [probedModels.value.length, modelMappings.value.length] as const,
  ([modelCount, mappingCount]) => {
    if (hasInitializedFromMappings.value || modelCount > 0 || mappingCount === 0) {
      return
    }
    hasInitializedFromMappings.value = true
    const seen = new Set<string>()
    const nextDrafts = { ...aliasDrafts.value }
    for (const row of modelMappings.value) {
      const targetModel = row.to.trim()
      if (!targetModel || Object.prototype.hasOwnProperty.call(nextDrafts, targetModel)) {
        continue
      }
      nextDrafts[targetModel] = row.from
    }
    aliasDrafts.value = nextDrafts
    probedModels.value = modelMappings.value
      .map((row) => row.to.trim())
      .filter((modelId) => {
        if (!modelId || seen.has(modelId)) {
          return false
        }
        seen.add(modelId)
        return true
      })
      .map((modelId) => ({
        id: modelId,
        display_name: formatModelDisplayName(modelId) || modelId,
        registry_state: 'existing' as const,
        registry_model_id: modelId
      }))
  },
  { immediate: true }
)

const isSelected = (modelId: string) => allowedModels.value.includes(modelId)
const isAcceptedProtocolSelected = (protocol: GatewayAcceptedProtocol) =>
  normalizedAcceptedProtocols.value.includes(protocol)
const isClientProfileSelected = (profile: GatewayClientProfile) =>
  clientProfiles.value.includes(profile)

const findMappingIndexByTarget = (modelId: string) =>
  modelMappings.value.findIndex((row) => row.to.trim() === modelId)

const ensureMappingForModel = (model: ProtocolGatewayProbeModel) => {
  const index = findMappingIndexByTarget(model.id)
  if (index >= 0) {
    const nextMappings = [...modelMappings.value]
    const current = nextMappings[index]
    const draftAlias = Object.prototype.hasOwnProperty.call(aliasDrafts.value, model.id)
      ? aliasDrafts.value[model.id]
      : current.from
    nextMappings[index] = {
      from: draftAlias,
      to: model.id
    }
    modelMappings.value = nextMappings
    return
  }
  const nextAlias = Object.prototype.hasOwnProperty.call(aliasDrafts.value, model.id)
    ? aliasDrafts.value[model.id]
    : model.id
  modelMappings.value = [
    ...modelMappings.value,
    {
      from: nextAlias,
      to: model.id
    }
  ]
}

const toggleModel = (model: ProtocolGatewayProbeModel) => {
  const nextSelected = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
  if (nextSelected.has(model.id)) {
    nextSelected.delete(model.id)
  } else {
    nextSelected.add(model.id)
  }
  syncSelectedModels(
    probedModels.value.filter((item) => nextSelected.has(item.id)).map((item) => item.id)
  )
}

const toggleAcceptedProtocol = (protocol: GatewayAcceptedProtocol) => {
  if (props.gatewayProtocol !== 'mixed') {
    acceptedProtocols.value = normalizeGatewayAcceptedProtocols(props.gatewayProtocol, [protocol])
    return
  }

  const next = isAcceptedProtocolSelected(protocol)
    ? acceptedProtocols.value.filter((item) => item !== protocol)
    : [...acceptedProtocols.value, protocol]

  acceptedProtocols.value = normalizeGatewayAcceptedProtocols(props.gatewayProtocol, next)
}

const toggleClientProfile = (profile: GatewayClientProfile) => {
  if (isClientProfileSelected(profile)) {
    clientProfiles.value = clientProfiles.value.filter((item) => item !== profile)
    clientRoutes.value = clientRoutes.value.filter((route) => route.client_profile !== profile)
    return
  }
  clientProfiles.value = [...clientProfiles.value, profile]
}

function resolveModelProtocol(model: ProtocolGatewayProbeModel): GatewayAcceptedProtocol {
  const sourceProtocol = normalizeGatewayAcceptedProtocol(model.source_protocol)
  if (sourceProtocol) {
    return sourceProtocol
  }
  if (props.gatewayProtocol !== 'mixed') {
    return props.gatewayProtocol as GatewayAcceptedProtocol
  }
  return normalizedAcceptedProtocols.value[0] || 'openai'
}

function protocolLabel(protocol: GatewayAcceptedProtocol) {
  return resolveGatewayProtocolDescriptor(protocol)?.displayName || protocol
}

function resolveProvider(model: ProtocolGatewayProbeModel) {
  return normalizeProviderSlug(model.provider)
}

function resolveProviderLabel(model: ProtocolGatewayProbeModel) {
  const provider = resolveProvider(model)
  const providerLabel = String(model.provider_label || '').trim()
  if (!provider && !providerLabel) {
    return ''
  }
  return formatProviderLabel(provider, providerLabel)
}

function displayModelName(model: ProtocolGatewayProbeModel) {
  return String(model.display_name || '').trim() || formatModelDisplayName(model.id) || model.id
}

const groupedProbedModels = computed(() => {
  const grouped = new Map<GatewayAcceptedProtocol, ProtocolGatewayProbeModel[]>()
  for (const model of probedModels.value) {
    const protocol = resolveModelProtocol(model)
    const bucket = grouped.get(protocol)
    if (bucket) {
      bucket.push(model)
      continue
    }
    grouped.set(protocol, [model])
  }

  const orderedProtocols = PROTOCOL_GATEWAY_ACCEPTED_PROTOCOLS.filter((protocol) => grouped.has(protocol))
  return orderedProtocols.map((protocol) => {
    const descriptor = resolveGatewayProtocolDescriptor(protocol)
    const models = [...(grouped.get(protocol) || [])].sort((left, right) => displayModelName(left).localeCompare(displayModelName(right)))
    return {
      protocol,
      label: descriptor?.displayName || protocol,
      requestFormats: (descriptor?.requestFormats || []).join(', '),
      models
    }
  })
})

const updateGatewayTestProvider = (provider: string) => {
  gatewayTestProvider.value = normalizeProviderSlug(provider)
  gatewayTestModelId.value = ''
}

const updateGatewayTestModelId = (modelId: string) => {
  gatewayTestModelId.value = String(modelId || '').trim()
}

const routeKeyForModel = (model: ProtocolGatewayProbeModel) =>
  `${resolveModelProtocol(model)}:${model.id}`

const routeProfileMap = computed(() => {
  const map = new Map<string, GatewayClientProfile>()
  for (const route of clientRoutes.value) {
    if (route.match_type !== 'exact') {
      continue
    }
    map.set(`${route.protocol}:${route.match_value}`, route.client_profile)
  }
  return map
})

const currentRouteProfile = (model: ProtocolGatewayProbeModel) =>
  routeProfileMap.value.get(routeKeyForModel(model))

const currentAlias = (modelId: string) => {
  if (Object.prototype.hasOwnProperty.call(aliasDrafts.value, modelId)) {
    return aliasDrafts.value[modelId]
  }
  return modelMappings.value.find((row) => row.to.trim() === modelId)?.from ?? modelId
}

const collectAliasByTarget = () =>
  new Map(
    modelMappings.value
      .map((row) => [row.to.trim(), currentAlias(row.to.trim())] as const)
      .filter(([target]) => Boolean(target))
  )

const buildMappingsForModelIds = (modelIds: string[], aliasByTarget = collectAliasByTarget()) =>
  modelIds.map((modelId) => ({
    from: aliasByTarget.get(modelId) || currentAlias(modelId),
    to: modelId
  }))

const syncSelectedModels = (modelIds: string[], aliasByTarget = collectAliasByTarget()) => {
  const nextAllowedModels = [...new Set(modelIds.map((item) => item.trim()).filter(Boolean))]
  allowedModels.value = nextAllowedModels
  modelMappings.value = buildMappingsForModelIds(nextAllowedModels, aliasByTarget)
}

const updateModelAlias = (model: ProtocolGatewayProbeModel, value: string) => {
  aliasDrafts.value = {
    ...aliasDrafts.value,
    [model.id]: value
  }
  ensureMappingForModel(model)
  modelMappings.value = modelMappings.value.map((row) =>
    row.to.trim() === model.id
      ? {
          from: value,
          to: model.id
        }
      : row
  )
}

const availableProfilesForModel = (model: ProtocolGatewayProbeModel) => {
  const protocol = resolveModelProtocol(model)
  return supportedGatewayClientProfilesForProtocol(protocol).filter((profile) =>
    clientProfiles.value.includes(profile)
  )
}

const setModelClientProfile = (
  model: ProtocolGatewayProbeModel,
  profile: GatewayClientProfile | ''
) => {
  const protocol = resolveModelProtocol(model)
  const nextRoutes = clientRoutes.value.filter(
    (route) =>
      !(route.match_type === 'exact' && route.protocol === protocol && route.match_value === model.id)
  )

  if (profile) {
    if (!clientProfiles.value.includes(profile)) {
      clientProfiles.value = [...clientProfiles.value, profile]
    }
    nextRoutes.push({
      protocol,
      match_type: 'exact',
      match_value: model.id,
      client_profile: profile
    })
  }

  clientRoutes.value = normalizeGatewayClientRoutes(nextRoutes)
}

const supportsProfile = (profile: GatewayClientProfile) =>
  availableClientProfiles.value.includes(profile)

const applyProfileToAll = (profile: GatewayClientProfile) => {
  if (!supportsProfile(profile)) {
    return
  }
  if (!clientProfiles.value.includes(profile)) {
    clientProfiles.value = [...clientProfiles.value, profile]
  }
  const nextRoutes = clientRoutes.value.filter((route) => route.match_type !== 'exact')
  for (const model of probedModels.value.filter((item) => isSelected(item.id))) {
    const protocol = resolveModelProtocol(model)
    if (!supportedGatewayClientProfilesForProtocol(protocol).includes(profile)) {
      continue
    }
    nextRoutes.push({
      protocol,
      match_type: 'exact',
      match_value: model.id,
      client_profile: profile
    })
  }
  clientRoutes.value = normalizeGatewayClientRoutes(nextRoutes)
}

const clearAllProfiles = () => {
  clientRoutes.value = clientRoutes.value.filter((route) => route.match_type !== 'exact')
}

const selectAllCurrentResults = () => {
  syncSelectedModels(probedModels.value.map((model) => model.id))
}

const clearSelectedModels = () => {
  allowedModels.value = []
  modelMappings.value = []
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

const selectionCardClass = (selected: boolean) => [
  'flex min-w-0 flex-col items-start gap-1 rounded-2xl border px-4 py-3 text-left transition',
  selected
    ? 'border-primary-300 bg-primary-50 text-primary-900 ring-2 ring-primary-400/40 dark:border-primary-500/60 dark:bg-primary-500/10 dark:text-primary-100'
    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-200 hover:text-primary-600 dark:border-dark-500 dark:bg-dark-700/80 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-200'
]

const routeButtonClass = (model: ProtocolGatewayProbeModel, profile: GatewayClientProfile) => {
  const active = currentRouteProfile(model) === profile
  return active
    ? 'border-primary-300 bg-primary-500 text-white dark:border-primary-400'
    : 'border-white/60 text-gray-700 hover:border-primary-300 hover:text-primary-600 dark:border-white/10 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-200'
}

const clientProfileLabel = (profile: GatewayClientProfile) =>
  profile === 'codex'
    ? t('admin.accounts.protocolGateway.clientProfileCodex')
    : t('admin.accounts.protocolGateway.clientProfileGeminiCli')

const clientProfileHint = (profile: GatewayClientProfile) =>
  profile === 'codex'
    ? t('admin.accounts.protocolGateway.clientProfileCodexHint')
    : t('admin.accounts.protocolGateway.clientProfileGeminiCliHint')

const handleProbe = async () => {
  if (probing.value) {
    return
  }
  if (!trimmedApiKey.value) {
    appStore.showWarning(t('admin.accounts.protocolGateway.probeRequiredApiKey'))
    return
  }

  const baseUrlCheck = checkProtocolGatewayBaseUrl(props.baseUrl)
  if (baseUrlCheck.status === 'invalid') {
    appStore.showWarning(t('admin.accounts.protocolGateway.baseUrlInvalidWarning'))
    return
  }
  if (baseUrlCheck.status === 'loopback') {
    appStore.showWarning(
      t('admin.accounts.protocolGateway.baseUrlLoopbackWarning', {
        host: baseUrlCheck.displayHost || baseUrlCheck.input
      })
    )
    return
  }

  probing.value = true
  try {
    const result = await adminAPI.accounts.probeProtocolGatewayModels({
      gateway_protocol: props.gatewayProtocol,
      accepted_protocols: normalizedAcceptedProtocols.value,
      base_url: props.baseUrl.trim() || undefined,
      api_key: trimmedApiKey.value,
      target_provider: gatewayTestProvider.value || undefined,
      target_model_id: gatewayTestModelId.value || undefined,
      manual_models: manualModels.value,
      proxy_id: props.proxyId ?? undefined
    })
    const aliasByTarget = collectAliasByTarget()
    const probedAt = new Date().toISOString()
    const selectedTargets = new Set(allowedModels.value.map((item) => item.trim()).filter(Boolean))
    const nextAllowedModels = result.models
      .map((model) => model.id)
      .filter((modelId) => selectedTargets.has(modelId))
    probedModels.value = result.models
    syncSelectedModels(nextAllowedModels, aliasByTarget)
    probeSource.value = result.probe_source || ''
    probeNotice.value = result.probe_notice || ''
    probeSnapshot.value = createAccountModelProbeSnapshotDraft({
      models: result.models.map((model) => model.id),
      updated_at: probedAt,
      source: 'manual_probe',
      probe_source: result.probe_source
    })
    resolvedUpstream.value =
      createResolvedUpstreamDraft({
        upstream_url: result.resolved_upstream_url,
        upstream_host: result.resolved_upstream_host,
        upstream_service: result.resolved_upstream_service,
        upstream_probe_source: result.probe_source,
        upstream_probed_at: probedAt
      }) || resolvedUpstream.value
    emit('probed', result)
  } catch (error: any) {
    console.error('Failed to probe protocol gateway models:', error)
    appStore.showError(error?.message || t('admin.accounts.protocolGateway.probeFailed'))
  } finally {
    probing.value = false
  }
}
</script>
