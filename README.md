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

### Custom CSS/JS overrides

- Place `custom.css` and/or `custom.js` in your templates directory (default: `content/templates`).
- They are copied to `static/css/custom.css` and `static/js/custom.js` and included after the embedded site assets on every page, so your overrides win the cascade.
- Files named `site.css` or `site.js` are also picked up, but they are still published as `custom.css`/`custom.js` in the generated output.

### Available template blocks

The base templates define the following overrideable blocks:

**Shared across all pages:**
- `navbar` — navigation bar (used by all pages)

**index.html** (home page):
- `index-head` — entire `<head>` section
- `index-body` — entire `<body>` content
- `index-content` — main content area
- `index-footer` — footer section
- `index-scripts` — scripts section

**about.html** (about page):
- `about-head` — entire `<head>` section
- `about-body` — entire `<body>` content
- `about-content` — main content area
- `about-contact` — contact section
- `about-footer` — footer section
- `about-scripts` — scripts section

**project.html** (project pages):
- `project-head` — entire `<head>` section (excluding styles)
- `project-styles` — inline styles section
- `project-body` — entire `<body>` content
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

## Hero images and index page grid layout

You can display project hero images on your homepage using a customizable grid layout.

### Setting a hero image for a project

Add the `hero_photo` field to your project's metadata YAML file (e.g., `content/projects/my-project.yaml`):

```yaml
title: My Project
slug: my-project
description: A beautiful photography project
hero_photo: abc123def456  # 12-character hash ID of the photo
created_at: 2024-01-01T00:00:00Z
```

The `hero_photo` value should be the 12-character hash ID of a photo from that project.

### Configuring the index page grid layout

Create `content/index-layout.yaml` to define how hero images are arranged on the homepage:

```yaml
grid_width: 12
placements:
  - project_slug: my-project
    position:
      top_left_x: 1
      top_left_y: 1
      bottom_right_x: 6
      bottom_right_y: 3
  - project_slug: another-project
    position:
      top_left_x: 7
      top_left_y: 1
      bottom_right_x: 12
      bottom_right_y: 2
```

**Configuration options:**

- `grid_width` — number of columns in the grid (typically 12)
- `placements` — array of hero image placements
  - `project_slug` — slug of the project whose hero image to display
  - `position` — grid position using 1-based coordinates
    - `top_left_x`, `top_left_y` — starting position
    - `bottom_right_x`, `bottom_right_y` — ending position (inclusive)

### Mobile layouts

You can define a separate grid layout for mobile devices:

```yaml
grid_width: 12
placements:
  # ... desktop placements ...

mobile_grid_width: 6
mobile_placements:
  - project_slug: my-project
    position:
      top_left_x: 1
      top_left_y: 1
      bottom_right_x: 6
      bottom_right_y: 2
```

If `mobile_grid_width` and `mobile_placements` are provided, the mobile layout will be used on screens ≤768px wide. Otherwise, the default single-column layout is used.

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