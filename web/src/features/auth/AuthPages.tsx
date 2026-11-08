import { zodResolver } from '@hookform/resolvers/zod'
import { useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { Link, Navigate, useLocation, useNavigate } from 'react-router'
import { z } from 'zod'
import { useSession } from '../../auth/useSession'
import { Logo } from '../../components/Layout'
import { Alert, Button, Card, Field, Input } from '../../components/ui'
import { applyApiErrors } from '../../lib/form'

function AuthLayout({
  title,
  footer,
  children,
}: {
  title: string
  footer: ReactNode
  children: ReactNode
}) {
  const { user } = useSession()
  const location = useLocation()
  const from = (location.state as { from?: string } | null)?.from ?? '/'
  if (user) return <Navigate to={from} replace />

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center px-4 py-12">
      <div className="mb-8">
        <Logo />
      </div>
      <Card className="w-full max-w-sm">
        <h1 className="mb-6 text-xl font-semibold">{title}</h1>
        {children}
      </Card>
      <p className="mt-6 text-sm text-slate-600 dark:text-slate-400">{footer}</p>
    </div>
  )
}

const loginSchema = z.object({
  email: z.email('Enter a valid email address.'),
  password: z.string().min(1, 'Enter your password.'),
})

export function LoginPage() {
  const { login } = useSession()
  const navigate = useNavigate()
  const location = useLocation()
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(loginSchema) })

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null)
    try {
      await login(values)
      navigate((location.state as { from?: string } | null)?.from ?? '/', { replace: true })
    } catch (err) {
      setFormError(applyApiErrors(err, setError, { email: 'email', password: 'password' }))
    }
  })

  return (
    <AuthLayout
      title="Sign in"
      footer={
        <>
          New here?{' '}
          <Link to="/register" className="font-medium text-indigo-600 hover:underline">
            Create an account
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        {formError && <Alert>{formError}</Alert>}
        <Field label="Email" error={errors.email?.message}>
          {(a11y) => <Input type="email" autoComplete="email" {...a11y} {...register('email')} />}
        </Field>
        <Field label="Password" error={errors.password?.message}>
          {(a11y) => (
            <Input
              type="password"
              autoComplete="current-password"
              {...a11y}
              {...register('password')}
            />
          )}
        </Field>
        <Button type="submit" className="w-full" loading={isSubmitting}>
          Sign in
        </Button>
      </form>
    </AuthLayout>
  )
}

const registerSchema = z.object({
  name: z.string().trim().min(1, 'Enter your name.').max(100, 'Keep it under 100 characters.'),
  email: z.email('Enter a valid email address.'),
  password: z
    .string()
    .min(8, 'Use at least 8 characters.')
    .refine((p) => new TextEncoder().encode(p).length <= 72, 'Use at most 72 bytes.'),
})

export function RegisterPage() {
  const { register: createAccount } = useSession()
  const navigate = useNavigate()
  const [formError, setFormError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(registerSchema) })

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null)
    try {
      await createAccount(values)
      navigate('/', { replace: true })
    } catch (err) {
      setFormError(
        applyApiErrors(err, setError, { name: 'name', email: 'email', password: 'password' }),
      )
    }
  })

  return (
    <AuthLayout
      title="Create your account"
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="font-medium text-indigo-600 hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        {formError && <Alert>{formError}</Alert>}
        <Field label="Name" error={errors.name?.message}>
          {(a11y) => <Input autoComplete="name" {...a11y} {...register('name')} />}
        </Field>
        <Field label="Email" error={errors.email?.message}>
          {(a11y) => <Input type="email" autoComplete="email" {...a11y} {...register('email')} />}
        </Field>
        <Field label="Password" error={errors.password?.message} hint="At least 8 characters.">
          {(a11y) => (
            <Input
              type="password"
              autoComplete="new-password"
              {...a11y}
              {...register('password')}
            />
          )}
        </Field>
        <Button type="submit" className="w-full" loading={isSubmitting}>
          Create account
        </Button>
      </form>
    </AuthLayout>
  )
}
