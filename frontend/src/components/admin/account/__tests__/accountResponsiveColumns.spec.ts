import { describe, expect, it } from 'vitest'
import { fitAccountColumns } from '../accountResponsiveColumns'

const columns = ['select','id','name','platform_type','capacity','status','schedulable','groups','usage','usage_reset_dates','last_used_at','created_at','expires_at','actions'].map(key => ({ key, label: key, class: 'w-[240px] min-w-[220px]' }))

describe('account responsive columns', () => {
  it.each([1050, 1296, 1616])('keeps core fields and fits a %i-pixel content area without modifying preferences', width => {
    const before = structuredClone(columns)
    const result = fitAccountColumns(columns, width)
    expect(result.reduce((total, column) => total + (column.width || 0), 0)).toBeLessThanOrEqual(width)
    expect(result.map(column => column.key)).toEqual(expect.arrayContaining(['select','name','platform_type','status','usage','actions']))
    expect(result.some(column => column.key === 'id' || column.key === 'usage_reset_dates')).toBe(false)
    expect(columns).toEqual(before)
    expect(result.every(column => !column.class?.includes('min-w-[220px]'))).toBe(true)
  })
  it('keeps reset dates when the usage column is hidden', () => {
    expect(fitAccountColumns(columns.filter(column => column.key !== 'usage'), 1616).some(column => column.key === 'usage_reset_dates')).toBe(true)
  })
})
