<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
import type { Assignment, AssignmentInput, Person, Project, TimeOff, TimeOffInput } from '~/types/api'
import { addDays, addWorkdays, ceilToWorkday, parseYMD, startOfWeekMonday, workdayAt, ymd } from '~/utils/dates'

const auth = useAuthStore()
const { call } = useApi()

// Visible window is a fixed 16 weeks (80 workdays). The window slides via the
// nav buttons; inside the window the user scrolls freely with the trackpad —
// no edge-trigger magic, which is what made the date jump years per gesture.
//
// Initial range: today's Monday minus 4 weeks (today sits ~25 % from the left).
const WINDOW_WORKDAYS = 80
const WORKDAYS_PER_WEEK = 5
const PAST_PAD_WORKDAYS = 20 // 4 weeks of past visible by default

function defaultRangeStart(): Date {
  return addWorkdays(ceilToWorkday(startOfWeekMonday(new Date())), -PAST_PAD_WORKDAYS)
}

const numWorkdays = ref<number>(WINDOW_WORKDAYS)
const rangeStart = ref<Date>(defaultRangeStart())
const rangeEnd = computed(() => workdayAt(rangeStart.value, numWorkdays.value - 1))

const people = ref<Person[]>([])
const projects = ref<Project[]>([])
const assignments = ref<Assignment[]>([])
const timeOff = ref<TimeOff[]>([])
const error = ref<string | null>(null)

async function loadAll() {
  try {
    const [ps, prjs, asg, off] = await Promise.all([
      call<Person[]>('/api/people'),
      call<Project[]>('/api/projects'),
      call<Assignment[]>(`/api/assignments?from=${ymd(rangeStart.value)}&to=${ymd(rangeEnd.value)}`),
      call<TimeOff[]>(`/api/time-off?from=${ymd(rangeStart.value)}&to=${ymd(rangeEnd.value)}`),
    ])
    people.value = ps
    projects.value = prjs
    assignments.value = asg
    timeOff.value = off
  } catch (e: any) {
    error.value = e?.data?.detail ?? e?.message ?? 'Failed to load'
  }
}

async function loadRangeData() {
  const [asg, off] = await Promise.all([
    call<Assignment[]>(`/api/assignments?from=${ymd(rangeStart.value)}&to=${ymd(rangeEnd.value)}`),
    call<TimeOff[]>(`/api/time-off?from=${ymd(rangeStart.value)}&to=${ymd(rangeEnd.value)}`),
  ])
  assignments.value = asg
  timeOff.value = off
}

// ----- Navigation -----
function shiftBy(workdays: number) {
  rangeStart.value = addWorkdays(rangeStart.value, workdays)
}

function today() {
  rangeStart.value = defaultRangeStart()
  // After the new range loads, scroll today into view (4 weeks from the left
  // = past pad).
  nextTick(() => {
    const wrapper = document.querySelector('[data-schedule-grid]') as HTMLElement | null
    if (wrapper) wrapper.scrollLeft = PAST_PAD_WORKDAYS * 36
  })
}

watch([rangeStart, numWorkdays], loadRangeData)

// ----- Edit / create modal -----
const showForm = ref(false)
const editing = ref<Assignment | null>(null)
const form = ref<AssignmentInput>(emptyForm())

function emptyForm(): AssignmentInput {
  return {
    person_id: '',
    project_id: '',
    start_date: ymd(rangeStart.value),
    end_date: ymd(rangeStart.value),
    hours_per_day: 8,
    notes: '',
  }
}

function openCreate(seed?: { personId?: string; start?: string; end?: string }) {
  editing.value = null
  const fallback = ymd(rangeStart.value)
  form.value = {
    person_id: seed?.personId ?? people.value[0]?.id ?? '',
    project_id: projects.value.find(p => !p.archived_at)?.id ?? '',
    start_date: seed?.start ?? fallback,
    end_date: seed?.end ?? seed?.start ?? fallback,
    hours_per_day: 8,
    notes: '',
  }
  showForm.value = true
}

function openEdit(a: Assignment) {
  editing.value = a
  form.value = {
    person_id: a.person_id,
    project_id: a.project_id,
    start_date: a.start_date,
    end_date: a.end_date,
    hours_per_day: Number(a.hours_per_day),
    notes: a.notes,
  }
  showForm.value = true
}

async function submit() {
  try {
    if (editing.value) {
      await call(`/api/assignments/${editing.value.id}`, { method: 'PATCH', body: form.value })
    } else {
      await call('/api/assignments', { method: 'POST', body: form.value })
    }
    showForm.value = false
    await loadRangeData()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Save failed')
  }
}

async function remove() {
  if (!editing.value) return
  if (!confirm('Delete this assignment?')) return
  try {
    await call(`/api/assignments/${editing.value.id}`, { method: 'DELETE' })
    showForm.value = false
    await loadRangeData()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Delete failed')
  }
}

// ----- Right-click context menu -----
type CtxState = { show: boolean; x: number; y: number; assignment: Assignment | null; date: string }
const ctx = ref<CtxState>({ show: false, x: 0, y: 0, assignment: null, date: '' })

function openCtxMenu(payload: { assignment: Assignment; x: number; y: number; date: string }) {
  ctx.value = { show: true, x: payload.x, y: payload.y, assignment: payload.assignment, date: payload.date }
}

function closeCtxMenu() {
  ctx.value = { ...ctx.value, show: false }
}

const canSplit = computed(() => {
  const a = ctx.value.assignment
  if (!a) return false
  return a.start_date !== a.end_date
})

function plusOne(s: string): string {
  return ymd(addWorkdays(parseYMD(s), 1))
}

function minusOne(s: string): string {
  return ymd(addWorkdays(parseYMD(s), -1))
}

function ctxEdit() {
  if (!ctx.value.assignment) return
  openEdit(ctx.value.assignment)
  closeCtxMenu()
}

async function ctxSplit() {
  const a = ctx.value.assignment
  const click = ctx.value.date
  closeCtxMenu()
  if (!a || a.start_date === a.end_date) return
  // Split such that the clicked day is the LAST day of the first half.
  // If the click landed on the assignment's last day, split before it instead.
  let aEnd: string, bStart: string
  if (click >= a.end_date) {
    aEnd = minusOne(a.end_date)
    bStart = a.end_date
  } else if (click < a.start_date) {
    aEnd = a.start_date
    bStart = plusOne(a.start_date)
  } else {
    aEnd = click
    bStart = plusOne(click)
  }
  try {
    await call(`/api/assignments/${a.id}`, {
      method: 'PATCH',
      body: {
        person_id: a.person_id,
        project_id: a.project_id,
        start_date: a.start_date,
        end_date: aEnd,
        hours_per_day: Number(a.hours_per_day),
        notes: a.notes,
      },
    })
    await call('/api/assignments', {
      method: 'POST',
      body: {
        person_id: a.person_id,
        project_id: a.project_id,
        start_date: bStart,
        end_date: a.end_date,
        hours_per_day: Number(a.hours_per_day),
        notes: a.notes,
      },
    })
    await loadRangeData()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Split failed')
  }
}

async function ctxDelete() {
  const a = ctx.value.assignment
  closeCtxMenu()
  if (!a) return
  if (!confirm('Delete this assignment?')) return
  try {
    await call(`/api/assignments/${a.id}`, { method: 'DELETE' })
    await loadRangeData()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Delete failed')
  }
}

// ----- Time-off modal -----
const showTOForm = ref(false)
const editingTO = ref<TimeOff | null>(null)
const toForm = ref<TimeOffInput>(emptyTOForm())

function emptyTOForm(): TimeOffInput {
  return {
    person_id: '',
    start_date: ymd(rangeStart.value),
    end_date: ymd(rangeStart.value),
    type: 'vacation',
    notes: '',
  }
}

function openCreateTO(seed?: { personId?: string; date?: string }) {
  editingTO.value = null
  toForm.value = {
    person_id: seed?.personId ?? people.value[0]?.id ?? '',
    start_date: seed?.date ?? ymd(rangeStart.value),
    end_date: seed?.date ?? ymd(rangeStart.value),
    type: 'vacation',
    notes: '',
  }
  showTOForm.value = true
}

function openEditTO(t: TimeOff) {
  editingTO.value = t
  toForm.value = {
    person_id: t.person_id,
    start_date: t.start_date,
    end_date: t.end_date,
    type: t.type,
    notes: t.notes,
  }
  showTOForm.value = true
}

async function submitTO() {
  try {
    if (editingTO.value) {
      await call(`/api/time-off/${editingTO.value.id}`, { method: 'PATCH', body: toForm.value })
    } else {
      await call('/api/time-off', { method: 'POST', body: toForm.value })
    }
    showTOForm.value = false
    await loadRangeData()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Save failed')
  }
}

async function removeTO() {
  if (!editingTO.value) return
  if (!confirm('Delete this time-off entry?')) return
  try {
    await call(`/api/time-off/${editingTO.value.id}`, { method: 'DELETE' })
    showTOForm.value = false
    await loadRangeData()
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Delete failed')
  }
}

async function onMoveAssignment(payload: { id: string; start: string; end: string; hoursPerDay: number }) {
  const a = assignments.value.find((x) => x.id === payload.id)
  if (!a) return
  const optimistic = {
    ...a,
    start_date: payload.start,
    end_date: payload.end,
    hours_per_day: payload.hoursPerDay,
  }
  assignments.value = assignments.value.map((x) => (x.id === a.id ? optimistic : x))
  try {
    await call(`/api/assignments/${a.id}`, {
      method: 'PATCH',
      body: {
        person_id: a.person_id,
        project_id: a.project_id,
        start_date: payload.start,
        end_date: payload.end,
        hours_per_day: payload.hoursPerDay,
        notes: a.notes,
      },
    })
  } catch (e: any) {
    alert(e?.data?.detail ?? e?.message ?? 'Update failed')
    await loadRangeData()
  }
}

const activePeople = computed(() => people.value.filter((p) => !p.archived_at))
const activeProjects = computed(() => projects.value.filter((p) => !p.archived_at))

onMounted(async () => {
  await loadAll()
  // Land with today positioned ~25% from the left edge so the user sees a
  // little past + a lot of upcoming work.
  await nextTick()
  const wrapper = document.querySelector('[data-schedule-grid]') as HTMLElement | null
  if (wrapper) wrapper.scrollLeft = PAST_PAD_WORKDAYS * 36
})
</script>

<template>
  <main class="mx-auto max-w-[1400px] px-6 py-6">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-900">Schedule</h1>
        <p class="text-sm text-slate-500">
          {{ rangeStart.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) }} –
          {{ rangeEnd.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }) }}
        </p>
      </div>
      <div class="flex items-center gap-1">
        <button class="rounded border border-slate-300 px-2 py-1.5 text-sm hover:bg-slate-100" title="Back 4 weeks" @click="shiftBy(-4 * WORKDAYS_PER_WEEK)">«</button>
        <button class="rounded border border-slate-300 px-2 py-1.5 text-sm hover:bg-slate-100" title="Back 1 week" @click="shiftBy(-WORKDAYS_PER_WEEK)">‹</button>
        <button class="rounded border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-100 ml-1 mr-1" @click="today">Today</button>
        <button class="rounded border border-slate-300 px-2 py-1.5 text-sm hover:bg-slate-100" title="Forward 1 week" @click="shiftBy(WORKDAYS_PER_WEEK)">›</button>
        <button class="rounded border border-slate-300 px-2 py-1.5 text-sm hover:bg-slate-100" title="Forward 4 weeks" @click="shiftBy(4 * WORKDAYS_PER_WEEK)">»</button>
        <button
          v-if="auth.isAdmin"
          class="ml-2 rounded border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-100"
          @click="openCreateTO()"
        >
          + Time off
        </button>
        <button
          v-if="auth.isAdmin"
          class="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-slate-700"
          @click="openCreate()"
        >
          + Assignment
        </button>
      </div>
    </div>

    <div v-if="error" class="mb-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>

    <ScheduleGrid
      :people="activePeople"
      :projects="projects"
      :assignments="assignments"
      :time-off="timeOff"
      :range-start="rangeStart"
      :num-workdays="numWorkdays"
      @edit-assignment="openEdit"
      @move-assignment="onMoveAssignment"
      @create-assignment="(p) => openCreate({ personId: p.personId, start: p.start, end: p.end })"
      @edit-time-off="openEditTO"
      @context-menu="openCtxMenu"
    />

    <!-- Right-click context menu for assignment bars -->
    <template v-if="ctx.show">
      <div class="fixed inset-0 z-40" @click="closeCtxMenu" @contextmenu.prevent="closeCtxMenu" />
      <div
        data-bar-context-menu
        class="fixed z-50 min-w-[160px] rounded border border-slate-200 bg-white py-1 shadow-lg"
        :style="{ top: ctx.y + 'px', left: ctx.x + 'px' }"
        @click.stop
        @contextmenu.prevent.stop
      >
        <button
          data-ctx-action="edit"
          class="block w-full px-3 py-1.5 text-left text-sm text-slate-700 hover:bg-slate-100"
          @click="ctxEdit"
        >
          Edit
        </button>
        <button
          data-ctx-action="split"
          class="block w-full px-3 py-1.5 text-left text-sm text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:text-slate-300 disabled:hover:bg-transparent"
          :disabled="!canSplit"
          @click="ctxSplit"
        >
          Split
        </button>
        <div class="my-1 border-t border-slate-100" />
        <button
          data-ctx-action="delete"
          class="block w-full px-3 py-1.5 text-left text-sm text-red-600 hover:bg-red-50"
          @click="ctxDelete"
        >
          Delete
        </button>
      </div>
    </template>

    <p class="mt-3 text-xs text-slate-500">
      Click a bar to edit. Drag the side handles to change duration, the top handle to change hours/day. Right-click a bar for Edit/Split/Delete. Drag across empty cells to create a new assignment. Use the navigation buttons to move the visible range; scroll the grid horizontally inside it.
    </p>

    <!-- Modal -->
    <div v-if="showForm" class="fixed inset-0 z-20 flex items-center justify-center bg-slate-900/40 p-4" @click.self="showForm = false">
      <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h2 class="text-lg font-semibold text-slate-900">
          {{ editing ? 'Edit assignment' : 'New assignment' }}
        </h2>
        <form class="mt-4 space-y-3 text-sm" @submit.prevent="submit">
          <div>
            <label class="block text-xs font-medium text-slate-600">Person</label>
            <select v-model="form.person_id" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
              <option v-for="p in activePeople" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Project</label>
            <select v-model="form.project_id" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
              <option v-for="p in activeProjects" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-slate-600">Start</label>
              <input v-model="form.start_date" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-600">End</label>
              <input v-model="form.end_date" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
            </div>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Hours per day</label>
            <input v-model.number="form.hours_per_day" type="number" min="0.25" max="24" step="0.25" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Notes</label>
            <textarea v-model="form.notes" rows="2" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"></textarea>
          </div>
          <div class="flex items-center justify-between pt-2">
            <button v-if="editing" type="button" class="text-xs text-red-600 hover:underline" @click="remove">Delete</button>
            <span v-else />
            <div class="flex gap-2">
              <button type="button" class="rounded px-3 py-1.5 text-slate-600 hover:bg-slate-100" @click="showForm = false">Cancel</button>
              <button type="submit" class="rounded bg-slate-900 px-3 py-1.5 text-white hover:bg-slate-700">Save</button>
            </div>
          </div>
        </form>
      </div>
    </div>

    <!-- Time-off modal -->
    <div v-if="showTOForm" class="fixed inset-0 z-20 flex items-center justify-center bg-slate-900/40 p-4" @click.self="showTOForm = false">
      <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h2 class="text-lg font-semibold text-slate-900">
          {{ editingTO ? 'Edit time off' : 'New time off' }}
        </h2>
        <form class="mt-4 space-y-3 text-sm" @submit.prevent="submitTO">
          <div>
            <label class="block text-xs font-medium text-slate-600">Person</label>
            <select v-model="toForm.person_id" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
              <option v-for="p in activePeople" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Type</label>
            <select v-model="toForm.type" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5">
              <option value="vacation">Vacation</option>
              <option value="sick">Sick</option>
              <option value="holiday">Holiday</option>
              <option value="other">Other</option>
            </select>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-slate-600">Start</label>
              <input v-model="toForm.start_date" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-600">End</label>
              <input v-model="toForm.end_date" type="date" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5" required>
            </div>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-600">Notes</label>
            <textarea v-model="toForm.notes" rows="2" class="mt-1 w-full rounded border border-slate-300 px-2 py-1.5"></textarea>
          </div>
          <div class="flex items-center justify-between pt-2">
            <button v-if="editingTO" type="button" class="text-xs text-red-600 hover:underline" @click="removeTO">Delete</button>
            <span v-else />
            <div class="flex gap-2">
              <button type="button" class="rounded px-3 py-1.5 text-slate-600 hover:bg-slate-100" @click="showTOForm = false">Cancel</button>
              <button type="submit" class="rounded bg-slate-900 px-3 py-1.5 text-white hover:bg-slate-700">Save</button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>
