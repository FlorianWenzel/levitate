/// <reference types="cypress" />

declare global {
  namespace Cypress {
    interface Chainable {
      /** Get a token via OIDC password grant and stash it for visitAuthed. */
      loginAs(username: string, password: string): Chainable<unknown>
      /** Like cy.visit, but injects the OIDC user into sessionStorage before page load. */
      visitAuthed(path: string): Chainable<unknown>
      /** Convenience: cy.request with the current bearer token attached. */
      apiRequest(opts: Partial<Cypress.RequestOptions> & { url: string }): Chainable<Cypress.Response<any>>
      /** Wipe assignments/time_off/projects/people/audit_log. Backend gates this on LEVITATE_ALLOW_TEST_RESET=true. */
      resetState(): Chainable<unknown>
    }
  }
}

function storageKey() {
  return `oidc.user:${Cypress.env('keycloakIssuer')}:${Cypress.env('oidcClient')}`
}

function decodeJwtPayload(token: string): any {
  const part = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
  const padded = part + '='.repeat((4 - (part.length % 4)) % 4)
  return JSON.parse(atob(padded))
}

Cypress.Commands.add('loginAs', (username: string, password: string) => {
  cy.request({
    method: 'POST',
    url: `${Cypress.env('keycloakIssuer')}/protocol/openid-connect/token`,
    form: true,
    body: {
      grant_type: 'password',
      client_id: Cypress.env('oidcClient'),
      username,
      password,
      scope: 'openid profile email',
    },
  }).then(({ body }) => {
    const profile = decodeJwtPayload(body.id_token)
    const user = {
      id_token: body.id_token,
      access_token: body.access_token,
      refresh_token: body.refresh_token,
      token_type: 'Bearer',
      scope: body.scope ?? 'openid profile email',
      profile,
      expires_at: Math.floor(Date.now() / 1000) + body.expires_in,
      session_state: body.session_state,
    }
    Cypress.env('oidcUser', user)
    Cypress.env('accessToken', body.access_token)
  })
})

Cypress.Commands.add('visitAuthed', (path: string) => {
  const user = Cypress.env('oidcUser')
  if (!user) throw new Error('Call cy.loginAs first')
  const key = storageKey()
  cy.visit(path, {
    onBeforeLoad(win) {
      win.sessionStorage.setItem(key, JSON.stringify(user))
    },
  })
})

Cypress.Commands.add('resetState', () => {
  cy.request({
    method: 'POST',
    url: `${Cypress.env('apiBase')}/api/test/reset`,
  }).its('status').should('eq', 200)
})

Cypress.Commands.add('apiRequest', (opts) => {
  const token = Cypress.env('accessToken')
  return cy.request({
    ...opts,
    url: opts.url.startsWith('http') ? opts.url : `${Cypress.env('apiBase')}${opts.url}`,
    headers: {
      ...(opts.headers ?? {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    failOnStatusCode: opts.failOnStatusCode ?? false,
  })
})

export {}
