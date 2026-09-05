import { useEffect, useRef, useState } from 'react'
import { apiAsync } from '../api'
import { guardedFetch } from '../execGuard'

// =============================================================================
// COSCA CANVAS — node editor visual estilo ComfyUI, PRÓPRIO (§21)
//
// Princípios (P1 do Blueprint): grafo = dados puros, execução derivada.
// O Don arrasta nós da paleta, conecta sockets por drag, e roda o pipeline
// no orquestrador cross-product (/api/engine/x/run). O canvas é puro React
// + SVG — zero lib de node graph (implementação própria).
// =============================================================================

// Socket: ponto de conexão de um nó.
interface Socket {
  id: string   // ex.: "load/output"
  label: string
  kind: 'input' | 'output'
  color: string
}

// CanvasNode: nó posicionado no canvas.
interface CanvasNode {
  id: string
  type: string
  label: string
  color: string
  x: number
  y: number
  inputs: Socket[]
  outputs: Socket[]
  params?: Record<string, unknown>
}

// Link: conexão entre dois sockets.
interface Link {
  id: string
  from: string // "nodeId/output"
  to: string   // "nodeId/input"
}

// Paleta de nós do ecossistema (§30) — cada domínio com cor própria.
interface PaletteEntry {
  type: string
  label: string
  color: string
  inputs: { label: string }[]
  outputs: { label: string }[]
  params?: Record<string, unknown>
}

const PALETTE: Record<string, PaletteEntry> = {
  image_generation: { type: 'image_generation', label: '✨ Gerar Imagem', color: '#8b5cf6', inputs: [], outputs: [{ label: 'image' }], params: { prompt: '', steps: 25 } },
  load_image: { type: 'load_image', label: 'Carregar Imagem', color: '#3b82f6', inputs: [], outputs: [{ label: 'image' }], params: { path: '' } },
  remove_bg: { type: 'remove_bg', label: 'Remover Fundo', color: '#10b981', inputs: [{ label: 'image' }], outputs: [{ label: 'image' }] },
  resize: { type: 'resize', label: 'Redimensionar', color: '#10b981', inputs: [{ label: 'image' }], outputs: [{ label: 'image' }], params: { width: 512, height: 512 } },
  convert: { type: 'convert', label: 'Converter', color: '#10b981', inputs: [{ label: 'image' }], outputs: [{ label: 'image' }], params: { format: 'webp' } },
  load_video: { type: 'load_video', label: 'Carregar Vídeo', color: '#ef4444', inputs: [], outputs: [{ label: 'video' }], params: { path: '' } },
  extract_audio: { type: 'extract_audio', label: 'Extrair Áudio', color: '#ef4444', inputs: [{ label: 'video' }], outputs: [{ label: 'audio' }], params: { format: 'wav' } },
  frame: { type: 'frame', label: 'Frame (Storyboard)', color: '#ef4444', inputs: [{ label: 'video' }], outputs: [{ label: 'image' }], params: { time: '00:00:01' } },
  load_audio: { type: 'load_audio', label: 'Carregar Áudio', color: '#f59e0b', inputs: [], outputs: [{ label: 'audio' }], params: { path: '' } },
  volume: { type: 'volume', label: 'Volume (Mixer)', color: '#f59e0b', inputs: [{ label: 'audio' }], outputs: [{ label: 'audio' }], params: { volume: '0dB' } },
  convert_audio: { type: 'convert_audio', label: 'Exportar Áudio', color: '#f59e0b', inputs: [{ label: 'audio' }], outputs: [{ label: 'audio' }], params: { format: 'mp3' } },
  load_3d: { type: 'load_3d', label: 'Carregar 3D', color: '#06b6d4', inputs: [], outputs: [{ label: 'model' }], params: { path: '' } },
  probe_3d: { type: 'probe_3d', label: 'Inspecionar 3D', color: '#06b6d4', inputs: [{ label: 'model' }], outputs: [{ label: 'info' }] },
  load_scene: { type: 'load_scene', label: 'Carregar Cena', color: '#22c55e', inputs: [], outputs: [{ label: 'scene' }], params: { path: '' } },
  scene_info: { type: 'scene_info', label: 'Info da Cena', color: '#22c55e', inputs: [{ label: 'scene' }], outputs: [{ label: 'info' }] },
  run_experiment: { type: 'run_experiment', label: 'Experimento', color: '#a855f7', inputs: [], outputs: [{ label: 'result' }], params: { expr: 'constant', value: 0.5 } },
  lab_best: { type: 'lab_best', label: 'Melhor (LAB)', color: '#a855f7', inputs: [], outputs: [{ label: 'best' }], params: { metric: 'result' } },
}

// TIPOS com saída de IMAGEM (para o remove_bg/resize consumirem diffusion).
const IMAGE_OUTPUT_TYPES = ['image_generation', 'load_image', 'remove_bg', 'resize', 'convert', 'frame']

// Fluxos de exemplo — pipelines prontos (§30).
interface FlowTemplate {
  name: string
  desc: string
  steps: { id?: string; type: string; params?: Record<string, unknown> }[]
}

const FLOWS: FlowTemplate[] = [
  {
    name: '✨ Imagem por IA',
    desc: 'diffusion → remove fundo → converte (webp)',
    steps: [
      { id: 'gen', type: 'image_generation', params: { prompt: 'paisagem futurista neon, cyberpunk, arte digital', steps: 25 } },
      { id: 'bg', type: 'remove_bg' },
      { id: 'out', type: 'convert', params: { format: 'webp' } },
    ],
  },
  {
    name: '🎬 Vídeo → Áudio → MP3',
    desc: 'extrai a trilha do vídeo, mixa volume, exporta mp3',
    steps: [
      { id: 'v', type: 'load_video', params: { path: 'intro.mp4' } },
      { id: 'a', type: 'extract_audio', params: { format: 'wav' } },
      { id: 'vol', type: 'volume', params: { volume: '3dB' } },
      { id: 'mp3', type: 'convert_audio', params: { format: 'mp3' } },
    ],
  },
  {
    name: '🎬 Vídeo → Storyboard',
    desc: 'frame do vídeo → redimensiona (still do §8)',
    steps: [
      { id: 'v', type: 'load_video', params: { path: 'intro.mp4' } },
      { id: 'f', type: 'frame', params: { time: '00:00:01' } },
      { id: 'r', type: 'resize', params: { width: 256, height: 256 } },
    ],
  },
  {
    name: '🧊 3D → Cena do Jogo',
    desc: 'inspeciona o modelo 3D e monta a cena do jogo',
    steps: [
      { id: 'm', type: 'load_3d', params: { path: 'scene.obj' } },
      { id: 'p', type: 'probe_3d' },
      { id: 's', type: 'load_scene', params: { path: 'level.json' } },
      { id: 'i', type: 'scene_info' },
    ],
  },
  {
    name: '🔬 Experimento + LAB',
    desc: 'experimento calculado → simulado → melhor (A/B)',
    steps: [
      { id: 'b', type: 'run_experiment', params: { expr: 'constant', value: 0.72 } },
      { id: 's', type: 'run_experiment', params: { expr: 'monte_carlo', samples: 1000, seed: 42 } },
      { id: 'best', type: 'lab_best', params: { metric: 'result' } },
    ],
  },
  {
    name: '🎨 Imagem → OCR',
    desc: 'carrega imagem e extrai texto (tesseract)',
    steps: [
      { id: 'img', type: 'load_image', params: { path: 'doc.png' } },
      { id: 'ocr', type: 'ocr', params: { lang: 'por' } },
    ],
  },
]


let idCounter = 0
function newId(prefix: string): string {
  idCounter++
  return `${prefix}${idCounter}`
}

// CoscaCanvas: o node editor visual.
export function CoscaCanvas() {
  const [nodes, setNodes] = useState<CanvasNode[]>([])
  const [links, setLinks] = useState<Link[]>([])
  const [dragging, setDragging] = useState<{ id: string; dx: number; dy: number } | null>(null)
  const [connecting, setConnecting] = useState<{ from: string; x: number; y: number } | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [run, setRun] = useState<{ executed: number; cached: number; results?: Record<string, unknown>; error?: string } | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState('')
  const canvasRef = useRef<HTMLDivElement>(null)

  // Adiciona um nó da paleta ao centro do canvas.
  const addNode = (type: string) => {
    const entry = PALETTE[type]
    if (!entry) return
    const rect = canvasRef.current?.getBoundingClientRect()
    const id = newId(entry.type.slice(0, 3) + '-')
    const node: CanvasNode = {
      id,
      type: entry.type,
      label: entry.label,
      color: entry.color,
      x: (rect?.width ?? 600) / 2 - 70 + Math.random() * 60,
      y: (rect?.height ?? 400) / 2 - 40 + Math.random() * 60,
      inputs: entry.inputs.map((i, idx) => ({ id: `${id}/in${idx}`, label: i.label, kind: 'input', color: entry.color })),
      outputs: entry.outputs.map((o, idx) => ({ id: `${id}/out${idx}`, label: o.label, kind: 'output', color: entry.color })),
      params: entry.params ? { ...entry.params } : undefined,
    }
    setNodes(prev => [...prev, node])
    setSelected(id)
  }

  // Carrega um fluxo de exemplo: constrói nós com posição + links prontos.
  const loadFlow = (flow: FlowTemplate) => {
    const built: CanvasNode[] = []
    const linkList: Link[] = []
    flow.steps.forEach((step, i) => {
      const entry = PALETTE[step.type]
      if (!entry) return
      const id = step.id || newId(entry.type.slice(0, 3) + '-')
      built.push({
        id,
        type: entry.type,
        label: entry.label,
        color: entry.color,
        x: 40 + i * 200,
        y: 120 + Math.sin(i * 1.2) * 40,
        inputs: entry.inputs.map((inp, idx) => ({ id: `${id}/in${idx}`, label: inp.label, kind: 'input', color: entry.color })),
        outputs: entry.outputs.map((outp, idx) => ({ id: `${id}/out${idx}`, label: outp.label, kind: 'output', color: entry.color })),
        params: step.params ? { ...step.params } : entry.params ? { ...entry.params } : undefined,
      })
      if (i > 0) {
        const prev = built[i - 1]
        const prevOut = prev.outputs[0]?.id
        const curIn = `${id}/in0`
        if (prevOut && curIn) {
          linkList.push({ id: newId('link-'), from: prevOut, to: curIn })
        }
      }
    })
    setNodes(built)
    setLinks(linkList)
    setRun(null)
    setError('')
    setSelected(built[0]?.id ?? null)
  }

  // Inicia o arrasto de um nó.
  const startDrag = (e: React.MouseEvent, id: string) => {
    e.stopPropagation()
    const node = nodes.find(n => n.id === id)
    if (!node) return
    setDragging({ id, dx: e.clientX - node.x, dy: e.clientY - node.y })
    setSelected(id)
  }

  // Move o nó durante o arrasto.
  const onMove = (e: React.MouseEvent) => {
    if (dragging) {
      setNodes(prev => prev.map(n => n.id === dragging.id ? { ...n, x: e.clientX - dragging.dx, y: e.clientY - dragging.dy } : n))
    }
    if (connecting) {
      setConnecting(prev => prev ? { ...prev, x: e.clientX, y: e.clientY } : prev)
    }
  }

  // Termina arrasto/coneção.
  const onUp = () => {
    setDragging(null)
    setConnecting(null)
  }

  // Inicia uma conexão a partir de um socket de saída.
  const startConnect = (e: React.MouseEvent, socketId: string) => {
    e.stopPropagation()
    setConnecting({ from: socketId, x: e.clientX, y: e.clientY })
  }

  // Aceita uma conexão num socket de entrada.
  const acceptConnect = (e: React.MouseEvent, socketId: string) => {
    e.stopPropagation()
    if (!connecting) return
    const [toNodeId] = socketId.split('/')
    const [fromNodeId] = connecting.from.split('/')
    if (toNodeId === fromNodeId) { setConnecting(null); return }
    const id = newId('link-')
    setLinks(prev => [...prev.filter(l => l.to !== socketId), { id, from: connecting.from, to: socketId }])
    setConnecting(null)
  }

  // Converte o canvas para o JSON do nodegraph (§21).
  const toGraphJSON = (): Record<string, unknown> => {
    const gNodes = nodes.map(n => {
      const inputs = links.filter(l => l.to.startsWith(n.id + '/')).map(l => {
        const [srcId] = l.from.split('/')
        return srcId
      })
      const cleaned: Record<string, unknown> = { id: n.id, type: n.type }
      if (n.params && Object.keys(n.params).some(k => String(n.params?.[k]).trim() !== '' && k !== 'prompt')) {
        cleaned.params = n.params
      }
      if (inputs.length > 0) cleaned.inputs = [...new Set(inputs)]
      return cleaned
    })
    return { name: 'cosca-canvas', nodes: gNodes }
  }

  // Executa o grafo no orquestrador cross-product (§30).
  const execute = async () => {
    if (nodes.length === 0 || running) return
    setError('')
    setRun(null)
    setRunning(true)
    try {
      const graph = toGraphJSON()
      const res = await guardedFetch(await apiAsync('/api/engine/x/run'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(graph),
      }).then(r => r.json())
      if (res.error) { setError(res.error); setRun(null); return }
      setRun({ executed: res.executed ?? 0, cached: res.cached_hits ?? 0, results: res.results })
    } catch (e) {
      setError(`falha na execução: ${(e as Error).message}`)
    } finally {
      setRunning(false)
    }
  }

  // Limpa o canvas.
  const clear = () => { setNodes([]); setLinks([]); setRun(null); setSelected(null) }

  // Remove um nó e suas conexões.
  const removeSelected = () => {
    if (!selected) return
    setNodes(prev => prev.filter(n => n.id !== selected))
    setLinks(prev => prev.filter(l => !l.from.startsWith(selected) && !l.to.startsWith(selected)))
    setSelected(null)
  }

  // Propriedades do nó selecionado.
  const selectedNode = nodes.find(n => n.id === selected)

  // Fecha o painel de propriedades com Esc.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { setSelected(null); setConnecting(null) }
      if (e.key === 'Delete' && selected) removeSelected()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [selected, nodes, links])

  return (
    <div className="canvas-wrap">
      {/* Paleta de nós */}
      <div className="canvas-palette">
        <div className="canvas-palette-title">EXEMPLOS</div>
        {FLOWS.map(flow => (
          <button key={flow.name} className="flow-item" onClick={() => loadFlow(flow)} title={flow.desc}>
            <span className="flow-name">{flow.name}</span>
            <span className="flow-desc">{flow.desc}</span>
          </button>
        ))}
        <div className="canvas-palette-title">NÓS</div>
        {Object.values(PALETTE).map(entry => (
          <button key={entry.type} className="palette-item" style={{ borderLeftColor: entry.color }} onClick={() => addNode(entry.type)}>
            <span className="palette-dot" style={{ background: entry.color }} />
            {entry.label}
          </button>
        ))}
      </div>

      {/* Canvas */}
      <div
        ref={canvasRef}
        className="canvas-area"
        onMouseMove={onMove}
        onMouseUp={onUp}
        onMouseLeave={onUp}
      >
        {nodes.length === 0 && (
          <div className="canvas-empty">
            Arraste nós da paleta ao lado e conecte os sockets.
            <br />
            <code>image_generation → remove_bg → convert</code>
          </div>
        )}

        {/* Links (SVG) */}
        <svg className="canvas-links">
          {links.map(link => {
            const [fromId] = link.from.split('/')
            const [toId] = link.to.split('/')
            const fromNode = nodes.find(n => n.id === fromId)
            const toNode = nodes.find(n => n.id === toId)
            const fromSock = fromNode?.outputs.find(s => s.id === link.from)
            const toSock = toNode?.inputs.find(s => s.id === link.to)
            if (!fromNode || !toNode || !fromSock || !toSock) return null
            const x1 = fromNode.x + 170
            const y1 = fromNode.y + 24 + fromNode.outputs.indexOf(fromSock) * 24
            const x2 = toNode.x
            const y2 = toNode.y + 24 + toNode.inputs.indexOf(toSock) * 24
            const mx = (x1 + x2) / 2
            return (
              <path key={link.id} d={`M ${x1} ${y1} C ${mx} ${y1}, ${mx} ${y2}, ${x2} ${y2}`} className="canvas-link" fill="none" />
            )
          })}
          {/* Conexão em andamento */}
          {connecting && (() => {
            const [fromId] = connecting.from.split('/')
            const fromNode = nodes.find(n => n.id === fromId)
            const fromSock = fromNode?.outputs.find(s => s.id === connecting.from)
            if (!fromNode || !fromSock) return null
            const rect = canvasRef.current?.getBoundingClientRect()
            const x1 = fromNode.x + 170
            const y1 = fromNode.y + 24 + fromNode.outputs.indexOf(fromSock) * 24
            const x2 = (connecting.x - (rect?.left ?? 0)) - 200
            const y2 = (connecting.y - (rect?.top ?? 0)) - 80
            const mx = (x1 + x2) / 2
            return <path d={`M ${x1} ${y1} C ${mx} ${y1}, ${mx} ${y2}, ${x2} ${y2}`} className="canvas-link pending" fill="none" />
          })()}
        </svg>

        {/* Nós */}
        {nodes.map(node => (
          <div
            key={node.id}
            className={`canvas-node ${selected === node.id ? 'selected' : ''}`}
            style={{ left: node.x, top: node.y, borderColor: node.color }}
            onMouseDown={e => startDrag(e, node.id)}
            onClick={e => { e.stopPropagation(); setSelected(node.id) }}
          >
            <div className="canvas-node-head" style={{ background: node.color }}>
              <span className="canvas-node-label">{node.label}</span>
              <span className="canvas-node-type">{node.type}</span>
            </div>
            <div className="canvas-node-body">
              {node.inputs.map(sock => (
                <div key={sock.id} className="canvas-socket-row input">
                  <span className="canvas-socket" style={{ background: sock.color }} onMouseDown={e => acceptConnect(e, sock.id)} />
                  <span className="canvas-socket-label">{sock.label}</span>
                </div>
              ))}
              {node.outputs.map(sock => (
                <div key={sock.id} className="canvas-socket-row output">
                  <span className="canvas-socket-label">{sock.label}</span>
                  <span className="canvas-socket" style={{ background: sock.color }} onMouseDown={e => startConnect(e, sock.id)} />
                </div>
              ))}
            </div>
          </div>
        ))}

        {/* Botão de execução flutuante */}
        {nodes.length > 0 && (
          <button className="canvas-run" onClick={execute} disabled={running}>
            {running ? '…rodando' : '▶ Rodar pipeline'}
          </button>
        )}
      </div>

      {/* Painel lateral: propriedades + resultado */}
      <div className="canvas-side">
        {selectedNode && (
          <div className="canvas-inspector">
            <div className="canvas-inspector-title">{selectedNode.label}</div>
            {selectedNode.params && Object.entries(selectedNode.params).map(([k, v]) => (
              <div key={k} className="canvas-param-row">
                <span>{k}</span>
                {typeof v === 'string' && k === 'prompt' ? (
                  <input className="canvas-param-input wide" value={v} onChange={e => updateParam(selectedNode.id, k, e.target.value)} placeholder="prompt…" />
                ) : (
                  <input className="canvas-param-input" value={String(v)} onChange={e => updateParam(selectedNode.id, k, isNaN(Number(e.target.value)) ? e.target.value : Number(e.target.value))} />
                )}
              </div>
            ))}
            <button className="canvas-del" onClick={removeSelected}>Excluir nó (Del)</button>
          </div>
        )}

        {run && !run.error && (
          <div className="canvas-result">
            <div className="canvas-result-title">Executado: {run.executed} · cache: {run.cached}</div>
            {run.results && Object.entries(run.results).map(([id, val]) => (
              <div key={id} className="canvas-result-row">
                <span>{id}</span>
                <code>{fmt(val)}</code>
              </div>
            ))}
          </div>
        )}
        {error && <div className="canvas-result error">{error}</div>}

        <button className="canvas-clear" onClick={clear}>Limpar canvas</button>
      </div>
    </div>
  )

  // Atualiza um param de um nó.
  function updateParam(nodeId: string, key: string, value: unknown) {
    setNodes(prev => prev.map(n => n.id === nodeId ? { ...n, params: { ...n.params, [key]: value } } : n))
  }
}

// fmt formata o resultado (caminho curto ou resumo).
function fmt(v: unknown): string {
  if (typeof v === 'string') return v.split('/').pop() ?? v
  if (typeof v === 'object' && v !== null) return JSON.stringify(v).slice(0, 50)
  return String(v)
}

void IMAGE_OUTPUT_TYPES
