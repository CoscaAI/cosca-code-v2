// TrustCenter.tsx — o TRUST CENTER do Cosca (pilar 8 da visão do professor):
// uma interface que mostra o que a IA PODE fazer no ambiente. SOMENTE LEITURA
// — usa os endpoints de leitura já existentes (/api/autonomy, /api/memory,
// /api/skills, /api/agent/run com listagem) — ZERO mudança no backend.
//
// A régua da casa: expor o que já existe, nunca arriscar o que funciona.
import { useEffect, useState } from 'react'
import { apiAsync } from '../api'

// ── tipos (espelham o backend, sem acoplar) ────────────────────────────────

interface AutonomyState {
  level: string
}

interface MemoryEntry {
  key: string
  value: string
  source: string
  timestamp: string
  confidence: number
  version: number
}

interface Skill {
  name: string
  description: string
  tools: string[]
}

// ── helpers de status (cor por estado) ─────────────────────────────────────

function confPct(c: number): string {
  return `${Math.round((c ?? 0) * 100)}%`
}

// ── componente ─────────────────────────────────────────────────────────────

export function TrustCenter() {
  const [autonomy, setAutonomy] = useState<string>('desconhecido')
  const [memory, setMemory] = useState<MemoryEntry[]>([])
  const [skills, setSkills] = useState<Skill[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let alive = true
    ;(async () => {
      try {
        const [autoResp, memResp, skillResp] = await Promise.all([
          fetch(await apiAsync('/api/autonomy')),
          fetch(await apiAsync('/api/memory')),
          fetch(await apiAsync('/api/skills')),
        ])
        if (autoResp.ok) {
          const a = (await autoResp.json()) as AutonomyState
          if (alive) setAutonomy(a.level ?? 'desconhecido')
        }
        if (memResp.ok) {
          const m = (await memResp.json()) as { entries: MemoryEntry[] }
          if (alive) setMemory(m.entries ?? [])
        }
        if (skillResp.ok) {
          const s = (await skillResp.json()) as { skills: Skill[] }
          if (alive) setSkills(s.skills ?? [])
        }
      } catch (e) {
        if (alive) setError(`backend indisponível: ${(e as Error).message}`)
      } finally {
        if (alive) setLoading(false)
      }
    })()
    return () => { alive = false }
  }, [])

  if (loading) return <div className="dock-panel trust-panel"><p className="empty">carregando Trust Center…</p></div>
  if (error) return <div className="dock-panel trust-panel"><p className="empty">{error}</p></div>

  return (
    <div className="dock-panel trust-panel">
      {/* 1. Autonomia — o nível de permissão atual */}
      <section className="trust-section">
        <h3 className="trust-title">Autonomia</h3>
        <div className="trust-autonomy">
          <span className={`trust-level ${autonomy === 'autonomous' ? 'st-bad' : 'st-ok'}`}>
            {autonomy}
          </span>
          <span className="trust-hint">
            {autonomy === 'autonomous'
              ? 'a IA executa sem pedir aprovação'
              : 'a IA pede aprovação em ações sensíveis'}
          </span>
        </div>
      </section>

      {/* 2. Agentes — o "time" da IA (lista de leitura dedicada na revisão) */}
      <section className="trust-section">
        <h3 className="trust-title">Agentes</h3>
        <p className="empty">
          listagem de agentes via endpoint de leitura dedicado (na revisão — sem
          risco ao backend; a execução /api/agent/run não é chamada aqui)
        </p>
      </section>

      {/* 3. Memória — o que a IA aprendeu */}
      <section className="trust-section">
        <h3 className="trust-title">Memória ({memory.length})</h3>
        {memory.length === 0 ? (
          <p className="empty">sem memórias registradas</p>
        ) : (
          <ul className="trust-memory">
            {memory.slice(0, 8).map((m) => (
              <li key={m.key}>
                <span className="mem-key">{m.key}</span>
                <span className="mem-val">{m.value}</span>
                <span className={`mem-conf ${m.confidence >= 0.7 ? 'st-ok' : 'st-idle'}`}>
                  {confPct(m.confidence)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* 4. Skills — as capacidades instaladas */}
      <section className="trust-section">
        <h3 className="trust-title">Skills ({skills.length})</h3>
        <ul className="trust-skills">
          {skills.slice(0, 10).map((s) => (
            <li key={s.name} className="trust-skill">
              <span className="sk-name">{s.name}</span>
              <span className="sk-desc">{s.description}</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}
