package web

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"time"
)

// Templates manages HTML template rendering.
type Templates struct {
	templates map[string]*template.Template
	partials  map[string]*template.Template
	funcs     template.FuncMap
}

// NewTemplates creates a new template manager by loading templates from the given filesystem.
func NewTemplates(templatesFS fs.FS) (*Templates, error) {
	t := &Templates{
		templates: make(map[string]*template.Template),
		partials:  make(map[string]*template.Template),
		funcs:     defaultFuncs(),
	}

	if err := t.load(templatesFS); err != nil {
		return nil, err
	}

	return t, nil
}

// Render renders a page template with the given data.
func (t *Templates) Render(w io.Writer, page string, data any) error {
	tmpl, ok := t.templates[page]
	if !ok {
		return fmt.Errorf("template %q not found", page)
	}

	// Execute the "base" template which includes the page content
	return tmpl.ExecuteTemplate(w, "base", data)
}

// RenderPartial renders a partial template (without base layout) with the given data.
func (t *Templates) RenderPartial(w io.Writer, partial string, data any) error {
	tmpl, ok := t.partials[partial]
	if !ok {
		return fmt.Errorf("partial %q not found", partial)
	}
	return tmpl.Execute(w, data)
}

// load parses all templates from the filesystem.
func (t *Templates) load(templatesFS fs.FS) error {
	// Load base layout
	layoutPattern := "layouts/*.html"
	layouts, err := fs.Glob(templatesFS, layoutPattern)
	if err != nil {
		return fmt.Errorf("finding layouts: %w", err)
	}

	// Load partials
	partialPattern := "partials/*.html"
	partials, err := fs.Glob(templatesFS, partialPattern)
	if err != nil {
		return fmt.Errorf("finding partials: %w", err)
	}

	// Load each page template with layouts and partials
	pagePattern := "pages/*.html"
	pages, err := fs.Glob(templatesFS, pagePattern)
	if err != nil {
		return fmt.Errorf("finding pages: %w", err)
	}

	// Common files to include with every page
	commonFiles := append(layouts, partials...)

	for _, page := range pages {
		// Create a new template for each page
		name := filepath.Base(page)
		name = name[:len(name)-len(".html")] // Remove .html extension

		files := append([]string{page}, commonFiles...)

		tmpl, err := template.New(name).Funcs(t.funcs).ParseFS(templatesFS, files...)
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", name, err)
		}

		// Execute "base" layout if it exists, otherwise the page itself
		t.templates[name] = tmpl
	}

	// Load partials as standalone templates for HTMX fragments
	for _, partial := range partials {
		name := filepath.Base(partial)
		name = name[:len(name)-len(".html")] // Remove .html extension

		tmpl, err := template.New(name).Funcs(t.funcs).ParseFS(templatesFS, partial)
		if err != nil {
			return fmt.Errorf("parsing partial %s: %w", name, err)
		}
		t.partials[name] = tmpl
	}

	return nil
}

// defaultFuncs returns the default template functions.
func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		// MoodColor returns an HSL color string based on energy and valence.
		// Energy maps to hue (cool indigo → warm orange)
		// Valence affects saturation and lightness
		"moodColor": func(energy, valence float64) string {
			hue := 264 - (energy * 229)
			if hue < 0 {
				hue += 360
			}
			saturation := 60 + (valence * 40)
			lightness := 40 + (valence * 20)
			return fmt.Sprintf("hsl(%.0f, %.0f%%, %.0f%%)", hue, saturation, lightness)
		},

		// formatDate formats a time as "Jan 2, 2006"
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},

		// formatDateRange formats a date range as "Jan 2 - Feb 3, 2006"
		"formatDateRange": func(start, end time.Time) string {
			if start.Year() == end.Year() && start.Month() == end.Month() {
				return fmt.Sprintf("%s - %s", start.Format("Jan 2"), end.Format("2, 2006"))
			}
			if start.Year() == end.Year() {
				return fmt.Sprintf("%s - %s", start.Format("Jan 2"), end.Format("Jan 2, 2006"))
			}
			return fmt.Sprintf("%s - %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))
		},

		// safeHTML marks a string as safe HTML (use with caution)
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s) //nolint:gosec // Intentional for trusted content
		},

		// add adds two integers (for 1-based indexing in loops)
		"add": func(a, b int) int {
			return a + b
		},
	}
}

// PageData contains common data passed to all page templates.
type PageData struct {
	Title       string
	User        *UserData
	Flash       *FlashMessage
	CurrentPath string
}

// UserData contains authenticated user information.
type UserData struct {
	ID   string
	Name string
}

// FlashMessage represents a temporary notification message.
type FlashMessage struct {
	Type    string // "success", "error", "warning", "info"
	Message string
}

// HomePageData contains data for the home page template.
type HomePageData struct {
	PageData
	Authenticated bool
}

// ErasPageData contains data for the eras page template.
type ErasPageData struct {
	PageData
	Eras []EraData
}

// EraData contains data for a single era in templates.
type EraData struct {
	ID         string
	Name       string
	TopTags    []string
	StartDate  time.Time
	EndDate    time.Time
	TrackCount int
	PlaylistID *string
}

// TrackData contains data for a single track in templates.
type TrackData struct {
	ID     string
	Name   string
	Artist string
	Album  string
}
