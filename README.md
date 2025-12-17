# Photography Portfolio Builder

A minimal CLI tool to manage projects and generate a static photography portfolio website.

This README is written for end users — no Go development knowledge required.

## Quick start — generate a website

1) Install the CLI (recommended):

```bash
go install go.lorenzomilicia.dev/photography-portfolio-builder/cmd@latest
```

2) Prepare images (optional):

```bash
builder images process -i photos -o dist/images
```

3) Generate the static site into `dist`:

```bash
builder website build -c content -o dist --host https://cdn.example.com
```

4) Preview the generated site locally:

```bash
builder website serve -d dist -p 8000
```

If you prefer not to install, run the same commands with `go run go.lorenzomilicia.dev/photography-portfolio-builder/cmd@latest` as a direct alternative.

## What the commands do

- `images process` — creates thumbnails and responsive image variants from your original photos.
- `website build` — generates the static HTML and assets into `dist` (references processed images or a remote host).
- `website serve` — serves the `dist` directory locally for preview.

## Quick tips

- If you do not process images locally, pass `--host` to `website build` pointing to a CDN or public image base URL.
- Keep your site content in `content/` (projects and metadata). This is the source used to generate the site.
- `photos/` is optional and used only by `images process`.

## Template customization

You can customize the look and feel of your generated website by providing custom templates that override the default templates.

**Default templates directory**: `content/templates`

You can specify a different directory using the `--templates` flag:

```bash
builder website build -c content -o dist --templates /path/to/custom/templates
```

### How template overrides work

1. The generator first loads the base templates (embedded in the CLI)
2. Then it loads any custom templates from your templates directory
3. Custom templates can override specific blocks defined in the base templates

### Available template blocks

The base templates define the following overrideable blocks:

**index.html** (home page):
- `index-head` — entire `<head>` section
- `index-body` — entire `<body>` content
- `index-navbar` — navigation bar
- `index-content` — main content area
- `index-footer` — footer section
- `index-scripts` — scripts section

**about.html** (about page):
- `about-head` — entire `<head>` section
- `about-body` — entire `<body>` content
- `about-navbar` — navigation bar
- `about-content` — main content area
- `about-contact` — contact section
- `about-footer` — footer section
- `about-scripts` — scripts section

**project.html** (project pages):
- `project-head` — entire `<head>` section (excluding styles)
- `project-styles` — inline styles section
- `project-body` — entire `<body>` content
- `project-navbar` — navigation bar
- `project-header` — project header
- `project-gallery` — gallery section
- `project-footer` — footer section
- `project-scripts` — scripts section

### Example: Custom footer

Create `content/templates/custom-footer.html`:

```html
{{define "index-footer"}}
<footer class="custom-footer">
    <p>&copy; {{.Copyright}} | Custom Design</p>
</footer>
{{end}}
```

This will override the footer on the home page only. To override footers on all pages, define all three blocks (`index-footer`, `about-footer`, `project-footer`).

## Images upload

After running `images process` you can upload the processed images to an S3-compatible store (Cloudflare R2, AWS S3, etc.) so they can be served from a CDN.

Example (dry run):

```bash
builder images upload -i dist/images -b <bucket> -r auto --endpoint <url> --base-url https://cdn.example.com --dry-run
```

Example (real upload):

```bash
builder images upload -i dist/images -b <bucket> -r auto --endpoint <url> --base-url https://cdn.example.com
```

Flags explained:
- `-i` : input directory containing processed images (usually `dist/images`).
- `-b` : target bucket name.
- `-r` : region (or `auto`); uploader will attempt to autodetect when supported.
- `--endpoint` : custom S3/R2 endpoint URL (required for R2 or non-AWS providers).
- `--base-url` : public base URL where uploaded images will be served (used when generating site links).
- `--dry-run` : perform a trial run without making changes — recommended first.

Tip: when you upload images to a CDN, pass the same `--host`/`--base-url` to `website build` so generated pages reference the CDN URLs rather than local `dist` paths.