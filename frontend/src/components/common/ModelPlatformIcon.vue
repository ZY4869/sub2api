<template>
  <svg
    v-if="iconInfo"
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="currentColor"
    fill-rule="evenodd"
    :class="sizeClass"
    class="shrink-0"
    aria-hidden="true"
  >
    <path
      v-for="(p, idx) in iconInfo.paths"
      :key="idx"
      :d="p"
      :fill="iconFillColor"
      :stroke="iconStrokeColor"
      :stroke-width="iconStrokeWidth"
      stroke-linejoin="round"
    />
  </svg>
  <svg
    v-else
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="currentColor"
    :class="sizeClass"
    class="shrink-0 text-gray-400 dark:text-gray-500"
    aria-hidden="true"
  >
    <path
      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"
    />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { normalizePlatformKey, PLATFORM_ICON_DATA } from '@/utils/platformIconData'

interface Props {
  platform?: string
  size?: 'xs' | 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm'
})

const sizeClass = computed(() => {
  const sizes = {
    xs: 'h-3 w-3',
    sm: 'h-3.5 w-3.5',
    md: 'h-4 w-4',
    lg: 'h-5 w-5'
  }
  return sizes[props.size] || sizes.sm
})

const iconInfo = computed(() => {
  const key = normalizePlatformKey(props.platform || '')
  if (!key) return null
  return PLATFORM_ICON_DATA[key] || null
})

const iconFillColor = computed(() => {
  const color = iconInfo.value?.color || ''
  const normalized = color.trim().toLowerCase()
  if (!normalized) return 'currentColor'
  // Avoid invisible pure-white icons on light backgrounds.
  if (normalized === '#fff' || normalized === '#ffffff') {
    return '#111827'
  }
  return color
})

function parseHexColor(value: string): { r: number; g: number; b: number } | null {
  const hex = String(value || '').trim().toLowerCase()
  if (!hex.startsWith('#')) return null
  const raw = hex.slice(1)
  if (raw.length === 3) {
    const r = parseInt(raw[0] + raw[0], 16)
    const g = parseInt(raw[1] + raw[1], 16)
    const b = parseInt(raw[2] + raw[2], 16)
    if (Number.isNaN(r) || Number.isNaN(g) || Number.isNaN(b)) return null
    return { r, g, b }
  }
  if (raw.length === 6) {
    const r = parseInt(raw.slice(0, 2), 16)
    const g = parseInt(raw.slice(2, 4), 16)
    const b = parseInt(raw.slice(4, 6), 16)
    if (Number.isNaN(r) || Number.isNaN(g) || Number.isNaN(b)) return null
    return { r, g, b }
  }
  return null
}

const isLightBrandColor = computed(() => {
  const color = iconInfo.value?.color || ''
  const rgb = parseHexColor(color)
  if (!rgb) return false
  const luminance = (0.2126 * rgb.r + 0.7152 * rgb.g + 0.0722 * rgb.b) / 255
  return luminance > 0.86
})

const iconStrokeColor = computed(() => {
  // Outline very light brand colors so the icon stays visible on light backgrounds.
  return isLightBrandColor.value ? 'rgba(17,24,39,0.35)' : undefined
})

const iconStrokeWidth = computed(() => (isLightBrandColor.value ? 0.9 : undefined))
</script>
