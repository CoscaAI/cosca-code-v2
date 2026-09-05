# RESEARCH INDEX — COSCA CODE

> Mapa completo do conhecimento extraído dos repositórios de referência.
> Fonte primária: GitHub. Nenhum código copiado — apenas padrões, decisões arquiteturais e abstrações.

## Documentos de pesquisa

| Documento | Fontes | Síntese central |
|---|---|---|
| [editor-patterns.md](editor-patterns.md) | vscode, monaco, theia, zed, lapce | Editor = buffer imutável (rope) + Document/View/Provider + workspace backend isolado por RPC + extensibilidade dual (declarativa + WASM) |
| [agent-patterns.md](agent-patterns.md) | opencode, codex, claude-code, cline, continue, roo-code | Core headless orientado a eventos + contexto orçado + allow/ask/deny + sandbox + edição por diff reversível + multi-agente isolado |
| [mcp-patterns.md](mcp-patterns.md) | modelcontextprotocol + servers + inspector | Tool = contrato declarativo (name+schema) + protocolo stateless + segurança per-tool + progressive discovery |
| [search-patterns.md](search-patterns.md) | tree-sitter, ast-grep, ripgrep (+ go-tree-sitter) | Sintaxe viva (incremental) + busca multi-estratégia + semântica delegada a LSP |
| [debugging-patterns.md](debugging-patterns.md) | vscode-js-debug, debugpy, DAP | Host DAP universal + adapters por linguagem + capability negotiation + lazy waterfall |

---

## Os 12 padrões transversais (a espinha dorsal do COSCA CODE)

1. **Separação sintaxe × semântica** — parsing sintático barato/local/incremental (tree-sitter) + semântica cara/global/precisa (LSP/compilador). Nunca misturar.
2. **Core headless + clientes finos** — o motor (SDK/backend) é independente da UI; IDE/CLI/Web/CI são clientes. Um só núcleo, múltiplas superfícies.
3. **Buffer imutável (rope) + undo ilimitado** — o conteúdo do documento é imutável/versionado; edição gera nova versão; undo/diff/split são consequência natural.
4. **Document/View/Provider** — separar o conteúdo (modelo), a renderização (view) e a inteligência (provider: completion, hover, diagnostics).
5. **Workspace backend isolado por RPC tipado** — local/remoto/container/SSH viram a mesma arquitetura; a UI fala com um proxy, não com o filesystem.
6. **Extensibilidade dual** — contribuições declarativas (rápidas, integradas) + plugins sandbox (WASM) para terceiros. Segurança não é opcional.
7. **Tool = contrato declarativo** — `name` + `description` + `inputSchema` (JSON Schema). O schema é a fronteira de validação e a "API do LLM".
8. **Permissões allow/ask/deny + sandbox por padrão** — nenhum agente com acesso irrestrito; matriz de permissão por comando/tool/glob; execução em sandbox.
9. **Contexto orçado e composto** — nunca enviar o projeto inteiro; montar contexto incremental com ranking de relevância e caps.
10. **Edição por diff reversível (Plan/Act)** — toda alteração é previewable/diffable/reversible/attributable; checkpoints + undo habilitam autonomia sem perda de controle.
11. **Broker/Adapter sobre wire protocol** — LSP e DAP como contratos estáveis; cada linguagem contribui um adapter fino; o host é genérico e universal.
12. **Event bus + observabilidade nativa** — tudo emite eventos (workspace.opened, agent.tool_called, test.failed…); observabilidade é tecida no sistema, não um add-on.

---

## Matriz de referência (capability × fonte)

| CAPABILITY | VS Code | Zed | Theia | OpenCode | Codex | Cline | MCP | tree-sitter | ripgrep | ast-grep | DAP/debugpy | **COSCA DESIGN** |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Editor core | workbench DOM | GPUI/Rust | DI/inversify | — | — | — | — | — | — | — | — | **rope + view/provider + canvas próprio** |
| Extensibilidade | extension host | plugins Rust | contribuições | MCP + agents | — | SDK | client/server | gramáticas | — | rules YAML | — | **contribuições + WASM + gramáticas runtime** |
| LSP | cliente nativo | nativo | nativo | — | — | — | — | — | — | — | — | **cliente LSP universal + auto-detect** |
| Debugging | DAP host | — | — | — | — | — | — | — | — | — | DAP | **COSCA DEBUG ENGINE (host DAP + adapters)** |
| Agentes | copilot | agent runtime | — | agent loop | exec loop | IDE+CLI | — | — | — | — | — | **Agent Engine (planner/coder/debugger…)** |
| Tools | — | — | — | tools+MCP | sandbox tools | tools | tools/resources/prompts | — | — | — | — | **Tool Registry + adaptadores MCP** |
| Permissões | — | — | — | allow/ask/deny | approval+sandbox | approval | per-tool HITL | — | — | — | — | **matriz de permissão + sandbox + audit** |
| Contexto | — | — | — | context mgmt | orçado/caps | context | — | — | — | — | — | **Context Engine (ranking + orçamento)** |
| Parsing/estrutura | — | tree-sitter | — | — | — | — | — | incremental/error-tolerant | — | — | — | **tree-sitter embutido (Go)** |
| Busca | ripgrep-like | — | — | — | — | — | — | — | on-the-fly git-aware | AST search | — | **multi-estratégia (texto/reg/símbolo/AST/semântica)** |
| Providers/modelos | — | multi-provider | — | providers | — | — | — | — | — | — | — | **Provider Engine + Model Router + catálogo** |

---

## Decisões arquiteturais que o COSCA NÃO herda

- **NÃO** DOM/Electron (VS Code) — adota Wails/WebView próprio, mais leve.
- **NÃO** framework de DI externo (Theia/inversify) — DI próprio ou composição explícita.
- **NÃO** UI custom em Rust (Zed/GPUI) — Go no core + React/Vite no front, equilíbrio velocidade/ecossistema.
- **NÃO** grepar re-scan obrigatório como única busca — combina scan on-the-fly + índice seletivo.
- **NÃO** debugger artificial — host DAP universal sobre engines reais.
- **NÃO** agentes como prompts independentes — infraestrutura comum (capabilities/tools/context/budget/permissions/memory).
