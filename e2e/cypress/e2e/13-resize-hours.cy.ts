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

describe('Schedule bar height = hours_per_day, drag-to-resize', () => {
  const PX_PER_HOUR = 8 // mirrors ScheduleGrid.vue (FULL_DAY_PX=64 / 8h)
  const FULL_DAY_PX = 64
  const MIN_BAR_HEIGHT = 16
  let personId = ''
  let projectId = ''
  let assignmentId = ''
  let assignmentIdSmall = ''

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `HoursTest ${Date.now()}`, client: '', color: '#8B5CF6', notes: '' },
    }).then((r) => {
      projectId = r.body.id
      const monday = startOfWeekMonday(new Date())
      const start = ymd(addDays(monday, 14)) // start of week 3 to keep clear of other fixtures
      const end = ymd(addDays(monday, 18))
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: { person_id: personId, project_id: projectId, start_date: start, end_date: end, hours_per_day: 4, notes: 'h-test 4h' },
      }).then((r2) => { assignmentId = r2.body.id })
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: { person_id: personId, project_id: projectId, start_date: start, end_date: end, hours_per_day: 1, notes: 'h-test 1h' },
      }).then((r2) => { assignmentIdSmall = r2.body.id })
    })
  })

  afterEach(() => {
    if (assignmentId) cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${assignmentId}` })
    if (assignmentIdSmall) cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${assignmentIdSmall}` })
    if (projectId) cy.apiRequest({ method: 'POST', url: `/api/projects/${projectId}/archive` })
  })

  it('bar height is proportional to hours_per_day', () => {
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).then(($big) => {
      cy.get(`[data-assignment-id="${assignmentIdSmall}"]`).then(($small) => {
        const bigH = $big[0].getBoundingClientRect().height
        const smallH = $small[0].getBoundingClientRect().height
        // 4 h * 8 = 32 px (half of FULL_DAY_PX); 1 h would be 8 but clamps to MIN_BAR_HEIGHT=16.
        expect(bigH).to.be.greaterThan(smallH)
        expect(bigH).to.be.closeTo(4 * PX_PER_HOUR, 1) // 32
        expect(bigH).to.be.lte(FULL_DAY_PX)
        expect(smallH).to.be.closeTo(MIN_BAR_HEIGHT, 1) // 16, clamped
      })
    })
    cy.screenshot('schedule-bar-heights')
  })

  it('top-edge drag up increases hours_per_day', () => {
    cy.intercept('PATCH', '**/api/assignments/*').as('patchAssignment')
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"] [data-resize="top"]`).should('exist').then(($handle) => {
      const rect = $handle[0].getBoundingClientRect()
      const x = rect.left + rect.width / 2
      const y0 = rect.top + 1
      const y1 = y0 - 3 * PX_PER_HOUR // up by exactly 3 hours (24 px / 8 = 3.0)
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($handle).trigger('pointerdown', { ...opts, clientX: x, clientY: y0 })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: x, clientY: y1 })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: x, clientY: y1 })
    })
    cy.wait('@patchAssignment').its('response.statusCode').should('eq', 200)
    cy.apiRequest({ url: `/api/assignments/${assignmentId}` }).then((res) => {
      expect(Number(res.body.hours_per_day)).to.eq(7) // 4 + 3
    })
  })

  it('top-edge drag down decreases hours_per_day, clamped at MIN_HOURS = 1', () => {
    cy.intercept('PATCH', '**/api/assignments/*').as('patchAssignment')
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"] [data-resize="top"]`).then(($handle) => {
      const rect = $handle[0].getBoundingClientRect()
      const x = rect.left + rect.width / 2
      const y0 = rect.top + 1
      const y1 = y0 + 200 // far below — should clamp to 1h not go negative
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($handle).trigger('pointerdown', { ...opts, clientX: x, clientY: y0 })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: x, clientY: y1 })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: x, clientY: y1 })
    })
    cy.wait('@patchAssignment').its('response.statusCode').should('eq', 200)
    cy.apiRequest({ url: `/api/assignments/${assignmentId}` }).then((res) => {
      expect(Number(res.body.hours_per_day)).to.eq(1)
    })
  })

  it('right-edge resize preserves hours_per_day', () => {
    cy.intercept('PATCH', '**/api/assignments/*').as('patchAssignment')
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"] [data-resize="right"]`).then(($handle) => {
      const rect = $handle[0].getBoundingClientRect()
      const x0 = rect.left + 1
      const y = rect.top + rect.height / 2
      const x1 = x0 + 36 + 4 // +1 day
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($handle).trigger('pointerdown', { ...opts, clientX: x0, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: x1, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: x1, clientY: y })
    })
    cy.wait('@patchAssignment').its('response.statusCode').should('eq', 200)
    cy.apiRequest({ url: `/api/assignments/${assignmentId}` }).then((res) => {
      expect(Number(res.body.hours_per_day)).to.eq(4) // unchanged by a date-edge resize
    })
  })
})
