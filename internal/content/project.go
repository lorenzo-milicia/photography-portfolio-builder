package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/util"
)

// ProjectMetadata holds project information
type ProjectMetadata struct {
	Title       string    `yaml:"title"`
	Slug        string    `yaml:"slug"`
	Description string    `yaml:"description"`
	Hidden      bool      `yaml:"hidden"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`
}

// GridPosition represents a photo's position in the grid
type GridPosition struct {
	TopLeftX     int `yaml:"top_left_x" json:"topLeftX"`
	TopLeftY     int `yaml:"top_left_y" json:"topLeftY"`
	BottomRightX int `yaml:"bottom_right_x" json:"bottomRightX"`
	BottomRightY int `yaml:"bottom_right_y" json:"bottomRightY"`
}

// PhotoPlacement holds a photo's placement in the grid
type PhotoPlacement struct {
	Filename string       `yaml:"filename" json:"filename"`
	Position GridPosition `yaml:"position" json:"position"`
}

// LayoutConfig holds layout configuration for grid-based composition
type LayoutConfig struct {
	GridWidth        int              `yaml:"grid_width" json:"gridWidth"`                                   // Width of the grid (default: 12)
	Placements       []PhotoPlacement `yaml:"placements" json:"placements"`                                  // Photo positions in the grid
	MobileGridWidth  int              `yaml:"mobile_grid_width,omitempty" json:"mobileGridWidth,omitempty"`  // Width of mobile grid (optional)
	MobilePlacements []PhotoPlacement `yaml:"mobile_placements,omitempty" json:"mobilePlacements,omitempty"` // Mobile photo positions (optional)
}

// HasMobileLayout returns true if a separate mobile layout is configured
func (lc *LayoutConfig) HasMobileLayout() bool {
	return lc.MobileGridWidth > 0 && len(lc.MobilePlacements) > 0
}

// Manager handles content operations
type Manager struct {
	contentDir string
}

// NewManager creates a new content manager
func NewManager(contentDir string) *Manager {
	return &Manager{contentDir: contentDir}
}

// SiteMetadata holds site-level metadata (global settings)
// About holds about page content
type About struct {
	Title       string   `yaml:"title"`
	Paragraphs  []string `yaml:"paragraphs"`
	Quote       string   `yaml:"quote"`
	QuoteSource string   `yaml:"quote_source,omitempty"`
}

// ProjectOrder defines the order of projects in the UI
type ProjectOrder struct {
	Slug  string `yaml:"slug"`
	Order int    `yaml:"order"`
}

type SiteMetadata struct {
	Copyright     string         `yaml:"copyright"`
	WebsiteName   string         `yaml:"website_name"`
	LogoPrimary   string         `yaml:"logo_primary"`
	LogoSecondary string         `yaml:"logo_secondary"`
	About         *About         `yaml:"about,omitempty"`
	Projects      []ProjectOrder `yaml:"projects,omitempty"`
}

// SiteMetaPath returns the path to the site-level metadata YAML file
func (m *Manager) SiteMetaPath() string {
	return filepath.Join(m.contentDir, "site.yaml")
}

// LoadSiteMeta loads the site metadata if present. If the file does not exist
// an empty SiteMetadata is returned with no error.
func (m *Manager) LoadSiteMeta() (*SiteMetadata, error) {
	var meta SiteMetadata
	if err := util.LoadYAML(m.SiteMetaPath(), &meta); err != nil {
		if os.IsNotExist(err) {
			// Initialize defaults for new sites
			meta.About = &About{}
			return &meta, nil
		}
		return nil, err
	}
	// Ensure About is initialized even if loaded from YAML
	if meta.About == nil {
		meta.About = &About{}
	}
	return &meta, nil
}

// SaveSiteMeta saves the site metadata
func (m *Manager) SaveSiteMeta(meta *SiteMetadata) error {
	return util.SaveYAML(m.SiteMetaPath(), meta)
}

// ProjectsDir returns the projects directory path
func (m *Manager) ProjectsDir() string {
	return filepath.Join(m.contentDir, "projects")
}

// PhotosDir returns the photos directory path
func (m *Manager) PhotosDir() string {
	return filepath.Join(m.contentDir, "photos")
}

// ProjectDir returns the directory for a specific project
func (m *Manager) ProjectDir(slug string) string {
	return filepath.Join(m.ProjectsDir(), slug)
}

// ProjectMetaPath returns the metadata file path for a project
func (m *Manager) ProjectMetaPath(slug string) string {
	return filepath.Join(m.ProjectDir(slug), "meta.yaml")
}

// ProjectLayoutPath returns the layout file path for a project
func (m *Manager) ProjectLayoutPath(slug string) string {
	return filepath.Join(m.ProjectDir(slug), "layout.yaml")
}

// ProjectPhotosDir returns the photos directory for a project
func (m *Manager) ProjectPhotosDir(slug string) string {
	return filepath.Join(m.PhotosDir(), slug)
}

// Slugify converts a title to a URL-friendly slug
func Slugify(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// CreateProject creates a new project
func (m *Manager) CreateProject(title, description string) (*ProjectMetadata, error) {
	slug := Slugify(title)
	if slug == "" {
		return nil, fmt.Errorf("invalid title: cannot create slug")
	}

	projectDir := m.ProjectDir(slug)
	if _, err := os.Stat(projectDir); err == nil {
		return nil, fmt.Errorf("project with slug '%s' already exists", slug)
	}

	// Create directories
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}
	if err := os.MkdirAll(m.ProjectPhotosDir(slug), 0755); err != nil {
		return nil, fmt.Errorf("failed to create photos directory: %w", err)
	}

	// Create metadata
	now := time.Now()
	meta := &ProjectMetadata{
		Title:       title,
		Slug:        slug,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := util.SaveYAML(m.ProjectMetaPath(slug), meta); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// Create default layout with 12-column grid and empty placements
	layout := &LayoutConfig{
		GridWidth:  12,
		Placements: []PhotoPlacement{},
	}
	if err := util.SaveYAML(m.ProjectLayoutPath(slug), layout); err != nil {
		return nil, fmt.Errorf("failed to save layout: %w", err)
	}

	return meta, nil
}

// GetProject retrieves a project's metadata
func (m *Manager) GetProject(slug string) (*ProjectMetadata, error) {
	var meta ProjectMetadata
	if err := util.LoadYAML(m.ProjectMetaPath(slug), &meta); err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}
	return &meta, nil
}

// UpdateProject updates a project's metadata
func (m *Manager) UpdateProject(slug string, title, description string, hidden bool) error {
	meta, err := m.GetProject(slug)
	if err != nil {
		return err
	}

	meta.Title = title
	meta.Description = description
	meta.Hidden = hidden
	meta.UpdatedAt = time.Now()

	return util.SaveYAML(m.ProjectMetaPath(slug), meta)
}

// DeleteProject deletes a project and all its files
func (m *Manager) DeleteProject(slug string) error {
	projectDir := m.ProjectDir(slug)
	photosDir := m.ProjectPhotosDir(slug)

	// Remove photos directory
	if err := os.RemoveAll(photosDir); err != nil {
		return fmt.Errorf("failed to remove photos: %w", err)
	}

	// Remove project directory
	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}

	return nil
}

// ListProjects returns all projects
func (m *Manager) ListProjects() ([]*ProjectMetadata, error) {
	projectsDir := m.ProjectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ProjectMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var projects []*ProjectMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := m.GetProject(entry.Name())
		if err != nil {
			continue // Skip invalid projects
		}
		projects = append(projects, meta)
	}

	// Load site metadata for project order
	siteMeta, err := m.LoadSiteMeta()
	if err != nil {
		// If failed to load, sort by title
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].Title < projects[j].Title
		})
		return projects, nil
	}

	// Create order map
	orderMap := make(map[string]int)
	for _, po := range siteMeta.Projects {
		orderMap[po.Slug] = po.Order
	}

	// Sort projects by order from site.yaml, then by title
	sort.Slice(projects, func(i, j int) bool {
		oi := orderMap[projects[i].Slug]
		if oi == 0 {
			oi = 999
		}
		oj := orderMap[projects[j].Slug]
		if oj == 0 {
			oj = 999
		}
		if oi != oj {
			return oi < oj
		}
		return projects[i].Title < projects[j].Title
	})

	return projects, nil
}

// GetLayout retrieves a project's layout configuration
func (m *Manager) GetLayout(slug string) (*LayoutConfig, error) {
	var layout LayoutConfig
	if err := util.LoadYAML(m.ProjectLayoutPath(slug), &layout); err != nil {
		return nil, fmt.Errorf("failed to load layout: %w", err)
	}
	return &layout, nil
}

// UpdateLayout updates a project's layout configuration
func (m *Manager) UpdateLayout(slug string, layout *LayoutConfig) error {
	return util.SaveYAML(m.ProjectLayoutPath(slug), layout)
}
