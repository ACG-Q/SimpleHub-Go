export default function SitesBatchCheck({ batchChecking, batchProgress }) {
  if (!batchChecking) return null
  return (
    <div className="bg-primary-bg border border-primary/20 rounded-xl p-4 mb-4 anim-in">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-semibold text-primary">一键检测中...</span>
        <span className="text-xs text-text-secondary">{batchProgress.current} / {batchProgress.total}</span>
      </div>
      <div className="w-full h-2 rounded-full bg-primary/10 overflow-hidden">
        <div className="h-full rounded-full bg-gradient-to-r from-primary to-primary-light transition-all duration-300"
          style={{ width: `${(batchProgress.current / (batchProgress.total || 1)) * 100}%` }} />
      </div>
      <p className="text-xs text-text-muted mt-1.5">当前: {batchProgress.currentSite}</p>
    </div>
  )
}
