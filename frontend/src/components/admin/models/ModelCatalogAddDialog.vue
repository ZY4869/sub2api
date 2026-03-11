<template>
  <BaseDialog :show="show" :title="t('admin.models.catalog.createTitle')" width="wide" close-on-click-outside @close="handleClose">
    <form class="space-y-4" @submit.prevent="handleSubmit">
      <div>
        <label class="input-label" for="catalog-model">{{ t('admin.models.catalog.modelId') }}</label>
        <input
          id="catalog-model"
          v-model.trim="model"
          type="text"
          class="input"
          :placeholder="t('admin.models.catalog.modelPlaceholder')"
        />
        <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.models.catalog.modelHint') }}
        </p>
      </div>

      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">{{ t('common.cancel') }}</button>
        <button type="submit" class="btn btn-primary" :disabled="!model">{{ t('common.add') }}</button>
      </div>
    </form>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'confirm', model: string): void
}>()

const { t } = useI18n()
const model = ref('')

watch(
  () => props.show,
  (value) => {
    if (value) {
      model.value = ''
    }
  }
)

function handleClose() {
  emit('close')
}

function handleSubmit() {
  if (!model.value) {
    return
  }
  emit('confirm', model.value)
}
</script>
