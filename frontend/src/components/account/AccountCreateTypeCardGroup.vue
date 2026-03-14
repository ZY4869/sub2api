<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

type Accent = 'orange' | 'purple' | 'green' | 'rose'
type IconName = 'sparkles' | 'key' | 'link' | 'cloud'

interface TypeOption {
  key: string
  title: string
  description: string
  icon: IconName
  accent: Accent
  active: boolean
}

defineProps<{
  label: string
  options: TypeOption[]
  tour?: string
}>()

const emit = defineEmits<{
  select: [key: string]
}>()

const inactiveCardClass =
  'border-gray-200 hover:border-gray-300 dark:border-dark-600 dark:hover:border-dark-500'
const inactiveIconClass = 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'

function cardClass(active: boolean, accent: Accent): string {
  if (!active) return inactiveCardClass
  if (accent === 'orange') return 'border-orange-500 bg-orange-50 dark:bg-orange-900/20'
  if (accent === 'green') return 'border-green-500 bg-green-50 dark:bg-green-900/20'
  if (accent === 'rose') return 'border-rose-500 bg-rose-50 dark:bg-rose-900/20'
  return 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
}

function iconClass(active: boolean, accent: Accent): string {
  if (!active) return inactiveIconClass
  if (accent === 'orange') return 'bg-orange-500 text-white'
  if (accent === 'green') return 'bg-green-500 text-white'
  if (accent === 'rose') return 'bg-rose-500 text-white'
  return 'bg-purple-500 text-white'
}
</script>

<template>
  <div>
    <label class="input-label">{{ label }}</label>
    <div class="mt-2 grid grid-cols-2 gap-3" :data-tour="tour">
      <button
        v-for="option in options"
        :key="option.key"
        type="button"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          cardClass(option.active, option.accent)
        ]"
        @click="emit('select', option.key)"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            iconClass(option.active, option.accent)
          ]"
        >
          <Icon :name="option.icon" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">{{ option.title }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ option.description }}</span>
        </div>
      </button>
    </div>
  </div>
</template>
