<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ t('imageBatches.title') }}
        </h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('imageBatches.description') }}
        </p>
      </div>

      <div class="card p-5">
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_auto]">
          <div>
            <label class="input-label">{{ t('imageBatches.apiKey') }}</label>
            <input
              v-model.trim="apiKeySecret"
              type="password"
              class="input"
              autocomplete="off"
              :placeholder="t('imageBatches.apiKeyPlaceholder')"
            />
          </div>
          <div class="flex items-end">
            <button type="button" class="btn btn-primary w-full lg:w-auto" :disabled="loadingAny" @click="refreshAll">
              {{ t('imageBatches.load') }}
            </button>
          </div>
        </div>
        <p v-if="errorMessage" class="mt-3 text-sm text-red-600 dark:text-red-400">
          {{ errorMessage }}
        </p>
      </div>

      <ImageBatchSubmitPanel
        :form="submitForm"
        :models="models"
        :submitting="submitting"
        :t="t"
        @submit="handleSubmit"
      />

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(360px,0.65fr)]">
        <ImageBatchJobsPanel
          :jobs="jobs"
          :loading="jobsLoading"
          :selected-job-id="selectedJob?.id"
          :t="t"
          @refresh="loadJobs"
          @select="selectJob"
          @download="handleDownloadJob"
          @cancel="handleCancelJob"
          @delete="handleDeleteJob"
          @delete-outputs="handleDeleteOutputs"
        />
        <ImageBatchItemsPanel
          :job-id="selectedJob?.id"
          :items="items"
          :loading="itemsLoading"
          :t="t"
          @refresh="loadSelectedItems"
          @download-item="handleDownloadItem"
        />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { saveAs } from 'file-saver'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores/app'
import {
  cancelImageBatch,
  deleteImageBatch,
  deleteImageBatchOutputs,
  downloadImageBatch,
  downloadImageBatchItem,
  listImageBatchItems,
  listImageBatchModels,
  listImageBatches,
  submitImageBatch,
  type ImageBatchItem,
  type ImageBatchJob,
  type ImageBatchModel,
} from '@/api/imageBatches'
import ImageBatchItemsPanel from './image-batches/ImageBatchItemsPanel.vue'
import ImageBatchJobsPanel from './image-batches/ImageBatchJobsPanel.vue'
import ImageBatchSubmitPanel from './image-batches/ImageBatchSubmitPanel.vue'

interface DraftItem {
  local_id: string
  custom_id: string
  prompt: string
  n: number
  size: string
}

const { t } = useI18n()
const appStore = useAppStore()
const apiKeySecret = ref('')
const models = ref<ImageBatchModel[]>([])
const jobs = ref<ImageBatchJob[]>([])
const items = ref<ImageBatchItem[]>([])
const selectedJob = ref<ImageBatchJob | null>(null)
const modelsLoading = ref(false)
const jobsLoading = ref(false)
const itemsLoading = ref(false)
const submitting = ref(false)
const errorMessage = ref('')

const makeDraftItem = (): DraftItem => {
  const id = crypto.randomUUID?.() || `${Date.now()}-${Math.random()}`
  return { local_id: id, custom_id: `item-${id.slice(0, 8)}`, prompt: '', n: 1, size: '' }
}

const submitForm = reactive({
  provider: 'gemini',
  model: '',
  size: '1024x1024',
  items: [makeDraftItem()],
})

const apiReady = computed(() => apiKeySecret.value.length > 0)
const loadingAny = computed(() => modelsLoading.value || jobsLoading.value || itemsLoading.value)

const friendlyError = (error: unknown) => {
  const data = (error as any)?.response?.data
  return data?.error?.message || (error as any)?.message || t('imageBatches.genericError')
}

const withGatewayError = async (action: () => Promise<void>) => {
  errorMessage.value = ''
  if (!apiReady.value) {
    errorMessage.value = t('imageBatches.apiKeyRequired')
    return
  }
  try {
    await action()
  } catch (error) {
    errorMessage.value = friendlyError(error)
  }
}

const loadModels = async () => {
  modelsLoading.value = true
  try {
    models.value = await listImageBatchModels(apiKeySecret.value)
    if (!submitForm.model && models.value.length > 0) {
      submitForm.model = models.value[0].id
    }
  } finally {
    modelsLoading.value = false
  }
}

const loadJobs = async () => {
  await withGatewayError(async () => {
    jobsLoading.value = true
    try {
      jobs.value = await listImageBatches(apiKeySecret.value)
    } finally {
      jobsLoading.value = false
    }
  })
}

const refreshAll = async () => {
  await withGatewayError(async () => {
    await Promise.all([loadModels(), loadJobs()])
  })
}

const selectJob = async (job: ImageBatchJob) => {
  selectedJob.value = job
  await loadSelectedItems()
}

const loadSelectedItems = async () => {
  if (!selectedJob.value) return
  await withGatewayError(async () => {
    itemsLoading.value = true
    try {
      items.value = await listImageBatchItems(apiKeySecret.value, selectedJob.value!.id)
    } finally {
      itemsLoading.value = false
    }
  })
}

const handleSubmit = async () => {
  const requestItems = submitForm.items
    .map((item) => ({
      custom_id: item.custom_id,
      prompt: item.prompt.trim(),
      n: Math.max(1, Number(item.n) || 1),
      size: item.size.trim() || submitForm.size.trim(),
    }))
    .filter((item) => item.prompt !== '')
  if (requestItems.length === 0 || !submitForm.model.trim()) {
    errorMessage.value = t('imageBatches.submitRequired')
    return
  }
  await withGatewayError(async () => {
    submitting.value = true
    try {
      const job = await submitImageBatch(
        apiKeySecret.value,
        {
          provider: submitForm.provider,
          model: submitForm.model,
          size: submitForm.size.trim(),
          items: requestItems,
        },
        crypto.randomUUID?.() || `${Date.now()}-${Math.random()}`,
      )
      appStore.showSuccess(t('imageBatches.submitted'))
      selectedJob.value = job
      submitForm.items = [makeDraftItem()]
      await Promise.all([loadJobs(), loadSelectedItems()])
    } finally {
      submitting.value = false
    }
  })
}

const handleCancelJob = async (job: ImageBatchJob) => {
  await withGatewayError(async () => {
    selectedJob.value = await cancelImageBatch(apiKeySecret.value, job.id)
    await loadJobs()
  })
}

const handleDeleteJob = async (job: ImageBatchJob) => {
  await withGatewayError(async () => {
    await deleteImageBatch(apiKeySecret.value, job.id)
    if (selectedJob.value?.id === job.id) {
      selectedJob.value = null
      items.value = []
    }
    await loadJobs()
  })
}

const handleDeleteOutputs = async (job: ImageBatchJob) => {
  await withGatewayError(async () => {
    await deleteImageBatchOutputs(apiKeySecret.value, job.id)
    await loadJobs()
    if (selectedJob.value?.id === job.id) {
      await loadSelectedItems()
    }
  })
}

const handleDownloadJob = async (job: ImageBatchJob) => {
  await withGatewayError(async () => {
    const blob = await downloadImageBatch(apiKeySecret.value, job.id)
    saveAs(blob, `${job.id}.zip`)
  })
}

const handleDownloadItem = async (item: ImageBatchItem) => {
  if (!selectedJob.value) return
  await withGatewayError(async () => {
    const blob = await downloadImageBatchItem(apiKeySecret.value, selectedJob.value!.id, item.custom_id)
    saveAs(blob, `${selectedJob.value!.id}-${item.custom_id}.bin`)
  })
}
</script>
