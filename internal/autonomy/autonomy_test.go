package autonomy

import "testing"

func TestPolicyReadOnly(t *testing.T) {
	p := &Policy{Level: ReadOnly}
	if !p.Allow(ActionRead, "x") {
		t.Error("read deveria ser permitido em read_only")
	}
	if p.Allow(ActionWrite, "x") {
		t.Error("write deveria ser negado em read_only")
	}
	if p.Allow(ActionExec, "x") {
		t.Error("exec deveria ser negado em read_only")
	}
	if p.Allow(ActionDestructive, "x") {
		t.Error("destructive deveria ser negado em read_only")
	}
}

func TestPolicyApprovalAsks(t *testing.T) {
	// Sem approver: "ask" vira deny (fail-safe).
	p := &Policy{Level: Approval}
	if !p.Allow(ActionRead, "x") {
		t.Error("read deveria ser permitido")
	}
	if p.Allow(ActionWrite, "x") {
		t.Error("write deveria pedir aprovação (e negar sem approver)")
	}

	// Com approver que aprova: write é permitido.
	pApproved := &Policy{Level: Approval, Approve: func(a Action, d string) bool { return true }}
	if !pApproved.Allow(ActionWrite, "x") {
		t.Error("write com aprovação deveria ser permitido")
	}

	// Com approver que nega: write é negado.
	pDenied := &Policy{Level: Approval, Approve: func(a Action, d string) bool { return false }}
	if pDenied.Allow(ActionWrite, "x") {
		t.Error("write com negação deveria ser negado")
	}
}

func TestPolicyAutonomous(t *testing.T) {
	p := &Policy{Level: Autonomous}
	if !p.Allow(ActionWrite, "x") || !p.Allow(ActionExec, "x") {
		t.Error("write/exec deveriam ser permitidos em autonomous")
	}
	if p.Allow(ActionDestructive, "x") {
		t.Error("destructive deveria pedir aprovação em autonomous")
	}
}

func TestParseLevel(t *testing.T) {
	if ParseLevel("read_only") != ReadOnly {
		t.Error("parse read_only")
	}
	if ParseLevel("full") != Full {
		t.Error("parse full")
	}
	if ParseLevel("desconhecido") != ReadOnly {
		t.Error("nível desconhecido deveria cair em read_only (default seguro)")
	}
}
