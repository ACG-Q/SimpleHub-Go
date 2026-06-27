import { useState, useCallback } from 'react'
import { showToast } from '../api/client'
import { useRedeemCode } from '../hooks/useApi'
import { Button } from '../components/ui/Button'
import { Textarea } from '../components/ui/Input'
import { Modal } from '../components/ui/Modal'

export default function SiteRedeem({ siteId }) {
  const [visible, setVisible] = useState(false)
  const [codes, setCodes] = useState('')
  const [loading, setLoading] = useState(false)
  const [results, setResults] = useState([])
  const redeemMutation = useRedeemCode(siteId)

  const handleRedeem = useCallback(async () => {
    if (!codes.trim()) { showToast('请输入兑换码', 'warning'); return }
    setLoading(true)
    const list = codes.split('\n').map(c => c.trim()).filter(Boolean)
    const res = []
    for (const code of list) {
      try {
        const data = await redeemMutation.mutateAsync(code)
        res.push({ code, success: data.success, message: data.message || (data.success ? '兑换成功' : '兑换失败') })
      } catch (e) { res.push({ code, success: false, message: e.message }) }
    }
    setResults(res)
    const ok = res.filter(r => r.success).length, fail = res.length - ok
    if (ok > 0 && fail === 0) showToast(`全部兑换成功！成功 ${ok} 个`)
    else if (ok > 0) showToast(`部分兑换成功：成功 ${ok} 个，失败 ${fail} 个`, 'warning')
    else showToast(`全部兑换失败！失败 ${fail} 个`, 'error')
    setLoading(false)
  }, [siteId, codes])

  return (
    <>
      <div onClick={() => { setVisible(true); setCodes(''); setResults([]) }}
        className="p-4 md:p-6 rounded-xl bg-gradient-to-br from-teal-500/10 to-teal-400/5 border border-teal-500/15 cursor-pointer transition-all duration-200 hover:shadow-lg hover:-translate-y-0.5 flex flex-col items-center justify-center gap-2 md:gap-3">
        <svg className="w-7 h-7 md:w-10 md:h-10 text-teal-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
        <span className="font-bold text-sm md:text-lg text-teal-600">兑换码</span>
        <span className="text-xs text-text-secondary hidden md:block">使用兑换码充值余额</span>
      </div>

      <Modal open={visible} onClose={() => setVisible(false)} title="兑换码" size="md"
        footer={<><Button variant="ghost" onClick={() => setVisible(false)}>关闭</Button><Button variant="primary" onClick={handleRedeem} disabled={loading}>{loading ? '处理中...' : '批量兑换'}</Button></>}>
        <div className="space-y-4">
          <Textarea label="输入兑换码（一行一个）" value={codes} onChange={e => setCodes(e.target.value)} placeholder="兑换码1&#10;兑换码2&#10;兑换码3" rows={5} />
          {results.length > 0 && <div className="space-y-1 max-h-60 overflow-y-auto">{results.map((r, i) => <div key={r.code + i} className={`flex items-center gap-2 text-sm p-2 rounded-lg ${r.success ? 'bg-success-bg' : 'bg-danger-bg'}`}><span className={r.success ? 'text-success' : 'text-danger'}>{r.success ? '✅' : '❌'}</span><span className="font-medium truncate">{r.code}</span><span className="text-text-muted text-xs shrink-0">{r.message}</span></div>)}</div>}
        </div>
      </Modal>
    </>
  )
}
