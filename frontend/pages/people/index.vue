<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Person, PersonInput } from '~/types/api'

const auth = useAuthStore()
const { call } = useApi()

const people = ref<Person[]>([])
const includeArchived = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)

const showForm = ref(false)
const editing = ref<Person | null>(null)
const form = ref<PersonInput>(emptyForm())

function emptyForm(): PersonInput {
  return { name: '', email: '', role: '', weekly_capacity_hours: 40 }
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const qs = includeArchived.value ? '?include_archived=true' : ''
    people.value = await call<Person[]>(`/api/people${qs}`)
  } catch (e: any) {
    error.value = e?.data?.detail ?? e?.message ?? 'Failed to load'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.value = emptyForm()
  showForm.value = true
}

function openEdit(p: Person) {
  editing.value = p
  form.value = {
    name: p.name,
    email: p.email,
    role: p.role,
    weekly_capacity_hours: Number(p.weekly_capacity_hours),
  }
  showForm.value = true
}

async function submit() {
  try {
    if (editing.value) {
      await call(`/api/people/${editing.value.id}`, {
        method: 'PATCH',
        body: form.value,
      })
    } else {
      await call('/api/people', {
        method: 'POST',
        body: form.value,
      })
    }
    showForm.value = false
    await load()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Save failed')
  }
}

async function archive(p: Person) {
  if (!confirm(`Archive ${p.name}?`)) return
  await call(`/api/people/${p.id}/archive`, { method: 'POST' })
  await load()
}

async function unarchive(p: Person) {
  await call(`/api/people/${p.id}/unarchive`, { method: 'POST' })
  await load()
}

watch(includeArchived, load)
onMounted(load)
</script>

<template>
  <main class="mx-auto max-w-6xl px-6 py-8">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-900">People</h1>
        <p class="text-sm text-slate-500">Schedulable humans — employees, contractors, anyone who shows up on the grid.</p>
      </div>
      <div class="flex items-center gap-3">
        <label class="flex items-center gap-2 text-sm text-slate-600">
          <input v-model="includeArchived" type="checkbox" class="rounded">
          Show archived
        </label>
        <button
          v-if="auth.isAdmin"
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="openCreate"
        >
          + New person
        </button>
      </div>
    </div>

    <div v-if="error" class="mt-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">
      {{ error }}
    </div>

    <div class="mt-6 overflow-hidden rounded border border-slate-200 bg-white">
      <table class="w-full text-sm">
        <thead class="bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
          <tr>
            <th class="px-4 py-2">Name</th>
            <th class="px-4 py-2">Email</th>
            <th class="px-4 py-2">Role</th>
            <th class="px-4 py-2 text-right">Capacity (h/wk)</th>
            <th class="px-4 py-2">Status</th>
            <th class="px-4 py-2"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-if="loading">
            <td colspan="6" class="px-4 py-6 text-center text-slate-400">Loading…</td>
          </tr>
          <tr v-else-if="!people.length">
            <td colspan="6" class="px-4 py-6 text-center text-slate-400">No people yet.</td>
          </tr>
          <tr v-for="p in people" :key="p.id" class="hover:bg-slate-50">
            <td class="px-4 py-2 font-medium text-slate-900">{{ p.name }}</td>
            <td class="px-4 py-2 text-slate-600">{{ p.email }}</td>
            <td class="px-4 py-2 text-slate-600">{{ p.role }}</td>
            <td class="px-4 py-2 text-right tabular-nums">{{ p.weekly_capacity_hours }}</td>
            <td class="px-4 py-2">
              <span
                v-if="p.archived_at"
                class="rounded bg-slate-200 px-1.5 py-0.5 text-xs text-slate-700"
              >Archived</span>
              <span v-else class="rounded bg-emerald-100 px-1.5 py-0.5 text-xs text-emerald-700">Active</span>
            </td>
            <td class="px-4 py-2 text-right">
              <div v-if="auth.isAdmin" class="flex justify-end gap-2 text-xs">
                <button class="text-slate-600 hover:underline" @click="openEdit(p)">Edit</button>
                <button v-if="!p.archived_at" class="text-red-600 hover:underline" @click="archive(p)">Archive</button>
                <button v-else class="text-emerald-700 hover:underline" @click="unarchive(p)">Restore</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal form -->
    <div v-if="showForm" class="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 p-4" @click.self="showForm = false">
      <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h2 class="text-lg font-semibold text-slate-900">
          {{ editing ? 'Edit person' : 'New person' }}
        </h2>
        <form class="mt-4 space-y-3 text-sm" @submit.prevent="submit">
          <div>
            <label class="block text-xs font-medium text-slate-600">Name</label>
            <input v-model="form.name" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Email</label>
            <input v-model="form.email" type="email" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Role</label>
            <input v-model="form.role" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Weekly capacity (hours)</label>
            <input v-model.number="form.weekly_capacity_hours" type="number" min="0" max="168" step="0.5" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="rounded px-3 py-1.5 text-slate-600 hover:bg-slate-100" @click="showForm = false">Cancel</button>
            <button type="submit" class="rounded bg-slate-900 px-3 py-1.5 text-white hover:bg-slate-700">Save</button>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>
