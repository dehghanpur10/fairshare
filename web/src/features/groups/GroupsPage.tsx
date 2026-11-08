import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router'
import { z } from 'zod'
import { useCreateGroup, useGroups } from '../../api/queries'
import { Alert, Button, Card, EmptyState, Field, Input, Loading, Select } from '../../components/ui'
import { formatDate } from '../../lib/dates'
import { applyApiErrors } from '../../lib/form'

const CURRENCIES = [
  'USD',
  'EUR',
  'GBP',
  'CAD',
  'AUD',
  'CHF',
  'JPY',
  'SEK',
  'NOK',
  'INR',
  'TRY',
  'AED',
] as const

const schema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Give the group a name.')
    .max(100, 'Keep it under 100 characters.'),
  currency: z.enum(CURRENCIES),
})
type GroupForm = z.infer<typeof schema>

function CreateGroupForm({ onCancel }: { onCancel: () => void }) {
  const navigate = useNavigate()
  const createGroup = useCreateGroup()
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<GroupForm>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', currency: 'EUR' },
  })

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null)
    try {
      const group = await createGroup.mutateAsync(values)
      navigate(`/groups/${group.id}`)
    } catch (err) {
      setFormError(applyApiErrors(err, setError, { name: 'name', currency: 'currency' }))
    }
  })

  return (
    <Card title="New group">
      <form
        onSubmit={onSubmit}
        className="grid gap-4 sm:grid-cols-[1fr_10rem_auto] sm:items-start"
        noValidate
      >
        {formError && (
          <div className="sm:col-span-3">
            <Alert>{formError}</Alert>
          </div>
        )}
        <Field label="Name" error={errors.name?.message}>
          {(a11y) => <Input placeholder="Lisbon trip" autoFocus {...a11y} {...register('name')} />}
        </Field>
        <Field label="Currency" error={errors.currency?.message}>
          {(a11y) => (
            <Select {...a11y} {...register('currency')}>
              {CURRENCIES.map((c) => (
                <option key={c}>{c}</option>
              ))}
            </Select>
          )}
        </Field>
        <div className="flex gap-2 sm:pt-7">
          <Button type="submit" loading={createGroup.isPending}>
            Create
          </Button>
          <Button variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </form>
    </Card>
  )
}

export function GroupsPage() {
  const groups = useGroups()
  const [creating, setCreating] = useState(false)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">Your groups</h1>
        {!creating && <Button onClick={() => setCreating(true)}>New group</Button>}
      </div>

      {creating && <CreateGroupForm onCancel={() => setCreating(false)} />}

      {groups.isPending ? (
        <Loading label="Loading groups" />
      ) : groups.isError ? (
        <Alert>Could not load your groups. {groups.error.message}</Alert>
      ) : groups.data.length === 0 ? (
        <Card>
          <EmptyState>No groups yet. Create one for a trip, a flat or a dinner club.</EmptyState>
        </Card>
      ) : (
        <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {groups.data.map((g) => (
            <li key={g.id}>
              <Link
                to={`/groups/${g.id}`}
                className="block rounded-xl bg-white p-5 shadow-sm ring-1 ring-slate-200 transition hover:ring-indigo-400 dark:bg-slate-900 dark:ring-slate-800"
              >
                <div className="flex items-start justify-between gap-2">
                  <h2 className="font-semibold">{g.name}</h2>
                  <span className="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                    {g.currency}
                  </span>
                </div>
                <p className="mt-2 text-sm text-slate-500">Created {formatDate(g.created_at)}</p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
