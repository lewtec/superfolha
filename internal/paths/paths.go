// Package paths is the single source of URL paths and mux patterns.
// Handlers register only Pattern* constants; redirects and hrefs use builders.
package paths

import (
	"net/url"
	"path"
)

const (
	Landing    = "/"
	LangCookie = "lang"

	// Mux path-value names. Pattern* strings interpolate these so PathValue
	// and the ServeMux pattern cannot drift.
	ParamID       = "id"
	ParamFilePath = "filePath"
)

func withQuery(p string, q url.Values) string {
	if len(q) == 0 {
		return p
	}
	return p + "?" + q.Encode()
}

func Login() string    { return "/login" }
func Register() string { return "/register" }
func Logout() string   { return "/logout" }
func Lang() string     { return "/lang" }
func Projects() string { return "/projects" }

func Editor(id string) string {
	return path.Join("/editor", id)
}

func ProjectDelete(id string) string {
	return path.Join("/projects", id, "delete")
}

func LoginNext(next string) string {
	if next == "" {
		return Login()
	}
	return withQuery(Login(), url.Values{"next": {next}})
}

func LoginError(err string) string {
	return withQuery(Login(), url.Values{"error": {err}})
}

func RegisterError(err string) string {
	return withQuery(Register(), url.Values{"error": {err}})
}

func ProjectsError(err string) string {
	return withQuery(Projects(), url.Values{"error": {err}})
}

func ProjectsFlash(flash string) string {
	return withQuery(Projects(), url.Values{"flash": {flash}})
}

// --- Static (embedded /static) ---

func Static(elem ...string) string {
	parts := make([]string, 0, len(elem)+1)
	parts = append(parts, "/static")
	parts = append(parts, elem...)
	return path.Join(parts...)
}

func StyleCSS() string  { return Static("style.css") }
func EditorJS() string  { return Static("editor.js") }
func BrandLogo() string { return Static("brand", "logo.png") }
func Favicon() string   { return Static("brand", "favicon.png") }

// --- JSON / WS APIs used by the editor island ---

func APILogout() string { return "/api/logout" }

func Compile(projectID, file string) string {
	q := url.Values{}
	if projectID != "" {
		q.Set("project", projectID)
	}
	if file != "" {
		q.Set("file", file)
	}
	return withQuery("/api/compile", q)
}

func Upload(id string) string {
	return path.Join("/api/projects", id, "upload-file")
}

func Download(id, filePath string) string {
	base := path.Join("/api/projects", id, "download")
	if filePath == "" {
		return base + "/"
	}
	return base + "/" + url.PathEscape(filePath)
}

func ProjectWS(id string) string {
	return path.Join("/ws/projects", id)
}

// Mux patterns (method + path). Register only these.
const (
	PatternLanding       = "GET /{$}"
	PatternLoginGet      = "GET /login"
	PatternLoginPost     = "POST /login"
	PatternRegisterGet   = "GET /register"
	PatternRegisterPost  = "POST /register"
	PatternLogout        = "POST /logout"
	PatternLang          = "POST /lang"
	PatternProjectsGet   = "GET /projects"
	PatternProjectsPost  = "POST /projects"
	PatternProjectDelete = "POST /projects/{" + ParamID + "}/delete"
	PatternEditorGet     = "GET /editor/{" + ParamID + "}"
	PatternStatic        = "GET /static/"

	PatternAPILogout = "POST /api/logout"
	PatternCompile   = "GET /api/compile"
	PatternUpload    = "POST /api/projects/{" + ParamID + "}/upload-file"
	PatternDownload  = "GET /api/projects/{" + ParamID + "}/download/{" + ParamFilePath + "...}"
	PatternProjectWS = "GET /ws/projects/{" + ParamID + "}"
)
