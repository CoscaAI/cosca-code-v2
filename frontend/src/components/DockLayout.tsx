// DockLayout.tsx — Fase 1 do Cosca Dynamic Workbench (a visão do professor):
// o workbench vira um LAYOUT EM ÁRVORE com docking real. Cada painel
// (Explorer, Editor, Terminal, AI, Output, Inspector...) é uma view que pode
// ser ARRASTADA e encaixada em qualquer região (esquerda/direita/cima/baixo/
// tab) — o padrão dos IDEs profissionais. Em cima disso, as fases 2-4
// (floating, layouts salvos, IA compõe o workspace) serão construídas.
//
// O layout pode ser serializado (o Dockview gera o JSON) — base para salvar
// presets (Development/Debug/Security) na Fase 3.
import { useEffect, useRef, useState } from 'react'
import {
  DockviewReact,
  DockviewReadyEvent,
} from 'dockview-react'
import { Explorer } from './Explorer'
import { Editor } from './Editor'
import { Terminal } from './Terminal'
import { OutputPanel } from './OutputPanel'
import { AIPanel } from './AIPanel'
import { Inspector } from './Inspector'
import { GitPanel } from './GitPanel'
import { ProblemsPanel } from './ProblemsPanel'
import { SettingsPanel } from './SettingsPanel'
import { TrustCenter } from './TrustCenter'
import { AIControlRoom } from './AIControlRoom'
import { ProjectMap } from './ProjectMap'
import type { Event, FileNode, ProjectInfo } from '../types'

// Props que o DockLayout recebe do App — os mesmos dados que os painéis
// existentes consomem (nenhum componente muda; só o container vira dock).
interface DockLayoutProps {
  tree: FileNode | null
  activeFile: string | null
  openFiles: string[]
  events: Event[]
  info: ProjectInfo | null
  provider: string
  model: string
  inspectedAsset: { id: string; type: string; size: number; version: string; source?: string; added_at?: string } | null
  onOpenFile: (path: string) => void
  onActivateFile: (path: string) => void
  onCloseFile: (path: string) => void
  onInspectAsset: (a: { id: string; type: string; size: number; version: string; source?: string; added_at?: string } | null) => void
}

// Undo/Redo de LAYOUT (mandamento 6 da doutrina): mover painel, fechar painel,
// alterar layout — tudo desfazível com Ctrl+Z / Ctrl+Shift+Z.
// Usa o toJSON do Dockview para capturar o estado e fromJSON para restaurar.
interface DockApiLike {
  toJSON: () => unknown
  fromJSON: (s: unknown) => void
  onDidMutateLayout?: (fn: () => void) => void
  onDidLayoutFromJSON?: (fn: () => void) => void
  // addPanel é a API de criação de painéis (a parte de setup).
  // O runtime aceita o panel retornado OU o id como referencePanel.
  addPanel: (p: {
    id: string
    component: string
    title: string
    position?: {
      referencePanel?: string | { id: string }
      direction?: 'left' | 'right' | 'above' | 'below' | 'within'
    }
  }) => { id: string }
}

export function DockLayout(props: DockLayoutProps) {
  const [ready, setReady] = useState(false)
  const apiRef = useRef<DockApiLike | null>(null)

  // Pilhas de undo/redo de layout (mandamento 6).
  const undoStack = useRef<unknown[]>([])
  const redoStack = useRef<unknown[]>([])
  const applyingRef = useRef(false) // evita capturar a própria restauração

  // pushLayout captura o estado atual para o undo (após uma mutação real).
  const pushLayout = () => {
    const api = apiRef.current
    if (!api || applyingRef.current) return
    try {
      undoStack.current.push(api.toJSON())
      if (undoStack.current.length > 30) undoStack.current.shift() // limite
      redoStack.current = [] // nova ação limpa o redo
    } catch {
      /* serialização falhou — segue sem undo deste estado */
    }
  }

  // applyLayout restaura um estado serializado.
  const applyLayout = (s: unknown) => {
    const api = apiRef.current
    if (!api) return
    applyingRef.current = true
    try {
      api.fromJSON(s)
    } finally {
      applyingRef.current = false
    }
  }

  const undoLayout = () => {
    const current = undoStack.current.pop()
    if (current === undefined) return
    const api = apiRef.current
    if (!api) return
    redoStack.current.push(api.toJSON())
    applyLayout(current)
  }

  const redoLayout = () => {
    const next = redoStack.current.pop()
    if (next === undefined) return
    const api = apiRef.current
    if (!api) return
    undoStack.current.push(api.toJSON())
    applyLayout(next)
  }

  // Ctrl+Z desfaz layout · Ctrl+Shift+Z refaz.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.ctrlKey || e.metaKey)) return
      const k = e.key.toLowerCase()
      if (k === 'z') {
        e.preventDefault()
        if (e.shiftKey) redoLayout()
        else undoLayout()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  // Registra todas as views do workbench quando o dock está pronto. O layout
  // inicial: Explorer à esquerda, Editor no centro, AI à direita, Terminal
  // embaixo — mas tudo ARRASTÁVEL a partir daí.
  const onReady = (event: DockviewReadyEvent) => {
    const api = event.api as unknown as DockApiLike
    apiRef.current = api
    // Toda mutação real de layout vira undo point.
    api.onDidMutateLayout?.(() => {
      // pequeno defer para o estado estar consistente
      setTimeout(() => pushLayout(), 0)
    })

    // Editor (centro) — grupo principal com abas.
    const editor = api.addPanel({
      id: 'editor',
      component: 'editor',
      title: 'Editor',
      position: { referencePanel: 'explorer', direction: 'right' },
    })

    // Explorer (esquerda).
    api.addPanel({
      id: 'explorer',
      component: 'explorer',
      title: 'Explorer',
      position: { referencePanel: editor, direction: 'left' },
    })

    // AI Panel (direita).
    api.addPanel({
      id: 'ai',
      component: 'ai',
      title: 'AI',
      position: { referencePanel: editor, direction: 'right' },
    })

    // Trust Center (aba ao lado da AI — o pilar 8 da visão do professor).
    api.addPanel({
      id: 'trust',
      component: 'trust',
      title: '🛡 Trust',
      position: { referencePanel: 'ai', direction: 'within' },
    })

    // AI Control Room (pilar 9 — modelos e estado, somente leitura).
    api.addPanel({
      id: 'control',
      component: 'control',
      title: '📊 Control',
      position: { referencePanel: 'trust', direction: 'within' },
    })

    // Project Map (pilar 6 — símbolos e dependentes do projeto).
    api.addPanel({
      id: 'map',
      component: 'map',
      title: '🗺️ Map',
      position: { referencePanel: 'control', direction: 'within' },
    })

    // Terminal + Output + Git + Problems (embaixo, como abas do grupo inferior).
    api.addPanel({
      id: 'terminal',
      component: 'terminal',
      title: 'Terminal',
      position: { referencePanel: editor, direction: 'below' },
    })
    api.addPanel({
      id: 'output',
      component: 'output',
      title: 'Output',
      position: { referencePanel: 'terminal', direction: 'within' },
    })
    api.addPanel({
      id: 'git',
      component: 'git',
      title: 'Git',
      position: { referencePanel: 'terminal', direction: 'within' },
    })
    api.addPanel({
      id: 'problems',
      component: 'problems',
      title: 'Problems',
      position: { referencePanel: 'terminal', direction: 'within' },
    })

    setReady(true)
  }

  // Renderiza cada view com as props que o componente original espera.
  // O Dockview passa { containerApi, api, params } via IDockviewPanelProps.
  const components = {
    editor: () => (
      <div className="dock-panel dock-editor">
        <Editor file={props.activeFile} info={props.info} />
      </div>
    ),
    explorer: () => (
      <div className="dock-panel">
        <Explorer tree={props.tree} activeFile={props.activeFile} onOpen={props.onOpenFile} />
      </div>
    ),
    ai: () => (
      <div className="dock-panel">
        <AIPanel provider={props.provider} model={props.model} />
      </div>
    ),
    terminal: () => (
      <div className="dock-panel">
        <Terminal open onClose={() => {}} />
      </div>
    ),
    output: () => (
      <div className="dock-panel">
        <OutputPanel events={props.events} />
      </div>
    ),
    git: () => (
      <div className="dock-panel">
        <GitPanel />
      </div>
    ),
    problems: () => (
      <div className="dock-panel">
        <ProblemsPanel file={props.activeFile} />
      </div>
    ),
    inspector: () => (
      <div className="dock-panel">
        <Inspector asset={props.inspectedAsset} file={props.activeFile} />
      </div>
    ),
    settings: () => (
      <div className="dock-panel">
        <SettingsPanel />
      </div>
    ),
    trust: () => (
      <div className="dock-panel">
        <TrustCenter />
      </div>
    ),
    control: () => (
      <div className="dock-panel">
        <AIControlRoom />
      </div>
    ),
    map: () => (
      <div className="dock-panel">
        <ProjectMap />
      </div>
    ),
  }

  return (
    <div className="dock-layout">
      <DockviewReact
        onReady={onReady}
        components={components}
        className="dockview-theme-dark"
      />
      {/* debug: mostra que o dock está pronto */}
      {ready && <span className="dock-ready" title="workbench arrastável ativo" />}
    </div>
  )
}
