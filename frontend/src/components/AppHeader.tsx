import { useEffect, useState } from 'react'
import { apiAsync } from '../api'

// Agents do ecossistema (para a animação de nomes).
const AGENTS = [
  'cosca-ceo', 'cosca-cto', 'cosca-architecture', 'cosca-backend',
  'cosca-frontend', 'cosca-database', 'cosca-security', 'cosca-testing',
  'cosca-devops', 'cosca-documentation', 'cosca-review', 'cosca-semantic-memory',
  'cosca-ai', 'cosca-automation', 'cosca-api', 'cosca-cache', 'cosca-cli',
  'cosca-compliance', 'cosca-context', 'cosca-critic', 'cosca-discovery',
  'cosca-evolution', 'cosca-governance', 'cosca-infrastructure', 'cosca-integrations',
  'cosca-kernel', 'cosca-memory-chief', 'cosca-messaging', 'cosca-migration',
  'cosca-mobile', 'cosca-monitoring', 'cosca-paradigm', 'cosca-performance',
  'cosca-platform', 'cosca-plugin', 'cosca-product', 'cosca-provider', 'cosca-qa',
  'cosca-release', 'cosca-review', 'cosca-runtime', 'cosca-sdk', 'cosca-technical-debt',
  'cosca-uiux', 'cosca-workflow-chief',
]

// ProviderInfo para os dropdowns.
interface ProviderInfo {
  provider: string
  models: string[]
}

interface HeaderProps {
  root: string
  busy: boolean
  provider: string
  model: string
  onProvider: (p: string) => void
  onModel: (m: string) => void
}

// AppHeader: o topo do COSCA — logo, pasta root, agent animado, status,
// dropdowns de modelo/provider e o Forge.
export function AppHeader({ root, busy, provider, model, onProvider, onModel }: HeaderProps) {
  const [catalog, setCatalog] = useState<ProviderInfo[]>([])
  const [agentIdx, setAgentIdx] = useState(0)
  const [showModels, setShowModels] = useState(false)
  const [showProviders, setShowProviders] = useState(false)
  const [showForge, setShowForge] = useState(false)

  // Carrega o catálogo de providers/modelos.
  useEffect(() => {
    let cancelled = false
    apiAsync('/api/ai/models')
      .then(url => fetch(url))
      .then(r => r.json())
      .then((d: { provider?: string; models?: { id?: string; name?: string }[] }) => {
        if (cancelled) return
        const entries: ProviderInfo[] = []
        // /api/ai/models devolve ModelInfo {id,name} — normalizamos para o nome
        // (string) que os dropdowns e o backend esperam no chat.
        const m = d.models ?? []
        if (m.length > 0) {
          entries.push({
            provider: d.provider ?? 'ollama',
            models: m.map(x => x.name || x.id || '').filter(Boolean),
          })
        }
        // Providers locais conhecidos (doutrina: local primeiro) — sem duplicar
        // os que já vieram do backend.
        const known: ProviderInfo[] = [
          { provider: 'ollama', models: ['qwen2.5-coder:14b', 'qwen2.5-coder:14b-128k', 'nomic-embed-text', 'llama3', 'mistral'] },
          { provider: 'diffusion', models: ['sd-1.5 (ROCM)'] },
        ]
        for (const k of known) {
          if (!entries.some(e => e.provider === k.provider)) entries.push(k)
        }
        setCatalog(entries)
      })
      .catch(() => {})
    return () => { cancelled = true }
  }, [])

  // Animação: troca o nome do agent a cada 2s.
  useEffect(() => {
    const t = setInterval(() => setAgentIdx(i => (i + 1) % AGENTS.length), 2000)
    return () => clearInterval(t)
  }, [])

  // Fecha dropdowns ao clicar fora.
  useEffect(() => {
    const onDoc = () => { setShowModels(false); setShowProviders(false); setShowForge(false) }
    document.addEventListener('click', onDoc)
    return () => document.removeEventListener('click', onDoc)
  }, [])

  return (
    <header className="app-header">
      {/* Logo: diamante + cosca */}
      <div className="hdr-logo" title="COSCA">
        <div className="hdr-diamond">◆</div>
        <span className="hdr-name">cosca</span>
      </div>

      {/* Pasta root (sandbox) */}
      <div className="hdr-root" title={`pasta root: ${root}`}>
        <span className="hdr-folder">▸</span>
        <span className="hdr-root-path">{root || 'pasta root'}</span>
      </div>

      {/* Agent animado */}
      <div className="hdr-agent" title="agentes do ecossistema">
        <span className="hdr-agent-label">agent</span>
        <span className="hdr-agent-name">{AGENTS[agentIdx]}</span>
      </div>

      <div className="hdr-spacer" />

      {/* Status: idle (apagado) / ativo (aceso) */}
      <div className={`hdr-status ${busy ? 'busy' : 'idle'}`} title={busy ? 'algo em execução…' : 'ocioso'}>
        <span className="hdr-pulse" />
        <span className="hdr-status-label">{busy ? 'executando' : 'idle'}</span>
      </div>

      {/* Dropdown provider (transparente com seta) */}
      <div className="hdr-drop" onClick={e => { e.stopPropagation(); setShowProviders(!showProviders) }}>
        <span className="hdr-drop-label">{provider}</span>
        <span className="hdr-chevron">▾</span>
        {showProviders && (
          <div className="hdr-drop-menu">
            {catalog.map(p => (
              <button key={p.provider} className="hdr-drop-item" onClick={e => { e.stopPropagation(); onProvider(p.provider); setShowProviders(false) }}>
                {p.provider}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Dropdown modelo (transparente com seta) */}
      <div className="hdr-drop" onClick={e => { e.stopPropagation(); setShowModels(!showModels) }}>
        <span className="hdr-drop-label">{model || 'modelo'}</span>
        <span className="hdr-chevron">▾</span>
        {showModels && (
          <div className="hdr-drop-menu">
            {(catalog.find(p => p.provider === provider)?.models ?? []).map(m => (
              <button key={m} className="hdr-drop-item" onClick={e => { e.stopPropagation(); onModel(m); setShowModels(false) }}>
                {m}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Forge: animação elegante */}
      <div className={`hdr-forge ${busy ? 'active' : ''}`} onClick={e => { e.stopPropagation(); setShowForge(!showForge) }} title="Forge">
        <span className="hdr-forge-gem">◈</span>
        <span className="hdr-forge-label">forge</span>
        {showForge && (
          <div className="hdr-drop-menu forge-menu">
            <div className="forge-menu-title">Forge — motores ativos</div>
            <div className="forge-item">🧠 qwen2.5-coder:14b (local)</div>
            <div className="forge-item">🎨 sd-1.5 (diffusion · ROCm)</div>
            <div className="forge-item">⚙️ ffmpeg + cosca-media</div>
            <div className="forge-item">🔗 websocket · api · eventos</div>
          </div>
        )}
      </div>
    </header>
  )
}
