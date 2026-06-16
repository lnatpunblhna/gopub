# Repository Guidelines

## Project Structure & Module Organization
This repository contains a Go 1.25 Beego backend and a Vue 3 frontend.
Backend code and runtime assets live under `src/`: `main.go`, `controllers/`,
`models/`, `routers/`, `library/`, `conf/`, `views/`, and `static/`.
Frontend source is in `frontend/src/`, with Vite configuration in
`frontend/vite.config.js`. Deployment and packaging files are at the repository root,
including `Dockerfile`, `control`, `gopub.service`, and `gopub-kubernetes.yml`.
Go dependencies are resolved with modules; do not reintroduce `vendor/`.

## Build, Test, and Development Commands
- `cd frontend && npm ci`: install locked frontend dependencies.
- `cd frontend && npm run dev`: start the Vue development server.
- `cd frontend && npm run build`: build frontend assets for the Go app.
- `./control build`: compile the Go backend using the project control script.
- `./control init`: initialize database state after configuring `src/conf/app.conf`.
- `./control start|stop|restart`: manage the service at `http://127.0.0.1:8192`.
- `go test ./...`: run Go tests when present.

## Coding Style & Naming Conventions
Format Go code with `gofmt`; keep package names short, lowercase, and aligned with
their directories. Controllers are grouped by feature under `src/controllers/`
(`task`, `conf`, `api`, `walle`), so add new handlers beside related behavior.
Follow existing controller file naming, and prefer descriptive lowercase names
for new packages. Vue code uses Vue 3, Vuex 4, Vue Router 4, Vite, and Element
Plus; keep store modules under `frontend/src/store/` and routes in
`frontend/src/router/`. Prefer ESM imports over CommonJS.

## Testing Guidelines
There are currently no committed Go or JavaScript tests. Add Go tests as
`*_test.go` beside the package under test and favor table-driven cases for
library and controller helpers. If frontend tests are introduced, use
`*.spec.js` or `*.test.js` near the component or module being tested and document
the new npm script in `package.json`.

## Commit & Pull Request Guidelines
Recent history uses short, direct messages such as `change import dir` and
`fix`; keep commits focused and imperative. Pull requests should
describe the behavior change, list manual verification commands, mention config
or database impacts, and include screenshots for UI changes. Link related issues
when available and avoid mixing dependency upgrades with feature changes.

## Security & Configuration Tips
Do not commit real credentials, SSH keys, or production connection strings.
Use `src/conf/app.conf` for local configuration and environment variables for
Docker deployments. Treat deployment hooks and SSH commands as sensitive because
they execute on managed hosts.
