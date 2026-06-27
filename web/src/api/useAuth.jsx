import { createContext, useContext, useState, useCallback } from 'react'
import { useLogin } from '../hooks/useApi'

export const AuthContext = createContext(null)

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem('token'))
  const loginMutation = useLogin()

  const login = useCallback(async (username, password) => {
    const data = await loginMutation.mutateAsync({ username, password })
    localStorage.setItem('token', data.token)
    setToken(data.token)
    return data
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('token')
    setToken(null)
  }, [])

  return (
    <AuthContext.Provider value={{ token, login, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  )
}
