<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Milestone, MilestoneInput, Person, Phase, PhaseInput, Project } from '~/types/api'

const auth = useAuthStore()
const { call } = useApi()
const route = useRoute()

const projectID = computed(() => String(route.params.id))

const project = ref<Project | null>(null)
const milestones = ref<Milestone[]>([])
const phases = ref<Phase[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const showForm = ref(false)
const editing = ref<Milestone | null>(null)
const form = ref<MilestoneInput>(emptyForm())

const showPhaseForm = ref(false)
const editingPhase = ref<Phase | null>(null)
const phaseForm = ref<PhaseInput>(emptyPhaseForm())

function emptyForm(): MilestoneInput {
  return { name: '', date: '', end_date: '' }
}

function emptyPhaseForm(): PhaseInput {
  return {
    name: '',
    color: '#0EA5E9',
    notes: '',
    start_date: '',
    end_date: '',
    budget_total: 0,
    default_hourly_rate: 0,
    non_billable: false,
    status: 2,
  }
}

const people = ref<Person[]>([])

const personById = computed(() => {
  const map = new Map<string, Person>()
  for (const p of people.value) map.set(p.id, p)
  return map
})

function personName(id: string): string {
  return personById.value.get(id)?.name ?? id
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const expand = 'expenses,project_tasks,project_team'
    const [p, m, ph, ppl] = await Promise.all([
      call<Project>(`/api/projects/${projectID.value}?expand=${expand}`),
      call<Milestone[]>(`/api/projects/${projectID.value}/milestones`),
      call<Phase[]>(`/api/projects/${projectID.value}/phases`),
      call<Person[]>(`/api/people`),
    ])
    project.value = p
    milestones.value = m
    phases.value = ph ?? []
    people.value = ppl ?? []
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

function openCreatePhase() {
  editingPhase.value = null
  phaseForm.value = emptyPhaseForm()
  showPhaseForm.value = true
}

function openEditPhase(p: Phase) {
  editingPhase.value = p
  phaseForm.value = {
    name: p.name,
    color: p.color,
    notes: p.notes,
    start_date: p.start_date ?? '',
    end_date: p.end_date ?? '',
    budget_total: p.budget_total,
    default_hourly_rate: p.default_hourly_rate,
    non_billable: p.non_billable,
    status: p.status,
  }
  showPhaseForm.value = true
}

async function submitPhase() {
  try {
    const body: PhaseInput = {
      name: phaseForm.value.name,
      color: phaseForm.value.color,
      notes: phaseForm.value.notes,
      start_date: phaseForm.value.start_date ? phaseForm.value.start_date : null,
      end_date: phaseForm.value.end_date ? phaseForm.value.end_date : null,
      budget_total: Number(phaseForm.value.budget_total ?? 0),
      default_hourly_rate: Number(phaseForm.value.default_hourly_rate ?? 0),
      non_billable: !!phaseForm.value.non_billable,
      status: phaseForm.value.status ?? 2,
    }
    if (editingPhase.value) {
      await call(`/api/phases/${editingPhase.value.id}`, { method: 'PATCH', body })
    } else {
      await call(`/api/projects/${projectID.value}/phases`, { method: 'POST', body })
    }
    showPhaseForm.value = false
    await load()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Save failed')
  }
}

async function removePhase(p: Phase) {
  if (!confirm(`Delete phase "${p.name}"?`)) return
  try {
    await call(`/api/phases/${p.id}`, { method: 'DELETE' })
    await load()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Delete failed')
  }
}

function phaseStatusLabel(s: 0 | 1 | 2): string {
  return s === 0 ? 'Draft' : s === 1 ? 'Tentative' : 'Confirmed'
}

function projectBudgetTypeLabel(t: Project['budget_type']): string {
  return t === 1 ? 'Total hours' : t === 2 ? 'Total fee' : t === 3 ? 'Hourly fee' : '—'
}

function projectBudgetPriorityLabel(p: Project['budget_priority']): string {
  return p === 0 ? 'Project-level' : p === 1 ? 'Phase-level' : p === 2 ? 'Task-level' : '—'
}

function projectBudgetTotalLabel(p: Project): string {
  if (p.budget_total === null) return '—'
  return p.budget_type === 1 ? `${p.budget_total} h` : `${p.budget_total}`
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

    <section
      v-if="project && project.budget_type !== null"
      class="mt-6 rounded border border-slate-200 bg-white p-4"
      data-cy="project-budget-summary"
    >
      <h2 class="text-sm font-semibold text-slate-900">Budget</h2>
      <dl class="mt-2 grid grid-cols-3 gap-4 text-sm">
        <div>
          <dt class="text-xs uppercase tracking-wider text-slate-500">Type</dt>
          <dd class="mt-0.5 text-slate-900" data-cy="project-detail-budget-type">{{ projectBudgetTypeLabel(project.budget_type) }}</dd>
        </div>
        <div>
          <dt class="text-xs uppercase tracking-wider text-slate-500">Total</dt>
          <dd class="mt-0.5 text-slate-900" data-cy="project-detail-budget-total">
            {{ projectBudgetTotalLabel(project) }}
          </dd>
        </div>
        <div>
          <dt class="text-xs uppercase tracking-wider text-slate-500">Priority</dt>
          <dd class="mt-0.5 text-slate-900" data-cy="project-detail-budget-priority">{{ projectBudgetPriorityLabel(project.budget_priority) }}</dd>
        </div>
      </dl>
    </section>

    <section
      v-if="project"
      class="mt-6 rounded border border-slate-200 bg-white p-4"
      data-cy="project-manager-summary"
    >
      <h2 class="text-sm font-semibold text-slate-900">Project manager</h2>
      <dl class="mt-2 grid grid-cols-2 gap-4 text-sm">
        <div>
          <dt class="text-xs uppercase tracking-wider text-slate-500">Manager</dt>
          <dd class="mt-0.5 text-slate-900" data-cy="project-detail-project-manager">
            {{ project.project_manager || '—' }}
          </dd>
        </div>
        <div>
          <dt class="text-xs uppercase tracking-wider text-slate-500">All PMs can schedule</dt>
          <dd class="mt-0.5 text-slate-900" data-cy="project-detail-all-pms-schedule">
            {{ project.all_pms_schedule ? 'Yes' : 'No' }}
          </dd>
        </div>
      </dl>
    </section>

    <section
      v-if="project"
      class="mt-8"
      data-cy="project-team-section"
    >
      <h2 class="text-lg font-semibold text-slate-900">Team</h2>
      <p class="text-sm text-slate-500">People assigned to this project, with their hourly rate.</p>
      <div class="mt-4 overflow-hidden rounded border border-slate-200 bg-white">
        <table class="w-full text-sm">
          <thead class="bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
            <tr>
              <th class="px-4 py-2">Person</th>
              <th class="px-4 py-2">Hourly rate</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="!(project.project_team?.length)">
              <td colspan="2" class="px-4 py-6 text-center text-slate-400" data-cy="project-team-empty">
                No team members yet.
              </td>
            </tr>
            <tr
              v-for="m in project.project_team ?? []"
              :key="m.people_id"
              class="hover:bg-slate-50"
              data-cy="project-team-row"
            >
              <td class="px-4 py-2 font-medium text-slate-900" data-cy="project-team-person">
                {{ personName(m.people_id) }}
              </td>
              <td class="px-4 py-2 text-slate-600" data-cy="project-team-rate">{{ m.hourly_rate }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section
      v-if="project"
      class="mt-8"
      data-cy="project-tasks-section"
    >
      <h2 class="text-lg font-semibold text-slate-900">Tasks</h2>
      <p class="text-sm text-slate-500">Scheduled work on this project.</p>
      <div class="mt-4 overflow-hidden rounded border border-slate-200 bg-white">
        <table class="w-full text-sm">
          <thead class="bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
            <tr>
              <th class="px-4 py-2">Name</th>
              <th class="px-4 py-2">Person</th>
              <th class="px-4 py-2">Hours</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="!(project.project_tasks?.length)">
              <td colspan="3" class="px-4 py-6 text-center text-slate-400" data-cy="project-tasks-empty">
                No tasks yet.
              </td>
            </tr>
            <tr
              v-for="t in project.project_tasks ?? []"
              :key="t.task_id"
              class="hover:bg-slate-50"
              data-cy="project-task-row"
            >
              <td class="px-4 py-2 text-slate-700" data-cy="project-task-name">{{ t.name || '—' }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="project-task-person">{{ personName(t.people_id) }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="project-task-hours">{{ t.hours }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section
      v-if="project"
      class="mt-8"
      data-cy="project-expenses-section"
    >
      <h2 class="text-lg font-semibold text-slate-900">Expenses</h2>
      <p class="text-sm text-slate-500">Non-labor costs charged to this project.</p>
      <div class="mt-4 overflow-hidden rounded border border-slate-200 bg-white">
        <table class="w-full text-sm">
          <thead class="bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
            <tr>
              <th class="px-4 py-2">Date</th>
              <th class="px-4 py-2">Note</th>
              <th class="px-4 py-2">Amount</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="!(project.expenses?.length)">
              <td colspan="3" class="px-4 py-6 text-center text-slate-400" data-cy="project-expenses-empty">
                No expenses yet.
              </td>
            </tr>
            <tr
              v-for="x in project.expenses ?? []"
              :key="x.expense_id"
              class="hover:bg-slate-50"
              data-cy="project-expense-row"
            >
              <td class="px-4 py-2 text-slate-600" data-cy="project-expense-date">{{ x.date }}</td>
              <td class="px-4 py-2 text-slate-700" data-cy="project-expense-note">{{ x.note || '—' }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="project-expense-amount">{{ x.amount }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

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

    <section class="mt-8" data-cy="phases-section">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-lg font-semibold text-slate-900">Phases</h2>
          <p class="text-sm text-slate-500">Time-bounded slices of the project with their own budget and rate.</p>
        </div>
        <button
          v-if="auth.isAdmin"
          data-cy="phase-create"
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="openCreatePhase"
        >
          + New phase
        </button>
      </div>

      <div class="mt-4 overflow-hidden rounded border border-slate-200 bg-white">
        <table class="w-full text-sm">
          <thead class="bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
            <tr>
              <th class="px-4 py-2"></th>
              <th class="px-4 py-2">Name</th>
              <th class="px-4 py-2">Start</th>
              <th class="px-4 py-2">End</th>
              <th class="px-4 py-2">Budget</th>
              <th class="px-4 py-2">Status</th>
              <th class="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-if="loading">
              <td colspan="7" class="px-4 py-6 text-center text-slate-400">Loading…</td>
            </tr>
            <tr v-else-if="!phases.length">
              <td colspan="7" class="px-4 py-6 text-center text-slate-400" data-cy="phases-empty">
                No phases yet.
              </td>
            </tr>
            <tr
              v-for="p in phases"
              :key="p.id"
              class="hover:bg-slate-50"
              :data-cy="`phase-row-${p.name}`"
            >
              <td class="px-4 py-2">
                <span class="inline-block h-4 w-4 rounded" :style="{ background: p.color || '#94A3B8' }"></span>
              </td>
              <td class="px-4 py-2 font-medium text-slate-900" data-cy="phase-name">{{ p.name }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="phase-start-date">{{ p.start_date || '—' }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="phase-end-date">{{ p.end_date || '—' }}</td>
              <td class="px-4 py-2 text-slate-600" data-cy="phase-budget">{{ p.budget_total }}</td>
              <td class="px-4 py-2">
                <span data-cy="phase-status" class="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700">
                  {{ phaseStatusLabel(p.status) }}
                </span>
                <span
                  v-if="p.non_billable"
                  class="ml-1 rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700"
                >Non-billable</span>
              </td>
              <td class="px-4 py-2 text-right">
                <div v-if="auth.isAdmin" class="flex justify-end gap-2 text-xs">
                  <button class="text-slate-600 hover:underline" :data-cy="`phase-edit-${p.name}`" @click="openEditPhase(p)">Edit</button>
                  <button class="text-red-600 hover:underline" :data-cy="`phase-delete-${p.name}`" @click="removePhase(p)">Delete</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div
      v-if="showPhaseForm"
      class="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 p-4"
      @click.self="showPhaseForm = false"
    >
      <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl" data-cy="phase-form">
        <h2 class="text-lg font-semibold text-slate-900">
          {{ editingPhase ? 'Edit phase' : 'New phase' }}
        </h2>
        <form class="mt-4 space-y-3 text-sm" @submit.prevent="submitPhase">
          <div>
            <label class="block text-xs font-medium text-slate-600">Name</label>
            <input
              v-model="phaseForm.name"
              data-cy="phase-name-input"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              required
            >
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Color</label>
            <div class="mt-1 flex items-center gap-2">
              <input v-model="phaseForm.color" type="color" class="h-9 w-12 rounded border border-slate-300">
              <input v-model="phaseForm.color" data-cy="phase-color-input" class="flex-1 rounded border border-slate-300 px-2 py-1.5 font-mono text-xs">
            </div>
          </div>
          <div class="grid grid-cols-2 gap-2">
            <div>
              <label class="block text-xs font-medium text-slate-600">Start date</label>
              <input
                v-model="phaseForm.start_date"
                type="date"
                data-cy="phase-start-date-input"
                class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              >
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-600">End date</label>
              <input
                v-model="phaseForm.end_date"
                type="date"
                data-cy="phase-end-date-input"
                class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              >
            </div>
          </div>
          <div class="grid grid-cols-2 gap-2">
            <div>
              <label class="block text-xs font-medium text-slate-600">Budget total</label>
              <input
                v-model.number="phaseForm.budget_total"
                type="number"
                min="0"
                step="0.01"
                data-cy="phase-budget-input"
                class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              >
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-600">Default hourly rate</label>
              <input
                v-model.number="phaseForm.default_hourly_rate"
                type="number"
                min="0"
                step="0.01"
                data-cy="phase-rate-input"
                class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              >
            </div>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Status</label>
            <select
              v-model.number="phaseForm.status"
              data-cy="phase-status-input"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
            >
              <option :value="0">Draft</option>
              <option :value="1">Tentative</option>
              <option :value="2">Confirmed</option>
            </select>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Notes</label>
            <textarea
              v-model="phaseForm.notes"
              rows="3"
              data-cy="phase-notes-input"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
            ></textarea>
          </div>
          <div>
            <label class="flex items-center gap-2 text-xs font-medium text-slate-600">
              <input v-model="phaseForm.non_billable" data-cy="phase-non-billable-input" type="checkbox" class="rounded">
              Non-billable
            </label>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="rounded px-3 py-1.5 text-slate-600 hover:bg-slate-100" @click="showPhaseForm = false">Cancel</button>
            <button
              type="submit"
              data-cy="phase-save"
              class="rounded bg-slate-900 px-3 py-1.5 text-white hover:bg-slate-700"
            >
              Save
            </button>
          </div>
        </form>
      </div>
    </div>

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
