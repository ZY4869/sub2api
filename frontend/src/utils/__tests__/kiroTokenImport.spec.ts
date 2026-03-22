import { describe, expect, it } from 'vitest'
import { parseKiroTokenImport } from '../kiroTokenImport'

describe('parseKiroTokenImport', () => {
  it('parses a raw access token fallback', () => {
    expect(parseKiroTokenImport('kiro-access-token')).toEqual({
      credentials: {
        access_token: 'kiro-access-token'
      },
      extra: {
        provider: 'kiro',
        source: 'kiro_import'
      }
    })
  })

  it('parses a nested Kiro token JSON payload', () => {
    const parsed = parseKiroTokenImport(JSON.stringify({
      credentials: {
        accessToken: 'access-123',
        refreshToken: 'refresh-456',
        expiresAt: '2026-03-22T12:00:00Z',
        clientId: 'client-id',
        clientSecret: 'client-secret',
        clientIdHash: 'client-hash',
        startUrl: 'https://kiro.awsapps.com/start',
        region: 'us-east-1',
        profileArn: 'arn:aws:iam::123456789012:role/Kiro'
      },
      user: {
        email: 'kiro@example.com',
        login: 'kiro-user',
        name: 'Kiro User'
      }
    }))

    expect(parsed).toEqual({
      credentials: {
        access_token: 'access-123',
        refresh_token: 'refresh-456',
        expires_at: '2026-03-22T12:00:00Z',
        client_id: 'client-id',
        client_secret: 'client-secret',
        client_id_hash: 'client-hash',
        start_url: 'https://kiro.awsapps.com/start',
        region: 'us-east-1',
        profile_arn: 'arn:aws:iam::123456789012:role/Kiro'
      },
      extra: {
        email: 'kiro@example.com',
        username: 'kiro-user',
        display_name: 'Kiro User',
        provider: 'kiro',
        source: 'kiro_import'
      },
      suggestedName: 'kiro@example.com'
    })
  })

  it('throws when access_token is missing', () => {
    expect(() => parseKiroTokenImport(JSON.stringify({ refresh_token: 'refresh-only' }))).toThrow(
      '未找到 access_token'
    )
  })
})
