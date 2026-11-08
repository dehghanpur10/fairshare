// The API speaks in integer minor units (cents). These helpers convert to and
// from what people type and read, without ever going through a float.

const digitsCache = new Map<string, number>()

/** Number of decimal places a currency uses: 2 for EUR, 0 for JPY. */
export function currencyDigits(currency: string): number {
  let digits = digitsCache.get(currency)
  if (digits === undefined) {
    digits =
      new Intl.NumberFormat('en', { style: 'currency', currency }).resolvedOptions()
        .maximumFractionDigits ?? 2
    digitsCache.set(currency, digits)
  }
  return digits
}

export function formatMoney(minor: number, currency: string, locale?: string): string {
  const digits = currencyDigits(currency)
  return new Intl.NumberFormat(locale, { style: 'currency', currency }).format(minor / 10 ** digits)
}

/**
 * Parses a non-negative decimal like "12.5" into an integer scaled by
 * 10^digits (1250 for two digits). Accepts "." or "," as the separator but no
 * grouping, and returns null for anything else, including too many decimals.
 */
export function parseFixed(input: string, digits: number): number | null {
  const match = /^\s*(\d{1,10})(?:[.,](\d*))?\s*$/.exec(input)
  if (!match) return null
  const [, whole, fraction = ''] = match
  if (fraction.length > digits) return null
  return Number(whole) * 10 ** digits + Number(fraction.padEnd(digits, '0') || '0')
}

export function parseMoney(input: string, currency: string): number | null {
  return parseFixed(input, currencyDigits(currency))
}

/** Renders minor units as a plain decimal for an input field: 1250 -> "12.50". */
export function toDecimalString(minor: number, currency: string): string {
  const digits = currencyDigits(currency)
  if (digits === 0) return String(minor)
  const s = String(minor).padStart(digits + 1, '0')
  return `${s.slice(0, -digits)}.${s.slice(-digits)}`
}
