package mcp

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CoscaAI/cosca-code/internal/tool"
)

// fakeServerScript é um servidor MCP mínimo em Python (initialize + tools/list
// + tools/call), usado para testar o cliente sem depender de um servidor real.
const fakeServerScript = `import sys, json
def send(m):
    sys.stdout.write(json.dumps(m) + "\n"); sys.stdout.flush()
for line in sys.stdin:
    if not line.strip(): continue
    msg = json.loads(line)
    m = msg.get("method")
    if m == "initialize":
        send({"jsonrpc":"2.0","id":msg["id"],"result":{"protocolVersion":"2024-11-05","capabilities":{}}})
    elif m == "tools/list":
        send({"jsonrpc":"2.0","id":msg["id"],"result":{"tools":[{"name":"echo","description":"eco de texto","inputSchema":{"type":"object","properties":{"text":{"type":"string"}}}}]}})
    elif m == "tools/call":
        args = msg["params"]["arguments"]
        send({"jsonrpc":"2.0","id":msg["id"],"result":{"content":[{"type":"text","text":"echo: " + args.get("text","")}]}})
`

func TestMCPClientEcho(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 não disponível")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake_server.py")
	if err := os.WriteFile(script, []byte(fakeServerScript), 0o755); err != nil {
		t.Fatal(err)
	}

	client, err := Start("python3", script)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools()
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("ListTools = %+v, want [echo]", tools)
	}

	out, err := client.CallTool("echo", map[string]any{"text": "olá"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if out != "echo: olá" {
		t.Fatalf("CallTool = %q, want 'echo: olá'", out)
	}
}

func TestRegisterTools(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 não disponível")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake_server.py")
	os.WriteFile(script, []byte(fakeServerScript), 0o755)

	client, err := Start("python3", script)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client.Close()

	reg := tool.NewRegistry()
	if err := RegisterTools(client, reg); err != nil {
		t.Fatalf("RegisterTools: %v", err)
	}
	if _, ok := reg.Get("echo"); !ok {
		t.Fatal("tool echo não foi registrada")
	}
	result, err := reg.Call(nil, "echo", map[string]any{"text": "x"})
	if err != nil || result != "echo: x" {
		t.Fatalf("reg.Call = %q, %v", result, err)
	}
}
