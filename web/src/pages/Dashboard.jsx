import { useDashboardStats } from '../hooks/useApi'
import { StatsCard, Card, CardHeader, CardBody } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'
import { Skeleton } from '../components/ui/Skeleton'

const typeColors = {
  newapi: 'blue',
  veloera: 'green',
  voapi: 'orange',
  donehub: 'teal',
  other: 'gray',
}

export default function Dashboard() {
  const { data: dashboardData, isLoading } = useDashboardStats()

  const stats = dashboardData?.stats
  const recentDiffs = dashboardData?.recentDiffs || []

  return (
    <div>
      <div className="flex items-center justify-between mb-6 anim-in">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">仪表盘</h1>
          <p className="text-sm text-text-secondary mt-0.5">系统概览与最近动态</p>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6 anim-in anim-in-d1">
        {isLoading ? (
          <>
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-24 w-full" />
          </>
        ) : (
          <>
            <StatsCard
              icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>}
              label="站点总数"
              value={stats?.totalSites ?? '-'}
              change={`${stats?.totalCategories ?? 0} 个分类`}
              color="blue"
            />
            <StatsCard
              icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>}
              label="最近新增模型"
              value={stats?.recentAdded ?? '-'}
              change="近 50 次变更累计"
              color="green"
            />
            <StatsCard
              icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 18 13.5 8.5 8.5 13.5 1 6"/><polyline points="17 18 23 18 23 12"/></svg>}
              label="最近移除模型"
              value={stats?.recentRemoved ?? '-'}
              change="近 50 次变更累计"
              color="orange"
            />
            <StatsCard
              icon={<svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>}
              label="未检测站点"
              value={stats?.neverChecked ?? '-'}
              change={stats?.neverChecked > 0 ? '请执行检测' : '全部已检测'}
              color={stats?.neverChecked > 0 ? 'red' : 'green'}
            />
          </>
        )}
      </div>

      {!isLoading && stats?.sitesByType && Object.keys(stats.sitesByType).length > 0 && (
        <Card className="mb-6 anim-in anim-in-d2">
          <CardHeader><span className="font-bold text-sm">站点类型分布</span></CardHeader>
          <CardBody>
            <div className="flex flex-wrap gap-2">
              {Object.entries(stats.sitesByType).map(([type, count]) => (
                <div key={type} className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-white/50 border border-border text-sm">
                  <Badge color={typeColors[type] || 'gray'}>{type}</Badge>
                  <span className="font-semibold">{count}</span>
                  <span className="text-text-muted text-xs">个</span>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>
      )}

      <Card className="anim-in anim-in-d3">
        <CardHeader><span className="font-bold text-sm">最近变更</span></CardHeader>
        <CardBody>
          {recentDiffs.length === 0 ? (
            <div className="py-8 text-center text-sm text-text-muted">
              <svg className="w-10 h-10 mx-auto mb-2 opacity-50" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
              <p>暂无变更记录</p>
            </div>
          ) : (
            <div className="space-y-2">
              {recentDiffs.map((d, i) => (
                <div key={`${d.siteId}-${i}`} className="flex items-center justify-between px-3.5 py-2.5 rounded-lg bg-white/50 border border-border text-sm">
                  <div className="flex items-center gap-3 min-w-0">
                    <span className="font-medium truncate">{d.siteName || d.siteId}</span>
                    <span className="text-text-muted text-xs shrink-0">{d.diffAt}</span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {d.added > 0 && <Badge color="green">+{d.added}</Badge>}
                    {d.removed > 0 && <Badge color="red">-{d.removed}</Badge>}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
