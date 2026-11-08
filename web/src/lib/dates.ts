const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
const calendarDateFormat = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeZone: 'UTC',
})

/** Formats an ISO timestamp in the viewer's time zone. */
export function formatDate(iso: string): string {
  return dateFormat.format(new Date(iso))
}

/**
 * Formats a calendar date like "2026-05-14". It is read as UTC midnight and
 * shown in UTC so it never shifts a day for viewers west of Greenwich.
 */
export function formatCalendarDate(date: string): string {
  return calendarDateFormat.format(new Date(`${date}T00:00:00Z`))
}

/** Today's date in the viewer's time zone as YYYY-MM-DD. */
export function today(): string {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}
