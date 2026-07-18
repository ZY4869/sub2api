<template>
  <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
      <label class="space-y-1">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.id') }}</span>
        <input v-model="model.id" class="input font-mono text-sm" />
      </label>
      <label class="space-y-1">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.name') }}</span>
        <input v-model="model.name" class="input" />
      </label>
      <label class="space-y-1 md:col-span-2">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.baseUrl') }}</span>
        <input v-model="model.base_url" class="input font-mono text-sm" placeholder="https://example.com" />
      </label>
      <label class="space-y-1 md:col-span-2">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.model') }}</span>
        <input v-model="model.model" class="input font-mono text-sm" />
      </label>
      <label class="space-y-1">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.timeout') }}</span>
        <input v-model.number="model.timeout_ms" class="input" type="number" min="100" max="30000" />
      </label>
      <label class="space-y-1">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.inputLimit') }}</span>
        <input v-model.number="model.input_limit" class="input" type="number" min="128" max="100000" />
      </label>
      <label class="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <input v-model="model.enabled" type="checkbox" class="h-4 w-4" />
        <span class="text-sm text-gray-800 dark:text-gray-100">{{ t('admin.promptAudit.endpoint.enabled') }}</span>
      </label>
      <label class="space-y-1">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.tokenMode') }}</span>
        <select v-model="model.token_mode" class="input">
          <option value="keep">{{ t('admin.promptAudit.endpoint.keepToken') }}</option>
          <option value="replace">{{ t('admin.promptAudit.endpoint.replaceToken') }}</option>
          <option value="clear">{{ t('admin.promptAudit.endpoint.clearToken') }}</option>
        </select>
      </label>
      <label v-if="model.token_mode === 'replace'" class="space-y-1 md:col-span-2">
        <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.endpoint.token') }}</span>
        <input v-model="model.token" class="input" type="password" autocomplete="new-password" />
      </label>
    </div>
    <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
      <div class="text-xs text-gray-500 dark:text-gray-400">
        {{ tokenStatus }}
        <span v-if="probeResult"> · {{ probeResult.status }} · {{ probeResult.latency_ms }} ms</span>
      </div>
      <div class="flex gap-2">
        <button class="btn btn-secondary btn-sm" type="button" :disabled="probing" @click="$emit('probe')">
          {{ probing ? t('admin.promptAudit.endpoint.probing') : t('admin.promptAudit.endpoint.probe') }}
        </button>
        <button class="btn btn-danger btn-sm" type="button" @click="$emit('remove')">
          {{ t('common.delete') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PromptAuditProbeResult } from '@/api/admin/promptAudit'
import type { PromptAuditEndpointDraft } from './types'

const model = defineModel<PromptAuditEndpointDraft>('endpoint', { required: true })

defineProps<{
  probing: boolean
  probeResult?: PromptAuditProbeResult
}>()

defineEmits<{ probe: []; remove: [] }>()

const { t } = useI18n()

const tokenStatus = computed(() => {
  if (model.value.token_mode === 'clear') return t('admin.promptAudit.endpoint.tokenWillClear')
  if (model.value.token_mode === 'replace') return t('admin.promptAudit.endpoint.tokenWillReplace')
  return model.value.has_token
    ? t('admin.promptAudit.endpoint.tokenStored')
    : t('admin.promptAudit.endpoint.tokenMissing')
})
</script>
