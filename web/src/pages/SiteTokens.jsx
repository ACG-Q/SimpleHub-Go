import { useState, useEffect, useCallback } from 'react'
import { showToast } from '../api/client'
import { Button } from '../components/ui/Button'
import { Input, Textarea } from '../components/ui/Input'
import { Modal } from '../components/ui/Modal'
import { Badge, Tag } from '../components/ui/Badge'
import { Table, THead, Th, TBody, Tr, Td } from '../components/ui/Table'
import { copyText } from './SitesList'
import { useTokens, useGroups, useCreateToken, useUpdateToken, useDeleteToken, useRevealTokenKey } from '../hooks/useApi'

export default function SiteTokens({ siteId, siteInfo }) {
  const { data: tokens = [] } = useTokens(siteId)
  const { data: groupsFromApi = [] } = useGroups(siteId)
  const createTokenMutation = useCreateToken(siteId)
  const updateTokenMutation = useUpdateToken(siteId)
  const revealTokenKeyMutation = useRevealTokenKey(siteId)
  const deleteTokenMutation = useDeleteToken(siteId)
  const [visible, setVisible] = useState(false)
  const [copyingIds, setCopyingIds] = useState(new Set())
  const [groups, setGroups] = useState([])
  const [createVisible, setCreateVisible] = useState(false)

  const defCreate = () => ({
    name: '', group: '__user_group__', neverExpire: true, expire: '',
    unlimited: true, quota: '50000000', modelLimits: false, modelLimitsText: '', allowIps: '',
  })
  const [createForm, setCreateForm] = useState(defCreate)
  const cf = (k) => (e) => setCreateForm(p => ({ ...p, [k]: e?.target?.value ?? e }))

  const [editVisible, setEditVisible] = useState(false)
  const [editingToken, setEditingToken] = useState(null)
  const [editForm, setEditForm] = useState(defCreate)
  const ef = (k) => (e) => setEditForm(p => ({ ...p, [k]: e?.target?.value ?? e }))

  useEffect(() => {
    if (groupsFromApi.length > 0) {
      const withDefault = groupsFromApi.some(x => x.value === '__user_group__')
        ? groupsFromApi
        : [{ value: '__user_group__', label: '用户分组' }, ...groupsFromApi]
      setGroups(withDefault)
    }
  }, [groupsFromApi])

  const copyTokenKey = useCallback(async (token) => {
    const key = token?.key || ''
    if (!key) { showToast('没有可复制的令牌', 'warning'); return }
    if (!String(key).includes('*')) { copyText(key, '令牌已复制'); return }
    if (!token?.id) { showToast('缺少令牌ID', 'error'); return }
    setCopyingIds(prev => new Set(prev).add(token.id))
    try {
      const data = await revealTokenKeyMutation.mutateAsync(token.id)
      const fullKey = data?.data?.key || data?.key
      if (!fullKey) throw new Error('获取完整令牌失败')
      copyText(fullKey, '完整令牌已复制')
    } catch (e) { showToast(e.message || '获取完整令牌失败', 'error') }
    finally { setCopyingIds(prev => { const n = new Set(prev); n.delete(token.id); return n }) }
  }, [siteId])

  const deleteTokenFn = useCallback(async (tokenId) => {
    try {
      await deleteTokenMutation.mutateAsync(tokenId)
      showToast('删除成功')
    } catch (e) { showToast(e.message || '删除失败', 'error') }
  }, [deleteTokenMutation])

  const updateTokenFn = useCallback(async () => {
    if (!editingToken) return
    try {
      const payload = {
        id: editingToken.id, name: editForm.name,
        group: editForm.group === '__user_group__' ? '' : editForm.group,
        expired_time: editForm.neverExpire ? -1 : Math.floor(new Date(editForm.expire).getTime() / 1000),
        unlimited_quota: editForm.unlimited, remain_quota: editForm.unlimited ? 0 : Number(editForm.quota || 0),
        model_limits_enabled: editForm.modelLimits, model_limits: editForm.modelLimitsText,
        allow_ips: editForm.allowIps, key: editingToken.key, uid: editingToken.uid || 0,
        used_quota: editingToken.used_quota || 0,
      }
      await updateTokenMutation.mutateAsync(payload)
      showToast('修改成功')
      setEditVisible(false)
      setEditingToken(null)
    } catch (e) { showToast(e.message || '修改失败', 'error') }
  }, [editingToken, editForm, updateTokenMutation])

  const createTokenFn = useCallback(async () => {
    if (!createForm.name.trim()) { showToast('请输入令牌名称', 'warning'); return }
    try {
      const payload = {
        name: createForm.name, group: createForm.group === '__user_group__' ? '' : createForm.group,
        expiredTime: createForm.neverExpire ? -1 : Math.floor(new Date(createForm.expire).getTime() / 1000),
        unlimitedQuota: createForm.unlimited, remainQuota: createForm.unlimited ? 0 : Number(createForm.quota || 0),
        modelLimitsEnabled: createForm.modelLimits, modelLimits: createForm.modelLimitsText, allowIps: createForm.allowIps,
      }
      if (siteInfo?.apiType === 'voapi') payload.groups = [parseInt(payload.group) || 1]
      await createTokenMutation.mutateAsync(payload)
      showToast('创建成功')
      setCreateVisible(false)
    } catch (e) { showToast(e.message || '创建失败', 'error') }
  }, [siteInfo, createForm, createTokenMutation])

  const openList = useCallback(() => {
    setVisible(true)
  }, [])

  const openCreate = useCallback(() => {
    setCreateForm(defCreate())
    setCreateVisible(true)
  }, [])

  const openEdit = useCallback(async (token) => {
    setEditingToken(token)
    setEditForm({
      name: token.name, group: token.group === '' ? '__user_group__' : (token.group || '__user_group__'),
      neverExpire: token.expired_time === -1,
      expire: token.expired_time !== -1 ? new Date(token.expired_time * 1000).toISOString().slice(0, 16) : '',
      unlimited: !!token.unlimited_quota, quota: String(token.remain_quota || 0),
      modelLimits: !!token.model_limits_enabled, modelLimitsText: token.model_limits || '',
      allowIps: token.allow_ips || '',
    })
    setEditVisible(true)
  }, [])

  return (
    <>
      <div onClick={openList}
        className="p-4 md:p-6 rounded-xl bg-gradient-to-br from-primary/10 to-primary/5 border border-primary/15 cursor-pointer transition-all duration-200 hover:shadow-lg hover:-translate-y-0.5 flex flex-col items-center justify-center gap-2 md:gap-3">
        <svg className="w-7 h-7 md:w-10 md:h-10 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>
        <span className="font-bold text-sm md:text-lg text-primary">令牌管理</span>
        <span className="text-xs text-text-secondary hidden md:block">查看、修改和删除 API 令牌</span>
      </div>

      <Modal open={visible} onClose={() => setVisible(false)} title="令牌管理" size="xl"
        footer={<Button variant="primary" onClick={openCreate}><svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>创建令牌</Button>}>
        <Table>
          <THead>
            <Th>ID</Th><Th>名称</Th><Th>令牌</Th><Th>分组</Th><Th>过期</Th><Th>额度</Th><Th>已使用</Th><Th>状态</Th><Th>创建时间</Th><Th>操作</Th>
          </THead>
          <TBody>
            {tokens.map(t => (
              <Tr key={t.id}>
                <Td className="text-xs">{t.id}</Td>
                <Td className="font-medium text-sm max-w-[120px] truncate">{t.name}</Td>
                <Td className="max-w-[180px]">
                  <div className="flex items-center gap-1">
                    <span className="text-xs truncate">{t.key}</span>
                    <button onClick={() => copyTokenKey(t)} disabled={copyingIds.has(t.id)}
                      className="shrink-0 text-text-muted hover:text-primary p-0.5 cursor-pointer">
                      <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                    </button>
                  </div>
                </Td>
                <Td><Tag color={t.group && t.group !== '' ? 'blue' : 'gray'}>{t.group && t.group !== '' ? t.group : '用户分组'}</Tag></Td>
                <Td className="text-xs">{t.expired_time === -1 ? <Badge color="green">永不过期</Badge> : new Date(t.expired_time * 1000).toLocaleString('zh-CN')}</Td>
                <Td className="text-xs">{t.unlimited_quota ? <Badge color="orange">无限额</Badge> : `${(t.remain_quota / 500000).toFixed(2)} $`}</Td>
                <Td className="text-xs">{((t.used_quota || 0) / 500000).toFixed(2)} $</Td>
                <Td><Badge color={t.status === 1 ? 'green' : 'red'}>{t.status === 1 ? '启用' : '禁用'}</Badge></Td>
                <Td className="text-xs">{new Date(t.created_time * 1000).toLocaleString('zh-CN')}</Td>
                <Td>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" onClick={() => openEdit(t)}><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></Button>
                    <Button variant="ghost" size="icon" onClick={() => { if (window.confirm('确定删除此令牌？')) deleteTokenFn(t.id) }}><svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></Button>
                  </div>
                </Td>
              </Tr>
            ))}
          </TBody>
        </Table>
      </Modal>

      <Modal open={createVisible} onClose={() => { setCreateVisible(false) }} title="创建令牌" size="md"
        footer={<><Button variant="ghost" onClick={() => { setCreateVisible(false) }}>取消</Button><Button variant="primary" onClick={createTokenFn}>创建</Button></>}>
        <div className="space-y-4">
          <Input label="名称" value={createForm.name} onChange={cf('name')} placeholder="输入令牌名称" />
          <div className="space-y-1.5">
            <label className="text-sm font-semibold text-text">分组</label>
            <select value={createForm.group} onChange={cf('group')}
              className="w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 text-sm outline-none focus:border-primary-light">
              {groups.map(g => <option key={g.value} value={g.value}>{g.label}</option>)}
            </select>
          </div>
          <div>
            <label className="text-sm font-semibold text-text block mb-2">过期时间</label>
            <div className="space-y-2">
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={createForm.neverExpire} onChange={e => setCreateForm(p => ({ ...p, neverExpire: e.target.checked }))}
                  className="w-4 h-4 rounded border-border text-primary focus:ring-primary" />
                <span className="text-sm text-text">永不过期</span>
              </label>
              {!createForm.neverExpire && <Input type="datetime-local" value={createForm.expire} onChange={cf('expire')} />}
            </div>
          </div>
          <div>
            <label className="text-sm font-semibold text-text block mb-2">额度</label>
            <div className="space-y-2">
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={createForm.unlimited} onChange={e => setCreateForm(p => ({ ...p, unlimited: e.target.checked }))}
                  className="w-4 h-4 rounded border-border text-primary focus:ring-primary" />
                <span className="text-sm text-text">无限额</span>
              </label>
              {!createForm.unlimited && <Input type="number" min={0} value={createForm.quota} onChange={cf('quota')} placeholder="额度（原始值）" />}
            </div>
          </div>
          <Textarea label="IP 白名单（一行一个）" value={createForm.allowIps} onChange={cf('allowIps')} placeholder="192.168.1.1" rows={3} />
          <div>
            <label className="inline-flex items-center gap-2 cursor-pointer mb-2">
              <input type="checkbox" checked={createForm.modelLimits} onChange={e => setCreateForm(p => ({ ...p, modelLimits: e.target.checked }))}
                className="w-4 h-4 rounded border-border text-primary focus:ring-primary" />
              <span className="text-sm text-text">启用模型限制</span>
            </label>
            {createForm.modelLimits && <Textarea value={createForm.modelLimitsText} onChange={cf('modelLimitsText')} placeholder="限制的模型列表" rows={3} />}
          </div>
        </div>
      </Modal>

      <Modal open={editVisible} onClose={() => { setEditVisible(false); setEditingToken(null) }} title="编辑令牌" size="md"
        footer={<><Button variant="ghost" onClick={() => { setEditVisible(false); setEditingToken(null) }}>取消</Button><Button variant="primary" onClick={updateTokenFn}>保存</Button></>}>
        <div className="space-y-4">
          <Input label="名称" value={editForm.name} onChange={ef('name')} />
          <div className="space-y-1.5">
            <label className="text-sm font-semibold text-text">分组</label>
            <select value={editForm.group} onChange={ef('group')}
              className="w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 text-sm outline-none focus:border-primary-light">
              {groups.map(g => <option key={g.value} value={g.value}>{g.label}</option>)}
            </select>
          </div>
          <div>
            <label className="text-sm font-semibold text-text block mb-2">过期时间</label>
            <div className="space-y-2">
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={editForm.neverExpire} onChange={e => setEditForm(p => ({ ...p, neverExpire: e.target.checked }))}
                  className="w-4 h-4 rounded border-border text-primary focus:ring-primary" />
                <span className="text-sm text-text">永不过期</span>
              </label>
              {!editForm.neverExpire && <Input type="datetime-local" value={editForm.expire} onChange={ef('expire')} />}
            </div>
          </div>
          <div>
            <label className="text-sm font-semibold text-text block mb-2">额度</label>
            <div className="space-y-2">
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={editForm.unlimited} onChange={e => setEditForm(p => ({ ...p, unlimited: e.target.checked }))}
                  className="w-4 h-4 rounded border-border text-primary focus:ring-primary" />
                <span className="text-sm text-text">无限额</span>
              </label>
              {!editForm.unlimited && <Input type="number" min={0} value={editForm.quota} onChange={ef('quota')} />}
            </div>
          </div>
          <Textarea label="IP 白名单" value={editForm.allowIps} onChange={ef('allowIps')} rows={3} />
          <div>
            <label className="inline-flex items-center gap-2 cursor-pointer mb-2">
              <input type="checkbox" checked={editForm.modelLimits} onChange={e => setEditForm(p => ({ ...p, modelLimits: e.target.checked }))}
                className="w-4 h-4 rounded border-border text-primary focus:ring-primary" />
              <span className="text-sm text-text">启用模型限制</span>
            </label>
            {editForm.modelLimits && <Textarea value={editForm.modelLimitsText} onChange={ef('modelLimitsText')} rows={3} />}
          </div>
        </div>
      </Modal>
    </>
  )
}
