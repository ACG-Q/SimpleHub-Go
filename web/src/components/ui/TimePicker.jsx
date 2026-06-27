import { useState, useRef, useEffect, useCallback } from 'react'

const HOURS = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'))
const MINUTES = ['00', '15', '30', '45']

export function TimePicker ({ value = '', onChange, label, allowClear }) {
  const [open, setOpen] = useState(false)
  const [tempH, setTempH] = useState('')
  const [tempM, setTempM] = useState('')
  const wrapperRef = useRef(null)
  const hourRef = useRef(null)
  const minuteRef = useRef(null)

  const parts = (value || '').split(':')
  const h = parts[0] || ''
  const m = parts[1] || ''

  const openPicker = () => {
    setTempH(h || '00')
    setTempM(m || '00')
    setOpen(true)
  }

  const selectTime = useCallback((newH, newM) => {
    onChange(`${newH}:${newM}`)
    setOpen(false)
  }, [onChange])

  useEffect(() => {
    if (!open) return
    const handler = (e) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  useEffect(() => {
    if (!open) return
    if (hourRef.current) {
      const el = hourRef.current.querySelector(`[data-h="${tempH}"]`)
      el?.scrollIntoView({ block: 'center' })
    }
    if (minuteRef.current) {
      const el = minuteRef.current.querySelector(`[data-m="${tempM}"]`)
      el?.scrollIntoView({ block: 'center' })
    }
  }, [open, tempH, tempM])

  return (
    <div className="w-full" ref={wrapperRef}>
      {label && <label className="block text-sm font-semibold mb-1.5 text-text">{label}</label>}
      <div className="relative">
        <div onClick={openPicker}
          className={`flex items-center h-[42px] px-3.5 py-2.5 rounded-lg border cursor-pointer text-sm transition-all duration-200
            ${value ? 'border-border bg-white/80 text-text' : 'border-border bg-white/80 text-text-muted'}
            hover:border-primary-light hover:shadow-[0_0_0_3px_rgba(37,99,235,0.1)]
            ${open ? 'border-primary-light shadow-[0_0_0_3px_rgba(37,99,235,0.1)]' : ''}`}>
          <svg className="w-4 h-4 mr-2 shrink-0 text-text-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          <span className="flex-1">{value || '设置时间'}</span>
          {value && allowClear && (
            <button type="button" onClick={(e) => { e.stopPropagation(); onChange('') }}
              className="mr-1 text-text-muted hover:text-danger cursor-pointer">
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          )}
          <svg className={`w-4 h-4 text-text-muted transition-transform ${open ? 'rotate-180' : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="6 9 12 15 18 9"/></svg>
        </div>

        {open && (
          <div className="absolute left-0 right-0 mt-1 z-50 rounded-lg border border-border bg-white shadow-lg overflow-hidden">
            <div className="flex items-stretch" style={{ height: '200px' }}>
              <div ref={hourRef} className="flex-1 overflow-y-auto py-1 hide-scrollbar border-r border-border/50 scroll-smooth">
                <div className="px-2 py-1 text-xs text-text-muted text-center font-medium sticky top-0 bg-white z-10">时</div>
                {HOURS.map(v => (
                  <div key={v} data-h={v} onClick={() => selectTime(v, tempM)}
                    className={`cursor-pointer mx-1 px-2 py-1.5 text-center text-sm leading-5 select-none rounded ${v === tempH ? 'bg-primary text-white font-medium' : 'text-text hover:bg-slate-100'}`}>
                    {v}
                  </div>
                ))}
              </div>
              <div ref={minuteRef} className="flex-1 overflow-y-auto py-1 hide-scrollbar scroll-smooth">
                <div className="px-2 py-1 text-xs text-text-muted text-center font-medium sticky top-0 bg-white z-10">分</div>
                {MINUTES.map(v => (
                  <div key={v} data-m={v} onClick={() => selectTime(tempH, v)}
                    className={`cursor-pointer mx-1 px-2 py-1.5 text-center text-sm leading-5 select-none rounded ${v === tempM ? 'bg-primary text-white font-medium' : 'text-text hover:bg-slate-100'}`}>
                    {v}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
