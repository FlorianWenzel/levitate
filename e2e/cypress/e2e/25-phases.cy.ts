describe('Project phases', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('exposes pagination headers on the phases list', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: 'Phase Header Project', client: '', color: '#888888', notes: '', billable: true },
    }).then((projectRes) => {
      expect(projectRes.status).to.eq(201)
      const projectID = projectRes.body.id

      cy.apiRequest({
        method: 'POST',
        url: `/api/projects/${projectID}/phases`,
        body: { name: 'Kickoff', start_date: '2026-07-01', end_date: '2026-07-15' },
      }).its('status').should('eq', 201)

      cy.apiRequest({ url: `/api/projects/${projectID}/phases` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.headers).to.have.property('x-pagination-total-count', '1')
        expect(res.headers).to.have.property('x-pagination-page-count', '1')
        expect(res.headers).to.have.property('x-pagination-current-page', '1')
        expect(res.headers).to.have.property('x-pagination-has-more', 'false')
      })
    })
  })

  it('imports phases from Float and links the milestone to the imported phase', () => {
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
          phases_created: 1,
          milestones_created: 1,
        })
      })

      cy.apiRequest({ url: '/api/projects' }).then((res) => {
        const project = res.body.find((p: any) => p.name === 'Float Website')
        expect(project, 'imported project').to.exist

        cy.apiRequest({ url: `/api/projects/${project.id}/phases` }).then((pRes) => {
          expect(pRes.status).to.eq(200)
          expect(pRes.body).to.have.length(1)
          const phase = pRes.body[0]
          expect(phase).to.include({
            name: 'Discovery',
            project_id: project.id,
            start_date: '2026-06-01',
            end_date: '2026-06-10',
            status: 2,
            active: 1,
            non_billable: false,
          })
          expect(phase.budget_total).to.eq(5000)
          expect(phase.default_hourly_rate).to.eq(100)

          cy.apiRequest({ url: `/api/projects/${project.id}/milestones` }).then((mRes) => {
            expect(mRes.status).to.eq(200)
            expect(mRes.body).to.have.length(1)
            expect(mRes.body[0]).to.include({
              name: 'Beta launch',
              phase_id: phase.id,
            })
          })

          cy.visitAuthed(`/projects/${project.id}`)
          cy.get('[data-cy="phases-section"]').within(() => {
            cy.get('[data-cy="phase-row-Discovery"]').should('be.visible').within(() => {
              cy.get('[data-cy="phase-name"]').should('contain.text', 'Discovery')
              cy.get('[data-cy="phase-start-date"]').should('contain.text', '2026-06-01')
              cy.get('[data-cy="phase-end-date"]').should('contain.text', '2026-06-10')
              cy.get('[data-cy="phase-status"]').should('contain.text', 'Confirmed')
            })
          })
        })
      })
    })
  })

  it('admin can create, edit and delete a phase for a project', () => {
    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    const projectName = `Phase Project ${Date.now()}`
    cy.contains('label', 'Name').siblings('input').first().type(projectName)
    cy.contains('button', 'Save').click()

    cy.get(`[data-cy="project-link-${projectName}"]`).click()
    cy.location('pathname').should('match', /^\/projects\//)

    cy.get('[data-cy="phases-empty"]').should('be.visible')

    cy.get('[data-cy="phase-create"]').click()
    cy.get('[data-cy="phase-form"]').should('be.visible')
    cy.get('[data-cy="phase-name-input"]').type('Discovery')
    cy.get('[data-cy="phase-start-date-input"]').type('2026-07-01')
    cy.get('[data-cy="phase-end-date-input"]').type('2026-07-10')
    cy.get('[data-cy="phase-budget-input"]').clear().type('1500')
    cy.get('[data-cy="phase-rate-input"]').clear().type('120')
    cy.get('[data-cy="phase-status-input"]').select('Tentative')
    cy.get('[data-cy="phase-save"]').click()

    cy.get('[data-cy="phase-row-Discovery"]').should('be.visible').within(() => {
      cy.get('[data-cy="phase-start-date"]').should('contain.text', '2026-07-01')
      cy.get('[data-cy="phase-end-date"]').should('contain.text', '2026-07-10')
      cy.get('[data-cy="phase-status"]').should('contain.text', 'Tentative')
    })

    cy.get('[data-cy="phase-edit-Discovery"]').click()
    cy.get('[data-cy="phase-name-input"]').clear().type('Build')
    cy.get('[data-cy="phase-non-billable-input"]').check()
    cy.get('[data-cy="phase-save"]').click()

    cy.get('[data-cy="phase-row-Build"]').should('be.visible')
    cy.get('[data-cy="phase-row-Discovery"]').should('not.exist')
    cy.contains('Non-billable').should('be.visible')

    cy.on('window:confirm', () => true)
    cy.get('[data-cy="phase-delete-Build"]').click()
    cy.get('[data-cy="phase-row-Build"]').should('not.exist')
    cy.get('[data-cy="phases-empty"]').should('be.visible')
  })

  it('member cannot create phases from the UI', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: 'Phase Read Only', client: '', color: '#888888', notes: '', billable: true },
    }).then((res) => {
      expect(res.status).to.eq(201)
      const projectID = res.body.id

      cy.loginAs('member@example.com', 'member')
      cy.visitAuthed(`/projects/${projectID}`)

      cy.get('[data-cy="phases-section"]').should('be.visible')
      cy.get('[data-cy="phase-create"]').should('not.exist')
    })
  })

  it('rejects phase creation when end_date precedes start_date', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: 'Validation Phase Project', client: '', color: '#888888', notes: '', billable: true },
    }).then((res) => {
      expect(res.status).to.eq(201)
      const projectID = res.body.id
      cy.apiRequest({
        method: 'POST',
        url: `/api/projects/${projectID}/phases`,
        body: { name: 'Bad Range', start_date: '2026-07-10', end_date: '2026-07-01' },
      }).then((bad) => {
        expect(bad.status).to.eq(422)
        expect(bad.body.title).to.eq('validation')
      })
    })
  })

  it('blocks members from POSTing a phase via the API', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: 'Member Phase API', client: '', color: '#888888', notes: '', billable: true },
    }).then((res) => {
      expect(res.status).to.eq(201)
      const projectID = res.body.id

      cy.loginAs('member@example.com', 'member')
      cy.apiRequest({
        method: 'POST',
        url: `/api/projects/${projectID}/phases`,
        body: { name: 'Forbidden Phase' },
      }).its('status').should('eq', 403)
    })
  })
})
