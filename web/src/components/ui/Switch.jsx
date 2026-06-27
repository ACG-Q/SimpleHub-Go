import { cn } from '../../utils/cn'

export function Switch({ checked, onChange, label, disabled }) {
  return (
    <label className={cn('inline-flex items-center gap-2 cursor-pointer', disabled && 'opacity-50 pointer-events-none')}>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange?.(!checked)}
        className={cn(
          'relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-all duration-200',
          checked ? 'bg-gradient-to-r from-primary to-primary-light' : 'bg-slate-200',
        )}
      >
        <span className={cn('block h-4 w-4 rounded-full bg-white shadow-sm transition-transform duration-200 translate-y-0', checked ? 'translate-x-4' : 'translate-x-0')} />
      </button>
      {label && <span className="text-sm text-text">{label}</span>}
    </label>
  )
}
