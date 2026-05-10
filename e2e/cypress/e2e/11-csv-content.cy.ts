function ymd(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}
function startOfWeekMonday(d: Date) {
  const c = new Date(d); const dow = c.getDay()
  c.setDate(c.getDate() - ((dow + 6) % 7)); c.setHours(0,0,0,0); return c
}
function addDays(d: Date, n: number) { const c = new Date(d); c.setDate(c.getDate() + n); return c }

function parseCSV(s: string): string[][] {
  // Simple CSV parser sufficient for our well-formed exports (no embedded newlines/quotes in test data).
  return s
    .replace(/\r/g, '')
    .split('\n')
    .filter((l) => l.length > 0)
    .map((l) => l.split(','))
}

describe('CSV content', () => {
  const stamp = Date.now()
  const monday = startOfWeekMonday(new Date())
  const start = addDays(monday, 7)
  const end = addDays(start, 4)
  const personName = `CSV Person ${stamp}`
  const projectName = `CSV Project ${stamp}`
  let personId: string
  let projectId: string
  let assignmentId: string

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    // Reset wipes everything; recreate the fixtures fresh per test.
    cy.apiRequest({
      method: 'POST',
      url: '/api/people',
      body: { name: personName, email: '', role: 'qa', weekly_capacity_hours: 40 },
    }).then((r) => { personId = r.body.id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: projectName, client: 'Acme', color: '#0EA5E9', notes: '' },
    }).then((r) => { projectId = r.body.id })
    cy.then(() => {
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: ymd(start),
          end_date: ymd(end),
          hours_per_day: 4,
          notes: 'csv fixture',
        },
      }).then((r) => { assignmentId = r.body.id })
    })
  })

  it('assignments.csv has the expected header and our row', () => {
    cy.apiRequest({
      url: `/api/reports/assignments.csv?from=${ymd(monday)}&to=${ymd(addDays(monday, 27))}`,
    }).then((res) => {
      expect(res.status).to.eq(200)
      expect(res.headers['content-disposition']).to.contain('assignments.csv')
      const rows = parseCSV(res.body as string)
      expect(rows[0]).to.deep.eq([
        'assignment_id', 'person_name', 'person_id',
        'project_name', 'client', 'start_date', 'end_date',
        'days', 'hours_per_day', 'total_hours', 'notes',
      ])
      const ours = rows.find((r) => r[0] === assignmentId)
      expect(ours, 'created assignment row in CSV').to.exist
      expect(ours![1]).to.eq(personName)
      expect(ours![3]).to.eq(projectName)
      expect(ours![4]).to.eq('Acme')
      expect(ours![5]).to.eq(ymd(start))
      expect(ours![6]).to.eq(ymd(end))
      expect(ours![7]).to.eq('5')   // 5 inclusive days
      expect(ours![8]).to.eq('4')   // hours/day
      expect(ours![9]).to.eq('20')  // total = 5*4
      expect(ours![10]).to.eq('csv fixture')
    })
  })

  it('utilization.csv has our person/week with correct hours', () => {
    cy.apiRequest({
      url: `/api/reports/utilization.csv?from=${ymd(monday)}&to=${ymd(addDays(monday, 27))}`,
    }).then((res) => {
      expect(res.status).to.eq(200)
      const rows = parseCSV(res.body as string)
      expect(rows[0]).to.deep.eq([
        'person_name', 'person_id', 'weekly_capacity_hours', 'week_start',
        'assigned_hours', 'time_off_hours', 'available_hours', 'utilization_pct', 'overallocated',
      ])
      const target = rows.find((r) => r[0] === personName && r[3] === ymd(start))
      expect(target, `utilization row for ${personName} on ${ymd(start)}`).to.exist
      expect(target![4]).to.eq('20') // 5 days * 4h
      expect(target![5]).to.eq('0')
      expect(target![6]).to.eq('40')
      expect(target![7]).to.eq('50') // 20/40 * 100
      expect(target![8]).to.eq('false')
    })
  })
})
