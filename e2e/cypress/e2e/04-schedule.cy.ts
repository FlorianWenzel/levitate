describe('Schedule', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    // Recreate the OIDC user's person row after reset.
    cy.apiRequest({ url: '/api/me' })
    // Ensure an active project exists for the assignment dropdown.
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: 'Schedule Test Project', client: 'QA', color: '#0EA5E9', notes: '' },
    }).its('status').should('eq', 201)
  })

  it('renders the grid with week + day headers and person rows', () => {
    cy.visitAuthed('/schedule')
    cy.contains('h1', 'Schedule').should('be.visible')
    cy.contains('Week').should('be.visible')
    cy.contains('Person').should('be.visible')
    // At least one ISO week label (W followed by digits)
    cy.contains(/W\d{1,2}/).should('be.visible')
    // Auto-created admin person row should be there.
    cy.contains('Ada Admin').should('be.visible')
    cy.screenshot('schedule-grid')
  })

  it('Today button scrolls today back into view', () => {
    cy.visitAuthed('/schedule')
    // Initial scroll position puts "today" ~4 weeks from the left (720 px).
    cy.get('[data-schedule-grid]').should(($w) => {
      expect($w[0].scrollLeft).to.be.closeTo(720, 5)
    })
    // Scroll far away.
    cy.get('[data-schedule-grid]').then(($w) => { $w[0].scrollLeft = 2000 })
    cy.contains('button', 'Today').click()
    cy.get('[data-schedule-grid]').should(($w) => {
      expect($w[0].scrollLeft).to.be.closeTo(720, 5)
    })
  })

  it('admin can create an assignment via the modal and persist it via API', () => {
    cy.intercept('POST', '**/api/assignments').as('postAssignment')
    cy.visitAuthed('/schedule')
    const note = `cy-${Date.now()}`
    cy.contains('button', '+ Assignment').click()
    cy.contains('label', 'Notes').siblings('textarea').first().type(note)
    cy.contains('button', 'Save').click()
    cy.contains('h2', 'New assignment').should('not.exist')
    cy.wait('@postAssignment').then((i) => {
      expect(i.response?.statusCode).to.eq(201)
      const body = i.response?.body as any
      expect(body.notes).to.eq(note)
      // Verify it persisted on the server.
      cy.apiRequest({ url: `/api/assignments/${body.id}` }).its('status').should('eq', 200)
    })
    cy.screenshot('schedule-after-create')
  })

  it('admin can open + Time off modal', () => {
    cy.visitAuthed('/schedule')
    cy.contains('button', '+ Time off').click()
    cy.contains('h2', 'New time off').should('be.visible')
    cy.contains('label', 'Type').should('be.visible')
    cy.contains('button', 'Cancel').click()
    cy.contains('h2', 'New time off').should('not.exist')
  })

  it('member sees the grid but no admin buttons', () => {
    cy.loginAs('member@example.com', 'member')
    cy.apiRequest({ url: '/api/me' })
    cy.visitAuthed('/schedule')
    cy.contains('h1', 'Schedule').should('be.visible')
    cy.contains('button', '+ Assignment').should('not.exist')
    cy.contains('button', '+ Time off').should('not.exist')
  })
})
