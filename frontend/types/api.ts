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

export type Project = {
  id: string
  name: string
  client: string
  color: string
  status: 'active' | 'archived'
  notes: string
  billable: boolean
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
  time_off_created: number
  time_off_skipped: number
  milestones_created: number
  milestones_skipped: number
  warnings: string[]
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
