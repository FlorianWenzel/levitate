<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type {
  Project,
  ProjectBudgetPriority,
  ProjectBudgetType,
  ProjectInput,
} from '~/types/api'

type BudgetFormState = {
  type: ProjectBudgetType | ''
  total: number | ''
  priority: ProjectBudgetPriority | ''
}

const auth = useAuthStore()
const { call } = useApi()

const projects = ref<Project[]>([])
const includeArchived = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)

const showForm = ref(false)
const editing = ref<Project | null>(null)
const form = ref<ProjectInput>(emptyForm())
const budgetForm = ref<BudgetFormState>(emptyBudget())
const tagsInput = ref('')

function emptyForm(): ProjectInput {
  return { name: '', client: '', color: '#0EA5E9', notes: '', billable: true, tags: [] }
}

function emptyBudget(): BudgetFormState {
  return { type: '', total: '', priority: '' }
}

function parseTagsInput(value: string): string[] {
  return value
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t.length > 0)
}

function budgetTypeLabel(t: ProjectBudgetType | null): string {
  return t === 1 ? 'Total hours' : t === 2 ? 'Total fee' : t === 3 ? 'Hourly fee' : '—'
}

function budgetPriorityLabel(p: ProjectBudgetPriority | null): string {
  return p === 0 ? 'Project' : p === 1 ? 'Phase' : p === 2 ? 'Task' : '—'
}

function formatBudgetTotal(p: Project): string {
  if (p.budget_total === null) return '—'
  return p.budget_type === 1 ? `${p.budget_total} h` : `${p.budget_total}`
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const qs = includeArchived.value ? '?include_archived=true' : ''
    projects.value = await call<Project[]>(`/api/projects${qs}`)
  } catch (e: any) {
    error.value = e?.data?.detail ?? e?.message ?? 'Failed to load'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  form.value = emptyForm()
  budgetForm.value = emptyBudget()
  tagsInput.value = ''
  showForm.value = true
}

function openEdit(p: Project) {
  editing.value = p
  form.value = {
    name: p.name,
    client: p.client,
    color: p.color,
    notes: p.notes,
    billable: p.billable,
    tags: [...(p.tags ?? [])],
  }
  budgetForm.value = {
    type: p.budget_type ?? '',
    total: p.budget_total ?? '',
    priority: p.budget_priority ?? '',
  }
  tagsInput.value = (p.tags ?? []).join(', ')
  showForm.value = true
}

async function submit() {
  try {
    const body: ProjectInput = { ...form.value }
    body.budget_type = budgetForm.value.type === '' ? null : (Number(budgetForm.value.type) as ProjectBudgetType)
    body.budget_total = budgetForm.value.total === '' ? null : Number(budgetForm.value.total)
    body.budget_priority = budgetForm.value.priority === '' ? null : (Number(budgetForm.value.priority) as ProjectBudgetPriority)
    body.tags = parseTagsInput(tagsInput.value)
    if (editing.value) {
      await call(`/api/projects/${editing.value.id}`, { method: 'PATCH', body })
    } else {
      await call('/api/projects', { method: 'POST', body })
    }
    showForm.value = false
    await load()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Save failed')
  }
}

async function archive(p: Project) {
  if (!confirm(`Archive ${p.name}?`)) return
  await call(`/api/projects/${p.id}/archive`, { method: 'POST' })
  await load()
}

async function unarchive(p: Project) {
  await call(`/api/projects/${p.id}/unarchive`, { method: 'POST' })
  await load()
}

watch(includeArchived, load)
onMounted(load)
</script>

<template>
  <main class="mx-auto max-w-6xl px-6 py-8">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-900">Projects</h1>
        <p class="text-sm text-slate-500">Things people work on. Each has a color used in the schedule grid.</p>
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
          + New project
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
            <th class="px-4 py-2"></th>
            <th class="px-4 py-2">Name</th>
            <th class="px-4 py-2">Client</th>
            <th class="px-4 py-2">Billable</th>
            <th class="px-4 py-2">Budget</th>
            <th class="px-4 py-2">Tags</th>
            <th class="px-4 py-2">Status</th>
            <th class="px-4 py-2"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-if="loading">
            <td colspan="8" class="px-4 py-6 text-center text-slate-400">Loading…</td>
          </tr>
          <tr v-else-if="!projects.length">
            <td colspan="8" class="px-4 py-6 text-center text-slate-400">No projects yet.</td>
          </tr>
          <tr v-for="p in projects" :key="p.id" class="hover:bg-slate-50" :data-cy="`project-row-${p.name}`">
            <td class="px-4 py-2">
              <span class="inline-block h-4 w-4 rounded" :style="{ background: p.color }"></span>
            </td>
            <td class="px-4 py-2 font-medium text-slate-900">
              <NuxtLink
                :to="`/projects/${p.id}`"
                class="hover:underline"
                :data-cy="`project-link-${p.name}`"
              >
                {{ p.name }}
              </NuxtLink>
            </td>
            <td class="px-4 py-2 text-slate-600">{{ p.client }}</td>
            <td class="px-4 py-2">
              <span
                v-if="p.billable"
                data-cy="billable-badge"
                class="rounded bg-emerald-100 px-1.5 py-0.5 text-xs text-emerald-700"
              >Billable</span>
              <span
                v-else
                data-cy="non-billable-badge"
                class="rounded bg-amber-100 px-1.5 py-0.5 text-xs text-amber-700"
              >Non-billable</span>
            </td>
            <td class="px-4 py-2 text-xs text-slate-600" data-cy="project-budget-cell">
              <template v-if="p.budget_type !== null">
                <div data-cy="project-budget-type">{{ budgetTypeLabel(p.budget_type) }}</div>
                <div data-cy="project-budget-total">{{ formatBudgetTotal(p) }}</div>
                <div class="text-[10px] uppercase text-slate-400" data-cy="project-budget-priority">
                  {{ budgetPriorityLabel(p.budget_priority) }}-level
                </div>
              </template>
              <span v-else class="text-slate-400">—</span>
            </td>
            <td class="px-4 py-2" data-cy="project-tags-cell">
              <div v-if="p.tags && p.tags.length" class="flex flex-wrap gap-1">
                <span
                  v-for="t in p.tags"
                  :key="t"
                  data-cy="project-tag-chip"
                  class="rounded bg-sky-100 px-1.5 py-0.5 text-xs text-sky-700"
                >{{ t }}</span>
              </div>
              <span v-else class="text-slate-400 text-xs">—</span>
            </td>
            <td class="px-4 py-2">
              <span v-if="p.status === 'archived'" class="rounded bg-slate-200 px-1.5 py-0.5 text-xs text-slate-700">Archived</span>
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

    <div v-if="showForm" class="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 p-4" @click.self="showForm = false">
      <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h2 class="text-lg font-semibold text-slate-900">
          {{ editing ? 'Edit project' : 'New project' }}
        </h2>
        <form class="mt-4 space-y-3 text-sm" @submit.prevent="submit">
          <div>
            <label class="block text-xs font-medium text-slate-600">Name</label>
            <input v-model="form.name" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Client</label>
            <input v-model="form.client" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Color</label>
            <div class="mt-1 flex items-center gap-2">
              <input v-model="form.color" type="color" class="h-9 w-12 rounded border border-slate-300">
              <input v-model="form.color" class="flex-1 rounded border border-slate-300 px-2 py-1.5 font-mono text-xs">
            </div>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Notes</label>
            <textarea v-model="form.notes" rows="3" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"></textarea>
          </div>
          <div>
            <label class="flex items-center gap-2 text-xs font-medium text-slate-600">
              <input v-model="form.billable" type="checkbox" data-cy="billable-toggle" class="rounded">
              Billable
            </label>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Tags</label>
            <input
              v-model="tagsInput"
              data-cy="project-tags-input"
              placeholder="Comma-separated, e.g. design, frontend"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
            >
          </div>
          <fieldset class="space-y-2 rounded border border-slate-200 p-3" data-cy="project-budget-fieldset">
            <legend class="px-1 text-xs font-medium text-slate-600">Budget</legend>
            <div class="grid grid-cols-2 gap-2">
              <div>
                <label class="block text-xs font-medium text-slate-600">Budget type</label>
                <select
                  v-model="budgetForm.type"
                  data-cy="project-budget-type-input"
                  class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
                >
                  <option value="">None</option>
                  <option :value="1">Total hours</option>
                  <option :value="2">Total fee</option>
                  <option :value="3">Hourly fee</option>
                </select>
              </div>
              <div>
                <label class="block text-xs font-medium text-slate-600">Budget priority</label>
                <select
                  v-model="budgetForm.priority"
                  data-cy="project-budget-priority-input"
                  class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
                >
                  <option value="">None</option>
                  <option :value="0">Project</option>
                  <option :value="1">Phase</option>
                  <option :value="2">Task</option>
                </select>
              </div>
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-600">Budget total</label>
              <input
                v-model.number="budgetForm.total"
                type="number"
                min="0"
                step="0.01"
                data-cy="project-budget-total-input"
                class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
                placeholder="Hours or amount"
              >
            </div>
          </fieldset>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="rounded px-3 py-1.5 text-slate-600 hover:bg-slate-100" @click="showForm = false">Cancel</button>
            <button type="submit" data-cy="project-save" class="rounded bg-slate-900 px-3 py-1.5 text-white hover:bg-slate-700">Save</button>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>
