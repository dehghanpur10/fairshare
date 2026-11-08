import { isRouteErrorResponse, Link, useRouteError } from 'react-router'

export function ErrorPage() {
  const error = useRouteError()
  const notFound = isRouteErrorResponse(error) && error.status === 404
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center gap-3 px-4 text-center">
      <h1 className="text-2xl font-semibold">
        {notFound ? 'Page not found' : 'Something went wrong'}
      </h1>
      <p className="text-slate-500">
        {notFound
          ? 'That page does not exist.'
          : 'Try refreshing. If it keeps happening, sign in again.'}
      </p>
      <Link to="/" className="font-medium text-indigo-600 hover:underline">
        Back to your groups
      </Link>
    </div>
  )
}
