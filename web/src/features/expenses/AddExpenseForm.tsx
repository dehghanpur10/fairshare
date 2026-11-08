import { zodResolver } from '@hookform/resolvers/zod'
import { useMemo, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { useCreateExpense } from '../../api/queries'
import { Alert, Button, Field, Input, Select } from '../../components/ui'
import { today } from '../../lib/dates'
import { applyApiErrors } from '../../lib/form'
import { formatMoney, parseMoney } from '../../lib/money'
import { buildSplit, type SplitMethod, type SplitRow } from '../../lib/split'
import { useGroupContext } from '../groups/context'

const METHODS: { value: SplitMethod; label: string; hint: string }[] = [
  { value: 'equal', label: 'Equally', hint: '' },
  { value: 'exact', label: 'Exact amounts', hint: 'Amount' },
  { value: 'percent', label: 'Percentages', hint: '%' },
  { value: 'shares', label: 'Shares', hint: 'Shares' },
]

const expenseSchema = (currency: string) =>
  z.object({
    description: z
      .string()
      .trim()
      .min(1, 'What was it for?')
      .max(200, 'Keep it under 200 characters.'),
    amount: z
      .string()
      .refine((v) => (parseMoney(v, currency) ?? 0) > 0, 'Enter an amount like 12.50.'),
    spentOn: z.iso.date('Pick a date.'),
    paidBy: z.string().min(1),
  })

export function AddExpenseForm({ onDone }: { onDone: () => void }) {
  const { group, meId, nameOf } = useGroupContext()
  const createExpense = useCreateExpense(group.id)
  const [formError, setFormError] = useState<string | null>(null)
  const [method, setMethod] = useState<SplitMethod>('equal')
  const [rows, setRows] = useState<SplitRow[]>(() =>
    group.members.map((m) => ({ userId: m.user_id, included: true, value: '' })),
  )

  const schema = useMemo(() => expenseSchema(group.currency), [group.currency])

  const {
    register,
    handleSubmit,
    setError,
    control,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: { description: '', amount: '', spentOn: today(), paidBy: meId },
  })
  const amountText = useWatch({ control, name: 'amount' })
  const amount = parseMoney(amountText ?? '', group.currency)
  const split = buildSplit(method, rows, amount, group.currency)

  const updateRow = (userId: string, patch: Partial<SplitRow>) =>
    setRows((prev) => prev.map((r) => (r.userId === userId ? { ...r, ...patch } : r)))

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null)
    if (!split.ok) {
      setFormError(split.error)
      return
    }
    try {
      await createExpense.mutateAsync({
        description: values.description,
        amount: parseMoney(values.amount, group.currency)!,
        spent_on: values.spentOn,
        paid_by: values.paidBy,
        split: split.split,
      })
      onDone()
    } catch (err) {
      setFormError(
        applyApiErrors(err, setError, {
          description: 'description',
          amount: 'amount',
          spent_on: 'spentOn',
          paid_by: 'paidBy',
        }),
      )
    }
  })

  return (
    <form
      onSubmit={onSubmit}
      className="mb-6 space-y-5 rounded-lg bg-slate-50 p-4 ring-1 ring-slate-200 dark:bg-slate-950 dark:ring-slate-800"
      aria-label="Add expense"
      noValidate
    >
      {formError && <Alert>{formError}</Alert>}

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Description" error={errors.description?.message}>
          {(a11y) => (
            <Input
              placeholder="Dinner, taxi, groceries…"
              autoFocus
              {...a11y}
              {...register('description')}
            />
          )}
        </Field>
        <Field label={`Amount (${group.currency})`} error={errors.amount?.message}>
          {(a11y) => (
            <Input inputMode="decimal" placeholder="0.00" {...a11y} {...register('amount')} />
          )}
        </Field>
        <Field label="Paid by" error={errors.paidBy?.message}>
          {(a11y) => (
            <Select {...a11y} {...register('paidBy')}>
              {group.members.map((m) => (
                <option key={m.user_id} value={m.user_id}>
                  {nameOf(m.user_id)}
                </option>
              ))}
            </Select>
          )}
        </Field>
        <Field label="Date" error={errors.spentOn?.message}>
          {(a11y) => <Input type="date" {...a11y} {...register('spentOn')} />}
        </Field>
      </div>

      <fieldset>
        <legend className="mb-2 text-sm font-medium">Split</legend>
        <div className="mb-3 flex flex-wrap gap-1.5" role="radiogroup" aria-label="Split method">
          {METHODS.map((m) => (
            <label
              key={m.value}
              className="cursor-pointer rounded-full px-3 py-1 text-sm ring-1 ring-slate-300 has-checked:bg-indigo-600 has-checked:text-white has-checked:ring-indigo-600 has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-indigo-600 dark:ring-slate-700"
            >
              <input
                type="radio"
                name="method"
                value={m.value}
                checked={method === m.value}
                onChange={() => setMethod(m.value)}
                className="sr-only"
              />
              {m.label}
            </label>
          ))}
        </div>

        <ul className="space-y-2">
          {rows.map((row) => {
            const share = split.ok ? split.shares.get(row.userId) : undefined
            const hint = METHODS.find((m) => m.value === method)!.hint
            return (
              <li key={row.userId} className="flex items-center gap-3 text-sm">
                <label className="flex min-w-0 flex-1 items-center gap-2">
                  <input
                    type="checkbox"
                    checked={row.included}
                    onChange={(e) => updateRow(row.userId, { included: e.target.checked })}
                    className="size-4 rounded accent-indigo-600"
                  />
                  <span className="truncate">{nameOf(row.userId)}</span>
                </label>
                {method !== 'equal' && (
                  // Width lives on a wrapper: Input is w-full by design.
                  <div className="w-28 shrink-0">
                    <Input
                      aria-label={`${hint} for ${nameOf(row.userId)}`}
                      inputMode={method === 'shares' ? 'numeric' : 'decimal'}
                      placeholder={hint}
                      value={row.value}
                      disabled={!row.included}
                      onChange={(e) => updateRow(row.userId, { value: e.target.value })}
                    />
                  </div>
                )}
                <span
                  className="w-24 text-right text-slate-500 tabular-nums"
                  data-testid={`share-${row.userId}`}
                >
                  {row.included && share !== undefined ? formatMoney(share, group.currency) : '—'}
                </span>
              </li>
            )
          })}
        </ul>
        {!split.ok && amount !== null && amount > 0 && (
          <p className="mt-2 text-sm text-amber-700 dark:text-amber-400">{split.error}</p>
        )}
      </fieldset>

      <div className="flex gap-2">
        <Button type="submit" loading={createExpense.isPending}>
          Save expense
        </Button>
        <Button variant="ghost" onClick={onDone}>
          Cancel
        </Button>
      </div>
    </form>
  )
}
