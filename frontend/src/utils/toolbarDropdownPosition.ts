export interface ToolbarDropdownPosition {
  top: number
  left: number
}

interface ToolbarDropdownPositionOptions {
  target: HTMLElement | null
  panelWidth: number
  panelHeight: number
  align?: 'left' | 'right'
  offset?: number
  viewportWidth?: number
  viewportHeight?: number
}

const VIEWPORT_PADDING = 8

export const resolveToolbarDropdownPosition = ({
  target,
  panelWidth,
  panelHeight,
  align = 'right',
  offset = 8,
  viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 0,
  viewportHeight = typeof window !== 'undefined' ? window.innerHeight : 0,
}: ToolbarDropdownPositionOptions): ToolbarDropdownPosition => {
  if (!target) {
    return {
      top: VIEWPORT_PADDING,
      left: Math.max(VIEWPORT_PADDING, viewportWidth - panelWidth - VIEWPORT_PADDING),
    }
  }

  const rect = target.getBoundingClientRect()
  const idealLeft =
    align === 'left' ? rect.left : rect.right - panelWidth
  const left = Math.max(
    VIEWPORT_PADDING,
    Math.min(idealLeft, viewportWidth - panelWidth - VIEWPORT_PADDING),
  )

  let top = rect.bottom + offset
  if (top + panelHeight > viewportHeight - VIEWPORT_PADDING) {
    top = rect.top - panelHeight - offset
  }
  if (top < VIEWPORT_PADDING) {
    top = VIEWPORT_PADDING
  }

  return { top, left }
}
