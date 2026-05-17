import { defineConfig } from 'cypress'
import http from 'node:http'

let floatMockServer

const floatMockData = {
  '/people': [
    {
      people_id: 101,
      name: 'Float Alice',
      email: 'alice.float@example.com',
      job_title: 'Designer',
      active: 1,
      work_days_hours: [0, 8, 8, 8, 8, 0, 0],
    },
  ],
  '/clients': [
    { client_id: 201, name: 'Float Client' },
  ],
  '/projects': [
    {
      project_id: 301,
      name: 'Float Website',
      project_code: 'WEB-001',
      client_id: 201,
      color: '00AEEF',
      notes: 'Imported from mock Float',
      active: 1,
      non_billable: 0,
      budget_type: 2,
      budget_total: 25000,
      budget_priority: 0,
      tags: ['design', 'frontend'],
    },
    {
      project_id: 302,
      name: 'Float Internal Tools',
      project_code: 'INT-002',
      client_id: 201,
      color: 'FFAA00',
      notes: 'Internal, non-billable work',
      active: 1,
      non_billable: 1,
      budget_type: 1,
      budget_total: 120,
      budget_priority: 1,
      tags: ['internal'],
    },
  ],
  '/tasks': [
    {
      task_id: 401,
      project_id: 301,
      people_id: 101,
      start_date: '2026-06-02',
      end_date: '2026-06-03',
      hours: 6,
      name: 'Design sprint',
      notes: 'Mock allocation',
    },
  ],
  '/timeoffs': [
    {
      timeoff_id: 501,
      timeoff_type_id: 601,
      start_date: '2026-06-04',
      end_date: '2026-06-04',
      timeoff_notes: 'Mock PTO',
      people_ids: [101],
      full_day: 1,
    },
  ],
  '/timeoff-types': [
    { timeoff_type_id: 601, name: 'Vacation' },
  ],
  '/milestones': [
    {
      milestone_id: 701,
      name: 'Beta launch',
      project_id: 301,
      phase_id: 901,
      date: '2026-06-05',
      end_date: '',
    },
  ],
  '/phases': [
    {
      phase_id: 901,
      project_id: 301,
      name: 'Discovery',
      color: '00AEEF',
      notes: 'Phase 1 of the website rebuild',
      start_date: '2026-06-01',
      end_date: '2026-06-10',
      budget_total: 5000,
      default_hourly_rate: 100,
      non_billable: 0,
      status: 2,
      active: 1,
    },
  ],
  '/logged-time': [
    {
      logged_time_id: 801,
      people_id: 101,
      project_id: 301,
      date: '2026-06-02',
      hours: 4,
      billable: 1,
      notes: 'Mock logged time',
      locked: 0,
      locked_date: null,
      created_by: 9001,
      modified_by: 9001,
      task_name: 'Design sprint',
      task_meta_id: 'meta-401',
    },
  ],
  '/statuses': [
    {
      status_id: 1001,
      status_type_id: 1,
      people_id: 101,
      status_name: '',
      start_date: '2026-06-02',
      end_date: '2026-06-02',
      repeat_state: 0,
      repeat_end_date: '',
    },
  ],
  '/roles': [
    {
      id: 1101,
      name: 'Senior Designer',
      default_hourly_rate: '260.000',
      cost_rate_history: [
        { rate: '180.000', effective_date: '2026-01-01' },
      ],
      people_ids: [101],
      people_count: 1,
    },
  ],
}

function floatMockLocalUrl() {
  if (!floatMockServer) return null
  const address = floatMockServer.address()
  return `http://127.0.0.1:${address.port}`
}

function startFloatMock() {
  const publicHost = process.env.FLOAT_MOCK_HOST || 'host.docker.internal'
  if (floatMockServer) {
    const address = floatMockServer.address()
    return `http://${publicHost}:${address.port}`
  }

  floatMockServer = http.createServer((req, res) => {
    const url = new URL(req.url, 'http://127.0.0.1')
    const token = req.headers.authorization ?? ''

    if (token !== 'Bearer mock-float-token') {
      res.writeHead(401, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ message: 'unauthorized' }))
      return
    }

    const rows = floatMockData[url.pathname]
    if (!rows) {
      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ message: 'not found' }))
      return
    }

    res.writeHead(200, {
      'Content-Type': 'application/json',
      'X-Pagination-Page-Count': '1',
      'X-Pagination-Current-Page': '1',
      'X-Pagination-Per-Page': String(rows.length),
      'X-Pagination-Total-Count': String(rows.length),
    })
    res.end(JSON.stringify(rows))
  })

  return new Promise((resolve) => {
    floatMockServer.listen(0, '0.0.0.0', () => {
      const address = floatMockServer.address()
      resolve(`http://${publicHost}:${address.port}`)
    })
  })
}

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
        log(msg) {
          // eslint-disable-next-line no-console
          console.log('[task]', msg)
          return null
        },
        startFloatMock,
        floatMockLocalUrl,
      })
    },
    env: {
      apiBase: 'http://localhost:8080',
      keycloakIssuer: 'http://localhost:8081/realms/levitate',
      oidcClient: 'levitate-frontend',
    },
  },
})
