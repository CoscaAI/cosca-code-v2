// AIControlRoom.tsx — o AI CONTROL ROOM (pilar 9 do professor): uma tela para
// administrar a IA do Cosca. SOMENTE LEITURA — usa os endpoints reais que
// existem (/api/ai/models, /api/memory, /api/skills, /api/autonomy,
// /api/git/status). O que o backend NÃO expõe (agents/usage/tokens) aparece
// como "not available" — NUNCA é mockado (P14).
import { useEffect, useState } from 'react'
import { apiAsync } from '../api'

interface ModelInfo {
  id: string
  name: string
}

interface MemoryEntry {
  key: string
  value: string
  source: string
  confidence: number
}

interface Skill {
  name: string
  description: string
}

interface GitStatus {
  branch: string
  files: Record<string, string>
}

interface ControlData {
  provider: string | null
  models: ModelInfo[]
  memoryCount: number
  skills: Skill[]
  autonomy: string | null
  git: GitStatus | null
  unavailable: string[] // recursos que o backend não expõe (honestidade P14)
}

export function AIControlRoom() {
  const [data, setData] = useState<ControlData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let alive = true
    ;(async () => {
      const d: ControlData = {
        provider: null,
        models: [],
        memoryCount: 0,
        skills: [],
        autonomy: null,
        git: null,
        unavailable: [],
      }
      try {
        // Modelos (real).
        const m = await fetch(await apiAsync('/api/ai/models'))
        if (m.ok) {
          const j = (await m.json()) as { provider?: string; models?: ModelInfo[] }
          d.provider = j.provider ?? null
          d.models = j.models ?? []
        } else {
          d.unavailable.push('models')
        }

        // Memória (real).
        const mem = await fetch(await apiAsync('/api/memory'))
        if (mem.ok) {
          const j = (await mem.json()) as { entries?: MemoryEntry[] }
          d.memoryCount = j.entries?.length ?? 0
        } else {
          d.unavailable.push('memory')
        }

        // Skills (real).
        const sk = await fetch(await apiAsync('/api/skills'))
        if (sk.ok) {
          const j = (await sk.json()) as { skills?: Skill[] }
          d.skills = j.skills ?? []
        } else {
          d.unavailable.push('skills')
        }

        // Autonomia (real).
        const au = await fetch(await apiAsync('/api/autonomy'))
        if (au.ok) {
          const j = (await au.json()) as { level?: string }
          d.autonomy = j.level ?? null
        } else {
          d.unavailable.push('autonomy')
        }

        // Git (real).
        const g = await fetch(await apiAsync('/api/git/status'))
        if (g.ok) {
          const j = (await g.json()) as GitStatus
          d.git = j
        } else {
          d.unavailable.push('git')
        }

        // Recursos que o professor descreveu mas o backend NÃO expõe em
        // leitura — honestidade P14: não inventar.
        d.unavailable.push('agents:status', 'tokens:usage', 'context:indexed')

        if (alive) setData(d)
      } catch (e) {
        if (alive) setError(`backend indisponível: ${(e as Error).message}`)
      } finally {
        if (alive) setLoading(false)
      }
    })()
    return () => { alive = false }
  }, [])

  if (loading) return <div className="dock-panel control-panel"><p className="empty">carregando Control Room…</p></div>
  if (error || !data) return <div className="dock-panel control-panel"><p className="empty">{error ?? 'sem dados'}</p></div>

  const changed = data.git ? Object.keys(data.git.files).length : 0

  return (
    <div className="dock-panel control-panel">
      {/* Modelos */}
      <section className="trust-section">
        <h3 className="trust-title">Modelos · {data.provider ?? 'n/d'}</h3>
        <ul className="trust-skills">
          {data.models.length === 0 ? (
            <li className="trust-skill"><span className="sk-desc">indisponível</span></li>
          ) : (
            data.models.map((m) => (
              <li key={m.id} className="trust-skill">
                <span className="sk-name">{m.name}</span>
              </li>
            ))
          )}
        </ul>
      </section>

      {/* Autonomia + Memória + Git (métricas reais) */}
      <section className="trust-section">
        <h3 className="trust-title">Estado</h3>
        <div className="metrics-grid">
          <div className="metric">
            <span className="k">autonomia</span>
            <span className={`v ${data.autonomy === 'autonomous' ? 'st-bad' : 'st-ok'}`}>{data.autonomy ?? '—'}</span>
          </div>
          <div className="metric">
            <span className="k">memória</span>
            <span className="v">{data.memoryCount}</span>
          </div>
          <div className="metric">
            <span className="k">skills</span>
            <span className="v">{data.skills.length}</span>
          </div>
          <div className="metric">
            <span className="k">git mudanças</span>
            <span className="v">{changed}</span>
          </div>
        </div>
      </section>

      {/* Recursos não expostos pelo backend — honestidade (P14) */}
      {data.unavailable.length > 0 && (
        <section className="trust-section">
          <h3 className="trust-title">Não disponível (backend não expõe)</h3>
          <p className="empty">{data.unavailable.join(' · ')}</p>
          <p className="trust-hint">recursos que o professor descreveu — a UI não inventa; serão conectados quando o backend os expuser em leitura.</p>
        </section>
      )}
    </div>
  )
}
