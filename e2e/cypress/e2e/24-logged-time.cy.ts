describe('Logged time (timesheets)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('supports CRUD and filtering against /api/logged-time', () => {
    let personId = ''
    let billableId = ''
    let nonBillableId = ''

    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })

    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: { name: `Billable ${Date.now()}`, client: '', color: '#0EA5E9', notes: '', billable: true },
      }).then((r) => {
        expect(r.status).to.eq(201)
        billableId = r.body.id
      })
      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: { name: `Internal ${Date.now()}`, client: '', color: '#0EA5E9', notes: '', billable: false },
      }).then((r) => {
        expect(r.status).to.eq(201)
        nonBillableId = r.body.id
      })
    })

    // Empty list to start.
    cy.then(() => {
      cy.apiRequest({ url: '/api/logged-time' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.be.an('array').that.is.empty
      })
    })

    // Create a billable entry — billable should be derived from the project,
    // not the request body. The endpoint must ignore any client-supplied
    // billable field (Float-compatible behaviour).
    let billableEntryId = ''
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: billableId,
          date: '2026-06-02',
          hours: 4.5,
          notes: 'design review',
          billable: false, // ignored
        },
      }).then((res) => {
        expect(res.status).to.eq(201)
        expect(res.body.billable).to.eq(true)
        expect(res.body.hours).to.eq(4.5)
        expect(res.body.project_id).to.eq(billableId)
        expect(res.body.person_id).to.eq(personId)
        expect(res.body.notes).to.eq('design review')
        billableEntryId = res.body.id
      })
    })

    // Non-billable project → derived non-billable entry.
    let nonBillableEntryId = ''
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: {
          person_id: personId,
          project_id: nonBillableId,
          date: '2026-06-03',
          hours: 2,
          notes: 'internal sync',
        },
      }).then((res) => {
        expect(res.status).to.eq(201)
        expect(res.body.billable).to.eq(false)
        nonBillableEntryId = res.body.id
      })
    })

    // GET /api/logged-time returns both.
    cy.then(() => {
      cy.apiRequest({ url: '/api/logged-time' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(2)
        const ids = res.body.map((e: any) => e.id).sort()
        expect(ids).to.deep.eq([billableEntryId, nonBillableEntryId].sort())
      })
    })

    // Filter by project_id.
    cy.then(() => {
      cy.apiRequest({ url: `/api/logged-time?project_id=${billableId}` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0].id).to.eq(billableEntryId)
      })
    })

    // Filter by date range — only 2026-06-03 falls in the window.
    cy.then(() => {
      cy.apiRequest({ url: '/api/logged-time?date_from=2026-06-03&date_to=2026-06-03' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0].id).to.eq(nonBillableEntryId)
      })
    })

    // PATCH updates hours, notes, and re-derives billable when project_id changes.
    cy.then(() => {
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/logged-time/${nonBillableEntryId}`,
        body: { hours: 3, notes: 'edited', project_id: billableId },
      }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.hours).to.eq(3)
        expect(res.body.notes).to.eq('edited')
        expect(res.body.project_id).to.eq(billableId)
        expect(res.body.billable).to.eq(true) // re-derived from new project
      })
    })

    // DELETE writes a deleted_log row exposed via /api/deleted/logged-time.
    cy.then(() => {
      cy.apiRequest({ method: 'DELETE', url: `/api/logged-time/${billableEntryId}` })
        .its('status').should('eq', 204)
    })
    cy.then(() => {
      cy.apiRequest({ url: '/api/deleted/logged-time' }).then((res) => {
        expect(res.status).to.eq(200)
        const ids = res.body.data.map((e: any) => e.id)
        expect(ids).to.include(billableEntryId)
      })
    })

    // Validation: hours > 24 rejected.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: { person_id: personId, project_id: billableId, date: '2026-06-04', hours: 25 },
        failOnStatusCode: false,
      }).its('status').should('eq', 422)
    })
  })

  it('requires admin scope for writes (members read-only)', () => {
    let personId = ''
    let projectId = ''
    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Member RO ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
    }).then((r) => { projectId = r.body.id })

    let entryId = ''
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: { person_id: personId, project_id: projectId, date: '2026-06-02', hours: 1, notes: '' },
      }).then((r) => { entryId = r.body.id })
    })

    // Switch to member — should be able to read but not write.
    cy.then(() => {
      cy.loginAs('member@example.com', 'member')
      cy.apiRequest({ url: '/api/me' })
    })
    cy.then(() => {
      cy.apiRequest({ url: '/api/logged-time' }).its('status').should('eq', 200)
      cy.apiRequest({
        method: 'POST',
        url: '/api/logged-time',
        body: { person_id: personId, project_id: projectId, date: '2026-06-05', hours: 1 },
        failOnStatusCode: false,
      }).its('status').should('eq', 403)
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/logged-time/${entryId}`,
        body: { hours: 2 },
        failOnStatusCode: false,
      }).its('status').should('eq', 403)
      cy.apiRequest({
        method: 'DELETE',
        url: `/api/logged-time/${entryId}`,
        failOnStatusCode: false,
      }).its('status').should('eq', 403)
    })
  })

  it('imports logged-time entries from a mocked Float /logged-time feed', () => {
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
        expect(res.body.logged_time_created).to.eq(1)
      })

      cy.apiRequest({ url: '/api/logged-time' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        const row = res.body[0]
        expect(row.date).to.eq('2026-06-02')
        expect(row.hours).to.eq(4)
        // Mock /logged-time row references project 301 ("Float Website"),
        // which is billable in the mock; verify the imported row inherited it.
        expect(row.billable).to.eq(true)
        expect(row.notes).to.eq('Mock logged time')
      })
    })
  })
})
