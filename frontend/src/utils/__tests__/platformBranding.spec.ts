import { describe, expect, it } from 'vitest'
import { getPlatformIconSources } from '../platformBranding'

describe('platformBranding', () => {
  it('uses the NewAPI icon for protocol gateway branding', () => {
    expect(getPlatformIconSources('protocol_gateway')).toEqual([
      '/lobehub-icons-static-svg/icons/newapi-color.svg',
      '/lobehub-icons-static-svg/icons/newapi.svg',
    ])
  })
})
