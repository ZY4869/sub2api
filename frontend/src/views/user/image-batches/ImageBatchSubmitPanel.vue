<template>
  <div class="card p-5">
    <div class="mb-4 flex items-center justify-between gap-3">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('imageBatches.submitTitle') }}
      </h2>
      <button type="button" class="btn btn-secondary btn-sm" @click="addItem">
        {{ t('imageBatches.addItem') }}
      </button>
    </div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <div>
        <label class="input-label">{{ t('imageBatches.provider') }}</label>
        <div class="flex h-10 items-center gap-2 rounded-lg border border-gray-200 px-3 dark:border-dark-600">
          <ModelPlatformIcon platform="gemini" size="sm" />
          <span class="text-sm text-gray-700 dark:text-gray-300">gemini</span>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('imageBatches.model') }}</label>
        <select v-model="form.model" class="input">
          <option value="">{{ t('imageBatches.selectModel') }}</option>
          <option v-for="model in models" :key="model.id" :value="model.id">
            {{ model.display_name || model.id }}
          </option>
        </select>
        <div v-if="form.model" class="mt-2 inline-flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
          <ModelIcon :model="form.model" provider="gemini" :display-name="form.model" size="14px" />
          <span>{{ form.model }}</span>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('imageBatches.size') }}</label>
        <input v-model.trim="form.size" type="text" class="input" placeholder="1024x1024" />
      </div>
    </div>

    <div class="mt-5 space-y-3">
      <div
        v-for="(item, index) in form.items"
        :key="item.local_id"
        class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
      >
        <div class="mb-3 flex items-center justify-between gap-3">
          <div class="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
            <span>{{ t('imageBatches.itemTitle', { index: index + 1 }) }}</span>
            <span class="text-xs text-gray-400">{{ item.custom_id }}</span>
          </div>
          <button
            type="button"
            class="text-sm text-red-600 hover:text-red-700 disabled:opacity-40"
            :disabled="form.items.length <= 1"
            @click="removeItem(index)"
          >
            {{ t('common.delete') }}
          </button>
        </div>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-[1fr_100px_140px]">
          <div>
            <label class="input-label">{{ t('imageBatches.prompt') }}</label>
            <textarea v-model.trim="item.prompt" rows="3" class="input"></textarea>
          </div>
          <div>
            <label class="input-label">{{ t('imageBatches.count') }}</label>
            <input v-model.number="item.n" type="number" min="1" max="4" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('imageBatches.itemSize') }}</label>
            <input v-model.trim="item.size" type="text" class="input" :placeholder="form.size" />
          </div>
        </div>
      </div>
    </div>

    <div class="mt-5 flex justify-end">
      <button type="button" class="btn btn-primary" :disabled="submitting || !canSubmit" @click="$emit('submit')">
        {{ submitting ? t('imageBatches.submitting') : t('imageBatches.submit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { ImageBatchModel } from '@/api/imageBatches'

interface DraftItem {
  local_id: string
  custom_id: string
  prompt: string
  n: number
  size: string
}

const props = defineProps<{
  form: { provider: string; model: string; size: string; items: DraftItem[] }
  models: ImageBatchModel[]
  submitting: boolean
  t: (key: string, params?: Record<string, unknown>) => string
}>()

defineEmits<{ submit: [] }>()

const form = props.form
const t = props.t
const canSubmit = computed(() =>
  form.model.trim() !== '' && form.items.some((item) => item.prompt.trim() !== ''),
)

const newItem = (): DraftItem => ({
  local_id: crypto.randomUUID?.() || `${Date.now()}-${Math.random()}`,
  custom_id: `item-${Date.now().toString(36)}-${(form.items.length + 1).toString(36)}`,
  prompt: '',
  n: 1,
  size: '',
})

const addItem = () => {
  form.items.push(newItem())
}

const removeItem = (index: number) => {
  if (form.items.length > 1) {
    form.items.splice(index, 1)
  }
}
</script>
