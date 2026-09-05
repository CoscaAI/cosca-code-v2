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

interface DesignResult {
  source: string
  path: string
  kind?: string
  experiment?: Record<string, unknown>
  error?: string
}

// SCI_PIPELINE: fluxo do COSCA SCIENTIFIC (§11/§12) — experimento + comparação:
// BASELINE (calculado) → SWEEP (simulado) → LAB BEST (A/B)
const SCI_PIPELINE: GraphData = {
  name: 'lab-sweep',
  nodes: [
    {
      id: 'b', type: 'run_experiment',
      params: { id: 'exp-baseline', name: 'baseline', kind: 'calculated', expr: 'constant', value: 0.72 },
    },
    {
      id: 's', type: 'run_experiment',
      params: { id: 'exp-sweep', name: 'sweep-1', kind: 'simulated', expr: 'monte_carlo', samples: 1000, seed: 42 },
    },
    { id: 'best', type: 'lab_best', params: { metric: 'result' } },
  ],
}

// ScientificPanel: o COSCA SCIENTIFIC (§11) + COSCA LAB (§12) — MAIN CANVAS
// científico. Consome POST /api/engine/sci/run (Fase 8) e mostra experimentos
// reprodutíveis (registrados em experiments.yaml) com comparação A/B.
export function ScientificPanel() {
  const [json, setJson] = useState(JSON.stringify(SCI_PIPELINE, null, 2))
  const [graph, setGraph] = useState<GraphData | null>(SCI_PIPELINE)
  const [run, setRun] = useState<RunResult | null>(null)
  const [design, setDesign] = useState<DesignResult | null>(null)
  const [description, setDescription] = useState('compare a acurácia de dois modelos no dataset de imagens')
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
      const res = await guardedFetch(await apiAsync('/api/engine/sci/run'), {
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
    setJson(JSON.stringify(SCI_PIPELINE, null, 2))
    setGraph(SCI_PIPELINE)
    setRun(null)
    setDesign(null)
    setError('')
  }

  // §31: o Don descreve o experimento, a IA local gera o experiment.json.
  const designExperiment = async () => {
    if (!description.trim() || designing) return
    setError('')
    setDesign(null)
    setDesigning(true)
    try {
      const res = await guardedFetch(await apiAsync('/api/engine/sci/design'), {
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
      <div className="sidebar-title">COSCA SCIENTIFIC (§11)</div>

      {/* §31 — IA→experimento */}
      <div className="sidebar-title">IA → EXPERIMENTO (§31)</div>
      <div className="design-box">
        <textarea
          className="design-input"
          value={description}
          onChange={e => setDescription(e.target.value)}
          placeholder="Descreva o experimento em linguagem natural…"
          spellCheck={false}
        />
        <button className="design-btn" onClick={designExperiment} disabled={designing}>
          {designing ? '…gerando (qwen local)' : '✨ Gerar experimento'}
        </button>
        {design && !design.error && (
          <div className="design-result">
            ✓ {design.kind} — gerado por IA ({design.source})
            <br />
            <code>{design.path}</code>
          </div>
        )}
      </div>

      {/* Canvas: pipeline científico */}
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
        <button onClick={execute} disabled={!graph || running}>{running ? '…' : 'Rodar experimento'}</button>
        <button onClick={reset}>Exemplo</button>
      </div>

      {/* Resultado — experimento + melhor */}
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

// fmt formata o resultado científico.
function fmt(v: unknown): string {
  if (typeof v === 'string') return v.split('/').pop() ?? v
  if (typeof v === 'object' && v !== null) {
    const o = v as Record<string, unknown>
    if (o.result !== undefined) return `${o.id} [${o.kind}] = ${o.result}`
    if (o.found !== undefined) return `melhor: ${o.id} = ${o.value} (de ${o.count})`
    return JSON.stringify(o).slice(0, 60)
  }
  return String(v)
}
