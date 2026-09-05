# Code Intelligence Patterns — tree-sitter, ast-grep, ripgrep

> Pesquisa de referência para o COSCA CODE (code intelligence engine).
> NÃO é código a copiar — é extração de padrões, decisões arquiteturais e estratégias.

---

## SOURCE: tree-sitter

- **WHAT IT SOLVES**: parsing estrutural de código que funciona em tempo real dentro de um editor. Sem tree-sitter, editores ou recaem em regex (frágil) ou re-parseiam o arquivo inteiro a cada keystroke (lento). Resolve o problema de "ter consciência sintática/estrutural do código enquanto ele muda".
- **HOW IT WORKS**: é um *parser generator* (gera parsers a partir de gramáticas) + *incremental parsing library* (runtime em C puro). Produz uma **concrete syntax tree** (CST) — não uma AST abstrata — e a **atualiza incrementalmente** quando o texto muda, reaproveitando os subárvores que não foram afetadas pela edição.
- **WHY IT WORKS**: os quatro objetivos explícitos se sustentam mutuamente:
  1. **General** — gramática declarativa cobre qualquer linguagem;
  2. **Fast** — parse incremental reusa subárvores intactas (custo ~O(tamanho da edição), não O(arquivo));
  3. **Robust** — *error-tolerant parsing*: mesmo com erro de sintaxe, produz uma árvore útil (com nós de erro), o que é essencial num editor onde código incompleto é o estado normal, não a exceção;
  4. **Dependency-free** — runtime em C puro, embutível em qualquer host (Go, Rust, JS/WASM, etc.).
- **STRENGTHS**: incremental parsing em keystroke; tolerância a erro; gramáticas declarativas; bindings multiplataforma; ecossistema enorme de gramáticas prontas.
- **WEAKNESSES**: CST é concreta (verbosa, espelha sintaxe, não semântica); não resolve *semântica* (tipos, referências, imports) — isso é papel de LSP/compilador; gramáticas variam em qualidade.
- **PATTERN**: **separar "estrutura sintática" (barata, local, incremental, tolerante a erro) de "semântica" (cara, global, via LSP/compilador)**. O editor mantém um índice sintático vivo via parsing incremental, e delega semântica a um serviço externo.
- **COSCA INTERPRETATION**: o COSCA CODE usa tree-sitter como **base sintática** do Project Intelligence Engine — parsing incremental a cada edição para alimentar syntax highlighting, outline/symbols, code navigation local e busca estrutural. Nunca tenta extrair semântica do tree-sitter (tipos/refs ficam no LSP). O motor estrutural do COSCA é *consumidor* do tree-sitter, não reimplementação.
- **ORIGINAL IMPLEMENTATION**: bindings Go do tree-sitter para embutir o runtime no backend do COSCA; gramáticas versionadas como recurso do workspace; índice sintático incremental persistido (não re-parse total em startup).

---

## SOURCE: ast-grep

- **WHAT IT SOLVES**: busca e *reescrita* de código por **estrutura** (AST), não por texto. Regex não entende código ("ache toda chamada `foo(x)`" é impossível de expressar com precisão em regex). ast-grep torna a manipulação estrutural tão simples quanto grep.
- **HOW IT WORKS**: montado **em cima do tree-sitter** (usa o CST como fonte). O usuário escreve um *pattern* que é **isomórfico a código** — um trecho de código real com `$WILDCARDS` (ex: `$A && $A()`) — e a ferramenta casa esse pattern contra nós AST. Suporta *rewrite* (substituir o nó casado por outro), lint (rules YAML) e manipulação em massa (codemods/migrações).
- **WHY IT WORKS**: o pattern ser "código de verdade" (não uma DSL estranha) torna a busca estrutural acessível a qualquer desenvolvedor. É a ponte entre "regex textual" (barato mas burro) e "programação de AST" (poderosa mas inacessível). Usar tree-sitter como backend dá robustez (tolera erro) e cobertura (muitas linguagens) de graça.
- **STRENGTHS**: pattern intuitivo (isomórfico a código); rewrite estrutural seguro; linting/codemods declarativos; multi-core; não exige aprender API de AST.
- **WEAKNESSES**: busca puramente *sintática* (não entende semântica — não sabe se `foo` é o mesmo símbolo em dois lugares); rewrite é estrutural, não semântico; ferramenta de CLI/codemod, não motor de busca interativo de IDE.
- **PATTERN**: **busca estrutural = "código como pattern" com wildcards de nó**, sobre uma AST tolerante a erro. Rewrite/codemod declarativo via o mesmo padrão. É uma *segunda camada* sobre o parsing (tree-sitter) que adiciona *expressividade de consulta*.
- **COSCA INTERPRETATION**: o COSCA CODE expõe **AST search / structural search** como uma das estratégias do Code Search (seção 16 do spec) — "ache todos os lugares com esta forma" — e usa o mesmo conceito para **refactoring estrutural** (rename/extract/move por padrão, não regex) e **codemods assistidos por IA**. O motor escolhe a estratégia (texto, regex, símbolo, AST, semântica, dependência, referência) automaticamente.
- **ORIGINAL IMPLEMENTATION**: query layer próprio sobre tree-sitter (pattern matching + rewrite), exposto como comando de search e como tool do agente (o agente diz "ache toda chamada `init(ctx)` e troque por `init(ctx, opts)`" e o motor executa estruturalmente).

---

## SOURCE: ripgrep

- **WHAT IT SOLVES**: busca textual recursiva **extremamente rápida** e com **filtragem inteligente por padrão**. É o "grep" que respeita o mundo real de um repositório (gitignore, binários, hidden files) sem que o usuário precise pedir.
- **HOW IT WORKS**: busca *line-oriented* com regex sobre o diretório corrente, recursivo. A velocidade vem de um conjunto de decisões, não de um truque único:
  1. **Rust regex engine** — finite automata + SIMD + *aggressive literal optimizations* (quando o padrão tem literais, usa-os para pular candidatos);
  2. **UTF-8 no DFA** — decodificação embutida no autômato, mantendo Unicode sem o custo clássico do grep;
  3. **memory maps vs buffer incremental** — escolhe automaticamente (mmap para arquivo único, buffer para diretório grande);
  4. **`RegexSet`** — casa um caminho contra múltiplos globs de ignore *simultaneamente*;
  5. **iterador de diretórios paralelo lock-free** (crossbeam + ignore).
- **WHY IT WORKS**: `rg` **não indexa** — é grep *on-the-fly*. A combinação de literal optimizations + paralelismo + filtragem automática faz com que, para a maioria dos repos, a busca bruta seja mais rápida que manter um índice. E a filtragem por padrão (gitignore) é exatamente o que o usuário de código espera (não quer resultados em `node_modules/` ou `.git/`).
- **STRENGTHS**: velocidade de ponta; filtragem automática (gitignore/ignore/rgignore/hidden/binary); Unicode rápido; preprocessors (zip, encodings); `--json` para consumidores programáticos.
- **WEAKNESSES**: **não indexa** — para repos gigantes, re-scan a cada busca tem custo; não entende estrutura/semântica (é texto puro); regex tem limites expressivos (sem look-around no motor default, PCRE2 é opt-in).
- **PATTERN**: **busca textual = grep on-the-fly com literal optimizations + paralelismo + filtragem automática por ignore rules**. Índice só compensa quando o custo de re-scan supera o de manter o índice — a decisão é de *estratégia*, não de dogma.
- **COSCA INTERPRETATION**: o COSCA CODE adota o mesmo princípio para **text/regex search**: busca direta on-the-fly, paralela, git-aware, com filtragem automática por gitignore — *sem* índice obrigatório. O índice entra *apenas* onde há ganho real (symbol graph, referências, semântica), e a escolha entre "scan on-the-fly" e "índice" é feita pelo Code Search Engine conforme o tipo de consulta. `--json`-like: toda busca expõe resultado estruturado consumível pelo agente.
- **ORIGINAL IMPLEMENTATION**: motor de busca textual próprio em Go (paralelo, git-aware, com literal optimizations), embutido no backend — o COSCA não executa `rg` como processo externo para buscas interativas; expõe busca estruturada ao agente (tool) e à UI.

---

## MATRIZ COMPARATIVA

| CAPABILITY | tree-sitter | ast-grep | ripgrep | COSCA DESIGN |
|---|---|---|---|---|
| **Parsing** | incremental, error-tolerant, CST | — (usa tree-sitter) | — (texto) | tree-sitter embutido (Go) como camada sintática viva |
| **Structural search** | — (só parsing) | pattern isomórfico a código + wildcards | — | query layer própria sobre tree-sitter + rewrite estrutural |
| **Text search** | — | — | grep on-the-fly, literal opt + paralelo + git-aware | mesmo princípio, motor Go próprio, sem índice obrigatório |
| **Indexing strategy** | CST incremental (implícito) | — | **sem índice** | híbrido: índice só p/ símbolos/refs/semântica; texto = scan |
| **Error tolerance** | sim (nós de erro) | herdado | n/a | herdado via tree-sitter (código incompleto é o normal) |
| **Semantic awareness** | não | não | não | LSP/compilador (separado do motor estrutural) |

---

## SÍNTESE PARA O COSCA CODE INTELLIGENCE ENGINE

Três camadas complementares, nunca concorrentes:

1. **Sintaxe viva** (tree-sitter): parse incremental a cada keystroke → highlighting, outline, navegação local, base para busca estrutural.
2. **Busca** (ripgrep-like + ast-grep-like): múltiplas estratégias — text, regex, symbol, AST, semantic, dependency, reference — com seleção automática. Texto = scan on-the-fly git-aware; estrutura = query sobre tree-sitter; semântica = índice (LSP/compilador).
3. **Semântica** (LSP): types, references, imports, call graph — delegada, nunca reimplementada no núcleo.

O princípio transversal: **separar sintaxe (barata/local/tolerante a erro) de semântica (cara/global/precisa)** e escolher a estratégia de busca pela natureza da pergunta, não por um único motor monolítico.

---

## IMPLEMENTAÇÃO GO: binding `github.com/tree-sitter/go-tree-sitter`

> Fonte: https://pkg.go.dev/github.com/tree-sitter/go-tree-sitter (v0.25.0, MIT, ~174 importers).

O COSCA CODE tem backend Go (spec seção 69/70). O binding Go oficial é o caminho para embutir o tree-sitter no núcleo sem reinventar o runtime C. Decisões relevantes extraídas:

- **Gramáticas NÃO embutidas por padrão** — cada linguagem é um pacote Go separado (`github.com/tree-sitter/tree-sitter-javascript/bindings/go`). Isso é exatamente o padrão de "traga só o que precisa" do COSCA: o núcleo não depende de nenhuma gramática; as linguagens são módulos opt-in.
- **API de incremental parsing**: `Parser.Parse(text, oldTree)` — passa a árvore anterior para re-parse incremental a cada keystroke. `Tree.Edit(edit)` + `Tree.ChangedRanges(other)` expõem o diff estrutural entre versões.
- **Query system**: `NewQuery(language, source)` + `QueryCursor.Captures/Matches` — é a base tanto para syntax highlighting (captures de nós) quanto para busca estrutural (matches de pattern). É a fundação do "ast-grep-like" próprio do COSCA.
- **Duas formas de carregar gramática**: (a) import direto do pacote Go da gramática (compile-time); (b) runtime via shared library + `purego` (`Dlopen` + `RegisterLibFunc` sobre `tree_sitter_<lang>()`). A (b) é a porta para **extensões de linguagem dinâmicas** (instalar um parser .so sem recompilar o COSCA) — alinha com a seção 12/13 do spec (extensões sem recompilar o núcleo).
- **Aviso de lifetime CGO**: objetos que alocam memória C (`Parser`, `Tree`, `TreeCursor`, `Query`, `QueryCursor`, `LookaheadIterator`) exigem `Close()` explícito (bugs de `runtime.SetFinalizer` + CGO). O COSCA deve encapsular isso numa camada de ownership (um tipo `ParsedDocument` que gerencia Parser+Tree+Close com RAII/`defer`), nunca expor os ponteiros crus.

**COSCA INTERPRETATION**: o `cosca-code` adota `go-tree-sitter` como runtime de parsing, mas envolve tudo numa camada própria (`internal/parsing`) que: (1) gerencia lifetimes CGO com um tipo dono (`Document`/`ParseSession`); (2) mantém o índice sintático incremental por arquivo; (3) expõe query de captures/matches como busca estrutural e highlighting; (4) suporta carregamento de gramática compile-time (padrões) E runtime/purego (extensões). Nunca vaza `*tree_sitter.Parser` para o resto do sistema.
