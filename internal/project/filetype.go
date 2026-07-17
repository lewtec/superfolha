package project

import (
	"net/http"
	"path/filepath"
	"strings"
)

// MaxCollabTextBytes is the max size for a text file to live in the project CRDT.
// Matches the historical GraphQL content cap order of magnitude (5 MiB).
const MaxCollabTextBytes = 5 << 20

// binaryExtensions lists well-known binary file extensions (lowercase, with dot).
// Kept in sync with frontend/src/utils/fileUtils.ts BINARY_EXTENSIONS.
var binaryExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".webp": {}, ".ico": {},
	".pdf": {},
	".zip": {}, ".tar": {}, ".gz": {}, ".rar": {}, ".7z": {},
	".exe": {}, ".dll": {}, ".bin": {}, ".out": {},
	".mp3": {}, ".wav": {}, ".ogg": {}, ".flac": {},
	".mp4": {}, ".avi": {}, ".mkv": {}, ".mov": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".otf": {},
	".sqlite": {}, ".db": {},
}

// IsBinary reports whether content should be treated as a blob (not collaborative text).
// Known binary extensions win so short/empty blobs are not misclassified; otherwise
// falls back to http.DetectContentType (text/* => not binary).
func IsBinary(content []byte, filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := binaryExtensions[ext]; ok {
		return true
	}
	contentType := http.DetectContentType(content)
	return !strings.HasPrefix(contentType, "text/")
}
