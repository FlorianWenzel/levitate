<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Milestone, MilestoneInput, Project } from '~/types/api'

const auth = useAuthStore()
const { call } = useApi()
const route = useRoute()

const projectID = computed(() => String(route.params.id))

const project = ref<Project | null>(null)
const milestones = ref<Milestone[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const showForm = ref(false)
const editing = ref<Milestone | null>(null)
const form = ref<MilestoneInput>(emptyForm())

function emptyForm(): MilestoneInput {
  return { name: '', date: '', end_date: '' }
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const [p, m] = await Promise.all([
      call<Project>(`/api/projects/${projectID.value}`),
      call<Milestone[]>(`/api/projects/${projectID.value}/milestones`),
    ])
    project.value = p
    milestones.value = m
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

function openEdit(m: Milestone) {
  editing.value = m
  form.value = {
    name: m.name,
    date: m.date,
    end_date: m.end_date ?? '',
  }
  showForm.value = true
}

async function submit() {
  try {
    const body: MilestoneInput = {
      name: form.value.name,
      date: form.value.date,
      end_date: form.value.end_date ? form.value.end_date : null,
    }
    if (editing.value) {
      await call(`/api/milestones/${editing.value.id}`, { method: 'PATCH', body })
    } else {
      await call(`/api/projects/${projectID.value}/milestones`, { method: 'POST', body })
    }
    showForm.value = false
    await load()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Save failed')
  }
}

async function remove(m: Milestone) {
  if (!confirm(`Delete milestone "${m.name}"?`)) return
  try {
    await call(`/api/milestones/${m.id}`, { method: 'DELETE' })
    await load()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Delete failed')
  }
}

onMounted(load)
</script>

<template>
  <main class="mx-auto max-w-5xl px-6 py-8">
    <div class="mb-6">
      <NuxtLink to="/projects" class="text-sm text-slate-500 hover:underline">← Back to projects</NuxtLink>
    </div>

    <div v-if="error" class="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">
      {{ error }}
    </div>

    <header v-if="project" class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-900" data-cy="project-name">{{ project.name }}</h1>
        <p class="text-sm text-slate-500">{{ project.client || 'No client' }}</p>
      </div>
      <span class="inline-block h-6 w-6 rounded" :style="{ background: project.color }"></span>
    </header>

    <section class="mt-8" data-cy="milestones-section">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-lg font-semibold text-slate-900">Milestones</h2>
          <p class="text-sm text-slate-500">Key delivery dates for this project.</p>
        </div>
        <button
          v-if="auth.isAdmin"
          data-cy="milestone-create"
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="openCreate"
        >
          + New milestone
        </button>
      </div>

      <div class="mt-4 overflow-hidden rounded border border-slate-200 bg-white">
        <table class="w-full text-sm">
          <thead class="bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
            <tr>
              <th class="px-4 py-2">Name</th>
              <th class="px-4 py-2">Date</th>
              <th class="px-4 py-2">End date</th>
              <th class="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="loading">
              <td colspan="4" class="px-4 py-6 text-center text-slate-400">Loading…</td>
            </tr>
            <tr v-else-if="!milestones.length">
              <td colspan="4" class="px-4 py-6 text-center text-slate-400" data-cy="milestones-empty">
                No milestones yet.
              </td>
            </tr>
            <tr
              v-for="m in milestones"
              :key="m.id"
              class="hover:bg-slate-50"
              :data-cy="`milestone-row-${m.name}`"
            >
              <td class="px-4 py-2 font-medium text-slate-900" data-cy="milestone-name">{{ m.name }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="milestone-date">{{ m.date }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="milestone-end-date">{{ m.end_date || '—' }}</td>
              <td class="px-4 py-2 text-right">
                <div v-if="auth.isAdmin" class="flex justify-end gap-2 text-xs">
                  <button class="text-slate-600 hover:underline" :data-cy="`milestone-edit-${m.name}`" @click="openEdit(m)">Edit</button>
                  <button class="text-red-600 hover:underline" :data-cy="`milestone-delete-${m.name}`" @click="remove(m)">Delete</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div
      v-if="showForm"
      class="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 p-4"
      @click.self="showForm = false"
    >
      <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl" data-cy="milestone-form">
        <h2 class="text-lg font-semibold text-slate-900">
          {{ editing ? 'Edit milestone' : 'New milestone' }}
        </h2>
        <form class="mt-4 space-y-3 text-sm" @submit.prevent="submit">
          <div>
            <label class="block text-xs font-medium text-slate-600">Name</label>
            <input
              v-model="form.name"
              data-cy="milestone-name-input"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              required
            >
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Date</label>
            <input
              v-model="form.date"
              type="date"
              data-cy="milestone-date-input"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              required
            >
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">End date (optional)</label>
            <input
              v-model="form.end_date"
              type="date"
              data-cy="milestone-end-date-input"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
            >
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="rounded px-3 py-1.5 text-slate-600 hover:bg-slate-100" @click="showForm = false">Cancel</button>
            <button
              type="submit"
              data-cy="milestone-save"
              class="rounded bg-slate-900 px-3 py-1.5 text-white hover:bg-slate-700"
            >
              Save
            </button>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>
