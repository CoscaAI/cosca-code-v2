import { useEffect, useState } from 'react'
import { api } from '../api'

interface FileStatus { path: string; staged: boolean; untracked: boolean; x: string; y: string }
interface GitStatus { branch: string; files: FileStatus[]; error?: string }

// GitPanel: painel de source control (spec seção 7) — branch + arquivos
// alterados + diff inline. Git de primeira classe, não um terminal separado.
export function GitPanel() {
  const [status, setStatus] = useState<GitStatus | null>(null)
  const [diffFile, setDiffFile] = useState<string | null>(null)
  const [diff, setDiff] = useState('')

  useEffect(() => {
    fetch(api('/api/git/status')).then(r => r.json()).then(setStatus).catch(() => {})
  }, [])

  const showDiff = (path: string) => {
    setDiffFile(path)
    fetch(api(`/api/git/diff?path=${encodeURIComponent(path)}`))
      .then(r => r.json())
      .then((d: { unstaged: string }) => setDiff(d.unstaged || '(sem mudanças não-stageadas)'))
      .catch(() => setDiff(''))
  }

  return (
    <aside className="sidebar">
      <div className="sidebar-title">SOURCE CONTROL</div>
      {status?.error ? (
        <div className="empty">{status.error}</div>
      ) : status ? (
        <>
          <div className="git-branch">⑂ {status.branch}</div>
          <div className="git-section">Changes ({status.files.length})</div>
          {status.files.map(f => (
            <div
              key={f.path}
              className={`tree-row file git-file ${diffFile === f.path ? 'active' : ''}`}
              onClick={() => showDiff(f.path)}
            >
              <span className="git-mark">{f.untracked ? 'U' : f.staged ? 'M' : 'M'}</span>
              <span className="name">{f.path}</span>
            </div>
          ))}
          {diffFile && (
            <pre className="git-diff">{diff}</pre>
          )}
        </>
      ) : (
        <div className="empty">carregando…</div>
      )}
    </aside>
  )
}
