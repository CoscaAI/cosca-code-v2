// ActivityBar: navegação primária do COSCA CODE (spec seção 35).
// Estrutura enterprise: seções agrupam por função (navegação reflete
// arquitetura de informação — doutrina PRINCIPLES L204).
// Identidade própria — símbolos tipográficos, sem emojis (CHECKLIST G).

// Group agrupa itens por função.
interface Item {
  id: string
  label: string
  glyph: string
}

interface Group {
  title: string
  items: Item[]
}

const GROUPS: Group[] = [
  {
    title: 'CRIAR',
    items: [
      { id: 'canvas', label: 'Canvas', glyph: '⬡' },
      { id: 'ngraph', label: 'Node Graph', glyph: '≋' },
      { id: 'cinema', label: 'Cinema', glyph: '▶' },
      { id: 'music', label: 'Music', glyph: '♪' },
      { id: 'td3d', label: '3D', glyph: '◈' },
      { id: 'game', label: 'Game', glyph: '▣' },
      { id: 'scientific', label: 'Scientific', glyph: 'Σ' },
    ],
  },
  {
    title: 'DADOS',
    items: [
      { id: 'explorer', label: 'Explorer', glyph: '▤' },
      { id: 'assets', label: 'Assets', glyph: '▦' },
      { id: 'search', label: 'Buscar', glyph: '⌕' },
      { id: 'cross', label: 'Cross-Product', glyph: '⇄' },
    ],
  },
  {
    title: 'SISTEMA',
    items: [
      { id: 'pipeline', label: 'Pipeline', glyph: '⇥' },
      { id: 'ai', label: 'IA', glyph: '✦' },
      { id: 'git', label: 'Git', glyph: '⑂' },
      { id: 'problems', label: 'Problemas', glyph: '!' },
      { id: 'debug', label: 'Debug', glyph: '⏵' },
      { id: 'ext', label: 'Extensões', glyph: '▦' },
      { id: 'settings', label: 'Ajustes', glyph: '⚙' },
    ],
  },
]

export function ActivityBar({ active, onSelect }: {
  active: string | null
  onSelect: (id: any) => void
}) {
  return (
    <nav className="activity-bar">
      <div className="activity-logo" title="COSCA">C</div>
      {GROUPS.map(group => (
        <div key={group.title} className="activity-group">
          {group.items.map(item => (
            <button
              key={item.id}
              title={item.label}
              className={`activity-item ${active === item.id ? 'active' : ''}`}
              onClick={() => onSelect(active === item.id ? null : item.id)}
            >
              {item.glyph}
            </button>
          ))}
          <div className="activity-group-label">{group.title}</div>
        </div>
      ))}
    </nav>
  )
}
