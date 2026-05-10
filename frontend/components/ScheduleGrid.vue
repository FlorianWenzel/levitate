<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Assignment, Person, Project, TimeOff } from '~/types/api'
import {
  addDays,
  addWorkdays,
  ceilToWorkday,
  floorToWorkday,
  isoWeek,
  isWorkday,
  parseYMD,
  startOfWeekMonday,
  workdayAt,
  workdayIndex,
  ymd,
} from '~/utils/dates'

type Props = {
  people: Person[]
  projects: Project[]
  assignments: Assignment[]
  timeOff?: TimeOff[]
  // First workday in the visible window. If a non-workday is passed it gets
  // snapped forward to the next Mon–Fri.
  rangeStart: Date
  // How many workday columns to render. Each column = one Mon–Fri date.
  numWorkdays?: number
  dayWidth?: number
  rowHeight?: number
}

const props = withDefaults(defineProps<Props>(), {
  numWorkdays: 60,
  dayWidth: 36,
  rowHeight: 72,
  timeOff: () => [],
})

// rangeStart snapped to a workday — the first column.
const firstWorkday = computed(() => ceilToWorkday(props.rangeStart))
// Inclusive last workday displayed.
const lastWorkday = computed(() => workdayAt(firstWorkday.value, props.numWorkdays - 1))

// Each row's default height is "8 h" worth of pixels. Bars are sized by their
// hours/day (4 h → half height) and stack flush on top of each other when they
// overlap horizontally — no gaps. Anything ≤ 2 h gets clamped to MIN_BAR_HEIGHT
// so the project label inside the bar stays readable.
const FULL_DAY_HOURS = 8
const FULL_DAY_PX = 64 // 8 h fills 64 px → PX_PER_HOUR = 8
const PX_PER_HOUR = FULL_DAY_PX / FULL_DAY_HOURS
// 16 px is enough for one line of the bar's text label — 0.25–2 h bars all
// render at this minimum height.
const MIN_BAR_HEIGHT = 16
const MIN_HOURS = 1
const MAX_HOURS = 24

function barVisualHeight(hours: number) {
  return Math.max(MIN_BAR_HEIGHT, Math.min(hours * PX_PER_HOUR, FULL_DAY_PX))
}

const emit = defineEmits<{
  (e: 'edit-assignment', a: Assignment): void
  (e: 'move-assignment', payload: { id: string; start: string; end: string; hoursPerDay: number }): void
  (e: 'create-assignment', payload: { personId: string; start: string; end: string }): void
  (e: 'edit-time-off', t: TimeOff): void
  (e: 'context-menu', payload: { assignment: Assignment; x: number; y: number; date: string }): void
}>()

const auth = useAuthStore()

// Inclusive day-after-last-visible-workday — used for clipping math below.
const rangeEnd = computed(() => lastWorkday.value)

const days = computed(() => {
  const out: { date: Date; dom: number; weekStart: boolean }[] = []
  for (let i = 0; i < props.numWorkdays; i++) {
    const d = workdayAt(firstWorkday.value, i)
    out.push({ date: d, dom: d.getDate(), weekStart: d.getDay() === 1 })
  }
  return out
})

const weekHeaders = computed(() => {
  const groups: { weekNo: number; label: string; cols: number; offset: number }[] = []
  let i = 0
  while (i < props.numWorkdays) {
    const d = workdayAt(firstWorkday.value, i)
    const wMonday = startOfWeekMonday(d)
    const weekNo = isoWeek(wMonday)
    const offset = i
    let cols = 0
    while (i < props.numWorkdays && +startOfWeekMonday(workdayAt(firstWorkday.value, i)) === +wMonday) {
      i++
      cols++
    }
    const label = wMonday.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
    groups.push({ weekNo, label, cols, offset })
  }
  return groups
})

const trackWidth = computed(() => props.numWorkdays * props.dayWidth)

// Workday-aware index helpers: convert a calendar date to a column index in
// the visible grid (or null if it falls outside / on a weekend off the grid).
function colIndexFor(d: Date): number {
  return workdayIndex(firstWorkday.value, d)
}

function projectColor(id: string): string {
  return props.projects.find((p) => p.id === id)?.color ?? '#64748B'
}

function projectName(id: string): string {
  return props.projects.find((p) => p.id === id)?.name ?? '—'
}

function assignmentsForPerson(personId: string): Assignment[] {
  return props.assignments.filter((a) => a.person_id === personId)
}

// Greedy lane assignment: sort by start_date, then place each assignment in
// the first lane whose last interval ends before this one starts. Open a new
// lane when no existing one fits.
function assignLanes(asgs: Assignment[]): Map<string, number> {
  const sorted = [...asgs].sort((a, b) =>
    a.start_date.localeCompare(b.start_date) || a.id.localeCompare(b.id),
  )
  const laneEnds: string[] = []
  const out = new Map<string, number>()
  for (const a of sorted) {
    let lane = laneEnds.findIndex((end) => end < a.start_date)
    if (lane === -1) {
      lane = laneEnds.length
      laneEnds.push(a.end_date)
    } else {
      laneEnds[lane] = a.end_date
    }
    out.set(a.id, lane)
  }
  return out
}

type RowLayout = {
  lanes: Map<string, number>     // assignment.id → lane index (0-based)
  tops: Map<string, number>      // assignment.id → top px (cumulative height of stuff stacked above)
  rowHeight: number              // tallest day-stack, never less than the 8 h default
}

// Build a row's visual layout: tight-packing where each bar's height = hours,
// and bars in higher lanes sit directly on top of overlapping bars below them
// (no gaps).
function computeRowLayout(asgs: Assignment[]): RowLayout {
  if (asgs.length === 0) {
    return { lanes: new Map(), tops: new Map(), rowHeight: FULL_DAY_PX }
  }
  const lanes = assignLanes(asgs)
  let laneCount = 1
  for (const v of lanes.values()) if (v + 1 > laneCount) laneCount = v + 1

  // Index every day from the earliest start to the latest end.
  let minStart = asgs[0].start_date
  let maxEnd = asgs[0].end_date
  for (const a of asgs) {
    if (a.start_date < minStart) minStart = a.start_date
    if (a.end_date > maxEnd) maxEnd = a.end_date
  }
  const startMs = parseYMD(minStart).getTime()
  const endMs = parseYMD(maxEnd).getTime()
  const dayCount = Math.round((endMs - startMs) / 86400000) + 1

  // fill[lane][dayIdx] = bar height at that lane/day (0 if none).
  const fill: number[][] = Array.from({ length: laneCount }, () => new Array(dayCount).fill(0))
  for (const a of asgs) {
    const lane = lanes.get(a.id) ?? 0
    const s = Math.round((parseYMD(a.start_date).getTime() - startMs) / 86400000)
    const e = Math.round((parseYMD(a.end_date).getTime() - startMs) / 86400000)
    const h = barVisualHeight(effectiveHours(a))
    for (let d = s; d <= e; d++) fill[lane][d] = h
  }

  // Lane tops = max across all days of the cumulative height of lower-numbered
  // lanes on that day. Bars in the same lane share the same visual top.
  const laneTops = new Array(laneCount).fill(0)
  for (let L = 1; L < laneCount; L++) {
    let m = 0
    for (let d = 0; d < dayCount; d++) {
      let cum = 0
      for (let k = 0; k < L; k++) cum += fill[k][d]
      if (cum > m) m = cum
    }
    laneTops[L] = m
  }

  // Row height = tallest day-stack, but never less than the 8 h default.
  let rowHeight = FULL_DAY_PX
  for (let d = 0; d < dayCount; d++) {
    let cum = 0
    for (let k = 0; k < laneCount; k++) cum += fill[k][d]
    if (cum > rowHeight) rowHeight = cum
  }

  const tops = new Map<string, number>()
  for (const a of asgs) tops.set(a.id, laneTops[lanes.get(a.id) ?? 0])

  return { lanes, tops, rowHeight }
}

const personLayouts = computed(() => {
  // Touch the drag draft so the layout recomputes during a resize-hours drag.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const _ = drag.value?.draftHoursPerDay
  const map = new Map<string, RowLayout>()
  for (const p of props.people) {
    map.set(p.id, computeRowLayout(assignmentsForPerson(p.id)))
  }
  return map
})

function layoutFor(personId: string): RowLayout {
  return personLayouts.value.get(personId) ?? { lanes: new Map(), tops: new Map(), rowHeight: FULL_DAY_PX }
}

function laneFor(assignmentId: string, personId: string): number {
  return layoutFor(personId).lanes.get(assignmentId) ?? 0
}

function topFor(assignmentId: string, personId: string): number {
  return layoutFor(personId).tops.get(assignmentId) ?? 0
}

function rowHeightFor(personId: string): number {
  return layoutFor(personId).rowHeight
}

function timeOffForPerson(personId: string): TimeOff[] {
  return (props.timeOff ?? []).filter((t) => t.person_id === personId)
}

// ----- Time-off auto-clip -----
//
// Given a desired date range and the day the user is "anchoring" on (where the
// pointer is or where the drag started), expand outward from the anchor as far
// as possible without crossing any of this person's time-off ranges. Returns
// null if the anchor itself falls inside a time-off range — i.e. the operation
// should be discarded.
function isBlocked(personId: string, d: Date): boolean {
  return (props.timeOff ?? []).some((t) => {
    if (t.person_id !== personId) return false
    const ts = parseYMD(t.start_date)
    const te = parseYMD(t.end_date)
    return +d >= +ts && +d <= +te
  })
}

function clipRange(personId: string, start: Date, end: Date, anchor: Date): { start: Date; end: Date } | null {
  if (isBlocked(personId, anchor)) return null
  // Walk in workday steps — weekends are not on the grid and don't count.
  let s = anchor
  while (+s > +start) {
    const prev = addWorkdays(s, -1)
    if (+prev < +start) break
    if (isBlocked(personId, prev)) break
    s = prev
  }
  let e = anchor
  while (+e < +end) {
    const nxt = addWorkdays(e, 1)
    if (+nxt > +end) break
    if (isBlocked(personId, nxt)) break
    e = nxt
  }
  return { start: s, end: e }
}

function clippedDrag(d: DragState, a: Assignment): { start: Date; end: Date } | null {
  if (d.mode === 'resize-hours') return { start: d.draftStart, end: d.draftEnd }
  if (d.mode === 'resize-right') {
    if (isBlocked(a.person_id, d.draftStart)) return null
    let e = d.draftStart
    while (+e < +d.draftEnd) {
      const nxt = addWorkdays(e, 1)
      if (+nxt > +d.draftEnd) break
      if (isBlocked(a.person_id, nxt)) break
      e = nxt
    }
    return { start: d.draftStart, end: e }
  }
  if (d.mode === 'resize-left') {
    if (isBlocked(a.person_id, d.draftEnd)) return null
    let s = d.draftEnd
    while (+s > +d.draftStart) {
      const prev = addWorkdays(s, -1)
      if (+prev < +d.draftStart) break
      if (isBlocked(a.person_id, prev)) break
      s = prev
    }
    return { start: s, end: d.draftEnd }
  }
  return null
}

// Project a calendar date range onto the workday grid. Snaps weekend ends to
// the nearest workday inside the range, then computes the visible column span.
function clip(a: { start_date: string; end_date: string }) {
  const startCal = parseYMD(a.start_date)
  const endCal = parseYMD(a.end_date)
  const startWd = ceilToWorkday(startCal)
  const endWd = floorToWorkday(endCal)
  if (+endWd < +startWd) return null // entirely on a weekend
  const left = Math.max(0, colIndexFor(startWd))
  const rightExclusive = Math.min(props.numWorkdays, colIndexFor(endWd) + 1)
  if (rightExclusive <= 0 || left >= props.numWorkdays) return null
  return {
    left,
    width: rightExclusive - left,
    startsBefore: startCal < firstWorkday.value,
    endsAfter: endCal > lastWorkday.value,
  }
}

// ----- Drag/resize state -----

type DragMode = 'resize-left' | 'resize-right' | 'resize-hours'
type DragState = {
  id: string
  mode: DragMode
  origStart: Date
  origEnd: Date
  origHoursPerDay: number
  startX: number
  startY: number
  draftStart: Date
  draftEnd: Date
  draftHoursPerDay: number
  moved: boolean
}

const drag = ref<DragState | null>(null)

// New-assignment range selection (click-and-drag on empty track).
type Selection = {
  personId: string
  rectLeft: number
  startDay: number
  endDay: number
}
const selection = ref<Selection | null>(null)

function selectionDays(s: Selection) {
  const lo = Math.min(s.startDay, s.endDay)
  const hi = Math.max(s.startDay, s.endDay)
  return { lo, hi, count: hi - lo + 1 }
}

function selectionDateRange(s: Selection) {
  const c = clippedSelection(s)
  if (c) return c
  const { lo, hi } = selectionDays(s)
  return { start: workdayAt(firstWorkday.value, lo), end: workdayAt(firstWorkday.value, hi) }
}

function clippedSelection(s: Selection): { start: Date; end: Date } | null {
  const { lo, hi } = selectionDays(s)
  const startD = workdayAt(firstWorkday.value, lo)
  const endD = workdayAt(firstWorkday.value, hi)
  const anchorD = workdayAt(firstWorkday.value, s.startDay)
  return clipRange(s.personId, startD, endD, anchorD)
}

function selectionRenderRange(s: Selection): { lo: number; hi: number } | null {
  const c = clippedSelection(s)
  if (!c) return null
  return {
    lo: colIndexFor(c.start),
    hi: colIndexFor(c.end),
  }
}

function selectionLabel(s: Selection) {
  const c = clippedSelection(s)
  if (!c) return 'time off'
  const fmt = (d: Date) => d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  if (+c.start === +c.end) return fmt(c.start)
  return `${fmt(c.start)} – ${fmt(c.end)}`
}

function selectionDayCount(s: Selection): number {
  const c = clippedSelection(s)
  if (!c) return 0
  // Count Mon–Fri workdays inclusive in [c.start, c.end].
  return workdayIndex(c.start, addDays(c.end, 1))
}

function onPointerDown(e: PointerEvent, a: Assignment, mode: DragMode) {
  if (!auth.isAdmin) return
  e.stopPropagation()
  ;(e.target as HTMLElement).setPointerCapture?.(e.pointerId)
  const hours = Number(a.hours_per_day) || 0
  drag.value = {
    id: a.id,
    mode,
    origStart: parseYMD(a.start_date),
    origEnd: parseYMD(a.end_date),
    origHoursPerDay: hours,
    startX: e.clientX,
    startY: e.clientY,
    draftStart: parseYMD(a.start_date),
    draftEnd: parseYMD(a.end_date),
    draftHoursPerDay: hours,
    moved: false,
  }
}

function onPointerMove(e: PointerEvent) {
  if (drag.value) {
    const dx = e.clientX - drag.value.startX
    const dy = e.clientY - drag.value.startY
    const shiftDays = Math.round(dx / props.dayWidth)
    // Up = increase hours; in screen coords up means dy negative.
    const shiftHours = Math.round(-dy / PX_PER_HOUR)
    if (drag.value.mode === 'resize-hours') {
      if (shiftHours !== 0) drag.value.moved = true
    } else {
      if (shiftDays !== 0) drag.value.moved = true
    }
    if (drag.value.mode === 'resize-left') {
      const ns = addWorkdays(drag.value.origStart, shiftDays)
      if (ns <= drag.value.origEnd) drag.value.draftStart = ns
    } else if (drag.value.mode === 'resize-right') {
      const ne = addWorkdays(drag.value.origEnd, shiftDays)
      if (ne >= drag.value.origStart) drag.value.draftEnd = ne
    } else if (drag.value.mode === 'resize-hours') {
      const next = drag.value.origHoursPerDay + shiftHours
      drag.value.draftHoursPerDay = Math.max(MIN_HOURS, Math.min(MAX_HOURS, next))
    }
  }
  if (selection.value) {
    const x = e.clientX - selection.value.rectLeft
    const day = Math.floor(x / props.dayWidth)
    selection.value.endDay = Math.max(0, Math.min(props.numWorkdays - 1, day))
  }
}

function onPointerUp(e: PointerEvent, a: Assignment) {
  finishDrag(e, a)
}

// Catches pointerup that lands outside the bar (e.g. after a long drag) or
// off the track. Commits both assignment-drag and new-range-selection.
function onPointerUpGlobal(e: PointerEvent) {
  if (drag.value) {
    const a = props.assignments.find((x) => x.id === drag.value!.id)
    if (a) finishDrag(e, a)
    else drag.value = null
  }
  if (selection.value) {
    const sel = selection.value
    selection.value = null
    const clipped = clippedSelection(sel)
    if (!clipped) return // anchor day was inside time-off
    emit('create-assignment', {
      personId: sel.personId,
      start: ymd(clipped.start),
      end: ymd(clipped.end),
    })
  }
}

function finishDrag(e: PointerEvent, a: Assignment) {
  const d = drag.value
  drag.value = null
  if (!d) return
  try { (e.target as HTMLElement).releasePointerCapture?.(e.pointerId) } catch {}
  // No-op if the user just clicked the handle without moving; the bar's @click
  // handler is responsible for opening the edit modal when the body is clicked.
  if (!d.moved) return
  let finalStart = d.draftStart
  let finalEnd = d.draftEnd
  if (d.mode !== 'resize-hours') {
    const clipped = clippedDrag(d, a)
    if (!clipped) return
    finalStart = clipped.start
    finalEnd = clipped.end
  }
  const datesUnchanged = +finalStart === +d.origStart && +finalEnd === +d.origEnd
  const hoursUnchanged = d.draftHoursPerDay === d.origHoursPerDay
  if (datesUnchanged && hoursUnchanged) return
  emit('move-assignment', {
    id: a.id,
    start: ymd(finalStart),
    end: ymd(finalEnd),
    hoursPerDay: d.draftHoursPerDay,
  })
}

function onPointerCancel() {
  drag.value = null
}

function draftOffsetForBar(a: Assignment) {
  if (!drag.value || drag.value.id !== a.id) return null
  const clipped = clippedDrag(drag.value, a)
  if (clipped) return { start: clipped.start, end: clipped.end }
  // Anchor blocked — show the un-clipped (overlapping) range so the user gets visual feedback
  // that the move is invalid; commit will revert.
  return { start: drag.value.draftStart, end: drag.value.draftEnd }
}

function effectiveHours(a: Assignment): number {
  if (drag.value && drag.value.id === a.id) return drag.value.draftHoursPerDay
  return Number(a.hours_per_day) || 0
}

function onBarContextMenu(e: MouseEvent, a: Assignment) {
  if (!auth.isAdmin) return
  const track = (e.currentTarget as HTMLElement).closest('.cursor-crosshair') as HTMLElement | null
  if (!track) return
  const tr = track.getBoundingClientRect()
  const dayIdx = Math.max(0, Math.min(props.numWorkdays - 1, Math.floor((e.clientX - tr.left) / props.dayWidth)))
  emit('context-menu', {
    assignment: a,
    x: e.clientX,
    y: e.clientY,
    date: ymd(workdayAt(firstWorkday.value, dayIdx)),
  })
}

function onTrackPointerDown(e: PointerEvent, person: Person) {
  if (!auth.isAdmin) return
  if (e.button !== 0) return
  // Don't start a selection if the pointer is on an existing bar/time-off.
  const t = e.target as HTMLElement
  if (t.closest('[data-bar]') || t.closest('[data-time-off]')) return
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  const day = Math.floor((e.clientX - rect.left) / props.dayWidth)
  if (day < 0 || day >= props.numWorkdays) return
  selection.value = {
    personId: person.id,
    rectLeft: rect.left,
    startDay: day,
    endDay: day,
  }
}
</script>

<template>
  <div
    data-schedule-grid
    class="select-none border border-slate-200 rounded bg-white overflow-x-auto"
    @pointermove="onPointerMove"
    @pointerup="onPointerUpGlobal"
    @pointercancel="onPointerCancel"
  >
    <!-- Header: weeks + days -->
    <div class="sticky top-0 z-10 bg-white">
      <div class="flex border-b border-slate-200">
        <div class="sticky left-0 z-20 w-44 shrink-0 border-r border-slate-200 bg-white px-3 py-1 text-xs font-medium text-slate-500">
          Week
        </div>
        <div class="flex" :style="{ width: trackWidth + 'px' }">
          <div
            v-for="w in weekHeaders"
            :key="w.weekNo + '-' + w.offset"
            class="border-r border-slate-200 px-2 py-1 text-xs text-slate-600 truncate"
            :style="{ width: w.cols * dayWidth + 'px' }"
          >
            <span class="font-medium">W{{ w.weekNo }}</span>
            <span class="ml-1 text-slate-400">{{ w.label }}</span>
          </div>
        </div>
      </div>
      <div class="flex border-b border-slate-200">
        <div class="sticky left-0 z-20 w-44 shrink-0 border-r border-slate-200 bg-white px-3 py-1 text-xs font-medium text-slate-500">
          Person
        </div>
        <div class="flex" :style="{ width: trackWidth + 'px' }">
          <div
            v-for="(d, i) in days"
            :key="i"
            class="flex flex-col items-center justify-center border-r border-slate-100 text-[10px] text-slate-500"
            :class="{ 'border-l-2 border-l-slate-300': d.weekStart && i > 0 }"
            :style="{ width: dayWidth + 'px', height: '36px' }"
          >
            <span>{{ d.date.toLocaleDateString(undefined, { weekday: 'narrow' }) }}</span>
            <span class="font-medium">{{ d.dom }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Person rows -->
    <div v-if="!people.length" class="px-6 py-10 text-center text-sm text-slate-400">
      No people yet — add some on the People page.
    </div>
    <div
      v-for="person in people"
      :key="person.id"
      :data-person-id="person.id"
      class="flex border-b border-slate-100"
    >
      <div
        class="sticky left-0 z-10 w-44 shrink-0 border-r border-slate-200 bg-white px-3 flex flex-col justify-center leading-tight"
        :style="{ height: rowHeightFor(person.id) + 'px' }"
      >
        <div class="text-[12px] font-medium text-slate-900 truncate">{{ person.name }}</div>
        <div class="text-[10px] text-slate-500">{{ person.weekly_capacity_hours }} h/wk</div>
      </div>
      <div
        class="relative cursor-crosshair"
        :style="{ width: trackWidth + 'px', height: rowHeightFor(person.id) + 'px' }"
        @pointerdown="onTrackPointerDown($event, person)"
      >
        <!-- Day cell backgrounds. Weekends are not rendered at all so the grid
             collapses Mon→Fri→Mon with no gap. -->
        <div
          v-for="(d, i) in days"
          :key="i"
          class="absolute top-0 h-full border-r border-slate-100"
          :class="{ 'border-l-2 border-l-slate-300': d.weekStart && i > 0 }"
          :style="{ left: i * dayWidth + 'px', width: dayWidth + 'px' }"
        />

        <!-- Range-selection ghost (drag-to-create). Auto-clipped to skip time-off. -->
        <template v-if="selection && selection.personId === person.id && selectionRenderRange(selection)">
          <div
            data-selection
            class="absolute top-2 bottom-2 rounded border-2 border-dashed border-slate-700/60 bg-slate-900/15 pointer-events-none flex items-center justify-center"
            :style="{
              left: selectionRenderRange(selection)!.lo * dayWidth + 'px',
              width: (selectionRenderRange(selection)!.hi - selectionRenderRange(selection)!.lo + 1) * dayWidth - 2 + 'px',
            }"
          >
            <span class="text-[11px] font-medium text-slate-700 bg-white/80 rounded px-1.5 py-0.5 shadow-sm whitespace-nowrap">
              {{ selectionLabel(selection) }}
              <span v-if="selectionDayCount(selection) > 1" class="text-slate-500">
                · {{ selectionDayCount(selection) }}d
              </span>
            </span>
          </div>
        </template>

        <!-- Time-off: full-row diagonal-striped grey blocker. Sits behind assignments
             but in front of day-cell backgrounds; assignments dragged into it are
             auto-clipped on commit. -->
        <template v-for="t in timeOffForPerson(person.id)" :key="'t-' + t.id">
          <div
            v-if="clip(t)"
            data-time-off
            :data-time-off-id="t.id"
            class="absolute inset-y-0 cursor-pointer group"
            :style="{
              left: clip(t)!.left * dayWidth + 'px',
              width: clip(t)!.width * dayWidth + 'px',
              backgroundImage:
                'repeating-linear-gradient(-45deg, rgba(100,116,139,0.35) 0 4px, rgba(203,213,225,0.35) 4px 8px)',
            }"
            :title="`${t.type}${t.notes ? ' — ' + t.notes : ''}`"
            @click.stop="emit('edit-time-off', t)"
            @pointerdown.stop
          >
            <div class="absolute top-0.5 inset-x-0 text-center text-[10px] font-medium uppercase tracking-wide text-slate-600 bg-white/70 truncate px-1">
              {{ t.type }}
            </div>
          </div>
        </template>

        <!-- Assignments -->
        <template v-for="a in assignmentsForPerson(person.id)" :key="a.id">
          <div
            v-if="clip(a)"
            data-bar
            :data-assignment-id="a.id"
            :data-hours="effectiveHours(a)"
            :data-lane="laneFor(a.id, person.id)"
            class="absolute rounded text-white text-[11px] leading-tight overflow-hidden flex items-stretch shadow-sm cursor-pointer"
            :style="{
              top: topFor(a.id, person.id) + 'px',
              height: barVisualHeight(effectiveHours(a)) + 'px',
              left: ((draftOffsetForBar(a)
                ? Math.max(0, colIndexFor(draftOffsetForBar(a)!.start))
                : clip(a)!.left) * dayWidth) + 'px',
              width: ((draftOffsetForBar(a)
                ? Math.max(1, colIndexFor(draftOffsetForBar(a)!.end) - colIndexFor(draftOffsetForBar(a)!.start) + 1)
                : clip(a)!.width) * dayWidth - 2) + 'px',
              background: projectColor(a.project_id),
              opacity: drag && drag.id === a.id ? 0.85 : 1,
            }"
            @click.stop="emit('edit-assignment', a)"
            @contextmenu.prevent.stop="onBarContextMenu($event, a)"
          >
            <div
              v-if="auth.isAdmin"
              data-resize="left"
              class="w-1.5 cursor-ew-resize bg-black/20 hover:bg-black/40 self-stretch pointer-events-auto"
              @pointerdown.stop="onPointerDown($event, a, 'resize-left')"
              @pointerup.stop="onPointerUp($event, a)"
              @click.stop
            />
            <div class="flex-1 relative px-1.5 py-0.5 truncate pointer-events-none">
              <div
                v-if="auth.isAdmin"
                data-resize="top"
                class="absolute inset-x-0 top-0 h-1.5 cursor-ns-resize bg-black/20 hover:bg-black/40 pointer-events-auto"
                @pointerdown.stop="onPointerDown($event, a, 'resize-hours')"
                @pointerup.stop="onPointerUp($event, a)"
                @click.stop
              />
              <template v-if="barVisualHeight(effectiveHours(a)) >= 24">
                <div class="font-medium truncate">{{ projectName(a.project_id) }}</div>
                <div class="text-[10px] opacity-90">{{ effectiveHours(a) }}h/d</div>
              </template>
              <template v-else>
                <div class="truncate text-[10px] font-medium pt-0.5">
                  {{ projectName(a.project_id) }} · {{ effectiveHours(a) }}h
                </div>
              </template>
            </div>
            <div
              v-if="auth.isAdmin"
              data-resize="right"
              class="w-1.5 cursor-ew-resize bg-black/20 hover:bg-black/40 self-stretch pointer-events-auto"
              @pointerdown.stop="onPointerDown($event, a, 'resize-right')"
              @pointerup.stop="onPointerUp($event, a)"
              @click.stop
            />
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
