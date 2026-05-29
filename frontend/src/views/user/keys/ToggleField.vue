<template>
  <div class="flex items-start justify-between gap-4">
    <div class="min-w-0">
      <label class="input-label mb-0">{{ label }}</label>
      <p v-if="hint" class="input-hint mt-1">{{ hint }}</p>
    </div>
    <button
      type="button"
      @click="$emit('update:modelValue', !modelValue)"
      :class="[
        'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
        alignStart ? 'mt-1' : '',
        modelValue ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600',
      ]"
    >
      <span
        :class="[
          'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
          modelValue ? 'translate-x-4' : 'translate-x-0',
        ]"
      />
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string;
  hint?: string;
  modelValue: boolean;
  alignStart?: boolean;
}>();

defineEmits<{
  "update:modelValue": [value: boolean];
}>();
</script>
