import { useState } from 'react'
import type { FileNode } from '../types'

// Explorer: árvore de arquivos (spec seção 32), renderizada recursivamente.
export function Explorer({ tree, activeFile, onOpen }: {
  tree: FileNode | null
  activeFile: string | null
  onOpen: (path: string) => void
}) {
  return (
    <aside className="sidebar">
      <div className="sidebar-title">EXPLORER</div>
      {tree ? <TreeItem node={tree} activeFile={activeFile} onOpen={onOpen} depth={0} /> : <div className="empty">carregando…</div>}
    </aside>
  )
}

function TreeItem({ node, activeFile, onOpen, depth }: {
  node: FileNode
  activeFile: string | null
  onOpen: (path: string) => void
  depth: number
}) {
  const [open, setOpen] = useState(depth === 0)

  if (node.is_dir) {
    return (
      <div>
        <div className="tree-row dir" style={{ paddingLeft: depth * 12 }} onClick={() => setOpen(!open)}>
          <span className="chevron">{open ? '▾' : '▸'}</span>
          <span className="name">{node.name}</span>
        </div>
        {open && node.children?.map(c => (
          <TreeItem key={c.path} node={c} activeFile={activeFile} onOpen={onOpen} depth={depth + 1} />
        ))}
      </div>
    )
  }

  return (
    <div
      className={`tree-row file ${activeFile === node.path ? 'active' : ''}`}
      style={{ paddingLeft: depth * 12 + 16 }}
      onClick={() => onOpen(node.path)}
    >
      <span className="name">{node.name}</span>
    </div>
  )
}
