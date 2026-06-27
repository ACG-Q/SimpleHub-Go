import { forwardRef } from 'react'
import { cn } from '../../utils/cn'

export const SearchBox = forwardRef(({ value, onChange, placeholder = '鎼滅储...', className }, ref) => (
  <div className={cn('flex items-center gap-2 bg-white/70 backdrop-blur-sm border border-border rounded-xl px-3.5 py-2 transition-all duration-200 focus-within:border-primary-light focus-within:shadow-[0_0_0_3px_rgba(37,99,235,0.1)]', className)}>
    <svg className="w-4 h-4 text-text-muted shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
    <input
      ref={ref}
      type="text"
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      className="border-none outline-none bg-transparent font-sans text-sm text-text placeholder:text-text-muted flex-1 min-w-0"
    />
  </div>
))
SearchBox.displayName = 'SearchBox'
