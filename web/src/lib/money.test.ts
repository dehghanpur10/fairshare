import { describe, expect, it } from 'vitest'
import { currencyDigits, formatMoney, parseFixed, parseMoney, toDecimalString } from './money'

describe('currencyDigits', () => {
  it('knows which currencies have cents', () => {
    expect(currencyDigits('EUR')).toBe(2)
    expect(currencyDigits('JPY')).toBe(0)
  })
})

describe('formatMoney', () => {
  it('formats minor units in the currency', () => {
    expect(formatMoney(123456, 'USD', 'en-US')).toBe('$1,234.56')
    expect(formatMoney(500, 'JPY', 'en-US')).toBe('¥500')
    expect(formatMoney(-2550, 'EUR', 'en-US')).toBe('-€25.50')
  })
})

describe('parseMoney', () => {
  it.each([
    ['12', 1200],
    ['12.5', 1250],
    ['12,50', 1250],
    ['0.01', 1],
    [' 7.00 ', 700],
  ])('parses %j as %d cents', (input, want) => {
    expect(parseMoney(input, 'EUR')).toBe(want)
  })

  it.each(['', 'abc', '-5', '1.234', '1,000.00', '1e3'])('rejects %j', (input) => {
    expect(parseMoney(input, 'EUR')).toBeNull()
  })

  it('has no cents for zero-decimal currencies', () => {
    expect(parseMoney('500', 'JPY')).toBe(500)
    expect(parseMoney('500.5', 'JPY')).toBeNull()
  })

  it('avoids float rounding errors', () => {
    // 0.29 * 100 is 28.999999999999996 in floating point.
    expect(parseMoney('0.29', 'USD')).toBe(29)
    expect(parseFixed('33.33', 2)).toBe(3333)
  })
})

describe('toDecimalString', () => {
  it('round-trips with parseMoney', () => {
    for (const minor of [0, 5, 99, 100, 123456]) {
      expect(parseMoney(toDecimalString(minor, 'EUR'), 'EUR')).toBe(minor)
    }
    expect(toDecimalString(5, 'EUR')).toBe('0.05')
    expect(toDecimalString(500, 'JPY')).toBe('500')
  })
})
