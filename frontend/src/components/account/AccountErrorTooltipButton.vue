<template>
  <div class="relative">
    <button
      ref="triggerRef"
      type="button"
      :class="['error-info-trigger inline-flex', buttonClass]"
      :aria-label="ariaLabel"
      @mouseenter="showTooltip"
      @mouseleave="hideTooltip"
      @focus="showTooltip"
      @blur="hideTooltip"
    >
      <Icon name="questionCircle" size="sm" :stroke-width="2" />
    </button>
    <Teleport to="body">
      <div
        v-if="tooltipVisible"
        ref="tooltipRef"
        class="error-info-tooltip pointer-events-none fixed z-[2100] rounded-lg bg-gray-800 px-3 py-2 text-xs text-white shadow-xl dark:bg-gray-900"
        :style="tooltipStyle"
        role="tooltip"
      >
        <div class="whitespace-pre-wrap break-words leading-relaxed text-gray-300">
          {{ message }}
        </div>
        <div
          class="absolute h-3 w-3 rotate-45 bg-gray-800 dark:bg-gray-900"
          :style="tooltipArrowStyle"
        ></div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onUnmounted, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  message: string
  ariaLabel: string
  buttonClass?: string
}>(), {
  buttonClass:
    'text-red-500 transition-colors hover:text-red-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-400/60 dark:text-red-400 dark:hover:text-red-300'
})

const tooltipVisible = ref(false)
const triggerRef = ref<HTMLElement | null>(null)
const tooltipRef = ref<HTMLElement | null>(null)
const tooltipStyle = ref<Record<string, string>>({})
const tooltipArrowStyle = ref<Record<string, string>>({})

const TOOLTIP_MARGIN = 12
const TOOLTIP_OFFSET = 10
const TOOLTIP_ARROW_SIZE = 12

const syncTooltipPosition = () => {
  const trigger = triggerRef.value
  const tooltip = tooltipRef.value
  if (!trigger || !tooltip || typeof window === 'undefined') {
    return
  }

  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight
  const maxWidth = Math.max(180, Math.min(360, viewportWidth - TOOLTIP_MARGIN * 2))
  tooltip.style.maxWidth = `${maxWidth}px`

  const triggerRect = trigger.getBoundingClientRect()
  const tooltipRect = tooltip.getBoundingClientRect()
  const spaceAbove = triggerRect.top - TOOLTIP_MARGIN
  const spaceBelow = viewportHeight - triggerRect.bottom - TOOLTIP_MARGIN

  let top = triggerRect.bottom + TOOLTIP_OFFSET
  let placement: 'top' | 'bottom' = 'bottom'
  if (tooltipRect.height > spaceBelow && spaceAbove >= spaceBelow) {
    placement = 'top'
    top = triggerRect.top - tooltipRect.height - TOOLTIP_OFFSET
  }
  top = Math.max(
    TOOLTIP_MARGIN,
    Math.min(top, viewportHeight - tooltipRect.height - TOOLTIP_MARGIN)
  )

  let left = triggerRect.left + triggerRect.width / 2 - tooltipRect.width / 2
  left = Math.max(
    TOOLTIP_MARGIN,
    Math.min(left, viewportWidth - tooltipRect.width - TOOLTIP_MARGIN)
  )

  const arrowLeft = Math.max(
    TOOLTIP_ARROW_SIZE,
    Math.min(
      triggerRect.left + triggerRect.width / 2 - left - TOOLTIP_ARROW_SIZE / 2,
      tooltipRect.width - TOOLTIP_ARROW_SIZE * 1.5
    )
  )

  tooltipStyle.value = {
    top: `${top}px`,
    left: `${left}px`,
    maxWidth: `${maxWidth}px`
  }
  tooltipArrowStyle.value = placement === 'top'
    ? { left: `${arrowLeft}px`, bottom: `-${TOOLTIP_ARROW_SIZE / 2}px` }
    : { left: `${arrowLeft}px`, top: `-${TOOLTIP_ARROW_SIZE / 2}px` }
}

const detachListeners = () => {
  if (typeof window === 'undefined') return
  window.removeEventListener('resize', syncTooltipPosition)
  window.removeEventListener('scroll', syncTooltipPosition, true)
}

const showTooltip = async () => {
  if (!props.message) return

  tooltipVisible.value = true
  await nextTick()
  syncTooltipPosition()
  if (typeof window !== 'undefined') {
    window.addEventListener('resize', syncTooltipPosition)
    window.addEventListener('scroll', syncTooltipPosition, true)
  }
}

const hideTooltip = () => {
  tooltipVisible.value = false
  detachListeners()
}

onUnmounted(() => {
  detachListeners()
})
</script>
