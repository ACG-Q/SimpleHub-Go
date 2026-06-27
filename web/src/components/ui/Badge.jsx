import { cn } from '../../utils/cn'

const colors = {
  green: 'bg-success-bg text-success',
  yellow: 'bg-warning-bg text-yellow-600',
  gray: 'bg-slate-100 text-text-secondary',
  blue: 'bg-primary-bg text-primary',
  orange: 'bg-orange-100 text-cta',
  red: 'bg-danger-bg text-danger',
}

export function Badge({ children, color = 'gray', className }) {
  return (
    <span className={cn('inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold', colors[color], className)}>
      {children}
    </span>
  )
}

export function StatusDot({ status }) {
  const colors = {
    green: 'bg-success shadow-[0_0_6px_rgba(16,185,129,0.4)]',
    red: 'bg-danger shadow-[0_0_6px_rgba(239,68,68,0.4)]',
    yellow: 'bg-warning shadow-[0_0_6px_rgba(245,158,11,0.4)]',
    gray: 'bg-slate-300',
  }
  return <span className={cn('inline-block w-2 h-2 rounded-full', colors[status] || colors.gray)} />
}

export function Tag({ children, color = 'gray', className }) {
  const colors = {
    blue: 'bg-primary-bg text-primary',
    green: 'bg-success-bg text-success',
    red: 'bg-danger-bg text-danger',
    gray: 'bg-border text-text-secondary',
    orange: 'bg-orange-100 text-cta',
  }
  return (
    <span className={cn('inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium', colors[color], className)}>
      {children}
    </span>
  )
}
