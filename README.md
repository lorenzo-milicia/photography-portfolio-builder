# Photography Portfolio Builder

A local builder application for creating and managing photography portfolio websites. Built with Go and htmx.

## Features

- 📷 **Project Management**: Create, edit, and organize photography projects
- 🖼️ **Photo Upload**: Easy photo upload and management
- 🎨 **Layout Options**: Choose from justified, grid, or manual grid layouts
- 🚀 **Static Site Generation**: Generate complete static HTML websites
- 👁️ **Live Preview**: Preview your generated site before publishing
- ⚡ **htmx-Powered**: Smooth, dynamic UI with minimal JavaScript

## Architecture

### Directory Structure

```
photography-portfolio-builder/
├── cmd/builder/           # Main application entry point
├── internal/
│   ├── builder/          # HTTP server and handlers
│   ├── content/          # Project and photo management
│   ├── generator/        # Static site generator
│   ├── images/           # Image processing (resize, thumbnails)
│   ├── layouts/          # Layout algorithms
│   └── util/             # Utilities (YAML handling)
├── templates/
│   ├── builder/          # Builder UI templates
│   └── site/             # Static site templates
├── static/
│   ├── builder/          # Builder UI assets
│   └── site/             # Generated site assets
├── content/
│   ├── projects/         # Project metadata (YAML)
│   └── photos/           # Original photos
└── output/
    └── public/           # Generated static site
```

### Core Components

1. **Builder App** (`internal/builder`): Local Go web server with htmx-driven UI
2. **Content Manager** (`internal/content`): Handles projects and photos
3. **Image Processor** (`internal/images`): Creates thumbnails and responsive variants
4. **Layout Engine** (`internal/layouts`): Calculates image positioning
5. **Site Generator** (`internal/generator`): Produces static HTML output

## Getting Started

### Prerequisites

- Go 1.21 or higher
- A modern web browser

### Installation

1. Clone the repository:
```bash
cd /home/lorenzo/dev/go/src/go.lorenzomilicia.dev/photography-portfolio-builder
```

2. Build the application:
```bash
go build -o bin/builder ./cmd/builder
```

3. Run the builder:
```bash
./bin/builder
```

4. Open your browser to `http://localhost:8080`

### Command-Line Options

```bash
./bin/builder -port 8080  # Specify custom port (default: 8080)
```

## Usage

### Creating a Project

1. Click "**+ New Project**" in the sidebar
2. Enter project title and description
3. Click "**Create Project**"

### Managing Photos

1. Select a project from the sidebar
2. Use the photo upload section to add images
3. Supported formats: JPG, JPEG, PNG, WebP
4. Photos are automatically organized by project

### Configuring Layout

Each project supports three layout types:

#### Justified Layout
- Photos arranged in rows with equal height
- Parameters:
  - **Row Height**: Target height in pixels (100-1000)
  - **Gap**: Space between images in pixels (0-100)

#### Grid Layout
- Photos arranged in a responsive grid
- Parameters:
  - **Columns**: Number of columns (1-12)
  - **Gap**: Space between images in pixels (0-100)

#### Manual Grid Layout
- Explicit control over photo positioning
- Supports custom spans and positioning

### Generating the Site

1. Click "**🚀 Generate Site**" in the sidebar
2. Wait for generation to complete
3. Click "**View Preview**" to see your site
4. Find the generated site in `output/public/`

### Preview vs. Final Output

- **Preview**: Access via builder at `/preview/`
- **Final Output**: Static files in `output/public/` ready for deployment

## Project Data

### Project Metadata (`content/projects/<slug>/meta.yaml`)

```yaml
title: My Photography Project
slug: my-photography-project
description: A collection of landscape photos
created_at: 2025-12-12T08:00:00Z
updated_at: 2025-12-12T08:30:00Z
```

### Layout Configuration (`content/projects/<slug>/layout.yaml`)

**Justified Layout:**
```yaml
type: justified
params:
  row_height: 300
  gap: 10
```

**Grid Layout:**
```yaml
type: grid
params:
  columns: 3
  gap: 15
```

**Manual Grid Layout:**
```yaml
type: manual
params: {}
```

## Generated Site Structure

```
output/public/
├── index.html                    # Portfolio homepage
├── <project-slug>/
│   └── index.html               # Project gallery page
└── static/
    ├── css/
    │   └── site.css             # Site styles
    └── images/
        └── <project-slug>/      # Project photos
```

## Technical Details

### Image Processing

- **Thumbnails**: 300px width
- **Responsive Variants**: 480px, 800px, 1200px widths
- **Format Support**: JPEG, PNG, WebP
- **Quality**: 90% JPEG compression
- **Algorithm**: Catmull-Rom resampling for high quality

### Layout Algorithms

- **Justified**: Dynamic row-based layout maintaining aspect ratios
- **Grid**: Fixed-column responsive grid
- **Manual**: Custom positioning with span support

### Static Generation

- Go `html/template` for templating
- Responsive images with proper sizing
- Lazy loading for performance
- SEO-friendly HTML structure

## Development

### Project Structure

The codebase follows Go best practices:

- `cmd/`: Application entry points
- `internal/`: Private application code
- `templates/`: HTML templates
- `static/`: Static assets
- `content/`: User content storage
- `output/`: Generated output

### Key Dependencies

- `gopkg.in/yaml.v3`: YAML parsing
- `golang.org/x/image`: Image processing
- `htmx.org`: Dynamic UI (CDN)

## Limitations

This builder intentionally does **not** include:

- Git integration
- Deployment automation
- CI/CD pipelines
- Authentication/authorization
- Multi-user support
- Database storage
- Pixel-perfect drag-and-drop editing

These features are out of scope to keep the builder focused and maintainable.

## Future Enhancements

Potential improvements:

- WebP generation for all images
- Image metadata caching
- Batch photo upload
- Layout templates
- Custom CSS themes
- Export/import projects
- Image optimization settings

## Contributing

This is a personal project by Lorenzo Milicia. Feel free to fork and modify for your own use.

## License

This project is provided as-is for personal use.

## Support

For issues or questions, refer to the code documentation or the project instructions in `.github/instructions/project.instructions.md`.

---

**Built with ❤️ by Lorenzo Milicia**
