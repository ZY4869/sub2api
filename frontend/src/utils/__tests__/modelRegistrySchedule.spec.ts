import { describe, expect, it } from 'vitest'
import { buildModelRegistryScheduleUpsertPayload } from '../modelRegistrySchedule'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'

function createModel(): ModelRegistryDetail {
  return {
    id: 'gpt-5.4',
    display_name: 'GPT 5.4',
    provider: 'openai',
    platforms: ['openai'],
    protocol_ids: ['gpt-5.4'],
    aliases: ['gpt-5-latest'],
    pricing_lookup_ids: ['gpt-5.4'],
    preferred_protocol_ids: { openai: 'gpt-5.4' },
    modalities: ['text'],
    capabilities: ['reasoning'],
    ui_priority: 12,
    exposed_in: ['runtime', 'test'],
    status: 'stable',
    source: 'seed',
    hidden: false,
    tombstoned: false,
    available: true,
  }
}

describe('buildModelRegistryScheduleUpsertPayload', () => {
  it('preserves registry fields while applying schedule patch', () => {
    const payload = buildModelRegistryScheduleUpsertPayload(createModel(), {
      available_from: '2026-06-01T00:00:00.000Z',
      access_time_policy: {
        enabled: true,
        timezone: 'Asia/Singapore',
        weekly_windows: [{ days: [1, 2, 3, 4, 5], start: '08:00', end: '20:00' }],
        daily_allowed_minutes: 720,
      },
    })

    expect(payload).toMatchObject({
      id: 'gpt-5.4',
      provider: 'openai',
      available_from: '2026-06-01T00:00:00.000Z',
      available_until: '',
      access_time_policy: expect.objectContaining({
        enabled: true,
        timezone: 'Asia/Singapore',
      }),
    })
    expect(payload.protocol_ids).toEqual(['gpt-5.4'])
    expect(payload.exposed_in).toEqual(['runtime', 'test'])
  })
})
