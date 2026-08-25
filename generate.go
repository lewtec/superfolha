// Package tools holds module-level go:generate directives.
package tools

//go:generate bun install
//go:generate bun run build:css
//go:generate bun run build:editor
//go:generate bun run build:login
//go:generate bun run build:sessions
//go:generate bun run build:chrome
//go:generate go tool templ generate
