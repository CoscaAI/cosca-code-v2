// Package search é o Search Engine do COSCA CODE (spec seção 16): busca
// multi-estratégia com seleção automática. Este arquivo implementa a estratégia
// TEXT/REGEX — grep on-the-fly, git-aware, paralelo (o padrão ripgrep extraído
// na pesquisa: filtragem automática por ignore rules + paralelismo). Symbol/
// AST/semantic são estratégias adicionais sobre outras camadas.
package search

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Match é uma ocorrência do padrão em uma linha.
type Match struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Content string `json:"content"`
}

// Options configura a busca.
type Options struct {
	Query         string
	Regex         bool
	CaseSensitive bool
	MaxResults    int    // 0 = ilimitado
	Include       string // extensão opcional (ex.: ".go")
}

// Result é o resultado da busca.
type Result struct {
	Matches []Match `json:"matches"`
	Files   int     `json:"files"`
	Total   int     `json:"total"`
}

// dirIgnores são diretórios nunca varridos (filtragem git-aware — não queremos
// dependências/artefatos nos resultados, mesmo padrão do workspace).
var dirIgnores = map[string]bool{
	".git": true, "node_modules": true, "target": true, "dist": true,
	"build": true, ".next": true, ".cache": true, "__pycache__": true,
	".venv": true, "venv": true, ".idea": true, ".vscode": true, "coverage": true,
}

// Search executa a busca text/regex sobre root.
func Search(root string, opts Options) (*Result, error) {
	re, err := compile(opts)
	if err != nil {
		return nil, err
	}

	result := &Result{}
	var mu sync.Mutex

	var wg sync.WaitGroup
	sem := make(chan struct{}, 16) // limite de goroutines de varredura

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
		if opts.Include != "" && filepath.Ext(path) != opts.Include {
			return nil
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()
			grepFile(root, p, re, &result.Matches, &mu)
		}(path)
		return nil
	})
	wg.Wait()

	result.Total = len(result.Matches)
	if opts.MaxResults > 0 && result.Total > opts.MaxResults {
		result.Matches = result.Matches[:opts.MaxResults]
	}
	seen := map[string]bool{}
	for _, m := range result.Matches {
		seen[m.Path] = true
	}
	result.Files = len(seen)
	return result, nil
}

// compile monta o regex a partir das opções.
func compile(opts Options) (*regexp.Regexp, error) {
	pattern := opts.Query
	if !opts.Regex {
		pattern = regexp.QuoteMeta(opts.Query)
	}
	if !opts.CaseSensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// grepFile varre um arquivo procurando o padrão.
func grepFile(root, path string, re *regexp.Regexp, matches *[]Match, mu *sync.Mutex) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	rel, _ := filepath.Rel(root, path)
	rel = filepath.ToSlash(rel)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // linhas longas
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		loc := re.FindStringIndex(line)
		if loc == nil {
			continue
		}
		mu.Lock()
		*matches = append(*matches, Match{
			Path:    rel,
			Line:    lineNo,
			Column:  loc[0] + 1,
			Content: strings.TrimSpace(line),
		})
		mu.Unlock()
	}
}
