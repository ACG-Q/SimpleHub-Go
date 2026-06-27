import { useState } from 'react'
import dayjs from 'dayjs'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Modal } from '../components/ui/Modal'
import { Tag } from '../components/ui/Badge'
import { Switch } from '../components/ui/Switch'
import { Skeleton } from '../components/ui/Skeleton'
import { TimePicker } from '../components/ui/TimePicker'
import { TagInput } from '../components/ui/TagInput'
import { cronToHourMin } from './SitesList'
import { SITE_TYPES_LIST, SITE_TYPE_DESCS, SITE_TYPE_LABELS } from '../constants/siteTypes'

const Desc = ({ children }) => <p className="text-xs text-text-muted mt-1">{children}</p>

export function SiteEditModal({
  open, editMode, editingSite, formData, billingExpanded, categories,
  onClose, onSave, updateField, setBillingExpanded,
}) {
  const [showApiKey, setShowApiKey] = useState(false)

  return (
    <Modal open={open} onClose={onClose}
      title={editMode ? '编辑站点' : '新增站点'} size="lg"
      footer={<>
        <Button variant="ghost" onClick={onClose}>取消</Button>
        <Button variant="primary" onClick={onSave}>{editMode ? '保存' : '创建'}</Button>
      </>}>
      <div className="space-y-4">
        <Input label="站点名称" value={formData.name || ''} onChange={e => updateField('name', e.target.value)} placeholder="例如: GPT-4 中继" />
        <Desc>给该站点取一个易识别的名称</Desc>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Input label="Base URL" value={formData.baseUrl || ''} onChange={e => updateField('baseUrl', e.target.value)} placeholder="https://api.openai.com" />
            <Desc>API 服务的基础地址</Desc>
          </div>
          <div>
            <Input label="Proxy URL" value={formData.proxyUrl || ''} onChange={e => updateField('proxyUrl', e.target.value)} placeholder="可选" />
            <Desc>可选，所有请求将通过此代理转发</Desc>
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <div className="space-y-1.5">
              <label className="text-sm font-semibold text-text">站点类型</label>
              <select value={formData.apiType || 'newapi'} onChange={e => updateField('apiType', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 font-sans text-sm text-text outline-none focus:border-primary-light focus:shadow-[0_0_0_3px_rgba(37,99,235,0.1)]">
                {SITE_TYPES_LIST.map(t => <option key={t} value={t}>{SITE_TYPE_LABELS[t]}</option>)}
              </select>
            </div>
            <Desc>{SITE_TYPE_DESCS[formData.apiType || 'newapi']}</Desc>
          </div>
          <div>
            <Input label="访问令牌" type={showApiKey ? 'text' : 'password'} value={formData.apiKey || ''} onChange={e => updateField('apiKey', e.target.value)}
              placeholder={editMode ? '留空则不修改' : '输入访问令牌'}
              rightElement={
                <button type="button" onClick={() => setShowApiKey(v => !v)} className="text-text-muted hover:text-text cursor-pointer">
                  {showApiKey ? (
                    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
                  ) : (
                    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                  )}
                </button>
              } />
            <Desc>该站点的访问密钥，用于 API 鉴权</Desc>
          </div>
        </div>
        {(formData.apiType === 'newapi' || formData.apiType === 'veloera' || formData.apiType === 'voapi') && <div><Input label="User ID" value={formData.userId || ''} onChange={e => updateField('userId', e.target.value)} placeholder={formData.apiType === 'newapi' ? 'New-Api-User 值' : formData.apiType === 'veloera' ? 'Veloera-User 值' : 'VOAPI User ID'} /><Desc>站点在 {formData.apiType === 'newapi' ? 'New API' : formData.apiType === 'veloera' ? 'Veloera' : 'VOAPI'} 的用户标识{formData.apiType !== 'voapi' ? '，用于 API 请求认证' : ''}</Desc></div>}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <div className="space-y-1.5">
              <label className="text-sm font-semibold text-text">分类</label>
              <select value={formData.categoryId || ''} onChange={e => updateField('categoryId', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 font-sans text-sm text-text outline-none focus:border-primary-light focus:shadow-[0_0_0_3px_rgba(37,99,235,0.1)]">
                <option value="">未分类</option>
                {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <Desc>将站点归类，便于批量管理</Desc>
          </div>
          <TimePicker label="定时检测" value={formData.scheduleTime ?? ''} onChange={v => updateField('scheduleTime', v)} allowClear />
          <Desc>设置每日自动检测的时间（北京时间），留空则不定时检测</Desc>
        </div>
        <div className="flex flex-wrap items-center gap-4">
          <Switch checked={!!formData.pinned} onChange={v => updateField('pinned', v)} label="置顶" />
          <Switch checked={!!formData.excludeFromBatch} onChange={v => updateField('excludeFromBatch', v)} label="不参与一键检测" />
          <Switch checked={!!formData.unlimitedQuota} onChange={v => updateField('unlimitedQuota', v)} label="无限余额" />
        </div>
        <div><Input label="外链" value={formData.extralink || ''} onChange={e => updateField('extralink', e.target.value)} placeholder="https://..." /><Desc>可选，指向该站点的外部链接</Desc></div>
        <div><Input label="备注" value={formData.remark || ''} onChange={e => updateField('remark', e.target.value)} placeholder="备注信息" /><Desc>可选，内部备注信息</Desc></div>
        {formData.apiType === 'other' && <div>
          <button type="button" onClick={() => setBillingExpanded(v => !v)}
            className="flex items-center gap-1 text-sm text-text-secondary hover:text-primary font-medium cursor-pointer">
            <svg className={`w-4 h-4 transition-transform ${billingExpanded ? 'rotate-90' : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="9 18 15 12 9 6"/></svg>
            {billingExpanded ? '收起' : '展开'}账单配置
          </button>
          {billingExpanded && <div className="mt-3 space-y-3 p-4 rounded-lg bg-slate-50 border border-border">
            <Input label="账单 URL" value={formData.billingUrl || ''} onChange={e => updateField('billingUrl', e.target.value)} />
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-semibold text-text">认证类型</label>
                <select value={formData.billingAuthType || 'token'} onChange={e => updateField('billingAuthType', e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 font-sans text-sm text-text outline-none focus:border-primary-light">
                  <option value="token">Token</option>
                  <option value="password">Password</option>
                </select>
              </div>
              <Input label="认证值" value={formData.billingAuthValue || ''} onChange={e => updateField('billingAuthValue', e.target.value)} />
            </div>
            <Input label="额度字段" value={formData.billingLimitField || ''} onChange={e => updateField('billingLimitField', e.target.value)} placeholder="JSON path" />
            <Input label="用量字段" value={formData.billingUsageField || ''} onChange={e => updateField('billingUsageField', e.target.value)} placeholder="JSON path" />
          </div>}
        </div>}
        {(formData.apiType === 'veloera' || formData.apiType === 'newapi' || formData.apiType === 'voapi') && (
          <div className="space-y-3 p-4 rounded-lg bg-slate-50 border border-border">
            <Switch checked={!!formData.enableCheckIn} onChange={v => updateField('enableCheckIn', v)} label="启用每日签到" />
            {formData.enableCheckIn && (
              <div className="space-y-1.5">
                <label className="text-sm font-semibold text-text">签到模式</label>
                <select value={formData.checkInMode || 'both'} onChange={e => updateField('checkInMode', e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 font-sans text-sm text-text outline-none focus:border-primary-light">
                  <option value="both">同时查询模型和账单</option>
                  <option value="models">仅查询模型</option>
                  <option value="billing">仅查询账单</option>
                </select>
              </div>
            )}
          </div>
        )}
      </div>
    </Modal>
  )
}

export function TimeModal({ open, timeSite, timeValue, onClose, onSave, setTimeValue }) {
  return (
    <Modal open={open} onClose={onClose} title="设置检测时间" size="sm"
      footer={<>
        <Button variant="ghost" onClick={onClose}>取消</Button>
        <Button variant="primary" onClick={onSave}>保存</Button>
      </>}>
      <TimePicker label="执行时间" value={timeValue} onChange={setTimeValue} allowClear />
    </Modal>
  )
}

export function DebugModal({ open, debugData, debugLoading, onClose }) {
  return (
    <Modal open={open} onClose={onClose} title={`调试信息 - ${debugData?.siteName || ''}`} size="xl" className="max-h-[90vh]">
      {debugLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-5 w-1/3" />
          <Skeleton className="h-4 w-1/2" />
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
        </div>
      ) : debugData && (
        <div className="space-y-3 text-sm">
          <div className="grid grid-cols-2 gap-3">
            <div className="p-3 rounded-lg bg-slate-50">
              <span className="text-text-muted text-xs">站点 URL</span>
              <p className="font-medium break-all">{debugData.siteUrl}</p>
            </div>
            <div className="p-3 rounded-lg bg-slate-50">
              <span className="text-text-muted text-xs">HTTP 状态</span>
              <p className="font-medium">{debugData.httpStatus || '-'}</p>
            </div>
            <div className="p-3 rounded-lg bg-slate-50">
              <span className="text-text-muted text-xs">响应时间</span>
              <p className="font-medium">{debugData.responseTime || '-'}ms</p>
            </div>
            <div className="p-3 rounded-lg bg-slate-50">
              <span className="text-text-muted text-xs">模型数</span>
              <p className="font-medium">{Array.isArray(debugData.models) ? debugData.models.length : '-'}</p>
            </div>
          </div>
          <div className="p-3 rounded-lg bg-slate-50">
            <span className="text-text-muted text-xs">错误信息</span>
            <p className={`font-medium break-all ${debugData.error ? 'text-danger' : 'text-success'}`}>{debugData.error || '无错误'}</p>
          </div>
          <div className="p-3 rounded-lg bg-slate-50">
            <span className="text-text-muted text-xs">账单</span>
            <pre className="mt-1 text-xs whitespace-pre-wrap break-all max-h-40 overflow-y-auto">{debugData.billing ? JSON.stringify(debugData.billing, null, 2) : '无'}</pre>
          </div>
          <div className="p-3 rounded-lg bg-slate-50">
            <span className="text-text-muted text-xs">原始响应</span>
            <pre className="mt-1 text-xs whitespace-pre-wrap break-all max-h-60 overflow-y-auto bg-white p-2 rounded border">{typeof debugData.rawResponse === 'string' ? debugData.rawResponse : JSON.stringify(debugData.rawResponse, null, 2)}</pre>
          </div>
        </div>
      )}
    </Modal>
  )
}

export function EmailConfigModal({
  open, emailApiKey, emailEmails, emailConfigData,
  onClose, saveEmailConfig, testEmail, setEmailApiKey, setEmailEmails,
}) {
  return (
    <Modal open={open} onClose={onClose} title="邮件通知配置" size="sm"
      footer={<>
        <Button variant="ghost" onClick={onClose}>关闭</Button>
        {emailConfigData?.enabled && <Button variant="ghost" onClick={testEmail}>发送测试邮件</Button>}
        <Button variant="primary" onClick={saveEmailConfig}>保存配置</Button>
      </>}>
      <div className="space-y-4">
        <div>
          <Input label="Resend API Key" type="password" value={emailApiKey} onChange={e => setEmailApiKey(e.target.value)} placeholder="输入 Resend API Key" />
          <Desc>Resend 服务的 API 密钥，用于发送通知邮件</Desc>
        </div>
        <div>
          <TagInput label="通知邮箱" value={emailEmails} onChange={setEmailEmails} placeholder="输入邮箱后按 Enter" />
          <Desc>输入邮箱地址后按 Enter 添加，支持多个邮箱</Desc>
        </div>
        {emailConfigData?.enabled && (
          <div className="flex items-center gap-2 p-3 rounded-lg bg-success-bg text-success text-sm">
            <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            邮件通知已启用
          </div>
        )}
      </div>
    </Modal>
  )
}

export function ScheduleConfigModal({
  open, schEnable, schScheduleTime, schInterval, schOverride,
  onClose, saveScheduleConfig, setSchEnable, setSchScheduleTime, setSchInterval, setSchOverride,
}) {
  return (
    <Modal open={open} onClose={onClose} title="定时检测配置" size="sm"
      footer={<>
        <Button variant="ghost" onClick={onClose}>取消</Button>
        <Button variant="primary" onClick={saveScheduleConfig}>保存</Button>
      </>}>
      <div className="space-y-4">
        <div>
          <Switch checked={schEnable} onChange={setSchEnable} label="启用全局定时检测" />
          <Desc>启用后，系统将在每日设定的时间自动检测所有站点</Desc>
        </div>
        {schEnable && <>
          <TimePicker label="执行时间" value={schScheduleTime} onChange={setSchScheduleTime} />
          <Desc>设置全局自动检测的执行时间（北京时间）</Desc>
          <div>
            <Input label="间隔(秒)" type="number" min={1} value={schInterval} onChange={e => setSchInterval(e.target.value)} />
            <Desc>每个站点检测完成后的等待时间，避免触发上游 API 限流</Desc>
          </div>
          <div>
            <Switch checked={schOverride} onChange={setSchOverride} label="覆盖个体站点定时设置" />
            <Desc>启用后，全局检测将对所有站点生效，包括已单独设置检测时间的站点</Desc>
          </div>
        </>}
      </div>
    </Modal>
  )
}

export function BatchResultModal({ open, batchResults, expandedSites, onClose, setExpandedSites }) {
  return (
    <Modal open={open} onClose={onClose} title="检测结果" size="xl">
      {batchResults.timestamp && (
        <p className="text-xs text-text-muted mb-4">
          检测时间: {dayjs(batchResults.timestamp).format('YYYY-MM-DD HH:mm:ss')} | 总站点: {batchResults.totalSites}
        </p>
      )}
      <div className="grid grid-cols-3 gap-3 mb-4">
        <div className="p-3 rounded-lg bg-success-bg text-center">
          <p className="text-2xl font-bold text-success">{batchResults.changes?.length || 0}</p>
          <p className="text-xs text-text-secondary">有变更</p>
        </div>
        <div className="p-3 rounded-lg bg-danger-bg text-center">
          <p className="text-2xl font-bold text-danger">{batchResults.failures?.length || 0}</p>
          <p className="text-xs text-text-secondary">失败</p>
        </div>
        <div className="p-3 rounded-lg bg-primary-bg text-center">
          <p className="text-2xl font-bold text-primary">{batchResults.totalSites || 0}</p>
          <p className="text-xs text-text-secondary">总检测</p>
        </div>
      </div>
      {batchResults.changes?.length > 0 && (
        <div className="space-y-2 mb-4">
          <h4 className="text-sm font-semibold text-success">有变更的站点</h4>
          {batchResults.changes.map((c, i) => (
            <div key={c.siteName + i} className="p-3 rounded-lg bg-white/50 border border-border">
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold text-sm">{c.siteName}</span>
                <button onClick={() => setExpandedSites(prev => { const n = new Set(prev); if (n.has(i)) n.delete(i); else n.add(i); return n })}
                  className="text-xs text-primary hover:underline cursor-pointer">
                  {expandedSites.has(i) ? '收起' : '展开'}
                </button>
              </div>
              {expandedSites.has(i) && (
                <div className="space-y-2">
                  <div className="flex flex-wrap gap-1.5 items-center">
                    <span className="text-xs text-success font-medium">新增 ({c.diff?.added?.length || 0})：</span>
                    {c.diff?.added?.slice(0, 10).map((m, j) => <Tag key={m.id} color="green">{m.id}</Tag>)}
                    {c.diff?.added?.length > 10 && <span className="text-xs text-text-muted">...还有 {c.diff.added.length - 10} 个</span>}
                  </div>
                  <div className="flex flex-wrap gap-1.5 items-center">
                    <span className="text-xs text-danger font-medium">移除 ({c.diff?.removed?.length || 0})：</span>
                    {c.diff?.removed?.slice(0, 10).map((m, j) => <Tag key={m.id} color="red">{m.id}</Tag>)}
                    {c.diff?.removed?.length > 10 && <span className="text-xs text-text-muted">...还有 {c.diff.removed.length - 10} 个</span>}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
      {batchResults.failures?.length > 0 && (
        <div>
          <h4 className="text-sm font-semibold text-danger mb-2">检测失败的站点</h4>
          <div className="space-y-1">
            {batchResults.failures.map((f, i) => (
              <div key={f.siteName + i} className="flex items-center gap-2 text-sm">
                <span className="font-medium">{f.siteName}</span>
                <span className="text-text-muted">-</span>
                <span className="text-danger text-xs">{f.error}</span>
              </div>
            ))}
          </div>
        </div>
      )}
      {(!batchResults.changes?.length && !batchResults.failures?.length) && (
        <div className="py-8 text-center text-sm text-text-muted">
          <svg className="w-12 h-12 mx-auto mb-3 text-text-muted opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
          <p>检测完成，所有站点无变更</p>
        </div>
      )}
    </Modal>
  )
}

export function CategoryModal({
  open, catName, catTime, editingCategory,
  onClose, saveCategoryHandler, setCatName, setCatTime,
}) {
  return (
    <Modal open={open} onClose={onClose} title={editingCategory ? '编辑分类' : '新增分类'} size="sm"
      footer={<>
        <Button variant="ghost" onClick={onClose}>取消</Button>
        <Button variant="primary" onClick={saveCategoryHandler}>{editingCategory ? '保存' : '创建'}</Button>
      </>}>
      <div className="space-y-4">
        <Input label="分类名称" value={catName} onChange={e => setCatName(e.target.value)} placeholder="例如: OpenAI 系列" />
        <TimePicker label="定时检测" value={catTime ?? ''} onChange={setCatTime} allowClear />
      </div>
    </Modal>
  )
}

export function CategoryManageModal({ open, categories, onClose, onEdit, onDelete, onAdd }) {
  return (
    <Modal open={open} onClose={onClose} title="分类管理" size="sm"
      footer={<>
        <Button variant="ghost" onClick={() => { onClose(); setTimeout(onAdd, 100) }}>+ 新增分类</Button>
        <Button variant="primary" onClick={onClose}>关闭</Button>
      </>}>
      {categories.length === 0 ? (
        <div className="py-8 text-center text-sm text-text-muted">暂无分类</div>
      ) : (
        <div className="space-y-1">
          {categories.map(cat => (
            <div key={cat.id} className="flex items-center justify-between px-3 py-2.5 rounded-lg hover:bg-slate-50 border-b border-border/50 last:border-0">
              <div className="flex items-center gap-2 min-w-0">
                <svg className="w-4 h-4 shrink-0 text-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                <span className="text-sm font-medium truncate">{cat.name}</span>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <button onClick={() => { onClose(); setTimeout(() => onEdit(cat), 100) }}
                  className="px-2 py-1 text-xs text-primary hover:bg-primary-bg rounded cursor-pointer">编辑</button>
                <button onClick={() => { if (confirm(`确定删除分类"${cat.name}"？`)) onDelete(cat.id) }}
                  className="px-2 py-1 text-xs text-danger hover:bg-danger-bg rounded cursor-pointer">删除</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </Modal>
  )
}
