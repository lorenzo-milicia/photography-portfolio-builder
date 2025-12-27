package builder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Response helpers for standardized HTMX responses

// ResponseWriter wraps http.ResponseWriter with helper methods for HTMX responses
type ResponseWriter struct {
	w http.ResponseWriter
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{w: w}
}

// ToastType represents the type of toast notification
type ToastType string

const (
	ToastSuccess ToastType = "success"
	ToastError   ToastType = "error"
	ToastInfo    ToastType = "info"
	ToastWarning ToastType = "warning"
)

// Toast sends a toast notification via HX-Trigger event
func (rw *ResponseWriter) Toast(toastType ToastType, message string) {
	events := map[string]interface{}{
		"showMessage": map[string]string{
			"type":    string(toastType),
			"message": message,
		},
	}
	rw.setTriggerEvents(events)
}

// Success sends a success toast
func (rw *ResponseWriter) Success(message string) {
	rw.Toast(ToastSuccess, message)
}

// Error sends an error toast
func (rw *ResponseWriter) Error(message string) {
	rw.Toast(ToastError, message)
}

// Info sends an info toast
func (rw *ResponseWriter) Info(message string) {
	rw.Toast(ToastInfo, message)
}

// Warning sends a warning toast
func (rw *ResponseWriter) Warning(message string) {
	rw.Toast(ToastWarning, message)
}

// Redirect sends an HTMX redirect
func (rw *ResponseWriter) Redirect(url string) {
	rw.w.Header().Set("HX-Redirect", url)
}

// Trigger adds custom HX-Trigger events
func (rw *ResponseWriter) Trigger(events map[string]interface{}) {
	rw.setTriggerEvents(events)
}

// setTriggerEvents sets HX-Trigger header with JSON events
func (rw *ResponseWriter) setTriggerEvents(events map[string]interface{}) {
	eventJSON, err := json.Marshal(events)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal trigger events")
		return
	}
	rw.w.Header().Set("HX-Trigger", string(eventJSON))
}

// OKWithToast sends 200 OK with a toast notification
func (rw *ResponseWriter) OKWithToast(toastType ToastType, message string) {
	rw.Toast(toastType, message)
	rw.w.WriteHeader(http.StatusOK)
}

// ErrorResponse sends an error response with toast notification
func (rw *ResponseWriter) ErrorResponse(message string, statusCode int) {
	rw.Error(message)
	rw.w.WriteHeader(statusCode)
}

// renderError renders an error message as HTML (for non-HTMX fallback)
func (rw *ResponseWriter) renderError(message string, statusCode int) {
	rw.w.WriteHeader(statusCode)
	fmt.Fprintf(rw.w, `<div class="alert alert-error">
		<span>❌</span>
		<div>%s</div>
	</div>`, template.HTMLEscapeString(message))
}

// TemplateRenderer handles template rendering with consistent error handling
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(templates *template.Template) *TemplateRenderer {
	return &TemplateRenderer{templates: templates}
}

// RenderPartial renders a template partial for HTMX requests
func (tr *TemplateRenderer) RenderPartial(w http.ResponseWriter, templateName string, data interface{}) error {
	if err := tr.templates.ExecuteTemplate(w, templateName, data); err != nil {
		log.Error().Err(err).Str("template", templateName).Msg("Template execution failed")
		return err
	}
	return nil
}

// RenderPartialOrFull renders a partial for HTMX requests, or wraps in full page for direct navigation
func (tr *TemplateRenderer) RenderPartialOrFull(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) error {
	isHtmxRequest := r.Header.Get("HX-Request") != ""

	if isHtmxRequest {
		// For HTMX requests, render just the partial
		return tr.RenderPartial(w, templateName, data)
	}

	// For direct navigation, render partial into buffer first
	var buf bytes.Buffer
	if err := tr.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		log.Error().Err(err).Str("template", templateName).Msg("Template execution failed")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return err
	}

	// Wrap in full page layout
	pageData := map[string]interface{}{
		"Content": template.HTML(buf.String()),
	}

	// Preserve any other data fields that might be needed by the page wrapper
	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			if k != "Content" {
				pageData[k] = v
			}
		}
	}

	return tr.RenderPartial(w, "index.html", pageData)
}

// RenderWithToast renders a template and adds a toast notification
func (tr *TemplateRenderer) RenderWithToast(w http.ResponseWriter, templateName string, data interface{}, toastType ToastType, message string) error {
	rw := NewResponseWriter(w)
	rw.Toast(toastType, message)
	return tr.RenderPartial(w, templateName, data)
}

// RenderJSON renders data as JSON
func (tr *TemplateRenderer) RenderJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("JSON encoding failed")
		return err
	}
	return nil
}
