import { useRef } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../api/useAuth.jsx'
import { ModalHost } from '../components/ui/Modal'
import { showToast } from '../api/client'
import { useExportSites, useImportSites } from '../hooks/useApi'

export default function App() {
  const { logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const fileRef = useRef(null)

  const exportSitesMutation = useExportSites()
  const importSitesMutation = useImportSites()

  const handleExport = async () => {
    try {
      const data = await exportSitesMutation.mutateAsync()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url; a.download = `sites-export-${Date.now()}.json`; a.click()
      URL.revokeObjectURL(url)
      showToast('导出成功')
    } catch (e) {
      showToast(e.message || '导出失败', 'error')
    }
  }

  const handleImport = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    try {
      const text = await file.text()
      const data = JSON.parse(text)
      if (!window.confirm(`确定导入 ${data.sites?.length || 0} 个站点？`)) return
      await importSitesMutation.mutateAsync(data)
      showToast('导入成功')
      window.location.reload()
    } catch (e) {
      showToast(e.message || '导入失败', 'error')
    }
    e.target.value = ''
  }

  const isActive = (path) => location.pathname === path

  const navItems = [
    { path: '/', label: '站点管理', icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z' },
    { path: '/dashboard', label: '仪表盘', icon: 'M3 3v18h18V3H3zm16 16H5V5h14v14zM7 9h2v6H7V9zm4-2h2v8h-2V7zm4 4h2v4h-2v-4z' },
    { path: '/settings', label: '设置', icon: 'M12 15a3 3 0 100-6 3 3 0 000 6z M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 01-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z' },
  ]

  return (
    <ModalHost>
    <div className="min-h-screen">
      <nav className="fixed top-3 left-1/2 -translate-x-1/2 w-[calc(100%-32px)] max-w-6xl z-50 bg-white/65 backdrop-blur-xl border border-white/80 rounded-2xl h-14 flex items-center justify-between px-5 shadow-sm">
        <div className="flex items-center gap-8">
          <button
            onClick={() => navigate('/')}
            className="flex items-center gap-2 font-bold text-base text-text"
          >
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="url(#primaryGrad)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <defs><linearGradient id="primaryGrad" x1="0" y1="0" x2="1" y2="1"><stop stopColor="#2563EB"/><stop offset="1" stopColor="#3B82F6"/></linearGradient></defs>
              <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
            </svg>
            <span className="bg-gradient-to-r from-primary to-primary-light bg-clip-text text-transparent">SimpleHub</span>
          </button>
          <div className="hidden md:flex items-center gap-1">
            {navItems.map(item => (
              <button
                key={item.path}
                onClick={() => navigate(item.path)}
                className={`px-3.5 py-1.5 rounded-lg text-sm font-medium transition-all duration-200 ${
                  isActive(item.path) ? 'bg-primary-bg text-primary' : 'text-text-secondary hover:bg-primary-bg/50 hover:text-primary'
                }`}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => fileRef.current?.click()} className="inline-flex items-center gap-1.5 px-2 sm:px-3 py-1.5 rounded-lg text-sm font-medium text-text-secondary hover:bg-white/80 hover:text-primary border border-border transition-all duration-200 cursor-pointer">
            <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
            <span className="hidden sm:inline">导入</span>
          </button>
          <input ref={fileRef} type="file" accept=".json" onChange={handleImport} className="hidden" />
          <button onClick={() => { logout(); navigate('/') }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-text-secondary hover:bg-danger-bg hover:text-danger transition-all duration-200 cursor-pointer">
            <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>
            <span className="hidden sm:inline">退出</span>
          </button>
        </div>
      </nav>

      <main className="max-w-6xl mx-auto px-4 pt-20 pb-8">
        <Outlet />
      </main>

      <nav className="md:hidden fixed bottom-0 left-0 right-0 bg-white/85 backdrop-blur-xl border-t border-border z-50 pb-safe">
        <div className="flex justify-around py-1.5">
          <button onClick={() => navigate('/')} className={`flex flex-col items-center gap-0.5 px-4 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 ${isActive('/') ? 'text-primary' : 'text-text-secondary'}`}>
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
            站点
          </button>
          <button onClick={() => navigate('/dashboard')} className={`flex flex-col items-center gap-0.5 px-4 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 ${isActive('/dashboard') ? 'text-primary' : 'text-text-secondary'}`}>
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
            仪表盘
          </button>
          <button onClick={() => navigate('/settings')} className={`flex flex-col items-center gap-0.5 px-4 py-1.5 rounded-lg text-xs font-medium transition-all duration-200 ${isActive('/settings') ? 'text-primary' : 'text-text-secondary'}`}>
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 01-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>
            设置
          </button>
        </div>
      </nav>
    </div>
    </ModalHost>
  )
}
