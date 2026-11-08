import { Link, Navigate, Outlet, useLocation } from 'react-router'
import { useSession } from '../auth/useSession'
import { Button } from './ui'

export function Logo() {
  return (
    <Link to="/" className="flex items-center gap-2 font-semibold tracking-tight">
      <img src="/favicon.svg" alt="" className="size-7" />
      Fairshare
    </Link>
  )
}

/** Shell for signed-in pages. Sends anyone else to the login page. */
export function AppLayout() {
  const { user, logout } = useSession()
  const location = useLocation()
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname }} />

  return (
    <div className="min-h-dvh">
      <header className="border-b border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <Logo />
          <div className="flex items-center gap-3 text-sm">
            <span className="hidden text-slate-600 sm:inline dark:text-slate-400">{user.name}</span>
            <Button variant="ghost" size="sm" onClick={logout}>
              Sign out
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
