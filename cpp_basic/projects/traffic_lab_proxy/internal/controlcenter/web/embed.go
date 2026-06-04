package web

import (
	"io/fs"
	"os"
	"path/filepath"
)

// WebRoot is the path to the web directory. During development it's relative
// to the project root. In Docker, set WEB_ROOT=/app/web .
var WebRoot = findWebRoot()

func findWebRoot() string {
	// Check environment variable first
	if root := os.Getenv("WEB_ROOT"); root != "" {
		return root
	}

	// Try common locations relative to the working directory
	candidates := []string{
		"web",
		"../web",
		"../../web",
		"../../../web",
	}

	for _, c := range candidates {
		if info, err := os.Stat(filepath.Join(c, "templates")); err == nil && info.IsDir() {
			return c
		}
	}

	// Default fallback
	return "web"
}

// TemplatesFS returns an os.DirFS for the templates directory.
func TemplatesFS() fs.FS {
	return os.DirFS(filepath.Join(WebRoot, "templates"))
}

// StaticFS returns an os.DirFS for the static files directory.
func StaticFS() fs.FS {
	return os.DirFS(filepath.Join(WebRoot, "static"))
}
