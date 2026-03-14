export interface AccountActionMenuPosition {
  top: number
  left: number
}

interface AccountActionMenuPositionOptions {
  event: Pick<MouseEvent, 'clientX' | 'clientY'>
  target: HTMLElement | null
  viewportWidth?: number
  viewportHeight?: number
}

const MENU_WIDTH = 200
const MENU_HEIGHT = 240
const MENU_PADDING = 8
const MOBILE_BREAKPOINT = 768

export const resolveAccountActionMenuPosition = ({
  event,
  target,
  viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 0,
  viewportHeight = typeof window !== 'undefined' ? window.innerHeight : 0
}: AccountActionMenuPositionOptions): AccountActionMenuPosition => {
  if (!target) {
    return { top: event.clientY, left: event.clientX - MENU_WIDTH }
  }

  const rect = target.getBoundingClientRect()
  if (viewportWidth < MOBILE_BREAKPOINT) {
    const left = Math.max(
      MENU_PADDING,
      Math.min(
        rect.left + rect.width / 2 - MENU_WIDTH / 2,
        viewportWidth - MENU_WIDTH - MENU_PADDING
      )
    )

    let top = rect.bottom + 4
    if (top + MENU_HEIGHT > viewportHeight - MENU_PADDING) {
      top = rect.top - MENU_HEIGHT - 4
      if (top < MENU_PADDING) {
        top = MENU_PADDING
      }
    }

    return { top, left }
  }

  const left = Math.max(
    MENU_PADDING,
    Math.min(
      event.clientX - MENU_WIDTH,
      viewportWidth - MENU_WIDTH - MENU_PADDING
    )
  )
  const top = Math.min(event.clientY, viewportHeight - MENU_HEIGHT - MENU_PADDING)
  return { top, left }
}
