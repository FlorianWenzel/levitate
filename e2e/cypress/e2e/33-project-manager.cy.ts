describe('Project project_manager and all_pms_schedule fields', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
  })

  it('persists project_manager and all_pms_schedule via the API', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'PM Project',
        client: 'Acme',
        color: '#0EA5E9',
        billable: true,
        project_manager: 'Ada Lovelace',
        all_pms_schedule: true,
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body).to.include({
        name: 'PM Project',
        project_manager: 'Ada Lovelace',
        all_pms_schedule: true,
      })
      const id = res.body.id as string

      cy.apiRequest({ url: `/api/projects/${id}` }).then((get) => {
        expect(get.status).to.eq(200)
        expect(get.body).to.include({
          project_manager: 'Ada Lovelace',
          all_pms_schedule: true,
        })
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'PM Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          project_manager: 'Alan Turing',
          all_pms_schedule: false,
        },
      }).then((patch) => {
        expect(patch.status).to.eq(200)
        expect(patch.body).to.include({
          project_manager: 'Alan Turing',
          all_pms_schedule: false,
        })
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'PM Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          project_manager: null,
          all_pms_schedule: false,
        },
      }).then((cleared) => {
        expect(cleared.status).to.eq(200)
        expect(cleared.body.project_manager).to.eq(null)
        expect(cleared.body.all_pms_schedule).to.eq(false)
      })
    })
  })

  it('defaults project_manager to null and all_pms_schedule to false when omitted', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Default PM Project',
        client: '',
        color: '#0EA5E9',
        billable: true,
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.project_manager).to.eq(null)
      expect(res.body.all_pms_schedule).to.eq(false)
    })
  })

  it('admin can set and edit the fields in the project form', () => {
    const stamp = Date.now()
    const name = `PM UI ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.get('[data-cy="project-manager-input"]').type('Grace Hopper')
    cy.get('[data-cy="all-pms-schedule-toggle"]').check()
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name)
      .should('be.visible')
      .find('[data-cy^="project-link-"]')
      .click()

    cy.get('[data-cy="project-manager-summary"]').should('be.visible')
    cy.get('[data-cy="project-detail-project-manager"]').should('contain.text', 'Grace Hopper')
    cy.get('[data-cy="project-detail-all-pms-schedule"]').should('contain.text', 'Yes')

    cy.contains('a', '← Back to projects').click()
    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="project-manager-input"]').should('have.value', 'Grace Hopper')
    cy.get('[data-cy="all-pms-schedule-toggle"]').should('be.checked')

    cy.get('[data-cy="project-manager-input"]').clear().type('Margaret Hamilton')
    cy.get('[data-cy="all-pms-schedule-toggle"]').uncheck()
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name)
      .find('[data-cy^="project-link-"]')
      .click()
    cy.get('[data-cy="project-detail-project-manager"]').should('contain.text', 'Margaret Hamilton')
    cy.get('[data-cy="project-detail-all-pms-schedule"]').should('contain.text', 'No')
  })

  it('imports project_manager and all_pms_schedule from a mocked Float API', () => {
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
        expect(website.project_manager).to.eq('Mock PM Alice')
        expect(website.all_pms_schedule).to.eq(true)

        const internal = res.body.find((p: any) => p.name === 'Float Internal Tools')
        expect(internal).to.exist
        expect(internal.project_manager).to.eq(null)
        expect(internal.all_pms_schedule).to.eq(false)
      })
    })
  })
})
