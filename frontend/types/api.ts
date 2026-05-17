export type Person = {
  id: string
  name: string
  email: string
  role: string
  weekly_capacity_hours: number
  archived_at: string | null
  created_at: string
  updated_at: string
}

export type PersonInput = {
  name: string
  email: string
  role: string
  weekly_capacity_hours: number
}

// Float Project budget enums (see https://developer.float.com/swagger-api-v3.yaml).
// `budget_type`: 1=Total hours, 2=Total fee, 3=Hourly fee.
// `budget_priority`: 0=Project-level, 1=Phase-level, 2=Task-level.
export type ProjectBudgetType = 1 | 2 | 3
export type ProjectBudgetPriority = 0 | 1 | 2

export type Project = {
  id: string
  name: string
  client: string
  color: string
  status: 'active' | 'archived'
  notes: string
  billable: boolean
  budget_type: ProjectBudgetType | null
  budget_total: number | null
  budget_priority: ProjectBudgetPriority | null
  tags: string[]
  archived_at: string | null
  created_at: string
  updated_at: string
}

export type ProjectInput = {
  name: string
  client: string
  color: string
  notes: string
  billable: boolean
  budget_type?: ProjectBudgetType | null
  budget_total?: number | null
  budget_priority?: ProjectBudgetPriority | null
  tags?: string[]
}

export type Assignment = {
  id: string
  person_id: string
  project_id: string
  start_date: string
  end_date: string
  hours_per_day: number
  notes: string
  created_at: string
  updated_at: string
}

export type AssignmentInput = {
  person_id: string
  project_id: string
  start_date: string
  end_date: string
  hours_per_day: number
  notes: string
}

export type TimeOffType = 'vacation' | 'sick' | 'holiday' | 'other'

export type TimeOff = {
  id: string
  person_id: string
  start_date: string
  end_date: string
  type: TimeOffType
  notes: string
  created_at: string
  updated_at: string
}

export type TimeOffInput = {
  person_id: string
  start_date: string
  end_date: string
  type: TimeOffType
  notes: string
}

export type UtilizationCell = {
  person_id: string
  person_name: string
  weekly_capacity_hours: number
  week_start: string
  assigned_hours: number
  time_off_hours: number
  available_hours: number
  utilization_pct: number
  overallocated: boolean
}

export type FloatImportInput = {
  api_token: string
  base_url: string
  start_date: string
  end_date: string
}

export type FloatImportResult = {
  people_created: number
  people_skipped: number
  projects_created: number
  projects_skipped: number
  assignments_created: number
  assignments_skipped: number
  assignments_deleted: number
  time_off_created: number
  time_off_skipped: number
  time_off_deleted: number
  milestones_created: number
  milestones_skipped: number
  logged_time_created: number
  logged_time_skipped: number
  logged_time_deleted: number
  warnings: string[]
}

export type LoggedTime = {
  id: string
  person_id: string
  project_id: string | null
  date: string
  hours: number
  billable: boolean
  notes: string
  // `locked` and `locked_date` mirror Float's LoggedTime schema
  // (https://developer.float.com/reference/logged-time): locked is a
  // server-managed read-only flag (set when project/phase/task lock
  // settings close the timesheet window), and locked_date records when
  // the lock was applied.
  locked: boolean
  locked_date: string | null
  created_at: string
  updated_at: string
}

export type LoggedTimeInput = {
  person_id: string
  date: string
  hours: number
  notes?: string
  project_id?: string | null
}

export type Milestone = {
  id: string
  project_id: string
  phase_id: string | null
  name: string
  date: string
  end_date: string | null
  created_at: string
  updated_at: string
}

export type MilestoneInput = {
  name: string
  date: string
  end_date?: string | null
  phase_id?: string | null
}

// Phase mirrors Float's Phase entity: a named time-bounded slice of a project
// with its own optional budget, hourly rate, and billable flag. `status`
// uses Float's encoding (0=Draft, 1=Tentative, 2=Confirmed) and `active`
// reflects the archive state (1=Active, 0=Archived).
export type Phase = {
  id: string
  project_id: string
  name: string
  color: string
  notes: string
  start_date: string | null
  end_date: string | null
  budget_total: number
  default_hourly_rate: number
  non_billable: boolean
  status: 0 | 1 | 2
  active: 0 | 1
  archived_at: string | null
  created_at: string
  updated_at: string
}

export type PhaseInput = {
  name: string
  color?: string
  notes?: string
  start_date?: string | null
  end_date?: string | null
  budget_total?: number
  default_hourly_rate?: number
  non_billable?: boolean
  status?: 0 | 1 | 2
}

// Role mirrors Float's Roles entity: a reusable job role with an associated
// hourly bill rate (`default_hourly_rate`, kept as a string to preserve
// Float's "260.000" format) and a historical cost-rate trail.
export type RoleCostRateEntry = {
  rate: string
  effective_date: string
}

export type Role = {
  id: string
  name: string
  default_hourly_rate: string
  cost_rate_history: RoleCostRateEntry[]
  people_ids: string[]
  people_count: number
  created_at: string
  updated_at: string
}

export type RoleInput = {
  name: string
  default_hourly_rate?: string | number
  cost_rate_history?: RoleCostRateEntry[]
}
