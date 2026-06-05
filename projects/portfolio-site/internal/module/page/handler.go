package page

import (
	"log/slog"
	"net/http"
	"time"

	"portfolio-site/internal/module/article"
	"portfolio-site/internal/module/project"
	"portfolio-site/internal/render"
)

// Handler handles public page requests.
type Handler struct {
	projectSvc *project.Service
	articleSvc *article.Service
	render     *render.Renderer
	logger     *slog.Logger
	siteTitle  string
}

// NewHandler creates a new page handler.
func NewHandler(
	projectSvc *project.Service,
	articleSvc *article.Service,
	render *render.Renderer,
	logger *slog.Logger,
	siteTitle string,
) *Handler {
	return &Handler{
		projectSvc: projectSvc,
		articleSvc: articleSvc,
		render:     render,
		logger:     logger,
		siteTitle:  siteTitle,
	}
}

// Home renders the homepage.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	featuredProjects, err := h.projectSvc.GetFeatured(r.Context(), 4)
	if err != nil {
		h.logger.Error("failed to get featured projects", "error", err)
		featuredProjects = []project.Project{}
	}

	latestArticles, _, err := h.articleSvc.ListPublished(r.Context(), article.Query{
		Page:     1,
		PageSize: 5,
		Status:   string(article.StatusPublished),
		OrderBy:  "published_at DESC",
	})
	if err != nil {
		h.logger.Error("failed to get latest articles", "error", err)
		latestArticles = []article.Article{}
	}

	h.render.Render(w, "home", map[string]interface{}{
		"Title":            h.siteTitle,
		"Description":      "Building quiet, reliable engineering systems.",
		"FeaturedProjects": featuredProjects,
		"LatestArticles":   latestArticles,
		"CurrentYear":      time.Now().Year(),
	})
}

// About renders the about page.
func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	h.render.Render(w, "about", map[string]interface{}{
		"Title":       "About - " + h.siteTitle,
		"Description": "About my engineering journey and technical focus.",
	})
}

// Resume renders the resume page.
func (h *Handler) Resume(w http.ResponseWriter, r *http.Request) {
	h.render.Render(w, "resume", map[string]interface{}{
		"Title":       "Resume - " + h.siteTitle,
		"Description": "Professional resume and work experience.",
	})
}

// Contact renders the contact page.
func (h *Handler) Contact(w http.ResponseWriter, r *http.Request) {
	h.render.Render(w, "contact", map[string]interface{}{
		"Title":       "Contact - " + h.siteTitle,
		"Description": "Get in touch for collaboration or inquiries.",
	})
}

// SubmitContact processes a contact form submission.
func (h *Handler) SubmitContact(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render.ErrorPage(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	subject := r.FormValue("subject")

	h.logger.Info("contact form submitted",
		"name", name,
		"email", email,
		"subject", subject,
	)

	// In a real app, save to database and send email notification.

	h.render.Render(w, "contact", map[string]interface{}{
		"Title":       "Contact - " + h.siteTitle,
		"Description": "Get in touch for collaboration or inquiries.",
		"Success":     "Thank you for your message! I will get back to you soon.",
	})
}

// NotFound renders the 404 page.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.render.NotFound(w)
}

// RobotsTxt serves robots.txt.
func (h *Handler) RobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("User-agent: *\nAllow: /\nSitemap: /sitemap.xml\n"))
}

// Sitemap serves a basic sitemap.
func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>/</loc></url>
  <url><loc>/projects</loc></url>
  <url><loc>/writings</loc></url>
  <url><loc>/about</loc></url>
  <url><loc>/resume</loc></url>
  <url><loc>/contact</loc></url>
</urlset>`))
}

// RSS serves the RSS feed.
func (h *Handler) RSS(w http.ResponseWriter, r *http.Request) {
	articles, _, err := h.articleSvc.ListPublished(r.Context(), article.Query{
		Page:     1,
		PageSize: 20,
		Status:   string(article.StatusPublished),
		OrderBy:  "published_at DESC",
	})
	if err != nil {
		h.render.ErrorPage(w, http.StatusInternalServerError, "Failed to generate RSS feed")
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
  <title>` + h.siteTitle + `</title>
  <link>/</link>
  <description>Engineering notes and field records.</description>
  <atom:link href="/rss.xml" rel="self" type="application/rss+xml"/>
`))
	for _, a := range articles {
		pubDate := ""
		if a.PublishedAt != nil {
			pubDate = a.PublishedAt.Format(time.RFC1123Z)
		}
		w.Write([]byte(`  <item>
    <title>` + a.Title + `</title>
    <link>/writings/` + a.Slug + `</link>
    <description>` + a.Summary + `</description>
    <pubDate>` + pubDate + `</pubDate>
    <guid>/writings/` + a.Slug + `</guid>
  </item>
`))
	}
	w.Write([]byte(`</channel>
</rss>`))
}
