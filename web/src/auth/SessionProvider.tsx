import { useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, setAccessToken, setUnauthorizedHandler, unwrap, type Schemas } from '../api/client'
import { SessionContext, type Session as SessionValue } from './useSession'

type Session = Schemas['Session']

const STORAGE_KEY = 'fairshare.session'

// The token lives in localStorage so a refresh keeps you signed in. That
// trades some XSS exposure for simplicity; the API sends no-store and the app
// renders no untrusted HTML. An httpOnly refresh cookie is the next step.
function loadSession(): Session | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const session = JSON.parse(raw) as Session
    return new Date(session.expires_at) > new Date() ? session : null
  } catch {
    return null
  }
}

function saveSession(session: Session | null) {
  try {
    if (session) localStorage.setItem(STORAGE_KEY, JSON.stringify(session))
    else localStorage.removeItem(STORAGE_KEY)
  } catch {
    // Private mode or blocked storage: the session just won't survive a reload.
  }
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [session, setSession] = useState<Session | null>(() => {
    const s = loadSession()
    setAccessToken(s?.access_token ?? null)
    return s
  })

  const start = useCallback(
    (s: Session | null) => {
      setAccessToken(s?.access_token ?? null)
      saveSession(s)
      setSession(s)
      queryClient.clear()
    },
    [queryClient],
  )

  useEffect(() => {
    setUnauthorizedHandler(() => start(null))
  }, [start])

  const value = useMemo<SessionValue>(
    () => ({
      user: session?.user ?? null,
      login: async (body) => start(await unwrap(api.POST('/v1/auth/login', { body }))),
      register: async (body) => start(await unwrap(api.POST('/v1/auth/register', { body }))),
      logout: () => start(null),
    }),
    [session, start],
  )

  return <SessionContext value={value}>{children}</SessionContext>
}
