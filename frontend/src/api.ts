// Helper de API: o backend roda em http://127.0.0.1:<porta>. A porta é
// DESCOBERTA em runtime (port discovery): o backend escolhe porta dinâmica
// quando a preferida (14126) está ocupada (Fase 2.2 — resolveFreePort), então
// o frontend NUNCA fixa uma porta. Estratégia:
//
//   1. Tenta a porta preferida (14126) — caso comum.
//   2. Se falhar, testa a faixa de portas da família (14120-14140) via
//      /health procurando "service":"cosca-code".
//   3. Falha só se nenhuma porta responder (backend não está no ar).
//
// Isso elimina a detecção frágil de ambiente (window.go) que quebrava o
// desktop — sem bindings gerados, usamos HTTP discovery puro.

// FAMILY_PORTS é a faixa de portas da família Cosca (serve 14120, runtime
// 14123, neural-link 14124, node 14125, code 14126) — o discovery testa esta
// faixa procurando o serviço cosca-code.
const FAMILY_PORTS = [14126, 14125, 14124, 14123, 14120, 14121, 14122, 14127, 14128, 14129, 14130, 14131, 14132, 14133, 14134, 14135, 14136, 14137, 14138, 14139, 14140]

let apiBase: string | null = null

// resolveBase descobre a base da API (uma vez, cacheada).
async function resolveBase(): Promise<string> {
  if (apiBase) return apiBase

  // 1. Porta preferida primeiro.
  if (await probePort(14126)) {
    apiBase = 'http://127.0.0.1:14126'
    return apiBase
  }
  // 2. Faixa da família.
  for (const port of FAMILY_PORTS) {
    if (port === 14126) continue // já tentada
    if (await probePort(port)) {
      apiBase = `http://127.0.0.1:${port}`
      return apiBase
    }
  }
  // 3. Nenhum backend — usa a preferida (o erro real aparecerá nos fetch).
  apiBase = 'http://127.0.0.1:14126'
  return apiBase
}

// probePort verifica se um backend cosca-code responde na porta (via /health).
async function probePort(port: number): Promise<boolean> {
  try {
    const ctl = new AbortController()
    const timer = setTimeout(() => ctl.abort(), 500)
    const resp = await fetch(`http://127.0.0.1:${port}/health`, { signal: ctl.signal })
    clearTimeout(timer)
    if (!resp.ok) return false
    const data = await resp.json()
    return data?.service === 'cosca-code'
  } catch {
    return false
  }
}

// base devolve a base resolvida (async — chame antes de usar api/wsUrl).
export async function base(): Promise<string> {
  return resolveBase()
}

export function api(path: string): string {
  // Síncrono para compatibilidade: usa a porta preferida como default e o
  // discovery ajusta no primeiro uso via apiAsync. Novos chamadores devem
  // preferir apiAsync para não depender da porta fixa.
  return `http://127.0.0.1:14126${path}`
}

// apiAsync resolve a base real (port discovery) e devolve a URL completa.
export async function apiAsync(path: string): Promise<string> {
  const b = await resolveBase()
  return `${b}${path}`
}

export function wsUrl(path: string): string {
  return `ws://127.0.0.1:14126${path}`
}

// wsUrlAsync resolve a base real e devolve a URL WebSocket completa.
export async function wsUrlAsync(path: string): Promise<string> {
  const b = await resolveBase()
  return b.replace(/^http/, 'ws') + path
}
