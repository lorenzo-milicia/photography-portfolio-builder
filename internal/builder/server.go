package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/generator"
)

// Server represents the builder HTTP server
type Server struct {
	templates  *template.Template
	staticDir  string
	contentMgr *content.Manager
	generator  *generator.Generator
	outputDir  string
}

// NewServer creates a new builder server
func NewServer(templatesDir, staticDir, contentDir, outputDir string) (*Server, error) {
	log.Debug().Str("templatesDir", templatesDir).Msg("Loading templates")

	// Parse templates
	builderTemplates := filepath.Join(templatesDir, "builder", "*.html")
	tmpl, err := template.ParseGlob(builderTemplates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	log.Info().Int("templates", len(tmpl.Templates())).Msg("Templates loaded")

	contentMgr := content.NewManager(contentDir)
	gen := generator.NewGenerator(contentDir, outputDir, templatesDir)

	return &Server{
		templates:  tmpl,
		staticDir:  staticDir,
		contentMgr: contentMgr,
		generator:  gen,
		outputDir:  outputDir,
	}, nil
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Serve static files for builder
	mux.Handle("/static/builder/", http.StripPrefix("/static/builder/",
		http.FileServer(http.Dir(filepath.Join(s.staticDir, "builder")))))

	// Serve content files (photos)
	mux.Handle("/content/", http.StripPrefix("/content/",
		http.FileServer(http.Dir(filepath.Dir(s.contentMgr.ProjectsDir())))))

	// Serve generated site preview
	mux.Handle("/preview/", http.StripPrefix("/preview/",
		http.FileServer(http.Dir(filepath.Join(s.outputDir, "public")))))

	// API routes (must be registered before catch-all)
	mux.HandleFunc("/api/project/create", s.handleProjectCreate)
	mux.HandleFunc("/api/project/update", s.handleProjectUpdate)
	mux.HandleFunc("/api/project/delete", s.handleProjectDelete)
	mux.HandleFunc("/api/project/photos/list", s.handlePhotoList)
	mux.HandleFunc("/api/project/layout/get", s.handleLayoutGet)
	mux.HandleFunc("/api/project/layout/update", s.handleLayoutUpdate)
	mux.HandleFunc("/api/generate", s.handleGenerate)

	// Builder UI routes
	mux.HandleFunc("/projects", s.handleProjectList)
	mux.HandleFunc("/project/new", s.handleProjectNew)
	mux.HandleFunc("/project/", s.handleProjectView)
	mux.HandleFunc("/layout/", s.handleLayoutEditor)

	// Catch-all: serve index for any unmatched route (SPA-like behavior)
	mux.HandleFunc("/", s.handleIndex)
}

// handleIndex shows the main builder interface (catch-all for SPA routing)
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	log.Debug().Str("path", r.URL.Path).Msg("Loading index page")

	projects, err := s.contentMgr.ListProjects()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to list projects")
		projects = []*content.ProjectMetadata{}
	}

	data := map[string]interface{}{
		"Projects": projects,
	}

	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Error().Err(err).Msg("Template execution failed")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleProjectList returns the project list (htmx partial)
func (s *Server) handleProjectList(w http.ResponseWriter, r *http.Request) {
	projects, err := s.contentMgr.ListProjects()
	if err != nil {
		log.Printf("Error listing projects: %v", err)
		http.Error(w, "Failed to list projects", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Projects": projects,
	}

	if err := s.templates.ExecuteTemplate(w, "project-list.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleProjectNew shows the new project form
func (s *Server) handleProjectNew(w http.ResponseWriter, r *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "project-new.html", nil); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleProjectView shows the project editor
func (s *Server) handleProjectView(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Path[len("/project/"):]
	if slug == "" {
		http.Error(w, "Project slug required", http.StatusBadRequest)
		return
	}

	project, err := s.contentMgr.GetProject(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	photos, err := s.contentMgr.ListPhotos(slug)
	if err != nil {
		log.Printf("Error listing photos: %v", err)
		photos = []*content.PhotoInfo{}
	}

	layout, err := s.contentMgr.GetLayout(slug)
	if err != nil {
		log.Printf("Error loading layout: %v", err)
		layout = &content.LayoutConfig{GridWidth: 12, Placements: []content.PhotoPlacement{}}
	}

	data := map[string]interface{}{
		"Project": project,
		"Photos":  photos,
		"Layout":  layout,
	}

	// If this is not an htmx request (direct navigation), return full page with content
	if r.Header.Get("HX-Request") == "" {
		// Create a buffer to render the project view
		var buf bytes.Buffer
		if err := s.templates.ExecuteTemplate(&buf, "project-view.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Wrap it in the full page layout
		pageData := map[string]interface{}{
			"Content": template.HTML(buf.String()),
		}
		if err := s.templates.ExecuteTemplate(w, "index.html", pageData); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// For htmx requests, return just the partial
	if err := s.templates.ExecuteTemplate(w, "project-view.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleLayoutEditor shows the full-page layout editor
func (s *Server) handleLayoutEditor(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Path[len("/layout/"):]
	if slug == "" {
		http.Error(w, "Project slug required", http.StatusBadRequest)
		return
	}

	project, err := s.contentMgr.GetProject(slug)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	photos, err := s.contentMgr.ListPhotos(slug)
	if err != nil {
		log.Printf("Error listing photos: %v", err)
		photos = []*content.PhotoInfo{}
	}

	layout, err := s.contentMgr.GetLayout(slug)
	if err != nil {
		log.Printf("Error loading layout: %v", err)
		layout = &content.LayoutConfig{GridWidth: 12, Placements: []content.PhotoPlacement{}}
	}

	data := map[string]interface{}{
		"Project": project,
		"Photos":  photos,
		"Layout":  layout,
	}

	if err := s.templates.ExecuteTemplate(w, "layout-editor.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleProjectCreate creates a new project
func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")

	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	log.Info().Str("title", title).Msg("Creating new project")

	project, err := s.contentMgr.CreateProject(title, description)
	if err != nil {
		log.Error().Err(err).Str("title", title).Msg("Failed to create project")
		http.Error(w, fmt.Sprintf("Failed to create project: %v", err), http.StatusInternalServerError)
		return
	}

	log.Info().Str("slug", project.Slug).Msg("Project created successfully")

	// Redirect to the new project
	w.Header().Set("HX-Redirect", fmt.Sprintf("/project/%s", project.Slug))
	w.WriteHeader(http.StatusOK)
}

// handleProjectUpdate updates a project
func (s *Server) handleProjectUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	slug := r.FormValue("slug")
	title := r.FormValue("title")
	description := r.FormValue("description")

	if slug == "" || title == "" {
		http.Error(w, "Slug and title are required", http.StatusBadRequest)
		return
	}

	if err := s.contentMgr.UpdateProject(slug, title, description); err != nil {
		log.Printf("Error updating project: %v", err)
		http.Error(w, "Failed to update project", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Project updated successfully")
}

// handleProjectDelete deletes a project
func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "Slug is required", http.StatusBadRequest)
		return
	}

	if err := s.contentMgr.DeleteProject(slug); err != nil {
		log.Printf("Error deleting project: %v", err)
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// handlePhotoList returns the photo list for a project
func (s *Server) handlePhotoList(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "Project slug is required", http.StatusBadRequest)
		return
	}

	photos, err := s.contentMgr.ListPhotos(slug)
	if err != nil {
		log.Printf("Error listing photos: %v", err)
		http.Error(w, "Failed to list photos", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Photos":  photos,
		"Project": map[string]string{"Slug": slug},
	}

	if err := s.templates.ExecuteTemplate(w, "photo-list.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleLayoutGet returns the layout configuration
func (s *Server) handleLayoutGet(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "Project slug is required", http.StatusBadRequest)
		return
	}

	layout, err := s.contentMgr.GetLayout(slug)
	if err != nil {
		log.Printf("Error loading layout: %v", err)
		http.Error(w, "Failed to load layout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(layout)
}

// handleLayoutUpdate updates the layout configuration
func (s *Server) handleLayoutUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var layout content.LayoutConfig
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		log.Error().Err(err).Msg("Failed to decode layout")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "Project slug is required", http.StatusBadRequest)
		return
	}

	// Validate grid width
	if layout.GridWidth < 1 || layout.GridWidth > 24 {
		http.Error(w, "Grid width must be between 1 and 24", http.StatusBadRequest)
		return
	}

	// Validate placements
	for _, placement := range layout.Placements {
		pos := placement.Position
		// Expect 1-based indices (inclusive)
		if pos.TopLeftX < 1 || pos.TopLeftX > layout.GridWidth ||
			pos.BottomRightX < 1 || pos.BottomRightX > layout.GridWidth ||
			pos.TopLeftY < 1 || pos.BottomRightY < 1 ||
			pos.TopLeftX > pos.BottomRightX || pos.TopLeftY > pos.BottomRightY {
			http.Error(w, fmt.Sprintf("Invalid position for %s", placement.Filename), http.StatusBadRequest)
			return
		}
	}

	if err := s.contentMgr.UpdateLayout(slug, &layout); err != nil {
		log.Error().Err(err).Str("slug", slug).Msg("Failed to update layout")
		http.Error(w, "Failed to update layout", http.StatusInternalServerError)
		return
	}

	log.Info().Str("slug", slug).Int("gridWidth", layout.GridWidth).Int("placements", len(layout.Placements)).Msg("Layout updated")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Layout updated successfully"})
}

// handleGenerate triggers static site generation
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Info().Msg("Starting site generation")

	// Generate with /preview base URL for local preview
	if err := s.generator.Generate("/preview"); err != nil {
		log.Error().Err(err).Msg("Site generation failed")
		http.Error(w, fmt.Sprintf("Failed to generate site: %v", err), http.StatusInternalServerError)
		return
	}

	log.Info().Msg("Site generated successfully")

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Site generated successfully! <a href='/preview/' target='_blank'>View Preview</a>")
}
