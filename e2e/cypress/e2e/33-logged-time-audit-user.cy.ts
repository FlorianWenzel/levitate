const uuidRe = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

describe('Logged time created_by / modified_by fields (Float parity)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    // /api/me triggers the user + person upsert so the principal has a row in
    // the users table; without it the audit columns would be NULL on the very
    // first write (lookup-by-sub returns ErrNoRows).
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
        body: { name: `Audit-fields ${Date.now()}`, client: '', color: '#0EA5E9', notes: '', billable: true },
      }).then((r) => {
        expect(r.status).to.eq(201)
        projectId = r.body.id
      })
    })

    return cy.then(() => ({ personId, projectId }))
  }

  it('sets created_by and modified_by to the authenticated admin on POST', () => {
    seedEntry().then(({ personId, projectId }) => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: projectId,
          date: '2026-06-02',
          hours: 3,
          notes: 'initial',
        },
      }).then((r) => {
        expect(r.status).to.eq(201)
        expect(r.body).to.have.property('created_by')
        expect(r.body).to.have.property('modified_by')
        expect(r.body.created_by).to.match(uuidRe)
        expect(r.body.modified_by).to.match(uuidRe)
        // The same admin both created and "last modified" the brand-new row.
        expect(r.body.modified_by).to.eq(r.body.created_by)
      })
    })
  })

  it('GET and list responses expose the audit fields on every row', () => {
    seedEntry().then(({ personId, projectId }) => {
      let entryId = ''
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: { person_id: personId, project_id: projectId, date: '2026-06-02', hours: 1, notes: '' },
      }).then((r) => {
        entryId = r.body.id
      })

      cy.then(() => {
        cy.apiRequest({ url: `/api/logged-time/${entryId}` }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.created_by).to.match(uuidRe)
          expect(r.body.modified_by).to.match(uuidRe)
        })

        cy.apiRequest({ url: '/api/logged-time' }).then((r) => {
          expect(r.status).to.eq(200)
          for (const row of r.body as any[]) {
            expect(row).to.have.property('created_by')
            expect(row).to.have.property('modified_by')
            expect(row.created_by).to.match(uuidRe)
            expect(row.modified_by).to.match(uuidRe)
          }
        })
      })
    })
  })

  it('PATCH leaves created_by stable and refreshes modified_by from the authenticated user', () => {
    seedEntry().then(({ personId, projectId }) => {
      let entryId = ''
      let originalCreatedBy = ''

      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: { person_id: personId, project_id: projectId, date: '2026-06-02', hours: 2, notes: 'pre-patch' },
      }).then((r) => {
        entryId = r.body.id
        originalCreatedBy = r.body.created_by
        expect(originalCreatedBy).to.match(uuidRe)
      })

      cy.then(() => {
        cy.apiRequest({
          method: 'PATCH',
          url: `/api/logged-time/${entryId}`,
          body: { hours: 4, notes: 'edited' },
        }).then((r) => {
          expect(r.status).to.eq(200)
          // created_by is immutable across edits — it's a creation-time stamp.
          expect(r.body.created_by).to.eq(originalCreatedBy)
          // modified_by is refreshed from the authenticated principal; with
          // the same admin patching, the UUID happens to equal created_by.
          expect(r.body.modified_by).to.match(uuidRe)
          expect(r.body.modified_by).to.eq(originalCreatedBy)
        })
      })
    })
  })

  it('lock and unlock refresh modified_by but leave created_by unchanged', () => {
    seedEntry().then(({ personId, projectId }) => {
      let entryId = ''
      let originalCreatedBy = ''

      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: { person_id: personId, project_id: projectId, date: '2026-06-02', hours: 1, notes: '' },
      }).then((r) => {
        entryId = r.body.id
        originalCreatedBy = r.body.created_by
      })

      cy.then(() => {
        cy.apiRequest({ method: 'POST', url: `/api/logged-time/${entryId}/lock` }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.created_by).to.eq(originalCreatedBy)
          expect(r.body.modified_by).to.match(uuidRe)
        })

        cy.apiRequest({ method: 'POST', url: `/api/logged-time/${entryId}/unlock` }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.created_by).to.eq(originalCreatedBy)
          expect(r.body.modified_by).to.match(uuidRe)
        })
      })
    })
  })

  it('Float-imported rows are attributed to the admin who triggered the import', () => {
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
        for (const row of r.body as any[]) {
          // Importer (admin) is recorded — Float's own numeric created_by from
          // the mock feed is not propagated; we attribute to the local user.
          expect(row.created_by).to.match(uuidRe)
          expect(row.modified_by).to.match(uuidRe)
        }
      })
    })
  })
})
