<template>
  <BaseDialog
    :show="show"
    :title="t('admin.models.available.manualAddDialog.title')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('admin.models.available.manualAddDialog.description') }}
      </p>

      <label class="block space-y-1.5">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          * {{ t('admin.models.available.manualAddDialog.modelIdLabel') }}
        </span>
        <input
          v-model.trim="modelId"
          type="text"
          class="input"
          :class="validationError ? 'border-red-400 focus:border-red-500 focus:ring-red-500/20' : ''"
          :placeholder="t('admin.models.available.manualAddDialog.modelIdPlaceholder')"
          :aria-invalid="validationError ? 'true' : 'false'"
          @keyup.enter="submit"
        />
        <p v-if="validationError" class="text-xs text-red-500">
          {{ validationError }}
        </p>
      </label>

      <label class="block space-y-1.5">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.models.available.manualAddDialog.displayNameLabel') }}
        </span>
        <input
          v-model.trim="displayName"
          type="text"
          class="input"
          :placeholder="t('admin.models.available.manualAddDialog.displayNamePlaceholder')"
          @keyup.enter="submit"
        />
      </label>

      <label class="block space-y-1.5">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.models.available.manualAddDialog.providerLabel') }}
        </span>
        <select v-model="provider" class="input">
          <option value="">{{ t('admin.models.available.manualAddDialog.providerAutoOption') }}</option>
          <option
            v-for="option in providerOptions"
            :key="option.value"
            :value="option.value"
          >
            {{ option.label }}
          </option>
        </select>
      </label>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="submitting" @click="submit">
          {{ t('admin.models.available.manualAddDialog.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ManualAddModelRegistryEntryPayload } from '@/api/admin/modelRegistry'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { formatProviderLabel, listKnownProviders } from '@/utils/providerLabels'

const props = withDefaults(defineProps<{
  show: boolean
  submitting?: boolean
}>(), {
  submitting: false
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', payload: ManualAddModelRegistryEntryPayload): void
}>()

const { t } = useI18n()
const modelId = ref('')
const displayName = ref('')
const provider = ref('')
const validationError = ref('')
const providerOptions = computed(() =>
  listKnownProviders().map((value) => ({
    value,
    label: formatProviderLabel(value)
  }))
)

function resetForm() {
  modelId.value = ''
  displayName.value = ''
  provider.value = ''
  validationError.value = ''
}

function handleClose() {
  resetForm()
  emit('close')
}

function submit() {
  if (!modelId.value.trim()) {
    validationError.value = t('admin.models.available.manualAddDialog.modelIdRequired')
    return
  }
  validationError.value = ''
  emit('submit', {
    id: modelId.value.trim(),
    display_name: displayName.value.trim() || undefined,
    provider: provider.value || undefined
  })
}

watch(
  () => props.show,
  (show) => {
    if (!show) {
      resetForm()
    }
  }
)
</script>
