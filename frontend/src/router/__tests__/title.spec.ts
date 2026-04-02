import { describe, expect, it } from 'vitest'
import { i18n } from '@/i18n'
import { resolveDocumentTitle } from '@/router/title'

describe('resolveDocumentTitle', () => {
  it('uses the route title when no title key is provided', () => {
    expect(resolveDocumentTitle('Usage Records', 'My Site')).toBe('Usage Records - My Site')
  })

  it('falls back to the site name when the route title is missing', () => {
    expect(resolveDocumentTitle(undefined, 'My Site')).toBe('My Site')
  })

  it('falls back to the default site name when the provided name is blank', () => {
    expect(resolveDocumentTitle('Dashboard', '')).toBe('Dashboard - Sub2API')
    expect(resolveDocumentTitle(undefined, '   ')).toBe('Sub2API')
  })

  it('updates the title when the site name changes', () => {
    const before = resolveDocumentTitle('Admin Dashboard', 'Alpha')
    const after = resolveDocumentTitle('Admin Dashboard', 'Beta')

    expect(before).toBe('Admin Dashboard - Alpha')
    expect(after).toBe('Admin Dashboard - Beta')
  })

  it('prefers the localized titleKey when a translation exists', () => {
    i18n.global.setLocaleMessage('zh', {
      ui: {
        routeTitles: {
          oauthCallback: '授权回调'
        }
      }
    })
    i18n.global.locale.value = 'zh'

    expect(resolveDocumentTitle('OAuth Callback', 'My Site', 'ui.routeTitles.oauthCallback')).toBe('授权回调 - My Site')
  })
})
