import { useParams } from 'react-router'
import { ApiError } from '../../api/client'
import { useGroup } from '../../api/queries'
import { useSession } from '../../auth/useSession'
import { Alert, Loading } from '../../components/ui'
import { BalancesCard } from '../balances/BalancesCard'
import { ExpensesCard } from '../expenses/ExpensesCard'
import { MembersCard } from './MembersCard'
import { GroupContext } from './context'

export function GroupPage() {
  const { groupId = '' } = useParams()
  const { user } = useSession()
  const group = useGroup(groupId)

  if (group.isPending) return <Loading label="Loading group" />
  if (group.isError) {
    return (
      <Alert>
        {group.error instanceof ApiError && group.error.status === 404
          ? 'This group does not exist, or you are not a member.'
          : `Could not load the group. ${group.error.message}`}
      </Alert>
    )
  }

  const me = group.data.members.find((m) => m.user_id === user?.id)
  const value = {
    group: group.data,
    meId: user?.id ?? '',
    isOwner: me?.role === 'owner',
    nameOf: (id: string) =>
      id === user?.id
        ? 'You'
        : (group.data.members.find((m) => m.user_id === id)?.name ?? 'Former member'),
  }

  return (
    <GroupContext value={value}>
      <div className="space-y-6">
        <div className="flex flex-wrap items-baseline gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">{group.data.name}</h1>
          <span className="text-sm text-slate-500">
            {group.data.members.length} {group.data.members.length === 1 ? 'member' : 'members'} ·{' '}
            {group.data.currency}
          </span>
        </div>
        <div className="grid gap-6 lg:grid-cols-[1fr_20rem]">
          <ExpensesCard />
          <div className="space-y-6">
            <BalancesCard />
            <MembersCard />
          </div>
        </div>
      </div>
    </GroupContext>
  )
}
