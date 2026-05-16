describe('Project code (Float parity)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('persists project_code on create, update, and clear via the API', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'API Coded Project',
        client: 'Acme',
        color: '#0EA5E9',
        billable: true,
        project_code: 'ACME-001',
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body).to.include({
        name: 'API Coded Project',
        project_code: 'ACME-001',
      })
      const id = res.body.id as string

      cy.apiRequest({ url: `/api/projects/${id}` }).then((get) => {
        expect(get.status).to.eq(200)
        expect(get.body.project_code).to.eq('ACME-001')
      })

      cy.apiRequest({ url: '/api/projects' }).then((list) => {
        expect(list.status).to.eq(200)
        const row = list.body.find((p: any) => p.id === id)
        expect(row).to.exist
        expect(row.project_code).to.eq('ACME-001')
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'API Coded Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          project_code: 'ACME-002',
        },
      }).then((patch) => {
        expect(patch.status).to.eq(200)
        expect(patch.body.project_code).to.eq('ACME-002')
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'API Coded Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          project_code: null,
        },
      }).then((cleared) => {
        expect(cleared.status).to.eq(200)
        expect(cleared.body.project_code).to.eq(null)
      })
    })
  })

  it('treats unset, missing, and empty project_code as null', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'No Code Project',
        client: '',
        color: '#0EA5E9',
        billable: true,
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.project_code).to.eq(null)
    })

    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Blank Code Project',
        client: '',
        color: '#0EA5E9',
        billable: true,
        project_code: '   ',
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.project_code).to.eq(null)
    })
  })

  it('rejects duplicate project_code with 409', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'First Coded',
        client: '',
        color: '#0EA5E9',
        billable: true,
        project_code: 'DUP-XYZ',
      },
    }).then((first) => {
      expect(first.status).to.eq(201)

      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: {
          name: 'Second Coded',
          client: '',
          color: '#0EA5E9',
          billable: true,
          project_code: 'DUP-XYZ',
        },
        failOnStatusCode: false,
      }).then((second) => {
        expect(second.status).to.eq(409)
        expect(second.body.detail).to.contain('project_code')
      })
    })
  })

  it('admin can set and clear project_code in the project form', () => {
    const stamp = Date.now()
    const name = `Code UI ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.get('[data-cy="project-code-input"]').type(`UI-${stamp}`)
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name).should('be.visible').within(() => {
      cy.get('[data-cy="project-code"]').should('contain.text', `UI-${stamp}`)
    })

    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="project-code-input"]').should('have.value', `UI-${stamp}`)
    cy.get('[data-cy="project-code-input"]').clear()
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="project-code"]').should('not.exist')
    })
  })

  it('imports project_code from a mocked Float API', () => {
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
        expect(website.project_code).to.eq('WEB-001')

        const internal = res.body.find((p: any) => p.name === 'Float Internal Tools')
        expect(internal).to.exist
        expect(internal.project_code).to.eq('INT-002')
      })
    })
  })
})
