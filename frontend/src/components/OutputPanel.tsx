import type { Event } from '../types'

// OutputPanel: log de eventos do barramento (observabilidade, spec seção 71).
// Mostra o fluxo vivo de eventos — o "pulso" do ambiente.
export function OutputPanel({ events }: { events: Event[] }) {
  return (
    <div className="terminal-panel">
      <div className="terminal-header">
        <span>OUTPUT — eventos do barramento</span>
      </div>
      <div className="terminal-output output-log">
        {events.length === 0 ? (
          <div className="empty">aguardando eventos…</div>
        ) : (
          events.map((ev, i) => (
            <div key={i} className="output-line">
              <span className="t">{new Date(ev.Timestamp).toLocaleTimeString('pt-BR', { hour12: false })}</span>
              <span className="src">{ev.Type}</span>
              <span className="msg">{ev.Source}</span>
              {ev.Payload ? <span className="payload">{typeof ev.Payload === 'string' ? ev.Payload : ''}</span> : null}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
