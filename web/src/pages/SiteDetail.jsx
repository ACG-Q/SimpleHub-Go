import { useState, useEffect, useMemo, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import dayjs from 'dayjs'
import { showToast } from '../api/client'
import { Button } from '../components/ui/Button'
import { Badge, Tag } from '../components/ui/Badge'
import { StatsCard, Card } from '../components/ui/Card'
import { CollapsePanel } from '../components/ui/Collapse'
import { SearchBox } from '../components/ui/SearchBox'
import { Skeleton } from '../components/ui/Skeleton'
import { useIsMobile } from '../hooks/useIsMobile'
import { copyText } from './SitesList'
import { useSite, useSitesDiffs, useSitesSnapshots, useCheckSite, usePricing } from '../hooks/useApi'
import SiteTokens from './SiteTokens'
import SiteRedeem from './SiteRedeem'

function ModelCard({ model, pricing }) {
  return (
    <div className="bg-white/50 border border-border rounded-lg px-3.5 py-2.5 transition-all duration-200 hover:border-primary-light hover:bg-white/80 hover:shadow-sm cursor-default">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium truncate">{model.id}</span>
        <button onClick={() => copyText(model.id, `${model.id} 已复制`)} className="text-text-muted hover:text-primary transition-colors p-0.5 shrink-0" title="复制模型名"><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg></button>
      </div>
      <div className="mt-1 space-y-0.5">
        {pricing?.type === 'per_token' && <>
          <div className="text-xs text-text-muted">输入 {pricing.inputPrice}/ 1M Tokens</div>
          <div className="text-xs text-text-muted">补全 {pricing.outputPrice}/ 1M Tokens</div>
        </>}
        {pricing?.type === 'per_call' && <div className="text-xs text-text-muted">{pricing.price}/ 次</div>}
        {pricing?.type === 'per_call_with_io' && <>
          <div className="text-xs text-text-muted">输入 {pricing.inputPrice}/ 次</div>
          <div className="text-xs text-text-muted">补全 {pricing.outputPrice}/ 次</div>
        </>}
        {pricing?.type === 'free' && <div className="text-xs text-success">免费</div>}
        {pricing?.available === false && <div className="text-xs text-danger">不可用</div>}
        {!pricing && <div className="text-xs text-text-muted">-</div>}
      </div>
    </div>
  )
}

export default function SiteDetail() {
  const { id } = useParams()
  const nav = useNavigate()
  const isMobile = useIsMobile()
  const { data: siteInfo } = useSite(id)
  const { data: diffs = [] } = useSitesDiffs(id)
  const { data: snapshots = [] } = useSitesSnapshots(id)
  const { data: pricingResponse } = usePricing(id)
  const [pricingData, setPricingData] = useState(null)
  const [selectedGroup, setSelectedGroup] = useState('')
  const [availableGroups, setAvailableGroups] = useState([])
  const [modelSearchText, setModelSearchText] = useState('')
  const [expandedDiffIds, setExpandedDiffIds] = useState(new Set())
  const toggleDiff = useCallback((id) => {
    setExpandedDiffIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }, [])

  const snapshot = snapshots.length > 0
    ? (snapshots[0].modelsJson || []).filter(m => !String(m.id || '').toLowerCase().includes('custom'))
    : []

  const getModelPricing = useCallback((modelId) => {
    if (!pricingData) return null
    if (pricingData.notSupported) return { available: true, type: 'not_supported' }
    if (!pricingData.data) return null
    const m = pricingData.data.find(x => x.model_name === modelId)
    if (!m) return null
    const gr = (pricingData.group_ratio && pricingData.group_ratio[selectedGroup]) || 1
    const avail = m.enable_groups && m.enable_groups.includes(selectedGroup)
    if (!avail) return { available: false }
    if (m.quota_type === 1) {
      if (m._donehub_input_price !== undefined && m._donehub_output_price !== undefined) {
        const ip = m._donehub_input_price * gr, op = m._donehub_output_price * gr
        if (ip === 0 && op === 0) return { available: true, type: 'free' }
        return { available: true, type: 'per_call_with_io', inputPrice: ip.toFixed(3), outputPrice: op.toFixed(3) }
      }
      if (m.model_price) {
        const p = m.model_price * gr
        if (p === 0) return { available: true, type: 'free' }
        return { available: true, type: 'per_call', price: p.toFixed(3) }
      }
    } else if (m.quota_type === 0 && m.model_ratio) {
      const ip = 2 * m.model_ratio * gr, op = ip * (m.completion_ratio || 1)
      if (ip === 0 && op === 0) return { available: true, type: 'free' }
      return { available: true, type: 'per_token', inputPrice: ip.toFixed(2), outputPrice: op.toFixed(2) }
    }
    return { available: true, type: 'free' }
  }, [pricingData, selectedGroup])

  const copyAllModels = useCallback((models) => {
    if (!models?.length) { showToast('没有可复制的模型', 'warning'); return }
    copyText(models.map(m => m.id).join(','), `已复制 ${models.length} 个模型名称`)
  }, [])

  useEffect(() => {
    const apiType = siteInfo?.type || siteInfo?.apiType
    if (!apiType) return

    if (apiType !== 'newapi' && apiType !== 'veloera' && apiType !== 'donehub' && apiType !== 'voapi') {
      if (apiType === 'other') setPricingData({ notSupported: true })
      return
    }
    if (!pricingResponse) return

    const data = pricingResponse
    if (data.code === 0 && apiType === 'voapi' && data.data) {
      const { models = [], groups: gs = [] } = data.data
      const allGroups = new Set(); const groupMap = {}
      gs.forEach(g => { allGroups.add(g.id); groupMap[g.id] = { name: g.name || `分组${g.id}`, ratio: g.ratio || 1 } })
      const nm = models.map(m => {
        const mg = m.ac || []; mg.forEach(id => allGroups.add(id))
        return { model_name: m.idKey, quota_type: m.chargingType === 1 ? 0 : 1, model_price: m.chargingType !== 1 ? parseFloat(m.singlePrice || 0) : null, model_ratio: m.chargingType === 1 ? parseFloat(m.inputPrice || 0) / 2 : null, completion_ratio: m.chargingType === 1 && parseFloat(m.inputPrice || 0) > 0 ? parseFloat(m.outputPrice || 0) / parseFloat(m.inputPrice || 0) : 1, enable_groups: mg.map(String) }
      })
      const ug = {}; const gr = {}
      allGroups.forEach(gid => { const gi = groupMap[gid]; ug[gid] = gi ? gi.name : `分组${gid}`; gr[gid] = gi ? gi.ratio : 1 })
      setPricingData({ success: true, data: nm, usable_group: ug, group_ratio: gr })
      const gl = Array.from(allGroups).map(String)
      setAvailableGroups(gl); setSelectedGroup(gl.includes('1') ? '1' : gl[0])
    } else if (data.success) {
      if (apiType === 'donehub' && data.data) {
        const allGroups = new Set()
        const nm = Object.entries(data.data).map(([name, info]) => {
          const gs = info.groups || ['default']; gs.forEach(g => allGroups.add(g))
          const isPT = info.price?.type === 'tokens'
          if (isPT) {
            const ip = (info.price.input || 0) * 2, op = (info.price.output || 0) * 2
            return { model_name: name, quota_type: 0, model_price: null, model_ratio: ip / 2, completion_ratio: ip > 0 ? op / ip : 1, enable_groups: gs }
          }
          const ip = (info.price.input || 0) * 0.002, op = (info.price.output || 0) * 0.002
          return { model_name: name, quota_type: 1, model_price: (ip + op) / 2, completion_ratio: 1, enable_groups: gs, _donehub_input_price: ip, _donehub_output_price: op }
        })
        const ug = {}; const gr = {}
        allGroups.forEach(g => { ug[g] = g === 'default' ? '默认分组' : g; gr[g] = 1 })
        setPricingData({ success: true, data: nm, usable_group: ug, group_ratio: gr })
        const gl = Array.from(allGroups)
        setAvailableGroups(gl); setSelectedGroup(gl.includes('default') ? 'default' : gl[0])
      } else if (data.usable_group) {
        setPricingData(data)
        const gl = Object.keys(data.usable_group || {}).filter(k => k !== '')
        setAvailableGroups(gl); setSelectedGroup(gl.includes('default') ? 'default' : gl[0])
      } else if (Array.isArray(data.data)) {
        const allGroups = new Set()
        data.data.forEach(m => (m.enable_groups || []).forEach(g => allGroups.add(g)))
        const ug = {}; const gr = {}
        allGroups.forEach(g => { ug[g] = g; gr[g] = 1 })
        data.usable_group = ug; data.group_ratio = gr
        setPricingData(data)
        const gl = Array.from(allGroups)
        setAvailableGroups(gl); setSelectedGroup(gl.includes('default') ? 'default' : gl[0])
      }
    }
  }, [pricingResponse, siteInfo])

  const checkMutation = useCheckSite()

  useEffect(() => {
    if (diffs.length > 0) {
      setExpandedDiffIds(prev => new Set([diffs[0].id, ...prev]))
    }
  }, [diffs])

  const initialLoading = !siteInfo && diffs.length === 0 && snapshot.length === 0
  const loading = checkMutation.isPending

  const checkNow = useCallback(() => {
    checkMutation.mutate({ id }, {
      onSuccess: () => showToast('检测完成，数据已刷新'),
      onError: (e) => showToast(e.message || '检测失败', 'error'),
    })
  }, [id, checkMutation])



  const filteredSnapshot = useMemo(() => {
    if (!modelSearchText.trim()) return snapshot
    const q = modelSearchText.toLowerCase()
    return snapshot.filter(m => m.id?.toLowerCase().includes(q))
  }, [snapshot, modelSearchText])

  const totalAdded = useMemo(() => diffs.reduce((s, d) => s + (d.addedJson?.length || 0), 0), [diffs])
  const totalRemoved = useMemo(() => diffs.reduce((s, d) => s + (d.removedJson?.length || 0), 0), [diffs])

  return (
    <div>
      <button onClick={() => nav('/')} className="inline-flex items-center gap-1.5 text-sm text-text-secondary font-medium mb-4 px-3 py-1.5 rounded-lg hover:bg-primary-bg hover:text-primary transition-all duration-200 cursor-pointer anim-in">
        <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="15 18 9 12 15 6"/></svg>
        返回站点列表
      </button>

      <div className="flex items-center justify-between mb-5 flex-wrap gap-3 anim-in anim-in-d1">
        <div><h1 className="text-xl font-bold tracking-tight">{siteInfo?.name || '站点详情'}</h1><p className="text-sm text-text-secondary">{siteInfo?.baseUrl} &bull; {siteInfo?.apiType || siteInfo?.type}</p></div>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" onClick={checkNow} disabled={loading}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>{loading ? '检测中...' : '检测'}</Button>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-3 mb-5 anim-in anim-in-d2">
        <StatsCard icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>} label="模型数" value={snapshot.length} color="blue" />
        <StatsCard icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>} label="新增" value={totalAdded} change="本周累计" changeType="up" color="green" />
        <StatsCard icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 18 13.5 8.5 8.5 13.5 1 6"/><polyline points="17 18 23 18 23 12"/></svg>} label="移除" value={totalRemoved} change="本周累计" changeType="down" color="orange" />
      </div>

      <div className="grid grid-cols-2 gap-3 mb-5 anim-in anim-in-d3">
        <SiteTokens siteId={id} siteInfo={siteInfo} />
        <SiteRedeem siteId={id} />
      </div>

      <Card className="mb-5 anim-in anim-in-d4">
        <div className="flex items-center justify-between px-5 py-4 flex-wrap gap-3">
          <div className="flex items-center gap-2"><svg className="w-5 h-5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg><span className="font-bold text-sm">当前模型</span><Badge color="blue">{snapshot.length} 个模型</Badge></div>
          <div className="flex items-center gap-2 flex-wrap">
            <SearchBox value={modelSearchText} onChange={e => setModelSearchText(e.target.value)} placeholder="搜索模型..." className="w-32 md:w-44" />
            {availableGroups.length > 1 && pricingData && <select value={selectedGroup} onChange={e => setSelectedGroup(e.target.value)} className="px-3 py-2 rounded-lg border border-border bg-white/80 text-sm outline-none focus:border-primary-light"><option value="">选择分组</option>{availableGroups.map(g => <option key={g} value={g}>{pricingData.usable_group?.[g] || g}</option>)}</select>}
            <Button variant="ghost" size="sm" onClick={() => copyAllModels(filteredSnapshot)} disabled={!filteredSnapshot.length}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>复制全部</Button>
            <Button variant="primary" size="sm" onClick={checkNow} disabled={loading}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>{loading ? '检测中...' : '检测'}</Button>

          </div>
        </div>
        {initialLoading ? <div className="px-5 pb-5"><div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-2"><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" /></div></div> : filteredSnapshot.length === 0 ? <div className="py-12 text-center text-sm text-text-muted"><svg className="w-12 h-12 mx-auto mb-3 opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg><p>{snapshot.length === 0 ? '暂无模型数据，请先执行检测' : '未找到匹配的模型'}</p></div> : <div className="px-5 pb-5"><div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-2">{filteredSnapshot.map(m => <ModelCard key={m.id} model={m} pricing={getModelPricing(m.id)} />)}</div></div>}
      </Card>

      <Card className="mb-5">
        <div className="px-5 py-4"><span className="font-bold text-sm">变更历史记录</span></div>
        <div className="px-5 pb-5">
          {diffs.length === 0 ? <div className="py-8 text-center text-sm text-text-muted"><svg className="w-10 h-10 mx-auto mb-2 opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg><p>暂无变更记录</p></div> :
          <div className="space-y-2">{diffs.map(d => <CollapsePanel key={d.id} open={expandedDiffIds.has(d.id)} onToggle={() => toggleDiff(d.id)} title={<><span className="font-semibold">{dayjs(d.diffAt).format('YYYY-MM-DD HH:mm:ss')}</span><Badge color="green">+{d.addedJson?.length || 0}</Badge><Badge color="red">-{d.removedJson?.length || 0}</Badge></>}><div className="space-y-3"><div><span className="text-xs text-success font-semibold block mb-1">新增模型 ({d.addedJson?.length || 0})</span>{d.addedJson?.length ? <div className="flex flex-wrap gap-1">{d.addedJson.slice(0, 20).map((m, i) => <Tag key={m.id} color="green">{m.id}</Tag>)}{d.addedJson.length > 20 && <span className="text-xs text-text-muted">...还有 {d.addedJson.length - 20} 个</span>}</div> : <span className="text-xs text-text-muted">无</span>}</div><div><span className="text-xs text-danger font-semibold block mb-1">移除模型 ({d.removedJson?.length || 0})</span>{d.removedJson?.length ? <div className="flex flex-wrap gap-1">{d.removedJson.slice(0, 20).map((m, i) => <Tag key={m.id} color="red">{m.id}</Tag>)}{d.removedJson.length > 20 && <span className="text-xs text-text-muted">...还有 {d.removedJson.length - 20} 个</span>}</div> : <span className="text-xs text-text-muted">无</span>}</div></div></CollapsePanel>)}</div>}
        </div>
      </Card>
    </div>
  )
}
