<template>
  <BaseDialog :show="show" :title="t('admin.users.userApiKeys')" width="wide" @close="handleClose">
    <div v-if="user" class="space-y-4">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div
          class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30"
        >
          <span class="text-lg font-medium text-primary-700 dark:text-primary-300">
            {{ user.email.charAt(0).toUpperCase() }}
          </span>
        </div>
        <div>
          <p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ user.username }}</p>
        </div>
      </div>

      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          ></circle>
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
      </div>

      <div v-else-if="apiKeys.length === 0" class="py-8 text-center">
        <p class="text-sm text-gray-500">{{ t('admin.users.noApiKeys') }}</p>
      </div>

      <div v-else class="max-h-[32rem] space-y-3 overflow-y-auto">
        <div
          v-for="key in apiKeys"
          :key="key.id"
          class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <div class="mb-1 flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ key.name }}</span>
                <span
                  :class="[
                    'badge text-xs',
                    key.status === 'active' ? 'badge-success' : 'badge-danger'
                  ]"
                >
                  {{ key.status }}
                </span>
              </div>
              <p class="truncate font-mono text-sm text-gray-500">
                {{ key.key.substring(0, 20) }}...{{ key.key.substring(key.key.length - 8) }}
              </p>
            </div>

            <button
              type="button"
              class="btn btn-secondary"
              :disabled="savingKeyIds.has(key.id)"
              @click="toggleBindingsEditor(key)"
            >
              {{ isEditingBindings(key.id) ? t('common.cancel') : t('admin.users.editGroupBindings') }}
            </button>
          </div>

          <div class="mt-3 flex flex-wrap gap-4 text-xs text-gray-500">
            <div>{{ t('admin.users.columns.created') }}: {{ formatDateTime(key.created_at) }}</div>
            <div>{{ t('admin.users.groupBindings') }}: {{ getDisplayBindings(key).length }}</div>
          </div>

          <div class="mt-4">
            <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t('admin.users.groupBindings') }}
            </div>

            <div v-if="getDisplayBindings(key).length" class="flex flex-wrap gap-2">
              <div
                v-for="binding in getDisplayBindings(key)"
                :key="`${key.id}-${binding.group_id}`"
                class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-sm dark:border-dark-500 dark:bg-dark-700"
              >
                <div class="flex flex-wrap items-center gap-2">
                  <GroupBadge
                    v-if="resolveGroup(binding.group_id)"
                    :name="resolveGroup(binding.group_id)!.name"
                    :platform="resolveGroup(binding.group_id)!.platform"
                    :subscription-type="resolveGroup(binding.group_id)!.subscription_type"
                    :rate-multiplier="resolveGroup(binding.group_id)!.rate_multiplier"
                  />
                  <span v-else class="font-medium text-gray-800 dark:text-gray-100">
                    {{ binding.group_name || `#${binding.group_id}` }}
                  </span>
                  <span class="rounded-full bg-white px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-800">
                    P{{ binding.priority ?? resolveGroup(binding.group_id)?.priority ?? 1 }}
                  </span>
                </div>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.users.groupQuota') }}:
                  {{ formatQuotaValue(binding.quota) }}
                  <span class="mx-1">/</span>
                  {{ t('admin.users.groupQuotaUsed') }}:
                  {{ formatQuotaValue(binding.quota_used) }}
                </div>
                <div
                  v-if="binding.model_patterns?.length"
                  class="mt-2 rounded-lg bg-white px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
                >
                  {{ binding.model_patterns.join(', ') }}
                </div>
              </div>
            </div>

            <div
              v-else
              class="rounded-xl border border-dashed border-gray-200 px-3 py-4 text-sm text-gray-400 dark:border-dark-500"
            >
              {{ t('admin.users.noGroupBindings') }}
            </div>
          </div>

          <div
            v-if="isEditingBindings(key.id)"
            class="mt-4 space-y-3 rounded-xl border border-primary-100 bg-primary-50/60 p-4 dark:border-primary-900/40 dark:bg-primary-900/10"
          >
            <div
              v-for="(binding, index) in draftBindingsByKeyId[key.id] || []"
              :key="`${key.id}-${index}`"
              class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-800"
            >
              <div class="grid gap-3 lg:grid-cols-[minmax(0,1.4fr)_140px]">
                <div>
                  <label class="input-label">{{ t('admin.users.group') }}</label>
                  <select v-model.number="binding.group_id" class="input">
                    <option :value="0">{{ t('admin.users.selectGroupBinding') }}</option>
                    <option
                      v-for="group in sortedGroups"
                      :key="group.id"
                      :value="group.id"
                      :disabled="isGroupSelectedInOtherBinding(key.id, group.id, index)"
                    >
                      {{ group.name }} · {{ group.platform }} · P{{ group.priority ?? 1 }}
                    </option>
                  </select>
                </div>

                <div>
                  <label class="input-label">{{ t('admin.users.groupQuota') }}</label>
                  <input
                    v-model.number="binding.quota"
                    type="number"
                    min="0"
                    step="0.01"
                    class="input"
                    :placeholder="t('admin.users.groupQuotaPlaceholder')"
                  />
                </div>
              </div>

              <div class="mt-3">
                <label class="input-label">{{ t('admin.users.modelPatterns') }}</label>
                <textarea
                  v-model="binding.model_patterns_text"
                  rows="3"
                  class="input"
                  placeholder="claude-opus-*"
                ></textarea>
                <p class="input-hint">{{ t('admin.users.modelPatternsHint') }}</p>
              </div>

              <div class="mt-3 flex justify-end">
                <button
                  type="button"
                  class="btn btn-secondary"
                  @click="removeBindingRow(key.id, index)"
                >
                  {{ t('admin.users.removeGroupBinding') }}
                </button>
              </div>
            </div>

            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.users.groupQuotaHint') }}
              </div>

              <div class="flex flex-wrap items-center gap-2">
                <button type="button" class="btn btn-secondary" @click="addBindingRow(key.id)">
                  {{ t('admin.users.addGroupBinding') }}
                </button>
                <button type="button" class="btn btn-secondary" @click="cancelEditingBindings(key.id)">
                  {{ t('common.cancel') }}
                </button>
                <button
                  type="button"
                  class="btn btn-primary"
                  :disabled="savingKeyIds.has(key.id)"
                  @click="saveBindings(key)"
                >
                  <svg
                    v-if="savingKeyIds.has(key.id)"
                    class="mr-2 h-4 w-4 animate-spin"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  {{ savingKeyIds.has(key.id) ? t('common.saving') : t('common.save') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import type { AdminGroup, AdminUser, ApiKey, ApiKeyGroup, ApiKeyGroupBindingInput } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'

interface EditableBinding {
  group_id: number
  quota: number | '' | null
  model_patterns_text: string
}

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits(['close'])

const { t } = useI18n()
const appStore = useAppStore()

const apiKeys = ref<ApiKey[]>([])
const allGroups = ref<AdminGroup[]>([])
const loading = ref(false)
const savingKeyIds = ref(new Set<number>())
const editingKeyIds = ref<Record<number, boolean>>({})
const draftBindingsByKeyId = ref<Record<number, EditableBinding[]>>({})

const sortedGroups = computed(() =>
  [...allGroups.value].sort((a, b) => {
    const priorityDiff = (a.priority ?? 1) - (b.priority ?? 1)
    if (priorityDiff !== 0) return priorityDiff
    return a.name.localeCompare(b.name)
  })
)

const groupMap = computed(() => {
  return new Map(sortedGroups.value.map((group) => [group.id, group] as const))
})

watch(
  () => props.show,
  (visible) => {
    if (visible && props.user) {
      load()
      loadGroups()
      return
    }
    resetEditorState()
  }
)

const resetEditorState = () => {
  editingKeyIds.value = {}
  draftBindingsByKeyId.value = {}
  savingKeyIds.value = new Set<number>()
}

const load = async () => {
  if (!props.user) return
  loading.value = true
  try {
    const res = await adminAPI.users.getUserApiKeys(props.user.id)
    apiKeys.value = res.items || []
  } catch (error) {
    console.error('Failed to load API keys:', error)
    appStore.showError(t('admin.users.failedToLoadApiKeys'))
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  try {
    allGroups.value = await adminAPI.groups.getAll()
  } catch (error) {
    console.error('Failed to load groups:', error)
    appStore.showError(t('admin.users.failedToLoadGroups'))
  }
}

const resolveGroup = (groupId: number | null | undefined): AdminGroup | undefined => {
  if (!groupId) return undefined
  return groupMap.value.get(groupId)
}

const buildFallbackBinding = (key: ApiKey): ApiKeyGroup[] => {
  if (!key.group_id) return []
  const group = key.group || resolveGroup(key.group_id)
  return [
    {
      group_id: key.group_id,
      group_name: group?.name || `#${key.group_id}`,
      platform: group?.platform || 'anthropic',
      priority: group?.priority ?? 1,
      quota: 0,
      quota_used: 0,
      model_patterns: []
    }
  ]
}

const getDisplayBindings = (key: ApiKey): ApiKeyGroup[] => {
  const bindings = key.api_key_groups?.length ? key.api_key_groups : buildFallbackBinding(key)
  return [...bindings].sort((a, b) => {
    const priorityDiff = (a.priority ?? 1) - (b.priority ?? 1)
    if (priorityDiff !== 0) return priorityDiff
    return a.group_id - b.group_id
  })
}

const isEditingBindings = (keyId: number): boolean => {
  return Boolean(editingKeyIds.value[keyId])
}

const bindingToDraft = (binding: ApiKeyGroup): EditableBinding => ({
  group_id: binding.group_id,
  quota: binding.quota ?? 0,
  model_patterns_text: (binding.model_patterns || []).join('\n')
})

const startEditingBindings = (key: ApiKey) => {
  editingKeyIds.value = { ...editingKeyIds.value, [key.id]: true }
  draftBindingsByKeyId.value = {
    ...draftBindingsByKeyId.value,
    [key.id]: getDisplayBindings(key).map(bindingToDraft)
  }
}

const cancelEditingBindings = (keyId: number) => {
  const nextEditing = { ...editingKeyIds.value }
  const nextDrafts = { ...draftBindingsByKeyId.value }
  delete nextEditing[keyId]
  delete nextDrafts[keyId]
  editingKeyIds.value = nextEditing
  draftBindingsByKeyId.value = nextDrafts
}

const toggleBindingsEditor = (key: ApiKey) => {
  if (isEditingBindings(key.id)) {
    cancelEditingBindings(key.id)
    return
  }
  startEditingBindings(key)
}

const addBindingRow = (keyId: number) => {
  const currentRows = draftBindingsByKeyId.value[keyId] || []
  draftBindingsByKeyId.value = {
    ...draftBindingsByKeyId.value,
    [keyId]: [...currentRows, { group_id: 0, quota: 0, model_patterns_text: '' }]
  }
}

const removeBindingRow = (keyId: number, index: number) => {
  const currentRows = draftBindingsByKeyId.value[keyId] || []
  draftBindingsByKeyId.value = {
    ...draftBindingsByKeyId.value,
    [keyId]: currentRows.filter((_, currentIndex) => currentIndex !== index)
  }
}

const isGroupSelectedInOtherBinding = (keyId: number, groupId: number, currentIndex: number): boolean => {
  const bindings = draftBindingsByKeyId.value[keyId] || []
  return bindings.some((binding, index) => index !== currentIndex && binding.group_id === groupId)
}

const normalizeQuota = (value: number | '' | null | undefined): number => {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

const parseModelPatterns = (value: string): string[] => {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

const buildPayloadBindings = (keyId: number): ApiKeyGroupBindingInput[] => {
  const bindings = draftBindingsByKeyId.value[keyId] || []
  const seen = new Set<number>()
  const payload: ApiKeyGroupBindingInput[] = []

  for (const binding of bindings) {
    const groupId = Number(binding.group_id)
    if (!Number.isFinite(groupId) || groupId <= 0 || seen.has(groupId)) {
      continue
    }
    seen.add(groupId)
    payload.push({
      group_id: groupId,
      quota: normalizeQuota(binding.quota),
      model_patterns: parseModelPatterns(binding.model_patterns_text)
    })
  }

  return payload
}

const saveBindings = async (key: ApiKey) => {
  savingKeyIds.value = new Set(savingKeyIds.value).add(key.id)
  try {
    const payload = buildPayloadBindings(key.id)
    const result = await adminAPI.apiKeys.updateApiKeyGroup(key.id, { groups: payload })
    const index = apiKeys.value.findIndex((item) => item.id === key.id)
    if (index !== -1) {
      apiKeys.value[index] = result.api_key
    }
    cancelEditingBindings(key.id)

    if (result.auto_granted_group_access && result.granted_group_name) {
      appStore.showSuccess(t('admin.users.groupChangedWithGrant', { group: result.granted_group_name }))
    } else {
      appStore.showSuccess(t('admin.users.groupBindingsUpdated'))
    }
  } catch (error: any) {
    console.error('Failed to update API key group bindings:', error)
    appStore.showError(error?.response?.data?.detail || t('admin.users.groupBindingsUpdateFailed'))
  } finally {
    const nextSaving = new Set(savingKeyIds.value)
    nextSaving.delete(key.id)
    savingKeyIds.value = nextSaving
  }
}

const formatQuotaValue = (value: number | null | undefined): string => {
  const parsed = Number(value ?? 0)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return t('admin.users.groupQuotaUnlimited')
  }
  return parsed.toFixed(parsed >= 100 ? 0 : 2)
}

const handleClose = () => {
  resetEditorState()
  emit('close')
}
</script>
