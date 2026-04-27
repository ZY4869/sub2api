import { createRequire } from 'node:module'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'
import type { createI18n as createI18nType } from 'vue-i18n'

import en from '../locales/en'
import zh from '../locales/zh'

async function loadI18nWithMessageCompiler(): Promise<typeof createI18nType> {
  const require = createRequire(import.meta.url)
  const module = require(resolve(process.cwd(), 'node_modules/vue-i18n/dist/vue-i18n.cjs')) as {
    createI18n: typeof createI18nType
  }
  return module.createI18n
}

describe('baidu document ai locale messages', () => {
  it.each([
    ['zh', zh],
    ['en', en]
  ])('renders direct API URL JSON placeholder for %s without i18n placeholder errors', async (locale, messages) => {
    const createI18n = await loadI18nWithMessageCompiler()
    const i18n = createI18n({
      legacy: false,
      locale,
      fallbackLocale: locale,
      messages: {
        [locale]: messages
      }
    })

    const placeholder = i18n.global.t('admin.accounts.baiduDocumentAI.directApiUrlsPlaceholder')

    expect(placeholder).toContain('{')
    expect(placeholder).toContain('"pp-ocrv5-server": "https://..."')
    expect(placeholder.trim().endsWith('}')).toBe(true)
  })
})
