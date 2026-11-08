import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Schemas } from '../../api/client'
import { formatMoney } from '../../lib/money'
import { GroupContext } from '../groups/context'
import { AddExpenseForm } from './AddExpenseForm'

const mutateAsync = vi.fn()
vi.mock('../../api/queries', () => ({
  useCreateExpense: () => ({ mutateAsync, isPending: false }),
}))

const member = (user_id: string, name: string, role: 'owner' | 'member' = 'member') => ({
  user_id,
  name,
  email: `${user_id}@example.com`,
  role,
  joined_at: '2026-05-01T10:00:00Z',
})

const group: Schemas['GroupDetail'] = {
  id: 'g1',
  name: 'Lisbon trip',
  currency: 'EUR',
  created_by: 'ana',
  created_at: '2026-05-01T10:00:00Z',
  members: [member('ana', 'Ana', 'owner'), member('ben', 'Ben'), member('cat', 'Cat')],
}

function renderForm() {
  const onDone = vi.fn()
  const nameOf = (id: string) =>
    id === 'ana' ? 'You' : (group.members.find((m) => m.user_id === id)?.name ?? '')
  render(
    <GroupContext value={{ group, meId: 'ana', isOwner: true, nameOf }}>
      <AddExpenseForm onDone={onDone} />
    </GroupContext>,
  )
  return { user: userEvent.setup(), onDone }
}

const eur = (minor: number) => formatMoney(minor, 'EUR')

describe('AddExpenseForm', () => {
  beforeEach(() => {
    mutateAsync.mockReset()
  })

  it('previews an equal split as you type', async () => {
    const { user } = renderForm()
    await user.type(screen.getByLabelText('Amount (EUR)'), '100')

    expect(screen.getByTestId('share-ana')).toHaveTextContent(eur(3334))
    expect(screen.getByTestId('share-ben')).toHaveTextContent(eur(3333))
    expect(screen.getByTestId('share-cat')).toHaveTextContent(eur(3333))
  })

  it('blocks percentages that do not add up to 100', async () => {
    const { user } = renderForm()
    await user.type(screen.getByLabelText('Description'), 'Car rental')
    await user.type(screen.getByLabelText('Amount (EUR)'), '300')
    await user.click(screen.getByLabelText('Percentages'))
    await user.click(screen.getByLabelText('Cat'))
    await user.type(screen.getByLabelText('% for You'), '50')
    await user.type(screen.getByLabelText('% for Ben'), '40')

    expect(screen.getByText('Percentages must add up to 100.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save expense' }))
    expect(mutateAsync).not.toHaveBeenCalled()
    expect(screen.getByRole('alert')).toHaveTextContent('Percentages must add up to 100.')
  })

  it('sends the request the API expects', async () => {
    mutateAsync.mockResolvedValue({})
    const { user, onDone } = renderForm()
    await user.type(screen.getByLabelText('Description'), 'Dinner at Taberna')
    await user.type(screen.getByLabelText('Amount (EUR)'), '90,50')
    await user.click(screen.getByLabelText('Cat'))
    await user.click(screen.getByRole('button', { name: 'Save expense' }))

    expect(mutateAsync).toHaveBeenCalledWith({
      description: 'Dinner at Taberna',
      amount: 9050,
      spent_on: expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/),
      paid_by: 'ana',
      split: { method: 'equal', participants: [{ user_id: 'ana' }, { user_id: 'ben' }] },
    })
    expect(onDone).toHaveBeenCalled()
  })

  it('validates required fields before calling the API', async () => {
    const { user } = renderForm()
    await user.click(screen.getByRole('button', { name: 'Save expense' }))

    expect(await screen.findByText('What was it for?')).toBeInTheDocument()
    expect(screen.getByText('Enter an amount like 12.50.')).toBeInTheDocument()
    expect(mutateAsync).not.toHaveBeenCalled()
  })
})
