import { useEffect, useState } from 'react'
import { CoscaCanvas } from './components/CoscaCanvas'
import { AppHeader } from './components/AppHeader'
import { AppFooter } from './components/AppFooter'
import { CommandPalette, type Command } from './components/CommandPalette'
import { DockLayout } from './components/DockLayout'
import { api, apiAsync } from './api'
import type { Event, FileNode, ProjectInfo } from './types'

// Asset descreve um asset do Cosca Engine (§2).
interface Asset {
  id: string
  type: string
  size: number
  version: string
  source?: string
  added_at?: string
}

export default function App() {
  const [info, setInfo] = useState<ProjectInfo | null>(null)
  const [tree, setTree] = useState<FileNode | null>(null)
  const [openFiles, setOpenFiles] = useState<string[]>([])
  const [activeFile, setActiveFile] = useState<string | null>(null)
  const [events, setEvents] = useState<Event[]>([])
  const [sidebar, setSidebar] = useState<'canvas' | 'dock'>('dock')
  const [panel, setPanel] = useState<'terminal' | 'output' | null>(null)
  const [paletteOpen, setPaletteOpen] = useState(false)
  const [inspectedAsset, setInspectedAsset] = useState<Asset | null>(null)
  const [busy, setBusy] = useState(false)
  const [provider, setProvider] = useState('ollama')
  const [model, setModel] = useState('qwen2.5-coder:14b')

  // Command palette (Ctrl/Cmd+Shift+P).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === 'p') {
        e.preventDefault()
        setPaletteOpen(prev => !prev)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  const commands: Command[] = [
    { id: 'workbench', title: '▦ Workbench (drag & drop)', hint: 'layout em árvore com docking', run: () => setSidebar('dock') },
    { id: 'canvas', title: '◈ Canvas visual', hint: 'tela cheia', run: () => setSidebar('canvas') },
    { id: 'terminal', title: '❯ Terminal', hint: 'shell', run: () => setPanel('terminal') },
    { id: 'output', title: '≡ Output', hint: 'eventos', run: () => setPanel('output') },
  ]

  // Abre um arquivo (adiciona à lista de abas e ativa).
  const openFile = (path: string) => {
    setOpenFiles(prev => prev.includes(path) ? prev : [...prev, path])
    setActiveFile(path)
  }

  // Fecha uma aba (ativa a vizinha se a ativa foi fechada).
  const closeFile = (path: string) => {
    setOpenFiles(prev => {
      const next = prev.filter(f => f !== path)
      if (activeFile === path) {
        setActiveFile(next.length ? next[next.length - 1] : null)
      }
      return next
    })
  }

  // Carrega info + tree do backend (port discovery — Fase 2.3).
  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const [infoRes, treeRes] = await Promise.all([
          fetch(await apiAsync('/api/info')),
          fetch(await apiAsync('/api/tree')),
        ])
        if (cancelled) return
        infoRes.json().then(setInfo).catch(() => {})
        treeRes.json().then(setTree).catch(() => {})
      } catch { /* backend indisponível — a UI continua e mostra vazio */ }
    }
    load()
    return () => { cancelled = true }
  }, [])

  // Stream de eventos (observabilidade ao vivo).
  useEffect(() => {
    let es: EventSource | null = null
    async function connect() {
      es = new EventSource(await apiAsync('/api/events'))
      es.onmessage = (msg) => {
        const ev: Event = JSON.parse(msg.data)
        setEvents(prev => [ev, ...prev].slice(0, 100))
        // Pulso de atividade: agent/tool/graph em execução acende o status.
        if (ev.Type?.includes('started') || ev.Type?.includes('executed') || ev.Type?.includes('tool_called')) {
          setBusy(true)
          setTimeout(() => setBusy(false), 2500)
        }
        // Se um arquivo mudou/foi criado, re-carrega a árvore.
        if (ev.Type === 'file.changed' || ev.Type === 'file.created' || ev.Type === 'file.deleted') {
          fetch(api('/api/tree')).then(r => r.json()).then(setTree).catch(() => {})
        }
      }
    }
    connect()
    return () => { es?.close() }
  }, [])

  return (
    <div className="workbench">
      {/* Header enterprise: logo + root + agent animado + status + dropdowns + forge */}
      <AppHeader
        root={info?.root ?? ''}
        busy={busy}
        provider={provider}
        model={model}
        onProvider={setProvider}
        onModel={setModel}
      />

      <div className="wb-body">
        {/* Fase 1 — Cosca Dynamic Workbench: layout em árvore com docking.
            Todos os painéis são arrastáveis e encaixam em qualquer região.
            O canvas visual continua disponível (modo tela cheia). */}
        {sidebar === 'canvas' ? (
          <CoscaCanvas />
        ) : (
          <DockLayout
            tree={tree}
            activeFile={activeFile}
            openFiles={openFiles}
            events={events}
            info={info}
            provider={provider}
            model={model}
            inspectedAsset={inspectedAsset}
            onOpenFile={openFile}
            onActivateFile={setActiveFile}
            onCloseFile={closeFile}
            onInspectAsset={setInspectedAsset}
          />
        )}

        {paletteOpen && <CommandPalette commands={commands} model={model} onClose={() => setPaletteOpen(false)} />}
      </div>

      {/* Footer: estatísticas ao vivo */}
      <AppFooter
        panel={panel}
        provider={provider}
        model={model}
        onTogglePanel={p => setPanel(panel === p ? null : p)}
      />
    </div>
  )
}
