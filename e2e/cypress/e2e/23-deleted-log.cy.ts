function ymd(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}
function startOfWeekMonday(d: Date) {
  const c = new Date(d); const dow = c.getDay()
  c.setDate(c.getDate() - ((dow + 6) % 7)); c.setHours(0, 0, 0, 0); return c
}
function addDays(d: Date, n: number) { const c = new Date(d); c.setDate(c.getDate() + n); return c }

describe('Delete log API (Float parity)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('returns the assignment UUID from GET /api/deleted/assignments after an admin deletes via the UI', () => {
    const monday = startOfWeekMonday(new Date())
    const start = ymd(addDays(monday, 7))
    const end = ymd(addDays(monday, 10))

    let personId = ''
    let projectId = ''
    let assignmentId = ''

    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Del Log Project ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
    }).then((r) => { projectId = r.body.id })

    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: start,
          end_date: end,
          hours_per_day: 6,
          notes: 'delete-log fixture',
        },
      }).then((r) => {
        expect(r.status).to.eq(201)
        assignmentId = r.body.id
      })
    })

    // The deleted log starts empty for this entity type.
    cy.then(() => {
      cy.apiRequest({ url: '/api/deleted/assignments' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.data).to.be.an('array').that.is.empty
        expect(res.headers['x-pagination-has-more']).to.eq('false')
      })
    })

    // Delete the assignment via the schedule grid's right-click → Delete UI.
    cy.intercept('DELETE', '**/api/assignments/*').as('deleteAssignment')
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${assignmentId}"]`).should('be.visible').then(($bar) => {
        const rect = $bar[0].getBoundingClientRect()
        cy.wrap($bar).trigger('contextmenu', {
          force: true,
          clientX: rect.left + 10,
          clientY: rect.top + rect.height / 2,
        })
      })
      cy.window().then((win) => {
        cy.stub(win, 'confirm').returns(true)
      })
      cy.get('[data-ctx-action="delete"]').click()
      cy.wait('@deleteAssignment').its('response.statusCode').should('eq', 204)
      cy.get(`[data-assignment-id="${assignmentId}"]`).should('not.exist')
    })

    // The delete log now contains the deleted UUID.
    cy.then(() => {
      cy.apiRequest({ url: '/api/deleted/assignments' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.headers['content-type']).to.match(/application\/json/)
        const ids = res.body.data.map((e: any) => e.id)
        expect(ids, 'deleted assignment IDs').to.include(assignmentId)
        const entry = res.body.data.find((e: any) => e.id === assignmentId)
        expect(entry, 'deleted-log entry').to.exist
        expect(entry.timestamp, 'deleted-log timestamp').to.match(/^\d{4}-\d{2}-\d{2}T/)
      })
    })

    // The other entity types remain empty.
    cy.apiRequest({ url: '/api/deleted/time-off' }).then((res) => {
      expect(res.status).to.eq(200)
      expect(res.body.data).to.be.an('array').that.is.empty
    })
    cy.apiRequest({ url: '/api/deleted/logged-time' }).then((res) => {
      expect(res.status).to.eq(200)
      expect(res.body.data).to.be.an('array').that.is.empty
    })
  })

  it('paginates with cursor + limit and surfaces X-Pagination-* headers', () => {
    // Create a few assignments and then delete them via the API to populate
    // the delete log, so we can exercise the pagination contract.
    let personId = ''
    let projectId = ''
    const created: string[] = []

    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Del Page Project ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
    }).then((r) => { projectId = r.body.id })

    const monday = startOfWeekMonday(new Date())
    cy.then(() => {
      for (let i = 0; i < 3; i++) {
        const day = ymd(addDays(monday, i + 1))
        cy.apiRequest({
          method: 'POST',
          url: '/api/assignments',
          body: {
            person_id: personId,
            project_id: projectId,
            start_date: day,
            end_date: day,
            hours_per_day: 4,
            notes: `pageable-${i}`,
          },
        }).then((r) => {
          created.push(r.body.id)
        })
      }
    })

    cy.then(() => {
      for (const id of created) {
        cy.apiRequest({ method: 'DELETE', url: `/api/assignments/${id}` })
          .its('status').should('eq', 204)
      }
    })

    // limit=2 → first page returns 2 entries, has_more=true, next cursor present.
    cy.then(() => {
      cy.apiRequest({ url: '/api/deleted/assignments?limit=2' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.data).to.have.length(2)
        expect(res.headers['x-pagination-has-more']).to.eq('true')
        const cursor = res.headers['x-pagination-next-cursor']
        expect(cursor, 'next cursor').to.be.a('string').and.not.empty

        cy.apiRequest({ url: `/api/deleted/assignments?limit=2&cursor=${encodeURIComponent(cursor)}` }).then((res2) => {
          expect(res2.status).to.eq(200)
          expect(res2.body.data).to.have.length(1)
          expect(res2.headers['x-pagination-has-more']).to.eq('false')
          // The IDs across both pages should equal the IDs we deleted.
          const all = [...res.body.data, ...res2.body.data].map((e: any) => e.id).sort()
          expect(all).to.deep.eq([...created].sort())
        })
      })
    })
  })

  it('requires admin scope (members get 403)', () => {
    cy.loginAs('member@example.com', 'member')
    cy.apiRequest({ url: '/api/me' })
    cy.apiRequest({ url: '/api/deleted/assignments' }).its('status').should('eq', 403)
    cy.apiRequest({ url: '/api/deleted/time-off' }).its('status').should('eq', 403)
    cy.apiRequest({ url: '/api/deleted/logged-time' }).its('status').should('eq', 403)
  })
})
