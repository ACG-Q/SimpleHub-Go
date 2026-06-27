import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../api/useAuth.jsx'
import { showToast } from '../api/client'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'

export default function Login() {
  const { entry } = useParams()
  const { login } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!username.trim() || !password.trim()) return
    setLoading(true)
    try {
      await login(username.trim(), password.trim())
      navigate('/')
    } catch (err) {
      showToast(err.message || '登录失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 via-indigo-50/30 to-orange-50/20 p-4">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-sm bg-white/70 backdrop-blur-2xl border border-white/90 rounded-2xl p-9 shadow-2xl anim-in"
      >
        <div className="text-center mb-8">
          <div className="text-2xl font-extrabold tracking-tight mb-1">
            <span className="bg-gradient-to-r from-primary to-primary-light bg-clip-text text-transparent">
              SimpleHub
            </span>
          </div>
          <p className="text-sm text-text-secondary">API 聚合监控管理系统</p>
        </div>

        <div className="space-y-4">
          <Input
            label="管理员账号"
            type="text"
            placeholder="请输入管理员账号"
            value={username}
            onChange={e => setUsername(e.target.value)}
            autoFocus
          />
          <Input
            label="登录密码"
            type="password"
            placeholder="请输入密码"
            value={password}
            onChange={e => setPassword(e.target.value)}
          />
        </div>

        <Button
          type="submit"
          variant="primary"
          className="w-full mt-5 py-2.5 text-base"
          disabled={loading}
        >
          {loading ? '登录中...' : '立即登录'}
        </Button>

        <p className="text-center mt-5 text-xs text-text-muted">
          安全入口访问 &bull; 加密传输
        </p>
      </form>
    </div>
  )
}
