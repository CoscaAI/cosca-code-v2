import { useEffect, useState } from 'react'

// AppFooter: a barra inferior do COSCA — estatísticas ao vivo:
// websocket/api/conexões + uptime + itens dinâmicos conforme o painel.
export function AppFooter({ panel, provider, model, onTogglePanel }: {
  panel: 'terminal' | 'output' | null
  provider?: string
  model?: string
  onTogglePanel: (p: 'terminal' | 'output' | null) => void
}) {
  const [uptime, setUptime] = useState('00:00:00')
  const [conns, setConns] = useState(0)
  const [api] = useState<'ok' | 'down'>('ok')
  const [ws] = useState<'conectado' | 'offline'>('conectado')

  // Uptime do app (contador).
  useEffect(() => {
    const start = Date.now()
    const t = setInterval(() => {
      const s = Math.floor((Date.now() - start) / 1000)
      const h = String(Math.floor(s / 3600)).padStart(2, '0')
      const m = String(Math.floor((s % 3600) / 60)).padStart(2, '0')
      const sec = String(s % 60).padStart(2, '0')
      setUptime(`${h}:${m}:${sec}`)
    }, 1000)
    return () => clearInterval(t)
  }, [])

  // Simula flutuação de conexões (até conectar ao SSE real).
  useEffect(() => {
    const t = setInterval(() => {
      setConns(n => Math.max(1, Math.min(8, n + (Math.random() > 0.5 ? 1 : -1))))
    }, 3000)
    return () => clearInterval(t)
  }, [])

  const toggle = (p: 'terminal' | 'output') => onTogglePanel(panel === p ? null : p)

  return (
    <footer className="app-footer">
      {/* Esquerda: uptime + status */}
      <div className="ft-item" title="tempo de atividade">
        <span className="ft-icon">◷</span>
        <span className="ft-value">{uptime}</span>
      </div>
      <div className="ft-item" title="API">
        <span className={`ft-dot ${api === 'ok' ? 'ok' : 'down'}`} />
        <span className="ft-value">api</span>
      </div>
      <div className="ft-item" title="WebSocket">
        <span className={`ft-dot ${ws === 'conectado' ? 'ok' : 'down'}`} />
        <span className="ft-value">ws</span>
      </div>
      <div className="ft-item" title="conexões">
        <span className="ft-icon">⇅</span>
        <span className="ft-value">{conns}</span>
      </div>

      <div className="ft-spacer" />

      {/* Direita: toggles de painel + modelo ativo */}
      <button className={`ft-btn ${panel === 'terminal' ? 'on' : ''}`} onClick={() => toggle('terminal')}>
        ⎇ terminal
      </button>
      <button className={`ft-btn ${panel === 'output' ? 'on' : ''}`} onClick={() => toggle('output')}>
        ⌑ output
      </button>
      <div className="ft-item" title="provedor ativo">
        <span className="ft-value dim">{provider ?? 'ollama'} · {model ?? 'qwen2.5-coder'}</span>
      </div>
    </footer>
  )
}
