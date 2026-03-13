<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>

    <div v-if="disabled" class="mb-3 rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20">
      <p class="text-xs text-amber-700 dark:text-amber-400">
        {{ t('admin.accounts.openai.modelRestrictionDisabledByPassthrough') }}
      </p>
    </div>

    <template v-else>
      <div class="mb-4 flex gap-2">
        <button
          type="button"
          @click="emit('update:mode', 'whitelist')"
          :class="mode === 'whitelist' ? activeModeClass('primary') : inactiveModeClass"
          class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all"
        >
          {{ t('admin.accounts.modelWhitelist') }}
        </button>
        <button
          type="button"
          @click="emit('update:mode', 'mapping')"
          :class="mode === 'mapping' ? activeModeClass('purple') : inactiveModeClass"
          class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all"
        >
          {{ t('admin.accounts.modelMapping') }}
        </button>
      </div>

      <div v-if="mode === 'whitelist'" class="space-y-3">
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.selectAllowedModels') }}
        </p>
        <div class="flex flex-wrap gap-2">
          <input
            v-model="searchQuery"
            type="text"
            class="input min-w-[220px] flex-1"
            :placeholder="t('admin.accounts.searchModels')"
          />
          <button type="button" class="btn btn-secondary" @click="emit('update:allowedModels', [])">
            {{ t('admin.accounts.clearAllModels') }}
          </button>
        </div>

        <div v-if="providerGroups.length === 0" class="rounded-lg border border-dashed border-gray-300 p-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400">
          {{ t('admin.accounts.noMatchingModels') }}
        </div>

        <div
          v-for="group in providerGroups"
          :key="group.provider"
          class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
        >
          <div class="mb-3 flex flex-wrap items-start justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ group.label }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ group.selectedCount }}/{{ group.entries.length }} {{ t('common.selectedCount', { count: group.selectedCount }) }}
              </div>
            </div>
            <div class="flex gap-2">
              <button type="button" class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700" @click="selectProvider(group.provider)">
                {{ t('common.all') }}
              </button>
              <button type="button" class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700" @click="clearProvider(group.provider)">
                {{ t('common.none') }}
              </button>
            </div>
          </div>

          <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
            <button
              v-for="entry in group.entries"
              :key="entry.id"
              type="button"
              class="flex items-start gap-3 rounded-lg border px-3 py-3 text-left transition-colors"
              :class="selectedSet.has(entry.id) ? 'border-primary-500 bg-primary-50 dark:border-primary-500 dark:bg-primary-900/20' : 'border-gray-200 bg-white hover:border-gray-300 dark:border-dark-600 dark:bg-dark-700 dark:hover:border-dark-500'"
              @click="toggleModel(entry.id)"
            >
              <span
                class="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border"
                :class="selectedSet.has(entry.id) ? 'border-primary-500 bg-primary-500 text-white' : 'border-gray-300 dark:border-dark-500'"
              >
                <svg v-if="selectedSet.has(entry.id)" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
                </svg>
              </span>
              <ModelIcon :model="entry.id" size="18px" />
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ entry.display_name || entry.id }}</span>
                  <span v-if="entry.status && entry.status !== 'stable'" class="shrink-0 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium uppercase text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                    {{ entry.status }}
                  </span>
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ entry.id }}</div>
              </div>
            </button>
          </div>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.selectedModels', { count: allowedModels.length }) }}
          <span v-if="allowedModels.length === 0">{{ t('admin.accounts.supportsAllModels') }}</span>
        </p>
      </div>

      <div v-else>
        <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
          <p class="text-xs text-purple-700 dark:text-purple-400">
            {{ t('admin.accounts.mapRequestModels') }}
          </p>
        </div>

        <div v-if="modelMappings.length > 0" class="mb-3 space-y-2">
          <div
            v-for="(mapping, index) in modelMappings"
            :key="getMappingKey(mapping)"
            class="flex items-center gap-2"
          >
            <input v-model="mapping.from" type="text" class="input flex-1" :placeholder="t('admin.accounts.requestModel')" />
            <svg class="h-4 w-4 flex-shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
            </svg>
            <input v-model="mapping.to" type="text" class="input flex-1" :placeholder="t('admin.accounts.actualModel')" />
            <button type="button" class="rounded-lg p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20" @click="emit('remove-mapping', index)">
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>

        <button type="button" class="mb-3 w-full rounded-lg border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300" @click="emit('add-mapping')">
          + {{ t('admin.accounts.addMapping') }}
        </button>

        <div class="flex flex-wrap gap-2">
          <button
            v-for="preset in presetMappings"
            :key="`${preset.label}-${preset.from}-${preset.to}`"
            type="button"
            class="rounded-lg px-3 py-1 text-xs transition-colors"
            :class="preset.color"
            @click="emit('add-preset', { from: preset.from, to: preset.to })"
          >
            + {{ preset.label }}
          </button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { ensureModelRegistryFresh, getModelRegistrySnapshot } from '@/stores/modelRegistry'
import type { ModelRegistryEntry, ModelRegistryPreset } from '@/generated/modelRegistry'
import type { ModelMapping } from '@/utils/accountFormShared'

interface Props {
  disabled?: boolean
  platform: string
  mode: 'whitelist' | 'mapping'
  allowedModels: string[]
  modelMappings: ModelMapping[]
  presetMappings: ModelRegistryPreset[]
  getMappingKey: (mapping: ModelMapping) => string
}

const props = withDefaults(defineProps<Props>(), { disabled: false })
const emit = defineEmits<{
  'update:mode': [value: 'whitelist' | 'mapping']
  'update:allowedModels': [value: string[]]
  'add-mapping': []
  'remove-mapping': [index: number]
  'add-preset': [payload: { from: string; to: string }]
}>()

const { t } = useI18n()
const searchQuery = ref('')
const selectedSet = computed(() => new Set(props.allowedModels))

const providerGroups = computed(() => {
  void ensureModelRegistryFresh()
  const snapshot = getModelRegistrySnapshot()
  const query = searchQuery.value.trim().toLowerCase()
  const normalizedPlatform = props.platform.trim().toLowerCase() === 'claude' ? 'anthropic' : props.platform.trim().toLowerCase()
  const groups = new Map<string, { provider: string; label: string; entries: ModelRegistryEntry[] }>()

  for (const entry of snapshot.models) {
    if (!entry.platforms.includes(normalizedPlatform)) continue
    if (query && !`${entry.id} ${entry.display_name} ${entry.provider}`.toLowerCase().includes(query)) continue
    const provider = (entry.provider || normalizedPlatform || 'unknown').trim().toLowerCase()
    const current = groups.get(provider) || { provider, label: formatProviderLabel(provider), entries: [] }
    current.entries.push(entry)
    groups.set(provider, current)
  }

  return [...groups.values()]
    .map((group) => ({
      ...group,
      entries: [...group.entries].sort((left, right) => (left.ui_priority - right.ui_priority) || left.id.localeCompare(right.id)),
      selectedCount: group.entries.filter((entry) => selectedSet.value.has(entry.id)).length
    }))
    .sort((left, right) => left.label.localeCompare(right.label))
})

const inactiveModeClass = 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'

function activeModeClass(color: 'primary' | 'purple') {
  return color === 'primary'
    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
    : 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
}

function toggleModel(modelId: string) {
  const next = new Set(props.allowedModels)
  next.has(modelId) ? next.delete(modelId) : next.add(modelId)
  emit('update:allowedModels', [...next].sort())
}

function selectProvider(provider: string) {
  const next = new Set(props.allowedModels)
  const group = providerGroups.value.find((item) => item.provider === provider)
  group?.entries.forEach((entry) => next.add(entry.id))
  emit('update:allowedModels', [...next].sort())
}

function clearProvider(provider: string) {
  const removeSet = new Set(providerGroups.value.find((item) => item.provider === provider)?.entries.map((entry) => entry.id) || [])
  emit('update:allowedModels', props.allowedModels.filter((modelId) => !removeSet.has(modelId)))
}

function formatProviderLabel(provider: string) {
  switch (provider) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    case 'antigravity':
      return 'Antigravity'
    case 'sora':
      return 'Sora'
    default:
      return provider ? provider.charAt(0).toUpperCase() + provider.slice(1) : 'Unknown'
  }
}
</script>
