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

describe('Schedule bar right-click context menu', () => {
  const dayWidth = 36
  const monday = startOfWeekMonday(new Date())
  let personId = ''
  let projectId = ''
  let assignmentId = ''
  const assignmentStart = ymd(addDays(monday, 7))   // day 7
  const assignmentEnd = ymd(addDays(monday, 12))    // day 12 (6 inclusive days)

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Ctx Project ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
    }).then((r) => { projectId = r.body.id })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: assignmentStart,
          end_date: assignmentEnd,
          hours_per_day: 6,
          notes: 'ctx-menu fixture',
        },
      }).then((r) => { assignmentId = r.body.id })
    })
  })

  it('right-click on a bar shows the menu near the cursor with Edit / Split / Delete', () => {
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).should('be.visible').then(($bar) => {
      const rect = $bar[0].getBoundingClientRect()
      cy.wrap($bar).trigger('contextmenu', {
        force: true,
        clientX: rect.left + 20,
        clientY: rect.top + rect.height / 2,
      })
    })
    cy.get('[data-bar-context-menu]').should('be.visible')
    cy.get('[data-ctx-action="edit"]').should('contain.text', 'Edit').and('not.be.disabled')
    cy.get('[data-ctx-action="split"]').should('contain.text', 'Split').and('not.be.disabled')
    cy.get('[data-ctx-action="delete"]').should('contain.text', 'Delete')
    cy.screenshot('schedule-ctx-menu')
  })

  it('clicking outside closes the menu', () => {
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).then(($bar) => {
      const rect = $bar[0].getBoundingClientRect()
      cy.wrap($bar).trigger('contextmenu', { force: true, clientX: rect.left + 10, clientY: rect.top + 5 })
    })
    cy.get('[data-bar-context-menu]').should('be.visible')
    cy.get('body').click(50, 50) // click far away
    cy.get('[data-bar-context-menu]').should('not.exist')
  })

  it('Edit opens the edit modal pre-filled', () => {
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).then(($bar) => {
      const rect = $bar[0].getBoundingClientRect()
      cy.wrap($bar).trigger('contextmenu', { force: true, clientX: rect.left + 10, clientY: rect.top + 5 })
    })
    cy.get('[data-ctx-action="edit"]').click()
    cy.get('[data-bar-context-menu]').should('not.exist')
    cy.contains('h2', 'Edit assignment').should('be.visible')
    cy.contains('label', 'Notes').siblings('textarea').should('have.value', 'ctx-menu fixture')
    cy.contains('button', 'Cancel').click()
  })

  it('Delete removes the assignment from the grid and from the API', () => {
    cy.intercept('DELETE', '**/api/assignments/*').as('deleteAssignment')
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).then(($bar) => {
      const rect = $bar[0].getBoundingClientRect()
      cy.wrap($bar).trigger('contextmenu', { force: true, clientX: rect.left + 10, clientY: rect.top + 5 })
    })
    cy.window().then((win) => {
      // Auto-confirm the native confirm() prompt.
      cy.stub(win, 'confirm').returns(true)
    })
    cy.get('[data-ctx-action="delete"]').click()
    cy.wait('@deleteAssignment').its('response.statusCode').should('eq', 204)
    cy.get(`[data-assignment-id="${assignmentId}"]`).should('not.exist')
    cy.apiRequest({ url: `/api/assignments/${assignmentId}` }).its('status').should('eq', 404)
  })

  it('Split divides the assignment at the clicked day into two halves', () => {
    cy.intercept('PATCH', '**/api/assignments/*').as('patchA')
    cy.intercept('POST', '**/api/assignments').as('postB')
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).then(($bar) => {
      const rect = $bar[0].getBoundingClientRect()
      // Right-click around the middle of the bar (rect spans 6 days from day 7→12).
      // Aim for day 9 — the third visible day inside the bar.
      const clickX = rect.left + 2 * dayWidth + dayWidth / 2 // day index 2 inside bar = day 9
      cy.wrap($bar).trigger('contextmenu', {
        force: true,
        clientX: clickX,
        clientY: rect.top + rect.height / 2,
      })
    })
    cy.get('[data-ctx-action="split"]').click()
    cy.get('[data-bar-context-menu]').should('not.exist')

    // PATCH on the original (now first half) and POST for the new second half.
    cy.wait('@patchA').then((i) => {
      const body = i.request.body as any
      expect(body.start_date).to.eq(assignmentStart) // day 7
      expect(body.end_date).to.eq(ymd(addDays(monday, 9))) // day 9 (clicked day = last day of first half)
      expect(body.hours_per_day).to.eq(6)
      expect(body.notes).to.eq('ctx-menu fixture')
    })
    let newId = ''
    cy.wait('@postB').then((i) => {
      expect(i.response?.statusCode).to.eq(201)
      const body = i.response?.body as any
      expect(body.start_date).to.eq(ymd(addDays(monday, 10))) // day 10
      expect(body.end_date).to.eq(assignmentEnd)             // day 12
      expect(body.hours_per_day).to.eq(6)
      newId = body.id
    })

    // Both bars now visible on the grid.
    cy.then(() => {
      cy.get(`[data-assignment-id="${assignmentId}"]`).should('be.visible')
      cy.get(`[data-assignment-id="${newId}"]`).should('be.visible')
    })

    // API double-check: total day-coverage matches the original (6 days = 3 + 3).
    cy.then(() => {
      cy.apiRequest({ url: `/api/assignments/${assignmentId}` }).then((res) => {
        expect(res.body.start_date).to.eq(assignmentStart)
        expect(res.body.end_date).to.eq(ymd(addDays(monday, 9)))
      })
      cy.apiRequest({ url: `/api/assignments/${newId}` }).then((res) => {
        expect(res.body.start_date).to.eq(ymd(addDays(monday, 10)))
        expect(res.body.end_date).to.eq(assignmentEnd)
      })
    })
    cy.screenshot('schedule-after-split')
  })

  it('Split is disabled on a single-day assignment', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/assignments',
      body: {
        person_id: personId,
        project_id: projectId,
        start_date: ymd(addDays(monday, 18)),
        end_date: ymd(addDays(monday, 18)),
        hours_per_day: 4,
        notes: 'single-day',
      },
    }).then((r) => {
      const id = r.body.id as string
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${id}"]`).then(($bar) => {
        const rect = $bar[0].getBoundingClientRect()
        cy.wrap($bar).trigger('contextmenu', { force: true, clientX: rect.left + 5, clientY: rect.top + 5 })
      })
      cy.get('[data-ctx-action="split"]').should('be.disabled')
    })
  })

  it('member (read-only) does not get a context menu when right-clicking', () => {
    cy.loginAs('member@example.com', 'member')
    cy.apiRequest({ url: '/api/me' })
    cy.visitAuthed('/schedule')
    cy.get(`[data-assignment-id="${assignmentId}"]`).then(($bar) => {
      const rect = $bar[0].getBoundingClientRect()
      cy.wrap($bar).trigger('contextmenu', { force: true, clientX: rect.left + 10, clientY: rect.top + 5 })
    })
    cy.get('[data-bar-context-menu]').should('not.exist')
  })
})
