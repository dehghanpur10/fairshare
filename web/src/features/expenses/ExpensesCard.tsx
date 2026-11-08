import { useState } from 'react'
import type { Schemas } from '../../api/client'
import { useDeleteExpense, useExpenses } from '../../api/queries'
import { Alert, Button, Card, EmptyState, Loading } from '../../components/ui'
import { formatCalendarDate } from '../../lib/dates'
import { formatMoney } from '../../lib/money'
import { useGroupContext } from '../groups/context'
import { AddExpenseForm } from './AddExpenseForm'

function TrashIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="currentColor" className="size-4" aria-hidden>
      <path
        fillRule="evenodd"
        d="M8.75 1A2.75 2.75 0 0 0 6 3.75v.44c-.8.08-1.59.18-2.38.3a.75.75 0 1 0 .23 1.49l.15-.02.84 10.42A2.75 2.75 0 0 0 7.58 19h4.84a2.75 2.75 0 0 0 2.74-2.62l.84-10.42.15.02a.75.75 0 0 0 .23-1.49c-.79-.12-1.58-.22-2.38-.3v-.44A2.75 2.75 0 0 0 11.25 1h-2.5ZM10 4c.84 0 1.67.03 2.5.08v-.33c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.33C8.33 4.03 9.16 4 10 4ZM8.58 7.72a.75.75 0 0 0-1.5.06l.3 7.5a.75.75 0 1 0 1.5-.06l-.3-7.5Zm4.34.06a.75.75 0 1 0-1.5-.06l-.3 7.5a.75.75 0 1 0 1.5.06l.3-7.5Z"
        clipRule="evenodd"
      />
    </svg>
  )
}

function ExpenseRow({ expense }: { expense: Schemas['Expense'] }) {
  const { group, meId, isOwner, nameOf } = useGroupContext()
  const remove = useDeleteExpense(group.id)
  const myShare = expense.shares.find((s) => s.user_id === meId)?.amount
  const canDelete = isOwner || expense.created_by === meId

  return (
    <li className="flex items-start gap-2 py-3">
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline justify-between gap-3">
          <p className="font-medium break-words">{expense.description}</p>
          <p className="shrink-0 font-medium tabular-nums">
            {formatMoney(expense.amount, group.currency)}
          </p>
        </div>
        <div className="mt-0.5 flex flex-wrap items-baseline justify-between gap-x-3 text-sm text-slate-500">
          <p>
            {nameOf(expense.paid_by)} paid · {formatCalendarDate(expense.spent_on)}
          </p>
          {myShare !== undefined && (
            <p className="tabular-nums">your share {formatMoney(myShare, group.currency)}</p>
          )}
        </div>
      </div>
      {canDelete && (
        <Button
          variant="danger"
          size="sm"
          loading={remove.isPending}
          onClick={() => {
            if (confirm(`Delete "${expense.description}"?`)) remove.mutate(expense.id)
          }}
          aria-label={`Delete ${expense.description}`}
          title="Delete"
          className="-mr-1.5"
        >
          {!remove.isPending && <TrashIcon />}
        </Button>
      )}
    </li>
  )
}

export function ExpensesCard() {
  const { group } = useGroupContext()
  const expenses = useExpenses(group.id)
  const [adding, setAdding] = useState(false)
  const items = expenses.data?.pages.flatMap((p) => p.items) ?? []

  return (
    <Card
      title="Expenses"
      actions={!adding && <Button onClick={() => setAdding(true)}>Add expense</Button>}
    >
      {adding && <AddExpenseForm onDone={() => setAdding(false)} />}

      {expenses.isPending ? (
        <Loading label="Loading expenses" />
      ) : expenses.isError ? (
        <Alert>{expenses.error.message}</Alert>
      ) : items.length === 0 ? (
        !adding && <EmptyState>Nothing yet. Add the first expense to get going.</EmptyState>
      ) : (
        <>
          <ul className="divide-y divide-slate-200 dark:divide-slate-800">
            {items.map((e) => (
              <ExpenseRow key={e.id} expense={e} />
            ))}
          </ul>
          {expenses.hasNextPage && (
            <div className="pt-3 text-center">
              <Button
                variant="ghost"
                loading={expenses.isFetchingNextPage}
                onClick={() => expenses.fetchNextPage()}
              >
                Show older expenses
              </Button>
            </div>
          )}
        </>
      )}
    </Card>
  )
}
