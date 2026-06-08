# VibeDeploy

English | [Simplified Chinese](./README.md)

VibeDeploy is a local workbench for multi-project development, database analysis, code generation, deployment operations, and API debugging. It uses a Go + Gin backend and a React frontend. The repository keeps part of the easy-deploy foundation, but the current product experience is centered on project cards, deployment routes, script workflows, database browsing, table lineage, and AI-assisted generation.

This is not just an admin-template repository. It is a practical console for keeping project metadata, deployment commands, database structure, API documentation, script workflows, and generation settings in one place.

## What The Project Pool Is For

The "Project Pool" navigation entry is the main "my projects" area in this app. It does not contain the business source code itself. Instead, each card represents how a business project is managed inside VibeDeploy.

You can use it to:

- Maintain project metadata, such as name, language, group, description, access URL, and local path.
- Configure deployment routes, such as local full deployment, local incremental deployment, remote incremental deployment, and dependency image builds.
- Run and stop deployment commands directly from project cards.
- View live deployment logs.
- Bind remote routes to server configuration.
- Open project script management when a project needs attached deployment or helper scripts.

In short, the Project Pool is the operational entry point for project management and deployment actions. The actual business code still lives in the local path or repository you configure.

## Main Features

### Project Pool

- Project groups, search, creation, editing, and deletion.
- Per-project deployment routes with custom commands and presentation colors.
- Local execution, remote execution, image build, and incremental deployment modes.
- Deployment log panel for real-time command output.

### Configuration Management

- Active project management.
- Database connection management with environment labels.
- Server connection management.
- AI provider, model, Base URL, API key, and default provider configuration.

### Database Browser And SQL Query

- Browse remote databases, tables, and table comments.
- Preview table data, field comments, primary-key markers, and DDL.
- Edit, delete, or generate table data where the backend supports write operations.
- Run read-only SQL queries and keep successful query history.
- The codebase includes support paths for MySQL, PostgreSQL, Oracle, SQL Server, SQLite, and ClickHouse. Write capability depends on the specific backend implementation.

### Table Relations

- Open database browsing, SQL query, and table-relation settings from a data-source card.
- Explore related data paths by keyword.
- Maintain field-level table relations.
- View incoming and outgoing lineage grouped by table.
- Import AI-analyzed table relations through a dedicated API.

### Code Generation

- Manage generation project cards grouped by business type.
- Configure frontend/backend project type, instances, path groups, models, and templates.
- Maintain database template scripts and generate SQL snippets.
- After generation, show absolute file paths and a prompt URL that can be handed to Codex.

### Script Library

- Manage script categories, workflows, and ordered steps.
- Step types include local execution, local upload, target download, and target execution.
- Resource bindings can inject database, server, or custom variables.
- Multi-stage pipelines inject server environment variables so scripts can reference stage information directly.

### Interface Forwarding And API Testing

- Import Swagger JSON and build a project/service/interface tree.
- Manage interface environments, users, headers, request params, response params, and logs.
- Use the built-in interface test panel for backend debugging and integration work.

### Table Samples And Agile Requests

- Save common business table selections as reusable table-sample plans.
- Use Agile Request as a lightweight API client.
- Import requests from Chrome "Copy as fetch".
- Persist request history in the backend for later review and reuse.

## Tech Stack

| Layer | Technologies |
| --- | --- |
| Frontend | React 18, Vite 5, TypeScript, Zustand, Tailwind CSS, Monaco Editor |
| Backend | Go 1.24, Gin, Gorm, Viper, Zap |
| Database / Connections | MySQL, PostgreSQL, Oracle, SQL Server, SQLite, ClickHouse, optional Redis |
| Desktop | Wails v2 |
| Build / Deployment | Docker, docker compose, Makefile, Kubernetes templates |

## Repository Layout

```text
.
├── web-react/                  # React frontend
│   ├── src/api/                # Frontend API wrappers
│   ├── src/components/         # Shared components, such as database browser and table preview
│   ├── src/views/              # Product pages
│   └── package.json
├── server/                     # Go backend and Wails desktop entry
│   ├── api/v1/                 # HTTP APIs
│   ├── cmd/http/               # Pure HTTP development entry
│   ├── config/                 # Config structures
│   ├── initialize/             # Initialization, routing, database setup
│   ├── model/                  # Data models
│   ├── resource/               # Code-generation and deployment templates
│   ├── router/                 # Gin router registration
│   ├── service/                # Business logic
│   └── wails.json
├── docs/                       # Feature and packaging notes
├── scripts/restart-dev.sh      # Local frontend/backend startup script
├── docker-compose.yml
└── Makefile
```

## Local Development

### Requirements

- Go 1.24+
- Node.js 18+
- npm 9+
- A usable system database configuration. The default template is `server/config.template.yaml`.

### Start Both Services

Run from the repository root:

```bash
./scripts/restart-dev.sh restart
```

The script will:

- Generate a temporary backend config from `server/config.template.yaml`.
- Start the backend and frontend on fixed default ports: backend `23638`, frontend `29527`.
- Start the backend with `go run ./cmd/http`.
- Start the frontend with Vite and set `VITE_BASE_API` to the backend URL.
- Run `npm ci` first if `web-react/node_modules` does not exist.

The terminal prints the fixed URLs:

```text
Backend:  http://localhost:23638
Frontend: http://localhost:29527
```

Stop recorded development services:

```bash
./scripts/restart-dev.sh stop
```

Temporarily override ports:

```bash
BACKEND_PORT=8008 FRONTEND_PORT=5175 ./scripts/restart-dev.sh restart
```

The development login page is prefilled with:

```text
username: admin
password: 123456
```

Whether this account works depends on your local database initialization.

### Manual Startup

Backend:

```bash
cd server
go run ./cmd/http -c /path/to/config.yaml
```

Frontend:

```bash
cd web-react
npm install
VITE_BASE_API=http://localhost:8008 npm run dev
```

## Build

Build the frontend:

```bash
cd web-react
npm run build
```

The output goes to:

```text
server/frontend/dist
```

Build the backend HTTP entry:

```bash
cd server
go build ./cmd/http
```

For desktop packaging, see:

```text
docs/desktop-build.md
```

## Useful Docs

- `docs/start-dev.md`: local development startup notes.
- `docs/desktop-build.md`: Wails desktop packaging.
- `docs/ai-table-relations-import.md`: AI table-relation import API.
- `docs/pipeline-env-injection.md`: pipeline server environment variable injection.
- `docs/ai-deploy-phase1.md`: AI deployment workflow notes.

## Development Notes

- The frontend uses the shared Axios instance in `web-react/src/utils/request.ts`.
- The active project and active connection are stored in Zustand.
- Private APIs require `x-token`; the frontend attaches it after login.
- Table edits, deletes, and generated data write directly to the remote database. Confirm the target environment before using them.
- Database browsing, table relations, and code generation depend on project and data-source configuration in Configuration Management.

## License

This project evolved from easy-deploy and keeps the original open-source license file. Read [LICENSE](./LICENSE) before using, distributing, or commercializing it.
