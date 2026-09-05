import { useEffect, useRef, useState } from 'react'
import { wsUrl } from '../api'

// Terminal: painel inferior com um shell real (PTY) via WebSocket. Sem xterm.js
// por ora — renderiza a saída como texto e envia input pelo prompt (a base do
// fluxo bidirecional; xterm.js entra como upgrade de renderização depois).
export function Terminal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [output, setOutput] = useState('')
  const [input, setInput] = useState('')
  const wsRef = useRef<WebSocket | null>(null)
  const outputRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const ws = new WebSocket(wsUrl('/api/term'))
    wsRef.current = ws
    ws.onmessage = (msg) => {
      // Assume mensagem binária (bytes do PTY).
      msg.data.arrayBuffer().then((buf: ArrayBuffer) => {
        setOutput(prev => prev + new TextDecoder().decode(buf))
      })
    }
    return () => ws.close()
  }, [open])

  // Auto-scroll.
  useEffect(() => {
    outputRef.current?.scrollTo({ top: outputRef.current.scrollHeight })
  }, [output])

  if (!open) return null

  const send = () => {
    wsRef.current?.send(new TextEncoder().encode(input + '\r'))
    setInput('')
  }

  return (
    <div className="terminal-panel">
      <div className="terminal-header">
        <span>TERMINAL</span>
        <button className="term-close" onClick={onClose}>✕</button>
      </div>
      <div ref={outputRef} className="terminal-output">
        {output}
        <span className="term-prompt">$ </span>
        <input
          className="term-input"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') send() }}
          autoFocus
        />
      </div>
    </div>
  )
}
