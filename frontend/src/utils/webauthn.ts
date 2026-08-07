function base64UrlToBuffer(value: string): ArrayBuffer {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(normalized.length + ((4 - (normalized.length % 4)) % 4), '=')
  const binary = window.atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}

function bufferToBase64Url(buffer: ArrayBuffer | null | undefined): string | null {
  if (!buffer) return null
  const bytes = new Uint8Array(buffer)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return window.btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

function normalizeDescriptor(descriptor: any): PublicKeyCredentialDescriptor {
  return {
    ...descriptor,
    id: typeof descriptor.id === 'string' ? base64UrlToBuffer(descriptor.id) : descriptor.id
  }
}

export function normalizeCreationOptions(input: any): PublicKeyCredentialCreationOptions {
  const source = input?.publicKey || input
  return {
    ...source,
    challenge: typeof source.challenge === 'string' ? base64UrlToBuffer(source.challenge) : source.challenge,
    user: {
      ...source.user,
      id: typeof source.user?.id === 'string' ? base64UrlToBuffer(source.user.id) : source.user?.id
    },
    excludeCredentials: Array.isArray(source.excludeCredentials)
      ? source.excludeCredentials.map(normalizeDescriptor)
      : source.excludeCredentials
  }
}

export function normalizeRequestOptions(input: any): PublicKeyCredentialRequestOptions {
  const source = input?.publicKey || input
  return {
    ...source,
    challenge: typeof source.challenge === 'string' ? base64UrlToBuffer(source.challenge) : source.challenge,
    allowCredentials: Array.isArray(source.allowCredentials)
      ? source.allowCredentials.map(normalizeDescriptor)
      : source.allowCredentials
  }
}

export function credentialToJSON(credential: PublicKeyCredential): Record<string, unknown> {
  const nativeJSON = (credential as PublicKeyCredential & {
    toJSON?: () => Record<string, unknown>
  }).toJSON?.()
  if (nativeJSON) {
    return nativeJSON
  }

  const response = credential.response
  if (response instanceof AuthenticatorAttestationResponse) {
    return {
      id: credential.id,
      rawId: bufferToBase64Url(credential.rawId),
      type: credential.type,
      authenticatorAttachment: credential.authenticatorAttachment,
      response: {
        clientDataJSON: bufferToBase64Url(response.clientDataJSON),
        attestationObject: bufferToBase64Url(response.attestationObject),
        transports: response.getTransports?.() || []
      }
    }
  }
  if (response instanceof AuthenticatorAssertionResponse) {
    return {
      id: credential.id,
      rawId: bufferToBase64Url(credential.rawId),
      type: credential.type,
      authenticatorAttachment: credential.authenticatorAttachment,
      response: {
        clientDataJSON: bufferToBase64Url(response.clientDataJSON),
        authenticatorData: bufferToBase64Url(response.authenticatorData),
        signature: bufferToBase64Url(response.signature),
        userHandle: bufferToBase64Url(response.userHandle)
      }
    }
  }
  throw new Error('Unsupported WebAuthn response')
}
