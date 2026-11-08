import { createContext, use } from 'react'
import type { Schemas } from '../../api/client'

type GroupContextValue = {
  group: Schemas['GroupDetail']
  meId: string
  isOwner: boolean
  /** Display name for a member, "You" for the signed-in user. */
  nameOf: (userId: string) => string
}

export const GroupContext = createContext<GroupContextValue | null>(null)

export function useGroupContext(): GroupContextValue {
  const ctx = use(GroupContext)
  if (!ctx) throw new Error('useGroupContext must be used inside GroupPage')
  return ctx
}
