<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import { useRolesStore } from '~/stores/roles'
import type { Role, RoleInput } from '~/types/api'

const auth = useAuthStore()
const rolesStore = useRolesStore()

const showForm = ref(false)
const editing = ref<Role | null>(null)
const form = ref<RoleInput>(emptyForm())
const submitError = ref<string | null>(null)

function emptyForm(): RoleInput {
  return { name: '', default_hourly_rate: '0.000', cost_rate_history: [] }
}

function openCreate() {
  editing.value = null
  form.value = emptyForm()
  submitError.value = null
  showForm.value = true
}

function openEdit(r: Role) {
  editing.value = r
  form.value = {
    name: r.name,
    default_hourly_rate: r.default_hourly_rate,
    cost_rate_history: r.cost_rate_history.map((e) => ({ ...e })),
  }
  submitError.value = null
  showForm.value = true
}

async function submit() {
  submitError.value = null
  try {
    if (editing.value) {
      await rolesStore.update(editing.value.id, form.value)
    } else {
      await rolesStore.create(form.value)
    }
    showForm.value = false
  } catch (e: any) {
    submitError.value = e?.data?.detail ?? e?.message ?? 'Save failed'
  }
}

async function remove(r: Role) {
  if (!confirm(`Delete role "${r.name}"?`)) return
  try {
    await rolesStore.remove(r.id)
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Delete failed')
  }
}

function addHistoryEntry() {
  const today = new Date().toISOString().slice(0, 10)
  form.value.cost_rate_history = [
    ...(form.value.cost_rate_history ?? []),
    { rate: '0.000', effective_date: today },
  ]
}

function removeHistoryEntry(idx: number) {
  form.value.cost_rate_history = (form.value.cost_rate_history ?? []).filter((_, i) => i !== idx)
}

onMounted(() => {
  rolesStore.fetch()
})
</script>

<template>
  <main class="mx-auto max-w-6xl px-6 py-8">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-900">Roles</h1>
        <p class="text-sm text-slate-500">
          Reusable job roles (e.g. "Senior Designer") with default billing rates and cost-rate history.
        </p>
      </div>
      <div class="flex items-center gap-3">
        <button
          v-if="auth.isAdmin"
          data-cy="new-role-button"
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="openCreate"
        >
          + New role
        </button>
      </div>
    </div>

    <div v-if="rolesStore.error" class="mt-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">
      {{ rolesStore.error }}
    </div>

    <div class="mt-6 overflow-hidden rounded border border-slate-200 bg-white">
      <table class="w-full text-sm" data-cy="roles-table">
        <thead class="bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
          <tr>
            <th class="px-4 py-2">Name</th>
            <th class="px-4 py-2 text-right">Default hourly rate</th>
            <th class="px-4 py-2 text-right">People</th>
            <th class="px-4 py-2 text-right">Cost-rate history</th>
            <th class="px-4 py-2"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-if="rolesStore.loading">
            <td colspan="5" class="px-4 py-6 text-center text-slate-400">Loading…</td>
          </tr>
          <tr v-else-if="!rolesStore.roles.length">
            <td colspan="5" class="px-4 py-6 text-center text-slate-400" data-cy="roles-empty">No roles yet.</td>
          </tr>
          <tr
            v-for="r in rolesStore.roles"
            :key="r.id"
            class="hover:bg-slate-50"
            :data-cy="`role-row-${r.name}`"
          >
            <td class="px-4 py-2 font-medium text-slate-900" data-cy="role-name">{{ r.name }}</td>
            <td class="px-4 py-2 text-right tabular-nums" data-cy="role-rate">{{ r.default_hourly_rate }}</td>
            <td class="px-4 py-2 text-right tabular-nums" data-cy="role-people-count">{{ r.people_count }}</td>
            <td class="px-4 py-2 text-right tabular-nums">{{ r.cost_rate_history.length }}</td>
            <td class="px-4 py-2 text-right">
              <div v-if="auth.isAdmin" class="flex justify-end gap-2 text-xs">
                <button class="text-slate-600 hover:underline" @click="openEdit(r)">Edit</button>
                <button class="text-red-600 hover:underline" @click="remove(r)">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="showForm"
      class="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 p-4"
      data-cy="role-form"
      @click.self="showForm = false"
    >
      <div class="w-full max-w-lg rounded-lg bg-white p-6 shadow-xl">
        <h2 class="text-lg font-semibold text-slate-900">
          {{ editing ? 'Edit role' : 'New role' }}
        </h2>
        <form class="mt-4 space-y-3 text-sm" @submit.prevent="submit">
          <div>
            <label class="block text-xs font-medium text-slate-600">Name</label>
            <input
              v-model="form.name"
              data-cy="role-form-name"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"
              required
            >
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Default hourly rate</label>
            <input
              v-model="form.default_hourly_rate"
              data-cy="role-form-rate"
              type="text"
              inputmode="decimal"
              placeholder="0.000"
              class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5 font-mono"
            >
            <p class="mt-1 text-xs text-slate-400">String-formatted hourly bill rate (e.g. "260.000") — matches Float.</p>
          </div>
          <div>
            <div class="flex items-center justify-between">
              <label class="block text-xs font-medium text-slate-600">Cost-rate history</label>
              <button
                type="button"
                class="text-xs text-slate-600 hover:underline"
                @click="addHistoryEntry"
              >
                + Add entry
              </button>
            </div>
            <div class="mt-2 space-y-2">
              <div
                v-for="(entry, idx) in form.cost_rate_history"
                :key="idx"
                class="flex items-center gap-2"
              >
                <input
                  v-model="entry.rate"
                  placeholder="rate (e.g. 180.000)"
                  class="flex-1 rounded border border-slate-300 px-2 py-1 font-mono text-xs"
                >
                <input
                  v-model="entry.effective_date"
                  type="date"
                  class="rounded border border-slate-300 px-2 py-1 text-xs"
                >
                <button
                  type="button"
                  class="text-xs text-red-600 hover:underline"
                  @click="removeHistoryEntry(idx)"
                >
                  Remove
                </button>
              </div>
              <p v-if="!form.cost_rate_history?.length" class="text-xs text-slate-400">No history entries.</p>
            </div>
          </div>
          <div v-if="submitError" class="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-700">
            {{ submitError }}
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <button
              type="button"
              class="rounded px-3 py-1.5 text-slate-600 hover:bg-slate-100"
              @click="showForm = false"
            >
              Cancel
            </button>
            <button
              type="submit"
              data-cy="role-form-submit"
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
