import { createContext, useContext, useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '../../utils/cn'

const ModalCtx = createContext(null)

export function ModalHost({ children }) {
  const ref = useRef(null)
  return (
    <ModalCtx.Provider value={ref}>
      {children}
      <div ref={ref} />
    </ModalCtx.Provider>
  )
}

export function Modal({ open, onClose, title, children, footer, className, size = 'md' }) {
  const portalRef = useContext(ModalCtx)
  const sizes = { sm: 'max-w-sm', md: 'max-w-lg', lg: 'max-w-2xl', xl: 'max-w-4xl', full: 'max-w-[90vw]' }

  useEffect(() => {
    if (!open) return
    const handler = (e) => { if (e.key === 'Escape') onClose?.() }
    document.addEventListener('keydown', handler)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', handler)
      document.body.style.overflow = ''
    }
  }, [open, onClose])

  if (!open || !portalRef?.current) return null

  return createPortal(
    <div
      className="fixed inset-0 z-60 flex items-center justify-center p-4"
      onClick={(e) => { if (e.target === e.currentTarget) onClose?.() }}
    >
      <div className="fixed inset-0 bg-black/20 backdrop-blur-sm" />
      <div className={cn(
        'relative w-full bg-white/70 backdrop-blur-xl border border-border-glass rounded-2xl shadow-2xl max-h-[85vh] overflow-y-auto anim-in',
        sizes[size],
        className,
      )}>
        <div className="flex items-center justify-between px-6 pt-5 pb-0">
          <h3 className="text-lg font-bold text-text">{title}</h3>
          <button
            onClick={onClose}
            className="w-7 h-7 flex items-center justify-center rounded-md text-text-secondary hover:bg-danger-bg hover:text-danger transition-all duration-200 text-lg cursor-pointer"
          >
            &times;
          </button>
        </div>
        <div className="px-6 py-5">{children}</div>
        {footer && (
          <div className="flex justify-end gap-2 px-6 py-4 border-t border-border">
            {footer}
          </div>
        )}
      </div>
    </div>,
    portalRef.current
  )
}
