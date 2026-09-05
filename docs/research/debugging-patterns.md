# Debugging & Debug Adapter Protocol — Padrões para o COSCA DEBUG ENGINE

> Pesquisa de arquitetura. Extração de **padrões, abstrações e protocolos** (não de código/UI/branding).
> Alvo: motor de debugging universal do **COSCA CODE** ("COSCA DEBUG ENGINE") — camada universal com adapters por linguagem/runtime, nunca um debugger artificial.

---

## SOURCE: Debug Adapter Protocol (DAP) — especificação

Fonte: `https://microsoft.github.io/debug-adapter-protocol/`

### WHAT IT SOLVES
O custo de implementar UI de debugging (breakpoints, variáveis em hover, multi-thread, watch, console, logpoints) é repetido por cada ferramenta, pois cada IDE/editor usa APIs próprias para a mesma funcionalidade. O DAP elimina essa duplicação ao padronizar **um único protocolo de comunicação** entre a UI genérica de debugging e o debugger concreto.

### HOW IT WORKS
- **Arquitetura de três partes**: `Development Tool (host/client)` ↔ `Debug Adapter (intermediário)` ↔ `Debugger/Runtime (backend)`. O adapter é um **intermediário de adaptação** — não o debugger em si. É isso que dá nome ao protocolo.
- **Wire protocol** (não API de biblioteca): permite que cada adapter seja implementado na linguagem mais adequada ao runtime. É o mesmo padrão do LSP.
- **Base protocol**: mensagens `header + content` (semelhante a HTTP). Header contém `Content-Length` (ASCII); content é JSON UTF-8. Três tipos de mensagem: `request`, `response`, `event`, todas com `seq` (sequência por ator) para ordenação, correlação request↔response e cancelamento.
- **Negociação por capacidades** (`capabilities`): cada feature tem uma flag (`supports*`). Ausência da flag = feature não suportada. Isso permite o protocolo crescer **sem versionamento** e de forma retrocompatível. O handshake `initialize` troca capacidades do cliente e do adapter (path format, line/column 0-ou-1-based, locale).
- **Fluxo de sessão**:
  1. `initialize` (troca de capabilities) → `launch`/`attach` (args são específicos do debugger; o protocolo não os especifica).
  2. Adapter emite `initialized` → cliente envia `setBreakpoints`, `setFunctionBreakpoints`, `setExceptionBreakpoints`, e `configurationDone`. O evento `initialized` livra o adapter de precisar fazer buffering de configuração.
  3. Ao parar (breakpoint/exception/pause/step), adapter emite `stopped` com `reason` e `threadId` + `hitBreakpointIds`.
  4. Consulta de estado por **waterfall preguiçoso**: `threads` → `stackTrace` → `scopes` → `variables` → `variables` (aninhado). É sob demanda, nunca eager.
- **Lifetime de object references**: referências (scopes, variáveis estruturadas) são válidas **apenas durante o estado suspenso atual**; resetam ao resumir. Contador pode recomeçar em 1. `threadId` é a exceção (não expira, pois `pause` precisa dele em running state).
- **Threads**: mesmo debuggers single-thread devem implementar `threads` retornando um thread dummy. `thread` event (optional) força atualização dinâmica de UI fora do estado stopped.
- **Encerramento**: `terminate` (graceful, para launch) vs `disconnect` (forçado). `terminated` event sempre; `exited` event com exit code. Para attach, `disconnect` desanexa sem matar o processo.
- **Reverse requests** (adapter→client): `runInTerminal` (lançar debuggee em terminal integrado/externo) e `startDebugging` (adapter pede ao cliente para abrir nova sessão — base para debug de subprocessos/workers).
- **Eventos de produtor**: `output` (categorias `console`/`stdout`/`stderr`/`telemetry`/`important`, com agrupamento e ANSI), `progressStart/Update/End`, `breakpoint` (changed/new/removed), `module`, `loadedSource`, `process`, `invalidated` (inválida snapshot da UI), `memory`.
- **Extensões avançadas de stepping**: `stepIn`/`stepOut`/`next` (granularidade de stepping), `stepInTargets`, `gotoTargets`/`goto`, `restartFrame`, e **reverse debugging** (`stepBack`, `reverseContinue`).
- **Richness de breakpoints**: breakpoints por source (`setBreakpoints` não-incremental — substitui tudo de um source e retorna posições "reais"), function breakpoints, data breakpoints, instruction breakpoints, `breakpointLocations` (posições possíveis num range), modos de breakpoint (`BreakpointMode`), exception breakpoints com filtros + `exceptionOptions` (condicionais por `error`).

### WHY IT WORKS
- Separa preocupações: a UI é **genérica e reutilizável**; o adapter é **barato** de escrever porque agrega APIs de debugger em estruturas de alto nível baseadas em **strings** (o que o usuário final vê), sem expor os detalhes finos do debugger.
- Padrão **pull-based**: o cliente só busca o que o usuário expande (waterfall), mantendo o protocolo leve.
- **Capabilities opt-in** eliminam versionamento e permitem evolução gradual por parte de cada adapter.
- Base protocol mínimo (HTTP-like) permite implementação trivial e reuso de SDKs.

### STRENGTHS
- Desacoplamento total UI ↔ runtime; um adapter serve N ferramentas; uma ferramenta serve M runtimes.
- Linguagem-agnóstico, string-first, de baixa complexidade de mapeamento.
- Extensível de forma retrocompatível (capabilities, reverse requests, eventos novos).
- Cobre ciclo de vida completo: config, execução, inspeção, expressões, memória, módulos, progresso, cancelamento, restart, reverse.

### WEAKNESSES
- `launch`/`attach` arguments são **não especificados** — cada tool precisa de um mecanismo de contribuição/validação (ex.: `debugger` contribuição com schema JSON) fora do protocolo.
- Protocolo de mensagens síncrono por correlação de `seq`; cancelamento é "best effort" (hint).
- Algumas áreas (avaliação com side-effects, formatação de valores, columns em UTF-16) dependem de capabilities que nem todos implementam.
- Não cobre como o adapter nasce (single-session via stdin/stdout vs multi-session via porta) — deixado para a ferramenta.

### PATTERN
**Broker/Adapter Pattern sobre wire protocol**: uma UI genérica fala um contrato estável; intermediários "adaptadores" traduzem o contrato para a API nativa de cada runtime. Complementado por: **capability negotiation** (opt-in por flag), **pull waterfall** (lazy state retrieval), **event-driven lifecycle** (`initialized`/`stopped`/`terminated`), **scoped object-reference lifetime**, e **reverse requests** (o adapter pode demandar recursos do host).

### COSCA INTERPRETATION
O COSCA DEBUG ENGINE deve adotar DAP como **protocolo de fronteira canônico**:
- Implementar um **host DAP** (client) na camada universal — responsável por gerir sessões, capabilities, waterfall, e ciclo de vida — totalmente agnóstico de linguagem.
- Expor uma **SPI de adapters** ("COSCA Debug Adapters") registrados por linguagem/runtime; cada adapter é um processo DAP (stdin/stdout) ou servidor (porta) que o host lança/descobre.
- O engine deve consumir as **capabilities** para renderizar UI condicionalmente (não assumir feature), e respeitar lifetime de object references e line/column base negociada.
- Suportar **reverse requests** (`runInTerminal`, `startDebugging`) para terminal integrado e debug de subprocessos/workers — diferenciais de um IDE moderno.
- Tratar `launch`/`attach` config como **schema contribuído** (JSON Schema por adapter), dando ao COSCA validação e autocomplete de launch config sem hardcode.

### ORIGINAL IMPLEMENTATION
Spec JSON Schema (`debugAdapterProtocol.json`); SDKs oficiais (`@vscode/debugadapter` p/ Node, `debugpy` p/ Python como implementação de referência). Ecossistema de adapters listado em `implementors/adapters/` (Delve p/ Go, lldb-dap, netcoredbg, js-debug, etc.).

---

## SOURCE: microsoft/vscode-js-debug

Fonte: `https://github.com/microsoft/vscode-js-debug`

### WHAT IT SOLVES
Debugging de JavaScript moderno com **configuração mínima ou zero**: Node.js, Chrome, Edge, WebView2, extension host, Blazor, React Native. Resolve a fragmentação de "quem debuga o quê" ao consolidar múltiplos runtimes sob um único adapter DAP e abstrair as diferenças (Node inspector protocol, CDP do Chrome, subprocessos, workers, WebAssembly).

### HOW IT WORKS
- **Um adapter DAP** que fala com vários backends via CDP (Chrome DevTools Protocol) / Node Inspector. O adapter é o ponto único de tradução DAP↔CDP.
- **Configuração por tipo de sessão** (`node`, `chrome`, `node-terminal`, `extensionHost`), cada um com um schema de opções: `program`, `runtimeExecutable`, `args`, `cwd`, `env`, `outFiles`, `sourceMaps`, `smartStep`, `skipFiles`, `sourceMapPathOverrides`, `webRoot`, `pathMapping`, etc.
- **Auto-attach**: debuga processos lançados no terminal (`autoAttachChildProcesses`), sem config — liga/desliga por comando/status bar.
- **Debug de processos filho e workers**: workers/service workers/iframes viram sessões DAP separadas (via `startDebugging`); stepping cruza `postMessage`.
- **Source maps como camada de resolução**: `sourceMaps` + `outFiles` (globs dos arquivos gerados) + `sourceMapPathOverrides` (reescrever paths do sourcemap → disco) + `resolveSourceMapLocations` (restringir onde aplicar) + `smartStep` (pular código gerado sem mapeamento) + `skipFiles` (pular node_modules/internals) + `sourceMapRenames` (mapear nomes minificados). `pauseForSourceMap`/`runtimeSourcemapPausePatterns` tratam sourcemaps tardios (ex.: Serverless).
- **Breakpoints avançados**: conditional exception breakpoints (filtrar por `error`), excluded callers (não parar se certo frame está no stack), instrumentation/event-listener breakpoints, return value interception (`$returnValue`).
- **Extras**: pretty-print de código minificado, profile CPU/heap sourcemap-aware, DWARF→WebAssembly debugging, network view, custom description/properties generators (objetos no painel de variáveis), `cascadeTerminateToConfigurations` (matar sessões dependentes).
- **Extensibilidade**: API de extensão + mecanismo de **CDP sharing** — expõe a conexão CDP subjacente a outras extensões via WebSocket (`requestCDPProxy`), com domínio `JsDebug` para subscribe de eventos.

### WHY IT WORKS
- **Zero/mínima configuração**: defaults inteligentes (descobrir `program` pelo package.json, `outFiles` glob, sourcemap overrides pré-populados) removem fricção.
- **Source maps tratados como problema de primeira classe**, não afterthought: resolução + override + smart step + renames formam um pipeline completo do binário minificado até a fonte original.
- **Modelo multi-sessão** (cada worker/subprocesso = sessão) espelha a realidade dos runtimes modernos e permite inspeção isolada.
- **Desacoplamento via capabilities e proxy**: expõe o backend real (CDP) para tooling especializado sem duplicar o adapter.

### STRENGTHS
- Cobertura de runtimes e configuração "it just works".
- Pipeline de source maps é o mais maduro do ecossistema (renames, smartStep, skipFiles, overrides).
- Debug de subprocessos/workers/postMessage como cidadão de primeira classe.
- Extensibilidade (CDP sharing) sem acoplamento.

### WEAKNESSES
- Complexidade interna alta; surface de opções gigantesca (dezenas por tipo).
- Forte acoplamento ao ecossistema JS (source maps V8/Node/Chrome) — pouco reutilizável fora dele.
- Muitas opções são heurísticas/por-runtime (webRoot, pathMapping, outFiles) que exigem tuning.
- CDP sharing é específico do backend CDP, não generalizável diretamente.

### PATTERN
**Single adapter, many backends + Source-Map Resolution Pipeline**: um adapter DAP abstrato sobre múltiplos protocolos de runtime equivalentes (CDP/Inspector), com:
- **multi-session via reverse request** (`startDebugging`) para workers/subprocessos,
- **pipeline de resolução de source maps** (glob → parse → override → rename → smartStep → skip) como serviço reutilizável,
- **configuração defaulted + schema por session-type**,
- **proxy de backend** para extensões especializadas.

### COSCA INTERPRETATION
- O COSCA não deve ter um "js-debug" próprio, mas sim aprender o **padrão de resolução de source maps como serviço compartilhado** do engine: um módulo `SourceMapResolver` (globs de outFiles + path overrides + smart-step + skip patterns + rename mapping) que qualquer adapter de linguagem possa reusar.
- Adotar o **modelo multi-sessão**: uma sessão lógica de debug = várias sessões DAP filhas (workers/subprocessos), agrupadas na UI como árvore.
- Suportar **auto-attach** como conceito de engine (hooks de terminal/processo), exposto de forma genérica.
- Considerar um **proxy de backend** opcional por adapter (expor protocolo nativo, ex.: CDP, a ferramentas especializadas) como padrão de extensibilidade.
- Registrar launch configs como **schemas declarativos por adapter** com defaults e variáveis de workspace (padrão `${workspaceFolder}`), não hardcode.

### ORIGINAL IMPLEMENTATION
Adapter DAP TypeScript (`src/adapter/`), extensão VS Code + servidor DAP standalone, `OPTIONS.md` como fonte dos schemas de launch config, mecanismo `cdpProxy`/PDL para CDP sharing.

---

## SOURCE: microsoft/debugpy

Fonte: `https://github.com/microsoft/debugpy`

### WHAT IT SOLVES
Implementação de DAP para Python 3: expõe o debugging de Python (breakpoints, attach, evaluation, exceções) de forma padrão a qualquer IDE que fale DAP, resolvendo o problema de "cada IDE precisa do seu próprio debugger Python".

### HOW IT WORKS
- **Arquitetura em camadas** (visível na árvore `src/debugpy/`):
  - `adapter/` — camada DAP (fala o wire protocol com o host; traduz para comandos internos).
  - `server/` — servidor socket (listen em host/porta) para o host conectar.
  - `launcher/` — dispara o processo "debuggee" como child process.
  - `_vendored/` — **pydevd vendado** (o debugger real de Python, do PyDev).
  - `common/` — infra compartilhada.
  - `public_api.py` — superfície pública (`listen`, `wait_for_client`, `breakpoint`, `trigger_exception_handler`, `log_to`).
- **Três modos de entrada equivalentes**:
  1. CLI (`python -m debugpy --listen HOST:PORT [--wait-for-client] [--pid PID] [--configure-subProcess] script.py`).
  2. Import API (`import debugpy; debugpy.listen(...); debugpy.wait_for_client()`).
  3. Attach por PID (`--pid 12345` injeta o debugger num processo já rodando).
- **Attach/launch**: `--listen` define onde o adapter escuta; `--wait-for-client` bloqueia até o host conectar; `0.0.0.0` para acesso remoto (com alerta de segurança explícito).
- **Debug de subprocessos**: `--configure-subProcess False` desliga o debug de filhos (default: ligado).
- **Breakpoints programáticos**: integra `breakpoint()` do Python; `debugpy.breakpoint()` como fallback; se nenhum cliente conectado, **no-op** (não interrompe execução).
- **Exceções**: `trigger_exception_handler()` faz post-mortem de exceção capturada (estilo `pdb.post_mortem`), respeitando filtros de exception breakpoints (`as_uncaught`).
- **Logging multi-componente**: `--log-to`/`debugpy.log_to()`/`DEBUGPY_LOG_DIR`; arquivos `debugpy*.log` separados por componente e por subprocesso.

### WHY IT WORKS
- **Separação limpa entre protocolo e engine**: o `adapter` (DAP) é fino; o `_vendored`/pydevd concentra a mecânica de debugging. Reusa décadas de pydevd sem reescrever o DAP.
- **Multi-entry de habilitação** (CLI/import/PID) cobre todos os cenários reais: script, módulo, processo já em execução.
- **`wait_for_client` + listen antes de executar** resolve o race de "debugar desde o início" sem instrumentação complexa.
- **No-op quando sem cliente** (breakpoint/exceção) torna o código instrumentado seguro em produção.

### STRENGTHS
- Adapter DAP de referência, arquitetura enxuta e bem fatorada.
- Attach por PID e debug de subprocessos resolvidos de forma simples.
- Integração idiomática com a linguagem (breakpoint(), exceções).
- Multi-interface (CLI, API, remote) sem mudar o protocolo.

### WEAKNESSES
- Single-language (Python); não é um motor universal.
- Debug de subprocessos é liga/desliga global, sem árvore de sessões rica (menos granular que js-debug).
- Evaluation/REPL menos rico que JS (limitações da linguagem/CPython).
- Attach por PID depende de injeção de runtime (nem todo runtime permite).

### PATTERN
**Thin DAP adapter over vendored native engine + multi-entry enablement**: separa (1) camada de protocolo (adapter/server), (2) launcher, e (3) engine vendado, com **N formas equivalentes de habilitar** o debugging (CLI / import API / PID attach), e **instrumentação inerte** (no-op sem cliente).

### COSCA INTERPRETATION
- A arquitetura em **três planos** (protocol adapter / launcher / engine) é o modelo canônico para cada COSCA adapter: a camada DAP é fina; a linguagem/runtime faz o trabalho pesado (ou um engine nativo é vendado).
- O COSCA DEBUG ENGINE deve oferecer aos adapters uma **infra de "enablement" uniforme**: um "debug launcher" e um "debug server" reutilizáveis, para que escrever um adapter novo = implementar só a tradução DAP.
- Suportar **múltiplas formas de entrada** por adapter (launch, attach por PID, connect a servidor) como contrato da SPI, com `waitForClient` como semântica padrão.
- Padronizar **instrumentação inerte** (no-op quando sem sessão) como regra para pontos de debug programáticos.
- Padronizar **logging multi-componente** por sessão/por subprocesso no engine (diagnóstico é crítico em debugging).

### ORIGINAL IMPLEMENTATION
`src/debugpy/{adapter,server,launcher,common,_vendored,public_api.py}`; CLI via `__main__.py`; wiki com Command-Line Reference e API Reference.

---

## MATRIZ COMPARATIVA

| CAPACIDADE | DAP (spec) | js-debug | debugpy | **COSCA DESIGN** |
|---|---|---|---|---|
| **BREAKPOINTS** | `setBreakpoints` (por-source, não-incremental, retorna posição "real" + `verified`), function/data/instruction breakpoints, `breakpointLocations`, `BreakpointMode`, exception breakpoints com filtros + `exceptionOptions` | conditional exception BP (filtro por `error`), excluded callers, instrumentation/event-listener BP, return value interception | breakpoint() integrado; breakpoints padrão via pydevd | SPI de breakpoints genérica: source + function + data + instruction; filtros de exceção com condições; **evento de breakpoint assíncrono** (`breakpoint` changed) para re-verificação dinâmica; suportar "exclude caller" como opção de breakpoint |
| **STEPPING** | `next`/`stepIn`/`stepOut` + `SteppingGranularity`, `stepInTargets`, `gotoTargets`/`goto`, `restartFrame`, `stepBack`/`reverseContinue` | step-in targets, smartStep (pular código sem mapeamento), postMessage crossing | stepping via pydevd | Modelo de stepping com **granularidade** + **targets de step-in** + **goto** como cidadãos de primeira classe; reservar hook para **reverse debugging** (capability), mesmo que poucos adapters suportem |
| **SCOPES/VARIABLES** | Waterfall `scopes`→`variables`; `variablesReference` com lifetime = estado suspenso; `setVariable`, `SetExpression`; `VariablePresentationHint`, `ValueFormat` | custom description/properties generators, return value como variável | variáveis via pydevd | Implementar waterfall **lazy** e **cache por estado suspenso**; invalidar ao resumir (honrar lifetime); permitir **presentation hints** e geradores de descrição/propriedades como extensão do engine |
| **SOURCE MAPS** | `Source` (path, sourceReference, origin, adapterData, checksums) | pipeline completo: outFiles globs, `sourceMapPathOverrides`, renames, smartStep, skipFiles, resolveSourceMapLocations, pauseForSourceMap | N/A (Python não usa) | **Serviço compartilhado `SourceMapResolver`** no engine (não no adapter): parse de sourcemap, override de paths, rename mapping, smart-step, skip patterns — reutilizável por qualquer linguagem compilada/minificada |
| **REMOTE/ATTACH** | `attach` request; args não-especificados; `process`/`module` events | attach por address+port, `restart` (reconectar), CDP proxy, websocket | listen host/porta, `--pid` attach, `0.0.0.0` remoto, `wait_for_client` | Contrato de attach genérico na SPI: **connect por endereço/porta OU PID OU servidor DAP multi-sessão**; suportar `waitForClient` e reconexão (`restart`) como semântica do engine |
| **LAUNCH CONFIG** | `launch`/`attach` args fora da spec; tool contribui schema/validação | schemas por session-type (node/chrome/...), defaults inteligentes, `${workspaceFolder}` | CLI switches + API | **Launch configs como JSON Schema contribuído por adapter**, com variáveis de workspace e defaults; o engine valida/autocompleta sem hardcode de linguagem |
| **DEBUG CONSOLE** | `output` event (categorias console/stdout/stderr/telemetry/important, group, ANSI), `evaluate` (REPL), `completions` | output via console/std, pretty-print | output via pydevd | Console unificado no engine: rotear `output` por categoria + agrupamento + ANSI; REPL via `evaluate` com `completions`; pretty-print como feature opcional de engine |
| **EVALUATE** | `evaluate` (contexto: watch/repl/hover/clipboard), `SetExpression`, `ExceptionInfo` | evaluate com renames de sourcemap; lldb-eval p/ wasm | evaluation via pydevd | `evaluate` com **contexto** explícito (REPL/hover/watch) e **remapping de identificadores** (rename de sourcemap) no engine; exceções via `exceptionInfo` |

---

## Padrões-chave sintetizados (para o COSCA DEBUG ENGINE)

1. **Broker/Adapter sobre wire protocol**: uma UI/engine genérico fala um contrato estável (DAP); cada linguagem contribui um adapter fino que traduz para o runtime. É o mesmo modelo do LSP e deve ser a espinha dorsal do COSCA.
2. **Separação em três planos por adapter**: `protocol adapter` (DAP) / `launcher` (spawn/connect) / `engine nativo` (ou vendado). Escrever um adapter novo = implementar só a tradução.
3. **Capability negotiation**: toda feature é opt-in por flag; o engine nunca assume feature e renderiza UI condicionalmente.
4. **Lazy pull waterfall + lifetime de referências**: buscar `threads→stackTrace→scopes→variables` sob demanda, invalidar no resume.
5. **Multi-session + reverse requests**: workers/subprocessos viram sessões filhas (`startDebugging`); `runInTerminal` conecta o debugger ao terminal integrado.
6. **Source maps como serviço compartilhado** (pipeline de resolução/override/rename/smart-step) reutilizável por qualquer adapter.
7. **Launch config como schema contribuído** (não hardcoded) com defaults e variáveis de workspace.
8. **Instrumentação inerte** (no-op sem cliente) + **enablement multi-entrada** (launch/attach/PID/connect).
