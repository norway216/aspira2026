package render

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Renderer handles HTML template rendering.
type Renderer struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

// NewRenderer creates a new template renderer.
func NewRenderer(templateDir string) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		funcMap: template.FuncMap{
			"formatDate":     func(t time.Time) string { return t.Format("2006-01-02") },
			"formatDateLong": func(t time.Time) string { return t.Format("January 2, 2006") },
			"formatDateTime": func(t time.Time) string { return t.Format("2006-01-02 15:04") },
			"truncate":       func(s string, n int) string { return truncate(s, n) },
			"add":            func(a, b int) int { return a + b },
			"sub":            func(a, b int) int { return a - b },
			"dict":           dict,
			"safeHTML":       func(s string) template.HTML { return template.HTML(s) },
			"split":          func(s, sep string) []string { return strings.Split(s, sep) },
		},
	}

	// Load all templates from the directory.
	pattern := filepath.Join(templateDir, "**", "*.html")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob templates: %w", err)
	}

	// Also try pages and layout subdirs.
	for _, sub := range []string{"layout", "pages", "admin"} {
		subPattern := filepath.Join(templateDir, sub, "*.html")
		subFiles, err := filepath.Glob(subPattern)
		if err == nil {
			files = append(files, subFiles...)
		}
	}

	for _, file := range files {
		name := filepath.Base(file)
		name = strings.TrimSuffix(name, ".html")

		tmpl := template.New(name).Funcs(r.funcMap)
		// Parse all layout files first.
		layoutPattern := filepath.Join(templateDir, "layout", "*.html")
		layoutFiles, _ := filepath.Glob(layoutPattern)
		if len(layoutFiles) > 0 {
			allFiles := append(layoutFiles, file)
			tmpl, err = tmpl.ParseFiles(allFiles...)
		} else {
			tmpl, err = tmpl.ParseFiles(file)
		}
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		r.templates[name] = tmpl
	}

	return r, nil
}

// Render executes a template and writes the result to w.
func (r *Renderer) Render(w http.ResponseWriter, name string, data interface{}) error {
	tmpl, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, "base", data)
}

// RenderWithName executes a named template and writes the result.
func (r *Renderer) RenderWithName(w http.ResponseWriter, templateName, layoutName string, data interface{}) error {
	tmpl, ok := r.templates[templateName]
	if !ok {
		return fmt.Errorf("template %s not found", templateName)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, layoutName, data)
}

// RenderString renders a template to a string.
func (r *Renderer) RenderString(name string, data interface{}) (string, error) {
	tmpl, ok := r.templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires even number of arguments")
	}
	m := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings")
		}
		m[key] = values[i+1]
	}
	return m, nil
}

// ErrorPage renders a standard error page.
func (r *Renderer) ErrorPage(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	data := map[string]interface{}{
		"Title":       fmt.Sprintf("%d - %s", statusCode, http.StatusText(statusCode)),
		"Description": message,
		"StatusCode":  statusCode,
	}
	// Fallback to plain text if template not available.
	if err := r.Render(w, "error", data); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, fmt.Sprintf("%d %s: %s", statusCode, http.StatusText(statusCode), message))
	}
}

// NotFound renders a 404 page.
func (r *Renderer) NotFound(w http.ResponseWriter) {
	r.ErrorPage(w, http.StatusNotFound, "Page not found")
}

// ServerError renders a 500 page.
func (r *Renderer) ServerError(w http.ResponseWriter) {
	r.ErrorPage(w, http.StatusInternalServerError, "Internal server error")
}

// Forbidden renders a 403 page.
func (r *Renderer) Forbidden(w http.ResponseWriter) {
	r.ErrorPage(w, http.StatusForbidden, "Access denied")
}
