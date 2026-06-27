import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import dayjs from 'dayjs'
import { showToast } from '../api/client'
import { Button } from '../components/ui/Button'
import { Badge, StatusDot, Tag } from '../components/ui/Badge'
import { ProgressBar } from '../components/ui/Progress'
import { Table, THead, Th, TBody, Tr, Td } from '../components/ui/Table'
import { CollapsePanel } from '../components/ui/Collapse'

export function sortSites(sites) {
  return [...sites].sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
    const d = (a.sortOrder ?? 0) - (b.sortOrder ?? 0)
    if (d !== 0) return d
    return (new Date(b.createdAt || 0).getTime()) - (new Date(a.createdAt || 0).getTime())
  })
}

export function cronToHourMin(cron) {
  if (!cron) return {}
  const p = String(cron).trim().split(/\s+/)
  if (p.length >= 2) return { h: Number(p[1]), m: Number(p[0]) }
  return {}
}

export function copyText(text, msg = '复制成功') {
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(() => showToast(msg)).catch(() => showToast('复制失败', 'error'))
  } else {
    const ta = document.createElement('textarea')
    ta.value = text; ta.style.cssText = 'position:fixed;left:-9999px'
    document.body.appendChild(ta); ta.select()
    try { document.execCommand('copy'); showToast(msg) }
    catch { showToast('复制失败', 'error') }
    finally { document.body.removeChild(ta) }
  }
}

function renderUsage(record) {
  const { billingLimit, billingUsage, billingError, unlimitedQuota } = record
  if (unlimitedQuota) return <Badge color="orange">无限余额</Badge>
  if (billingError) return <span className="text-text-muted text-xs">无法获取</span>
  if (typeof billingLimit === 'number' && typeof billingUsage === 'number') {
    const pct = (billingUsage / billingLimit) * 100
    const color = pct > 90 ? 'text-danger' : pct > 75 ? 'text-cta' : 'text-success'
    return (
      <div className="w-full max-w-[140px]">
        <div className="flex justify-between items-center mb-1">
          <span className="text-xs text-text-secondary">剩余</span>
          <span className={`text-sm font-bold ${color}`}>${(billingLimit - billingUsage).toFixed(2)}</span>
        </div>
        <ProgressBar value={billingUsage} max={billingLimit} />
        <div className="flex justify-between text-[10px] text-text-muted mt-1">
          <span>已用 ${billingUsage.toFixed(1)}</span>
          <span>总额 ${billingLimit.toFixed(2)}</span>
        </div>
      </div>
    )
  }
  if (typeof billingUsage === 'number') return <Badge color="green">${billingUsage.toFixed(2)}</Badge>
  return null
}

function renderCheckIn(record) {
  if ((record.apiType === 'veloera' || record.apiType === 'newapi' || record.apiType === 'voapi') && record.enableCheckIn) {
    if (record.checkInSuccess === true) return <StatusDot status="green" />
    if (record.checkInSuccess === false) return <StatusDot status="red" />
    return <StatusDot status="yellow" />
  }
  return null
}

export default function SitesList({
  sites, categories, collapsedGroups, isMobile, searchKeyword, categoryCheckingId,
  onCheck, onDelete, updateSortOrder, openEditModal, openDebugModal, onOpenTimeModal,
  toggleCollapse, checkCategory, checkGroup, deleteCategory, openEditCategoryModal,
}) {
  const nav = useNavigate()

  const { pinnedSites, uncategorizedSites, categorySitesMap } = useMemo(() => {
    const p = []; const u = []; const cm = new Map(categories.map(c => [c.id, []]))
    for (const s of sites) {
      if (s.pinned) { p.push(s); continue }
      if (s.categoryId && cm.has(s.categoryId)) { cm.get(s.categoryId).push(s); continue }
      u.push(s)
    }
    return { pinnedSites: p, uncategorizedSites: u, categorySitesMap: cm }
  }, [sites, categories])

  const columns = useMemo(() => [
    { key: 'sort', label: '排序', width: 'w-14', render: (r) => (
      <input className="w-11 px-1 py-0.5 text-center text-xs rounded-md border border-border bg-white/80 outline-none focus:border-primary-light"
        defaultValue={r.sortOrder ?? 0}
        onBlur={(e) => { const v = parseInt(e.target.value, 10) || 0; if (v !== (r.sortOrder ?? 0)) updateSortOrder(r.id, v) }}
        onKeyDown={(e) => e.key === 'Enter' && e.target.blur()} />
    )},
    { key: 'name', label: '名称', render: (r) => (
      <div className="flex flex-col gap-1 min-w-0">
        <div className="flex items-center gap-1.5 min-w-0">
          <span className={`w-2 h-2 rounded-full shrink-0 ${r.lastCheckStatus === 'error' ? 'bg-danger shadow-[0_0_6px_rgba(239,68,68,0.4)]' : 'bg-success shadow-[0_0_6px_rgba(16,185,129,0.4)]'}`} />
          <a className="font-semibold text-sm text-text truncate hover:text-primary transition-colors" href={r.baseUrl} target="_blank" rel="noopener">{r.name}</a>
          {r.pinned && <Tag color="orange">置顶</Tag>}
          {r.excludeFromBatch && <Tag color="red">排除</Tag>}
        </div>
        {r.extralink && <a href={r.extralink} target="_blank" rel="noopener" className="text-xs text-text-muted truncate hover:text-primary">{r.extralink}</a>}
      </div>
    )},
    { key: 'usage', label: '用量', render: (r) => renderUsage(r) },
    { key: 'checkin', label: '签到', render: (r) => renderCheckIn(r) },
    { key: 'schedule', label: '定时计划', render: (r) => {
      const hm = cronToHourMin(r.scheduleCron)
      return hm.h != null ? <Badge color="green">{String(hm.h).padStart(2,'0')}:{String(hm.m).padStart(2,'0')}</Badge>
        : r.scheduleCron ? <Badge color="green">个体</Badge> : <Badge color="blue">全局</Badge>
    }},
    { key: 'lastCheck', label: '上次检测', render: (r) => (
      <span className="text-xs text-text-secondary">{r.lastCheckedAt ? dayjs(r.lastCheckedAt).format('MM-DD HH:mm') : '未检测'}</span>
    )},
    { key: 'remark', label: '备注', render: (r) => (
      <span className="text-xs text-text-muted truncate max-w-[120px] inline-block">{r.remark}</span>
    )},
    { key: 'actions', label: '操作', render: (r) => (
      <div className="flex gap-0.5">
        <button onClick={() => nav(`/sites/${r.id}`)} className="w-7 h-7 inline-flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs" title="查看">
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
        </button>
        <button onClick={() => onCheck(r.id)} className="w-7 h-7 inline-flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs" title="检测">
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
        </button>
        <button onClick={() => openEditModal(r)} className="w-7 h-7 inline-flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs" title="编辑">
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
        </button>
        <button onClick={() => onOpenTimeModal(r)} className="w-7 h-7 inline-flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs" title="定时">
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        </button>
        <button onClick={() => openDebugModal(r)} className="w-7 h-7 inline-flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs" title="调试">
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        </button>
        <button onClick={() => { if (window.confirm('确定删除？')) onDelete(r) }} className="w-7 h-7 inline-flex items-center justify-center rounded-md text-text-secondary hover:bg-danger-bg hover:text-danger transition-colors text-xs" title="删除">
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
        </button>
      </div>
    )},
  ], [nav, onCheck, onDelete, updateSortOrder, openEditModal, openDebugModal, onOpenTimeModal])

  const renderMobile = (sites, title, id) => {
    if (!sites?.length) return null
    const collapsed = collapsedGroups.has(id)
    return (
      <div className="mb-4">
        <div onClick={() => toggleCollapse(id)}
          className="flex items-center gap-2 px-4 py-3 rounded-xl bg-white/70 backdrop-blur-sm border border-border-glass shadow-sm cursor-pointer select-none transition-colors hover:bg-primary-bg/20 mb-2">
          <svg className={`w-3 h-3 text-text-muted transition-transform duration-200 shrink-0 ${collapsed ? '' : 'rotate-90'}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"><polyline points="9 18 15 12 9 6"/></svg>
          <span className="text-sm font-semibold flex-1">{title}</span>
          <Badge color="blue">{sites.length}</Badge>
        </div>
        {!collapsed && (
          <div className="space-y-2 pl-2">
            {sites.map(s => (
              <div key={s.id} className="bg-white/60 backdrop-blur-sm border border-border-glass rounded-xl p-3">
                <div className="flex justify-between items-start mb-2">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5">
                      <span className={`w-2 h-2 rounded-full shrink-0 ${s.lastCheckStatus === 'error' ? 'bg-danger' : 'bg-success'}`} />
                      <span className="text-sm font-semibold truncate">{s.name}</span>
                      {s.pinned && <Tag color="orange">P</Tag>}
                    </div>
                    <span className="text-[10px] text-text-muted">{s.apiType?.toUpperCase()}</span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">{renderUsage(s)}{renderCheckIn(s)}</div>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-[10px] text-text-muted">{s.lastCheckedAt ? dayjs(s.lastCheckedAt).format('MM-DD HH:mm') : '未检测'}</span>
                  <div className="flex gap-1">
                    <button onClick={() => nav(`/sites/${s.id}`)} className="w-7 h-7 flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs">
                      <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                    </button>
                    <button onClick={() => onCheck(s.id)} className="w-7 h-7 flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs">
                      <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
                    </button>
                    <button onClick={() => openEditModal(s)} className="w-7 h-7 flex items-center justify-center rounded-md text-text-secondary hover:bg-primary-bg hover:text-primary transition-colors text-xs">
                      <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                    </button>
                    <button onClick={() => { if (window.confirm('确定删除？')) onDelete(s) }} className="w-7 h-7 flex items-center justify-center rounded-md text-text-secondary hover:bg-danger-bg hover:text-danger transition-colors text-xs">
                      <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-3 anim-in anim-in-d2">
      {pinnedSites.length > 0 && (isMobile ? renderMobile(pinnedSites, '置顶站点', 'pinned') : (
        <CollapsePanel title={<><span className="bg-gradient-to-r from-cta to-cta-light bg-clip-text text-transparent">置顶站点</span><Badge color="orange">{pinnedSites.length}</Badge></>}
          defaultOpen={!collapsedGroups.has('pinned')}
          extra={<div className="flex gap-1"><Button variant="ghost" size="sm" onClick={() => checkGroup('pinned', '置顶')}><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>检测</Button></div>}
          onClick={() => toggleCollapse('pinned')}>
          <Table><THead>{columns.map(c => <Th key={c.key} className={c.width}>{c.label}</Th>)}</THead><TBody>{pinnedSites.map(r => <Tr key={r.id}>{columns.map(c => <Td key={c.key}>{c.render(r)}</Td>)}</Tr>)}</TBody></Table>
        </CollapsePanel>
      ))}

      {categories.map(cat => {
        const items = categorySitesMap.get(cat.id) || []
        if (!items.length && !searchKeyword) return null
        return isMobile ? renderMobile(items, cat.name, cat.id) : (
          <CollapsePanel key={cat.id}
            title={<><span>{cat.name}</span><Badge color="blue">{items.length}</Badge></>}
            defaultOpen={!collapsedGroups.has(cat.id)}
            extra={<div className="flex gap-1">
              <Button variant="ghost" size="sm" onClick={() => checkCategory(cat.id, cat.name)} disabled={categoryCheckingId === cat.id}><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>检测</Button>
              <Button variant="ghost" size="sm" onClick={() => openEditCategoryModal(cat)}><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>编辑</Button>
              <Button variant="ghost" size="sm" onClick={() => deleteCategory(cat.id)}><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>删除</Button>
            </div>}
            onClick={() => toggleCollapse(cat.id)}>
            {items.length ? <Table><THead>{columns.map(c => <Th key={c.key} className={c.width}>{c.label}</Th>)}</THead><TBody>{items.map(r => <Tr key={r.id}>{columns.map(c => <Td key={c.key}>{c.render(r)}</Td>)}</Tr>)}</TBody></Table>
              : <div className="py-8 text-center text-sm text-text-muted">该分类下暂无站点</div>}
          </CollapsePanel>
        )
      })}

      {uncategorizedSites.length > 0 && (isMobile ? renderMobile(uncategorizedSites, '未分类', 'uncategorized') : (
        <CollapsePanel title={<><span className="text-text-muted">未分类</span><Badge color="gray">{uncategorizedSites.length}</Badge></>}
          defaultOpen={!collapsedGroups.has('uncategorized')}
          extra={<div className="flex gap-1"><Button variant="ghost" size="sm" onClick={() => checkGroup('uncategorized', '未分类')}><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>检测</Button></div>}
          onClick={() => toggleCollapse('uncategorized')}>
          <Table><THead>{columns.map(c => <Th key={c.key} className={c.width}>{c.label}</Th>)}</THead><TBody>{uncategorizedSites.map(r => <Tr key={r.id}>{columns.map(c => <Td key={c.key}>{c.render(r)}</Td>)}</Tr>)}</TBody></Table>
        </CollapsePanel>
      ))}
    </div>
  )
}
