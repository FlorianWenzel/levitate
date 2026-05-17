describe('Float import', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('imports people, projects, allocations, and time off from a mocked Float API', () => {
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
          people_created: 1,
          projects_created: 2,
          assignments_created: 1,
          time_off_created: 1,
          logged_time_created: 1,
        })
      })

      cy.apiRequest({ url: '/api/people' }).then((res) => {
        expect(res.status).to.eq(200)
        const person = res.body.find((p: any) => p.email === 'alice.float@example.com')
        expect(person).to.exist
        expect(res.body).to.deep.include({
          id: person.id,
          name: 'Float Alice',
          email: 'alice.float@example.com',
          role: 'Designer',
          weekly_capacity_hours: 32,
          archived_at: null,
          created_at: person.created_at,
          updated_at: person.updated_at,
        })
      })

      cy.apiRequest({ url: '/api/projects' }).then((res) => {
        expect(res.status).to.eq(200)
        const project = res.body.find((p: any) => p.name === 'Float Website')
        expect(project).to.exist
        expect(res.body).to.deep.include({
          id: project.id,
          name: 'Float Website',
          client: 'Float Client',
          color: '#00AEEF',
          status: 'active',
          notes: 'Imported from mock Float',
          billable: true,
          budget_type: 2,
          budget_total: 25000,
          budget_priority: 0,
          tags: ['design', 'frontend'],
          archived_at: null,
          created_at: project.created_at,
          updated_at: project.updated_at,
        })
        const nonBillable = res.body.find((p: any) => p.name === 'Float Internal Tools')
        expect(nonBillable).to.exist
        expect(nonBillable.billable).to.eq(false)
      })

      cy.apiRequest({ url: '/api/assignments?from=2026-06-01&to=2026-06-07' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0]).to.include({
          start_date: '2026-06-02',
          end_date: '2026-06-03',
          hours_per_day: 6,
          notes: 'Design sprint — Mock allocation',
        })
      })

      cy.apiRequest({ url: '/api/time-off?from=2026-06-01&to=2026-06-07' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0]).to.include({
          start_date: '2026-06-04',
          end_date: '2026-06-04',
          type: 'vacation',
          notes: 'Vacation — Mock PTO',
        })
      })
    })
  })

  it('lets admins run the Float import from the UI', () => {
    cy.task('startFloatMock').then((baseUrl) => {
      cy.visitAuthed('/import')
      cy.contains('h1', 'Import from Float').should('be.visible')
      cy.contains('label', 'Float API token').siblings('input').first().type('mock-float-token')
      cy.contains('label', 'Schedule start').siblings('input').first().clear().type('2026-06-01')
      cy.contains('label', 'Schedule end').siblings('input').first().clear().type('2026-06-07')
      cy.contains('label', 'API base URL').siblings('input').first().clear().type(String(baseUrl))
      cy.contains('button', 'Import Float data').click()

      cy.contains('Imported 7 records').should('be.visible')
      cy.contains('tr', 'People').contains('1')
      cy.contains('tr', 'Projects').contains('2')
      cy.contains('tr', 'Assignments').contains('1')
      cy.contains('tr', 'Time off').contains('1')
      cy.contains('tr', 'Milestones').contains('1')
      cy.contains('tr', 'Logged time').contains('1')
    })
  })
})
