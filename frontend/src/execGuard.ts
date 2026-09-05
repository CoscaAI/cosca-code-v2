// execGuard.ts — o BLOQUEIO DE EXECUÇÃO do Cosca Code (ordem do Don 2026-08-14:
// "bloqueia qualquer execucao por enquando somente leitura").
//
// A P14 + esta ordem: a UI NÃO pode executar NADA por enquanto. Este módulo
// intercepta os endpoints de execução (agent, team, workflow, plan, test,
// save, engine/run, inline) e os transforma em um bloqueio EXPLÍCITO — a UI
// mostra "read-only" em vez de tentar executar. A leitura continua livre.
//
// IMPORTANTE: isso é a camada da UI. O backend continua intacto — o bloqueio
// é uma política de segurança da interface, não uma mudança de servidor.
export const READONLY_MODE = true

// Endpoints de EXECUÇÃO — qualquer chamada a eles é bloqueada nesta fase.
// Cobre agentes, times, workflows, plan, testes, save e os engines de mídia
// (td3d/cinema/sci/x). A LEITURA (GET) continua livre em todos.
const EXEC_PATHS = [
  '/api/agent/run',
  '/api/team/run',
  '/api/workflow/run',
  '/api/plan/generate',
  '/api/plan/approve',
  '/api/plan/reject',
  '/api/test/run',
  '/api/save',
  '/api/engine/graph/run',
  '/api/engine/td3d/run',
  '/api/engine/cinema/run',
  '/api/engine/sci/run',
  '/api/engine/sci/design',
  '/api/engine/x/run',
  '/api/ai/inline',
]

// isExecPath devolve true se a URL aponta para um endpoint de execução.
// REGRA: qualquer path contendo "/run", "/design", "/save", "/approve",
// "/reject" ou "/generate" é execução (ou mutação) — a leitura usa GET e
// esses endpoints são POST por definição.
export function isExecPath(url: string): boolean {
  for (const p of EXEC_PATHS) {
    if (url.includes(p)) return true
  }
  // Fallback conservador: path com ação mutável, mesmo que não listado.
  const u = url.replace(/^[^?]*\/\/[^/]+/, '').split('?')[0]
  return /(\/run|\/design|\/save|\/approve|\/reject|\/generate)$/.test(u)
}

// guardedFetch é o fetch seguro: bloqueia execução, libera leitura (GET).
// Use nos componentes no lugar de fetch para chamadas POST.
export async function guardedFetch(url: string, init?: RequestInit): Promise<Response> {
  const method = (init?.method ?? 'GET').toUpperCase()

  // Execução bloqueada — a ordem do Don.
  if (isExecPath(url) && method !== 'GET') {
    return new Response(
      JSON.stringify({ error: 'read-only: execução bloqueada nesta fase (ordem do Don)' }),
      { status: 403, headers: { 'Content-Type': 'application/json' } },
    )
  }

  // Leitura (GET) e chat (POST /api/ai/chat — não é execução de agente)
  // continuam livres.
  return fetch(url, init)
}

// readOnlyError devolve o erro padrão de execução bloqueada (para UI mostrar).
export function readOnlyError(): Error {
  return new Error('read-only: execução bloqueada nesta fase')
}
