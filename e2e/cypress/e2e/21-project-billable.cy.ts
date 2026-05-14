describe('Project billable flag', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
  })

  it('defaults new projects to billable and shows the billable badge', () => {
    const stamp = Date.now()
    const name = `Billable Default ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.get('[data-cy="billable-toggle"]').should('be.checked')
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).should('be.visible').within(() => {
      cy.get('[data-cy="billable-badge"]').should('be.visible').and('contain.text', 'Billable')
      cy.get('[data-cy="non-billable-badge"]').should('not.exist')
    })
  })

  it('admin can create a non-billable project and see the non-billable indicator', () => {
    const stamp = Date.now()
    const name = `Non Billable ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.contains('label', 'Client').siblings('input').first().type('Internal')
    cy.get('[data-cy="billable-toggle"]').uncheck()
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).should('be.visible').within(() => {
      cy.contains('Internal')
      cy.get('[data-cy="non-billable-badge"]').should('be.visible').and('contain.text', 'Non-billable')
      cy.get('[data-cy="billable-badge"]').should('not.exist')
    })
  })

  it('admin can toggle a billable project to non-billable via edit', () => {
    const stamp = Date.now()
    const name = `Toggle Billable ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="billable-badge"]').should('be.visible')
    })

    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="billable-toggle"]').should('be.checked').uncheck()
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="non-billable-badge"]').should('be.visible')
      cy.get('[data-cy="billable-badge"]').should('not.exist')
    })

    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="billable-toggle"]').should('not.be.checked').check()
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="billable-badge"]').should('be.visible')
      cy.get('[data-cy="non-billable-badge"]').should('not.exist')
    })
  })
})
