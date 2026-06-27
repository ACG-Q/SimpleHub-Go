import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { AuthProvider, useAuth } from './api/useAuth.jsx'
import { ErrorBoundary } from './components/ErrorBoundary'
import './app.css'

const App = React.lazy(() => import('./pages/App'))
const Login = React.lazy(() => import('./pages/Login'))
const Sites = React.lazy(() => import('./pages/Sites'))
const SiteDetail = React.lazy(() => import('./pages/SiteDetail'))

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 30000, refetchOnWindowFocus: false } },
})

function RequireAuth({ children }) {
  const { isAuthenticated } = useAuth()
  if (!isAuthenticated) {
    return (
      <div className="flex h-screen items-center justify-center text-text-muted text-lg">
        请通过安全入口登录
      </div>
    )
  }
  return children
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <Toaster position="top-right" richColors closeButton />
          <React.Suspense fallback={<div className="p-8 text-center text-text-muted">页面加载中...</div>}>
            <ErrorBoundary>
              <Routes>
                <Route path="/" element={<RequireAuth><App /></RequireAuth>}>
                  <Route index element={<ErrorBoundary key="sites"><Sites /></ErrorBoundary>} />
                  <Route path="sites/:id" element={<ErrorBoundary key="detail"><SiteDetail /></ErrorBoundary>} />
                </Route>
                <Route path="/:entry" element={<Login />} />
              </Routes>
            </ErrorBoundary>
          </React.Suspense>
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>
)
