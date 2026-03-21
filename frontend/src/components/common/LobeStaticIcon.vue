<template>
  <img
    v-if="activeSrc"
    :src="activeSrc"
    :alt="alt"
    :style="iconStyle"
    class="lobe-static-icon shrink-0 object-contain"
    :class="{ 'lobe-static-icon--mono': isMonoIcon }"
    loading="lazy"
    decoding="async"
    @error="handleError"
  />
  <span
    v-else
    :style="badgeStyle"
    class="inline-flex shrink-0 items-center justify-center rounded-md font-semibold shadow-sm"
    :class="badgeClass"
    aria-hidden="true"
  >
    {{ badgeText }}
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  sources: string[]
  badgeText: string
  size?: string
  alt?: string
  variant?: 'model' | 'platform'
}>(), {
  size: '18px',
  alt: '',
  variant: 'model'
})

const sourceIndex = ref(0)

const activeSrc = computed(() => props.sources[sourceIndex.value] || '')
const isMonoIcon = computed(() => Boolean(activeSrc.value) && !activeSrc.value.endsWith('-color.svg'))
const iconStyle = computed(() => ({
  width: props.size,
  height: props.size
}))
const badgeStyle = computed(() => ({
  width: props.size,
  height: props.size,
  fontSize: `calc(${props.size} * 0.42)`
}))
const badgeClass = computed(() => {
  return props.variant === 'platform'
    ? 'bg-sky-600 text-white dark:bg-sky-500'
    : 'bg-violet-600 text-white dark:bg-violet-500'
})

watch(
  () => props.sources.join('|'),
  () => {
    sourceIndex.value = 0
  },
  { immediate: true }
)

function handleError() {
  if (sourceIndex.value >= props.sources.length - 1) {
    sourceIndex.value = props.sources.length
    return
  }
  sourceIndex.value += 1
}
</script>

<style scoped>
.dark .lobe-static-icon--mono {
  filter: brightness(0) invert(1);
}
</style>
