export interface AbsoluteTimeLabels {
  today: string
  tomorrow: string
}

const pad = (value: number): string => String(value).padStart(2, '0')

const isSameDay = (left: Date, right: Date): boolean =>
  left.getFullYear() === right.getFullYear() &&
  left.getMonth() === right.getMonth() &&
  left.getDate() === right.getDate()

export function parseEffectiveResetAt(
  resetsAt?: string | null,
  remainingSeconds?: number | null,
  baseTime: Date = new Date()
): Date | null {
  if (typeof resetsAt === 'string' && resetsAt.trim() !== '') {
    const parsed = new Date(resetsAt)
    if (!Number.isNaN(parsed.getTime())) {
      return parsed
    }
  }

  if (typeof remainingSeconds === 'number' && Number.isFinite(remainingSeconds)) {
    const base = new Date(baseTime)
    if (Number.isNaN(base.getTime())) {
      return null
    }
    return new Date(base.getTime() + Math.max(0, remainingSeconds) * 1000)
  }

  return null
}

export function formatLocalAbsoluteTime(
  date: Date,
  now: Date,
  labels: AbsoluteTimeLabels
): string {
  if (isSameDay(date, now)) {
    return `${labels.today} ${pad(date.getHours())}:${pad(date.getMinutes())}`
  }

  const tomorrow = new Date(now)
  tomorrow.setDate(tomorrow.getDate() + 1)
  if (isSameDay(date, tomorrow)) {
    return `${labels.tomorrow} ${pad(date.getHours())}:${pad(date.getMinutes())}`
  }

  if (date.getFullYear() === now.getFullYear()) {
    return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
  }

  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function formatLocalTimestamp(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

export function formatResetCountdown(date: Date, now: Date, nowLabel: string): string {
  const diffMs = date.getTime() - now.getTime()
  if (diffMs <= 0) {
    return nowLabel
  }

  const totalMinutes = Math.max(1, Math.ceil(diffMs / 60000))
  if (totalMinutes >= 24 * 60) {
    const days = Math.floor(totalMinutes / (24 * 60))
    const hours = Math.ceil((totalMinutes - days * 24 * 60) / 60)
    if (hours <= 0) {
      return `${days}d`
    }
    if (hours >= 24) {
      return `${days + 1}d`
    }
    return `${days}d ${hours}h`
  }

  if (totalMinutes >= 60) {
    const hours = Math.floor(totalMinutes / 60)
    const minutes = totalMinutes % 60
    if (minutes === 0) {
      return `${hours}h`
    }
    return `${hours}h ${minutes}m`
  }

  return `${totalMinutes}m`
}
