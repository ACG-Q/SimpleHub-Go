import { forwardRef } from 'react'
import { cn } from '../../utils/cn'

export const Input = forwardRef(({ className, label, error, rightElement, ...props }, ref) => (
  <div className="w-full">
    {label && <label className="block text-sm font-semibold mb-1.5 text-text">{label}</label>}
    <div className="relative">
      <input
        ref={ref}
        className={cn(
          'w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 font-sans text-sm text-text placeholder:text-text-muted transition-all duration-200 outline-none',
          'focus:border-primary-light focus:shadow-[0_0_0_3px_rgba(37,99,235,0.1)]',
          rightElement && 'pr-10',
          error && 'border-danger focus:border-danger focus:shadow-[0_0_0_3px_rgba(239,68,68,0.1)]',
          className,
        )}
        {...props}
      />
      {rightElement && (
        <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center">
          {rightElement}
        </div>
      )}
    </div>
    {error && <p className="mt-1 text-xs text-danger">{error}</p>}
  </div>
))
Input.displayName = 'Input'

export const Textarea = forwardRef(({ className, label, ...props }, ref) => (
  <div className="w-full">
    {label && <label className="block text-sm font-semibold mb-1.5 text-text">{label}</label>}
    <textarea
      ref={ref}
      className={cn(
        'w-full px-3.5 py-2.5 rounded-lg border border-border bg-white/80 font-sans text-sm text-text placeholder:text-text-muted transition-all duration-200 outline-none resize-y',
        'focus:border-primary-light focus:shadow-[0_0_0_3px_rgba(37,99,235,0.1)]',
        className,
      )}
      {...props}
    />
  </div>
))
Textarea.displayName = 'Textarea'
