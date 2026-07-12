package server

import (
	"testing"
)

func TestHasBinary(t *testing.T) {
	t.Parallel()

	// Minimal PNG signature; DetectContentType returns image/png.
	pngHeader := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	textContent := []byte("\\documentclass{article}\n\\begin{document}Hi\\end{document}\n")

	tests := []struct {
		name     string
		content  []byte
		filename string
		want     bool
	}{
		{
			name:     "known binary extension empty content",
			content:  nil,
			filename: "logo.png",
			want:     true,
		},
		{
			name:     "known binary extension case insensitive",
			content:  []byte{},
			filename: "Font.WOFF2",
			want:     true,
		},
		{
			name:     "pdf extension",
			content:  nil,
			filename: "docs/report.pdf",
			want:     true,
		},
		{
			name:     "text content with tex extension",
			content:  textContent,
			filename: "main.tex",
			want:     false,
		},
		{
			name:     "text content without extension",
			content:  textContent,
			filename: "README",
			want:     false,
		},
		{
			name:     "binary content without extension",
			content:  pngHeader,
			filename: "blob",
			want:     true,
		},
		{
			name:     "empty content unknown extension",
			content:  nil,
			filename: "notes.txt",
			want:     false,
		},
		{
			name:     "path with directory and binary ext",
			content:  []byte("not really png"),
			filename: "assets/images/icon.ICO",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := HasBinary(tt.content, tt.filename)
			if got != tt.want {
				t.Errorf("HasBinary(%q, %q) = %v, want %v", tt.content, tt.filename, got, tt.want)
			}
		})
	}
}
