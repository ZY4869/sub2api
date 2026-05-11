import { describe, expect, it } from 'vitest'
import { resolveToolbarDropdownPosition } from '@/utils/toolbarDropdownPosition'

const createTarget = (rect: Partial<DOMRect>): HTMLElement => ({
  getBoundingClientRect: () => ({
    left: 0,
    top: 0,
    right: 0,
    bottom: 0,
    width: 0,
    height: 0,
    x: 0,
    y: 0,
    toJSON: () => ({}),
    ...rect,
  } as DOMRect),
}) as HTMLElement

describe('toolbarDropdownPosition', () => {
  it('aligns the panel to the right edge by default', () => {
    const position = resolveToolbarDropdownPosition({
      target: createTarget({ left: 820, right: 920, top: 140, bottom: 172 }),
      panelWidth: 224,
      panelHeight: 240,
      viewportWidth: 1280,
      viewportHeight: 800,
    })

    expect(position).toEqual({ top: 180, left: 696 })
  })

  it('moves the panel above the trigger when there is no room below', () => {
    const position = resolveToolbarDropdownPosition({
      target: createTarget({ left: 1080, right: 1176, top: 720, bottom: 752 }),
      panelWidth: 224,
      panelHeight: 240,
      viewportWidth: 1280,
      viewportHeight: 800,
    })

    expect(position).toEqual({ top: 472, left: 952 })
  })

  it('keeps the panel inside the viewport when the trigger is near the left edge', () => {
    const position = resolveToolbarDropdownPosition({
      target: createTarget({ left: 12, right: 108, top: 200, bottom: 232 }),
      panelWidth: 224,
      panelHeight: 200,
      viewportWidth: 360,
      viewportHeight: 640,
      align: 'left',
    })

    expect(position).toEqual({ top: 240, left: 12 })
  })
})
