describe('API authorization', () => {
  const apiBase = Cypress.env('apiBase') as string

  it('public /healthz works without auth', () => {
    cy.request(`${apiBase}/healthz`).then((res) => {
      expect(res.status).to.eq(200)
      expect(res.body.status).to.eq('ok')
    })
  })

  it('/api/me requires Bearer token', () => {
    cy.request({ url: `${apiBase}/api/me`, failOnStatusCode: false }).then((res) => {
      expect(res.status).to.eq(401)
      expect(res.headers['content-type']).to.match(/problem\+json/)
    })
  })

  it('rejects malformed bearer', () => {
    cy.request({
      url: `${apiBase}/api/me`,
      headers: { Authorization: 'Bearer not-a-jwt' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401)
    })
  })

  it('rejects non-Bearer scheme', () => {
    cy.request({
      url: `${apiBase}/api/me`,
      headers: { Authorization: 'Basic dXNlcjpwYXNz' },
      failOnStatusCode: false,
    }).then((res) => {
      expect(res.status).to.eq(401)
    })
  })

  describe('member is read-only', () => {
    beforeEach(() => {
      cy.loginAs('member@example.com', 'member')
      cy.apiRequest({ url: '/api/me' }) // sync user/person
    })

    it('member can read /api/people', () => {
      cy.apiRequest({ url: '/api/people' }).then((res) => {
        expect(res.status).to.eq(200)
      })
    })

    it('member cannot POST /api/people', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/people',
        body: { name: 'Sneaky', email: '', role: '', weekly_capacity_hours: 40 },
      }).then((res) => {
        expect(res.status).to.eq(403)
        expect(res.body.title).to.eq('forbidden')
      })
    })

    it('member cannot POST /api/projects', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: { name: 'Sneaky', client: '', color: '#000', notes: '' },
      }).then((res) => {
        expect(res.status).to.eq(403)
      })
    })

    it('member cannot POST /api/assignments', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: '00000000-0000-0000-0000-000000000000',
          project_id: '00000000-0000-0000-0000-000000000000',
          start_date: '2026-06-01',
          end_date: '2026-06-05',
          hours_per_day: 8,
          notes: '',
        },
      }).then((res) => {
        expect(res.status).to.eq(403)
      })
    })

    it('member cannot DELETE /api/assignments/:id', () => {
      cy.apiRequest({
        method: 'DELETE',
        url: '/api/assignments/00000000-0000-0000-0000-000000000000',
      }).then((res) => {
        expect(res.status).to.eq(403)
      })
    })

    it('member cannot POST /api/time-off', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/time-off',
        body: {
          person_id: '00000000-0000-0000-0000-000000000000',
          start_date: '2026-06-01',
          end_date: '2026-06-05',
          type: 'vacation',
          notes: '',
        },
      }).then((res) => {
        expect(res.status).to.eq(403)
      })
    })

    it('member cannot archive a person', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/people/00000000-0000-0000-0000-000000000000/archive',
      }).then((res) => {
        expect(res.status).to.eq(403)
      })
    })
  })
})
