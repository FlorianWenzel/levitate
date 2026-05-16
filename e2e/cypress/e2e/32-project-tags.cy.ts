describe('Project tags', () => {
  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
  })

  it('persists tags through create, update, get, and list via the API', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'API Tagged Project',
        client: 'Acme',
        color: '#0EA5E9',
        billable: true,
        tags: ['design', 'frontend', '  design  ', '', 'Frontend'],
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.tags).to.deep.eq(['design', 'frontend'])
      const id = res.body.id as string

      cy.apiRequest({ url: `/api/projects/${id}` }).then((get) => {
        expect(get.status).to.eq(200)
        expect(get.body.tags).to.deep.eq(['design', 'frontend'])
      })

      cy.apiRequest({ url: '/api/projects' }).then((list) => {
        expect(list.status).to.eq(200)
        const found = (list.body as any[]).find((p) => p.id === id)
        expect(found).to.exist
        expect(found.tags).to.deep.eq(['design', 'frontend'])
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'API Tagged Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          tags: ['backend', 'urgent'],
        },
      }).then((patch) => {
        expect(patch.status).to.eq(200)
        expect(patch.body.tags).to.deep.eq(['backend', 'urgent'])
      })

      cy.apiRequest({
        method: 'PATCH',
        url: `/api/projects/${id}`,
        body: {
          name: 'API Tagged Project',
          client: 'Acme',
          color: '#0EA5E9',
          billable: true,
          tags: [],
        },
      }).then((cleared) => {
        expect(cleared.status).to.eq(200)
        expect(cleared.body.tags).to.deep.eq([])
      })
    })
  })

  it('defaults tags to an empty array when omitted from the payload', () => {
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: {
        name: 'Untagged Project',
        client: '',
        color: '#0EA5E9',
        billable: true,
      },
    }).then((res) => {
      expect(res.status).to.eq(201)
      expect(res.body.tags).to.deep.eq([])
    })
  })

  it('admin can create, edit, and clear tags from the project form', () => {
    const stamp = Date.now()
    const name = `Tag UI ${stamp}`

    cy.visitAuthed('/projects')
    cy.contains('button', '+ New project').click()
    cy.contains('label', 'Name').siblings('input').first().type(name)
    cy.get('[data-cy="project-tags-input"]').type('alpha, beta, alpha')
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name)
      .should('be.visible')
      .within(() => {
        cy.get('[data-cy="project-tag-chip"]').should('have.length', 2)
        cy.get('[data-cy="project-tag-chip"]').eq(0).should('contain.text', 'alpha')
        cy.get('[data-cy="project-tag-chip"]').eq(1).should('contain.text', 'beta')
      })

    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="project-tags-input"]').should('have.value', 'alpha, beta')
    cy.get('[data-cy="project-tags-input"]').clear().type('gamma')
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="project-tag-chip"]').should('have.length', 1)
      cy.get('[data-cy="project-tag-chip"]').eq(0).should('contain.text', 'gamma')
    })

    cy.contains('tr', name).contains('button', 'Edit').click()
    cy.get('[data-cy="project-tags-input"]').clear()
    cy.get('[data-cy="project-save"]').click()

    cy.contains('tr', name).within(() => {
      cy.get('[data-cy="project-tag-chip"]').should('not.exist')
      cy.get('[data-cy="project-tags-cell"]').should('contain.text', '—')
    })
  })

  it('imports project tags from a mocked Float API', () => {
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

      cy.apiRequest({ url: '/api/projects' }).then((res) => {
        expect(res.status).to.eq(200)
        const website = res.body.find((p: any) => p.name === 'Float Website')
        expect(website).to.exist
        expect(website.tags).to.deep.eq(['design', 'frontend'])

        const internal = res.body.find((p: any) => p.name === 'Float Internal Tools')
        expect(internal).to.exist
        expect(internal.tags).to.deep.eq(['internal'])
      })
    })
  })
})
