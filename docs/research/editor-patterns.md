# Pesquisa de Padrões Arquiteturais — Editores/IDEs Open Source

> Fonte de pesquisa para o **COSCA CODE** (codename COSCA STUDIO).
> Objetivo: extrair PADRÕES, DECISÕES ARQUITETURAIS, ABSTRAÇÕES e ESTRATÉGIAS — não código, não UI, não branding.
> Data: 2026-08-13. Fontes: microsoft/vscode, microsoft/monaco-editor, eclipse-theia/theia, zed-industries/zed, lapce/lapce.

---

## Índice de conceitos transversais

Antes dos estudos de caso, as grandes famílias de abstração que se repetem nos cinco projetos:

1. **Separação Modelo/Visão/Provedor** — o conteúdo editável (model), a superfície renderizada (view/editor) e a inteligência de linguagem (providers/LSP) são entidades independentes com ciclo de vida próprio.
2. **Núcleo em camadas com ambientes de execução** — código particionado por capacidade de runtime (`common`/`browser`/`node`/`desktop`/`web`) para reutilizar lógica entre desktop e cloud.
3. **Contribuição declarativa (contribution points)** — funcionalidades se auto-registram no shell através de manifestos e entrypoints, sem o núcleo conhecer os contribuidores.
4. **Processo/sandbox isolado para extensões** — extensões rodam fora do processo da UI, comunicando-se via protocolo (extension host, WASI, workers, proxy RPC).
5. **Injeção de dependência por identificador** — serviços resolvidos por token/interface, permitindo override por ambiente e testes.
6. **Proxy/backend separado do frontend** — um processo intermediário que media filesystem, LSP, plugins e terminal, viabilizando remote development.
7. **Rendering GPU e estruturas de dados persistentes** — rope/árvores imutáveis + renderização direta para máxima latência.

---

### SOURCE: microsoft/vscode

- **WHAT IT SOLVES**: construir um IDE de propósito geral que seja ao mesmo tempo "simples como editor" e "capaz como IDE", distribuível em desktop (Electron) e web, com um ecossistema massivo de extensões de terceiros sem comprometer estabilidade/segurança do núcleo.

- **HOW IT WORKS**: o produto é um `core` em camadas (`base` → `platform` → `editor` → `workbench` → `code`/`server`) particionado por ambiente de runtime. A extensibilidade é garantida por dois caminhos: (a) extensões de processo isolado rodando no **extension host** (processo separado do renderer) expostas via uma API versionada (`vscode.d.ts`); (b) contribuições internas do workbench que se auto-registram via arquivos `*.contribution.ts`, sem que o workbench dependa delas. Tudo é costurado por **injeção de dependência por identificador** (serviços com `@IService` decorations resolvidos pelo `InstantiationService`, registrados via `registerSingleton` com instanciação lazy). O mesmo código alimenta desktop e web através de entrypoints distintos (`workbench.desktop.main.ts`, `workbench.web.main.ts`, `workbench.common.main.ts`). Recursos como search, SCM, debug, tasks, terminal e settings são todos "contributions" que se plugam no shell.

- **WHY IT WORKS**: a combinação de núcleo fino + contribuições plugáveis + extensões em processo separado resolve o problema fundamental de IDEs: *escala de funcionalidade sem acoplamento*. A DI por token permite substituir implementações por ambiente (desktop vs web) sem `if/else`. O modelo de contribuição permite que features sejam desenvolvidas de forma independente por equipes paralelas. O extension host isola crash/segurança e garante estabilidade da UI.

- **STRENGTHS**: ecossistema de extensões insuperável; camadas com regras de dependência rígidas (um contrib não pode depender do interior de outro); navegação trivial do código; API pública versionada e estável; mesmo código para desktop/web/remote.

- **WEAKNESSES**: performance limitada por JS/DOM (latência de input perceptível em arquivos grandes); complexidade colossal (163k commits); acoplamento implícito via "services" globais; extensões dependem de APIs proprietárias que não portam para outros editores; o núcleo monolítico dificulta fork/embed.

- **PATTERN**: **"Layered core + contribution points + out-of-process extension host + token-based DI"**. As abstrações centrais são: *service identifier* (token), *contribution* (registro declarativo), *target environment partition*, *extension host boundary*.

- **COSCA INTERPRETATION**: o COSCA pode adotar o *conceito* de "núcleo fino + contribuições declarativas" e "serviços resolvidos por token", mas sem o peso do legado. Interpretação original: um **registry de capabilities** onde cada feature (painel, comando, linguagem, provedor) se registra com um descritor declarativo; um **bus de comandos** unificado (command + contexto + keybinding) desacoplado da UI; e um **limite de extensibilidade baseado em protocolo** (ver Lapce/WASI e LSP), em vez de API proprietária. Isso preserva a força da extensibilidade mas evita o acoplamento ao legado da API do VS Code.

- **ORIGINAL IMPLEMENTATION**: implementação própria de: service registry e DI (sem `@IService` decorations), command palette/registry, contribution manifest loader, settings schema e cascata de configuração, keybinding resolver (modos + when-clauses), file watcher, search engine. Nada copiado; apenas os conceitos.

---

### SOURCE: microsoft/monaco-editor

- **WHAT IT SOLVES**: oferecer um editor de código embarcável no navegador, extraído do núcleo do VS Code, com todas as features de edição (syntax highlight, completion, diff, minimap, find) sem a "casca" do IDE.

- **HOW IT WORKS**: a arquitetura gira em torno de **quatro conceitos**:
  - **Model**: representa um arquivo aberto — conteúdo, linguagem, URI, histórico de edição. É independente de qualquer UI.
  - **URI**: identifica cada model num *virtual file system* lógico (`file://`, `inmemory://model/1`), permitindo que providers resolvam contexto (ex.: schema JSON) por URI.
  - **Editor**: a *visão* de um model anexada ao DOM; um model pode ter múltiplas visões (split/diff).
  - **Providers**: fornecem inteligência (completion, hover, actions) por model; frequentemente mapeiam para LSP.
  - **Disposables**: quase tudo implementa `dispose()` para liberar listeners/recursos de forma determinística.
  Linguagem services pesados (tokenization, parsing) rodam em **web workers** para não bloquear a UI thread. Há `monaco-lsp-client` para conectar a language servers via LSP. Tokenização usa gramáticas Monarch (simples, regex) com suporte a TextMate via libs externas.

- **WHY IT WORKS**: a separação estrita **Model (estado) / Editor (visão) / Providers (inteligência)** desacopla o ciclo de vida do documento do ciclo de vida da renderização. Isso permite splits, diffs, in-memory documents e múltiplos editores por modelo sem duplicação. Workers isolam trabalho pesado da thread de interação. `dispose()` torna o gerenciamento de recursos previsível (sem leaks de listener).

- **STRENGTHS**: embarcável e leve; API pública limpa (`monaco.d.ts`); modelo/visão/provedor altamente reutilizável; workers para offload; diff/minimap/decorations de alta qualidade.

- **WEAKNESSES**: escopo limitado ao editor (sem workbench/extensões nativas); gramáticas Monarch são mais fracas que tree-sitter/TextMate; sem suporte mobile; extensões do VS Code não funcionam; deprecou AMD.

- **PATTERN**: **"Document/View/Provider trinity + URI-addressed virtual FS + web-worker offload"**. A abstração central é o *text model* como fonte de verdade desacoplada, endereçada por URI, alimentada por providers assíncronos.

- **COSCA INTERPRETATION**: o COSCA deve adotar essa trindade como **pedra fundamental do editor core**: um `Buffer`/`Document` imutável endereçado por URI (fonte de verdade), uma `View` renderizada separada, e um `Provider`/`Adapter` para inteligência. Originalização: em vez de web workers, usar **threads/isolates nativos ou processos via RPC** (linguagem de sistema) com a mesma regra de "nunca bloquear a UI thread"; usar **tree-sitter** em vez de Monarch/TextMate para parsing incremental; decorations/diagnostics como *span sets* imutáveis aplicados à view, não ao documento.

- **ORIGINAL IMPLEMENTATION**: implementação própria de: text buffer com rope (edição O(log n)), sistema de decorations/diagnostics por span, diff engine, minimap renderer, find/replace incremental, completion/code-action pipeline conectado a LSP. Abstrações inspiradas, código próprio.

---

### SOURCE: eclipse-theia/theia

- **WHAT IT SOLVES**: um *framework* (não apenas um editor) para construir IDEs cloud e desktop personalizados, sob governança neutra, mantendo compatibilidade com extensões do VS Code.

- **HOW IT WORKS**: a Theia é organizada em **pacotes por funcionalidade** (`packages/`), cada um particionado por plataforma: `common/` (JS puro), `browser/` (DOM), `node/` (Node), `electron-*` (main/renderer específicos). Usa **inversão de controle via InversifyJS**: serviços registrados como contribuições (`@injectable`, `@inject`) e resolvidos num container global. O **frontend/backend split** é explícito: a UI roda no browser (ou Electron renderer) e o backend (filesystem, processos, terminal, plugins) roda no Node — conectados por JSON-RPC. O **plugin system** suporta dois modelos: extensões nativas Theia (contribuições DI no processo) e extensões compatíveis com a API do VS Code rodando num *plugin host* que emula a API. A **application shell** é um conjunto de contribuições de UI (menus, views, widgets) composto no bootstrap.

- **WHY IT WORKS**: a Theia separa *plataforma* (framework) de *produto* (blueprint), permitindo que qualquer empresa crie seu próprio IDE. A DI por InversifyJS torna cada feature um módulo substituível. O split frontend/backend é o que habilita "cloud IDE" de verdade (a UI pode estar em qualquer lugar; o backend no servidor). A camada de compatibilidade VS Code reduz o custo de adoção ao reusar o ecossistema de extensões.

- **STRENGTHS**: máximo de customização (é framework, não produto); cloud e desktop com a mesma base; dois modelos de extensão; governança vendor-neutral; contribuições como cidadãos de primeira classe.

- **WEAKNESSES**: complexidade de DI difícil de debugar; performance abaixo do VS Code (mais camadas de abstração); a compatibilidade VS Code é uma perseguição eterna (a API se move); dividida em muitos pacotes/repos (custo de manutenção alto); menos polida que produtos focados.

- **PATTERN**: **"Platform-as-framework: DI container + frontend/backend split por RPC + dual extension model + composable application shell"**. Abstração central: *inversão de controle onde todo recurso é uma contribuição injetável* e *o processo dividido por um contrato RPC*.

- **COSCA INTERPRETATION**: dois padrões valiosos para o COSCA: (1) **frontend/backend separados por um contrato RPC** — o editor UI é cliente, o "workspace service" (FS, LSP, terminal, search, git) é servidor, permitindo rodar local ou remoto sem mudar o código da UI; (2) **tudo é contribuição** — menus, views, comandos, providers registrados num container. Originalização: não usar um framework DI externo, mas um **registry de serviços próprio com resolução lazy e override por ambiente**, e um **protocolo de capabilities** (JSON-RPC/bincode) versionado e tipado (ver Lapce `lapce-rpc`) em vez de RPC genérico.

- **ORIGINAL IMPLEMENTATION**: implementação própria de: kernel DI/registry, application shell, protocolo RPC workspace↔frontend, plugin host (se adotado, com ABI própria, não a API VS Code), bootstrap de contribuições. Sem InversifyJS, sem packages Theia.

---

### SOURCE: zed-industries/zed

- **WHAT IT SOLVES**: provar que um editor pode ter latência próxima de zero ("code at the speed of thought"), colaboração multiplayer em tempo real e IA de primeira classe, sem o peso de Electron — escrito do zero em Rust.

- **HOW IT WORKS**: Zed é um workspace Cargo com dezenas de `crates` especializadas (`editor`, `project`, `collab`, `agent`, `dap`, `extension_host`, `git`, `terminal`, `diagnostics`, `file_finder`, `fuzzy`...). A fundação é a **GPUI** (`crates/gpui`), um framework de UI próprio, renderizado por GPU via **wgpu** (Metal/Vulkan/DirectX) com backends por plataforma (`gpui_macos`, `gpui_windows`, `gpui_linux`, `gpui_web`). O modelo de concorrência é async em **Tokio** combinado com um *deterministic main thread* para UI. Parsing é feito com **tree-sitter** (syntax trees incrementais). A **colaboração** é central: o crate `collab` + servidor (`collab` server via LiveKit) sincronizam o workspace entre pares. A **IA** é estrutural, não um add-on: `agent` (agentes), `copilot`/`edit_prediction` (completion inline + edição preditiva), `context_server` (MCP), e clientes para múltiplos provedores (`anthropic`, `openai`, `deepseek`, `google_ai`, `bedrock`, `cloud_llm_client`) — todos plugáveis.

- **WHY IT WORKS**: Rust + GPU rendering + estruturas de dados eficientes eliminam o gargalo que limita editores baseados em DOM/Electron. Tree-sitter fornece análise incremental que reusa trabalho entre keystrokes. A modularização em crates com responsabilidade única permite que cada subsistema (editor, projeto, git, AI) evolua com contratos claros. Fazer colaboração e IA "cidadãos de primeira classe" (não extensões) garante que sejam profundamente integrados e rápidos.

- **STRENGTHS**: latência e performance de ponta; rendering GPU próprio (GPUI); tree-sitter integrado; colaboração multiplayer nativa; AI profundamente integrada com agentes/MCP; crates bem separadas.

- **WEAKNESSES**: ecossistema de extensões ainda jovem (extension host com ABI própria, menos maduro); UI custom não tem a riqueza de componentes do DOM; alto custo de engenharia (framework próprio); sem modo remoto/cloud completo; ferramenta de UI proprietária limita contribuição externa.

- **PATTERN**: **"Native performance via GPU rendering + tree-sitter incremental parsing + workspace-as-collaborative-entity + AI/agents as first-class subsystems"**. Abstrações: *crate-per-subsystem com contrato*, *parsing incremental reutilizável*, *agent runtime* (agentes + ferramentas + MCP) integrado ao editor.

- **COSCA INTERPRETATION**: o COSCA deve absorver três decisões estratégicas do Zed: (1) **parsing incremental como serviço central** (tree-sitter) alimentando highlighting, outline, folding, refactoring — não apenas coloração; (2) **IA/agentes como subsistema estrutural** com runtime próprio de agentes + ferramentas + MCP, não uma extensão; (3) **separação rígida de subsistemas com contratos** (editor, projeto, git, search, terminal, agent) para evolução paralela. Originalização: não é obrigatório um framework GPU próprio; o COSCA pode usar rendering nativo/DOM/GPU existente, mas deve manter a *mesma meta de latência* (fila de input com prioridade, trabalho pesado fora da UI thread) e o *mesmo modelo mental* de agentes. IA plugável por provedor (múltiplos backends) com camada de abstração própria.

- **ORIGINAL IMPLEMENTATION**: implementação própria de: integração tree-sitter, agent runtime (planner/tools/MCP client), provider-agnostic LLM layer, project model, fuzzy finder, keymap. Não usar GPUI nem crates do Zed; apenas os conceitos de design.

---

### SOURCE: lapce/lapce

- **WHAT IT SOLVES**: um editor "lightning-fast" em Rust puro, com arquitetura que separa claramente frontend, backend e plugins para viabilizar remote development e extensibilidade segura (WASI).

- **HOW IT WORKS**: a arquitetura é dividida em crates:
  - **`lapce-core`**: estruturas de dados do editor baseadas em **Rope Science** (do Xi-Editor) — edição de texto O(log n) com operações imutáveis/persistentes.
  - **`lapce-app`**: frontend/GUI construído com **Floem** (framework UI Rust) renderizado via **wgpu**.
  - **`lapce-proxy`**: processo backend que media **filesystem, plugins e LSP** entre o frontend e o sistema — a chave para remote development (o proxy roda no host remoto).
  - **`lapce-rpc`**: o contrato RPC tipado que conecta app ↔ proxy.
  - **Plugins WASI**: extensões compiladas para WebAssembly (WASI) em C/Rust/AssemblyScript, isoladas e seguras por sandbox.
  O fluxo de edição: o `lapce-app` pede conteúdo ao `lapce-proxy` (que lê do disco), mantém o texto localmente, aplica mudanças de forma síncrona com o proxy, e persiste via proxy ao salvar. LSP embutido fornece completion/diagnostics/code actions. Terminal integrado e modal editing (Vim) são nativos.

- **WHY IT WORKS**: a **separação física app/proxy via RPC** transforma "remote development" numa propriedade da arquitetura (mude onde o proxy roda), não num caso especial. O **rope** garante edição eficiente e undo/redo baratos. **WASI** dá extensibilidade com segurança real (sandbox de verdade, sem processos arbitrários). Rust puro + wgpu dão performance nativa.

- **STRENGTHS**: arquitetura limpa e separada por contrato RPC; extensões WASI seguras e multi-linguagem; rope science para edição eficiente; remote development de primeira classe; modal editing.

- **WEAKNESSES**: ecossistema de plugins/linguagens muito menor; Floem ainda em amadurecimento; menos features de IDE (debug, refactor avançado) que os concorrentes; comunidade pequena; ritmo de desenvolvimento lento.

- **PATTERN**: **"App/Proxy split via typed RPC + WASI sandboxed extensions + rope-based immutable editing"**. Abstrações: *backend workspace isolável (proxy)*, *contrato RPC versionado (rpc crate)*, *plugins em sandbox WASM*, *text buffer imutável por rope*.

- **COSCA INTERPRETATION**: o padrão mais valioso do Lapce para o COSCA é a **"workspace backend como processo isolável atrás de um contrato RPC"** — o mesmo proxy resolve local, remoto (SSH) ou container sem tocar na UI. Junto com o rope: o COSCA deve implementar seu **text buffer como estrutura imutável/persistente (rope ou equivalente)** para undo ilimitado, edição assíncrona e diffs baratos. Sobre extensibilidade: a **sandbox WASM (WASI/component model)** é a decisão mais moderna para plugins seguros, superando o modelo de "processo host" do VS Code em segurança e portabilidade. Originalização: contrato RPC tipado e versionado próprio (com schema evolutivo), buffer com rope próprio, e plugin ABI em WASM/component model definida pelo COSCA (não copiar a API do Lapce).

- **ORIGINAL IMPLEMENTATION**: implementação própria de: rope buffer, RPC contract (schema + versioning), proxy workspace (FS/LSP/terminal/git bridge), plugin host WASM (com component model e capability-based grants), terminal. Abstrações inspiradas, código próprio.

---

## MATRIZ COMPARATIVA

| Capacidade | VS Code | Monaco | Theia | Zed | Lapce | **COSCA DESIGN** |
|---|---|---|---|---|---|---|
| **EDITOR CORE** | Monaco core (model/view/provider) sobre DOM | Model/View/Provider + workers (referência) | Reusa Monaco como editor | Rope + tree-sitter + GPUI/wgpu | Rope (Xi) + Floem/wgpu | **Rope buffer imutável endereçado por URI + view separada + tree-sitter incremental** (trindade do Monaco, estruturas do Lapce/Zed) |
| **EXTENSIBILIDADE** | Extension host em processo separado, API versionada | Providers (in-proc) | Dual: contribuições DI + plugin host compat VS Code | Extension host com ABI própria (jovem) | Plugins WASI (sandbox) | **Plugins WASM (component model, capability grants) + contribuições declarativas internas** — segurança do Lapce, ergonomia do VS Code |
| **PERFORMANCE** | Limitada por JS/DOM | Boa (workers) | Menor que VS Code | Excelente (GPU, Rust) | Excelente (Rust, wgpu) | **Linguagem de sistema + parsing incremental + trabalho pesado fora da UI thread + fila de input prioritária** (metas do Zed/Lapce) |
| **LSP** | Cliente maduro + language features | monaco-lsp-client | Suportado | Via crate `project`/`lsp` | Proxy embutido | **Cliente LSP no workspace backend (proxy), multiplexado, assíncrono, com cache de resultados** |
| **TERMINAL** | xterm.js integrado | — | Integrado | Crate `terminal` própria | Integrado | **Terminal próprio no backend via PTY + render de baixo custo** (emulação leve, sem xterm.js) |
| **DEBUGGING** | DAP maduro (adapters) | — | DAP | Crates `dap`/`debugger_ui` | Ausente/mínimo | **Cliente DAP no workspace backend + protocolo neutro** (não acoplar a um vendor) |
| **AI INTEGRATION** | Copilot como extensão (acoplada) | — | Plugins | **Subsistema de agentes + MCP + multi-provider nativo** | Via plugin | **Agent runtime próprio (planner + tools + MCP client) com LLM layer provider-agnostic** — decisão estrutural, não extensão |
| **REMOTE/CLOUD** | Remote dev (server) | — | Frontend/backend RPC (cloud) | Colaboração, sem remote completo | Proxy remoto nativo | **App/proxy split por RPC tipado: mesmo backend local/SSH/container** (Lapce + Theia) |
| **COLABORAÇÃO** | Extensão Live Share | — | Extensões | **Multiplayer nativo (CRDT/colab)** | — | *Fase 2*: workspace sincronizado por CRDT, se exigido |
| **ARQUITETURA GERAL** | Layered core + contributions + DI | Library embarcável | Framework DI + split | Crates + GPUI | Crates + RPC split | **Núcleo em camadas + contribuições declarativas + DI por token + app/proxy split** (síntese de VS Code + Theia + Lapce) |

### Decisão de síntese (resumo da coluna COSCA)

1. **Editor core**: trindade Document/View/Provider (Monaco) implementada com rope imutável (Lapce) + tree-sitter (Zed) como serviço de parsing central.
2. **Processo**: workspace backend (proxy) isolado atrás de contrato RPC tipado/versionado, isolável por ambiente (local/SSH/container) — herança de Lapce + Theia.
3. **Extensibilidade**: contribuições declarativas internas (VS Code/Theia) + plugins WASM sandbox (Lapce) para extensões de terceiros.
4. **AI**: agent runtime + MCP + LLM provider-agnostic como subsistema de primeira classe (Zed), não como extensão.
5. **Performance**: linguagem de sistema, parsing incremental, offload de trabalho pesado, meta de latência estilo Zed.

---

## Resumo executivo (para o COSCA CODE)

Os padrões mais valiosos encontrados:

1. **Trindade Document/View/Provider + buffer imutável (rope)** — fonte de verdade desacoplada da renderização e da inteligência; é o que permite splits, diffs, undo ilimitado e edição assíncrona. (Monaco + Lapce)
2. **Workspace backend isolado atrás de um contrato RPC tipado** — transforma local/remoto/container na mesma arquitetura; o proxy medeia FS, LSP, terminal, git, plugins. (Lapce + Theia)
3. **Extensibilidade dual: contribuições declarativas internas + plugins WASM sandbox** — ergonomia de desenvolvimento interno com segurança real de terceiros. (VS Code + Lapce)
4. **AI/agentes como subsistema estrutural (agent runtime + MCP + multi-provider), não como extensão** — garante integração profunda e performática. (Zed)
5. **Núcleo em camadas com ambientes de execução e DI por token** — regras de dependência rígidas permitem escalar funcionalidade sem acoplamento. (VS Code/Theia)

**Recomendação principal**: o COSCA CODE deve construir um **núcleo fino com um workspace backend (proxy) isolado por RPC**, um **editor core baseado em rope + tree-sitter com a trindade documento/visão/provedor**, **extensibilidade por contribuições declarativas + plugins WASM**, e **um agent runtime de IA como subsistema de primeira classe**. Essa combinação captura o melhor de cada fonte — a ergonomia do VS Code, a segurança do Lapce, a performance do Zed/Lapce e a extensibilidade do Theia — sem herdar os legados (DOM/Electron, API proprietária, framework DI externo, UI custom).
