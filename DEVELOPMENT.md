# Development Guide

This guide explains the architecture and how to extend the Photography Portfolio Builder.

## Architecture Overview

The application follows a clean, modular architecture:

```
User Browser (htmx)
    ↓
HTTP Server (internal/builder)
    ↓
├── Content Manager (internal/content)
│   ├── Projects (YAML)
│   └── Photos (Files)
├── Image Processor (internal/images)
├── Layout Engine (internal/layouts)
└── Site Generator (internal/generator)
```

## Key Concepts

### 1. Content Storage

Projects are stored as YAML files with the following structure:
- `content/projects/{slug}/meta.yaml` - Project metadata
- `content/projects/{slug}/layout.yaml` - Layout configuration
- `content/photos/{slug}/` - Project photos

### 2. Builder Server

The `internal/builder` package handles:
- HTTP routing
- Template rendering
- htmx partial responses
- API endpoints

### 3. Content Management

The `internal/content` package provides:
- Project CRUD operations
- Photo upload/delete
- Layout configuration
- YAML serialization

### 4. Static Generation

The `internal/generator` package:
- Reads project metadata
- Renders HTML templates
- Copies assets
- Generates complete static sites

## Adding New Features

### Adding a New Layout Type

1. **Update Layout Types** in `internal/content/project.go`:
```go
type LayoutConfig struct {
    Type   string                 `yaml:"type"` // justified, grid, manual, YOUR_TYPE
    Params map[string]interface{} `yaml:"params"`
}
```

2. **Implement Algorithm** in `internal/layouts/layout.go`:
```go
func YourLayout(imagePaths []string, params map[string]interface{}) ([]*LayoutItem, error) {
    // Your layout calculation logic
}
```

3. **Update Builder Template** in `templates/builder/project-view.html`:
```html
<option value="your-type">Your Layout Type</option>
```

4. **Update Handler** in `internal/builder/server.go`:
```go
case "your-type":
    // Handle your layout params
```

### Adding Photo Metadata

1. **Extend PhotoInfo** in `internal/content/photo.go`:
```go
type PhotoInfo struct {
    Filename   string    `json:"filename"`
    Path       string    `json:"path"`
    Size       int64     `json:"size"`
    UploadedAt time.Time `json:"uploaded_at"`
    // Add your fields
    Width      int       `json:"width"`
    Height     int       `json:"height"`
    Caption    string    `json:"caption"`
}
```

2. **Update SavePhoto** to populate new fields
3. **Create Metadata File** (optional YAML sidecar)

### Adding Image Filters

1. **Extend Processor** in `internal/images/processor.go`:
```go
func (p *Processor) ApplyFilter(img image.Image, filter string) image.Image {
    switch filter {
    case "grayscale":
        return p.toGrayscale(img)
    case "sepia":
        return p.toSepia(img)
    // Add more filters
    }
}
```

2. **Add UI Controls** in builder templates
3. **Update Generator** to use filters

### Adding Custom Themes

1. **Create Theme Structure**:
```go
type Theme struct {
    Name       string            `yaml:"name"`
    Colors     map[string]string `yaml:"colors"`
    Fonts      []string          `yaml:"fonts"`
    CustomCSS  string            `yaml:"custom_css"`
}
```

2. **Add Theme Selection** in project editor
3. **Update Generator** to apply theme CSS

## Code Organization

### Package Responsibilities

| Package | Responsibility | Dependencies |
|---------|---------------|-------------|
| `cmd/builder` | Entry point | `internal/builder` |
| `internal/builder` | HTTP server | `content`, `generator` |
| `internal/content` | Data management | `util` |
| `internal/generator` | Site generation | `content`, `layouts` |
| `internal/images` | Image processing | - |
| `internal/layouts` | Layout algorithms | - |
| `internal/util` | Utilities | - |

### Data Flow

**Project Creation:**
```
User Form → Builder Handler → Content Manager → YAML File
```

**Photo Upload:**
```
Multipart Form → Builder Handler → Content Manager → File System
```

**Site Generation:**
```
Generate Button → Builder Handler → Generator → Read Projects → 
Render Templates → Copy Photos → Write HTML/CSS
```

## Testing

### Manual Testing

1. **Test Project CRUD:**
```bash
# Create project
curl -X POST http://localhost:8080/api/project/create \
  -d "title=Test&description=Test description"

# List projects (in browser)
open http://localhost:8080/
```

2. **Test Photo Upload:**
```bash
curl -X POST "http://localhost:8080/api/project/photos/upload?slug=test" \
  -F "photo=@test-image.jpg"
```

3. **Test Generation:**
```bash
curl -X POST http://localhost:8080/api/generate
```

### Unit Tests (Example)

Create `internal/content/project_test.go`:
```go
package content

import "testing"

func TestSlugify(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"My Project", "my-project"},
        {"Test 123", "test-123"},
        {"Special!@#$%", "special"},
    }
    
    for _, tt := range tests {
        result := Slugify(tt.input)
        if result != tt.expected {
            t.Errorf("Slugify(%q) = %q; want %q", tt.input, result, tt.expected)
        }
    }
}
```

Run tests:
```bash
make test
```

## Common Development Tasks

### Adding a New API Endpoint

1. **Define Handler** in `internal/builder/server.go`:
```go
func (s *Server) handleYourEndpoint(w http.ResponseWriter, r *http.Request) {
    // Your logic here
}
```

2. **Register Route** in `RegisterRoutes`:
```go
mux.HandleFunc("/api/your-endpoint", s.handleYourEndpoint)
```

3. **Update UI** to call endpoint via htmx or fetch

### Modifying Templates

Templates use Go's `html/template` syntax:

**Conditionals:**
```html
{{if .Project}}
    <h1>{{.Project.Title}}</h1>
{{else}}
    <p>No project</p>
{{end}}
```

**Loops:**
```html
{{range .Photos}}
    <img src="{{.Path}}" alt="{{.Filename}}">
{{end}}
```

**Nested Data:**
```html
{{$.Project.Slug}}  <!-- Access parent scope -->
```

### Adding Configuration

1. **Add to Server struct**:
```go
type Server struct {
    templates   *template.Template
    staticDir   string
    contentMgr  *content.Manager
    generator   *generator.Generator
    outputDir   string
    config      *Config  // Add this
}

type Config struct {
    MaxUploadSize int64
    ImageQuality  int
    // etc.
}
```

2. **Pass in NewServer**
3. **Use in handlers**

## Performance Considerations

### Image Processing
- Process images asynchronously for large batches
- Cache processed images
- Use goroutines for parallel processing

### Site Generation
- Generate incrementally (only changed projects)
- Cache layout calculations
- Use concurrent file operations

### Memory Usage
- Stream large file uploads
- Close file handles promptly
- Limit concurrent operations

## Debugging

### Enable Verbose Logging

Modify `cmd/builder/main.go`:
```go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

### Check File Paths

Add debug logging:
```go
log.Printf("Project dir: %s", m.ProjectDir(slug))
log.Printf("Photo path: %s", photoPath)
```

### Inspect Generated YAML

```bash
cat content/projects/my-project/meta.yaml
cat content/projects/my-project/layout.yaml
```

### Check Generated Output

```bash
ls -la output/public/
cat output/public/index.html
```

## Best Practices

1. **Error Handling**: Always return meaningful errors
2. **Logging**: Log important operations and errors
3. **Validation**: Validate user input before processing
4. **Clean Code**: Follow Go conventions and idioms
5. **Documentation**: Comment complex logic
6. **Testing**: Add tests for critical functionality

## Resources

- [Go Documentation](https://golang.org/doc/)
- [htmx Documentation](https://htmx.org/docs/)
- [Go html/template](https://pkg.go.dev/html/template)
- [YAML v3](https://github.com/go-yaml/yaml)

## Getting Help

Check these locations for information:
- `README.md` - General overview
- `QUICKSTART.md` - Quick start guide
- `PROJECT_STATUS.md` - Implementation status
- `.github/instructions/project.instructions.md` - Original requirements

---

Happy coding! 🚀
