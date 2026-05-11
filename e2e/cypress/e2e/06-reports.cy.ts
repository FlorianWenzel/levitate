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

describe('Reports', () => {
  const monday = startOfWeekMonday(new Date())

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
  })

  it('renders configurable report graph and CSV downloads', () => {
    cy.visitAuthed('/reports')
    cy.contains('h1', 'Reports').should('be.visible')
    cy.get('input[type=date]').should('have.length', 2)
    cy.get('[data-report-group]').should('have.value', 'week')
    cy.get('[data-report-metric]').should('have.value', 'utilization_pct')
    cy.get('[data-report-chart-type]').should('have.value', 'bar')
    cy.get('[data-report-chart]').within(() => {
      cy.contains('Utilization % by week').should('be.visible')
      cy.get('[data-bar-chart]').should('be.visible')
      cy.get('[data-report-row]').should('have.length.at.least', 1)
    })
    cy.contains('button', 'Download utilization CSV').should('be.visible')
    cy.contains('button', 'Download assignments CSV').should('be.visible')
    cy.screenshot('reports-page')
  })

  it('lets users switch graph shape, grouping, metric, and row limit', () => {
    const stamp = Date.now()
    let personId = ''
    let personName = ''
    let projectId = ''
    let assignmentId = ''

    cy.apiRequest({
      method: 'POST',
      url: '/api/people',
      body: { name: `Reports Person ${stamp}`, email: '', role: '', weekly_capacity_hours: 40 },
    }).then((r) => {
      personId = r.body.id
      personName = r.body.name
    })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Reports Graph ${stamp}`, client: '', color: '#0EA5E9', notes: '' },
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
          hours_per_day: 6,
          notes: '',
        },
      }).then((r) => { assignmentId = r.body.id })
    })

    cy.then(() => {
      cy.visitAuthed('/reports')
      cy.get('[data-report-chart-type]').select('line')
      cy.get('[data-line-chart]').should('be.visible')

      cy.get('[data-report-group]').select('person')
      cy.get('[data-report-chart-type]').should('have.value', 'bar')
      cy.get('[data-report-metric]').select('assigned_hours')
      cy.get('[data-report-limit]').select('999')
      cy.get('[data-report-chart]').within(() => {
        cy.contains('Assigned hours by person').should('be.visible')
        cy.contains(personName).should('be.visible')
        cy.contains('30h').should('be.visible')
      })
    })

    cy.then(() => {
      if (assignmentId) cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${assignmentId}` })
      if (personId) cy.apiRequest({ method: 'POST', url: `/api/people/${personId}/archive` })
      if (projectId) cy.apiRequest({ method: 'POST', url: `/api/projects/${projectId}/archive` })
    })
  })

  it('utilization CSV endpoint returns text/csv', () => {
    cy.apiRequest({
      url: '/api/reports/utilization.csv?from=2026-05-04&to=2026-05-31',
    }).then((resp) => {
      expect(resp.status).to.eq(200)
      expect(resp.headers['content-type']).to.match(/text\/csv/)
      expect(resp.body as string).to.contain('person_name')
      expect(resp.body as string).to.contain('utilization_pct')
    })
  })

  it('assignments CSV endpoint returns text/csv', () => {
    cy.apiRequest({
      url: '/api/reports/assignments.csv?from=2026-05-01&to=2026-05-31',
    }).then((resp) => {
      expect(resp.status).to.eq(200)
      expect(resp.headers['content-type']).to.match(/text\/csv/)
      expect(resp.body as string).to.contain('assignment_id')
    })
  })
})

export {}
