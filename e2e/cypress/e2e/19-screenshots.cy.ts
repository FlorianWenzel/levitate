// Generates the populated screenshots used in the README. Each test sets up
// a rich fixture from scratch (the global beforeEach reset wipes between
// tests), then visits the relevant page and snaps it.

function ymd(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}
function startOfWeekMonday(d: Date) {
  const c = new Date(d); const dow = c.getDay()
  c.setDate(c.getDate() - ((dow + 6) % 7)); c.setHours(0, 0, 0, 0); return c
}
function addWorkdays(d: Date, n: number) {
  const c = new Date(d); c.setHours(0, 0, 0, 0)
  if (n === 0) return c
  const step = n > 0 ? 1 : -1
  let left = Math.abs(n)
  while (left > 0) { c.setDate(c.getDate() + step); const dow = c.getDay(); if (dow !== 0 && dow !== 6) left-- }
  return c
}

// Computer-pioneers theme. Ada Lovelace + Alan Turing come from Keycloak
// (admin / member); the rest are added as contractors via the API so the
// schedule looks lived-in.
const CONTRACTORS = [
  { name: 'Grace Hopper',       capacity: 40 },
  { name: 'Donald Knuth',       capacity: 40 },
  { name: 'Edsger Dijkstra',    capacity: 32 },
  { name: 'Margaret Hamilton',  capacity: 40 },
]

const PROJECTS = [
  { name: 'Analytical Engine',  client: 'CHM',         color: '#0EA5E9' },
  { name: 'Enigma',             client: 'GC&CS',       color: '#EF4444' },
  { name: 'Apollo Guidance',    client: 'NASA',        color: '#F97316' },
  { name: 'ARPANET',            client: 'DARPA',       color: '#10B981' },
  { name: 'Compiler',           client: 'Internal',    color: '#8B5CF6' },
]

describe('README screenshots (populated demo data)', () => {
  const monday = startOfWeekMonday(new Date())

  // The setup helpers chain through cypress so tests can reuse them; each
  // returns a thenable yielding the populated state.
  function seed() {
    const peopleByName = new Map<string, string>()
    const projectByName = new Map<string, string>()

    cy.loginAs('admin@example.com', 'admin')
    // Sync Ada Lovelace (admin) and create her people row.
    cy.apiRequest({ url: '/api/me' })
    // Sync Alan Turing too so both Keycloak users show up.
    cy.loginAs('member@example.com', 'member')
    cy.apiRequest({ url: '/api/me' })
    // Switch back to admin for the writes.
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/people' }).then((r) => {
      for (const p of r.body as any[]) peopleByName.set(p.name, p.id)
    })

    for (const c of CONTRACTORS) {
      cy.apiRequest({
        method: 'POST', url: '/api/people',
        body: { name: c.name, email: '', role: '', weekly_capacity_hours: c.capacity },
      }).then((r) => peopleByName.set(c.name, r.body.id))
    }
    for (const pr of PROJECTS) {
      cy.apiRequest({
        method: 'POST', url: '/api/projects',
        body: { name: pr.name, client: pr.client, color: pr.color, notes: '' },
      }).then((r) => projectByName.set(pr.name, r.body.id))
    }

    // Distribute assignments across the next 8 weeks. Several rows get two
    // overlapping bars to show the lane-stacking behavior.
    cy.then(() => {
      type Asg = {
        person: string
        project: string
        start: number  // workday offset from current Monday
        end: number
        hours: number
      }
      const plan: Asg[] = [
        // Ada Lovelace (admin)
        { person: 'Ada Lovelace',      project: 'Analytical Engine', start: 0,  end: 9,  hours: 6 },
        { person: 'Ada Lovelace',      project: 'Compiler',          start: 5,  end: 14, hours: 2 },
        { person: 'Ada Lovelace',      project: 'ARPANET',           start: 15, end: 24, hours: 4 },
        // Alan Turing (member)
        { person: 'Alan Turing',       project: 'Enigma',            start: 0,  end: 14, hours: 8 },
        { person: 'Alan Turing',       project: 'Compiler',          start: 15, end: 22, hours: 4 },
        // Grace Hopper
        { person: 'Grace Hopper',      project: 'Compiler',          start: 0,  end: 14, hours: 6 },
        { person: 'Grace Hopper',      project: 'ARPANET',           start: 8,  end: 19, hours: 2 },
        // Donald Knuth
        { person: 'Donald Knuth',      project: 'Compiler',          start: 0,  end: 19, hours: 4 },
        { person: 'Donald Knuth',      project: 'Analytical Engine', start: 12, end: 24, hours: 4 },
        // Edsger Dijkstra (32 h capacity)
        { person: 'Edsger Dijkstra',   project: 'ARPANET',           start: 0,  end: 9,  hours: 6 },
        { person: 'Edsger Dijkstra',   project: 'Analytical Engine', start: 10, end: 24, hours: 6 },
        // Margaret Hamilton
        { person: 'Margaret Hamilton', project: 'Apollo Guidance',   start: 0,  end: 24, hours: 8 },
        { person: 'Margaret Hamilton', project: 'Compiler',          start: 5,  end: 12, hours: 2 },
      ]
      for (const a of plan) {
        cy.apiRequest({
          method: 'POST', url: '/api/assignments',
          body: {
            person_id: peopleByName.get(a.person),
            project_id: projectByName.get(a.project),
            start_date: ymd(addWorkdays(monday, a.start)),
            end_date:   ymd(addWorkdays(monday, a.end)),
            hours_per_day: a.hours,
            notes: '',
          },
        })
      }
      // A couple of upcoming time-off entries.
      cy.apiRequest({
        method: 'POST', url: '/api/time-off',
        body: {
          person_id: peopleByName.get('Grace Hopper'),
          start_date: ymd(addWorkdays(monday, 17)),
          end_date:   ymd(addWorkdays(monday, 21)),
          type: 'vacation', notes: '',
        },
      })
      cy.apiRequest({
        method: 'POST', url: '/api/time-off',
        body: {
          person_id: peopleByName.get('Donald Knuth'),
          start_date: ymd(addWorkdays(monday, 25)),
          end_date:   ymd(addWorkdays(monday, 29)),
          type: 'vacation', notes: '',
        },
      })
    })
  }

  it('captures the dashboard for Ada Lovelace', () => {
    seed()
    cy.visitAuthed('/')
    cy.contains('h1', /Hello, Ada Lovelace/).should('be.visible')
    cy.get('[data-utilization-chart]').should('be.visible')
    cy.get('[data-project-chart]').should('be.visible')
    cy.get('[data-time-off-list]').should('be.visible')
    cy.screenshot('readme-dashboard', { capture: 'viewport' })
  })

  it('captures a busy schedule view', () => {
    seed()
    cy.visitAuthed('/schedule')
    cy.get('[data-schedule-grid]').should('be.visible')
    cy.contains('Margaret Hamilton').should('be.visible')
    cy.contains('Apollo Guidance').should('be.visible')
    cy.screenshot('readme-schedule', { capture: 'viewport' })
  })

  it('captures the capacity heatmap', () => {
    seed()
    cy.visitAuthed('/capacity')
    cy.contains('h1', 'Capacity').should('be.visible')
    cy.contains('Margaret Hamilton').should('be.visible')
    cy.screenshot('readme-capacity', { capture: 'viewport' })
  })
})
