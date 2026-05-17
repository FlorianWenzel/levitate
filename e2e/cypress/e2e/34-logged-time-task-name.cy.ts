describe('Logged time task_name / task_meta_id fields (Float parity)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  function seedEntry() {
    let personId = ''
    let projectId = ''

    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })

    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: { name: `Task-fields ${Date.now()}`, client: '', color: '#0EA5E9', notes: '', billable: true },
      }).then((r) => {
        expect(r.status).to.eq(201)
        projectId = r.body.id
      })
    })

    return cy.then(() => ({ personId, projectId }))
  }

  it('persists task_name and task_meta_id supplied on POST and exposes them on GET', () => {
    seedEntry().then(({ personId, projectId }) => {
      let entryId = ''
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: projectId,
          date: '2026-06-02',
          hours: 3,
          notes: 'initial',
          task_name: 'Design sprint',
          task_meta_id: 'meta-401',
        },
      }).then((r) => {
        expect(r.status).to.eq(201)
        expect(r.body.task_name).to.eq('Design sprint')
        expect(r.body.task_meta_id).to.eq('meta-401')
        entryId = r.body.id
      })

      cy.then(() => {
        cy.apiRequest({ url: `/api/logged-time/${entryId}` }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.task_name).to.eq('Design sprint')
          expect(r.body.task_meta_id).to.eq('meta-401')
        })

        cy.apiRequest({ url: '/api/logged-time' }).then((r) => {
          expect(r.status).to.eq(200)
          for (const row of r.body as any[]) {
            expect(row).to.have.property('task_name')
            expect(row).to.have.property('task_meta_id')
          }
        })
      })
    })
  })

  it('defaults task_name and task_meta_id to null when omitted', () => {
    seedEntry().then(({ personId, projectId }) => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: projectId,
          date: '2026-06-02',
          hours: 1,
          notes: '',
        },
      }).then((r) => {
        expect(r.status).to.eq(201)
        expect(r.body.task_name).to.eq(null)
        expect(r.body.task_meta_id).to.eq(null)
      })
    })
  })

  it('PATCH can update task_name and task_meta_id, and clearing with empty string yields null', () => {
    seedEntry().then(({ personId, projectId }) => {
      let entryId = ''
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: projectId,
          date: '2026-06-02',
          hours: 2,
          notes: 'pre-patch',
          task_name: 'Original task',
          task_meta_id: 'meta-orig',
        },
      }).then((r) => {
        entryId = r.body.id
      })

      cy.then(() => {
        cy.apiRequest({
          method: 'PATCH',
          url: `/api/logged-time/${entryId}`,
          body: { task_name: 'Renamed task', task_meta_id: 'meta-updated' },
        }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.task_name).to.eq('Renamed task')
          expect(r.body.task_meta_id).to.eq('meta-updated')
        })

        // Unrelated PATCH leaves task fields intact.
        cy.apiRequest({
          method: 'PATCH',
          url: `/api/logged-time/${entryId}`,
          body: { hours: 5 },
        }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.task_name).to.eq('Renamed task')
          expect(r.body.task_meta_id).to.eq('meta-updated')
          expect(r.body.hours).to.eq(5)
        })

        // Empty string clears the fields.
        cy.apiRequest({
          method: 'PATCH',
          url: `/api/logged-time/${entryId}`,
          body: { task_name: '', task_meta_id: '' },
        }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.task_name).to.eq(null)
          expect(r.body.task_meta_id).to.eq(null)
        })
      })
    })
  })

  it('Float-imported logged-time rows carry task_name and task_meta_id from the Float feed', () => {
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
      }).its('status').should('eq', 200)

      cy.apiRequest({ url: '/api/logged-time' }).then((r) => {
        expect(r.status).to.eq(200)
        expect(r.body).to.have.length.greaterThan(0)
        const imported = (r.body as any[]).find((row) => row.task_name === 'Design sprint')
        expect(imported, 'imported entry with task_name=Design sprint').to.exist
        expect(imported.task_meta_id).to.eq('meta-401')
      })
    })
  })
})
