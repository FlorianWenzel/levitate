// Float Project API parity: `expand=expenses,project_tasks,project_team`.
// Each token populates the matching nested array on the project response
// using the schema documented at https://developer.float.com/swagger-api-v3.yaml.

describe('Project expansions (Float parity)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  function seedProjectWithAssignment(): Cypress.Chainable<{
    projectId: string
    personId: string
    assignmentId: string
  }> {
    const stamp = Date.now()
    const state: { projectId: string; personId: string; assignmentId: string } = {
      projectId: '',
      personId: '',
      assignmentId: '',
    }

    return cy
      .apiRequest({
        method: 'POST',
        url: '/api/people',
        body: {
          name: `Expansions Person ${stamp}`,
          email: `expansions-${stamp}@example.com`,
          role: 'Designer',
          weekly_capacity_hours: 40,
        },
      })
      .then((personRes) => {
        expect(personRes.status).to.eq(201)
        state.personId = personRes.body.id

        return cy.apiRequest({
          method: 'POST',
          url: '/api/projects',
          body: {
            name: `Expansions Project ${stamp}`,
            client: 'Acme',
            color: '#0EA5E9',
            notes: '',
            billable: true,
          },
        })
      })
      .then((projectRes) => {
        expect(projectRes.status).to.eq(201)
        state.projectId = projectRes.body.id

        return cy.apiRequest({
          method: 'POST',
          url: '/api/assignments',
          body: {
            person_id: state.personId,
            project_id: state.projectId,
            start_date: '2026-06-01',
            end_date: '2026-06-05',
            hours_per_day: 6,
            notes: 'Design sprint',
          },
        })
      })
      .then((assignmentRes) => {
        expect(assignmentRes.status).to.eq(201)
        state.assignmentId = assignmentRes.body.id
        return cy.wrap(state)
      })
  }

  it('omits expansion fields when ?expand= is absent', () => {
    seedProjectWithAssignment().then(({ projectId }) => {
      cy.apiRequest({ url: `/api/projects/${projectId}` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.not.have.property('expenses')
        expect(res.body).to.not.have.property('project_tasks')
        expect(res.body).to.not.have.property('project_team')
      })
    })
  })

  it('exposes Float-shaped arrays when ?expand= lists each field', () => {
    seedProjectWithAssignment().then(({ projectId, personId, assignmentId }) => {
      cy.apiRequest({
        url: `/api/projects/${projectId}?expand=expenses,project_tasks,project_team`,
      }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.id).to.eq(projectId)
        expect(res.body.expenses).to.deep.eq([])
        expect(res.body.project_tasks).to.have.length(1)
        expect(res.body.project_tasks[0]).to.deep.include({
          task_id: assignmentId,
          name: 'Design sprint',
          hours: 6,
          people_id: personId,
        })
        expect(res.body.project_team).to.have.length(1)
        expect(res.body.project_team[0].people_id).to.eq(personId)
        expect(res.body.project_team[0].hourly_rate).to.be.a('number')
      })

      cy.apiRequest({
        url: '/api/projects?expand=expenses,project_tasks,project_team',
      }).then((listRes) => {
        expect(listRes.status).to.eq(200)
        const row = (listRes.body as any[]).find((p) => p.id === projectId)
        expect(row).to.exist
        expect(row.expenses).to.deep.eq([])
        expect(row.project_tasks).to.have.length(1)
        expect(row.project_tasks[0].task_id).to.eq(assignmentId)
        expect(row.project_team).to.have.length(1)
        expect(row.project_team[0].people_id).to.eq(personId)
      })
    })
  })

  it('honors a partial expand= list', () => {
    seedProjectWithAssignment().then(({ projectId, assignmentId }) => {
      cy.apiRequest({ url: `/api/projects/${projectId}?expand=project_tasks` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.project_tasks).to.have.length(1)
        expect(res.body.project_tasks[0].task_id).to.eq(assignmentId)
        expect(res.body).to.not.have.property('expenses')
        expect(res.body).to.not.have.property('project_team')
      })
    })
  })

  it('renders team, tasks, and expenses sections on the project detail page', () => {
    seedProjectWithAssignment().then(({ projectId }) => {
      cy.visitAuthed(`/projects/${projectId}`)

      cy.get('[data-cy="project-team-section"]').should('be.visible').within(() => {
        cy.get('[data-cy="project-team-row"]').should('have.length', 1)
        cy.get('[data-cy="project-team-person"]').should('contain.text', 'Expansions Person')
        cy.get('[data-cy="project-team-rate"]').should('be.visible')
      })

      cy.get('[data-cy="project-tasks-section"]').should('be.visible').within(() => {
        cy.get('[data-cy="project-task-row"]').should('have.length', 1)
        cy.get('[data-cy="project-task-name"]').should('contain.text', 'Design sprint')
        cy.get('[data-cy="project-task-hours"]').should('contain.text', '6')
      })

      cy.get('[data-cy="project-expenses-section"]').should('be.visible').within(() => {
        cy.get('[data-cy="project-expenses-empty"]').should('exist')
      })
    })
  })
})
