package asset

import "time"

// Asset represents an uploaded file.
type Asset struct {
	ID           int64     `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name,omitempty"`
	Path         string    `json:"path"`
	MimeType     string    `json:"mime_type,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	UsageType    string    `json:"usage_type,omitempty"`
	AltText      string    `json:"alt_text,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// UsageType categorizes assets.
type UsageType string

const (
	UsageProjectCover      UsageType = "project_cover"
	UsageProjectScreenshot UsageType = "project_screenshot"
	UsageArticleCover      UsageType = "article_cover"
	UsageResumePDF         UsageType = "resume_pdf"
	UsageAvatar            UsageType = "avatar"
	UsageLogo              UsageType = "logo"
	UsageDiagram           UsageType = "architecture_diagram"
)
