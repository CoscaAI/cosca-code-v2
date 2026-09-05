import { useEffect, useState } from 'react'
import { api } from '../api'

interface CatalogEntry {
  id: string
  name: string
  base_url: string
  kind: string
  models: string[]
  needs_key: boolean
  local: boolean
}

// SettingsPanel: configurações (spec seção 8/49) — catálogo de providers
// (models.dev), conectar com API key, modelo e nível de autonomia.
export function SettingsPanel() {
  const [catalog, setCatalog] = useState<CatalogEntry[]>([])
  const [connected, setConnected] = useState<string[]>([])
  const [selectedId, setSelectedId] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [models, setModels] = useState<string[]>([])
  const [autonomy, setAutonomy] = useState('')
  const [status, setStatus] = useState('')

  useEffect(() => {
    fetch(api('/api/catalog')).then(r => r.json()).then(d => {
      setCatalog(d.catalog ?? [])
      setConnected(d.connected ?? [])
    }).catch(() => {})
    fetch(api('/api/autonomy')).then(r => r.json()).then(d => setAutonomy(d.level ?? '')).catch(() => {})
  }, [])

  const selectProvider = (e: CatalogEntry) => {
    setSelectedId(e.id)
    setModels(e.models)
    setApiKey('')
  }

  const connect = async () => {
    if (!selectedId) return
    const res = await fetch(api('/api/provider/connect'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: selectedId, api_key: apiKey }),
    }).then(r => r.json())
    if (res.error) { setStatus(`⚠ ${res.error}`); return }
    setConnected(prev => prev.includes(selectedId) ? prev : [...prev, selectedId])
    setStatus(`✓ conectado a ${selectedId}`)
  }

  const setAutonomyLevel = async (level: string) => {
    setAutonomy(level)
    await fetch(api('/api/autonomy'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ level }),
    })
  }

  return (
    <aside className="sidebar settings-panel">
      <div className="sidebar-title">CONFIGURAÇÕES</div>

      <div className="settings-section">Providers</div>
      <div className="settings-note">catálogo (models.dev) — clique para selecionar</div>
      <div className="catalog-list">
        {catalog.map(e => (
          <div
            key={e.id}
            className={`catalog-item ${selectedId === e.id ? 'selected' : ''} ${connected.includes(e.id) ? 'connected' : ''}`}
            onClick={() => selectProvider(e)}
          >
            <span className="catalog-name">{e.name}</span>
            <span className="catalog-meta">{e.models.length} modelos · {e.local ? 'local' : e.kind}</span>
            {connected.includes(e.id) && <span className="catalog-dot">✓</span>}
          </div>
        ))}
      </div>

      {selectedId && (
        <div className="settings-connect">
          {catalog.find(e => e.id === selectedId)?.needs_key && (
            <input
              className="settings-input"
              type="password"
              placeholder="API key"
              value={apiKey}
              onChange={e => setApiKey(e.target.value)}
            />
          )}
          <button className="run-tests" onClick={connect}>conectar</button>
          {models.length > 0 && (
            <select className="model-select" defaultValue={models[0]}>
              {models.map(m => <option key={m} value={m}>{m}</option>)}
            </select>
          )}
        </div>
      )}
      {status && <div className="settings-status">{status}</div>}

      <div className="settings-section" style={{ marginTop: 16 }}>Autonomia</div>
      <div className="autonomy-levels">
        {['read_only', 'assisted', 'approval', 'autonomous', 'full'].map(level => (
          <button
            key={level}
            className={`autonomy-btn ${autonomy === level ? 'active' : ''}`}
            onClick={() => setAutonomyLevel(level)}
          >
            {level}
          </button>
        ))}
      </div>
    </aside>
  )
}
