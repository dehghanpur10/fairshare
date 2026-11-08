import type { FieldValues, Path, UseFormSetError } from 'react-hook-form'
import { ApiError } from '../api/client'

/**
 * Copies the API's per-field validation messages onto a react-hook-form form
 * and returns a message for anything that has no matching input. fieldMap
 * maps API field names (paid_by) to form field names (paidBy).
 */
export function applyApiErrors<T extends FieldValues>(
  error: unknown,
  setError: UseFormSetError<T>,
  fieldMap: Partial<Record<string, Path<T>>>,
): string | null {
  if (!(error instanceof ApiError)) return 'Something went wrong. Please try again.'

  const unmatched: string[] = []
  for (const [apiField, message] of Object.entries(error.fieldErrors)) {
    const field = fieldMap[apiField]
    if (field) setError(field, { message: capitalize(message) })
    else unmatched.push(message)
  }
  if (Object.keys(error.fieldErrors).length === 0) return error.message
  return unmatched.length ? unmatched.map(capitalize).join(' ') : null
}

const capitalize = (s: string) =>
  s.charAt(0).toUpperCase() + s.slice(1) + (s.endsWith('.') ? '' : '.')
