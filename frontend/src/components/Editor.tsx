import { useEffect, useRef, useState } from 'react'
import { EditorState } from '@codemirror/state'
import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLineGutter,
  highlightActiveLine,
  drawSelection,
  hoverTooltip,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'
import { go } from '@codemirror/lang-go'
import { oneDark } from '@codemirror/theme-one-dark'
import { api } from '../api'
import { guardedFetch, READONLY_MODE } from '../execGuard'
import { lspCompletion, diagnosticsField, loadDiagnostics } from '../lsp'
import { InlineAI, type InlineRequest } from './InlineAI'
import type { ProjectInfo } from '../types'

// Editor: view CodeMirror 6 sobre o conteúdo do arquivo ativo.
// ⚠ READ-ONLY (ordem do Don 2026-08-14): o save (Ctrl+S) está BLOQUEADO — o
// conteúdo é editável na tela, mas nada é gravado até a fase de escrita.
export function Editor({ file, info }: { file: string | null; info: ProjectInfo | null }) {
  const container = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const fileRef = useRef<string | null>(null)
  fileRef.current = file

  const [, setDirty] = useState(false)
  const [inlineReq, setInlineReq] = useState<InlineRequest | null>(null)
  const diagTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Ref para abrir o Inline AI a partir do keymap (evita closure velha).
  const inlineOpenRef = useRef<(req: InlineRequest) => void>(() => {})
  inlineOpenRef.current = (req: InlineRequest) => setInlineReq(req)

  // openInlineAI captura a seleção (ou a linha inteira se vazia) e abre o overlay.
  const openInlineAI = (view: EditorView) => {
    const sel = view.state.selection.main
    if (sel.from !== sel.to) {
      inlineOpenRef.current({ from: sel.from, to: sel.to, text: view.state.sliceDoc(sel.from, sel.to) })
    } else {
      const line = view.state.doc.lineAt(sel.from)
      inlineOpenRef.current({ from: line.from, to: line.to, text: line.text })
    }
  }

  // Função de save capturada via ref (evita closure velha na extensão).
  // READ-ONLY: bloqueia o POST /api/save — a edição fica em tela, não persiste.
  const saveRef = useRef<() => void>(() => {})
  saveRef.current = () => {
    const f = fileRef.current
    const view = viewRef.current
    if (!f || !view) return
    if (READONLY_MODE) {
      setDirty(false)
      return
    }
    const content = view.state.doc.toString()
    guardedFetch(api('/api/save'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: f, content }),
    })
      .then(() => setDirty(false))
      .catch(() => {})
  }

  // Cria o EditorView uma única vez (a extensão de linguagem depende da
  // linguagem detectada do projeto).
  useEffect(() => {
    if (!container.current) return
    const view = new EditorView({
      parent: container.current,
      state: EditorState.create({
        doc: '',
        extensions: [
          lineNumbers(),
          highlightActiveLineGutter(),
          highlightActiveLine(),
          history(),
          drawSelection(),
          indentOnInput(),
          bracketMatching(),
          syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
          keymap.of([
            ...defaultKeymap, ...historyKeymap, indentWithTab,
            { key: 'Mod-s', run: () => { saveRef.current(); return true } },
            { key: 'Mod-k', run: () => { openInlineAI(view); return true } },
          ]),
          EditorView.updateListener.of(update => {
            if (update.docChanged) {
              setDirty(true)
              // Debounce: recarrega diagnostics após pausa na digitação.
              if (diagTimer.current) clearTimeout(diagTimer.current)
              diagTimer.current = setTimeout(() => {
                const f = fileRef.current
                const v = viewRef.current
                if (f && v) loadDiagnostics(v, f)
              }, 600)
            }
          }),
          lspCompletion(() => fileRef.current),
          diagnosticsField,
          lspHover(() => fileRef.current),
          info?.language === 'go' ? go() : [],
          oneDark,
        ],
      }),
    })
    viewRef.current = view
    return () => view.destroy()
  }, [info?.language])

  // Carrega o conteúdo quando o arquivo muda (e limpa o dirty).
  useEffect(() => {
    if (!file || !viewRef.current) return
    fetch(api(`/api/read?path=${encodeURIComponent(file)}`))
      .then(r => r.json())
      .then((data: { content: string }) => {
        viewRef.current?.dispatch({
          changes: { from: 0, to: viewRef.current.state.doc.length, insert: data.content },
        })
        setDirty(false)
        if (viewRef.current) loadDiagnostics(viewRef.current, file)
      })
      .catch(() => {})
  }, [file])

  if (!file) {
    return (
      <div className="editor-welcome">
        <div className="logo">COSCA CODE</div>
        <p>Abra um arquivo no explorador para começar.</p>
      </div>
    )
  }

  return (
    <div className="editor-shell">
      <div ref={container} className="cm-host" />

      {inlineReq && (
        <InlineAI
          request={inlineReq}
          onApply={(text) => {
            viewRef.current?.dispatch({
              changes: { from: inlineReq.from, to: inlineReq.to, insert: text },
            })
            setInlineReq(null)
          }}
          onDiscard={() => setInlineReq(null)}
        />
      )}
    </div>
  )
}

// lspHover consulta o LSP real (hover) na posição do cursor e mostra um tooltip
// com a documentação/assinatura do símbolo.
function lspHover(getFile: () => string | null) {
  return hoverTooltip(async (view, pos) => {
    const file = getFile()
    if (!file) return null
    const line = view.state.doc.lineAt(pos)
    const char = pos - line.from
    try {
      const res = await fetch(
        api(`/api/lsp/hover?path=${encodeURIComponent(file)}&line=${line.number - 1}&char=${char}`),
      ).then(r => r.json())
      if (res?.hover) {
        return {
          pos,
          end: line.to,
          above: true,
          create: () => ({ dom: hoverDom(res.hover) }),
        }
      }
    } catch {
      /* LSP indisponível */
    }
    return null
  })
}

function hoverDom(text: string): HTMLElement {
  const el = document.createElement('div')
  el.className = 'cm-hover'
  // Renderiza markdown simples: remove fences e quebra linhas.
  const clean = text.replace(/```\w*\n/g, '').replace(/```/g, '')
  el.textContent = clean
  el.style.maxWidth = '560px'
  el.style.whiteSpace = 'pre-wrap'
  return el
}
