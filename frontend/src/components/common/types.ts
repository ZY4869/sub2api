/**
 * Common component types
 */
import type { CSSProperties } from 'vue'

export interface Column {
  key: string
  label: string
  sortable?: boolean
  class?: string
  formatter?: (value: any, row: any) => string
}

export type TableRowClassResolver = (row: any, index: number) => string | string[] | undefined
export type TableRowStyleResolver = (row: any, index: number) => CSSProperties | undefined
export type TableCellSpanResult = number | { colspan?: number; skip?: boolean } | null | undefined
export type TableCellSpanResolver = (
  row: any,
  column: Column,
  colIndex: number,
  columns: Column[],
) => TableCellSpanResult
