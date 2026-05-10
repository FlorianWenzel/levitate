describe('Auth', () => {
  it('redirects unauthenticated visit to /login', () => {
    cy.visit('/')
    cy.location('pathname', { timeout: 10000 }).should('eq', '/login')
    cy.contains('Sign in with Keycloak').should('be.visible')
    cy.screenshot('auth-login-page')
  })

  it('redirects unauthenticated /people to /login', () => {
    cy.visit('/people')
    cy.location('pathname', { timeout: 10000 }).should('eq', '/login')
  })

  it('signs in as admin and lands on dashboard', () => {
    cy.loginAs('admin@example.com', 'admin')
    cy.visitAuthed('/')
    cy.location('pathname').should('eq', '/')
    cy.contains('h1', /Hello/i).should('be.visible')
    cy.contains('admin@example.com').should('be.visible')
    cy.contains('a', 'Dashboard').should('have.attr', 'href', '/')
    cy.contains('a', 'People').should('have.attr', 'href', '/people')
    cy.contains('a', 'Projects').should('have.attr', 'href', '/projects')
    cy.contains('a', 'Schedule').should('have.attr', 'href', '/schedule')
    cy.contains('a', 'Capacity').should('have.attr', 'href', '/capacity')
    cy.contains('a', 'Reports').should('have.attr', 'href', '/reports')
    cy.screenshot('auth-dashboard-admin')
  })

  it('Ping backend button shows ok', () => {
    cy.loginAs('admin@example.com', 'admin')
    cy.visitAuthed('/')
    cy.contains('button', 'Ping backend').click()
    cy.contains('/healthz → ').should('be.visible')
    cy.contains('ok').should('be.visible')
  })
})
