describe('Project budget fields', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
  })

  it('persists budget type, total, and priority via the API', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'API Budget Project',
        client: 'Acme',
        color: '#0EA5E9',
        billable: true,
        budget_type: 2,
        budget_total: 1500.5,
        budget_priority: 1,
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body).to.include({
        name: 'API Budget Project',
        budget_type: 2,
        budget_total: 1500.5,
        budget_priority: 1,
      })
      const id = res.body.id as string

      cy.apiRequest({ url: `/api/projects/${id}` }).then((get) => {
        expect(get.status).to.eq(200)
        expect(get.body).to.include({
          budget_type: 2,
          budget_total: 1500.5,
          budget_priority: 1,
        })
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'API Budget Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          budget_type: 1,
          budget_total: 80,
          budget_priority: 2,
        },
      }).then((patch) => {
        expect(patch.status).to.eq(200)
        expect(patch.body).to.include({
          budget_type: 1,
          budget_total: 80,
          budget_priority: 2,
        })
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'API Budget Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          budget_type: null,
          budget_total: null,
          budget_priority: null,
        },
      }).then((cleared) => {
        expect(cleared.status).to.eq(200)
        expect(cleared.body.budget_type).to.eq(null)
        expect(cleared.body.budget_total).to.eq(null)
        expect(cleared.body.budget_priority).to.eq(null)
      })
    })
  })

  it('rejects out-of-range budget enum values with 422', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Bad Budget Project',
        client: '',
        color: '#0EA5E9',
        billable: true,
        budget_type: 9,
      },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(422)
      expect(res.body.detail).to.contain('budget_type')
    })

    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Bad Budget Project',
        client: '',
        color: '#0EA5E9',
        billable: true,
        budget_priority: 5,
      },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(422)
      expect(res.body.detail).to.contain('budget_priority')
    })
  })

  it('admin can set, edit, and clear budget fields in the project form', () => {
    const stamp = Date.now()
    const name = `Budget UI ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.get('[data-cy="project-budget-type-input"]').select('Total fee')
    cy.get('[data-cy="project-budget-total-input"]').clear().type('25000')
    cy.get('[data-cy="project-budget-priority-input"]').select('Phase')
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name)
      .should('be.visible')
      .within(() => {
        cy.get('[data-cy="project-budget-type"]').should('contain.text', 'Total fee')
        cy.get('[data-cy="project-budget-total"]').should('contain.text', '25000')
        cy.get('[data-cy="project-budget-priority"]').should('contain.text', 'Phase-level')
      })

    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="project-budget-type-input"]').should('have.value', '2')
    cy.get('[data-cy="project-budget-total-input"]').should('have.value', '25000')
    cy.get('[data-cy="project-budget-priority-input"]').should('have.value', '1')

    cy.get('[data-cy="project-budget-type-input"]').select('Total hours')
    cy.get('[data-cy="project-budget-total-input"]').clear().type('160')
    cy.get('[data-cy="project-budget-priority-input"]').select('Project')
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="project-budget-type"]').should('contain.text', 'Total hours')
      cy.get('[data-cy="project-budget-total"]').should('contain.text', '160 h')
      cy.get('[data-cy="project-budget-priority"]').should('contain.text', 'Project-level')
    })

    cy.contains('tr', name)
      .find('[data-cy^="project-link-"]')
      .click()
    cy.get('[data-cy="project-budget-summary"]').should('be.visible')
    cy.get('[data-cy="project-detail-budget-type"]').should('contain.text', 'Total hours')
    cy.get('[data-cy="project-detail-budget-total"]').should('contain.text', '160 h')
    cy.get('[data-cy="project-detail-budget-priority"]').should('contain.text', 'Project-level')

    cy.contains('a', '← Back to projects').click()
    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="project-budget-type-input"]').select('None')
    cy.get('[data-cy="project-budget-priority-input"]').select('None')
    cy.get('[data-cy="project-budget-total-input"]').clear()
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="project-budget-cell"]').should('contain.text', '—')
      cy.get('[data-cy="project-budget-type"]').should('not.exist')
    })
  })

  it('imports project budget fields from a mocked Float API', () => {
    cy.task('startFloatMock').then((baseUrl) => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/import/float',
        body: {
          api_token: 'mock-float-token',
          base_url: baseUrl,
          start_date: '2026-06-01',
          end_date: '2026-06-07',
        },
      }).then((res) => {
        expect(res.status).to.eq(200)
      })

      cy.apiRequest({ url: '/api/projects' }).then((res) => {
        expect(res.status).to.eq(200)
        const website = res.body.find((p: any) => p.name === 'Float Website')
        expect(website).to.exist
        expect(website.budget_type).to.eq(2)
        expect(website.budget_total).to.eq(25000)
        expect(website.budget_priority).to.eq(0)

        const internal = res.body.find((p: any) => p.name === 'Float Internal Tools')
        expect(internal).to.exist
        expect(internal.budget_type).to.eq(1)
        expect(internal.budget_total).to.eq(120)
        expect(internal.budget_priority).to.eq(1)
      })
    })
  })
})
