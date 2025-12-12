# 📘 Project Instructions for GitHub Copilot Agent  
## **Project: Photography Portfolio Builder & Static Site Generator**  
# Project: Photography Portfolio Builder — Developer Notes

This file contains concise instructions for running and testing the project locally. Keep it short — other details are in the repo.

Quick commands (from repository root):

- Build the binary:

```bash
just build
```

- Run the interactive builder server (local development):

```bash
just run        # runs the VS Code task or ./bin/builder builder
```

- Generate static site for production (no prefix):

```bash
just generate-prod
```

- Generate static site for preview (prefixes assets with /preview):

```bash
````instructions
# Project: Photography Portfolio Builder — Developer Notes

Keep this file short. Always use the VS Code tasks (not manual shell commands) to build, generate and run the server — that ensures terminals are managed consistently.

VS Code task names (open `Terminal -> Run Task`):

- `Build` — compiles `./cmd/builder` (runs `go build`)
- `Run` — starts the interactive builder server (development)
- `Run (port)` — same as `Run` but prompts for a port
- `Generate Site (Production)` — builds and generates production site (root asset paths)
- `Generate Site (Preview)` — builds and generates preview site (assets prefixed with `/preview`)
- `Serve Static Site` — serves `output/public` for manual inspection

How to use (recommended):

1. Run `Build` once to compile (or use the `Build` task in the status bar).
2. Use `Run` to start the builder UI while developing.
3. Use `Generate Site (Preview)` to preview generated output under `/preview/`.
4. Use `Generate Site (Production)` to produce final site files with root asset paths.
5. Use `Serve Static Site` and open `http://localhost:8000/` to inspect `output/public`.

Do not run the `just` commands from the instructions — use the VS Code tasks listed above.

## **3. Static Site Generation**
  - Project editor  
  - Photo browser  
  - Layout editor  
  - Generate/preview  

Left sidebar: project list  
Main area: editor

---

# ✔️ Acceptance Criteria
1. Create/edit/delete projects  
2. Upload/list photos  
3. Edit layout type + parameters  
4. Generate complete static website  
5. Preview site locally  
6. Responsive images work  
7. Builder UI functions with htmx  
8. No Cloudflare/Git code  

---

# 📋 Implementation Plan

## **Phase 1 — Scaffolding**
- Create directories  
- Initialize Go modules  
- Implement basic router + static file serving  

## **Phase 2 — Content Layer**
- YAML load/save utilities  
- Project CRUD  
- Photo upload handler  

## **Phase 3 — Builder UI**
- Templates: `/templates/builder`  
- Implement project list, editor, photo list, layout editor  

## **Phase 4 — Image Processing**
- Thumbnails  
- Responsive variants  
- Metadata caching  

## **Phase 5 — Layout Algorithms**
- Justified layout  
- Grid layout  
- Manual grid layout  

## **Phase 6 — Static Generator**
- Generate index  
- Generate gallery pages  
- Copy CSS  
- Write images  

## **Phase 7 — Preview**
- Serve `/output/public`  
- UI button to trigger regeneration  

## **Phase 8 — Polish**
- Error handling  
- Logs  
- UX adjustments  

---

# 🧪 Test Cases

### Functional
- YAML created/updated correctly  
- Images upload & resize  
- Layout YAML reads/writes  
- Generator outputs correct HTML structure  

### UI
- HTMX partial refresh:  
  - Photo upload  
  - Layout update  
  - Project rename  

---

# 🏁 Final Deliverable
A modular Go application that:

- Provides a full builder interface  
- Allows managing photos + projects  
- Supports configurable layout options  
- Generates a complete static site  
- Can be previewed locally  
- Contains no deployment/CI/Git logic  

---

# 🔧 Development Workflow

## Running the Application
**Always use VS Code tasks to run the application** so it stays running in a separate terminal.

- Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
- Type "Run Task" and select "Tasks: Run Task"
- Choose "Run Builder" to start the server on port 8080
- Or choose "Run Builder (custom port)" to specify a different port

This keeps the server running in the background while you continue development work.

## Testing Direct Navigation
When testing routes, ensure that direct navigation to project URLs (e.g., `http://localhost:8080/project/some-slug`) returns the full HTML page, not just partial content. The server checks for the `HX-Request` header to determine whether to return the full page or just the htmx partial.

## Preview vs Production Builds
The static site generator accepts a `baseURL` parameter that prefixes all asset references and links:

- **Preview Mode**: When generating from the builder UI, the generator uses `/preview` as the base URL. All assets and links are prefixed (e.g., `/preview/static/css/site.css`). The builder serves the preview at `http://localhost:8080/preview/`.

- **Production Mode**: For final deployment, the generator should be called with an empty string `""` as the base URL, resulting in absolute paths from root (e.g., `/static/css/site.css`). This is the correct format for static hosting.

**Important**: The generated HTML in `output/public/` is configured for preview by default. Before deploying to production, regenerate the site with an empty base URL or modify the generator call in `handleGenerate` to use `""` instead of `"/preview"`.


