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

describe('Schedule grid: bar lane stacking', () => {
  const monday = startOfWeekMonday(new Date())
  let personId = ''
  let projectAId = ''
  let projectBId = ''
  let projectCId = ''

  beforeEach(() => {
    cy.loginAs('admin@example.com', 'admin')
    cy.apiRequest({ url: '/api/me' }).its('body.sub').should('exist')
    cy.apiRequest({ url: '/api/people' }).then((r) => { personId = (r.body as any[])[0].id })
    cy.apiRequest({
      method: 'POST', url: '/api/projects',
      body: { name: 'Stack A', client: '', color: '#EF4444', notes: '' },
    }).then((r) => { projectAId = r.body.id })
    cy.apiRequest({
      method: 'POST', url: '/api/projects',
      body: { name: 'Stack B', client: '', color: '#3B82F6', notes: '' },
    }).then((r) => { projectBId = r.body.id })
    cy.apiRequest({
      method: 'POST', url: '/api/projects',
      body: { name: 'Stack C', client: '', color: '#10B981', notes: '' },
    }).then((r) => { projectCId = r.body.id })
  })

  function createAssignment(personId: string, projectId: string, start: Date, end: Date, hours = 4) {
    return cy.apiRequest({
      method: 'POST', url: '/api/assignments',
      body: {
        person_id: personId, project_id: projectId,
        start_date: ymd(start), end_date: ymd(end),
        hours_per_day: hours, notes: '',
      },
    })
  }

  it('two overlapping bars stack flush — second bar top = first bar bottom (no gap, no overlap)', () => {
    let a1 = '', a2 = ''
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 7), addDays(monday, 12), 6).then((r) => { a1 = r.body.id }))
    cy.then(() => createAssignment(personId, projectBId, addDays(monday, 9), addDays(monday, 14), 4).then((r) => { a2 = r.body.id }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${a1}"]`).should('be.visible')
      cy.get(`[data-assignment-id="${a2}"]`).should('be.visible')
      cy.get(`[data-assignment-id="${a1}"]`).then(($b1) => {
        cy.get(`[data-assignment-id="${a2}"]`).then(($b2) => {
          const r1 = $b1[0].getBoundingClientRect()
          const r2 = $b2[0].getBoundingClientRect()
          expect($b1.attr('data-lane'), 'lane1').to.eq('0')
          expect($b2.attr('data-lane'), 'lane2').to.eq('1')
          // Bar 2's top must equal bar 1's bottom — touching, no gap, no overlap.
          expect(r2.top, 'bar2.top').to.be.closeTo(r1.bottom, 1)
          // Heights reflect hours with PX_PER_HOUR=8: 6h → 48px, 4h → 32px.
          expect(r1.height).to.be.closeTo(48, 1) // 6 * 8
          expect(r2.height).to.be.closeTo(32, 1) // 4 * 8
        })
      })
      cy.screenshot('schedule-two-lanes')
    })
  })

  it('three mutually-overlapping assignments produce three lanes', () => {
    const ids: string[] = []
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 7), addDays(monday, 14), 3).then((r) => { ids.push(r.body.id) }))
    cy.then(() => createAssignment(personId, projectBId, addDays(monday, 8), addDays(monday, 15), 3).then((r) => { ids.push(r.body.id) }))
    cy.then(() => createAssignment(personId, projectCId, addDays(monday, 9), addDays(monday, 16), 3).then((r) => { ids.push(r.body.id) }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      const lanes = new Set<string>()
      cy.wrap(ids).each((id: any) => {
        cy.get(`[data-assignment-id="${id}"]`).then(($bar) => {
          lanes.add($bar.attr('data-lane') as string)
        })
      })
      cy.then(() => {
        expect(lanes.size, 'distinct lanes').to.eq(3)
        expect(lanes).to.include('0')
        expect(lanes).to.include('1')
        expect(lanes).to.include('2')
      })
      // Verify bars touch with no gap and no overlap.
      cy.then(() => {
        const rects: DOMRect[] = []
        cy.wrap(ids).each((id: any) => {
          cy.get(`[data-assignment-id="${id}"]`).then(($b) => rects.push($b[0].getBoundingClientRect()))
        }).then(() => {
          rects.sort((a, b) => a.top - b.top) // top-to-bottom order
          for (let i = 0; i + 1 < rects.length; i++) {
            // Adjacent bars touch: next.top ≈ current.bottom.
            expect(rects[i + 1].top, `bar[${i + 1}].top`).to.be.closeTo(rects[i].bottom, 1)
          }
        })
      })
      cy.screenshot('schedule-three-lanes')
    })
  })

  it('non-overlapping assignments share lane 0', () => {
    let a1 = '', a2 = ''
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 1), addDays(monday, 4), 4).then((r) => { a1 = r.body.id }))
    cy.then(() => createAssignment(personId, projectBId, addDays(monday, 8), addDays(monday, 11), 4).then((r) => { a2 = r.body.id }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${a1}"]`).should('have.attr', 'data-lane', '0')
      cy.get(`[data-assignment-id="${a2}"]`).should('have.attr', 'data-lane', '0')
    })
  })

  it('bars are top-anchored within their lane (4h fills only top half of the lane)', () => {
    let smallId = '', fullId = ''
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 7), addDays(monday, 10), 8).then((r) => { fullId = r.body.id }))
    cy.then(() => createAssignment(personId, projectBId, addDays(monday, 14), addDays(monday, 17), 4).then((r) => { smallId = r.body.id }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${fullId}"]`).then(($full) => {
        cy.get(`[data-assignment-id="${smallId}"]`).then(($small) => {
          const fullR = $full[0].getBoundingClientRect()
          const smallR = $small[0].getBoundingClientRect()
          // Both share lane 0 (no overlap), so they have the same `top` (top-anchored).
          expect(Math.abs(fullR.top - smallR.top)).to.be.lessThan(2)
          // Full bar extends further down than the half-hour bar.
          expect(fullR.bottom - smallR.bottom).to.be.greaterThan(10)
        })
      })
    })
  })

  it('default row height = 8h (64px) regardless of how few hours the bar booked', () => {
    let a1 = ''
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 7), addDays(monday, 10), 4).then((r) => { a1 = r.body.id }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-person-id="${personId}"]`).then(($row) => {
        const rowR = $row[0].getBoundingClientRect()
        cy.get(`[data-assignment-id="${a1}"]`).then(($bar) => {
          const barR = $bar[0].getBoundingClientRect()
          // Row is 64 px tall (8 h default) even though the bar is only 32 px (4 h).
          expect(rowR.height).to.be.closeTo(64, 1)
          expect(barR.height).to.be.closeTo(32, 1) // 4 h * 8
          // Bar is top-anchored: 4 h occupies the upper half of the row.
          expect(barR.top).to.be.closeTo(rowR.top, 1)
          expect(barR.bottom - rowR.top).to.be.closeTo(32, 1)
        })
      })
    })
  })

  it('bars under 2h all render at the readable minimum height (16 px)', () => {
    const ids: string[] = []
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 7), addDays(monday, 9), 0.25).then((r) => { ids.push(r.body.id) }))
    cy.then(() => createAssignment(personId, projectBId, addDays(monday, 11), addDays(monday, 13), 0.5).then((r) => { ids.push(r.body.id) }))
    cy.then(() => createAssignment(personId, projectCId, addDays(monday, 15), addDays(monday, 17), 1).then((r) => { ids.push(r.body.id) }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.wrap(ids).each((id: any) => {
        cy.get(`[data-assignment-id="${id}"]`).then(($bar) => {
          // 0.25, 0.5, 1 h all clamp to MIN_BAR_HEIGHT (16 px).
          expect($bar[0].getBoundingClientRect().height).to.be.closeTo(16, 1)
          // Project label still renders inside the bar.
          expect($bar.text().trim().length, 'has visible label').to.be.greaterThan(0)
        })
      })
      cy.screenshot('schedule-min-height-bars')
    })
  })

  it('a bar grows past the minimum height once hours > ~2h', () => {
    let smallId = '', mediumId = ''
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 7), addDays(monday, 9), 1).then((r) => { smallId = r.body.id }))
    cy.then(() => createAssignment(personId, projectBId, addDays(monday, 11), addDays(monday, 13), 4).then((r) => { mediumId = r.body.id }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-assignment-id="${smallId}"]`).then(($s) => {
        cy.get(`[data-assignment-id="${mediumId}"]`).then(($m) => {
          const sH = $s[0].getBoundingClientRect().height
          const mH = $m[0].getBoundingClientRect().height
          expect(sH).to.be.closeTo(16, 1)
          expect(mH).to.be.closeTo(32, 1)
          expect(mH).to.be.greaterThan(sH)
        })
      })
    })
  })

  it('row grows tall enough that no bar pokes outside the row container', () => {
    let a1 = '', a2 = ''
    cy.then(() => createAssignment(personId, projectAId, addDays(monday, 7), addDays(monday, 12), 8).then((r) => { a1 = r.body.id }))
    cy.then(() => createAssignment(personId, projectBId, addDays(monday, 9), addDays(monday, 14), 8).then((r) => { a2 = r.body.id }))
    cy.then(() => {
      cy.visitAuthed('/schedule')
      cy.get(`[data-person-id="${personId}"]`).then(($row) => {
        const rowR = $row[0].getBoundingClientRect()
        cy.get(`[data-assignment-id="${a1}"]`).then(($b1) => {
          cy.get(`[data-assignment-id="${a2}"]`).then(($b2) => {
            const r1 = $b1[0].getBoundingClientRect()
            const r2 = $b2[0].getBoundingClientRect()
            // Each bar is fully contained vertically in the row's bounds.
            expect(r1.top, 'b1.top').to.be.gte(rowR.top - 1)
            expect(r1.bottom, 'b1.bottom').to.be.lte(rowR.bottom + 1)
            expect(r2.top, 'b2.top').to.be.gte(rowR.top - 1)
            expect(r2.bottom, 'b2.bottom').to.be.lte(rowR.bottom + 1)
          })
        })
      })
    })
  })
})
