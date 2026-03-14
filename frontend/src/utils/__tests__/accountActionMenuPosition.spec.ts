import { describe, expect, it } from 'vitest'
import { resolveAccountActionMenuPosition } from '@/utils/accountActionMenuPosition'

const createTarget = (rect: Partial<DOMRect>): HTMLElement => ({
  getBoundingClientRect: () => ({
    left: 0,
    top: 0,
    bottom: 0,
    width: 0,
    height: 0,
    right: 0,
    x: 0,
    y: 0,
    toJSON: () => ({}),
    ...rect
  } as DOMRect)
}) as HTMLElement

describe('accountActionMenuPosition', () => {
  it('centers menu around the trigger on narrow viewports', () => {
    const position = resolveAccountActionMenuPosition({
      event: { clientX: 140, clientY: 200 },
      target: createTarget({ left: 100, top: 120, bottom: 150, width: 80, height: 30 }),
      viewportWidth: 360,
      viewportHeight: 640
    })

    expect(position).toEqual({ top: 154, left: 40 })
  })

  it('keeps menu inside the desktop viewport', () => {
    const position = resolveAccountActionMenuPosition({
      event: { clientX: 1180, clientY: 780 },
      target: createTarget({ left: 1100, top: 700, bottom: 730, width: 48, height: 30 }),
      viewportWidth: 1280,
      viewportHeight: 800
    })

    expect(position).toEqual({ top: 552, left: 980 })
  })

  it('falls back to cursor coordinates when no trigger element is available', () => {
    const position = resolveAccountActionMenuPosition({
      event: { clientX: 320, clientY: 180 },
      target: null
    })

    expect(position).toEqual({ top: 180, left: 120 })
  })
})
