# Project guidance

Build on the existing app shell. `README.md` defines the requirements,
validation rules, JSON contract, and submission expectations.

## Project structure

```text
.
├── AGENTS.md
├── README.md
├── NOTES.md
├── Makefile
├── go.mod                       # Single Go module: leadcapture
├── sqlc.yaml
├── leads.csv
├── api/
│   ├── cmd/leads/
│   │   ├── main.go               # serve/import entry point; embeds dist
│   │   └── dist/                 # Generated Vue build; gitignored
│   └── internal/
│       ├── db/                   # SQLite setup and sqlc-generated Go
│       │   ├── schema.sql
│       │   └── queries.sql
│       ├── lead/                 # Domain types, normalization, validation
│       ├── httpapi/              # Gin routes and handlers
│       └── csvimport/            # CSV parsing and row diagnostics
└── web/
    ├── vite.config.ts
    └── src/
        ├── App.vue
        ├── api/leads.ts          # Typed API calls
        ├── types/lead.ts         # API contract types
        ├── components/
        │   ├── LeadForm.vue
        │   └── LeadsView.vue
        ├── main.ts
        └── style.css
```

This is the target layout; create files only as needed.
Go tests live alongside the code they exercise.

## Backend

- Keep backend code under `api/` and retain the root Go module.
- Use Gin, `database/sql`, `modernc.org/sqlite`, and sqlc. Keep CGO disabled.
- Maintain SQL and database code in `api/internal/db/`. Configure sqlc in the
  root `sqlc.yaml`; run `sqlc generate` from the repository root. Commit generated
  Go files with SQL changes; never edit generated files manually.
- Keep `main.go` thin. Avoid unnecessary layers and generic abstractions.
- Share normalization and validation between CSV import and HTTP creation.
- Enforce email uniqueness in SQLite and translate failures into field errors.
- Preserve the README contract, including nulls, empty arrays, and HTTP statuses.
- Use `gofmt`, explicit error handling, and context-aware database calls.

## Frontend and embedding

- Retain Vue 3, TypeScript, `<script setup>`, Tailwind v4, and existing components.
- Follow the existing two-space, single-quote, semicolon-free TypeScript style.
- Centralize typed API calls; use relative `/api` URLs and show API field errors.
- Set Vite `build.outDir` to `../api/cmd/leads/dist` and `emptyOutDir` to `true`.
- Embed `dist` from `api/cmd/leads` using Go's `embed` package.
  Build the frontend before compiling or testing packages that embed it.
- Preserve the existing Make targets; build/run Go through `./api/cmd/leads`.
  Update cleanup and ignore rules for the new output directory.
- Do not commit build output, local databases, secrets, or installed dependencies.

## Workflow and checks

- Plan and obtain approval before non-trivial changes unless told to skip planning.
- Keep scope small; complete required behavior before stretch goals.
- Test validation boundaries, CSV diagnostics, persistence, and HTTP contracts.
- Verify implementation changes with `make test`, `make build`, and
  `npm --prefix web run lint`; report checks not run.
- Record decisions, next steps, and an actual rejected AI suggestion in `NOTES.md`.
