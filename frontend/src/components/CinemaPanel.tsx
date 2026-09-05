import { useState } from 'react'
import { apiAsync } from '../api'
import { guardedFetch } from '../execGuard'

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

interface RunResult {
  name: string
  executed: number
  cached_hits: number
  results?: Record<string, unknown>
  error?: string
}

// CINEMA_PIPELINE: fluxo cinematográfico do manifesto §8:
// VIDEO → EXTRACT_AUDIO → TRANSCRIÇÃO (futuro whisper) → FRAME (storyboard) → RENDER
const CINEMA_PIPELINE: GraphData = {
  name: 'cinema-shots',
  nodes: [
    { id: 'v', type: 'load_video', params: { path: 'intro.mp4' } },
    { id: 'p', type: 'probe', inputs: ['v'] },
    { id: 'a', type: 'extract_audio', params: { format: 'wav' }, inputs: ['v'] },
    { id: 'f', type: 'frame', params: { time: '00:00:01' }, inputs: ['v'] },
    { id: 'r', type: 'transcode', params: { quality: 'draft', format: 'mkv' }, inputs: ['v'] },
  ],
}

// CinemaPanel: o fluxo de trabalho cinematográfico (§8) — MAIN CANVAS do
// COSCA CINEMA. Consome POST /api/engine/cinema/run (Fase 4) e mostra os
// artefatos gerados (wav/png/mkv) pelo executor de vídeo (Media Engine ffmpeg).
export function CinemaPanel() {
  const [json, setJson] = useState(JSON.stringify(CINEMA_PIPELINE, null, 2))
  const [graph, setGraph] = useState<GraphData | null>(CINEMA_PIPELINE)
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
    if (g) { setGraph(g); setError(''); setRun(null) }
  }

  const execute = async () => {
    const g = parse()
    if (!g) return
    setError('')
    setRun(null)
    setRunning(true)
    try {
      const res = await guardedFetch(await apiAsync('/api/engine/cinema/run'), {
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

  const reset = () => {
    setJson(JSON.stringify(CINEMA_PIPELINE, null, 2))
    setGraph(CINEMA_PIPELINE)
    setRun(null)
    setError('')
  }

  return (
    <aside className="sidebar graph-sidebar">
      <div className="sidebar-title">COSCA CINEMA (§8)</div>

      {/* Canvas: pipeline cinematográfico */}
      <div className="graph-canvas">
        {graph?.nodes.map(n => (
          <div key={n.id} className="graph-node">
            <div className="graph-node-head">
              <span className="graph-node-id">{n.id}</span>
              <span className="graph-node-type">{n.type}</span>
            </div>
            {n.inputs && n.inputs.length > 0 && (
              <div className="graph-node-inputs">← {n.inputs.join(' + ')}</div>
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
      </div>

      {/* Ações */}
      <div className="graph-actions">
        <button onClick={execute} disabled={!graph || running}>{running ? '…' : 'Rodar pipeline'}</button>
        <button onClick={reset}>Exemplo</button>
      </div>

      {/* Resultado */}
      {run && run.error && <div className="graph-result error">{run.error}</div>}
      {run && !run.error && (
        <div className="graph-result ok">
          ✓ {run.name}: {run.executed} executados · {run.cached_hits} do cache
        </div>
      )}
      {run?.results && Object.keys(run.results).length > 0 && (
        <div className="graph-sigs">
          {Object.entries(run.results).map(([id, val]) => (
            <div key={id} className="graph-sig-row">
              <span>{id}</span>
              <code>{fmt(val)}</code>
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

// fmt formata o valor do resultado (caminho curto ou resumo).
function fmt(v: unknown): string {
  if (typeof v === 'string') return v.split('/').pop() ?? v
  if (typeof v === 'object' && v !== null) {
    const o = v as Record<string, unknown>
    return `duration=${o.duration}s · audio=${o.has_audio}`
  }
  return String(v)
}
