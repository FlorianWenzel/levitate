function ymd(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}
function startOfWeekMonday(d: Date) {
  const c = new Date(d); const dow = c.getDay()
  c.setDate(c.getDate() - ((dow + 6) % 7)); c.setHours(0,0,0,0); return c
}
function addDays(d: Date, n: number) { const c = new Date(d); c.setDate(c.getDate() + n); return c }

describe('Archive cascading and capacity edge cases', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
  })

  it('archived person disappears from the schedule grid', () => {
    const stamp = Date.now()
    let personId = ''
    cy.apiRequest({
      method: 'POST',
      url: '/api/people',
      body: { name: `Archive Test ${stamp}`, email: '', role: '', weekly_capacity_hours: 40 },
    }).then((r) => {
      personId = r.body.id
      // Verify they show up first.
      cy.visitAuthed('/schedule')
      cy.contains(`Archive Test ${stamp}`).should('be.visible')
      // Archive and re-visit.
      cy.apiRequest({ method: 'POST', url: `/api/people/${personId}/archive` })
      cy.visitAuthed('/schedule')
      cy.contains(`Archive Test ${stamp}`).should('not.exist')
    })
  })

  it('archived project is hidden from the assignment dropdown', () => {
    const stamp = Date.now()
    let projectId = ''
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Archived Proj ${stamp}`, client: '', color: '#000', notes: '' },
    }).then((r) => {
      projectId = r.body.id
      cy.apiRequest({ method: 'POST', url: `/api/projects/${projectId}/archive` })
      cy.visitAuthed('/schedule')
      cy.contains('button', '+ Assignment').click()
      cy.get('select').then(($s) => {
        const text = [...$s].map((s) => s.outerHTML).join('\n')
        expect(text).not.to.contain(`Archived Proj ${stamp}`)
      })
      cy.contains('button', 'Cancel').click()
    })
  })

  it('overallocated week shows in red on capacity report', () => {
    const stamp = Date.now()
    const monday = startOfWeekMonday(new Date())
    const target = addDays(monday, 7) // start of next week
    const projName = `Overalloc ${stamp}`
    let personId = ''
    let projectId = ''
    let assignmentId = ''

    cy.apiRequest({
      method: 'POST',
      url: '/api/people',
      body: { name: `Heavy ${stamp}`, email: '', role: '', weekly_capacity_hours: 40 },
    }).then((r) => {
      personId = r.body.id
    })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: projName, client: '', color: '#EF4444', notes: '' },
    }).then((r) => {
      projectId = r.body.id
    })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: ymd(target),
          end_date: ymd(addDays(target, 6)),
          // Span Mon..Sun, but utilization counts workdays only: 5 × 9 = 45 h > 40 h capacity.
          hours_per_day: 9,
          notes: '',
        },
      }).then((r) => { assignmentId = r.body.id })
    })

    cy.then(() => {
      cy.apiRequest({
        url: `/api/reports/utilization?from=${ymd(monday)}&to=${ymd(addDays(monday, 27))}`,
      }).then((res) => {
        expect(res.status).to.eq(200)
        const cell = (res.body as any[]).find(
          (c: any) => c.person_name === `Heavy ${stamp}` && c.week_start === ymd(target),
        )
        expect(cell, `cell for Heavy ${stamp} on ${ymd(target)}`).to.exist
        expect(cell.assigned_hours).to.eq(45) // 5 workdays × 9 h
        expect(cell.utilization_pct).to.be.greaterThan(100)
        expect(cell.overallocated).to.eq(true)
      })
    })

    cy.then(() => {
      cy.visitAuthed('/capacity')
      cy.contains('tr', `Heavy ${stamp}`).within(() => {
        cy.contains('113%') // 45/40 * 100 = 112.5 → rounded to 113%
      })
      cy.screenshot('capacity-overallocated')
    })

    cy.then(() => {
      if (assignmentId) cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${assignmentId}` })
      if (personId) cy.apiRequest({ method: 'POST', url: `/api/people/${personId}/archive` })
      if (projectId) cy.apiRequest({ method: 'POST', url: `/api/projects/${projectId}/archive` })
    })
  })

  it('time-off renders as an amber bar on the grid', () => {
    const stamp = Date.now()
    const monday = startOfWeekMonday(new Date())
    const target = addDays(monday, 8)
    let personId = ''
    let timeOffId = ''
    cy.apiRequest({
      method: 'POST',
      url: '/api/people',
      body: { name: `OffTest ${stamp}`, email: '', role: '', weekly_capacity_hours: 40 },
    }).then((r) => { personId = r.body.id })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/time-off',
        body: {
          person_id: personId,
          start_date: ymd(target),
          end_date: ymd(addDays(target, 4)),
          type: 'vacation',
          notes: 'cypress vacation',
        },
      }).then((r) => { timeOffId = r.body.id })
    })
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-time-off-id="${timeOffId}"]`).should('exist').and('be.visible')
      cy.screenshot('schedule-time-off-bar')
    })
    cy.then(() => {
      if (timeOffId) cy.apiRequest({ method: 'DELETE', url: `/api/time-off/${timeOffId}` })
      if (personId) cy.apiRequest({ method: 'POST', url: `/api/people/${personId}/archive` })
    })
  })
})
