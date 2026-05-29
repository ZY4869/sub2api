<template>
  <nav class="flex gap-8 border-b border-slate-200/80 bg-white px-6 dark:border-dark-700 dark:bg-dark-900 md:px-10">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      type="button"
      class="relative flex items-center gap-2 pb-4 text-[15px] font-bold transition-all"
      :class="modelValue === tab.id ? 'text-indigo-600 dark:text-indigo-300' : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100'"
      :data-testid="`public-model-detail-tab-${tab.id}`"
      @click="emit('update:modelValue', tab.id)"
    >
      <Icon :name="tab.icon" size="sm" :stroke-width="2.5" />
      {{ tab.label }}
      <span
        v-if="modelValue === tab.id"
        class="absolute bottom-0 left-0 h-[3px] w-full rounded-t-full bg-indigo-600 shadow-[0_-2px_10px_rgba(79,70,229,0.3)] dark:bg-indigo-300"
      ></span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

export interface PublicModelDetailTab {
  id: string
  label: string
  icon: 'infoCircle' | 'chart' | 'terminal'
}

defineProps<{
  modelValue: string
  tabs: PublicModelDetailTab[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>
