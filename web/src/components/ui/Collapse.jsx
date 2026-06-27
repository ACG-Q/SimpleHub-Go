import { useState } from 'react'
import { cn } from '../../utils/cn'

export function Collapse({ children, className }) {
  return <div className={cn('space-y-3', className)}>{children}</div>
}

export function CollapsePanel({ title, defaultOpen, open: controlledOpen, onToggle, extra, children }) {
  const isControlled = controlledOpen !== undefined
  const [internalOpen, setInternalOpen] = useState(defaultOpen ?? true)
  const open = isControlled ? controlledOpen : internalOpen
  const handleClick = () => {
    if (isControlled) {
      onToggle?.()
    } else {
      setInternalOpen(!open)
    }
  }
  return (
    <div className="rounded-xl bg-white/70 backdrop-blur-sm border border-border-glass shadow-sm overflow-hidden">
      <div
        onClick={handleClick}
        className="flex items-center gap-2 px-4 py-3.5 cursor-pointer select-none transition-colors duration-200 hover:bg-primary-bg/20"
      >
        <svg
          className={cn('w-3 h-3 text-text-muted transition-transform duration-200 shrink-0', open && 'rotate-90')}
          viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"
        >
          <polyline points="9 18 15 12 9 6" />
        </svg>
        <div className="flex-1 flex items-center gap-2 text-sm font-semibold">{title}</div>
        {extra && <div onClick={e => e.stopPropagation()}>{extra}</div>}
      </div>
      {open && <div className="px-4 pb-4">{children}</div>}
    </div>
  )
}
