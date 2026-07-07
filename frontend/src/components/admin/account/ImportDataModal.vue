<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.dataImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.dataImportHint') }}
      </div>
      <div
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
      >
        {{ t('admin.accounts.dataImportWarning') }}
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <AccountTierSelector
          v-model:tier="openAIImportTier"
          platform="openai"
          :show-apply-capacity="false"
        />
        <AccountTierSelector
          v-model:tier="claudeImportTier"
          platform="anthropic"
          :show-apply-capacity="false"
        />
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.dataImportDefaultsHint') }}
      </p>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex items-center justify-between gap-3 rounded-lg border border-dashed px-4 py-3 transition"
          :class="dragActive
            ? 'border-primary-400 bg-primary-50 dark:border-primary-500 dark:bg-primary-500/10'
            : 'border-gray-300 bg-gray-50 dark:border-dark-600 dark:bg-dark-800'"
          @dragenter.prevent="handleDragEnter"
          @dragover.prevent
          @dragleave.prevent="handleDragLeave"
          @drop.prevent="handleDrop"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ fileLabel || t('admin.accounts.dataImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.accounts.dataImportFileHint') }}
            </div>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="openFilePicker">
            {{ t('common.chooseFile') }}
          </button>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept="application/json,.json"
          multiple
          @change="handleFileChange"
        />
      </div>

      <div
        v-if="result"
        class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>
        <div v-if="job" class="mt-3">
          <div class="mb-1 flex items-center justify-between text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('admin.accounts.dataImportJobStatus', { status: job.status }) }}</span>
            <span>{{ progressText }}</span>
          </div>
          <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
            <div class="h-full bg-primary-600 transition-all" :style="{ width: `${progressPercent}%` }" />
          </div>
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div
            class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800"
          >
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} — {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          v-if="canCancelJob"
          class="btn btn-secondary"
          type="button"
          :disabled="cancelling"
          @click="handleCancelJob"
        >
          {{ cancelling ? t('admin.accounts.dataImportCancelling') : t('admin.accounts.dataImportCancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="import-data-form"
          :disabled="importing"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AccountTierSelector from '@/components/account/AccountTierSelector.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AccountTier, AdminAccountImportJob, AdminDataPayload, ClaudeAccountTier, OpenAIAccountTier } from '@/types'
import {
  DEFAULT_CLAUDE_ACCOUNT_TIER,
  DEFAULT_OPENAI_ACCOUNT_TIER,
  isClaudeAccountTier,
  isOpenAIAccountTier
} from '@/utils/accountTier'
import { useAccountImportJobPolling } from './useAccountImportJobPolling'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'imported', job: AdminAccountImportJob): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const cancelling = ref(false)
const files = ref<File[]>([])
const dragDepth = ref(0)
const openAIImportTier = ref<AccountTier | ''>(DEFAULT_OPENAI_ACCOUNT_TIER)
const claudeImportTier = ref<AccountTier | ''>(DEFAULT_CLAUDE_ACCOUNT_TIER)

const fileInput = ref<HTMLInputElement | null>(null)
const dragActive = computed(() => dragDepth.value > 0)
const fileLabel = computed(() => {
  if (files.value.length === 0) return ''
  if (files.value.length === 1) return files.value[0].name
  return t('admin.accounts.dataImportSelectedFiles', { count: files.value.length })
})
const {
  job,
  result,
  canCancelJob: canCancelActiveJob,
  progressPercent,
  resetImportJob,
  updateFromJob,
  pollImportJob,
  clearPollTimer
} = useAccountImportJobPolling()

const errorItems = computed(() => result.value?.errors || [])
const canCancelJob = computed(() =>
  Boolean(importing.value && canCancelActiveJob.value)
)
const progressText = computed(() => {
  const progress = job.value?.progress
  if (!progress) return ''
  return t('admin.accounts.dataImportProgress', {
    processed: progress.processed,
    total: progress.total
  })
})

watch(
  () => props.show,
  (open) => {
    if (open) {
      files.value = []
      dragDepth.value = 0
      openAIImportTier.value = DEFAULT_OPENAI_ACCOUNT_TIER
      claudeImportTier.value = DEFAULT_CLAUDE_ACCOUNT_TIER
      resetImportJob()
      cancelling.value = false
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    } else {
      clearPollTimer()
    }
  }
)

const openFilePicker = () => {
  fileInput.value?.click()
}

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  files.value = normalizeImportFiles(target.files)
}

const handleDragEnter = () => {
  if (importing.value) return
  dragDepth.value += 1
}

const handleDragLeave = () => {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
}

const handleDrop = (event: DragEvent) => {
  dragDepth.value = 0
  if (importing.value) return
  files.value = normalizeImportFiles(event.dataTransfer?.files || null)
}

const handleClose = () => {
  if (importing.value) return
  emit('close')
}

const readFileAsText = async (sourceFile: File): Promise<string> => {
  if (typeof sourceFile.text === 'function') {
    return sourceFile.text()
  }

  if (typeof sourceFile.arrayBuffer === 'function') {
    const buffer = await sourceFile.arrayBuffer()
    return new TextDecoder().decode(buffer)
  }

  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(sourceFile)
  })
}

const normalizeImportFiles = (list: FileList | null): File[] => {
  if (!list) return []
  return Array.from(list).filter((item) =>
    item.name.toLowerCase().endsWith('.json') || item.type === 'application/json'
  )
}

const isValidDataPayload = (value: unknown): value is AdminDataPayload => {
  return Boolean(
    value &&
    typeof value === 'object' &&
    Array.isArray((value as AdminDataPayload).accounts) &&
    Array.isArray((value as AdminDataPayload).proxies)
  )
}

const mergeDataPayloads = (payloads: AdminDataPayload[]): AdminDataPayload => {
  if (payloads.length === 1) return payloads[0]

  const firstPayload = payloads[0]
  return {
    type: payloads.find((item) => item.type)?.type || firstPayload?.type || 'sub2api_admin_export',
    version: payloads.find((item) => item.version)?.version || firstPayload?.version || 1,
    exported_at: payloads.length === 1 ? firstPayload.exported_at : new Date().toISOString(),
    proxies: payloads.flatMap((item) => item.proxies || []),
    accounts: payloads.flatMap((item) => item.accounts || [])
  }
}

const handleImport = async () => {
  if (files.value.length === 0) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }

  importing.value = true
  try {
    const dataPayloads: AdminDataPayload[] = []
    for (const sourceFile of files.value) {
      let parsed: unknown
      try {
        parsed = JSON.parse(await readFileAsText(sourceFile))
      } catch {
        appStore.showError(t('admin.accounts.dataImportParseFailedFile', { name: sourceFile.name }))
        return
      }
      if (!isValidDataPayload(parsed)) {
        appStore.showError(t('admin.accounts.dataImportParseFailedFile', { name: sourceFile.name }))
        return
      }
      dataPayloads.push(parsed)
    }

    const createdJob = await adminAPI.accounts.createImportJob({
      data: mergeDataPayloads(dataPayloads),
      skip_default_group_bind: true,
      account_defaults: {
        openai_tier: isOpenAIAccountTier(openAIImportTier.value)
          ? openAIImportTier.value as OpenAIAccountTier
          : undefined,
        claude_tier: isClaudeAccountTier(claudeImportTier.value)
          ? claudeImportTier.value as ClaudeAccountTier
          : undefined
      }
    })
    const finalJob = await pollImportJob(createdJob.job_id)
    const res = finalJob.result

    updateFromJob(finalJob)

    const msgParams: Record<string, unknown> = {
      account_created: res.account_created,
      account_failed: res.account_failed,
      proxy_created: res.proxy_created,
      proxy_reused: res.proxy_reused,
      proxy_failed: res.proxy_failed,
    }
    if (finalJob.status === 'cancelled') {
      appStore.showError(t('admin.accounts.dataImportCancelled'))
    } else if (finalJob.status === 'failed') {
      appStore.showError(finalJob.error || t('admin.accounts.dataImportFailed'))
    } else if (res.account_failed > 0 || res.proxy_failed > 0) {
      appStore.showError(t('admin.accounts.dataImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.dataImportSuccess', msgParams))
    }
    if ((finalJob.created_accounts_summary?.length ?? 0) > 0 && finalJob.status !== 'cancelled') {
      emit('imported', finalJob)
    }
  } catch (error: any) {
    if (error instanceof SyntaxError) {
      appStore.showError(t('admin.accounts.dataImportParseFailed'))
    } else {
      appStore.showError(error?.message || t('admin.accounts.dataImportFailed'))
    }
  } finally {
    importing.value = false
  }
}

const handleCancelJob = async () => {
  if (!job.value || cancelling.value) return
  cancelling.value = true
  try {
    const nextJob = await adminAPI.accounts.cancelImportJob(job.value.job_id)
    updateFromJob(nextJob)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.dataImportCancelFailed'))
  } finally {
    cancelling.value = false
  }
}
</script>
