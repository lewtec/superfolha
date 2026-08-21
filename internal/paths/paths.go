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
	PatternProjectDelete = "POST /projects/{id}/delete"
	PatternEditorGet     = "GET /editor/{id}"
	PatternStatic        = "GET /static/"
	PatternAPILogout     = "POST /api/logout"
	PatternCompile       = "GET /api/compile"
	PatternUpload        = "POST /api/projects/{projectId}/upload-file"
	PatternDownload      = "GET /api/projects/{projectId}/download/{filePath...}"
	PatternProjectWS     = "GET /ws/projects/{projectId}"
)
