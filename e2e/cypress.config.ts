import { defineConfig } from 'cypress'

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:3000',
    supportFile: 'cypress/support/e2e.ts',
    specPattern: 'cypress/e2e/**/*.cy.ts',
    video: false,
    screenshotsFolder: 'cypress/screenshots',
    viewportWidth: 1400,
    viewportHeight: 900,
    defaultCommandTimeout: 10000,
    setupNodeEvents(on) {
      on('task', {
        log(msg: string) {
          // eslint-disable-next-line no-console
          console.log('[task]', msg)
          return null
        },
      })
    },
    env: {
      apiBase: 'http://localhost:8080',
      keycloakIssuer: 'http://localhost:8081/realms/levitate',
      oidcClient: 'levitate-frontend',
    },
  },
})
