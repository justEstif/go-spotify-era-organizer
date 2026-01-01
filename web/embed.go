// Package web provides embedded static assets and templates for the web application.
package web

import "embed"

// TemplatesFS contains the embedded HTML templates.
//
//go:embed all:templates
var TemplatesFS embed.FS

// StaticFS contains the embedded static assets (CSS, JS).
//
//go:embed all:static
var StaticFS embed.FS
