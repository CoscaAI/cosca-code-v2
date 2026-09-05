# MCP (Model Context Protocol) — Extração de Padrões para o Sistema de Tools do COSCA CODE

> Pesquisa de arquitetura. Objetivo: extrair **padrões, protocolos, abstrações e modelos de dados** do ecossistema MCP para projetar o sistema de TOOLS próprio do COSCA CODE. **Não** copiar código/UI/branding. Interoperar com MCP como transport/contract, não como implementação.
>
> Fontes analisadas: especificação MCP (`modelcontextprotocol`), docs oficiais (`modelcontextprotocol.io`), servidores de referência (`modelcontextprotocol/servers`), e o Inspector (`modelcontextprotocol/inspector`).

---

## SOURCE: MCP Specification — `github.com/modelcontextprotocol/modelcontextprotocol`

Repo que contém a especificação formal, o schema (TypeScript primeiro, JSON Schema derivado), os SEPs (Specification Enhancement Proposals) e os docs oficiais. Versão de referência analisada: `2026-07-28`.

- **WHAT IT SOLVES**: Define um contrato wire-level único para conectar aplicações de IA a sistemas externos (dados, tools, workflows). Elimina o problema N×M de integração (cada app × cada tool) com um padrão "USB-C para IA": servidores expõem capacidades uma única vez, qualquer host as consome.
- **HOW IT WORKS**: Arquitetura **client/server em camadas**. **Camada de dados**: JSON-RPC 2.0 puro. **Camada de transporte**: abstrai comunicação (stdio e Streamable HTTP). Três participantes: **Host** (app de IA que coordena N clients), **Client** (1 conexão dedicada por servidor), **Server** (fornece contexto). Primitivas de servidor: **Tools** (ações), **Resources** (dados/contexto), **Prompts** (templates). Primitivas de cliente: **Elicitation** (pedir input ao usuário); Sampling/Logging estão *deprecated*. Descoberta via `server/discover`; listagem via `tools/list`, `resources/list`, `prompts/list`; execução via `tools/call`, `resources/read`, `prompts/get`.
- **WHY IT WORKS**: Separação estrita entre **protocolo de troca de contexto** (dados) e **mecânica de transporte**. O protocolo é **stateless**: cada request carrega em `_meta` a versão do protocolo + capacidades do cliente + identidade, então o servidor processa cada mensagem isoladamente (não há sessão implícita). Isso permite cache, escala horizontal e simplifica reconexão/restart. Todas as primitivas seguem o mesmo ciclo: **discover → list → get/call**.
- **STRENGTHS**: Schema-first (TypeScript → JSON Schema, dialeto default 2020-12); capability negotiation embutida; notificações de mudança opt-in (`subscriptions/listen`); paginação e caching (`ttlMs`/`cacheScope`) em toda listagem; erro em duas camadas (erro de protocolo vs. erro de execução de tool).
- **WEAKNESSES**: Dois "eras" de protocolo (legacy `initialize` vs. moderno `server/discover` per-request) geram complexidade de compat; tools/resources/prompts têm modelos de dados quase idênticos mas não unificados; nomes de tools colidem entre servidores (disambiguation é responsabilidade do cliente); `x-mcp-header` e mirroring HTTP adicionam superfície de validação sutil.
- **PATTERN**: **Primitiva declarativa + JSON Schema + capability flag + list/get/call + notification opt-in + caching TTL**. Tudo é descoberto dinamicamente; nada é hardcoded no cliente.
- **COSCA INTERPRETATION**: O COSCA terá seu PRÓPRIO modelo de tool (nativo do IDE, com handler tipado, permissões, UI de confirmação), mas o **contrato de tool** (nome, description, inputSchema, outputSchema, annotations) será um superset compatível com o schema MCP, permitindo importar/exportar tools MCP como "tool definitions" COSCA. O COSCA implementa um **MCP Client adapter** que traduz `tools/list`/`tools/call` para o registry nativo de tools do COSCA, e um **MCP Server adapter** que expõe tools nativas do COSCA como ferramentas MCP para outros hosts.
- **ORIGINAL IMPLEMENTATION**: `schema/2025-11-25/schema.ts` (schema TS canônico) → `schema.json`; `seps/` (governança de mudanças); `docs/` (Mintlify). JSON-RPC 2.0 como substrato, `_meta.io.modelcontextprotocol/*` como namespace de metadados.

---

## SOURCE: MCP Official Docs / Architecture — `modelcontextprotocol.io`

Conteúdo canônico: conceitos, especificação normativa (tools/resources/prompts/transports/auth) e guias (client best practices, security).

### Arquitetura e descoberta

- **WHAT IT SOLVES**: Define o modelo mental oficial de Host/Client/Server e o fluxo de descoberta/execução que os hosts de IA usam.
- **HOW IT WORKS**: `server/discover` retorna `supportedVersions`, `capabilities` e `serverInfo` com `ttlMs`/`cacheScope` (cacheável). Cada request subsequente carrega `_meta` com `protocolVersion`, `clientCapabilities`, `clientInfo`. Negociação de versão: servidor rejeita com `UnsupportedProtocolVersionError` listando versões suportadas; cliente re-tenta. Notificações: cliente abre stream longa `subscriptions/listen` com filtro (`toolsListChanged`, `resourceSubscriptions`); servidor responde `notifications/subscriptions/acknowledged` (com `subscriptionId`) e então emite `notifications/tools/list_changed` etc.
- **WHY IT WORKS**: Descoberta é *opcional* (o cliente pode disparar qualquer request e tratar erro de versão), mas barata (cacheável). Capabilities viajam *por request*, não por conexão — alinhado à estatelessness.
- **PATTERN**: **Discovery cacheável + capability por-request + versão negociada por erro + mudança notificada opt-in com subscriptionId correlacionado**.
- **COSCA INTERPRETATION**: O registry de tools do COSCA deve ser **reativo**: mantém um cache de definições com TTL, invalida por notificação `list_changed` (não por polling), e indexa tools por servidor de origem para disambiguation de nomes.
- **ORIGINAL IMPLEMENTATION**: `learn/architecture.md`, `server/discover`, `basic/patterns/subscriptions`, `basic/versioning`.

### Tools (primitiva de ação)

- **WHAT IT SOLVES**: Permite que LLMs invoquem ações (file ops, API calls, queries, computação) com contrato tipado e controlável.
- **HOW IT WORKS**: `tools/list` retorna array de **Tool**: `name` (único, 1–128 chars, `[A-Za-z0-9_.-]`), `title` (human-readable), `description`, `icons`, `inputSchema` (JSON Schema 2020-12), `outputSchema` (opcional), `annotations` (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`). `tools/call` com `{ name, arguments }` retorna `resultType: "complete"` + `content[]` (text/image/audio/resource_link/resource) + opcional `structuredContent` (validável contra `outputSchema`). **Erros**: erro de protocolo (JSON-RPC error, ex.: tool desconhecida `-32602`) vs. **erro de execução** (`isError: true` no resultado — feedback acionável para o modelo se auto-corrigir). **Stateful tools**: sem sessão no protocolo → servidor emite um *handle* opaco (ex.: `basket_id`) no resultado e o recebe como argumento no próximo call (o modelo carrega o handle; o servidor valida autorização a cada call).
- **WHY IT WORKS**: Descrição + schema são a *interface de programação do LLM* — a qualidade da descrição determina a taxa de seleção correta de tool. Dois canais de erro separam "bug de request" de "feedback recuperável". `outputSchema` + `structuredContent` permitem integração tipada em código e validação cliente-side. Annotations permitem ao host decidir políticas (confirmar destrutivas, cachear idempotentes) sem entender a semântica da tool.
- **STRENGTHS**: Contrato auto-documentado; tipos ricos; erros distinguíveis; annotations para política de segurança/UX.
- **WEAKNESSES**: Annotations são dicas *não-confiáveis* (hosts devem tratar como untrusted de servidores não confiáveis); `x-mcp-header` adiciona regras de validação delicadas; sem namespace global de nomes.
- **PATTERN**: **Tool = { nome estável, descrição densa, inputSchema JSON Schema, outputSchema opcional, annotations de política }**; resultado = **content[] multimídia + structuredContent tipado + isError**.
- **COSCA INTERPRETATION**: O "tool definition" nativo do COSCA será estruturalmente equivalente (name, description, inputSchema, outputSchema, annotations) mas com campos nativos extras: **permission policy** (requireConfirmation, allowlist), **handler tipado** (TypeScript-first, como Zod), e **categoria/ícone nativo**. `isError` vira exceção tipada `ToolExecutionError` no runtime COSCA (o modelo recebe a mensagem; o handler pode re-lançar).
- **ORIGINAL IMPLEMENTATION**: `specification/2026-07-28/server/tools.md`; ver servidores de referência abaixo.

### Resources (primitiva de contexto)

- **WHAT IT SOLVES**: Compartilhar dados que fornecem contexto ao modelo (arquivos, schemas, records) com identidade por **URI**.
- **HOW IT WORKS**: `resources/list` (paginado/cacheável), `resources/read` (pode retornar múltiplos contents), `resources/templates/list` (URI templates RFC 6570 com argumentos auto-completáveis). Content é text (`text`) ou binary (`blob`, base64). **Annotations**: `audience` (`user`/`assistant`), `priority` (0.0–1.0), `lastModified`. `https://` → cliente pode buscar direto; `file://` → deve sanitizar contra path traversal; `git://`; esquemas custom RFC3986. Subscriptions por-URI via `subscriptions/listen` + `notifications/resources/updated`.
- **WHY IT WORKS**: Separa **referência** (URI/lista leve) de **conteúdo** (read pesado), com prioridade/audiência guiando a janela de contexto do host. Templates transformam recursos dinâmicos (por argumento) sem enumerar tudo.
- **PATTERN**: **Contexto endereçável por URI + lista leve + read on-demand + annotations de prioridade/audiência + template parametrizado**.
- **COSCA INTERPRETATION**: O COSCA já tem um modelo nativo de "contexto" (arquivos do workspace, símbolos, buffers). O padrão a importar é **URI canônica + annotations de prioridade/audiência** como forma de rankear o que entra na janela de contexto — o COSCA exporá seus recursos nativos como `cosca://` e poderá consumir `file://`/`https://` de servidores MCP.
- **ORIGINAL IMPLEMENTATION**: `specification/2026-07-28/server/resources.md`.

### Prompts (primitiva de template)

- **WHAT IT SOLVES**: Templates reutilizáveis de mensagens/instruções, selecionados pelo *usuário* (ex.: slash commands), não pelo modelo.
- **HOW IT WORKS**: `prompts/list` + `prompts/get` (com `arguments`). Prompt = `name`, `title`, `description`, `arguments[]`, e `messages[]` (`role` user/assistant + content text/image/audio/resource).
- **WHY IT WORKS**: Diferencia o *momento de decisão*: tools são model-controlled, resources são application-driven, prompts são user-controlled. Isso evita que o modelo invoque templates fora do fluxo do usuário.
- **PATTERN**: **Três modos de interação distintos por primitiva** (model/application/user controlled) com conteúdo de prompt = lista de mensagens multimídia.
- **COSCA INTERPRETATION**: O COSCA mapeia prompts para **slash commands / snippets nativos**. O valor de padrão é o modelo `role + content[]` para mensagens, e a separação clara de "quem decide quando disparar".
- **ORIGINAL IMPLEMENTATION**: `specification/2026-07-28/server/prompts.md`.

### Transports

#### stdio
- **WHAT/HOW**: Cliente lança o servidor como subprocesso; JSON-RPC newline-delimited em stdin/stdout; stderr só para log. Cancellation via `notifications/cancelled` (referenciando id). Shutdown: fechar stdin → servidor sai. Restart on crash (protocolo stateless permite re-executar requests perdidos).
- **WHY IT WORKS**: Zero rede, zero overhead, isolamento por processo; adequado a tools locais. Framing trivial (1 linha = 1 mensagem).
- **PATTERN**: **Subprocesso com framing newline-JSON-RPC + canais separados (dados vs. log) + restart stateless**.
- **COSCA**: Modelo de execução de tools locais do COSCA (spawn por process, log separado, restart transparente). Este é o modelo de confiabilidade a replicar para o runner nativo.

#### Streamable HTTP
- **WHAT/HOW**: Um único endpoint POST. Cliente envia cada JSON-RPC como POST próprio com headers espelhados (`MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name`, `Mcp-Param-*`). Servidor responde com JSON único OU stream SSE escopada ao request (progress/notificações → resposta final). Interações server→client (elicitation, sampling) são embutidas em `InputRequiredResult` via **MRTR** (Multi Round-Trip Requests), não como requests independentes. Streams longas de mudança via `subscriptions/listen`. Segurança: validar `Origin` (anti DNS-rebinding), bind localhost, `X-Accel-Buffering: no`, keep-alive SSE.
- **WHY IT WORKS**: Uma conexão transporta muitos clientes; streaming dá progresso em tempo real sem abrir canais paralelos; headers espelhados deixam proxies/WAFs rotear sem parsear o body.
- **PATTERN**: **POST-per-request + SSE de resposta escopada + server→client embutido no resultado (MRTR) + header mirroring para intermediários**.
- **COSCA**: O COSCA rodará tools nativas em-processo/por-worker, mas seu **remote provider** usará este modelo: um endpoint, resposta JSON ou stream de progresso, e input do usuário resolvido em round-trips (o COSCA já terá UI de confirmação nativa para isso).
- **ORIGINAL IMPLEMENTATION**: `specification/2026-07-28/basic/transports/streamable-http.md`.

### Authorization & Security

- **WHAT IT SOLVES**: Autorização de servidores HTTP remotos (OAuth 2.1 subset) + catálogo de vetores de ataque e mitigações para hosts/servidores.
- **HOW IT WORKS**:
  - **Auth** (opcional, só HTTP): servidor MCP = resource server OAuth 2.1; cliente = OAuth client. Descoberta do authorization server via **RFC 9728 Protected Resource Metadata** + metadata RFC 8414 / OIDC. Client registration via **Client ID Metadata Documents (CIMD)**, ou RFC 7591 (deprecated), ou pre-registration. Flow: 401 com `WWW-Authenticate` → metadata → PKCE + `resource` (RFC 8707) → authorization code + `iss` (RFC 9207) → token. **Scope minimization**: começar com escopos mínimos; elevar via `WWW-Authenticate: Bearer error="insufficient_scope" scope="..."` (step-up authorization, união de escopos client-side). Token deve ser enviado em `Authorization: Bearer` em **todo** request; validar audience.
  - **Security**: confused deputy (proxy com static client id), token passthrough (proibido), SSRF (validar HTTPS/IP privado/egress proxy), state handle hijacking (handle ≠ autenticação; bindar `<user_id>:<handle>`), local server compromise (consent + sandbox), OAuth URL validation (rejeitar `javascript:`/shell), stdio proxy escalation, mix-up (`iss` binding), scope minimization.
- **WHY IT WORKS**: Reutiliza primitivas OAuth consolidadas (não reinventa) e empurra o princípio de least-privilege para o fluxo de escopos incremental.
- **STRENGTHS**: Modelo de confiança claro (server não confia no cliente; cliente não confia em annotations/URLs de servidor); escopos progressivos reduzem blast radius.
- **WEAKNESSES**: Complexidade real de implementação (descoberta, CIMD, PKCE, iss, step-up) é alta para um "simples tool host".
- **PATTERN**: **Autorização por-request (Bearer em todo request) + escopos mínimos progressivos + validação de audience + never trust server-provided URLs/annotations + human-in-the-loop em ações destrutivas**.
- **COSCA INTERPRETATION**: O COSCA terá seu PRÓPRIO modelo de permissão nativo (per-tool `requireConfirmation`, allowlist por workspace, sandbox de execução), mas adotará: (1) escopos mínimos progressivos; (2) token com audience binding (nunca passthrough); (3) consent UI obrigatório antes de conectar server MCP local (mostrando comando exato, com destaque de padrões perigosos); (4) validação estrita de URLs e IPs (anti-SSRF) ao falar com servidores remotos; (5) handle de estado opaco + bindado a identidade.
- **ORIGINAL IMPLEMENTATION**: `specification/2026-07-28/basic/authorization/*`, `docs/.../security_best_practices.md`.

### Client Best Practices (escala)

- **WHAT IT SOLVES**: Degradação quando um host acumula centenas/milhares de tools (definições consomem a janela de contexto; resultados intermediários inflam tokens).
- **HOW IT WORKS**: **Progressive discovery** (3 camadas: catalog `search_tools` → inspect `get_tool_details` → execute) e **Programmatic tool calling / code mode** (modelo escreve código em sandbox que chama tools via stubs interceptados pelo host broker; só o resultado final volta ao contexto). Caching: respeitar `ttlMs`/`cacheScope`, invalidar em `list_changed`, preservar prompt-cache (append após breakpoint; ou rotear tudo por um único meta-tool estável `call_tool({name,args})`).
- **WHY IT WORKS**: Separa "o que está disponível" (catálogo barato) de "o que está carregado" (definição completa), e move a composição de chamadas para fora do contexto do modelo.
- **PATTERN**: **Catalog → Inspect → Execute** (índice leve → detalhe sob demanda → execução), e **host-broker sandbox** para composição sem passar dados pelo modelo.
- **COSCA INTERPRETATION**: Recurso central. O COSCA terá um **tool index** nativo (search semântico + filtro por permission) que carrega definições sob demanda, e um **COSCA Script/Runner sandbox** que compõe tools nativamente — o modelo escreve código contra a API tipada gerada das tools, sem passar resultados intermediários pelo contexto.
- **ORIGINAL IMPLEMENTATION**: `docs/.../clients/client-best-practices.md`.

---

## SOURCE: Reference Servers — `github.com/modelcontextprotocol/servers`

Coleção de servidores de referência mantidos pelo steering group (filesystem, git, memory, sequentialthinking, time, fetch, everything). Demonstram o padrão de modelagem de tool em SDKs reais (TS com Zod; Python com Pydantic).

- **WHAT IT SOLVES**: Mostra como uma tool é *concretamente* modelada e como handlers expõem dados/ações a um LLM de forma segura.
- **HOW IT WORKS**: Cada servidor instancia `McpServer({ name, version })`, registra tools com `registerTool(name, { title, description, inputSchema, outputSchema, annotations }, handler)`, expõe resources e conecta via `StdioServerTransport`. Schemas tipados: **Zod** (TS) / **Pydantic** (Python) → JSON Schema. Handlers retornam `{ content: [...], structuredContent }`.
- **WHY IT WORKS**: **Schema é a fronteira de confiança**: validação de input acontece antes do handler; `validatePath`/`validate_repo_path` impõem allowlist de diretório/repo em *todo* call (defense-in-depth). `structuredContent` carrega a payload tipada; `content` carrega o texto legível. `notifyGraphUpdated()` propaga mudança aos assinantes do resource.
- **STRENGTHS**: Nomenclatura consistente (`git_status`, `read_text_file`, `create_entities`); annotations de política em todas as tools; descrições longas e instrucionais (a descrição é a documentação do LLM); defesa contra flag injection (`-` prefix) e path traversal explícitas.
- **WEAKNESSES**: Alto boilerplate por tool; duplicação entre `content` e `structuredContent`; estado persistido em arquivo JSONL (memory) é simples mas não escala; sem versionamento de schema entre versões de tool.
- **PATTERN** (síntese dos 4 servidores analisados):

| Servidor | Modelagem de tool | Padrão-chave |
|---|---|---|
| **filesystem** | ~14 tools granulares (`read_text_file`, `write_file`, `edit_file`, `search_files`, `directory_tree`…) | **Granularidade fina + allowlist de diretórios + `readOnlyHint`/`destructiveHint`/`idempotentHint` para política + `outputSchema` sempre presente** |
| **git** | 12 tools mapeando verbos do git; Pydantic `BaseModel` → `model_json_schema()` | **1 tool = 1 verbo de domínio + validação de repo_path em todo call + defesa anti flag-injection** |
| **memory** | 8 tools sobre um knowledge graph (entities/relations/observations) + 1 resource `memory://knowledge-graph` com subscribe | **Modelo de dados como grafo + resource espelho + notificação de mudança (`resources/updated`)** |
| **sequentialthinking** | 1 única tool com ~10 params (`thought`, `thoughtNumber`, `nextThoughtNeeded`, `isRevision`, `revisesThought`, `branchFromThought`…) | **Tool "de processo" que mantém estado via argumentos ricos, não via sessão — o modelo conduz a máquina de estados** |

- **COSCA INTERPRETATION**: O COSCA adotará o padrão **Zod/Pydantic → JSON Schema como fonte única de verdade do input de tool**, com handler tipado e validado. As tools nativas do COSCA serão granulares e verbo-nomeadas (`file.read`, `file.write`, `git.status`), cada uma com annotations de política mapeadas para o permission model nativo. O padrão "1 tool = 1 verbo + allowlist + defense-in-depth" é a base do runner seguro do COSCA.
- **ORIGINAL IMPLEMENTATION**: `src/filesystem/index.ts`, `src/memory/index.ts`, `src/sequentialthinking/index.ts`, `src/git/src/mcp_server_git/server.py`.

---

## SOURCE: MCP Inspector — `github.com/modelcontextprotocol/inspector`

Ferramenta de teste/debug de servidores MCP, v2. Uma base de código, três frontends (Web Vite+React+Mantine, CLI, TUI Ink) sobre um core compartilhado `@inspector/core`.

- **WHAT IT SOLVES**: Inspecionar, testar e depurar servidores MCP de forma interativa e scriptável, cobrindo os dois "eras" do protocolo e features de ponta (MRTR, tasks, OAuth, MCP Apps).
- **HOW IT WORKS**: `InspectorClient` (`core/mcp/`) é dono da conexão, do lifecycle request/response e de **state stores**; `core/react/` expõe hooks React sobre essas stores para Web e TUI (UI "dumb components" + Storybook). `core/auth/` fatora OAuth em lógica isomórfica + backends (browser/node/remote). Servidores de teste **composables** (presets de tools/resources/prompts/tasks) exercitados in-process e como subprocesso stdio. Cobre protocol eras (legacy stateful vs. modern stateless `server/discover`), paginação, `structuredContent`, header mirroring, erros taxonômicos (`-32020`/`-32021`/`-32022`/`-32601`), MRTR, tasks extension, subscriptions.
- **WHY IT WORKS**: **Estado como stores observáveis** desacopla o runtime do protocolo das UIs; o mesmo `core` garante comportamento idêntico em Web/CLI/TUI. Servidores de teste reais (não mocks) validam o protocolo de ponta a ponta. Quality gate agressivo (coverage ≥90% por arquivo, typecheck coverage, format coverage, smoke e2e).
- **STRENGTHS**: Divisão core/UI exemplar; testes contra servidor real; cobertura de protocolo de borda (erros, paginação, duplicidade de nomes, headers).
- **WEAKNESSES**: Complexo (monorepo de 4 clients + core sem package.json); focado em MCP (não em features de IDE).
- **PATTERN**: **Core de protocolo agnóstico de UI + stores reativas + múltiplas skins (Web/CLI/TUI) + servidor de teste composable + taxaonomia de erros renderizada**.
- **COSCA INTERPRETATION**: Arquitetura interna do COSCA: um **Tool Runtime core** (conexão, request/response, stores de estado) agnóstico de UI, consumido pela UI do IDE e por um CLI headless (para automação/CI). Um **COSCA Tool Playground** (equivalente ao Inspector) para testar/debuggar tools nativas e MCP com painel de protocolo, erros tipados e replay de requests. Servidores/tools de teste composables para CI.
- **ORIGINAL IMPLEMENTATION**: `core/mcp/InspectorClient`, `core/react/`, `core/auth/`, `test-servers/src/preset-registry.ts`, `clients/{web,cli,tui}`.

---

## MATRIZ COMPARATIVA

| DIMENSÃO | MCP | COSCA DESIGN |
|---|---|---|
| **TOOL DEFINITION** | `{ name, title, description, icons, inputSchema (JSON Schema 2020-12), outputSchema, annotations (readOnly/destructive/idempotent/openWorld) }` | Superset nativo compatível: mesmos campos + `permission` (requireConfirmation/allowlist), `handler` tipado (Zod/TS), `category`/`icon`. `name` segue `[A-Za-z0-9_.-]`, namespaced por origem. |
| **TRANSPORT** | stdio (subprocess newline-JSON-RPC) e Streamable HTTP (POST-per-request + SSE escopada + MRTR + header mirroring) | Runner nativo em-processo/worker (rápido, isolado, log separado, restart stateless) + **MCP adapter** para stdio e Streamable HTTP. Remote provider usa endpoint único + stream de progresso. |
| **SECURITY / AUTHORIZATION** | OAuth 2.1 (Bearer por request, PKCE, `resource`, `iss`, escopos progressivos, audience binding); stdio usa creds de env; consent + sandbox para servers locais | Permission model nativo: per-tool confirm, allowlist por workspace, sandbox por processo, nunca token passthrough, escopos mínimos progressivos, validação anti-SSRF de URLs, human-in-the-loop em ações destrutivas. OAuth reutilizado para remoto. |
| **DISCOVERY** | `server/discover` (cacheável, TTL) + `tools/list` (paginado/cacheável) | **Tool Index** nativo reativo: cache de definições com TTL, invalidação por notificação de mudança, indexado por origem, com **Catalog→Inspect→Execute** (search semântico, carga sob demanda). |
| **CAPABILITY NEGOTIATION** | Por-request em `_meta` (`protocolVersion`, `clientCapabilities`, `clientInfo`) + `supportedVersions`/`capabilities` do discover + erro `UnsupportedProtocolVersionError` | Igual para o MCP adapter (compat full). Internamente, o COSCA usa **feature flags de tool** declaradas no registry (sem negociação wire), com fallback em runtime. |
| **RESOURCES / PROMPTS** | Resources endereçados por URI (list/read/templates + annotations priority/audience + subscribe) ; Prompts = templates user-controlled (list/get + messages) | Contexto nativo do COSCA (arquivos, símbolos, buffers) exposto como `cosca://` com annotations de prioridade/audiência; prompts = slash commands nativos; ambos interoperáveis via adapter MCP. |
| **ERROS** | Duas camadas: protocolo (JSON-RPC error) vs. execução (`isError: true`) | Exceções tipadas: `ToolExecutionError` (recuperável pelo modelo) vs. `ToolNotFoundError`/`InvalidParamsError` (protocolo). Taxonomia de erros renderizada no playground. |

---

## Resumo executivo

Os padrões mais valiosos extraídos do MCP para o sistema de tools do COSCA CODE:

1. **Tool = contrato declarativo** (`name` + `description` + `inputSchema`/`outputSchema` em JSON Schema + `annotations` de política), com o schema tipado (Zod/Pydantic) como fonte única de verdade e fronteira de validação.
2. **Protocolo stateless + descoberta cacheável**: capabilities por request, version negotiation por erro, e mudança de tool set notificada opt-in (`subscriptions/listen`) em vez de polling.
3. **Dois canais de erro** (protocolo vs. execução recuperável) e **resultado duplo** (`content` textual + `structuredContent` tipado) para permitir auto-correção do modelo e integração tipada.
4. **Segurança por princípio**: permissão per-tool com confirmação humana, allowlist de diretórios/repo, escopos mínimos progressivos, audience binding (nunca passthrough), handles de estado opacos bindados à identidade, e nunca confiar em annotations/URLs vindas de servidor.
5. **Escala via Catalog→Inspect→Execute** (progressive discovery) e **code mode** (sandbox com broker que compõe tools sem passar resultados intermediários pelo contexto).

**Recomendação**: o COSCA deve ter um **Tool Registry nativo** com modelo de tool compatível-superset do MCP (mesmo contrato de nome/descrição/schema/annotations, mais permissões e handler tipado), um **Tool Runtime core agnóstico de UI**, e **dois adaptadores MCP** (client + server) para interoperar sem reimplementar o protocolo. Assim o COSCA tem o melhor dos dois mundos: execução nativa, segura e integrada ao IDE, mantendo o ecossistema MCP como superfície de interop.
