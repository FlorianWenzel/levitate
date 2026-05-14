describe('Project milestones', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('imports milestones from Float linked to the matching project', () => {
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
        expect(res.body).to.include({
          milestones_created: 1,
        })
      })

      cy.apiRequest({ url: '/api/projects' }).then((res) => {
        expect(res.status).to.eq(200)
        const project = res.body.find((p: any) => p.name === 'Float Website')
        expect(project, 'imported project').to.exist

        cy.apiRequest({ url: `/api/projects/${project.id}/milestones` }).then((mRes) => {
          expect(mRes.status).to.eq(200)
          expect(mRes.body).to.have.length(1)
          expect(mRes.body[0]).to.include({
            name: 'Beta launch',
            date: '2026-06-05',
            project_id: project.id,
          })
        })

        cy.visitAuthed(`/projects/${project.id}`)
        cy.get('[data-cy="project-name"]').should('contain.text', 'Float Website')
        cy.get('[data-cy="milestones-section"]').within(() => {
          cy.get('[data-cy="milestone-row-Beta launch"]').should('be.visible').within(() => {
            cy.get('[data-cy="milestone-name"]').should('contain.text', 'Beta launch')
            cy.get('[data-cy="milestone-date"]').should('contain.text', '2026-06-05')
          })
        })
      })
    })
  })

  it('admin can create, edit and delete a milestone for a project', () => {
    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    const projectName = `Milestone Project ${Date.now()}`
    cy.contains('label', 'Name').siblings('input').first().type(projectName)
    cy.contains('button', 'Save').click()

    cy.get(`[data-cy="project-link-${projectName}"]`).click()
    cy.location('pathname').should('match', /^\/projects\//)

    cy.get('[data-cy="milestones-empty"]').should('be.visible')

    cy.get('[data-cy="milestone-create"]').click()
    cy.get('[data-cy="milestone-form"]').should('be.visible')
    cy.get('[data-cy="milestone-name-input"]').type('Kickoff')
    cy.get('[data-cy="milestone-date-input"]').type('2026-07-01')
    cy.get('[data-cy="milestone-end-date-input"]').type('2026-07-05')
    cy.get('[data-cy="milestone-save"]').click()

    cy.get('[data-cy="milestone-row-Kickoff"]').should('be.visible').within(() => {
      cy.get('[data-cy="milestone-date"]').should('contain.text', '2026-07-01')
      cy.get('[data-cy="milestone-end-date"]').should('contain.text', '2026-07-05')
    })

    cy.get('[data-cy="milestone-edit-Kickoff"]').click()
    cy.get('[data-cy="milestone-name-input"]').clear().type('Project kickoff')
    cy.get('[data-cy="milestone-date-input"]').clear().type('2026-07-02')
    cy.get('[data-cy="milestone-save"]').click()

    cy.get('[data-cy="milestone-row-Project kickoff"]').should('be.visible').within(() => {
      cy.get('[data-cy="milestone-date"]').should('contain.text', '2026-07-02')
    })
    cy.get('[data-cy="milestone-row-Kickoff"]').should('not.exist')

    cy.on('window:confirm', () => true)
    cy.get('[data-cy="milestone-delete-Project kickoff"]').click()
    cy.get('[data-cy="milestone-row-Project kickoff"]').should('not.exist')
    cy.get('[data-cy="milestones-empty"]').should('be.visible')
  })

  it('member cannot create milestones from the UI', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: 'Member Read Only', client: '', color: '#888888', notes: '', billable: true },
    }).then((res) => {
      expect(res.status).to.eq(201)
      const projectID = res.body.id

      cy.loginAs('member@example.com', 'member')
      cy.visitAuthed(`/projects/${projectID}`)

      cy.get('[data-cy="milestones-section"]').should('be.visible')
      cy.get('[data-cy="milestone-create"]').should('not.exist')
    })
  })
})
