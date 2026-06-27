import { forwardRef } from 'react'
import { cn } from '../../utils/cn'

const variants = {
  primary: 'bg-gradient-to-r from-primary to-primary-light text-white shadow-md shadow-primary/20 hover:shadow-lg hover:shadow-primary/30 hover:-translate-y-px active:translate-y-0',
  ghost: 'bg-white/70 backdrop-blur-sm border border-border text-text hover:border-primary-light hover:text-primary active:bg-primary-bg',
  cta: 'bg-gradient-to-r from-cta to-cta-light text-white shadow-md shadow-cta/20 hover:shadow-lg hover:shadow-cta/30 hover:-translate-y-px active:translate-y-0',
  danger: 'bg-danger/10 text-danger border border-danger/20 hover:bg-danger/20 active:bg-danger/30',
}

const sizes = {
  sm: 'px-3 py-1.5 text-xs gap-1.5',
  md: 'px-4.5 py-2 text-sm gap-1.5',
  icon: 'w-8 h-8 p-0 justify-center text-sm',
}

export const Button = forwardRef(({ variant = 'ghost', size = 'md', className, children, ...props }, ref) => (
  <button
    ref={ref}
    className={cn(
      'inline-flex items-center justify-center rounded-lg font-semibold transition-all duration-200 cursor-pointer select-none active:scale-[0.97] disabled:opacity-50 disabled:pointer-events-none',
      variants[variant],
      sizes[size],
      className,
    )}
    {...props}
  >
    {children}
  </button>
))
Button.displayName = 'Button'
