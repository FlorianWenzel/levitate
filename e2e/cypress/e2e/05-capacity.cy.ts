describe('Capacity', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
  })

  it('renders weekly heatmap with the legend', () => {
    cy.visitAuthed('/capacity')
    cy.contains('h1', 'Capacity').should('be.visible')
    cy.contains('Person').should('be.visible')
    cy.contains('h/wk').should('be.visible')
    cy.contains(/W\d{1,2}/).should('be.visible')
    cy.contains('Over capacity').should('be.visible')
    cy.screenshot('capacity-heatmap')
  })

  it('range selector switches between 4/8/12 weeks', () => {
    cy.visitAuthed('/capacity')
    cy.get('select').first().select('4 weeks')
    cy.get('select').first().select('12 weeks')
  })
})
