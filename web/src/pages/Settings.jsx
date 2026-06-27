import { useState, useEffect, useRef } from 'react'
import { Card, CardHeader, CardBody } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Switch } from '../components/ui/Switch'
import { TagInput } from '../components/ui/TagInput'
import { TimePicker } from '../components/ui/TimePicker'
import { CollapsePanel } from '../components/ui/Collapse'
import { showToast } from '../api/client'
import {
  useEmailConfig, useSaveEmailConfig, useTestEmail,
  useScheduleConfig, useSaveScheduleConfig, useTriggerGlobalCheck,
  useExportSites, useImportSites,
} from '../hooks/useApi'

const Desc = ({ children }) => <p className="text-xs text-text-muted mt-1">{children}</p>

export default function Settings() {
  const fileRef = useRef(null)

  const { data: emailConfigData } = useEmailConfig()
  const scheduleQuery = useScheduleConfig()
  const saveEmailConfigMutation = useSaveEmailConfig()
  const testEmailMutation = useTestEmail()
  const saveScheduleConfigMutation = useSaveScheduleConfig()
  const triggerGlobalCheckMutation = useTriggerGlobalCheck()
  const exportSitesMutation = useExportSites()
  const importSitesMutation = useImportSites()

  const [emailApiKey, setEmailApiKey] = useState('')
  const [emailEmails, setEmailEmails] = useState([])

  const [schEnable, setSchEnable] = useState(false)
  const [schScheduleTime, setSchScheduleTime] = useState('09:00')
  const [schInterval, setSchInterval] = useState('30')
  const [schOverride, setSchOverride] = useState(false)

  useEffect(() => {
    if (emailConfigData?.notifyEmails) {
      setEmailEmails(emailConfigData.notifyEmails.split(',').map(s => s.trim()).filter(Boolean))
    }
  }, [emailConfigData])

  useEffect(() => {
    if (scheduleQuery.data?.config) {
      const c = scheduleQuery.data.config
      setSchEnable(c.enabled)
      setSchScheduleTime(`${String(c.hour ?? 9).padStart(2, '0')}:${String(c.minute ?? 0).padStart(2, '0')}`)
      setSchInterval(String(c.interval ?? 30))
      setSchOverride(c.overrideIndividual ?? false)
    }
  }, [scheduleQuery.data])

  const saveEmailConfig = async () => {
    try {
      await saveEmailConfigMutation.mutateAsync({ resendApiKey: emailApiKey, notifyEmails: emailEmails.join(','), enabled: true })
      setEmailApiKey(''); setEmailEmails([])
      showToast('邮件通知配置成功')
    } catch (e) { showToast(e.message || '保存失败', 'error') }
  }

  const testEmail = async () => {
    try {
      await testEmailMutation.mutateAsync({ resendApiKey: emailApiKey, notifyEmails: emailEmails.join(',') })
      showToast('测试邮件已发送')
    } catch (e) { showToast(e.message || '发送失败', 'error') }
  }

  const saveScheduleConfig = async () => {
    try {
      const [h = '9', m = '0'] = (schScheduleTime || '').split(':')
      await saveScheduleConfigMutation.mutateAsync({
        enabled: schEnable, hour: Number(h) || 9, minute: Number(m) || 0,
        interval: Number(schInterval) || 30, overrideIndividual: schOverride,
      })
      showToast('定时配置保存成功')
    } catch (e) { showToast(e.message || '保存失败', 'error') }
  }

  const triggerSchedule = async () => {
    try {
      await triggerGlobalCheckMutation.mutateAsync()
      showToast('全局检测已触发')
    } catch (e) { showToast(e.message || '触发失败', 'error') }
  }

  const handleExport = async () => {
    try {
      const data = await exportSitesMutation.mutateAsync()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url; a.download = `sites-export-${Date.now()}.json`; a.click()
      URL.revokeObjectURL(url)
      showToast('导出成功')
    } catch (e) { showToast(e.message || '导出失败', 'error') }
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

  return (
    <div>
      <div className="flex items-center justify-between mb-6 anim-in">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">设置</h1>
          <p className="text-sm text-text-secondary mt-0.5">系统配置管理</p>
        </div>
      </div>

      <div className="space-y-4">
        <CollapsePanel defaultOpen title={<span className="font-semibold">邮件通知配置</span>}>
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
            <div className="flex items-center gap-2 pt-2">
              <Button variant="primary" onClick={saveEmailConfig}>保存配置</Button>
              {emailConfigData?.enabled && <Button variant="ghost" onClick={testEmail}>发送测试邮件</Button>}
            </div>
          </div>
        </CollapsePanel>

        <CollapsePanel defaultOpen title={<span className="font-semibold">定时检测配置</span>}>
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
            <div className="flex items-center gap-2 pt-2">
              <Button variant="primary" onClick={saveScheduleConfig}>保存定时配置</Button>
              <Button variant="ghost" onClick={triggerSchedule}>立即执行</Button>
            </div>
          </div>
        </CollapsePanel>

        <CollapsePanel defaultOpen title={<span className="font-semibold">数据导入 / 导出</span>}>
          <div className="space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <Button variant="primary" onClick={handleExport}>
                <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                导出站点数据
              </Button>
              <Button variant="ghost" onClick={() => fileRef.current?.click()}>
                <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                导入站点数据
              </Button>
              <input ref={fileRef} type="file" accept=".json" onChange={handleImport} className="hidden" />
            </div>
            <Desc>导入站点数据会合并已有的站点，导出数据包含所有站点信息和分类信息</Desc>
          </div>
        </CollapsePanel>
      </div>
    </div>
  )
}
