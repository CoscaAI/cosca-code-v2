// TabBar: abas de arquivos abertos (spec seção 32/35). Múltiplos arquivos,
// com ativação e fechamento.
export function TabBar({ files, active, onActivate, onClose }: {
  files: string[]
  active: string | null
  onActivate: (path: string) => void
  onClose: (path: string) => void
}) {
  if (files.length === 0) return null
  return (
    <div className="editor-tabbar">
      {files.map(f => (
        <div
          key={f}
          className={`tab ${active === f ? 'active' : ''}`}
          onClick={() => onActivate(f)}
          title={f}
        >
          <span className="tab-name">{f.split('/').pop()}</span>
          <button
            className="tab-close"
            onClick={e => { e.stopPropagation(); onClose(f) }}
          >
            ✕
          </button>
        </div>
      ))}
    </div>
  )
}
