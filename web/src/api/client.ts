import createClient, { type Middleware } from 'openapi-fetch'
import type { components, paths } from './schema'

export type Schemas = components['schemas']

/** An error response from the API, parsed from its problem+json body. */
export class ApiError extends Error {
  readonly status: number
  readonly fieldErrors: Record<string, string>

  constructor(status: number, message: string, fieldErrors: Record<string, string> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fieldErrors = fieldErrors
  }
}

let accessToken: string | null = null
let onUnauthorized: () => void = () => {}

export function setAccessToken(token: string | null) {
  accessToken = token
}

/** Called when a signed-in request comes back 401, i.e. the token expired. */
export function setUnauthorizedHandler(handler: () => void) {
  onUnauthorized = handler
}

const auth: Middleware = {
  onRequest({ request }) {
    if (accessToken) request.headers.set('Authorization', `Bearer ${accessToken}`)
    return request
  },
  onResponse({ response }) {
    if (response.status === 401 && accessToken) onUnauthorized()
    return response
  },
}

export const api = createClient<paths>({
  baseUrl: import.meta.env.VITE_API_URL || window.location.origin,
})
api.use(auth)

function toApiError(error: unknown, response: Response): ApiError {
  const problem = (typeof error === 'object' && error !== null ? error : {}) as Partial<
    Schemas['Problem']
  >
  const message =
    problem.detail ??
    (problem.errors ? 'Please fix the highlighted fields.' : undefined) ??
    problem.title ??
    response.statusText ??
    'Something went wrong.'
  return new ApiError(response.status, message, problem.errors)
}

/**
 * Resolves an openapi-fetch call to its data or throws an ApiError, which is
 * the contract React Query expects from a query or mutation function.
 */
export async function unwrap<T>(
  call: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
  const { data, error, response } = await call
  if (!response.ok) throw toApiError(error, response)
  return data as T
}
