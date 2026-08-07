import { apiClient } from './client'
import type { AuthResponse, CaptchaProof } from '@/types'
import {
  credentialToJSON,
  normalizeCreationOptions,
  normalizeRequestOptions
} from '@/utils/webauthn'

export interface PasskeyCredential {
  id: number
  name: string
  credential_id: string
  last_used_at?: string
  created_at: string
  updated_at: string
}

export interface PasskeyBeginResponse {
  session_id: string
  options: Record<string, unknown>
}

export async function listPasskeys(): Promise<PasskeyCredential[]> {
  const { data } = await apiClient.get<PasskeyCredential[]>('/user/passkeys')
  return data
}

export async function registerPasskey(input: {
  password: string
  name?: string
}): Promise<PasskeyCredential> {
  const begin = await beginPasskeyRegistration(input)
  const publicKey = normalizeCreationOptions(begin.options)
  const credential = await navigator.credentials.create({ publicKey })
  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error('Passkey registration was cancelled')
  }
  return finishPasskeyRegistration({
    session_id: begin.session_id,
    name: input.name,
    credential: credentialToJSON(credential)
  })
}

export async function loginWithPasskey(proof: CaptchaProof = {}): Promise<AuthResponse> {
  const begin = await beginPasskeyLogin(proof)
  const publicKey = normalizeRequestOptions(begin.options)
  const credential = await navigator.credentials.get({ publicKey })
  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error('Passkey login was cancelled')
  }
  return finishPasskeyLogin({
    session_id: begin.session_id,
    credential: credentialToJSON(credential)
  })
}

export async function renamePasskey(id: number, name: string): Promise<void> {
  await apiClient.patch(`/user/passkeys/${id}`, { name })
}

export async function deletePasskey(id: number, password: string): Promise<void> {
  await apiClient.delete(`/user/passkeys/${id}`, { data: { password } })
}

async function beginPasskeyRegistration(input: {
  password: string
  name?: string
}): Promise<PasskeyBeginResponse> {
  const { data } = await apiClient.post<PasskeyBeginResponse>('/user/passkeys/register/begin', input)
  return data
}

async function finishPasskeyRegistration(input: {
  session_id: string
  name?: string
  credential: Record<string, unknown>
}): Promise<PasskeyCredential> {
  const { data } = await apiClient.post<PasskeyCredential>('/user/passkeys/register/finish', input)
  return data
}

async function beginPasskeyLogin(proof: CaptchaProof): Promise<PasskeyBeginResponse> {
  const { data } = await apiClient.post<PasskeyBeginResponse>('/auth/passkey/login/begin', proof)
  return data
}

async function finishPasskeyLogin(input: {
  session_id: string
  credential: Record<string, unknown>
}): Promise<AuthResponse> {
  const { data } = await apiClient.post<AuthResponse>('/auth/passkey/login/finish', input)
  return data
}

export const passkeyAPI = {
  listPasskeys,
  registerPasskey,
  loginWithPasskey,
  renamePasskey,
  deletePasskey
}
