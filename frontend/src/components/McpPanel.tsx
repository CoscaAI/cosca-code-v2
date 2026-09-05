import { useEffect, useState } from 'react'
import { api } from '../api'

// McpPanel: gerenciamento de servidores MCP (spec seção 4). Conecta um servidor
// MCP (comando + args) e suas tools entram no agente automaticamente.
export function McpPanel() {
  const [servers, setServers] = useState<string[]>([])
  const [command, setCommand] = useState('')
  const [args, setArgs] = useState('')
  const [status, setStatus] = useState('')

  useEffect(() => {
    fetch(api('/api/mcp')).then(r => r.json()).then(d => setServers(d.servers ?? [])).catch(() => {})
  }, [])

  const connect = async () => {
    if (!command.trim()) return
    const res = await fetch(api('/api/mcp/connect'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ command: command.trim(), args: args.trim() ? args.trim().split(/\s+/) : [] }),
    }).then(r => r.json())
    if (res.error) { setStatus(`⚠ ${res.error}`); return }
    setServers(prev => [...prev, command.trim()])
    setCommand('')
    setArgs('')
    setStatus(`✓ ${res.command} conectado (tools registradas)`)
  }

  return (
    <aside className="sidebar">
      <div className="sidebar-title">MCP SERVERS</div>
      <div className="settings-note">servidores conectados (tools entram no agente)</div>
      {servers.length === 0 ? (
        <div className="empty">nenhum servidor MCP conectado</div>
      ) : (
        servers.map((s, i) => (
          <div key={i} className="catalog-item connected">
            <span className="catalog-dot">✓</span>
            <span className="catalog-name">{s}</span>
          </div>
        ))
      )}

      <div className="settings-section" style={{ marginTop: 12 }}>Conectar servidor</div>
      <div className="settings-connect">
        <input className="settings-input" placeholder="comando (ex: npx -y @modelcontextprotocol/server-filesystem)" value={command} onChange={e => setCommand(e.target.value)} />
        <input className="settings-input" placeholder="args (ex: /home/cosca)" value={args} onChange={e => setArgs(e.target.value)} />
        <button className="run-tests" onClick={connect} disabled={!command.trim()}>conectar</button>
      </div>
      {status && <div className="settings-status">{status}</div>}
    </aside>
  )
}
