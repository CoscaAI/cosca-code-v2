import { useEffect, useState } from 'react'
import { api } from '../api'
import { guardedFetch } from '../execGuard'

interface Diag {
  severity: number
  message: string
  range: { start: { line: number; character: number } }
}

// ProblemsPanel: mostra os diagnostics (erros/avisos) do arquivo ativo + botão
// de rodar os testes do projeto. Spec seção 2 (Problems) da Fase 2.
export function ProblemsPanel({ file }: { file: string | null }) {
  const [diags, setDiags] = useState<Diag[]>([])
  const [testResult, setTestResult] = useState<string | null>(null)
  const [running, setRunning] = useState(false)

  useEffect(() => {
    if (!file) { setDiags([]); return }
    fetch(api(`/api/lsp/diagnostics?path=${encodeURIComponent(file)}`))
      .then(r => r.json())
      .then((d: { diagnostics?: Diag[] }) => setDiags(d.diagnostics ?? []))
      .catch(() => setDiags([]))
  }, [file])

  const runTests = () => {
    setRunning(true)
    setTestResult('rodando testes…')
    // /api/test/run EXECUTA testes — passa pelo guard read-only (o GET anterior
    // burlava o bloqueio; POST via guardedFetch respeita a ordem do Don).
    guardedFetch(api('/api/test/run'), { method: 'POST' })
      .then(r => r.json())
      .then((res: { passed: boolean; output: string; error?: string }) => {
        if (res.error) setTestResult(res.error)
        else setTestResult(`${res.passed ? '✓ PASS' : '✗ FAIL'}\n${res.output}`)
      })
      .catch(e => setTestResult(String(e)))
      .finally(() => setRunning(false))
  }

  return (
    <aside className="sidebar">
      <div className="sidebar-title">PROBLEMS</div>
      {!file ? (
        <div className="empty">abra um arquivo para ver diagnostics</div>
      ) : diags.length === 0 ? (
        <div className="empty">nenhum problema em {file.split('/').pop()}</div>
      ) : (
        diags.map((d, i) => (
          <div key={i} className={`problem ${d.severity === 1 ? 'err' : 'warn'}`}>
            <span className="problem-line">{d.range.start.line + 1}</span>
            <span className="problem-msg">{d.message}</span>
          </div>
        ))
      )}

      <div className="sidebar-title" style={{ marginTop: 16 }}>TESTS</div>
      <div className="tests-box">
        <button className="run-tests" onClick={runTests} disabled={running}>
          {running ? 'rodando…' : '▶ run tests'}
        </button>
        {testResult && <pre className="tests-output">{testResult}</pre>}
      </div>
    </aside>
  )
}
