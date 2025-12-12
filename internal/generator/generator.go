package generator

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
)

// Generator handles static site generation
type Generator struct {
	contentMgr   *content.Manager
	outputDir    string
	templatesDir string
	templates    *template.Template
	baseURL      string
}

// NewGenerator creates a new site generator
func NewGenerator(contentDir, outputDir, templatesDir string) *Generator {
	return &Generator{
		contentMgr:   content.NewManager(contentDir),
		outputDir:    outputDir,
		templatesDir: templatesDir,
	}
}

// Generate generates the complete static site with the given base URL prefix
func (g *Generator) Generate(baseURL string) error {
	g.baseURL = baseURL
	log.Debug().Msg("Loading site templates")

	// Create template with helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sanitizeClass": func(s string) string {
			// Replace dots and special chars with hyphens for valid CSS class names
			result := ""
			for _, c := range s {
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
					result += string(c)
				} else {
					result += "-"
				}
			}
			return result
		},
	}

	// Load site templates
	siteTemplates := filepath.Join(g.templatesDir, "site", "*.html")
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(siteTemplates)
	if err != nil {
		return fmt.Errorf("failed to parse site templates: %w", err)
	}
	g.templates = tmpl

	// Create output directory
	publicDir := filepath.Join(g.outputDir, "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get all projects
	projects, err := g.contentMgr.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	log.Info().Int("count", len(projects)).Msg("Generating site for projects")

	// Generate index page
	log.Debug().Msg("Generating index page")
	if err := g.generateIndex(projects); err != nil {
		return fmt.Errorf("failed to generate index: %w", err)
	}

	// Generate project pages
	for _, project := range projects {
		log.Debug().Str("slug", project.Slug).Str("title", project.Title).Msg("Generating project page")
		if err := g.generateProjectPage(project); err != nil {
			return fmt.Errorf("failed to generate project %s: %w", project.Slug, err)
		}
	}

	// Copy static assets
	log.Debug().Msg("Copying static assets")
	if err := g.copyStaticAssets(); err != nil {
		return fmt.Errorf("failed to copy static assets: %w", err)
	}

	log.Info().Msg("Site generation completed")

	return nil
}

// generateIndex generates the main index page
func (g *Generator) generateIndex(projects []*content.ProjectMetadata) error {
	publicDir := filepath.Join(g.outputDir, "public")
	indexPath := filepath.Join(publicDir, "index.html")

	file, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer file.Close()

	data := map[string]interface{}{
		"Projects": projects,
		"BaseURL":  g.baseURL,
	}

	if err := g.templates.ExecuteTemplate(file, "index.html", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// generateProjectPage generates a single project page
func (g *Generator) generateProjectPage(project *content.ProjectMetadata) error {
	publicDir := filepath.Join(g.outputDir, "public")
	projectDir := filepath.Join(publicDir, project.Slug)

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Get only selected photos
	photos, err := g.contentMgr.ListSelectedPhotos(project.Slug)
	if err != nil {
		return fmt.Errorf("failed to list photos: %w", err)
	}

	log.Debug().Str("slug", project.Slug).Int("photoCount", len(photos)).Msg("Generating page with selected photos")

	// Get layout
	layout, err := g.contentMgr.GetLayout(project.Slug)
	if err != nil {
		return fmt.Errorf("failed to load layout: %w", err)
	}

	// Validate layout placements (bounds and overlaps)
	if err := validateLayout(layout); err != nil {
		return fmt.Errorf("invalid layout for project %s: %w", project.Slug, err)
	}

	// Create a map of photos by filename for easy lookup
	photoMap := make(map[string]*content.PhotoInfo)
	for _, photo := range photos {
		photoMap[photo.Filename] = photo
	}

	// Create project page
	pagePath := filepath.Join(projectDir, "index.html")
	file, err := os.Create(pagePath)
	if err != nil {
		return fmt.Errorf("failed to create project page: %w", err)
	}
	defer file.Close()

	data := map[string]interface{}{
		"Project":  project,
		"Photos":   photos,
		"PhotoMap": photoMap,
		"Layout":   layout,
		"BaseURL":  g.baseURL,
	}

	if err := g.templates.ExecuteTemplate(file, "project.html", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Copy photos to output
	if err := g.copyProjectPhotos(project.Slug); err != nil {
		return fmt.Errorf("failed to copy photos: %w", err)
	}

	return nil
}

// validateLayout checks that placements are within grid bounds and that no two
// placements overlap. Returns an error describing the first problem found.
func validateLayout(layout *content.LayoutConfig) error {
	if layout == nil {
		return fmt.Errorf("layout is nil")
	}
	if layout.GridWidth <= 0 {
		return fmt.Errorf("grid_width must be > 0")
	}

	// occupancy map: key "x,y" -> placement index
	occ := make(map[string]int)

	for i, p := range layout.Placements {
		pos := p.Position
		// Basic bounds: configuration uses 1-based indices (inclusive)
		if pos.TopLeftX < 1 || pos.TopLeftY < 1 {
			return fmt.Errorf("placement %d (%s) has top-left coordinates less than 1", i, p.Filename)
		}
		if pos.BottomRightX < pos.TopLeftX || pos.BottomRightY < pos.TopLeftY {
			return fmt.Errorf("placement %d (%s) has bottom-right before top-left", i, p.Filename)
		}
		if pos.BottomRightX > layout.GridWidth {
			return fmt.Errorf("placement %d (%s) extends beyond grid width (%d): bottom_right_x=%d", i, p.Filename, layout.GridWidth, pos.BottomRightX)
		}

		// Check overlaps by marking occupied cells
		for x := pos.TopLeftX; x <= pos.BottomRightX; x++ {
			for y := pos.TopLeftY; y <= pos.BottomRightY; y++ {
				key := fmt.Sprintf("%d,%d", x, y)
				if other, ok := occ[key]; ok {
					return fmt.Errorf("placement %d (%s) overlaps with placement %d (%s) at cell %s", i, p.Filename, other, layout.Placements[other].Filename, key)
				}
				occ[key] = i
			}
		}
	}

	return nil
}

// copyProjectPhotos copies project photos to the output directory
func (g *Generator) copyProjectPhotos(slug string) error {
	sourceDir := g.contentMgr.ProjectPhotosDir(slug)
	destDir := filepath.Join(g.outputDir, "public", "static", "images", slug)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read photos directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(sourceDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if err := copyFile(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy photo %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// copyStaticAssets copies static assets (CSS, JS) to output
func (g *Generator) copyStaticAssets() error {
	cssDir := filepath.Join(g.outputDir, "public", "static", "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		return fmt.Errorf("failed to create css directory: %w", err)
	}

	// Create modern, professional CSS file
	cssPath := filepath.Join(cssDir, "site.css")
	css := `/* Photography Portfolio - Modern Professional Styles */

:root {
    --primary: #2c3e50;
    --primary-light: #34495e;
    --accent: #3498db;
    --accent-dark: #2980b9;
    --text: #2c3e50;
    --text-light: #7f8c8d;
    --bg: #ffffff;
    --bg-alt: #f8f9fa;
    --border: #ecf0f1;
    
    --shadow-sm: 0 2px 4px rgba(0, 0, 0, 0.05);
    --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.07);
    --shadow-lg: 0 10px 15px rgba(0, 0, 0, 0.1);
    --shadow-xl: 0 20px 25px rgba(0, 0, 0, 0.1);
    
    --transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    line-height: 1.7;
    color: var(--text);
    background: var(--bg);
}

.container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 3rem 2rem;
    animation: fadeIn 0.6s ease-out;
}

@keyframes fadeIn {
    from {
        opacity: 0;
        transform: translateY(20px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

/* Header Styles */
header {
    margin-bottom: 4rem;
    text-align: center;
}

header h1 {
    font-size: 3.5rem;
    font-weight: 800;
    color: var(--primary);
    margin-bottom: 0.5rem;
    letter-spacing: -1px;
    background: linear-gradient(135deg, var(--primary), var(--accent));
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}

header p {
    font-size: 1.25rem;
    color: var(--text-light);
    font-weight: 400;
}

header nav {
    margin-bottom: 2rem;
}

header nav a {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--accent);
    text-decoration: none;
    font-weight: 600;
    padding: 0.75rem 1.5rem;
    border: 2px solid var(--accent);
    border-radius: 50px;
    transition: var(--transition);
}

header nav a:hover {
    background: var(--accent);
    color: white;
    transform: translateX(-4px);
    box-shadow: var(--shadow-md);
}

/* Typography */
h1, h2, h3 {
    margin-bottom: 1rem;
    font-weight: 700;
    line-height: 1.2;
    color: var(--primary);
}

h1 {
    font-size: 3rem;
}

h2 {
    font-size: 2rem;
}

h3 {
    font-size: 1.5rem;
}

p {
    margin-bottom: 1rem;
    color: var(--text-light);
}

/* Project List Grid */
.project-list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
    gap: 2.5rem;
    margin-top: 3rem;
}

.project-card {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 2rem;
    text-decoration: none;
    color: inherit;
    transition: var(--transition);
    position: relative;
    overflow: hidden;
    box-shadow: var(--shadow-sm);
}

.project-card::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: linear-gradient(90deg, var(--accent), var(--accent-dark));
    transform: scaleX(0);
    transition: transform 0.3s ease;
}

.project-card:hover::before {
    transform: scaleX(1);
}

.project-card:hover {
    transform: translateY(-8px);
    box-shadow: var(--shadow-xl);
    border-color: var(--accent);
}

.project-card h2 {
    font-size: 1.75rem;
    margin-bottom: 0.75rem;
    color: var(--primary);
    transition: var(--transition);
}

.project-card:hover h2 {
    color: var(--accent);
}

.project-card p {
    color: var(--text-light);
    line-height: 1.6;
}

/* Gallery Styles */
.gallery {
    margin-top: 3rem;
    animation: fadeIn 0.8s ease-out;
}

/* Justified Gallery Layout */
.gallery.justified {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
}

.gallery.justified .gallery-item {
    flex-grow: 1;
    height: 300px;
    position: relative;
}

.gallery.justified .gallery-item img {
    height: 100%;
    width: 100%;
    object-fit: cover;
}

/* Grid Gallery Layout */
.gallery.grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
    gap: 1.5rem;
}

/* Manual Gallery Layout */
.gallery.manual {
    display: grid;
    grid-template-columns: repeat(12, 1fr);
    gap: 1.5rem;
}

/* Gallery Item */
.gallery-item {
    position: relative;
    overflow: hidden;
    border-radius: 12px;
    box-shadow: var(--shadow-sm);
    transition: var(--transition);
    cursor: pointer;
    background: var(--bg-alt);
}

.gallery-item:hover {
    transform: scale(1.02);
    box-shadow: var(--shadow-lg);
    z-index: 10;
}

.gallery-item img {
    width: 100%;
    height: auto;
    display: block;
    transition: var(--transition);
}

.gallery-item:hover img {
    transform: scale(1.05);
}

/* Loading Animation */
.gallery-item img {
    animation: imageLoad 0.6s ease-out;
}

@keyframes imageLoad {
    from {
        opacity: 0;
        transform: scale(0.95);
    }
    to {
        opacity: 1;
        transform: scale(1);
    }
}

/* Responsive Images */
.gallery-item img[loading="lazy"] {
    background: linear-gradient(90deg, var(--bg-alt) 0%, var(--border) 50%, var(--bg-alt) 100%);
    background-size: 200% 100%;
    animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
    0% {
        background-position: -200% 0;
    }
    100% {
        background-position: 200% 0;
    }
}

/* Lightbox Effect (Optional) */
.gallery-item::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(180deg, transparent 0%, rgba(0,0,0,0.3) 100%);
    opacity: 0;
    transition: var(--transition);
}

.gallery-item:hover::after {
    opacity: 1;
}

/* Footer */
footer {
    margin-top: 6rem;
    padding-top: 3rem;
    border-top: 1px solid var(--border);
    text-align: center;
    color: var(--text-light);
    font-size: 0.9rem;
}

/* Responsive Design */
@media (max-width: 1024px) {
    .container {
        padding: 2rem 1.5rem;
    }
    
    header h1 {
        font-size: 2.5rem;
    }
    
    .project-list {
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 2rem;
    }
    
    .gallery.grid {
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 1rem;
    }
}

@media (max-width: 768px) {
    .container {
        padding: 1.5rem 1rem;
    }
    
    header h1 {
        font-size: 2rem;
    }
    
    header p {
        font-size: 1rem;
    }
    
    .project-list {
        grid-template-columns: 1fr;
        gap: 1.5rem;
    }
    
    .gallery.grid {
        grid-template-columns: 1fr;
        gap: 1rem;
    }
    
    .gallery.justified .gallery-item {
        height: 250px;
    }
}

/* Print Styles */
@media print {
    .gallery-item {
        break-inside: avoid;
    }
    
    header nav {
        display: none;
    }
}

/* Accessibility */
@media (prefers-reduced-motion: reduce) {
    *,
    *::before,
    *::after {
        animation-duration: 0.01ms !important;
        animation-iteration-count: 1 !important;
        transition-duration: 0.01ms !important;
    }
}

/* Dark mode support (optional) */
@media (prefers-color-scheme: dark) {
    :root {
        --primary: #ecf0f1;
        --primary-light: #bdc3c7;
        --text: #ecf0f1;
        --text-light: #95a5a6;
        --bg: #1a1a1a;
        --bg-alt: #2c2c2c;
        --border: #34495e;
    }
    
    .gallery-item img[loading="lazy"] {
        background: linear-gradient(90deg, var(--bg-alt) 0%, var(--border) 50%, var(--bg-alt) 100%);
    }
}
`
	if err := os.WriteFile(cssPath, []byte(css), 0644); err != nil {
		return fmt.Errorf("failed to write CSS: %w", err)
	}

	log.Debug().Msg("Created modern portfolio CSS")

	return nil
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceData, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, sourceData, 0644)
}
