---
trigger: always_on
glob:
description:
---

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

**Templates:** Base templates in `assets/templates/site/` (go:embed), custom overrides in `<content>/templates/`. Template discovery is recursive - all `.html` files in subdirectories are loaded. See README.md for override blocks.

**Custom Assets:** `custom.css`/`custom.js` OR `site.css`/`site.js` in templates directory are discovered via `discoverCustomAssets()`, copied to output as `custom.css`/`custom.js`, and included after base assets in all pages.

**Image Processing:** `internal/processing/` creates 480w/800w/1200w/1920w variants + 300w thumbnails, strips EXIF, supports WebP.

**Content:** YAML files in `<content>/projects/` and `<content>/photos/`. Models in `internal/content/`.

**Hero Images:** Projects can have `hero_photo` field (12-char hash ID). Index page can display hero images using grid layout defined in `content/index-layout.yaml` (`IndexLayoutConfig` with desktop/mobile placements).

## Common Issues

**Build timing:** First build 40-45s, subsequent 1-2s. Wait up to 60s on first build for downloads.

**Missing dirs:** `.gitignore` excludes `/content/`, `/photos/`, `/dist/`, `/output/`. Create if needed: `mkdir -p content dist photos`

**Air:** `.generate.air.toml` exists but `air` NOT installed - don't assume availability.

**No CI/CD:** No GitHub Actions workflows. Manual validation only: `go fmt`, build, test, manually verify CLI.

## Making Changes

**CLI commands (`cmd/cli/`):** Edit `.go` file → rebuild → test command → update README.md if user-facing

**Templates (`assets/templates/`):** Edit template → **MUST rebuild** (go:embed) → test with `website build`

**CSS/JS assets (`assets/static/site/`):** Edit → **MUST rebuild** (go:embed) → test with `website build` + `website serve`. CSS uses variables like `--gallery-gap-desktop`, `--gallery-gap-mobile`, `--gallery-wrapper-padding-horizontal-desktop`.

**Image processing:** Edit code → rebuild → test: `./bin/builder images process -i photos -o dist/images --force`

**Site generation:** Edit code → rebuild → test: `website build` + `website serve` → verify in browser

**Content models (`internal/content/`):** Edit structs → rebuild → test YAML loading/saving. Key models: `ProjectMetadata` (includes `HeroPhoto`), `LayoutConfig`, `IndexLayoutConfig`, `IndexHeroPlacement`.

**Dependencies:** Import → `go mod tidy` → `go mod verify` → commit go.mod/go.sum

## Quick Reference

**Key files:** `cmd/main.go` (entry), `cmd/cli/root.go` (root cmd), `internal/generator/generator.go` (site gen), `internal/builder/server.go` (builder UI), `assets/assets.go` (embeds)

**Template functions:** Generator provides template functions like `calculateSizes`, `calculateMobileSizes`, `calculateIndexSizes`, `calculateIndexMobileSizes` for responsive image sizing. `sanitizeClass` for CSS class names.

**Git ignores:** `/bin/`, `/content/`, `/photos/`, `/dist/`, `/output/`, `tmp/`, `.env`

**UI/UX features:** Fade-in animations for project titles/images (CSS keyframes), scroll-based navbar controller reveal, mobile menu overlay, CSS variables for consistent spacing.

**Trust these instructions** - validated by building/testing. Only search codebase if: instructions incomplete, commands fail, or need implementation details not covered here.
