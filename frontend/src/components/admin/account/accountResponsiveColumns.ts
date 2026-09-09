import type { Column } from '@/components/common/types'

const widths: Record<string, number> = {
  select: 36, id: 76, name: 190, platform_type: 124, capacity: 112,
  status: 164, schedulable: 64, groups: 120, today_stats: 164, usage: 230,
  usage_reset_dates: 220, proxy: 120, priority: 72, scheduler_score: 112,
  rate_multiplier: 96, last_used_at: 112, created_at: 144, expires_at: 144,
  notes: 180, actions: 184,
}
const core = new Set(['select', 'name', 'platform_type', 'status', 'usage', 'actions'])

/** Fold for display only; never write the user's selected-column preferences. */
export function fitAccountColumns(columns: Column[], availableWidth: number): Column[] {
  const selected = new Set(columns.map(column => column.key))
  const candidates = columns.filter(column =>
    !(column.key === 'id' && selected.has('name')) &&
    !(column.key === 'usage_reset_dates' && selected.has('usage')),
  )
  const visible = new Set(candidates.filter(column => core.has(column.key)).map(column => column.key))
  let used = candidates.filter(column => visible.has(column.key)).reduce((sum, column) => sum + (widths[column.key] || 144), 0)
  for (const column of candidates) {
    if (visible.has(column.key)) continue
    const width = widths[column.key] || 144
    if (used + width <= availableWidth) { visible.add(column.key); used += width }
  }
  return candidates.filter(column => visible.has(column.key)).map(column => ({
    ...column,
    width: widths[column.key] || 144,
    class: [...(column.class || '').split(/\s+/).filter(value => !/^(?:min-|max-)?w-/.test(value) && value !== 'whitespace-nowrap'), 'min-w-0 whitespace-normal'].join(' '),
  }))
}
