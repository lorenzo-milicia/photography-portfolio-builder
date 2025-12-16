# 📘 Project Instructions for GitHub Copilot Agent  
## **Project: Photography Portfolio Builder & Static Site Generator**  
# Project: Photography Portfolio Builder — Developer Notes

This file contains concise instructions for running and testing the project locally. Keep it short — other details are in the repo.

Quick commands (from repository root):

- Build binary:

```bash
go build -o bin/builder ./cmd
```

- Run interactive builder server (dev):

```bash
./bin/builder builder serve -p 8080 -c content -o dist
```

- Process images:

```bash
./bin/builder images process -i photos -o dist/images [--force]
```

- Upload processed images to S3/R2:

```bash
./bin/builder images upload -i dist/images -b <bucket> -r <region|auto> --endpoint <url> --base-url <public-url>
```

- Build static website:

```bash
./bin/builder website build -c content -o dist --host https://cdn.example.com
```

- Serve generated site for preview:

```bash
./bin/builder website serve -d dist -p 8000
```

Prefer using the provided VS Code tasks (Build / Run / Generate / Images) for consistent terminal management.

VS Code task names (open `Terminal -> Run Task`):

- `Build` — compiles the binary (`go build -o bin/builder ./cmd`)
- `Run` — runs the interactive builder (task starts `./bin/builder builder serve`)
- `Generate Site (Production)` — runs `website build` and writes to `dist`
- `Images: Process` — processes `photos` into `dist/images`
- `Images: Process (force)` — same as above with `--force`
- `Images: Upload (dry-run)` — simulates uploading `dist/images` to remote
- `Images: Upload to R2` — uploads `dist/images` to configured R2/S3
- `Serve Static Site` — serves `dist` for preview (`website serve -d dist -p 8000`)

How to use (recommended):

1. Run `Build` once to compile (or use the `Build` task in the status bar).
2. Use `Run` to start the builder UI while developing.
3. Use `Generate Site (Production)` for live testing of template/CSS changes (uses air for auto-regeneration on file changes).
4. Use `Generate Site (Preview)` to preview generated output under `/preview/`.
5. Use `Generate Site (Production)` to produce final site files with root asset paths.
6. Use `Serve Static Site` and open `http://localhost:8080/` to inspect `output/public`.

Note: Avoid manual shell commands — always use VS Code tasks for consistent terminal management.


## Project — Developer Quickstart

Short, focused instructions for local development and common flows.

### Tools & Frameworks
- Language: Go (modules)
- Frontend: Vanilla CSS + HTMX for builder UI
- Image processing: internal Go package (processor)
- Uploads: S3/R2 adapter (uploader)
- Build tooling: VS Code tasks (predefined) + `go build`

### Common Tasks
- Build binary: `go build -o bin/builder ./cmd`
- Run builder UI (dev): use VS Code task `Run` or `./bin/builder builder serve`
- Generate static site (production): use task `Generate Site (Production)` or `./bin/builder website build -o ./dist`
- Process images: use task `Images: Process` (or `./bin/builder images process -i ./photos -o ./dist/images`)
- Upload images: use task `Images: Upload` (configurable R2/S3 inputs)

Prefer VS Code tasks (Build / Run / Generate / Images) to keep terminals consistent.

### Glossary — main areas
- `cmd/` : CLI entrypoints (`builder`, subcommands)
- `assets/` : bundled static CSS/JS/templates used by builder and generated site
- `templates/` : HTML templates for builder UI and generated site
- `internal/generator` : static site generation logic
- `internal/images` : image processing utilities
- `internal/uploader` : S3/R2 upload adapters
- `internal/content` : models for `project` and `photo` and YAML utilities
- `projects/`, `photos/` : example content used in development

### Dev flow (recommended)
1. `Build` (task) to compile the binary.
2. `Run` (task) to open the builder UI and iterate using HTMX.
3. Edit templates/CSS; use `Generate Site (Production)` or `Generate Site (Preview)` to validate output.
4. Use `Images: Process` to create responsive variants, then `Images: Upload` to push to R2/S3 if needed.

### Notes
- Keep `projects/` and `photos/` synchronized with the builder during development.
- The builder UI uses HTMX headers; when testing direct links ensure full HTML is returned (not partial).

That's it — short and actionable. Keep this in sync as the project evolves.


