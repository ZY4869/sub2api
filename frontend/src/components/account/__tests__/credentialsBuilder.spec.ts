import { describe, it, expect } from 'vitest'
import {
  applyAccountRequestHeaders,
  applyInterceptWarmup,
  buildAccountRequestHeaders,
  formatAccountRequestHeaders
} from '../credentialsBuilder'

describe('applyInterceptWarmup', () => {
  it('create + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, true, 'create')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('create + enabled=false: should not add the field', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, false, 'create')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, true, 'edit')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('edit + enabled=false + field exists: should delete the field', () => {
    const creds: Record<string, unknown> = { api_key: 'sk', intercept_warmup_requests: true }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=false + field absent: should not throw', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('should not affect other fields', () => {
    const creds: Record<string, unknown> = {
      api_key: 'sk',
      base_url: 'url',
      intercept_warmup_requests: true
    }
    applyInterceptWarmup(creds, false, 'edit')
    expect(creds.api_key).toBe('sk')
    expect(creds.base_url).toBe('url')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })
})

describe('account request header overrides', () => {
  it('normalizes JSON header names and values', () => {
    const result = buildAccountRequestHeaders('{ "X-Custom-Feature": " enabled ", "empty": "" }')

    expect(result).toEqual({
      ok: true,
      headers: {
        'x-custom-feature': 'enabled'
      }
    })
  })

  it('rejects sensitive protocol headers', () => {
    expect(buildAccountRequestHeaders('{ "Authorization": "Bearer bad" }')).toEqual({
      ok: false,
      errorKey: 'admin.accounts.requestHeadersForbiddenName'
    })
  })

  it('applies request_headers and removes legacy keys', () => {
    const creds: Record<string, unknown> = {
      request_header_overrides: { 'x-old': '1' },
      header_overrides: { 'x-legacy': '2' }
    }

    const err = applyAccountRequestHeaders(creds, '{ "X-New": "3" }')

    expect(err).toBeNull()
    expect(creds).toEqual({
      request_headers: { 'x-new': '3' }
    })
  })

  it('formats stored request headers for editing', () => {
    expect(formatAccountRequestHeaders({
      request_headers: {
        'X-Z': ' z ',
        authorization: 'blocked',
        'x-a': 'a'
      }
    })).toBe('{\n  "x-a": "a",\n  "x-z": "z"\n}')
  })
})
