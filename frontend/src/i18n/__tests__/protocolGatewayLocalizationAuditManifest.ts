export type ProtocolGatewayLocalizationSink = 'template_text' | 'template_attr' | 'script_string'

export interface ProtocolGatewayLocalizationExactLiteral {
  file: string
  sink: ProtocolGatewayLocalizationSink
  literal: string
}

export const protocolGatewayAuditedVueFiles = [
  'src/components/account/AccountProtocolGatewayModelProbeEditor.vue',
  'src/components/account/CreateAccountModal.vue',
  'src/components/account/EditAccountModal.vue',
  'src/components/admin/account/AccountBatchTestModal.vue',
  'src/components/admin/account/AccountTestModal.vue',
  'src/components/admin/account/AccountTestModelSelectionFields.vue',
  'src/components/admin/account/BlacklistRetestModal.vue'
]

export const protocolGatewayAuditedScriptFiles = [
  ...protocolGatewayAuditedVueFiles,
  'src/utils/accountGatewayTestDefaults.ts',
  'src/utils/accountModelScopeCandidates.ts',
  'src/utils/providerLabels.ts'
]

export const protocolGatewayExcludedGeneratedFiles = [
  'src/generated/modelRegistry.ts'
]

export const protocolGatewayExactLiteralAllowlist: ProtocolGatewayLocalizationExactLiteral[] = []

export function isProtocolGatewayExactLiteralAllowlisted(
  file: string,
  sink: ProtocolGatewayLocalizationSink,
  literal: string
): boolean {
  return protocolGatewayExactLiteralAllowlist.some((item) =>
    item.file === file && item.sink === sink && item.literal === literal
  )
}
