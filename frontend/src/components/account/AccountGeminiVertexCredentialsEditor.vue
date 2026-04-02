<template>
  <section class="space-y-4 rounded-xl border border-sky-200 bg-sky-50/70 p-4 dark:border-sky-900/40 dark:bg-sky-950/20">
    <div class="space-y-1">
      <h3 class="text-sm font-semibold text-sky-950 dark:text-sky-100">
        {{ t('admin.accounts.gemini.vertex.formTitle') }}
      </h3>
      <p class="text-xs leading-5 text-sky-800 dark:text-sky-300">
        {{ t('admin.accounts.gemini.vertex.formDescription') }}
      </p>
    </div>

    <div class="space-y-2">
      <span class="input-label">{{ t('admin.accounts.gemini.vertex.authModeLabel') }}</span>
      <div class="grid gap-3 md:grid-cols-2">
        <button
          type="button"
          :class="modeCardClass('service_account')"
          @click="authMode = 'service_account'"
        >
          <div class="text-sm font-semibold">
            {{ t('admin.accounts.gemini.vertex.authModes.serviceAccountTitle') }}
          </div>
          <div class="mt-1 text-xs opacity-80">
            {{ t('admin.accounts.gemini.vertex.authModes.serviceAccountDesc') }}
          </div>
        </button>
        <button
          type="button"
          :class="modeCardClass('express_api_key')"
          @click="authMode = 'express_api_key'"
        >
          <div class="text-sm font-semibold">
            {{ t('admin.accounts.gemini.vertex.authModes.expressTitle') }}
          </div>
          <div class="mt-1 text-xs opacity-80">
            {{ t('admin.accounts.gemini.vertex.authModes.expressDesc') }}
          </div>
        </button>
      </div>
    </div>

    <template v-if="authMode === 'service_account'">
      <label class="space-y-2">
        <span class="input-label">{{ t('admin.accounts.gemini.vertex.serviceAccountJson') }}</span>
        <input
          ref="fileInputRef"
          type="file"
          accept=".json,application/json"
          class="hidden"
          @change="handleFileChange"
        />
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-secondary" @click="fileInputRef?.click()">
            {{ summary ? t('admin.accounts.gemini.vertex.replaceFile') : t('admin.accounts.gemini.vertex.uploadFile') }}
          </button>
          <span v-if="summary" class="text-xs text-sky-700 dark:text-sky-300">
            {{ summary.client_email }}
          </span>
        </div>
        <p class="input-hint">{{ t('admin.accounts.gemini.vertex.serviceAccountJsonHint') }}</p>
        <p v-if="uploadError" class="text-xs text-rose-600 dark:text-rose-300">{{ uploadError }}</p>
      </label>

      <div
        v-if="summary"
        class="grid gap-3 rounded-xl border border-sky-100 bg-white/70 p-3 text-xs text-sky-900 dark:border-sky-900/50 dark:bg-sky-950/30 dark:text-sky-100 md:grid-cols-2"
      >
        <div>
          <div class="font-medium">{{ t('admin.accounts.gemini.vertex.clientEmail') }}</div>
          <div class="break-all opacity-80">{{ summary.client_email }}</div>
        </div>
        <div>
          <div class="font-medium">{{ t('admin.accounts.gemini.vertex.privateKeyId') }}</div>
          <div class="break-all opacity-80">{{ summary.private_key_id || '-' }}</div>
        </div>
        <div>
          <div class="font-medium">{{ t('admin.accounts.gemini.vertex.projectIdFromFile') }}</div>
          <div class="break-all opacity-80">{{ summary.project_id || '-' }}</div>
        </div>
        <div>
          <div class="font-medium">{{ t('admin.accounts.gemini.vertex.tokenUri') }}</div>
          <div class="break-all opacity-80">{{ summary.token_uri }}</div>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-3">
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.gemini.vertex.projectId') }}</span>
          <input
            v-model="projectId"
            type="text"
            class="input"
            :placeholder="t('admin.accounts.gemini.vertex.projectIdPlaceholder')"
          />
          <p class="input-hint">{{ t('admin.accounts.gemini.vertex.projectIdHint') }}</p>
        </label>

        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.gemini.vertex.location') }}</span>
          <Select
            :model-value="location"
            :options="locationOptions"
            searchable
            @update:model-value="updateLocation"
          />
          <p class="input-hint">{{ t('admin.accounts.gemini.vertex.locationHint') }}</p>
        </label>

        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.gemini.vertex.baseUrl') }}</span>
          <input
            v-model="baseUrl"
            type="text"
            class="input"
            :placeholder="resolvedBaseUrl"
          />
          <p class="input-hint">{{ t('admin.accounts.gemini.vertex.baseUrlHint') }}</p>
        </label>
      </div>

      <div
        v-if="legacyMode"
        class="space-y-3 rounded-xl border border-amber-200 bg-amber-50/80 p-3 dark:border-amber-900/40 dark:bg-amber-950/20"
      >
        <div class="space-y-1">
          <div class="text-sm font-semibold text-amber-900 dark:text-amber-100">
            {{ t('admin.accounts.gemini.vertex.legacyTitle') }}
          </div>
          <p class="text-xs text-amber-800 dark:text-amber-300">
            {{ t('admin.accounts.gemini.vertex.legacyHint') }}
          </p>
        </div>

        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.gemini.vertex.accessToken') }}</span>
          <textarea
            v-model="legacyAccessToken"
            rows="3"
            class="input w-full resize-y font-mono text-sm"
            :placeholder="t('admin.accounts.gemini.vertex.accessTokenPlaceholderEdit')"
          ></textarea>
          <p class="input-hint">{{ t('admin.accounts.gemini.vertex.accessTokenHintEdit') }}</p>
        </label>

        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.gemini.vertex.expiresAt') }}</span>
          <input v-model="legacyExpiresAtInput" type="datetime-local" class="input" />
          <p class="input-hint">{{ t('admin.accounts.gemini.vertex.expiresAtHint') }}</p>
        </label>
      </div>
    </template>

    <template v-else>
      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.gemini.vertex.expressApiKey') }}</span>
          <textarea
            v-model="apiKey"
            rows="3"
            class="input w-full resize-y font-mono text-sm"
            :placeholder="t('admin.accounts.gemini.vertex.expressApiKeyPlaceholder')"
          ></textarea>
          <p class="input-hint">{{ t('admin.accounts.gemini.vertex.expressApiKeyHint') }}</p>
        </label>

        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.gemini.vertex.baseUrl') }}</span>
          <input
            v-model="baseUrl"
            type="text"
            class="input"
            :placeholder="resolvedBaseUrl"
          />
          <p class="input-hint">{{ t('admin.accounts.gemini.vertex.expressBaseUrlHint') }}</p>
        </label>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import {
  extractVertexServiceAccountSummary,
  normalizeVertexLocation,
  resolveVertexAuthBaseUrl,
  VERTEX_DEFAULT_LOCATION,
  VERTEX_LOCATION_OPTIONS,
  type VertexAuthMode,
  type VertexServiceAccountSummary
} from '@/utils/vertexAi'
import { formatVertexLocationLabel } from '@/utils/displayLabels'

interface Props {
  mode?: 'create' | 'edit'
  legacyMode?: boolean
}

withDefaults(defineProps<Props>(), {
  mode: 'create',
  legacyMode: false
})

const authMode = defineModel<VertexAuthMode>('authMode', { required: true })
const projectId = defineModel<string>('projectId', { required: true })
const location = defineModel<string>('location', { required: true })
const serviceAccountJson = defineModel<string>('serviceAccountJson', { required: true })
const apiKey = defineModel<string>('apiKey', { required: true })
const legacyAccessToken = defineModel<string>('legacyAccessToken', { required: true })
const legacyExpiresAtInput = defineModel<string>('legacyExpiresAtInput', { required: true })
const baseUrl = defineModel<string>('baseUrl', { required: true })

const { t, locale } = useI18n()
const fileInputRef = ref<HTMLInputElement | null>(null)
const summary = ref<VertexServiceAccountSummary | null>(null)
const uploadError = ref('')
const lastAutoBaseUrl = ref('')
const locationOptions = computed(() =>
  VERTEX_LOCATION_OPTIONS.map((option) => ({
    ...option,
    label: formatVertexLocationLabel(option.value, locale.value)
  })) as Array<Record<string, unknown>>
)

const resolvedBaseUrl = computed(() => resolveVertexAuthBaseUrl(authMode.value, location.value))

const modeCardClass = (mode: VertexAuthMode) => [
  'rounded-xl border p-3 text-left transition',
  authMode.value === mode
    ? 'border-sky-400 bg-white/80 text-sky-950 shadow-sm dark:border-sky-400/60 dark:bg-sky-950/30 dark:text-sky-100'
    : 'border-sky-100 bg-white/50 text-sky-800 hover:border-sky-300 dark:border-sky-900/50 dark:bg-sky-950/20 dark:text-sky-200'
]

const syncBaseUrl = (force = false) => {
  const nextBaseUrl = resolvedBaseUrl.value
  const currentBaseUrl = baseUrl.value.trim()
  if (force || !currentBaseUrl || currentBaseUrl === lastAutoBaseUrl.value) {
    baseUrl.value = nextBaseUrl
    lastAutoBaseUrl.value = nextBaseUrl
  }
}

const refreshSummary = (raw: string) => {
  const trimmed = raw.trim()
  if (!trimmed) {
    summary.value = null
    uploadError.value = ''
    return
  }
  try {
    const nextSummary = extractVertexServiceAccountSummary(trimmed)
    summary.value = nextSummary
    uploadError.value = ''
    if (!projectId.value.trim() && nextSummary.project_id) {
      projectId.value = nextSummary.project_id
    }
  } catch {
    summary.value = null
    uploadError.value = t('admin.accounts.gemini.vertex.serviceAccountJsonInvalid')
  }
}

watch(
  location,
  (value) => {
    if (authMode.value !== 'service_account') {
      return
    }
    const normalized = normalizeVertexLocation(value)
    if (normalized !== value) {
      location.value = normalized
      return
    }
    syncBaseUrl()
  },
  { immediate: true }
)

watch(
  authMode,
  () => {
    if (authMode.value === 'service_account' && !location.value.trim()) {
      location.value = VERTEX_DEFAULT_LOCATION
      return
    }
    syncBaseUrl()
  },
  { immediate: true }
)

watch(
  serviceAccountJson,
  (value) => {
    refreshSummary(value)
  },
  { immediate: true }
)

const updateLocation = (value: string | number | boolean | null) => {
  location.value = String(value || VERTEX_DEFAULT_LOCATION)
}

const handleFileChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const raw = await file.text()
    const nextSummary = extractVertexServiceAccountSummary(raw)
    serviceAccountJson.value = raw
    summary.value = nextSummary
    uploadError.value = ''
    if (!projectId.value.trim() && nextSummary.project_id) {
      projectId.value = nextSummary.project_id
    }
    syncBaseUrl(true)
  } catch {
    uploadError.value = t('admin.accounts.gemini.vertex.serviceAccountJsonInvalid')
  } finally {
    input.value = ''
  }
}
</script>
