// ProjectMap.tsx — o PROJECT MAP (pilar 6 do professor): uma visão de quem é
// quem no código. SOMENTE LEITURA — usa /api/intel/symbols (símbolos reais do
// projeto) e /api/intel/dependents (quem depende de um arquivo). Quando o
// backend não tem dado, mostra "sem dados" — nunca inventa (P14).
import { useEffect, useState } from 'react'
import { apiAsync } from '../api'

interface SymbolInfo {
  name: string
  kind: string
  file: string
  line: number
}

interface ProjectMapState {
  symbols: SymbolInfo[]
  selected: SymbolInfo | null
  dependents: string[] | null
  usages: { name: string; kind: string; file: string; line: number }[] | null
  unavailable: string[]
  loading: boolean
}

export function ProjectMap() {
  const [st, setSt] = useState<ProjectMapState>({
    symbols: [],
    selected: null,
    dependents: null,
    usages: null,
    unavailable: [],
    loading: true,
  })

  // Carrega os símbolos do projeto (real).
  useEffect(() => {
    let alive = true
    ;(async () => {
      try {
        const url = await apiAsync('/api/intel/symbols')
        const resp = await fetch(url)
        if (!resp.ok) {
          if (alive) setSt(s => ({ ...s, loading: false, unavailable: [...s.unavailable, 'symbols'] }))
          return
        }
        const data = (await resp.json()) as { symbols?: SymbolInfo[] }
        if (alive) setSt(s => ({ ...s, symbols: data.symbols ?? [], loading: false }))
      } catch {
        if (alive) setSt(s => ({ ...s, loading: false, unavailable: [...s.unavailable, 'backend'] }))
      }
    })()
    return () => { alive = false }
  }, [])

  // Ao selecionar um símbolo, busca dependentes do arquivo (real).
  const select = async (sym: SymbolInfo) => {
    setSt(s => ({ ...s, selected: sym, dependents: null, usages: null, loading: true }))
    try {
      const url = await apiAsync(`/api/intel/dependents?file=${encodeURIComponent(sym.file)}`)
      const resp = await fetch(url)
      if (!resp.ok) {
        setSt(s => ({ ...s, dependents: [], loading: false }))
        return
      }
      const data = (await resp.json()) as { dependents?: string[] }
      setSt(s => ({ ...s, dependents: data.dependents ?? [], loading: false }))
    } catch {
      setSt(s => ({ ...s, dependents: [], loading: false }))
    }
  }

  if (st.loading && st.symbols.length === 0) {
    return <div className="dock-panel map-panel"><p className="empty">carregando mapa do projeto…</p></div>
  }

  const kinds: Record<string, string> = {
    func: 'fn', type: 'type', method: 'm', struct: 's', interface: 'i', const: 'c', var: 'v',
  }

  return (
    <div className="dock-panel map-panel">
      {/* Símbolos do projeto */}
      <section className="trust-section">
        <h3 className="trust-title">Símbolos ({st.symbols.length})</h3>
        {st.symbols.length === 0 ? (
          <p className="empty">sem índice de símbolos (backend sem intel)</p>
        ) : (
          <ul className="map-symbols">
            {st.symbols.slice(0, 30).map((s) => (
              <li
                key={`${s.name}-${s.file}-${s.line}`}
                className={`map-symbol ${st.selected?.name === s.name ? 'active' : ''}`}
                onClick={() => select(s)}
              >
                <span className="map-kind">{kinds[s.kind] ?? s.kind}</span>
                <span className="map-name">{s.name}</span>
                <span className="map-file">{s.file}:{s.line}</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Detalhe do símbolo selecionado — dependentes reais */}
      {st.selected && (
        <section className="trust-section">
          <h3 className="trust-title">Dependentes de {st.selected.file}</h3>
          {st.dependents === null ? (
            <p className="empty">consultando…</p>
          ) : st.dependents.length === 0 ? (
            <p className="empty">nenhum dependente indexado (ou arquivo sem importadores)</p>
          ) : (
            <ul className="map-symbols">
              {st.dependents.map((d) => (
                <li key={d} className="map-symbol"><span className="map-name">{d}</span></li>
              ))}
            </ul>
          )}
        </section>
      )}

      {/* Honestidade P14 */}
      {st.unavailable.length > 0 && (
        <section className="trust-section">
          <h3 className="trust-title">Não disponível</h3>
          <p className="empty">{st.unavailable.join(' · ')}</p>
        </section>
      )}
    </div>
  )
}
