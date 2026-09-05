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

interface RunResult {
  name: string
  executed: number
  cached_hits: number
  results?: Record<string, unknown>
  error?: string
}

// MUSIC_PIPELINE: fluxo do COSCA MUSIC (§9) — trilha completa:
// AUDIO → PROBE → CONVERT (exportação) → VOLUME (mixer) 
const MUSIC_PIPELINE: GraphData = {
  name: 'music-master',
  nodes: [
    { id: 'a', type: 'load_audio', params: { path: 'tone.wav' } },
    { id: 'p', type: 'probe_audio', inputs: ['a'] },
    { id: 'c', type: 'convert_audio', params: { format: 'mp3' }, inputs: ['a'] },
    { id: 'v', type: 'volume', params: { volume: '3dB' }, inputs: ['a'] },
  ],
}

// MusicPanel: a DAW básica do COSCA MUSIC (§9) — MAIN CANVAS musical.
// Consome POST /api/engine/music/run (Fase 5) e mostra os artefatos
// (mp3/wav) gerados pelo executor de áudio (Media Engine ffmpeg).
export function MusicPanel() {
  const [json, setJson] = useState(JSON.stringify(MUSIC_PIPELINE, null, 2))
  const [graph, setGraph] = useState<GraphData | null>(MUSIC_PIPELINE)
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
      const res = await fetch(await apiAsync('/api/engine/music/run'), {
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
    setJson(JSON.stringify(MUSIC_PIPELINE, null, 2))
    setGraph(MUSIC_PIPELINE)
    setRun(null)
    setError('')
  }

  return (
    <aside className="sidebar graph-sidebar">
      <div className="sidebar-title">COSCA MUSIC (§9)</div>

      {/* Canvas: pipeline musical */}
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
        <button onClick={execute} disabled={!graph || running}>{running ? '…' : 'Processar trilha'}</button>
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
    return `codec=${o.codec ?? '?'} · ch=${o.channels ?? '?'} · ${o.sample_rate ?? ''}Hz`
  }
  return String(v)
}
