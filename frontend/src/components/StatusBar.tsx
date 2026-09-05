import type { Event, ProjectInfo } from '../types'

// StatusBar: linha de estado discreta (spec seção 39 — métricas subordinadas ao
// editor, nunca um dashboard corporativo).
export function StatusBar({ info, events, panel, onToggleTerminal, onToggleOutput }: {
  info: ProjectInfo | null
  events: Event[]
  panel: string | null
  onToggleTerminal: () => void
  onToggleOutput: () => void
}) {
  const lastEvent = events[0]

  return (
    <footer className="status-bar">
      <span className="sb-item">⎈ {info?.root ?? '—'}</span>
      <span className="sb-item">{info?.language ? `◉ ${info.language}` : ''}</span>
      {info?.framework ? <span className="sb-item">⬢ {info.framework}</span> : null}
      <span className="sb-spacer" />
      <button className={`sb-item sb-btn ${panel === 'terminal' ? 'sb-on' : ''}`} onClick={onToggleTerminal}>❯ terminal</button>
      <button className={`sb-item sb-btn ${panel === 'output' ? 'sb-on' : ''}`} onClick={onToggleOutput}>≡ output</button>
      {lastEvent ? (
        <span className="sb-item dim" title="último evento do barramento">
          {lastEvent.Type} · {lastEvent.Source}
        </span>
      ) : null}
    </footer>
  )
}
