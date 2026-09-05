import { useState } from 'react'
import { api } from '../api'

interface Match { path: string; line: number; column: number; content: string }
interface SearchResult { matches: Match[]; files: number; total: number; error?: string }

// SearchPanel: busca text/regex no workspace (spec seção 16). Ao clicar num
// resultado, abre o arquivo no editor.
export function SearchPanel({ onOpen }: { onOpen: (path: string) => void }) {
  const [query, setQuery] = useState('')
  const [regex, setRegex] = useState(false)
  const [result, setResult] = useState<SearchResult | null>(null)

  const run = () => {
    if (!query) return
    const params = new URLSearchParams({ q: query })
    if (regex) params.set('regex', '1')
    params.set('max', '200')
    fetch(api(`/api/search?${params}`))
      .then(r => r.json())
      .then(setResult)
      .catch(() => setResult(null))
  }

  return (
    <aside className="sidebar">
      <div className="sidebar-title">SEARCH</div>
      <div className="search-bar">
        <input
          className="search-input"
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') run() }}
          placeholder="buscar no projeto…"
          autoFocus
        />
        <label className="search-regex">
          <input type="checkbox" checked={regex} onChange={e => setRegex(e.target.checked)} /> .*
        </label>
      </div>

      {result?.error ? (
        <div className="empty">{result.error}</div>
      ) : result ? (
        <>
          <div className="git-section">{result.total} resultados em {result.files} arquivos</div>
          {result.matches.map((m, i) => (
            <div key={i} className="search-row" onClick={() => onOpen(m.path)}>
              <span className="search-file">{m.path}:{m.line}</span>
              <span className="search-content">{m.content}</span>
            </div>
          ))}
        </>
      ) : (
        <div className="empty">digite e pressione Enter</div>
      )}
    </aside>
  )
}
