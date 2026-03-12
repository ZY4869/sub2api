<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        ref="overlayRef"
        class="modal-overlay"
        :style="zIndexStyle"
        :aria-labelledby="dialogId"
        role="dialog"
        aria-modal="true"
        @pointerdown="handleOverlayPointerDown"
        @pointerup="handleOverlayPointerUp"
        @pointercancel="resetOverlayPointerState"
      >
        <div ref="dialogRef" :class="['modal-content', widthClasses]" @click.stop>
          <div class="modal-header">
            <h3 :id="dialogId" class="modal-title">
              {{ title }}
            </h3>
            <button
              class="-mr-2 rounded-xl p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:text-dark-500 dark:hover:bg-dark-700 dark:hover:text-dark-300"
              aria-label="Close modal"
              @click="emit('close')"
            >
              <Icon name="x" size="md" />
            </button>
          </div>

          <div class="modal-body">
            <slot></slot>
          </div>

          <div v-if="$slots.footer" class="modal-footer">
            <slot name="footer"></slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'

let dialogIdCounter = 0
const dialogId = `modal-title-${++dialogIdCounter}`

const dialogRef = ref<HTMLElement | null>(null)
const overlayRef = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null

const OVERLAY_CLOSE_MAX_DISTANCE = 6

type DialogWidth = 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'

interface Props {
  show: boolean
  title: string
  width?: DialogWidth
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  zIndex?: number
}

interface Emits {
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  width: 'normal',
  closeOnEscape: true,
  closeOnClickOutside: false,
  zIndex: 50
})

const emit = defineEmits<Emits>()

const zIndexStyle = computed(() => (props.zIndex !== 50 ? { zIndex: props.zIndex } : undefined))

const widthClasses = computed(() => {
  const widths: Record<DialogWidth, string> = {
    narrow: 'max-w-md',
    normal: 'max-w-lg',
    wide: 'w-full sm:max-w-2xl md:max-w-3xl lg:max-w-4xl',
    'extra-wide': 'w-full sm:max-w-3xl md:max-w-4xl lg:max-w-5xl xl:max-w-6xl',
    full: 'w-full sm:max-w-4xl md:max-w-5xl lg:max-w-6xl xl:max-w-7xl'
  }
  return widths[props.width]
})

const overlayPointerState = ref<{
  pointerId: number
  startedOnOverlay: boolean
  x: number
  y: number
} | null>(null)

const handleClose = () => {
  if (props.closeOnClickOutside) {
    emit('close')
  }
}

const resetOverlayPointerState = () => {
  overlayPointerState.value = null
}

const handleOverlayPointerDown = (event: PointerEvent) => {
  if (!props.closeOnClickOutside) {
    return
  }
  overlayPointerState.value = {
    pointerId: event.pointerId,
    startedOnOverlay: event.target === event.currentTarget,
    x: event.clientX,
    y: event.clientY
  }
}

const handleOverlayPointerUp = (event: PointerEvent) => {
  if (!props.closeOnClickOutside) {
    return
  }
  const state = overlayPointerState.value
  overlayPointerState.value = null
  if (!state || state.pointerId !== event.pointerId) {
    return
  }
  if (!state.startedOnOverlay || event.target !== event.currentTarget) {
    return
  }
  const movedDistance = Math.hypot(event.clientX - state.x, event.clientY - state.y)
  if (movedDistance <= OVERLAY_CLOSE_MAX_DISTANCE) {
    handleClose()
  }
}

const handleEscape = (event: KeyboardEvent) => {
  if (props.show && props.closeOnEscape && event.key === 'Escape') {
    emit('close')
  }
}

watch(
  () => props.show,
  async (isOpen) => {
    if (isOpen) {
      previousActiveElement = document.activeElement as HTMLElement
      document.body.classList.add('modal-open')

      await nextTick()
      if (dialogRef.value) {
        const firstFocusable = dialogRef.value.querySelector<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        )
        firstFocusable?.focus()
      }
      return
    }

    resetOverlayPointerState()
    document.body.classList.remove('modal-open')
    if (previousActiveElement && typeof previousActiveElement.focus === 'function') {
      previousActiveElement.focus()
    }
    previousActiveElement = null
  },
  { immediate: true }
)

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleEscape)
  resetOverlayPointerState()
  document.body.classList.remove('modal-open')
})
</script>
