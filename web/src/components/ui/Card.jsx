import { cn } from '../../utils/cn'

export function Card({ children, className, hoverable, onClick }) {
  return (
    <div
      onClick={onClick}
      className={cn(
        'bg-white/70 backdrop-blur-sm border border-border-glass rounded-xl shadow-sm transition-all duration-200',
        hoverable && 'hover:shadow-md hover:-translate-y-0.5 cursor-pointer',
        className,
      )}
    >
      {children}
    </div>
  )
}

export function CardHeader({ children, className }) {
  return <div className={cn('flex items-center justify-between px-5 py-4', className)}>{children}</div>
}

export function CardBody({ children, className }) {
  return <div className={cn('px-5 pb-5', className)}>{children}</div>
}

export function StatsCard({ icon, label, value, change, changeType, color = 'blue' }) {
  const iconColors = {
    blue: 'bg-primary-bg text-primary',
    green: 'bg-success-bg text-success',
    orange: 'bg-warning-bg text-warning',
    red: 'bg-danger-bg text-danger',
  }
  return (
    <div className="bg-white/70 backdrop-blur-sm border border-border-glass rounded-xl p-5 shadow-sm transition-all duration-200 hover:shadow-md hover:-translate-y-0.5 flex items-start gap-4">
      <div className={cn('w-11 h-11 rounded-lg flex items-center justify-center shrink-0', iconColors[color])}>
        {icon}
      </div>
      <div className="min-w-0">
        <div className="text-xs font-medium text-text-secondary">{label}</div>
        <div className="text-2xl font-bold tracking-tight text-text leading-tight">{value}</div>
        {change && (
          <div className={cn('text-xs mt-0.5', changeType === 'up' ? 'text-success' : changeType === 'down' ? 'text-danger' : 'text-text-secondary')}>
            {change}
          </div>
        )}
      </div>
    </div>
  )
}
