import { useState, useRef, useCallback } from 'react'

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function TagInput({ value = [], onChange, label, placeholder = '输入', validator }) {
  const [input, setInput] = useState('')
  const ref = useRef(null)

  const addTag = useCallback((raw) => {
    const t = raw.trim()
    if (!t || value.includes(t)) return
    if (validator && !validator(t)) return
    onChange([...value, t])
  }, [value, onChange, validator])

  const removeTag = useCallback((idx) => {
    onChange(value.filter((_, i) => i !== idx))
  }, [value, onChange])

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      if (input.trim()) { addTag(input); setInput('') }
    }
    if (e.key === 'Backspace' && !input && value.length) {
      onChange(value.slice(0, -1))
    }
  }

  const handleBlur = () => {
    if (input.trim()) { addTag(input); setInput('') }
  }

  const handlePaste = (e) => {
    const text = e.clipboardData.getData('text')
    if (/[,;\n]/.test(text)) {
      e.preventDefault()
      text.split(/[,;\n]+/).forEach(t => addTag(t))
    }
  }

  return (
    <div className="w-full">
      {label && <label className="block text-sm font-semibold mb-1.5 text-text">{label}</label>}
      <div className="flex flex-wrap items-center gap-1.5 px-3 py-2 rounded-lg border border-border bg-white/80 min-h-[42px] cursor-text transition-all duration-200 focus-within:border-primary-light focus-within:shadow-[0_0_0_3px_rgba(37,99,235,0.1)]"
        onClick={() => ref.current?.focus()}>
        {value.map((t, i) => (
          <span key={t + i} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-primary-bg text-primary text-sm max-w-full">
            <span className="truncate">{t}</span>
            <button type="button" onClick={(e) => { e.stopPropagation(); removeTag(i) }}
              className="text-primary/60 hover:text-danger cursor-pointer text-base leading-none">&times;</button>
          </span>
        ))}
        <input ref={ref} value={input} onChange={e => setInput(e.target.value)}
          onKeyDown={handleKeyDown} onBlur={handleBlur} onPaste={handlePaste}
          className="flex-1 min-w-[80px] outline-none text-sm bg-transparent text-text placeholder:text-text-muted"
          placeholder={value.length ? '' : placeholder} />
      </div>
    </div>
  )
}
