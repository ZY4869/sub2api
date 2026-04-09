import fs from 'node:fs'
import path from 'node:path'

import { describe, expect, it } from 'vitest'

import {
  isProtocolGatewayExactLiteralAllowlisted,
  protocolGatewayAuditedScriptFiles,
  protocolGatewayAuditedVueFiles,
  protocolGatewayExcludedGeneratedFiles,
  type ProtocolGatewayLocalizationSink
} from './protocolGatewayLocalizationAuditManifest'

const frontendRoot = process.cwd()

const allowedTemplateTextPatterns = [
  /^[\s():,./_-]*$/,
  /^\d+$/
]

const allowedAttributeValuePatterns = [
  /^[\s():,./-]*$/,
  /^[A-Za-z0-9._:/-]+$/
]

function toAbsolute(relativePath: string): string {
  return path.join(frontendRoot, relativePath)
}

function extractTemplate(source: string): string {
  const match = source.match(/<template>([\s\S]*?)<\/template>/)
  return match?.[1] || ''
}

function allowlisted(file: string, sink: ProtocolGatewayLocalizationSink, literal: string): boolean {
  return isProtocolGatewayExactLiteralAllowlisted(file, sink, literal.trim())
}

function collectScriptStringOffenders(relativePath: string, source: string): string[] {
  const offenders: string[] = []
  const patterns = [
    /show(?:Error|Success|Warning)\(\s*['"`]([^'"`]+)['"`]/g,
    /throw new Error\(\s*['"`]([^'"`]+)['"`]/g
  ]

  for (const pattern of patterns) {
    for (const match of source.matchAll(pattern)) {
      const value = match[1]?.trim()
      if (!value) {
        continue
      }
      if (allowedAttributeValuePatterns.some((allowed) => allowed.test(value))) {
        continue
      }
      if (allowlisted(relativePath, 'script_string', value)) {
        continue
      }
      offenders.push(value)
    }
  }

  return offenders
}

function collectTemplateTextOffenders(relativePath: string, template: string): string[] {
  const sanitized = template.replace(/\{\{[\s\S]*?\}\}/g, '')
  const matches = sanitized.match(/>([^<>\n]+)</g) || []
  return matches
    .map((item) => item.slice(1, -1).trim())
    .filter(Boolean)
    .filter((item) => !allowedTemplateTextPatterns.some((pattern) => pattern.test(item)))
    .filter((item) => !allowlisted(relativePath, 'template_text', item))
}

function collectStaticAttributeOffenders(relativePath: string, template: string): string[] {
  const matches = [...template.matchAll(/(?:^|[\s<])(?:placeholder|title|aria-label)="([^"]+)"/gm)]
  return matches
    .map((match) => match[1].trim())
    .filter(Boolean)
    .filter((item) => !item.startsWith('{{'))
    .filter((item) => !allowedAttributeValuePatterns.some((pattern) => pattern.test(item)))
    .filter((item) => !allowlisted(relativePath, 'template_attr', item))
}

describe('protocol gateway localization audit', () => {
  it('keeps the audit manifest pinned to the touched frontend scope', () => {
    expect([...protocolGatewayAuditedVueFiles].sort()).toEqual([
      'src/components/account/AccountProtocolGatewayModelProbeEditor.vue',
      'src/components/account/CreateAccountModal.vue',
      'src/components/account/EditAccountModal.vue',
      'src/components/admin/account/AccountBatchTestModal.vue',
      'src/components/admin/account/AccountTestModal.vue',
      'src/components/admin/account/AccountTestModelSelectionFields.vue',
      'src/components/admin/account/BlacklistRetestModal.vue'
    ])
    expect([...protocolGatewayAuditedScriptFiles].sort()).toEqual([
      'src/components/account/AccountProtocolGatewayModelProbeEditor.vue',
      'src/components/account/CreateAccountModal.vue',
      'src/components/account/EditAccountModal.vue',
      'src/components/admin/account/AccountBatchTestModal.vue',
      'src/components/admin/account/AccountTestModal.vue',
      'src/components/admin/account/AccountTestModelSelectionFields.vue',
      'src/components/admin/account/BlacklistRetestModal.vue',
      'src/utils/accountGatewayTestDefaults.ts',
      'src/utils/accountModelScopeCandidates.ts',
      'src/utils/providerLabels.ts'
    ])
    expect(protocolGatewayExcludedGeneratedFiles).toEqual(['src/generated/modelRegistry.ts'])
  })

  it('does not leave hard-coded user text in audited Vue templates', () => {
    const offenders = protocolGatewayAuditedVueFiles.flatMap((relativePath) => {
      const source = fs.readFileSync(toAbsolute(relativePath), 'utf8')
      const template = extractTemplate(source)
      return [
        ...collectTemplateTextOffenders(relativePath, template).map((value) => `${relativePath}::text::${value}`),
        ...collectStaticAttributeOffenders(relativePath, template).map((value) => `${relativePath}::attr::${value}`)
      ]
    })

    expect(offenders).toEqual([])
  })

  it('does not leave hard-coded user text in audited scripts', () => {
    const offenders = protocolGatewayAuditedScriptFiles.flatMap((relativePath) => {
      const source = fs.readFileSync(toAbsolute(relativePath), 'utf8')
      return collectScriptStringOffenders(relativePath, source).map((value) => `${relativePath}::script::${value}`)
    })

    expect(offenders).toEqual([])
  })
})
