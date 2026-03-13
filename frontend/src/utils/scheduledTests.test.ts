import {
  getCronExpressionForPreset,
  getFrequencyPresetFromCron,
  scheduledTestFrequencyPresetMap
} from './scheduledTests'

describe('scheduled test frequency presets', () => {
  it('maps preset cron expressions back to the preset key', () => {
    expect(getFrequencyPresetFromCron(scheduledTestFrequencyPresetMap['1h'])).toBe('1h')
    expect(getFrequencyPresetFromCron(scheduledTestFrequencyPresetMap['2h'])).toBe('2h')
    expect(getFrequencyPresetFromCron(scheduledTestFrequencyPresetMap['5h'])).toBe('5h')
    expect(getFrequencyPresetFromCron(scheduledTestFrequencyPresetMap['12h'])).toBe('12h')
    expect(getFrequencyPresetFromCron(scheduledTestFrequencyPresetMap['24h'])).toBe('24h')
  })

  it('falls back to custom for unsupported cron expressions', () => {
    expect(getFrequencyPresetFromCron('*/30 * * * *')).toBe('custom')
    expect(getFrequencyPresetFromCron('')).toBe('custom')
  })

  it('resolves preset cron expressions for create/update payloads', () => {
    expect(getCronExpressionForPreset('1h', '')).toBe('0 * * * *')
    expect(getCronExpressionForPreset('custom', '  */30 * * * *  ')).toBe('*/30 * * * *')
  })
})
