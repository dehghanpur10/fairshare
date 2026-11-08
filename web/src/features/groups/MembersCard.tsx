import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { useAddMember } from '../../api/queries'
import { Alert, Button, Card, Field, Input } from '../../components/ui'
import { applyApiErrors } from '../../lib/form'
import { useGroupContext } from './context'

const schema = z.object({ email: z.email('Enter a valid email address.') })

function AddMemberForm() {
  const { group } = useGroupContext()
  const addMember = useAddMember(group.id)
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors },
  } = useForm({ resolver: zodResolver(schema), defaultValues: { email: '' } })

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null)
    try {
      await addMember.mutateAsync(values)
      reset()
    } catch (err) {
      setFormError(applyApiErrors(err, setError, { email: 'email' }))
    }
  })

  return (
    <form
      onSubmit={onSubmit}
      className="mt-4 space-y-3 border-t border-slate-200 pt-4 dark:border-slate-800"
      noValidate
    >
      {formError && <Alert>{formError}</Alert>}
      <Field
        label="Add someone by email"
        error={errors.email?.message}
        hint="They need a Fairshare account."
      >
        {(a11y) => (
          <Input type="email" placeholder="friend@example.com" {...a11y} {...register('email')} />
        )}
      </Field>
      <Button type="submit" variant="secondary" size="sm" loading={addMember.isPending}>
        Add member
      </Button>
    </form>
  )
}

export function MembersCard() {
  const { group, isOwner, meId } = useGroupContext()
  return (
    <Card title="Members">
      <ul className="space-y-2 text-sm">
        {group.members.map((m) => (
          <li key={m.user_id} className="flex items-center justify-between gap-2">
            <span className="truncate">
              {m.name}
              {m.user_id === meId && <span className="text-slate-500"> (you)</span>}
            </span>
            {m.role === 'owner' && (
              <span className="rounded bg-indigo-50 px-1.5 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300">
                Owner
              </span>
            )}
          </li>
        ))}
      </ul>
      {isOwner && <AddMemberForm />}
    </Card>
  )
}
