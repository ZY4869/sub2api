import type { GatewayAcceptedProtocol, GatewayProtocol } from '@/types'

export interface GeneratedProtocolGatewayDescriptor {
  id: GatewayProtocol
  displayName: string
  requestFormats: string[]
  defaultBaseUrl: string
  apiKeyPlaceholder: string
  modelImportStrategy: GatewayProtocol
  testStrategy: GatewayProtocol
  targetGroupPlatform: GatewayAcceptedProtocol | ''
}

export const generatedProtocolGatewayBuiltAt = "2026-04-09T05:39:03Z"

export const generatedProtocolGatewayDescriptors: Record<GatewayProtocol, GeneratedProtocolGatewayDescriptor> = {
  "anthropic": {
    "id": "anthropic",
    "displayName": "Anthropic",
    "requestFormats": [
      "/v1/messages",
      "/v1/messages/count_tokens"
    ],
    "defaultBaseUrl": "https://api.anthropic.com",
    "apiKeyPlaceholder": "sk-ant-...",
    "modelImportStrategy": "anthropic",
    "testStrategy": "anthropic",
    "targetGroupPlatform": "anthropic"
  },
  "gemini": {
    "id": "gemini",
    "displayName": "Gemini",
    "requestFormats": [
      "/v1beta/models/{model}:generateContent",
      "/v1beta/models/{model}:streamGenerateContent",
      "/v1beta/models/{model}:countTokens",
      "/v1beta/files",
      "/upload/v1beta/files",
      "/download/v1beta/files",
      "/v1beta/models/{model}:batchGenerateContent",
      "/v1beta/batches/{batch}",
      "/google/batch/archive/v1beta/batches",
      "/google/batch/archive/v1beta/files",
      "/v1/projects/:project/locations/:location/publishers/google/models",
      "/v1/projects/:project/locations/:location/batchPredictionJobs"
    ],
    "defaultBaseUrl": "https://generativelanguage.googleapis.com",
    "apiKeyPlaceholder": "AIza...",
    "modelImportStrategy": "gemini",
    "testStrategy": "gemini",
    "targetGroupPlatform": "gemini"
  },
  "mixed": {
    "id": "mixed",
    "displayName": "Mixed",
    "requestFormats": [
      "/v1/chat/completions",
      "/v1/responses",
      "/v1/images/generations",
      "/v1/images/edits",
      "/v1/videos",
      "/v1/videos/generations",
      "/v1/videos/:request_id",
      "/v1/messages",
      "/v1/messages/count_tokens",
      "/v1beta/models/{model}:generateContent",
      "/v1beta/models/{model}:streamGenerateContent",
      "/v1beta/models/{model}:countTokens",
      "/v1beta/files",
      "/upload/v1beta/files",
      "/download/v1beta/files",
      "/v1beta/models/{model}:batchGenerateContent",
      "/v1beta/batches/{batch}",
      "/google/batch/archive/v1beta/batches",
      "/google/batch/archive/v1beta/files",
      "/v1/projects/:project/locations/:location/publishers/google/models",
      "/v1/projects/:project/locations/:location/batchPredictionJobs"
    ],
    "defaultBaseUrl": "",
    "apiKeyPlaceholder": "gateway-key-...",
    "modelImportStrategy": "mixed",
    "testStrategy": "mixed",
    "targetGroupPlatform": ""
  },
  "openai": {
    "id": "openai",
    "displayName": "OpenAI",
    "requestFormats": [
      "/v1/chat/completions",
      "/v1/responses",
      "/v1/images/generations",
      "/v1/images/edits",
      "/v1/videos",
      "/v1/videos/generations",
      "/v1/videos/:request_id"
    ],
    "defaultBaseUrl": "https://api.openai.com",
    "apiKeyPlaceholder": "sk-proj-...",
    "modelImportStrategy": "openai",
    "testStrategy": "openai",
    "targetGroupPlatform": "openai"
  }
}
