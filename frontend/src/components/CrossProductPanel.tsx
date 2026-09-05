import { useEffect, useState } from 'react'
import { apiAsync } from '../api'
import { guardedFetch } from '../execGuard'

// Product/Flow descrevem o ecossistema (§30).
interface Product { name: string; glyph: string; nodes: string[] }
interface Flow { name: string; desc: string; nodes: string[] }

interface Node {
  id: string
  type: string
  params?: Record<string, unknown>
  inputs?: string[]
}

interface GraphData { name: string; nodes: Node[] }

interface RunResult {
  name: string
  executed: number
  cached_hits: number
  results?: Record<string, unknown>
  error?: string
}

// CROSS_FLOW: fluxo canônico §30 — cinema → audio → music num único grafo.
const CROSS_FLOW: GraphData = {
  name: 'cross-flow',
  nodes: [
    { id: 'v', type: 'load_video', params: { path: 'intro.mp4' } },
    { id: 'a', type: 'extract_audio', params: { format: 'wav' }, inputs: ['v'] },
    { id: 'vol', type: 'volume', params: { volume: '3dB' }, inputs: ['a'] },
    { id: 'mp3', type: 'convert_audio', params: { format: 'mp3' }, inputs: ['vol'] },
  ],
}

// CrossProductPanel: o CROSS-PRODUCT (§30) — os produtos do ecossistema
// conversam num ÚNICO node graph. Cada nó é roteado para o executor do seu
// domínio (image/cinema/music/3d/game/sci) — o orquestrador unificado.
export function CrossProductPanel() {
  const [json, setJson] = useState(JSON.stringify(CROSS_FLOW, null, 2))
  const [graph, setGraph] = useState<GraphData | null>(CROSS_FLOW)
  const [products, setProducts] = useState<Product[]>([])
  const [flows, setFlows] = useState<Flow[]>([])
  const [run, setRun] = useState<RunResult | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState('')

  // Carrega produtos e fluxos do §30.
  useEffect(() => {
    let cancelled = false
    apiAsync('/api/engine/x/flows')
      .then(url => fetch(url))
      .then(r => r.json())
      .then((d: { products: Product[]; flows: Flow[] }) => {
        if (cancelled) return
        setProducts(d.products ?? [])
        setFlows(d.flows ?? [])
      })
      .catch(() => {})
    return () => { cancelled = true }
  }, [])

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

  const loadFlow = (f: Flow) => {
    // Constrói o grafo do fluxo (nós com input encadeado).
    const nodes: Node[] = f.nodes.map((type, i) => ({
      id: `n${i}`,
      type,
      params: i === 0 ? { path: 'intro.mp4' } : undefined,
      inputs: i > 0 ? [`n${i - 1}`] : undefined,
    }))
    setJson(JSON.stringify({ name: f.name.replace(/\W+/g, '-'), nodes }, null, 2))
    setGraph({ name: f.name.replace(/\W+/g, '-'), nodes })
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
      const res = await guardedFetch(await apiAsync('/api/engine/x/run'), {
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
      <div className="sidebar-title">CROSS-PRODUCT (§30)</div>

      {/* Produtos do ecossistema */}
      <div className="cross-products">
        {products.map(p => (
          <span key={p.name} className="cross-product" title={p.nodes.join(', ')}>
            {p.glyph} {p.name.replace('COSCA ', '')}
          </span>
        ))}
      </div>

      {/* Fluxos canônicos §30 */}
      <div className="sidebar-title">FLUXOS</div>
      <div className="cross-flows">
        {flows.map(f => (
          <button key={f.name} className="cross-flow" onClick={() => loadFlow(f)} title={f.desc}>
            <span className="cross-flow-name">{f.name}</span>
            <span className="cross-flow-desc">{f.desc}</span>
          </button>
        ))}
      </div>

      {/* Canvas: pipeline cross-product */}
      <div className="sidebar-title">PIPELINE</div>
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
          </div>
        ))}
      </div>

      {/* Ações */}
      <div className="graph-actions">
        <button onClick={execute} disabled={!graph || running}>{running ? '…' : 'Rodar fluxo'}</button>
        <button onClick={() => loadFlow(flows[0] ?? { name: 'cross', desc: '', nodes: [] })}>Exemplo</button>
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
              <code>{String(val).split('/').pop()}</code>
            </div>
          ))}
        </div>
      )}

      {error && <div className="graph-result error">{error}</div>}

      {/* JSON editável */}
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
