describe('Projects', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
  })

  it('admin can create and archive a project', () => {
    const stamp = Date.now()
    const name = `Test Project ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('h1', 'Projects').should('be.visible')

    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.contains('label', 'Client').siblings('input').first().type('Acme Co')
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).should('be.visible').within(() => {
      cy.contains('Acme Co')
      cy.contains('Active')
    })
    cy.screenshot('projects-after-create')

    cy.on('window:confirm', () => true)
    cy.contains('tr', name).contains('button', 'Archive').click()
    cy.contains('tr', name).should('not.exist')

    cy.contains('label', 'Show archived').click()
    cy.contains('tr', name).should('be.visible')
  })

  it('member cannot mutate', () => {
    cy.loginAs('member@example.com', 'member')
    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').should('not.exist')
  })
})
