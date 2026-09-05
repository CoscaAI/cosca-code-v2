// Package workspace é o Workspace Engine do COSCA CODE (spec seções 32, 48,
// 50, 70): abre um projeto, detecta sua natureza (linguagem, build, testes,
// framework), constrói a árvore de arquivos e observa mudanças, emitindo
// eventos no Event Bus. A UI nunca toca o filesystem diretamente — tudo passa
// por este "backend isolado".
package workspace

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectInfo é o "PROJECT MAP" detectado ao abrir um projeto (spec seção 48):
// o que o Cosca infere automaticamente, sem configuração manual.
type ProjectInfo struct {
	Root           string   `json:"root"`
	Language       string   `json:"language"`        // primária
	Languages      []string `json:"languages"`       // todas detectadas
	Framework      string   `json:"framework"`       // ex.: react, next, django
	PackageManager string   `json:"package_manager"` // ex.: "go mod", "npm", "cargo"
	BuildCommand   string   `json:"build_command"`
	TestCommand    string   `json:"test_command"`
	HasGit         bool     `json:"has_git"`
	HasDocker      bool     `json:"has_docker"`
	HasCI          bool     `json:"has_ci"`
	HasDocs        bool     `json:"has_docs"`
	// ProjectType é o MODO do Cosca Editor (§6): tipo declarado no
	// Project Manifest (.cosca/project.yaml) do Cosca Engine — editor,
	// image, cinema, music, game, scientific, 3d, animation, document, lab.
	// Sem manifest, o tipo default é "editor" (texto/código).
	ProjectType string `json:"project_type"`
	// ProjectTypeSource indica de onde o tipo veio: "manifest" (project.yaml)
	// ou "default" (sem manifesto → editor/código).
	ProjectTypeSource string `json:"project_type_source,omitempty"`
}

// marker associa um arquivo de configuração a uma linguagem + comandos padrão.
type marker struct {
	file  string
	lang  string
	build string
	test  string
}

var markers = []marker{
	{"go.mod", "go", "go build ./...", "go test ./..."},
	{"Cargo.toml", "rust", "cargo build", "cargo test"},
	{"package.json", "javascript", "npm run build", "npm test"},
	{"pyproject.toml", "python", "pip install -e .", "pytest"},
	{"requirements.txt", "python", "pip install -r requirements.txt", "pytest"},
	{"setup.py", "python", "pip install -e .", "pytest"},
	{"Pipfile", "python", "pipenv install", "pytest"},
	{"pom.xml", "java", "mvn package", "mvn test"},
	{"build.gradle", "java", "gradle build", "gradle test"},
	{"build.gradle.kts", "kotlin", "gradle build", "gradle test"},
	{"Gemfile", "ruby", "bundle install", "bundle exec rake test"},
	{"composer.json", "php", "composer install", "composer test"},
	{"Package.swift", "swift", "swift build", "swift test"},
	{"pubspec.yaml", "dart", "dart pub get", "dart test"},
	{"tsconfig.json", "typescript", "npm run build", "npm test"},
}

// ProjectManifestFile é o caminho do Project Manifest do Cosca Engine
// (criado por `cosca project new --type <tipo>` — Fase 1.1).
const ProjectManifestFile = ".cosca/project.yaml"

// Modos do Cosca Editor (§6) — tipos válidos do Project Manifest.
var validProjectTypes = map[string]bool{
	"editor": true, "image": true, "cinema": true, "music": true,
	"game": true, "scientific": true, "3d": true, "animation": true,
	"document": true, "lab": true,
}

// Detect analisa o diretório root e devolve o ProjectInfo inferido. É
// puramente baseado em evidência (arquivos marcadores) — nunca inventa.
func Detect(root string) ProjectInfo {
	info := ProjectInfo{Root: root}

	seen := map[string]bool{}
	for _, m := range markers {
		if fileExists(filepath.Join(root, m.file)) {
			if !seen[m.lang] {
				info.Languages = append(info.Languages, m.lang)
				seen[m.lang] = true
			}
			if info.PackageManager == "" {
				info.PackageManager = pkgManagerFor(m.file)
			}
			if info.BuildCommand == "" {
				info.BuildCommand = m.build
			}
			if info.TestCommand == "" {
				info.TestCommand = m.test
			}
		}
	}

	sort.Strings(info.Languages)
	if len(info.Languages) > 0 {
		info.Language = info.Languages[0]
	}

	info.Framework = detectFramework(root, info.Language)
	info.HasGit = fileExists(filepath.Join(root, ".git")) || dirExists(filepath.Join(root, ".git"))
	info.HasDocker = fileExists(filepath.Join(root, "Dockerfile")) || fileExists(filepath.Join(root, "docker-compose.yml")) || fileExists(filepath.Join(root, "docker-compose.yaml"))
	info.HasCI = dirExists(filepath.Join(root, ".github", "workflows")) || fileExists(filepath.Join(root, ".gitlab-ci.yml"))
	info.HasDocs = fileExists(filepath.Join(root, "README.md")) || fileExists(filepath.Join(root, "README")) || dirExists(filepath.Join(root, "docs"))

	// Modo do Cosca Editor (§6): Project Manifest decide; sem manifest,
	// default "editor" (texto/código).
	info.ProjectType, info.ProjectTypeSource = detectProjectType(root)

	return info
}

// detectProjectType lê o tipo do Project Manifest (.cosca/project.yaml) do
// Cosca Engine. Sem manifesto (ou com tipo inválido), devolve "editor"
// como default (modo texto/código) com source "default".
func detectProjectType(root string) (string, string) {
	path := filepath.Join(root, ProjectManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "editor", "default"
	}
	// Parsing mínimo do YAML (type: <valor>) — sem dependência externa. O
	// tipo declarado tem prioridade; um manifest sem type cai no default.
	typ := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "type:") {
			typ = strings.TrimSpace(strings.TrimPrefix(line, "type:"))
			typ = strings.Trim(typ, `"'`)
			break
		}
	}
	if typ != "" && validProjectTypes[typ] {
		return typ, "manifest"
	}
	return "editor", "default"
}

func pkgManagerFor(file string) string {
	switch file {
	case "go.mod":
		return "go mod"
	case "Cargo.toml":
		return "cargo"
	case "package.json", "tsconfig.json":
		return "npm"
	case "pyproject.toml", "requirements.txt", "setup.py":
		return "pip"
	case "Pipfile":
		return "pipenv"
	case "pom.xml":
		return "maven"
	case "build.gradle", "build.gradle.kts":
		return "gradle"
	case "Gemfile":
		return "bundler"
	case "composer.json":
		return "composer"
	case "Package.swift":
		return "swiftpm"
	case "pubspec.yaml":
		return "pub"
	default:
		return ""
	}
}

// detectFramework infere o framework a partir de dependências declaradas.
func detectFramework(root, lang string) string {
	if lang == "go" {
		if data, err := os.ReadFile(filepath.Join(root, "go.mod")); err == nil {
			// Só dependências DIRETAS (ignora // indirect — Wails/echo/etc.
			// aparecem como transitivas e gerariam falso positivo).
			for _, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, "// indirect") {
					continue
				}
				if strings.Contains(line, "github.com/gin-gonic/gin") {
					return "gin"
				}
				if strings.Contains(line, "github.com/labstack/echo") {
					return "echo"
				}
			}
		}
		return ""
	}
	if lang == "javascript" || lang == "typescript" {
		data, err := os.ReadFile(filepath.Join(root, "package.json"))
		if err != nil {
			return ""
		}
		s := string(data)
		for _, fw := range []struct{ dep, name string }{
			{"\"next\"", "next"}, {"\"react\"", "react"}, {"\"vue\"", "vue"},
			{"\"svelte\"", "svelte"}, {"\"angular\"", "angular"}, {"\"express\"", "express"},
			{"\"fastify\"", "fastify"}, {"\"nestjs\"", "nest"},
		} {
			if strings.Contains(s, fw.dep) {
				return fw.name
			}
		}
	}
	if lang == "python" {
		data, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
		if err == nil {
			s := string(data)
			if strings.Contains(s, "django") {
				return "django"
			}
			if strings.Contains(s, "flask") {
				return "flask"
			}
			if strings.Contains(s, "fastapi") {
				return "fastapi"
			}
		}
	}
	return ""
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
