import { useEffect, useRef, useState } from 'react'
import { api } from '../api'

interface ChatMessage { role: string; content: string }
interface ModelInfo { id: string; name: string }

// AIPanel: o painel AI nativo (spec seções 36-38) — conversa com o modelo via
// Provider Engine. Entende o contexto do workspace e responde no editor.
export function AIPanel({ provider: providerProp, model: modelProp }: { provider?: string; model?: string }) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [models, setModels] = useState<ModelInfo[]>([])
  const [provider, setProvider] = useState('')
  const [model, setModel] = useState('')
  const [loading, setLoading] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  // Carrega os modelos do backend. Prefere o modelo escolhido no header (se o
  // backend o informa); senão usa o primeiro disponível.
  useEffect(() => {
    fetch(api('/api/ai/models'))
      .then(r => r.json())
      .then((d: { provider?: string; models?: ModelInfo[] }) => {
        setProvider(d.provider ?? '')
        setModels(d.models ?? [])
        const list = d.models ?? []
        const preferred = modelProp && list.some(m => m.id === modelProp) ? modelProp : list[0]?.id
        if (preferred) setModel(preferred)
      })
      .catch(() => {})
  }, [])

  // O header é a fonte do modelo; a troca manual no dropdown continua local.
  useEffect(() => {
    if (modelProp) setModel(modelProp)
  }, [modelProp])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading])

  const send = async () => {
    const text = input.trim()
    if (!text || loading) return
    const next = [...messages, { role: 'user', content: text }]
    setMessages(next)
    setInput('')
    setLoading(true)
    try {
      const res = await fetch(api('/api/ai/chat'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ messages: next, model }),
      }).then(r => r.json())
      if (res.error) {
        setMessages([...next, { role: 'assistant', content: `⚠ ${res.error}` }])
      } else {
        setMessages([...next, { role: 'assistant', content: res.reply }])
      }
    } catch (e) {
      setMessages([...next, { role: 'assistant', content: `⚠ ${e}` }])
    } finally {
      setLoading(false)
    }
  }

  return (
    <aside className="sidebar ai-panel">
      <div className="sidebar-title">AI — {provider || providerProp || 'sem provider'}</div>
      {(models.length > 0 || !!model) && (
        <select className="model-select" value={model} onChange={e => setModel(e.target.value)}>
          {(model && !models.some(m => m.id === model) ? [{ id: model, name: model }, ...models] : models)
            .map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
        </select>
      )}

      <div className="ai-messages">
        {messages.map((m, i) => (
          <div key={i} className={`ai-msg ai-${m.role}`}>
            <span className="ai-role">{m.role === 'user' ? 'você' : 'cosca'}</span>
            <div className="ai-content">{m.content}</div>
          </div>
        ))}
        {loading && <div className="ai-msg ai-assistant"><div className="ai-content">pensando…</div></div>}
        <div ref={bottomRef} />
      </div>

      <div className="ai-input-box">
        <textarea
          className="ai-input"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send() } }}
          placeholder="pergunte sobre o projeto… (Enter envia)"
          rows={3}
        />
        <button className="ai-send" onClick={send} disabled={loading}>→</button>
      </div>
    </aside>
  )
}
