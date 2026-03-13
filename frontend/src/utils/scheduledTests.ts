export type ScheduledTestFrequencyPreset = '1h' | '2h' | '5h' | '12h' | '24h' | 'custom'

export const scheduledTestFrequencyPresetMap: Record<
  Exclude<ScheduledTestFrequencyPreset, 'custom'>,
  string
> = {
  '1h': '0 * * * *',
  '2h': '0 */2 * * *',
  '5h': '0 */5 * * *',
  '12h': '0 */12 * * *',
  '24h': '0 0 * * *'
}

export function getFrequencyPresetFromCron(
  cronExpression: string | null | undefined
): ScheduledTestFrequencyPreset {
  const cron = String(cronExpression ?? '').trim()
  const matched = Object.entries(scheduledTestFrequencyPresetMap).find(
    ([, value]) => value === cron
  )
  return (matched?.[0] as ScheduledTestFrequencyPreset | undefined) ?? 'custom'
}

export function getCronExpressionForPreset(
  preset: ScheduledTestFrequencyPreset,
  customCronExpression: string
): string {
  if (preset === 'custom') {
    return customCronExpression.trim()
  }
  return scheduledTestFrequencyPresetMap[preset]
}
