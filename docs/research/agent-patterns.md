# COSCA CODE — Pesquisa de Padrões Arquiteturais em AI Coding Agents

> Estudo comparativo de arquiteturas de agentes de codificação open source.
> Objetivo: extrair **padrões, decisões arquiteturais, abstrações e estratégias** para o design do **COSCA CODE** (codename COSCA STUDIO).
> Pesquisa de arquitetura — não copiamos código, UI, branding, nomenclatura ou prompts proprietários.

Fontes: OpenCode (`sst/opencode` → hoje `anomalyco/opencode`), Codex CLI (`openai/codex`), Claude Code (`anthropics/claude-code`), Cline (`cline/cline`), Continue (`continuedev/continue`), Roo Code (`RooCodeInc/Roo-Code` + `Roo-Code-Docs`).

---

### SOURCE: OpenCode (`sst/opencode` → `anomalyco/opencode`)

- **WHAT IT SOLVES**
  Frontend-agnóstico e agnóstico de modelo para um agente de codificação: um único *engine* com múltiplos clientes (TUI, Web, desktop, IDE) e suporte a dezenas de providers sem amarrar o produto a um LLM. Também resolve a necessidade de um agente extensível via config declarativa, MCP, LSP e plugins, sem uma UI proprietária obrigatória.

- **HOW IT WORKS**
  Arquitetura **client/server**: um servidor local (Node/Bun + TypeScript) expõe um protocolo de eventos; clientes (TUI, Web/desktop, extensão VSCode, SDK JS) conectam e trocam mensagens. O modelo é abstraído por `provider/model-id` (Anthropic, OpenAI, Google, Bedrock, local, Zen, etc.). O sistema de **agentes** separa `primary` (build/plan) de `subagent` (general/explore/scout) e ainda tem agentes de sistema ocultos (compaction, title, summary). O sistema de **permissions** (allow/ask/deny) com wildcards e globs por comando/ferramenta controla o que cada agente pode fazer. Ferramentas: `read`, `edit`, `write`, `apply_patch`, `glob`, `grep`, `list`, `bash`, `task`, `webfetch`, `websearch`, `lsp`, `skill`, `question`, `todowrite/todoread`, `external_directory`. Configuração em `opencode.json` + agentes em markdown (frontmatter), global (`~/.config/opencode`) ou por projeto (`.opencode/`).

- **WHY IT WORKS**
  A separação client/server permite *um* engine reutilizado por N superfícies; a abstração de provider evita lock-in; o modelo de agentes+permissions torna o mesmo core útil para plan/build/review/debug; agentes markdown baixam a barreira de extensão (config declarativa, não código).

- **STRENGTHS**
  - Abstração de modelo limpa (`provider/model-id`) e multi-provider.
  - Permissions granulares (por ferramenta, por comando bash com glob, por agente, por subagente via `task`).
  - Subagentes + sessões-filhas com navegação própria (parent/child/cycle).
  - Agentes declarativos em markdown + frontmatter.
  - `doom_loop` (recuperação de agente travado), `external_directory` (fronteira do worktree).
  - Protocolo client/server reutilizável + SDK.

- **WEAKNESSES**
  - Servidor local sempre ligado; overhead de um protocolo para operações simples.
  - Config em duas formas (JSON + markdown) pode confundir; `tools` legado vs `permission` novo (depreciação em curso).
  - Superfície de features ampla e em rápida evolução (churn de config).

- **PATTERN**
  **"Engine-core + clientes finos" com "Agents-as-config" e "Permission-matrix"**: um core orientado a eventos expõe um protocolo; agentes são objetos configuráveis (prompt, modelo, permissions, temperature, steps) em vez de classes hardcoded; a segurança é uma matriz de allow/ask/deny por tool×padrão.

- **COSCA INTERPRETATION**
  Adotar o padrão de **agente declarativo** (frontmatter + prompt) e a **matriz de permissões** (allow/ask/deny com glob por comando). Modelar o IDE como um *cliente* de um core separado, permitindo depois headless/CLI/Web sem reescrever o engine. Incluir `external_directory` como fronteira de segurança explícita e `doom_loop` como recuperação.

- **ORIGINAL IMPLEMENTATION**
  Monorepo TS (Bun) com `packages/opencode` (core/servidor), `packages/sdk-js`, `packages/console/app` (TUI), `packages/web` (lander/docs), `packages/slack`, launcher Go com auto-update, `sdks/vscode`. Config `opencode.json` + `~/.config/opencode/agents/*.md`.

---

### SOURCE: Codex CLI (`openai/codex`)

- **WHAT IT SOLVES**
  Um agente de codificação **leve, local e de baixo overhead** que roda no terminal, com foco em **sandboxing e safety por padrão** (execução de comandos com rede desabilitada e aprovação), e um **app-server** que permite separar o processo de UI do processo de execução (inclusive em máquinas/OS diferentes).

- **HOW IT WORKS**
  Core em **Rust** (rewrite `codex-rs`) com **crate por responsabilidade** (`codex-core`, `codex-tui`, `codex-mcp`, `codex-app-server`, `codex-app-server-protocol`, `codex-utils`, etc.). Arquitetura **app-server + exec-server**: o app-server fala JSON-RPC (protocolo v2 versionado, tipos gerados com `#[ts]`/ts_rs) e delega execução; podem rodar em hosts distintos. O loop do agente é por **turnos** (`Op::UserTurn`) sobre eventos SSE (`ev_response_created`, `ev_function_call`, `ev_completed`). Sandbox via **Seatbelt** (`sandbox-exec` no macOS) e `CODEX_SANDBOX_NETWORK_DISABLED` (rede desligada por padrão na tool `shell`). Contexto com **montagem incremental** (sem reescrita de histórico → cache-friendly), itens limitados e com hard cap (<10k tokens), fragmentos tipados implementando `ContextualUserFragment`. MCP via `mcp_connection_manager.rs`. Config em `config.toml`/`requirements.toml` (camadas gerenciadas por admins) + lifecycle hooks. `AGENTS.md` como convenção de instruções do repo.

- **WHY IT WORKS**
  Rust dá performance e segurança de memória para um agente local; a separação app-server/exec-server habilita execução remota segura; a política de contexto *incremental + boundado* preserva cache de prompt (custo/latência) e evita estouro; o sandbox com rede desabilitada por padrão reduz o blast radius de comandos não supervisionados.

- **STRENGTHS**
  - Modelo de **contexto incremental e boundado** (explícito, com hard caps e tipagem).
  - Separação app-server/exec-server → execução remota/cross-OS.
  - Sandboxing por padrão (Seatbelt + rede off + política de aprovação).
  - Protocolo versionado (v2) com código gerado de tipo e gating experimental.
  - Disciplina de crate pequeno (anti-bloat do `codex-core`) e snapshot tests (`insta`) para UI.

- **WEAKNESSES**
  - Orientado primariamente ao ecossistema OpenAI (embora aberto); não é agnóstico de provider como OpenCode/Cline.
  - Complexidade de build (Bazel + Cargo + Python) alta para contribuir.
  - TUI em ratatui exige disciplina extra de manutenção.

- **PATTERN**
  **"Policy-first sandboxed agent" + "Versioned server protocol" + "Incremental bounded context"**: segurança e determinismo como camadas arquiteturais de primeira classe; UI e execução desacopladas por um protocolo RPC versionado e tipado; contexto tratado como recurso com orçamento.

- **COSCA INTERPRETATION**
  Tratar **contexto como recurso orçado** (incremental, cacheável, com cap e tipagem de fragmentos) — decisão central para custo e qualidade. Separar **UI de execução** num protocolo interno tipado (permite headless, remoto, e futuras superfícies). Sandboxing por padrão para comandos (rede desligada salvo opt-in) como baseline de segurança do COSCA.

- **ORIGINAL IMPLEMENTATION**
  `codex-rs/` (Rust: `core/`, `tui/`, `mcp/`, `app-server/`, `app-server-protocol/`), `codex-cli/` (Python legado), `sdk/` (TypeScript), `tools/`, `docs/`. Build Bazel+Cargo; `just` como task runner; testes de integração via `TestCodexBuilder`/`ResponseMock` com SSE montado.

---

### SOURCE: Claude Code (`anthropics/claude-code`)

- **WHAT IT SOLVES**
  Agente de codificação **terminal-first** com forte **compreensão de repositório** e workflow git, priorizando autonomia controlada (executa tarefas de rotina, explica código, gerencia git) por linguagem natural. É o benchmark de UX/produto do segmento, ainda que o motor seja proprietário.

- **HOW IT WORKS**
  CLI no terminal com agente que mantém **memória de projeto** (`CLAUDE.md`/`AGENTS.md`), constrói um **repo-map** para entender o código, e opera por turnos de tool-use (leitura, edição, execução de comandos, MCP). Sistema de **permissions** com aprovação por ação; **plan mode** separa planejamento de execução; **hooks** e **commands** (`/` commands) estendem o fluxo; **subagents** delegam tarefas; **plugins** (diretório `plugins/`) adicionam comandos e agentes. Este repositório é majoritariamente docs/CLI + plugins/commands/scripts — o core é fechado.

- **WHY IT WORKS**
  A combinação de memória persistente + repo-map dá contexto de alta qualidade por baixo custo; aprovação granular mantém confiança sem travar o fluxo; git-first reduz fricção no ciclo real do dev; o modelo mental de "agente que vive no terminal" é simples e poderoso.

- **STRENGTHS**
  - Melhor-in-class em **compreensão de repositório** (repo-map) e **workflow git**.
  - Memória de projeto persistente (`CLAUDE.md`/`AGENTS.md`).
  - Extensibilidade por plugins/commands/subagents/hooks.
  - Data/telemetria com salvaguardas de privacidade explícitas.

- **WEAKNESSES**
  - Core proprietário — o repo não expõe o engine; pouco material arquitetural concreto.
  - Menos agnóstico de modelo que OpenCode/Cline.
  - Dependência forte de UX de terminal (não é IDE-first).

- **PATTERN**
  **"Repo-aware memory + tool-use loop"**: contexto persistente e indexado do repositório (repo-map) como diferencial, com plano/execução separados e extensibilidade via hooks/commands/subagents.

- **COSCA INTERPRETATION**
  Investir em **compreensão de repositório de alta qualidade** (mapa/índice próprio, não só grep) e em **memória de projeto versionável** (arquivo de convenções equivalente ao `AGENTS.md`/`CLAUDE.md`), lidos automaticamente. Adotar separação **plan/act** e ciclo git-first.

- **ORIGINAL IMPLEMENTATION**
  Repo com `plugins/`, `.claude/commands/`, `examples/`, `scripts/`, `Script/`, docs e CHANGELOG. Instalação via script/brew/winget; package npm `@anthropic-ai/claude-code` (deprecated).

---

### SOURCE: Cline (`cline/cline`)

- **WHAT IT SOLVES**
  Um agente de codificação **autônomo** exposto como **SDK, extensão de IDE, CLI e board multi-agente (Kanban)**, com o mesmo *engine* compartilhado. Foco em autonomia real (auto-approve), execução de comandos em tempo real e coordenação de múltiplos agentes em paralelo.

- **HOW IT WORKS**
  Monorepo com **engine compartilhado** (`@cline/sdk`) usado por CLI, extensão VSCode, plugin JetBrains e Kanban (web). Loop do agente com **Plan/Act modes**, **aprovação human-in-the-loop** (com auto-approve), **checkpoints** para undo, edições como **diff** revisável, e monitoramento contínuo de erros de linter/compilador com auto-correção. Executa **bash** no terminal do usuário e reage a output em tempo real (incl. processos longos). Extensão via **plugins** (registrar tools + lifecycle hooks) e **MCP**. **Multi-agent teams**: um coordenador quebra o trabalho e delega a agentes especialistas com tools/contexto próprios, com estado persistente entre sessões. **Kanban**: muitos agentes paralelos, cada card com **worktree próprio, auto-commit e cadeias de dependência**. **Scheduled agents** (cron), **connectors** (Slack/Telegram/Discord), **headless** com stream de eventos JSON (`--json`, eventos `agent_event`).

- **WHY IT WORKS**
  Um engine → muitas superfícies; Plan/Act + checkpoints + diffs dão segurança para autonomia; execução de comandos *no terminal real* com reação a output fecha o loop dev (teste/erro) sem abstração artificial; o board Kanban + worktrees paraleliza o trabalho sem colidir no mesmo tree.

- **STRENGTHS**
  - **SDK-first**: o agente é uma biblioteca, não um app (multi-superfície real).
  - Multi-agent com **worktrees isolados + auto-commit + dependências**.
  - Checkpoints/diff para undo e revisão.
  - Integração com linter/compilador (auto-fix de erros observados).
  - Headless JSON (CI/CD) + scheduled + connectors.

- **WEAKNESSES**
  - Complexidade de superfície muito alta (CLI+IDE+SDK+Kanban+JetBrains) — difícil manter paridade.
  - Autonomia total (auto-approve) exige confiança e pode ser arriscada.
  - Worktrees/auto-commit introduzem fricção de git (branch por agente).

- **PATTERN**
  **"Agent as SDK + shared engine"** com **"Isolated parallel execution"** (worktree por agente), **"Plan/Act with checkpointed diff"** e **"Observability-driven fixing"** (reage a linter/compilador/processos em tempo real).

- **COSCA INTERPRETATION**
  Projetar o COSCA como **SDK/core reutilizável** (o IDE é apenas o cliente principal). Adotar **Plan/Act** + **checkpoints/diff reversível** como UX de confiança. Considerar **workspaces isolados por agente** (worktree/sandbox por task) para paralelismo seguro e **reação a sinais do build/compilador** como loop de qualidade.

- **ORIGINAL IMPLEMENTATION**
  Monorepo TS: `apps/cli`, `sdk`, extensão VSCode (raiz), JetBrains (fechado), `cline/kanban` (repo separado), `docs/`, `evals/`, `patches/`. `package.json`/biome/vitest; regras em `.clinerules`; skills em `.cline/skills`.

---

### SOURCE: Continue (`continuedev/continue`)

- **WHAT IT SOLVES**
  Agente de codificação **open source e extensível**, multiplataforma (CLI, VS Code, JetBrains), com forte **integração de contexto/indexação** do codebase e **regras/config declarativas**, atuando como "IDE-first" com chat embutido no editor.

- **HOW IT WORKS**
  Arquitetura **core/gui/extensions**: `core` (TypeScript) contém o motor; `gui` é a UI; `extensions/` (vscode, intellij, cli) são os adaptadores; `binary` e `sync` complementam. **Context providers** alimentam o prompt via `@` mentions (arquivos, codebase indexado, terminal, etc.); **indexação** (`@codebase`) para busca semântica; **rules** globais/por-projeto; **MCP** e **hub de blocos** para extensão; config em `config.yaml`/`config.ts` com JSON Schema. **Edit mode** e **apply mode** para aplicação de mudanças. Projeto encerrado (read-only) com release final 2.0.0 (telemetria e auth removidas).

- **WHY IT WORKS**
  A separação core/gui/extensions permitiu reutilizar o mesmo motor em 3 IDEs + CLI; context providers modulares tornam o contexto composto e configurável; regras declarativas alinham o agente à convenção do time; a remoção de telemetria/auth no release final preservou utilidade sem dependência de serviços.

- **STRENGTHS**
  - Separação limpa **core/gui/extensions** (portabilidade real).
  - **Context providers** modulares + `@codebase` (indexação).
  - Rules/config declarativas com schema.
  - Histórico relevante (indexação/contexto) mesmo que o projeto tenha encerrado.

- **WEAKNESSES**
  - **Não mais mantido** (read-only) — como referência histórica.
  - Indexação local pode ser pesada para codebases grandes.
  - Menos foco em execução autônoma de comandos que Cline/Codex.

- **PATTERN**
  **"Portable core + pluggable context providers"**: o motor é portável entre superfícies e o contexto é uma **composição de provedores** configuráveis (não um blob fixo), com indexação do codebase como feature de primeiro nível.

- **COSCA INTERPRETATION**
  Modelar o contexto do COSCA como **composição de providers** (arquivos, símbolos, índice do repo, terminal, docs) e tratar **indexação do codebase** como subsistema separado. Manter **core desacoplado de UI** para permitir múltiplas superfícies sem reescrever.

- **ORIGINAL IMPLEMENTATION**
  Monorepo TS: `core/`, `gui/`, `extensions/{vscode,intellij,cli}`, `binary/`, `sync/`, `actions/`, `packages/`, `docs/`, `eval/`, `skills/`, `manual-testing-sandbox/`. Config `config.yaml`/`config.ts`.

---

### SOURCE: Roo Code (`RooCodeInc/Roo-Code` + `Roo-Code-Docs`)

- **WHAT IT SOLVES**
  Um **"dev team de agentes"** dentro do editor, via **modos de agente** especializados (Code, Architect, Ask, Debug, Custom) e **delegação/orquestração** de sub-tarefas. Origem: fork do Cline, com foco em **papéis (roles) configuráveis** e orquestração multi-agente.

- **HOW IT WORKS**
  Extensão VSCode (fork Cline) com `webview-ui` (React) + `src` (extension host) + `packages`. O conceito central são os **Modes** (`.roomodes`): cada modo é um papel com prompt, tools, contexto e regras próprios; o usuário alterna de papel conforme a tarefa. Suporta **custom modes**, **rules** (`.roorules`), **skills**, **MCP servers**, `.rooignore` (exclusão de contexto), delegação de tarefas entre agentes. Documentação em Docusaurus (`Roo-Code-Docs`). Arquivado em 15/mai/2026 (comunidade seguiu para ZooCode; origem retorna ao Cline).

- **WHY IT WORKS**
  Papéis especializados (modes) concentram o contexto/prompt certo por tarefa, reduzindo o desperdício de contexto e melhorando a qualidade; custom modes democratizam a criação de papéis por time; a herança do Cline dá base sólida de ferramentas e aprovação.

- **STRENGTHS**
  - **Modos como papéis** (architect/ask/debug/custom) — orquestração por persona.
  - Custom modes declarativos + rules + skills + `.rooignore`.
  - Forte documentação conceitual (Docusaurus).
  - Modelo mental de "equipe" dentro de um editor.

- **WEAKNESSES**
  - **Arquivado** — sem manutenção; risco de forks divergentes.
  - Acoplado à VSCode (webview) — menos portável que Cline/OpenCode.
  - Multi-agente por "troca de papel" é mais sequencial que o paralelismo real do Cline/Kanban.

- **PATTERN**
  **"Role/Mode-based agent"**: papéis como unidades de configuração (prompt+tools+contexto+rules) que o usuário alterna, com delegação de subtarefas — orquestração por *persona* em vez de por processo.

- **COSCA INTERPRETATION**
  Adotar **modos/papéis configuráveis** como camada de UX sobre o mesmo engine (build/plan/review/debug), com **custom modes declarativos** para times. Combinar com a delegação por subagentes do OpenCode (melhor dos dois: persona + subagente).

- **ORIGINAL IMPLEMENTATION**
  Extensão VSCode TS: `src/`, `webview-ui/` (React), `packages/`, `locales/`, `schemas/`, `apps/`; config `.roomodes`, `.roorules`, `.rooignore`. Docs: `Roo-Code-Docs` (Docusaurus, `docs/`, `src/`).

---

## MATRIZ COMPARATIVA

| DIMENSÃO | OpenCode | Codex CLI | Claude Code | Cline | Continue | Roo Code | **COSCA DESIGN** |
|---|---|---|---|---|---|---|---|
| **AGENT LOOP** | Turnos com agentes primary/subagent; `task` para paralelo; `steps` (max iterações) | Turnos por `Op::UserTurn` sobre SSE (`ev_response_created/function_call/completed`) | Loop tool-use terminal-first com plan/act | Plan/Act + auto-approve; monitora build/compilador em tempo real | Chat + edit/apply mode | Modos/papéis com delegação | **Plan/Act separados + loop de turnos com orçamento (`steps`) e sinais de build no loop** |
| **TOOL SYSTEM** | Tools nativas + MCP + LSP + custom tools + skills + ACP | Tools nativas + MCP (`mcp_connection_manager`) | Tools nativas + MCP + hooks + plugins | Tools nativas + MCP + plugins (registrar tools) | Tools + MCP + hub de blocos | Tools (herdado Cline) + MCP + skills | **Registry de tools unificado (nativas + MCP + LSP + custom/plugins), com descrição auto-gerada para o modelo** |
| **CONTEXT MANAGEMENT** | Regras/arquivos + `@` mentions + subagentes isolados | **Incremental, boundado, tipado** (`ContextualUserFragment`, caps, cache-friendly) | Repo-map + memória persistente (`CLAUDE.md`/`AGENTS.md`) | Regras (`.clinerules`) + skills + contexto do editor | **Context providers compostos** + `@codebase` (indexação) | Rules + skills + `.rooignore` | **Contexto orçado e incremental (Codex) + providers compostos (Continue) + repo-map/memória (Claude) + ignore-file** |
| **APPROVAL/PERMISSIONS** | **allow/ask/deny com glob** por tool/comando/agente + `external_directory` | **Sandbox por padrão** (Seatbelt, rede off) + policy de aprovação | Aprovação granular por ação | Human-in-the-loop + auto-approve + checkpoints | Regras + config declarativa | Aprovação (herdado Cline) | **Matriz allow/ask/deny por tool×padrão (glob) + sandbox com rede off por padrão + fronteira de worktree** |
| **PATCH/DIFF** | `apply_patch` + diff | Edição de arquivos + patch no TUI | Edição de arquivos | **Diff revisável + checkpoints (undo)** + auto-fix de linter | Edit/apply mode | Edição (herdado Cline) | **Edição como diff reversível com checkpoints/snapshot, aplicado via apply_patch atômico** |
| **MULTI-AGENT** | Subagentes + sessões-filhas + `task` | Foco single-agent (turnos) | Subagents + delegation | **Coordinator + especialistas; Kanban com worktrees + auto-commit + dependências** | Menos foco | Modos/papéis + delegação de subtarefas | **Subagentes delegáveis + modos/papéis + workspaces isolados (worktree) por tarefa paralela** |
| **RECOVERY** | `doom_loop` (detecção de travamento) | Contexto incremental evita drift; snapshots | Aprovação + undo | **Checkpoints/undo** + reação a erros | — | — | **Checkpoints (undo) + detecção de loop (`doom_loop`) + reexecução/resume de sessão** |
| **UI/UX** | **Multi-cliente**: TUI + Web + desktop + IDE + SDK | TUI (ratatui) + app-server (desktop/IDE) | Terminal-first | CLI + VSCode + JetBrains + Kanban (web) | IDE-first (VS Code/JetBrains) + CLI | Editor (VSCode webview) | **IDE-first (COSCA STUDIO) + core headless (CLI/Web) — UI como cliente fino do engine** |

### Dimensões-chave para o COSCA (síntese)

1. **Core/SDK desacoplado da UI** (Cline + OpenCode + Continue): o IDE é um cliente; o engine é reutilizável para CLI/headless/CI.
2. **Contexto orçado e incremental** (Codex): caps, fragmentos tipados, cache-friendly — qualidade e custo.
3. **Permissions como matriz allow/ask/deny com glob** (OpenCode) + **sandbox/rede-off por padrão** (Codex).
4. **Plan/Act + diffs reversíveis com checkpoints** (Cline/Claude): confiança para autonomia.
5. **Multi-agente por subagentes + papéis + workspaces isolados** (OpenCode + Roo + Cline/Kanban): paralelismo sem colisão.
6. **Compreensão de repositório própria** (Claude repo-map) + **providers de contexto compostos** (Continue).
7. **Recuperação** (checkpoints + detecção de loop) e **observabilidade** (reação a linter/compilador).

---

## RESUMO EXECUTIVO

- **Padrão dominante**: separar um **engine-core reutilizável (SDK)** das superfícies (IDE/CLI/Web), expondo um protocolo tipado e orientado a eventos — visto em Cline, OpenCode e Codex.
- **Contexto é o diferencial**: Codex prova que contexto **incremental, boundado e tipado** preserva cache e custo; Claude mostra que **repo-map + memória persistente** eleva a qualidade; Continue generaliza como **providers compostos**.
- **Segurança como primeira classe**: OpenCode entrega a melhor **matriz de permissões** (allow/ask/deny + glob); Codex entrega o melhor **sandbox por padrão** (rede off + aprovação).
- **Confiança via reversibilidade**: Plan/Act + **diffs com checkpoints/undo** (Cline) são o caminho para permitir autonomia sem perder controle.
- **Multi-agente**: subagentes delegáveis (OpenCode) + modos/papéis (Roo) + **workspaces isolados por tarefa** (Cline Kanban) formam a combinação mais completa.

**Recomendação principal para o COSCA CODE**: construir um **core headless (SDK) orientado a eventos** com (a) **contexto orçado/composto e compreensão de repositório própria**, (b) **matriz de permissões allow/ask/deny + sandbox com rede off por padrão**, (c) **Plan/Act com edição por diff reversível (checkpoints)**, e (d) **subagentes + modos + workspaces isolados** — o IDE COSCA STUDIO sendo o cliente fino dessa fundação.
