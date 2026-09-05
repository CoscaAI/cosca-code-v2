// Package intel é o Project Intelligence Engine do COSCA CODE (spec seção 15):
// indexa o projeto (símbolos + dependências) para responder "onde isso é
// usado?", "o que quebra se eu alterar isso?", "quem chama essa função?".
// Extração por regex por linguagem (rápido, zero dependência) — a base; a
// migração para tree-sitter/LSP workspace symbols vem como camada semântica.
package intel

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Symbol é uma definição de símbolo em um arquivo.
type Symbol struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // func, method, type, var, const, class, def, struct, fn
	File string `json:"file"`
	Line int    `json:"line"`
}

// Index é o índice de projeto: símbolos + grafo de dependências.
type Index struct {
	Symbols      []Symbol            `json:"symbols"`
	Dependencies map[string][]string `json:"dependencies"` // file → imports (paths de módulo/arquivo)
	reverse      map[string][]string // import → files que importam
}

// dirIgnores espelha o workspace/search (não indexar dependências/artefatos).
var dirIgnores = map[string]bool{
	".git": true, "node_modules": true, "target": true, "dist": true,
	"build": true, ".next": true, ".cache": true, "__pycache__": true,
	".venv": true, "venv": true, ".idea": true, ".vscode": true, "coverage": true,
}

// Build varre root e constrói o índice (símbolos + dependências).
func Build(root string) (*Index, error) {
	idx := &Index{
		Dependencies: map[string][]string{},
		reverse:      map[string][]string{},
	}

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() != filepath.Base(root) && dirIgnores[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		lang := langForExt(filepath.Ext(path))
		if lang == "" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		extract(rel, path, lang, idx)
		return nil
	})

	return idx, nil
}

// Usages devolve os símbolos com o nome dado (definições no projeto).
func (idx *Index) Usages(name string) []Symbol {
	var out []Symbol
	for _, s := range idx.Symbols {
		if s.Name == name {
			out = append(out, s)
		}
	}
	return out
}

// Dependents devolve os arquivos que importam/dependem de `file` (deduplicado).
// O matching é por igualdade ou sufixo de path — nunca substring (evita que
// "os" case dentro de "Cosca", "sort" dentro de "sorting", etc.).
func (idx *Index) Dependents(file string) []string {
	base := strings.TrimSuffix(file, filepath.Ext(file))
	seen := map[string]bool{}
	var out []string
	for importer, imports := range idx.Dependencies {
		for _, imp := range imports {
			if matchDep(imp, base) {
				if !seen[importer] {
					seen[importer] = true
					out = append(out, importer)
				}
			}
		}
	}
	return out
}

// matchDep reporta se um import corresponde a um alvo (igualdade ou sufixo).
func matchDep(imp, base string) bool {
	if imp == base {
		return true
	}
	return strings.HasSuffix(imp, "/"+base) || strings.HasSuffix(base, "/"+imp)
}

// langForExt mapeia extensão → linguagem.
func langForExt(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx", ".js", ".jsx":
		return "js"
	case ".py":
		return "py"
	case ".rs":
		return "rs"
	default:
		return ""
	}
}

// extract extrai símbolos e imports de um arquivo conforme a linguagem.
func extract(rel, path, lang string, idx *Index) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for _, sym := range symbolsFor(lang, lines) {
		sym.File = rel
		idx.Symbols = append(idx.Symbols, sym)
	}
	for _, imp := range importsFor(lang, lines) {
		idx.Dependencies[rel] = append(idx.Dependencies[rel], imp)
		idx.reverse[imp] = append(idx.reverse[imp], rel)
	}
}

// symbolsFor extrai símbolos (definições) linha a linha.
func symbolsFor(lang string, lines []string) []Symbol {
	var out []Symbol
	patterns := symbolPatterns(lang)
	for i, line := range lines {
		for _, p := range patterns {
			if m := p.re.FindStringSubmatch(line); m != nil {
				out = append(out, Symbol{Name: m[1], Kind: p.kind, Line: i + 1})
			}
		}
	}
	return out
}

type symbolPattern struct {
	re   *regexp.Regexp
	kind string
}

func symbolPatterns(lang string) []symbolPattern {
	switch lang {
	case "go":
		return []symbolPattern{
			{regexp.MustCompile(`^func\s+\([^)]*\)\s+(\w+)`), "method"},
			{regexp.MustCompile(`^func\s+(\w+)`), "func"},
			{regexp.MustCompile(`^type\s+(\w+)`), "type"},
			{regexp.MustCompile(`^(?:var|const)\s+(\w+)`), "var"},
		}
	case "js":
		return []symbolPattern{
			{regexp.MustCompile(`^(?:export\s+)?(?:function|class)\s+(\w+)`), "func"},
			{regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(\w+)`), "var"},
		}
	case "py":
		return []symbolPattern{
			{regexp.MustCompile(`^(?:def|class)\s+(\w+)`), "func"},
		}
	case "rs":
		return []symbolPattern{
			{regexp.MustCompile(`^(?:pub\s+)?fn\s+(\w+)`), "fn"},
			{regexp.MustCompile(`^(?:pub\s+)?(?:struct|enum|trait)\s+(\w+)`), "type"},
		}
	default:
		return nil
	}
}

// importsFor extrai os imports/dependências de um arquivo.
func importsFor(lang string, lines []string) []string {
	var out []string
	switch lang {
	case "go":
		// Import de linha única (`import "x"`) e bloco (`import ( "a" "b" )`).
		text := strings.Join(lines, "\n")
		singleRe := regexp.MustCompile(`import\s+"([^"]+)"`)
		for _, m := range singleRe.FindAllStringSubmatch(text, -1) {
			out = append(out, m[1])
		}
		blockRe := regexp.MustCompile(`import\s*\(\s*([^)]*)\)`)
		strRe := regexp.MustCompile(`"([^"]+)"`)
		for _, m := range blockRe.FindAllStringSubmatch(text, -1) {
			for _, s := range strRe.FindAllStringSubmatch(m[1], -1) {
				out = append(out, s[1])
			}
		}
	case "js":
		re := regexp.MustCompile(`from\s*["']([^"']+)["']|import\s*["']([^"']+)["']`)
		for _, line := range lines {
			for _, m := range re.FindAllStringSubmatch(line, -1) {
				if m[1] != "" {
					out = append(out, m[1])
				} else if m[2] != "" {
					out = append(out, m[2])
				}
			}
		}
	case "py":
		re := regexp.MustCompile(`^(import|from)\s+([\w.]+)`)
		for _, line := range lines {
			if m := re.FindStringSubmatch(line); m != nil {
				out = append(out, m[2])
			}
		}
	case "rs":
		re := regexp.MustCompile(`^use\s+([\w:]+)`)
		for _, line := range lines {
			if m := re.FindStringSubmatch(line); m != nil {
				out = append(out, m[1])
			}
		}
	}
	return out
}
