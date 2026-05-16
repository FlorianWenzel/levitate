describe('Project default_hourly_rate (Float parity)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('accepts a string rate on create and exposes it on read', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Rate Project',
        client: 'Acme',
        color: '#0EA5E9',
        notes: '',
        billable: true,
        default_hourly_rate: '75.50',
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body).to.include({
        name: 'Rate Project',
        client: 'Acme',
        billable: true,
        default_hourly_rate: '75.500',
      })
      const projectId = res.body.id

      cy.apiRequest({ url: `/api/projects/${projectId}` }).then((getRes) => {
        expect(getRes.status).to.eq(200)
        expect(getRes.body.default_hourly_rate).to.eq('75.500')
      })

      cy.apiRequest({ url: '/api/projects' }).then((listRes) => {
        expect(listRes.status).to.eq(200)
        const found = listRes.body.find((p: any) => p.id === projectId)
        expect(found).to.exist
        expect(found.default_hourly_rate).to.eq('75.500')
      })
    })
  })

  it('updates the rate via PATCH and PUT (numeric and string forms)', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Update Rate Project',
        client: '',
        color: '#0EA5E9',
        notes: '',
        billable: true,
        default_hourly_rate: '50.000',
      },
    }).then((createRes) => {
      expect(createRes.status).to.eq(201)
      const projectId = createRes.body.id

      // PATCH with a numeric rate — server accepts numbers too.
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${projectId}`,
        body: {
          name: 'Update Rate Project',
          client: '',
          color: '#0EA5E9',
          notes: '',
          billable: true,
          default_hourly_rate: 99,
        },
      }).then((patchRes) => {
        expect(patchRes.status).to.eq(200)
        expect(patchRes.body.default_hourly_rate).to.eq('99.000')
      })

      // PUT with a string rate (Float's native form).
      cy.apiRequest({
        method: 'PUT',
        url: `/api/projects/${projectId}`,
        body: {
          name: 'Update Rate Project',
          client: '',
          color: '#0EA5E9',
          notes: '',
          billable: true,
          default_hourly_rate: '120.250',
        },
      }).then((putRes) => {
        expect(putRes.status).to.eq(200)
        expect(putRes.body.default_hourly_rate).to.eq('120.250')
      })

      // PATCH without the field preserves the existing rate.
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${projectId}`,
        body: {
          name: 'Update Rate Project',
          client: '',
          color: '#0EA5E9',
          notes: '',
          billable: true,
        },
      }).then((patchRes) => {
        expect(patchRes.status).to.eq(200)
        expect(patchRes.body.default_hourly_rate).to.eq('120.250')
      })
    })
  })

  it('defaults to 0.000 when not provided on create', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'No Rate Project',
        client: '',
        color: '#0EA5E9',
        notes: '',
        billable: true,
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.default_hourly_rate).to.eq('0.000')
    })
  })

  it('rejects a non-numeric rate with 422', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Bad Rate Project',
        client: '',
        color: '#0EA5E9',
        notes: '',
        billable: true,
        default_hourly_rate: 'not-a-number',
      },
    }).its('status').should('eq', 422)
  })

  it('rejects a negative rate with 422', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Negative Rate Project',
        client: '',
        color: '#0EA5E9',
        notes: '',
        billable: true,
        default_hourly_rate: -10,
      },
    }).its('status').should('eq', 422)
  })

  it('imports default_hourly_rate from Float', () => {
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
        const project = res.body.find((p: any) => p.name === 'Float Website')
        expect(project, 'imported project').to.exist
        expect(project.default_hourly_rate).to.eq('125.500')
      })
    })
  })

  it('lets an admin set the rate through the UI', () => {
    const stamp = Date.now()
    const name = `UI Rate Project ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.get('[data-cy="project-rate-input"]').clear().type('210.75')
    cy.contains('button', 'Save').click()

    cy.contains('tr', name).should('be.visible')

    cy.apiRequest({ url: '/api/projects' }).then((res) => {
      expect(res.status).to.eq(200)
      const project = res.body.find((p: any) => p.name === name)
      expect(project, 'created project').to.exist
      expect(project.default_hourly_rate).to.eq('210.750')
    })
  })
})
