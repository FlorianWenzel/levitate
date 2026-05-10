// Bar interactions: click-to-edit and right-edge resize. The whole-bar
// drag-to-move was intentionally removed — only the side handles change a
// bar's date range.

function ymd(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function startOfWeekMonday(d: Date) {
  const c = new Date(d)
  const dow = c.getDay()
  c.setDate(c.getDate() - ((dow + 6) % 7))
  c.setHours(0, 0, 0, 0)
  return c
}

function addDays(d: Date, n: number) {
  const c = new Date(d)
  c.setDate(c.getDate() + n)
  return c
}

function addWorkdays(d: Date, n: number) {
  const c = new Date(d); c.setHours(0, 0, 0, 0)
  if (n === 0) return c
  const step = n > 0 ? 1 : -1
  let left = Math.abs(n)
  while (left > 0) {
    c.setDate(c.getDate() + step)
    const dow = c.getDay()
    if (dow !== 0 && dow !== 6) left--
  }
  return c
}

describe('Schedule grid interactions', () => {
  const dayWidth = 36
  let personId: string
  let projectId: string
  let assignmentId: string
  let originalStart: string
  let originalEnd: string

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' }).then((r) => { personId = r.body.sub })
    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `DragTest ${Date.now()}`, client: '', color: '#10B981', notes: '' },
    }).then((r) => {
      projectId = r.body.id
      const monday = startOfWeekMonday(new Date())
      originalStart = ymd(addDays(monday, 7))
      originalEnd = ymd(addDays(monday, 11))
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: originalStart,
          end_date: originalEnd,
          hours_per_day: 8,
          notes: 'click/resize fixture',
        },
      }).then((r2) => { assignmentId = r2.body.id })
    })
  })

  it('clicking the bar body opens the edit modal pre-filled with current values', () => {
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).should('be.visible').click()
    cy.contains('h2', 'Edit assignment').should('be.visible')
    cy.contains('label', 'Notes').siblings('textarea').should('have.value', 'click/resize fixture')
    cy.contains('label', 'Start').siblings('input').should('have.value', originalStart)
    cy.contains('label', 'End').siblings('input').should('have.value', originalEnd)
    cy.contains('button', 'Cancel').click()
    cy.contains('h2', 'Edit assignment').should('not.exist')
  })

  it('right-edge resize extends the end date — verified via UI position and via API', () => {
    cy.intercept('PATCH', '**/api/assignments/*').as('patchAssignment')
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"] [data-resize="right"]`).should('exist').then(($handle) => {
      const rect = $handle[0].getBoundingClientRect()
      const startX = rect.left + 1
      const startY = rect.top + rect.height / 2
      const endX = startX + 3 * dayWidth + 8 // +3 days
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($handle).trigger('pointerdown', { ...opts, clientX: startX, clientY: startY })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: endX, clientY: startY })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: endX, clientY: startY })
    })
    cy.wait('@patchAssignment').its('response.statusCode').should('eq', 200)
    cy.apiRequest({ url: `/api/assignments/${assignmentId}` }).then((res) => {
      const origEnd = new Date(originalEnd + 'T00:00:00')
      const expectedEnd = ymd(addWorkdays(origEnd, 3)) // drag = +3 workday columns
      expect(res.body.end_date).to.eq(expectedEnd)
      // start_date is unchanged on a right-edge resize.
      expect(res.body.start_date).to.eq(originalStart)
    })
  })

  it('left-edge resize moves the start date earlier — start_date changes, end_date does not', () => {
    cy.intercept('PATCH', '**/api/assignments/*').as('patchAssignment')
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"] [data-resize="left"]`).should('exist').then(($handle) => {
      const rect = $handle[0].getBoundingClientRect()
      const x0 = rect.left + 1
      const y = rect.top + rect.height / 2
      const x1 = x0 - 2 * dayWidth - 8 // drag left by ~2 days
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($handle).trigger('pointerdown', { ...opts, clientX: x0, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: x1, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: x1, clientY: y })
    })
    cy.wait('@patchAssignment').its('response.statusCode').should('eq', 200)
    cy.apiRequest({ url: `/api/assignments/${assignmentId}` }).then((res) => {
      const origStartDate = new Date(originalStart + 'T00:00:00')
      const expectedStart = ymd(addWorkdays(origStartDate, -2)) // drag = -2 workday columns
      expect(res.body.start_date).to.eq(expectedStart)
      expect(res.body.end_date).to.eq(originalEnd)
    })
  })
})
