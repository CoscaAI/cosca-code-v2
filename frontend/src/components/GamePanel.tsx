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

interface DesignResult {
  source: string
  path: string
  entities: number
  scene?: Record<string, unknown>
  error?: string
}

// GAME_PIPELINE: fluxo do COSCA GAME (§10) — protótipo de plataforma:
// CENA → INFO (entidades/componentes) → ADD ENTITY (protótipo jogável)
const GAME_PIPELINE: GraphData = {
  name: 'platformer',
  nodes: [
    { id: 's', type: 'load_scene', params: { path: 'level.json' } },
    { id: 'i', type: 'scene_info', inputs: ['s'] },
    {
      id: 'a', type: 'add_entity',
      params: {
        id: 'coin_1', name: 'coin',
        components: [{ type: 'transform' }, { type: 'score', params: { value: 10 } }],
      },
      inputs: ['s'],
    },
  ],
}

// GamePanel: o COSCA GAME (§10) — MAIN CANVAS de jogo. Consome
// POST /api/engine/game/run (Fase 7) e mostra o protótipo da cena
// (entidades/componentes) pela Game Engine ECS pura Go.
export function GamePanel() {
  const [json, setJson] = useState(JSON.stringify(GAME_PIPELINE, null, 2))
  const [graph, setGraph] = useState<GraphData | null>(GAME_PIPELINE)
  const [run, setRun] = useState<RunResult | null>(null)
  const [design, setDesign] = useState<DesignResult | null>(null)
  const [description, setDescription] = useState('crie um jogo de plataforma com um herói e moedas')
  const [designing, setDesigning] = useState(false)
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
      const res = await fetch(await apiAsync('/api/engine/game/run'), {
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
    setJson(JSON.stringify(GAME_PIPELINE, null, 2))
    setGraph(GAME_PIPELINE)
    setRun(null)
    setDesign(null)
    setError('')
  }

  // §31: o Don descreve o jogo, a IA local gera o level.json.
  const designGame = async () => {
    if (!description.trim() || designing) return
    setError('')
    setDesign(null)
    setDesigning(true)
    try {
      const res = await fetch(await apiAsync('/api/engine/game/design'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ description }),
      }).then(r => r.json())
      if (res.error) { setError(res.error); return }
      setDesign(res)
    } catch (e) {
      setError(`falha na geração: ${(e as Error).message}`)
    } finally {
      setDesigning(false)
    }
  }

  return (
    <aside className="sidebar graph-sidebar">
      <div className="sidebar-title">COSCA GAME (§10)</div>

      {/* §31 — IA→projeto */}
      <div className="sidebar-title">IA → JOGO (§31)</div>
      <div className="design-box">
        <textarea
          className="design-input"
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder="Descreva o jogo em linguagem natural…"
          spellCheck={false}
        />
        <button className="design-btn" onClick={designGame} disabled={designing}>
          {designing ? '…gerando (qwen local)' : '✨ Gerar cena'}
        </button>
        {design && !design.error && (
          <div className="design-result">
            ✓ {design.entities} entidades geradas por IA ({design.source})
            <br />
            <code>{design.path}</code>
          </div>
        )}
      </div>

      {/* Canvas: pipeline de jogo */}
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
        <button onClick={execute} disabled={!graph || running}>{running ? '…' : 'Montar protótipo'}</button>
        <button onClick={reset}>Exemplo</button>
      </div>

      {/* Resultado — protótipo da cena */}
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

// fmt formata o resultado do protótipo.
function fmt(v: unknown): string {
  if (typeof v === 'string') return v.split('/').pop() ?? v
  if (typeof v === 'object' && v !== null) {
    const o = v as Record<string, unknown>
    if (o.entities !== undefined) return `entities=${o.entities} · players=${o.players} · enemies=${o.enemies}`
    if (o.added !== undefined) return `+${o.id} (${o.name})`
    return JSON.stringify(o).slice(0, 60)
  }
  return String(v)
}
