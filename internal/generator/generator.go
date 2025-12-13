package generator

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/images"
)

// Generator handles static site generation
type Generator struct {
	contentMgr     *content.Manager
	imageProcessor *images.Processor
	outputDir      string
	templatesDir   string
	templates      *template.Template
	baseURL        string
}

// NewGenerator creates a new site generator
func NewGenerator(contentDir, outputDir, templatesDir string) *Generator {
	cacheDir := filepath.Join(outputDir, ".cache")
	return &Generator{
		contentMgr:     content.NewManager(contentDir),
		imageProcessor: images.NewProcessor(cacheDir),
		outputDir:      outputDir,
		templatesDir:   templatesDir,
	}
}

// Generate generates the complete static site with the given base URL prefix
func (g *Generator) Generate(baseURL string) error {
	g.baseURL = baseURL
	log.Debug().Msg("Loading site templates")

	// Create template with helper functions
	funcMap := template.FuncMap{
		"add":    func(a, b int) int { return a + b },
		"sub":    func(a, b int) int { return a - b },
		"mul":    func(a, b float64) float64 { return a * b },
		"le":     func(a, b int) bool { return a <= b },
		"printf": fmt.Sprintf,
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

	// Generate about page
	log.Debug().Msg("Generating about page")
	if err := g.generateAbout(); err != nil {
		return fmt.Errorf("failed to generate about: %w", err)
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
		"Projects":      projects,
		"BaseURL":       g.baseURL,
		"Copyright":     siteMeta.Copyright,
		"WebsiteName":   siteMeta.WebsiteName,
		"LogoPrimary":   siteMeta.LogoPrimary,
		"LogoSecondary": siteMeta.LogoSecondary,
	}

	if err := g.templates.ExecuteTemplate(file, "index.html", data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// generateAbout generates the about page
func (g *Generator) generateAbout() error {
	publicDir := filepath.Join(g.outputDir, "public")
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
	projects, err := g.contentMgr.ListProjects()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list projects for navigation")
		projects = []*content.ProjectMetadata{}
	}

	data := map[string]interface{}{
		"BaseURL":       g.baseURL,
		"WebsiteName":   siteMeta.WebsiteName,
		"LogoPrimary":   siteMeta.LogoPrimary,
		"LogoSecondary": siteMeta.LogoSecondary,
		"AllProjects":   projects,
		"About":         siteMeta.About,
		"Copyright":     siteMeta.Copyright,
	}

	if err := g.templates.ExecuteTemplate(file, "about.html", data); err != nil {
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

	// Optimize and copy photos to output (must be done before generating HTML)
	variantsMap, err := g.optimizeProjectPhotos(project.Slug, layout)
	if err != nil {
		return fmt.Errorf("failed to optimize photos: %w", err)
	}

	// Create a map from original filename to variants for template use
	originalToVariants := make(map[string]*images.ImageVariants)
	for originalName, variants := range variantsMap {
		originalToVariants[originalName] = variants
	}

	// Update layout with optimized filenames and create photo info map
	optimizedLayout := &content.LayoutConfig{
		GridWidth:        layout.GridWidth,
		Placements:       make([]content.PhotoPlacement, len(layout.Placements)),
		MobileGridWidth:  layout.MobileGridWidth,
		MobilePlacements: make([]content.PhotoPlacement, len(layout.MobilePlacements)),
	}
	optimizedPhotoMap := make(map[string]*content.PhotoInfo)

	for i, placement := range layout.Placements {
		variants, ok := variantsMap[placement.Filename]
		if !ok || variants == nil || len(variants.Variants) == 0 {
			return fmt.Errorf("no variants generated for %s", placement.Filename)
		}

		// Use the largest variant as the default filename
		largestVariant := variants.Variants[len(variants.Variants)-1]

		optimizedLayout.Placements[i] = content.PhotoPlacement{
			Filename: largestVariant.Filename,
			Position: placement.Position,
		}

		// Create photo info with responsive variants
		if photo, ok := photoMap[placement.Filename]; ok {
			newPhoto := *photo // Copy
			newPhoto.Filename = largestVariant.Filename
			optimizedPhotoMap[largestVariant.Filename] = &newPhoto
		}
	}

	// Process mobile placements if they exist
	for i, placement := range layout.MobilePlacements {
		variants, ok := variantsMap[placement.Filename]
		if !ok || variants == nil || len(variants.Variants) == 0 {
			return fmt.Errorf("no variants generated for mobile placement %s", placement.Filename)
		}

		largestVariant := variants.Variants[len(variants.Variants)-1]
		optimizedLayout.MobilePlacements[i] = content.PhotoPlacement{
			Filename: largestVariant.Filename,
			Position: placement.Position,
		}
	}

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
		"PhotoMap":       optimizedPhotoMap,
		"Layout":         optimizedLayout,
		"OriginalLayout": layout, // Keep original for variant lookup
		"VariantsMap":    originalToVariants,
		"BaseURL":        g.baseURL,
		"AllProjects":    allProjects,
		"WebsiteName":    siteMeta.WebsiteName,
		"LogoPrimary":    siteMeta.LogoPrimary,
		"LogoSecondary":  siteMeta.LogoSecondary,
		"Copyright":      siteMeta.Copyright,
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

	return nil
}

// optimizeProjectPhotos optimizes and copies project photos to the output directory
// Returns a map from original filenames to image variants
func (g *Generator) optimizeProjectPhotos(slug string, layout *content.LayoutConfig) (map[string]*images.ImageVariants, error) {
	sourceDir := g.contentMgr.ProjectPhotosDir(slug)
	destDir := filepath.Join(g.outputDir, "public", "static", "images", slug)

	// Use the image processor to optimize images based on layout
	variantsMap, err := g.imageProcessor.ProcessProjectImages(sourceDir, destDir, layout)
	if err != nil {
		return nil, fmt.Errorf("failed to process images: %w", err)
	}

	totalVariants := 0
	for _, variants := range variantsMap {
		totalVariants += len(variants.Variants)
	}

	log.Info().
		Str("project", slug).
		Int("images", len(variantsMap)).
		Int("variants", totalVariants).
		Msg("Optimized project images")

	return variantsMap, nil
}

// copyStaticAssets copies static assets (CSS, JS) to output
func (g *Generator) copyStaticAssets() error {
	cssDir := filepath.Join(g.outputDir, "public", "static", "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		return fmt.Errorf("failed to create css directory: %w", err)
	}

	// Copy site CSS from repo static/site/site.css into the generated output
	cssPath := filepath.Join(cssDir, "site.css")
	// static directory is located next to templatesDir (workDir/static/site/site.css)
	sourceCSS := filepath.Join(filepath.Dir(g.templatesDir), "static", "site", "site.css")
	if _, err := os.Stat(sourceCSS); err == nil {
		if err := copyFile(sourceCSS, cssPath); err != nil {
			return fmt.Errorf("failed to copy site CSS: %w", err)
		}
		log.Debug().Str("source", sourceCSS).Str("dest", cssPath).Msg("Copied site CSS")
	} else {
		// Fallback: write an empty placeholder CSS to avoid missing file
		if err := os.WriteFile(cssPath, []byte("/* site.css missing - please add static/site/site.css */"), 0644); err != nil {
			return fmt.Errorf("failed to write placeholder CSS: %w", err)
		}
		log.Warn().Str("expected", sourceCSS).Msg("site.css not found; wrote placeholder")
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
