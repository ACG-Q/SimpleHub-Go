import { cn } from '../../utils/cn'

export function Table({ children, className }) {
  return (
    <div className={cn('overflow-x-auto rounded-lg', className)}>
      <table className="w-full border-collapse text-sm">{children}</table>
    </div>
  )
}

export function THead({ children }) {
  return (
    <thead>
      <tr className="border-b border-border">
        {children}
      </tr>
    </thead>
  )
}

export function Th({ children, className }) {
  return (
    <th className={cn('px-3.5 py-3 text-left text-xs font-semibold text-text-secondary uppercase tracking-wider whitespace-nowrap', className)}>
      {children}
    </th>
  )
}

export function Td({ children, className }) {
  return (
    <td className={cn('px-3.5 py-3 align-middle', className)}>
      {children}
    </td>
  )
}

export function Tr({ children, className, onClick }) {
  return (
    <tr
      onClick={onClick}
      className={cn(
        'border-b border-border transition-colors duration-150',
        onClick && 'cursor-pointer',
        'hover:bg-primary-bg/30',
        className,
      )}
    >
      {children}
    </tr>
  )
}

export function TBody({ children }) {
  return <tbody>{children}</tbody>
}
