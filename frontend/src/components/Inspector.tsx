import { useEffect, useState } from 'react'
import { api, apiAsync } from '../api'

// Asset descreve um asset (mesma shape do AssetPanel).
interface Asset {
  id: string
  type: string
  size: number
  version: string
  source?: string
  added_at?: string
}

interface ProvenanceData {
  claims?: unknown[]
  generations?: unknown[]
  licenses?: unknown[]
}

// Inspector: propriedades da seleção (asset ou arquivo). Para um asset mostra
// o content hash, tipo, versão, origem e a provenance (§32-35) se houver.
// Para um arquivo mostra o caminho e a linguagem inferida.
export function Inspector({ asset, file }: { asset: Asset | null; file: string | null }) {
  const [provenance, setProvenance] = useState<ProvenanceData | null>(null)

  useEffect(() => {
    if (!asset) { setProvenance(null); return }
    let cancelled = false
    apiAsync('/api/engine/provenance')
      .then(url => fetch(url))
      .then(r => r.json())
      .then((d: ProvenanceData) => { if (!cancelled) setProvenance(d) })
      .catch(() => {})
    return () => { cancelled = true }
  }, [asset])

  if (!asset && !file) {
    return (
      <aside className="sidebar">
        <div className="sidebar-title">INSPECTOR</div>
        <div className="empty">Selecione um asset ou arquivo</div>
      </aside>
    )
  }

  const genCount = provenance?.generations?.length ?? 0

  return (
    <aside className="sidebar">
      <div className="sidebar-title">INSPECTOR</div>
      {asset && (
        <div className="inspector-body">
          <div className="inspector-title">{asset.type}</div>
          <div className="inspector-field">
            <span>Hash</span><code>{asset.id.slice(0, 16)}…</code>
          </div>
          <div className="inspector-field">
            <span>Tipo</span><code>{asset.type}</code>
          </div>
          <div className="inspector-field">
            <span>Versão</span><code>{asset.version}</code>
          </div>
          <div className="inspector-field">
            <span>Tamanho</span><code>{asset.size} B</code>
          </div>
          {asset.source && (
            <div className="inspector-field">
              <span>Origem</span><code>{asset.source}</code>
            </div>
          )}
          {asset.added_at && (
            <div className="inspector-field">
              <span>Adicionado</span><code>{asset.added_at}</code>
            </div>
          )}
          <div className="inspector-section">Provenance (§33)</div>
          <div className="inspector-field">
            <span>Gerações</span><code>{genCount}</code>
          </div>
          {asset.id && <GenerationLine assetID={asset.id} />}
        </div>
      )}
      {!asset && file && (
        <div className="inspector-body">
          <div className="inspector-title">Arquivo</div>
          <div className="inspector-field">
            <span>Caminho</span><code>{file}</code>
          </div>
          <div className="inspector-field">
            <span>Editor</span><code>CodeMirror</code>
          </div>
        </div>
      )}
    </aside>
  )
}

// GenerationLine mostra a geração de provenance do asset (se existir).
function GenerationLine({ assetID }: { assetID: string }) {
  const [found, setFound] = useState<{ model?: string; seed?: number; prompt?: string } | null>(null)
  useEffect(() => {
    let cancelled = false
    apiAsync('/api/engine/provenance')
      .then(url => fetch(url))
      .then(r => r.json())
      .then((d: ProvenanceData) => {
        if (cancelled || !d.generations) return
        const g = (d.generations as Array<{ asset_id: string; model?: string; seed?: number; prompt?: string }>)
          .find(x => x.asset_id === assetID)
        if (g) setFound(g)
      })
      .catch(() => {})
    return () => { cancelled = true }
  }, [assetID])

  if (!found) return null
  return (
    <>
      <div className="inspector-field">
        <span>Modelo</span><code>{found.model}</code>
      </div>
      {found.seed !== undefined && (
        <div className="inspector-field">
          <span>Seed</span><code>{found.seed}</code>
        </div>
      )}
      {found.prompt && (
        <div className="inspector-field">
          <span>Prompt</span><code>{found.prompt.slice(0, 40)}</code>
        </div>
      )}
    </>
  )
}

// Referência ao api síncrono para o padrão dos outros painéis.
void api
