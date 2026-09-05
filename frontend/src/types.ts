// Tipos espelhando o backend Go (internal/workspace, internal/server).

export interface ProjectInfo {
  root: string
  language: string
  languages: string[]
  framework: string
  package_manager: string
  build_command: string
  test_command: string
  has_git: boolean
  has_docker: boolean
  has_ci: boolean
  has_docs: boolean
}

export interface FileNode {
  name: string
  path: string
  is_dir: boolean
  children?: FileNode[]
}

export interface Event {
  ID: string
  Type: string
  Timestamp: string
  Source: string
  Payload?: any
  CorrelationID?: string
}
