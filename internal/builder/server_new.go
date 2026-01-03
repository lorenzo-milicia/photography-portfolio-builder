package builder

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"go.lorenzomilicia.dev/photography-portfolio-builder/assets"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/processing"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/uploader"
)

type ServerNew struct {
	tmpl           *template.Template
	gin            *gin.Engine
	mgr            *content.Manager
	outputDir      string
	imageURLPrefix string
	uploader       uploader.Uploader
}

func NewServerNew() *ServerNew {
	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict expects even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
	tmpl := template.Must(template.New("templates").Funcs(funcMap).ParseFS(assets.TemplatesFS, "templates/builder_new/*.html", "templates/builder_new/components/*.html", "templates/builder_new/partials/*.html"))

	// create default content manager using content dir
	mgr := content.NewManager("content")

	// Get imageURLPrefix from environment (IMAGE_HOST or IMAGE_URL_PREFIX)
	imageURLPrefix := os.Getenv("IMAGE_HOST")
	if imageURLPrefix == "" {
		imageURLPrefix = os.Getenv("IMAGE_URL_PREFIX")
	}

	// Try to initialize uploader from site config
	var uploaderInstance uploader.Uploader
	siteMeta, err := mgr.LoadSiteMeta()
	if err == nil && siteMeta.Storage != nil && siteMeta.Storage.Driver == "s3" {
		ctx := context.Background()
		s3Config := uploader.S3Config{
			Endpoint:        siteMeta.Storage.Endpoint,
			Region:          siteMeta.Storage.Region,
			Bucket:          siteMeta.Storage.Bucket,
			BaseURL:         siteMeta.Storage.BaseURL,
			AccessKeyID:     "", // Read from environment
			SecretAccessKey: "", // Read from environment
		}

		uploaderInstance, err = uploader.NewS3Uploader(ctx, s3Config)
		if err != nil {
			zlog.Warn().Err(err).Msg("Failed to initialize S3 uploader, will use filesystem only")
			uploaderInstance = nil
		} else {
			zlog.Info().Str("bucket", s3Config.Bucket).Str("region", s3Config.Region).Msg("S3 uploader initialized successfully")
		}
	}

	return NewServerNewWithManager(mgr, tmpl, "dist", imageURLPrefix, uploaderInstance)
}

// NewServerNewWithManager constructs a server injecting the content manager and templates
func NewServerNewWithManager(mgr *content.Manager, tmpl *template.Template, outputDir string, imageURLPrefix string, uploaderInstance uploader.Uploader) *ServerNew {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Use zerolog console writer for gin logs
	console := zerolog.ConsoleWriter{Out: os.Stderr}
	r.Use(gin.LoggerWithWriter(console))
	r.Use(gin.Recovery())

	srv := &ServerNew{
		tmpl:           tmpl,
		gin:            r,
		mgr:            mgr,
		outputDir:      outputDir,
		imageURLPrefix: imageURLPrefix,
		uploader:       uploaderInstance,
	}

	srv.setupRoutes()

	return srv
}

func (s *ServerNew) setupRoutes() {
	s.gin.GET("/", s.handleHome)
	s.gin.GET("/projects/new", s.handleProjectNew)
	s.gin.POST("/projects", s.handleProjectCreate)
	s.gin.POST("/projects/:slug", s.handleProjectUpdate)
	s.gin.POST("/projects/:slug/delete", s.handleProjectDelete)
	s.gin.DELETE("/projects/:slug", s.handleProjectDelete)
	s.gin.GET("/projects/:slug", s.handleProject)
	s.gin.POST("/projects/:slug/photos/upload", s.handlePhotoUpload)
	s.gin.DELETE("/projects/:slug/photos/:hashID", s.handlePhotoDelete)
	s.gin.GET("/config", s.handleConfig)
	s.gin.POST("/config", s.handleConfigSave)

	// Serve processed images (including thumbnails) from dist/images directory
	// This serves: dist/images/{project}/.thumbs/thumb-{hashID}.webp
	// and dist/images/{project}/{hashID}/{hashID}-{width}w.webp
	imagesDir := filepath.Join(s.outputDir, "images")
	s.gin.Static("/images", imagesDir)

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
		// For any other type, merge Projects into the content struct
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

	// Load processed photos from dist/images/{slug}/.thumbs/
	processedImagesDir := filepath.Join(s.outputDir, "images")
	photos, err := s.mgr.ListProcessedPhotos(slug, processedImagesDir)
	if err != nil {
		zlog.Error().Err(err).Str("slug", slug).Msg("failed to load processed photos")
		// Continue without photos rather than failing
		photos = []*content.PhotoInfo{}
	}

	// Update photo paths based on imageURLPrefix
	for _, photo := range photos {
		if s.imageURLPrefix != "" {
			// External storage: use imageURLPrefix
			photo.ThumbPath = fmt.Sprintf("%s/static/images/%s/.thumbs/%s", s.imageURLPrefix, slug, photo.Filename)
		}
		// else: already set correctly by ListProcessedPhotos to /images/{slug}/.thumbs/{filename}
	}

	// Build data structure for template
	data := struct {
		Project *content.ProjectMetadata
		Photos  []*content.PhotoInfo
	}{
		Project: proj,
		Photos:  photos,
	}

	// If this is an HTMX request, return only the partial; otherwise return full page
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		if err := s.tmpl.ExecuteTemplate(c.Writer, "projectConfig", data); err != nil {
			zlog.Error().Err(err).Msg("failed to render project config")
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// Full page render via shared helper to ensure consistent sidebar data
	if err := s.renderFullPage(c.Writer, "project.html", data); err != nil {
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
		// Load photos for the newly created project (will be empty initially)
		photos, _ := s.mgr.ListPhotos(proj.Slug)
		if photos == nil {
			photos = []*content.PhotoInfo{}
		}

		// Build data structure for template
		projectData := struct {
			Project *content.ProjectMetadata
			Photos  []*content.PhotoInfo
		}{
			Project: proj,
			Photos:  photos,
		}

		// Render combined response using the `projectCreated` template
		var respBuf bytes.Buffer
		respData := struct {
			Project interface{}
			Message string
		}{
			Project: projectData,
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

func (s *ServerNew) handleProjectUpdate(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		zlog.Warn().Msg("empty project slug for update")
		c.Status(http.StatusBadRequest)
		return
	}

	if s.mgr == nil {
		zlog.Error().Msg("no content manager configured")
		c.Status(http.StatusInternalServerError)
		return
	}

	// Load existing project
	proj, err := s.mgr.GetProject(slug)
	if err != nil {
		zlog.Error().Err(err).Str("slug", slug).Msg("project not found for update")
		c.Status(http.StatusNotFound)
		return
	}

	// Update fields from form
	title := c.PostForm("title")
	description := c.PostForm("description")
	hiddenStr := c.PostForm("hidden")

	if title == "" {
		c.String(http.StatusBadRequest, "title is required")
		return
	}

	proj.Title = title
	proj.Description = description
	proj.Hidden = hiddenStr == "on"

	// Save updated project
	if err := s.mgr.UpdateProject(slug, title, description, proj.Hidden); err != nil {
		zlog.Error().Err(err).Str("slug", slug).Msg("failed to update project")
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Return toast notification
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		var toastBuf bytes.Buffer
		toastData := struct{ Message string }{Message: fmt.Sprintf("Project %s updated", proj.Title)}
		if err := s.tmpl.ExecuteTemplate(&toastBuf, "toastMessage", toastData); err != nil {
			zlog.Warn().Err(err).Msg("failed to render toastMessage")
		} else {
			c.Status(http.StatusOK)
			c.Writer.Write(toastBuf.Bytes())
		}
		return
	}

	// Full page redirect
	c.Redirect(http.StatusSeeOther, "/projects/"+slug)
}

func (s *ServerNew) handlePhotoUpload(c *gin.Context) {
	slug := c.Param("slug")
	ctx := c.Request.Context()

	zlog.Info().Str("slug", slug).Msg("Starting photo upload")

	// Get the uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		zlog.Error().Err(err).Str("slug", slug).Msg("Failed to get uploaded file")
		c.String(http.StatusBadRequest, "no file uploaded")
		return
	}

	zlog.Info().
		Str("slug", slug).
		Str("filename", file.Filename).
		Int64("size", file.Size).
		Msg("Received photo upload")

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		zlog.Warn().Str("slug", slug).Str("contentType", contentType).Msg("Invalid content type")
		c.String(http.StatusBadRequest, "uploaded file is not an image")
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		zlog.Warn().Str("slug", slug).Str("extension", ext).Msg("Unsupported file extension")
		c.String(http.StatusBadRequest, "only .jpg, .jpeg, .png, and .webp files are supported")
		return
	}

	// Create image processor with configuration
	processor := processing.NewProcessor(processing.ProcessConfig{
		Widths:             []int{480, 800, 1200, 1920},
		Quality:            80,
		Force:              false, // Don't regenerate existing variants
		GenerateThumbnails: true,
		ThumbnailWidth:     300,
	})

	// Create source from multipart file
	source := &processing.MultipartFileSource{Header: file}

	// Compute hash to get the unique identifier
	hashFull, err := processor.ComputeHash(source)
	if err != nil {
		zlog.Error().Err(err).Str("slug", slug).Str("filename", file.Filename).Msg("Failed to compute image hash")
		c.String(http.StatusInternalServerError, "failed to process photo")
		return
	}
	hashID := hashFull[:12]

	zlog.Info().
		Str("slug", slug).
		Str("filename", file.Filename).
		Str("hashID", hashID).
		Msg("Computed image hash")

	// Set up destination directory: dist/images/{slug}
	imageOutputDir := filepath.Join(s.outputDir, "images", slug)
	destination := &processing.Destination{
		OutputDir: imageOutputDir,
	}

	// Process the image (create all variants and thumbnail)
	zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("Processing image variants")
	if err := processor.ProcessImage(source, destination); err != nil {
		zlog.Error().Err(err).Str("slug", slug).Str("hashID", hashID).Msg("Failed to process image")
		c.String(http.StatusInternalServerError, "failed to process photo")
		return
	}

	zlog.Info().
		Str("slug", slug).
		Str("hashID", hashID).
		Int("variants", len(processor.Config.Widths)).
		Bool("thumbnail", processor.Config.GenerateThumbnails).
		Msg("Image processing completed successfully")

	// Upload to S3/R2 if uploader is configured
	if s.uploader != nil {
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("Uploading processed images to remote storage")

		uploadCount := 0
		var uploadErrors []error

		// Upload all variants
		for _, width := range processor.Config.Widths {
			filename := fmt.Sprintf("%s-%dw.webp", hashID, width)
			variantPath := filepath.Join(imageOutputDir, hashID, filename)
			remoteKey := fmt.Sprintf("images/%s/%s/%s", slug, hashID, filename)

			// Open variant file
			variantFile, err := os.Open(variantPath)
			if err != nil {
				zlog.Error().Err(err).Str("variantPath", variantPath).Msg("Failed to open variant for upload")
				uploadErrors = append(uploadErrors, fmt.Errorf("failed to open %s: %w", filename, err))
				continue
			}

			// Upload variant
			if err := s.uploader.Upload(ctx, remoteKey, variantFile, "image/webp"); err != nil {
				zlog.Error().Err(err).Str("remoteKey", remoteKey).Msg("Failed to upload variant")
				uploadErrors = append(uploadErrors, fmt.Errorf("failed to upload %s: %w", filename, err))
				variantFile.Close()
				continue
			}
			variantFile.Close()

			zlog.Debug().Str("remoteKey", remoteKey).Msg("Uploaded variant")
			uploadCount++
		}

		// Upload thumbnail
		if processor.Config.GenerateThumbnails {
			thumbFilename := fmt.Sprintf("thumb-%s.webp", hashID)
			thumbPath := filepath.Join(imageOutputDir, ".thumbs", thumbFilename)
			remoteKey := fmt.Sprintf("images/%s/.thumbs/%s", slug, thumbFilename)

			thumbFile, err := os.Open(thumbPath)
			if err != nil {
				zlog.Error().Err(err).Str("thumbPath", thumbPath).Msg("Failed to open thumbnail for upload")
				uploadErrors = append(uploadErrors, fmt.Errorf("failed to open thumbnail: %w", err))
			} else {
				if err := s.uploader.Upload(ctx, remoteKey, thumbFile, "image/webp"); err != nil {
					zlog.Error().Err(err).Str("remoteKey", remoteKey).Msg("Failed to upload thumbnail")
					uploadErrors = append(uploadErrors, fmt.Errorf("failed to upload thumbnail: %w", err))
				} else {
					zlog.Debug().Str("remoteKey", remoteKey).Msg("Uploaded thumbnail")
					uploadCount++
				}
				thumbFile.Close()
			}
		}

		if len(uploadErrors) > 0 {
			zlog.Warn().
				Str("slug", slug).
				Str("hashID", hashID).
				Int("uploadedCount", uploadCount).
				Int("errorCount", len(uploadErrors)).
				Msg("Completed upload with some errors")
		} else {
			zlog.Info().
				Str("slug", slug).
				Str("hashID", hashID).
				Int("uploadedCount", uploadCount).
				Msg("All images uploaded successfully to remote storage")
		}
	} else {
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("No remote storage configured, images saved to filesystem only")
	}

	// Compute aspect ratio from one of the processed variants for the photo item
	variantPath := filepath.Join(imageOutputDir, hashID, fmt.Sprintf("%s-800w.webp", hashID))
	aspectRatio, ratioWidth, ratioHeight := 1.5, 3, 2 // defaults
	if imgFile, err := os.Open(variantPath); err == nil {
		if imgConfig, _, err := image.DecodeConfig(imgFile); err == nil {
			aspectRatio = float64(imgConfig.Width) / float64(imgConfig.Height)
			// Simple integer ratio calculation
			if aspectRatio >= 1.4 && aspectRatio <= 1.6 {
				ratioWidth, ratioHeight = 3, 2
			} else if aspectRatio >= 0.6 && aspectRatio <= 0.8 {
				ratioWidth, ratioHeight = 2, 3
			} else if aspectRatio >= 1.7 {
				ratioWidth, ratioHeight = 16, 9
			} else if aspectRatio <= 0.6 {
				ratioWidth, ratioHeight = 9, 16
			} else if aspectRatio >= 0.95 && aspectRatio <= 1.05 {
				ratioWidth, ratioHeight = 1, 1
			}
		}
		imgFile.Close()
	}

	// Build thumbnail path
	thumbFilename := fmt.Sprintf("thumb-%s.webp", hashID)
	thumbPath := fmt.Sprintf("/images/%s/.thumbs/%s", slug, thumbFilename)
	if s.imageURLPrefix != "" {
		thumbPath = fmt.Sprintf("%s/static/images/%s/.thumbs/%s", s.imageURLPrefix, slug, thumbFilename)
	}

	// Return toast notification and new photo item with OOB swap
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		var responseBuf bytes.Buffer

		// Render toast message
		toastData := struct{ Message string }{
			Message: fmt.Sprintf("Photo processed successfully (ID: %s)", hashID),
		}
		if err := s.tmpl.ExecuteTemplate(&responseBuf, "toastMessage", toastData); err != nil {
			zlog.Warn().Err(err).Msg("Failed to render toastMessage")
		}

		// Render photo item wrapped with OOB swap
		var photoItemBuf bytes.Buffer
		photoData := struct {
			HashID      string
			Filename    string
			ThumbPath   string
			RatioWidth  int
			RatioHeight int
			ProjectSlug string
		}{
			HashID:      hashID,
			Filename:    thumbFilename,
			ThumbPath:   thumbPath,
			RatioWidth:  ratioWidth,
			RatioHeight: ratioHeight,
			ProjectSlug: slug,
		}
		if err := s.tmpl.ExecuteTemplate(&photoItemBuf, "photoItem", photoData); err != nil {
			zlog.Error().Err(err).Msg("Failed to render photoItem")
		} else {
			// Wrap the photo item with OOB swap directive
			responseBuf.WriteString(`<div hx-swap-oob="beforeend:#photo-grid">`)
			responseBuf.Write(photoItemBuf.Bytes())
			responseBuf.WriteString(`</div>`)
		}

		c.Status(http.StatusOK)
		c.Writer.Write(responseBuf.Bytes())
		return
	}

	// Full page redirect
	c.Redirect(http.StatusSeeOther, "/projects/"+slug)
}

func (s *ServerNew) handlePhotoDelete(c *gin.Context) {
	slug := c.Param("slug")
	hashID := c.Param("hashID")
	ctx := c.Request.Context()

	zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("Starting photo deletion")

	// Validate hashID format (should be 12 characters)
	if len(hashID) != 12 {
		zlog.Warn().Str("slug", slug).Str("hashID", hashID).Msg("Invalid hashID format")
		c.String(http.StatusBadRequest, "invalid photo ID")
		return
	}

	imageOutputDir := filepath.Join(s.outputDir, "images", slug)

	// Delete the variant directory: dist/images/{slug}/{hashID}/
	variantDir := filepath.Join(imageOutputDir, hashID)
	if _, err := os.Stat(variantDir); err == nil {
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Str("path", variantDir).Msg("Deleting variant directory")
		if err := os.RemoveAll(variantDir); err != nil {
			zlog.Error().Err(err).Str("slug", slug).Str("hashID", hashID).Msg("Failed to delete variant directory")
			c.String(http.StatusInternalServerError, "failed to delete photo variants")
			return
		}
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("Deleted variant directory")
	} else {
		zlog.Warn().Str("slug", slug).Str("hashID", hashID).Msg("Variant directory not found")
	}

	// Delete the thumbnail: dist/images/{slug}/.thumbs/thumb-{hashID}.webp
	thumbPath := filepath.Join(imageOutputDir, ".thumbs", fmt.Sprintf("thumb-%s.webp", hashID))
	if _, err := os.Stat(thumbPath); err == nil {
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Str("path", thumbPath).Msg("Deleting thumbnail")
		if err := os.Remove(thumbPath); err != nil {
			zlog.Error().Err(err).Str("slug", slug).Str("hashID", hashID).Msg("Failed to delete thumbnail")
			// Continue anyway, variants are more important
		}
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("Deleted thumbnail")
	} else {
		zlog.Warn().Str("slug", slug).Str("hashID", hashID).Msg("Thumbnail not found")
	}

	// Delete from remote storage if uploader is configured
	if s.uploader != nil {
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("Deleting from remote storage")

		deleteCount := 0
		var deleteErrors []error

		// Delete all variants
		widths := []int{480, 800, 1200, 1920}
		for _, width := range widths {
			filename := fmt.Sprintf("%s-%dw.webp", hashID, width)
			remoteKey := fmt.Sprintf("images/%s/%s/%s", slug, hashID, filename)

			if err := s.uploader.Delete(ctx, remoteKey); err != nil {
				zlog.Error().Err(err).Str("remoteKey", remoteKey).Msg("Failed to delete variant from remote")
				deleteErrors = append(deleteErrors, fmt.Errorf("failed to delete %s: %w", filename, err))
				continue
			}
			zlog.Debug().Str("remoteKey", remoteKey).Msg("Deleted variant from remote")
			deleteCount++
		}

		// Delete thumbnail from remote
		thumbFilename := fmt.Sprintf("thumb-%s.webp", hashID)
		remoteKey := fmt.Sprintf("images/%s/.thumbs/%s", slug, thumbFilename)
		if err := s.uploader.Delete(ctx, remoteKey); err != nil {
			zlog.Error().Err(err).Str("remoteKey", remoteKey).Msg("Failed to delete thumbnail from remote")
			deleteErrors = append(deleteErrors, fmt.Errorf("failed to delete thumbnail: %w", err))
		} else {
			zlog.Debug().Str("remoteKey", remoteKey).Msg("Deleted thumbnail from remote")
			deleteCount++
		}

		if len(deleteErrors) > 0 {
			zlog.Warn().
				Str("slug", slug).
				Str("hashID", hashID).
				Int("deletedCount", deleteCount).
				Int("errorCount", len(deleteErrors)).
				Msg("Completed remote deletion with some errors")
		} else {
			zlog.Info().
				Str("slug", slug).
				Str("hashID", hashID).
				Int("deletedCount", deleteCount).
				Msg("All files deleted from remote storage")
		}
	} else {
		zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("No remote storage configured, local deletion only")
	}

	zlog.Info().Str("slug", slug).Str("hashID", hashID).Msg("Photo deletion completed successfully")

	// Return toast notification (HTMX will remove the photo item from DOM)
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		var toastBuf bytes.Buffer
		toastData := struct{ Message string }{
			Message: fmt.Sprintf("Photo deleted successfully (ID: %s)", hashID),
		}
		if err := s.tmpl.ExecuteTemplate(&toastBuf, "toastMessage", toastData); err != nil {
			zlog.Warn().Err(err).Msg("Failed to render toastMessage")
		} else {
			c.Status(http.StatusOK)
			c.Writer.Write(toastBuf.Bytes())
		}
		return
	}

	// Non-HTMX fallback
	c.Status(http.StatusOK)
}

func ServeNew() {
	srv := NewServerNew()
	srv.Serve()
}

func (s *ServerNew) handleConfig(c *gin.Context) {
	if s.mgr == nil {
		zlog.Error().Msg("no content manager configured")
		c.Status(http.StatusInternalServerError)
		return
	}

	meta, err := s.mgr.LoadSiteMeta()
	if err != nil {
		zlog.Error().Err(err).Msg("failed to load site metadata")
		// Continue with empty metadata explicitly handled if needed, or error out
		// LoadSiteMeta returns default if not found, so error is real IO error
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Helper to render response
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		if err := s.tmpl.ExecuteTemplate(c.Writer, "config", meta); err != nil {
			zlog.Error().Err(err).Msg("failed to render config partial")
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// Full page render
	// We need a wrapper or use index.html with conditional content?
	// The current renderFullPage assumes "Content" is injected into main.
	// But index.html implementation (based on sidebar) seems to expect nothing in main initially?
	// Wait, let's look at renderFullPage again.
	// It uses "index.html" or "project.html".
	// We might need a "config-full.html" or use a generic wrapper.
	// For now, let's assume we can reuse a layout or create a simple wrapper.
	// Actually, looking at renderFullPage, it executes a named template.
	// "index.html" has {{template "sidebar" .}} {{template "main" .}} {{template "toast" .}}
	// "main" usually comes from the page template itself? No, "main" is defined in main.html presumably?
	// Let's check main.html.

	// If I use renderFullPage with "config-full.html", I need that file.
	// Alternatively, I can use a generic layout and inject the "config" template as content.
	// But `templates` usually define the structure.

	// Let's create a dynamic render or just use "index.html" style but with config injected.
	// Actually, `handleHome` uses `index.html`.
	// `handleProject` uses `project.html`.
	// I should probably have `config-full.html` that extends the base layout and includes `config`.

	// For this step, I'll render "config.html" directly if partial.
	// For full page, I'll assume I need to create `config_page.html` or similar.
	// Let's check if I can just pass the config template as the main content.
	// The `renderFullPage` takes a template name.

	// I will use "configPage" as the template name for full page, and I will define it.

	if err := s.renderFullPage(c.Writer, "config_page.html", meta); err != nil {
		zlog.Error().Err(err).Msg("failed to render config page")
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (s *ServerNew) handleConfigSave(c *gin.Context) {
	if s.mgr == nil {
		zlog.Error().Msg("no content manager configured")
		c.Status(http.StatusInternalServerError)
		return
	}

	// Load existing meta to preserve other fields (like Projects)
	meta, err := s.mgr.LoadSiteMeta()
	if err != nil {
		zlog.Error().Err(err).Msg("failed to load existing site metadata")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Update fields from form
	meta.WebsiteName = c.PostForm("website_name")
	meta.Copyright = c.PostForm("copyright")
	meta.LogoPrimary = c.PostForm("logo_primary")
	meta.LogoSecondary = c.PostForm("logo_secondary")

	// Update About
	if meta.About == nil {
		meta.About = &content.About{}
	}
	meta.About.Title = c.PostForm("about_title")
	meta.About.Quote = c.PostForm("about_quote")
	meta.About.QuoteSource = c.PostForm("about_quote_source")

	paragraphsRaw := c.PostForm("about_paragraphs")
	// Split by double newline to treat each block as a separate paragraph
	var paragraphs []string
	// Standardize line endings
	paragraphsRaw = strings.ReplaceAll(paragraphsRaw, "\r\n", "\n")
	// Split by double newlines (or more) to get separate paragraphs
	for _, p := range strings.Split(paragraphsRaw, "\n\n") {
		// Trim whitespace from each paragraph
		p = strings.TrimSpace(p)
		// Skip empty paragraphs
		if p != "" {
			paragraphs = append(paragraphs, p)
		}
	}
	meta.About.Paragraphs = paragraphs

	// Update Contact
	if meta.Contact == nil {
		meta.Contact = &content.Contact{}
	}
	meta.Contact.Email = c.PostForm("contact_email")
	meta.Contact.Instagram = c.PostForm("contact_instagram")
	meta.Contact.Website = c.PostForm("contact_website")

	// Update Projects order
	// Parse all project_slug_N and project_order_N pairs
	var projectOrders []content.ProjectOrder
	for i := 0; ; i++ {
		slugKey := fmt.Sprintf("project_slug_%d", i)
		orderKey := fmt.Sprintf("project_order_%d", i)

		slug := c.PostForm(slugKey)
		if slug == "" {
			break // No more projects
		}

		orderStr := c.PostForm(orderKey)
		order := 0
		if orderStr != "" {
			fmt.Sscanf(orderStr, "%d", &order)
		}

		projectOrders = append(projectOrders, content.ProjectOrder{
			Slug:  slug,
			Order: order,
		})
	}
	meta.Projects = projectOrders

	// Save
	if err := s.mgr.SaveSiteMeta(meta); err != nil {
		zlog.Error().Err(err).Msg("failed to save site metadata")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Render response
	// If HTMX, return only success toast OOB
	hx := c.GetHeader("HX-Request")
	if hx != "" {
		// Toast
		var toastBuf bytes.Buffer
		toastData := struct{ Message string }{Message: "Configuration saved"}

		if err := s.tmpl.ExecuteTemplate(&toastBuf, "toastMessage", toastData); err != nil {
			zlog.Warn().Err(err).Msg("failed to render toastMessage")
		} else {
			c.Status(http.StatusOK)
			c.Writer.Write(toastBuf.Bytes())
		}
		return
	}

	// Full page redirect
	c.Redirect(http.StatusSeeOther, "/config")
}
