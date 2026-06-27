import { cn } from '../../utils/cn'

export function ProgressBar({ value, max, className }) {
  const pct = Math.min((value / (max || 1)) * 100, 100)
  const color = pct > 90 ? 'bg-gradient-to-r from-danger to-red-400' : pct > 75 ? 'bg-gradient-to-r from-warning to-cta' : 'bg-gradient-to-r from-primary to-primary-light'
  return (
    <div className={cn('w-full h-1.5 rounded-full bg-slate-200 overflow-hidden', className)}>
      <div className={cn('h-full rounded-full transition-all duration-500 ease-out', color)} style={{ width: `${pct}%` }} />
    </div>
  )
}
