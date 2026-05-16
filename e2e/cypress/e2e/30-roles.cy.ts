describe('Roles CRUD and Float parity', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.resetState()
    // Trigger syncUser middleware so the admin shows up as a person.
    cy.apiRequest({ url: '/api/me' })
  })

  it('supports CRUD against /api/roles', () => {
    // Empty list.
    cy.apiRequest({ url: '/api/roles' }).then((res) => {
      expect(res.status).to.eq(200)
      expect(res.body).to.be.an('array').that.is.empty
    })

    // Create with a string rate (Float's native shape).
    let roleId = ''
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/roles',
        body: {
          name: 'Senior Designer',
          default_hourly_rate: '260.000',
          cost_rate_history: [
            { rate: '180.000', effective_date: '2026-01-01' },
          ],
        },
      }).then((res) => {
        expect(res.status).to.eq(201)
        expect(res.body).to.include({
          name: 'Senior Designer',
          default_hourly_rate: '260.000',
          people_count: 0,
        })
        expect(res.body.cost_rate_history).to.have.length(1)
        expect(res.body.cost_rate_history[0]).to.include({
          rate: '180.000',
          effective_date: '2026-01-01',
        })
        expect(res.body.people_ids).to.be.an('array').that.is.empty
        roleId = res.body.id
      })
    })

    // List shows the new role.
    cy.then(() => {
      cy.apiRequest({ url: '/api/roles' }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body).to.have.length(1)
        expect(res.body[0].id).to.eq(roleId)
        expect(res.body[0].default_hourly_rate).to.eq('260.000')
      })
    })

    // GET by id.
    cy.then(() => {
      cy.apiRequest({ url: `/api/roles/${roleId}` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.id).to.eq(roleId)
      })
    })

    // PATCH the rate using a numeric value — server accepts both forms.
    cy.then(() => {
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/roles/${roleId}`,
        body: { default_hourly_rate: 295 },
      }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.default_hourly_rate).to.eq('295.000')
      })
    })

    // Duplicate name -> 409.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/roles',
        body: { name: 'senior designer' },
      }).its('status').should('eq', 409)
    })

    // Missing name -> 422.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/roles',
        body: { name: '' },
      }).its('status').should('eq', 422)
    })

    // Bad rate -> 422.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/roles',
        body: { name: 'Junior Designer', default_hourly_rate: 'not-a-number' },
      }).its('status').should('eq', 422)
    })

    // DELETE.
    cy.then(() => {
      cy.apiRequest({ method: 'DELETE', url: `/api/roles/${roleId}` })
        .its('status').should('eq', 204)
    })
    cy.then(() => {
      cy.apiRequest({ url: `/api/roles/${roleId}` }).its('status').should('eq', 404)
    })
  })

  it('derives people_ids and people_count from the people.role text column', () => {
    // Create a role.
    let roleId = ''
    cy.apiRequest({
      method: 'POST',
      url: '/api/roles',
      body: { name: 'Engineer', default_hourly_rate: '120.000' },
    }).then((res) => {
      expect(res.status).to.eq(201)
      roleId = res.body.id
    })

    // Create a person whose role text matches the role name.
    let personId = ''
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/people',
        body: { name: 'Grace Hopper', email: 'grace@example.com', role: 'engineer', weekly_capacity_hours: 40 },
      }).then((res) => {
        expect(res.status).to.eq(201)
        personId = res.body.id
      })
    })

    // people_count reflects the matching person (case-insensitive).
    cy.then(() => {
      cy.apiRequest({ url: `/api/roles/${roleId}` }).then((res) => {
        expect(res.status).to.eq(200)
        expect(res.body.people_count).to.eq(1)
        expect(res.body.people_ids).to.include(personId)
      })
    })
  })

  it('blocks members from creating, updating, or deleting roles', () => {
    let roleId = ''
    cy.apiRequest({
      method: 'POST',
      url: '/api/roles',
      body: { name: 'Producer', default_hourly_rate: '90.000' },
    }).then((res) => {
      expect(res.status).to.eq(201)
      roleId = res.body.id
    })

    cy.then(() => {
      cy.loginAs('member@example.com', 'member')
      cy.apiRequest({ url: '/api/me' })
    })

    // Members can read.
    cy.then(() => {
      cy.apiRequest({ url: '/api/roles' }).its('status').should('eq', 200)
    })
    // But not write.
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/roles',
        body: { name: 'Sneaky', default_hourly_rate: '0' },
      }).its('status').should('eq', 403)
    })
    cy.then(() => {
      cy.apiRequest({
        method: 'PATCH',
        url: `/api/roles/${roleId}`,
        body: { default_hourly_rate: '0.000' },
      }).its('status').should('eq', 403)
    })
    cy.then(() => {
      cy.apiRequest({ method: 'DELETE', url: `/api/roles/${roleId}` })
        .its('status').should('eq', 403)
    })
  })

  it('lets an admin create a role through the UI and see it on the Roles page', () => {
    const stamp = Date.now()
    const name = `UX Lead ${stamp}`

    cy.visitAuthed('/roles')
    cy.contains('h1', 'Roles').should('be.visible')
    cy.get('[data-cy=roles-empty]').should('be.visible')

    cy.get('[data-cy=new-role-button]').click()
    cy.get('[data-cy=role-form]').should('be.visible')
    cy.get('[data-cy=role-form-name]').type(name)
    cy.get('[data-cy=role-form-rate]').clear().type('310.000')
    cy.get('[data-cy=role-form-submit]').click()

    cy.get(`[data-cy="role-row-${name}"]`).should('be.visible').within(() => {
      cy.get('[data-cy=role-name]').should('have.text', name)
      cy.get('[data-cy=role-rate]').should('have.text', '310.000')
      cy.get('[data-cy=role-people-count]').should('have.text', '0')
    })
    cy.screenshot('roles-after-create')
  })

  it('serves the Float-parity /roles payload from the mock API', () => {
    cy.task('startFloatMock').then(() => {
      cy.task('floatMockLocalUrl').then((localUrl) => {
        cy.request({
          url: `${localUrl}/roles`,
          headers: { Authorization: 'Bearer mock-float-token' },
          failOnStatusCode: false,
        }).then((res) => {
          expect(res.status).to.eq(200)
          expect(res.body).to.have.length(1)
          expect(res.body[0]).to.include({
            id: 1101,
            name: 'Senior Designer',
            default_hourly_rate: '260.000',
            people_count: 1,
          })
          expect(res.body[0].cost_rate_history[0]).to.include({
            rate: '180.000',
            effective_date: '2026-01-01',
          })
          expect(res.body[0].people_ids).to.deep.eq([101])
        })
      })
    })
  })
})
