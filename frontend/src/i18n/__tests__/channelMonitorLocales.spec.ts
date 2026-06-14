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

describe('channel monitor locale messages', () => {
  it.each([
    ['zh', zh],
    ['en', en]
  ])('renders JSON hint for %s without i18n placeholder errors', async (locale, messages) => {
    const createI18n = await loadI18nWithMessageCompiler()
    const i18n = createI18n({
      legacy: false,
      locale,
      fallbackLocale: locale,
      messages: {
        [locale]: messages
      }
    })

    const hint = i18n.global.t('admin.channelMonitors.fields.jsonHint')

    expect(hint).toContain('{"x-foo":"bar"}')
  })
})
