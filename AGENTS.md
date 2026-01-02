# AGENTS.md - Go Spotify Era Organizer

Guidance for AI coding agents working on this Go web application.

## Project Overview

Web application that analyzes Spotify liked songs, detects listening "eras" via tag-based clustering, and generates playlists.

**Stack:** Go 1.25.5+, `zmb3/spotify/v2`, `golang.org/x/oauth2`, HTMX, single binary distribution.

## Build, Test, Lint Commands

```bash
# Build
go build -o spotify-era-organizer ./cmd/...
go build ./...                              # Syntax check only

# Test
go test ./...                               # All tests
go test -v ./...                            # Verbose
go test -v -run TestFunctionName ./pkg      # Single test by name
go test -v ./internal/clustering            # Single package
go test -cover ./...                        # With coverage
go test -race ./...                         # Race detector

# Lint (ALWAYS run before committing)
go fmt ./...                                # Format code
go vet ./...                                # Check for issues
staticcheck ./...                           # Static analysis
golangci-lint run                           # Full linting

# Dependencies
go get github.com/example/package           # Add dependency
go mod tidy                                 # Clean up go.mod/go.sum
```

## Tool Installation

**Do NOT use `go install` globally.** Add tools to `mise.toml` and run `mise up`:

```toml
[tools]
"go:golang.org/x/tools/gopls" = "latest"
"go:honnef.co/go/tools/cmd/staticcheck" = "latest"
"go:github.com/golangci/golangci-lint/cmd/golangci-lint" = "latest"
```

## Project Structure

```
cmd/spotify-era-organizer/main.go    # Entrypoint (web server)
internal/web/                         # HTTP handlers, server, sessions, templates
internal/spotify/                     # Spotify API client wrapper
internal/clustering/                  # Era detection algorithm (k-means on tags)
internal/lastfm/                      # Last.fm API client for genre tags
internal/tags/                        # Tag enrichment service
web/                                  # Static files, templates, embedded assets
docs/                                 # Documentation
```

## Code Style

### Imports (three groups, blank line separated)

```go
import (
    "context"
    "fmt"

    "github.com/zmb3/spotify/v2"

    "github.com/justestif/go-spotify-era-organizer/internal/clustering"
)
```

### Naming Conventions

| Element    | Style                          | Example                       |
| ---------- | ------------------------------ | ----------------------------- |
| Packages   | lowercase, single-word         | `clustering`, `web`           |
| Files      | lowercase, underscores         | `token_cache.go`              |
| Exported   | PascalCase                     | `DetectEras`, `ClusterConfig` |
| Unexported | camelCase                      | `calculateGap`, `tokenStore`  |
| Interfaces | `-er` suffix for single-method | `Clusterer`, `TokenFetcher`   |
| Acronyms   | Consistent case                | `HTTPClient`, `userID`        |

### Error Handling

```go
// Always wrap errors with context
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}

// Define sentinel errors
var ErrInvalidToken = errors.New("invalid token")

// Use errors.Is/As for comparison
if errors.Is(err, ErrNotFound) { ... }
```

### Functions

```go
// Context first, error last
func ProcessTracks(ctx context.Context, fetcher TrackFetcher) (*Result, error)

// Accept interfaces, return concrete types
// Keep functions focused and small
```

### Testing (table-driven)

```go
func TestDetectEras(t *testing.T) {
    tests := []struct {
        name     string
        input    []Track
        gapDays  int
        expected []Era
    }{
        {"single era", []Track{...}, 7, []Era{...}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := DetectEras(tt.input, tt.gapDays)
            // assertions
        })
    }
}
```

### Documentation

```go
// Package clustering implements era detection using tag-based clustering.
package clustering

// DetectMoodEras groups tracks into eras based on tag similarity.
func DetectMoodEras(tracks []Track, cfg TagClusterConfig) ([]MoodEra, []Track)
```

## Task Tracking (Beads)

```bash
bd ready                    # Show ready issues
bd show <id>               # View details
bd create --title="..." --type=task --priority=2
bd update <id> --status=in_progress
bd close <id>
bd sync --from-main        # Sync beads data
```

## Domain Patterns

**OAuth:** Session-based token storage (web application)

**Rate Limiting:** Implement exponential backoff, batch operations

**Clustering:**

- Default clusters: 3 (configurable)
- Minimum cluster size: 3 songs
- Tracks without tags become outliers

## Frontend Development

When building frontend, UI, web pages, or visual components, **always load the `brand-guidelines` skill first** to ensure consistent styling with the project's Liquid Vinyl visual identity.

```
/skill brand-guidelines
```

Key resources:
- `.opencode/skill/brand-guidelines/SKILL.md` - Colors, typography, mood mapping, components
- `docs/implementation-notes.md` - Go template + HTMX patterns
