# Implementation Notes: Go Templates + HTMX

High-level patterns for implementing the brand guidelines in the Go web stack.

## File Organization

```
web/
  templates/
    layouts/
      base.html           # Main layout with head, nav, scripts
    partials/
      era-card.html       # Era card component
      track-list.html     # Track list component
      loading.html        # Loading/skeleton states
    pages/
      home.html           # Landing/auth page
      eras.html           # Era detection results
      
  static/
    css/
      variables.css       # CSS custom properties (from brand guidelines)
      base.css            # Reset, typography, base styles
      components.css      # Component styles
      utilities.css       # Utility classes
    js/
      htmx.min.js         # HTMX library
      app.js              # Custom interactions (if needed)
```

## Go Template Patterns

### Mood Color Function

Register a template function for dynamic mood colors:

```go
func templateFuncs() template.FuncMap {
    return template.FuncMap{
        "moodColor": func(energy, valence float64) string {
            hue := 264 - (energy * 229)
            if hue < 0 {
                hue += 360
            }
            saturation := 60 + (valence * 40)
            lightness := 40 + (valence * 20)
            return fmt.Sprintf("hsl(%.0f, %.0f%%, %.0f%%)", hue, saturation, lightness)
        },
    }
}
```

### Era Card Partial

```html
{{define "era-card"}}
<article class="era-card" style="--era-color: {{moodColor .AvgEnergy .AvgValence}}">
  <div class="era-card__header">
    <h2 class="era-card__title">{{.Name}}</h2>
    <span class="era-card__dates">{{.StartDate.Format "Jan 2"}} - {{.EndDate.Format "Jan 2, 2006"}}</span>
  </div>
  <div class="era-card__mood-bar"></div>
  <div class="era-card__stats">
    <span>{{len .Tracks}} tracks</span>
  </div>
  <div class="era-card__actions">
    <button 
      class="btn-primary"
      hx-post="/api/playlists"
      hx-vals='{"era_id": "{{.ID}}"}'
      hx-swap="outerHTML"
      hx-indicator="#spinner-{{.ID}}"
    >
      Create Playlist
    </button>
    <button 
      class="btn-secondary"
      hx-get="/api/eras/{{.ID}}/tracks"
      hx-target="#tracks-{{.ID}}"
      hx-swap="innerHTML"
    >
      Show Tracks
    </button>
  </div>
  <div id="tracks-{{.ID}}" class="era-card__tracks"></div>
</article>
{{end}}
```

## HTMX Patterns

### Loading States with hx-indicator

```html
<button 
  hx-post="/api/detect-eras"
  hx-target="#results"
  hx-indicator="#loading"
>
  Detect Eras
</button>

<div id="loading" class="htmx-indicator">
  <div class="spinner"></div>
  <span>Analyzing your music...</span>
</div>
```

```css
.htmx-indicator {
  display: none;
}

.htmx-request .htmx-indicator,
.htmx-request.htmx-indicator {
  display: flex;
}
```

### Partial Updates for Era Expansion

```html
<!-- Expand track list -->
<button 
  hx-get="/api/eras/{{.ID}}/tracks"
  hx-target="#tracks-{{.ID}}"
  hx-swap="innerHTML"
  hx-trigger="click"
>
  Show Tracks
</button>

<div id="tracks-{{.ID}}">
  <!-- Tracks loaded here via HTMX -->
</div>
```

### Out-of-Band Updates

For updating multiple elements after an action:

```go
// Handler returns multiple elements with hx-swap-oob
func createPlaylistHandler(w http.ResponseWriter, r *http.Request) {
    // ... create playlist logic ...
    
    w.Header().Set("Content-Type", "text/html")
    tmpl.ExecuteTemplate(w, "playlist-created", data)
    // Include OOB update for notification
    tmpl.ExecuteTemplate(w, "notification-oob", notification)
}
```

```html
{{define "notification-oob"}}
<div id="notifications" hx-swap-oob="beforeend">
  <div class="toast toast--success">
    Playlist created!
  </div>
</div>
{{end}}
```

## CSS Organization

### variables.css

Contains all CSS custom properties from the brand guidelines skill. Import first.

### base.css

```css
@import url('https://fonts.googleapis.com/css2?family=Bebas+Neue&family=JetBrains+Mono:wght@400;500;600&family=Oswald:wght@400;500;600;700&display=swap');

*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html {
  background: var(--bg-deep);
  color: var(--text-primary);
  font-family: var(--font-body);
}

/* ... base element styles ... */
```

### components.css

Component classes from the brand guidelines: `.era-card`, `.btn-primary`, `.track-list`, etc.

### utilities.css

Optional utility classes for spacing, text alignment, visibility, etc.

## Serving Static Files

```go
// Serve static files
fs := http.FileServer(http.Dir("web/static"))
http.Handle("/static/", http.StripPrefix("/static/", fs))
```

```html
<!-- In base.html layout -->
<link rel="stylesheet" href="/static/css/variables.css">
<link rel="stylesheet" href="/static/css/base.css">
<link rel="stylesheet" href="/static/css/components.css">
<script src="/static/js/htmx.min.js"></script>
```
