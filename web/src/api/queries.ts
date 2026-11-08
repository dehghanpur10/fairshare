import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, unwrap, type Schemas } from './client'

// Every key for a group starts with ['groups', id], so invalidating that
// prefix after a write refreshes the group, its expenses and its balances.
export const keys = {
  groups: ['groups'] as const,
  group: (id: string) => ['groups', id] as const,
  expenses: (id: string) => ['groups', id, 'expenses'] as const,
  balances: (id: string) => ['groups', id, 'balances'] as const,
  settlements: (id: string) => ['groups', id, 'settlements'] as const,
}

const inGroup = (groupID: string) => ({ params: { path: { groupID } } })

export function useGroups() {
  return useQuery({
    queryKey: keys.groups,
    queryFn: async () => (await unwrap(api.GET('/v1/groups'))).items,
  })
}

export function useGroup(groupID: string) {
  return useQuery({
    queryKey: keys.group(groupID),
    queryFn: () => unwrap(api.GET('/v1/groups/{groupID}', inGroup(groupID))),
  })
}

export function useExpenses(groupID: string) {
  return useInfiniteQuery({
    queryKey: keys.expenses(groupID),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) =>
      unwrap(
        api.GET('/v1/groups/{groupID}/expenses', {
          params: { path: { groupID }, query: { limit: 20, cursor: pageParam } },
        }),
      ),
    getNextPageParam: (page) => page.next_cursor,
  })
}

export function useBalances(groupID: string) {
  return useQuery({
    queryKey: keys.balances(groupID),
    queryFn: () => unwrap(api.GET('/v1/groups/{groupID}/balances', inGroup(groupID))),
  })
}

export function useSettlements(groupID: string) {
  return useQuery({
    queryKey: keys.settlements(groupID),
    queryFn: async () =>
      (await unwrap(api.GET('/v1/groups/{groupID}/settlements', inGroup(groupID)))).items,
  })
}

export function useCreateGroup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: Schemas['CreateGroupRequest']) => unwrap(api.POST('/v1/groups', { body })),
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.groups }),
  })
}

/** Mutations scoped to one group refresh everything under that group. */
function useGroupMutation<TVars, TData>(groupID: string, fn: (vars: TVars) => Promise<TData>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.group(groupID) }),
  })
}

export function useAddMember(groupID: string) {
  return useGroupMutation(groupID, (body: Schemas['AddMemberRequest']) =>
    unwrap(api.POST('/v1/groups/{groupID}/members', { ...inGroup(groupID), body })),
  )
}

export function useCreateExpense(groupID: string) {
  return useGroupMutation(groupID, (body: Schemas['CreateExpenseRequest']) =>
    unwrap(api.POST('/v1/groups/{groupID}/expenses', { ...inGroup(groupID), body })),
  )
}

export function useDeleteExpense(groupID: string) {
  return useGroupMutation(groupID, (expenseID: string) =>
    unwrap(
      api.DELETE('/v1/groups/{groupID}/expenses/{expenseID}', {
        params: { path: { groupID, expenseID } },
      }),
    ),
  )
}

export function useCreateSettlement(groupID: string) {
  return useGroupMutation(groupID, (body: Schemas['CreateSettlementRequest']) =>
    unwrap(api.POST('/v1/groups/{groupID}/settlements', { ...inGroup(groupID), body })),
  )
}
