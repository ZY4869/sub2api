import { describe, expect, it } from 'vitest'
import {
  buildKiroOAuthPayload,
  parseKiroOAuthCallback
} from '../kiroOAuth'

describe('kiroOAuth utils', () => {
  it('parses callback url and extracts code/state', () => {
    expect(
      parseKiroOAuthCallback('http://localhost:19877/oauth/callback?code=abc123&state=state456')
    ).toEqual({
      code: 'abc123',
      state: 'state456'
    })
  })

  it('builds normalized account payload from exchange result', () => {
    expect(
      buildKiroOAuthPayload({
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        expires_at: '2026-03-22T12:00:00Z',
        auth_method: 'builder_id',
        client_id: 'client-id',
        client_secret: 'client-secret',
        region: 'us-east-1',
        email: 'kiro@example.com',
        username: 'kiro-user',
        display_name: 'Kiro User'
      })
    ).toEqual({
      credentials: {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        expires_at: '2026-03-22T12:00:00Z',
        auth_method: 'builder_id',
        client_id: 'client-id',
        client_secret: 'client-secret',
        region: 'us-east-1'
      },
      extra: {
        source: 'kiro_browser_oauth',
        provider: 'aws',
        email: 'kiro@example.com',
        username: 'kiro-user',
        display_name: 'Kiro User'
      },
      suggestedName: 'kiro@example.com'
    })
  })
})
