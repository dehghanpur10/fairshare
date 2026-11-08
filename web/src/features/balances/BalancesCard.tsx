import { useBalances, useCreateSettlement, useSettlements } from '../../api/queries'
import type { Schemas } from '../../api/client'
import { Alert, Button, Card, EmptyState, Loading } from '../../components/ui'
import { formatDate } from '../../lib/dates'
import { formatMoney } from '../../lib/money'
import { useGroupContext } from '../groups/context'

function NetAmount({ net, currency }: { net: number; currency: string }) {
  if (net === 0) return <span className="text-slate-500">settled up</span>
  return net > 0 ? (
    <span className="text-emerald-700 dark:text-emerald-400">
      gets back {formatMoney(net, currency)}
    </span>
  ) : (
    <span className="text-amber-700 dark:text-amber-400">owes {formatMoney(-net, currency)}</span>
  )
}

function SuggestedTransfer({ transfer }: { transfer: Schemas['Transfer'] }) {
  const { group, meId, isOwner, nameOf } = useGroupContext()
  const settle = useCreateSettlement(group.id)
  const canRecord = isOwner || transfer.from === meId || transfer.to === meId

  return (
    <li className="flex items-center justify-between gap-2">
      <span>
        <strong className="font-medium">{nameOf(transfer.from)}</strong> → {nameOf(transfer.to)}{' '}
        <span className="tabular-nums">{formatMoney(transfer.amount, group.currency)}</span>
      </span>
      {canRecord && (
        <Button
          variant="secondary"
          size="sm"
          loading={settle.isPending}
          onClick={() => settle.mutate(transfer)}
          aria-label={`Record that ${nameOf(transfer.from)} paid ${nameOf(transfer.to)}`}
        >
          Mark paid
        </Button>
      )}
    </li>
  )
}

function RecentRepayments() {
  const { group, nameOf } = useGroupContext()
  const settlements = useSettlements(group.id)
  if (!settlements.data?.length) return null

  return (
    <div className="mt-5 border-t border-slate-200 pt-4 dark:border-slate-800">
      <h3 className="mb-2 text-sm font-medium text-slate-600 dark:text-slate-400">
        Recent repayments
      </h3>
      <ul className="space-y-1.5 text-sm">
        {settlements.data.slice(0, 5).map((s) => (
          <li key={s.id} className="flex justify-between gap-2 text-slate-600 dark:text-slate-400">
            <span>
              {nameOf(s.from)} paid {nameOf(s.to)}
            </span>
            <span className="tabular-nums">{formatMoney(s.amount, group.currency)}</span>
          </li>
        ))}
      </ul>
      <p className="mt-2 text-xs text-slate-500">
        Latest on {formatDate(settlements.data[0].created_at)}
      </p>
    </div>
  )
}

export function BalancesCard() {
  const { group, nameOf } = useGroupContext()
  const balances = useBalances(group.id)

  return (
    <Card title="Balances">
      {balances.isPending ? (
        <Loading />
      ) : balances.isError ? (
        <Alert>{balances.error.message}</Alert>
      ) : (
        <>
          <ul className="space-y-2 text-sm">
            {balances.data.balances.map((b) => (
              <li key={b.user_id} className="flex justify-between gap-2">
                <span className="truncate">{nameOf(b.user_id)}</span>
                <NetAmount net={b.net} currency={group.currency} />
              </li>
            ))}
          </ul>

          <div className="mt-5 border-t border-slate-200 pt-4 dark:border-slate-800">
            <h3 className="mb-2 text-sm font-medium text-slate-600 dark:text-slate-400">
              Fewest payments to settle up
            </h3>
            {balances.data.transfers.length === 0 ? (
              <EmptyState>Everyone is square.</EmptyState>
            ) : (
              <ul className="space-y-2 text-sm">
                {balances.data.transfers.map((t) => (
                  <SuggestedTransfer key={`${t.from}-${t.to}`} transfer={t} />
                ))}
              </ul>
            )}
          </div>
          <RecentRepayments />
        </>
      )}
    </Card>
  )
}
