# COSCA CODE — Arquitetura Original

> Proposta de arquitetura. Baseada no RESEARCH_INDEX (padrões extraídos de VS Code, Zed, Theia, OpenCode, Codex, Cline, MCP, tree-sitter, ripgrep, ast-grep, DAP) — mas com identidade própria. Nenhuma cópia.

## 1. Princípios fundadores

1. **Core headless, clientes finos.** O coração é um motor Go desacoplado (SDK). IDE (Wails/React) é UM cliente; CLI, Web e CI são outros. Nenhuma lógica crítica vive na UI.
2. **Event bus como sistema nervoso.** Tudo emite eventos (`workspace.opened`, `agent.tool_called`, `test.failed`). Observabilidade e extensibilidade nascem do fluxo de eventos, não de callbacks espalhados.
3. **Sintaxe × semântica separadas.** Sintaxe = tree-sitter incremental (barato/local). Semântica = LSP/compilador (caro/global). Nunca misturar.
4. **Segurança por default.** Nenhum agente com acesso irrestrito. Permissões, sandbox e audit são a fundação, não um recurso.
5. **Reversibilidade.** Toda mutação é diffable/reversible/attributable. Autonomia só existe com undo e rastro.

---

## 2. Camadas

```
┌─────────────────────────────────────────────────────────────┐
│  UI (React + Vite + TS, Wails desktop)                       │
│  Workbench · Explorer · Editor · Tabs · Terminal · Panels     │
│  Command Palette · AI Panel · Agent Panel · Model/Provider    │
└───────────────▲─────────────────────────────▲───────────────┘
                │ RPC tipado (contrato)         │ eventos (stream)
┌───────────────┴─────────────────────────────┴───────────────┐
│  CORE (Go) — headless, orientado a eventos                    │
│                                                              │
│  ┌────────────┐  ┌─────────────┐  ┌──────────────────────┐  │
│  │ EVENT BUS  │◄─┤ AGENT ENGINE│  │ ORCHESTRATOR         │  │
│  └─────┬──────┘  │ planner/    │  │ intent→plan→execute   │  │
│        │         │ coder/debug │  │ →observe→test→verify  │  │
│        │         │ tester/rev… │  └──────────────────────┘  │
│        │         └──────┬──────┘                             │
│        │                │                                    │
│  ┌─────┴────────────────┴──────────────────────────────┐     │
│  │  ENGINES                                            │     │
│  │  Workspace · Editor(rope) · Language(LSP)           │     │
│  │  CodeIntelligence · Search · Debug(DAP) · Git       │     │
│  │  Terminal(PTY) · Provider · ModelRouter · Tool      │     │
│  │  Context · Security · Memory · Task · Workflow      │     │
│  └─────────────────────────────────────────────────────┘     │
│        │                                                     │
│  ┌─────┴─────────────────────────────────────────────┐       │
│  │  STORAGE: SQLite (index/cache/memory) · Secrets   │       │
│  └───────────────────────────────────────────────────┘       │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. Componentes do Core (responsabilidades)

### 3.1 Event Bus
Fila de eventos tipados com pub/sub. Fonte única de verdade para observabilidade, extensões e telemetry. Todo componente emite/consome via bus — nunca via acoplamento direto.

### 3.2 Workspace Engine
Multi-root, detecção de projeto, file watching incremental, RPC proxy (local/remote/SSH/container = mesmo contrato). É o "backend isolado" — a UI nunca toca o filesystem diretamente.

### 3.3 Editor Engine
Rope (buffer imutável) + undo ilimitado + diff + tabs + splits. Consome tree-sitter para highlighting/outline via Document/View/Provider. O editor é uma *view* sobre o rope; a inteligência vem de providers.

### 3.4 Language Services (LSP)
Cliente LSP universal com auto-detecção (linguagem → projeto → ferramenta → LSP). Completion, hover, definition, references, rename, diagnostics, code actions, formatting, symbols, semantic tokens. Zero-config por padrão, override sempre possível.

### 3.5 Code Intelligence Engine
Indexação incremental: symbol graph, dependency graph, imports/exports, referências, tipos, testes, configs, docs, Git history. Responde "onde isso é usado?", "o que quebra se eu alterar?".

### 3.6 Search Engine
Multi-estratégia com seleção automática: text, regex, symbol, AST, semantic, dependency, reference. Texto = scan on-the-fly git-aware (ripgrep-like); estrutura = query tree-sitter (ast-grep-like); semântica = índice.

### 3.7 Debug Engine (COSCA DEBUG ENGINE)
Host DAP universal + adapters por linguagem/runtime. Breakpoints (condicional/logpoint/exceção), stepping, call stack, scopes, variables, watch, evaluate, threads, source maps, launch/attach/restart, debug console. Nunca um debugger artificial.

### 3.8 Git Engine
Source control nativo: diff, inline diff, staging, branches, history, blame, merge/conflicts, stash, commit, reset, cherry-pick, rebase, worktrees, patches.

### 3.9 Terminal Engine
PTY múltiplos, split, tabs, shells, ambientes, SSH, containers. O agente usa terminal via tools controladas (nunca acesso direto irrestrito).

### 3.10 Provider Engine + Model Router
Abstração universal de provider (first-party/cloud/openai-compatible/local/self-hosted/enterprise/aggregator). Catálogo de modelos com capability matrix (context window, tool calling, structured output, vision, reasoning, coding, pricing, latency). Router AUTO (por tarefa/custo/latência) ou MANUAL. Failover entre providers. Multi-provider/multi-model simultâneos.

### 3.11 Agent Engine
Agentes (planner, coder, researcher, debugger, tester, reviewer, security, architect, docs, refactor, release) sobre infraestrutura comum: capabilities, tools, context policy, budget, permissions, memory, objectives, verification strategy. Subagentes + modos + workspaces isolados.

### 3.12 Tool Engine (Tool Registry)
Sistema próprio de tools: `name` + `description` + `inputSchema` + handler tipado. Adaptadores MCP (client/server) para interoperar. Progressive discovery (catalog → inspect → execute).

### 3.13 Context Engine
Montagem dinâmica de contexto com ranking de relevância: arquivos relevantes, símbolos, dependências, erros, Git, tarefa, histórico, testes, docs, memória, decisões. Orçado (caps) e composto (fragmentos tipados). Nunca o projeto inteiro.

### 3.14 Security Engine
Permissões allow/ask/deny por comando/tool/glob; sandbox de execução; secret store (nunca enviar segredos ao modelo sem necessidade); audit trail; níveis de autonomia (read-only/assisted/approval/autonomous/full).

### 3.15 Memory + Knowledge
Memória com provenance/timestamp/confidence/version (nunca verdade absoluta). Project Knowledge (arquitetura, convenções, decisões, padrões, falhas). Onboarding: análise do projeto → PROJECT MAP.

### 3.16 Task System + Workflow Engine
Tarefas (ID/título/status/agente/modelo/arquivos/testes/evidência). Workflows (bug fix, feature, refactor, migration, security review, testing, release) com agentes diferentes.

---

## 4. Fluxo de dados (o loop do orquestrador)

```
USER → INTENT → PLAN → CONTEXT → AGENT → TOOLS → EXECUTION
     → OBSERVATION → TEST → REVIEW → VERIFY → RESULT
```

Cada etapa emite eventos no bus → o usuário acompanha (Agent Observability, seção 44), pode replay (45), e o audit registra tudo (42).

---

## 5. Decisões-chave (e por quê)

| Decisão | Justificativa |
|---|---|
| **Go no core, React/Vite/TS na UI, Wails desktop** | Go = desempenho + concorrência + binário único + bindings tree-sitter; React = ecossistema de editor; Wails = leve (vs Electron). Spec 69/70. |
| **Rope (buffer imutável)** | undo ilimitado, diff, split e colaboração como consequência natural (padrão Lapce/Zed). |
| **tree-sitter embutido (go-tree-sitter)** | sintaxe viva/incremental/tolerante a erro sem depender de LSP para highlighting/outline. |
| **LSP/DAP como contratos, adapters finos** | universalidade + zero reimplementação de semântica/debugging (padrão VS Code/debugpy). |
| **Core headless orientado a eventos** | IDE/CLI/Web/CI = clientes do mesmo motor (padrão Cline SDK/OpenCode/Continue). |
| **Tool Registry + adaptadores MCP** | sistema próprio de tools (seguro, tipado) + interop com o ecossistema MCP. |
| **Segurança por default (sandbox + allow/ask/deny + audit)** | nenhum agente com acesso irrestrito (padrão OpenCode/Codex). |
| **SQLite para índice/cache/memória** | portátil, embutido, sem serviço externo (offline-friendly, seção 67). |

---

## 6. Fases de implementação (spec seção 92)

- **FASE 1 — Fundação**: Core, Desktop, Workspace, Editor (rope), Tabs, Explorer, Terminal, Git, LSP.
- **FASE 2 — Inteligência de código**: Search, Index, Project Intelligence, Debugger, Tests, Problems, Output.
- **FASE 3 — IA**: Provider Engine, Model Catalog, Model Router, AI Panel, Inline AI.
- **FASE 4 — Agentes**: Agent Engine, Tools, MCP, Skills, Memory, Workflows.
- **FASE 5 — Autonomia**: Multi-agent, execução autônoma, verificação, recovery, debugging avançado.
- **FASE 6 — Ecossistema**: Extensões, plugin ecosystem, marketplace.
- **FASE 7 — Escala**: Performance, grandes repos, remote development, enterprise.

Cada fase valida antes de avançar (Definition of Done, seção 93).
