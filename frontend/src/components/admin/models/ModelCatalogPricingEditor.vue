<template>
  <div class="grid gap-4 xl:grid-cols-2">
    <ModelCatalogPricingEditorSection
      :detail="detail"
      layer="official"
      :saving="saving"
      @save="emit('save-official', $event)"
      @reset="emit('reset-official', $event)"
    />
    <ModelCatalogPricingEditorSection
      :detail="detail"
      layer="sale"
      :saving="saving"
      @save="emit('save-sale', $event)"
      @reset="emit('reset-sale', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { ModelCatalogDetail, UpdatePricingOverridePayload } from '@/api/admin/models'
import ModelCatalogPricingEditorSection from './ModelCatalogPricingEditorSection.vue'

defineProps<{
  detail: ModelCatalogDetail
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'save-official', payload: UpdatePricingOverridePayload): void
  (e: 'reset-official', model: string): void
  (e: 'save-sale', payload: UpdatePricingOverridePayload): void
  (e: 'reset-sale', model: string): void
}>()
</script>
