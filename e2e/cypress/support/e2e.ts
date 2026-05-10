import './commands'

// Don't fail tests on app's manifest-route-rule warning or other unrelated console errors.
Cypress.on('uncaught:exception', () => false)

// Every test starts from a wiped database — TRUNCATE is fast (~5ms) and far
// safer than relying on per-spec cleanup ordering. OIDC-synced people are
// recreated on the next /api/me call.
beforeEach(() => {
  cy.resetState()
})
