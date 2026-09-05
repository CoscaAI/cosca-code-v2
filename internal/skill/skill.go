// Package skill é o sistema de Skills do COSCA CODE (spec seção 82): uma skill
// define objetivo, instruções e tools recomendadas — uma "especialização" do
// agente (Go Developer, Security Auditor, Tester...). Carregar uma skill
// ajusta o system prompt do agente.
package skill

import "strings"

// Skill é uma especialização do agente.
type Skill struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Instructions string   `json:"instructions"` // system prompt especializado
	Tools        []string `json:"tools"`        // tools recomendadas
}

// Builtin devolve as skills nativas do COSCA CODE.
func Builtin() []Skill {
	return []Skill{
		{
			Name:        "go_developer",
			Description: "Desenvolvimento em Go.",
			Instructions: `Você é um desenvolvedor Go sênior. Escreva código idiomático,
use error handling explícito, prefira a stdlib, e teste com 'go test'.
Siga as convenções do projeto.`,
			Tools: []string{"read_file", "list_files", "search"},
		},
		{
			Name:        "security_auditor",
			Description: "Auditoria de segurança de código.",
			Instructions: `Você é um auditor de segurança. Procure vulnerabilidades:
injeção, escape de path, exposição de segredos, validação ausente.
Reporte achados com severidade e localização. Nunca invente vulnerabilidades.`,
			Tools: []string{"read_file", "list_files", "search"},
		},
		{
			Name:        "tester",
			Description: "Geração e revisão de testes.",
			Instructions: `Você é um engenheiro de testes. Escreva testes que cobrem
casos de borda e contraexemplos. Separe GERAÇÃO de VERIFICAÇÃO — nunca afirme
que algo passou sem evidência.`,
			Tools: []string{"read_file", "list_files", "search"},
		},
	}
}

// Prompt devolve o system prompt da skill (para injetar no agente).
func (s Skill) Prompt() string {
	var sb strings.Builder
	sb.WriteString("Skill: " + s.Name + " — " + s.Description + "\n")
	sb.WriteString(s.Instructions)
	return sb.String()
}
