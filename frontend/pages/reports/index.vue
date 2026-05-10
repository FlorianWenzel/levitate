<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { addDays, startOfWeekMonday, ymd } from '~/utils/dates'

const auth = useAuthStore()
const config = useRuntimeConfig()

const fromDate = ref(ymd(startOfWeekMonday(new Date())))
const toDate = ref(ymd(addDays(startOfWeekMonday(new Date()), 27)))

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
</script>

<template>
  <main class="mx-auto max-w-3xl px-6 py-8">
    <h1 class="text-2xl font-semibold text-slate-900">Reports</h1>
    <p class="mt-1 text-sm text-slate-500">Export schedule and utilization data as CSV.</p>

    <section class="mt-6 rounded border border-slate-200 bg-white p-5">
      <h2 class="text-sm font-medium text-slate-900">Date range</h2>
      <div class="mt-3 grid grid-cols-2 gap-3 text-sm">
        <div>
          <label class="block text-xs font-medium text-slate-600">From</label>
          <input v-model="fromDate" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
        </div>
        <div>
          <label class="block text-xs font-medium text-slate-600">To</label>
          <input v-model="toDate" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
        </div>
      </div>
    </section>

    <section class="mt-6 space-y-3">
      <div class="flex items-center justify-between rounded border border-slate-200 bg-white p-4">
        <div>
          <h3 class="text-sm font-medium text-slate-900">Utilization</h3>
          <p class="text-xs text-slate-500">Per-person, per-week assigned vs. available hours.</p>
        </div>
        <button
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="downloadCSV('/api/reports/utilization.csv', 'utilization.csv')"
        >
          Download CSV
        </button>
      </div>
      <div class="flex items-center justify-between rounded border border-slate-200 bg-white p-4">
        <div>
          <h3 class="text-sm font-medium text-slate-900">Assignments</h3>
          <p class="text-xs text-slate-500">All assignments overlapping the selected range.</p>
        </div>
        <button
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="downloadCSV('/api/reports/assignments.csv', 'assignments.csv')"
        >
          Download CSV
        </button>
      </div>
    </section>
  </main>
</template>
