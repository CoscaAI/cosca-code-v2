import { useEffect, useState } from 'react'
import { apiAsync } from '../api'

// Asset descreve um asset do Cosca Engine (§2 — content-addressable).
interface Asset {
  id: string
  type: string
  size: number
  version: string
  source?: string
  added_at?: string
}

interface AssetList {
  count: number
  assets: Asset[]
}

// AssetPanel: a árvore de assets do projeto (imagem/vídeo/áudio/3D/...).
// Consome GET /api/engine/assets do Cosca Engine integrado (Fase 2.2).
// Cada asset mostra tipo + tamanho + origem — a base do ecossistema criativo.
export function AssetPanel({ onInspect }: { onInspect: (asset: Asset) => void }) {
  const [assets, setAssets] = useState<Asset[]>([])
  const [count, setCount] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    apiAsync('/api/engine/assets')
      .then(url => fetch(url))
      .then(r => r.json())
      .then((d: AssetList) => {
        if (cancelled) return
        setAssets(d.assets ?? [])
        setCount(d.count ?? 0)
      })
      .catch(() => {})
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [])

  const fmtSize = (n: number): string => {
    if (!n) return '0 B'
    if (n < 1024) return `${n} B`
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
    return `${(n / (1024 * 1024)).toFixed(1)} MB`
  }

  return (
    <aside className="sidebar">
      <div className="sidebar-title">ASSETS ({count})</div>
      <div className="asset-list">
        {loading && <div className="empty">carregando…</div>}
        {!loading && assets.length === 0 && (
          <div className="empty">
            Nenhum asset registrado.
            <br />
            <code>cosca asset add</code>
          </div>
        )}
        {assets.map(a => (
          <button
            key={a.id}
            className="asset-item"
            onClick={() => onInspect(a)}
            title={`${a.id}\n${a.source ?? 'sem origem'}`}
          >
            <span className="asset-glyph">{glyphFor(a.type)}</span>
            <span className="asset-meta">
              <span className="asset-type">{a.type}</span>
              <span className="asset-size">{fmtSize(a.size)}</span>
            </span>
            <span className="asset-id">{a.id.slice(0, 8)}</span>
          </button>
        ))}
      </div>
    </aside>
  )
}

// glyphFor devolve um símbolo simples por tipo de asset (doutrina L203:
// sem emojis como ícones — símbolos tipográficos).
function glyphFor(type: string): string {
  switch (type) {
    case 'image': return '▦'
    case 'video': return '▷'
    case 'audio': return '♪'
    case '3d': return '◈'
    case 'model': return '◇'
    case 'document': return '▤'
    case 'script': return '≡'
    default: return '•'
  }
}
