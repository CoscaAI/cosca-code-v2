import { useState } from 'react'
import { apiAsync } from '../api'

// Node descreve um nó do node graph (§21).
interface Node {
  id: string
  type: string
  params?: Record<string, unknown>
  inputs?: string[]
}

interface GraphData {
  name: string
  nodes: Node[]
}

interface ValidateResult {
  valid: boolean
  name: string
  nodes: number
  order?: string[]
  error?: string
}

interface RunResult {
  name: string
  executed: number
  cached_hits: number
  results?: Record<string, string>
  error?: string
}

// WORKFLOW_EXAMPLE é o pipeline canônico do manifesto §21:
// VIDEO → EXTRACT_AUDIO → WHISPER → TRANSLATE → SUBTITLE → RENDER
const WORKFLOW_EXAMPLE: GraphData = {
  name: 'video-subtitles',
  nodes: [
    { id: 'v', type: 'load_video', params: { path: 'intro.mp4' } },
    { id: 'a', type: 'extract_audio', inputs: ['v'] },
    { id: 'w', type: 'whisper', params: { model: 'large-v3' }, inputs: ['a'] },
    { id: 't', type: 'translate', params: { target: 'pt-BR' }, inputs: ['w'] },
    { id: 's', type: 'subtitle', inputs: ['t'] },
    { id: 'r', type: 'render', params: { format: 'mp4' }, inputs: ['v', 's'] },
  ],
}

// NodeGraphPanel: editor visual do node graph (§21) — o MAIN CANVAS criativo.
// Consome POST /api/engine/graph/validate e /sig (Fase 2.2). Mostra os nós em
// ordem topológica com inputs, valida DAG e exibe assinaturas (cache keys §23).
export function NodeGraphPanel() {
  const [json, setJson] = useState(JSON.stringify(WORKFLOW_EXAMPLE, null, 2))
  const [graph, setGraph] = useState<GraphData | null>(WORKFLOW_EXAMPLE)
  const [result, setResult] = useState<ValidateResult | null>(null)
  const [sigs, setSigs] = useState<Record<string, string> | null>(null)
  const [run, setRun] = useState<RunResult | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState('')

  const parse = (): GraphData | null => {
    try {
      const g = JSON.parse(json) as GraphData
      if (!g.name || !Array.isArray(g.nodes)) throw new Error('grafo inválido')
      return g
    } catch (e) {
      setError(`JSON inválido: ${(e as Error).message}`)
      return null
    }
  }

  const apply = () => {
    const g = parse()
    if (g) {
      setGraph(g)
      setError('')
      setResult(null)
      setSigs(null)
    }
  }

  const validate = async () => {
    const g = parse()
    if (!g) return
    setError('')
    setResult(null)
    try {
      const res = await fetch(await apiAsync('/api/engine/graph/validate'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(g),
      }).then(r => r.json())
      if (res.error) { setError(res.error); return }
      setResult(res)
    } catch (e) {
      setError(`falha na validação: ${(e as Error).message}`)
    }
  }

  const signatures = async () => {
    const g = parse()
    if (!g) return
    setError('')
    setSigs(null)
    try {
      const res = await fetch(await apiAsync('/api/engine/graph/sig'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(g),
      }).then(r => r.json())
      if (res.error) { setError(res.error); return }
      setSigs(res.signatures ?? {})
    } catch (e) {
      setError(`falha nas assinaturas: ${(e as Error).message}`)
    }
  }

  const reset = () => {
    setJson(JSON.stringify(WORKFLOW_EXAMPLE, null, 2))
    setGraph(WORKFLOW_EXAMPLE)
    setResult(null)
    setSigs(null)
    setRun(null)
    setError('')
  }

  const execute = async () => {
    const g = parse()
    if (!g) return
    setError('')
    setRun(null)
    setRunning(true)
    try {
      const res = await fetch(await apiAsync('/api/engine/graph/run'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(g),
      }).then(r => r.json())
      if (res.error) { setError(res.error); return }
      setRun(res)
    } catch (e) {
      setError(`falha na execução: ${(e as Error).message}`)
    } finally {
      setRunning(false)
    }
  }

  return (
    <aside className="sidebar graph-sidebar">
      <div className="sidebar-title">NODE GRAPH (§21)</div>

      {/* Canvas: nós + conexões em ordem topológica */}
      <div className="graph-canvas">
        {graph?.nodes.map(n => (
          <div key={n.id} className="graph-node">
            <div className="graph-node-head">
              <span className="graph-node-id">{n.id}</span>
              <span className="graph-node-type">{n.type}</span>
            </div>
            {n.inputs && n.inputs.length > 0 && (
              <div className="graph-node-inputs">
                ← {n.inputs.join(' + ')}
              </div>
            )}
            {n.params && Object.keys(n.params).length > 0 && (
              <div className="graph-node-params">
                {Object.entries(n.params).map(([k, v]) => (
                  <span key={k} className="graph-param">{k}: {String(v)}</span>
                ))}
              </div>
            )}
          </div>
        ))}
        {!graph && <div className="empty">Sem grafo</div>}
      </div>

      {/* Ações */}
      <div className="graph-actions">
        <button onClick={validate} disabled={!graph}>Validar DAG</button>
        <button onClick={signatures} disabled={!graph}>Assinaturas</button>
        <button onClick={execute} disabled={!graph || running}>{running ? '…' : 'Executar'}</button>
        <button onClick={reset}>Exemplo</button>
      </div>

      {/* Resultado da validação */}
      {result && result.error && <div className="graph-result error">{result.error}</div>}
      {result && result.valid && (
        <div className="graph-result ok">
          ✓ {result.name}: {result.nodes} nós · ordem: {result.order?.join(' → ')}
        </div>
      )}

      {/* Assinaturas (cache keys §23) */}
      {sigs && (
        <div className="graph-sigs">
          {Object.entries(sigs).map(([id, sig]) => (
            <div key={id} className="graph-sig-row">
              <span>{id}</span>
              <code>{sig.slice(0, 12)}…</code>
            </div>
          ))}
        </div>
      )}

      {/* Resultado da execução (Fase 3 — pipeline de imagem) */}
      {run && run.error && <div className="graph-result error">{run.error}</div>}
      {run && !run.error && (
        <div className="graph-result ok">
          ✓ {run.name}: {run.executed} executados · {run.cached_hits} do cache
        </div>
      )}
      {run?.results && Object.keys(run.results).length > 0 && (
        <div className="graph-sigs">
          {Object.entries(run.results).map(([id, path]) => (
            <div key={id} className="graph-sig-row">
              <span>{id}</span>
              <code>{String(path).split('/').pop()}</code>
            </div>
          ))}
        </div>
      )}

      {error && <div className="graph-result error">{error}</div>}

      {/* JSON editável */}
      <div className="sidebar-title">JSON</div>
      <textarea
        className="graph-json"
        value={json}
        onChange={e => setJson(e.target.value)}
        spellCheck={false}
      />
      <div className="graph-actions">
        <button onClick={apply}>Aplicar</button>
      </div>
    </aside>
  )
}
