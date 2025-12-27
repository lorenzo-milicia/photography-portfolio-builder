package builder

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"go.lorenzomilicia.dev/photography-portfolio-builder/assets"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
)

type ServerNew struct {
	tmpl *template.Template
	gin  *gin.Engine
	mgr  *content.Manager
}

func NewServerNew() *ServerNew {
	funcMap := template.FuncMap{}
	tmpl := template.Must(template.New("templates").Funcs(funcMap).ParseFS(assets.TemplatesFS, "templates/builder_new/*.html", "templates/builder_new/components/*.html", "templates/builder_new/partials/*.html"))

	// create default content manager using content dir
	mgr := content.NewManager("content")

	return NewServerNewWithManager(mgr, tmpl)
}

// NewServerNewWithManager constructs a server injecting the content manager and templates
func NewServerNewWithManager(mgr *content.Manager, tmpl *template.Template) *ServerNew {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Use zerolog console writer for gin logs
	console := zerolog.ConsoleWriter{Out: os.Stderr}
	r.Use(gin.LoggerWithWriter(console))
	r.Use(gin.Recovery())

	srv := &ServerNew{
		tmpl: tmpl,
		gin:  r,
		mgr:  mgr,
	}

	srv.setupRoutes()

	return srv
}

func (s *ServerNew) setupRoutes() {
	s.gin.GET("/", s.handleHome)
	s.gin.GET("/projects/new", s.handleProjectNew)
	s.gin.POST("/projects", s.handleProjectCreate)
	s.gin.POST("/projects/:slug/delete", s.handleProjectDelete)
	s.gin.DELETE("/projects/:slug", s.handleProjectDelete)
	s.gin.GET("/projects/:slug", s.handleProject)
	// Serve embedded assets under /static using the embedded "static" directory as root
	subFS, err := fs.Sub(assets.StaticFS, "static")
	if err != nil {
		zlog.Error().Err(err).Msg("failed to create sub filesystem for static assets")
		// fallback to root FS (may not match expected paths)
		s.gin.StaticFS("/static", http.FS(assets.StaticFS))
		return
	}
	s.gin.StaticFS("/static", http.FS(subFS))
}

func (s *ServerNew) Serve() {
	zlog.Info().Msg("Starting server on :8080")
	if err := s.gin.Run(":8080"); err != nil {
		zlog.Fatal().Err(err).Msg("server failed")
	}
}

func (s *ServerNew) handleHome(c *gin.Context) {
	// Render full page with projects list injected
	if err := s.renderFullPage(c.Writer, "index.html", nil); err != nil {
		zlog.Error().Err(err).Msg("template execute error")
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

// renderFullPage builds the common page context (projects list) and renders the
// provided template. If content is non-nil it will be embedded in the page data
// in a template-friendly way (e.g. embedding *content.ProjectMetadata).
func (s *ServerNew) renderFullPage(w http.ResponseWriter, tmplName string, contentObj interface{}) error {
	var allProjects []*content.ProjectMetadata
	if s.mgr != nil {
		if pList, err := s.mgr.ListProjects(); err == nil {
			allProjects = pList
		} else {
			zlog.Error().Err(err).Msg("failed to list projects for page render")
		}
	}

	switch v := contentObj.(type) {
	case *content.ProjectMetadata:
		pageData := struct {
			Projects []*content.ProjectMetadata
			*content.ProjectMetadata
		}{
			Projects:        allProjects,
			ProjectMetadata: v,
		}
		return s.tmpl.ExecuteTemplate(w, tmplName, pageData)
	default:
		pageData := struct {
			Projects []*content.ProjectMetadata
			Content  interface{}
		}{
			Projects: allProjects,
			Content:  contentObj,
		}
		return s.tmpl.ExecuteTemplate(w, tmplName, pageData)
	}
}

func (s *ServerNew) handleProject(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		zlog.Warn().Msg("empty project slug")
		c.Status(http.StatusBadRequest)
		return
	}

	if s.mgr == nil {
		zlog.Error().Msg("no content manager configured")
		c.Status(http.StatusInternalServerError)
		return
	}

	proj, err := s.mgr.GetProject(slug)
	if err != nil {
		zlog.Error().Err(err).Str("slug", slug).Msg("project not found")
		c.Status(http.StatusNotFound)
		return
	}

	// If this is an HTMX request, return only the partial; otherwise return full page
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		if err := s.tmpl.ExecuteTemplate(c.Writer, "projectConfig", proj); err != nil {
			zlog.Error().Err(err).Msg("failed to render project config")
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// Full page render via shared helper to ensure consistent sidebar data
	if err := s.renderFullPage(c.Writer, "project.html", proj); err != nil {
		zlog.Error().Err(err).Msg("failed to render project page")
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (s *ServerNew) handleProjectNew(c *gin.Context) {
	// If this is an HTMX request, return only the partial; otherwise return full page
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		if err := s.tmpl.ExecuteTemplate(c.Writer, "projectNew", nil); err != nil {
			zlog.Error().Err(err).Msg("failed to render project new partial")
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	if err := s.renderFullPage(c.Writer, "project-new.html", nil); err != nil {
		zlog.Error().Err(err).Msg("failed to render project new page")
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (s *ServerNew) handleProjectCreate(c *gin.Context) {
	if s.mgr == nil {
		zlog.Error().Msg("no content manager configured")
		c.Status(http.StatusInternalServerError)
		return
	}

	// Parse form values
	title := c.PostForm("title")
	description := c.PostForm("description")

	if title == "" {
		c.String(http.StatusBadRequest, "title is required")
		return
	}

	proj, err := s.mgr.CreateProject(title, description)
	if err != nil {
		zlog.Error().Err(err).Str("title", title).Msg("failed to create project")
		// If project exists, return conflict
		c.String(http.StatusConflict, err.Error())
		return
	}

	hx := c.GetHeader("HX-Request")
	// For HTMX requests, return the project partial plus OOB swaps for toast and project list
	if hx != "" {
		var mainBuf bytes.Buffer
		if err := s.tmpl.ExecuteTemplate(&mainBuf, "projectConfig", proj); err != nil {
			zlog.Error().Err(err).Msg("failed to render project config for htmx response")
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// Render combined response using the `projectCreated` template
		var respBuf bytes.Buffer
		respData := struct {
			Project *content.ProjectMetadata
			Message string
		}{
			Project: proj,
			Message: fmt.Sprintf("Project %s created", proj.Title),
		}
		if err := s.tmpl.ExecuteTemplate(&respBuf, "projectCreated", respData); err != nil {
			zlog.Error().Err(err).Msg("failed to render projectCreated for htmx response")
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
		if _, err := c.Writer.Write(respBuf.Bytes()); err != nil {
			zlog.Error().Err(err).Msg("failed to write htmx projectCreated response")
		}
		return
	}

	// For normal form submissions, redirect to the new project page
	c.Redirect(http.StatusSeeOther, "/projects/"+proj.Slug)
}

func (s *ServerNew) handleProjectDelete(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		zlog.Warn().Msg("empty project slug for delete")
		c.Status(http.StatusBadRequest)
		return
	}

	if s.mgr == nil {
		zlog.Error().Msg("no content manager configured")
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := s.mgr.DeleteProject(slug); err != nil {
		zlog.Error().Err(err).Str("slug", slug).Msg("failed to delete project")
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	hx := c.GetHeader("HX-Request")
	if hx != "" {
		// Render a combined HTMX response: main content, OOB delete for project item, and toast
		var buf bytes.Buffer
		respData := struct {
			Slug    string
			Message string
		}{
			Slug:    slug,
			Message: fmt.Sprintf("Project %s deleted", slug),
		}
		if err := s.tmpl.ExecuteTemplate(&buf, "projectDeleted", respData); err != nil {
			zlog.Error().Err(err).Msg("failed to render projectDeleted for htmx response")
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
		if _, err := c.Writer.Write(buf.Bytes()); err != nil {
			zlog.Error().Err(err).Msg("failed to write htmx projectDeleted response")
		}
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
}

func ServeNew() {
	srv := NewServerNew()
	srv.Serve()
}
