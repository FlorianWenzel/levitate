describe('Validation', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
  })

  describe('API: people', () => {
    it('rejects empty name with 422', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/people',
        body: { name: '', email: 'x@y.com', role: 'eng', weekly_capacity_hours: 40 },
      }).then((res) => {
        expect(res.status).to.eq(422)
        expect(res.body.title).to.eq('validation')
        expect(res.body.detail).to.contain('name')
      })
    })

    it('rejects negative capacity', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/people',
        body: { name: 'X', email: '', role: '', weekly_capacity_hours: -5 },
      }).then((res) => {
        expect(res.status).to.eq(422)
        expect(res.body.detail).to.contain('weekly_capacity_hours')
      })
    })

    it('rejects capacity > 168', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/people',
        body: { name: 'X', email: '', role: '', weekly_capacity_hours: 200 },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })

    it('returns 404 on bogus id update', () => {
      cy.apiRequest({
        method: 'PATCH',
        url: '/api/people/00000000-0000-0000-0000-000000000000',
        body: { name: 'X', email: '', role: '', weekly_capacity_hours: 40 },
      }).then((res) => {
        expect(res.status).to.eq(404)
      })
    })

    it('returns 400 on malformed id', () => {
      cy.apiRequest({ url: '/api/people/not-a-uuid' }).then((res) => {
        expect(res.status).to.eq(400)
      })
    })
  })

  describe('API: projects', () => {
    it('rejects empty name', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: { name: '', client: '', color: '', notes: '' },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })
  })

  describe('API: assignments', () => {
    let personId: string
    let projectId: string
    before(() => {
      cy.loginAs('admin@example.com', 'admin')
      cy.apiRequest({ url: '/api/people' }).then((r) => {
        personId = (r.body as any[])[0].id
      })
      cy.apiRequest({
        method: 'POST',
        url: '/api/projects',
        body: { name: `Validation Project ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
      }).then((r) => {
        projectId = r.body.id
      })
    })

    it('rejects end_date before start_date', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: '2026-06-10',
          end_date: '2026-06-01',
          hours_per_day: 8,
          notes: '',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })

    it('rejects hours_per_day = 0', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: '2026-06-01',
          end_date: '2026-06-05',
          hours_per_day: 0,
          notes: '',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })

    it('rejects hours_per_day > 24', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: '2026-06-01',
          end_date: '2026-06-05',
          hours_per_day: 30,
          notes: '',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })

    it('list requires from and to params', () => {
      cy.apiRequest({ url: '/api/assignments' }).then((res) => {
        expect(res.status).to.eq(400)
      })
    })

    it('list rejects malformed dates', () => {
      cy.apiRequest({ url: '/api/assignments?from=2026-99-99&to=2026-12-31' }).then((res) => {
        expect(res.status).to.eq(400)
      })
    })
  })

  describe('API: time-off', () => {
    let personId: string
    before(() => {
      cy.loginAs('admin@example.com', 'admin')
      cy.apiRequest({ url: '/api/people' }).then((r) => {
        personId = (r.body as any[])[0].id
      })
    })

    it('rejects unknown type', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/time-off',
        body: {
          person_id: personId,
          start_date: '2026-06-01',
          end_date: '2026-06-05',
          type: 'unicorn',
          notes: '',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
        expect(res.body.detail).to.match(/type/)
      })
    })

    it('rejects end before start', () => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/time-off',
        body: {
          person_id: personId,
          start_date: '2026-06-10',
          end_date: '2026-06-01',
          type: 'vacation',
          notes: '',
        },
      }).then((res) => {
        expect(res.status).to.eq(422)
      })
    })
  })

  describe('UI: form HTML5 validation', () => {
    it('person name field is required and blocks save', () => {
      cy.visitAuthed('/people')
      cy.contains('button', '+ New person').click()
      cy.get('form').within(() => {
        // Name is the first input; submit without typing.
        cy.contains('button', 'Save').click()
        // Browser HTML5 validation prevents submit; modal stays open.
      })
      cy.contains('h2', 'New person').should('be.visible')
    })
  })
})
