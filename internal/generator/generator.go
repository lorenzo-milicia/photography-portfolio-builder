package generator

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
)

// Generator handles static site generation
type Generator struct {
	contentMgr     *content.Manager
	outputDir      string
	templatesFS    fs.FS
	staticFS       fs.FS
	templates      *template.Template
	baseURL        string
	imageURLPrefix string
}

// NewGenerator creates a new site generator
func NewGenerator(contentDir, outputDir string, templatesFS, staticFS fs.FS) *Generator {
	return &Generator{
		contentMgr:  content.NewManager(contentDir),
		outputDir:   outputDir,
		templatesFS: templatesFS,
		staticFS:    staticFS,
	}
}

// Generate generates the complete static site with the given base URL prefix and optional image URL prefix
func (g *Generator) Generate(baseURL string, imageURLPrefix string) error {
	g.baseURL = baseURL
	g.imageURLPrefix = imageURLPrefix

	// Capture build timestamp for cache busting
	buildTimestamp := time.Now().Unix()

	log.Debug().Msg("Loading site templates")

	// Create template with helper functions
	funcMap := template.FuncMap{
		"add":    func(a, b int) int { return a + b },
		"sub":    func(a, b int) int { return a - b },
		"mul":    func(a, b float64) float64 { return a * b },
		"le":     func(a, b int) bool { return a <= b },
		"printf": fmt.Sprintf,
		"nl2br":  func(s string) template.HTML { return template.HTML(strings.ReplaceAll(s, "\n", "<br>")) },
		"stripAt": func(s string) string {
			if strings.HasPrefix(s, "@") {
				return s[1:]
			}
			return s
		},
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
		"calculateSizes": func(placement content.PhotoPlacement) string {
			// Calculate the viewport width percentage for this image
			// Grid is 12 columns, max container is 1400px
			colSpan := placement.Position.BottomRightX - placement.Position.TopLeftX + 1
			percentage := (colSpan * 100) / 12

			// Generate sizes attribute for responsive images
			// Format: (max-width: breakpoint) width, default-width
			return fmt.Sprintf("(max-width: 768px) 100vw, (max-width: 1024px) %dvw, %dpx",
				percentage, (colSpan*1400)/12)
		},
	}

	// Load site templates
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(g.templatesFS, "templates/site/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse site templates: %w", err)
	}
	g.templates = tmpl

	// Create output directory (use the root of the provided outputDir)
	publicDir := g.outputDir
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get all projects
	allProjects, err := g.contentMgr.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	// Filter out hidden projects
	var projects []*content.ProjectMetadata
	for _, p := range allProjects {
		if !p.Hidden {
			projects = append(projects, p)
		}
	}

	log.Info().Int("total", len(allProjects)).Int("active", len(projects)).Msg("Generating site for projects")

	// Generate index page
	log.Debug().Msg("Generating index page")
	if err := g.generateIndex(projects, buildTimestamp); err != nil {
		return fmt.Errorf("failed to generate index: %w", err)
	}

	// Generate about page
	log.Debug().Msg("Generating about page")
	if err := g.generateAbout(buildTimestamp); err != nil {
		return fmt.Errorf("failed to generate about: %w", err)
	}

	// Generate project pages
	for _, project := range projects {
		log.Debug().Str("slug", project.Slug).Str("title", project.Title).Msg("Generating project page")
		if err := g.generateProjectPage(project, buildTimestamp); err != nil {
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
func (g *Generator) generateIndex(projects []*content.ProjectMetadata, buildTimestamp int64) error {
	publicDir := g.outputDir
	indexPath := filepath.Join(publicDir, "index.html")

	file, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer file.Close()

	// Load optional site metadata (e.g. copyright) to pass to templates
	siteMeta, err := g.contentMgr.LoadSiteMeta()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load site metadata")
		siteMeta = &content.SiteMetadata{
			Copyright:   "2025 Photography Portfolio. All rights reserved.",
			WebsiteName: "Photography Portfolio",
		}
	}
	if siteMeta.Copyright == "" {
		siteMeta.Copyright = "2025 Photography Portfolio. All rights reserved."
	}
	if siteMeta.WebsiteName == "" {
		siteMeta.WebsiteName = "Photography Portfolio"
	}
	if siteMeta.LogoPrimary == "" {
		siteMeta.LogoPrimary = "portfolio"
	}
	if siteMeta.LogoSecondary == "" {
		siteMeta.LogoSecondary = "photography"
	}

	data := map[string]interface{}{
		"Projects":       projects,
		"BaseURL":        g.baseURL,
		"Copyright":      siteMeta.Copyright,
		"WebsiteName":    siteMeta.WebsiteName,
		"LogoPrimary":    siteMeta.LogoPrimary,
		"LogoSecondary":  siteMeta.LogoSecondary,
		"BuildTimestamp": buildTimestamp,
	}

	if err := g.templates.ExecuteTemplate(file, "index.html", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// generateAbout generates the about page
func (g *Generator) generateAbout(buildTimestamp int64) error {
	publicDir := g.outputDir
	aboutDir := filepath.Join(publicDir, "about")

	if err := os.MkdirAll(aboutDir, 0755); err != nil {
		return fmt.Errorf("failed to create about directory: %w", err)
	}

	aboutPath := filepath.Join(aboutDir, "index.html")

	file, err := os.Create(aboutPath)
	if err != nil {
		return fmt.Errorf("failed to create about.html: %w", err)
	}
	defer file.Close()

	// Load site metadata
	siteMeta, err := g.contentMgr.LoadSiteMeta()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load site metadata")
		siteMeta = &content.SiteMetadata{
			Copyright:   "2025 Photography Portfolio. All rights reserved.",
			WebsiteName: "Photography Portfolio",
		}
	}
	if siteMeta.Copyright == "" {
		siteMeta.Copyright = "2025 Photography Portfolio. All rights reserved."
	}
	if siteMeta.WebsiteName == "" {
		siteMeta.WebsiteName = "Photography Portfolio"
	}
	if siteMeta.LogoPrimary == "" {
		siteMeta.LogoPrimary = "portfolio"
	}
	if siteMeta.LogoSecondary == "" {
		siteMeta.LogoSecondary = "photography"
	}

	// Get all projects for navigation
	allProjects, err := g.contentMgr.ListProjects()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list projects for navigation")
		allProjects = []*content.ProjectMetadata{}
	}

	var projects []*content.ProjectMetadata
	for _, p := range allProjects {
		if !p.Hidden {
			projects = append(projects, p)
		}
	}

	data := map[string]interface{}{
		"BaseURL":        g.baseURL,
		"WebsiteName":    siteMeta.WebsiteName,
		"LogoPrimary":    siteMeta.LogoPrimary,
		"LogoSecondary":  siteMeta.LogoSecondary,
		"AllProjects":    projects,
		"About":          siteMeta.About,
		"Contact":        siteMeta.Contact,
		"Copyright":      siteMeta.Copyright,
		"BuildTimestamp": buildTimestamp,
	}

	if err := g.templates.ExecuteTemplate(file, "about.html", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// generateProjectPage generates a single project page
func (g *Generator) generateProjectPage(project *content.ProjectMetadata, buildTimestamp int64) error {
	publicDir := g.outputDir
	projectDir := filepath.Join(publicDir, project.Slug)

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Get all photos
	photos, err := g.contentMgr.ListPhotos(project.Slug)
	if err != nil {
		return fmt.Errorf("failed to list photos: %w", err)
	}

	log.Debug().Str("slug", project.Slug).Int("photoCount", len(photos)).Msg("Generating page with photos")

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

	// Get all projects for navigation tabs
	allProjects, err := g.contentMgr.ListProjects()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list projects for navigation")
		allProjects = []*content.ProjectMetadata{}
	}

	var projects []*content.ProjectMetadata
	for _, p := range allProjects {
		if !p.Hidden {
			projects = append(projects, p)
		}
	}

	// Layout now contains hash IDs (12 chars) as filenames
	// No image processing needed - images are pre-processed by `images process` command
	// We just use the hash IDs from layout to construct image URLs

	// Create project page
	pagePath := filepath.Join(projectDir, "index.html")
	file, err := os.Create(pagePath)
	if err != nil {
		return fmt.Errorf("failed to create project page: %w", err)
	}
	defer file.Close()

	// Load optional site metadata
	siteMeta, err := g.contentMgr.LoadSiteMeta()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load site metadata")
		siteMeta = &content.SiteMetadata{
			Copyright:   "2025 Photography Portfolio. All rights reserved.",
			WebsiteName: "Photography Portfolio",
		}
	}
	if siteMeta.Copyright == "" {
		siteMeta.Copyright = "2025 Photography Portfolio. All rights reserved."
	}
	if siteMeta.WebsiteName == "" {
		siteMeta.WebsiteName = "Photography Portfolio"
	}
	if siteMeta.LogoPrimary == "" {
		siteMeta.LogoPrimary = "portfolio"
	}
	if siteMeta.LogoSecondary == "" {
		siteMeta.LogoSecondary = "photography"
	}

	data := map[string]interface{}{
		"Project":        project,
		"Photos":         photos,
		"PhotoMap":       photoMap,
		"Layout":         layout,
		"BaseURL":        g.baseURL,
		"ImageURLPrefix": g.imageURLPrefix,
		"AllProjects":    projects,
		"WebsiteName":    siteMeta.WebsiteName,
		"LogoPrimary":    siteMeta.LogoPrimary,
		"LogoSecondary":  siteMeta.LogoSecondary,
		"Copyright":      siteMeta.Copyright,
		"BuildTimestamp": buildTimestamp,
	}

	if err := g.templates.ExecuteTemplate(file, "project.html", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
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

	// Validate Mobile Placements (Independent Grid)
	mobileOcc := make(map[string]int)
	mobileGridWidth := layout.MobileGridWidth
	if mobileGridWidth <= 0 {
		mobileGridWidth = 6 // Default fallback matching JS logic
	}

	for i, p := range layout.MobilePlacements {
		pos := p.Position
		if pos.TopLeftX < 1 || pos.TopLeftY < 1 {
			return fmt.Errorf("mobile placement %d (%s) has top-left coordinates less than 1", i, p.Filename)
		}
		if pos.BottomRightX < pos.TopLeftX || pos.BottomRightY < pos.TopLeftY {
			return fmt.Errorf("mobile placement %d (%s) has bottom-right before top-left", i, p.Filename)
		}
		if pos.BottomRightX > mobileGridWidth {
			return fmt.Errorf("mobile placement %d (%s) extends beyond mobile grid width (%d): bottom_right_x=%d", i, p.Filename, mobileGridWidth, pos.BottomRightX)
		}

		// Check overlaps
		for x := pos.TopLeftX; x <= pos.BottomRightX; x++ {
			for y := pos.TopLeftY; y <= pos.BottomRightY; y++ {
				key := fmt.Sprintf("%d,%d", x, y)
				if other, ok := mobileOcc[key]; ok {
					return fmt.Errorf("mobile placement %d (%s) overlaps with mobile placement %d (%s) at cell %s", i, p.Filename, other, layout.MobilePlacements[other].Filename, key)
				}
				mobileOcc[key] = i
			}
		}
	}

	return nil
}

// optimizeProjectPhotos optimizes and copies project photos to the output directory
// getImagePath constructs the full image URL with optional prefix
// hashID is the 12-character hash prefix, filename is the variant file (e.g., "hash-480w.webp")
func (g *Generator) getImagePath(slug string, hashID string, filename string) string {
	// New structure: /static/images/{project}/{hashID}/{filename}
	relPath := fmt.Sprintf("/static/images/%s/%s/%s", slug, hashID, filename)

	if g.imageURLPrefix != "" {
		// External hosting: prefix + relPath
		return g.imageURLPrefix + relPath
	}

	// Local hosting: baseURL + relPath
	return g.baseURL + relPath
}

// getThumbnailPath constructs the thumbnail URL
func (g *Generator) getThumbnailPath(slug string, filename string) string {
	// Thumbnails are in /static/images/{project}/.thumbs/{filename}
	relPath := fmt.Sprintf("/static/images/%s/.thumbs/%s", slug, filename)

	if g.imageURLPrefix != "" {
		return g.imageURLPrefix + relPath
	}

	return g.baseURL + relPath
}

// copyStaticAssets copies static assets (CSS, JS) to output
func (g *Generator) copyStaticAssets() error {
	// Create CSS directory under the output root
	cssDir := filepath.Join(g.outputDir, "static", "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		return fmt.Errorf("failed to create css directory: %w", err)
	}

	// Copy site CSS from embedded static/site/site.css into the generated output
	cssPath := filepath.Join(cssDir, "site.css")
	cssData, err := fs.ReadFile(g.staticFS, "static/site/site.css")
	if err != nil {
		// Fallback: write an empty placeholder CSS to avoid missing file
		if err := os.WriteFile(cssPath, []byte("/* site.css missing - please add static/site/site.css */"), 0644); err != nil {
			return fmt.Errorf("failed to write placeholder CSS: %w", err)
		}
	} else {
		if err := os.WriteFile(cssPath, cssData, 0644); err != nil {
			return fmt.Errorf("failed to write site CSS: %w", err)
		}
		log.Debug().Str("dest", cssPath).Msg("Copied site CSS")
	}

	// Create JS directory under the output root
	jsDir := filepath.Join(g.outputDir, "static", "js")
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		return fmt.Errorf("failed to create js directory: %w", err)
	}

	// Copy site JS from embedded static/site/site.js into the generated output
	jsPath := filepath.Join(jsDir, "site.js")
	jsData, err := fs.ReadFile(g.staticFS, "static/site/site.js")
	if err != nil {
		// Fallback: write an empty placeholder JS to avoid missing file
		if err := os.WriteFile(jsPath, []byte("/* site.js missing - please add static/site/site.js */"), 0644); err != nil {
			return fmt.Errorf("failed to write placeholder JS: %w", err)
		}
	} else {
		if err := os.WriteFile(jsPath, jsData, 0644); err != nil {
			return fmt.Errorf("failed to write site JS: %w", err)
		}
		log.Debug().Str("dest", jsPath).Msg("Copied site JS")
	}

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
