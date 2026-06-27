import { useEffect, useState, useCallback, useMemo, useRef, startTransition } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { showToast } from '../api/client'
import { Button } from '../components/ui/Button'
import { Card, StatsCard } from '../components/ui/Card'
import { SearchBox } from '../components/ui/SearchBox'
import { useIsMobile } from '../hooks/useIsMobile'
import { cronToHourMin } from './SitesList'
import SitesList from './SitesList'
import { Modal } from '../components/ui/Modal'
import {
  useSites, useSite, useCategories,
  useCreateSite, useUpdateSite, useDeleteSite, useCheckSite,
  useBatchCheckCategory,
  useUpdateSortOrder as useUpdateSortOrderMutation, useUpdateSchedule,
  useCreateCategory, useUpdateCategory, useDeleteCategory,
  useLatestSnapshot,
} from '../hooks/useApi'
import {
  SiteEditModal, TimeModal, DebugModal,
  BatchResultModal, CategoryModal, CategoryManageModal,
} from './SitesModals'
import SitesBatchCheck from './SitesBatchCheck'

export default function Sites() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [searchKeyword, setSearchKeyword] = useState('')
  const { data: list = [] } = useSites(searchKeyword)
  const { data: categories = [] } = useCategories()

  const [open, setOpen] = useState(false)
  const [editMode, setEditMode] = useState(false)
  const [editingSite, setEditingSite] = useState(null)
  const [formData, setFormData] = useState({})
  const [timeOpen, setTimeOpen] = useState(false)
  const [timeSite, setTimeSite] = useState(null)
  const [timeValue, setTimeValue] = useState('')
  const [debugOpen, setDebugOpen] = useState(false)
  const [debugData, setDebugData] = useState(null)
  const [debugLoading, setDebugLoading] = useState(false)
  const [batchChecking, setBatchChecking] = useState(false)
  const [batchProgress, setBatchProgress] = useState({ current: 0, total: 0, currentSite: '' })
  const [batchResultOpen, setBatchResultOpen] = useState(false)
  const [batchResults, setBatchResults] = useState({ changes: [], failures: [], timestamp: null, totalSites: 0 })
  const [expandedSites, setExpandedSites] = useState(new Set())
  const [hasLastResult, setHasLastResult] = useState(false)
  const [billingExpanded, setBillingExpanded] = useState(false)
  const [catOpen, setCatOpen] = useState(false)
  const [catName, setCatName] = useState('')
  const [catTime, setCatTime] = useState('')
  const [editingCategory, setEditingCategory] = useState(null)
  const [categoryCheckingId, setCategoryCheckingId] = useState(null)
  const [categoryManageOpen, setCategoryManageOpen] = useState(false)
  const [diffModal, setDiffModal] = useState({ open: false, title: '', html: '' })
  const isMobile = useIsMobile()
  const [editingSiteId, setEditingSiteId] = useState(null)
  const [editingSiteFromList, setEditingSiteFromList] = useState(null)
  const { data: editSiteDetail, error: editSiteError, isError: isEditSiteError } = useSite(editingSiteId)
  const [debuggingSiteId, setDebuggingSiteId] = useState(null)
  const [debuggingSiteFromList, setDebuggingSiteFromList] = useState(null)
  const { data: latestSnapshot, error: snapshotError, isError: isSnapshotError } = useLatestSnapshot(debuggingSiteId)
  const [collapsedGroups, setCollapsedGroups] = useState(() => {
    try { return new Set(JSON.parse(sessionStorage.getItem('sitesCollapsedGroups') || '[]')) }
    catch { return new Set(['pinned', 'uncategorized']) }
  })
  const listRef = useRef(list)
  const searchRef = useRef(searchKeyword)
  const collapsedRef = useRef(collapsedGroups)
  const createSite = useCreateSite()
  const updateSite = useUpdateSite()
  const deleteSite = useDeleteSite()
  const checkSite = useCheckSite()
  const batchCheckCategory = useBatchCheckCategory()
  const updateSortOrderMutation = useUpdateSortOrderMutation()
  const updateSchedule = useUpdateSchedule()
  const createCategory = useCreateCategory()
  const updateCategory = useUpdateCategory()
  const deleteCategoryMutation = useDeleteCategory()
  useEffect(() => { listRef.current = list }, [list])
  useEffect(() => { searchRef.current = searchKeyword }, [searchKeyword])
  useEffect(() => {
    collapsedRef.current = collapsedGroups
    sessionStorage.setItem('sitesCollapsedGroups', JSON.stringify([...collapsedGroups]))
  }, [collapsedGroups])

  useEffect(() => {
    checkLastBatchResult()
    const sp = sessionStorage.getItem('sitesScrollPosition')
    if (sp) {
      const y = parseInt(sp, 10)
      requestAnimationFrame(() => { window.scrollTo(0, y); sessionStorage.removeItem('sitesScrollPosition') })
    }
  }, [])

  useEffect(() => {
    if (categories.length > 0) {
      const saved = sessionStorage.getItem('sitesCollapsedGroups')
      if (!saved) setCollapsedGroups(new Set(['pinned', 'uncategorized', ...categories.map(c => c.id)]))
    }
  }, [categories.length])

  useEffect(() => {
    if (isEditSiteError && editSiteError) {
      showToast(editSiteError.message || '获取站点详情失败', 'error')
      setEditingSiteId(null)
      setEditingSiteFromList(null)
    }
  }, [isEditSiteError, editSiteError])

  useEffect(() => {
    if (editSiteDetail && editingSiteFromList) {
      const cur = { ...editingSiteFromList, ...editSiteDetail }
      setEditMode(true)
      setEditingSite(cur)
      const hm = cronToHourMin(cur.scheduleCron)
      setFormData({
        name: cur.name, baseUrl: cur.baseUrl, apiKey: '', apiType: cur.apiType || 'newapi',
        userId: cur.userId || '', scheduleTime: hm.h ? `${String(hm.h).padStart(2,'0')}:${String(hm.m).padStart(2,'0')}` : '',
        pinned: !!cur.pinned, excludeFromBatch: !!cur.excludeFromBatch,
        categoryId: cur.categoryId || '', unlimitedQuota: !!cur.unlimitedQuota,
        billingUrl: cur.billingUrl || '', billingAuthType: cur.billingAuthType || 'token',
        billingAuthValue: cur.billingAuthValue || '', proxyUrl: cur.proxyUrl || '',
        billingLimitField: cur.billingLimitField || '', billingUsageField: cur.billingUsageField || '',
        enableCheckIn: !!cur.enableCheckIn, checkInMode: cur.checkInMode || 'both',
        extralink: cur.extralink || '', remark: cur.remark || '', scheduleCron: cur.scheduleCron,
      })
      setOpen(true)
      setEditingSiteId(null)
      setEditingSiteFromList(null)
    }
  }, [editSiteDetail, editingSiteFromList])

  useEffect(() => {
    if (isSnapshotError && snapshotError) {
      showToast(snapshotError.message || '获取快照失败', 'error')
      setDebugOpen(false)
      setDebuggingSiteId(null)
      setDebuggingSiteFromList(null)
    }
  }, [isSnapshotError, snapshotError])

  useEffect(() => {
    if (latestSnapshot && debuggingSiteFromList) {
      const site = debuggingSiteFromList
      setDebugData({
        ...latestSnapshot,
        models: latestSnapshot.modelsJson,
        httpStatus: latestSnapshot.statusCode,
        error: latestSnapshot.errorMessage,
        billing: latestSnapshot.billingError ? { limit: latestSnapshot.billingLimit, usage: latestSnapshot.billingUsage, error: latestSnapshot.billingError } : null,
        siteName: site.name,
        siteUrl: site.baseUrl,
      })
      setDebugLoading(false)
      setDebuggingSiteId(null)
      setDebuggingSiteFromList(null)
    }
  }, [latestSnapshot, debuggingSiteFromList])

  const catLookup = useMemo(() => new Map(categories.map(c => [c.id, c])), [categories])

  const { pinnedSites, uncategorizedSites, categorySitesMap } = useMemo(() => {
    const p = []; const u = []; const cm = new Map(categories.map(c => [c.id, []]))
    for (const s of list) {
      if (s.pinned) { p.push(s); continue }
      if (s.categoryId && cm.has(s.categoryId)) { cm.get(s.categoryId).push(s); continue }
      u.push(s)
    }
    return { pinnedSites: p, uncategorizedSites: u, categorySitesMap: cm }
  }, [list, categories])

  const handleSearch = (v) => { setSearchKeyword(v) }

  const toggleCollapse = useCallback((id) => {
    startTransition(() => setCollapsedGroups(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    }))
  }, [])

  const checkLastBatchResult = () => {
    try { setHasLastResult(!!localStorage.getItem('lastBatchCheckResult')) }
    catch { setHasLastResult(false) }
  }

  const loadLastBatchResult = () => {
    try {
      const saved = localStorage.getItem('lastBatchCheckResult')
      if (saved) { setBatchResults(JSON.parse(saved)); setExpandedSites(new Set()); setBatchResultOpen(true) }
      else showToast('没有历史检测结果', 'info')
    } catch { showToast('加载历史结果失败', 'error') }
  }

  const openEditModal = useCallback((site) => {
    setEditingSiteFromList(site)
    setEditingSiteId(site.id)
  }, [])

  const onAdd = async () => {
    try {
      const v = { ...formData }
      if (v.scheduleTime) { const [h, m] = v.scheduleTime.split(':'); v.scheduleCron = `${m} ${h} * * *`; v.timezone = 'Asia/Shanghai' }
      delete v.scheduleTime
      if (!v.apiType) v.apiType = 'newapi'
      if (v.enableCheckIn && !v.checkInMode) v.checkInMode = 'both'
      v.pinned = !!v.pinned; v.excludeFromBatch = !!v.excludeFromBatch; v.unlimitedQuota = !!v.unlimitedQuota; v.enableCheckIn = !!v.enableCheckIn
      if (v.apiType === 'voapi') v.billingAuthValue = null
      else if (typeof v.billingAuthValue === 'string') v.billingAuthValue = v.billingAuthValue.trim() || null
      v.proxyUrl = v.proxyUrl?.trim() || null
      await createSite.mutateAsync(v)
      setOpen(false); setFormData({}); showToast('站点创建成功')
    } catch (e) { showToast(e.message || '创建失败', 'error') }
  }

  const onEdit = async () => {
    try {
      const v = { ...formData }
      const payload = {
        id: editingSite.id, name: v.name, baseUrl: v.baseUrl, apiType: v.apiType || 'newapi', userId: v.userId || null,
        pinned: !!v.pinned, excludeFromBatch: !!v.excludeFromBatch, unlimitedQuota: !!v.unlimitedQuota,
        categoryId: v.categoryId || null, billingUrl: v.billingUrl || null,
        billingAuthType: v.billingAuthType || 'token', proxyUrl: v.proxyUrl?.trim() || null,
        billingLimitField: v.billingLimitField || null, billingUsageField: v.billingUsageField || null,
        enableCheckIn: !!v.enableCheckIn, extralink: v.extralink || null, remark: v.remark || null,
      }
      if (v.apiType === 'voapi') payload.billingAuthValue = null
      else if (v.billingAuthValue?.trim()) payload.billingAuthValue = v.billingAuthValue.trim()
      if (v.enableCheckIn && v.checkInMode) payload.checkInMode = v.checkInMode
      else if (v.enableCheckIn) payload.checkInMode = 'both'
      if (v.apiKey?.trim()) payload.apiKey = v.apiKey
      if (v.scheduleTime) {
        const [h, m] = v.scheduleTime.split(':')
        payload.scheduleCron = `${m} ${h} * * *`
        payload.timezone = 'Asia/Shanghai'
      } else { payload.scheduleCron = null; payload.timezone = 'UTC' }
      await updateSite.mutateAsync(payload)
      setOpen(false); setEditMode(false); setEditingSite(null); setFormData({}); showToast('站点更新成功')
    } catch (e) { showToast(e.message || '更新失败', 'error') }
  }

  const onDelete = useCallback(async (site) => {
    try {
      await deleteSite.mutateAsync(site.id)
      showToast(`站点"${site.name}"已删除`)
    } catch (e) { showToast(e.message || '删除失败', 'error') }
  }, [deleteSite])

  const onCheck = useCallback(async (id) => {
    try {
      const data = await checkSite.mutateAsync({ id, skipNotification: true })
      if (data?.hasChanges && data.diff) {
        const site = listRef.current.find(s => s.id === id)
        const name = site?.name || '未知站点'
        const addHtml = (items) => items.slice(0, 10).map(m => `<span style="background:rgba(16,185,129,0.1);color:#10B981;padding:1px 8px;border-radius:4px;font-size:12px">${m.id}</span>`).join('')
        const remHtml = (items) => items.slice(0, 10).map(m => `<span style="background:rgba(239,68,68,0.1);color:#EF4444;padding:1px 8px;border-radius:4px;font-size:12px">${m.id}</span>`).join('')
        let html = `<div style="margin:12px 0"><strong style="font-size:15px;display:block;margin-bottom:10px">站点：${name}</strong>`
        if (data.diff.added?.length) html += `<div style="margin-bottom:10px"><span style="color:#10B981;font-weight:600">✅ 新增模型 (${data.diff.added.length}个)：</span><div style="margin-top:4px;display:flex;flex-wrap:wrap;gap:4px">${addHtml(data.diff.added)}${data.diff.added.length > 10 ? `<span style="color:#94A3B8;font-size:12px">...还有 ${data.diff.added.length - 10} 个</span>` : ''}</div></div>`
        if (data.diff.removed?.length) html += `<div style="margin-bottom:10px"><span style="color:#EF4444;font-weight:600">❌ 移除模型 (${data.diff.removed.length}个)：</span><div style="margin-top:4px;display:flex;flex-wrap:wrap;gap:4px">${remHtml(data.diff.removed)}${data.diff.removed.length > 10 ? `<span style="color:#94A3B8;font-size:12px">...还有 ${data.diff.removed.length - 10} 个</span>` : ''}</div></div>`
        html += `<span style="color:#94A3B8;font-size:11px">💡 点击"查看详情"可查看完整变更历史</span></div>`
        setDiffModal({ open: true, title: '🔄 检测到模型变更', html })
        showToast('检测完成，发现模型变更！')
      } else showToast('检测完成，无模型变更')
    } catch (e) { showToast(e.message || '检测失败', 'error') }
  }, [checkSite])

  const onCheckAll = async () => {
    const toCheck = list.filter(s => !s.excludeFromBatch)
    if (toCheck.length === 0) { showToast('没有可检测的站点', 'warning'); return }
    if (toCheck.length < list.length) showToast(`已排除 ${list.length - toCheck.length} 个站点`, 'info')
    setBatchChecking(true)
    setBatchProgress({ current: 0, total: toCheck.length, currentSite: '' })
    const changes = []; const failures = []
    for (let i = 0; i < toCheck.length; i++) {
      const s = toCheck[i]
      setBatchProgress({ current: i + 1, total: toCheck.length, currentSite: s.name })
      try {
        const data = await checkSite.mutateAsync({ id: s.id, skipNotification: true })
        if (data?.hasChanges && data.diff) changes.push({ siteName: s.name, diff: data.diff })
      } catch (e) { failures.push({ siteName: s.name, error: e.message || '检测失败' }) }
      if (i < toCheck.length - 1) await new Promise(r => setTimeout(r, 5000))
    }
    setBatchChecking(false)
    setBatchProgress({ current: 0, total: 0, currentSite: '' })
    qc.invalidateQueries({ queryKey: ['sites'] })
    const results = { changes, failures, timestamp: new Date().toISOString(), totalSites: toCheck.length }
    setBatchResults(results)
    try { localStorage.setItem('lastBatchCheckResult', JSON.stringify(results)); setHasLastResult(true) }
    catch {}
    setExpandedSites(new Set())
    setBatchResultOpen(true)
  }

  const updateSortOrder = useCallback(async (id, val) => {
    try {
      await updateSortOrderMutation.mutateAsync({ id, sortOrder: Number(val) })
      showToast('排序已更新')
    } catch (e) { showToast('更新排序失败', 'error') }
  }, [updateSortOrderMutation])

  const saveTime = async () => {
    if (!timeSite) return
    try {
      if (timeValue) {
        const [h, m] = timeValue.split(':')
        await updateSchedule.mutateAsync({ id: timeSite.id, scheduleCron: `${m} ${h} * * *` })
        showToast('检测时间设置成功')
      } else {
        await updateSchedule.mutateAsync({ id: timeSite.id, scheduleCron: null })
        showToast('已取消定时检测')
      }
      setTimeOpen(false); setTimeSite(null); setTimeValue('')
    } catch (e) { showToast(e.message || '保存失败', 'error') }
  }

  const openDebugModal = useCallback((site) => {
    setDebugOpen(true); setDebugLoading(true); setDebugData(null)
    setDebuggingSiteFromList(site)
    setDebuggingSiteId(site.id)
  }, [])

  const saveCategoryHandler = async () => {
    try {
      if (!catName.trim()) { showToast('请输入分类名称', 'warning'); return }
      const payload = { name: catName.trim() }
      if (catTime) {
        const [h, m] = catTime.split(':')
        payload.scheduleCron = `${m} ${h} * * *`
        payload.timezone = 'Asia/Shanghai'
      }
      if (editingCategory) {
        await updateCategory.mutateAsync({ id: editingCategory.id, ...payload })
        showToast('分类更新成功')
      } else {
        await createCategory.mutateAsync(payload)
        showToast('分类创建成功')
      }
      setCatOpen(false); setEditingCategory(null); setCatName(''); setCatTime('')
    } catch (e) { showToast(e.message || '保存失败', 'error') }
  }

  const deleteCategory = async (id) => {
    try {
      await deleteCategoryMutation.mutateAsync(id)
      showToast('分类删除成功')
    } catch (e) { showToast(e.message || '删除失败', 'error') }
  }

  const checkCategory = async (id, name) => {
    if (categoryCheckingId) { showToast('正在检测中', 'warning'); return }
    setCategoryCheckingId(id)
    try {
      const r = await batchCheckCategory.mutateAsync(id)
      setBatchResults({ ...r, timestamp: new Date().toISOString() })
      setExpandedSites(new Set()); setBatchResultOpen(true)
      showToast(r.changes?.length === 0 && r.failures?.length === 0 ? '检测完成，所有站点无变更' : '检测完成！')
    } catch (e) { showToast(e.message || '检测失败', 'error') }
    finally { setCategoryCheckingId(null) }
  }

  const checkGroup = async (type, name) => {
    if (categoryCheckingId) { showToast('正在检测中', 'warning'); return }
    const sites = type === 'pinned' ? list.filter(s => s.pinned && !s.excludeFromBatch) : list.filter(s => !s.categoryId && !s.pinned && !s.excludeFromBatch)
    if (!sites.length) { showToast(`${name}下没有可检测的站点`, 'warning'); return }
    setCategoryCheckingId(type)
    const changes = []; const failures = []
    for (let i = 0; i < sites.length; i++) {
      const s = sites[i]
      try {
        const data = await checkSite.mutateAsync({ id: s.id, skipNotification: true })
        if (data?.hasChanges && data.diff) changes.push({ siteName: s.name, diff: data.diff })
      } catch (e) { failures.push({ siteName: s.name, error: e.message || '检测失败' }) }
      if (i < sites.length - 1) await new Promise(r => setTimeout(r, 5000))
    }
    setCategoryCheckingId(null)
    setBatchResults({ changes, failures, timestamp: new Date().toISOString(), totalSites: sites.length })
    setExpandedSites(new Set()); setBatchResultOpen(true)
    qc.invalidateQueries({ queryKey: ['sites'] })
    showToast(changes.length === 0 && failures.length === 0 ? '检测完成，所有站点无变更' : '检测完成！')
  }

  const updateField = (key, val) => setFormData(prev => ({ ...prev, [key]: val }))

  const openTimeModal = useCallback((site) => {
    const hm = cronToHourMin(site.scheduleCron)
    setTimeSite(site); setTimeValue(hm.h ? `${String(hm.h).padStart(2,'0')}:${String(hm.m).padStart(2,'0')}` : ''); setTimeOpen(true)
  }, [])

  const openEditCategoryModal = useCallback((cat) => {
    setEditingCategory(cat); setCatName(cat.name)
    const hm = cronToHourMin(cat.scheduleCron)
    setCatTime(hm.h ? `${String(hm.h).padStart(2,'0')}:${String(hm.m).padStart(2,'0')}` : ''); setCatOpen(true)
  }, [])

  return (
    <div>
      <div className="flex items-center justify-between mb-6 flex-wrap gap-3 anim-in">
        <div><h1 className="text-2xl font-bold tracking-tight">站点管理</h1><p className="text-sm text-text-secondary mt-0.5">管理和监控所有 API 中继站点</p></div>
        <div className="flex items-center gap-2 flex-wrap">
          <SearchBox value={searchKeyword} onChange={e => handleSearch(e.target.value)} placeholder="搜索站点..." className="w-44" />
          {!isMobile && <><Button variant="ghost" size="sm" onClick={onCheckAll} disabled={batchChecking}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>一键检测</Button>{hasLastResult && <Button variant="ghost" size="sm" onClick={loadLastBatchResult}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>查看结果</Button>}<Button variant="ghost" size="sm" onClick={() => navigate('/settings')}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/><polyline points="22,6 12,13 2,6"/></svg>邮件通知</Button><Button variant="ghost" size="sm" onClick={() => navigate('/settings')}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>定时检测</Button></>}
          <Button variant="primary" size="sm" onClick={() => { setEditMode(false); setFormData({apiType: 'newapi'}); setBillingExpanded(false); setOpen(true) }}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>新增站点</Button>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6 anim-in anim-in-d1">
        <StatsCard icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>} label="站点总数" value={list.length} change={`${categories.length} 个分类`} color="blue" />
        <StatsCard icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>} label="在线站点" value={list.filter(s => s.lastCheckStatus !== 'error').length} change={`${list.length ? Math.round(list.filter(s => s.lastCheckStatus !== 'error').length / list.length * 100) : 0}% 可用率`} color="green" />
        <StatsCard icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 20V10"/><path d="M18 20V4"/><path d="M6 20v-4"/></svg>} label="异常站点" value={list.filter(s => s.lastCheckStatus === 'error').length} change="需关注" color="red" />
        <StatsCard icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>} label="分类数" value={categories.length} change={`${new Set(list.map(s => s.categoryId).filter(Boolean)).size} 已使用`} color="orange" />
      </div>

      <SitesBatchCheck batchChecking={batchChecking} batchProgress={batchProgress} />

      <div className="flex gap-2 flex-wrap mb-3">
        <Button variant="ghost" size="sm" onClick={() => setCategoryManageOpen(true)}>
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>分类管理
        </Button>
        <Button variant="ghost" size="sm" onClick={() => { setEditingCategory(null); setCatName(''); setCatTime(''); setCatOpen(true) }}>
          <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>新增分类
        </Button>
      </div>

      <SitesList
        sites={list}
        categories={categories}
        collapsedGroups={collapsedGroups}
        isMobile={isMobile}
        searchKeyword={searchKeyword}
        categoryCheckingId={categoryCheckingId}
        onCheck={onCheck}
        onDelete={onDelete}
        updateSortOrder={updateSortOrder}
        openEditModal={openEditModal}
        openDebugModal={openDebugModal}
        onOpenTimeModal={openTimeModal}
        toggleCollapse={toggleCollapse}
        checkCategory={checkCategory}
        checkGroup={checkGroup}
        deleteCategory={deleteCategory}
        openEditCategoryModal={openEditCategoryModal}
      />

      <SiteEditModal
        open={open} editMode={editMode} editingSite={editingSite}
        formData={formData} billingExpanded={billingExpanded} categories={categories}
        onClose={() => { setOpen(false); setEditMode(false); setEditingSite(null); setFormData({}); setBillingExpanded(false) }}
        onSave={editMode ? onEdit : onAdd}
        updateField={updateField}
        setBillingExpanded={setBillingExpanded}
      />

      <TimeModal
        open={timeOpen} timeSite={timeSite} timeValue={timeValue}
        onClose={() => { setTimeOpen(false); setTimeSite(null); setTimeValue('') }}
        onSave={saveTime} setTimeValue={setTimeValue}
      />

      <DebugModal
        open={debugOpen} debugData={debugData} debugLoading={debugLoading}
        onClose={() => { setDebugOpen(false); setDebugData(null) }}
      />

      <BatchResultModal
        open={batchResultOpen} batchResults={batchResults} expandedSites={expandedSites}
        onClose={() => setBatchResultOpen(false)}
        setExpandedSites={setExpandedSites}
      />

      <CategoryManageModal
        open={categoryManageOpen} categories={categories}
        onClose={() => setCategoryManageOpen(false)}
        onEdit={openEditCategoryModal}
        onDelete={deleteCategory}
        onAdd={() => { setEditingCategory(null); setCatName(''); setCatTime(''); setCatOpen(true) }}
      />

      <CategoryModal
        open={catOpen} catName={catName} catTime={catTime}
        editingCategory={editingCategory}
        onClose={() => { setCatOpen(false); setEditingCategory(null); setCatName(''); setCatTime('') }}
        saveCategoryHandler={saveCategoryHandler}
        setCatName={setCatName} setCatTime={setCatTime}
      />

      <Modal open={diffModal.open} onClose={() => setDiffModal({ open: false, title: '', html: '' })} title={diffModal.title}>
        <div dangerouslySetInnerHTML={{ __html: diffModal.html }} />
      </Modal>
    </div>
  )
}
