import { describe, expect, it } from 'vitest'

import {
  resolveGatewayProtocolDescriptor,
  resolveProtocolGatewayBatchRequestFormats
} from '../accountProtocolGateway'

describe('accountProtocolGateway', () => {
  it('reads gemini request formats from the generated protocol gateway snapshot', () => {
    expect(resolveGatewayProtocolDescriptor('gemini')?.requestFormats).toEqual([
      '/v1beta/models/{model}:generateContent',
      '/v1beta/models/{model}:streamGenerateContent',
      '/v1beta/models/{model}:countTokens',
      '/v1beta/files',
      '/upload/v1beta/files',
      '/download/v1beta/files',
      '/v1beta/models/{model}:batchGenerateContent',
      '/v1beta/batches/{batch}',
      '/v1beta/cachedContents',
      '/v1beta/fileSearchStores',
      '/v1beta/documents',
      '/v1beta/operations',
      '/v1beta/embeddings',
      '/v1beta/models/{model}:embedContent',
      '/v1beta/models/{model}:batchEmbedContents',
      '/v1beta/models/{model}:asyncBatchEmbedContent',
      '/v1beta/interactions',
      '/v1beta/live',
      '/v1beta/openai',
      '/google/batch/archive/v1beta/batches',
      '/google/batch/archive/v1beta/files',
      '/v1/projects/:project/locations/:location/publishers/google/models',
      '/v1/projects/:project/locations/:location/batchPredictionJobs'
    ])
  })

  it('derives protocol gateway gemini batch request formats from gemini request formats', () => {
    expect(resolveProtocolGatewayBatchRequestFormats({ gatewayProtocol: 'gemini' })).toEqual([
      '/v1beta/files',
      '/upload/v1beta/files',
      '/download/v1beta/files',
      '/v1beta/models/{model}:batchGenerateContent',
      '/v1beta/batches/{batch}',
      '/google/batch/archive/v1beta/batches',
      '/google/batch/archive/v1beta/files',
      '/v1/projects/:project/locations/:location/batchPredictionJobs'
    ])

    expect(
      resolveProtocolGatewayBatchRequestFormats({
        gatewayProtocol: 'mixed',
        acceptedProtocols: ['openai', 'gemini']
      })
    ).toEqual([
      '/v1beta/files',
      '/upload/v1beta/files',
      '/download/v1beta/files',
      '/v1beta/models/{model}:batchGenerateContent',
      '/v1beta/batches/{batch}',
      '/google/batch/archive/v1beta/batches',
      '/google/batch/archive/v1beta/files',
      '/v1/projects/:project/locations/:location/batchPredictionJobs'
    ])
  })
})
