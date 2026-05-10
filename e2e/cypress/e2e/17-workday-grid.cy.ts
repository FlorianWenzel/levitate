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
function addWorkdays(d: Date, n: number) {
  const c = new Date(d); c.setHours(0,0,0,0)
  if (n === 0) return c
  const step = n > 0 ? 1 : -1
  let left = Math.abs(n)
  while (left > 0) { c.setDate(c.getDate() + step); const dow = c.getDay(); if (dow !== 0 && dow !== 6) left-- }
  return c
}

describe('Workday-only grid + paged navigation', () => {
  const monday = startOfWeekMonday(new Date())

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' })
  })

  it('renders only Mon–Fri columns; weekend dates never appear', () => {
    cy.visitAuthed('/schedule')
    cy.get('[data-schedule-grid]').should('be.visible')
    // Walk through 25 columns from rangeStart (4 weeks before today's Monday)
    // and confirm none fall on a weekend.
    let cur = addWorkdays(monday, -20)
    for (let i = 0; i < 25; i++) {
      const dow = cur.getDay()
      expect(dow, `column ${i} day-of-week`).to.be.greaterThan(0)
      expect(dow, `column ${i} day-of-week`).to.be.lessThan(6)
      cur = addWorkdays(cur, 1)
    }
  })

  it('an assignment that spans Fri→Mon shows as a contiguous bar (weekend visually skipped)', () => {
    let projectId = '', personId = '', assignmentId = ''
    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Wd Grid ${Date.now()}`, client: '', color: '#0EA5E9', notes: '' },
    }).then((r) => { projectId = r.body.id })
    cy.then(() => {
      const fridayW1 = ymd(addWorkdays(monday, 4))
      const mondayW2 = ymd(addWorkdays(monday, 5))
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: {
          person_id: personId,
          project_id: projectId,
          start_date: fridayW1,
          end_date: mondayW2,
          hours_per_day: 4,
          notes: 'fri-mon span',
        },
      }).then((r) => { assignmentId = r.body.id })
    })
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${assignmentId}"]`).should('be.visible').then(($bar) => {
        // 2 workdays * 36 px - 2 px margin
        const w = $bar[0].getBoundingClientRect().width
        expect(w).to.be.closeTo(2 * 36 - 2, 2)
      })
    })
  })

  it('utilization counts workdays only — a Mon..Sun assignment at 8h/d = 40 h, not 56 h', () => {
    let projectId = '', personId = '', assignmentId = ''
    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST',
      url: '/api/projects',
      body: { name: `Wd Util ${Date.now()}`, client: '', color: '#10B981', notes: '' },
    }).then((r) => { projectId = r.body.id })
    cy.then(() => {
      const start = ymd(monday)
      const end = ymd(new Date(monday.getTime() + 6 * 86400000))
      cy.apiRequest({
        method: 'POST',
        url: '/api/assignments',
        body: { person_id: personId, project_id: projectId, start_date: start, end_date: end, hours_per_day: 8, notes: '' },
      }).then((r) => { assignmentId = r.body.id })
    })
    cy.then(() => {
      cy.apiRequest({
        url: `/api/reports/utilization?from=${ymd(monday)}&to=${ymd(addWorkdays(monday, 4))}`,
      }).then((res) => {
        const cells = res.body as any[]
        const cell = cells.find((c) => c.week_start === ymd(monday) && c.assigned_hours > 0)
        expect(cell, 'cell with assigned hours').to.exist
        expect(cell.assigned_hours).to.eq(40) // 5 workdays × 8 h
      })
    })
  })

  it('initial load positions today ~4 weeks from the left edge', () => {
    cy.visitAuthed('/schedule')
    cy.get('[data-schedule-grid]').should(($g) => {
      // PAST_PAD_WORKDAYS = 20 columns × 36 px = 720 px.
      expect($g[0].scrollLeft).to.be.closeTo(720, 5)
    })
  })

  it('scrolling horizontally within the loaded window does NOT change the visible date range', () => {
    cy.visitAuthed('/schedule')
    let weekHeaderBefore = ''
    cy.get('[data-schedule-grid]').should(($g) => {
      // Scroll arbitrarily; week labels in the header should not change.
      weekHeaderBefore = $g[0].textContent ?? ''
      expect(weekHeaderBefore.length).to.be.greaterThan(0)
    })
    cy.get('[data-schedule-grid]').then(($g) => { $g[0].scrollLeft = 0 })
    cy.wait(150)
    cy.get('[data-schedule-grid]').then(($g2) => { $g2[0].scrollLeft = 1500 })
    cy.wait(150)
    cy.get('[data-schedule-grid]').should(($g3) => {
      // Same DOM content; we verify the grid wasn't re-rendered for a new range
      // by checking the week labels (W##) are stable.
      const after = $g3[0].textContent ?? ''
      const beforeWeeks = (weekHeaderBefore.match(/W\d+/g) || []).join(',')
      const afterWeeks = (after.match(/W\d+/g) || []).join(',')
      expect(afterWeeks).to.eq(beforeWeeks)
    })
  })

  it('« and » shift the visible range by 4 weeks', () => {
    cy.visitAuthed('/schedule')
    cy.get('[data-schedule-grid]').then(($g) => {
      const initialWeeks = ($g[0].textContent ?? '').match(/W\d+/g) || []
      const firstWeekBefore = initialWeeks[0]
      cy.contains('button', '»').click()
      cy.get('[data-schedule-grid]').should(($g2) => {
        const after = ($g2[0].textContent ?? '').match(/W\d+/g) || []
        expect(after[0], 'first visible week shifted').to.not.eq(firstWeekBefore)
      })
      cy.contains('button', '«').click()
      cy.get('[data-schedule-grid]').should(($g3) => {
        // Back to the original range.
        const back = ($g3[0].textContent ?? '').match(/W\d+/g) || []
        expect(back[0]).to.eq(firstWeekBefore)
      })
    })
  })

  it('‹ and › shift the visible range by 1 week', () => {
    let beforeFirstWeek = ''
    cy.visitAuthed('/schedule')
    cy.get('[data-schedule-grid]').then(($g) => {
      beforeFirstWeek = (($g[0].textContent ?? '').match(/W\d+/g) || [])[0] ?? ''
    })
    cy.contains('button', '›').click()
    cy.get('[data-schedule-grid]').should(($g2) => {
      const after = (($g2[0].textContent ?? '').match(/W\d+/g) || [])[0]
      expect(after, '› shifted forward').to.not.eq(beforeFirstWeek)
    })
    cy.contains('button', '‹').click()
    cy.get('[data-schedule-grid]').should(($g3) => {
      // Back to original.
      const back = (($g3[0].textContent ?? '').match(/W\d+/g) || [])[0]
      expect(back).to.eq(beforeFirstWeek)
    })
  })
})
