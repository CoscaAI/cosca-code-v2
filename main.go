// Command cosca-code é o COSCA CODE (codename COSCA STUDIO) — o ambiente de
// engenharia de software AI-native. Dois modos:
//   - desktop (padrão): empacota o frontend (React/Vite) no binário via Wails;
//   - serve <dir>: sobe só o backend HTTP (desenvolvimento com npm run dev).
package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/CoscaAI/cosca-code/internal/doctor"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Modo CLI (dev): `cosca-code serve <dir>` sobe só o backend HTTP.
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		root := "."
		if len(os.Args) > 2 {
			root = os.Args[2]
		}
		runServe(root)
		return
	}
	// Modo benchmark: `cosca-code bench <n>` mede performance em n arquivos.
	if len(os.Args) > 1 && os.Args[1] == "bench" {
		n := 1000
		if len(os.Args) > 2 {
			fmt.Sscanf(os.Args[2], "%d", &n)
		}
		runBench(n)
		return
	}
	// Modo doctor (mandamento 23 da doutrina): `cosca-code doctor [porta]`
	// verifica a saúde de cada subsistema via endpoints de LEITURA — somente
	// observa, nunca executa. Alinhado à P14 e à ordem do Don (read-only).
	if len(os.Args) > 1 && os.Args[1] == "doctor" {
		port := "14126"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		checks, ok := doctor.Run(context.Background(), "http://127.0.0.1:"+port)
		doctor.Print(checks, ok)
		if !ok {
			os.Exit(1)
		}
		return
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "COSCA CODE",
		Width:     1440,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 14, B: 20, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
