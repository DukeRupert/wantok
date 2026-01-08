package render

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
)

//go:embed templates/layouts/*.html templates/pages/*.html
var templateFS embed.FS

// Renderer handles template rendering.
type Renderer struct {
	templates *template.Template
}

// New creates a new template renderer.
func New() (*Renderer, error) {
	tmpl, err := template.New("").ParseFS(templateFS,
		"templates/layouts/*.html",
		"templates/pages/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	return &Renderer{templates: tmpl}, nil
}

// Render renders a full page template.
func (r *Renderer) Render(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := r.templates.ExecuteTemplate(w, name, data); err != nil {
		return fmt.Errorf("executing template %s: %w", name, err)
	}
	return nil
}

// RenderPartial renders a partial template (for HTMX responses).
func (r *Renderer) RenderPartial(w io.Writer, name string, data any) error {
	if err := r.templates.ExecuteTemplate(w, name, data); err != nil {
		return fmt.Errorf("executing partial %s: %w", name, err)
	}
	return nil
}
