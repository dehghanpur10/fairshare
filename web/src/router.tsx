import { createBrowserRouter } from 'react-router'
import { ErrorPage } from './components/ErrorPage'
import { AppLayout } from './components/Layout'
import { LoginPage, RegisterPage } from './features/auth/AuthPages'
import { GroupPage } from './features/groups/GroupPage'
import { GroupsPage } from './features/groups/GroupsPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage />, errorElement: <ErrorPage /> },
  { path: '/register', element: <RegisterPage />, errorElement: <ErrorPage /> },
  {
    element: <AppLayout />,
    errorElement: <ErrorPage />,
    children: [
      { index: true, element: <GroupsPage /> },
      { path: 'groups/:groupId', element: <GroupPage /> },
    ],
  },
])
