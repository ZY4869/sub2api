import { describe, expect, it } from 'vitest'

import { generatedProtocolGatewayDescriptors } from '@/generated/protocolGateway'

import {
  resolveGatewayProtocolDescriptor,
  resolveProtocolGatewayBatchRequestFormats
} from '../accountProtocolGateway'

function isGeminiBatchRequestFormat(format: string): boolean {
  const normalized = format.trim().toLowerCase()
  if (!normalized) {
    return false
  }

  return (
    normalized === '/v1beta/files' ||
    normalized === '/upload/v1beta/files' ||
    normalized === '/download/v1beta/files' ||
    normalized.includes(':batchgeneratecontent') ||
    normalized.includes('/batches/') ||
    normalized.includes('/google/batch/archive/') ||
    normalized.includes('/batchpredictionjobs')
  )
}

describe('accountProtocolGateway', () => {
  it('reads gemini request formats from the generated protocol gateway snapshot', () => {
    expect(resolveGatewayProtocolDescriptor('gemini')?.requestFormats).toEqual(
      generatedProtocolGatewayDescriptors.gemini.requestFormats
    )
  })

  it('derives protocol gateway gemini batch request formats from gemini request formats', () => {
    const expectedBatchFormats = generatedProtocolGatewayDescriptors.gemini.requestFormats.filter(
      isGeminiBatchRequestFormat
    )

    expect(resolveProtocolGatewayBatchRequestFormats({ gatewayProtocol: 'gemini' })).toEqual(
      expectedBatchFormats
    )

    expect(
      resolveProtocolGatewayBatchRequestFormats({
        gatewayProtocol: 'mixed',
        acceptedProtocols: ['openai', 'gemini']
      })
    ).toEqual(expectedBatchFormats)
  })
})
