import { computed, nextTick, onMounted, onUnmounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import { useWindowVirtualizer } from '@tanstack/vue-virtual'

type VirtualCardRow<T> = {
  key: string
  index: number
  start: number
  items: T[]
}

const resolveColumnCount = (width: number) => {
  if (width >= 1280) return 4
  if (width >= 1024) return 3
  if (width >= 640) return 2
  return 1
}

const chunkRows = <T>(items: T[], size: number) => {
  if (items.length === 0) return [] as T[][]

  const safeSize = Math.max(1, size)
  const rows: T[][] = []
  for (let index = 0; index < items.length; index += safeSize) {
    rows.push(items.slice(index, index + safeSize))
  }
  return rows
}

export function useVirtualAccountCardRows<T>(options: {
  items: Ref<T[]> | ComputedRef<T[]>
  estimateRowHeight?: number
  overscan?: number
}) {
  const rootRef = ref<HTMLElement | null>(null)
  const rootWidth = ref(0)
  const windowScrollMargin = ref(0)

  const updateLayoutMetrics = () => {
    if (typeof window === 'undefined') return
    const element = rootRef.value
    if (!element) return

    rootWidth.value = element.clientWidth || window.innerWidth
    windowScrollMargin.value = Math.max(
      0,
      element.getBoundingClientRect().top + window.scrollY,
    )
  }

  const columnCount = computed(() =>
    resolveColumnCount(rootWidth.value || (typeof window !== 'undefined' ? window.innerWidth : 0)),
  )

  const rows = computed(() => chunkRows(options.items.value, columnCount.value))

  const rowVirtualizer = useWindowVirtualizer(computed(() => ({
    count: rows.value.length,
    estimateSize: () => options.estimateRowHeight ?? 420,
    overscan: options.overscan ?? 2,
    scrollMargin: windowScrollMargin.value,
  })))

  const virtualItems = computed(() => rowVirtualizer.value.getVirtualItems())

  const shouldFallbackToDirectRows = computed(() =>
    rows.value.length > 0 && virtualItems.value.length === 0,
  )

  const renderedRows = computed<VirtualCardRow<T>[]>(() =>
    virtualItems.value
      .map((item) => ({
        key: String(item.key ?? item.index),
        index: item.index,
        start: Math.max(0, item.start - windowScrollMargin.value),
        items: rows.value[item.index] ?? [],
      }))
      .filter((row) => row.items.length > 0),
  )

  const totalHeight = computed(() =>
    Math.max(0, rowVirtualizer.value.getTotalSize() - windowScrollMargin.value),
  )

  const measureRow = (element: Element | null) => {
    if (element) {
      rowVirtualizer.value.measureElement(element)
    }
  }

  let resizeObserver: ResizeObserver | null = null
  let resizeHandler: (() => void) | null = null

  const attachTracking = () => {
    updateLayoutMetrics()
    if (rootRef.value && typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => {
        updateLayoutMetrics()
      })
      resizeObserver.observe(rootRef.value)
      return
    }

    if (typeof window !== 'undefined') {
      resizeHandler = () => {
        updateLayoutMetrics()
      }
      window.addEventListener('resize', resizeHandler)
    }
  }

  const detachTracking = () => {
    resizeObserver?.disconnect()
    resizeObserver = null

    if (resizeHandler && typeof window !== 'undefined') {
      window.removeEventListener('resize', resizeHandler)
      resizeHandler = null
    }
  }

  watch(
    [rows, columnCount],
    async () => {
      await nextTick()
      updateLayoutMetrics()
    },
    { flush: 'post' },
  )

  onMounted(() => {
    attachTracking()
  })

  onUnmounted(() => {
    detachTracking()
  })

  return {
    rootRef,
    directRows: rows,
    renderedRows,
    shouldFallbackToDirectRows,
    totalHeight,
    measureRow,
  }
}
