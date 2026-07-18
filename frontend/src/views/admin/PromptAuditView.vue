<template>
  <AppLayout>
    <div class="space-y-6">
      <header>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ t('admin.promptAudit.title') }}
        </h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.promptAudit.description') }}
        </p>
      </header>

      <div v-if="config.loading.value" class="card p-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>
      <template v-else>
        <PromptAuditRuntimePanel :runtime="config.runtime.value" />
        <PromptAuditConfigPanel
          v-model:draft="config.draft.value"
          :saving="config.saving.value"
          :probing="config.probing.value"
          :probe-results="config.probeResults.value"
          @save="saveConfig"
          @refresh="reloadAll"
          @probe="probeEndpoint"
          @add-endpoint="config.addEndpoint"
          @remove-endpoint="config.removeEndpoint"
          @select-all-scanners="config.selectAllScanners"
        />
        <PromptAuditEventsPanel
          :events="events.events.value"
          v-model:filters="events.filters"
          :total="events.total.value"
          :loading="events.loading.value"
          :delete-preview="events.deletePreview.value"
          @refresh="events.load"
          @reset="resetEvents"
          @open="openEvent"
          @page="changePage"
          @page-size="changePageSize"
          @preview-delete="previewDelete"
          @delete-filter="deleteByFilter"
        />
        <PromptAuditEventDetail
          :event="events.selected.value"
          :deleting="events.deleting.value"
          @delete="deleteEvent"
        />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PromptAuditEndpointDraft } from '@/features/prompt-audit/types'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import PromptAuditConfigPanel from '@/features/prompt-audit/PromptAuditConfigPanel.vue'
import PromptAuditEventDetail from '@/features/prompt-audit/PromptAuditEventDetail.vue'
import PromptAuditEventsPanel from '@/features/prompt-audit/PromptAuditEventsPanel.vue'
import PromptAuditRuntimePanel from '@/features/prompt-audit/PromptAuditRuntimePanel.vue'
import { usePromptAuditConfig } from '@/features/prompt-audit/usePromptAuditConfig'
import { usePromptAuditEvents } from '@/features/prompt-audit/usePromptAuditEvents'

const { t } = useI18n()
const appStore = useAppStore()
const config = usePromptAuditConfig()
const events = usePromptAuditEvents()

function errorMessage(error: unknown, fallbackKey: string) {
  const message = error && typeof error === 'object' && 'message' in error ? String((error as { message?: unknown }).message || '') : ''
  return message || t(fallbackKey)
}

function requestStepUpTotp(promptKey: string, requiredKey: string) {
  const code = window.prompt(t(promptKey))?.trim() || ''
  if (!code) {
    appStore.showError(t(requiredKey))
    return ''
  }
  return code
}

async function reloadAll() {
  try {
    await Promise.all([config.load(), events.load()])
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.promptAudit.messages.loadFailed'))
  }
}

async function saveConfig() {
  const code = requestStepUpTotp('admin.promptAudit.messages.stepUpConfigPrompt', 'admin.promptAudit.messages.stepUpRequired')
  if (!code) return
  try {
    await config.save(code)
    appStore.showSuccess(t('admin.promptAudit.messages.saved'))
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.promptAudit.messages.saveFailed'))
  }
}

async function probeEndpoint(endpoint: PromptAuditEndpointDraft) {
  try {
    const result = await config.probe(endpoint)
    appStore.showSuccess(result.ok ? t('admin.promptAudit.messages.probeOk') : result.message)
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.promptAudit.messages.probeFailed'))
  }
}

async function openEvent(id: number) {
  try {
    await events.open(id)
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.promptAudit.messages.detailFailed'))
  }
}

async function deleteEvent(id: number) {
  const code = requestStepUpTotp('admin.promptAudit.messages.stepUpDeletePrompt', 'admin.promptAudit.messages.stepUpRequired')
  if (!code) return
  try {
    await events.remove(id, code)
    appStore.showSuccess(t('admin.promptAudit.messages.deleted'))
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.promptAudit.messages.deleteFailed'))
  }
}

async function previewDelete() {
  try {
    await events.previewDeleteCurrentFilter()
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.promptAudit.messages.previewFailed'))
  }
}

async function deleteByFilter() {
  const code = requestStepUpTotp('admin.promptAudit.messages.stepUpDeletePrompt', 'admin.promptAudit.messages.stepUpRequired')
  if (!code) return
  try {
    await events.deleteCurrentFilter(code)
    appStore.showSuccess(t('admin.promptAudit.messages.deleted'))
  } catch (error) {
    appStore.showError(errorMessage(error, 'admin.promptAudit.messages.deleteFailed'))
  }
}

function changePage(page: number) {
  events.filters.page = page
  void events.load()
}

function changePageSize(pageSize: number) {
  events.filters.page_size = pageSize
  events.filters.page = 1
  void events.load()
}

function resetEvents() {
  events.resetFilters()
  void events.load()
}

onMounted(() => {
  void reloadAll()
})
</script>
