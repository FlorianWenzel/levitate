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

describe('Dashboard', () => {
  const monday = startOfWeekMonday(new Date())

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    // Sync the OIDC user → people row.
    cy.apiRequest({ url: '/api/me' })
  })

  it('renders the four KPI cards with numeric values', () => {
    // Populate the dashboard with realistic fixtures so the screenshot matches
    // what a real user would see, not an empty-state placeholder.
    const ids: string[] = []
    const projectIds: string[] = []
    cy.apiRequest({ url: '/api/people' }).then((r) => {
      const personId = (r.body as any[])[0].id
      const make = (name: string, color: string, hours: number, lenWorkdays: number) => {
        cy.apiRequest({
          method: 'POST', url: '/api/projects',
          body: { name, client: '', color, notes: '' },
        }).then((pr) => {
          projectIds.push(pr.body.id)
          cy.apiRequest({
            method: 'POST', url: '/api/assignments',
            body: {
              person_id: personId, project_id: pr.body.id,
              start_date: ymd(monday),
              end_date: ymd(addWorkdays(monday, lenWorkdays - 1)),
              hours_per_day: hours, notes: '',
            },
          }).then((a) => ids.push(a.body.id))
        })
      }
      make('Apollo', '#0EA5E9', 4, 5)
      make('Mercury', '#EF4444', 3, 8)
      make('Gemini', '#10B981', 2, 3)
    })
    cy.then(() => {
      cy.visitAuthed('/')
      cy.contains('h1', /Hello/).should('be.visible')
      cy.get('[data-kpis]').within(() => {
        cy.get('[data-kpi="people"]').should('contain.text', 'People')
        cy.get('[data-kpi="projects"]').should('contain.text', 'Projects')
        cy.get('[data-kpi="hours"]').should('contain.text', 'This week')
        cy.get('[data-kpi="utilization"]').should('contain.text', 'Utilization')
      })
      cy.get('[data-kpi="people"]').invoke('text').should('match', /\d/)
      cy.get('[data-kpi="projects"]').invoke('text').should('match', /\d/)
      cy.get('[data-kpi="hours"]').invoke('text').should('match', /\d+(\.\d+)?h/)
      cy.get('[data-kpi="utilization"]').invoke('text').should('match', /\d+%/)
      cy.screenshot('dashboard')
    })
    cy.then(() => {
      for (const id of ids) cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${id}` })
      for (const pid of projectIds) cy.apiRequest({ method: 'POST', url: `/api/projects/${pid}/archive` })
    })
  })

  it('utilization chart renders one row per ISO week with a fill bar', () => {
    cy.visitAuthed('/')
    cy.get('[data-utilization-chart]').should('be.visible').within(() => {
      cy.contains('Team utilization').should('be.visible')
      // 8 weekly rows in the lookahead window.
      cy.get('[data-week]').should('have.length', 8)
      // Each row has a percentage label like "12%" or "100%".
      cy.get('[data-week]').first().invoke('text').should('match', /\d+%/)
    })
  })

  it('overallocation shows up on the dashboard (red bar + KPI flag)', () => {
    let projectId = ''
    let personId = ''
    let assignmentId = ''
    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Dash Overalloc ${Date.now()}`, client: '', color: '#EF4444', notes: '' },
    }).then((r) => { projectId = r.body.id })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: ymd(monday),
          end_date: ymd(addWorkdays(monday, 4)),
          // 5 workdays * 12 h = 60 h > 40 h capacity = overallocated.
          hours_per_day: 12,
          notes: '',
        },
      }).then((r) => { assignmentId = r.body.id })
    })
    cy.then(() => {
      cy.visitAuthed('/')
      cy.get(`[data-week="${ymd(monday)}"]`).within(() => {
        // Row text shows 60h and >100% utilization.
        cy.contains('60h').should('be.visible')
        cy.contains(/1\d\d%/).should('be.visible')
      })
      cy.get('[data-kpi="utilization"]').should('contain.text', 'overallocated')
    })
    cy.then(() => {
      if (assignmentId) cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${assignmentId}` })
      if (projectId) cy.apiRequest({ method: 'POST', url: `/api/projects/${projectId}/archive` })
    })
  })

  it('hours-by-project chart lists assignments grouped by project', () => {
    let personId = ''
    let p1 = '', p2 = ''
    const ids: string[] = []
    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Apollo ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
    }).then((r) => { p1 = r.body.id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Mercury ${Date.now()}`, client: '', color: '#10B981', notes: '' },
    }).then((r) => { p2 = r.body.id })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId, project_id: p1,
          start_date: ymd(monday), end_date: ymd(addWorkdays(monday, 4)),
          hours_per_day: 8, notes: '',
        },
      }).then((r) => ids.push(r.body.id))
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId, project_id: p2,
          start_date: ymd(monday), end_date: ymd(addWorkdays(monday, 1)),
          hours_per_day: 4, notes: '',
        },
      }).then((r) => ids.push(r.body.id))
    })
    cy.then(() => {
      cy.visitAuthed('/')
      cy.get('[data-project-chart]').within(() => {
        cy.contains('Apollo').should('be.visible')
        cy.contains('Mercury').should('be.visible')
        // Apollo: 5 workdays × 8 h = 40h. Mercury: 2 workdays × 4 h = 8h.
        cy.contains('40h').should('be.visible')
        cy.contains('8h').should('be.visible')
      })
      // Apollo (40h) should be sorted ABOVE Mercury (8h).
      cy.get('[data-project-chart] [data-project]').then(($rows) => {
        expect($rows[0].textContent).to.contain('Apollo')
        expect($rows[1].textContent).to.contain('Mercury')
      })
    })
    cy.then(() => {
      for (const id of ids) cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${id}` })
      if (p1) cy.apiRequest({ method: 'POST', url: `/api/projects/${p1}/archive` })
      if (p2) cy.apiRequest({ method: 'POST', url: `/api/projects/${p2}/archive` })
    })
  })

  it('upcoming time-off list shows future entries with name, type and date range', () => {
    let personId = ''
    let timeOffId = ''
    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/time-off',
        body: {
          person_id: personId,
          start_date: ymd(addWorkdays(monday, 7)),
          end_date: ymd(addWorkdays(monday, 11)),
          type: 'vacation',
          notes: '',
        },
      }).then((r) => { timeOffId = r.body.id })
    })
    cy.then(() => {
      cy.visitAuthed('/')
      cy.get('[data-time-off-list]').within(() => {
        cy.get(`[data-time-off-id="${timeOffId}"]`).should('be.visible').within(() => {
          cy.contains('Ada Admin')
          cy.contains('vacation')
        })
      })
    })
    cy.then(() => {
      if (timeOffId) cy.apiRequest({ method: 'DELETE', url: `/api/time-off/${timeOffId}` })
    })
  })

  it('empty state renders gracefully with zeroed KPIs', () => {
    cy.visitAuthed('/')
    cy.get('[data-kpis]').should('be.visible')
    // After resetState there are no assignments yet — utilization KPI is 0%.
    cy.get('[data-kpi="utilization"]').should('contain.text', '0%')
    cy.get('[data-kpi="hours"]').should('contain.text', '0h')
  })
})
