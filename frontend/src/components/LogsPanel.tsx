import { useState } from 'react'

interface LogsPanelProps {
  logs: string
  success: boolean
}

export default function LogsPanel({ logs, success }: LogsPanelProps) {
  const [isOpen, setIsOpen] = useState(false)

  if (!logs) return null

  return (
    <div className="collapse collapse-arrow bg-base-200">
      <input type="checkbox" checked={isOpen} onChange={() => setIsOpen(!isOpen)} />
      <div className="collapse-title text-xl font-medium">
        Compilation Logs {success ? '✓' : '✗'}
      </div>
      <div className="collapse-content">
        <pre className="text-xs overflow-x-auto bg-base-100 p-4 rounded">
          {logs}
        </pre>
      </div>
    </div>
  )
}
