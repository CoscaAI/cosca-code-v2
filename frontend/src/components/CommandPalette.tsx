import { useEffect, useRef, useState } from 'react'
import { apiAsync } from '../api'

export interface Command {
  id: string
  title: string
  hint?: string
  run: () => void
}

// Resultado de uma busca do Command Center — comando local, símbolo do
// backend (intel) ou resposta natural (chat real). NADA é mockado: se o
// backend não responde, a seção mostra "indisponível" (P14).
interface CenterItem {
  id: string
  kind: 'command' | 'symbol' | 'natural'
  title: string
  hint?: string
  run: () => void
}

// CommandCenter — o pilar 2 do professor: UMA entrada para controlar o Cosca.
//  1. Comandos locais (navegação/painéis) — rápidos.
//  2. Símbolos do projeto — busca REAL via /api/intel/symbols.
//  3. Linguagem natural — roteia pelo /api/ai/chat REAL (sem mock).
// Abre com Ctrl/Cmd+Shift+P.
export function CommandPalette({ commands, onClose, model }: {
  commands: Command[]
  onClose: () => void
  model?: string
}) {
  const [query, setQuery] = useState('')
  const [symbols, setSymbols] = useState<{ name: string; kind: string; file: string }[]>([])
  const [natural, setNatural] = useState<string | null>(null)
  const [naturalBusy, setNaturalBusy] = useState(false)
  const [symbolsUnavailable, setSymbolsUnavailable] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  // Foca no abrir.
  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  // Busca símbolos REAIS do backend quando o usuário digita (≥2 chars).
  useEffect(() => {
    if (query.trim().length < 2) {
      setSymbols([])
      return
    }
    let alive = true
    ;(async () => {
      try {
        const url = await apiAsync(`/api/intel/symbols?name=${encodeURIComponent(query.trim())}`)
        const resp = await fetch(url)
        if (!resp.ok) {
          if (alive) setSymbolsUnavailable(true)
          return
        }
        const data = (await resp.json()) as { symbols: { name: string; kind: string; file: string }[] }
        if (alive) {
          setSymbols(data.symbols ?? [])
          setSymbolsUnavailable(false)
        }
      } catch {
        if (alive) setSymbolsUnavailable(true)
      }
    })()
    return () => { alive = false }
  }, [query])

  // Comandos locais filtrados.
  const local = commands.filter(c => c.title.toLowerCase().includes(query.toLowerCase()))

  // Monta a lista unificada.
  const items: CenterItem[] = [
    ...local.map(c => ({ id: `cmd-${c.id}`, kind: 'command' as const, title: c.title, hint: c.hint, run: c.run })),
    ...symbols.slice(0, 5).map(s => ({
      id: `sym-${s.name}`, kind: 'symbol' as const, title: s.name,
      hint: `${s.kind} · ${s.file}`, run: () => {},
    })),
  ]

  // Linguagem natural: se o texto não casa com comandos, oferece perguntar
  // ao Cosca (rota pelo /api/ai/chat REAL).
  const showNatural = query.trim().length >= 6 && local.length === 0

  const askCosca = async () => {
    if (naturalBusy) return
    setNaturalBusy(true)
    setNatural(null)
    try {
      const url = await apiAsync('/api/ai/chat')
      const resp = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          // Usa o modelo selecionado no header (não um valor fixo).
          model: model || 'qwen2.5-coder:14b',
          messages: [{ role: 'user', content: query.trim() }],
        }),
      })
      if (!resp.ok) {
        setNatural('(resposta indisponível — backend sem resposta)')
        return
      }
      const data = (await resp.json()) as { reply?: string }
      setNatural(data.reply ?? '(sem resposta)')
    } catch {
      setNatural('(backend indisponível — não posso responder agora)')
    } finally {
      setNaturalBusy(false)
    }
  }

  const run = (c: CenterItem) => {
    onClose()
    if (c.kind === 'command') c.run()
  }

  return (
    <div className="palette-overlay" onClick={onClose}>
      <div className="palette center-palette" onClick={e => e.stopPropagation()}>
        <input
          ref={inputRef}
          className="palette-input"
          placeholder="comando, símbolo ou linguagem natural…"
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Escape') onClose()
            if (e.key === 'Enter') {
              if (local.length > 0) run(items[0])
              else if (showNatural) askCosca()
            }
          }}
        />
        <div className="palette-list">
          {items.map((c, i) => (
            <div
              key={c.id}
              className={`palette-item ${i === 0 ? 'selected' : ''} ${c.kind}`}
              onClick={() => run(c)}
            >
              <span className="palette-title">{c.title}</span>
              {c.hint && <span className="palette-hint">{c.hint}</span>}
            </div>
          ))}

          {symbolsUnavailable && query.trim().length >= 2 && (
            <div className="empty">busca de símbolos indisponível (backend sem intel)</div>
          )}

          {items.length === 0 && !showNatural && (
            <div className="empty">nenhum comando — continue digitando para perguntar ao Cosca</div>
          )}

          {showNatural && (
            <div className="palette-item natural" onClick={askCosca}>
              <span className="palette-title">✨ perguntar ao Cosca: “{query.trim()}”</span>
              <span className="palette-hint">roteado pelo /api/ai/chat real</span>
            </div>
          )}

          {natural !== null && (
            <div className="natural-reply">{natural}</div>
          )}
          {naturalBusy && <div className="empty">Cosca pensando…</div>}
        </div>
      </div>
    </div>
  )
}
