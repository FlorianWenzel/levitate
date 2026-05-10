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
function addWorkdays(d: Date, n: number) {
  const c = new Date(d); c.setHours(0,0,0,0)
  if (n === 0) return c
  const step = n > 0 ? 1 : -1
  let left = Math.abs(n)
  while (left > 0) { c.setDate(c.getDate() + step); const dow = c.getDay(); if (dow !== 0 && dow !== 6) left-- }
  return c
}

describe('Schedule drag-to-create', () => {
  const dayWidth = 36
  const monday = startOfWeekMonday(new Date())
  // The grid preloads 20 past workdays, so today's Monday is at column 20.
  const TODAY_COL = 20

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
    // Make sure there's at least one active project to populate the modal dropdown.
    cy.apiRequest({ url: '/api/projects' }).then((r) => {
      const active = (r.body as any[]).filter((p) => !p.archived_at)
      if (active.length === 0) {
        cy.apiRequest({
          method: 'POST',
          url: '/api/projects',
          body: { name: `DragCreate Project ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
        })
      }
    })
  })

  it('dragging across days shows the ghost selection with date range and day count', () => {
    cy.visitAuthed('/schedule')
    cy.get('[data-schedule-grid]').should('be.visible')
    // Find the first track (empty space below the day-cell row of the first person row).
    cy.get('.relative.cursor-crosshair').first().then(($track) => {
      const rect = $track[0].getBoundingClientRect()
      // Start at day 7 (start of week 2), drag to day 11 (5 days inclusive).
      const startX = rect.left + (TODAY_COL + 7) * dayWidth + dayWidth / 2
      const endX = rect.left + (TODAY_COL + 11) * dayWidth + dayWidth / 2
      const y = rect.top + rect.height / 2
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($track).trigger('pointerdown', { ...opts, clientX: startX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: endX, clientY: y })
      cy.get('[data-selection]').should('exist').and('contain.text', '5d')
      cy.screenshot('schedule-drag-to-create-ghost')
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: endX, clientY: y })
    })
    // The modal should open with the dragged date range pre-filled.
    cy.contains('h2', 'New assignment').should('be.visible')
    const expectedStart = ymd(addWorkdays(monday, 7))
    const expectedEnd = ymd(addWorkdays(monday, 11))
    cy.contains('label', 'Start').siblings('input').should('have.value', expectedStart)
    cy.contains('label', 'End').siblings('input').should('have.value', expectedEnd)
    cy.contains('button', 'Cancel').click()
  })

  it('a plain click (no drag) opens the modal with a single-day range', () => {
    cy.visitAuthed('/schedule')
    cy.get('.relative.cursor-crosshair').first().then(($track) => {
      const rect = $track[0].getBoundingClientRect()
      const x = rect.left + (TODAY_COL + 3) * dayWidth + dayWidth / 2 // day 3
      const y = rect.top + rect.height / 2
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($track).trigger('pointerdown', { ...opts, clientX: x, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: x, clientY: y })
    })
    cy.contains('h2', 'New assignment').should('be.visible')
    const day3 = ymd(addWorkdays(monday, 3))
    cy.contains('label', 'Start').siblings('input').should('have.value', day3)
    cy.contains('label', 'End').siblings('input').should('have.value', day3)
    cy.contains('button', 'Cancel').click()
  })

  it('dragging in reverse (right-to-left) still picks up the correct range', () => {
    cy.visitAuthed('/schedule')
    cy.get('.relative.cursor-crosshair').first().then(($track) => {
      const rect = $track[0].getBoundingClientRect()
      const startX = rect.left + (TODAY_COL + 14) * dayWidth + dayWidth / 2 // day 14
      const endX = rect.left + (TODAY_COL + 10) * dayWidth + dayWidth / 2   // day 10
      const y = rect.top + rect.height / 2
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($track).trigger('pointerdown', { ...opts, clientX: startX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: endX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: endX, clientY: y })
    })
    const expectedStart = ymd(addWorkdays(monday, 10))
    const expectedEnd = ymd(addWorkdays(monday, 14))
    cy.contains('h2', 'New assignment').should('be.visible')
    cy.contains('label', 'Start').siblings('input').should('have.value', expectedStart)
    cy.contains('label', 'End').siblings('input').should('have.value', expectedEnd)
    cy.contains('button', 'Cancel').click()
  })

  it('drag-to-create then save persists the assignment via API', () => {
    cy.intercept('POST', '**/api/assignments').as('postAssignment')
    cy.visitAuthed('/schedule')
    cy.get('.relative.cursor-crosshair').first().then(($track) => {
      const rect = $track[0].getBoundingClientRect()
      const startX = rect.left + (TODAY_COL + 8) * dayWidth + dayWidth / 2
      const endX = rect.left + (TODAY_COL + 10) * dayWidth + dayWidth / 2
      const y = rect.top + rect.height / 2
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($track).trigger('pointerdown', { ...opts, clientX: startX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: endX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: endX, clientY: y })
    })
    cy.contains('h2', 'New assignment').should('be.visible')
    cy.contains('label', 'Notes').siblings('textarea').type('drag-to-create fixture')
    cy.contains('button', 'Save').click()
    cy.wait('@postAssignment').then((i) => {
      expect(i.response?.statusCode).to.eq(201)
      const body = i.response?.body as any
      expect(body.start_date).to.eq(ymd(addWorkdays(monday, 8)))
      expect(body.end_date).to.eq(ymd(addWorkdays(monday, 10)))
      expect(body.notes).to.eq('drag-to-create fixture')
      cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${body.id}` })
    })
  })
})
