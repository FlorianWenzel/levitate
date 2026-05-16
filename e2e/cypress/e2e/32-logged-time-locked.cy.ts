describe('Logged time locked / locked_date fields (Float parity)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  function seedEntry() {
    let personId = ''
    let projectId = ''
    let entryId = ''

    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })

    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: { name: `Locked-fields ${Date.now()}`, client: '', color: '#0EA5E9', notes: '', billable: true },
      }).then((r) => {
        expect(r.status).to.eq(201)
        projectId = r.body.id
      })
    })

    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: projectId,
          date: '2026-06-02',
          hours: 3,
          notes: 'pre-lock',
        },
      }).then((r) => {
        expect(r.status).to.eq(201)
        // Newly created entries default to unlocked, locked_date null.
        expect(r.body.locked).to.eq(false)
        expect(r.body.locked_date).to.eq(null)
        entryId = r.body.id
      })
    })

    return cy.then(() => ({ personId, projectId, entryId }))
  }

  it('exposes `locked` and `locked_date` on GET/POST and ignores client-supplied lock state', () => {
    seedEntry().then(({ personId, projectId, entryId }) => {
      // POST cannot set locked via the body — server keeps it false.
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: projectId,
          date: '2026-06-03',
          hours: 2,
          notes: 'attempted-lock',
          locked: true, // ignored
          locked_date: '2025-01-01T00:00:00Z', // ignored
        } as any,
      }).then((r) => {
        expect(r.status).to.eq(201)
        expect(r.body.locked).to.eq(false)
        expect(r.body.locked_date).to.eq(null)
      })

      // GET surfaces the fields as well.
      cy.apiRequest({ url: `/api/logged-time/${entryId}` }).then((r) => {
        expect(r.status).to.eq(200)
        expect(r.body).to.have.property('locked', false)
        expect(r.body).to.have.property('locked_date', null)
      })

      // List returns the fields on every entry.
      cy.apiRequest({ url: '/api/logged-time' }).then((r) => {
        expect(r.status).to.eq(200)
        for (const row of r.body as any[]) {
          expect(row).to.have.property('locked')
          expect(row).to.have.property('locked_date')
        }
      })
    })
  })

  it('admin /lock sets locked=true and stamps locked_date; /unlock clears both', () => {
    seedEntry().then(({ entryId }) => {
      const before = new Date()

      cy.apiRequest({
        method: 'POST',
        url: `/api/logged-time/${entryId}/lock`,
      }).then((r) => {
        expect(r.status).to.eq(200)
        expect(r.body.locked).to.eq(true)
        expect(r.body.locked_date).to.not.eq(null)
        const stamped = new Date(r.body.locked_date)
        expect(stamped.getTime()).to.be.at.least(before.getTime() - 1000)
      })

      // Re-locking is idempotent and does not refresh locked_date.
      let firstStamp = ''
      cy.apiRequest({ url: `/api/logged-time/${entryId}` }).then((r) => {
        firstStamp = r.body.locked_date
      })
      cy.then(() => {
        cy.apiRequest({ method: 'POST', url: `/api/logged-time/${entryId}/lock` }).then((r) => {
          expect(r.status).to.eq(200)
          expect(r.body.locked).to.eq(true)
          expect(r.body.locked_date).to.eq(firstStamp)
        })
      })

      // Unlock clears both fields back to defaults.
      cy.apiRequest({
        method: 'POST',
        url: `/api/logged-time/${entryId}/unlock`,
      }).then((r) => {
        expect(r.status).to.eq(200)
        expect(r.body.locked).to.eq(false)
        expect(r.body.locked_date).to.eq(null)
      })
    })
  })

  it('PATCH on a locked entry returns 409 (Float treats locked entries as immutable)', () => {
    seedEntry().then(({ entryId }) => {
      cy.apiRequest({ method: 'POST', url: `/api/logged-time/${entryId}/lock` }).its('status').should('eq', 200)

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/logged-time/${entryId}`,
        body: { hours: 5 },
        failOnStatusCode: false,
      }).then((r) => {
        expect(r.status).to.eq(409)
        expect(r.body.title).to.eq('locked')
      })

      // After unlocking, PATCH works again.
      cy.apiRequest({ method: 'POST', url: `/api/logged-time/${entryId}/unlock` }).its('status').should('eq', 200)
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/logged-time/${entryId}`,
        body: { hours: 5 },
      }).then((r) => {
        expect(r.status).to.eq(200)
        expect(r.body.hours).to.eq(5)
      })
    })
  })

  it('members cannot lock or unlock (admin scope required)', () => {
    let entryId = ''
    seedEntry().then(({ entryId: id }) => {
      entryId = id
    })

    cy.then(() => {
      cy.loginAs('member@example.com', 'member')
      cy.apiRequest({ url: '/api/me' })
    })

    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: `/api/logged-time/${entryId}/lock`,
        failOnStatusCode: false,
      }).its('status').should('eq', 403)

      cy.apiRequest({
        method: 'POST',
        url: `/api/logged-time/${entryId}/unlock`,
        failOnStatusCode: false,
      }).its('status').should('eq', 403)

      // But reading the fields is allowed.
      cy.apiRequest({ url: `/api/logged-time/${entryId}` }).then((r) => {
        expect(r.status).to.eq(200)
        expect(r.body).to.have.property('locked')
        expect(r.body).to.have.property('locked_date')
      })
    })
  })

  it('Float import surfaces locked + locked_date fields on imported rows', () => {
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

      cy.apiRequest({ url: '/api/logged-time' }).then((res) => {
        expect(res.status).to.eq(200)
        for (const row of res.body as any[]) {
          // The mock /logged-time feed advertises locked=0; the imported
          // rows reflect that and surface both fields in the response.
          expect(row).to.have.property('locked', false)
          expect(row).to.have.property('locked_date', null)
        }
      })
    })
  })
})
