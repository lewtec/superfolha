```markdown
# LaTeX Editor Web - Especificação Técnica

Construa um editor LaTeX web completo seguindo esta especificação.

## Stack Tecnológica

### Backend
- Go com stdlib HTTP router (ServeMux)
- GraphQL via gqlgen
- PostgreSQL com sqlc para queries
- dbmate para migrations (rodar no boot)
- libgit2 para operações Git
- cobra para CLI
- embed.FS para servir bundle Vite

### Frontend
- React + Relay (GraphQL client com codegen)
- DaisyUI (Tailwind CSS) para UI
- CodeMirror 6 para editor de código
- PDF.js para visualização de PDF
- React Router para navegação

### Deploy
- Binário único que embeda frontend
- Docker com base texlive/texlive:latest
- TexLive completo instalado

## Configuração CLI

```bash
./server --state-dir=/var/latex-editor --db="postgres://..."
```

Argumentos via cobra, suportam variáveis de ambiente:
- `--state-dir` (env: STATE_DIR) - diretório para repositórios Git
- `--db` (env: DATABASE_URL) - connection string PostgreSQL

## Arquitetura Backend

### Estrutura de Diretórios
```
/cmd/server
  main.go

/internal
  /server       (HTTP handlers, GraphQL resolvers)
  /db          (sqlc generated code, migrations)
  /git         (libgit2 wrapper operations)
  /compiler    (compilação LaTeX)
  /auth        (JWT, bcrypt)

/web         (Vite bundle embedded aqui)
/migrations  (dbmate SQL files)
```

### Rotas HTTP
```
GET  /*          - serve SPA (index.html) para qualquer não-API
POST /api/graphql - GraphQL endpoint
POST /api/compile - compilação LaTeX
```

### Database Schema

```sql
-- users table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

-- projects table
CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  git_path TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_projects_user_id ON projects(user_id);
```

### GraphQL Schema

```graphql
type User {
  id: ID!
  email: String!
}

type Project {
  id: ID!
  name: String!
  createdAt: Time!
  updatedAt: Time!
}

type File {
  path: String!
  content: String!
}

type Commit {
  hash: String!
  message: String!
  author: String!
  date: Time!
}

type Query {
  me: User
  projects: [Project!]!
  project(id: ID!): Project
  files(projectId: ID!): [File!]!
  file(projectId: ID!, path: String!): File
  history(projectId: ID!): [Commit!]!
}

type Mutation {
  register(email: String!, password: String!): AuthPayload!
  login(email: String!, password: String!): AuthPayload!
  createProject(name: String!): Project!
  deleteProject(id: ID!): Boolean!
  saveFile(projectId: ID!, path: String!, content: String!): File!
  deleteFile(projectId: ID!, path: String!): Boolean!
  commit(projectId: ID!, message: String!): Commit!
}

type AuthPayload {
  token: String!
  user: User!
}
```

### Compilação LaTeX

Endpoint: `POST /api/compile`

**Request:**
- Content-Type: multipart/form-data
- Field: `tarball` (arquivo .tar.gz com projeto)

**Process:**
1. Extrai tarball em `/tmp/compile-{uuid}/`
2. Detecta `main.tex` ou único arquivo `.tex` no root
3. Executa: `pdflatex -synctex=1 -interaction=nonstopmode <file>.tex`
4. Captura stdout/stderr para logs
5. Retorna PDF + logs + synctex.gz
6. Cleanup do diretório temporário

**Response JSON:**
```json
{
  "pdf": "base64-encoded-pdf",
  "logs": "compilation output",
  "synctex": "base64-encoded-synctex.gz",
  "success": true
}
```

### Git Operations (libgit2)

Operações necessárias:
- `InitRepo(path)` - cria novo repositório
- `AddAll(repoPath)` - adiciona arquivos ao index
- `Commit(repoPath, author, message)` - cria commit
- `GetHistory(repoPath)` - retorna lista de commits
- `ReadFile(repoPath, path)` - lê arquivo do repo
- `WriteFile(repoPath, path, content)` - escreve arquivo
- `DeleteFile(repoPath, path)` - remove arquivo

### Inicialização

1. Parse CLI args com cobra
2. Conecta PostgreSQL
3. Roda migrations dbmate
4. Inicia servidor HTTP
5. Serve bundle embedded em `/`
6. Expõe GraphQL em `/api/graphql`
7. Expõe compilador em `/api/compile`

## Arquitetura Frontend

### Estrutura de Diretórios
```
/src
  /components
    Editor.jsx          (CodeMirror wrapper)
    FileTree.jsx        (árvore de arquivos)
    PDFViewer.jsx       (PDF.js wrapper)
    Navbar.jsx          (barra superior)
    LogsPanel.jsx       (logs de compilação)
  /pages
    Login.jsx           (tela de login)
    Register.jsx        (tela de registro)
    Projects.jsx        (lista de projetos)
    EditorPage.jsx      (interface completa)
  /relay
    (generated code)
  /services
    autocomplete.js     (lógica de autocomplete)
  /lib
    compiler.js         (empacota tarball, chama API)

/public
  /completions
    core.json           (comandos LaTeX básicos)
    *.json             (pacotes LaTeX)
```

### Layout Principal (EditorPage)

```
┌─────────────────────────────────────────────┐
│ Navbar: Logo | Project Name | [Compile]     │
├──────────┬──────────────────────────────────┤
│          │ Toolbar: [Code] [PDF]            │
│ FileTree │──────────────────────────────────│
│          │                                  │
│ (DaisyUI │     CodeMirror OU PDF.js         │
│  menu)   │     (toggle via toolbar)         │
│          │                                  │
│          │                                  │
└──────────┴──────────────────────────────────┘
           └─ LogsPanel (collapsible) ────────┘
```

### Componentes DaisyUI

Use estes componentes DaisyUI:
- **Navbar**: barra superior
- **Menu**: árvore de arquivos na sidebar
- **Tabs**: toggle Code/PDF
- **Button**: ações (compile, save, etc)
- **Card**: lista de projetos
- **Modal**: confirmações (delete, etc)
- **Alert**: mensagens de erro/sucesso
- **Collapse**: panel de logs expansível
- **Form controls**: login/register

### Rotas React Router

```jsx
/login          - Login page
/register       - Register page
/projects       - Lista de projetos (protected)
/editor/:id     - Editor completo (protected)
```

Protected routes checam JWT e redirecionam para `/login` se não autenticado.

### Auth Flow

1. Login/Register → mutation GraphQL
2. Recebe JWT token
3. Armazena em localStorage
4. Relay network layer inclui token em header `Authorization: Bearer <token>`
5. Backend valida JWT em cada request

### Autocomplete (Client-side)

**Arquivos de Completions:**
- Converter LaTeX-cwl `.cwl` files para JSON
- Estrutura: `{ "commands": ["\\section{}", "\\textbf{}"], "environments": ["itemize", "equation"] }`
- Servir em `/public/completions/`

**Lógica:**
1. Carrega `core.json` na inicialização
2. Watcher no editor detecta `\usepackage{X}` via regex
3. Fetch `/completions/X.json` (cache no browser)
4. Merge comandos disponíveis
5. CodeMirror autocomplete provider usa lista merged
6. Trigger autocomplete ao digitar `\`

**Regex úteis:**
```javascript
// Detecta packages
/\\usepackage(?:\[.*?\])?\{([\w,]+)\}/g

// Detecta labels
/\\label\{([^}]+)\}/g

// Detecta citations (para .bib parsing futuro)
/\\cite\{([^}]+)\}/g
```

### Compilação Flow

1. Usuário clica "Compile"
2. Frontend empacota arquivos do projeto em tarball (usar lib tar.js)
3. POST multipart/form-data para `/api/compile`
4. Mostra loading state
5. Recebe PDF + logs
6. Troca para tab PDF, renderiza com PDF.js
7. Parseia logs, extrai erros com linha/coluna
8. Adiciona markers no CodeMirror para erros

**Parsing de Erros:**
Regex para logs pdflatex:
```
! LaTeX Error: ...
l.42 ...
```
Extrair número da linha e mostrar inline no editor.

### Criar Projeto Flow

1. Mutation `createProject(name)`
2. Backend:
   - Cria diretório em `state-dir/repos/{uuid}`
   - Inicializa Git repo (libgit2)
   - Cria `main.tex` template
   - Commit inicial
   - Insere no DB
3. Redirect para `/editor/{id}`

## Features MVP

### Essenciais
- [x] Auth (register, login com JWT)
- [x] CRUD projetos (create, list, delete)
- [x] Explorador de arquivos (FileTree navegável)
- [x] Editor com syntax highlighting LaTeX (CodeMirror)
- [x] Salvar arquivos (mutation saveFile)
- [x] Commit manual (mutation commit)
- [x] Compilar LaTeX → PDF
- [x] Visualizar PDF (PDF.js)
- [x] Logs de compilação com parsing de erros
- [x] Error markers inline no editor
- [x] Autocomplete básico (core LaTeX commands)
- [x] Upload de arquivos (imagens, .bib)

### Layout
- [x] Toggle Code/PDF via tabs
- [x] FileTree sidebar com DaisyUI menu
- [x] Logs panel collapsible na parte inferior
- [x] Navbar com botão compile
- [x] Responsive (mobile: toggle exclusivo, desktop: pode split)

## Detalhes de Implementação

### CodeMirror Setup
```javascript
import { EditorView, basicSetup } from "codemirror"
import { latex } from "@codemirror/legacy-modes/mode/stex"
import { StreamLanguage } from "@codemirror/language"
import { autocompletion } from "@codemirror/autocomplete"

const latexLanguage = StreamLanguage.define(latex)

// Theme: usar oneDark ou basicLight dependendo do DaisyUI theme
```

### PDF.js Setup
```javascript
import * as pdfjsLib from 'pdfjs-dist'

pdfjsLib.GlobalWorkerOptions.workerSrc = 
  'pdfjs-dist/build/pdf.worker.min.js'

// Renderizar PDF em canvas
```

### Relay Setup
```javascript
// RelayEnvironment com JWT token
const fetchQuery = async (operation, variables) => {
  const token = localStorage.getItem('token')
  const response = await fetch('/api/graphql', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token && { 'Authorization': `Bearer ${token}` })
    },
    body: JSON.stringify({ query: operation.text, variables })
  })
  return response.json()
}
```

### Template main.tex Inicial
```latex
\documentclass{article}
\usepackage[utf8]{inputenc}

\title{Untitled}
\author{}
\date{}

\begin{document}

\maketitle

\section{Introduction}

Your content here.

\end{document}
```

## Build e Deploy

### Development
```bash
# Frontend
cd frontend && npm install && npm run dev

# Backend
cd backend && go mod download
go run cmd/server/main.go --state-dir=./tmp --db="postgres://..."
```

### Production Build
```bash
# Build frontend
cd frontend && npm run build

# Build backend (embeds frontend)
cd backend && go build -o server cmd/server/main.go

# Docker
docker build -t latex-editor .
docker run -v /data:/data -p 8080:8080 latex-editor \
  --state-dir=/data/repos \
  --db="postgres://user:pass@host/db"
```

### Dockerfile
```dockerfile
FROM texlive/texlive:latest

# Install dependencies se necessário
RUN apt-get update && apt-get install -y ca-certificates

# Copy binary
COPY server /usr/local/bin/server

EXPOSE 8080

ENTRYPOINT ["server"]
```

## Notas Importantes

1. **Segurança**: Compilação LaTeX em `/tmp` isolado, cleanup obrigatório
2. **Performance**: Limitar compilações simultâneas (ex: 4-8 concurrent)
3. **Erros**: Sempre retornar logs mesmo em falha de compilação
4. **Git**: Cada projeto = repositório Git independente
5. **JWT**: Expiração de 7 dias, validação em todos os endpoints protegidos
6. **CORS**: Configurar se frontend dev server separado

## Testing

Implementar testes para:
- Auth flow (register, login, JWT validation)
- Git operations (init, commit, read/write files)
- Compilação (success, error handling, cleanup)
- GraphQL mutations/queries

Use testcontainers para Postgres em testes de integração.
```
