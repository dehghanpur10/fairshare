import type { Schemas } from '../api/client'
import { parseFixed, parseMoney } from './money'

export type SplitMethod = Schemas['SplitMethod']

export type SplitRow = {
  userId: string
  included: boolean
  /** Raw text from the input: an amount, a percentage or a weight. */
  value: string
}

export type SplitResult =
  | { ok: true; split: Schemas['CreateExpenseRequest']['split']; shares: Map<string, number> }
  | { ok: false; error: string }

/**
 * Divides total in proportion to weights using the largest remainder method,
 * the same algorithm the API uses, so the preview matches what gets saved.
 * BigInt keeps total * weight exact beyond 2^53.
 */
export function splitWeighted(total: number, weights: number[]): number[] {
  const sum = weights.reduce((a, b) => a + BigInt(b), 0n)
  const exact = weights.map((w) => BigInt(total) * BigInt(w))
  const shares = exact.map((e) => e / sum)
  let leftover = BigInt(total) - shares.reduce((a, b) => a + b, 0n)

  const order = weights
    .map((_, i) => i)
    .sort((a, b) => {
      const ra = exact[a] % sum
      const rb = exact[b] % sum
      return ra === rb ? a - b : rb > ra ? 1 : -1
    })
  for (const i of order) {
    if (leftover === 0n) break
    shares[i] += 1n
    leftover -= 1n
  }
  return shares.map(Number)
}

/**
 * Turns the expense form's split section into an API request, checking the
 * same rules the server does so mistakes show up before submitting.
 */
export function buildSplit(
  method: SplitMethod,
  rows: SplitRow[],
  amount: number | null,
  currency: string,
): SplitResult {
  const included = rows.filter((r) => r.included)
  if (included.length === 0) return { ok: false, error: 'Pick at least one person.' }
  if (amount === null || amount <= 0) return { ok: false, error: 'Enter the amount first.' }

  let values: number[]
  switch (method) {
    case 'equal':
      values = included.map(() => 1)
      break
    case 'exact': {
      const parsed = included.map((r) => parseMoney(r.value || '0', currency))
      if (parsed.some((v) => v === null)) return { ok: false, error: 'Check the amounts.' }
      values = parsed as number[]
      const sum = values.reduce((a, b) => a + b, 0)
      if (sum !== amount) return { ok: false, error: 'Amounts must add up to the total.' }
      break
    }
    case 'percent': {
      const parsed = included.map((r) => parseFixed(r.value, 2))
      if (parsed.some((v) => v === null || v === 0))
        return { ok: false, error: 'Every percentage must be above zero.' }
      values = parsed as number[]
      if (values.reduce((a, b) => a + b, 0) !== 10_000)
        return { ok: false, error: 'Percentages must add up to 100.' }
      break
    }
    case 'shares': {
      const parsed = included.map((r) => Number(r.value))
      if (parsed.some((v) => !Number.isInteger(v) || v < 1))
        return { ok: false, error: 'Shares must be whole numbers of 1 or more.' }
      values = parsed
      break
    }
  }

  const amounts = method === 'exact' ? values : splitWeighted(amount, values)
  return {
    ok: true,
    split: {
      method,
      participants: included.map((r, i) =>
        method === 'equal' ? { user_id: r.userId } : { user_id: r.userId, value: values[i] },
      ),
    },
    shares: new Map(included.map((r, i) => [r.userId, amounts[i]])),
  }
}
