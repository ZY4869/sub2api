import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const DESKTOP_VIEWPORT_QUERY = '(min-width: 768px)'
const VIEWPORT_ROOT_MARGIN = '200px 0px'
const VIEWPORT_THRESHOLD = 0.01

export function useViewportAutoLoadGate() {
  const rootRef = ref<HTMLElement | null>(null)
  const isDesktopViewport = ref(
    typeof window === 'undefined' || typeof window.matchMedia !== 'function'
      ? true
      : window.matchMedia(DESKTOP_VIEWPORT_QUERY).matches,
  )
  const hasEnteredViewport = ref(isDesktopViewport.value)

  let mediaQuery: MediaQueryList | null = null
  let mediaQueryListener: ((event: MediaQueryListEvent) => void) | null = null
  let visibilityObserver: IntersectionObserver | null = null

  const detachVisibilityObserver = () => {
    visibilityObserver?.disconnect()
    visibilityObserver = null
  }

  const attachVisibilityObserver = () => {
    detachVisibilityObserver()
    if (isDesktopViewport.value || hasEnteredViewport.value) return
    if (typeof window === 'undefined' || typeof IntersectionObserver === 'undefined') {
      hasEnteredViewport.value = true
      return
    }
    if (!rootRef.value) return

    visibilityObserver = new IntersectionObserver((entries) => {
      if (!entries.some((entry) => entry.isIntersecting)) return
      hasEnteredViewport.value = true
      detachVisibilityObserver()
    }, {
      root: null,
      rootMargin: VIEWPORT_ROOT_MARGIN,
      threshold: VIEWPORT_THRESHOLD,
    })
    visibilityObserver.observe(rootRef.value)
  }

  onMounted(() => {
    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
      mediaQuery = window.matchMedia(DESKTOP_VIEWPORT_QUERY)
      isDesktopViewport.value = mediaQuery.matches
      mediaQueryListener = (event: MediaQueryListEvent) => {
        isDesktopViewport.value = event.matches
      }
      if (typeof mediaQuery.addEventListener === 'function') {
        mediaQuery.addEventListener('change', mediaQueryListener)
      } else {
        mediaQuery.addListener(mediaQueryListener)
      }
    }

    if (isDesktopViewport.value) {
      hasEnteredViewport.value = true
      return
    }
    attachVisibilityObserver()
  })

  watch(
    rootRef,
    () => {
      attachVisibilityObserver()
    },
    { flush: 'post' },
  )

  watch(isDesktopViewport, (isDesktop) => {
    if (isDesktop) {
      hasEnteredViewport.value = true
      detachVisibilityObserver()
      return
    }
    hasEnteredViewport.value = false
    attachVisibilityObserver()
  })

  onBeforeUnmount(() => {
    detachVisibilityObserver()
    if (mediaQuery && mediaQueryListener) {
      if (typeof mediaQuery.removeEventListener === 'function') {
        mediaQuery.removeEventListener('change', mediaQueryListener)
      } else {
        mediaQuery.removeListener(mediaQueryListener)
      }
    }
    mediaQuery = null
    mediaQueryListener = null
  })

  const autoLoadEnabled = computed(() => {
    return isDesktopViewport.value || hasEnteredViewport.value
  })

  return {
    rootRef,
    autoLoadEnabled,
  }
}
