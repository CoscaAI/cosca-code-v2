// Integrações LSP no editor: autocomplete (completion) e diagnostics
// sublinhados. Ambos consultam o language server real via API HTTP.
import { StateField, StateEffect } from '@codemirror/state'
import { Decoration, DecorationSet, EditorView } from '@codemirror/view'
import { RangeSetBuilder } from '@codemirror/state'
import { autocompletion, type CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import { api } from './api'

// ── Completion ────────────────────────────────────────────────────────

export function lspCompletion(getFile: () => string | null) {
  return autocompletion({
    activateOnTyping: true,
    override: [async (ctx: CompletionContext): Promise<CompletionResult | null> => {
      const file = getFile()
      if (!file) return null
      const pos = ctx.pos
      const line = ctx.state.doc.lineAt(pos)
      const char = pos - line.from
      try {
        const res = await fetch(
          api(`/api/lsp/completion?path=${encodeURIComponent(file)}&line=${line.number - 1}&char=${char}`),
        ).then(r => r.json())
        if (!res?.items?.length) return null
        return {
          from: pos,
          options: res.items.map((it: { label: string; detail?: string; kind?: number }) => ({
            label: it.label,
            detail: it.detail,
            type: it.kind === 3 || it.kind === 12 ? 'function' : it.kind === 6 ? 'variable' : undefined,
          })),
        }
      } catch {
        return null
      }
    }],
  })
}

// ── Diagnostics ───────────────────────────────────────────────────────

interface LSPDiag {
  severity: number
  message: string
  range: { start: { line: number; character: number }; end: { line: number; character: number } }
}

const setDiagnostics = StateEffect.define<LSPDiag[]>()

const diagWarning = Decoration.mark({ class: 'cm-diag-warn' })
const diagError = Decoration.mark({ class: 'cm-diag-error' })

const diagnosticsField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none
  },
  update(deco, tr) {
    deco = deco.map(tr.changes)
    for (const e of tr.effects) {
      if (e.is(setDiagnostics)) {
        deco = buildDecorations(tr.state.doc, e.value)
      }
    }
    return deco
  },
  provide: f => EditorView.decorations.from(f),
})

export { diagnosticsField }

function buildDecorations(doc: { line(n: number): { from: number } }, diags: LSPDiag[]): DecorationSet {
  const builder = new RangeSetBuilder<Decoration>()
  for (const d of diags) {
    const from = lineCharToOffset(doc, d.range.start.line, d.range.start.character)
    const to = lineCharToOffset(doc, d.range.end.line, d.range.end.character)
    if (from >= to) continue
    builder.add(from, to, d.severity === 1 ? diagError : diagWarning)
  }
  return builder.finish()
}

function lineCharToOffset(doc: { line(n: number): { from: number } }, line: number, char: number): number {
  const n = line + 1
  if (n < 1) return 0
  try {
    return doc.line(n).from + char
  } catch {
    return 0
  }
}

// loadDiagnostics consulta o LSP e atualiza as decorations de erro.
export function loadDiagnostics(view: EditorView, file: string) {
  fetch(api(`/api/lsp/diagnostics?path=${encodeURIComponent(file)}`))
    .then(r => r.json())
    .then((res: { diagnostics?: LSPDiag[] }) => {
      view.dispatch({ effects: setDiagnostics.of(res.diagnostics ?? []) })
    })
    .catch(() => {})
}
