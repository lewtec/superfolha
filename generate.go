// Package tools holds module-level go:generate directives.
package tools

//go:generate bun install
//go:generate bun run build:css
//go:generate bun run build:editor
//go:generate go tool templ generate
