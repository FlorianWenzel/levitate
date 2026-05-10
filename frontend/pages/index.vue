<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Assignment, Person, Project, TimeOff, UtilizationCell } from '~/types/api'
import { addDays, addWorkdays, isoWeek, parseYMD, startOfWeekMonday, ymd } from '~/utils/dates'

const auth = useAuthStore()
const { call } = useApi()

const monday = startOfWeekMonday(new Date())
const lookaheadWeeks = 8
const lookaheadEnd = addDays(monday, lookaheadWeeks * 7 - 1)

const people = ref<Person[]>([])
const projects = ref<Project[]>([])
const assignments = ref<Assignment[]>([])
const timeOff = ref<TimeOff[]>([])
const utilization = ref<UtilizationCell[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
  if (!auth.me) await auth.fetchMe()
  try {
    const [ps, prjs, asg, off, util] = await Promise.all([
      call<Person[]>('/api/people'),
      call<Project[]>('/api/projects'),
      call<Assignment[]>(`/api/assignments?from=${ymd(monday)}&to=${ymd(lookaheadEnd)}`),
      call<TimeOff[]>(`/api/time-off?from=${ymd(monday)}&to=${ymd(addDays(monday, 60))}`),
      call<UtilizationCell[]>(`/api/reports/utilization?from=${ymd(monday)}&to=${ymd(lookaheadEnd)}`),
    ])
    people.value = ps
    projects.value = prjs
    assignments.value = asg
    timeOff.value = off
    utilization.value = util
  } catch (e: any) {
    error.value = e?.data?.detail ?? e?.message ?? 'Failed to load dashboard'
  } finally {
    loading.value = false
  }
})

// ----- Aggregates -----

const activePeople = computed(() => people.value.filter((p) => !p.archived_at))
const activeProjects = computed(() => projects.value.filter((p) => !p.archived_at))

const thisWeekStart = ymd(monday)

const thisWeekCells = computed(() =>
  utilization.value.filter((c) => c.week_start === thisWeekStart),
)

const thisWeekHours = computed(() =>
  thisWeekCells.value.reduce((sum, c) => sum + (c.assigned_hours || 0), 0),
)

const thisWeekUtilization = computed(() => {
  const cells = thisWeekCells.value.filter((c) => c.weekly_capacity_hours > 0)
  if (cells.length === 0) return 0
  const totalAssigned = cells.reduce((s, c) => s + c.assigned_hours, 0)
  const totalAvailable = cells.reduce((s, c) => s + c.available_hours, 0)
  if (totalAvailable === 0) return 0
  return (totalAssigned / totalAvailable) * 100
})

const overallocatedThisWeek = computed(() =>
  thisWeekCells.value.filter((c) => c.overallocated).length,
)

// Per-week aggregates across the whole team for the chart.
type WeekRow = {
  weekStart: string
  weekNo: number
  label: string
  assigned: number
  capacity: number
  pct: number
  overallocated: boolean
}

const weeklyRows = computed<WeekRow[]>(() => {
  const byWeek = new Map<string, { assigned: number; available: number; capacity: number }>()
  for (const c of utilization.value) {
    const cur = byWeek.get(c.week_start) ?? { assigned: 0, available: 0, capacity: 0 }
    cur.assigned += c.assigned_hours
    cur.available += c.available_hours
    cur.capacity += c.weekly_capacity_hours
    byWeek.set(c.week_start, cur)
  }
  return [...byWeek.entries()]
    .sort((a, b) => (a[0] < b[0] ? -1 : 1))
    .map(([weekStart, sums]) => {
      const d = parseYMD(weekStart)
      const pct = sums.available > 0 ? (sums.assigned / sums.available) * 100 : 0
      return {
        weekStart,
        weekNo: isoWeek(d),
        label: d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
        assigned: round1(sums.assigned),
        capacity: round1(sums.capacity),
        pct,
        overallocated: sums.assigned > sums.available + 0.01,
      }
    })
})

// Top projects by hours over the visible 8-week window.
type ProjectRow = { id: string; name: string; color: string; hours: number }

const projectsById = computed(() => {
  const m = new Map<string, Project>()
  for (const p of projects.value) m.set(p.id, p)
  return m
})

const projectRows = computed<ProjectRow[]>(() => {
  const totals = new Map<string, number>()
  for (const a of assignments.value) {
    const start = parseYMD(a.start_date)
    const end = parseYMD(a.end_date)
    let workdays = 0
    let cur = start < monday ? new Date(monday) : new Date(start)
    while (cur <= end && cur <= lookaheadEnd) {
      const dow = cur.getDay()
      if (dow !== 0 && dow !== 6) workdays++
      cur.setDate(cur.getDate() + 1)
    }
    if (workdays <= 0) continue
    const hrs = workdays * Number(a.hours_per_day)
    totals.set(a.project_id, (totals.get(a.project_id) ?? 0) + hrs)
  }
  return [...totals.entries()]
    .map(([id, hours]) => {
      const p = projectsById.value.get(id)
      return {
        id,
        name: p?.name ?? '—',
        color: p?.color ?? '#64748B',
        hours: round1(hours),
      }
    })
    .sort((a, b) => b.hours - a.hours)
    .slice(0, 6)
})

const maxProjectHours = computed(() =>
  projectRows.value.reduce((m, r) => Math.max(m, r.hours), 1),
)

// Upcoming time-off ordered by start.
const upcomingTimeOff = computed(() => {
  const today = ymd(new Date())
  return timeOff.value
    .filter((t) => t.end_date >= today)
    .sort((a, b) => (a.start_date < b.start_date ? -1 : 1))
    .slice(0, 6)
})

const peopleById = computed(() => {
  const m = new Map<string, Person>()
  for (const p of people.value) m.set(p.id, p)
  return m
})

function personName(id: string) {
  return peopleById.value.get(id)?.name ?? '—'
}

function fmtRange(start: string, end: string) {
  const a = parseYMD(start)
  const b = parseYMD(end)
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' }
  if (start === end) return a.toLocaleDateString(undefined, opts)
  return `${a.toLocaleDateString(undefined, opts)} – ${b.toLocaleDateString(undefined, opts)}`
}

function round1(n: number) {
  return Math.round(n * 10) / 10
}
</script>

<template>
  <main class="mx-auto max-w-6xl px-6 py-8">
    <header class="flex items-end justify-between mb-6">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight text-slate-900">
          Hello, {{ auth.me?.name || auth.me?.email || 'there' }}
        </h1>
        <p class="mt-1 text-sm text-slate-500">
          A snapshot of your team — this week and the next {{ lookaheadWeeks - 1 }}.
        </p>
      </div>
      <NuxtLink
        to="/schedule"
        class="hidden sm:inline-flex rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
      >
        Open schedule →
      </NuxtLink>
    </header>

    <div v-if="error" data-dashboard-error class="mb-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">
      {{ error }}
    </div>

    <!-- KPI cards -->
    <section data-kpis class="grid grid-cols-2 gap-3 sm:grid-cols-4 mb-6">
      <div data-kpi="people" class="rounded-lg border border-slate-200 bg-white p-4">
        <div class="text-xs uppercase tracking-wide text-slate-500">People</div>
        <div class="mt-1 text-2xl font-semibold text-slate-900 tabular-nums">
          {{ activePeople.length }}
        </div>
        <div class="text-[11px] text-slate-400">active</div>
      </div>
      <div data-kpi="projects" class="rounded-lg border border-slate-200 bg-white p-4">
        <div class="text-xs uppercase tracking-wide text-slate-500">Projects</div>
        <div class="mt-1 text-2xl font-semibold text-slate-900 tabular-nums">
          {{ activeProjects.length }}
        </div>
        <div class="text-[11px] text-slate-400">active</div>
      </div>
      <div data-kpi="hours" class="rounded-lg border border-slate-200 bg-white p-4">
        <div class="text-xs uppercase tracking-wide text-slate-500">This week</div>
        <div class="mt-1 text-2xl font-semibold text-slate-900 tabular-nums">
          {{ round1(thisWeekHours) }}h
        </div>
        <div class="text-[11px] text-slate-400">assigned</div>
      </div>
      <div data-kpi="utilization" class="rounded-lg border border-slate-200 bg-white p-4">
        <div class="text-xs uppercase tracking-wide text-slate-500">Utilization</div>
        <div
          class="mt-1 text-2xl font-semibold tabular-nums"
          :class="overallocatedThisWeek > 0 ? 'text-red-600' : 'text-slate-900'"
        >
          {{ Math.round(thisWeekUtilization) }}%
        </div>
        <div class="text-[11px] text-slate-400">
          <span v-if="overallocatedThisWeek > 0">
            {{ overallocatedThisWeek }} overallocated
          </span>
          <span v-else>this week</span>
        </div>
      </div>
    </section>

    <!-- Team utilization by week -->
    <section data-utilization-chart class="rounded-lg border border-slate-200 bg-white p-5 mb-6">
      <header class="flex items-center justify-between mb-4">
        <h2 class="text-sm font-semibold text-slate-900">Team utilization · next {{ lookaheadWeeks }} weeks</h2>
        <NuxtLink to="/capacity" class="text-xs text-slate-500 hover:text-slate-700">View full capacity →</NuxtLink>
      </header>
      <div v-if="loading" class="text-sm text-slate-400">Loading…</div>
      <div v-else-if="!weeklyRows.length" class="text-sm text-slate-400">No data yet — add people and assignments to see this fill in.</div>
      <ul v-else class="space-y-2">
        <li
          v-for="w in weeklyRows"
          :key="w.weekStart"
          :data-week="w.weekStart"
          class="grid grid-cols-[80px_1fr_60px] items-center gap-3 text-xs"
        >
          <div class="text-slate-500">
            <span class="font-medium text-slate-700">W{{ w.weekNo }}</span>
            <span class="ml-1 text-slate-400">{{ w.label }}</span>
          </div>
          <div class="relative h-6 rounded bg-slate-100 overflow-hidden">
            <div
              class="absolute inset-y-0 left-0 rounded"
              :class="w.overallocated ? 'bg-red-300' : (w.pct >= 90 ? 'bg-emerald-400' : 'bg-emerald-200')"
              :style="{ width: Math.min(120, w.pct) + '%' }"
            />
            <div class="absolute inset-0 flex items-center px-2 text-[11px] font-medium text-slate-700">
              {{ w.assigned }}h / {{ w.capacity }}h
            </div>
          </div>
          <div
            class="text-right font-medium tabular-nums"
            :class="w.overallocated ? 'text-red-600' : 'text-slate-700'"
          >
            {{ Math.round(w.pct) }}%
          </div>
        </li>
      </ul>
    </section>

    <!-- Two-column: project hours + upcoming time-off -->
    <section class="grid gap-4 md:grid-cols-2">
      <div data-project-chart class="rounded-lg border border-slate-200 bg-white p-5">
        <header class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-slate-900">Hours by project · next {{ lookaheadWeeks }} weeks</h2>
          <NuxtLink to="/projects" class="text-xs text-slate-500 hover:text-slate-700">All projects →</NuxtLink>
        </header>
        <div v-if="loading" class="text-sm text-slate-400">Loading…</div>
        <div v-else-if="!projectRows.length" class="text-sm text-slate-400">No assigned hours yet.</div>
        <ul v-else class="space-y-2">
          <li
            v-for="p in projectRows"
            :key="p.id"
            :data-project="p.id"
            class="grid grid-cols-[140px_1fr_50px] items-center gap-3 text-xs"
          >
            <div class="flex items-center gap-2 min-w-0">
              <span class="h-3 w-3 shrink-0 rounded" :style="{ background: p.color }" />
              <span class="truncate text-slate-700" :title="p.name">{{ p.name }}</span>
            </div>
            <div class="relative h-3 rounded bg-slate-100 overflow-hidden">
              <div
                class="absolute inset-y-0 left-0 rounded"
                :style="{ width: (p.hours / maxProjectHours * 100) + '%', background: p.color }"
              />
            </div>
            <div class="text-right tabular-nums text-slate-700">{{ p.hours }}h</div>
          </li>
        </ul>
      </div>

      <div data-time-off-list class="rounded-lg border border-slate-200 bg-white p-5">
        <header class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-slate-900">Upcoming time off</h2>
          <NuxtLink to="/schedule" class="text-xs text-slate-500 hover:text-slate-700">Schedule →</NuxtLink>
        </header>
        <div v-if="loading" class="text-sm text-slate-400">Loading…</div>
        <div v-else-if="!upcomingTimeOff.length" class="text-sm text-slate-400">Nobody is out in the next 60 days.</div>
        <ul v-else class="divide-y divide-slate-100">
          <li
            v-for="t in upcomingTimeOff"
            :key="t.id"
            :data-time-off-id="t.id"
            class="flex items-center justify-between py-2 text-sm"
          >
            <div class="min-w-0">
              <div class="font-medium text-slate-900 truncate">{{ personName(t.person_id) }}</div>
              <div class="text-[11px] uppercase tracking-wide text-slate-400">{{ t.type }}</div>
            </div>
            <div class="text-xs text-slate-500 shrink-0 ml-3">{{ fmtRange(t.start_date, t.end_date) }}</div>
          </li>
        </ul>
      </div>
    </section>
  </main>
</template>
