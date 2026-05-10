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
function addWorkdays(d: Date, n: number) {
  const c = new Date(d); c.setHours(0,0,0,0)
  if (n === 0) return c
  const step = n > 0 ? 1 : -1
  let left = Math.abs(n)
  while (left > 0) { c.setDate(c.getDate() + step); const dow = c.getDay(); if (dow !== 0 && dow !== 6) left-- }
  return c
}

describe('Time-off blocker: rendering and auto-clip', () => {
  const dayWidth = 36
  const monday = startOfWeekMonday(new Date())
  let personId = ''
  let projectId = ''
  let timeOffId = ''
  let assignmentIds: string[] = []
  // Time-off occupies workday columns 10..12 (Mon..Wed of week 3).
  const TO_FIRST_COL = 10
  const TO_LAST_COL = 12
  // Schedule preloads 20 past workdays, so today's Monday sits at grid column 20.
  const TODAY_COL = 20

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
    cy.apiRequest({
      method: 'POST',
      url: '/api/people',
      body: { name: `Clip Test ${Date.now()}`, email: '', role: '', weekly_capacity_hours: 40 },
    }).then((r) => { personId = r.body.id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Clip Project ${Date.now()}`, client: '', color: '#10B981', notes: '' },
    }).then((r) => { projectId = r.body.id })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/time-off',
        body: {
          person_id: personId,
          start_date: ymd(addWorkdays(monday, TO_FIRST_COL)),
          end_date: ymd(addWorkdays(monday, TO_LAST_COL)),
          type: 'vacation',
          notes: 'clip fixture',
        },
      }).then((r) => { timeOffId = r.body.id })
    })
  })

  afterEach(() => {
    for (const id of assignmentIds) {
      cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${id}` })
    }
    assignmentIds = []
    if (timeOffId) cy.apiRequest({ method: 'DELETE', url: `/api/time-off/${timeOffId}` })
    if (personId) cy.apiRequest({ method: 'POST', url: `/api/people/${personId}/archive` })
    if (projectId) cy.apiRequest({ method: 'POST', url: `/api/projects/${projectId}/archive` })
  })

  it('renders time-off as a full-row striped block with the type label', () => {
    cy.visitAuthed('/schedule')
    cy.get(`[data-time-off-id="${timeOffId}"]`).should('be.visible').then(($el) => {
      const rect = $el[0].getBoundingClientRect()
      // Spans 3 workday columns inclusive.
      expect(rect.width).to.be.closeTo(3 * dayWidth, 2)
      expect(rect.height).to.be.gte(30)
      const bg = window.getComputedStyle($el[0]).backgroundImage
      expect(bg).to.contain('repeating-linear-gradient')
    })
    cy.get(`[data-time-off-id="${timeOffId}"]`).should('contain.text', 'vacation')
    cy.screenshot('schedule-time-off-blocker')
  })

  it('drag-to-create across time-off clips to the side anchored at pointerdown', () => {
    cy.intercept('POST', '**/api/assignments').as('postAssignment')
    cy.visitAuthed('/schedule')
    cy.get(`[data-time-off-id="${timeOffId}"]`).should('be.visible')
    cy.get(`[data-person-id="${personId}"] .relative.cursor-crosshair`).then(($track) => {
      const rect = $track[0].getBoundingClientRect()
      // Anchor at col 7 (Wed week 2, free), drag forward through time-off (cols 10..12) to col 14.
      const startX = rect.left + (TODAY_COL + 7) * dayWidth + dayWidth / 2
      const endX = rect.left + (TODAY_COL + 14) * dayWidth + dayWidth / 2
      const y = rect.top + rect.height / 2
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($track).trigger('pointerdown', { ...opts, clientX: startX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: endX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: endX, clientY: y })
    })
    cy.contains('h2', 'New assignment').should('be.visible')
    cy.contains('label', 'Start').siblings('input').should('have.value', ymd(addWorkdays(monday, 7)))
    cy.contains('label', 'End').siblings('input').should('have.value', ymd(addWorkdays(monday, TO_FIRST_COL - 1))) // col 9
    cy.contains('button', 'Save').click()
    cy.wait('@postAssignment').then((i) => {
      expect(i.response?.statusCode).to.eq(201)
      const body = i.response?.body as any
      expect(body.start_date).to.eq(ymd(addWorkdays(monday, 7)))
      expect(body.end_date).to.eq(ymd(addWorkdays(monday, TO_FIRST_COL - 1)))
      assignmentIds.push(body.id)
    })
  })

  it('drag-to-create starting INSIDE time-off is rejected silently', () => {
    cy.visitAuthed('/schedule')
    cy.get(`[data-time-off-id="${timeOffId}"]`).should('be.visible')
    cy.get(`[data-person-id="${personId}"] .relative.cursor-crosshair`).then(($track) => {
      const rect = $track[0].getBoundingClientRect()
      // Anchor at col 11 — squarely inside time-off (cols 10..12).
      const startX = rect.left + (TODAY_COL + 11) * dayWidth + dayWidth / 2
      const endX = rect.left + (TODAY_COL + 16) * dayWidth + dayWidth / 2
      const y = rect.top + rect.height / 2
      const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
      cy.wrap($track).trigger('pointerdown', { ...opts, clientX: startX, clientY: y })
      cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: endX, clientY: y })
      cy.get('[data-selection]').should('not.exist')
      cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: endX, clientY: y })
    })
    cy.contains('h2', 'New assignment').should('not.exist')
  })

  it('right-edge resize that crosses time-off clips at the boundary', () => {
    cy.intercept('PATCH', '**/api/assignments/*').as('patchAssignment')
    cy.apiRequest({
      method: 'POST',
      url: '/api/assignments',
      body: {
        person_id: personId,
        project_id: projectId,
        // Days 7..8 (Wed-Thu week 2), well before time-off at cols 10..12.
        start_date: ymd(addWorkdays(monday, 7)),
        end_date: ymd(addWorkdays(monday, 8)),
        hours_per_day: 4,
        notes: 'clip-resize-fixture',
      },
    }).then((r) => {
      const id = r.body.id as string
      assignmentIds.push(id)
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${id}"] [data-resize="right"]`).then(($handle) => {
        const rect = $handle[0].getBoundingClientRect()
        const x0 = rect.left + 1
        const y = rect.top + rect.height / 2
        const x1 = x0 + 6 * dayWidth + 4 // try to extend +6 workdays, time-off should clip it
        const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
        cy.wrap($handle).trigger('pointerdown', { ...opts, clientX: x0, clientY: y })
        cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: x1, clientY: y })
        cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: x1, clientY: y })
      })
      cy.wait('@patchAssignment').its('response.statusCode').should('eq', 200)
      cy.apiRequest({ url: `/api/assignments/${id}` }).then((res) => {
        // start unchanged at col 7, end clipped at col 9 (last free workday before time-off at col 10).
        expect(res.body.start_date).to.eq(ymd(addWorkdays(monday, 7)))
        expect(res.body.end_date).to.eq(ymd(addWorkdays(monday, TO_FIRST_COL - 1)))
      })
    })
  })

  it('right-edge resize fully through time-off (anchor blocked) does not change dates', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/assignments',
      body: {
        person_id: personId,
        project_id: projectId,
        // Bar starts inside time-off so resize-right anchor is blocked.
        start_date: ymd(addWorkdays(monday, TO_FIRST_COL)),     // col 10 (inside time-off)
        end_date: ymd(addWorkdays(monday, TO_FIRST_COL + 1)),   // col 11
        hours_per_day: 4,
        notes: 'resize-blocked-fixture',
      },
    }).then((r) => {
      const id = r.body.id as string
      assignmentIds.push(id)
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${id}"] [data-resize="right"]`).then(($handle) => {
        const rect = $handle[0].getBoundingClientRect()
        const x0 = rect.left + 1
        const y = rect.top + rect.height / 2
        const x1 = x0 + 5 * dayWidth
        const opts = { eventConstructor: 'PointerEvent', button: 0, pointerId: 1, isPrimary: true, force: true }
        cy.wrap($handle).trigger('pointerdown', { ...opts, clientX: x0, clientY: y })
        cy.get('[data-schedule-grid]').trigger('pointermove', { ...opts, clientX: x1, clientY: y })
        cy.get('[data-schedule-grid]').trigger('pointerup', { ...opts, clientX: x1, clientY: y })
      })
      cy.wait(300)
      cy.apiRequest({ url: `/api/assignments/${id}` }).then((res) => {
        expect(res.body.start_date).to.eq(ymd(addWorkdays(monday, TO_FIRST_COL)))
        expect(res.body.end_date).to.eq(ymd(addWorkdays(monday, TO_FIRST_COL + 1)))
      })
    })
  })
})
