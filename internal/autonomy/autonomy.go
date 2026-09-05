// Package autonomy é o Safe Autonomy do COSCA CODE (spec seção 85): níveis de
// autonomia configuráveis por workspace. Cada operação é classificada por
// risco (read/write/exec/destructive), e a Policy decide allow/ask/deny
// conforme o nível. Nenhum agente age sem passar pela policy.
package autonomy

// Level é o nível de autonomia (spec §85).
type Level int

const (
	ReadOnly   Level = iota // só leitura — write/exec/destructive negados
	Assisted                // com assistência — destrutivo pede aprovação
	Approval                // write/exec/destructive pedem aprovação
	Autonomous              // age sozinho (destrutivo ainda pede aprovação)
	Full                    // controle total
)

// String devolve o nome legível do nível.
func (l Level) String() string {
	switch l {
	case ReadOnly:
		return "read_only"
	case Assisted:
		return "assisted"
	case Approval:
		return "approval"
	case Autonomous:
		return "autonomous"
	case Full:
		return "full"
	default:
		return "unknown"
	}
}

// ParseLevel converte uma string para Level.
func ParseLevel(s string) Level {
	switch s {
	case "read_only":
		return ReadOnly
	case "assisted":
		return Assisted
	case "approval":
		return Approval
	case "autonomous":
		return Autonomous
	case "full":
		return Full
	default:
		return ReadOnly // default seguro
	}
}

// Action classifica o risco de uma operação.
type Action string

const (
	ActionRead        Action = "read"        // ler, listar, buscar
	ActionWrite       Action = "write"       // editar, salvar, criar
	ActionExec        Action = "exec"        // rodar comando, teste
	ActionDestructive Action = "destructive" // deletar, reset, force
)

// Decision é a decisão da policy para uma ação.
type Decision int

const (
	Allow Decision = iota
	Ask
	Deny
)

// Approver é uma função que pergunta ao usuário (retorna true para aprovar).
// Usada nos níveis Assisted/Approval para ações que exigem confirmação.
type Approver func(action Action, detail string) bool

// Policy decide allow/ask/deny para cada ação, dado um nível.
type Policy struct {
	Level   Level
	Approve Approver // nil = nega quando "ask" (fail-safe)
}

// Decide devolve a decisão para uma ação no nível atual.
func (p *Policy) Decide(action Action) Decision {
	switch p.Level {
	case ReadOnly:
		if action == ActionRead {
			return Allow
		}
		return Deny
	case Assisted:
		if action == ActionRead || action == ActionWrite {
			return Allow
		}
		if action == ActionExec {
			return Allow
		}
		return Ask // destrutivo pede aprovação
	case Approval:
		if action == ActionRead {
			return Allow
		}
		return Ask // write/exec/destructive pedem aprovação
	case Autonomous:
		if action == ActionDestructive {
			return Ask
		}
		return Allow
	case Full:
		return Allow
	default:
		return Deny
	}
}

// Allow executa a decisão com o approver (se "ask"). Retorna true se a ação
// pode prosseguir.
func (p *Policy) Allow(action Action, detail string) bool {
	switch p.Decide(action) {
	case Allow:
		return true
	case Deny:
		return false
	case Ask:
		if p.Approve != nil {
			return p.Approve(action, detail)
		}
		return false // fail-safe: sem approver, nega
	}
	return false
}
