import { describe, expect, it } from 'vitest'
import { checkProtocolGatewayBaseUrl } from '../protocolGatewayBaseUrl'

describe('protocolGatewayBaseUrl', () => {
  it('marks loopback hosts used inside dockerized deployments', () => {
    expect(checkProtocolGatewayBaseUrl('http://127.0.0.1:8082')).toMatchObject({
      status: 'loopback',
      hostname: '127.0.0.1',
      displayHost: '127.0.0.1:8082'
    })
    expect(checkProtocolGatewayBaseUrl('http://localhost:8082')).toMatchObject({
      status: 'loopback',
      hostname: 'localhost',
      displayHost: 'localhost:8082'
    })
    expect(checkProtocolGatewayBaseUrl('http://0.0.0.0:8082')).toMatchObject({
      status: 'loopback',
      hostname: '0.0.0.0',
      displayHost: '0.0.0.0:8082'
    })
    expect(checkProtocolGatewayBaseUrl('http://[::1]:8082')).toMatchObject({
      status: 'loopback',
      hostname: '::1',
      displayHost: '[::1]:8082'
    })
  })

  it('rejects obviously invalid urls', () => {
    expect(checkProtocolGatewayBaseUrl('http://127.0.0.1.8082')).toMatchObject({
      status: 'invalid'
    })
    expect(checkProtocolGatewayBaseUrl('localhost:8082')).toMatchObject({
      status: 'invalid'
    })
  })

  it('accepts valid reachable upstream urls', () => {
    expect(checkProtocolGatewayBaseUrl('https://gateway.example.com')).toMatchObject({
      status: 'valid',
      hostname: 'gateway.example.com',
      normalizedUrl: 'https://gateway.example.com/'
    })
  })
})
