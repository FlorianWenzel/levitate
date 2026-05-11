<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { UtilizationCell } from '~/types/api'
import { addDays, isoWeek, parseYMD, startOfWeekMonday, ymd } from '~/utils/dates'

type GroupBy = 'week' | 'person'
type ChartType = 'bar' | 'line'
type MetricKey = 'utilization_pct' | 'assigned_hours' | 'available_hours' | 'time_off_hours'
type SortBy = 'value' | 'label'

type ReportRow = {
  id: string
  label: string
  sublabel: string
  assigned: number
  available: number
  timeOff: number
  utilization: number
  overallocated: boolean
  value: number
}

const auth = useAuthStore()
const config = useRuntimeConfig()
const { call } = useApi()

const fromDate = ref(ymd(startOfWeekMonday(new Date())))
const toDate = ref(ymd(addDays(startOfWeekMonday(new Date()), 27)))
const groupBy = ref<GroupBy>('week')
const chartType = ref<ChartType>('bar')
const metricKey = ref<MetricKey>('utilization_pct')
const sortBy = ref<SortBy>('value')
const personLimit = ref(8)

const cells = ref<UtilizationCell[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const metricOptions: { key: MetricKey; label: string; suffix: string }[] = [
  { key: 'utilization_pct', label: 'Utilization %', suffix: '%' },
  { key: 'assigned_hours', label: 'Assigned hours', suffix: 'h' },
  { key: 'available_hours', label: 'Available hours', suffix: 'h' },
  { key: 'time_off_hours', label: 'Time off', suffix: 'h' },
]

const selectedMetric = computed(() => metricOptions.find((m) => m.key === metricKey.value) ?? metricOptions[0])

async function loadReport() {
  loading.value = true
  error.value = null
  try {
    cells.value = await call<UtilizationCell[]>(`/api/reports/utilization?from=${fromDate.value}&to=${toDate.value}`)
  } catch (e: any) {
    error.value = e?.data?.detail ?? e?.message ?? 'Failed to load report'
  } finally {
    loading.value = false
  }
}

async function downloadCSV(path: string, filename: string) {
  const url = `${config.public.apiBase}${path}?from=${fromDate.value}&to=${toDate.value}`
  // $fetch streams text; we manually trigger a download via blob.
  const res = await fetch(url, {
    headers: auth.accessToken ? { Authorization: `Bearer ${auth.accessToken}` } : {},
  })
  if (!res.ok) {
    alert(`Download failed: ${res.status}`)
    return
  }
  const blob = await res.blob()
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(a.href)
}

const rows = computed<ReportRow[]>(() => {
  if (groupBy.value === 'week') return weeklyRows()
  return personRows()
})

const sortedRows = computed(() => {
  if (chartType.value === 'line' && groupBy.value === 'week') return rows.value
  const out = [...rows.value]
  if (sortBy.value === 'label') out.sort((a, b) => a.label.localeCompare(b.label))
  else out.sort((a, b) => b.value - a.value || a.label.localeCompare(b.label))
  if (groupBy.value === 'person') return out.slice(0, personLimit.value)
  return out
})

const maxValue = computed(() => {
  const max = sortedRows.value.reduce((m, r) => Math.max(m, r.value), 0)
  return metricKey.value === 'utilization_pct' ? Math.max(100, max) : Math.max(1, max)
})

const chartTitle = computed(() => `${selectedMetric.value.label} by ${groupBy.value}`)

const linePoints = computed(() => {
  if (sortedRows.value.length === 0) return ''
  const width = 760
  const height = 220
  const pad = 18
  return sortedRows.value.map((row, index) => {
    const x = sortedRows.value.length === 1
      ? width / 2
      : pad + (index * (width - pad * 2)) / (sortedRows.value.length - 1)
    const y = height - pad - (row.value / maxValue.value) * (height - pad * 2)
    return `${round1(x)},${round1(y)}`
  }).join(' ')
})

function weeklyRows(): ReportRow[] {
  const byWeek = new Map<string, { assigned: number; available: number; timeOff: number; overallocated: boolean }>()
  for (const c of cells.value) {
    const cur = byWeek.get(c.week_start) ?? { assigned: 0, available: 0, timeOff: 0, overallocated: false }
    cur.assigned += c.assigned_hours
    cur.available += c.available_hours
    cur.timeOff += c.time_off_hours
    cur.overallocated ||= c.overallocated
    byWeek.set(c.week_start, cur)
  }
  return [...byWeek.entries()]
    .sort((a, b) => (a[0] < b[0] ? -1 : 1))
    .map(([weekStart, sums]) => {
      const date = parseYMD(weekStart)
      const utilization = sums.available > 0 ? (sums.assigned / sums.available) * 100 : 0
      return withMetric({
        id: weekStart,
        label: `W${isoWeek(date)}`,
        sublabel: date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
        assigned: sums.assigned,
        available: sums.available,
        timeOff: sums.timeOff,
        utilization,
        overallocated: sums.overallocated,
        value: 0,
      })
    })
}

function personRows(): ReportRow[] {
  const byPerson = new Map<string, { name: string; assigned: number; available: number; timeOff: number; overallocated: boolean }>()
  for (const c of cells.value) {
    const cur = byPerson.get(c.person_id) ?? {
      name: c.person_name,
      assigned: 0,
      available: 0,
      timeOff: 0,
      overallocated: false,
    }
    cur.assigned += c.assigned_hours
    cur.available += c.available_hours
    cur.timeOff += c.time_off_hours
    cur.overallocated ||= c.overallocated
    byPerson.set(c.person_id, cur)
  }
  return [...byPerson.entries()].map(([personId, sums]) => withMetric({
    id: personId,
    label: sums.name,
    sublabel: `${round1(sums.assigned)}h assigned / ${round1(sums.available)}h available`,
    assigned: sums.assigned,
    available: sums.available,
    timeOff: sums.timeOff,
    utilization: sums.available > 0 ? (sums.assigned / sums.available) * 100 : 0,
    overallocated: sums.overallocated,
    value: 0,
  }))
}

function withMetric(row: ReportRow): ReportRow {
  const valueByMetric: Record<MetricKey, number> = {
    utilization_pct: row.utilization,
    assigned_hours: row.assigned,
    available_hours: row.available,
    time_off_hours: row.timeOff,
  }
  return { ...row, value: round1(valueByMetric[metricKey.value]) }
}

function formatMetric(value: number) {
  return `${Math.round(value * 10) / 10}${selectedMetric.value.suffix}`
}

function barWidth(value: number) {
  return `${Math.min(100, (value / maxValue.value) * 100)}%`
}

function round1(n: number) {
  return Math.round(n * 10) / 10
}

watch([fromDate, toDate], loadReport)
watch(groupBy, () => {
  if (groupBy.value === 'person' && chartType.value === 'line') chartType.value = 'bar'
})
onMounted(loadReport)
</script>

<template>
  <main class="mx-auto max-w-6xl px-6 py-8">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-900">Reports</h1>
        <p class="mt-1 text-sm text-slate-500">Explore utilization data visually, then export CSV when needed.</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="downloadCSV('/api/reports/utilization.csv', 'utilization.csv')"
        >
          Download utilization CSV
        </button>
        <button
          class="rounded border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-100"
          @click="downloadCSV('/api/reports/assignments.csv', 'assignments.csv')"
        >
          Download assignments CSV
        </button>
      </div>
    </div>

    <section class="mt-6 rounded border border-slate-200 bg-white p-5">
      <h2 class="text-sm font-medium text-slate-900">Report settings</h2>
      <div class="mt-3 grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-6">
        <label class="block">
          <span class="block text-xs font-medium text-slate-600">From</span>
          <input v-model="fromDate" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
        </label>
        <label class="block">
          <span class="block text-xs font-medium text-slate-600">To</span>
          <input v-model="toDate" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
        </label>
        <label class="block">
          <span class="block text-xs font-medium text-slate-600">Group by</span>
          <select v-model="groupBy" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" data-report-group>
            <option value="week">Week</option>
            <option value="person">Person</option>
          </select>
        </label>
        <label class="block">
          <span class="block text-xs font-medium text-slate-600">Metric</span>
          <select v-model="metricKey" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" data-report-metric>
            <option v-for="option in metricOptions" :key="option.key" :value="option.key">{{ option.label }}</option>
          </select>
        </label>
        <label class="block">
          <span class="block text-xs font-medium text-slate-600">Chart</span>
          <select v-model="chartType" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" data-report-chart-type>
            <option value="bar">Bar</option>
            <option value="line" :disabled="groupBy === 'person'">Line trend</option>
          </select>
        </label>
        <label class="block">
          <span class="block text-xs font-medium text-slate-600">Sort</span>
          <select v-model="sortBy" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" data-report-sort>
            <option value="value">Metric value</option>
            <option value="label">Name/date</option>
          </select>
        </label>
      </div>
      <label v-if="groupBy === 'person'" class="mt-3 block max-w-xs text-sm">
        <span class="block text-xs font-medium text-slate-600">People shown</span>
        <select v-model.number="personLimit" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" data-report-limit>
          <option :value="5">Top 5</option>
          <option :value="8">Top 8</option>
          <option :value="12">Top 12</option>
          <option :value="999">All</option>
        </select>
      </label>
    </section>

    <div v-if="error" class="mt-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>

    <section class="mt-6 rounded border border-slate-200 bg-white p-5" data-report-chart>
      <div class="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 class="text-lg font-semibold text-slate-900">{{ chartTitle }}</h2>
          <p class="text-sm text-slate-500">{{ sortedRows.length }} groups in selected range.</p>
        </div>
        <div v-if="loading" class="text-sm text-slate-400">Loading…</div>
      </div>

      <div v-if="!loading && !sortedRows.length" class="mt-8 rounded bg-slate-50 p-8 text-center text-sm text-slate-500">
        No data for selected range.
      </div>

      <div v-else-if="chartType === 'line'" class="mt-5 overflow-x-auto" data-line-chart>
        <svg viewBox="0 0 760 220" class="min-w-[760px] rounded bg-slate-50" role="img" :aria-label="chartTitle">
          <line x1="18" y1="202" x2="742" y2="202" stroke="#CBD5E1" />
          <polyline :points="linePoints" fill="none" stroke="#0F172A" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />
          <circle
            v-for="(row, index) in sortedRows"
            :key="row.id"
            :cx="sortedRows.length === 1 ? 380 : 18 + (index * 724) / (sortedRows.length - 1)"
            :cy="202 - (row.value / maxValue) * 184"
            r="4"
            :fill="row.overallocated ? '#DC2626' : '#0F172A'"
          />
        </svg>
      </div>

      <div v-else class="mt-5 space-y-3" data-bar-chart>
        <div v-for="row in sortedRows" :key="row.id" class="grid gap-2 sm:grid-cols-[9rem_1fr_5rem] sm:items-center" data-report-row>
          <div>
            <div class="truncate text-sm font-medium text-slate-900">{{ row.label }}</div>
            <div class="text-xs text-slate-500">{{ row.sublabel }}</div>
          </div>
          <div class="h-8 overflow-hidden rounded bg-slate-100">
            <div
              class="h-full rounded bg-emerald-500 transition-all"
              :class="row.overallocated ? 'bg-red-500' : 'bg-emerald-500'"
              :style="{ width: barWidth(row.value) }"
            />
          </div>
          <div class="text-right text-sm font-semibold tabular-nums text-slate-900">{{ formatMetric(row.value) }}</div>
        </div>
      </div>

      <div class="mt-5 grid gap-3 border-t border-slate-100 pt-4 text-xs text-slate-500 sm:grid-cols-4">
        <div><span class="font-medium text-slate-700">Assigned:</span> sum of scheduled workday hours</div>
        <div><span class="font-medium text-slate-700">Available:</span> capacity minus time off</div>
        <div><span class="font-medium text-slate-700">Utilization:</span> assigned / available</div>
        <div><span class="font-medium text-red-700">Red:</span> at least one overallocated group</div>
      </div>
    </section>
  </main>
</template>
