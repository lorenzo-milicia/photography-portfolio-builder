# Copilot Agent Instructions - Photography Portfolio Builder

## Project Overview

**Photography Portfolio Builder** - Go CLI tool (~4k LOC) generating static photography portfolio websites. Processes images (responsive variants + thumbnails), provides HTMX-based builder UI, generates static HTML, uploads to S3/R2.

**Stack:** Go 1.25+, Cobra CLI, HTMX, AWS SDK v2, imaging/webp libraries, YAML content, go:embed templates
**Module:** `go.lorenzomilicia.dev/photography-portfolio-builder`

## Build & Test

**Prerequisites:** Go 1.25.4+ (no additional tools needed)

**Build binary (required before running):**
```bash
go build -o bin/builder ./cmd
```
- First build: ~42s (downloads dependencies), subsequent: ~2s
- Output: `bin/builder` (28MB)
- **ALWAYS build before testing CLI commands**

**Run tests:**
```bash
go test ./...
```

**Pre-commit validation:**
```bash
go fmt ./...              # Format code
go build -o bin/builder ./cmd  # Verify build
go mod verify             # Verify dependencies
go mod tidy               # If adding/removing deps
```

## Project Structure

```
cmd/                    # CLI: main.go, cli/*.go (root, builder_serve, images_process, images_upload, website_build, website_serve)
internal/               # Core: builder/ content/ generator/ processing/ uploader/ layouts/ util/
assets/                 # Embedded (go:embed): templates/builder/, templates/site/, static/
scripts/                # run_generate.sh, migrate_layout_filenames.go
.generate.air.toml      # Air config (auto-reload)
go.mod, go.sum          # Go modules
```

## CLI Commands

All commands support `--env <file>` to load environment variables.

**Commands:** `builder serve`, `images process`, `images upload`, `website build`, `website serve`

**Common workflows:**
```bash
# Development UI
./bin/builder builder serve -p 8080 -c content -o dist

# Process images
./bin/builder images process -i photos -o dist/images [--force]

# Generate static site
./bin/builder website build -c content -o dist [--host https://cdn.example.com]

# Preview site
./bin/builder website serve -d dist -p 8000

# Upload to S3/R2
./bin/builder images upload -i dist/images -b bucket -r auto --endpoint URL --base-url URL [--dry-run]
```

**Environment variables:** `IMAGE_HOST` or `IMAGE_URL_PREFIX` (CDN URLs), AWS credentials (standard SDK vars)

## Architecture

**Templates:** Base templates in `assets/templates/site/` (go:embed), custom overrides in `<content>/templates/`. See README.md lines 50-107 for override blocks.

**Image Processing:** `internal/processing/` creates 480w/800w/1200w/1920w variants + 300w thumbnails, strips EXIF, supports WebP.

**Content:** YAML files in `<content>/projects/` and `<content>/photos/`. Models in `internal/content/`.

## Common Issues

**Build timing:** First build 40-45s, subsequent 1-2s. Wait up to 60s on first build for downloads.

**Missing dirs:** `.gitignore` excludes `/content/`, `/photos/`, `/dist/`, `/output/`. Create if needed: `mkdir -p content dist photos`

**Air:** `.generate.air.toml` exists but `air` NOT installed - don't assume availability.

**No CI/CD:** No GitHub Actions workflows. Manual validation only: `go fmt`, build, test, manually verify CLI.

## Making Changes

**CLI commands (`cmd/cli/`):** Edit `.go` file → rebuild → test command → update README.md if user-facing

**Templates (`assets/templates/`):** Edit template → **MUST rebuild** (go:embed) → test with `website build`

**Image processing:** Edit code → rebuild → test: `./bin/builder images process -i photos -o dist/images --force`

**Site generation:** Edit code → rebuild → test: `website build` + `website serve` → verify in browser

**Dependencies:** Import → `go mod tidy` → `go mod verify` → commit go.mod/go.sum

## Quick Reference

**Key files:** `cmd/main.go` (entry), `cmd/cli/root.go` (root cmd), `internal/generator/generator.go` (site gen, 21k LOC), `internal/builder/server.go` (builder UI, 20k LOC), `assets/assets.go` (embeds)

**Git ignores:** `/bin/`, `/content/`, `/photos/`, `/dist/`, `/output/`, `tmp/`, `.env`

**Trust these instructions** - validated by building/testing. Only search codebase if: instructions incomplete, commands fail, or need implementation details not covered here.
