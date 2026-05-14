<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { FloatImportInput, FloatImportResult } from '~/types/api'

const auth = useAuthStore()
const { call } = useApi()

const loading = ref(false)
const error = ref<string | null>(null)
const result = ref<FloatImportResult | null>(null)

const form = ref<FloatImportInput>({
  api_token: '',
  base_url: 'https://api.float.com/v3',
  start_date: dateYearsFromNow(-1),
  end_date: dateYearsFromNow(1),
})

function dateYearsFromNow(years: number) {
  const d = new Date()
  d.setFullYear(d.getFullYear() + years)
  return d.toISOString().slice(0, 10)
}

async function submit() {
  if (!auth.isAdmin) return
  loading.value = true
  error.value = null
  result.value = null
  try {
    result.value = await call<FloatImportResult>('/api/import/float', {
      method: 'POST',
      body: form.value,
    })
    form.value.api_token = ''
  } catch (e: any) {
    error.value = e?.data?.detail ?? e?.message ?? 'Import failed'
  } finally {
    loading.value = false
  }
}

const totalCreated = computed(() => {
  if (!result.value) return 0
  return result.value.people_created
    + result.value.projects_created
    + result.value.assignments_created
    + result.value.time_off_created
    + result.value.milestones_created
})

const rows = computed(() => {
  if (!result.value) return []
  return [
    { label: 'People', created: result.value.people_created, skipped: result.value.people_skipped },
    { label: 'Projects', created: result.value.projects_created, skipped: result.value.projects_skipped },
    { label: 'Assignments', created: result.value.assignments_created, skipped: result.value.assignments_skipped },
    { label: 'Time off', created: result.value.time_off_created, skipped: result.value.time_off_skipped },
    { label: 'Milestones', created: result.value.milestones_created, skipped: result.value.milestones_skipped },
  ]
})
</script>

<template>
  <main class="mx-auto max-w-4xl px-6 py-8">
    <header>
      <p class="text-xs font-semibold uppercase tracking-wide text-sky-600">Float migration</p>
      <h1 class="mt-1 text-2xl font-semibold text-slate-900">Import from Float</h1>
      <p class="mt-2 max-w-2xl text-sm text-slate-500">
        Bring over people, projects, allocations, and time off with a Float API token. Credentials are used only for this import request.
      </p>
    </header>

    <div v-if="!auth.isAdmin" class="mt-6 rounded border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
      Only admins can import data from Float.
    </div>

    <form v-else class="mt-6 rounded-lg border border-slate-200 bg-white p-5 shadow-sm" @submit.prevent="submit">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="sm:col-span-2">
          <label class="block text-xs font-medium text-slate-600">Float API token</label>
          <input
            v-model="form.api_token"
            type="password"
            autocomplete="off"
            class="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm"
            placeholder="Paste your Float bearer token"
            required
          >
          <p class="mt-1 text-xs text-slate-400">
            Create this in Float under Account Settings → Integrations → API.
          </p>
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-600">Schedule start</label>
          <input v-model="form.start_date" type="date" class="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm" required>
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-600">Schedule end</label>
          <input v-model="form.end_date" type="date" class="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm" required>
        </div>

        <div class="sm:col-span-2">
          <label class="block text-xs font-medium text-slate-600">API base URL</label>
          <input v-model="form.base_url" class="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm font-mono">
        </div>
      </div>

      <div v-if="error" class="mt-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">
        {{ error }}
      </div>

      <div class="mt-5 flex items-center justify-between gap-3">
        <p class="text-xs text-slate-500">
          Repeat imports skip records that already match existing people, projects, assignments, and time off.
        </p>
        <button
          type="submit"
          class="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:cursor-not-allowed disabled:opacity-60"
          :disabled="loading"
        >
          {{ loading ? 'Importing…' : 'Import Float data' }}
        </button>
      </div>
    </form>

    <section v-if="result" class="mt-6 rounded-lg border border-emerald-200 bg-emerald-50 p-5">
      <h2 class="text-lg font-semibold text-emerald-950">
        Imported {{ totalCreated }} records
      </h2>
      <div class="mt-4 overflow-hidden rounded border border-emerald-200 bg-white">
        <table class="w-full text-sm">
          <thead class="bg-emerald-50 text-left text-xs uppercase tracking-wide text-emerald-700">
            <tr>
              <th class="px-4 py-2">Data</th>
              <th class="px-4 py-2 text-right">Created</th>
              <th class="px-4 py-2 text-right">Skipped</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-emerald-100">
            <tr v-for="row in rows" :key="row.label">
              <td class="px-4 py-2 font-medium text-slate-900">{{ row.label }}</td>
              <td class="px-4 py-2 text-right tabular-nums text-slate-700">{{ row.created }}</td>
              <td class="px-4 py-2 text-right tabular-nums text-slate-500">{{ row.skipped }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="mt-4 flex gap-3">
        <NuxtLink to="/people" class="rounded bg-emerald-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-emerald-800">Review people</NuxtLink>
        <NuxtLink to="/schedule" class="rounded bg-white px-3 py-1.5 text-sm font-medium text-emerald-800 ring-1 ring-emerald-200 hover:bg-emerald-100">Open schedule</NuxtLink>
      </div>
      <div v-if="result.warnings?.length" class="mt-4 rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
        <p v-for="warning in result.warnings" :key="warning">{{ warning }}</p>
      </div>
    </section>
  </main>
</template>
