describe('People', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    // Trigger the syncUser middleware so the seeded OIDC users exist as people.
    cy.apiRequest({ url: '/api/me' })
  })

  it('lists seeded OIDC users as auto-created people', () => {
    cy.visitAuthed('/people')
    cy.contains('h1', 'People').should('be.visible')
    cy.contains('Ada Admin').should('be.visible')
    cy.screenshot('people-list-with-auto-created')
  })

  it('admin can create, edit, archive a person', () => {
    const stamp = Date.now()
    const name = `Test Person ${stamp}`
    const renamed = `${name} (renamed)`

    cy.visitAuthed('/people')
    cy.contains('button', '+ New person').click()
    cy.get('input[type=email]').type(`p${stamp}@example.com`)
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.contains('label', 'Role').siblings('input').first().type('engineer')
    cy.contains('label', 'Weekly capacity (hours)').siblings('input').first().clear().type('32')
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).should('be.visible').within(() => {
      cy.contains('engineer')
      cy.contains('32')
      cy.contains('Active')
    })
    cy.screenshot('people-after-create')

    // Edit
    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.contains('label', 'Name').siblings('input').first().clear().type(renamed)
    cy.contains('button', 'Save').click()
    cy.contains('tr', renamed).should('be.visible')

    // Archive
    cy.on('window:confirm', () => true)
    cy.contains('tr', renamed).contains('button', 'Archive').click()
    cy.contains('tr', renamed).should('not.exist')

    // Toggle archived
    cy.contains('label', 'Show archived').click()
    cy.contains('tr', renamed).should('be.visible').within(() => {
      cy.contains('Archived')
    })
    cy.screenshot('people-with-archived')
  })

  it('member sees no admin actions', () => {
    cy.loginAs('member@example.com', 'member')
    cy.apiRequest({ url: '/api/me' })
    cy.visitAuthed('/people')
    cy.contains('button', '+ New person').should('not.exist')
    cy.get('table').within(() => {
      cy.contains('button', 'Edit').should('not.exist')
      cy.contains('button', 'Archive').should('not.exist')
    })
    cy.screenshot('people-as-member')
  })
})
