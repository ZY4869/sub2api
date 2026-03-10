<template>
  <div class="flex flex-wrap items-center gap-3">
    <div class="min-w-[220px] flex-1">
      <SearchInput
        :model-value="search"
        :placeholder="t('admin.models.searchPlaceholder')"
        @update:model-value="emit('update:search', $event)"
        @search="emit('search')"
      />
    </div>
    <div class="w-full sm:w-40">
      <Select :model-value="provider" :options="providerOptions" @update:model-value="emit('update:provider', String($event ?? ''))" />
    </div>
    <div class="w-full sm:w-40">
      <Select :model-value="mode" :options="modeOptions" @update:model-value="emit('update:mode', String($event ?? ''))" />
    </div>
    <div class="w-full sm:w-40">
      <Select :model-value="availability" :options="availabilityOptions" @update:model-value="emit('update:availability', String($event ?? ''))" />
    </div>
    <div class="w-full sm:w-44">
      <Select :model-value="pricingSource" :options="pricingSourceOptions" @update:model-value="emit('update:pricingSource', String($event ?? ''))" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SearchInput from '@/components/common/SearchInput.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'

defineProps<{
  search: string
  provider: string
  mode: string
  availability: string
  pricingSource: string
  providerOptions: SelectOption[]
  modeOptions: SelectOption[]
  availabilityOptions: SelectOption[]
  pricingSourceOptions: SelectOption[]
}>()

const emit = defineEmits<{
  (e: 'update:search', value: string): void
  (e: 'update:provider', value: string): void
  (e: 'update:mode', value: string): void
  (e: 'update:availability', value: string): void
  (e: 'update:pricingSource', value: string): void
  (e: 'search'): void
}>()

const { t } = useI18n()
</script>
