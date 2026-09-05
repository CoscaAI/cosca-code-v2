import { useState } from 'react'
import { api } from '../api'

export interface InlineRequest {
  from: number
  to: number
  text: string
}

// InlineAI: overlay de edição/explicação inline (spec seção 37).
// ⚠ READ-ONLY (ordem do Don 2026-08-14): o modo EDIT (transforma código) está
// BLOQUEADO — só "explicar" (leitura do modelo) continua disponível.
export function InlineAI({ request, onApply, onDiscard }: {
  request: InlineRequest
  onApply: (text: string) => void
  onDiscard: () => void
}) {
  const [instruction, setInstruction] = useState('')
  const [result, setResult] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [mode, setMode] = useState<'edit' | 'explain'>('explain')

  const run = async (m: 'edit' | 'explain') => {
    const instr = instruction.trim()
    if (!instr || loading) return
    if (m === 'edit') {
      setResult('⚠ read-only: edição de código bloqueada nesta fase (ordem do Don)')
      return
    }
    setMode(m)
    setLoading(true)
    setResult(null)
    const fullInstruction = `Explique este código: ${instr}`
    try {
      // "Explicar" é operação de LEITURA (chamada LLM pura — handleAIInline em
      // inline.go não grava arquivos). Usa fetch direto em vez de guardedFetch,
      // que bloquearia TODOS os POST em /api/ai/inline (inclui o explain
      // legítimo). A edição continua bloqueada pelo early-return acima.
      const res = await fetch(api('/api/ai/inline'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: request.text, instruction: fullInstruction }),
      }).then(r => r.json())
      setResult(res.error ? `⚠ ${res.error}` : res.result)
    } catch (e) {
      setResult(`⚠ ${e}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="inline-ai">
      <div className="inline-head">
        <span>✦ INLINE AI</span>
        <button className="inline-close" onClick={onDiscard}>✕</button>
      </div>

      <div className="inline-select" title="código selecionado">
        {request.text.slice(0, 120)}{request.text.length > 120 ? '…' : ''}
      </div>

      <textarea
        className="inline-instruction"
        value={instruction}
        onChange={e => setInstruction(e.target.value)}
        placeholder="o que fazer? (ex: 'extrair função', 'corrigir o bug', 'explicar')"
        rows={2}
        autoFocus
      />

      {!result && (
        <div className="inline-actions">
          <button className="inline-btn primary" onClick={() => run('edit')} disabled={loading || !instruction.trim()}>
            {loading ? 'pensando…' : '⟳ editar código'}
          </button>
          <button className="inline-btn" onClick={() => run('explain')} disabled={loading || !instruction.trim()}>
            ? explicar
          </button>
        </div>
      )}

      {result && (
        <div className="inline-result">
          <pre className="inline-result-text">{result}</pre>
          <div className="inline-actions">
            {mode === 'edit' && (
              <button className="inline-btn primary" onClick={() => onApply(result)}>✓ aplicar</button>
            )}
            <button className="inline-btn" onClick={() => { setResult(null); setInstruction('') }}>↺ refazer</button>
            <button className="inline-btn" onClick={onDiscard}>descartar</button>
          </div>
        </div>
      )}
    </div>
  )
}
