<template>
  <div v-if="form.platform === 'gemini'" class="border-t pt-4">
    <div class="mb-4 flex items-center justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.groups.imageBatch.title') }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.imageBatch.hint') }}
        </p>
      </div>
      <Toggle v-model="form.image_batch_enabled" />
    </div>

    <div v-if="form.image_batch_enabled" class="space-y-4 border-l-2 border-primary-200 pl-4 dark:border-primary-800">
      <div>
        <label class="input-label">{{ t('admin.groups.imageBatch.allowedProviders') }}</label>
        <input
          :value="joinList(form.image_batch_allowed_providers)"
          type="text"
          class="input"
          placeholder="gemini"
          @input="updateList('image_batch_allowed_providers', $event)"
        />
        <p class="input-hint">{{ t('admin.groups.imageBatch.allowedProvidersHint') }}</p>
      </div>

      <div>
        <label class="input-label">{{ t('admin.groups.imageBatch.allowedModels') }}</label>
        <textarea
          :value="joinLines(form.image_batch_allowed_models)"
          rows="3"
          class="input font-mono text-xs"
          :placeholder="t('admin.groups.imageBatch.allowedModelsPlaceholder')"
          @input="updateLines('image_batch_allowed_models', $event)"
        ></textarea>
        <p class="input-hint">{{ t('admin.groups.imageBatch.allowedModelsHint') }}</p>
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <div>
          <label class="input-label">{{ t('admin.groups.imageBatch.maxItems') }}</label>
          <input
            v-model.number="form.image_batch_max_items"
            type="number"
            min="1"
            max="1000"
            class="input"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.imageBatch.maxDownloadMB') }}</label>
          <input
            :value="downloadMB"
            type="number"
            min="1"
            class="input"
            @input="updateDownloadMB"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.groups.imageBatch.downloadConcurrency') }}</label>
          <input
            v-model.number="form.image_batch_download_concurrency"
            type="number"
            min="1"
            max="16"
            class="input"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Toggle from '@/components/common/Toggle.vue'

const props = defineProps<{
  form: Record<string, any>
  t: (key: string, params?: Record<string, unknown>) => string
}>()

const bytesPerMB = 1024 * 1024
const form = props.form
const t = props.t
const downloadMB = computed(() =>
  Math.max(1, Math.round(Number(form.image_batch_max_download_bytes || bytesPerMB) / bytesPerMB)),
)

const splitText = (value: string, separator: RegExp) =>
  value
    .split(separator)
    .map((item) => item.trim())
    .filter(Boolean)

const joinList = (values?: string[]) => (Array.isArray(values) ? values.join(', ') : '')
const joinLines = (values?: string[]) => (Array.isArray(values) ? values.join('\n') : '')

const updateList = (key: string, event: Event) => {
  form[key] = splitText((event.target as HTMLInputElement).value, /[,，\n]/)
}

const updateLines = (key: string, event: Event) => {
  form[key] = splitText((event.target as HTMLTextAreaElement).value, /\n/)
}

const updateDownloadMB = (event: Event) => {
  const value = Number((event.target as HTMLInputElement).value)
  form.image_batch_max_download_bytes = Number.isFinite(value) && value > 0
    ? Math.round(value * bytesPerMB)
    : bytesPerMB
}
</script>
