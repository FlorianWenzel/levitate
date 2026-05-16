describe('User statuses (Home / Travel / Custom / Office)', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    cy.apiRequest({ url: '/api/me' })
  })

  it('supports CRUD against /api/statuses', () => {
    let personId = ''

    cy.apiRequest({ url: '/api/people' }).then((r) => {
      expect(r.status).to.eq(200)
      personId = (r.body as any[])[0].id
    })

    cy.then(() => {
      cy.apiRequest({ url: '/api/statuses' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.be.an('array').that.is.empty
      })
    })

    // Create a "Home" status (status_type_id = 1).
    let statusId = ''
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 1,
          people_id: personId,
          start_date: '2026-06-02',
          end_date: '2026-06-02',
        },
      }).then((res) => {
        expect(res.status).to.eq(201)
        expect(res.body).to.include({
          status_type_id: 1,
          people_id: personId,
          status_name: '',
          start_date: '2026-06-02',
          end_date: '2026-06-02',
          repeat_state: 0,
        })
        expect(res.body.repeat_end_date).to.be.null
        statusId = res.body.id
      })
    })

    // GET returns the new status.
    cy.then(() => {
      cy.apiRequest({ url: '/api/statuses' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0].id).to.eq(statusId)
        expect(res.body[0].status_type_id).to.eq(1)
      })
    })

    // GET by id.
    cy.then(() => {
      cy.apiRequest({ url: `/api/statuses/${statusId}` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.id).to.eq(statusId)
      })
    })

    // PATCH to set repeat_state to weekly (1) — requires a repeat_end_date.
    cy.then(() => {
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/statuses/${statusId}`,
        body: {
          repeat_state: 1,
          repeat_end_date: '2026-08-31',
        },
      }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.repeat_state).to.eq(1)
        expect(res.body.repeat_end_date).to.eq('2026-08-31')
      })
    })

    // Filtering: people_id and status_type_id narrowing.
    cy.then(() => {
      cy.apiRequest({ url: `/api/statuses?people_id=${personId}&status_type_id=1` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0].id).to.eq(statusId)
      })
    })

    cy.then(() => {
      cy.apiRequest({ url: `/api/statuses?status_type_id=4` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.be.an('array').that.is.empty
      })
    })

    // DELETE.
    cy.then(() => {
      cy.apiRequest({
        method: 'DELETE',
        url: `/api/statuses/${statusId}`,
      }).its('status').should('eq', 204)
    })

    cy.then(() => {
      cy.apiRequest({ url: `/api/statuses/${statusId}` }).its('status').should('eq', 404)
    })

    cy.then(() => {
      cy.apiRequest({ url: '/api/statuses' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.be.an('array').that.is.empty
      })
    })
  })

  it('requires status_name for custom statuses and validates repeat_state coupling', () => {
    let personId = ''
    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })

    // Custom (3) without status_name -> 422.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 3,
          people_id: personId,
          start_date: '2026-06-02',
          end_date: '2026-06-02',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })

    // Custom (3) with status_name succeeds and stores the label.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 3,
          people_id: personId,
          status_name: 'Conference',
          start_date: '2026-06-02',
          end_date: '2026-06-02',
        },
      }).then((res) => {
        expect(res.status).to.eq(201)
        expect(res.body.status_name).to.eq('Conference')
      })
    })

    // repeat_state > 0 without repeat_end_date -> 422.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 2,
          people_id: personId,
          start_date: '2026-06-02',
          end_date: '2026-06-02',
          repeat_state: 1,
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })

    // status_type_id out of range -> 422.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 9,
          people_id: personId,
          start_date: '2026-06-02',
          end_date: '2026-06-02',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })

    // end_date before start_date -> 422.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 1,
          people_id: personId,
          start_date: '2026-06-05',
          end_date: '2026-06-02',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })
  })

  it('blocks members from creating, updating, or deleting statuses', () => {
    let personId = ''
    let statusId = ''
    cy.apiRequest({ url: '/api/people' }).then((r) => {
      personId = (r.body as any[])[0].id
    })

    // Admin seeds a status.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 4,
          people_id: personId,
          start_date: '2026-06-02',
          end_date: '2026-06-02',
        },
      }).then((res) => {
        expect(res.status).to.eq(201)
        statusId = res.body.id
      })
    })

    // Member can read but not write.
    cy.then(() => {
      cy.loginAs('member@example.com', 'member')
      cy.apiRequest({ url: '/api/me' })
    })
    cy.then(() => {
      cy.apiRequest({ url: '/api/statuses' }).its('status').should('eq', 200)
    })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/statuses',
        body: {
          status_type_id: 1,
          people_id: personId,
          start_date: '2026-06-02',
          end_date: '2026-06-02',
        },
      }).its('status').should('eq', 403)
    })
    cy.then(() => {
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/statuses/${statusId}`,
        body: { repeat_state: 1, repeat_end_date: '2026-08-01' },
      }).its('status').should('eq', 403)
    })
    cy.then(() => {
      cy.apiRequest({
        method: 'DELETE',
        url: `/api/statuses/${statusId}`,
      }).its('status').should('eq', 403)
    })
  })

  it('serves the Float-parity /statuses payload from the mock API', () => {
    cy.task('startFloatMock').then((baseUrl) => {
      cy.apiRequest({
        url: `${baseUrl}/statuses`,
        headers: { Authorization: 'Bearer mock-float-token' },
      }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0]).to.include({
          status_id: 1001,
          status_type_id: 1,
          people_id: 101,
          start_date: '2026-06-02',
          end_date: '2026-06-02',
          repeat_state: 0,
        })
      })
    })
  })
})
