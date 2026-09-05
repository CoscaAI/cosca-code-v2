import { useState } from 'react'
import { api } from '../api'
import { guardedFetch, READONLY_MODE } from '../execGuard'

interface Step { tool: string; args: any; result: string }

// PipelinePanel: mostra o pipeline do agente (spec seções 19/44).
// ⚠ READ-ONLY (ordem do Don 2026-08-14): a execução está BLOQUEADA — o painel
// explica o estado e não envia nada que execute.
export function PipelinePanel() {
  const [task, setTask] = useState('')
  const [mode, setMode] = useState<'agent' | 'team' | 'workflow'>('agent')
  const [steps, setSteps] = useState<Step[]>([])
  const [final, setFinal] = useState('')
  const [running, setRunning] = useState(false)
  const [error, setError] = useState(READONLY_MODE ? 'read-only: execução bloqueada nesta fase (ordem do Don)' : '')

  const run = async () => {
    const t = task.trim()
    if (!t || running) return
    if (READONLY_MODE) {
      setError('read-only: execução bloqueada — somente leitura nesta fase (ordem do Don)')
      return
    }
    setRunning(true)
    setSteps([])
    setFinal('')
    setError('')

    const endpoint = mode === 'agent' ? '/api/agent/run' : mode === 'team' ? '/api/team/run' : '/api/workflow/run'
    const body = mode === 'workflow' ? { workflow: 'bug_fix', task: t } : { task: t }

    try {
      const res = await guardedFetch(api(endpoint), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }).then(r => r.json())

      if (res.error) {
        setError(res.error)
      } else if (mode === 'agent') {
        setSteps(res.steps ?? [])
        setFinal(res.final ?? '')
      } else if (mode === 'team') {
        setSteps([
          { tool: 'planner', args: {}, result: res.plan ?? '' },
          { tool: 'coder', args: {}, result: res.code ?? '' },
          { tool: 'reviewer', args: {}, result: res.review ?? '' },
        ])
        setFinal(res.review ?? '')
      } else {
        setSteps((res.steps ?? []).map((s: any) => ({ tool: s.name, args: {}, result: s.output })))
        setFinal(res.final ?? '')
      }
    } catch (e) {
      setError(String(e))
    } finally {
      setRunning(false)
    }
  }

  return (
    <aside className="sidebar pipeline-panel">
      <div className="sidebar-title">PIPELINE</div>
      <div className="settings-connect">
        <textarea
          className="ai-input"
          rows={3}
          placeholder="descreva a tarefa… (ex: quantos arquivos .go existem?)"
          value={task}
          onChange={e => setTask(e.target.value)}
        />
        <div className="pipeline-modes">
          {(['agent', 'team', 'workflow'] as const).map(m => (
            <button key={m} className={`autonomy-btn ${mode === m ? 'active' : ''}`} onClick={() => setMode(m)}>{m}</button>
          ))}
        </div>
        <button className="run-tests" onClick={run} disabled={running || !task.trim()}>
          {running ? 'executando…' : `▶ executar (${mode})`}
        </button>
      </div>

      {error && <div className="pipeline-error">⚠ {error}</div>}

      <div className="pipeline-steps">
        {steps.map((s, i) => (
          <div key={i} className="pipeline-step">
            <div className="step-head">
              <span className="step-index">{i + 1}</span>
              <span className="step-tool">{s.tool}</span>
            </div>
            <div className="step-result">{s.result}</div>
          </div>
        ))}
      </div>

      {final && (
        <div className="pipeline-final">
          <div className="step-head"><span className="step-index">✓</span><span className="step-tool">resultado final</span></div>
          <div className="step-result final">{final}</div>
        </div>
      )}
    </aside>
  )
}
