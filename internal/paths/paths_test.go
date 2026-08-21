package paths

import (
	"strings"
	"testing"
)

func TestBuilders(t *testing.T) {
	t.Parallel()
	if Login() != "/login" || Register() != "/register" || Projects() != "/projects" {
		t.Fatal("root builders")
	}
	if got := Editor("abc"); got != "/editor/abc" {
		t.Fatalf("Editor = %q", got)
	}
	if got := ProjectDelete("abc"); got != "/projects/abc/delete" {
		t.Fatalf("ProjectDelete = %q", got)
	}
}

func TestPatternsMatchBuilders(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern string
		sample  string
	}{
		{PatternLoginGet, Login()},
		{PatternRegisterGet, Register()},
		{PatternProjectsGet, Projects()},
		{PatternEditorGet, Editor("x")},
		{PatternProjectDelete, ProjectDelete("x")},
		{PatternLang, Lang()},
		{PatternLogout, Logout()},
	}
	for _, tc := range cases {
		pathPart := strings.TrimSpace(strings.SplitN(tc.pattern, " ", 2)[1])
		pathPart = strings.ReplaceAll(pathPart, "{id}", "x")
		if !strings.HasPrefix(tc.sample, strings.Split(pathPart, "{")[0]) && pathPart != tc.sample {
			// Compare after substituting the one path param we use.
			if pathPart != tc.sample {
				t.Fatalf("pattern %q vs builder %q", tc.pattern, tc.sample)
			}
		}
	}
}

func TestLoginNext(t *testing.T) {
	t.Parallel()
	if got := LoginNext(""); got != "/login" {
		t.Fatalf("empty next = %q", got)
	}
	if got := LoginNext("/projects"); !strings.Contains(got, "next=") {
		t.Fatalf("next missing: %q", got)
	}
}
