import { createContext, use } from 'react'
import type { Schemas } from '../api/client'

export type Session = {
  user: Schemas['User'] | null
  login: (body: Schemas['LoginRequest']) => Promise<void>
  register: (body: Schemas['RegisterRequest']) => Promise<void>
  logout: () => void
}

export const SessionContext = createContext<Session | null>(null)

export function useSession(): Session {
  const ctx = use(SessionContext)
  if (!ctx) throw new Error('useSession must be used inside SessionProvider')
  return ctx
}
