describe('Reports', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
  })

  it('renders date range inputs and download buttons', () => {
    cy.visitAuthed('/reports')
    cy.contains('h1', 'Reports').should('be.visible')
    cy.get('input[type=date]').should('have.length', 2)
    cy.contains('Utilization').should('be.visible')
    cy.contains('Assignments').should('be.visible')
    cy.contains('button', 'Download CSV').should('have.length.at.least', 1)
    cy.screenshot('reports-page')
  })

  it('utilization CSV endpoint returns text/csv', () => {
    cy.apiRequest({
      url: '/api/reports/utilization.csv?from=2026-05-04&to=2026-05-31',
    }).then((resp) => {
      expect(resp.status).to.eq(200)
      expect(resp.headers['content-type']).to.match(/text\/csv/)
      expect(resp.body as string).to.contain('person_name')
      expect(resp.body as string).to.contain('utilization_pct')
    })
  })

  it('assignments CSV endpoint returns text/csv', () => {
    cy.apiRequest({
      url: '/api/reports/assignments.csv?from=2026-05-01&to=2026-05-31',
    }).then((resp) => {
      expect(resp.status).to.eq(200)
      expect(resp.headers['content-type']).to.match(/text\/csv/)
      expect(resp.body as string).to.contain('assignment_id')
    })
  })
})
