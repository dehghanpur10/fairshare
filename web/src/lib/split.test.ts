import { describe, expect, it } from 'vitest'
import { buildSplit, splitWeighted, type SplitRow } from './split'

const rows = (...values: string[]): SplitRow[] =>
  values.map((value, i) => ({ userId: `u${i + 1}`, included: true, value }))

describe('splitWeighted', () => {
  it('matches the API for the documented cases', () => {
    expect(splitWeighted(1000, [1, 1, 1])).toEqual([334, 333, 333])
    expect(splitWeighted(10, [1, 2])).toEqual([3, 7])
    expect(splitWeighted(333, [3333, 3333, 3334])).toEqual([111, 111, 111])
  })

  it('always accounts for every cent', () => {
    for (let total = 1; total < 500; total += 7) {
      const weights = [3, 1, 4, 1, 5]
      expect(splitWeighted(total, weights).reduce((a, b) => a + b, 0)).toBe(total)
    }
  })

  it('stays exact past the float safe-integer limit', () => {
    expect(splitWeighted(1_000_000_000_000, [999_999, 1])).toEqual([999_999_000_000, 1_000_000])
  })
})

describe('buildSplit', () => {
  it('splits equally between the people ticked', () => {
    const input = rows('', '', '')
    input[2].included = false
    const result = buildSplit('equal', input, 1001, 'EUR')
    expect(result.ok && [...result.shares.values()]).toEqual([501, 500])
    expect(result.ok && result.split).toEqual({
      method: 'equal',
      participants: [{ user_id: 'u1' }, { user_id: 'u2' }],
    })
  })

  it('requires exact amounts to add up', () => {
    expect(buildSplit('exact', rows('6', '4'), 1000, 'EUR').ok).toBe(true)
    expect(buildSplit('exact', rows('6', '3.99'), 1000, 'EUR')).toEqual({
      ok: false,
      error: 'Amounts must add up to the total.',
    })
  })

  it('sends percentages as basis points', () => {
    const result = buildSplit('percent', rows('70', '30'), 30000, 'EUR')
    expect(result.ok && result.split.participants.map((p) => p.value)).toEqual([7000, 3000])
    expect(buildSplit('percent', rows('50', '40'), 100, 'EUR').ok).toBe(false)
  })

  it('rejects fractional or zero shares', () => {
    expect(buildSplit('shares', rows('2', '1'), 900, 'EUR').ok).toBe(true)
    expect(buildSplit('shares', rows('1.5', '1'), 900, 'EUR').ok).toBe(false)
    expect(buildSplit('shares', rows('0', '1'), 900, 'EUR').ok).toBe(false)
  })

  it('needs an amount and at least one person', () => {
    expect(buildSplit('equal', rows(''), null, 'EUR').ok).toBe(false)
    expect(buildSplit('equal', [], 100, 'EUR').ok).toBe(false)
  })
})
