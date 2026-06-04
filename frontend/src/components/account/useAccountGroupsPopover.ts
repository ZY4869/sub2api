import { computed, onMounted, onUnmounted, ref } from 'vue'

export function useAccountGroupsPopover() {
  const moreButtonRef = ref<HTMLElement | null>(null)
  const showPopover = ref(false)

  const popoverStyle = computed(() => {
    if (!moreButtonRef.value) return {}
    const rect = moreButtonRef.value.getBoundingClientRect()
    const viewportHeight = window.innerHeight
    const viewportWidth = window.innerWidth

    let top = rect.bottom + 8
    let left = rect.left

    if (top + 280 > viewportHeight) {
      top = Math.max(8, rect.top - 280)
    }
    if (left + 384 > viewportWidth) {
      left = Math.max(8, viewportWidth - 392)
    }

    return {
      top: `${top}px`,
      left: `${left}px`,
    }
  })

  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      showPopover.value = false
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown)
  })

  return {
    moreButtonRef,
    showPopover,
    popoverStyle,
  }
}
