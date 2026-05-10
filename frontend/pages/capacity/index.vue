<script setup lang="ts">
import type { UtilizationCell } from '~/types/api'
import { addDays, parseYMD, startOfWeekMonday, ymd, isoWeek } from '~/utils/dates'

const { call } = useApi()

const numWeeks = ref(8)
const rangeStart = ref<Date>(startOfWeekMonday(new Date()))
const rangeEnd = computed(() => addDays(rangeStart.value, numWeeks.value * 7 - 1))

const cells = ref<UtilizationCell[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    cells.value = await call<UtilizationCell[]>(
      `/api/reports/utilization?from=${ymd(rangeStart.value)}&to=${ymd(rangeEnd.value)}`,
    )
  } catch (e: any) {
    error.value = e?.data?.detail ?? e?.message ?? 'Failed to load'
  } finally {
    loading.value = false
  }
}

const weeks = computed(() => {
  const seen = new Set<string>()
  const out: { weekStart: string; label: string; weekNo: number }[] = []
  for (const c of cells.value) {
    if (seen.has(c.week_start)) continue
    seen.add(c.week_start)
    const d = parseYMD(c.week_start)
    out.push({
      weekStart: c.week_start,
      label: d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
      weekNo: isoWeek(d),
    })
  }
  return out.sort((a, b) => (a.weekStart < b.weekStart ? -1 : 1))
})

const peopleRows = computed(() => {
  const map = new Map<string, { id: string; name: string; capacity: number; cells: Map<string, UtilizationCell> }>()
  for (const c of cells.value) {
    if (!map.has(c.person_id)) {
      map.set(c.person_id, {
        id: c.person_id,
        name: c.person_name,
        capacity: c.weekly_capacity_hours,
        cells: new Map(),
      })
    }
    map.get(c.person_id)!.cells.set(c.week_start, c)
  }
  return [...map.values()].sort((a, b) => a.name.localeCompare(b.name))
})

function bgFor(c: UtilizationCell | undefined): string {
  if (!c) return ''
  if (c.overallocated) return 'bg-red-200 text-red-900'
  if (c.assigned_hours === 0 && c.time_off_hours === 0) return 'bg-slate-50 text-slate-400'
  if (c.utilization_pct >= 90) return 'bg-emerald-300 text-emerald-900'
  if (c.utilization_pct >= 60) return 'bg-emerald-200 text-emerald-900'
  if (c.utilization_pct >= 30) return 'bg-emerald-100 text-emerald-900'
  return 'bg-slate-100 text-slate-700'
}

function shift(weeks: number) {
  rangeStart.value = addDays(rangeStart.value, weeks * 7)
}

function today() {
  rangeStart.value = startOfWeekMonday(new Date())
}

watch([rangeStart, numWeeks], load)
onMounted(load)
</script>

<template>
  <main class="mx-auto max-w-[1400px] px-6 py-6">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-900">Capacity</h1>
        <p class="text-sm text-slate-500">Per-person utilization by ISO week. Red = overallocated.</p>
      </div>
      <div class="flex items-center gap-2">
        <select v-model.number="numWeeks" class="rounded border border-slate-300 px-2 py-1 text-sm">
          <option :value="4">4 weeks</option>
          <option :value="8">8 weeks</option>
          <option :value="12">12 weeks</option>
        </select>
        <button class="rounded border border-slate-300 px-2 py-1 text-sm hover:bg-slate-100" @click="shift(-numWeeks)">«</button>
        <button class="rounded border border-slate-300 px-2 py-1 text-sm hover:bg-slate-100" @click="shift(-1)">‹</button>
        <button class="rounded border border-slate-300 px-2 py-1 text-sm hover:bg-slate-100" @click="today">Today</button>
        <button class="rounded border border-slate-300 px-2 py-1 text-sm hover:bg-slate-100" @click="shift(1)">›</button>
        <button class="rounded border border-slate-300 px-2 py-1 text-sm hover:bg-slate-100" @click="shift(numWeeks)">»</button>
      </div>
    </div>

    <div v-if="error" class="mb-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>

    <div class="overflow-x-auto rounded border border-slate-200 bg-white">
      <table class="w-full text-xs">
        <thead class="bg-slate-50 text-left">
          <tr>
            <th class="sticky left-0 z-10 w-44 bg-slate-50 px-3 py-2 font-medium text-slate-500">Person</th>
            <th class="w-20 px-3 py-2 text-right font-medium text-slate-500">h/wk</th>
            <th
              v-for="w in weeks"
              :key="w.weekStart"
              class="px-2 py-2 text-center font-medium text-slate-500"
            >
              <div>W{{ w.weekNo }}</div>
              <div class="text-[10px] text-slate-400">{{ w.label }}</div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-if="loading">
            <td :colspan="2 + weeks.length" class="px-4 py-6 text-center text-slate-400">Loading…</td>
          </tr>
          <tr v-else-if="!peopleRows.length">
            <td :colspan="2 + weeks.length" class="px-4 py-6 text-center text-slate-400">No data.</td>
          </tr>
          <tr v-for="row in peopleRows" :key="row.id" class="hover:bg-slate-50">
            <td class="sticky left-0 z-10 bg-white px-3 py-2 font-medium text-slate-900">{{ row.name }}</td>
            <td class="px-3 py-2 text-right tabular-nums text-slate-500">{{ row.capacity }}</td>
            <td
              v-for="w in weeks"
              :key="w.weekStart"
              class="px-2 py-1 text-center tabular-nums"
              :class="bgFor(row.cells.get(w.weekStart))"
            >
              <template v-if="row.cells.get(w.weekStart)">
                <div class="font-medium">
                  {{ Math.round(row.cells.get(w.weekStart)!.utilization_pct) }}%
                </div>
                <div class="text-[10px] opacity-80">
                  {{ row.cells.get(w.weekStart)!.assigned_hours }}h
                  <span v-if="row.cells.get(w.weekStart)!.time_off_hours > 0" class="text-amber-700">
                    · -{{ row.cells.get(w.weekStart)!.time_off_hours }}h off
                  </span>
                </div>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex items-center gap-3 text-xs text-slate-500">
      <span class="flex items-center gap-1.5"><span class="h-3 w-3 rounded bg-slate-100"></span> &lt; 30%</span>
      <span class="flex items-center gap-1.5"><span class="h-3 w-3 rounded bg-emerald-100"></span> 30–60%</span>
      <span class="flex items-center gap-1.5"><span class="h-3 w-3 rounded bg-emerald-200"></span> 60–90%</span>
      <span class="flex items-center gap-1.5"><span class="h-3 w-3 rounded bg-emerald-300"></span> 90–100%</span>
      <span class="flex items-center gap-1.5"><span class="h-3 w-3 rounded bg-red-200"></span> Over capacity</span>
    </div>
  </main>
</template>
