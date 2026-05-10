// Lightweight date utilities — all dates are treated as local-day (no time component).

export function ymd(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function parseYMD(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(y, m - 1, d)
}

export function addDays(d: Date, n: number): Date {
  const c = new Date(d)
  c.setDate(c.getDate() + n)
  return c
}

// Days between (inclusive) — both arguments must be local-midnight dates.
export function daysBetween(a: Date, b: Date): number {
  const ms = b.getTime() - a.getTime()
  return Math.round(ms / (1000 * 60 * 60 * 24)) + 1
}

// Day index (0-based) of `d` relative to `start`.
export function dayIndex(start: Date, d: Date): number {
  return Math.round((d.getTime() - start.getTime()) / (1000 * 60 * 60 * 24))
}

export function startOfWeekMonday(d: Date): Date {
  const c = new Date(d)
  const dow = c.getDay() // 0=Sun .. 6=Sat
  const diff = (dow + 6) % 7 // shift so Monday=0
  c.setDate(c.getDate() - diff)
  c.setHours(0, 0, 0, 0)
  return c
}

export function isWeekend(d: Date): boolean {
  const dow = d.getDay()
  return dow === 0 || dow === 6
}

export function isWorkday(d: Date): boolean {
  return !isWeekend(d)
}

// Number of workdays from `start` (inclusive) up to but not including `target`.
// `start` does NOT need to be a workday — the count is just how many Mon–Fri
// days fall in the half-open interval [start, target).
export function workdayIndex(start: Date, target: Date): number {
  if (+target <= +start) return 0
  let count = 0
  let cur = new Date(start)
  cur.setHours(0, 0, 0, 0)
  const end = new Date(target)
  end.setHours(0, 0, 0, 0)
  while (+cur < +end) {
    if (isWorkday(cur)) count++
    cur.setDate(cur.getDate() + 1)
  }
  return count
}

// Date of the n-th workday at-or-after `start`. n=0 returns `start` if it's a
// workday, otherwise the next Mon–Fri.
export function workdayAt(start: Date, n: number): Date {
  let cur = new Date(start)
  cur.setHours(0, 0, 0, 0)
  let i = 0
  while (true) {
    if (isWorkday(cur)) {
      if (i === n) return cur
      i++
    }
    cur.setDate(cur.getDate() + 1)
    if (i > 10000) return cur
  }
}

// Add or subtract n workdays from `d`. Negative n moves into the past.
export function addWorkdays(d: Date, n: number): Date {
  if (n === 0) {
    const c = new Date(d)
    c.setHours(0, 0, 0, 0)
    return c
  }
  const cur = new Date(d)
  cur.setHours(0, 0, 0, 0)
  const step = n > 0 ? 1 : -1
  let left = Math.abs(n)
  while (left > 0) {
    cur.setDate(cur.getDate() + step)
    if (isWorkday(cur)) left--
  }
  return cur
}

// Workday after `d` (the next Mon–Fri strictly after `d`).
export function nextWorkday(d: Date, n = 1): Date {
  let cur = new Date(d)
  cur.setHours(0, 0, 0, 0)
  let left = n
  while (left > 0) {
    cur.setDate(cur.getDate() + 1)
    if (isWorkday(cur)) left--
  }
  return cur
}

// Workday before `d` (the previous Mon–Fri strictly before `d`).
export function prevWorkday(d: Date, n = 1): Date {
  let cur = new Date(d)
  cur.setHours(0, 0, 0, 0)
  let left = n
  while (left > 0) {
    cur.setDate(cur.getDate() - 1)
    if (isWorkday(cur)) left--
  }
  return cur
}

// First workday on or after `d`.
export function ceilToWorkday(d: Date): Date {
  let cur = new Date(d)
  cur.setHours(0, 0, 0, 0)
  while (!isWorkday(cur)) cur.setDate(cur.getDate() + 1)
  return cur
}

// Last workday on or before `d`.
export function floorToWorkday(d: Date): Date {
  let cur = new Date(d)
  cur.setHours(0, 0, 0, 0)
  while (!isWorkday(cur)) cur.setDate(cur.getDate() - 1)
  return cur
}

export function isoWeek(d: Date): number {
  // ISO 8601 week number.
  const t = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()))
  const dayNum = (t.getUTCDay() + 6) % 7
  t.setUTCDate(t.getUTCDate() - dayNum + 3)
  const firstThursday = new Date(Date.UTC(t.getUTCFullYear(), 0, 4))
  const diff = (t.getTime() - firstThursday.getTime()) / 86400000
  return 1 + Math.round((diff - 3 + ((firstThursday.getUTCDay() + 6) % 7)) / 7)
}

export function clampDay(start: Date, end: Date, d: Date): Date {
  if (d < start) return start
  if (d > end) return end
  return d
}
