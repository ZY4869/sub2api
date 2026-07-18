<template>
  <section class="card p-5">
    <div class="flex flex-col gap-3 border-b border-gray-100 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.promptAudit.config.title') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.promptAudit.config.description') }}
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-secondary" type="button" @click="$emit('refresh')">
          {{ t('common.refresh') }}
        </button>
        <button class="btn btn-primary" type="button" :disabled="saving" @click="$emit('save')">
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </div>

    <div v-if="draft" class="mt-5 space-y-6">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <label class="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <input v-model="draft.enabled" type="checkbox" class="h-4 w-4" />
          <span class="text-sm text-gray-800 dark:text-gray-100">{{ t('admin.promptAudit.config.enabled') }}</span>
        </label>
        <label class="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <input v-model="draft.blocking_enabled" type="checkbox" class="h-4 w-4" :disabled="!draft.enabled" />
          <span class="text-sm text-gray-800 dark:text-gray-100">{{ t('admin.promptAudit.config.blocking') }}</span>
        </label>
        <label class="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <input v-model="draft.store_pass_events" type="checkbox" class="h-4 w-4" />
          <span class="text-sm text-gray-800 dark:text-gray-100">{{ t('admin.promptAudit.config.storePass') }}</span>
        </label>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <label class="space-y-1">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.config.strategy') }}</span>
          <input v-model="draft.strategy" class="input" readonly />
        </label>
        <label class="space-y-1">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.config.workerCount') }}</span>
          <input v-model.number="draft.worker_count" class="input" type="number" min="1" max="32" />
        </label>
        <label class="space-y-1">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.config.queueCapacity') }}</span>
          <input v-model.number="draft.queue_capacity" class="input" type="number" min="1" max="100000" />
        </label>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.promptAudit.config.scanners') }}</h3>
          <button class="btn btn-ghost btn-sm" type="button" @click="$emit('select-all-scanners')">
            {{ t('admin.promptAudit.config.selectAllScanners') }}
          </button>
        </div>
        <div class="grid grid-cols-1 gap-2 md:grid-cols-3">
          <label v-for="scanner in scannerOptions" :key="scanner.id" class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
            <input v-model="draft.scanners" type="checkbox" class="h-4 w-4" :value="scanner.id" />
            <span>{{ scanner.label }}</span>
          </label>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <label class="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <input v-model="draft.all_groups" type="checkbox" class="h-4 w-4" />
          <span class="text-sm text-gray-800 dark:text-gray-100">{{ t('admin.promptAudit.config.allGroups') }}</span>
        </label>
        <label class="space-y-1">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.promptAudit.config.groupIds') }}</span>
          <input v-model="draft.group_ids_text" class="input" :disabled="draft.all_groups" />
        </label>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.promptAudit.config.endpoints') }}</h3>
          <button class="btn btn-secondary btn-sm" type="button" @click="$emit('add-endpoint')">
            {{ t('admin.promptAudit.config.addEndpoint') }}
          </button>
        </div>
        <div v-if="draft.endpoints.length === 0" class="rounded-lg border border-dashed border-gray-300 p-5 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
          {{ t('admin.promptAudit.config.noEndpoints') }}
        </div>
        <PromptAuditEndpointEditor
          v-for="(endpoint, index) in draft.endpoints"
          :key="endpoint.id"
          v-model:endpoint="draft.endpoints[index]"
          :probing="!!probing[endpoint.id]"
          :probe-result="probeResults[endpoint.id]"
          @probe="$emit('probe', endpoint)"
          @remove="$emit('remove-endpoint', index)"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PromptAuditProbeResult } from '@/api/admin/promptAudit'
import { promptAuditScanners } from './constants'
import type { PromptAuditConfigDraft, PromptAuditEndpointDraft } from './types'
import PromptAuditEndpointEditor from './PromptAuditEndpointEditor.vue'

const draft = defineModel<PromptAuditConfigDraft | null>('draft', { required: true })

defineProps<{
  saving: boolean
  probing: Record<string, boolean>
  probeResults: Record<string, PromptAuditProbeResult>
}>()

defineEmits<{
  save: []
  refresh: []
  probe: [endpoint: PromptAuditEndpointDraft]
  'add-endpoint': []
  'remove-endpoint': [index: number]
  'select-all-scanners': []
}>()

const { t } = useI18n()

const scannerOptions = computed(() =>
  promptAuditScanners.map((id) => ({
    id,
    label: t(`admin.promptAudit.scanners.${id}`)
  }))
)
</script>
