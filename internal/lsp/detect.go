package lsp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// serverSpec descreve o binário (e args) de um language server por linguagem.
type serverSpec struct {
	bins []string // candidatos de binário, em ordem de preferência
	args []string // args padrão (ex.: --stdio)
}

// servers mapeia linguagem → language server (auto-detecção, spec seção 14/50).
var servers = map[string]serverSpec{
	"go":         {bins: []string{"gopls"}},
	"rust":       {bins: []string{"rust-analyzer"}},
	"python":     {bins: []string{"pyright-langserver", "pyright"}},
	"typescript": {bins: []string{"typescript-language-server"}, args: []string{"--stdio"}},
	"javascript": {bins: []string{"typescript-language-server"}, args: []string{"--stdio"}},
	"c":          {bins: []string{"clangd"}},
	"cpp":        {bins: []string{"clangd"}},
}

// Detect resolve o binário e args de um language server para a linguagem,
// procurando no PATH e em ~/go/bin. Retorna erro se a linguagem não tem server
// mapeado ou nenhum binário foi encontrado.
func Detect(lang string) (bin string, args []string, err error) {
	spec, ok := servers[lang]
	if !ok {
		return "", nil, fmt.Errorf("sem language server mapeado para %q", lang)
	}
	for _, candidate := range spec.bins {
		if p := resolveBin(candidate); p != "" {
			return p, spec.args, nil
		}
	}
	return "", nil, fmt.Errorf("language server não encontrado para %q (tentei %v)", lang, spec.bins)
}

// resolveBin procura o binário no PATH e em diretórios comuns (~/go/bin,
// ~/.cargo/bin, ~/.local/bin).
func resolveBin(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, dir := range []string{"go/bin", ".cargo/bin", ".local/bin"} {
		p := filepath.Join(home, dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// Supported devolve as linguagens com language server mapeado.
func Supported() []string {
	var langs []string
	for lang := range servers {
		langs = append(langs, lang)
	}
	return langs
}
